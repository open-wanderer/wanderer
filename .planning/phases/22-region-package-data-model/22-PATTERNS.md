# Phase 22: Region & Package Data Model - Pattern Map

**Mapped:** 2026-07-21
**Files analyzed:** 4 (+1 asset)
**Analogs found:** 4 / 4

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `app/assets/map/regions.json` | config (bundled asset) | file-I/O | none (new asset kind) | no-analog |
| `app/lib/models/region_manifest.dart` | model (freezed parse model) | transform (JSON → typed object) | `app/lib/models/map_cell.dart` | exact |
| `app/lib/entities/region_entity.dart` | model (ObjectBox entity) | CRUD (local persistence) | `app/lib/entities/trail_entity.dart` (dual-id + ToOne shape) | exact |
| `app/lib/entities/downloaded_tile_package_entity.dart` | model (ObjectBox entity) | CRUD (local persistence) | `app/lib/entities/active_navigation_entity.dart` (status-enum shadow property shape) | role-match |

## Pattern Assignments

### `app/lib/models/region_manifest.dart` (model, transform)

**Analog:** `app/lib/models/map_cell.dart` (in full, in-context above — 53 lines, single Read)

**Imports pattern** (lines 1-4):
```dart
import 'package:freezed_annotation/freezed_annotation.dart';

part 'map_cell.freezed.dart';
part 'map_cell.g.dart';
```
For the new file, use `part 'region_manifest.freezed.dart';` / `part 'region_manifest.g.dart';`.

**Core pattern — list wrapper + entry class, snake_case JSON keys via `@JsonKey`** (lines 17-37):
```dart
@freezed
abstract class MapCellInfoList with _$MapCellInfoList {
  const factory MapCellInfoList({required List<MapCellInfo> cells}) =
      _MapCellInfoList;

  factory MapCellInfoList.fromJson(Map<String, dynamic> json) =>
      _$MapCellInfoListFromJson(json);
}

@freezed
abstract class MapCellInfo with _$MapCellInfo {
  const factory MapCellInfo({
    required String key,
    required MapCellStatus status,
    required String url,
    @JsonKey(name: 'size_bytes') int? sizeBytes,
  }) = _MapCellInfo;

  factory MapCellInfo.fromJson(Map<String, dynamic> json) =>
      _$MapCellInfoFromJson(json);
}
```
**Applied:** `RegionManifest { required List<RegionManifestEntry> regions }` (mirrors `MapCellInfoList`); `RegionManifestEntry { required String id, required String name, @JsonKey(name: 'min_lon') required double minLon, minLat, maxLon, maxLat, @JsonKey(name: 'vector_url') required String vectorUrl, @JsonKey(name: 'vector_size_bytes') int? vectorSizeBytes, @JsonKey(name: 'dem_url') String? demUrl, @JsonKey(name: 'dem_size_bytes') int? demSizeBytes }` (mirrors `MapCellInfo`'s field-per-JSON-key shape, including the nullable trailing fields pattern).

**Asset-load pattern** (from `app/lib/provider/map_style_json_provider.dart`, verified in RESEARCH.md):
```dart
import 'package:flutter/services.dart' show rootBundle;
final raw = await rootBundle.loadString('assets/map/regions.json');
```
Then: `RegionManifest.fromJson(jsonDecode(raw))`. Note this loader itself is not a model-file precedent (it loads a raw string for MapLibre) — only the `rootBundle.loadString` call shape should be copied, not the surrounding class structure.

**No error-handling precedent needed:** `fromJson` codegen throws on missing/malformed fields; no manual validation layer exists in `map_cell.dart` and none should be added here (per RESEARCH.md V5 Input Validation note — bundled asset, not user input).

---

### `app/lib/entities/region_entity.dart` (model, CRUD)

**Analog:** `app/lib/entities/trail_entity.dart` (in full, in-context above — 174 lines, single Read)

**Imports pattern** (lines 1-7):
```dart
import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/actor_entity.dart';
import 'package:wanderer/entities/category_entity.dart';
```
(Drop the `gpx`/`waypoint_entity`/`gpx_util` imports — not relevant to Region; keep the `objectbox` import and add an import for `downloaded_tile_package_entity.dart`.)

**Dual-id pattern** (lines 9-16):
```dart
@Entity()
class TrailEntity {
  @Id()
  int obxId = 0;

  @Index()
  @Unique(onConflict: ConflictStrategy.replace)
  String id;
```
**Applied to `RegionEntity`:** same shape — `obxId` (ObjectBox internal id) + `@Index() @Unique(onConflict: ConflictStrategy.replace) String id` (the manifest's `id`, e.g. `"de-nrw"`).

**ToOne relation pattern** (lines 62-63):
```dart
final author = ToOne<ActorEntity>();
final category = ToOne<CategoryEntity>();
```
**Applied to `RegionEntity`** (per D-06 — no `ToMany`/`@Backlink`, no discriminator):
```dart
final vectorPackage = ToOne<DownloadedTilePackageEntity>();
final demPackage = ToOne<DownloadedTilePackageEntity>();
```
Access: `region.vectorPackage.target?.status` (`.target` nullable by construction).

**`fromModel`/`toModel` extension mapping pattern** (lines 89-133, 136-173):
```dart
factory TrailEntity.fromModel(Trail trail) {
  final entity = TrailEntity(
    id: trail.id,
    name: trail.name,
    ...
  );
  ...
  return entity;
}
```
```dart
extension TrailEntityMapping on TrailEntity {
  Trail toModel() {
    return Trail(
      id: id,
      name: name,
      ...
    );
  }
}
```
**Applied:** `RegionEntity.fromManifestEntry(RegionManifestEntry entry)` factory constructor (mirrors `fromModel`, source model is `RegionManifestEntry` instead of a PocketBase model) mapping bbox/name/id/vectorUrl/demUrl/size fields verbatim; no `toModel()` extension needed yet since nothing downstream reads `Region` back into a manifest shape this phase (flag as optional/YAGNI for planner).

**Note — do NOT copy `dbDifficulty` (lines 44-57):** This is the `.index`-based anti-pattern explicitly forbidden by D-01/D-02. Use the pattern from `active_navigation_entity.dart` below instead, swapping `.index` for `.code`.

---

### `app/lib/entities/downloaded_tile_package_entity.dart` (model, CRUD)

**Analog:** `app/lib/entities/active_navigation_entity.dart` (in full, in-context above — 87 lines, single Read)

**Imports pattern** (line 1):
```dart
import 'package:objectbox/objectbox.dart';
```

**Status-enum shadow-property SHAPE to copy** (lines 18-27) — **structural shape only, `.index` usage is the anti-pattern to avoid:**
```dart
@Transient()
ActiveSessionType sessionType = ActiveSessionType.nav;

int get dbSessionType => sessionType.index;              // ANTI-PATTERN — do not copy this line

set dbSessionType(int value) {
  sessionType = (value >= 0 && value < ActiveSessionType.values.length)
      ? ActiveSessionType.values[value]
      : ActiveSessionType.nav;
}
```

**REQUIRED replacement shape, per D-01/D-02** (verbatim from CONTEXT.md/RESEARCH.md — copy this, not the block above):
```dart
enum PackageStatus {
  notDownloaded(0),
  downloading(1),
  downloaded(2),
  updateAvailable(3);
  const PackageStatus(this.code);
  final int code;
}

@Transient()
PackageStatus status = PackageStatus.notDownloaded;

int get dbStatus => status.code;                          // .code, never .index

set dbStatus(int value) {
  status = PackageStatus.values.firstWhere(
    (s) => s.code == value,
    orElse: () => PackageStatus.notDownloaded,
  );
}
```
Same `enum RegionStatus { ...; const RegionStatus(this.code); final int code; }` shape applies to `RegionStatus`, defined separately (likely alongside `RegionEntity` or in its own small file) since `Region.status` is a computed getter per D-07, not a shadow-persisted field — it does NOT need a `dbStatus` int property on `RegionEntity` itself.

**Constructor pattern (named params, sensible defaults)** (lines 68-85):
```dart
ActiveNavigationEntity({
  this.obxId = 0,
  this.sessionType = ActiveSessionType.nav,
  ...
  required this.updatedAtUtc,
});
```
**Applied to `DownloadedTilePackageEntity`:** named-param constructor with `this.status = PackageStatus.notDownloaded`, `required String localFilePath` (or nullable pre-download), `DateTime? downloadedAtUtc` (`@Property(type: PropertyType.dateUtc)` per the `date`/`updated` fields on `TrailEntity`/`updatedAtUtc` on `ActiveNavigationEntity`), `int? sizeBytesOnDisk`.

**Dual-id note:** `DownloadedTilePackageEntity` should also follow `trail_entity.dart`'s dual-id pattern (`@Id() int obxId = 0`) since it's a standalone ObjectBox row referenced via `ToOne` from `Region`, not a singleton row like `ActiveNavigationEntity` (which only uses a bare `obxId` since there's "at most one row at a time").

---

## Shared Patterns

### Dual-id ObjectBox convention
**Source:** `app/lib/entities/trail_entity.dart` lines 9-16
**Apply to:** `RegionEntity`, `DownloadedTilePackageEntity` — both are standalone rows (not singletons), so both need `@Id() int obxId = 0` plus a stable business-string id where applicable (`RegionEntity.id` matches manifest `id`; `DownloadedTilePackageEntity` likely has no external business id, just `obxId`, since it's only ever addressed via its owning `Region`'s `ToOne`).

### Enum-to-int persistence via explicit `.code` (never `.index`)
**Source:** `.planning/phases/22-region-package-data-model/22-CONTEXT.md` D-01/D-02 (verbatim example), contrasted with the anti-pattern at `app/lib/entities/trail_entity.dart` lines 44-57 and `app/lib/entities/active_navigation_entity.dart` lines 18-27
**Apply to:** `RegionStatus` and `PackageStatus` enums, and their shadow int properties on `DownloadedTilePackageEntity` (`Region` itself has no stored status field — D-07, computed getter only).

### `ToOne` relation, nullable, no cascade
**Source:** `app/lib/entities/trail_entity.dart` lines 62-63 (`final author = ToOne<ActorEntity>();`)
**Apply to:** `RegionEntity.vectorPackage` / `RegionEntity.demPackage`, both `ToOne<DownloadedTilePackageEntity>()`, both nullable via `.target`, no discriminator field (D-06).

### `@freezed` + `json_serializable` parse model
**Source:** `app/lib/models/map_cell.dart` (full file)
**Apply to:** `RegionManifest` / `RegionManifestEntry` in `region_manifest.dart`.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `app/assets/map/regions.json` | config (bundled asset) | file-I/O | No prior bundled structured-data JSON asset in this codebase (`assets/map/` currently holds only the MapLibre style JSON, consumed as a raw string, not a data manifest) — content itself follows RESEARCH.md's proposed schema/values (see RESEARCH.md "Code Examples" section), not a codebase analog. `pubspec.yaml`'s `assets/map/` directory declaration already covers it — no config-file edit needed. |

## Metadata

**Analog search scope:** `app/lib/entities/`, `app/lib/models/`, `app/lib/provider/`
**Files scanned:** `trail_entity.dart`, `active_navigation_entity.dart`, `map_cell.dart`, `map_style_json_provider.dart` (all read in full this session; each ≤ 174 lines, single Read per file, no re-reads)
**Pattern extraction date:** 2026-07-21
