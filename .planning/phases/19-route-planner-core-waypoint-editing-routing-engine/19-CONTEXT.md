# Phase 19: Route Planner Core — Waypoint Editing & Routing Engine - Context

**Gathered:** 2026-07-16
**Status:** Ready for planning

<domain>
## Phase Boundary

A user can build a route from scratch directly on the map — tapping to add route anchors, dragging to reposition them, inserting mid-segment — with an auto-routing toggle (Valhalla, fixed foot/bike profile set at entry) and undo/redo, all backed by a dedicated route-planner state provider. This phase covers WAYP-01/02/03 and ROUTE-01/02/04/05.

</domain>

<decisions>
## Implementation Decisions

### Terminology (locked — applies to all downstream artifacts)
- **D-01:** The in-progress route's tap points are called **"route anchors"**, never "waypoints," anywhere in code, UI copy, or docs for this phase. This is a deliberate distinction from the existing `Waypoint` model (`app/lib/models/waypoint.dart`), which belongs to a persisted `Trail` and is a different data type/concept. A new type (e.g. `RouteAnchor`) should back the in-progress route rather than reusing `Waypoint`.
- **D-02:** Route anchors are numbered in ascending order as the user adds them. Inserting a new anchor mid-route (WAYP-03) renumbers every anchor after the insertion point so the sequence stays contiguous and ascending.

### Waypoint (route-anchor) gesture disambiguation
- **D-03:** A tap on empty map space always adds a new route anchor (appended to the end of the route) — no explicit "add mode" toggle.
- **D-04:** Tap-on-marker vs. tap-on-segment vs. tap-on-empty-map is disambiguated by giving markers a larger invisible hit-radius (matching the existing 32px marker + 36px proximity-nudge pattern in `trail_layer.dart`), checked *before* segment hit-testing. Marker wins on overlap.
- **D-05:** Route-anchor drag reuses the existing `GestureDetector.onPanStart/Update/End` pattern from `TrailMarkerLayer` (`trail_layer.dart`) — proven to coexist with native map pan/zoom gestures. No live route preview during drag; the anchor shows at a straight temporary screen position while dragging, and connected segments re-resolve to the current routing mode only once the drag ends.

### Auto-routing toggle
- **D-06:** The auto-routing on/off toggle lives in the top-right map control buttons, matching the existing `TrailMap.controls` Column pattern (top-right corner) and consistent with Phase 20's planned waypoint-list/elevation toggle buttons in the same spot.
- **D-07 (SCOPE CHANGE):** There is **no in-planner travel-profile switch**. The foot/bike profile is set once via the Phase 21 entry hike/bike dialog (HANDOFF-03) and is fixed for the entire planning session. **ROUTE-03 was cut** as a result of this discussion. `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, and `.planning/PROJECT.md` were amended in this session: ROUTE-03 removed from Phase 19's requirement list, HANDOFF-03's wording updated to "fixed — no in-planner profile switch," and the cut item tracked as **PLANNER-07** in REQUIREMENTS.md's v2/deferred section. Phase 19's requirement list is now WAYP-01/02/03, ROUTE-01/02/04/05 (ROUTE-03 removed).

### Blocked segment & retry
- **D-08:** A blocked segment (ROUTE-05 — failed to auto-route) renders as a dashed, red/warning-colored line, visually distinct from both a normal Valhalla-routed segment and a straight-line (auto-routing-off) segment.
- **D-09:** Retry lives on the blocked segment itself — tapping a blocked segment retries auto-routing for that segment. This reuses the same tap-on-segment gesture as WAYP-03's insert-anchor interaction; on a *blocked* segment specifically, tap means retry instead of insert (mutually exclusive by segment state, not competing gestures).

### Undo/redo
- **D-10:** Undo/redo (ROUTE-04) are icon buttons in the screen's app bar — not grouped with the top-right map controls.
- **D-11:** Undo/redo buttons are disabled (grayed out, non-interactive) when their respective history direction is empty — standard undo/redo UX, not always-tappable no-ops.

### Claude's Discretion
None flagged — all gray areas resolved to explicit user decisions (mostly the recommended option, except the profile-switch scope cut).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & roadmap (amended this session)
- `.planning/REQUIREMENTS.md` — ROUTE-03 removed from v1 Requirements/Routing and from the Traceability table; HANDOFF-03 reworded; PLANNER-07 added to v2/deferred; Out of Scope table updated. Requirements coverage is now 15/15 mapped (was 16/16).
- `.planning/ROADMAP.md` — Phase 19 goal/requirements/success-criteria text updated to drop ROUTE-03 and the "profile switch" framing; Phase 21 success criterion 2 updated to say the profile is fixed for the session, not "still changeable afterward via the toggle from Phase 19."
- `.planning/PROJECT.md` — "Active" requirements bullet and "Context" target-features bullet updated to match; Out of Scope table gained a line for the cut mid-session profile switch.

### v1.5 research flags (from STATE.md, still open — validate during Phase 19 research/planning)
- `.planning/STATE.md` §Blockers/Concerns — `package:maplibre` 0.3.5's exact `MapGestures`/`MapOptions` pan/rotate-disable API surface is MEDIUM confidence; validate with a small spike early in Phase 19 before committing to the drag-vs-pan gesture-arena solution (this phase's D-04/D-05 assume the existing `TrailMarkerLayer` pattern already solves this coexistence, but it hasn't been spiked at Route-Planner interaction density).
- `.planning/STATE.md` §Blockers/Concerns — the generation-counter/CancelToken race-guard pattern for out-of-order Valhalla responses is MEDIUM confidence against this project's pinned `riverpod_annotation` 4.0.2; confirm the idiom during Phase 19 planning/execution.

No other external specs/ADRs — requirements fully captured in decisions above.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `app/lib/components/map/trail_layer.dart` (`TrailMarkerLayer`): the direct precedent for route-anchor markers — tappable + draggable `ml.Marker` widgets in a `WidgetLayer`, `GestureDetector.onPanStart/Update/End` drag pattern, `AnimatedScale` selection state, `_buildCircularMarker` pin styling. Route Planner's anchor layer should follow this shape, not reuse it directly (it's bound to `Trail`/`Waypoint`, not an in-progress unsaved route).
- `app/lib/components/base/trail_map.dart` (`TrailMap`): the native `MapLibreMap` host pattern — `onTap`/`onWaypointTap`/`onWaypointDragEnd` callback shape, `onEvent`/`MapEventClick` tap routing, `controls` top-right `Column` slot for map control buttons (compass etc. today; the auto-routing toggle and Phase 20's list/elevation toggles will join this slot).
- `app/lib/util/gpx_util.dart`: `costingForCategory` (foot/bike costing string derivation — needed for the fixed-at-entry profile), `buildNavShape` (point downsampling for Valhalla requests), `GpxMappingUtils.distanceFromStartTo` (along-track projection, relevant if route anchors need a `distanceFromStart`).
- `web/src/routes/api/v1/valhalla/route/+server.ts`: existing proxy endpoint for ROUTE-01's Valhalla routing calls — already exists, confirmed no backend changes needed.

### Established Patterns
- Native map screens hold `ml.MapController` as a nullable field set from `onMapCreated`, buffer a `_pendingStyle`/style-loaded event that can arrive before `onMapCreated` fires (race condition hit twice already in Phase 16/17).
- Riverpod 3.x + `riverpod_annotation` codegen throughout; `AsyncValue` listener closures must be explicitly typed (`AsyncValue<T>`) or extension methods like `.isLoading` resolve to `dynamic` and throw at runtime (Phase quick-260712-pac lesson).
- Camera "instant" moves use `Duration(milliseconds: 1)`, never `Duration.zero` (crashes Android native binding).

### Integration Points
- Top-right `controls` slot on the map host widget — auto-routing toggle joins here (D-06); Phase 20's waypoint-list/elevation-profile toggle buttons are expected in the same slot.
- App bar — undo/redo buttons (D-10), separate from the map controls slot.
- `/api/v1/valhalla/route` — existing SvelteKit proxy, POST with shape array + costing profile.

</code_context>

<specifics>
## Specific Ideas

- "Route anchor" is the user's own chosen term, specifically to avoid confusion with the existing `Waypoint` model tied to persisted trails. This should propagate into type names, provider names, and file names in Phase 19's implementation (e.g. prefer `RouteAnchor`/`route_anchor_provider.dart` over anything with "waypoint" in the name), not just UI copy.
- Ascending-order numbering with renumber-on-insert (D-02) is a functional requirement on the state model, not just a display detail — the planner state must treat anchor order as the source of truth and recompute displayed numbers whenever the list mutates (add/insert/delete/reorder).

</specifics>

<deferred>
## Deferred Ideas

- **PLANNER-07** (new, added to REQUIREMENTS.md v2/deferred): mid-session travel-profile switching, cut from ROUTE-03 during this discussion. Profile is fixed at entry via HANDOFF-03 instead.

None — discussion stayed within phase scope beyond the ROUTE-03 cut, which was formally reconciled in REQUIREMENTS.md/ROADMAP.md/PROJECT.md rather than left as a loose idea.

</deferred>

---

*Phase: 19-route-planner-core-waypoint-editing-routing-engine*
*Context gathered: 2026-07-16*
