---
phase: 34-dart-conversion-port
plan: 01
subsystem: gpx-conversion
tags: [dart, gpx, flutter, elevation, haversine, offline]

# Dependency graph
requires:
  - phase: 33-conversion-correctness
    provides: The corrected TS gpx-metrics-computation.ts/gpx.ts algorithm (CONV-01..05 fixes) this plan ports
provides:
  - "A pure, offline Dart module (app/lib/util/gpx_conversion_util.dart) computing distance/elevation/duration/bbox/centroid from a Gpx with no network call"
  - "sanitizeGpxEleAndTime + parseGpxSafely neutralizing package:gpx's five confirmed GpxReader crash inputs"
  - "GpxMetricsComputation: line-for-line Dart port of the defer-then-publish elevation noise filter and threshold-gated distance smoothing"
  - "computeTrailMetrics(Gpx): the public entry point assembling distance/elevationGain/elevationLoss/duration/bbox/centroid/pointCount"
  - "A 32-test Dart suite pinning every CONV-01..05 defect case plus the D-04 final-vs-smoothed guard, matching the corrected TS suite's expected values"
affects: [35-offline-trail-creation, 36-local-first-recording-and-automatic-upload]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pre-parse sanitize pass (sanitizeGpxEleAndTime) mirroring the existing sanitizeGpxEmail regex-rewrite precedent, chained ahead of GpxReader().fromString()"
    - "Defer-then-publish elevation noise filter ported statement-for-statement, not simplified into a plain accumulator"
    - "finalElevationGain/finalElevationLoss (not the monotonic totalElevationGainSmoothed/LossSmoothed) as the sole public elevation output for a completed track"

key-files:
  created:
    - app/lib/util/gpx_conversion_util.dart
    - app/test/util/gpx_conversion_util_test.dart
  modified: []

key-decisions:
  - "Imports (dart:math, package:maplibre) added incrementally per task rather than all upfront in Task 1, so each task's own flutter analyze run stays warning-free (unused_import) instead of only becoming clean once Task 2 lands"
  - "Task 2 and Task 3's bare-GpxMetricsComputation cases (D-04 guard, CONV-05 jitter) parse a GPX via parseGpxSafely then flatten it, rather than constructing Wpt objects by hand, so every fixture also exercises the sanitize pass"

patterns-established:
  - "New Dart ports of TS algorithms live in their own file (not extending Gpx via an extension), named distinctly from any existing same-purpose extension to avoid collision (see gpx_util.dart's older, still-buggy GpxMappingUtils.getTotals(), untouched by this plan)"

requirements-completed: [PORT-01]

# Metrics
duration: 20min
completed: 2026-07-31
---

# Phase 34 Plan 01: Dart Conversion Port — Core Algorithm Summary

**Line-for-line Dart port of the Phase 33-corrected GPX→trail metrics algorithm (defer-then-publish elevation filter, smoothed distance, sanitize-before-parse), pinned by a 32-test suite that reproduces every CONV-01..05 defect value from the TS suite exactly.**

## Performance

- **Duration:** ~20 min
- **Tasks:** 3
- **Files modified:** 2 (both new)

## Accomplishments
- `app/lib/util/gpx_conversion_util.dart`: a pure, offline (D-14, zero dio/riverpod/dart:io imports) Dart module exposing `sanitizeGpxEleAndTime`, `parseGpxSafely`, `parseGpxElevation`, `GpxMetricsComputation`, `GpxTrailMetrics`, `computeTrailMetrics`
- Neutralized all four confirmed `GpxReader` string-body crash inputs (`<ele></ele>`, whitespace-only `<ele>`, non-numeric `<ele>`, empty `<time></time>`) via a pre-parse regex sanitize pass mirroring `sanitizeGpxEmail`'s precedent, while preserving genuine `<ele>0</ele>` and pretty-printed elevations
- Ported the full defer-then-publish elevation noise filter and threshold-gated distance smoothing verbatim; `finalElevationGain`/`finalElevationLoss` (never the monotonic `*Smoothed` pair) are what `computeTrailMetrics` reports
- Proved the port matches the corrected TS suite's exact expected values (88, 80, 15, 24, 16, 7, 134.59, 100.075, 110.083, 444.78) across 32 Dart tests, all green on the first run
- Proved the suite is non-vacuous: temporarily reading the monotonic smoothed fields instead of `final*` fails 7 of the 32 cases (including the 88m scramble and the mid-swing case), then reverted cleanly (`git status` clean on the source file)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add the pre-parse GPX sanitize pass and the safe-parse entry point** - `67095f8e` (feat)
2. **Task 2: Port GpxMetricsComputation and computeTrailMetrics from the Phase 33-corrected TypeScript** - `36f1a797` (feat)
3. **Task 3: Pin every CONV-01..05 defect case with a Dart unit suite** - `2420ae5f` (test)

_Non-TDD plan; each commit is a complete, verified unit of work rather than a RED/GREEN pair._

## Files Created/Modified
- `app/lib/util/gpx_conversion_util.dart` - Sanitize pass, safe parse entry point, `GpxMetricsComputation` (elevation noise filter + distance smoothing), `GpxTrailMetrics`, `computeTrailMetrics`
- `app/test/util/gpx_conversion_util_test.dart` - Sanitize/parse crash-input suite (16 tests) + `computeTrailMetrics` CONV-01..05 defect suite (16 tests, 32 total)

## Decisions Made
- Deferred adding `dart:math`/`package:maplibre` imports to Task 2 (instead of all-at-once in Task 1 as the plan's literal action text listed) so Task 1's own `flutter analyze` run has zero unused-import warnings before those symbols (`max`, `Geographic`, `SphericalGreatCircle`) are actually used — same end-state file, no behavior difference, purely a within-plan sequencing choice
- Reworded three doc comments that would otherwise have tripped the plan's own literal `grep -c "cumulativeDistance"`/`grep -c "getTotals"` acceptance gates (the gates count comment text too) — replaced the literal identifier names with equivalent descriptive phrasing ("the per-point cumulative-distance array", "the public-metrics assembly logic") while preserving the same rationale

## Deviations from Plan

None — plan executed as written; the two items above are within-plan sequencing/wording choices, not scope changes, bug fixes, or missing-functionality additions.

## Issues Encountered

None. The port matched the corrected TS suite's exact values on the first test run for all 32 cases, including all five previously-fixed CONV-01..05 defect classes and the D-04 final-vs-smoothed guard — no debugging iteration was needed on the algorithm itself.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `computeTrailMetrics`, `parseGpxSafely`, and `GpxMetricsComputation` are ready for later plans in this phase to redirect the three capture paths (recording, planner, `.gpx` import) onto, per PORT-03
- `app/lib/util/gpx_util.dart` is untouched, as scoped — its older, still CONV-01-buggy `GpxMappingUtils.getTotals()` extension remains live for `elevation_profile.dart`/`trail_panel.dart` until plan 34-04 (D-17) redirects and deletes it
- The shared fixture corpus (PORT-02) and the redirect of the three capture paths (PORT-03) are separate, not-yet-executed plans in this phase; this plan's module is the dependency they build on

---
*Phase: 34-dart-conversion-port*
*Completed: 2026-07-31*

## Self-Check: PASSED

- FOUND: app/lib/util/gpx_conversion_util.dart
- FOUND: app/test/util/gpx_conversion_util_test.dart
- FOUND: .planning/phases/34-dart-conversion-port/34-01-SUMMARY.md
- FOUND commit: 67095f8e
- FOUND commit: 36f1a797
- FOUND commit: 2420ae5f
