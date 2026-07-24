# Phase 20: Route Planner Views — Waypoint List, Elevation & Location Search - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-16
**Phase:** 20-route-planner-views-waypoint-list-elevation-location-search
**Areas discussed:** Terminology (waypoint vs route anchor), Waypoint/anchor list sheet — delete & reorder UX, Elevation profile — reuse & live-update behavior, Location search screen — presentation & pan/zoom behavior

---

## Terminology: "waypoint" vs "route anchor"

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — "route anchor" everywhere | Sheet is RouteAnchorListSheet, list items are route anchors, delete/reorder acts on route anchors. Matches Phase 19's locked terminology. | ✓ |
| Keep "waypoint" for this phase's UI/code | Use "waypoint" literally as written in ROADMAP/REQUIREMENTS. | |

**User's choice:** Yes — "route anchor" everywhere (Recommended)
**Notes:** REQUIREMENTS.md WAYP-04/05 and PLANUI-01/02 text amended to "route anchor" as a result (D-01).

---

## Waypoint/anchor list sheet — delete & reorder UX

| Option | Description | Selected |
|--------|-------------|----------|
| Trailing delete icon | Explicit tap target next to each row, avoids drag-gesture conflict. | ✓ |
| Swipe-to-delete | Dismissible/swipe row pattern. | |

**User's choice:** Trailing delete icon (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Immediate delete, no confirmation | Undo/redo (Phase 19 app bar) is the safety net. | ✓ |
| Confirm before delete | Dialog or snackbar-undo before committing. | |

**User's choice:** Immediate delete, no confirmation (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| No guard — delete freely down to zero | Simplest behavior. | ✓ |
| Guard — always keep at least 1 anchor | Delete disabled/no-op on last anchor. | |

**User's choice:** No guard — delete freely down to zero (Recommended)
**Notes:** Reorder follows the existing `ReorderableListView.builder` pattern from `settings_categories_screen.dart` (not separately debated — confirmed via codebase scout as the reuse target).

---

## Elevation profile — reuse & live-update behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Adapt existing ElevationProfile | Make `trail` optional, use `gpx.getTotals()`, replace/drop waypoint-icon overlay. | (superseded, see notes) |
| New lightweight widget | Purpose-built simpler chart for the planner. | |

**User's choice (free text):** "Build the trail together with the route. It will be needed anyways when handing it over to the trail_create_screen in phase 21, so we might as well build it incrementally. That way we can simply hand it over to the elevation profile. Route anchors do not need to appear in the profile."
**Notes:** This reframed the question — the decision became "adapt `ElevationProfile` (trail optional) fed by an incrementally-synthesized `Gpx`, with no route-anchor overlay" (D-09/D-12). Follow-up questions resolved the remaining mechanics:

| Follow-up | Option | Selected |
|-----------|--------|----------|
| Where does the synthesized Gpx live? | Derived provider (watches routeAnchorsProvider) | ✓ (D-10) |
| Derived provider | Owned by RouteAnchorsState | |
| When to fetch `/valhalla/height`? | Only while elevation view is open, debounced | ✓ (D-11) |
| When to fetch `/valhalla/height`? | Always fetch in background | |
| Empty state (<2 anchors)? | Empty-state message (reuse `_EmptyState`) | ✓ (D-13) |
| Empty state (<2 anchors)? | Disable the elevation toggle button entirely | |

---

## Location search screen — presentation & pan/zoom behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Full-screen push | Mirrors GlobalSearchScreen exactly, filtered to locations. | ✓ (D-14) |
| Modal/bottom sheet | Overlay on top of the planner map. | |

**User's choice:** Full-screen push (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Fixed zoom 13 | Matches existing GlobalSearchScreen → map handoff. | ✓ (D-15) |
| Keep current planner zoom level | Pan only, preserve zoom. | |

**User's choice:** Fixed zoom 13 (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-routing → List/Elevation toggle → Search | Auto-routing stays on top. | |
| Search → Auto-routing → List/Elevation toggle | Search first, route-editing controls grouped below. | ✓ (superseded — see notes) |

**User's choice:** Search → Auto-routing → List/Elevation toggle (Recommended... but see below)
**Notes:** This ordering question was asked assuming a 3-button controls column (search + auto-routing + a list/elevation toggle). The very next question replaced the list/elevation toggle button with a persistent tabbed sheet entirely (see below), so the final controls column is just Search → Auto-routing (D-04), with the third button dropped.

---

## Sheet mechanism — how list & elevation views are toggled

| Option | Description | Selected |
|--------|-------------|----------|
| Two separate buttons | One opens list, one opens elevation, mutually closing. | |
| Single toggle button | One button, icon/action changes based on current view. | |

**User's choice (free text):** "Maybe two tabs in the same draggablescrollable sheet is the better UX?"

Follow-up — confirm interpretation:

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — single button, tabbed sheet | One map control button opens a sheet with a TabBar inside. | (superseded, see notes) |
| No — go back to two separate buttons | Revert to earlier plan. | |

**User's choice (free text):** "No map button. There is only one sheet with two tabs: 1. Route anchor list 2. elevation profile. What do you think?"
**Notes:** Removed the map-control-button trigger entirely, not just merged the two views into tabs. This is a SCOPE CHANGE from the original ROADMAP/REQUIREMENTS wording ("toggled via map control buttons") — reconciled directly in `.planning/ROADMAP.md` and `.planning/REQUIREMENTS.md` (D-02).

Follow-up — sheet visibility:

| Option | Description | Selected |
|--------|-------------|----------|
| Always docked, peek height when route has ≥1 anchor | Sheet appears once route has ≥1 anchor, hidden on empty route. | ✓ (D-03) |
| Always docked regardless of anchor count | Sheet present from screen open even at 0 anchors. | |

**User's choice:** Always docked, peek height when route has ≥1 anchor (Recommended)

---

## Claude's Discretion

- Exact `Gpx`-from-points construction helper location/signature (likely a new `gpx_util.dart` function) — no existing helper does this.
- Exact debounce duration tuning beyond the ~500ms reference point.
- `DraggableScrollableSheet` peek height, tab icon choices, and exact size-fraction constants — follow `WaypointSheet`'s existing values as a starting point.

## Deferred Ideas

None — discussion stayed within phase scope. The sheet-mechanism change and incremental-Gpx-synthesis decision are implementation-mechanism refinements of already-scoped requirements, not new capabilities.
