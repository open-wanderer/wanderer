---
phase: quick-260611-whq
plan: 01
subsystem: mobile/offline-map
tags: [flutter, pmtiles, offline-map, dart]
dependency_graph:
  requires: []
  provides: [MultiPmTilesVectorTileProvider]
  affects: [app/lib/components/base/wanderer_map.dart]
tech_stack:
  added: []
  patterns: [fan-out-tile-lookup, parallel-archive-open, union-zoom-range]
key_files:
  created: []
  modified:
    - app/lib/vendor/vector_map_tiles/pm_tile_provider.dart
    - app/lib/components/base/wanderer_map.dart
decisions:
  - "Sequential tile lookup across archives: try each in order, return first hit, 404 only when all miss"
  - "Union zoom range: minimumZoom = min of all archive minZooms, maximumZoom = max of all archive maxZooms"
  - "New class extends same file as existing PmTilesVectorTileProvider rather than a new file"
  - "fromSources static helper opens all archives in parallel via Future.wait"
  - "ArgumentError thrown on empty sources/archives list to fail loudly rather than silently 404ing"
metrics:
  duration: "~8 minutes"
  completed: "2026-06-11T21:43:47Z"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 2
---

# Phase quick-260611-whq Plan 01: Support Multiple PMTiles Sources in Offline Map Summary

**One-liner:** Added `MultiPmTilesVectorTileProvider` with parallel archive opening and sequential tile fan-out, then wired `WandererMap._initOffline` to load all `trail.pmTiles` entries instead of only `pmTiles[0]`.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add MultiPmTilesVectorTileProvider | da0702ac | app/lib/vendor/vector_map_tiles/pm_tile_provider.dart |
| 2 | Wire WandererMap to use all archives | ed7ecf7f | app/lib/components/base/wanderer_map.dart |

## What Was Built

### MultiPmTilesVectorTileProvider (pm_tile_provider.dart)

A new `VectorTileProvider` implementation alongside the existing `PmTilesVectorTileProvider`. Key design:

- **Constructor:** `fromArchives(List<PmTilesArchive> archives, {TileProviderType type})` — takes already-opened archives.
- **Static helper:** `fromSources(List<String> sources)` — opens all sources in parallel via `Future.wait` and constructs the provider.
- **Zoom range:** Union strategy — `minimumZoom` is the minimum across all archive `minZoom` values; `maximumZoom` is the maximum across all `maxZoom` values.
- **Tile lookup:** Sequential fan-out — iterates archives in order, returning the first successful tile. If no archive contains the tile, throws `ProviderException(statusCode: 404)` matching the single-archive convention.
- **Empty guard:** Both `fromArchives` and `fromSources` throw `ArgumentError` on empty input.

### WandererMap update (wanderer_map.dart)

- `_offlineTileProvider` field type changed from `PmTilesVectorTileProvider?` to `MultiPmTilesVectorTileProvider?`
- `_initOffline()` now calls `MultiPmTilesVectorTileProvider.fromSources(widget.trail.pmTiles)` instead of `PmTilesVectorTileProvider.fromSource(widget.trail.pmTiles[0])`
- `_buildTileLayer()` and `_ready` getter required no changes — both already work with the `VectorTileProvider` contract
- `wanderer_offline_map.dart` was explicitly left unchanged per plan scope

## Verification Results

- `dart analyze lib/vendor/vector_map_tiles/pm_tile_provider.dart lib/components/base/wanderer_map.dart` reports no issues
- `grep -c "pmTiles\[0\]" wanderer_map.dart` returns 0 (no single-archive shortcut remains)
- `git diff --name-only HEAD~2 HEAD` shows only `pm_tile_provider.dart` and `wanderer_map.dart` changed
- `wanderer_offline_map.dart` was not touched

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None — all data flows are wired. The `MultiPmTilesVectorTileProvider` reads from actual `PmTilesArchive` instances.

## Threat Flags

None — no new network endpoints, auth paths, or trust boundaries introduced. Changes are local to the tile rendering pipeline.

## Self-Check: PASSED

- da0702ac exists: confirmed in git log
- ed7ecf7f exists: confirmed in git log
- app/lib/vendor/vector_map_tiles/pm_tile_provider.dart modified and committed
- app/lib/components/base/wanderer_map.dart modified and committed
