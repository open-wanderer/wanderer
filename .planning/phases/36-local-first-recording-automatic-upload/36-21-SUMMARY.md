---
phase: 36-local-first-recording-automatic-upload
plan: 21
subsystem: ui
tags: [flutter, riverpod, widget-test, sync-state, trail-detail]

# Dependency graph
requires:
  - phase: 36-08
    provides: SyncStatusChip (four-state sync indicator) and its two existing call sites (trail_list_item.dart, trail_card.dart)
  - phase: 36-19
    provides: retired-local-trail detail-screen redirect and the round-2 UAT run that surfaced this gap
provides:
  - "TrailPanel's detail screen now keys its badge on trail.syncState rather than trail.isLocal cache provenance"
  - "A widget test that mounts the real TrailPanel and pins all four badge branches (pending/uploading/failed/synced) plus a remote control"
affects: [trail-detail-screen, library-detail-screen, sync-status-chip]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Synchronous Auth stub override (UserEntity? build() => ...) to satisfy ref.watch(authProvider).requireValue! in a widget test without an async first-frame gap"
    - "Gate a chip's construction on a domain predicate (isUnsyncedState) rather than relying on the chip's own self-suppression, so a control case's widget tree is provably unchanged (findsNothing, not findsOneWidget-but-empty)"

key-files:
  created:
    - app/test/components/trail/trail_panel_sync_badge_test.dart
  modified:
    - app/lib/components/trail/trail_panel.dart

key-decisions:
  - "Offline pill guard narrowed to `trail.isLocal && !isUnsyncedState(trail.syncState)` rather than replacing it outright -- D-10 guarantees isLocal-and-unsynced and isLocal-and-synced partition every local trail, so narrowing cannot make both badges disappear at once"
  - "SyncStatusChip placed below the title (not inside the existing badge Row) -- that Row has no overflow protection and already holds the date Text; a ~180px failed-state chip there would overflow on a narrow screen"
  - "Chip render gated on isUnsyncedState even though SyncStatusChip self-suppresses for synced trails, specifically so a downloaded trail's widget tree is provably byte-identical (findsNothing) rather than merely visually empty"

requirements-completed: [REC-03, SYNC-02, SYNC-03]

duration: ~15min
completed: 2026-08-03
---

# Phase 36 Plan 21: Detail-screen sync badge gap closure Summary

**An unsynced trail's detail screen (`/trail/local/:localId`) now shows `SyncStatusChip`'s Waiting to upload / Uploading… / Upload failed · Tap to retry instead of the generic Offline badge, with a widget test that mounts the real `TrailPanel` and fails if the fix ever regresses.**

## Performance

- **Duration:** ~15 min
- **Tasks:** 2
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments

- Closed the one open gap from round 2 of `36-UAT.md`: `TrailPanel` (rendered by both `trail_detail_screen.dart` and `library_detail_screen.dart`) previously hardcoded an `if (trail.isLocal)` "Offline" badge that could not distinguish a never-uploaded recording from an ordinary downloaded trail.
- Added a mounted-widget test (`trail_panel_sync_badge_test.dart`) that exercises the real `TrailPanel` for pending, in-flight, failed, downloaded-control, and remote-control cases -- no source-text grepping.
- Narrowed the Offline pill's guard to `trail.isLocal && !isUnsyncedState(trail.syncState)` and added a gated `SyncStatusChip` render directly below the title, matching the placement convention already used by `trail_list_item.dart` and `trail_card.dart`.

## Task Commits

1. **Task 1: RED — mount the real TrailPanel and pin all four badge cases** - `9419f872` (test)
2. **Task 2: GREEN — the detail screen's badge is keyed on sync state, not cache provenance** - `bdfde398` (fix)

_No plan-metadata commit is listed separately here — it follows below per `<final_commit>`._

## Files Created/Modified

- `app/test/components/trail/trail_panel_sync_badge_test.dart` - Mounts the real `TrailPanel` (via a `Trail.empty().copyWith(expand: null, ...)` fixture that structurally avoids the native `maplibre` `TrailMap`), with three provider overrides (`authProvider` synchronous stub, `unitProvider.overrideWithValue`, `trailSyncProvider` in-flight/retry stub). Five cases: pending, in-flight, failed, downloaded control, remote control.
- `app/lib/components/trail/trail_panel.dart` - Added `import 'package:wanderer/components/trail/sync_status_chip.dart'`; narrowed the Offline pill's guard; inserted a `SyncStatusChip` render (gated on `isUnsyncedState(trail.syncState)`) below the title, before the author `InkWell`.

## RED-phase failure messages (Task 1, against the unmodified `trail_panel.dart`)

- **Case A (pending):** `Expected: exactly one matching candidate / Actual: Found 0 widgets with text "Waiting to upload"`
- **Case B (in-flight):** `Expected: exactly one matching candidate / Actual: Found 0 widgets with text "Uploading…"`
- **Case C (failed):** `Expected: exactly one matching candidate / Actual: Found 0 widgets with text "Upload failed · Tap to retry"`
- **Case D (downloaded control):** PASSED unmodified.
- **Case E (remote control):** PASSED unmodified.

This is the exact asymmetry the plan required: A/B/C red, D/E green, before any implementation change.

## Decisions Made

- Kept the Offline pill's markup byte-identical, only narrowing its guard — verified via the falsification runs below that this alone (without the chip render) still fails Case A on the missing chip/text, and leaves D/E untouched.
- Used the plan's fallback-free synchronous-`Auth`-stub approach for `authProvider` (`UserEntity? build() => ...`, a legal covariant override of the generated `FutureOr<UserEntity?> build()`), which published `AsyncData` on the very first frame — the documented fallback (`ProviderContainer` + `await container.read(authProvider.future)`) was not needed.

## Deviations from Plan

None — plan executed exactly as written. No architectural changes, no new dependency, no new l10n key, no schema change.

## Falsifications performed (Task 2 acceptance criteria)

**(a) Revert edit 2 only (remove the `SyncStatusChip` render block, keep the narrowed Offline guard):**
Ran `flutter test test/components/trail/trail_panel_sync_badge_test.dart` — result: **A, B, C failed; D, E passed**, exactly as predicted. Case A's observed failure:
```
Expected: exactly one matching candidate
Actual: Found 0 widgets with text "Waiting to upload": []
```
Restored immediately after observing the result.

**(b) Revert edit 1 only (restore `if (trail.isLocal)` unnarrowed, keep the chip render):**
Ran the same test — result: **A failed (both `'Offline'` renders AND the chip renders); B, C, D, E passed**, exactly as predicted. Case A's observed failure:
```
Expected: no matching candidates
Actual: Found 1 widget with text "Offline": [Text("Offline", ...)]
```
Restored immediately after observing the result.

Both partial-revert falsifications were performed with targeted `Edit` reverts (not `git stash`), verified to fail as predicted, and restored before proceeding to the final commit.

## Issues Encountered

None.

## Verification

- `cd app && flutter test test/components/trail/trail_panel_sync_badge_test.dart` → 5/5 pass (post-fix).
- `cd app && flutter test` → 946 tests passed, 1 pre-existing skip (unrelated to this plan), 0 failures.
- `cd app && flutter analyze --no-pub` → 0 errors (36 pre-existing info-level lints in unrelated files, unchanged).
- `git diff --stat -- app/lib/components/trail/trail_panel.dart` → the only file this plan touched under `app/lib` (the working tree carried four other pre-existing, unrelated uncommitted modifications flagged at session start — `wanderer_sort_chip_group.dart`, `list_list_item.dart`, `trail_list_item.dart`, `profile_screen.dart` — none of which this plan staged, modified, or committed).
- Both Task 1 RED-phase failures and both Task 2 falsifications recorded above with their exact observed messages.

### Human-check (deferred to device pass)

Not performed in this session — this plan's scope is the widget-test-verified UI fix. Per the plan's own `<verification><human-check>`, the on-device confirmation (unsynced trail reads "Waiting to upload"/"Uploading…"/"Upload failed · Tap to retry" and retries on tap; downloaded trail still reads "Offline" unchanged) remains an item for the phase's device-verification pass, same as the rest of `36-VERIFICATION.md`'s device items.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- REC-03, SYNC-02, and SYNC-03 are now satisfied on all three trail surfaces the phase touches: `trail_list_item.dart`, `trail_card.dart`, and (as of this plan) `trail_panel.dart`'s detail screen.
- `library_detail_screen.dart` needed no change — it only renders downloaded (`synced`) trails, so the new gate is a structural no-op there, matching the plan's stated scope.
- No known blockers for the phase's remaining device-verification pass.

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-03*
