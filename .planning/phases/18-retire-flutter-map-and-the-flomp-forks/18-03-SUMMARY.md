---
phase: 18-retire-flutter-map-and-the-flomp-forks
plan: 03
subsystem: mobile-map
tags: [flutter, maplibre, verification, on-device, gap-candidates]

# Dependency graph
requires:
  - phase: 18-retire-flutter-map-and-the-flomp-forks (plan 01)
    provides: "effectiveBrightness relocated, LocationMarkerPosition/ServiceDisabledException local classes, dead files deleted"
  - phase: 18-retire-flutter-map-and-the-flomp-forks (plan 02)
    provides: "six map packages + two flomp/* overrides removed from pubspec.yaml; maplibre pinned to 0.3.5 exact"
provides:
  - "On-device confirmation that CLEAN-01/02/03 landed with no functional regression across all six map surfaces, online and in airplane mode"
  - "Confirmation the LocationMarkerPosition local-class swap did not break the location puck on map screen or navigation"
  - "Confirmation removing flutter_map_location_marker/flutter_map_marker_cluster's native plugin registrations left no runtime crash or permission artifact"
  - "Six documented pre-existing UI polish gaps for a follow-up gap-closure plan (not caused by this phase's dependency removal — see Gap Candidates below)"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified: []

key-decisions:
  - "No code changes in this plan by design — verification-only checkpoint. All 7 acceptance-criteria checks passed; 6 additional UI observations were captured during the walk as candidates for a separate gap-closure plan, since they are cosmetic/behavioral gaps on existing screens rather than regressions introduced by removing the six packages."

patterns-established: []

requirements-completed: [CLEAN-01, CLEAN-02, CLEAN-03]

# Metrics
duration: ~20min (automated precondition + human on-device pass)
completed: 2026-07-10
---

# Phase 18 Plan 03: On-Device Verification Summary

**All seven acceptance-criteria checks passed on a physical Android device: all six map surfaces render online and in airplane mode with no regression, the location puck survives the LocationMarkerPosition local-class swap, and no crash/leaked-subscription/missing-plugin error appeared after removing the six map packages and two flomp forks. CLEAN-01/02/03 are confirmed with no regression, closing Phase 18 and the v1.4 MapLibre Migration milestone. Six additional UI polish issues (unrelated to the dependency removal) were observed during the walk and are logged below as gap candidates.**

## Automated Precondition

- `cd app && flutter clean && flutter pub get`: exit 0, resolved cleanly.
- `flutter analyze`: 36 pre-existing issues (deprecated icon names, one unused test import), zero errors, none map-related.
- `flutter test`: 80/83 pass; 3 pre-existing failures (`feed_item_test.dart` x2, `settings_screen_test.dart` x1) — same baseline as plans 18-01/18-02, confirmed unrelated.

## Human On-Device Verification (Android, physical device, fresh install)

| # | Check | Result |
|---|-------|--------|
| 1 | Trail detail nav flow: transition into trail map, no crash | PASS |
| 2 | Trail map (WandererMap): basemap, track, arrows, waypoints, pins, compass online; `.pmtiles` + labels offline | PASS |
| 3 | List (SearchMap): every polyline renders, camera fits all; same offline | PASS |
| 4 | List map (SearchMap full-screen): renders and fits every trail; same offline | PASS |
| 5 | Map screen (SearchMap): clusters, cluster/point tap, category icons, location puck renders/updates; puck tracks offline | PASS |
| 6 | Navigation (ml.MapLibreMap): route, native puck, heading-up follow, maneuvers, drag-break/recenter, compass online; offline maneuvers from cache, offline indicator, `.pmtiles` + puck | PASS |
| 7 | No-crash sanity: background/re-foreground, no crash/leaked subscription/missing-plugin error | PASS |

All 7/7 acceptance criteria PASS. CLEAN-01, CLEAN-02, CLEAN-03 confirmed on real hardware with no regression.

## Gap Candidates (observed during the walk, NOT required acceptance criteria — feed to `/gsd-plan-phase 18 --gaps` or a new phase)

These are pre-existing UI behaviors on screens this migration touched, not regressions caused by the package removal itself. Recorded verbatim from on-device observation:

1. **Missing directional arrows on trail polylines in `list_detail_screen.dart` and `list_detail_map_screen.dart`** — trails render as plain lines without direction-of-travel arrows (present on `trail_detail_map_screen.dart` and `navigation_screen.dart`).
2. **No compass widget in `list_detail_map_screen.dart`** — other map surfaces (trail map, navigation) show a compass control; this screen doesn't.
3. **No scale/attribution control visible in `list_detail_screen.dart`.**
4. **Attribution control should always start collapsed** across all map screens (currently inconsistent/expanded on load in at least one surface).
5. **Camera in `trail_detail_map_screen.dart` and `trail_detail_screen.dart` centers on the trail's start point instead of fitting the full trail bounds** — other surfaces (list, list map, map screen) correctly fit bounds to all visible trails; the single-trail detail view does not fit the one trail's own bounds.
6. **Control layout shift in `trail_detail_map_screen.dart` when rotating the map**: when the map is rotated off-north, the compass control correctly appears, but the other map controls shift to horizontally centered instead of staying right-aligned; they return to the correct right-aligned position once the map is rotated back to north and the compass disappears.

## Issues Encountered

None blocking. See Gap Candidates above for non-blocking follow-up items.

## Next Phase Readiness

- Phase 18 complete — CLEAN-01, CLEAN-02, CLEAN-03 all confirmed on real hardware.
- v1.4 "MapLibre Migration" milestone is complete: all six map surfaces run on native MapLibre, `flutter_map` family + flomp forks are gone, `maplibre` is pinned exact.
- The 6 gap candidates above are recommended for a follow-up gap-closure plan or a small polish phase — none block milestone completion since they predate/are orthogonal to this phase's dependency-removal scope.

## Self-Check: PASSED

- No files modified (verification-only plan, as declared in frontmatter `files_modified: []`).

---
*Phase: 18-retire-flutter-map-and-the-flomp-forks*
*Completed: 2026-07-10*
