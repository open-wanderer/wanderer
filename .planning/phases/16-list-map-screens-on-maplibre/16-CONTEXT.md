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
- **Claude's Discretion (see canonical_refs):** port `web/src/lib/vendor/maplibre-layer-manager/cluster-layer.ts`'s exact `circle-radius` step ramp (`step, point_count, 10, 5, 12, 10, 15, 50, 18, 100, 22, 500, 25`), `circle-color: #242734`, `circle-stroke-width: 2`, `circle-stroke-color: #fff`, and the `unclustered-point` / `cluster-count` layer definitions verbatim as native `CircleStyleLayer`/`SymbolStyleLayer` — this was not discussed as a gray area since ROADMAP.md explicitly says "matching web's ClusterLayer step ramp" and the source is a direct, unambiguous port target.

### `is_large` trail handling (map_screen)
- **D-03:** `is_large` trails are simply invisible on the map screen for this phase — no placeholder, badge, or indicator. Matches web's own current behavior (web also filters `is_large` out of its circle/point layers with no separate treatment) and PROJECT.md's explicit FUT-01 deferral ("Render `is_large` trails as full polylines on the map screen" — future milestone). Filter server-side response the same way web does (`["!=", ["get", "is_large"], true]`), not client-side.

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
