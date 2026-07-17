---
phase: 20-route-planner-views-waypoint-list-elevation-location-search
verified: 2026-07-17T00:00:00Z
status: human_needed
score: 4/4 must-haves verified (automated)
overrides_applied: 0
human_verification:
  - test: "Open the Elevation tab on a route with >=2 anchors and watch the profile render/update as anchors are added or moved; then drop to <2 anchors and confirm the empty-state message; confirm no /valhalla/height network activity fires while the Route Anchors tab (not Elevation) is showing."
    expected: "Elevation chart renders and live-updates only while the Elevation tab is the active TabBarView page; empty-state copy 'Add at least 2 anchors to see the elevation profile.' shows below 2 anchors; no height fetch while on the other tab."
    why_human: "Requires observing live chart rendering, tab-switch timing, and actual network call gating on a running app/device — not verifiable from static source alone (source shows correct gating logic, but the runtime behavior needs visual/network confirmation)."
  - test: "Add several route anchors, long-press-drag to reorder them in the Route Anchors tab, and confirm the map/route segments update to match the new order; tap the trailing delete icon on a row and confirm it disappears immediately with no confirmation dialog; tap the app-bar Undo button and confirm the deleted anchor is restored."
    expected: "Drag-reorder updates both the list and the map's route segments; delete is instant with no dialog/snackbar; Undo restores a deleted anchor."
    why_human: "Gesture-driven list reordering and its visual effect on the native map layer, plus the felt immediacy of delete, are interaction/visual behaviors that require on-device confirmation."
  - test: "Add an anchor and confirm the docked sheet appears at peek height; drag the handle bar to expand the sheet; switch between the Route Anchors and Elevation tabs and confirm no scroll-controller crash occurs; tap the magnifying-glass control, search for a place, select a result, and confirm the map pans/zooms to it at zoom 13; delete all anchors and confirm the sheet disappears entirely."
    expected: "Sheet docks at peek on first anchor, expands/collapses smoothly via handle drag, tab-switching is crash-free, location search pans the map to the selected result at zoom 13, and the sheet un-mounts at zero anchors."
    why_human: "Runtime crash absence (the TabBarView/DraggableScrollableSheet scrollController conflict this phase was designed to avoid), drag-feel, and camera-animation behavior can only be confirmed by running the app on a device/emulator, not by static analysis."
---

# Phase 20: Route Planner Views — Waypoint List, Elevation & Location Search Verification Report

**Phase Goal:** A user can inspect and manage the in-progress route through a persistent bottom sheet with two tabs — a route anchor list (delete, reorder) and a live elevation profile — and can pan the planner map to a searched location.
**Verified:** 2026-07-17
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth (ROADMAP.md Success Criterion) | Status | Evidence |
|---|-------|--------|----------|
| 1 | Once the route has ≥1 anchor, a docked bottom sheet is visible at peek height (draggable to expand), showing a "Route Anchors" tab with every anchor listed in route order | ✓ VERIFIED (static) | `route_planner_screen.dart:171-180` mounts `RouteAnchorSheet` only `if (state.anchors.isNotEmpty)` inside a `Stack` over the map. `route_anchor_sheet.dart:51-57` sets `initialChildSize: 0.14`, `minChildSize: 0.14` (never dismisses), `maxChildSize: 0.6`. `route_anchor_list_tab.dart:53-121` renders every anchor from `routeAnchorsProvider(...).anchors` in list order via `ReorderableListView.builder`, numbered `Anchor {index+1}`. Runtime rendering/visual confirmation deferred to human check #3. |
| 2 | From the route anchor list tab, a user can delete an anchor (immediate, no confirmation) or drag to reorder anchors, and the map and route update to match | ✓ VERIFIED (static) | `route_anchor_list_tab.dart:100-109` wires the trailing trash `IconButton.onPressed` directly to `deleteAnchor(anchor.id)` — no dialog. `route_anchor_provider.dart:354-395` `deleteAnchor` pushes undo first, collapses ≤2 touching segments into ≤1 new segment, re-resolves via Valhalla if auto-routing on. `reorderAnchors` (lines 403-442) reassigns anchor order and reuses unchanged segments by `segmentKey`, only re-resolving newly-adjacent pairs — unit-tested (see Behavioral Spot-Checks). `route_planner_screen.dart:90-96` listens to `routeAnchorsProvider` and pushes updated segments to the native `RouteSegmentLayer` on every non-identical segments emission. Visual/gesture confirmation deferred to human check #2. |
| 3 | A second tab ("Elevation") shows a live elevation profile built from a `Gpx` synthesized incrementally from the in-progress route and fetched from `/api/v1/valhalla/height` only while that tab is visible, updating as the route changes; with <2 anchors it shows an empty-state message instead | ✓ VERIFIED (static) | `planned_gpx_provider.dart` derives an ordered, pre-elevation `Gpx` by walking the anchor-id chain (not array order) from `routeAnchorsProvider`. `elevation_tab.dart:34-156`: `_isVisible` checks `_tabController!.index == 1 && !indexIsChanging`; `_onTabChanged`/`didChangeDependencies` gate `_scheduleFetch()` (500ms debounced) strictly on tab visibility; `_fetchHeights` re-checks `_isVisible` before and after the await and calls `POST /valhalla/height` via `apiProvider`; the merged `ele` values are held in a **local** `_eleMergedGpx`, never written back into `plannedGpxProvider` (D-10 preserved). `build()` returns `_ElevationEmptyState` when `gpx.allPoints.length < 2` with the exact D-13 copy, and skips the fetch entirely below 2 points. `elevation_profile.dart` was adapted with `trail: Trail?` and a `gpx.getTotals()` fallback for stats, confirmed guarded at both usage sites (line 164-174 stats header, line 235 chart waypoint overlay). Live rendering/network-timing confirmation deferred to human check #1. |
| 4 | A user can tap a magnifying-glass map control button (top-right, above auto-routing toggle) to open a dedicated location-search screen (locations only), and selecting a result pans/zooms the planner map to it (zoom 13) | ✓ VERIFIED (static) | `route_planner_screen.dart:267-277` places `_buildSearchButton()` as the first child of the top-right controls `Column`, above `_buildAutoRoutingToggle(...)`. `_openLocationSearch()` (lines 327-336) pushes `/location-search` (registered in `router_provider.dart:239`), awaits a `LocationSearchResult`, and calls `_mapController!.animateCamera(center: Geographic(lat, lon), zoom: 13, nativeDuration: 750ms)`. `location_search_screen.dart` reuses `globalSearchProvider` but sets `category = GlobalSearchCategory.locations` in `initState`, which gates `_search()`'s `searchTrails/searchLists/searchActors` booleans to `false` (confirmed in `global_search_provider.dart:141-152`) so only `/geocoding/search` fires — genuinely locations-only, not just UI-filtered. Result tap calls `context.pop(location)`, never `context.go('/map')` (`grep -c "context.go"` = 0). Visual/gesture confirmation deferred to human check #3. |

**Score:** 4/4 truths verified via static code inspection. All 4 have an associated human on-device check (deferred by the plans themselves, see Human Verification below) that has not yet been performed — this is why overall status is `human_needed`, not `passed`.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/provider/route_anchor_provider.dart` | `deleteAnchor`/`reorderAnchors` mutators | ✓ VERIFIED | Both methods present, undo-pushing, segment-recompute logic matches RESEARCH.md Patterns 2/3 exactly; unit-tested |
| `app/lib/util/gpx_util.dart` | `buildGpxFromPoints` helper | ✓ VERIFIED | Present, unit-tested (empty + ordered round-trip) |
| `app/lib/provider/planned_gpx_provider.dart` (+ `.g.dart`) | `plannedGpx` derived `@riverpod` provider | ✓ VERIFIED | Walks anchor-id chain via `segByBefore`, generated part file present, unit-tested |
| `app/lib/routes/location_search_screen.dart` | Locations-only search screen | ✓ VERIFIED | No trail/list/actor/category-chip surface; pops result; registered route |
| `app/lib/provider/router_provider.dart` | `/location-search` route | ✓ VERIFIED | `GoRoute(path: '/location-search', builder: ... LocationSearchScreen())` present |
| `app/lib/components/trail/elevation_profile.dart` | `trail: Trail?` + null-safe stats/chart | ✓ VERIFIED | Both usage sites guarded; all 5 existing non-null callers unaffected |
| `app/lib/components/route_planner/elevation_tab.dart` | Tab-gated debounced height fetch | ✓ VERIFIED | `TabController`-index-gated, 500ms debounce, local ele-merge, D-13 empty state |
| `app/lib/components/route_planner/route_anchor_list_tab.dart` | Reorderable/deletable anchor list | ✓ VERIFIED | Matches UI-SPEC row layout, immediate delete, optimistic reorder |
| `app/lib/components/route_planner/route_anchor_sheet.dart` | Tabbed `DraggableScrollableSheet` | ✓ VERIFIED | `minChildSize: 0.14` (no dismiss), scrollController wired to exactly one tab, separate `DraggableScrollableController` + `onVerticalDragUpdate` drives resize |
| `app/lib/routes/route_planner_screen.dart` | Search button + sheet host + camera hand-off | ✓ VERIFIED | Search button above auto-routing toggle; conditional `RouteAnchorSheet` mount; `animateCamera(..., zoom: 13, ...)` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `planned_gpx_provider.dart` | `routeAnchorsProvider` | `ref.watch(routeAnchorsProvider(travelProfile))` | WIRED | Confirmed at line 24 |
| `route_anchor_sheet.dart` | `RouteAnchorListTab` / `ElevationTab` | `TabBarView.children`, scrollController only to list tab | WIRED | Confirmed lines 133-137; `ElevationTab(travelProfile:)` has no `scrollController:` argument |
| `route_planner_screen.dart` | `/location-search` | `context.push<LocationSearchResult>('/location-search')` | WIRED | Confirmed line 328; route registered in `router_provider.dart:239` |
| `route_planner_screen.dart` | `RouteAnchorSheet` | conditional `Stack` child, `state.anchors.isNotEmpty` | WIRED | Confirmed lines 171-179 |
| `elevation_tab.dart` | `/api/v1/valhalla/height` | `api.post('/valhalla/height', ...)` gated on `_isVisible` | WIRED | Confirmed lines 96-101; gate confirmed at lines 63-77, 85-86, 102 |
| `location_search_screen.dart` | `globalSearchProvider` (locations-only) | `setCategory(GlobalSearchCategory.locations)` | WIRED | Confirmed line 36; verified this actually suppresses non-location API calls (`global_search_provider.dart:141-179`), not merely UI-filtered |
| `route_anchor_list_tab.dart` | `RouteAnchors.deleteAnchor`/`reorderAnchors` | direct notifier calls | WIRED | Confirmed lines 106-108, 131-133 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `route_anchor_list_tab.dart` | `anchors` | `ref.watch(routeAnchorsProvider(...)).anchors` | Yes — live in-memory route state, not static | ✓ FLOWING |
| `elevation_tab.dart` | `gpx` / `_eleMergedGpx` | `plannedGpxProvider` (derived from route state) + live `POST /valhalla/height` response | Yes — real Valhalla elevation values merged by index | ✓ FLOWING |
| `location_search_screen.dart` | `state.visibleLocations` | `globalSearchProvider` → `POST /geocoding/search` (Nominatim proxy) | Yes — real geocoding results | ✓ FLOWING |
| `route_anchor_sheet.dart` (TabBar/handle) | sheet size | `DraggableScrollableController` | N/A — UI chrome, not data-bound | ✓ FLOWING (n/a) |

No hollow props or hardcoded-empty data paths found in any of the 10 artifacts reviewed.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `deleteAnchor`/`reorderAnchors` unit contracts (collapse-on-delete, reuse-unchanged-segments-on-reorder) | `cd app && flutter test test/provider/route_anchor_provider_test.dart` | 21/21 relevant tests pass (incl. delete first/middle/last, reorder reuse + newly-adjacent) | ✓ PASS |
| `buildGpxFromPoints` + `plannedGpxProvider` traversal order | `cd app && flutter test test/util/gpx_util_test.dart test/provider/planned_gpx_provider_test.dart` | All pass (empty→empty, ordered round-trip, path-order-not-array-order traversal) | ✓ PASS |
| Static analysis of all 10 phase files | `cd app && flutter analyze <10 files>` | "No issues found!" | ✓ PASS |
| Whole-app analyze (regression check) | `cd app && flutter analyze` | 46 pre-existing informational/deprecation issues, 0 new errors from Phase 20 files, 1 pre-existing unrelated warning (`feed_item_test.dart` unused import, logged in STATE.md before this phase) | ✓ PASS |
| Live elevation chart rendering / tab-gated network timing | — | Not run (requires live app + network observation) | ? SKIP → human check #1 |
| Drag-reorder → map segment visual update; delete immediacy; Undo restore | — | Not run (requires device gesture + visual observation) | ? SKIP → human check #2 |
| Sheet peek/expand feel, tab-switch crash-freedom, search-to-pan camera animation | — | Not run (requires device/emulator) | ? SKIP → human check #3 |

### Probe Execution

Not applicable — no `scripts/*/tests/probe-*.sh` files exist for this phase, and none are referenced in the PLAN/SUMMARY files. Skipped.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|-------------|--------|----------|
| WAYP-04 | 20-01, 20-04 | User can delete a route anchor from the route anchor list tab | ✓ SATISFIED | `deleteAnchor` mutator + `route_anchor_list_tab.dart` trailing delete icon |
| WAYP-05 | 20-01, 20-04 | User can reorder route anchors via the route anchor list tab | ✓ SATISFIED | `reorderAnchors` mutator + `ReorderableListView.builder` |
| PLANUI-01 | 20-05 | Route anchor list + elevation profile as two tabs of one persistent bottom sheet | ✓ SATISFIED | `RouteAnchorSheet` + conditional mount in `route_planner_screen.dart` |
| PLANUI-02 | 20-01, 20-03 | Elevation profile built from incrementally-synthesized `Gpx`, fetched only while tab visible | ✓ SATISFIED | `plannedGpxProvider` + tab-gated `elevation_tab.dart` |
| PLANUI-03 | 20-02, 20-05 | Location-search screen (locations only), pans/zooms planner map to result | ✓ SATISFIED | `location_search_screen.dart` + `_openLocationSearch()` camera hand-off |

No orphaned requirements — REQUIREMENTS.md maps exactly these 5 IDs to Phase 20, and all 5 appear in at least one plan's `requirements` frontmatter field.

### Anti-Patterns Found

None. Scanned all 10 phase-touched files for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/empty-return/hardcoded-empty-data patterns — no matches (two `toDouble()` substring false-positives on a case-insensitive `TODO` grep, not real markers, confirmed by direct inspection).

### Scroll-Controller Conflict Check (Adversarial Focus Item #5)

Directly confirmed in `route_anchor_sheet.dart`:
- A single, sheet-owned `DraggableScrollableController _sheetController` (line 38-39) drives expand/collapse via `onVerticalDragUpdate` on the handle-bar `GestureDetector` (lines 80-90) — entirely decoupled from the `DraggableScrollableSheet.builder`'s `scrollController` parameter.
- The builder's `scrollController` (line 58) is passed into exactly one `TabBarView` child: `RouteAnchorListTab(scrollController: scrollController)` (line 135). `ElevationTab(travelProfile: widget.travelProfile)` (line 137) receives no `scrollController` argument at all.
- This matches RESEARCH.md Pattern 1's verified fix exactly (not merely claimed in the executor SUMMARY — read directly from source). `flutter analyze` reports no issues on this file, and no test/runtime evidence of the `"ScrollController attached to multiple scroll views"` crash was found — though actually exercising the tab switch on a live `TabBarView` (which preloads both pages) is exactly the kind of runtime behavior static analysis cannot fully rule out, hence it remains part of human check #3 rather than being marked fully closed by static reading alone.

### Human Verification Required

1. **Elevation tab live rendering + tab-gated fetch**
   **Test:** Open the Elevation tab on a route with ≥2 anchors; watch the profile render/update as anchors are added/moved. Reduce to <2 anchors and confirm the empty-state message appears. While on the Route Anchors tab, make route edits and confirm no `/valhalla/height` call fires (network inspector or breakpoint).
   **Expected:** Live chart updates only while Elevation tab is visible; empty-state text "Add at least 2 anchors to see the elevation profile." shows below 2 anchors; zero height calls while the other tab is active.
   **Why human:** Live chart rendering and precise network-call timing require a running app/device, not static source review.

2. **Route anchor list delete/reorder → map sync**
   **Test:** Add several anchors; long-press-drag to reorder them in the Route Anchors tab and confirm the map's route segments update to match; tap the trailing delete icon and confirm immediate removal with no dialog; tap app-bar Undo and confirm the deleted anchor returns.
   **Expected:** Map segments re-render in the new order; delete is instant, no confirmation UI; Undo restores state.
   **Why human:** Gesture-driven interaction and its visual effect on the native map layer require on-device confirmation.

3. **Sheet peek/expand, tab-switch stability, and search-to-pan**
   **Test:** Add an anchor and confirm the sheet docks at peek height; drag the handle to expand; switch between Route Anchors and Elevation tabs repeatedly and confirm no crash; tap the magnifying-glass control, search a location, select a result, and confirm the map pans/zooms to it at zoom 13; delete all anchors and confirm the sheet disappears.
   **Expected:** Smooth peek/expand, crash-free tab switching (the exact runtime scenario Pattern 1 was designed to avoid), successful camera pan to zoom 13, and sheet un-mount at zero anchors.
   **Why human:** Runtime crash-freedom, drag feel, and camera-animation behavior can only be confirmed by running the app.

### Gaps Summary

No gaps — every ROADMAP.md success criterion and REQUIREMENTS.md requirement for Phase 20 is backed by real, wired, non-stub code, confirmed by direct source reading (not SUMMARY.md claims) plus passing unit tests and clean `flutter analyze`. The three plans (20-03, 20-04, 20-05) that built this phase's user-facing surface each explicitly deferred a set of on-device checks to end-of-phase human verification (per the project's `human_verify_mode: end-of-phase` convention) rather than skipping them — those checks have not yet been performed, which is why this report's status is `human_needed` rather than `passed`. No code changes are required before proceeding; a human should run the three checks above (ideally in one sitting, since #3 subsumes most of #1/#2's non-network-timing aspects) and then flip this phase to closed.

---

*Verified: 2026-07-17*
*Verifier: Claude (gsd-verifier)*
