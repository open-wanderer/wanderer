---
phase: 25-map-rendering-region-based-viewport-pipeline
plan: 03
subsystem: mobile-map-rendering
tags: [flutter, maplibre, riverpod, offline-tiles, region-registry]

# Dependency graph
requires:
  - phase: 25-01
    provides: RENDER-03 settled on incremental composition (option-b); hillshade z-order fix (belowLayerId) and add-only scope (no repaint-on-remove branch needed here)
  - phase: 25-02
    provides: TileRepositoryManager.localTilePathsForBounds returning a typed ({vectorPaths, demPaths}) record
provides:
  - TrailMap._composeStyle sources its offline vector/DEM tile paths from the app-wide region registry (localTilePathsForBounds(widget.trail.bounds)) instead of Trail.pmTiles/demPmTiles
  - TrailMap._addRegionComposition: an incremental ADD-ONLY reconcile that addSource/addLayer's a mid-session region's sources/layers without a full setStyle reload, hillshade inserted below the first vector layer
  - _addedSourceIds/_addedLayerIds tracking sets seeded from the baked style in _onStyleLoaded via _seedRegionTracking
  - _sourceFromJson/_layerFromJson JSON->typed maplibre Source/StyleLayer translation helpers
affects: [26-trail-download-guard, 27-legacy-cleanup]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Region-sourced offline tile composition: localTilePathsForBounds(trail.bounds) replaces trail-scoped pmTiles/demPmTiles fields as the single source of truth for a screen's offline style"
    - "Incremental add-only style reconcile: diff a freshly-composed style's source/layer ids against a tracked Set, addSource/addLayer only the newly-relevant ones, never remove — the correct pattern for a screen whose viewport/bounds never change post-mount"

key-files:
  created: []
  modified:
    - app/lib/components/base/trail_map.dart

key-decisions:
  - "Followed PLAN.md's revised incremental-composition design (RENDER-03 option-b) verbatim, not the earlier full-setStyle-reload description still present in 25-PATTERNS.md — the plan's frontmatter/objective explicitly superseded that pattern map after 25-01's on-device spike; PLAN.md is the authoritative, more specific source (exact grep gates, threat model, must_haves)"
  - "Reworded a doc comment that literally spelled out widget.trail.pmTiles/demPmTiles (to explain what was replaced) to avoid falsely tripping the plan's own zero-occurrence grep gate for those legacy field reads"

patterns-established:
  - "JSON->typed maplibre translation pair (_sourceFromJson/_layerFromJson) for converting rewriteStyleForOffline's composed style JSON into ml.Source/ml.StyleLayer objects consumable by StyleController.addSource/addLayer — reusable if navigation_screen.dart or another incremental-composition screen is added later"

requirements-completed: [RENDER-01]

# Metrics
duration: ~20min
completed: 2026-07-23
---

# Phase 25 Plan 03: Region-Sourced TrailMap Composition Summary

**TrailMap's offline basemap/hillshade now sources vector+DEM tile paths from the app-wide region registry (`localTilePathsForBounds(trail.bounds)`) instead of the trail's own `pmTiles`/`demPmTiles` fields, and applies a mid-session region download incrementally via `addSource`/`addLayer` (hillshade below the first vector layer) instead of a full `setStyle` reload.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-07-23
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- `_composeStyle` now queries `TileRepositoryManager.localTilePathsForBounds(widget.trail.bounds)` fresh on every call (bake-time bounds fixed per D-05, but the region set behind them is re-evaluated each call) and passes the typed `vectorPaths`/`demPaths` split to `rewriteStyleForOffline`'s `cellPaths`/`demCellPaths` params — never conflated.
- An uncovered viewport (`tiles.vectorPaths.isEmpty`) makes `_composeStyle` return `null` before `rewriteStyleForOffline` would throw on empty `cellPaths`, so the existing `build()` passthrough renders a blank basemap with no banner (D-01/D-02).
- New `_addRegionComposition()` incremental ADD-ONLY reconcile: recomputes the composed style for the trail's fixed bounds, diffs its `sources`/`layers` against `_addedSourceIds`/`_addedLayerIds`, and `addSource`/`addLayer`s only the newly-relevant ids — wired to `ref.listen(regionListNotifierProvider, ...)` so a region finishing download while the trail detail screen is open updates the map without remounting or a full-style flash (RENDER-03 option-b).
- Hillshade layers are added via `addLayer(layer, belowLayerId: firstVectorLayerId)` so relief renders underneath the vector basemap, not on top (25-01 finding 2).
- `_addedSourceIds`/`_addedLayerIds` are (re)seeded from whatever's actually baked into the style every time a full style load fires (`_onStyleLoaded` → `_seedRegionTracking`), covering both the initial mount and a theme-toggle `setStyle` swap.
- No remove branch exists anywhere in the file (`grep -Ec "removeSource|removeLayer"` = 0) — matches D-06: TrailMap's bounds never change post-mount, so a region only ever adds coverage.

## Task Commits

1. **Task 1: Source TrailMap's offline style from the region registry and apply mid-session regions incrementally** - `a03d6eec` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/components/base/trail_map.dart` - `_composeStyle` rewired off `Trail.pmTiles`/`demPmTiles` onto the region registry; new `_addRegionComposition`/`_seedRegionTracking`/`_sourceFromJson`/`_layerFromJson`; new `_addedSourceIds`/`_addedLayerIds` tracking sets; new third `regionListNotifierProvider` listen routed to `_addRegionComposition` (not `_swapStyle`)

## Decisions Made
- Followed PLAN.md's incremental-composition design (RENDER-03 = option-b) as the authoritative source over the stale full-`setStyle`-reload description still present in `25-PATTERNS.md` (that pattern map predates 25-01's on-device spike findings that PLAN.md's own frontmatter says superseded it).
- Reworded one doc comment that literally spelled out the legacy `widget.trail.pmTiles`/`demPmTiles` field names (to explain what was replaced) so it doesn't itself trip the plan's own zero-occurrence acceptance-criteria grep for those reads.

## Deviations from Plan

None - plan executed exactly as written (one cosmetic doc-comment wording adjustment to satisfy the plan's own grep gate, not a functional deviation).

## Issues Encountered

None. `flutter analyze lib/components/base/trail_map.dart` reports no issues; `flutter analyze` across the whole app shows only pre-existing, unrelated warnings/info in other files (dead code in `map_screen.dart`, deprecated FontAwesome icon names in `icon_util.dart`, etc.) — none touch `trail_map.dart`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- RENDER-01 is now fully closed end-to-end: the typed vector/DEM split (25-02) feeds directly into TrailMap's offline composition (this plan).
- RENDER-03's incremental-composition strategy is implemented for `TrailMap`; `navigation_screen.dart`'s parallel rewiring (camera-idle-driven, live viewport bounds) is a separate concern for a later plan/phase per D-06's scope split (not part of this plan's `files_modified`).
- On-device human verification (airplane mode: offline render with hillshade underneath; uncovered-viewport blank basemap; mid-session region download applied incrementally without remount/flash) is deferred to end-of-phase UAT per this project's `human_verify_mode=end-of-phase` convention — not blocking for this plan's completion.
- Pre-existing unrelated local working-tree changes (`router_provider.g.dart`, `settings_offline_regions_screen.dart`, `region_render_spike_harness.dart`, untracked `25-PATTERNS.md`) were left untouched — out of this plan's `files_modified` scope, not committed here.

---
*Phase: 25-map-rendering-region-based-viewport-pipeline*
*Completed: 2026-07-23*

## Self-Check: PASSED

- FOUND: app/lib/components/base/trail_map.dart
- FOUND: a03d6eec (Task 1 commit)
- FOUND: 14c415fb (summary commit)
