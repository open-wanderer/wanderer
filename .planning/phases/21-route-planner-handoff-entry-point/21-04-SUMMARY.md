---
phase: 21-route-planner-handoff-entry-point
plan: 04
subsystem: mobile-route-planner
tags: [flutter, riverpod, go_router, route-planner, handoff]

# Dependency graph
requires:
  - phase: 21-route-planner-handoff-entry-point
    plan: 01
    provides: "finishPlanning(ref, navContext, travelProfile) orchestration util"
  - phase: 21-route-planner-handoff-entry-point
    plan: 03
    provides: "real /route-planner entry point supplying travelProfile into RoutePlannerScreen"
provides:
  - "App-bar Finish action on RoutePlannerScreen, gated on >=2 route anchors (D-05), wired to finishPlanning (HANDOFF-01)"
  - "Undo/redo relocated from the app bar into the top-right map controls Column (D-04)"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Instance-scoped re-entrancy guard (_finishing bool) around an async handoff handler, per Phase 19's established non-static-field discipline"
    - "Literal single-line acceptance-criteria greps preserved over dart format's default line-wrapping, per this phase's own established precedent (20-05, 21-01)"

key-files:
  created: []
  modified:
    - app/lib/routes/route_planner_screen.dart

key-decisions:
  - "Kept `tooltip: state.anchors.length >= 2 ? l10n.finish : l10n.finish_disabled_hint,` and the finishPlanning(...) call each on one line, overriding dart format's default 80-col wrap, so the plan's literal acceptance-criteria greps match verbatim (established precedent: 20-05, 21-01)"

requirements-completed: [HANDOFF-01]

# Metrics
duration: 10min
completed: 2026-07-17
---

# Phase 21 Plan 04: Route Planner Handoff Trigger — App-Bar Finish Action Summary

**App-bar "Finish" action on `RoutePlannerScreen`, gated on >=2 route anchors, wired to Plan 01's `finishPlanning`; undo/redo relocated into the top-right map controls Column to free the app-bar slot Finish now occupies.**

## Performance

- **Duration:** 10 min
- **Completed:** 2026-07-17T15:11:17Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments

- App-bar `actions` now holds a single Finish `IconButton` (`FontAwesomeIcons.check`, size 18, `IconButton.styleFrom(backgroundColor: colorScheme.surface)`), replacing the two undo/redo `IconButton`s that previously lived there
- Finish is disabled below 2 anchors (`onPressed: null`, Flutter's standard ~38%-opacity treatment) with a long-press tooltip of `l10n.finish_disabled_hint`; enabled at >=2 anchors with tooltip `l10n.finish` (D-05)
- Tapping Finish invokes the new `_onFinish` handler, which awaits Plan 01's `finishPlanning(ref: ref, navContext: context, travelProfile: widget.travelProfile)` — the one-time elevation merge + GPX-track-only draft-Trail handoff to `/trail/create/edit` (HANDOFF-01)
- `_onFinish` is guarded by an instance-scoped `bool _finishing` field (never `static`, per Phase 19's own established discipline) so a double-tap cannot fire two `/valhalla/height` fetches or two navigations while a handoff is in flight (T-21-04-01)
- Undo/redo moved out of the app bar into the top-right map controls `Column` (`Positioned(top: 128, right: 0)`), stacking below the auto-routing toggle (Auto-routing → Undo → Redo, D-04) via new `_buildUndoButton`/`_buildRedoButton` methods that mirror `_buildAutoRoutingToggle`'s exact pill styling (canvasColor container, 24px radius, compact `IconButton`, accent-primary enabled tint / `.4`-alpha disabled tint) — only their position changed, not their visual treatment or behavior
- Updated the stale controls-Column doc comment to reflect undo/redo's new home and that Finish took the app-bar slot
- `flutter analyze lib/routes/route_planner_screen.dart` and whole-app `flutter analyze` both report no new issues (46 pre-existing info/warning-level issues unrelated to this file)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add Finish app-bar action, relocate undo/redo to the controls Column, wire finishPlanning (D-04/D-05, HANDOFF-01)** - `78de79d2` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `app/lib/routes/route_planner_screen.dart` - Added `import 'package:wanderer/i18n/app_localizations.dart'` and `import 'package:wanderer/util/route_planner_handoff_util.dart'`; added `bool _finishing` field; `build()` now resolves `l10n`; app-bar `actions` swapped for `_buildFinishAction(state, l10n)`; new `_buildFinishAction`, `_onFinish`, `_buildUndoButton`, `_buildRedoButton` methods; controls `Column` in `_buildMap` now stacks auto-routing toggle + undo + redo

## Decisions Made

- Kept the tooltip ternary and the `finishPlanning(...)` call each on a single line (overriding `dart format`'s default 80-column wrap) so the plan's own literal acceptance-criteria greps (`finishPlanning(ref: ref, navContext: context, travelProfile: widget.travelProfile)`, `l10n.finish_disabled_hint` on the tooltip line) match verbatim — same precedent this phase already established in 20-05 and 21-01 for grep-sensitive lines
- Left `state.anchors.length >= 2` inlined twice (tooltip choice + `onPressed` gate) rather than hoisting to a local `canFinish` variable, since the plan's acceptance criteria explicitly checks `grep -c "anchors.length >= 2"` returns >= 2

## Deviations from Plan

None - plan executed exactly as written. All `<read_first>` context (existing app-bar/controls layout, `_buildAutoRoutingToggle` pill styling, `finishPlanning` signature) matched the plan's description precisely; no architectural, bug-fix, or missing-functionality deviations were needed.

## Issues Encountered

- Running `dart format` on the file (habitual post-edit step) reformatted the tooltip ternary and the `finishPlanning(...)` call onto multiple lines, breaking two of the plan's literal acceptance-criteria greps. Reverted just those two spans back to single-line by hand (not a plan deviation — a formatting/tooling interaction, resolved before verification, matching this phase's own established precedent for grep-sensitive lines).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

This is the final plan of Phase 21 and the final plan of the v1.5 Route Planner milestone. The plan→create loop is now closed end-to-end:
`TrailSourceSelectScreen` → hike/bike sheet (Plan 03) → `RoutePlannerScreen` (Phase 19/20) → app-bar Finish (this plan) → `finishPlanning` (Plan 01) → `/trail/create/edit`.

HUMAN on-device verification (per this plan's `<verification>` section and the phase's `human_verify_mode: end-of-phase` config) is still outstanding: confirm Finish disabled/enabled states, tooltip copy, undo/redo functioning from their new controls-Column home, and the full handoff (map preview renders, category pre-selected, waypoints section shows its normal empty state, saved trail has a real non-zero-distance track).

No blockers for milestone completion beyond that manual on-device pass.

---
*Phase: 21-route-planner-handoff-entry-point*
*Completed: 2026-07-17*

## Self-Check: PASSED

- FOUND: app/lib/routes/route_planner_screen.dart
- FOUND: .planning/phases/21-route-planner-handoff-entry-point/21-04-SUMMARY.md
- FOUND commit: 78de79d2
