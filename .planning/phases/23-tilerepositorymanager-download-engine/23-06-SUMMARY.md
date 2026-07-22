---
phase: 23-tilerepositorymanager-download-engine
plan: 06
subsystem: mobile-offline-tiles
tags: [dart, flutter, riverpod, objectbox, maplibre, on-device-harness]

# Dependency graph
requires:
  - phase: 23-tilerepositorymanager-download-engine
    provides: "23-05 tileRepositoryManagerProvider (construction-only seam) + TileRepositoryStatus keepAlive notifier + localTilePathsForBounds/deleteRegion"
provides:
  - "app/test/services/tile_repository_manager_harness.dart -- standalone on-device Flutter entry point exercising every public TileRepositoryManager method (startVectorDownload/startDemDownload/pauseRegion/resumeRegion/deleteRegion/localTilePathsForBounds) against a real region and live ObjectBox Store"
  - "End-of-phase human-check checklist (5 behaviors: TILE-02 resume, TILE-03 disk refusal, TILE-04 backgrounding pause, DEM-01/02 independence, TILE-05 bbox query) recorded in this plan's <verify><human-check> block for the project's human_verify_mode: end-of-phase workflow"
affects: ["Phase 23 end-of-phase verification pass (human sign-off on the 5 on-device behaviors)"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Standalone debug-driver entry point pattern: a second `main()` under app/test/ launched via `flutter run -t test/services/<name>.dart` (not `flutter test`), replicating main.dart's ObjectBox/cookie-jar bootstrap + ProviderScope overrides, kept fully out of router_provider.dart/production navigation -- reusable for any future on-device-only verification harness in this codebase"

key-files:
  created:
    - app/test/services/tile_repository_manager_harness.dart
  modified: []

key-decisions:
  - "Harness drives TileRepositoryManager directly via tileRepositoryManagerProvider (not through the TileRepositoryStatus notifier) for all download/pause/resume/delete/query actions, because TileRepositoryStatus's onProgress closure collapses received/total bytes into a UI-only progress fraction (RegionDownloadState.vectorProgress/demProgress) -- the plan's acceptance criteria explicitly requires debugPrinting raw received/total byte counts, which only a direct manager call preserves. TileRepositoryStatus is referenced in a doc comment for context (satisfying the plan's key_link grep on the string 'tileRepositoryManager') but not driven, since acceptance criteria explicitly allows 'TileRepositoryStatus notifier equivalents' as an alternative, not a requirement."
  - "Added a 'Backend base URL' TextField + Connect button wired to apiProvider.notifier.updateBaseUrl (Rule 2 -- missing critical functionality): the harness's own main() builds a fresh ProviderScope outside the normal main.dart/auth_provider bootstrap that normally calls updateBaseUrl from a saved user session, so without this the Dio client would stay pointed at the api_provider.dart placeholder 'https://unknown-server.local' and every catalog/download call would fail immediately on a physical device."
  - "localTilePathsForBounds inside/outside bbox fixtures are computed from each region's own bbox (center-quarter rectangle for 'inside', bbox+10deg offset rectangle for 'outside') rather than a hardcoded LngLatBounds literal, since the plan's regions are backend/catalog-dependent and a hardcoded fixture could accidentally sit outside every real region's bbox"

requirements-completed: [TILE-02, TILE-03, TILE-04, TILE-05, DEM-01, DEM-02]

# Metrics
duration: 15min
completed: 2026-07-22
---

# Phase 23 Plan 06: On-Device Driver Harness + End-of-Phase Human-Check Summary

**Standalone `flutter run -t`-launched debug driver exercising every public `TileRepositoryManager` method against a real region, plus the recorded 5-behavior end-of-phase human-check for TILE-02/03/04, DEM-01/02, and TILE-05.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-07-22T10:10:00Z
- **Completed:** 2026-07-22T10:25:00Z
- **Tasks:** 1 completed
- **Files modified:** 1 (created)

## Accomplishments

- `app/test/services/tile_repository_manager_harness.dart` -- a standalone Flutter entry point (own `main()`, own `ProviderScope`, launched via `flutter run -t test/services/tile_repository_manager_harness.dart` rather than through the shipped app or `flutter test`) that:
  - Bootstraps ObjectBox + a persisted cookie jar identically to `main.dart`, plus a "Backend base URL" field wired to `apiProvider.notifier.updateBaseUrl` so a tester can point it at any reachable Wanderer backend
  - Refreshes the region catalog via `regionRepositoryProvider.refreshCatalog()` and lists every persisted `RegionEntity` with its computed `RegionStatus` and DEM `PackageStatus`
  - Wires per-region buttons directly to `tileRepositoryManagerProvider`'s `startVectorDownload`, `startDemDownload`, `pauseRegion`, `resumeRegion`, and `deleteRegion`
  - `debugPrint`s the raw `received`/`total` byte counts (plus percentage) on every `onProgress` callback, and the resulting `RegionStatus`/`PackageStatus` after each action -- the exact signal a tester watches during an interrupt/resume cycle
  - Exposes "Query inside bbox" / "Query outside bbox" buttons per region that call `localTilePathsForBounds` with a computed center-quarter (inside) or +10deg-offset (outside) `LngLatBounds` fixture and `debugPrint`/render the result
- Recorded the plan's 5-item `<verify><human-check>` block (already present in `23-06-PLAN.md`) as the artifact this project's `human_verify_mode: end-of-phase` workflow will present at the phase's end-of-phase verification pass: TILE-02 (Range resume, must hit `206` not `200`), TILE-03 (disk refusal blocks with no partial file), TILE-04 (backgrounding shows `paused` and resumes cleanly), DEM-01/02 (independent toggle/delete), TILE-05 (bbox query hits/misses correctly)

## Task Commits

1. **Task 1: Build a minimal on-device driver harness and record the end-of-phase on-device human-check** - `71ac974a` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `app/test/services/tile_repository_manager_harness.dart` (new) - Standalone Flutter entry point + `ConsumerStatefulWidget` debug screen driving every public `TileRepositoryManager` method against a real region/backend

## Decisions Made

See `key-decisions` frontmatter for the three decisions made during execution: driving the manager directly (not via `TileRepositoryStatus`) to preserve raw byte counts for `debugPrint`, adding a backend base-URL connect control (Rule 2 -- without it the harness's isolated `ProviderScope` would never point Dio at a real server), and computing bbox fixtures from each region's own bbox rather than a hardcoded literal.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added a backend base-URL input + Connect button**
- **Found during:** Task 1 (harness construction)
- **Issue:** The harness's `main()` builds a fresh `ProviderScope` independent of the shipped app's `main.dart`/`auth_provider.dart` bootstrap, which is the only place that normally calls `apiProvider.notifier.updateBaseUrl(...)` (from a saved user session). Without an equivalent call, `apiProvider`'s `Dio` client stays pointed at the hardcoded placeholder `"https://unknown-server.local"` (see `api_provider.dart`), so `refreshCatalog()` and every download call would fail immediately with a DNS/connection error on a physical device -- the harness would be non-functional as delivered.
- **Fix:** Added a `TextField` (defaulting to the Android-emulator-loopback `http://10.0.2.2:8090`) and a "Connect" button calling `ref.read(apiProvider.notifier).updateBaseUrl(...)` before the tester taps refresh.
- **Files modified:** `app/test/services/tile_repository_manager_harness.dart`
- **Verification:** `flutter analyze` clean; manual code-path review confirms `updateBaseUrl` mutates the same `Dio` instance `regionRepositoryProvider`/`tileRepositoryManagerProvider` read via `apiProvider`.
- **Committed in:** `71ac974a` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Necessary for the harness to be usable at all on a physical device against a real backend; no scope creep beyond making the delivered artifact actually functional.

## Issues Encountered

None.

## User Setup Required

None for building/analyzing this plan's artifact. **Live on-device execution of the 5-behavior human-check is a required human follow-up step**, deferred to this project's end-of-phase verification pass (`human_verify_mode: end-of-phase` per `.planning/config.json`) rather than a mid-flight gate -- per this plan's `<checkpoint_note>`, no `checkpoint:human-verify` was raised during this execution. To actually run the check: launch `flutter run -t test/services/tile_repository_manager_harness.dart` on a physical device against a backend whose region catalog has at least one `ready` region with a non-trivial archive size, and walk through the 5 items in `23-06-PLAN.md`'s `<verify><human-check>` block (RESUME, DISK REFUSAL, BACKGROUNDING, DEM INDEPENDENCE, QUERY).

## Next Phase Readiness

- Every public `TileRepositoryManager` method now has both unit coverage (pure seams, Plans 02-05) and an on-device driver (this plan) -- Phase 23's download engine is code-complete pending the human sign-off described above.
- No blockers for Phase 24 (Settings -- Offline Maps/Regions UI), which subscribes to `tileRepositoryManagerProvider`/`TileRepositoryStatus` directly and does not depend on this harness.
- The harness itself is a permanent verification tool, not throwaway -- it can be re-run for regression-checking TileRepositoryManager behavior after any future change to the download/resume/disk-space/backgrounding logic.

---
*Phase: 23-tilerepositorymanager-download-engine*
*Completed: 2026-07-22*

## Self-Check: PASSED

`app/test/services/tile_repository_manager_harness.dart` confirmed present on disk; commit `71ac974a` confirmed in git log; `flutter analyze test/services/tile_repository_manager_harness.dart` reports no issues; grep confirms zero references to the harness file in `lib/provider/router_provider.dart` or `lib/main.dart`.
