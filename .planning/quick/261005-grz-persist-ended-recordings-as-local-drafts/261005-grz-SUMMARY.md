---
phase: quick-261005-grz
plan: 01
subsystem: app/recording, app/local-trail-store
tags: [flutter, objectbox, recording, data-loss, sync-state]
requirements: [DRAFT-01, DRAFT-02, DRAFT-03, DRAFT-04, DRAFT-05, DRAFT-06]
key-files:
  modified:
    - app/lib/models/trail_sync_state.dart
    - app/lib/store/local_trail_store.dart
    - app/lib/routes/navigation_screen.dart
    - app/lib/components/trail/sync_status_chip.dart
    - app/lib/i18n/app_en.arb
    - app/lib/i18n/app_de.arb
  created:
    - app/test/store/local_trail_draft_gate_test.dart
    - app/test/routes/navigation_screen_draft_gate_test.dart
metrics:
  tasks: 3
  completed: 2026-10-05
---

# Quick 261005-grz: Persist ended recordings as local drafts

An ended recording is now written as a durable `TrailEntity` draft row (`TrailSyncState.draft`) before the active-navigation row is cleared, so an OS kill between "Save track" and the form's Save no longer loses it.

## What changed

- **Enum (DRAFT-01):** `TrailSyncState.draft` appended last (index 4); `synced` stays 0. No codegen needed (`dbSyncState` shadow int).
- **Store (DRAFT-02/04):**
  - `syncStateAfterLocalEdit` promotes draft to pending. `updateLocalTrail` uses it, so the edit form's save updates the same row and the existing drain uploads it.
  - `selectDrainCandidates` excludes drafts in the query. `isDrainDue` was already false for drafts (pinned by a test). `resetDrainBackoff` is a no-op for drafts.
  - `saveNewLocalTrail(asDraft:)` added; the form's createLocal path still creates pending rows.
  - `persistDraftThenClearSession` is a Store-free write-then-clear primitive, so the ordering is unit-testable.
  - Drafts still count as unsynced for the sign-out warning, delete gating, live-capture checks and the orphan-photo sweep (`isUnsyncedState` unchanged).
- **Recording flow (DRAFT-03/05):** `_saveRecordedTrack` writes the draft (owner read fresh via `currentAccountId`, author actor linked) via `persistDraftThenClearSession`. Only the `clearSession` closure cancels the persist timer and calls `active_nav.clear`. It then invalidates `trailLibraryProvider` and `profileTrailsProvider`, and opens `/trail/create/edit` with `draftTrail` (carries `localId` and draft state) in both `extra` and `pendingImportedTrail`. A failed draft write falls into the existing catch (error toast), leaving the session and timer intact.
- **Chip (DRAFT-06):** non-tappable pen-icon "Draft" / "Entwurf" chip (`sync_draft`), placed before the in-flight check.

## Commits

- c2ca3503d fix(app): add a draft sync state the upload drain never picks up
- a8a8ccde6 fix(app): persist an ended recording as a local draft before clearing its session
- 7f435bce4 feat(app): show a Draft chip on recordings saved as local drafts

## Deviations from Plan

**1. [Rule 3 - Blocking/coordination] l10n changes landed in another session's commit.**
Another session was working in the same working tree (settings About screen). It ran `git add` on `app/lib/i18n/*` and committed 741fb36b2 "feat(app): add an about screen to settings" while my uncommitted `sync_draft` additions (en "Draft", de "Entwurf") and the regenerated `app_localizations*.dart` / `untranslated_messages.json` were in the working tree. So `sync_draft` is in HEAD via that commit, not via my Task 3 commit. Task 3's commit (7f435bce4) contains only the chip and its tests. Content is correct (verified `sync_draft` in HEAD's arb and generated files; `flutter gen-l10n` produces no further diff). History cannot be tidied without rewriting a commit that is not mine.

**2. Formatting side effects reverted.** `dart format test/store` initially reformatted unrelated test files (and whole-file reformatting of two touched test files). Reverted those and re-applied only my edits, so the commits carry no unrelated format churn. One existing test call in `sync_status_chip_test.dart` was reflowed by `dart format`.

Otherwise the plan was executed as written.

## Verification

- Baseline (before changes) `flutter test`: 1185 passed, 1 skipped, 2 failed:
  - `test/provider/account_scope_invalidation_test.dart` (load error)
  - `test/routes/settings_screen_test.dart` ("lists all 7 rows including Offline Maps/Regions")
- Final `flutter test`: 1216 passed, 1 skipped, 1 failed: only `settings_screen_test.dart` (pre-existing from the baseline; not touched by this plan). `account_scope_invalidation_test.dart` loaded fine on the final run; it was not touched by this plan.
- Baseline `flutter analyze`: 5 infos. Final: 5 infos, the same ones (all pre-existing, none in touched code). No new issues.
- No non-exhaustive `switch` over `TrailSyncState` in `app/lib`.
- No `flutter build`, adb or device command was run. Nothing was pushed. All three commits are headline-only.

## Pending on-device check (user)

Record at least 2 points, exit and choose Save track, confirm the form opens. Kill the app from recents, reopen: no resume prompt, the profile own-trails list shows the recording with a "Draft" chip. Edit and Save: chip becomes "Waiting to upload" and it uploads once online as a single trail.

## Known Stubs

None.

## Self-Check: PASSED

- Commits c2ca3503d, a8a8ccde6, 7f435bce4 exist on feature/app.
- Created files present: `app/test/store/local_trail_draft_gate_test.dart`, `app/test/routes/navigation_screen_draft_gate_test.dart`.
