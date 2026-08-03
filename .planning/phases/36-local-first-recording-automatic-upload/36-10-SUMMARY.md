---
phase: 36-local-first-recording-automatic-upload
plan: 10
subsystem: ui
tags: [riverpod, objectbox, flutter, trail-create]

# Dependency graph
requires:
  - phase: 36-07
    provides: profile_trails_provider.dart's local-first own-trails merge, and the `profileTrailsProvider('@<handle>')` family key the invalidation must match
  - phase: 36-09
    provides: the `.select`/`skipLoadingOnReload`/`copyWithPrevious` render path that keeps the own-trails list visible during a seamless (non-reload) refresh, guaranteeing this plan's invalidation doesn't reintroduce a spinner flash
provides:
  - "_invalidateOwnTrailsList(), the only explicit propagation path from an ObjectBox row write to the own-trails list (this app has no ObjectBox Query.watch() streams)"
  - "Both trail_create_screen.dart save tails (_finishLocalSave, _saveViaNetwork) notifying trailLibraryProvider + profileTrailsProvider('@<handle>') on success"
affects: [profile, trail-create, offline-ux]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A private _invalidate*() helper on a ConsumerState, called from every save/mutation tail that writes to a store with no reactive stream, spelled identically to the sibling invalidation an equivalent async path (drain) already performs"

key-files:
  created: []
  modified:
    - app/lib/routes/trail_create_screen.dart
    - app/test/routes/trail_create_screen_local_save_gate_test.dart

key-decisions:
  - "No ownTrailsHandle/ownTrailsInvalidationTargets extraction into app/lib/util/ -- an earlier draft's lib-wide gate was dropped by explicit decision; the inline spelling matches trail_sync_provider.dart's today and the phase is verified on device (recorded in 36-10-PLAN.md, not reopened during execution)"
  - "No null guard on the '@${...preferredUsername}' interpolation -- the only null window (mid-logout) is unreachable from every route that mounts this screen, and even then the result is a no-op invalidate on an unwatched key"
  - "No ownHandleProvider (@Riverpod(keepAlive: true) derived handle) -- would force a build_runner run this wave cannot take (36-11 runs concurrently and the lock is package-wide), and no call site needs reactivity"

requirements-completed: [REC-05, REC-06]

# Metrics
duration: ~15min
completed: 2026-08-03
---

# Phase 36 Plan 10: Both save tails notify the own-trails list Summary

**Closes UAT gap 3: `_finishLocalSave` and `_saveViaNetwork` now both call a new `_invalidateOwnTrailsList()` that invalidates `trailLibraryProvider` + `profileTrailsProvider('@<handle>')` -- the exact pair `trail_sync_provider.dart` invalidates after a successful drain -- so an offline (or online) edit to a trail shows up in the own-trails list on pop-back with no manual pull-to-refresh.**

## Performance

- **Duration:** ~15 min
- **Tasks:** 1 completed
- **Files modified:** 2

## Accomplishments
- `trail_create_screen.dart` gained `_invalidateOwnTrailsList()`, a private `ConsumerState` method that reads the signed-in username fresh at call time (D-13) and invalidates `trailLibraryProvider` plus `profileTrailsProvider('@<username>')`.
- `_finishLocalSave` (the shared tail of both local-first branches, `createLocal`/`updateLocal`) now calls it as its first statement, guarded by `if (mounted)`, before the existing `drainIfOnline()` fire-and-forget.
- `_saveViaNetwork` (server-side save, reached by `LocalSaveMode.networkUpdate` and the `LocalUpdateOutcome.alreadySynced` escape hatch) now calls it immediately after the success `setState` that assigns `trail = result.trail`.
- `trail_create_screen_local_save_gate_test.dart` gained three new call-site placement assertions (`_finishLocalSave` calls it, `_saveViaNetwork` calls it, the method itself invalidates the right pair), and its `onSaveEnd` helper was generalized into a reusable `methodEnd(int start)` used by all six assertions in the file now.

## Task Commits

Each task was committed atomically:

1. **Task 1: Both save tails notify the own-trails list** - `33953ff1` (feat)

## Files Created/Modified
- `app/lib/routes/trail_create_screen.dart` - Added `profile_trails_provider.dart` and `trail_library_provider.dart` imports; added `_invalidateOwnTrailsList()`; wired it into `_finishLocalSave` and `_saveViaNetwork`
- `app/test/routes/trail_create_screen_local_save_gate_test.dart` - Generalized `onSaveEnd` into `methodEnd`; added three placement assertions for invariant 3; extended header doc comment with the new invariant and an explicit scope note

## Decisions Made
- No architectural extraction (`ownTrailsHandle`/`ownHandleProvider`) -- these were considered and rejected in the plan itself before execution started; not reopened.
- The gate test's scope note was written explicitly into the file header per the plan's instruction, distinguishing this gate (pins reachable, UAT-proven placement) from `trail_dropdown_delete_gate_test.dart`'s gate (pinned unreachable UI shape).

## Deviations from Plan

None - plan executed exactly as written. The plan's own literal action text (imports, method body, call-site placement, test assertions, `methodEnd` rename) compiled and analyzed cleanly with no adjustments needed.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `flutter analyze --no-pub` on the full project reports zero errors (35 pre-existing `info`-level issues in unrelated files: `icon_util.dart` deprecated FontAwesome constants, two `dangling_library_doc_comments`, one `unnecessary_import` -- none in this plan's `files_modified`, out of scope per the deviation rules' scope boundary).
- `flutter test` (full suite, 833 tests) passes with this change in place.
- Device re-test (this plan's PRIMARY verification signal per its `<verification>` block) still needed: airplane-mode edit-and-pop-back on an unsynced trail, and an online edit-and-pop-back on an already-uploaded trail, confirming the own-trails list updates with no pull-to-refresh and no spinner flash. `flutter test` cannot mount this screen (no ObjectBox store, router, image picker, or map controller), so this is the only way to confirm propagation, not just call-site placement.
- No blockers for subsequent Phase 36 plans.

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-03*

## Self-Check: PASSED

All modified files verified present on disk; task commit hash (`33953ff1`) verified present in git history.
