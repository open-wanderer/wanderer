---
phase: 20-route-planner-views-waypoint-list-elevation-location-search
plan: 01
subsystem: mobile-route-planner
tags: [flutter, riverpod, riverpod_annotation, gpx, route-planner, dart]

# Dependency graph
requires:
  - phase: 19-route-planner-core-waypoint-editing-routing-engine
    provides: "RouteAnchorsState/RouteAnchors notifier (anchors/segments/undo-redo, segmentKey adjacency identity), _pushUndo/_resolveSegment mutator conventions"
provides:
  - "RouteAnchors.deleteAnchor(String) — removes an anchor, collapses its <=2 touching segments into <=1 new straight segment"
  - "RouteAnchors.reorderAnchors(List<String>) — reassigns anchor order, reuses unchanged segments by adjacency diff (zero re-fetch for still-adjacent pairs)"
  - "buildGpxFromPoints(List<Geographic>) — Gpx-from-points synthesis helper in gpx_util.dart"
  - "plannedGpxProvider (plannedGpx family fn) — derives a live, pre-elevation Gpx from routeAnchorsProvider, keyed by travelProfile"
affects: [route-planner-views-waypoint-list, route-planner-elevation-tab, phase-21-handoff]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Adjacency-diff reorder: build oldByKey = {segmentKey(before,after): segment}, walk the reordered anchor list pairwise, reuse the RouteSegment instance verbatim when segmentKey matches, else synthesize a fresh straight segment"
    - "Anchor-id-chain traversal for derived Gpx: segByBefore map keyed by beforeAnchorId, walk from anchors.first following afterAnchorId links, append polyline.skip(1) per segment to avoid duplicating the shared boundary point"

key-files:
  created:
    - app/lib/provider/planned_gpx_provider.dart
    - app/lib/provider/planned_gpx_provider.g.dart
    - app/test/provider/planned_gpx_provider_test.dart
  modified:
    - app/lib/provider/route_anchor_provider.dart
    - app/lib/util/gpx_util.dart
    - app/test/provider/route_anchor_provider_test.dart
    - app/test/util/gpx_util_test.dart

key-decisions:
  - "buildGpxFromPoints never sets ele/time (D-10) — the elevation tab merges ele from Valhalla later; this stays a pure points-only skeleton"
  - "plannedGpxProvider is keyed by travelProfile (same family key as routeAnchorsProvider) so Phase 21's handoff (HANDOFF-01) can consume the same provider"

patterns-established:
  - "Pattern: derived provider walks a linked-list-like chain (anchor-id -> segment -> next anchor-id) rather than trusting array order, matching the existing segmentKey adjacency-identity convention from Phase 19"

requirements-completed: [WAYP-04, WAYP-05, PLANUI-02]

# Metrics
duration: 30min
completed: 2026-07-16
---

# Phase 20 Plan 01: Route Anchor List/Elevation Data Layer Summary

**Added deleteAnchor/reorderAnchors mutators to RouteAnchors (segment-collapse-on-delete, adjacency-diff reuse-on-reorder) plus a buildGpxFromPoints helper and plannedGpxProvider that derives a live pre-elevation Gpx by walking the anchor-id chain.**

## Performance

- **Duration:** 30 min
- **Started:** 2026-07-16T21:10:37Z
- **Completed:** 2026-07-16T21:41:19Z
- **Tasks:** 2 completed
- **Files modified:** 7 (2 created + 1 generated + 4 modified)

## Accomplishments
- `RouteAnchors.deleteAnchor(String)` removes an anchor and, when it sat between two surviving anchors, collapses its two touching segments into one new straight segment spanning predecessor/successor — pushes an undo snapshot first and auto-resolves via Valhalla when auto-routing is on (WAYP-04)
- `RouteAnchors.reorderAnchors(List<String>)` reassigns anchor order and reuses the existing `RouteSegment` instance (preserving any resolved polyline/state) for pairs that remain adjacent, issuing zero Dio calls for them; only newly-adjacent pairs get a fresh straight segment (WAYP-05, Pitfall 4)
- `buildGpxFromPoints(List<Geographic>)` in `gpx_util.dart` synthesizes a minimal `Gpx` (one `Trk`/`Trkseg`, one `Wpt` per point, no `ele`/`time`)
- `plannedGpxProvider` (`@riverpod` function provider) derives an ordered, pre-elevation `Gpx` from the in-progress route by walking the anchor-id chain from `anchors.first`, following `beforeAnchorId -> afterAnchorId` links and deduping the shared boundary point via `polyline.skip(1)` (PLANUI-02)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add deleteAnchor and reorderAnchors mutators to RouteAnchors (WAYP-04/05, D-08)** - `8c88a454` (feat)
2. **Task 2: Add buildGpxFromPoints helper + plannedGpxProvider derived provider (PLANUI-02, D-09/D-10)** - `72f1d967` (feat)

_Note: both tasks were TDD-flagged, but the implementation + tests landed together in a single commit per task since the work (found already staged in the working tree for Task 1, and written fresh for Task 2) was verified green before each commit — no separate RED-only commit was created._

## Files Created/Modified
- `app/lib/provider/route_anchor_provider.dart` - Added `deleteAnchor`/`reorderAnchors` mutators following the existing `_pushUndo`/`_resolveSegment`/`segmentKey` conventions
- `app/test/provider/route_anchor_provider_test.dart` - Added a 3-anchor seeded-container harness (`_buildThreeAnchorSeededContainer`) and delete/reorder unit tests
- `app/lib/util/gpx_util.dart` - Added `buildGpxFromPoints(List<Geographic>)` alongside the existing `buildNavShape`/`GpxMappingUtils` helpers
- `app/lib/provider/planned_gpx_provider.dart` - New `@riverpod Gpx plannedGpx(Ref ref, String travelProfile)` derived provider
- `app/lib/provider/planned_gpx_provider.g.dart` - Generated via `dart run build_runner build --delete-conflicting-outputs`
- `app/test/util/gpx_util_test.dart` - Added `buildGpxFromPoints` empty/round-trip tests
- `app/test/provider/planned_gpx_provider_test.dart` - New file: empty-route, path-order traversal (deliberately reversed segment array order to prove chain-walk not array-order), and reactive-recompute tests

## Decisions Made
- `buildGpxFromPoints` sets no `ele`/`time` on any `Wpt` — the planner has no timestamps and the elevation tab is the sole owner of elevation-merged data (D-10), matching the plan's Open Question 1 resolution
- `plannedGpxProvider` uses the exact same `travelProfile` family-key convention as `routeAnchorsProvider` (rather than a parameterless provider) so Phase 21's handoff can watch the same instance without re-deriving the key scheme

## Deviations from Plan

Task 1's implementation (`deleteAnchor`/`reorderAnchors` mutators and their unit tests) was found already present as **uncommitted working-tree changes** at the start of this execution — matching the plan's Task 1 action text verbatim (same algorithm, same doc comments, same test cases). This was verified against every acceptance criterion (target test file passes all 19 cases including the new delete/reorder ones; `deleteAnchor`/`reorderAnchors` grep counts are each exactly 1; zero array-index segment lookups; `flutter analyze` clean) before being committed as Task 1's commit. No code was rewritten — the existing implementation already matched the plan's must_haves truths.

None of Task 2's files existed yet; it was implemented fresh per the plan's action text, with one syntax correction:

**1. [Rule 1 - Bug] `Gpx` has no `trks:` named constructor parameter**
- **Found during:** Task 2 (writing `buildGpxFromPoints`)
- **Issue:** The plan's action text describes `Gpx(trks: [...])`, but `package:gpx` 2.3.0's `Gpx` class is a plain mutable-field class with only a no-arg constructor (`trks` is a settable field, not a constructor parameter) — attempting `Gpx(trks: [...])` fails to compile ("The named parameter 'trks' isn't defined").
- **Fix:** Construct a bare `Gpx()` and then assign `gpx.trks = [...]` before returning, for the non-empty-points branch; the empty-points branch returns the bare `Gpx()` directly.
- **Files modified:** `app/lib/util/gpx_util.dart`
- **Verification:** `flutter analyze lib/util/gpx_util.dart` clean; `buildGpxFromPoints` unit tests pass (empty -> empty Gpx, 3-point round-trip via `.allPoints`)
- **Committed in:** `72f1d967` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug — plan's constructor-call syntax didn't match the installed `gpx` package's actual API)
**Impact on plan:** Fix necessary for the file to compile at all; no scope creep, no behavior change from the plan's intent (still an empty-vs-populated `Gpx` with the same track/segment/point shape).

## Issues Encountered
None beyond the deviation above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `RouteAnchors.deleteAnchor`/`reorderAnchors` are ready for the Wave 2 waypoint-list-sheet UI (WAYP-04/05) to call directly.
- `plannedGpxProvider(travelProfile)` is ready for the Wave 2/3 elevation tab to `ref.watch` for its pre-elevation point source, and for Phase 21's handoff (HANDOFF-01) to reuse without a new provider.
- No blockers identified for subsequent Phase 20 plans.

---
*Phase: 20-route-planner-views-waypoint-list-elevation-location-search*
*Completed: 2026-07-16*

## Self-Check: PASSED

All 7 created/modified files verified present on disk; both task commits (`8c88a454`, `72f1d967`) verified present in git log.
