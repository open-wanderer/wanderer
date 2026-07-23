# Phase 25: Map Rendering — Region-Based Viewport Pipeline - Context

**Gathered:** 2026-07-23
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase rewires two existing offline map-rendering call sites — `TrailMap` and `navigation_screen` — off `Trail.pmTiles`/`demPmTiles` and onto `TileRepositoryManager.localTilePathsForBounds`, so offline basemap/hillshade rendering is sourced from the app-wide region registry instead of trail-scoped caches. It also resolves RENDER-03's highest-risk unknown (a spike confirming whether maplibre 0.3.5 supports incremental style source/layer add/remove vs. only full style reload) before committing to a composition strategy, and scopes viewport-based style-source swapping (RENDER-02) to where the camera actually moves. Nothing about the Settings/Regions UI (Phase 24, done), the trail-download guard (Phase 26), or legacy trail-scoped code deletion (Phase 27) is touched here. `offline_style_rewriter.dart` is reused unchanged — only its callers' data source changes.

</domain>

<decisions>
## Implementation Decisions

### Uncovered-Viewport Behavior
- **D-01:** When the map viewport (TrailMap's fixed trail-bounds query, or navigation_screen's live GPS position) falls in an area with no downloaded region, render a blank/empty basemap — the same failure mode as today's offline trail-map when `cellPaths` is empty. No new "no offline data" banner or online-fallback attempt; this stays a pure offline-rendering pipeline swap with no new UI surface.
- **D-02:** This blank-basemap behavior is uniform across both `TrailMap` and `navigation_screen` — no screen-specific "no map data here" indicator. Navigation continues live GPS/maneuver tracking regardless of whether basemap tiles are rendering.

### Region-Swap Visual Feel (navigation_screen)
- **D-03:** A brief flicker/camera-reset when crossing a region boundary during live navigation is an acceptable v1.6 tradeoff if the RENDER-03 spike finds maplibre 0.3.5 only supports full style reload (not incremental `addSource`/`removeSource`). Seamless swapping is NOT a hard requirement gating the composition strategy — ship whichever approach the spike confirms works, matching research/PITFALLS.md's own "acceptable for initial spike" framing.
- **D-04:** Region-swap recomposition is debounced on camera-idle/moveend, not recomputed on every camera-move event — mirrors the existing cluster-search bbox-query debounce pattern in `map_screen.dart` (per research/ARCHITECTURE.md's own recommendation) and minimizes exposure to Pitfall 5's known `maplibre` 0.3.5 style-reload quirks (`onStyleLoaded`/sprite `file://` issues).

### Trail Spanning Two Downloaded Regions (TrailMap)
- **D-05:** If a trail's bounds straddle two downloaded regions, both regions' vector/DEM sources render together in the same style composition — not swapped. `TrailMap`'s viewport is fixed for the screen's lifetime (queried once from `trail.bounds`, same as today's one-time `_composeStyle` call), so `localTilePathsForBounds(trail.bounds)` naturally returns every overlapping region's paths in one query. This preserves the existing multi-cell trail behavior (15-06) that already handles a trail spanning multiple download cells.

### Viewport Scope: Which Screen Gets Swap Logic
- **D-06:** RENDER-02's viewport-scoped swap-in/accumulate logic (debounced camera-idle recompute, per D-04) runs on `navigation_screen` only. `TrailMap` does a one-time bounds query at build/composition time (per D-05) — its camera doesn't move relative to the trail in a way that changes which regions are relevant, so "viewport scoping" only has real meaning where the camera moves freely during a session (navigation_screen), and is the only screen where regions could otherwise accumulate unconditionally, per RENDER-02's own wording.

### Claude's Discretion
- The exact composition strategy (incremental `addSource`/`removeSource` vs. full style reload) is Claude's/the spike's call, per D-03 — resolve via RENDER-03's direct package-API verification against the installed `maplibre` 0.3.5, not a user preference.
- Whether the RENDER-03 spike ships as a standalone throwaway harness (mirroring 23-06's `tile_repository_manager_harness.dart` precedent) or is folded directly into the first implementation plan's Task 1 is a planning-level call.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Roadmap
- `.planning/ROADMAP.md` — Phase 25 section (goal, 4 success criteria, "Depends on: Phase 24", risk gate calling out RENDER-03 as the milestone's highest-risk unknown)
- `.planning/REQUIREMENTS.md` — RENDER-01, RENDER-02, RENDER-03 (all currently `Pending`)

### Prior Research (this phase's risk gate and architecture were already scoped by milestone research)
- `.planning/research/ARCHITECTURE.md` §Q1 (lines ~24-38) — recommended `regionTileRepositoryProvider`/`localTilePathsForBounds` composition, viewport resolution on camera-idle not per-frame, keep `mapStyleJsonProvider`/region query/`offline_style_rewriter.dart` as parallel composable layers (Anti-Pattern 2 explicitly warns against merging them)
- `.planning/research/ARCHITECTURE.md` §Q4 build order (lines ~67-80) — this phase is "map-screen rewiring," step 4 of 6; leaves `TrailEntity.pmTiles`/`demPmTiles` physically in place (deletion deferred to Phase 27)
- `.planning/research/PITFALLS.md` Pitfall 5 (lines ~102-122) — full style-reload-on-every-swap risk, LOW/MEDIUM confidence on maplibre 0.3.5's incremental source/layer API availability, explicit "verify directly against the installed package" mitigation — this is RENDER-03's spike
- `.planning/research/SUMMARY.md` (lines ~115, 131-159) — confirms Phase 4 (this phase) needs direct API verification + an empirical 10-20 duplicated-source layer-count/performance spike before committing to a strategy (matches ROADMAP's success criterion 1)

### Prior Phase Context (data model + download engine this phase consumes)
- `.planning/phases/23-tilerepositorymanager-download-engine/23-05-SUMMARY.md` and `23-RESEARCH.md` §Pattern 4 — `localTilePathsForBounds(LngLatBounds)` signature, hand-rolled `bboxOverlaps` axis-aligned overlap test, skips regions with a null package target
- `.planning/phases/23-tilerepositorymanager-download-engine/23-06-PLAN.md`/`23-06-SUMMARY.md` — precedent for a standalone on-device spike/verification harness (`app/test/services/tile_repository_manager_harness.dart`) kept out of production routes; a reusable pattern if RENDER-03's spike needs the same shape

### Data Model & Download Engine (query surface this phase calls)
- `app/lib/services/tile_repository_manager.dart` — `localTilePathsForBounds(LngLatBounds query)` (the query this phase's callers switch to), `bboxOverlaps`
- `app/lib/provider/region/tile_repository_provider.dart` — Riverpod seam if a live-updating region provider (not just a one-shot manager call) is needed for navigation_screen's debounced recompute

### Existing Rendering Call Sites (this phase's primary edit targets)
- `app/lib/components/base/trail_map.dart` — `_composeStyle` (lines ~127-145) currently reads `widget.trail.pmTiles`/`demPmTiles` directly; called once per build plus on theme-toggle `_swapStyle` (lines ~146-155); `bounds` used at line ~281 for camera fit
- `app/lib/routes/navigation_screen.dart` — inline `_composeStyle` (lines ~871-905) reads `ref.read(trailProvider(widget.id)).value?.pmTiles`/`demPmTiles`; no camera-idle listener currently drives recomposition — RENDER-02's debounced swap (D-04/D-06) is new logic here, not a data-source swap of existing logic
- `app/lib/util/offline_style_rewriter.dart` — pure function taking `cellPaths`/`demCellPaths` as opaque path lists; reused unchanged per D-05/D-06 and ROADMAP success criterion 4 — only callers' path-list source changes

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `TileRepositoryManager.localTilePathsForBounds(LngLatBounds)` (Phase 23) — already implemented, tested (bbox-overlap unit tests), and returns exactly the `(vectorPaths, demPaths)`-shaped data both call sites need; this phase is primarily a consumer, not a new build.
- `rewriteStyleForOffline` (`offline_style_rewriter.dart`) — signature already accepts opaque `cellPaths`/`demCellPaths` lists; no changes needed, confirmed by both ROADMAP success criterion 4 and research/ARCHITECTURE.md.
- `map_screen.dart`'s existing cluster-search bbox-query debounce pattern — the precedent D-04's camera-idle debounce on navigation_screen should mirror.

### Established Patterns
- `TrailMap`'s existing dual-listen pattern (`ref.listen(mapStyleJsonProvider)` + `ref.listen(glyphSpriteCacheProvider)`) extends directly to a third `ref.listen(regionTileRepositoryProvider-equivalent)` per ARCHITECTURE.md — a region finishing download mid-session should live-swap the style the same way a glyph-cache-warm completion does today.
- Known `maplibre` 0.3.5 quirks to respect regardless of spike outcome: `onStyleLoaded` can fire before `onMapCreated` (buffered-replay pattern, already used in both `TrailMap` and `navigation_screen`); "instant" camera moves must use `Duration(milliseconds: 1)`, never `Duration.zero`.

### Integration Points
- `TrailMap._composeStyle` and `navigation_screen`'s inline `_composeStyle` are the two edit points for RENDER-01 (swap data source).
- Debounced camera-idle listener is new logic added to `navigation_screen` for RENDER-02 (D-04/D-06) — no equivalent needed in `TrailMap` (D-06).
- RENDER-03's spike is a precondition for finalizing HOW recomposition applies the new style (incremental vs. full reload) — must land before or alongside the RENDER-01/02 rewiring, since the composition strategy affects how both call sites' recomposition is implemented.

</code_context>

<specifics>
## Specific Ideas

- Blank basemap for uncovered viewport, no new banner/indicator UI (D-01/D-02) — keeps this phase a pure rendering-pipeline swap.
- Debounce on camera-idle for navigation_screen's region-swap recompute, matching the existing map_screen.dart cluster-search debounce (D-04).
- A trail spanning two regions renders both regions' sources together via one bbox query — never a swap (D-05).

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed entirely within phase scope. No new capabilities were proposed; all four discussed areas were implementation-mechanics questions about the already-scoped rendering pipeline swap.

</deferred>

---

*Phase: 25-map-rendering-region-based-viewport-pipeline*
*Context gathered: 2026-07-23*
