---
phase: quick-261005-pwz
plan: 01
subsystem: app (Flutter) - local recording drafts, sync status UI
tags: [flutter, riverpod, objectbox, l10n, sync-state, drafts]
requires:
  - quick 261005-grz (local recording drafts, round 1)
provides:
  - syncStateColors helper (single source of sync-state colors)
  - resolveSyncDisplayState (shared chip/section resolver)
  - SyncStatusSection on the trail detail screen
  - promoteDraftToPending / isOwnDraft / isOwnDraftRow store functions
  - TrailSync.uploadDraft / TrailSync.discardDraft
affects:
  - app/lib/components/trail/trail_panel.dart (title chip replaced by section)
tech-stack:
  added: []
  patterns:
    - owner-scoped raw-row predicate (isOwnDraft) gating both a write and a delete
    - source-level gate tests for Store-touching code (no ObjectBox in flutter test)
key-files:
  created:
    - app/lib/theme/sync_state_colors.dart
    - app/lib/components/trail/sync_status_section.dart
    - app/test/models/trail_sync_state_test.dart
    - app/test/theme/sync_state_colors_test.dart
    - app/test/components/trail/sync_status_section_test.dart
  modified:
    - app/lib/models/trail_sync_state.dart
    - app/lib/theme/colors.dart
    - app/lib/components/trail/sync_status_chip.dart
    - app/lib/components/trail/trail_panel.dart
    - app/lib/store/local_trail_store.dart
    - app/lib/provider/trail/trail_sync_provider.dart
    - app/lib/i18n/app_en.arb
    - app/lib/i18n/app_de.arb
    - app/lib/i18n/app_localizations*.dart
    - app/lib/i18n/untranslated_messages.json
    - app/test/components/trail/sync_status_chip_test.dart
    - app/test/components/trail/trail_panel_sync_badge_test.dart
    - app/test/store/local_trail_store_test.dart
    - app/test/store/local_trail_draft_gate_test.dart
    - DESIGN.md (on disk only, see Deviations)
decisions:
  - "Upload now calls TrailSync.retry(localId), not drainIfOnline (orchestrator decision), so a tap also uploads a pending row that is still inside its retry backoff window"
  - "Draft promotion has exactly two paths (edit-form save and promoteDraftToPending), pinned by a lib-wide gate test"
  - "Discard is draft-only in the UI and re-checked at execution time via isOwnDraftRow before deleteUnsynced"
metrics:
  tasks: 3
  commits: 4
  completed: 2026-10-05
---

# Phase quick-261005-pwz Plan 01: Sync status section on the trail detail screen Summary

The trail detail screen now shows a flat, colored sync status section below the stat chips (replacing the title chip), sharing one state resolver and one color helper with the list chip, and a local draft can be uploaded as is or discarded after a confirmation, both through owner-scoped store guards.

## What was built

- **Task 1** (341f40830): `resolveSyncDisplayState` (pure, in `trail_sync_state.dart`) and `syncStateColors(TrailSyncState, ThemeData)` (new `sync_state_colors.dart`). Draft keeps its exact amber values (folded in from `AppColors`, which no longer has them). Failed now has a red container (opaque light red with dark red label in light mode; 0.18-alpha red with light red label in dark mode). Pending and uploading are neutral. `SyncStatusChip` takes every color from the helper and has no `Colors.red`. Contrast of draft and failed labels is at least 4.5 in both themes (unit tested).
- **Task 2** (17b8abd65): `isOwnDraft` (pure), `promoteDraftToPending` (owner-scoped write in a transaction, resets attempts/backoff), `isOwnDraftRow` (owner-scoped raw-row read, never `toModel`). `TrailSync.uploadDraft` (fresh account, promote, invalidate detail/library/profile providers, `drainIfOnline`) and `TrailSync.discardDraft` (fresh account, `isOwnDraftRow` re-check, then `deleteUnsynced`, true only on `deleted`). `_drainOne`'s catch block now also invalidates `localTrailProvider(localId)` so a mounted detail screen re-reads a failed/backed-off row.
- **Task 3** (25ee4dce4 strings, c29f4114d section): eight new keys in en and de only, gen-l10n output committed straight after generating. `SyncStatusSection` with per-state copy and actions: draft (Upload + Discard with confirmation, pops on success, error toast on refusal), pending online (Upload now via `retry`), pending offline (explanatory text, no button), uploading (spinner, no buttons), failed (Retry). `TrailPanel` renders it after the stat-chip Wrap, gated on `isUnsyncedState`; cards and list items keep the chip.

## Orchestrator decision noted

"Upload now" calls `TrailSync.retry(localId)` rather than `drainIfOnline`, so an explicit tap also clears any backoff window on a pending row and uploads it immediately. The widget test asserts `retry` is called and no bare drain happens.

## Verification

Baseline (before any edit) and final, both from `app/`:

- `flutter analyze`: 13 issues before, 13 after (all pre-existing infos in entities, providers, `local_photo_store.dart`, `util/local/id.dart` and vendored tiptap). No new issues.
- `flutter test`: baseline 1 failure, final 1 failure, the same one: `test/routes/settings_screen_test.dart` "settings screen lists all 7 rows including Offline Maps/Regions" (pre-existing, unrelated). Passing count rose from about 1220 to about 1292 with the new tests.
- `flutter gen-l10n` produces no further diff after the strings commit.
- `grep SyncStatusChip(` in lib: only `trail_card.dart` and `trail_list_item.dart` (plus the chip's own file). `SyncStatusSection(` only in `trail_panel.dart`. No `draftContainer`/`onDraftContainer` anywhere.
- No `flutter build`, `adb` or device command was run. Nothing was pushed. `pubspec.lock` unchanged.

The on-device human check from the plan (record a draft, Upload, Discard, airplane mode, dark mode) is left to the user.

## Deviations from Plan

### Auto-fixed Issues

None for code behaviour.

### Environment deviations

**1. [Rule 3 - Blocking] DESIGN.md is gitignored, so its one-line exception was not committed**
- **Found during:** Task 1 commit (`git add DESIGN.md` refused: path is ignored by .gitignore, `git ls-files DESIGN.md` is empty).
- **Fix:** The new "Sync-State Exception" paragraph (SYNCUI-06) is written to `/Users/christianbeutel/Documents/svelte/wanderer/DESIGN.md` directly after The White Accent Rule, but it is not in any commit. I did not `git add -f`. If DESIGN.md should be tracked, that is a separate decision for the user.
- **Files modified:** DESIGN.md (on disk only)

**2. Formatting of pre-unformatted files**
- `test/store/local_trail_store_test.dart`, `test/components/trail/trail_panel_sync_badge_test.dart` and `lib/components/trail/trail_panel.dart` were not `dart format`-clean at HEAD, so per the plan's rule I did not run the formatter on them and hand-formatted my additions instead (the remaining format diffs in those files are all pre-existing lines).

## Known Stubs

None.

## Threat Flags

None. All mitigations from the plan's threat register are implemented: T-pwz-01 (owner-scoped promotion with `isOwnDraft` re-check in the write transaction), T-pwz-02 (draft-only button, confirmation dialog, `isOwnDraftRow` re-read before `deleteUnsynced`), T-pwz-04 (lib-wide gate: `syncStateAfterLocalEdit(` appears exactly 3 times, only in the store; `promoteDraftToPending(` called once, in `uploadDraft`).

## Commits

| Commit | Message |
| ------ | ------- |
| 341f40830 | feat(app): share sync-state colors between status surfaces and give failed a red container |
| 17b8abd65 | feat(app): add owner-scoped upload and discard actions for recording drafts |
| 25ee4dce4 | feat(app): add strings for the trail detail sync status section |
| c29f4114d | feat(app): show a sync status section with draft actions on the trail detail screen |

## Self-Check: PASSED

- Created files exist: sync_state_colors.dart, sync_status_section.dart, trail_sync_state_test.dart, sync_state_colors_test.dart, sync_status_section_test.dart.
- Commits 341f40830, 17b8abd65, 25ee4dce4, c29f4114d are in `git log`, each headline-only with no trailer.
