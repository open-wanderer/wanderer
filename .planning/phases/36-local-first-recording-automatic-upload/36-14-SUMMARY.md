---
phase: 36-local-first-recording-automatic-upload
plan: 14
subsystem: mobile-offline-sync
tags: [flutter, objectbox, riverpod, local-first, trail-sync]

# Dependency graph
requires:
  - phase: 36-local-first-recording-automatic-upload (plans 10, 11, 13)
    provides: own-trails list wiring, local-trail route, dropdown delete gating
provides:
  - "retireUploadedLocalTrail: deletes (or demotes) a local capture row the instant its upload completes, closing UAT gap 4"
  - "shouldDeleteUploadedRow / resolveLocalSaveModeForRow: the two pure decisions the retirement flow is built on"
  - "trail_create_screen's save routing now treats a retired row as 'uploaded', never a silent local write"
affects: [36-local-first-recording-automatic-upload]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Retire-not-retain: a successful upload deletes the local capture row (or demotes it to an ordinary downloaded row when some account holds it in its offline library) instead of marking it synced and keeping it"
    - "Row-first, files-second bookkeeping order in the drain so a crash between the two self-heals via the startup orphan sweep"

key-files:
  created:
    - app/test/util/local_trail_retirement_gate_test.dart
  modified:
    - app/lib/util/local_trail_store.dart
    - app/lib/provider/trail/trail_sync_provider.dart
    - app/lib/provider/trail/trail_sync_provider.g.dart
    - app/lib/routes/trail_create_screen.dart
    - app/test/util/local_trail_store_test.dart
    - app/test/routes/trail_create_screen_local_save_gate_test.dart

key-decisions:
  - "2026-08-03 product decision (superseding the 2026-08-02 'local rows survive promotion' call): a local trail row is deleted once it uploads successfully. No row survives a successful upload, so the orphan class UAT gap 4 found cannot exist."
  - "trail_dropdown.dart is untouched -- the earlier owner-scoped, server-id-keyed cleanup approach from 36-14's first draft was dropped in favor of retiring the row at its root cause."
  - "36-15 was dropped entirely (not reassigned): its retry-policy/chrome-scaffold work existed only to handle the 404 this plan's retirement makes impossible to reach that way."

requirements-completed: [REC-06, SYNC-05]

# Metrics
duration: ~20min
completed: 2026-08-03
---

# Phase 36 Plan 14: Retire local trail rows on successful upload Summary

**A trail's local capture row is deleted (or demoted to an ordinary downloaded row) the instant its upload finishes, closing the permanent post-delete orphan UAT gap 4 found -- with the create screen taught that a missing row means "it uploaded" so a post-upload edit never reports false success.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-08-03T12:05Z
- **Tasks:** 4/4
- **Files modified:** 6 (1 new test file)

## Accomplishments

- `retireUploadedLocalTrail` retires a capture row inside one write transaction: deletes it (and its `WaypointEntity` children) in the ordinary case, or demotes it (clears `owner`/`localId`/sync bookkeeping) when some account holds the trail in its offline library -- never destroying that account's `library/<id>/` entry.
- The drain's step 4 now calls `retireUploadedLocalTrail(store, localId)` instead of `markTrailSynced`, which was deleted from the codebase entirely (zero remaining call sites, verified by a gate test walking every file under `app/lib`).
- `trail_create_screen.dart`'s save routing now distinguishes "never saved anywhere" from "saved locally, and the row is gone because it uploaded" via `resolveLocalSaveModeForRow`, and `LocalUpdateOutcome.missing` joins `alreadySynced` on the network-save branch so a post-upload edit always reports failure rather than a false success toast.
- Phase 36 ends at a codegen fixpoint: `build_runner` ran once, regenerated `trail_sync_provider.g.dart` (the only stale generated file), and a second run confirmed byte-identical output.

## Task Commits

Each task was committed atomically:

1. **Task 1: Retirement -- two pure decisions and the write transaction that acts on them** - `e0532e85` (feat)
2. **Task 2: The drain retires the capture row instead of marking it synced** - `44de045f` (feat)
3. **Task 3: A save whose row was retired can never report success for an edit that landed nowhere** - `37eff077` (feat)
4. **Task 4: The phase ends at a codegen fixpoint** - `4e38ae0b` (chore)

**Plan metadata:** (this commit)

## Files Created/Modified

- `app/lib/util/local_trail_store.dart` - Added `shouldDeleteUploadedRow`, `resolveLocalSaveModeForRow`, and `retireUploadedLocalTrail`; deleted `markTrailSynced` and repaired its dangling dartdoc reference in `writeServerTrailId`'s comment
- `app/lib/provider/trail/trail_sync_provider.dart` - Drain step 4 calls `retireUploadedLocalTrail(store, localId)` instead of `markTrailSynced`
- `app/lib/provider/trail/trail_sync_provider.g.dart` - Regenerated (source hash only; logic unchanged)
- `app/lib/routes/trail_create_screen.dart` - `_onSave` routes via `resolveLocalSaveModeForRow`; the `updateLocal` branch treats `LocalUpdateOutcome.missing` the same as `alreadySynced`
- `app/test/util/local_trail_store_test.dart` - New `shouldDeleteUploadedRow` and `resolveLocalSaveModeForRow` groups with real assertions
- `app/test/util/local_trail_retirement_gate_test.dart` - New source-level gate test pinning the transaction's structural invariants and the drain's ordering/absence facts
- `app/test/routes/trail_create_screen_local_save_gate_test.dart` - Two new gates: routing through `resolveLocalSaveModeForRow`, and the `missing`-outcome network fallback

## Decisions Made

- Followed the plan's product decision exactly: retirement (delete-or-demote) replaces retention as the response to a successful upload. No reconciliation sweep is needed because the orphan class this would reconcile cannot exist anymore.
- `trail_dropdown.dart` was left untouched, matching the plan's explicit scope boundary -- 36-13's dropdown gate tests keep passing against unmodified code.
- Did not reintroduce any of the dropped 36-15 work (`isTerminalFetchFailure`, `isTrailGoneFailure`, `trailFetchRetry`, `_chromeScaffold`) and did not touch `trail_provider.dart` or `trail_detail_screen.dart`, per the plan's explicit instruction.

## Deviations from Plan

None - plan executed exactly as written. Every action, file location, and gate assertion matched the plan's `<action>` blocks verbatim; no Rule 1-4 auto-fixes were needed.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- This was the last plan in Phase 36. `flutter analyze --no-pub` is clean (only pre-existing `info`-level lints) and `flutter test` is fully green with its one pre-existing skip.
- Two accepted risks are flagged for the device re-test rather than fixed in this plan: T-36-14-06 (Meilisearch index-add lag could cause a momentary own-trails-list blink right after a badge clears) and T-36-14-07 (an uploaded-and-never-downloaded trail is intentionally absent from the offline own-trails list once the device goes offline, per the 2026-08-03 product decision).
- A pre-existing, out-of-scope defect remains and is explicitly not fixed here: editing a trail while its create screen is still open across a successful upload fails with `error_saving_trail` because the screen never learns the server id the drain assigned (`_saveViaNetwork` posts an empty id). This plan guarantees the hiker is told, not that the edit lands.
- The device re-test in the plan's `<verification>` (UAT Test 5, plus the two accepted risks) has not been run by this plan and should be run before the phase is considered fully closed out.

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-03*

## Self-Check: PASSED

All 8 created/modified files confirmed present on disk; all 4 task commit hashes (`e0532e85`, `44de045f`, `37eff077`, `4e38ae0b`) confirmed present in git log.
