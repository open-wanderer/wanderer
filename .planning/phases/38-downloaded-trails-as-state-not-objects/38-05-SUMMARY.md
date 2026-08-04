---
phase: 38-downloaded-trails-as-state-not-objects
plan: 05
subsystem: ui
tags: [flutter, riverpod, trail-detail, popup-menu, l10n]

# Dependency graph
requires:
  - phase: 38-downloaded-trails-as-state-not-objects
    provides: "38-01 retired forceOffline; 38-02 minted remove_download_confirm_body and edit_needs_connection; 38-03 applyServerTrailToLibraryRow"
  - phase: 36-local-first-recording-automatic-upload
    provides: "D-10 -- unsynced and downloaded are mutually exclusive by construction"
provides:
  - "TrailDropdown menu with Update / Remove download (when downloaded) or Download (when not) instead of the old inert 'Available offline' item"
  - "_confirmRemoveDownload, a local-only un-download confirm that never pops the route and never issues a server delete"
  - "_allowDelete derived from isUnsyncedState + authorship, with zero Trail.isLocal references left in trail_dropdown.dart"
affects: [38-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Availability-by-membership: destructive/maintenance menu actions gate on widget.availableOffline (library membership) and authorship, never on a cached-provenance model flag"
    - "Awaited showDialog<bool> + if (confirmed != true) return; confirm-dialog shape (mirrored from settings_offline_regions_screen.dart's _onDeleteRegion) used for a second async confirm in the same file"

key-files:
  created: []
  modified:
    - app/lib/components/trail/trail_dropdown.dart
    - app/test/components/trail/trail_dropdown_delete_gate_test.dart

key-decisions:
  - "Update and Remove download reuse regions_update_action and remove (already translated in all 14 locales) rather than minting trail-scoped duplicates, per D-06"
  - "Update's onTap is byte-identical to Download's onTap (ref.read(downloadingTrailIdsProvider.notifier).download(trail)) -- it is a re-download of the same trail, not a new code path (D-12)"
  - "_confirmRemoveDownload calls only trailLibraryProvider.deleteTrail and never pops the route -- the trail still exists under the single-object model, so the screen stays and the menu flips back to Download"
  - "_allowDelete's isUnsyncedState(...) escape hatch is checked before authorship, preserving a hiker's ability to delete their own unsynced recording whose author can be the Trail.author placeholder"

requirements-completed: [DL-01, DL-06, DL-07]

duration: ~40min
completed: 2026-08-04
---

# Phase 38 Plan 05: Menu availability derived from membership and authorship, not isLocal Summary

**Split the inert "Available offline" menu item into Update and Remove download, and rewrote `_allowDelete`/`_deleteTrail` so destructive-action availability derives from library membership and authorship instead of `Trail.isLocal` -- closing the timeout-arms-the-wrong-branch defect this phase exists to fix.**

## Performance

- **Duration:** ~40 min
- **Completed:** 2026-08-04T14:44:18Z
- **Tasks:** 2 completed
- **Files modified:** 2 (`trail_dropdown.dart`, plus one stale test file fixed as a direct consequence)

## Accomplishments
- A downloaded trail's menu now offers two flat items, *Update* and *Remove download*, instead of one inert checkmark item; an undownloaded trail still offers a single *Download* item
- *Update* re-downloads in place via the existing `downloadingTrailIdsProvider.download(trail)` entry point -- no new download path
- *Remove download* confirms via a new `_confirmRemoveDownload`, modelled on the offline-regions dialog, that touches only `trailLibraryProvider` and never pops the route or reaches the server
- `_allowDelete` now derives from `isUnsyncedState(...)` (escape hatch for the hiker's own unsynced recording) and authorship (`trail.author == user.actorId`) -- the unconditional `if (widget.trail.isLocal) return true;` is gone
- `_deleteTrail`'s un-download fall-through (`if (trail.isLocal) {...}`) is gone; after the unsynced branch, control falls straight through to `_deleteOnServer`
- Zero `isLocal` references remain anywhere in `trail_dropdown.dart` (grep-verified)
- A trail the hiker authored and downloaded now shows both *Remove download* and *Delete trail* (D-02); someone else's downloaded trail shows *Remove download* only

## Task Commits

Each task was committed atomically:

1. **Task 1: Replace the inert offline item with Update and Remove download** - `66bbab3e` (feat)
2. **Task 2: Derive Delete trail from authorship alone and remove the un-download fall-through** - `59bc39f5` (fix)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/components/trail/trail_dropdown.dart` - TrailAction gains `update`/`removeDownload`; the downloaded-branch menu items; `_confirmRemoveDownload`; rewritten `_allowDelete`; simplified `_deleteTrail`
- `app/test/components/trail/trail_dropdown_delete_gate_test.dart` - source-text gate rewritten to assert the new two-branch shape instead of the removed `isLocal` branch (see Deviations)

## Decisions Made
- Kept the `downloadEnabled` variable name (now `!isDownloading`, shared by all three download-family items: Download / Update / Remove download) rather than introducing a second identically-valued variable, since the plan's acceptance criteria describe the gate in those exact terms.
- Placed `_confirmRemoveDownload` immediately after `build()` (before `_canEditTrail`), grouping it with the menu-building code it serves rather than next to `_confirmDelete`, which belongs to an unrelated authorship-gated flow.
- Rewrote three explanatory comments (top-of-`build()`, `_allowDelete`'s doc comment, `_deleteTrail`'s doc comment, and the WR-08 null-`localId` comment) to describe the new invariants without literally naming `Trail.isLocal`, since the acceptance criteria's `grep -c "isLocal"` check does not distinguish code from prose.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated a stale source-text gate test that asserted the removed `isLocal` branch**
- **Found during:** Task 2 verification (`flutter test` full-suite run)
- **Issue:** `test/components/trail/trail_dropdown_delete_gate_test.dart` (not in the plan's `files_modified`) contains two source-text assertions that specifically grep for `if (trail.isLocal) {` inside `_deleteTrail` and assert it exists with a `return;`. Task 2's own acceptance criteria require that exact branch to be deleted, so these two tests failed as a direct, expected consequence of the required change -- not a pre-existing failure.
- **Fix:** Rewrote both tests (and the file's header doc comment) to assert the new invariant instead: `_deleteTrail`'s body contains neither `isLocal` nor `trailLibraryProvider`, falls straight through to `await _deleteOnServer(context, trail);` after the unsynced branch, and the unsynced branch is still checked first and still returns. The three tests in the same file that were unaffected (download-item visibility gate, `delete_unsynced_trail_confirm` key) were left untouched.
- **Files modified:** `app/test/components/trail/trail_dropdown_delete_gate_test.dart`
- **Verification:** `flutter test test/components/trail/trail_dropdown_delete_gate_test.dart` -- all 5 tests pass; full `flutter test` -- 961 passed, 1 pre-existing skip, 0 failures
- **Committed in:** `59bc39f5` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - bug/stale-test fix directly required by the plan's own design change)
**Impact on plan:** Necessary to keep the suite green after the intentional removal the plan mandates. No scope creep -- the fix only updates assertions about code this same plan deletes.

## Issues Encountered
- Two explanatory comments I initially wrote (the D-07 status-line comment and the `_confirmRemoveDownload` "must never..." comment) contained the literal strings `'Available offline'` and `trailSaveProvider`/`_deleteOnServer`, which would have false-positived the acceptance criteria's literal-string greps despite being prose, not code. Rephrased both to describe intent without repeating the forbidden literals verbatim.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Plan 38-06 (device verification / new widget tests for this menu shape) can proceed: `TrailAction.update` and `TrailAction.removeDownload` exist, `_confirmRemoveDownload` is reachable, and `widget.availableOffline` remains a plain constructor bool testable through the existing harness with zero ObjectBox `Store` involvement.
- No blockers. `trail_panel.dart`'s "Offline" pill re-gate (D-10) is out of this plan's `files_modified` scope and remains for whichever plan owns it.

---
*Phase: 38-downloaded-trails-as-state-not-objects*
*Completed: 2026-08-04*

## Self-Check: PASSED

All created/modified files and all task/summary commit hashes verified present on disk and in git history.
