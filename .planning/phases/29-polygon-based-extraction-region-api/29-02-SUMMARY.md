---
phase: 29-polygon-based-extraction-region-api
plan: 02
subsystem: backend
tags: [go, pocketbase, regions-table, region-api, hierarchy]

# Dependency graph
requires:
  - phase: 29-polygon-based-extraction-region-api
    plan: 01
    provides: "Relaxed regionIDPattern/IsValidRegionID (allows '.'/'\'' with a '..' traversal guard) and region_archives.region_id keyed by regions.path"
provides:
  - "GET /api/v1/regions returns the full group+leaf regions-table catalog with id/name/kind/parent/path/depth on every row"
  - "Leaf rows additionally carry bbox/enabled plus the existing build-state shape (status/version/vector_url/vector_size/dem_status/dem_url/dem_size/error), joined to region_archives on region_id == path"
  - "region_config.json / REGION_CATALOG_CONFIG_PATH parsing fully retired (LoadRegionCatalog/Region/ValidateRegion deleted)"
affects: [31-flutter-settings-hierarchy]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "RegionsList reads the seeded regions PocketBase table directly via FindAllRecords, branching entry shape on kind==leaf rather than reading an admin config file"

key-files:
  created: []
  modified:
    - db/routes/regions_get.go
    - db/services/regions/config.go
    - db/services/regions/config_test.go

key-decisions:
  - "Group rows emit only {id, name, kind, parent, path, depth} — no bbox/status/build-state — matching 29-RESEARCH.md's recommendation so Phase 31's Flutter tree gets group labels; the pre-Phase-31 Flutter parser (RegionCatalogEntry.fromJson) safely ignores these entries since it requires bbox/status"
  - "Leaf entries keep 'id' as the PocketBase record id (not path) so a client links children to parents by matching a row's parent against another row's id; path is still exposed as its own field and used for the region_archives join and download URLs"
  - "writeFile test helper removed alongside TestLoadRegionCatalog since it had no other caller after the loader's tests were deleted"

patterns-established:
  - "A `kind` discriminator branch (group vs leaf) inside a single FindAllRecords loop cleanly layers hierarchy-only fields under group rows and hierarchy-plus-build-state fields under leaf rows, without two separate query passes"

requirements-completed: [EXTRACT-02, EXTRACT-03]

# Metrics
duration: ~15min
completed: 2026-07-26
---

# Phase 29 Plan 2: Hierarchy-Aware Region API & Config Loader Retirement Summary

**`GET /api/v1/regions` now reads the seeded `regions` PocketBase table directly (both group and leaf rows) and emits `parent`/`path`/`depth`/`kind` on every entry, while `region_config.json` parsing (`LoadRegionCatalog`/`Region`/`ValidateRegion`) is deleted entirely from the backend.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-07-26T17:55:19Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Rewrote `RegionsList` (`db/routes/regions_get.go`) to iterate `e.App.FindAllRecords("regions")` instead of `regions.LoadRegionCatalog()`, returning every row (group + leaf) with hierarchy fields; leaf rows additionally carry `bbox`/`enabled` and the existing build-state shape, now joined to `region_archives` on `region_id == r.GetString("path")`
- Deleted `LoadRegionCatalog`, the `Region` struct, and `ValidateRegion` from `db/services/regions/config.go`, along with their now-unused `encoding/json`/`fmt`/`log`/`os` imports, retiring the `region_config.json` admin-config loader entirely (EXTRACT-02)
- Removed `TestLoadRegionCatalog` and its now-orphaned `writeFile` test helper from `db/services/regions/config_test.go`; id-validation and path-builder tests (`TestIsValidRegionID`, `TestRegionIDCannotProduceTraversalPath`, `TestRegionArchivePath`, `TestRegionDemPath`) retained unchanged

## Task Commits

Each task was committed atomically:

1. **Task 1: Hierarchy-aware RegionsList reading the regions table** - `3f797f4d` (feat)
2. **Task 2: Delete the retired region_config.json loader** - `320d7196` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `db/routes/regions_get.go` - `RegionsList` now queries `regions` directly, branches leaf-vs-group entry shape on `kind`, and joins `region_archives` on the leaf's `path`; doc comment updated to reflect the new source of truth
- `db/services/regions/config.go` - `LoadRegionCatalog`/`Region`/`ValidateRegion` removed; package doc comment updated; `RegionCacheDir`, `regionIDPattern`, `IsValidRegionID`, `RegionArchivePath`, `RegionDemPath` retained unchanged
- `db/services/regions/config_test.go` - `TestLoadRegionCatalog` and its `writeFile` helper removed; all other tests retained unchanged

## Decisions Made
- Group rows are hierarchy-only (`id`/`name`/`kind`/`parent`/`path`/`depth`) with no `bbox`/`status`/build-state — an intentional, backward-safe choice since the shipped Flutter `RegionCatalogEntry.fromJson` requires `bbox`/`status` and silently drops entries missing them, so group rows are ignored by the pre-Phase-31 app rather than crashing it
- Leaf entry `id` stays the PocketBase record id (not `path`) so `parent` (a relation field's raw value) can be matched against sibling rows' `id` to reconstruct the tree client-side; `path` remains present as its own field for the `region_archives` join and download URL construction
- Removed the `writeFile` test helper together with `TestLoadRegionCatalog` since no other retained test referenced it (confirmed via grep before removal)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `db` package builds, vets, and tests clean (`go build -C db ./...`, `go vet -C db ./...`, `go test -C db ./services/regions/`)
- `GET /api/v1/regions` now exposes the hierarchy fields (`parent`/`path`/`depth`/`kind`) that Phase 31's Flutter Settings Hierarchy work depends on; leaf-entry shape (`id`/`name`/`bbox`/`status`/...) is unchanged and additive, so the shipped Flutter offline-regions feature is not regressed
- `LoadRegionCatalog`/`Region`/`ValidateRegion` no longer exist anywhere in the repo (EXTRACT-02 fully closed) — confirmed via repo-wide grep
- Cross-phase flag carried forward from the plan's own notes (not actioned here, backend-only phase): Phase 31 must relax `app/lib/util/region_file_path.dart`'s `regionIdPattern` in lockstep with 29-01's backend regex change, since region ids can now contain `.`/`'` once an admin enables a materialized-path leaf
- Live response-shape backward-compatibility and hierarchy-field presence are deferred to the 29-04 human-verify checkpoint (curl `GET /api/v1/regions`), per this plan's own `<verification>` section — not re-proven by unit tests here

## Self-Check: PASSED

- `db/routes/regions_get.go` contains `FindAllRecords("regions")` (1 occurrence) ✓
- `db/routes/regions_get.go` emits all four hierarchy keys (`parent`/`path`/`depth`/`kind`, 6 occurrences across the two keyed literals) ✓
- `db/routes/regions_get.go` contains zero non-comment occurrences of `LoadRegionCatalog` ✓
- `db/services/regions/config.go` contains zero non-comment occurrences of `func LoadRegionCatalog`, `func ValidateRegion`, `type Region ` ✓
- `db/services/regions/config.go` retains `func IsValidRegionID` and `func RegionArchivePath` (1 occurrence each) ✓
- Repo-wide: zero non-test files reference `LoadRegionCatalog` ✓
- `git log --oneline --grep="29-02"` — not applicable (commits use `feat(29-02):` prefix, verified via `git log --oneline -3` showing both task commits) ✓
- `go build -C db ./...`, `go vet -C db ./...`, `go test -C db ./services/regions/` all pass ✓

---
*Phase: 29-polygon-based-extraction-region-api*
*Completed: 2026-07-26*
