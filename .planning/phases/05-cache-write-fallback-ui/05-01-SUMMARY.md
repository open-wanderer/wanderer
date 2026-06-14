---
phase: 05-cache-write-fallback-ui
plan: "01"
subsystem: flutter-app-util
tags: [dart, flutter, navigation, offline, utility, testing]
dependency_graph:
  requires: []
  provides:
    - "buildNavShape(List<LatLng>): shared Valhalla shape-downsampling helper in gpx_util.dart"
    - "app/test/util/gpx_util_test.dart: unit tests for buildNavShape"
  affects:
    - "app/lib/util/navigation_launch_util.dart (05-03 will call buildNavShape)"
    - "app/lib/services/trail_download_service.dart (05-04 will call buildNavShape)"
tech_stack:
  added: []
  patterns:
    - "Top-level public function outside extension for shared utility in gpx_util.dart"
    - "Coordinate-value dedup comparison for Map<String,double> (identity != not safe)"
key_files:
  created:
    - app/test/util/gpx_util_test.dart
  modified:
    - app/lib/util/gpx_util.dart
decisions:
  - "Fix Map<String,double> reference-equality dedup bug: compare coordinate values not map identity"
  - "buildNavShape placed as top-level function (not in GpxMappingUtils extension) since it takes List<LatLng> not Gpx"
metrics:
  duration_minutes: 10
  completed_date: "2026-06-14T15:19:56Z"
  tasks_completed: 2
  files_modified: 2
---

# Phase 05 Plan 01: Extract buildNavShape Helper Summary

**One-liner:** Extracted `buildNavShape(List<LatLng>)` downsampling helper into `gpx_util.dart` with ≤500-point cap, first/last preservation, and coordinate-value dedup fix; 4 unit tests all pass.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Extract buildNavShape helper into gpx_util.dart | 3cb3fbc4 | app/lib/util/gpx_util.dart |
| 2 | Unit-test buildNavShape downsampling guarantees | f6671041 | app/lib/util/gpx_util.dart, app/test/util/gpx_util_test.dart |

## What Was Built

`buildNavShape(List<LatLng> points)` — a top-level public function in `app/lib/util/gpx_util.dart` that converts a GPX point list into a Valhalla shape list with these guarantees:

- When `points.length <= 500`: maps all points to `{'lat': p.latitude, 'lon': p.longitude}` unchanged.
- When `points.length > 500`: computes `step = (points.length / 499).ceil()`, samples every `i % step == 0` point, then appends the last point (deduplicating by coordinate values to avoid double-entry when modulo sampling already captured it).

The function serves as the single shared contract for OFFLINE-01/D-08: both the online launch path (`navigation_launch_util.dart`, wired in 05-03) and the download cache write path (`trail_download_service.dart`, wired in 05-04) will call this same function so the cached shape and the online-requested shape are byte-identical.

## Verification

- `flutter analyze lib/util/gpx_util.dart` — no issues
- `flutter test test/util/gpx_util_test.dart` — 4/4 tests pass:
  - 2-point passthrough: output length 2, correct lat/lon maps
  - 500-point no-op: all 500 returned unchanged
  - 1000-point cap: output <= 500, first/last preserved
  - 501-point dedup: output <= 500, last point appears exactly once

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed Map identity-equality dedup in buildNavShape**

- **Found during:** Task 2 (501-point dedup test failure)
- **Issue:** The original code in `navigation_launch_util.dart` uses `sampled.last != lastPoint` to deduplicate the last point. In Dart, `Map<String, double>` uses reference identity for `!=`, not structural equality. Two separate `Map` instances with the same key-value pairs are NOT equal with `!=`. So when step=2 and index 500 was already sampled by the modulo loop, `sampled.last != lastPoint` returned `true` (different objects), causing a duplicate.
- **Fix:** Replaced `sampled.last != lastPoint` with explicit coordinate comparison: `sampled.last['lat'] != lastLat || sampled.last['lon'] != lastLon`. This is correct Dart value comparison.
- **Note:** The same bug exists in `navigation_launch_util.dart` but only triggers when `points.length` happens to be a multiple where the modulo loop already captures the last index. The 501-point case exposes it because step=2 visits index 500 exactly. Plan 05-03 (which replaces the inline logic with `buildNavShape`) will automatically benefit from this fix.
- **Files modified:** app/lib/util/gpx_util.dart
- **Commit:** f6671041

## Self-Check: PASSED

- FOUND: 05-01-SUMMARY.md
- FOUND: app/lib/util/gpx_util.dart
- FOUND: app/test/util/gpx_util_test.dart
- FOUND: commit 3cb3fbc4 (feat(05-01): add buildNavShape helper to gpx_util.dart)
- FOUND: commit f6671041 (test(05-01): unit tests for buildNavShape downsampling and fix dedup)
