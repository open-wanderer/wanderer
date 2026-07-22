---
phase: 23-tilerepositorymanager-download-engine
plan: 05
subsystem: mobile-offline-tiles
tags: [dart, flutter, riverpod, objectbox, maplibre, bbox-overlap]

# Dependency graph
requires:
  - phase: 23-tilerepositorymanager-download-engine
    provides: "23-04 TileRepositoryManager resumable download engine (startVectorDownload/startDemDownload/pauseRegion/resumeRegion, per-(region,kind) CancelToken map)"
provides:
  - "TileRepositoryManager.localTilePathsForBounds(LngLatBounds) — bbox-overlap viewport query for downloaded region files (TILE-05)"
  - "TileRepositoryManager.deleteRegion(regionId) — cascade delete of package rows + on-disk vector/dem/.part files"
  - "tileRepositoryManagerProvider — construction-only keepAlive Riverpod seam"
  - "TileRepositoryStatus — keepAlive per-region download-state notifier (downloadVector/downloadDem/pause/resume/delete)"
  - "RegionDownloadState — freezed UI-only per-region download state"
affects: ["Phase 24 (Settings — Offline Maps/Regions UI subscribes to TileRepositoryStatus)", "Phase 25 (Map Rendering reads localTilePathsForBounds)", "23-06 (on-device checkpoint verification)"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Negated-disjoint axis-aligned bbox-overlap test (bboxOverlaps): `!(maxLon < query.west || minLon > query.east || maxLat < query.south || minLat > query.north)` — strict `<`/`>` so edge-touching counts as overlap; there is no `intersects()` on this app's `LngLatBounds`"
    - "Cascade delete as one logical unit: package rows removed + region ToOne targets cleared inside one runInTransaction, on-disk files (final path + `.part` sibling for both vector and dem) deleted best-effort outside the transaction with existence guards — ObjectBox does not cascade a ToOne target's removal"
    - "keepAlive notifier wrapping a construction-only manager seam (TileRepositoryStatus -> tileRepositoryManagerProvider), mirroring DownloadingTrailIds -> trailDownloadServiceProvider: idempotent in-flight guard, always-clear-in-finally, immutable map reassignment on every state update"

key-files:
  created:
    - app/lib/models/region_download_state.dart
    - app/lib/models/region_download_state.freezed.dart
    - app/lib/provider/region/tile_repository_provider.dart
    - app/lib/provider/region/tile_repository_provider.g.dart
    - app/test/models/region_download_state_test.dart
  modified:
    - app/lib/services/tile_repository_manager.dart
    - app/test/services/tile_repository_manager_test.dart

key-decisions:
  - "TileRepositoryStatus clears a region's map entry entirely in every method's finally block (`state = {...state}..remove(regionId)`) rather than settling it to a final RegionDownloadState — the authoritative download lifecycle already lives on RegionEntity.status/DownloadedTilePackageEntity (persisted by the manager itself in 23-04), so this ephemeral UI map only needs to track 'currently in flight' state, exactly mirroring DownloadingTrailIds' Set<String> membership-clear pattern rather than inventing a settled-state concept the plan didn't specify"
  - "deleteRegion silently no-ops when the region id resolves to no RegionEntity row (rather than throwing) — deleting an unknown/never-downloaded region has nothing to clean up and isn't a caller error, unlike startVectorDownload/startDemDownload's StateError-on-unknown-region (which guard against downloading something that was never in the catalog)"
  - "deleteRegion also clears region.lastDownloadedVersion (in addition to the plan-specified vectorPackage.target/demPackage.target = null) since 'reset any relevant status' in the plan's action text — a deleted vector package with a stale lastDownloadedVersion would otherwise leave that field pointing at a version no longer reflected by any downloaded file"

requirements-completed: [TILE-01, TILE-05]

# Metrics
duration: 20min
completed: 2026-07-22
---

# Phase 23 Plan 05: TileRepositoryManager Query/Teardown Surface + Riverpod Wiring Summary

**Added `localTilePathsForBounds`/`bboxOverlaps`/`deleteRegion` to `TileRepositoryManager` and wired it into Riverpod via a construction-only `tileRepositoryManagerProvider` seam plus a `keepAlive` `TileRepositoryStatus` notifier exposing per-region `RegionDownloadState`.**

## Performance

- **Duration:** 20 min
- **Started:** 2026-07-22T09:50:00Z
- **Completed:** 2026-07-22T10:10:00Z
- **Tasks:** 2 completed
- **Files modified:** 7 (2 modified, 5 created)

## Accomplishments

- `bboxOverlaps` — pure, `@visibleForTesting` top-level function implementing the negated-disjoint axis-aligned rectangle-overlap test against `LngLatBounds` (no `intersects()` exists on this app's maplibre type); unit-tested for overlapping, disjoint, and edge-touching (strict `<`/`>`, so touching counts as overlap) cases
- `localTilePathsForBounds(LngLatBounds query)` iterates every `RegionEntity`, skips non-overlapping regions via `bboxOverlaps`, and collects non-null `vectorPackage.target?.localFilePath`/`demPackage.target?.localFilePath` — regions with a null package target (not downloaded) contribute nothing
- `deleteRegion(regionId)` cancels any in-flight `:vector`/`:dem` `CancelToken`s for the region, removes both `DownloadedTilePackageEntity` rows and clears the region's `ToOne` targets + `lastDownloadedVersion` inside one `runInTransaction`, then best-effort deletes the vector/dem final paths and their `.part` siblings (existence-guarded) and the region's storage dir if left empty — all outside the transaction, since ObjectBox does not cascade a `ToOne` target's removal
- `RegionDownloadState` — a `@freezed`, JSON-free UI-only value type (`status`, `vectorProgress`, `demProgress`) mirroring `route_anchor.dart`'s no-serialization freezed shape
- `tileRepositoryManagerProvider` — construction-only `@Riverpod(keepAlive: true)` seam building `TileRepositoryManager(store, api)` from `objectBoxProvider`/`apiProvider`, mirroring `regionRepositoryProvider` verbatim (no I/O at build time)
- `TileRepositoryStatus` — `keepAlive` notifier with `Map<String, RegionDownloadState> build() => {}` and `downloadVector`/`downloadDem`/`pause`/`resume`/`delete` methods, each guarding re-entry on an already-`downloading` id, delegating to the manager with a progress callback that reassigns a new map, and clearing the region's entry in a `finally`

## Task Commits

1. **Task 1: localTilePathsForBounds + bboxOverlaps + deleteRegion cascade** - `d5dc4200` (feat)
2. **Task 2: RegionDownloadState model + tile_repository_provider (manager seam + keepAlive status notifier)** - `39a79f59` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `app/lib/services/tile_repository_manager.dart` - Added top-level `bboxOverlaps`, `TileRepositoryManager.localTilePathsForBounds`, `TileRepositoryManager.deleteRegion`; updated class doc comment (TILE-01..05 now complete on this class)
- `app/test/services/tile_repository_manager_test.dart` - Added `bboxOverlaps` group (overlap/disjoint/edge-touching)
- `app/lib/models/region_download_state.dart` (new) - `RegionDownloadState` freezed class
- `app/lib/models/region_download_state.freezed.dart` (generated)
- `app/lib/provider/region/tile_repository_provider.dart` (new) - `tileRepositoryManager` provider + `TileRepositoryStatus` notifier
- `app/lib/provider/region/tile_repository_provider.g.dart` (generated)
- `app/test/models/region_download_state_test.dart` (new) - Equality + default-state tests

## Decisions Made

See `key-decisions` frontmatter for the three decisions made during execution (all Rule-1/interpretation-level, no architectural deviation): the finally-clears-map-entry choice for `TileRepositoryStatus`, `deleteRegion`'s silent no-op on an unknown region, and the extra `lastDownloadedVersion` reset in `deleteRegion`.

## Deviations from Plan

None - plan executed exactly as written. The three items in `key-decisions` are interpretation choices within the plan's stated scope (the plan explicitly said "reset any relevant status" and left `TileRepositoryStatus`'s exact "clear/settle the in-flight marker" wording open), not corrections to broken plan guidance.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `localTilePathsForBounds`/`deleteRegion` are unverified against a live Store by design — Plan 06's on-device checkpoint covers them, matching this codebase's existing precedent (`upsertCatalog` has no unit test either).
- `tileRepositoryManagerProvider`/`TileRepositoryStatus` are ready for Phase 24's Settings/Regions screen to subscribe to; no UI wiring exists yet (out of scope for this plan).
- No blockers identified for Plan 06.

---
*Phase: 23-tilerepositorymanager-download-engine*
*Completed: 2026-07-22*

## Self-Check: PASSED

All 7 created/modified files confirmed present on disk; both task commits (`d5dc4200`, `39a79f59`) confirmed in git log.
