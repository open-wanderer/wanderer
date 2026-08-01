---
phase: 260721-eob
plan: 01
subsystem: mobile-navigation
tags: [flutter, valhalla, sveltekit, zod, gpx, i18n, tdd]

requires:
  - phase: 19-route-planner-core
    provides: costingForCategory / categoryForTravelProfile, mergeHeightsIntoGpx, buildNavShape, /valhalla/height precedent
  - phase: quick-260719-fjw
    provides: _saveRecordedTrack / buildDraftTrail save-track handoff to trail_create_screen
provides:
  - Authenticated POST /api/v1/valhalla/trace-route SvelteKit proxy (decoded {lat,lon} shape via Valhalla trace_route map-matching)
  - VALHALLA_TRACE_ROUTE_URL env wiring (ExternalServiceUrlKey + getValhallaTraceRouteUrl)
  - Track-save options bottom sheet (showTrackSaveOptionsSheet) with Recalculate-heights / Follow-roads toggles, both off by default
  - snapShapeToRoads + snapResultAcceptable pipeline helpers (best-effort road-snap with a bbox-diagonal truncation guard)
  - _saveRecordedTrack rewired to snap-then-heights pipeline shared by both the exit-dialog and completion-banner save entry points
affects: [mobile-navigation, route-planner-handoff]

tech-stack:
  added: []
  patterns:
    - "New Valhalla actions are cloned as sibling SvelteKit proxy routes (dedicated Zod schema per action, not shared) under web/src/routes/api/v1/valhalla/<action>/+server.ts"
    - "Best-effort Valhalla transforms in Flutter always fall back silently to the pre-transformation value on any error (no toast, no rethrow) — established by buildFinalPlannedGpx, now reused by snapShapeToRoads"

key-files:
  created:
    - web/src/lib/models/api/valhalla_trace_route_schema.ts
    - web/src/routes/api/v1/valhalla/trace-route/+server.ts
    - app/lib/components/navigation/track_save_options_sheet.dart
  modified:
    - web/src/lib/server/url.ts
    - web/src/lib/server/valhalla.ts
    - app/lib/util/route_planner_handoff_util.dart
    - app/lib/routes/navigation_screen.dart
    - app/lib/i18n/app_en.arb
    - app/test/util/route_planner_handoff_util_test.dart

key-decisions:
  - "snapResultAcceptable rejects on bbox-diagonal shrink (<0.6x original), never on point count — trace_route legitimately re-vertexes at Valhalla's own density"
  - "Any transform path (snap and/or heights) produces a timeless track (elevation-only merge helpers); only the no-transform path preserves the recorded breadcrumb's timestamps verbatim"
  - "New trace-route proxy is authenticated (locals.user gate), matching /valhalla/navigate's trust class rather than the unauthenticated /valhalla/route and /valhalla/height siblings"

patterns-established:
  - "TDD gate applied to the pure snapResultAcceptable helper: RED (compile-error-backed failing test) -> GREEN (implementation) -> no REFACTOR needed"

requirements-completed: [quick-260721-eob]

duration: ~35min
completed: 2026-07-21
---

# Quick Task 260721-eob: Track-Save Options Sheet Summary

**Two-toggle bottom sheet (Recalculate heights / Follow roads, both off by default) gates `_saveRecordedTrack`, backed by a new authenticated `/valhalla/trace-route` SvelteKit proxy and a bbox-diagonal truncation guard for the road-snap result.**

## Performance

- **Duration:** ~35 min
- **Completed:** 2026-07-21T08:58:13Z
- **Tasks:** 2 (Task 2 executed as TDD: RED + GREEN)
- **Files modified:** 6 modified, 3 created (plus 14 auto-regenerated `app_localizations_*.dart` getter files)

## Accomplishments
- New authenticated `POST /api/v1/valhalla/trace-route` proxy, decoding `trip.legs[].shape` into `{lat,lon}` points via the existing `decodePolyline` helper — no maneuvers, since this is track cleanup, not turn-by-turn guidance.
- New `showTrackSaveOptionsSheet` bottom sheet: both toggles default off; cancel/dismiss aborts the save with no session change.
- `snapShapeToRoads` + `snapResultAcceptable`: best-effort road-snap via the new proxy, gated by a pure bbox-diagonal truncation guard (rejects Valhalla's partial map-match/valhalla#4802 behavior without penalizing legitimate re-vertexing).
- `_saveRecordedTrack` rewired so both the exit-dialog and completion-banner save entry points open the sheet first, then run snap-then-heights on the resulting shape, with silent fallback on any Valhalla failure.

## Task Commits

Each task was committed atomically:

1. **Task 1: New authenticated Valhalla trace-route SvelteKit proxy + env wiring** - `d35be0e6` (feat)
2. **Task 2: Track-save options sheet + snap-then-heights pipeline** - TDD cycle:
   - `5c93547a` (test) - RED: failing `snapResultAcceptable` tests (function undefined)
   - `9346ef8e` (feat) - GREEN: sheet, pipeline helpers, `_saveRecordedTrack` rewire, i18n keys + generated getters

**Plan metadata:** pending (orchestrator commits SUMMARY.md/STATE.md/ROADMAP.md separately)

## Files Created/Modified
- `web/src/lib/server/url.ts` - Added `VALHALLA_TRACE_ROUTE_URL` to `ExternalServiceUrlKey`
- `web/src/lib/server/valhalla.ts` - Added `getValhallaTraceRouteUrl()`
- `web/src/lib/models/api/valhalla_trace_route_schema.ts` - New `TraceRouteRequestSchema` (bounded shape array, costing enum)
- `web/src/routes/api/v1/valhalla/trace-route/+server.ts` - New authenticated POST proxy to Valhalla `trace_route`
- `app/lib/components/navigation/track_save_options_sheet.dart` - New two-toggle bottom sheet
- `app/lib/util/route_planner_handoff_util.dart` - Added `snapResultAcceptable` (pure) and `snapShapeToRoads` (best-effort network call)
- `app/lib/routes/navigation_screen.dart` - `_saveRecordedTrack` now opens the sheet first and runs the snap-then-heights pipeline
- `app/lib/i18n/app_en.arb` - New keys: `save_recording_options`, `recalculate_heights`, `recalculate_heights_description`, `follow_roads`, `follow_roads_description` (reused existing `save` key for the confirm button)
- `app/test/util/route_planner_handoff_util_test.dart` - New `snapResultAcceptable` test group (4 cases)

## Decisions Made
- Followed the plan's discretion notes exactly: `trace-route` (hyphenated) route naming, `VALHALLA_TRACE_ROUTE_URL` env key, authenticated like `/valhalla/navigate`, i18n keys added to `app_en.arb` only.
- Reused the existing `save` ARB key for the sheet's confirm button rather than adding a duplicate.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. `npm run check` (svelte-check) reported 0 errors/warnings across all 2458 files. `flutter analyze` on the three touched Dart files surfaced only 2 pre-existing `use_build_context_synchronously` info-level lints in code paths untouched by this task (verified via `git diff` — those lines are unchanged from before this plan). All 27 tests in `route_planner_handoff_util_test.dart` pass, including the 4 new `snapResultAcceptable` cases.

## User Setup Required

None - no external service configuration required. Note (pre-existing, out of scope per CONTEXT.md): `VALHALLA_TRACE_ROUTE_URL` is not set in the repo's own `docker-compose*.yml`, matching the existing discrepancy for `VALHALLA_ROUTE_URL`, `VALHALLA_HEIGHT_URL`, and `VALHALLA_NAVIGATE_URL`.

## Next Phase Readiness

Feature is complete and automated-gate-verified. Manual on-device verification is still required (per the plan's `<verification>` section, not automatable here): trigger save from both the exit dialog and the completion banner, confirm the sheet appears with both toggles off, confirm Follow-roads snaps the path, confirm Recalculate-heights refreshes elevations, confirm cancel aborts, and confirm the save still completes (silent fallback) with network disabled.

---
*Phase: 260721-eob*
*Completed: 2026-07-21*

## Self-Check: PASSED

All created/modified files verified present on disk; all 3 referenced commit hashes (`d35be0e6`, `5c93547a`, `9346ef8e`) verified present in git log.
