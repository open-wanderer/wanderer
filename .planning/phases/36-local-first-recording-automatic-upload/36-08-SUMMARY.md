---
phase: 36-local-first-recording-automatic-upload
plan: 08
subsystem: mobile-trail-ui
tags: [flutter, riverpod, sync-state, trail-card, trail-dropdown]

# Dependency graph
requires:
  - phase: 36-local-first-recording-automatic-upload
    plan: 01
    provides: "TrailSyncState enum, isUnsyncedState(), TrailSummary.syncState/localId getters"
  - phase: 36-local-first-recording-automatic-upload
    plan: 02
    provides: "sync_pending/sync_uploading/sync_failed/delete_unsynced_trail_confirm l10n keys"
  - phase: 36-local-first-recording-automatic-upload
    plan: 04
    provides: "trail_sync_provider.dart: TrailSync in-flight Set<String>, retry(), deleteUnsynced()"
provides:
  - "SyncStatusChip: four-state (absent/Pending/Uploading/Failed) sync indicator shared by trail_card.dart and trail_list_item.dart"
  - "trail_dropdown.dart's isUnsynced/isDraining split -- download hidden for an unsynced trail, delete disabled mid-drain, three-way _deleteTrail branch"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "SyncStatusChip mirrors trail_card.dart's private _Chip shape exactly (padding/radius/background) rather than introducing a new chip visual language"
    - "isUnsyncedState(trail.syncState), not trail.isLocal, is the correct predicate for 'never reached the server' -- isLocal is also true for a downloaded trail post-36-01"

key-files:
  created:
    - app/lib/components/trail/sync_status_chip.dart
    - app/test/components/trail/sync_status_chip_test.dart
  modified:
    - app/lib/components/trail/trail_card.dart
    - app/lib/components/trail/trail_list_item.dart
    - app/lib/components/trail/trail_dropdown.dart
    - app/test/components/trail/trail_dropdown_delete_gate_test.dart

key-decisions:
  - "The download PopupMenuDivider is wrapped inside the same if (!isUnsynced) collection-if as the download item (not left as an always-shown divider above it), avoiding a double-divider artifact when download is hidden -- matches the existing _canEditTrail/_allowDelete divider+item grouping convention in the same file"
  - "Widget tests for the Uploading state use tester.pump() instead of pumpAndSettle() -- CircularProgressIndicator's indeterminate animation never settles, so pumpAndSettle times out"

requirements-completed: [REC-03, SYNC-02, SYNC-03]

# Metrics
duration: ~20min
completed: 2026-08-02
---

# Phase 36 Plan 08: Sync-status chip and the trail_dropdown isLocal split Summary

**A four-state `SyncStatusChip` (absent when synced, otherwise Pending/Uploading/Failed-with-retry) now renders below the title on both `trail_card.dart` and `trail_list_item.dart`, and `trail_dropdown.dart`'s two `isLocal`-branching menu actions are split onto the correct `isUnsynced` predicate -- download hidden entirely for an unsynced trail, delete disabled (not hidden) while it drains, and `_deleteTrail` gains a three-way unsynced-first branch so an unsynced trail's delete can never silently no-op on an empty server id.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-08-02
- **Completed:** 2026-08-02
- **Tasks:** 3
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments
- `SyncStatusChip` (`ConsumerWidget`, takes a `TrailSummary`): returns `SizedBox.shrink()` for a synced trail (D-08); otherwise watches `trailSyncProvider`'s in-flight set (authoritative in-the-moment signal) with a fallback to the persisted `TrailSyncState.uploading` for a restart mid-drain; renders Pending (`cloudArrowUp`, muted `onSurface` 60%), Uploading (12x12 `CircularProgressIndicator`, `strokeWidth: 2`), or Failed (`triangleExclamation`, `Colors.red`, tappable `InkWell` calling `trailSyncProvider.notifier.retry(localId)`) -- visual shell matches `trail_card.dart`'s existing `_Chip` exactly (`horizontal: 8, vertical: 4` padding, `BorderRadius.circular(8)`, `surfaceContainerHighest` background)
- Mounted unconditionally below the title on both `trail_card.dart` and `trail_list_item.dart`, wrapped in `Align(alignment: Alignment.centerLeft)`, kept off the existing top-right public/shared badge axis (D-09) -- the widget itself collapses to nothing for a synced trail, so neither call site needs its own condition
- `trail_dropdown.dart`: computed `isUnsynced = isUnsyncedState(trail.syncState)` and `isDraining = trail.localId != null && trailSyncProvider.contains(trail.localId)` once in `build()`; download `PopupMenuItem` (plus its divider) now wrapped in `if (!isUnsynced) ...[...]` (D-17, hide-not-disable, no tooltip-on-disabled convention in this file); delete stays visible but `enabled: !isDraining` with greyed icon/label and a null `onTap` while draining (D-14, disable-not-hide)
- `_deleteTrail` gained a new first branch: `if (isUnsyncedState(trail.syncState) && trail.localId != null)` pops the menu, awaits `trailSyncProvider.notifier.deleteUnsynced(trail.localId!)`, and returns -- ordered strictly before the existing `if (trail.isLocal)` un-download branch, since an unsynced trail also reports `isLocal == true` and the wrong order would route it into `trailLibraryProvider.deleteTrail('')`, a silent no-op that destroys the hiker's only copy with no feedback
- `_confirmDelete` shows `l10n.delete_unsynced_trail_confirm` ("...this can't be undone") for an unsynced trail and keeps `l10n.delete_trail_confirm` otherwise, since the latter also doubles as the genuinely-reversible un-download confirm in `library_screen.dart`
- Extended (not replaced) `trail_dropdown_delete_gate_test.dart` with four new source-level assertions: unsynced branch precedes and returns before the `isLocal` branch, download is gated behind `if (!isUnsynced)`, and `_confirmDelete` references the new l10n key

## Task Commits

Each task was committed atomically:

1. **Task 1: The four-state sync chip** - `15853a88` (feat)
2. **Task 2: Mount the chip on the trail card and the trail list item** - `b3053e40` (feat)
3. **Task 3: Split the trail_dropdown isLocal branch** - `a8d0fc8d` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/components/trail/sync_status_chip.dart` - `SyncStatusChip` widget, four states
- `app/test/components/trail/sync_status_chip_test.dart` - 5 widget tests (synced/pending/uploading/failed/in-flight-override)
- `app/lib/components/trail/trail_card.dart` - chip mounted below title, above date
- `app/lib/components/trail/trail_list_item.dart` - chip mounted below title row, above the date/author row
- `app/lib/components/trail/trail_dropdown.dart` - `isUnsynced`/`isDraining` computed once; download hidden, delete disabled-while-draining, three-way `_deleteTrail`, per-state confirm copy
- `app/test/components/trail/trail_dropdown_delete_gate_test.dart` - extended with 4 new source-level assertions (not deleted)

## Decisions Made
- The download item's `PopupMenuDivider` is grouped inside the same `if (!isUnsynced) ...[...]` block as the item itself, mirroring how `_canEditTrail`/`_allowDelete` already group their own dividers -- prevents an orphaned divider from rendering directly above the (now-hidden) download item
- Uploading-state widget tests use `tester.pump()` rather than `tester.pumpAndSettle()`, since `CircularProgressIndicator`'s indeterminate animation runs forever and `pumpAndSettle` would time out waiting for it to stop

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. `flutter analyze --no-pub` (project-wide) reports zero errors -- only pre-existing `info`-level lints unrelated to this plan (deprecated Font Awesome constants in `icon_util.dart`, two dangling-library-doc-comment infos). The full `flutter test` suite (771 tests, 1 pre-existing skip) passed with no regressions.

## Device-Verified-Only Behaviours

Per the plan's own scope, the device-check portion of Task 3's verification (menu behavior for an unsynced trail: download absent, delete confirm wording, delete greying out mid-upload; unchanged behavior on a downloaded trail) was not run on a physical device in this session -- it is covered here only by `flutter analyze`, the extended source-level gate test, and the widget-level `SyncStatusChip` tests. All four are source/widget-testable and green; the live end-to-end interaction was not separately re-verified on hardware.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- This is the last plan of Phase 36. All REC-*/SYNC-* requirements assigned to this phase now have a visible, tappable surface: `SyncStatusChip` covers REC-03/SYNC-02/SYNC-03, and `trail_dropdown.dart`'s split closes the `isLocal`-conflation landmine flagged by both CONTEXT.md and RESEARCH.md for D-17/D-14.
- No blockers.

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-02*

## Self-Check: PASSED

All 6 files created/modified in this plan verified present on disk; all 3 task commits (`15853a88`, `b3053e40`, `a8d0fc8d`) verified present in git log.
