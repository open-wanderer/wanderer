---
phase: quick-260717-seb
plan: 01
subsystem: route-planner
tags: [flutter, dio, reverse-geocoding, nominatim, route-planner]

# Dependency graph
requires:
  - phase: 20-route-planner-views
    provides: RouteAnchorListTab (WAYP-04/05 reorder/delete list) that this quick task adds titles to
provides:
  - "reverse_geocode_util.dart: pure Dart port of web's getLocationDescription/getReverseLocationResult, plus searchLocationReverseStructured Dio fetch"
  - "Route Anchors tab rows show resolved street/place titles instead of 'Anchor N'"
affects: [route-planner, quick-tasks]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Sequential (non-parallel) per-item async batch with a single shared CancelToken, cancelled and replaced on each re-trigger — respects a server-side rate limit without a request queue library"
    - "Local widget-state result cache keyed by rounded-coordinate string, mirroring a Svelte component's module-level Map + $state record — no new provider, no model field"

key-files:
  created:
    - app/lib/util/reverse_geocode_util.dart
    - app/test/util/reverse_geocode_util_test.dart
  modified:
    - app/lib/components/route_planner/route_anchor_list_tab.dart

key-decisions:
  - "Built a dedicated set of pure functions in reverse_geocode_util.dart rather than extending global_search_provider.dart's private _buildLocationDescription, since that helper lacks the road step and label/fullLabel split this feature needs"
  - "Resolved titles cached as local widget state (_locations Map) in _RouteAnchorListTabState, not added to the immutable RouteAnchor model or a new Riverpod provider"

patterns-established:
  - "Reverse-geocode batch: cancel previous CancelToken, create fresh one, sequential for-await loop (never Future.wait) checking isCancelled each iteration, cache-then-pending dedupe map — reusable for any future per-row/per-marker geocoding sweep in this app"

requirements-completed: [QUICK-260717-seb]

# Metrics
duration: 25min
completed: 2026-07-17
---

# Quick Task 260717-seb: Reverse-geocode route planner anchors Summary

**Route Anchors tab rows now show resolved street/place titles (e.g. "Bahnhofstrasse, Zürich, Switzerland") via a sequential, coordinate-cached, best-effort Nominatim reverse-geocode batch, ported one-to-one from web's `search_store.ts`/`trail_anchor_list.svelte`.**

## Performance

- **Duration:** ~25 min
- **Completed:** 2026-07-17
- **Tasks:** 2 (TDD task 1 + task 2)
- **Files modified:** 3 (1 new util, 1 new test file, 1 existing widget)

## Accomplishments
- New pure-Dart `reverse_geocode_util.dart` mirroring web's `getLocationDescription`/`getReverseLocationResult` exactly (road-first ordering, city fallback chain, label-omits-country vs fullLabel-includes-country, label-falls-back-to-fullLabel-when-empty), plus a `searchLocationReverseStructured` Dio fetch against `/geocoding/reverse`.
- 13 unit tests covering every parsing rule in the plan's `<behavior>` block, all passing.
- `route_anchor_list_tab.dart` wired with a sequential (never `Future.wait`), coordinate-cached, single-shared-`CancelToken` reverse-geocode batch triggered from `build()` via a signature-guarded `addPostFrameCallback`, replacing the `'Anchor N'` literal title with the resolved location (falling back to `'Anchor N'` until resolved or on failure).

## Task Commits

Each task was committed atomically (TDD RED -> GREEN, then the wiring task):

1. **Task 1 (RED): add failing tests for reverse-geocode util** - `aefa55fb` (test)
2. **Task 1 (GREEN): add reverse-geocode util** - `f56d31e7` (feat)
3. **Task 2: wire reverse-geocoded titles into route anchor list** - `90cbaa37` (feat)

**Plan metadata:** committed separately by the orchestrator after this summary.

## Files Created/Modified
- `app/lib/util/reverse_geocode_util.dart` - `ReverseLocationResult` class, pure `getLocationDescription`/`getReverseLocationResult`, async `searchLocationReverseStructured(Dio, lat, lon, {includeRoad, cancelToken})`
- `app/test/util/reverse_geocode_util_test.dart` - 13 unit tests for the pure parsing functions (no Dio/network mocking, per plan instruction)
- `app/lib/components/route_planner/route_anchor_list_tab.dart` - `_locations`/`_pending`/`_batchToken`/`_lastAnchorSignature` state, `_locationCacheKey`/`_loadAnchorLocation`/`_loadAnchorLocations`/`_commonAnchorCountry`/`_anchorTitle` helpers, post-frame-triggered batch on anchor-signature change, `dispose()` cancels in-flight batch, ListTile title now `_anchorTitle(...)` with `maxLines: 1` + ellipsis overflow

## Decisions Made
- Dedicated reverse-geocode util instead of extending `global_search_provider.dart`'s private helper (see key-decisions above) — kept per plan's explicit instruction.
- Resolved-title cache lives as local `_RouteAnchorListTabState` fields, not the `RouteAnchor` model or a new provider — matches web's module-scoped `Map`/`$state` pattern and keeps the anchor model's `id`/`lat`/`lon` fields untouched per success criteria.

## Deviations from Plan

None - plan executed exactly as written. `dart format` was run over both touched files as a final formatting pass (routine, no behavior change) before the Task 2 commit.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required. Feature uses the already-deployed `/api/v1/geocoding/reverse` server proxy (sibling of the existing `/geocoding/search` endpoint already used by `global_search_provider.dart`).

## Next Phase Readiness
- Manual on-device verification recommended (per plan's `<verification>` section, optional): open the Route Planner, drop 2-3 anchors, open the Route Anchors tab, confirm each row resolves from "Anchor N" to a street/place name within a few seconds, and confirm delete/reorder does not re-fetch already-resolved anchors.
- No blockers for future route-planner work; `RouteAnchor`'s `id`/`lat`/`lon` fields and route saving/persistence/handoff flow are unaffected.

---
*Quick Task: 260717-seb*
*Completed: 2026-07-17*

## Self-Check: PASSED

All created/modified files verified present on disk; all 3 task commit hashes (aefa55fb, f56d31e7, 90cbaa37) verified present in git log.
