---
phase: 23-tilerepositorymanager-download-engine
plan: 04
subsystem: mobile-offline-tiles
tags: [dart, flutter, dio, resumable-download, objectbox, pmtiles, applifecyclelistener]

# Dependency graph
requires:
  - phase: 23-tilerepositorymanager-download-engine
    provides: "23-01 SvelteKit Range-forwarding proxy, 23-02 PackageStatus.paused/error + region_file_path.dart, 23-03 disk_space_util.dart (hasEnoughSpace/freeDiskSpaceBytes)"
provides:
  - "TileRepositoryManager: resumable vector + DEM .part download engine (Range + FileAccessMode.append), disk-space pre-check, PmTilesArchive validation gate, batched ObjectBox status writes, AppLifecycleListener-driven backgrounding pause"
  - "resumePlanFor pure helper (fresh vs resumed download plan), unit-tested"
affects: ["23-05 (Riverpod wiring, localTilePathsForBounds, deleteRegion)", "23-06 (on-device checkpoint verification)"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - ".part file + Range-header resume: File(partPath).lengthSync() gives the resume offset; resumePlanFor(offset) returns the Dio FileAccessMode/deleteOnError/Range-header plan as one pure decision, unit-tested independently of the network call"
    - "Post-download validation before promotion: PmTilesArchive.fromFile(partFile) must succeed before a .part is renamed to its final path or DownloadedTilePackageEntity is marked downloaded"
    - "Per-(region,kind) CancelToken keying ('<id>:vector' / '<id>:dem') so vector and DEM downloads for the same region run concurrently without clobbering each other's cancellation"
    - "AppLifecycleListener in a plain (non-widget) service class for backgrounding-aware pause, distinct cancel reason ('app-backgrounded') from a user-initiated pauseRegion ('paused')"

key-files:
  created:
    - app/lib/services/tile_repository_manager.dart
    - app/test/services/tile_repository_manager_test.dart
  modified: []

key-decisions:
  - "Combined the plan's Task 1 (skeleton/primitive/validation/batched writes) and Task 2 (public start/pause/resume + AppLifecycleListener) into a single commit rather than two — building Task 1 in isolation triggers Dart's unused_element lint on every private helper not yet called from Task 2's code, which would fail Task 1's own 'flutter analyze reports no issues' acceptance criterion. The two tasks are inseparable at the analyzer level for a single new file; splitting the commit would only be cosmetic, not a real per-task verification boundary."
  - "Token keying scheme is '$id:vector' / '$id:dem' (plan left this as an open choice) — lets pauseRegion cancel both kinds for a region via a simple key-prefix match, and lets resumeRegion re-invoke only the paused kind(s)"
  - "_getOrCreatePackage (not explicitly named in the plan's symbol list) added as a small private helper so the create-or-get-then-persist-via-ToOne-cascade logic isn't duplicated across startVectorDownload/startDemDownload"

patterns-established:
  - "Resumable download primitive (resumePlanFor + _downloadResumable) is the first of its kind in this codebase; any future large-file resumable download should reuse this exact FileAccessMode/deleteOnError/Range-header decision shape rather than re-deriving it"

requirements-completed: [TILE-01, TILE-02, TILE-03, TILE-04, DEM-01, DEM-02]

# Metrics
duration: 18min
completed: 2026-07-22
---

# Phase 23 Plan 04: TileRepositoryManager Resumable Download Engine Summary

**TileRepositoryManager downloads a region's vector and DEM `.pmtiles` archives to a `.part` file, resuming from the existing byte offset via HTTP Range + `FileAccessMode.append`, refusing to write when disk space is tight, validating with `PmTilesArchive.fromFile` before promoting `.part` to its final path, and treating app backgrounding as a deliberate `AppLifecycleListener`-driven pause.**

## Performance

- **Duration:** 18 min
- **Started:** 2026-07-22T09:47:00Z
- **Completed:** 2026-07-22T10:05:00Z
- **Tasks:** 2 completed
- **Files modified:** 2 (both created)

## Accomplishments

- `resumePlanFor(existingPartBytes)` — pure, unit-tested decision returning the exact `FileAccessMode`/`sendRange`/`deleteOnError`/`offset` combination for a fresh (0-byte) vs resumed (>0-byte) `.part` file, encoding RESEARCH.md Pitfall 2 (deleteOnError must be false when appending)
- `_downloadResumable` drives every vector/DEM download through that plan, sending `Range: bytes=<offset>-` only when resuming and reporting progress as `offset + received` so callers see the true cumulative byte count
- `_isValidPmTiles` gates `.part` → final-path promotion on `PmTilesArchive.fromFile` succeeding (catches `CorruptArchiveException`/`UnsupportedError`) — a truncated or corrupt archive is deleted and the package marked `error`, never silently accepted
- `startVectorDownload`/`startDemDownload` are fully independent: disk pre-check (`hasEnoughSpace`) refuses with `PackageStatus.error` and zero bytes written when space is tight; only the vector path touches `region.lastDownloadedVersion` (DEM has no staleness concept); a DEM failure never marks the vector package
- `pauseRegion`/`resumeRegion` cancel/re-invoke by region id; `AppLifecycleListener(onPause: _pauseAllActiveDownloads)` cancels every active `CancelToken` with a distinct `'app-backgrounded'` reason on backgrounding, preserving every `.part` file for later resume
- Request paths are built as `/regions/<validated id>/download[-dem]` from the validated id — never from the catalog's `/api/v1`-prefixed `vectorUrl`/`demUrl` strings, avoiding a double-prefixed 404 and keeping the request path off any catalog-sourced string

## Task Commits

Both tasks landed in a single commit (see Decisions Made for why):

1. **Task 1 + Task 2: Manager skeleton, resumable primitive, disk check, validation, batched writes, public start/pause/resume, AppLifecycleListener backgrounding pause** - `b167a1ab` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `app/lib/services/tile_repository_manager.dart` - `TileRepositoryManager` class: `resumePlanFor` (top-level, `@visibleForTesting`), `startVectorDownload`/`startDemDownload`/`pauseRegion`/`resumeRegion`/`dispose` (public), `_regionById`/`_requestPathFor`/`_downloadResumable`/`_isValidPmTiles`/`_getOrCreatePackage`/`_updatePackageStatus`/`_pauseAllActiveDownloads` (private helpers)
- `app/test/services/tile_repository_manager_test.dart` - Unit tests for `resumePlanFor`'s fresh (0-byte) and resumed (1048576-byte) cases

## Decisions Made

- Combined Task 1 and Task 2 into one commit — see `key-decisions` frontmatter for the full rationale (Task 1 in isolation fails its own `flutter analyze` acceptance criterion due to Dart's `unused_element` lint on private helpers not yet called).
- Token keying scheme `'$id:vector'` / `'$id:dem'` chosen (plan explicitly left this as an open choice) so vector and DEM downloads for the same region can run concurrently without clobbering each other's `CancelToken`, and `pauseRegion` can cancel both with a simple key-prefix match.
- `_getOrCreatePackage` added as a small private helper (not explicitly named in the plan's symbol list) to avoid duplicating the create-or-get-then-persist-via-`ToOne`-cascade logic between `startVectorDownload` and `startDemDownload`. Persisting a newly-created package goes through `RegionEntity`'s own box `put()` (ObjectBox cascades a new, unstored `ToOne` target automatically, per the `objectbox` package's own documented behavior), while subsequent status-only updates go through `_updatePackageStatus`'s `DownloadedTilePackageEntity` box `put()`.
- Directory creation (`Directory(regionStorageDir(root, id)).createSync(recursive: true)`) happens in `startVectorDownload`/`startDemDownload` (where `root`/`id` are in scope) rather than inside `_downloadResumable`, since that helper's signature (per the plan's exact symbol spec) only takes `requestPath`/`partPath`/`cancelToken`/`onProgress`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Task 1/Task 2 commit split abandoned in favor of a single combined commit**

- **Found during:** Attempting to write Task 1's file content in isolation before Task 2's additions, per the atomic-per-task-commit protocol.
- **Issue:** `flutter analyze` on a Task-1-only version of `tile_repository_manager.dart` (skeleton + private helpers, no public methods yet) reported 8 warnings: unused imports (`path_provider`, `disk_space_util.dart`, `region_file_path.dart`) and unused-declaration warnings for every private helper (`_regionById`, `_requestPathFor`, `_downloadResumable`, `_isValidPmTiles`, `_updatePackageStatus`) and the `_activeCancelTokens` field, none of which are referenced until Task 2 adds the public methods that call them. This would fail Task 1's own literal acceptance criterion (`flutter analyze lib/services/tile_repository_manager.dart` reports no issues).
- **Fix:** Restored the full Task 1 + Task 2 combined file (as originally authored) and committed both tasks' work in a single `feat` commit, since the plan's two tasks are not independently analyzer-clean for this particular new-file split.
- **Files modified:** `app/lib/services/tile_repository_manager.dart`
- **Verification:** `flutter analyze lib/services/tile_repository_manager.dart test/services/tile_repository_manager_test.dart` reports no issues; `flutter test test/services/tile_repository_manager_test.dart` passes both `resumePlanFor` cases.
- **Committed in:** `b167a1ab`

---

**Total deviations:** 1 auto-fixed (1 blocking — commit-granularity adjustment only, no code/behavior change)
**Impact on plan:** No functional deviation from the plan's specified implementation — both tasks' code landed exactly as specified, only the commit boundary changed.

## Issues Encountered

None beyond the commit-granularity issue documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `TileRepositoryManager` is ready for Plan 05 to wrap in a Riverpod `keepAlive` provider (mirroring `trail_download_state_provider.dart`) and add `localTilePathsForBounds`/`deleteRegion`.
- `startVectorDownload`/`startDemDownload`/`pauseRegion`/`resumeRegion` are unverified against a live network/on-device backgrounding signal by design — that verification is explicitly scoped to Plan 06's on-device checkpoint, matching this codebase's existing precedent (`upsertCatalog` has no unit test either).
- No blockers identified for Plan 05.

---
*Phase: 23-tilerepositorymanager-download-engine*
*Completed: 2026-07-22*
