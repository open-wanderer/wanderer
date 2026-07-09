# Phase 16: List & Map Screens on MapLibre - Context

**Gathered:** 2026-07-09
**Status:** Ready for planning

<domain>
## Phase Boundary

The browse surfaces (`list_detail_map_screen`, `list_detail_screen`'s inline map, and `map_screen`) move onto `MapLibreMap`. `list_detail_map_screen`/`list_detail_screen` fit every trail in a list on one map (CORE-08). `map_screen` switches from client-side clustering (`flutter_map_marker_cluster` over `/search/trails`) to server-side clustering (`POST /search/trails/cluster`) rendered as native circle/symbol layers (CLUS-01..05).

`navigation_screen` stays on `flutter_map` (Phase 17). `WandererMap`/`trail_detail_map_screen` (Phase 15) are already done and out of scope here except as the established pattern to follow.

</domain>

<decisions>
## Implementation Decisions

### Search trigger model (map_screen)
- **D-01:** Keep today's manual "Search this area" button (appears after the user drags the map, tap to re-query) — do NOT implement auto-debounced re-query on pan/zoom. ROADMAP.md's "debounced exactly as today" phrasing is corrected by this decision to mean "same trigger mechanism as today" (manual button), not a new auto-debounce feature. This is a pure engine swap (client-cluster → server-cluster, flutter_map → MapLibre), not a UX change.

### Unclustered-point tap behavior (map_screen)
- **D-02:** Tapping an unclustered point selects the trail immediately (bottom selection card appears, matching today), then fetches that trail's polyline via the existing `trailPolylineProvider` (already used in `map_screen.dart` today) and fits the camera to it once the fetch resolves. No new two-stage/instant-flyTo-then-refine camera pattern — reuse the fetch-then-fit approach already in the codebase. Brief camera lag while the polyline loads on slow connections is acceptable.
- Contrast with web: web's `ClusterLayer.zoomOnUnclusteredPoint` does a fixed `flyTo(zoom: 12)` with no polyline fetch — the app is intentionally richer here per ROADMAP.md's explicit wording ("fits the camera to its polyline"), this is not a parity gap with web.

### Cluster circle styling (map_screen)
- **D-04 (post-research correction):** Port `web/src/lib/vendor/maplibre-layer-manager/cluster-layer.ts`'s exact `circle-radius` step ramp (`step, point_count, 10, 5, 12, 10, 15, 50, 18, 100, 22, 500, 25`), `circle-color: #242734`, `circle-stroke-width: 2`, `circle-stroke-color: #fff`, and the `cluster-count` symbol layer (`point_count_abbreviated` label) verbatim as native `CircleStyleLayer`/`SymbolStyleLayer` — for GROUPED points (actual clusters) only.
- **D-05 (post-research correction, overrides a literal "verbatim port"):** Research found that a literal verbatim port also flattens UNCLUSTERED (individual) trail markers to web's plain `circle-color: #242734, circle-radius: 5` dot — losing today's app-specific category icon on single-trail markers. User explicitly prioritized keeping today's look over web parity: **unclustered points keep a `WidgetLayer`/`Marker` with the trail's category icon** (same pattern `list_detail_map_screen`/`TrailMarkerLayer` already use), filtered to only render for features where `point_count == 1`. Only the grouped-cluster circles + count label are the verbatim web port; the `unclustered-point` native circle layer from `cluster-layer.ts` is NOT ported — it's replaced by the widget-marker approach.
- This does not affect `list_detail_map_screen`/`list_detail_screen` (CORE-08) — no clustering there, they already use category-icon `WidgetLayer` markers per the original plan.
- **Tap-handling consequence of D-05:** since unclustered points are now `WidgetLayer` markers (not a native `unclustered-point` circle layer), CLUS-03's tap detection simplifies — the marker's own `GestureDetector.onTap` handles the unclustered-tap case directly (same as `TrailMarkerLayer`/list-screen markers), and `MapController.featuresAtPoint(...)` is only needed to detect taps on the native `clusters` circle layer (RESEARCH.md Pattern 2's `featuresAtPoint(layerIds: ['unclustered-point'])` branch is now unnecessary — drop it).

### `is_large` trail handling (map_screen)
- **D-03 (superseded during 16-03's device checkpoint):** Originally: `is_large` trails simply invisible on the map screen, no placeholder. **Correction:** on-device testing found the server marks the top `MAP_MAX_POLYLINES` (100) trails by size *currently in view* as `is_large` once zoom passes `clusteringMaxZoom` (~11) — with fewer than 100 trails in view (the normal single-trail zoom-in case), this can mean ALL visible trails, not the rare huge ones D-03 assumed. Corrected decision: unclustered category-icon markers (map_screen.dart) do NOT filter on `is_large` — any unclustered point (`point_count == 1`) renders as a marker regardless. The native cluster/count circle layers' `is_large` filter is left as-is (harmless — `is_large` features always carry `point_count: 1`, already excluded from those layers by the `point_count > 1` clause). Full-polyline rendering for genuinely large/complex trails remains deferred to FUT-01 — this correction only restores marker visibility, it doesn't add polyline rendering.

### Claude's Discretion
- Exact native layer/source ids for the cluster circles/count/unclustered-point layers (naming convention only — functionally must match web's filter/paint/layout values per D above).
- Whether `list_detail_map_screen`/`list_detail_screen`'s CORE-08 port reuses `WandererMap` directly or needs its own thin MapLibre host — not discussed; these screens don't show a single trail's track/waypoints (no `TrailLayer`/`TrailMarkerLayer` need), just markers + camera-fit-to-bounds, so a lighter-weight approach than reusing the full `WandererMap` may be appropriate. Investigate during planning/research.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Web reference implementation (port target for cluster styling)
- `web/src/lib/vendor/maplibre-layer-manager/cluster-layer.ts` — the exact `circle-radius` step ramp, colors, and `unclustered-point`/`cluster-count` layer definitions this phase ports natively (CLUS-02). Also documents web's simpler `flyTo`-based tap handlers (`zoomOnCluster`: `flyTo(zoom: currentZoom + 2)`; `zoomOnUnclusteredPoint`: `flyTo(zoom: 12)`) — the app's cluster-tap behavior (CLUS-03) should match web's zoom+2-on-cluster-tap approach; the unclustered-point tap is intentionally richer per D-02 above.

### Server endpoint (already exists, used as-is)
- `web/src/routes/api/v1/search/trails/cluster/+server.ts` (or equivalent backend route) — `POST /search/trails/cluster`, already runs Supercluster server-side, applies category-preference filters, splits `is_large` trails out, returns `point_count`/`point_count_abbreviated` per PROJECT.md's "Context" section. No server-side changes needed this phase.

### Current app implementation being replaced
- `app/lib/routes/map_screen.dart` — current client-side clustering via `flutter_map_marker_cluster` over `/search/trails` (not the cluster endpoint), the manual `_searchAreaController`/"Search this area" button (D-01's reference), and the existing `trailPolylineProvider` fetch-then-fit-camera pattern (D-02's reference) at the cluster-tap handler.
- `app/lib/routes/list_detail_map_screen.dart`, `app/lib/routes/list_detail_screen.dart` — current `_combinedBounds` bounds-fit-all-trails logic (flutter_map-based) that CORE-08 ports to MapLibre.

### Prior phase work this phase builds on
- `.planning/phases/15-maplibre-core-trail-rendering-offline-parity/15-06-SUMMARY.md` and related Phase 15 SUMMARYs — `WandererMap`, `mapStyleJsonProvider`, the `ml.MapController`/`onMapCreated` pattern, and the established `@Riverpod(keepAlive: true)` provider conventions this phase should follow for consistency.
- `.planning/phases/15-maplibre-core-trail-rendering-offline-parity/15-VERIFICATION.md` — notes the `trail-arrow` self-registered sprite image id fix; if this phase's cluster layers register any custom images, use a similarly distinct, collision-free id (not a bare generic name that might clash with the basemap sprite).

</canonical_refs>

<specifics>
## Specific Ideas

None beyond the decisions above.

</specifics>

<deferred>
## Deferred Ideas

- Auto-debounced re-query on pan/zoom (considered and rejected in favor of keeping the manual "Search this area" button — D-01).
- Two-stage instant-flyTo-then-refine camera animation on unclustered-point tap (considered and rejected in favor of the existing fetch-then-fit pattern — D-02).
- `is_large` trail placeholder/indicator on the map screen (considered and rejected — invisible for now, per FUT-01 — D-03).
- Rendering `is_large` trails as full polylines on the map screen — explicitly out of scope, tracked as FUT-01 in a future milestone.

</deferred>

---

*Phase: 16-list-map-screens-on-maplibre*
*Context gathered: 2026-07-09*
