---
phase: 03-stats-sheet
plan: 02
subsystem: ui
tags: [flutter, riverpod, draggable-sheet, pageview, navigation, stats, i18n]

# Dependency graph
requires:
  - phase: 03-stats-sheet
    provides: "navigationStatsProvider (onPosition/togglePause), NavigationStats state, formatSpeed/formatElapsed (Wave 1, 03-01)"
  - phase: 02-navigation
    provides: "NavigationScreen single broadcast GPS stream, navigationProvider, NavigateResponse"
provides:
  - "Stats sheet UI on NavigationScreen — DraggableScrollableSheet + button-driven PageView (live stats + reused ElevationProfile) + [Elevation profile | Pause/Resume | Exit] button row"
  - "Stats fed from the single existing GPS listener via navigationStatsProvider.onPosition(Position) (D-13)"
  - "i18n labels time/pause/resume/exit_navigation (en + de)"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "DraggableScrollableSheet covering only the bottom band (Pitfall 4 — map + bottom-right controls stay reachable)"
    - "Button-driven PageView (NeverScrollableScrollPhysics) switched via PageController.animateToPage — no horizontal swipe (CONTEXT)"
    - "ListView(controller: scrollController, ClampingScrollPhysics) forwards drag-to-expand (Pitfall 1)"
    - "Stats fed from the existing single GPS listener — no second stream (D-13)"

key-files:
  created: []
  modified:
    - app/lib/routes/navigation_screen.dart
    - app/lib/i18n/app_en.arb
    - app/lib/i18n/app_de.arb
    - app/lib/i18n/app_localizations.dart
    - app/lib/i18n/app_localizations_en.dart
    - app/lib/i18n/app_localizations_de.dart

key-decisions:
  - "Reused the existing 'speed' i18n key for current speed (skipped adding 'current_speed') — single source per the plan's pick-one option"
  - "Sheet snaps: initialChildSize/minChildSize 0.18, maxChildSize 0.45, snapSizes [0.18, 0.45] (A2, confirmed in human-verify — no tuning needed)"
  - "PageView fixed height 220; stats page expanded row is always present below a Divider, revealed naturally as the sheet grows (no snap-extent conditional)"
  - "ElevationProfile page typed against AsyncValue<dynamic> to avoid importing the Trail type into the screen signature; enableLineTouch:false"

requirements-completed: [STATS-01, STATS-02, STATS-03, STATS-04, STATS-05]

# Metrics
duration: complete
completed: 2026-06-13
---

# Phase 3 Plan 02: Stats Sheet UI Summary

**A DraggableScrollableSheet wired into NavigationScreen with a button-driven two-page PageView (live stats + the reused ElevationProfile chart) and a [Elevation profile | Pause/Resume | Exit] button row, with stats fed from the existing single GPS stream and the old top-left X overlay removed.**

> STATUS: COMPLETE. All 3 tasks done. Tasks 1 and 2 committed (`1c8920bd`, `e45e6bf3`); Task 3 (blocking human-verify) approved by the user on-device.

## Accomplishments

### Task 1 — i18n labels (committed `1c8920bd`)
- Added `time`, `pause`, `resume`, `exit_navigation` to both `app_en.arb` and `app_de.arb` (German: Zeit / Pause / Fortsetzen / Beenden)
- Reused the existing `speed` key for current speed (skipped `current_speed`)
- Regenerated `app_localizations*.dart`; `flutter gen-l10n` exits 0; getters available (snake_case in this project's gen config: `localizations.time`, `localizations.pause`, etc.)

### Task 2 — stats sheet UI (committed `e45e6bf3`)
- `DraggableScrollableSheet` added as a direct child of the outer Stack (bottom band only — Pitfall 4), snaps 0.18 ↔ 0.45
- `ListView(scrollController, ClampingScrollPhysics)` forwards drag-to-expand (Pitfall 1); drag handle at top
- `PageView(NeverScrollableScrollPhysics)` height 220 — page 0 live stats, page 1 reused `ElevationProfile(enableLineTouch:false)` with a back arrow returning to page 0
- Stats page: compact row Time / Distance / Elevation gain; expanded row Elevation loss / Current speed / Average speed; values `fontSize 24, FontWeight.bold` with subdued labels (CONTEXT specifics)
- Button row: left elevation-profile `IconButton` → `animateToPage(1)`; center dominant `FilledButton.icon` Pause/Resume → `navigationStatsProvider.togglePause()`; right Exit `IconButton` → `context.pop()`
- Stats fed from the SINGLE existing `_positionStream.listen` callback via `navigationStatsProvider(widget.response).notifier.onPosition(pos)` (D-13 — exactly one `getPositionStream`)
- Removed the old top-left `SafeArea > Align > _buildExitButton` overlay and the `_buildExitButton` helper (NAV-07 consolidated)
- Added `_pageController` and `_sheetController` fields, both disposed in `dispose()`

## Verification

- `flutter gen-l10n` — exits 0; new getters present
- `flutter analyze lib/routes/navigation_screen.dart` — No issues found
- Acceptance greps all pass: DraggableScrollableSheet(3), NeverScrollableScrollPhysics(2), navigationStatsProvider(3), ElevationProfile((1), togglePause(1), `_buildExitButton` non-comment(0), getPositionStream(1, D-13), controller disposes(2)
- `flutter test` — Phase 03 source/test files and navigate_response tests pass; 2 pre-existing out-of-scope failures remain (see Deviations)

## Decisions Made
- Reused `speed` key for current speed (single i18n source).
- Sheet snaps 0.18 / 0.45 (A2) — confirmed during human-verify that the bottom-right control column stays reachable at both snaps; no tuning applied.
- PageView fixed height 220; expanded stats row always present below a Divider, revealed by sheet growth (no snap-extent conditional).
- `ElevationProfile` page typed against `AsyncValue<dynamic>` to keep the Trail type out of the screen method signature.

## Deviations from Plan

### Inherited working-tree change (not authored by this plan)
- The starting working tree already contained an uncommitted cleanup of `navigation_screen.dart` removing an unused recenter-button `AnimationController`/scale animation (from a prior session). It was committed alongside the Task 2 stats-sheet edits since both touch the same file and the cleanup is consistent with the screen's current behavior. No functional regression.

### Out-of-scope pre-existing test failures (NOT fixed — scope boundary)
- 2 tests fail in `test/models/feed_item_test.dart` originating in generated `trail.g.dart` / `list.g.dart` (`int` vs `String`/`List` cast). These files are untouched by this phase and the failures pre-date Wave 2 (documented in 03-01 SUMMARY). Logged to `deferred-items.md`.

**Total deviations:** 1 inherited cleanup (committed with Task 2), 0 auto-fixes required. Plan executed as written.

## Output spec items (per plan <output>)
- **Final snap sizes:** 0.18 (collapsed) / 0.45 (expanded), snapSizes [0.18, 0.45] — confirmed during human-verify; no tuning required (controls stayed reachable)
- **Speed key:** reused existing `speed` (not `current_speed`)
- **PageView fixed height:** 220
- **Post-verify UI tuning:** none — the user approved the sheet as built; no changes needed after the checkpoint

## Checkpoint Status
- **Task 3 (human-verify, gate=blocking):** APPROVED. The user ran the app on a device/emulator, walked the 9 verification steps (collapsed/expanded snaps, map + bottom-right controls reachable, button-driven PageView with no horizontal swipe, pause/resume freeze + no jump, exit pops the route, old top-left X gone, plausible live distance/elevation/speed), and replied "approved". No issues reported; no follow-up edits required.

## Self-Check: PASSED

All modified files verified present on disk; task commits (1c8920bd, e45e6bf3) and the finalization commit (ad33809f) verified in git log. SUMMARY present on disk. Requirements STATS-01..05 marked complete; ROADMAP phase 03 shows 2/2 plans complete.
