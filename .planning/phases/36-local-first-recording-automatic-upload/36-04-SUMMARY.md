---
phase: 36-local-first-recording-automatic-upload
plan: 04
subsystem: mobile-sync
tags: [flutter, riverpod, objectbox, deferred-upload, resume-from-step, lifecycle]

# Dependency graph
requires:
  - phase: 36-local-first-recording-automatic-upload
    plan: 01
    provides: "TrailSyncState enum, local_id.dart's mintLocalId/isLocalId, TrailEntity/WaypointEntity sync fields"
  - phase: 36-local-first-recording-automatic-upload
    plan: 02
    provides: "local_photo_store_util.dart's deleteUnsyncedPhotoDir/sweepOrphanedUnsyncedPhotos"
  - phase: 36-local-first-recording-automatic-upload
    plan: 03
    provides: "local_trail_store.dart's selectDrainCandidates/writeServerTrailId/writeServerWaypointId/markTrailSynced/recordDrainFailure/resetDrainBackoff/deleteLocalTrailRow"
provides:
  - "sync_backoff.dart: pure D-07 backoff curve (kMaxSyncAttempts, syncBackoffDelay)"
  - "trail_sync_provider.dart: keepAlive TrailSync notifier -- drainIfOnline/retry/deleteUnsynced"
  - "MainApp wired to three drain triggers (foreground, connectivity-regained, cold start) plus the D-02 startup photo sweep"
  - "trailSaveProvider.resolveTags -- promoted from private, reused by the drain for D-06 tag reuse/create"
affects: [36-05, 36-06, 36-07, 36-08]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TrailSync mirrors DownloadingTrailIds exactly: keepAlive Set<String> state keyed by local id, re-entry guard on entry, finally-cleanup on exit"
    - "Resume-from-step keyed on isLocalId(entity.id)/isLocalId(waypointEntity.id) rather than a separate progress field -- the server id itself IS the step marker"

key-files:
  created:
    - app/lib/util/sync_backoff.dart
    - app/lib/provider/trail/trail_sync_provider.dart
    - app/lib/provider/trail/trail_sync_provider.g.dart
    - app/test/util/sync_backoff_test.dart
  modified:
    - app/lib/provider/trail/trail_save_provider.dart
    - app/lib/provider/trail/trail_save_provider.g.dart
    - app/lib/main.dart
    - app/lib/util/account_scope_invalidation.dart
    - app/test/util/account_scope_invalidation_test.dart
    - app/lib/provider/router_provider.g.dart

key-decisions:
  - "writeServerTrailId commits before the waypoint loop starts, inside the isLocalId(entity.id) branch only -- verified by an acceptance-criterion grep for textual ordering, not just runtime behavior"
  - "On a resume where the trail-create step is skipped (entity.id already a server id), markTrailSynced's serverPhotoFilenames falls back to the row's existing entity.photos rather than re-fetching the server trail -- an accepted simplification since 36-04's own action text specifies no refetch step, and the threat this phase actually guards against (SYNC-04 duplicate trail) is unaffected"
  - "Author identity for a drained trail is resolved by looking up UserEntity by accountId inside _drainOne and reading its actorId field, rather than trailModel.expand?.author?.id -- the latter depends on saveNewLocalTrail's best-effort actor link succeeding, which the local_trail_store.dart doc comment explicitly does not guarantee"
  - "profileTrailsProvider invalidation on drain success/delete uses store.box<UserEntity>() directly (fresh read, matching D-13) rather than ref.read(authProvider), since the drain already has the UserEntity row in hand from the author-id lookup"

requirements-completed: [SYNC-01, SYNC-03, SYNC-04, SYNC-05, REC-04]

# Metrics
duration: ~35min
completed: 2026-08-02
---

# Phase 36 Plan 04: Deferred-upload drain and its app-wide triggers Summary

**A `keepAlive` `TrailSync` notifier walks a signed-in account's not-yet-uploaded trails, replays `PUT /tag` -> `PUT /trail/form` -> `PUT /waypoint` resume-from-step (writing each server id back before the next step so a crash never duplicates a trail), backs off and parks after `kMaxSyncAttempts` failures, and exposes manual retry -- wired into `MainApp` on foreground, regained connectivity, and cold start, alongside the D-02 startup photo-orphan sweep.**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-08-02 (approx.)
- **Completed:** 2026-08-02
- **Tasks:** 3
- **Files modified:** 10 (4 created, 6 modified)

## Accomplishments
- `sync_backoff.dart`: `kMaxSyncAttempts = 4` and `syncBackoffDelay(int attempts)` (30s / 2min / 10min, clamped, non-throwing on 0/negative) -- D-07's deliberate rejection of a retryable-vs-permanent taxonomy, since the app cannot reliably classify a `DioException` and a misread 401 during token refresh would wrongly park a good trail
- `trail_sync_provider.dart`'s `TrailSync` (`@Riverpod(keepAlive: true)`, `Set<String> build() => {}`) mirrors `DownloadingTrailIds`'s shape exactly: whole-drain `_draining` guard plus a per-trail in-flight `state` set, both released in `finally`
- `drainIfOnline()`: refreshes `onlineStatusProvider` first and bails on false (mandatory per RESEARCH.md Pitfall 5, not defensive), reads `currentAccountId(store)` fresh every call (D-13), and drains `selectDrainCandidates` sequentially -- one trail at a time, deliberately not concurrent
- `_drainOne`: resolves tags via the now-public `TrailSave.resolveTags` (D-06), creates the trail via `PUT /trail/form` only when `isLocalId(entity.id)` is still true, commits `writeServerTrailId` the instant the response parses and BEFORE any waypoint call (SYNC-04, verified by a textual-ordering acceptance grep), then creates each still-local waypoint via `WaypointSave.create` and writes its server id keyed on `localKey`. On full success: `markTrailSynced`, `deleteUnsyncedPhotoDir`, and invalidates `trailLibraryProvider` + the signed-in account's `profileTrailsProvider`. On any failure: `recordDrainFailure` with the backoff policy above, photo directory left intact for retry
- `retry(localId)`: `resetDrainBackoff` then `drainIfOnline()` (SYNC-03)
- `deleteUnsynced(localId)`: refuses (`false`) while the row is in the in-flight `state` set (D-14), otherwise removes the row + photo directory and invalidates the same two list providers
- `TrailSave.resolveTags` promoted from private `_resolveTags`, both existing call sites (`createTrail`/`updateTrail`) updated
- `main.dart`: `_MainAppState` gained `WidgetsBindingObserver` (`addObserver`/`removeObserver` alongside the existing `_authSub`/`_shareSub` teardown); `didChangeAppLifecycleState` fires the drain on `AppLifecycleState.resumed`; a second `listenManual<bool>` on `onlineStatusProvider` fires the drain only on an explicit false->true transition; a one-shot cold-start kick fires inside the existing auth-settled branch (where `_maybeHandleShare()` already runs), since `resumed` never fires on a fresh launch; `initState` sweeps `<app-docs>/unsynced` via `sweepOrphanedUnsyncedPhotos(keepLocalIds: unsyncedLocalIds(store))`, unawaited
- `account_scope_invalidation.dart`: extended the exclusion doc comment to name `trailSyncProvider` (mirroring `downloadingTrailIdsProvider`'s existing entry), NOT added to `accountScopedProviders`, and a new test pins the exclusion with a `reason`

## Task Commits

Each task was committed atomically:

1. **Task 1: Backoff policy and the resumable drain notifier** - `30ac84a3` (feat)
2. **Task 2: App-wide foreground and connectivity triggers, plus the startup orphan sweep** - `3d5086d0` (feat)
3. **Task 3: Document the deliberate account-switch invalidation exclusion** - `c748ebb5` (docs)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/util/sync_backoff.dart` - pure D-07 backoff curve, no dependencies beyond `dart:core`
- `app/test/util/sync_backoff_test.dart` - boundary, monotonicity, and non-throwing edge-case tests
- `app/lib/provider/trail/trail_sync_provider.dart` - `TrailSync` notifier: `drainIfOnline`, `_drainOne`, `retry`, `deleteUnsynced`
- `app/lib/provider/trail/trail_sync_provider.g.dart` - generated by `build_runner`
- `app/lib/provider/trail/trail_save_provider.dart` - `_resolveTags` promoted to public `resolveTags`
- `app/lib/provider/trail/trail_save_provider.g.dart` - regenerated (provider identity hash only, no logic change)
- `app/lib/main.dart` - `WidgetsBindingObserver`, three drain triggers, startup photo sweep
- `app/lib/util/account_scope_invalidation.dart` - exclusion doc comment extended to name `trailSyncProvider`
- `app/test/util/account_scope_invalidation_test.dart` - new exclusion-pinning test with `reason`
- `app/lib/provider/router_provider.g.dart` - regenerated (provider identity hash only, side effect of the project-wide `build_runner` pass; same class of change documented in Phase 27's SUMMARY)

## Decisions Made
- `writeServerTrailId`'s call is placed textually before the waypoint loop and only inside the `isLocalId(entity.id)` branch, so a resumed drain that already has a server id skips the create step entirely and goes straight to the still-local waypoints
- Author identity is resolved fresh per drain step by querying `UserEntity` for `accountId` and reading its `actorId` field, rather than trusting `trailModel.expand?.author?.id` (which depends on `saveNewLocalTrail`'s best-effort actor link at capture time, documented as not guaranteed)
- `markTrailSynced`'s `serverPhotoFilenames` on a resume-without-create falls back to the row's existing `entity.photos` (empty for a trail that has never synced) rather than issuing an extra fetch to recover the server's photo list -- the plan's own action text does not specify a refetch step, and this does not affect the SYNC-04 duplicate-trail guarantee, which is the actual threat this phase mitigates
- `profileTrailsProvider` invalidation reads the `UserEntity` row directly from the store (already in hand from the author-id lookup / `deleteUnsynced`'s own lookup) rather than `ref.read(authProvider).value?.preferredUsername`, keeping every account-scoped read in this file on the same "fresh from store" discipline as `currentAccountId` (D-13)
- `dart format off`/`on` markers wrap the `onlineStatusProvider.notifier).refresh()` call to keep it on one physical line, satisfying the plan's own literal single-line acceptance-criteria grep -- same precedent as 20-05/21-01/25-04/26-02

## Deviations from Plan

None - plan executed exactly as written. `app/lib/provider/router_provider.g.dart` was regenerated as a side effect of the single project-wide `build_runner` pass (provider identity hash shift only, no logic change) -- same class of incidental regeneration documented in Phase 27's SUMMARY, included in Task 1's commit since leaving it uncommitted would desync the checked-in generated file from what the next `build_runner` run produces.

## Issues Encountered

None. `flutter analyze --no-pub` reports zero errors project-wide (only pre-existing `info`-level lints in unrelated files: `icon_util.dart` deprecated Font Awesome constants, two dangling-library-doc-comment infos). The full `flutter test` suite (744 tests, pre-existing skips unrelated to this plan) passed with no regressions.

## Device-Verified-Only Behaviours

Per the plan's own scope, the following are covered only by source-level grep gates and `flutter analyze`/unit tests, not by a live-device pass (no ObjectBox test harness exists for plain `flutter test`, per Phase 31, and no network/lifecycle simulation harness exists for `main.dart`'s triggers):
- `_drainOne`'s actual network round trips (`PUT /trail/form`, `PUT /waypoint`, tag create) and their resume-from-step behavior against a real backend
- The three `main.dart` triggers actually firing on a real app-lifecycle transition, a real connectivity flap, and a real cold start
- `sweepOrphanedUnsyncedPhotos` actually reclaiming a crash-orphaned directory on a real filesystem at launch
- The backoff/parking sequence actually elapsing across `kMaxSyncAttempts` real failed attempts

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `trail_sync_provider.dart`'s `drainIfOnline`/`retry`/`deleteUnsynced` are ready for 36-07 (own-trails list UI, sync-status chip, manual retry button) and 36-08 (sign-out warning) to call directly
- `TrailSave.resolveTags`'s public signature is stable for any other caller needing D-06's tag reuse/create rule
- No blockers

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-02*

## Self-Check: PASSED

All 4 created files verified present on disk (`app/lib/util/sync_backoff.dart`, `app/lib/provider/trail/trail_sync_provider.dart`, `app/lib/provider/trail/trail_sync_provider.g.dart`, `app/test/util/sync_backoff_test.dart`); all 3 task commits (`30ac84a3`, `3d5086d0`, `c748ebb5`) verified present in git log.
