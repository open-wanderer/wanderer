# Requirements: Wanderer Trail Navigation — v1.5 Route Planner

**Defined:** 2026-07-16
**Core Value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.

## v1 Requirements

Requirements for the v1.5 Route Planner milestone. Each maps to a roadmap phase.

### Waypoint Editing

- [x] **WAYP-01**: User can tap the map to add a waypoint to the in-progress route
- [x] **WAYP-02**: User can drag an existing waypoint to reposition it, with connected segments re-resolving
- [x] **WAYP-03**: User can tap an existing route segment to insert a new waypoint between its endpoints
- [x] **WAYP-04**: User can delete a route anchor from the route anchor list tab
- [x] **WAYP-05**: User can reorder route anchors via the route anchor list tab

### Routing

- [x] **ROUTE-01**: User can toggle auto-routing on (Valhalla-routed segments via `/api/v1/valhalla/route`, foot/bike profile only) or off (straight-line segments between waypoints)
- [x] **ROUTE-02**: Toggling auto-routing on re-resolves all existing segments via Valhalla; toggling off leaves existing segments untouched and only affects segments created afterward
- [x] **ROUTE-04**: User can undo/redo waypoint add/move/insert/delete/reorder actions (in-memory only, cleared on exit)
- [x] **ROUTE-05**: When a segment fails to auto-route (unreachable backend, no route found), that segment is blocked and a retry action is surfaced — it never silently falls back to a straight line while auto-routing is on

### Planning UI

- [ ] **PLANUI-01** *(SCOPE CHANGE — see 20-CONTEXT.md)*: A route anchor list and a live elevation profile are available as two tabs of a single persistent bottom sheet (docked at peek height once the route has ≥1 anchor, draggable to expand) — not separate views toggled via map control buttons
- [x] **PLANUI-02**: The elevation profile is built from a `Gpx` synthesized incrementally from the in-progress route (via `/api/v1/valhalla/height` for elevation, fetched only while the elevation tab is visible) and updates live as the route changes
- [ ] **PLANUI-03**: User can open a dedicated location-search screen (magnifying-glass map control button) that searches locations only (not trails/lists/accounts) and pans/zooms the planner map to the selected result

### Handoff

- [ ] **HANDOFF-01**: User can finish planning and hand off the route as a draft Trail (synthesized GPX + named waypoints) to the existing trail create/edit screen
- [ ] **HANDOFF-02**: The Route Planner is reachable from a new entry point in the trail-source-select flow, alongside the existing "import trail file" option
- [ ] **HANDOFF-03**: Tapping "Open trail planner" shows a hike/bike selection dialog before the planner screen opens; the selection sets the Route Planner's initial travel profile for the whole session (fixed — no in-planner profile switch)

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Route Planner

- **PLANNER-01**: Car/driving costing profile for auto-routing
- **PLANNER-02**: Editing an existing trail's route (not just from-scratch planning)
- **PLANNER-03**: Per-segment travel profiles
- **PLANNER-04**: Offline caching of in-progress route plans
- **PLANNER-05**: Confirm-placement drag affordance (tap-drop-adjust-confirm), if mis-drops prove common
- **PLANNER-06**: Offline-aware graceful degradation banner for auto-routing
- **PLANNER-07**: Mid-session travel profile switch (cut from v1.5 as ROUTE-03; profile is now fixed at entry via HANDOFF-03)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Car/driving costing profile | `costingForCategory` only maps foot/bike today; out of scope for v1.5, tracked as PLANNER-01 |
| Editing an existing trail's route | v1.5 is from-scratch planning only |
| Per-segment travel profiles | One profile applies to the whole route |
| Offline route caching for in-progress plans | Not needed for v1.5 |
| Route optimization / automatic waypoint reordering | Conflicts with user-intended waypoint ordering |
| Automatic loop generation, multi-day touring | Not requested; adds complexity disproportionate to a from-scratch mobile planner |
| Reusing `GlobalSearchScreen` for location search | It also returns trails/lists/accounts; a dedicated location-only search screen is required instead |
| Any Go backend or SvelteKit changes | `/api/v1/valhalla/route` and `/api/v1/valhalla/height` already exist and require no changes (confirmed via research) |
| Switching travel profile after entry (ROUTE-03, cut during Phase 19 discussion) | Profile is set once via the hike/bike dialog at entry (HANDOFF-03) and fixed for the planning session; tracked as PLANNER-07 |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| WAYP-01 | Phase 19 | Complete |
| WAYP-02 | Phase 19 | Complete |
| WAYP-03 | Phase 19 | Complete |
| WAYP-04 | Phase 20 | Complete |
| WAYP-05 | Phase 20 | Complete |
| ROUTE-01 | Phase 19 | Complete |
| ROUTE-02 | Phase 19 | Complete |
| ROUTE-04 | Phase 19 | Complete |
| ROUTE-05 | Phase 19 | Complete |
| PLANUI-01 | Phase 20 | Pending |
| PLANUI-02 | Phase 20 | Complete |
| PLANUI-03 | Phase 20 | Pending |
| HANDOFF-01 | Phase 21 | Pending |
| HANDOFF-02 | Phase 21 | Pending |
| HANDOFF-03 | Phase 21 | Pending |

**Coverage:**

- v1 requirements: 15 total
- Mapped to phases: 15
- Unmapped: 0 ✓

---
*Requirements defined: 2026-07-16*
*Last updated: 2026-07-16 — ROUTE-03 cut during Phase 19 discussion (profile fixed at entry, tracked as PLANNER-07); requirements now 15/15 mapped*
