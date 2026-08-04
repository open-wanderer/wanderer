---
phase: 38-downloaded-trails-as-state-not-objects
plan: 04
subsystem: ui
tags: [flutter, riverpod, trail-panel, library, l10n, objectbox]

# Dependency graph
requires:
  - phase: 38-downloaded-trails-as-state-not-objects
    provides: "38-01 retired forceOffline entirely; 38-02 minted remove_download_confirm_body and edit_needs_connection; 38-03 added applyServerTrailToLibraryRow and the opportunistic library-row refresh"
provides:
  - "Single stored-on-device badge in TrailPanel, gated purely on availableOffline (library membership) — the D-10 grey/green pill conflict is gone"
  - "library_detail_screen.dart forwards availableOffline: true since every row on that screen is a library member by definition"
  - "Library's own destructive action relabelled Remove (circleMinus icon, no red) and confirmed with honest remove_download_confirm_body copy via _confirmRemoveDownload"
  - "Widget test coverage pinning badge-follows-membership-not-cache-provenance (new Case D2)"
affects: [38-05, trail-detail-screen, trail-dropdown]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Awaited showDialog<bool> + if (confirmed != true) return, mirroring settings_offline_regions_screen.dart's _onDeleteRegion, with l10n resolved before the first await and context.mounted checked after"

key-files:
  created: []
  modified:
    - app/lib/components/trail/trail_panel.dart
    - app/lib/routes/library_detail_screen.dart
    - app/lib/routes/library_screen.dart
    - app/test/components/trail/trail_panel_sync_badge_test.dart

key-decisions:
  - "Deleted the grey l18n.offline/Icons.cloud_off pill block entirely rather than re-gating it — D-10 says the badge's axis is library membership, and two badges on one axis with opposite isLocal terms was the defect itself, not something to preserve alongside the fix"
  - "Kept the green pill's existing available_offline vocabulary and styling rather than inventing new copy — it's already the string the dropdown menu uses after D-07"
  - "Library's remove item lost its red styling entirely (icon and text) — removing a download is not a deletion, so it shouldn't carry destructive-red weight"

patterns-established:
  - "Membership-derived UI state must be forwarded via constructor bool (availableOffline), never re-derived by reading objectBoxProvider inside the widget — keeps the widget testable on a host with no libobjectbox.dylib"

requirements-completed: [DL-01, DL-07]

# Metrics
duration: 12min
completed: 2026-08-04
---

# Phase 38 Plan 04: Membership-Derived Offline Badge and Library Remove Vocabulary Summary

**Collapsed trail_panel's two network-flipping "stored offline" pills into one badge gated on library membership, and relabelled the Library's un-download action from Delete to Remove with honest confirm copy**

## Performance

- **Duration:** 12 min
- **Started:** 2026-08-04T16:22:00Z (approx)
- **Completed:** 2026-08-04T16:27:00Z (approx)
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- `trail_panel.dart` now renders exactly one stored-on-device badge, gated on `availableOffline` alone — it no longer appears when a fetch fails or vanishes when one succeeds, and `trail.isLocal` is referenced nowhere in the file (code or comments)
- `library_detail_screen.dart` forwards `availableOffline: true` to `TrailPanel` so the badge doesn't silently disappear from the Library detail sheet
- `library_screen.dart`'s context menu Delete item is now Remove (`circleMinus` icon, no red), confirmed via `_confirmRemoveDownload` — an awaited `showDialog<bool>` using `remove_download_confirm_body`, which makes no "cannot be undone" claim
- New widget test case (Case D2) pins the invariant that a cached-looking trail model with no library membership renders no badge at all, closing the exact gap D-10 describes

## Task Commits

Each task was committed atomically:

1. **Task 1: Collapse the two contradictory pills into one membership-derived badge** - `14ecdc9c` (fix)
2. **Task 2: Give the Library context menu the Remove-download vocabulary** - `84e35276` (fix)
3. **Task 3: Update the panel badge widget test for the single membership-derived badge** - `f218c9a2` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/components/trail/trail_panel.dart` — deleted the grey `l18n.offline` pill block and its `trail.isLocal` gate; surviving green pill now gated on `if (availableOffline)`; reworded the WR-11 doc comment to avoid the literal `isLocal` substring the plan's own grep gate checks for
- `app/lib/routes/library_detail_screen.dart` — forwards `availableOffline: true` to `TrailPanel` with a one-line comment explaining why it's a tautology on this screen
- `app/lib/routes/library_screen.dart` — `_confirmDelete` replaced with `_confirmRemoveDownload`; menu item relabelled to `l18n.remove` with `FontAwesomeIcons.circleMinus`; dialog uses `remove_download_confirm_body`, red-styled confirm action, `l18n.cancel`
- `app/test/components/trail/trail_panel_sync_badge_test.dart` — Case D updated to assert `Available offline` renders and `Offline` does not; new Case D2 asserts no badge renders for a non-member cached-looking fixture; file-header comment updated to note the grey pill no longer exists

## Decisions Made
- Deleted the grey pill entirely rather than merely re-gating it, per D-10's framing that two badges on one axis was itself the defect
- Kept the green pill's existing `available_offline` vocabulary/styling — no new copy needed since D-07 already aligned the dropdown menu on this string
- Library's remove item dropped red styling on both icon and text, matching D-04's "removing a download is not a deletion" framing
- Comments referencing `trail.isLocal`/`isLocal` were reworded (not just the code) since the plan's acceptance criteria grep for the literal string across the whole file, not just executable code

## Deviations from Plan

None — plan executed exactly as written. Two minor in-flight corrections (rewording doc comments to avoid the literal `isLocal`/`delete_trail_confirm` substrings caught by the plan's own negative-grep acceptance criteria) were applied during Task 1 and Task 2 before committing, not after — no separate deviation commits were needed.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The badge and Library menu vocabulary fixes from this plan are self-contained; no blockers for 38-05 (the detail-screen dropdown menu's Update/Remove download items), which reuses the same `circleMinus` icon convention established here
- `flutter analyze --no-pub lib test` reports zero errors/warnings (36 pre-existing info-level lines only)
- `flutter test` is fully green: 961/961 tests pass (960 prior + 1 new Case D2)

---
*Phase: 38-downloaded-trails-as-state-not-objects*
*Completed: 2026-08-04*

## Self-Check: PASSED

All 4 created/modified files confirmed present on disk; all 3 task commit hashes (`14ecdc9c`, `84e35276`, `f218c9a2`) confirmed present in git log.
