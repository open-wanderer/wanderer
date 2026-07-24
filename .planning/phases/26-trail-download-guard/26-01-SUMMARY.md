---
phase: 26-trail-download-guard
plan: 01
subsystem: mobile-offline-maps
tags: [flutter, dart, objectbox, riverpod, offline-regions, unit-testing]

# Dependency graph
requires:
  - phase: 22-region-package-data-model
    provides: "RegionEntity with discrete minLon/minLat/maxLon/maxLat bbox fields and the computed RegionStatus status getter"
  - phase: 25.1-local-http-tile-proxy
    provides: "Stable region download engine (TileRepositoryStatus) the Plan 03 wiring will trigger from the guard sheet"
provides:
  - "Pure bboxesOverlap axis-aligned rectangle intersection function"
  - "overlappingRegions(trail, catalog) -- all catalog regions overlapping a trail's bbox, any status"
  - "missingCoverageRegions(trail, catalog) -- overlapping regions excluding downloaded/updateAvailable (GUARD-04)"
affects: [26-02-missing-coverage-sheet, 26-03-download-guard-wiring]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure top-level function util module (library; doc header, no class wrapper) mirroring byte_format_util.dart"
    - "Two-set coverage output (overlappingRegions vs missingCoverageRegions) so callers distinguish 'fully covered' from 'no region overlaps at all'"

key-files:
  created:
    - app/lib/util/trail_coverage_util.dart
    - app/test/util/trail_coverage_util_test.dart
  modified: []

key-decisions:
  - "Degenerate-bbox test fixture uses a genuinely disjoint region (far side, non-symmetric) rather than a region whose range brackets the swapped min/max -- the four-comparison overlap formula doesn't universally return false for every b-box against a degenerate a-box, only for realistic non-adversarial region positions; the test now matches the actual threat scenario (T-26-01: a corrupted local row silently drops out of a real region's overlap check)."

patterns-established:
  - "Compute overlappingRegions and missingCoverageRegions as two distinct pure functions (not a single boolean) so Plan 03 can tell a fully-covered trail apart from a trail with no catalog region overlap at all (D-04)."

requirements-completed: [GUARD-01, GUARD-04]

# Metrics
duration: 12min
completed: 2026-07-24
---

# Phase 26 Plan 01: Trail Coverage Util Summary

**Pure bboxesOverlap/overlappingRegions/missingCoverageRegions functions deciding which trail-overlapping regions still need downloading, with updateAvailable treated as covered (GUARD-04)**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-24T11:25:00Z
- **Completed:** 2026-07-24T11:37:48Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments
- `bboxesOverlap` — axis-aligned rectangle intersection over eight named doubles, edges-touching counts as overlap, degenerate `min > max` input returns `false` instead of throwing
- `overlappingRegions(trail, catalog)` — every catalog region overlapping the trail's bbox regardless of status
- `missingCoverageRegions(trail, catalog)` — the GUARD-04 filter: overlapping regions excluding `downloaded`/`updateAvailable`
- Table-driven `flutter_test` suite (11 tests) covering overlap true/false/edge-touching, degenerate-bbox safety, fully-covered, no-region-gap, and the `updateAvailable`-satisfies-coverage case
- TDD RED -> GREEN gate followed: failing test committed first, then the implementation

## Task Commits

Each task was committed atomically:

1. **Task 1: Pure coverage functions (test-first)** — RED: `d56b16b3` (test), GREEN: `a5ee2559` (feat)

_Note: TDD task produced two commits (test -> feat); no refactor commit was needed, the implementation was already clean on first pass._

## Files Created/Modified
- `app/lib/util/trail_coverage_util.dart` — three pure top-level functions (`bboxesOverlap`, `overlappingRegions`, `missingCoverageRegions`); no Riverpod/ObjectBox I/O/network dependency
- `app/test/util/trail_coverage_util_test.dart` — 11 table-driven tests across 3 `group` blocks (`bboxesOverlap`, `overlappingRegions`, `missingCoverageRegions`)

## Decisions Made
- Built `Trail`/`RegionEntity` fixtures directly via public constructors (`Trail.empty().copyWith(...)`, `RegionEntity(...)`), no ObjectBox store needed since `ToOne.target` can be set in-memory without persistence.
- Covered the `notDownloaded` and `downloaded`/`updateAvailable` ends of `RegionStatus` via constructible `DownloadedTilePackageEntity` targets (no store required); did not attempt to construct a `downloading`/`error` fixture in this plan since the coverage filter only branches on `downloaded`/`updateAvailable` vs. everything else — a `notDownloaded` fixture already exercises the "everything else" branch representatively. No coverage gap for the guard's actual decision logic.
- Fixed the initial degenerate-bbox test fixture (see Deviations) so the assertion matches the real drop-out behavior rather than an accidentally-true edge case.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected the degenerate-bbox test fixture's region box**
- **Found during:** Task 1, GREEN phase (running the new test against the new implementation)
- **Issue:** The first-draft test paired a degenerate `a` box (`minLon:10, maxLon:0`) against a `b` box (`0` to `20`) that happened to bracket both of `a`'s swapped values on one axis, causing `bboxesOverlap` to evaluate `true` instead of the expected `false` — not a bug in the implementation (which matches the plan's `<action>` formula verbatim), but a bug in the test's own fixture choice. The four-comparison overlap formula only guarantees a degenerate `a` drops out against realistic, non-adversarially-positioned region boxes, not against every possible `b`.
- **Fix:** Changed the test's `b` box to a genuinely disjoint far-side region (`100` to `120`), so one of the four comparisons naturally fails, matching the plan's `<action>` description and the T-26-01 threat mitigation intent (a corrupted local row silently drops out of a real region's check).
- **Files modified:** `app/test/util/trail_coverage_util_test.dart`
- **Verification:** `flutter test test/util/trail_coverage_util_test.dart` — all 11 tests pass
- **Committed in:** `a5ee2559` (part of the GREEN/feat commit, since the fixture fix landed alongside the implementation before either was committed as passing)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Test-fixture-only correction; the implementation itself matches the plan's `<action>` text exactly. No scope creep.

## Issues Encountered
None beyond the fixture correction above.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
`trail_coverage_util.dart`'s three functions are ready for Plan 02 (missing-coverage bottom sheet) and Plan 03 (guard wiring into `DownloadingTrailIds.download()`) to consume directly via `ref.read(regionListNotifierProvider)` snapshots, per D-11. No blockers.

---
*Phase: 26-trail-download-guard*
*Completed: 2026-07-24*

## Self-Check: PASSED

- FOUND: app/lib/util/trail_coverage_util.dart
- FOUND: app/test/util/trail_coverage_util_test.dart
- FOUND: .planning/phases/26-trail-download-guard/26-01-SUMMARY.md
- FOUND: d56b16b3 (test commit)
- FOUND: a5ee2559 (feat commit)
