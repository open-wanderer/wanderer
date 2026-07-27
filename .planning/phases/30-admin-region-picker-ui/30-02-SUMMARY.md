---
phase: 30-admin-region-picker-ui
plan: 02
subsystem: admin-ui
tags: [pocketbase, alpinejs, maplibre-gl, region-toggle, coverage-map]

# Dependency graph
requires:
  - phase: 30-01
    provides: "regionsApp() Alpine shell (auth, apiFetch, tree state/render), .toggle-slot and #region-map placeholder slots, rowErrors/map/enabledPolygons state stubs"
provides:
  - "Optimistic leaf enable/disable toggle (PATCH /api/collections/regions/records/{id}) with revert-on-failure and inline per-row error"
  - "Live MapLibre coverage map rendering the boundary polygon of every currently-enabled leaf, fit to their union on load"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Optimistic toggle-with-revert (flip -> PATCH -> revert+rowErrors on failure), mirroring federation_ui.html's approve/reject/disconnect shape"
    - "Chunked OR-filter fetch against a path-keyed sibling collection (region_polygons), chunk size derived from the actual max path length rather than hardcoded"
    - "MapLibre source/layer id scheme keyed by region path: region-{path}, region-fill-{path}, region-line-{path}"

key-files:
  modified:
    - db/routes/regions_ext/regions_ui.html

key-decisions:
  - "addPolygonForRow/removePolygonForRow implemented as instance methods with an internal `if (!this.map) return;` guard (rather than guarding at each call site in toggleRegion) — keeps toggleRegion's body identical to the RESEARCH.md reference pattern and made Task 1 independently testable before Task 2 replaced the stub bodies"
  - "Map empty-state overlay visibility (`enabledCount === 0`) is a live reactive x-show binding, not a separately-tracked state flag re-computed only at fit time — D-06 only restricts re-fitBounds on toggle, not the overlay's reactivity, so binding directly to the existing enabledCount getter is simpler and can't drift out of sync"
  - "Chunk size for the region_polygons OR-filter is computed per-request from the actual max path length in the enabled set (`floor(3400 / (maxPathLen + 10))`, falling back to 60), per Research Assumption A2, rather than a hardcoded 60"

requirements-completed: [ADMINUI-02, ADMINUI-03]

# Metrics
duration: 10min
completed: 2026-07-27
---

# Phase 30 Plan 02: Admin Region Picker UI — Toggle + Live Map Summary

**Optimistic leaf enable/disable toggle with PATCH-and-revert plus a live MapLibre map that renders every enabled leaf's boundary polygon and fits to their union on load, completing Phase 30's two remaining requirements**

## Performance

- **Duration:** ~10 min
- **Tasks:** 2/2
- **Files modified:** 1 (`db/routes/regions_ext/regions_ui.html`)

## Accomplishments
- Each leaf tree row now renders an accessible `role="switch"` toggle (keyboard-operable via Space/Enter, dynamic `aria-label` "Enable {name}"/"Disable {name}") that flips `row.enabled` optimistically and PATCHes `/api/collections/regions/records/{id}` with a literal boolean body — on failure it reverts the toggle, undoes the optimistic map change, and shows an inline `.row-error` (x-text-only, no `x-html`) with no retry button (ADMINUI-02)
- `regionsApp()` now initializes a MapLibre GL JS map on the right pane after auth + tree load, styled with OpenFreeMap's Liberty style; on the map's `load` event it fetches every enabled leaf's polygon via a chunked `region_polygons` OR-filter (chunk size derived from the actual max path length, per Research A2) and renders a fill(18% accent)+line(2px accent) layer pair for each, then fits the map to the union of enabled bboxes with 48px padding — falling back to a Europe-centered view + "No regions enabled" overlay when zero leaves are enabled (ADMINUI-03)
- Toggling a leaf now adds/removes only that single region's polygon layer via a fresh single-path `region_polygons` fetch — the map never re-fits on individual toggles, only on initial load (D-06)
- Loading spinner (`.spinner-accent`) shown over the map pane until the MapLibre style finishes loading

## Task Commits

1. **Task 1: Optimistic leaf toggle with revert + inline per-row error (ADMINUI-02)** - `7cc71446` (feat)
2. **Task 2: Live MapLibre coverage map + chunked polygon fetch + toggle wiring (ADMINUI-03)** - `b0f91521` (feat)

**Plan metadata:** pending (this SUMMARY commit)

## Files Created/Modified
- `db/routes/regions_ext/regions_ui.html` - Added `toggleRegion(row)` (optimistic PATCH + revert + rowErrors), the toggle-switch markup replacing 30-01's `.toggle-slot` placeholder, `initMap()`/`loadEnabledPolygons()`/`addRegionLayer()`/`removeRegionLayer()`/`fitToEnabled()`/`enabledLeafRows()`, real `addPolygonForRow`/`removePolygonForRow` implementations (replacing Task 1's map-hook stubs), and the map loading-spinner + zero-enabled empty-state overlay markup/CSS

## Decisions Made
- `addPolygonForRow`/`removePolygonForRow` guard internally on `this.map` rather than at each `toggleRegion` call site, keeping `toggleRegion`'s body identical to the RESEARCH.md reference pattern
- Map empty-state overlay binds directly to the existing `enabledCount` getter (`x-show="mapLoaded && enabledCount === 0"`) instead of a separately-tracked flag, since D-06 only restricts re-`fitBounds()` on toggle, not general map-pane reactivity
- Polygon-fetch chunk size is computed per-request from the actual max path length in the enabled set (`floor(3400 / (maxPathLen + 10))`, falling back to 60) rather than a hardcoded constant, per Research Assumption A2

## Deviations from Plan

None - plan executed exactly as written. Task 1's map-hook stubs were guarded internally (`if (!this.map) return;` inside the method bodies) exactly as the plan's action text specified ("safe no-op stubs guarded by `if (this.map)`"), and Task 2 replaced those bodies with the real MapLibre calls as directed.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required. `/region-catalog/` is immediately usable once the server restarts, gated by the existing PocketBase superuser login.

## Next Phase Readiness
- Phase 30 (Admin Region Picker UI) is now feature-complete: ADMINUI-01 (30-01), ADMINUI-02, and ADMINUI-03 (this plan) are all implemented.
- The deferred end-of-phase human-check (tree render/expand/filter from 30-01, plus this plan's toggle-persist/map-render/failure-revert/many-enabled-chunking checks) should be run on a live instance before the milestone moves on to Phase 31 (Flutter Settings Hierarchy).
- Phase 31 depends on Phase 29's `GET /api/v1/regions` (already shipped), not on this admin page, so it is unblocked regardless of when the live human-check runs.

---
*Phase: 30-admin-region-picker-ui*
*Completed: 2026-07-27*
