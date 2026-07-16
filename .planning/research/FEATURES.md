# Feature Research

**Domain:** Mobile route-planning tool for hiking/cycling (interactive map-based route builder)
**Researched:** 2026-07-16
**Milestone:** v1.5 Route Planner
**Confidence:** MEDIUM-HIGH (verified against Komoot, Strava, gpx.studio, OsmAnd documentation + Wanderer's own maplibre/Valhalla/trail-save infrastructure)

## Feature Landscape

### Table Stakes (Users Expect These)

Anyone who has used Komoot, Strava's Route Builder, gpx.studio, or OsmAnd's route planner will expect these. Missing them makes the planner feel broken, not "MVP."

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Tap-to-add waypoint (appends to route end) | Universal across Komoot, Strava, OsmAnd, gpx.studio — single tap on empty map drops the next point | LOW | Wanderer already has tap-handling patterns on `wanderer_map.dart`/`search_map.dart`; new part is turning a tap into a route point + re-render |
| Drag existing waypoint to reposition | Komoot: "existing waypoints can be moved around via drag and drop"; Strava: "drag and drop existing waypoints to adjust the route" | MEDIUM-HIGH | `maplibre` 0.3.5 has no native draggable-annotation API. Verified fix: render waypoints via `WidgetLayer`/`Marker` (`allowInteraction: true`) + `GestureDetector`/`PanUpdate`, convert the resulting screen offset back to a coordinate with `MapController.toLngLat(Offset)` (confirmed present in `maplibre_platform_interface` 0.3.5 source). This is the core new interaction to build |
| Insert waypoint mid-segment (tap/drag the line itself) | Komoot: "drag the line anywhere to create a new waypoint"; Strava: "add new ones by clicking on the line between two points" | MEDIUM-HIGH | Needs hit-testing against the rendered route polyline with a touch-friendly tolerance band (a 2–3px line is not tappable with a finger — pad the hit area, don't rely on exact pixel intersection), then split the segment at the projected point |
| Delete a waypoint | Implicit in every tool listed; also explicit in Wanderer's own v1.5 scope | LOW | Standard list/marker delete action; must also re-resolve the routed segment(s) that touched the deleted point |
| Reorder waypoints | Komoot's waypoint list supports drag-handle reordering; OsmAnd markers list is order-driven | LOW-MEDIUM | Cheaper and more mobile-friendly as a **list drag-handle** (`ReorderableListView`, already used for category priority per Key Decisions) than as free-form map drag — matches Wanderer's own precedent of avoiding pointer-drag-on-map for ordering |
| Undo/redo | gpx.studio ships explicit undo/redo buttons; expected in any "editor" mental model once users start making mistakes with a finger | MEDIUM | No existing undo/redo infra anywhere in the app (settings screens are single-state auto-save). Needs a command/snapshot stack scoped to the planning session only (not persisted) |
| Snap-to-path ("routed") as default, freehand as an explicit opt-out | Komoot: "by default the planner locks onto roads and trails"; Strava's "Manual Mode" is the opt-out; gpx.studio's "Routing mode vs Off-road mode" | MEDIUM | Directly maps to Wanderer's planned Valhalla toggle. Re-resolving on toggle/profile change is the correct behavior per all three tools — matches PROJECT.md's stated design |
| Live distance / elevation feedback while building | Strava: "real-time updates on distance, elevation gain, and estimated moving time" as you place points | LOW-MEDIUM | Wanderer already has `GpxMappingUtils.getTotals()` (distance/duration/elevation gain+loss) — reusable as-is against a synthesized in-memory `Gpx`, no new stats math needed |
| Live elevation profile chart | Strava and Komoot both surface an elevation graph during planning, not just after saving | LOW (reuse) | `ElevationProfile` widget (`app/lib/components/trail/elevation_profile.dart`) already renders a scrubbing chart from a `Trail` + `Gpx` — feed it the in-progress synthesized `Gpx` and it works largely unmodified |
| Waypoint list view | Komoot's "See waypoints list"; OsmAnd's marker list | LOW | Straightforward list backed by the same route-point array driving the map |
| Search-to-pan to a start location | Every planning tool (incl. desktop ones) lets you find a place before you start dropping points | LOW (already built) | Wanderer's `GlobalSearchScreen`/`global_search_provider` already does this — reuse, don't rebuild |
| Hand off finished route into a save/edit flow | Every consumer tool eventually exports/saves the route (GPX export, "Save route", etc.) | MEDIUM | Wanderer already has the exact mechanism: `pendingImportedTrail` global + `context.push('/trail/create/edit', extra: trail)`, used today by GPX import (`trail_import_util.dart`). Net-new work is *synthesizing* a `Trail`+`Gpx` from planner state (waypoints + routed/straight segments) — no equivalent synthesis code exists yet, must be written |

### Differentiators (Competitive Advantage)

Not required for a defensible v1, but where a mobile hiking-focused planner can distinguish itself. None of these are in the v1.5 Active scope — listed so the roadmap can consciously defer them rather than accidentally build a sliver of each.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Foot/bike-only profile switch mid-plan with instant re-routing | Matches Wanderer's actual audience (hikers/cyclists) better than Strava/Komoot's broader profile lists | LOW (already scoped) | Already in Active scope — this is the "differentiator that's actually in v1", not a stretch feature |
| Trail-category-aware waypoint icons on the elevation profile & map (reusing existing waypoint icon system) | Consistent visual language with the rest of the app (trail detail, navigation) rather than a generic pin | LOW | `ElevationProfile` already draws category icons for a saved trail's waypoints — extending this to in-progress planning is cheap and reinforces brand consistency ("Terrain Log" per DESIGN.md) |
| Offline-aware graceful degradation (auto-routing unavailable → falls back to freehand with a clear banner, rather than a hard error) | Hikers plan routes at trailheads with poor signal; Komoot/Strava assume near-constant connectivity | MEDIUM | Reuses the existing Dio-try-catch-as-offline-gate pattern from navigation (Key Decisions) rather than a new connectivity check |
| Single-thumb "confirm placement" step after drag (tap-drop-adjust-confirm rather than pure live-drag) | Desktop tools assume mouse precision; a confirm step reduces mis-placed waypoints from "fat finger" drag on a phone | LOW-MEDIUM | Not shown in any competitor (they're all mouse-first) — a genuine mobile-only affordance worth prototyping, not copying |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|------------------|-------------|
| Route optimization / auto-reorder waypoints (TSP-style "optimize my stops") | Some desktop route builders (Strava web) offer it; feels like a natural "smart" addition | Optimization solves a delivery/errand-routing problem, not a hiking-route problem — hikers place waypoints in the order they intend to walk them; auto-reordering would silently rewrite user intent. High algorithmic complexity for negative value here | Don't build it. If ordering is wrong, the user reorders manually via the list |
| Automatic loop/route generation ("suggest a 10km loop from here") | Komoot/Strava "Discover" features generate routes from a starting point + distance/heading | Requires a generation service (heuristic loop-finding over the road/trail graph), far beyond Valhalla's point-to-point routing; entirely different backend capability | Out of scope; if wanted later, it's a distinct milestone requiring new backend infra, not an extension of this planner |
| Multi-day / touring route planning (overnight stops, lodging) | Komoot's Tour Planner supports multi-day cycling tours | Large scope: day-splitting logic, lodging POIs, per-day stats — none of it maps to Wanderer's single-`Trail` model | Not applicable to Wanderer's data model; explicitly out of scope |
| Per-segment travel profile (walk this bit, bike that bit) | Desktop tools with mixed-mode trips sometimes support it | Already explicitly ruled out in PROJECT.md ("Per-segment travel profiles... a single profile applies to the whole route") — flagging here because it's the single most common escalation request once users see foot/bike toggle exists | Single profile for the whole route; if truly needed later, is its own milestone |
| Free-form vertex/curve smoothing tools (simplify, smooth spline, snap-to-nearest-track) | gpx.studio (desktop-first) offers these as power-user GPX cleanup tools | Desktop-precision editing operations assume a mouse and a large canvas; on a phone they're fiddly to a degree that actively hurts usability, and they duplicate what auto-routing already does | Auto-routing (Valhalla snap) already produces clean paths; freehand segments are intentionally freehand — don't add a "clean this up" tool on top |
| Real-time collaborative editing of a single route (two people planning together live) | Sounds appealing for group trip planning | No sync/conflict-resolution infra exists anywhere in the app; this is a distributed-systems problem bolted onto what's currently a local, single-session editor | Out of scope; sharing happens after save via existing trail-sharing/federation features, not during planning |
| Desktop-style always-visible side-by-side map + waypoint-list + elevation-chart panels | It's how Strava/Komoot/gpx.studio look on the web | Screen real estate on a phone can't fit three simultaneous panels without shrinking the map to uselessness | Mutually-exclusive bottom-sheet toggle between waypoint list and elevation profile (already the planned v1.5 design) — matches how Strava's *own* mobile app collapses these into a single toggled panel, unlike its web builder |
| Offline caching / resumable drafts of an in-progress plan | Users may expect "my draft survives if I background the app or lose signal," matching offline navigation | Explicitly out of scope in PROJECT.md; auto-routing calls need network anyway, so a fully offline planning session degrades to freehand-only regardless of caching | Freehand-only planning still works offline (no network call needed for straight lines); auto-routing simply becomes unavailable until reconnected, surfaced via a banner, not silently cached |

## Feature Dependencies

```
Tap-to-add waypoint
    └──requires──> Editable route-point array (new state, no existing analog)
                       └──requires──> Interactive map layer (WidgetLayer + Marker, allowInteraction:true)

Drag-to-reposition waypoint
    └──requires──> Interactive map layer (WidgetLayer + Marker)
                       └──requires──> MapController.toLngLat(Offset) for screen→coordinate conversion

Insert-mid-segment
    └──requires──> Rendered route polyline with touch-tolerant hit-testing
    └──requires──> Editable route-point array

Auto-routing toggle (Valhalla)
    └──requires──> POST /api/v1/valhalla/navigate (existing endpoint, reused for shape resolution)
    └──requires──> costingForCategory() (existing, already foot/bike-only — matches v1.5 scope)

Live distance/elevation feedback ──uses──> GpxMappingUtils.getTotals() (existing)
Live elevation profile view ──uses──> ElevationProfile widget (existing, needs synthesized Gpx input)

Undo/redo
    └──requires──> New command/snapshot stack (no existing analog anywhere in app)
    └──enhances──> All point-mutating actions (add/drag/insert/delete/reorder)

Handoff to trail_create_screen
    └──requires──> Route → Trail/Gpx synthesis (new code)
    └──requires──> pendingImportedTrail + context.push('/trail/create/edit', extra: trail) (existing pattern from GPX import)

Search-to-focus panning ──uses──> GlobalSearchScreen / global_search_provider (existing, no new work)

Route optimization (anti-feature) ──conflicts──> User-intended waypoint ordering
Multi-day touring (anti-feature) ──conflicts──> Single-Trail data model
```

### Dependency Notes

- **Drag-to-reposition requires the interactive map layer, not the reverse:** build the `WidgetLayer`/`Marker` + gesture-handling scaffold first; tap-to-add, drag, and insert-mid-segment all sit on top of the same editable point array and the same interactive layer, so this is naturally the first phase's foundation, not a late add-on.
- **Auto-routing toggle requires no new backend work:** `/api/v1/valhalla/navigate` and `costingForCategory()` already exist and already restrict to `pedestrian`/`bicycle` — this lines up exactly with PROJECT.md's "foot/bike only" constraint. The only new work is calling it per-segment (or for the whole waypoint chain) during planning and discarding the maneuver data (the planner only needs `shape`, not turn-by-turn instructions).
- **Live elevation profile enhances, doesn't require, the routed/straight-line distinction:** it needs *a* `Gpx`, and works the same whether the underlying segment is Valhalla-routed or a straight line — synthesize the `Gpx` from whatever the current route-point array + resolved segments are at any given moment.
- **Undo/redo has no existing analog to build on** — every other stateful editor in the app (settings screens) is auto-save/single-state, not multi-step. This is the one piece of "table stakes" with the least code reuse and should be scoped generously in complexity estimates.
- **Handoff requires new Trail/Gpx synthesis code** — the existing `pendingImportedTrail` handoff mechanism (from `trail_import_util.dart`) only ever received an already-server-converted `Trail`. The route planner is the first caller that must build a `Trail` (with `expand.gpx` and raw `gpxData`) entirely client-side from a list of route points, using the `gpx` package's own model classes (`Gpx`/`Trk`/`Trkseg`/`Wpt`).
- **Route optimization conflicts with user-intended ordering:** any "smart reorder" feature is fundamentally incompatible with hikers placing waypoints in the order they intend to walk — this is why it's an anti-feature, not a deferred differentiator.

## MVP Definition

### Launch With (v1 — matches PROJECT.md's v1.5 Active scope)

- [ ] Tap-to-add / drag / insert-mid-route / delete / reorder waypoints — the entire editing surface; without all five, the tool feels half-built (reorder and delete are the two most likely to be under-scoped)
- [ ] Undo/redo — mobile drag precision is worse than desktop mouse precision; users *will* mis-place waypoints, and without undo the only recovery is delete-and-redo, which is punishing
- [ ] Auto-routing toggle (Valhalla, foot/bike) vs straight-line, re-resolved on toggle/profile change — this is the single feature every comparable tool treats as core, not optional
- [ ] Live distance/elevation stats + elevation profile view (mutually exclusive with waypoint list) — validates the core value ("build a route and see what you're getting") before the user commits to saving
- [ ] Search-to-focus panning — needed just to get to a starting area; already built, low cost to include
- [ ] Handoff to trail_create_screen as a draft Trail — without this the planner is a dead end with no output

### Add After Validation (v1.x)

- [ ] Offline-aware graceful degradation banner for auto-routing (fall back to freehand automatically when Valhalla is unreachable, matching the existing navigation-offline UX pattern) — add once real-world trailhead connectivity issues surface in usage
- [ ] Confirm-placement affordance for drag (tap-drop-adjust-confirm instead of live-drag-only) — add if user testing shows mis-drops are common with pure live-drag

### Future Consideration (v2+)

- [ ] Per-segment travel profiles — explicitly deferred in PROJECT.md; requires rethinking the single-profile Trail model
- [ ] Automatic loop/route generation from a starting point — requires new backend route-generation capability beyond Valhalla point-to-point routing
- [ ] Multi-day touring routes — requires a data model beyond the single `Trail`
- [ ] Editing an *existing* trail's route in the planner (as opposed to from-scratch only) — explicitly deferred in PROJECT.md

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Tap-to-add waypoint | HIGH | LOW | P1 |
| Drag-to-reposition waypoint | HIGH | MEDIUM-HIGH | P1 |
| Insert-mid-segment | HIGH | MEDIUM-HIGH | P1 |
| Delete / reorder waypoint | HIGH | LOW-MEDIUM | P1 |
| Undo/redo | HIGH | MEDIUM | P1 |
| Auto-routing toggle (Valhalla) | HIGH | MEDIUM | P1 |
| Live distance/elevation stats | MEDIUM-HIGH | LOW | P1 |
| Live elevation profile view | MEDIUM-HIGH | LOW (reuse) | P1 |
| Search-to-focus panning | MEDIUM | LOW (already built) | P1 |
| Handoff to trail_create_screen | HIGH | MEDIUM | P1 |
| Offline-aware auto-routing fallback banner | MEDIUM | LOW-MEDIUM | P2 |
| Confirm-placement drag affordance | MEDIUM | LOW-MEDIUM | P2 |
| Per-segment profiles | LOW-MEDIUM | HIGH | P3 |
| Auto loop generation | MEDIUM | HIGH | P3 |
| Route optimization / auto-reorder | LOW | HIGH | Do not build |

**Priority key:**
- P1: Must have for launch
- P2: Should have, add when possible
- P3: Nice to have, future consideration

## Competitor Feature Analysis

| Feature | Komoot (mobile) | Strava (mobile) | gpx.studio (web-only) | Wanderer's Approach |
|---------|------------------|------------------|------------------------|----------------------|
| Add waypoint | Tap map, choose "add to end" or "as waypoint," confirm | Manual Mode → tap map to drop a point | Select "Add Waypoint" tool, then tap map | Tap map appends to route end — simpler single-mode interaction, no mode confirmation dialog |
| Reposition waypoint | Drag-and-drop on map; Android requires "unfold" + tap-hold two lines | Drag and drop existing waypoints | Click + drag on map | Drag via `WidgetLayer`/`Marker` + `toLngLat` — same end result, native-GL implementation |
| Insert mid-segment | Drag the route line to create a new waypoint | Click/tap the line between two points | Click on the track line | Tap/drag on rendered polyline with padded hit-tolerance for touch |
| Snap-to-path vs freehand | Default "locks onto roads and trails"; no explicit freehand toggle documented for mobile | Explicit "Manual Mode" toggle for direct/freehand lines | Explicit "Routing mode" vs "Off-road mode" toggle | Explicit toggle (matches Strava/gpx.studio pattern) with foot/bike-only profiles, re-resolves on change |
| Elevation profile | Shown during planning | Toggleable graph in lower-right corner, real-time updates | Full graph, point-linked | Reuses existing `ElevationProfile` widget already used on trail detail — visual/interaction consistency with the rest of the app |
| Undo/redo | Not documented as explicit buttons (relies on manual re-edit) | Not documented as explicit buttons | Explicit undo/redo buttons | Explicit, always-visible toolbar buttons — closer to gpx.studio's approach than Komoot/Strava's mobile apps, because mobile drag precision needs a safety net more than desktop does |
| List vs map-panel layout | Bottom sheet / "More" menu for waypoint list | Single mobile view, no side panel | Side-by-side panels (desktop assumes wide screen) | Mutually-exclusive bottom-sheet toggle (list *or* elevation, never both) — deliberately narrower than gpx.studio's desktop layout, closer to Strava mobile's single-panel collapse |

## Mobile-Specific UX Constraints

- **Touch target sizing:** waypoint markers and the polyline's tappable hit-area both need generous touch tolerance (44×44pt minimum per platform guidance) — a marker or line rendered at its visual pixel size is not reliably tappable with a finger ("fat finger" problem); this affects insert-mid-segment most, since a thin route line is the smallest target in the whole planner.
- **Tap vs pan disambiguation:** the map already handles single-finger pan/zoom; tap-to-add must not fire on every incidental tap during panning. Follow Komoot/OsmAnd's pattern of a deliberate, discrete tap on otherwise-empty map surface, and keep pan gestures uninterrupted elsewhere (the existing `WidgetLayer` requires `TranslucentPointer` when `allowInteraction: false ` precisely to avoid blocking map panning — the inverse needs equal care once interaction is turned on for markers).
- **Drag precision:** phones lack a mouse's pixel-precise pointer; a live-drag-only interaction risks frequent mis-placement, especially outdoors in bright sun or with gloves. A tap-drop-then-adjust flow (differentiator, above) mitigates this; at minimum, undo must be cheap and always visible (toolbar button, not buried in a menu) since it is the primary recovery path from drag error.
- **One-handed / thumb-reach layout:** primary controls — add-point affordance, undo/redo, auto-route toggle, waypoint-list/elevation-profile toggle — should sit within thumb reach at the bottom of the screen, consistent with Wanderer's existing bottom-sheet and bottom-nav patterns, not in top app-bar corners that require a hand-shift to reach one-handed on a trail.
- **Small screen real estate:** a phone cannot show map + waypoint list + elevation profile simultaneously without the map becoming too small to use for placing points accurately — the planned mutually-exclusive bottom-sheet toggle (list *or* profile) is the correct mobile adaptation of what Strava/Komoot/gpx.studio show as separate panels on their web builders.
- **Field conditions:** hikers planning routes at a trailhead may have poor connectivity (auto-routing calls can fail or lag) and may be planning one-handed with a pack on — favor large, forgiving hit targets and an always-available freehand fallback over any interaction that assumes ideal conditions.

## Sources

- [Plan routes in the app – komoot](https://support.komoot.com/hc/en-us/articles/10206792140826-Plan-routes-in-the-app) — MEDIUM confidence (official support docs)
- [Route Planner Tips and Tricks – komoot](https://www.komoot.com/help/routeplanner) — MEDIUM confidence
- [Advanced route planning – komoot](https://support.komoot.com/hc/en-us/articles/10268757747738-Advanced-route-planning) — MEDIUM confidence
- [Planning Tours on iOS – komoot](https://support.komoot.com/hc/en-us/articles/4403138423066-Planning-Tours-on-iOS) — MEDIUM confidence
- [Planning Tours on Android – komoot](https://support.komoot.com/hc/en-us/articles/4402920434458-Planning-Tours-on-Android) — MEDIUM confidence
- [Creating Routes on Mobile – Strava Help Center](https://support.strava.com/hc/en-us/articles/18001474720397-Creating-Routes-on-Mobile) — MEDIUM confidence (official support docs)
- [Strava rolls out new finger-dragging route creation feature — DC Rainmaker](https://www.dcrainmaker.com/2019/02/dragging-creation-feature.html) — MEDIUM confidence (independent hands-on review, corroborates official docs)
- [gpx.studio — help | Route planning and editing](https://gpx.studio/help/toolbar/routing) — MEDIUM confidence (official docs)
- [gpx.studio — help | Edit actions](https://gpx.studio/help/menu/edit) — MEDIUM confidence (official docs, confirms undo/redo)
- [OsmAnd — Plan a Route](https://osmand.net/docs/user/plan-route/create-route/) — MEDIUM confidence (official docs)
- [OsmAnd — Map Markers](https://osmand.net/blog/map-markers/) — MEDIUM confidence
- [Map UI Design: Best Practices, Tools & Real-World Examples — Eleken](https://www.eleken.co/blog-posts/map-ui-design) — LOW-MEDIUM confidence (secondary UX-blog source, used only for general touch-target/one-handed guidance, not product-specific claims)
- [One-Handed Mobile UX: Best Practices — Upslide Design Studio](https://upslidedesignstudio.com/blogs/one-handed-mobile-ux-design-best-practices-for-better-mobile-apps) — LOW-MEDIUM confidence (general UX guidance, not hiking-specific)
- Wanderer codebase (HIGH confidence, primary source): `app/lib/util/gpx_util.dart`, `app/lib/util/navigation_launch_util.dart`, `app/lib/util/trail_import_util.dart`, `app/lib/components/trail/elevation_profile.dart`, `app/lib/provider/router_provider.dart`, and `maplibre`/`maplibre_platform_interface` 0.3.5 package source (`widget_layer.dart`, `map_controller.dart`) confirming `WidgetLayer`/`Marker`/`MapController.toLngLat(Offset)` are the available primitives for drag interactions

---
*Feature research for: Mobile route-planning tool (Wanderer v1.5 Route Planner)*
*Researched: 2026-07-16*
