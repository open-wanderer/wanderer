---
phase: quick-260711-lzb
plan: 01
subsystem: map
tags: [go, pocketbase, pmtiles, flutter, maplibre, offline, hillshade, objectbox, freezed]

requires: []
provides:
  - "Per-cell DEM pmtiles extraction against download.mapterhorn.com (Go), tracked independently from the vector cell lifecycle"
  - "GET /map/cells/{cellKey}/download-dem route + dem_download_url in cell status JSON"
  - "demPmTiles threaded through Trail/TrailEntity/TrailDownloadService"
  - "raster-dem special case in offline_style_rewriter.dart that fixes the hillshadeSource-repointed-at-vector-archive bug"
affects: [map, offline-download, navigation]

tech-stack:
  added: []
  patterns:
    - "Per-cell companion-archive pattern: any future per-cell offline asset can clone the DemCellPath/dem_status/download-dem route shape used here"
    - "_rewriteSourceGroup in offline_style_rewriter.dart generalizes the vector cell-duplication machinery so a second source group (raster-dem) reuses it via a pointFn callback"

key-files:
  created:
    - db/migrations/1781900000_added_dem_fields_to_tile_cells.go
  modified:
    - db/services/tiles/generator.go
    - db/routes/map_cells_id.go
    - db/main.go
    - app/lib/models/map_cell.dart
    - app/lib/models/trail.dart
    - app/lib/entities/trail_entity.dart
    - app/lib/services/trail_download_service.dart
    - app/lib/util/offline_style_rewriter.dart
    - app/test/util/offline_style_rewriter_test.dart
    - app/lib/components/base/trail_map.dart
    - app/lib/routes/navigation_screen.dart

key-decisions:
  - "DEM extraction is a fully independent lifecycle from the vector cell (separate dem_status/dem_size_bytes/dem_error_message fields) — a DEM extract failure never blocks vector serving, and pre-existing vector-only cells are backfilled with a DEM on next EnsureCell call rather than requiring a data migration script."
  - "DEM download on the Flutter side is best-effort per cell — a vector cell download failure stays fatal (trail download aborts), but a missing/failed DEM archive is silently skipped for that cell since hillshade is cosmetic."
  - "offline_style_rewriter.dart's cell-duplication logic (previously vector-only) was refactored into a shared _rewriteSourceGroup helper parameterized by a pointFn callback, so the new raster-dem group reuses the exact same cell-0 + extra-cell clone machinery instead of duplicating it."

requirements-completed: [HILLSHADE-OFFLINE-01]

duration: ~40min
completed: 2026-07-11
---

# Quick Task 260711-lzb: Make hillshading work offline in the Flutter app Summary

**Cloned the vector-cell pmtiles pipeline for a companion DEM archive (Go) and fixed the offline style rewriter bug that repointed `hillshadeSource` at the vector `.pmtiles` archive instead of a dedicated DEM archive with terrarium/512 encoding.**

## Performance

- **Duration:** ~40 min
- **Tasks:** 3/3 completed
- **Files modified:** 4 Go files (1 new) + 11 Flutter files (4 generated)

## Accomplishments

- Go: a second `pmtiles extract` per grid cell against `download.mapterhorn.com/planet.pmtiles` at `--maxzoom=12`, with `dem_status`/`dem_size_bytes`/`dem_error_message` tracked independently on `tile_cells`, and `GET /map/cells/{cellKey}/download-dem` serving the archive.
- Flutter: `demPmTiles` threaded through `Trail` (freezed), `TrailEntity` (objectbox), and `TrailDownloadService` — the download loop now pulls a best-effort `${key}_dem.pmtiles` per cell alongside the vector cell.
- Fixed the root bug: `offline_style_rewriter.dart` no longer sweeps `hillshadeSource` into the vector-cell `tiledKeys` path (it was matched purely because it carries a `url` key). `type: raster-dem` sources now route to a dedicated `demCellPaths` param and get `encoding: terrarium` + `tileSize: 512` + `maxzoom: 12` injected (absent from the raw style JSON since Mapterhorn's online tilejson normally supplies them).
- Both offline style callers (`trail_map.dart`, `navigation_screen.dart`) wired to pass `demPmTiles`.
- 6 new tests (single-cell DEM pointing, multi-cell DEM source+layer clone, empty-DEM drop-with-no-leak, 3x DEM path-safety) — all 17 tests in `offline_style_rewriter_test.dart` pass, including all 11 pre-existing (unchanged, no regression).

## Task Commits

Each task was committed atomically:

1. **Task 1: Go — per-cell DEM extraction, tile_cells DEM fields, download-dem route** - `3f67cf37` (feat)
2. **Task 2: Flutter — thread DEM cell paths through model, entity, download loop** - `68501626` (feat)
3. **Task 3 (TDD RED): add failing raster-dem rewriter tests** - `ab2be809` (test)
3. **Task 3 (TDD GREEN): raster-dem special case in the rewriter** - `3d6ff5e1` (fix)
3. **Task 3: wire demCellPaths into both offline style callers** - `d95b2c97` (feat)
3. **Task 3 (cleanup): remove unused import flagged by flutter analyze** - `21a516a4` (refactor)

_TDD task (Task 3) has 4 commits: test (RED) → fix (GREEN) → feat (caller wiring) → refactor (lint cleanup)._

## Files Created/Modified

- `db/migrations/1781900000_added_dem_fields_to_tile_cells.go` - additive migration adding `dem_status`/`dem_size_bytes`/`dem_error_message` to `tile_cells`
- `db/services/tiles/generator.go` - `demMaxZoom=12`, `mapterhornSource` const, `DemCellPath()`, `generateDemCell()`, `EnsureCell` early-return now requires both vector-ready and DEM-ready
- `db/routes/map_cells_id.go` - `MapCellsDownloadDem`, `dem_download_url` in `MapCellsGet`/`MapCellsStatus` ready payloads
- `db/main.go` - registers `GET /{cellKey}/download-dem`
- `app/lib/models/map_cell.dart` - `MapCellStatusResponse.demDownloadUrl`
- `app/lib/models/trail.dart` - `Trail.demPmTiles`
- `app/lib/entities/trail_entity.dart` - `TrailEntity.demPmTiles`, wired into `toModel()`
- `app/lib/services/trail_download_service.dart` - `_downloadMapTiles` returns `(List<String> vector, List<String> dem)`; best-effort DEM download per cell; DEM cache-hit short-circuit
- `app/lib/util/offline_style_rewriter.dart` - `demCellPaths` param, `_offlineDemMaxZoom=12`, `_pointDemSourceAtCell`, `_rewriteSourceGroup` refactor, empty-DEM drop path
- `app/test/util/offline_style_rewriter_test.dart` - fixture extended with `hillshadeSource`/`hillshade` layer, 6 new tests
- `app/lib/components/base/trail_map.dart` - passes `demCellPaths: widget.trail.demPmTiles`
- `app/lib/routes/navigation_screen.dart` - passes `demCellPaths: ref.read(trailProvider(widget.id)).value?.demPmTiles ?? const []` (staged as a targeted hunk — see Deviations)
- `app/lib/objectbox-model.json`, `app/lib/objectbox.g.dart`, `app/lib/models/map_cell.freezed.dart`, `app/lib/models/map_cell.g.dart`, `app/lib/models/trail.freezed.dart`, `app/lib/models/trail.g.dart` - regenerated by `build_runner`

## Decisions Made

- DEM lifecycle kept fully independent from the vector lifecycle (separate PocketBase fields, separate download URL, separate best-effort failure handling) so hillshade — being cosmetic — can never regress the vector basemap.
- `EnsureCell`'s vector `pmtiles extract` is now guarded to only run when the vector file is actually missing (previously it always ran on every non-early-return call), so a DEM-only backfill of a pre-existing vector-ready cell doesn't needlessly re-download the vector archive.
- Refactored the rewriter's per-cell duplication logic into a shared `_rewriteSourceGroup(sources, layerList, layers, keys, paths, pointFn)` helper so the new raster-dem group reuses the exact vector-cell duplication semantics via a callback, rather than a parallel copy-pasted implementation.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed unused `dart:convert` import from the new test file**
- **Found during:** Task 3 (post-implementation `flutter analyze` full-repo sweep)
- **Issue:** An earlier draft of the "empty demCellPaths" test used `jsonEncode(result)` for a leak check; the final version iterates source `url` values directly instead, leaving the import unused.
- **Fix:** Removed the import.
- **Files modified:** `app/test/util/offline_style_rewriter_test.dart`
- **Verification:** `flutter analyze` clean on the file; `flutter test` still 17/17 passing.
- **Committed in:** `21a516a4`

### Scope note (not a deviation, but worth flagging)

`navigation_screen.dart` had pre-existing **uncommitted** unrelated work in the working tree at task start (a `headingUp`-aware location puck, per the dispatch instructions). Per the dispatch constraints, this task's edit (`demCellPaths` wiring inside `_composeStyle`) was staged as a **targeted patch** (`git apply --cached` on an isolated single-hunk diff) rather than `git add`-ing the whole file, so the puck changes remain uncommitted exactly as the user left them. Verified post-commit: `git diff app/lib/routes/navigation_screen.dart` shows only the puck hunks remain unstaged; `git log` for `d95b2c97` shows only the `demCellPaths` hunk.

Similarly, `4` regenerated riverpod `.g.dart` files (`glyph_sprite_cache_provider.g.dart`, `map_style_json_provider.g.dart`, `navigation_stats_provider.g.dart`, `trail/map_cluster_search_provider.g.dart`) picked up unrelated stale-doc-comment drift from running `build_runner` (Task 2's required regen step touches the whole `lib/` tree). These were reverted via `git checkout --` to keep commits scoped, and logged to `.planning/quick/260711-lzb-make-hillshading-work-offline-in-the-flu/deferred-items.md`.

---

**Total deviations:** 1 auto-fixed (1 bug/lint), plus 2 scope-boundary notes (both handled per dispatch constraints, no functional impact).
**Impact on plan:** None on scope. No architectural changes, no scope creep.

## Issues Encountered

None beyond the deviations above.

## Manual Verification Required

**Device smoke test (plan's Task 3 `<human-check>`, A2/A3):** The plan calls for a physical-device airplane-mode test — download a trail online, enable airplane mode, open it, and confirm hillshade relief renders (not blank/garbage). This is a `<human-check>` embedded in an otherwise `type="auto"` task's `<verify>` block, not a `type="checkpoint:*"` task, so execution proceeded without pausing per the plan's own task typing. **This still needs to happen** before the feature is considered field-verified — the research flagged two `[ASSUMED]` items (MapLibre's default raster-dem encoding being `mapbox` so `terrarium` injection is necessary, and native webp-terrarium raster-dem decode from a local pmtiles archive) that only a device test can confirm. If relief renders garbage or blank, capture device logs before deciding on a fallback (RESEARCH.md A2/A3).

## Next Phase Readiness

- All automated verification passed: `go build ./...`, `go vet ./...` (full repo), `flutter test test/util/offline_style_rewriter_test.dart` (17/17), `flutter analyze` clean on all touched files, `dart run build_runner build` clean, lockstep constants confirmed equal (Go `demMaxZoom=12` == Dart `_offlineDemMaxZoom=12`).
- No blockers for closing out this quick task. Pending: the device smoke test noted above (informational, not blocking per the plan's own task typing).

---
*Phase: quick-260711-lzb*
*Completed: 2026-07-11*

## Self-Check: PASSED

All 12 files referenced in this summary confirmed present on disk; all 6 commit hashes (`3f67cf37`, `68501626`, `ab2be809`, `3d6ff5e1`, `d95b2c97`, `21a516a4`) confirmed present in `git log --all`.
