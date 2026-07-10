---
phase: 17-navigation-on-maplibre
plan: 03
subsystem: mobile-map
tags: [flutter, maplibre, verification, on-device]

# Dependency graph
requires:
  - phase: 17-navigation-on-maplibre (plan 01)
    provides: "navigation_screen.dart fully migrated to ml.MapLibreMap — native puck, heading-up follow, compass, drag-break heuristic, offline basemap compose"
  - phase: 17-navigation-on-maplibre (plan 02)
    provides: "flutter_map holdout files deleted; repo-wide grep clean"
provides:
  - "On-device confirmation that NAV-01/02/03/04 and CORE-07 behave correctly on real hardware (Android)"
  - "Confirmation the pointer-count drag heuristic (RESEARCH Open Question 1) correctly discriminates one-finger drag from two-finger pinch/rotate through the platform view"
  - "Confirmation offline (airplane-mode) navigation renders the native puck and .pmtiles basemap unregressed (RESEARCH Open Question 2)"
  - "Confirmation the map style, puck, trail layers, and breadcrumb survive a live theme swap mid-navigation (Pitfall 4 re-arm)"
affects: [18-retire-flutter-map-and-flomp-forks]

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified: []

key-decisions:
  - "No code changes in this plan by design — verification-only checkpoint. All 7 on-device checks passed, so no gap-closure plan is needed."

patterns-established: []

requirements-completed: [NAV-01, NAV-02, NAV-03, NAV-04, CORE-07]

# Metrics
duration: ~15min (automated precondition + human on-device pass)
completed: 2026-07-10
---

# Phase 17 Plan 03: On-Device Verification Summary

**All seven on-device checks passed on a physical Android device, confirming NAV-01/02/03/04 and CORE-07 behave correctly on `ml.MapLibreMap`: native puck + heading-up follow, single-finger-drag-only follow-break, compass toggle with north re-animate, offline maneuver/basemap/puck rendering in airplane mode, and puck/track/breadcrumb survival across a live theme swap. No defects found — no gap-closure plan required.**

## Automated Precondition

- `cd app && flutter analyze`: 36 pre-existing issues (deprecated icon names, one unused test import), zero errors, none in navigation-related files.
- `flutter test`: 3 pre-existing failures (`feed_item_test.dart` x2, `settings_screen_test.dart` x1) — confirmed present at the pre-phase-17 baseline commit (`8aa5f369`) via `git checkout <commit> -- .` diffing, unrelated to this phase's changes. All navigation/offline-rewrite suites green.

## Human On-Device Verification (Android, physical device)

| # | Check | Result |
|---|-------|--------|
| 1 | NAV-01 render: route line, arrows, pins, native puck; camera follows puck | PASS |
| 2 | NAV-02 drag-break: 1-finger drag breaks follow; 2-finger pinch/rotate do NOT break follow | PASS |
| 3 | NAV-02/D-03: recenter re-engages follow, preserves heading-up across repeated cycles | PASS |
| 4 | NAV-03 compass: always visible, toggles heading-up, animates to north (~400ms), no crash | PASS |
| 5 | NAV-04 offline: cached maneuvers advance, `cloud_off` indicator, `.pmtiles` basemap, native puck all render in airplane mode | PASS |
| 6 | Theme-swap (Pitfall 4): puck, trail track, breadcrumb survive live light/dark toggle | PASS |
| 7 | No-crash: exit/re-enter navigation, no leaked GPS subscription or duplicate puck | PASS |

All 7/7 PASS. No FAILs to feed into a gap-closure plan.

## Issues Encountered

None.

## Next Phase Readiness

- NAV-01, NAV-02, NAV-03, NAV-04, and CORE-07 are confirmed working on real hardware — phase 17 goal achieved.
- Phase 18 (retire flutter_map and flomp forks — `pubspec.yaml` dependency removal, remaining vendor cleanup) can proceed.

## Self-Check: PASSED

- No files modified (verification-only plan, as declared in frontmatter `files_modified: []`).

---
*Phase: 17-navigation-on-maplibre*
*Completed: 2026-07-10*
