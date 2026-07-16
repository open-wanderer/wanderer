# Phase 20: Route Planner Views — Waypoint List, Elevation & Location Search - Context

**Gathered:** 2026-07-16
**Status:** Ready for planning

<domain>
## Phase Boundary

A user can inspect and manage the in-progress route through a persistent bottom sheet with two tabs — a route anchor list (delete, reorder) and a live elevation profile — and can pan the planner map to a searched location. This phase covers WAYP-04/05 and PLANUI-01/02/03.

</domain>

<decisions>
## Implementation Decisions

### Terminology (locked — applies to all downstream artifacts)
- **D-01:** This phase uses "route anchor" terminology throughout — code, provider/type names, and UI copy — consistent with Phase 19's D-01 (`RouteAnchor`, never "waypoint," to avoid confusion with the persisted `Waypoint` model). The sheet's list tab is the "route anchor list," not "waypoint list." `.planning/REQUIREMENTS.md` was amended: WAYP-04/05 and PLANUI-01/02 text now reads "route anchor."

### Sheet architecture (SCOPE CHANGE from original PRD wording)
- **D-02 (SCOPE CHANGE):** The route anchor list and elevation profile are **not** two separate views toggled via map control buttons (as originally worded in ROADMAP.md/REQUIREMENTS.md). Instead they are two tabs (TabBar) inside a **single persistent `DraggableScrollableSheet`**: "Route Anchors" tab and "Elevation" tab. No dedicated map control button opens this sheet. `.planning/ROADMAP.md` and `.planning/REQUIREMENTS.md` were amended to reflect this — PLANUI-01 and Phase 20's success criteria 1-3 now describe the tabbed-sheet mechanism instead of "toggled via map control buttons." Same user-visible capability (inspect route as list or elevation, mutually exclusive at a time), simpler mechanism.
- **D-03:** The sheet is **always docked** at a small peek height (handle bar + tab labels visible) as soon as the route has ≥1 anchor; hidden entirely on an empty route (0 anchors). User drags up to expand and interact with either tab; drags back down to peek. Follows the existing `DraggableScrollableSheet` pattern (`WaypointSheet`), but without the drag-to-fully-dismiss `onClose` behavior — it collapses to peek, not zero.
- **D-04:** The magnifying-glass location-search map control button sits in the top-right controls column, ordered **Search → Auto-routing toggle** (search above auto-routing). No third button is needed for list/elevation since D-02 replaced that with the tabbed sheet.

### Route anchor list tab — delete & reorder UX
- **D-05:** Delete uses a trailing delete icon button per row (not swipe-to-delete), to avoid gesture conflict with `ReorderableListView`'s long-press-anywhere-on-row drag start.
- **D-06:** Delete is immediate — no confirmation dialog or snackbar-undo. Phase 19's app-bar Undo/Redo is the safety net for a mistaken delete.
- **D-07:** No minimum-anchor guard — a user can delete anchors freely down to zero (the route/sheet simply becomes empty; D-03 already covers the sheet auto-hiding at 0 anchors).
- **D-08:** Reorder follows the existing `ReorderableListView.builder` pattern from `settings_categories_screen.dart` (local `_orderedIds` working-copy list seeded from provider state, `_reordering` guard during async persist, canonical `newIndex > oldIndex ? newIndex -= 1` index-shift adjustment, optimistic reorder with revert-on-error). `route_anchor_provider.dart` needs new `deleteAnchor(id)` and `reorderAnchors(newOrder)` methods (neither exists yet), each following the existing mutators' `_pushUndo()` → mutate `anchors`/`segments` → recompute affected segments pattern.

### Elevation profile — Gpx synthesis & live update
- **D-09:** The `Gpx` for the elevation profile is built **incrementally** as the route is edited — not synthesized fresh only when the elevation tab opens. This same synthesized `Gpx` will be reused for the Phase 21 handoff (HANDOFF-01, "synthesized GPX + named waypoints"), so building it once now avoids duplicating synthesis logic in Phase 21.
- **D-10:** The synthesized `Gpx` lives in a **derived provider** (e.g. `plannedGpxProvider`) that watches `routeAnchorsProvider` and recomputes the `Gpx` (points/segments, pre-elevation) whenever anchors/segments change. `RouteAnchorsState` itself stays focused on anchors/segments/undo-redo — no Gpx synthesis logic added to the core state class.
- **D-11:** Elevation values (`ele`) are fetched from `/api/v1/valhalla/height` **only while the Elevation tab is visible/open** — not on every route edit regardless of which tab is showing. While the tab is open, further route changes debounce the height refetch (~500ms, matching the existing debounce idiom in `global_search_provider.dart`).
- **D-12:** Route anchors do **not** appear as markers/overlay on the elevation profile chart (unlike the existing `ElevationProfile` widget's `Trail`-bound waypoint-icon overlay). The existing `ElevationProfile` widget (`app/lib/components/trail/elevation_profile.dart`) should be adapted with its `trail` param made optional — when `trail` is null, use `gpx.getTotals()` directly for the stats header and skip the waypoint-icon overlay entirely, rather than building a new lightweight widget from scratch.
- **D-13:** With fewer than 2 route anchors, the Elevation tab shows an empty-state message (reuse `ElevationProfile`'s existing `_EmptyState` widget/pattern) rather than an empty chart or a `/valhalla/height` call with insufficient points.

### Location search screen
- **D-14:** Full-screen push (`context.push`), mirroring the existing `GlobalSearchScreen` pattern exactly — same debounced search field + result-list UX, filtered to locations only (drop the `GlobalSearchCategory` chip row and the trails/lists/actors branches from `_search()`). On selection, pop back to `RoutePlannerScreen` with the result (not `context.go('/map')`, since the planner is a distinct already-open route) and animate the planner's own `MapController` camera directly.
- **D-15:** On result selection, animate the planner map camera to **zoom 13** (matching `GlobalSearchScreen`'s existing `/map` handoff behavior), not "keep current zoom."

### Claude's Discretion
- Exact `Gpx`-from-points construction helper (`gpx` package `Gpx(trks: [Trk(trksegs: [Trkseg(trkpts: ...)])])`) — no existing helper does this; implementer's discretion on where it lives (likely a new function in `gpx_util.dart`).
- Exact debounce duration tuning beyond the ~500ms reference point (D-11).
- Sheet peek height, tab icon choices, and exact `DraggableScrollableSheet` size fractions (`initialChildSize`/`minChildSize`/`maxChildSize`) — implementer's discretion, following `WaypointSheet`'s existing values as a starting point.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & roadmap (amended this session)
- `.planning/REQUIREMENTS.md` — WAYP-04/05 reworded to "route anchor"; PLANUI-01 marked SCOPE CHANGE (tabbed sheet, not toggle buttons) and reworded; PLANUI-02 reworded for incremental synthesis + tab-visible-only fetching.
- `.planning/ROADMAP.md` — Phase 20 goal, requirements-list intro line, and all 4 success criteria updated to describe the single tabbed sheet mechanism instead of "toggled via map control buttons"; explicit SCOPE CHANGE note added after the success criteria.

### Prior phase context
- `.planning/phases/19-route-planner-core-waypoint-editing-routing-engine/19-CONTEXT.md` — D-01 (route anchor terminology, never "waypoint"), D-06 (top-right map controls Column is the shared extension point — this phase's search button joins it, per D-04 above), D-10 (undo/redo in app bar, the safety net referenced in D-06 above).

No other external specs/ADRs — requirements fully captured in decisions above.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `app/lib/components/trail/elevation_profile.dart` (`ElevationProfile`): chart/gradient/smoothing/scrub logic (lines ~222-517) is generic and only needs a `Gpx`. Make `trail` optional per D-12; drop/skip the waypoint-icon overlay when `trail` is null; use `gpx.getTotals()` for the stats header. Has an existing `_EmptyState` widget to reuse for D-13.
- `app/lib/routes/global_search_screen.dart` (`GlobalSearchScreen`) + `app/lib/provider/search/global_search_provider.dart` (`globalSearchProvider`, `GlobalSearchNotifier._search()`): pattern to mirror for the location-only search screen (D-14) — 500ms debounced search field, `/geocoding/search` API call, `LocationSearchResult` model (`app/lib/models/global_search_models.dart:169-178`), `_LocationTile` row styling. Strip the category-chip row and non-location search branches.
- `app/lib/routes/settings_categories_screen.dart` (lines 244-327): the `ReorderableListView.builder` + optimistic-reorder + revert-on-error pattern to follow for D-08.
- `app/lib/components/trail/waypoint_sheet.dart` (`WaypointSheet`): `DraggableScrollableSheet` chrome pattern (rounded-top container, drag handle, shadow) — base for the new tabbed sheet, minus its full-dismiss `onClose` behavior (D-03).
- `app/lib/util/gpx_util.dart`: `costingForCategory`, `buildNavShape`, `GpxStats`/`Gpx.getTotals()`, `Gpx.allWaypoints`/`allPoints`, `Gpx.getBounds()`, `Gpx.distanceFromStartTo()` — all reusable; no existing "build Gpx from points" helper (Claude's Discretion above).
- `web/src/routes/api/v1/valhalla/height/+server.ts`: existing pass-through proxy to Valhalla's native `/height` endpoint — confirmed no backend changes needed. Request/response shape is Valhalla's own (`{shape: [{lat,lon}]}` → `{height: [...]}`); no existing app-side caller to copy from, and no sibling test file — verify exact shape against the Valhalla `VALHALLA_HEIGHT_URL` target before implementing the app-side call.
- `web/src/routes/api/v1/geocoding/search/+server.ts`: existing Nominatim search proxy (`fetchNominatim(event, "/search", ...)`, returns raw GeoJSON `FeatureCollection`) — reuse as-is for D-14, no backend changes needed.

### Established Patterns
- Riverpod 3.x + `riverpod_annotation` codegen throughout; `AsyncValue` listener closures must be explicitly typed (`AsyncValue<T>`) (Phase 19/prior lesson, still applies).
- Debounced search idiom: 500ms `Timer`/debounce in `global_search_provider.dart` — reuse the same idiom for D-11's height-refetch debounce.

### Integration Points
- `app/lib/provider/route_anchor_provider.dart` (`RouteAnchors` / `routeAnchorsProvider(travelProfile)`): needs new `deleteAnchor(id)` and `reorderAnchors(newOrder)` mutation methods (D-08). No `deleteAnchor`/`reorderAnchors` exists yet — current methods are `appendAnchor`, `dragAnchor`, `insertAnchorOnSegment`, `retrySegment`, `toggleAutoRouting`, `undo`, `redo`.
- `app/lib/routes/route_planner_screen.dart`: top-right controls `Positioned` Column (currently only `_buildAutoRoutingToggle`, lines ~250-310) gains the search button per D-04. The new tabbed sheet is a new widget added to this screen's build tree (not the controls Column — it's a bottom-docked sheet, D-03).
- New derived provider (D-10, e.g. `plannedGpxProvider`) watches `routeAnchorsProvider(travelProfile)` — this is also the artifact Phase 21's handoff (HANDOFF-01) will consume, so name/shape it with that reuse in mind.

</code_context>

<specifics>
## Specific Ideas

- The tabbed-sheet idea (D-02/D-03) came directly from the user during discussion, replacing the original two-map-control-buttons plan: "Maybe two tabs in the same draggablescrollable sheet is the better UX?" then "No map button. There is only one sheet with two tabs: 1. Route anchor list 2. elevation profile."
- The incremental-Gpx-synthesis idea (D-09) also came from the user: "Build the trail together with the route. It will be needed anyways when handing it over to the trail_create_screen in phase 21, so we might as well build it incrementally... Route anchors do not need to appear in the profile" (→ D-12).

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. The sheet-mechanism change (D-02) and incremental-Gpx-synthesis decision (D-09/D-10) are implementation-mechanism refinements of already-scoped requirements, not new capabilities, and were reconciled directly in REQUIREMENTS.md/ROADMAP.md rather than left as loose ideas.

</deferred>

---

*Phase: 20-route-planner-views-waypoint-list-elevation-location-search*
*Context gathered: 2026-07-16*
