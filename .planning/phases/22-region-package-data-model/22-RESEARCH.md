# Phase 22: Region & Package Data Model - Research

**Researched:** 2026-07-21
**Domain:** Flutter/ObjectBox data-model foundation for offline region tile repository (bundled JSON manifest + `freezed` parse model + ObjectBox entities); narrow-scope research targeting only CONTEXT.md's flagged open item, D-05
**Confidence:** HIGH (in-repo verified for stack/architecture/patterns) / MEDIUM (region-splitting convention, cross-referenced against one primary source) / LOW-flagged where noted (exact bbox coordinates, byte-size estimates)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Status Enum Persistence (REGN-02, REGN-03)**
- **D-01:** Both `RegionStatus` and any `DownloadedTilePackage` status use Dart enhanced enums with an explicit `code` int field per value, e.g.:
  ```dart
  enum RegionStatus {
    notDownloaded(0),
    downloading(1),
    downloaded(2),
    updateAvailable(3);
    const RegionStatus(this.code);
    final int code;
  }
  ```
- **D-02:** Persist via the codebase's existing shadow-property pattern (`@Transient()` enum field + `int get/set` shadow property), but the getter/setter reads/writes `.code` — **never `.index`**. This is a deliberate deviation from `TrailEntity`/`ActiveNavigationEntity`, which both use `.index` today (flagged by research as the exact anti-pattern REGN-02 forbids). New statuses added later just pick an unused int; no ordering dependency.

**regions.json Initial Content (REGN-01)**
- **D-03:** Ship real regions with real bbox coordinates and real vector-PMTiles/DEM URLs — not placeholder/test data. Model region granularity after how OsmAnd and CoMaps split the world into downloadable regions (country/state or sub-country admin-boundary level), not arbitrary custom boxes.
- **D-04:** Include 3-4 regions in this initial manifest.
- **D-05 (flag for research phase):** Exact region boundaries and the concrete URL-sourcing mechanism (how vector-PMTiles/DEM URLs map onto the existing backend pipeline — `db/services/tiles/generator.go` / build.protomaps.com-derived cells + Mapterhorn DEM) are NOT fully locked here. The phase researcher should investigate OsmAnd/CoMaps' region-splitting convention and propose 3-4 concrete regions + real URLs before planning finalizes the manifest content. **(This is the item this research document addresses — see Summary, Code Examples, Assumptions Log A1/A2/A4.)**

**DownloadedTilePackage Shape (REGN-03)**
- **D-06:** `Region` has two nullable `ToOne<DownloadedTilePackage>` fields: `vectorPackage` and `demPackage` (unset when no DEM downloaded). No package-type discriminator field, no `ToMany`/`@Backlink` — direct field access (`region.vectorPackage.target?.status`). Rejected alternative: `ToMany<DownloadedTilePackage>` + a `PackageType` enum discriminator (more extensible but adds a filter step to every read; not worth it for exactly two known package types).

**Region vs Package Status Relationship (REGN-02, REGN-03)**
- **D-07:** `Region.status` is a **computed getter, not a stored field.** It derives from `vectorPackage.target?.status` (folding in `demPackage` status when a DEM is required/present), defaulting to `RegionStatus.notDownloaded` when no package rows exist yet (the pre-download state). This guarantees Region and package status can never drift out of sync — no dual-write step needed when `TileRepositoryManager` (Phase 23) updates package state.
  - **Note for planner:** REGN-02's literal wording says Region "persists... a live status... and the status survives an app restart" as if it were a stored field. The computed-getter approach still satisfies restart-survival (it reads from persisted `DownloadedTilePackage` rows), but does not add a literal stored column on `Region`. This is an intentional, discussed interpretation — not an oversight. If future phases need to query/filter/sort by Region status directly in ObjectBox (e.g., "list all downloaded regions"), that will require either a stored+synced field or an in-memory filter after fetching all regions; flag this to the user if Phase 23/24 planning finds ObjectBox query-by-computed-field to be a blocker.

### Claude's Discretion
- None — every gray area identified was explicitly decided above.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| REGN-01 | A bundled `regions.json` app asset defines regions (id, name, bbox, vector PMTiles URL + size, optional DEM URL + size) | See Summary, Architecture Patterns (Pattern 1), Code Examples for the proposed manifest schema/content and the `freezed` parse-model precedent (`map_cell.dart`). Common Pitfalls 1-3 explain why the URL/size fields must be interpreted as a bbox-cells-endpoint reference + size estimate, not a literal single-file URL/exact byte count — flagged for planner/user confirmation (Assumption A4). |
| REGN-02 | ObjectBox `Region` entity stores manifest fields plus live status (notDownloaded/downloading/downloaded/updateAvailable), using explicit stable int constants (not index-backed enum persistence) | See Architecture Patterns (Pattern 2) for the exact `.code`-based shadow-property implementation, contrasted against the in-repo `.index` anti-pattern it must NOT copy. D-07 (computed-getter status) already resolves the "live status" storage question — this research confirms the pattern's mechanics only. |
| REGN-03 | ObjectBox `DownloadedTilePackage` entity tracks vector and DEM as independent packages per region — local file path, timestamp, size on disk, status | See Architecture Patterns (Pattern 2, Pattern 3) for the `ToOne` relation shape and `.code`-based status persistence, matching D-06. |

</phase_requirements>

## Summary

This phase is almost entirely pre-decided by `22-CONTEXT.md` (D-01 through D-07 lock the enum-persistence pattern, the `ToOne` package shape, and the computed-status-getter approach). The only open item, D-05, asks two things: (1) how OsmAnd/CoMaps split the world into downloadable regions, with 3-4 concrete proposed regions, and (2) how those regions' "vector PMTiles URL"/"DEM URL" fields map onto this project's *existing, unmodified* backend tile pipeline.

On (1): OsmAnd's public `regions.xml` (the authoritative, machine-readable source for its region hierarchy) confirms a continent → country → admin-subdivision pattern, where **large countries split at state/province level** (Germany → `nordrhein-westfalen`, `bayern`, etc.; the very largest German states split one level further into government regions) and **small-to-medium countries stay as a single leaf region** (Switzerland, Austria, Berlin, Brandenburg). This project's own OpenAPI examples (`bbox=6.1,51.2,6.8,51.7`) center on North Rhine-Westphalia — the strongest available signal for a natural anchor region.

On (2): this is the more consequential finding. The existing backend (`db/services/tiles/generator.go`, `db/routes/map_cells.go`/`map_cells_id.go`, and the SvelteKit `GET /api/v1/map/cells` proxy) has **no region-sized archive and no static single-file URL**. It only knows fixed 0.5°×0.5° grid cells, generated on demand, served through *authenticated* per-cell endpoints. The one endpoint that is genuinely region-shaped is `GET /api/v1/map/cells?bbox=minLon,minLat,maxLon,maxLat` — it already resolves an **arbitrary** bounding box (not just a trail's bbox) into the covering grid cells and returns per-cell status/request/download URLs. `trail_download_service.dart` already calls exactly this endpoint today for trail-scoped downloads. `research/SUMMARY.md` (prior milestone research) independently confirms: *"Backend requires zero changes: the grid-cell/bbox endpoints ... are already trail-agnostic; region is a purely client-side, bundled-manifest concept."*

**Primary recommendation:** Model each `regions.json` entry's "vector/DEM URL" field as the real, already-working, relative endpoint `/map/cells?bbox={minLon},{minLat},{maxLon},{maxLat}` (baked in per region at manifest-authoring time, using the region's own bbox) — not a literal single downloadable `.pmtiles` file. This is a real URL against a real, existing, unmodified backend route, satisfying D-03's "not placeholder data" instruction, while remaining honest about the fact that a "region download" is actually N individual cell downloads discovered at request time (exactly matching the established multi-cell pattern already proven for trails). Flag this explicitly to the user/planner as a deviation from REGN-01's literal "vector PMTiles URL" wording (singular direct-download URL) — analogous to how D-07 already flagged a similar wording-vs-architecture gap for `Region.status`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| `regions.json` bundled manifest (asset) | Mobile Client (Flutter, bundled asset) | — | Static app asset, no network fetch — matches D-03/REGN-F04 ("bundled asset only, not remote-fetched") |
| `RegionManifest`/`RegionManifestEntry` parse model (`freezed`) | Mobile Client | — | Pure data parsing, no I/O beyond `rootBundle.loadString` |
| `Region`/`DownloadedTilePackage` ObjectBox entities | Mobile Client (Database/Storage — on-device) | — | Local persistence only; no server-side counterpart exists or is needed |
| `Region.status` computed getter | Mobile Client (in-memory business logic) | — | Derives from persisted `DownloadedTilePackage` rows per D-07; not a stored column |
| Vector/DEM tile URL resolution (`/api/v1/map/cells?bbox=`) | API/Backend (SvelteKit proxy → Go/PocketBase) | Mobile Client (consumer) | Already exists, already trail-agnostic, unmodified by this phase — Phase 22 only *references* it as a manifest field value; Phase 23 (`TileRepositoryManager`) is the actual caller |
| Tile cell generation (pmtiles extract, Mapterhorn DEM) | API/Backend (Go, `generator.go`) | — | Unmodified this phase; on-demand, per-0.5°-cell, authenticated |

## Standard Stack

### Core
No new dependencies are required for this phase. All patterns reuse packages already pinned in `app/pubspec.yaml`:

| Library | Version (pubspec.yaml) | Purpose | Why Standard (in this repo) |
|---------|-------------------------|---------|------------------------------|
| `objectbox` / `objectbox_flutter_libs` | ^5.3.1 `[VERIFIED: app/pubspec.yaml]` | On-device persistence for `Region`/`DownloadedTilePackage` | Same `Store`, same codegen flow already used by `TrailEntity`/`ActiveNavigationEntity`/`WaypointEntity` |
| `freezed_annotation` / `freezed` (dev) | ^3.1.0 / ^3.2.5 `[VERIFIED: app/pubspec.yaml]` | Immutable parse model for `regions.json` | Matches `MapCellInfo`/`MapCellInfoList`/`MapCellStatusResponse` in `app/lib/models/map_cell.dart` — the closest existing precedent for a manifest-shaped JSON model in this codebase |
| `json_annotation` / `json_serializable` (dev) | ^4.11.0 / ^6.13.0 `[VERIFIED: app/pubspec.yaml]` | `fromJson`/`toJson` codegen paired with `freezed` | Same pairing used throughout `app/lib/models/` |
| `build_runner` (dev) | ^2.13.1 `[VERIFIED: app/pubspec.yaml]` | Codegen driver for both `freezed`/`json_serializable` and `objectbox_generator` | One `dart run build_runner build` regenerates `objectbox.g.dart`, `objectbox-model.json`, and the new model's `.freezed.dart`/`.g.dart` |

**Installation:** None — no `pubspec.yaml` changes needed. Verify with:
```bash
dart run build_runner build --delete-conflicting-outputs
```

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Bundled `assets/map/regions.json` + `freezed` model | Hand-parsed raw `Map<String, dynamic>` (like `map_style_json_provider.dart`'s raw-string style load) | Rejected by the phase's own context notes — `map_style_json_provider.dart` loads a raw *string* (style JSON is consumed by MapLibre directly, never typed), not a precedent for a manifest that the app itself must read/query field-by-field. `map_cell.dart`'s typed `@freezed` pattern is the correct precedent. |
| `.index`-backed enum persistence (existing `TrailEntity`/`ActiveNavigationEntity` pattern) | Explicit `code` int (D-01/D-02) | Already locked by CONTEXT.md — do not reconsider; this is the exact anti-pattern PITFALLS.md (Pitfall 6) flags as the reason this phase exists in the current form. |

## Package Legitimacy Audit

Not applicable — this phase introduces zero new external packages. All dependencies used (`objectbox`, `freezed`, `json_serializable`, `build_runner`) are already present and pinned in `app/pubspec.yaml`, verified directly by reading that file (`[VERIFIED: app/pubspec.yaml]`). No `slopcheck`/registry verification needed.

## Architecture Patterns

### System Architecture Diagram

```
assets/map/regions.json (bundled app asset, ships with the .ipa/.apk)
        │
        │  rootBundle.loadString('assets/map/regions.json')  [Phase 22]
        ▼
RegionManifest.fromJson(jsonDecode(raw))          (freezed parse model, Phase 22)
        │
        │  (nothing consumes this yet — Phase 22 is purely additive)
        ▼
[Phase 23, NOT this phase] TileRepositoryManager seeds ObjectBox Region rows
from the parsed manifest on first app launch / manifest-version bump
        │
        ▼
ObjectBox Region entity (bbox, name, id, vectorUrl, demUrl, vectorSizeBytes,
demSizeBytes) ──ToOne──> DownloadedTilePackage (vectorPackage)
                ──ToOne──> DownloadedTilePackage (demPackage, nullable)
        │
        │ Region.status getter reads vectorPackage.target?.status /
        │ demPackage.target?.status  (D-07 — computed, not stored)
        ▼
[Phase 23, NOT this phase] When download is actually triggered:
  GET {region.vectorUrl}  ==  GET /api/v1/map/cells?bbox=<region's own bbox>
        │  (SvelteKit proxy → Go/PocketBase, ALREADY EXISTS, unmodified)
        ▼
  { "cells": [ { key, status, request_url, status_url, download_url,
                 dem_download_url? }, ... one entry per 0.5° grid cell ... ] }
        │
        ▼
  Per-cell GET download_url / dem_download_url  (same mechanism
  trail_download_service.dart already uses for trail-scoped downloads)
```

A reader tracing "how does a region's vector PMTiles data actually get downloaded" follows: bundled JSON → parse model → (Phase 23) ObjectBox row → (Phase 23) hit the region's stored `vectorUrl`, which is the existing bbox-to-cells endpoint → per-cell download URLs → per-cell file download. **Phase 22 only builds the top three boxes** (asset → parse model → entities); the rest is drawn to make the URL-sourcing decision (D-05, part 2) legible to the planner, not because Phase 22 implements it.

### Recommended Project Structure
```
app/
├── assets/map/
│   └── regions.json              # NEW — bundled manifest, see Code Examples
├── lib/models/
│   └── region_manifest.dart      # NEW — @freezed parse model (mirrors map_cell.dart)
└── lib/entities/
    ├── region_entity.dart        # NEW — @Entity() Region
    └── downloaded_tile_package_entity.dart  # NEW — @Entity() DownloadedTilePackage
```
`assets/map/` is **already** declared as an asset directory in `app/pubspec.yaml` (`assets: - assets/map/`), covering the whole directory — no `pubspec.yaml` edit needed to add `regions.json` to it. `[VERIFIED: app/pubspec.yaml]`

### Pattern 1: `@freezed` manifest parse model
**What:** A `@freezed` class pair (list wrapper + entry) parsed via `part '*.freezed.dart'` / `part '*.g.dart'` and a `factory fromJson`.
**When to use:** Any bundled or fetched JSON that the app needs to read field-by-field (as opposed to a style JSON blob MapLibre consumes as an opaque string).
**Example (verbatim precedent, `app/lib/models/map_cell.dart`):**
```dart
// Source: app/lib/models/map_cell.dart (in-repo, verified)
import 'package:freezed_annotation/freezed_annotation.dart';

part 'map_cell.freezed.dart';
part 'map_cell.g.dart';

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
**Applied to this phase:** `RegionManifest { required List<RegionManifestEntry> regions }` + `RegionManifestEntry { id, name, minLon, minLat, maxLon, maxLat, vectorUrl, vectorSizeBytes, demUrl?, demSizeBytes? }`, snake_case JSON keys via `@JsonKey(name: '...')` exactly as `size_bytes`/`download_url` are handled above.

**Loading the asset (verbatim precedent, `app/lib/provider/map_style_json_provider.dart`):**
```dart
// Source: app/lib/provider/map_style_json_provider.dart (in-repo, verified)
import 'package:flutter/services.dart' show rootBundle;
final raw = await rootBundle.loadString('assets/map/regions.json');
```
Apply the same call with `'assets/map/regions.json'`, then `RegionManifest.fromJson(jsonDecode(raw))`.

### Pattern 2: Dual-id ObjectBox entity + explicit-`code` enum shadow property
**What:** `@Id() int obxId = 0` (ObjectBox internal id) + a separate business-meaning `@Index() @Unique(onConflict: ConflictStrategy.replace) String id` (the manifest's `id` field, e.g. `"de-nrw"`), plus `@Transient()` enum + int shadow property using `.code` (D-02), never `.index`.
**When to use:** `Region` and `DownloadedTilePackage` entities.
**Example (verbatim precedent + explicit deviation, per D-01/D-02):**
```dart
// Source: app/lib/entities/trail_entity.dart (dual-id pattern, in-repo, verified)
@Id()
int obxId = 0;

@Index()
@Unique(onConflict: ConflictStrategy.replace)
String id;
```
```dart
// Source: app/lib/entities/active_navigation_entity.dart (shadow-property SHAPE to copy —
// note this file itself uses `.index`, which D-02 explicitly forbids copying verbatim)
@Transient()
ActiveSessionType sessionType = ActiveSessionType.nav;

int get dbSessionType => sessionType.index;              // <- ANTI-PATTERN, do not copy this line

set dbSessionType(int value) {
  sessionType = (value >= 0 && value < ActiveSessionType.values.length)
      ? ActiveSessionType.values[value]
      : ActiveSessionType.nav;
}
```
```dart
// REQUIRED shape for this phase, per D-01/D-02 (RegionStatus example from CONTEXT.md)
enum RegionStatus {
  notDownloaded(0),
  downloading(1),
  downloaded(2),
  updateAvailable(3);
  const RegionStatus(this.code);
  final int code;
}

@Transient()
RegionStatus status = RegionStatus.notDownloaded;

int get dbStatus => status.code;                          // .code, never .index

set dbStatus(int value) {
  status = RegionStatus.values.firstWhere(
    (s) => s.code == value,
    orElse: () => RegionStatus.notDownloaded,
  );
}
```
Note the decode side must also change shape: `ActiveNavigationEntity`'s `.index`-based decode uses `ActiveSessionType.values[value]` (positional lookup — breaks if enum order changes); the `.code`-based decode must use `.firstWhere((s) => s.code == value, orElse: ...)` (value lookup — safe under reordering) since `code` is an explicit, non-positional int.

### Pattern 3: `ToOne` relation, nullable, no cascade
**What:** `final vectorPackage = ToOne<DownloadedTilePackage>();` / `final demPackage = ToOne<DownloadedTilePackage>();` on `Region` (D-06 — no `ToMany`/`@Backlink`, no discriminator).
**Example (verbatim precedent, `app/lib/entities/trail_entity.dart`):**
```dart
// Source: app/lib/entities/trail_entity.dart (in-repo, verified)
final author = ToOne<ActorEntity>();
final category = ToOne<CategoryEntity>();
```
Access pattern per D-06: `region.vectorPackage.target?.status` — `.target` is nullable by construction; no additional null-handling scaffolding needed beyond what `ToOne` already provides.

**Note for planner (from PITFALLS.md Pitfall 7, informational only — not actionable in Phase 22):** ObjectBox does not cascade-delete. Phase 22 does not implement any delete path (purely additive, nothing downstream reads these entities yet), so this is not a Phase 22 concern — but the planner should be aware Phase 23 (which *does* implement region deletion) will need an explicit cleanup transaction, not an assumed cascade.

### Anti-Patterns to Avoid
- **`.index`-backed enum persistence:** The exact anti-pattern REGN-02 forbids (D-01/D-02) and PITFALLS.md Pitfall 6 documents in detail — silently reinterprets on-device values if a future phase inserts (not appends) an enum value. `TrailEntity.dbDifficulty` and `ActiveNavigationEntity.dbSessionType` both do this today; do not extend the pattern to `Region`/`DownloadedTilePackage`.
- **Treating `regions.json`'s "vector PMTiles URL" as a single static downloadable file:** No such file exists anywhere in this backend for a region-sized area (see Common Pitfalls below) — the value must be the bbox-based cells-list endpoint.
- **A stored `Region.status` column:** Locked as a computed getter by D-07 — do not add a persisted status field to `Region` itself.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Resolving a region's bbox into downloadable tile URLs | A new backend endpoint, or a hardcoded list of individual cell URLs baked into `regions.json` | The existing `GET /api/v1/map/cells?bbox=...` endpoint (already trail-agnostic, already handles arbitrary bboxes) | It already does exactly this; a bespoke Phase 22/23 endpoint would duplicate `db/services/tiles/generator.go`'s grid/cache/generation logic for no benefit — confirmed zero-backend-change by prior milestone research (`research/SUMMARY.md`) |
| JSON manifest parsing | Hand-rolled `Map<String, dynamic>` field access | `@freezed` + `json_serializable`, matching `map_cell.dart` | Type safety, `==`/`copyWith` for free, consistent with every other parsed-JSON model in `app/lib/models/` |
| Enum-to-int persistence | `.index` (matches existing code, but is the anti-pattern this phase exists to avoid) | Explicit `code` int per enum value (D-01/D-02) | `.index` silently breaks on any future non-append enum change; explicit `code` is immune to reordering |

**Key insight:** The single biggest "don't hand-roll" in this phase is resisting the urge to invent a new region-shaped backend contract. The existing cell/bbox pipeline already generalizes to arbitrary regions — Phase 22's job is purely to store that fact in the data model (the URL field's *value*), not to build new backend surface.

## Common Pitfalls

### Pitfall 1: Assuming "vector PMTiles URL" means a single directly-downloadable `.pmtiles` file
**What goes wrong:** REGN-01's literal wording ("vector PMTiles URL + size") reads as if a region resolves to one static file, the way `db/services/tiles/generator.go`'s `CellPath`/`DemCellPath` resolve one *cell* to one static file. Region-sized areas have no such file — `generateCell` only ever produces one 0.5°×0.5° archive at a time, on demand, per cell.
**Why it happens:** The requirement's wording was written before the backend's grid-cell architecture was re-examined in detail for D-05; a region spanning, say, all of North Rhine-Westphalia (~3.6° × 2.2°) decomposes into on the order of 8×5 ≈ 40 individual 0.5° cells, each independently generated/cached/authenticated. `[VERIFIED: db/services/tiles/grid.go GridSize = 0.5, BboxToGridCells]`
**How to avoid:** Store the bbox-based cells-list endpoint (`/map/cells?bbox=<region bbox>`) as the manifest's URL value, as this research recommends. Document this explicitly in the manifest model's doc comment so a future reader doesn't "fix" it into a literal single-file URL.
**Warning signs:** Any code path (this phase or Phase 23) that tries a raw `Dio.download()` directly against `region.vectorUrl` expecting one `.pmtiles` file will get back a JSON `{ "cells": [...] }` response instead — a clear, fail-fast signal if it happens, not a silent data-corruption risk.

### Pitfall 2: Per-cell endpoints require authentication — a bundled manifest cannot embed a fully-resolvable public URL
**What goes wrong:** `/api/v1/map/cells` (list) and every per-cell download endpoint are called through `event.locals.pb` / an authenticated `Dio` instance with a `Bearer` token (see `trail_download_service.dart`'s `_api.get(...)`, `web/src/routes/api/v1/map/cells/[cellKey]/download-dem/+server.ts`'s `Authorization: Bearer ${token}`). A `regions.json` manifest value can only ever be a *relative path* (resolved against the app's already-configured, authenticated `Dio` instance), never a fully-qualified, standalone-fetchable URL — because the operator's server host itself is only known at runtime (`api_provider.dart`'s `updateBaseUrl`, self-hosted instance picker), not at manifest-authoring time. `[VERIFIED: app/lib/provider/api_provider.dart, app/lib/services/trail_download_service.dart]`
**Why it happens:** "URL" in the requirement's wording suggests a complete, host-qualified string; the actual mechanism is host-relative and identity-bound.
**How to avoid:** Store only the relative path + query string (e.g. `/map/cells?bbox=5.87,50.32,9.46,52.53`) in the manifest; Phase 23's `TileRepositoryManager` must reuse the existing authenticated `Dio` instance (`ref.watch(apiProvider)`), never construct a bare `http.get` against a manifest-embedded absolute host.
**Warning signs:** A region download that works in dev against one operator's server but 401s against another self-hosted instance — the tell that a host got baked into the manifest instead of staying relative.

### Pitfall 3: No real per-region byte-size data exists yet
**What goes wrong:** `vector_size_bytes`/`dem_size_bytes` values in the initial manifest cannot be measured today — cell generation is lazy/on-demand (`tile_cells.status` starts `"pending"`), so no region has ever been fully generated to produce a real aggregate size. Any number shipped in Phase 22 is necessarily an **estimate**, not a measurement.
**Why it happens:** D-03 asks for "real ... size" data, but the backend architecture makes an exact figure genuinely unavailable without first generating every cell in every proposed region (a multi-GB, multi-hour operation not appropriate for a data-model-only, zero-UI phase).
**How to avoid:** Flag size fields explicitly as estimates in the manifest/model doc comments (e.g., "approximate, not measured — refine once Phase 23's download engine can report actual `size_bytes` per cell"). This is called out as Assumption A3 below; the planner should decide whether to (a) ship a rough estimate now, or (b) leave size nullable/zero and treat "populate real sizes" as an explicit Phase 23 follow-up task once cells can actually be generated and measured.
**Warning signs:** None at build time — this is a data-accuracy issue, not a functional bug; surfaces later as a Settings UI (Phase 24) showing a materially wrong pre-download size estimate if not revisited.

## Code Examples

### Proposed `assets/map/regions.json` content (D-03/D-04/D-05)
```json
{
  "regions": [
    {
      "id": "de-nrw",
      "name": "North Rhine-Westphalia, Germany",
      "min_lon": 5.87,
      "min_lat": 50.32,
      "max_lon": 9.46,
      "max_lat": 52.53,
      "vector_url": "/map/cells?bbox=5.87,50.32,9.46,52.53",
      "vector_size_bytes": 190000000,
      "dem_url": "/map/cells?bbox=5.87,50.32,9.46,52.53",
      "dem_size_bytes": 55000000
    },
    {
      "id": "ch",
      "name": "Switzerland",
      "min_lon": 5.96,
      "min_lat": 45.82,
      "max_lon": 10.49,
      "max_lat": 47.81,
      "vector_url": "/map/cells?bbox=5.96,45.82,10.49,47.81",
      "vector_size_bytes": 165000000,
      "dem_url": "/map/cells?bbox=5.96,45.82,10.49,47.81",
      "dem_size_bytes": 70000000
    },
    {
      "id": "at",
      "name": "Austria",
      "min_lon": 9.53,
      "min_lat": 46.37,
      "max_lon": 17.16,
      "max_lat": 49.02,
      "vector_url": "/map/cells?bbox=9.53,46.37,17.16,49.02",
      "vector_size_bytes": 210000000,
      "dem_url": "/map/cells?bbox=9.53,46.37,17.16,49.02",
      "dem_size_bytes": 95000000
    },
    {
      "id": "us-co",
      "name": "Colorado, United States",
      "min_lon": -109.06,
      "min_lat": 36.99,
      "max_lon": -102.04,
      "max_lat": 41.00,
      "vector_url": "/map/cells?bbox=-109.06,36.99,-102.04,41.00",
      "vector_size_bytes": 240000000,
      "dem_url": "/map/cells?bbox=-109.06,36.99,-102.04,41.00",
      "dem_size_bytes": 130000000
    }
  ]
}
```
`[ASSUMED — see Assumptions Log A1-A3]` bbox coordinates and byte-size estimates are approximate; the URL construction pattern (`/map/cells?bbox=<region's own bbox>`) is `[VERIFIED: db/routes/map_cells.go, web/src/routes/api/v1/map/cells/+server.ts]` — that endpoint genuinely exists, accepts an arbitrary bbox, and returns real per-cell download URLs today.

### Region-splitting convention reference (OsmAnd, primary source)
```
# Source: https://github.com/osmandapp/OsmAnd-resources/blob/master/countries-info/regions.xml (fetched, verified)
<region name="germany" ...>                                    # country level
  <region name="nordrhein-westfalen" inner_download_prefix="germany_nordrhein-westfalen" ...>  # state level (large country)
    <region name="dusseldorf" .../>                             # further split (largest states only)
    <region name="cologne-government-region" .../>
  </region>
  <region name="berlin" .../>                                   # small state = leaf region, no further split
</region>
```

## State of the Art

No "old vs. new" API changes apply here — this is new data-model surface, not a migration of existing region logic (there is no prior region concept in this codebase; today's tile cache is entirely trail-scoped).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The 4 proposed regions' bbox coordinates (NRW, Switzerland, Austria, Colorado) are accurate to within normal administrative-boundary tolerance | Code Examples | Low — a bbox that's off by a fraction of a degree only slightly over/under-covers the named area; does not break the download mechanism (cells are generated from whatever bbox is given). Recommend a quick sanity-check against a real geodata source (e.g. Natural Earth admin-1 bounding boxes) before shipping, but not launch-blocking for this additive, zero-UI phase. |
| A2 | `vector_size_bytes`/`dem_size_bytes` values (150-240 MB / 55-130 MB per region) are rough order-of-magnitude estimates, not measurements | Common Pitfalls (Pitfall 3), Code Examples | Medium — if surfaced verbatim in Phase 24's Settings UI without revision, users see an inaccurate pre-download size. Flag as a Phase 23/24 follow-up: replace with real aggregated `size_bytes` once cells can actually be generated and measured. |
| A3 | CoMaps follows the same country/admin-subdivision splitting convention as OsmAnd (only OsmAnd's `regions.xml` was directly verified; CoMaps, as an Organic Maps/Maps.me-lineage fork, was not found to publish an equivalently detailed machine-readable region list in this session) | Summary, State of the Art | Low — the recommendation (mixed country-level + state-level granularity) is well-supported by OsmAnd alone and is a reasonable, defensible convention regardless; if CoMaps' actual granularity differs materially, it only affects the *aesthetic* precedent-citation, not the functional design (D-04 only requires 3-4 concrete regions, not exact OsmAnd/CoMaps parity). |
| A4 | Storing the bbox-based `/map/cells?bbox=...` endpoint as the manifest's "vector/DEM URL" field (rather than a literal single-file URL) is the correct interpretation of REGN-01 | Summary, Pattern 1, Pitfall 1 | Medium — this is an architectural interpretation, not a locked user decision (D-05 explicitly deferred exact URL-sourcing mechanism to research). Flag to planner/user for confirmation, same as D-07's flagged wording gap; if the user wants a genuinely different mechanism (e.g. a new backend region-archive endpoint), that changes Phase 22's manifest schema and adds backend scope not currently in v1.6's phase list. |

## Open Questions

1. **Should `vector_url`/`dem_url` be identical per region (both point at the same bbox-cells endpoint), or should the schema drop `dem_url` as a distinct field entirely?**
   - What we know: Both vector and DEM cell readiness/URLs are discovered through the *same* `/map/cells?bbox=` response (each cell entry conditionally includes `dem_download_url`) — there is no backend concept of a region-level DEM-only endpoint distinct from the vector one.
   - What's unclear: Whether REGN-01's "optional DEM URL" literally means a schema field that can be null (e.g., for a hypothetical future region with no DEM coverage), or whether it's satisfied by DEM simply being optional at the *cell* level (already true today — DEM generation is best-effort and independent of vector).
   - Recommendation: Keep `dem_url` as a schema field (satisfies the literal requirement, keeps `Region`/manifest shape self-describing) but default it to the same value as `vector_url` for all initial regions, since they resolve through the identical endpoint. Planner should confirm this reading with the user before finalizing the `RegionManifestEntry` field list.

2. **Real byte-size measurement path — deferred to when?**
   - What we know: No real size data can exist until cells are actually generated (Pitfall 3).
   - What's unclear: Whether Phase 23 (download engine) or Phase 24 (Settings UI) is the right place to replace estimated sizes with measured ones, or whether this stays an accepted approximation indefinitely (bounding-box download sizes are inherently estimates in every reviewed competitor app too — OsmAnd/CoMaps/Organic Maps all show a "~" or rounded estimate, not an exact byte count, pre-download).
   - Recommendation: Not blocking for Phase 22; note in the manifest model's doc comment that sizes are estimates, and let Phase 24 planning decide whether "~" prefixing or a refresh mechanism is worth building.

## Environment Availability

Skipped — this phase has no new external tool/service dependencies. `objectbox`, `freezed`, `json_serializable`, and `build_runner` are already installed and used throughout `app/lib/`; `dart run build_runner build` is the standard, already-used codegen command for this repo.

## Security Domain

`security_enforcement` is `true` in `.planning/config.json` (ASVS level 1). This phase is narrow (bundled-asset parsing + local persistence, zero network calls, zero UI), so most ASVS categories don't apply — documented here for completeness rather than as new work.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | This phase makes no network calls; the manifest URL fields are *stored strings*, not dereferenced until Phase 23 (which already reuses the existing authenticated `Dio` instance) |
| V3 Session Management | No | Same as above |
| V4 Access Control | No | Local-only data, no multi-user access boundary on-device |
| V5 Input Validation | Yes | The `freezed`/`json_serializable` `fromJson` codegen already throws on missing required fields or type mismatches (e.g. missing `id`); no additional hand-rolled validation needed for a bundled (not user-supplied, not network-fetched) asset. Recommend an app-startup assertion/test that `regions.json` parses successfully, so a malformed manifest fails CI rather than crashing at runtime on-device. |
| V6 Cryptography | No | No secrets, tokens, or cryptographic material touched by this phase |

### Known Threat Patterns for this phase's stack
None applicable — no untrusted input crosses a trust boundary in this phase (the manifest is a bundled, developer-authored asset, not user- or network-supplied; `ObjectBox` local storage has no remote attack surface introduced here).

## Project Constraints (from CLAUDE.md)

`./CLAUDE.md` (checked into the repo) contains two distinct sets of content. The parts directly applicable to this phase:

- **Tech stack conventions:** `objectbox` 0.26.8-line/^5.3.1, `freezed_annotation`/`json_annotation` codegen, Dart/Flutter naming conventions (snake_case files, PascalCase types/classes, camelCase functions/variables, boolean `is`-prefix) — all followed by the patterns recommended in this document (mirrors `TrailEntity`/`map_cell.dart` exactly).
- **Type safety:** "All files use strict mode" — applies as-is; no phase-specific deviation needed.
- **Module design:** "Classes used for models" and "Named exports for utilities" (web convention) don't directly map to Dart, but the Dart-side equivalent already established in this codebase (`@freezed` classes for parsed models, `@Entity()` classes for persistence) is what this research recommends.

**Not applicable to this phase:** CLAUDE.md's top-level `## Project` section describes a *different* initiative — "Wanderer Instance Federation" (Go/PocketBase ActivityPub instance-to-instance federation, with its own constraints around `Application` actor types, Follow/Accept/Undo activities, and `is_public` fanout filtering). That work touches the Go backend federation layer; Phase 22 is a Flutter-only, client-side, purely-additive data-model phase with zero backend changes. None of the federation constraints apply here. Flagging this explicitly since CLAUDE.md is otherwise treated as authoritative — the mismatch appears to be leftover content from a different planning session rather than a constraint on this phase.

## Sources

### Primary (HIGH confidence)
- In-repo (all read directly this session): `db/services/tiles/generator.go`, `db/services/tiles/grid.go`, `db/routes/map_cells.go`, `db/routes/map_cells_id.go`, `web/src/routes/api/v1/map/cells/+server.ts`, `web/src/routes/api/v1/map/cells/[cellKey]/download-dem/+server.ts`, `app/lib/services/trail_download_service.dart`, `app/lib/provider/api_provider.dart`, `app/lib/entities/trail_entity.dart`, `app/lib/entities/active_navigation_entity.dart`, `app/lib/models/map_cell.dart`, `app/lib/provider/map_style_json_provider.dart`, `app/pubspec.yaml`
- `.planning/research/SUMMARY.md`, `.planning/research/PITFALLS.md` (prior milestone research, same v1.6 milestone, explicitly confirms zero-backend-change reuse of `/map/cells?bbox=`)
- [OsmAnd-resources `regions.xml`](https://github.com/osmandapp/OsmAnd-resources/blob/master/countries-info/regions.xml) — fetched and read directly, confirms country/state/sub-state hierarchy with `germany`/`nordrhein-westfalen`/`dusseldorf` example chain

### Secondary (MEDIUM confidence)
- [OsmAnd Download Maps docs](https://osmand.net/docs/user/start-with/download-maps/) — general region/folder navigation UX, corroborates the hierarchy found in `regions.xml`
- [CoMaps official site](https://www.comaps.app/) / [CoMaps OSM Wiki page](https://wiki.openstreetmap.org/wiki/CoMaps) — confirms CoMaps is an Organic Maps-lineage, OSM-data, offline-first app, but did not surface an equivalently detailed public region-hierarchy document in this session (see Assumption A3)

### Tertiary (LOW confidence)
- Proposed bbox coordinates for the 4 regions (NRW/Switzerland/Austria/Colorado) — drawn from general geographic knowledge, not cross-checked against an authoritative geodata source in this session (Assumption A1)
- Estimated vector/DEM byte sizes per region — order-of-magnitude only, no real measurement possible against the current on-demand generation architecture (Assumption A2)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies, every pattern verified against an in-repo file read this session
- Architecture (URL-sourcing mechanism, D-05 part 2): HIGH — directly verified against `generator.go`/`grid.go`/`map_cells.go`/`map_cells_id.go`/the SvelteKit proxy/`trail_download_service.dart`, cross-confirmed by prior milestone `research/SUMMARY.md`
- Region-splitting convention (D-05 part 1): MEDIUM — OsmAnd's `regions.xml` is a primary, directly-fetched source; CoMaps' exact convention was not independently confirmed to the same depth
- Concrete bbox/size values: LOW — flagged explicitly in Assumptions Log, not launch-blocking for this additive/zero-UI phase

**Research date:** 2026-07-21
**Valid until:** No expiry pressure — this researches a stable, already-shipped backend pipeline (unmodified by this phase) and a competitor-app convention unlikely to change; safe to treat as valid for the remainder of the v1.6 milestone (Phases 22-27)
