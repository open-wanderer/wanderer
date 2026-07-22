---
phase: 22-region-package-data-model
plan: 01
subsystem: mobile-offline-data-model
tags: [flutter, dart, objectbox, freezed, json_serializable, enhanced-enums]

# Dependency graph
requires:
  - phase: 21.5-region-catalog-archive-pre-build-backend
    provides: "GET /api/v1/regions catalog endpoint (Go regions_get.go) and its exact conditional-field response shape that this plan's parse model mirrors"
provides:
  - "RegionCatalogEntry freezed parse model for one GET /api/v1/regions array element"
  - "CatalogStatus/RegionStatus/PackageStatus enhanced enums with explicit .code int persistence (never .index)"
  - "DownloadedTilePackageEntity ObjectBox entity (status/path/timestamp/size)"
  - "RegionEntity ObjectBox entity: catalog fields, computed RegionStatus getter, two ToOne package links, fromCatalogEntry/applyCatalogEntry upsert mapping"
affects: [23-tile-repository-manager-download-engine, 24-settings-offline-maps-regions-ui, 25-map-rendering-region-based-viewport-pipeline, 26-trail-download-guard, 27-legacy-cleanup]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Enhanced Dart enums with explicit `final int code` + const constructor, persisted via `@Transient()` + int shadow getter/setter, decoded by `.firstWhere((s) => s.code == value)` value-lookup — never `Enum.values[index]` — so future reordering of enum members cannot silently reinterpret on-device rows"
    - "Computed getter (no setter, no @Property) for a status derived purely from ToOne relation targets, guaranteeing it can never persist out of sync with its source data"
    - "fromX/applyX pair on an ObjectBox entity: fromX constructs fresh (all local-only fields at defaults), applyX mutates in place touching ONLY the fields the source of truth owns, explicitly preserving obxId/relations/local-only fields — the upsert-safe pattern for a catalog-fetch merge"

key-files:
  created:
    - app/lib/models/region_status.dart
    - app/lib/models/region_catalog_entry.dart
    - app/lib/models/region_catalog_entry.freezed.dart
    - app/lib/models/region_catalog_entry.g.dart
    - app/lib/entities/downloaded_tile_package_entity.dart
    - app/lib/entities/region_entity.dart
    - app/test/models/region_status_test.dart
    - app/test/models/region_catalog_entry_test.dart
    - app/test/entities/region_entity_test.dart
  modified:
    - app/lib/objectbox.g.dart
    - app/lib/objectbox-model.json

key-decisions:
  - "CatalogStatus gained a 4th entity-only sentinel member `absent` (code 3, no @JsonValue) so RegionEntity.demStatus can stay a non-nullable explicit-int-enum shadow instead of needing a nullable ObjectBox column, per 22-CONTEXT.md D-05/D-09"
  - "RegionEntity.status getter's updateAvailable branch checks only the vector version/lastDownloadedVersion pair — DEM staleness has no concept in the API (no dem_version field), matching D-07 and the existing 'DEM lifecycle fully independent from vector' precedent"

patterns-established:
  - "Explicit-.code enhanced-enum persistence (contrasts with TrailEntity.dbDifficulty / ActiveNavigationEntity.dbSessionType's .index anti-pattern) — the sanctioned shape for all future status-bearing ObjectBox entities in this codebase"

requirements-completed: [REGN-01, REGN-02, REGN-03]

# Metrics
duration: 6min
completed: 2026-07-22
---

# Phase 22 Plan 01: Region & Package Data Model Summary

**Typed freezed parse model for GET /api/v1/regions plus ObjectBox RegionEntity/DownloadedTilePackageEntity, all status enums persisted via explicit `.code` int constants (never `.index`)**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-22T09:39Z (approx, first task commit)
- **Completed:** 2026-07-22T09:44Z
- **Tasks:** 3 completed
- **Files modified:** 11 (9 created, 2 regenerated)

## Accomplishments
- `RegionCatalogEntry` freezed model parses one `/api/v1/regions` array element with correct required/nullable field shape matching `regions_get.go`'s conditional construction (full-ready and minimal-building fixtures both covered by tests)
- `CatalogStatus`/`RegionStatus`/`PackageStatus` enhanced enums persist/decode by explicit `.code` value, never positional `.index` — the exact REGN-02 anti-pattern this phase exists to avoid
- `DownloadedTilePackageEntity` tracks a single package's status/path/timestamp/size independently
- `RegionEntity` persists every catalog field, links vector and DEM packages via two independent nullable `ToOne`s, and exposes a computed 4-state `RegionStatus get status` getter that can never drift from its source data
- `fromCatalogEntry`/`applyCatalogEntry` implement the upsert-safe merge contract (D-01): insert leaves local-only fields at defaults; apply overwrites only catalog-owned fields and explicitly preserves `obxId`, both `ToOne` targets, and `lastDownloadedVersion`
- 29 new unit tests all green; `dart analyze` clean on every new file; `objectbox-model.json`/`objectbox.g.dart` regenerated additively (only `lastEntityId`/`lastIndexId` counters bumped, no existing entity altered)

## Task Commits

Each task was committed atomically:

1. **Task 1: Status enums + RegionCatalogEntry freezed parse model** - `754da7d1` (feat)
2. **Task 2: DownloadedTilePackageEntity (PackageStatus .code shadow)** - `d2582457` (feat)
3. **Task 3: RegionEntity — catalog fields, .code shadows, computed status getter, catalog mapping** - `fe99217d` (feat)

_All three tasks were TDD-tagged; tests were written alongside each entity/model in the same commit per the plan's action text (fixtures + behavior assertions), and the full suite for all three files passed before each commit._

## Files Created/Modified
- `app/lib/models/region_status.dart` - `CatalogStatus`/`RegionStatus`/`PackageStatus` enhanced enums with explicit `.code`
- `app/lib/models/region_catalog_entry.dart` (+ `.freezed.dart`/`.g.dart`) - freezed parse model for one catalog array element
- `app/lib/entities/downloaded_tile_package_entity.dart` - ObjectBox entity for a single downloaded package
- `app/lib/entities/region_entity.dart` - ObjectBox entity for a catalog region, computed status getter, upsert mapping
- `app/lib/objectbox.g.dart` / `app/lib/objectbox-model.json` - regenerated codegen (additive: new entities registered)
- `app/test/models/region_status_test.dart` - `.code` round-trip coverage for all three enums
- `app/test/models/region_catalog_entry_test.dart` - full-ready/minimal-building/error fixture parsing
- `app/test/entities/region_entity_test.dart` - `DownloadedTilePackageEntity`/`RegionEntity` shadow round-trips, all 4 `status` states, `fromCatalogEntry`/`applyCatalogEntry` preservation and malformed-bbox `FormatException`

## Decisions Made
- Followed 22-CONTEXT.md decisions verbatim (D-01 through D-12); no new gray areas surfaced during implementation
- `switch` on `PackageStatus` in `RegionEntity.status` is exhaustive over all 3 members (no `default` needed) since `PackageStatus` has no `updateAvailable` member — staleness is derived, not a package-status value

## Deviations from Plan

None - plan executed exactly as written. All acceptance criteria matched verbatim (no `.index` usage anywhere in the four new files, `RegionCatalogEntry` has exactly one `@freezed` class, `RegionEntity.status` has no setter/`@Property`).

## Issues Encountered

`dart run build_runner build` regenerated two unrelated riverpod-codegen hash files (`app/lib/provider/route_anchor_provider.g.dart`, `app/lib/provider/router_provider.g.dart`) as a side effect of running codegen across the whole `app/` tree — these are pre-existing generated files outside this plan's `files_modified` scope (no source changes causing them), left untouched/unstaged per the scope-boundary rule; a future build_runner run in any other plan will regenerate them again regardless.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 23 (TileRepositoryManager — Download Engine) can now read/write `RegionEntity`/`DownloadedTilePackageEntity` rows and call `RegionEntity.fromCatalogEntry`/`applyCatalogEntry` once it builds the actual fetch-and-upsert function (D-02, explicitly out of this plan's scope — no call site or Riverpod provider wired yet)
- No blockers - app builds and runs unchanged; nothing reads these entities yet, matching the plan's "purely additive" objective

---
*Phase: 22-region-package-data-model*
*Completed: 2026-07-22*
