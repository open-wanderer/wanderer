---
phase: 260718-e9j
plan: 01
subsystem: ui
tags: [flutter, riverpod, go_router, gpx, maplibre, route-planner]

requires:
  - phase: 21 (route-planner-handoff-entry-point)
    provides: RoutePlannerScreen, routeAnchorsProvider, finishPlanning/buildDraftTrail handoff, /route-planner router registration
provides:
  - anchorsFromTrack(gpx) — segment-boundary anchor derivation mirroring web's initRouteAnchors
  - mergeRouteIntoTrail(existing, finalGpx) — preserves all non-track Trail fields while replacing the route
  - buildFinalPlannedGpx(ref) — extracted elevation-merge seam reused by both import and edit paths
  - RouteAnchors.seedFromTrack(points, profile, opts) — edit-mode seed path on the keepAlive provider
  - RoutePlannerScreen edit mode (seedAnchors) — seeds instead of resetting, pops instead of forward-pushing
  - trail_create_screen app-bar "Edit route" entry point
affects: [route-planner, trail-create-screen, PLANNER-02]

tech-stack:
  added: []
  patterns:
    - "Pop-with-result navigation (context.push<Gpx> awaited, context.pop(finalGpx)) as the edit-mode counterpart to the import path's forward-push handoff"
    - "Edit mode inferred from a non-empty seedAnchors list rather than a separate boolean/mode flag threaded through state"

key-files:
  created: []
  modified:
    - app/lib/util/route_planner_handoff_util.dart
    - app/lib/provider/route_anchor_provider.dart
    - app/lib/routes/route_planner_screen.dart
    - app/lib/provider/router_provider.dart
    - app/lib/routes/trail_create_screen.dart
    - app/test/util/route_planner_handoff_util_test.dart
    - app/test/provider/route_anchor_provider_test.dart

key-decisions:
  - "Edit mode is inferred purely from RoutePlannerScreen.seedAnchors being non-null/non-empty — no parallel boolean/mode flag threaded through router extras or widget state"
  - "seedFromTrack builds anchors/segments directly (not via a loop of appendAnchor) so the seed itself is never undoable, matching resetForSession's own contract"
  - "Default travel profile for the edit entry point is 'pedestrian' (no profile is stored on a Trail to restore)"
  - "_onFinish's edit-mode pop guards on State.mounted (not context.mounted) to match the established local convention (_openLocationSearch) and avoid an analyzer use_build_context_synchronously info"

requirements-completed: [PLANNER-02]

duration: ~20min
completed: 2026-07-18
---

# Quick Task 260718-e9j: Edit an existing route in the trail planner Summary

**Wired a full pop-with-result round trip so an existing trail's recorded track can be opened, edited, and merged back in the Flutter Route Planner — pulling PLANNER-02 forward from its v2 deferral.**

## Performance

- **Duration:** ~20 min
- **Tasks:** 3
- **Files modified:** 7 (5 source, 2 test)

## Accomplishments
- `trail_create_screen` gained an app-bar "Edit route" button (left of Save), enabled only when the trail has a recorded track, that opens the Route Planner seeded from that track and merges the edited result back on Finish.
- Anchor prepopulation mirrors the web app's `initRouteAnchors` exactly (segment-boundary anchors only, no interior sampling, no reverse-geocode at seed time).
- The planner's existing GPX-import forward-push flow (`finishPlanning` → `/trail/create/edit`) is completely unaffected — the edit path is a fully separate, additive pop-with-result mechanism sharing only the pure elevation-merge helper.

## Task Commits

Each task was committed atomically:

1. **Task 1: Reusable seed + reconcile helpers (provider + handoff util)** - `ee3bbe1d` (feat)
2. **Task 2: Planner edit-mode — seed on entry, pop the result on Finish** - `e93c0ed2` (feat)
3. **Task 3: Entry point — app-bar button in trail_create_screen that round-trips the route** - `9c171b27` (feat)

_No separate docs commit yet — final metadata commit follows this SUMMARY._

## Files Created/Modified
- `app/lib/util/route_planner_handoff_util.dart` - Extracted `buildFinalPlannedGpx(ref)` from `finishPlanning`; added `anchorsFromTrack(gpx)` and `mergeRouteIntoTrail(existing, finalGpx)`.
- `app/lib/provider/route_anchor_provider.dart` - Added `RouteAnchors.seedFromTrack(points, profile, opts)`.
- `app/lib/routes/route_planner_screen.dart` - Added optional `seedAnchors` constructor field, `_editMode` getter, edit-mode branch in the post-frame `initState` callback, and edit-mode pop branch in `_onFinish`.
- `app/lib/provider/router_provider.dart` - `/route-planner` builder now reads `seedAnchors` from `extra` and forwards it to `RoutePlannerScreen`.
- `app/lib/routes/trail_create_screen.dart` - New app-bar `IconButton` (gated on `trail.expand?.gpx` having a non-empty track) and `_onEditRoute` handler that pushes/awaits `Gpx` and merges it via `mergeRouteIntoTrail`.
- `app/test/util/route_planner_handoff_util_test.dart` - Added `anchorsFromTrack` and `mergeRouteIntoTrail` test groups.
- `app/test/provider/route_anchor_provider_test.dart` - Added a `seedFromTrack` test group.

## Decisions Made
- Edit mode is inferred from `seedAnchors` presence rather than a separate `mode` boolean — simpler state, fewer places that can disagree (the `mode: 'edit'` key is still passed through `extra` for readability/future debugging but is not read by any branch).
- `seedFromTrack` builds its anchor/segment lists directly and assigns state once, rather than looping `appendAnchor`, so the initial seed never lands on the undo stack (matches `resetForSession`'s existing contract of a fresh, non-undoable baseline).
- Used `mounted` (State) instead of `context.mounted` in the edit-mode `_onFinish` branch, matching the file's own established idiom (`_openLocationSearch`) and avoiding an `use_build_context_synchronously` analyzer info that `context.mounted` triggered in that specific spot.

## Deviations from Plan

None - plan executed exactly as written. The plan's literal action text for `_onFinish` used `context.mounted`; this was adjusted to `mounted` (functionally identical State-mounted check) purely to satisfy `flutter analyze` cleanly and match the file's existing convention — not a behavior change, and still satisfies the plan's own grep gate (`context.pop(finalGpx)`).

## Issues Encountered
- `flutter analyze` flagged `use_build_context_synchronously` (info-level) on the plan's literal `if (!context.mounted) return;` wording inside `_onFinish`'s edit branch, because the same method's `finally` block separately checks `State.mounted`. Switched the edit-mode guard to `mounted` to resolve — see Deviations above.
- One pre-existing `curly_braces_in_flow_control_structures` info-level lint remains in `route_anchor_provider.dart` (line ~176, inside `_resolveSegment`'s `DioException` catch, untouched by this plan) — confirmed pre-existing via `git show HEAD:...` diff, out of scope per the deviation rules' scope boundary.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None found — the entry button, seed path, and merge path are all fully wired to real data (no hardcoded/placeholder values).

## Threat Flags

None — this plan's own `<threat_model>` register already covers every new surface introduced (mergeRouteIntoTrail's dual-field write, seedFromTrack's reuse of Phase 19's Valhalla race-guard, and the no-reverse-geocode-at-seed accept disposition). No additional new surface was found during implementation.

## Next Phase Readiness

- PLANNER-02 (edit an existing trail's route) is fully implemented and can be marked complete in ROADMAP.md's deferred-items table.
- **On-device verification still needed** (Task 3's `<human-check>`): open a trail with a recorded track in create/edit, confirm the button appears (and is disabled without a track), confirm the planner opens seeded/centered on the route, confirm Finish returns to the same screen with the edited track merged and other fields intact, and confirm backing out leaves the trail unchanged.

---
*Phase: 260718-e9j*
*Completed: 2026-07-18*

## Self-Check: PASSED

All 7 modified files verified present on disk; all 3 task commit hashes (`ee3bbe1d`, `e93c0ed2`, `9c171b27`) verified present in `git log --oneline --all`.
