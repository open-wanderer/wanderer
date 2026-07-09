# Phase 16: List & Map Screens on MapLibre - Research

**Researched:** 2026-07-09
**Domain:** Flutter native GL map layers (MapLibre 0.3.5) — multi-trail bounds-fit maps and server-driven cluster rendering
**Confidence:** HIGH

## Summary

This phase ports three flutter_map surfaces to `MapLibreMap`: the two multi-trail list maps (`list_detail_map_screen.dart`, the inline `_ListMap` in `list_detail_screen.dart` — CORE-08) and the search map screen's marker rendering (`map_screen.dart` — CLUS-01..05). Both are additive next to Phase 15's `WandererMap`, which stays scoped to single-trail rendering and is **not** the right host to reuse here — it requires a `Trail` and does trail-specific offline/bounds/track work none of these three screens need. The right move is a small new lightweight `MapLibreMap` host (no `Trail` param, no offline branch, no trail-bounds camera fit) that both CORE-08 screens and `map_screen` build on, all three consuming the same `mapStyleJsonProvider` Phase 15 already established.

The two requirement groups need genuinely different techniques. CORE-08 is straightforward: `ml.PolylineLayer` (a declarative convenience layer already in the `maplibre` package) replaces flutter_map's `PolylineLayer`+manual `Polyline` list for all trails in a list, and `ml.WidgetLayer`/`ml.Marker` (the same pattern `TrailMarkerLayer` already established in Phase 15) replaces flutter_map's `MarkerLayer`. There is no declarative "fit to bounds on init" option in `MapOptions` — camera fit must happen imperatively in `onStyleLoaded` via `controller.fitBounds()`, exactly like `WandererMap._fitInitialCamera()` already does for a single trail.

CLUS-01..05 is the harder half. The cluster endpoint (`POST /search/trails/cluster`, already implemented, no server changes) returns a GeoJSON `FeatureCollection` whose points carry only `id`, `point_count`, `point_count_abbreviated`, `is_large`, `bounding_box_diagonal` — not full trail metadata. That FeatureCollection becomes a `GeoJsonSource` + three native `CircleStyleLayer`/`SymbolStyleLayer`s ported **verbatim** from `web/src/lib/vendor/maplibre-layer-manager/cluster-layer.ts` (this is a locked decision, not a design choice). Tap handling needs `MapController.featuresAtPoint()` — the Dart package's equivalent of web's `queryRenderedFeatures` — but `RenderedFeature` only exposes `id`/`properties`, **not geometry**, so cluster-tap zoom must center on the click's own `MapEventClick.point` rather than the tapped feature's coordinates (web reads `feature.geometry.coordinates`; that path does not exist in this package). Because the cluster endpoint's minimal attribute set can't power the bottom sheet's `TrailCard` list or the unclustered-tap selection card, the existing `mapTrailSearchProvider` (hitting `/search/trails` with full attributes) must keep running in parallel, not be replaced — a new `mapClusterSearchProvider` is added alongside it, not instead of it.

**Primary recommendation:** Build one new lightweight, trail-agnostic MapLibreMap host (style-loading + live theme swap only, no offline, no single-trail bounds) shared by CORE-08's two screens and `map_screen`; port `cluster-layer.ts`'s three style layers verbatim into native `CircleStyleLayer`/`SymbolStyleLayer`s driven by a `GeoJsonSource` updated via `StyleController.updateGeoJsonSource()`; keep `mapTrailSearchProvider` untouched and add a new `mapClusterSearchProvider` beside it, not in place of it.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Multi-trail bounds-fit map (CORE-08) | Client (Flutter/MapLibre) | API (existing `/list/:id` expand) | Pure rendering of already-fetched list/trail data; no new backend work |
| Cluster search + styling (CLUS-01/02) | Client (Flutter/MapLibre) | API (`POST /search/trails/cluster`, unchanged) | Server already runs Supercluster and returns styled-ready GeoJSON; client only renders and re-queries |
| Cluster/point tap → camera or selection (CLUS-03) | Client (Flutter/MapLibre) | API (`/trail/:id` via `trailPolylineProvider`) | Tap hit-testing and camera math are client-only; polyline fetch is the one network hop, mirroring today's app behavior |
| Pan/zoom re-query trigger (CLUS-04, corrected by D-01) | Client (Flutter/MapLibre) | — | Manual "Search this area" button only, per locked decision — no auto-debounce work needed |
| Category/subcategory filter enforcement (CLUS-05) | API (`withTrailPreferenceMeiliFilter`) | Client (passes `filterText`) | Already implemented server-side; app's only job is to keep sending the same `TrailFilter.toFilterText()` output it sends today |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|---------------|
| `maplibre` | 0.3.5 (pinned in `pubspec.lock`; `pubspec.yaml` still `^0.3.3+2`) | Native GL map host, style layers, GeoJSON source | Already the phase-15-established engine; this phase adds no new package, only new usages of it |
| `supercluster` (server, Node) | Existing, unchanged | Server-side point clustering | Already running in `web/src/routes/api/v1/search/trails/cluster/+server.ts`; out of scope to touch |

No new Flutter or SvelteKit packages are introduced by this phase. **Package Legitimacy Audit is not applicable** — nothing new is installed.

**Version verification:** `pubspec.lock` locks `maplibre` at `0.3.5` (`sha256: 581e17a1eca80b2be555b00acc03afff22464efd0712d68fcc46be56efa61e59`, source `hosted`/pub.dev) `[VERIFIED: npm/pub registry — read directly from pubspec.lock]`. `pubspec.yaml`'s `^0.3.3+2` range is satisfied by the locked `0.3.5`; CLEAN-03 (Phase 18) pins this to an exact version later, out of scope here.

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `geobase` (transitive, re-exported by `maplibre.dart`) | ^1.5.0 | `Geographic`, `LngLatBounds`, `Feature<LineString>` types | Already used throughout the app since Phase 14; `ml.PolylineLayer` consumes `Feature<LineString>` built from it |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Reusing `WandererMap` for list/map screens | Extending `WandererMap` to make `trail` optional | Rejected: `WandererMap`'s offline composition (`widget.offline`, `trail.pmTiles`), single-trail `_fitInitialCamera`, and trail-track seam are all trail-specific dead weight for screens with 0-or-N trails and no offline mode. A thin new host is less code than retrofitting conditionals into `WandererMap`. |
| Manual `addSource`/`addLayer` for CORE-08 polylines (like `trail_layer.dart` does for the single-trail track) | `ml.PolylineLayer` declarative convenience widget | `ml.PolylineLayer` already exists in the installed package (`lib/src/layer/polyline_layer.dart`) and does exactly what today's flutter_map `PolylineLayer` does (uniform color/width across N `Feature<LineString>`s, no per-trail styling) — a direct drop-in, no hand-rolled GeoJSON encoding needed. |
| Native `CircleStyleLayer` with `cluster: true` on `GeoJsonSource` | Client-side supercluster | Both explicitly out of scope per `REQUIREMENTS.md`'s "Out of Scope" table — the Dart `GeoJsonSource` (confirmed in this research) has no `cluster`/`clusterRadius` fields, and client clustering is the cost this migration exists to shed. |

**Installation:** None — no new dependencies for this phase.

## Package Legitimacy Audit

Not applicable. This phase adds zero new packages to `pubspec.yaml` on either the Flutter app or the SvelteKit web side; it only writes new Dart code against the already-installed, already-verified `maplibre` 0.3.5.

## Architecture Patterns

### System Architecture Diagram — CLUS-01..05 (map_screen)

```
User drags/zooms map (gesture)
        │
        ▼
MapEventStartMoveCamera(reason: apiGesture)   ── native maplibre event, replaces
        │                                         flutter_map's MapEventMoveEnd+userGestures check
        ▼
"Search this area" button reveals (D-01: manual trigger only, no auto re-query)
        │  (user taps button)
        ▼
controller.getVisibleRegion() + controller.getCamera().zoom
        │
        ▼
mapClusterSearchProvider.searchInBounds(bounds, zoom)
        │  builds POST body: {southWest:{lat,lng}, northEast:{lat,lng}, zoom, filterText, q:""}
        │  filterText = TrailFilter.toFilterText(actor:, includeGeo:false)  (existing helper, reused)
        ▼
POST /search/trails/cluster  (server: Supercluster, unchanged)
        │  → GeoJSON FeatureCollection: cluster + unclustered-point + large features
        ▼
StyleController.updateGeoJsonSource(id: 'cluster-trails', data: jsonEncode(featureCollection))
        │  (source added once via addSource on first style load; updated thereafter — no remove/re-add)
        ▼
Native layers re-render: clusters (circle) / unclustered-point (circle) / cluster-count (symbol)
        │  ported verbatim from web's cluster-layer.ts (D: Claude's discretion — locked to verbatim port)
        ▼
User taps a circle
        │
        ▼
onEvent(MapEventClick e)
        │
        ├─ controller.featuresAtPoint(e.screenPoint, layerIds:['clusters'])  → non-empty?
        │       │
        │       ▼ yes (cluster tap, CLUS-03)
        │  controller.animateCamera(center: e.point, zoom: currentZoom + 2)
        │  (NOT feature.geometry.coordinates — RenderedFeature has no geometry field in this package;
        │   e.point IS the click's own geographic coordinate, close enough for "zoom toward it")
        │
        └─ controller.featuresAtPoint(e.screenPoint, layerIds:['unclustered-point'])  → non-empty?
                │
                ▼ yes (unclustered tap, D-02)
           trailId = feature.properties['id']
           lookup full TrailSearchResult from mapTrailSearchProvider's parallel /search/trails
              results (same bounds) — cluster endpoint doesn't carry name/thumbnail/stats
           show bottom selection card immediately
           trailPolylineProvider(trailId).future → fitBounds to polyline (existing pattern, reused)
```

### System Architecture Diagram — CORE-08 (list_detail_map_screen / list_detail_screen)

```
listProvider(id) → WandererList.expand.trails (already expanded, no new fetch)
        │
        ▼
combinedBounds = min/max lat/lon across trails (existing _combinedBounds logic, ported as-is)
        │
        ▼
New lightweight MapLibreMap host
        │  ref.watch(mapStyleJsonProvider) → initStyle
        ▼
onStyleLoaded: controller.fitBounds(bounds: combinedBounds, padding:, nativeDuration: Duration.zero)
        │  (no declarative "initialCameraFit" option exists in MapOptions — must be imperative,
        │   exactly like WandererMap._fitInitialCamera does for a single trail)
        ▼
layers: [ ml.PolylineLayer(polylines: trails.map(decoded polyline).toList()) ]
children: [ ml.WidgetLayer(markers: trails.map(tappable category-icon Marker)) ]
        │  tap on a marker → controller.fitBounds(trail.bounds, nativeDuration: 750ms)  (existing pattern)
```

### Recommended Project Structure

```
app/lib/
├── components/base/
│   ├── wanderer_map.dart          # UNCHANGED — Phase 15, single-trail only
│   └── search_map.dart            # NEW — lightweight host: style load + live theme swap,
│                                   #   no Trail param, no offline branch, no bounds-fit-on-init
│                                   #   (bounds/camera work stays imperative in each screen's onStyleLoaded)
├── components/map/
│   ├── trail_layer.dart           # UNCHANGED — Phase 15 single-trail track/markers
│   └── cluster_layer.dart         # NEW — mirrors web's cluster-layer.ts: builds the
│                                   #   GeoJsonSource + 3 style layers, exposes an
│                                   #   addClusterLayers(style, geojson) / update helper
├── provider/trail/
│   ├── map_trail_search_provider.dart   # UNCHANGED — still powers bottom-sheet TrailCard list
│   │                                     #   and unclustered-tap trail metadata lookup
│   └── map_cluster_search_provider.dart # NEW — hits POST /search/trails/cluster,
│                                          #   debounced same as existing provider, keyed by
│                                          #   bounds+zoom+filterText
└── routes/
    ├── map_screen.dart              # MODIFIED — MapLibreMap host, cluster layers, native tap handling
    ├── list_detail_map_screen.dart  # MODIFIED — MapLibreMap host, PolylineLayer + WidgetLayer
    └── list_detail_screen.dart      # MODIFIED — _ListMap inner widget same treatment, gestures: none
```

### Pattern 1: Native cluster style layers (verbatim port)

**What:** Three `CircleStyleLayer`/`SymbolStyleLayer`s over one `GeoJsonSource`, filter/paint/layout values copied 1:1 from `web/src/lib/vendor/maplibre-layer-manager/cluster-layer.ts`.
**When to use:** `map_screen`'s cluster rendering (CLUS-02). This is a locked decision (CONTEXT.md D, "port ... verbatim"), not a design choice — do not restyle.

```dart
// Source: web/src/lib/vendor/maplibre-layer-manager/cluster-layer.ts (ported)
const String kClusterSourceId = 'cluster-trails';

Future<void> addClusterLayers(ml.StyleController style, String geojson) async {
  await style.addSource(ml.GeoJsonSource(id: kClusterSourceId, data: geojson));

  await style.addLayer(const ml.CircleStyleLayer(
    id: 'clusters',
    sourceId: kClusterSourceId,
    filter: <Object>[
      'all',
      <Object>['!=', <Object>['get', 'is_large'], true],
      <Object>['>', <Object>['get', 'point_count'], 1],
    ],
    paint: <String, Object>{
      'circle-color': '#242734',
      'circle-radius': <Object>[
        'step', <Object>['get', 'point_count'],
        10, 5, 12, 10, 15, 50, 18, 100, 22, 500, 25,
      ],
      'circle-stroke-width': 2,
      'circle-stroke-color': '#fff',
    },
  ));

  await style.addLayer(const ml.CircleStyleLayer(
    id: 'unclustered-point',
    sourceId: kClusterSourceId,
    filter: <Object>[
      'all',
      <Object>['!=', <Object>['get', 'is_large'], true],
      <Object>['==', <Object>['get', 'point_count'], 1],
    ],
    paint: <String, Object>{
      'circle-color': '#242734',
      'circle-radius': 5,
      'circle-stroke-width': 2,
      'circle-stroke-color': '#fff',
    },
  ));

  await style.addLayer(const ml.SymbolStyleLayer(
    id: 'cluster-count',
    sourceId: kClusterSourceId,
    filter: <Object>[
      'all',
      <Object>['!=', <Object>['get', 'is_large'], true],
      <Object>['>', <Object>['get', 'point_count'], 1],
    ],
    layout: <String, Object>{
      'text-field': <Object>['get', 'point_count_abbreviated'],
      'text-font': <Object>['Noto Sans Regular'],
      'text-size': 11,
      'text-allow-overlap': true,
      'text-ignore-placement': true,
    },
    paint: <String, Object>{'text-color': '#fff'},
  ));
}
```

**Re-query pattern (CLUS-04):** after the first `addSource`, subsequent searches call `style.updateGeoJsonSource(id: kClusterSourceId, data: jsonEncode(newFeatureCollection))` — do **not** `removeSource`/`addSource` again; `StyleController.updateGeoJsonSource` exists exactly for this (confirmed in `maplibre_platform_interface-0.3.5/lib/src/style_controller.dart`).

### Pattern 2: Cluster/unclustered tap detection via `featuresAtPoint`

**What:** `MapController.featuresAtPoint(Offset point, {List<String>? layerIds})` is this package's equivalent of web's `map.queryRenderedFeatures(e.point, {layers: [...]})` — confirmed present in `maplibre_platform_interface-0.3.5/lib/src/map_controller.dart`.
**When to use:** Inside `WandererMap`-style `onEvent` handling for `MapEventClick`, to distinguish a cluster tap from an unclustered-point tap (CLUS-03).

```dart
// Source: maplibre_platform_interface-0.3.5 map_controller.dart (featuresAtPoint),
// map_events.dart (MapEventClick.point / .screenPoint)
onEvent: (event) {
  if (event is! ml.MapEventClick) return;
  final controller = _controller;
  if (controller == null) return;

  final clusterHits = controller.featuresAtPoint(
    event.screenPoint,
    layerIds: const ['clusters'],
  );
  if (clusterHits.isNotEmpty) {
    // CLUS-03, D: match web's zoom+2. RenderedFeature has NO geometry field in
    // this package (only id/properties) — use the click's own geographic point,
    // not the feature's coordinates (web reads feature.geometry.coordinates;
    // that path does not exist here).
    final currentZoom = controller.getCamera().zoom;
    controller.animateCamera(center: event.point, zoom: currentZoom + 2);
    return;
  }

  final pointHits = controller.featuresAtPoint(
    event.screenPoint,
    layerIds: const ['unclustered-point'],
  );
  if (pointHits.isNotEmpty) {
    final trailId = pointHits.first.properties['id'] as String?;
    if (trailId != null) _selectTrail(trailId); // D-02: existing trailPolylineProvider fetch-then-fit
  }
},
```

### Pattern 3: Multi-trail bounds fit (CORE-08)

**What:** `MapOptions` has no declarative bounds-fit — must call `controller.fitBounds()` imperatively in `onStyleLoaded`, exactly like `WandererMap._fitInitialCamera()` (Phase 15).
**When to use:** `list_detail_map_screen.dart` and `list_detail_screen.dart`'s `_ListMap`.

```dart
// Source: pattern established in wanderer_map.dart's _fitInitialCamera (Phase 15),
// confirmed no init-time bounds field exists in maplibre_platform_interface's MapOptions.
onStyleLoaded: (style) async {
  final bounds = combinedBounds; // existing _combinedBounds(trails) logic, ported as-is
  if (bounds != null) {
    await _controller?.fitBounds(
      bounds: bounds,
      padding: const EdgeInsets.all(40),
      nativeDuration: Duration.zero, // instant on first load, matches today's initialCameraFit
    );
  }
},
```

### Anti-Patterns to Avoid

- **Reusing `WandererMap` as-is for list/map screens:** forces a synthetic/nullable `Trail`, drags in offline composition and single-trail camera-fit logic that doesn't apply. Build the new thin host instead.
- **Rebuilding the cluster `GeoJsonSource` on every pan/zoom:** use `updateGeoJsonSource`, not `removeSource`+`addSource` — the latter also forces re-adding all three style layers (id collisions/flicker risk), the former is a single data swap.
- **Reading `feature.geometry` on a `RenderedFeature`:** does not exist in this package (confirmed — only `id`/`properties`). Any cluster-centering logic that assumes a JS-parity `queryRenderedFeatures` return shape will fail to compile or silently return null.
- **Dropping `mapTrailSearchProvider` in favor of the cluster endpoint for the bottom sheet list:** the cluster endpoint's `attributesToRetrieve: ["id", "_geo", "bounding_box_diagonal"]` cannot power `TrailCard`/`TrailListItem` (needs `name`, `authorName`, `distance`, thumbnails, etc.). Keep both providers.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Multi-polyline rendering (CORE-08) | Manual GeoJSON `LineString` FeatureCollection + `addSource`/`addLayer` | `ml.PolylineLayer(polylines: List<Feature<LineString>>)` | Already ships in the installed `maplibre` package (`lib/src/layer/polyline_layer.dart`), builds its own source/layer per widget instance, matches today's uniform-color flutter_map behavior exactly |
| Cluster/point hit-testing | Manual screen-space distance math against decoded GeoJSON coordinates | `MapController.featuresAtPoint(point, layerIds:)` | Native hit-testing against the actual rendered tiles/features — the same capability web's `queryRenderedFeatures` provides, already exposed by the platform interface |
| "Search this area" gesture-vs-programmatic distinction | Custom pan/zoom delta tracking | `MapEventStartMoveCamera(reason: CameraChangeReason.apiGesture)` | The native event model already tags *why* the camera moved (user gesture vs. developer/API animation) — no need to reimplement flutter_map's `MapEventSource` set-membership check |
| Building the cluster endpoint's filter string | New filter-serialization logic | `TrailFilter.toFilterText(actor:, includeGeo: false)` (existing, in `trail.dart`) | Already produces exactly the raw Meilisearch filter string format the cluster endpoint's `filterText` field expects — `map_trail_search_provider.dart` already calls it with `includeGeo: false` for the same reason (bbox is passed separately) |

**Key insight:** every piece of this phase's "hard part" (cluster hit-testing, GeoJSON source updates, filter serialization) already has a sanctioned helper either in the installed `maplibre` package or in this codebase's existing map-search provider — the work is wiring, not invention.

## Common Pitfalls

### Pitfall 1: `RenderedFeature` has no `geometry` field
**What goes wrong:** Code ported too literally from `cluster-layer.ts`'s `zoomOnCluster` tries to read `feature.geometry.coordinates` off a `featuresAtPoint()` result.
**Why it happens:** The Dart `RenderedFeature` class (confirmed in `maplibre_platform_interface-0.3.5/lib/src/queried_layer.dart`) only carries `id` and `properties` — a deliberately smaller surface than mapbox-gl-js's `queryRenderedFeatures` return shape.
**How to avoid:** Use `MapEventClick.point` (the click's own `Geographic` coordinate, always available) as the `animateCamera` center instead. Functionally equivalent for "zoom toward the tapped cluster."
**Warning signs:** Compile error (`.geometry` undefined on `RenderedFeature`) — caught immediately, not a runtime surprise.

### Pitfall 2: Cluster endpoint response can't power the bottom sheet or selection card
**What goes wrong:** Assuming the cluster `FeatureCollection`'s `properties` are enough to render `TrailCard`/`TrailListItem` (name, distance, thumbnail, author).
**Why it happens:** `POST /search/trails/cluster`'s Meilisearch query explicitly limits `attributesToRetrieve` to `["id", "_geo", "bounding_box_diagonal"]` for performance (it can return up to 10,000 hits pre-clustering) — this is deliberate, not an oversight.
**How to avoid:** Keep `mapTrailSearchProvider` (full `/search/trails` attributes) running alongside the new cluster provider, both keyed to the same bounds. Resolve tapped-trail metadata and the sheet's trail list from it, exactly as today's code does (`trails.firstWhere((t) => t.id == trailId)`).
**Warning signs:** Bottom sheet renders trails with blank names/no thumbnails, or the selection card crashes on a null field.

### Pitfall 3: No declarative "fit to bounds on load" in `MapOptions`
**What goes wrong:** Looking for a `MapOptions.initialCameraFit` or similar (flutter_map's `CameraFit.bounds` passed at `MapOptions` construction) and not finding one.
**Why it happens:** `maplibre_platform_interface`'s `MapOptions` only exposes `initCenter`/`initZoom` (confirmed by reading `map_options.dart` in full) — there is no bounds-fit-at-init field.
**How to avoid:** Fit bounds imperatively inside `onStyleLoaded`, exactly as `WandererMap._fitInitialCamera()` already does — this phase's list-map screens need the identical pattern, just against `combinedBounds` instead of `trail.bounds`.
**Warning signs:** Map opens centered on `Geographic(lat: 0, lon: 0)` at the default zoom (today's fallback value used at `initCenter`) instead of fitted to the list's trails.

### Pitfall 4: Event model mismatch breaks the "Search this area" button reveal logic
**What goes wrong:** Porting flutter_map's `onMapEvent: (event) { if (event is MapEventMoveEnd) { if (userGestures.contains(event.source)) ... } }` literally — `MapEventMoveEnd` and `MapEventSource` don't exist in `maplibre_platform_interface`.
**Why it happens:** maplibre's native event model is gesture-*start*-based (`MapEventStartMoveCamera(reason: CameraChangeReason)`) plus a separate `MapEventCameraIdle`, not a single move-end event carrying a gesture-source enum.
**How to avoid:** Reveal the button on `MapEventStartMoveCamera` where `reason == CameraChangeReason.apiGesture` (mirrors "user-initiated, not programmatic"); read the actual bounds fresh from `controller.getVisibleRegion()` at button-tap time (already the pattern for reading state on demand, not on every move event) rather than trying to capture it at move-end.
**Warning signs:** Button never appears (dead code path checking for a nonexistent event type), or `flutter analyze` immediately flags `MapEventMoveEnd`/`MapEventSource` as undefined identifiers.

### Pitfall 5: `GeoJsonSource.data` is a `String`, not a typed FeatureCollection object
**What goes wrong:** Passing the parsed `Map<String, dynamic>` FeatureCollection (e.g. from `Dio`'s `response.data`) directly to `GeoJsonSource(data: ...)`.
**Why it happens:** `GeoJsonSource.data` is typed `final String data` (confirmed in `geo_json_source.dart`) — a JSON string or URL, matching the JS style spec's source-data contract, not a Dart object.
**How to avoid:** `jsonEncode()` the response body (or re-encode the raw response string if Dio already parsed it) before constructing `GeoJsonSource`/calling `updateGeoJsonSource`.
**Warning signs:** Type error at the call site, or (if using `dynamic`) a silently empty/invalid source with no rendered features.

### Pitfall 6: Longitude wrap-around at the antimeridian
**What goes wrong:** Sending raw `bounds.longitudeEast`/`longitudeWest` straight into the cluster request body when the visible region crosses ±180°.
**Why it happens:** The existing `map_trail_search_provider.dart` already had to solve this for `/search/trails` (`wrapLng` helper) — the same bbox geometry issue applies identically to `/search/trails/cluster`'s `southWest`/`northEast` params, and the server route (`+server.ts`) has its own `southWest.lng > northEast.lng` branch specifically to handle a wrapped request.
**How to avoid:** Reuse (or literally share) the existing `wrapLng` logic when building the new cluster provider's request body.
**Warning signs:** Empty or wildly wrong cluster results near the antimeridian (rare in practice for most deployments, but a silent correctness gap if skipped).

## Code Examples

### Cluster search request body (matches the server route's exact expected shape)

```dart
// Source: web/src/routes/api/v1/search/trails/cluster/+server.ts (read in full) —
// confirms southWest/northEast are {lat, lng} objects (NOT the app's LngLatBounds field
// names), zoom is a plain number, filterText is a raw Meilisearch filter string.
double _wrapLng(double lng) => ((((lng + 180) % 360) + 360) % 360) - 180;

Future<Map<String, dynamic>> _buildClusterBody(
  ml.LngLatBounds bounds,
  double zoom,
  String filterText,
) async {
  return {
    'southWest': {
      'lat': bounds.latitudeSouth,
      'lng': _wrapLng(bounds.longitudeWest),
    },
    'northEast': {
      'lat': bounds.latitudeNorth,
      'lng': _wrapLng(bounds.longitudeEast),
    },
    'zoom': zoom,
    'filterText': filterText, // TrailFilter.toFilterText(actor:, includeGeo:false)
    'q': '',
  };
}
```

### GeoJsonSource update on re-search (CLUS-04)

```dart
// Source: maplibre_platform_interface-0.3.5 style_controller.dart —
// updateGeoJsonSource(id:, data:) exists specifically to avoid remove+re-add churn.
Future<void> _applyClusterResult(
  ml.StyleController style,
  Map<String, dynamic> featureCollection,
) async {
  await style.updateGeoJsonSource(
    id: kClusterSourceId,
    data: jsonEncode(featureCollection),
  );
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Client-side clustering via `flutter_map_marker_cluster` over `/search/trails` (up to 100 hits, `hitsPerPage: 100`) | Server-side Supercluster via `POST /search/trails/cluster` (up to 10,000 hits pre-clustering) rendered as native circle/symbol layers | This phase (CLUS-01/02) | Clustering math moves off the UI isolate; the app can represent far more trails per viewport without a client-side re-cluster cost on every frame |
| Per-marker category icon on the map screen (colored circle + `trailCategoryIcon` glyph) | Plain circle, no category icon — verbatim parity with web's `cluster-layer.ts` `unclustered-point` layer | This phase, by the locked verbatim-port decision | Observable UX change on `map_screen` only: individual trail markers lose their category glyph. `list_detail_map_screen`/`list_detail_screen` (CORE-08) are unaffected — they keep custom widget markers with category icons, since CORE-08 carries no clustering requirement |
| `flutter_map`'s `MapEventMoveEnd` + `MapEventSource` set-membership check for "was this a user drag" | `MapEventStartMoveCamera(reason: CameraChangeReason.apiGesture)` | This phase, forced by the engine swap | Different event shape, same intent — see Pitfall 4 |

**Deprecated/outdated:**
- `flutter_map_marker_cluster`'s `MarkerClusterLayerWidget` on `map_screen`: replaced by native `CircleStyleLayer`/`SymbolStyleLayer`. Package removal itself is deferred to Phase 18 (CLEAN-01) since `navigation_screen` still needs other `flutter_map` plugins until Phase 17.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | A new lightweight, trail-agnostic MapLibreMap host (not `WandererMap`) is the right shape for CORE-08 + `map_screen` | Architecture Patterns, Recommended Project Structure | Low — this was explicitly left to research/planning discretion in 16-CONTEXT.md; if the planner disagrees, the alternative (extending `WandererMap` with an optional `Trail?`) is a mechanical refactor, not a redesign |
| A2 | `MapEventClick.point` is an acceptable substitute for the tapped cluster feature's centroid, given `RenderedFeature` exposes no geometry | Pattern 2, Pitfall 1 | Low-medium — verified by reading the installed package source (`queried_layer.dart`), not by running the app; on-device testing during execution should confirm the visual "zoom toward the cluster" feel matches web's exact-centroid behavior closely enough. Flagged for verification, not blocking. |
| A3 | `mapClusterSearchProvider` should be a new provider alongside `mapTrailSearchProvider`, not a replacement | Summary, Architecture Patterns, Pitfall 2 | Medium if wrong — this was the exact question 16-CONTEXT.md's additional_context flagged as undetermined; the evidence (cluster endpoint's minimal `attributesToRetrieve`) is strong but the planner should confirm this design during plan review, not treat it as unquestionable |

## Open Questions

1. **Exact debounce duration/trigger semantics for the new `mapClusterSearchProvider`**
   - What we know: D-01 locks the trigger to the manual "Search this area" button only (no auto pan/zoom re-query); the existing `mapTrailSearchProvider` still has a 400ms `Timer`-based debounce even though its own triggers are similarly manual-ish (button + initial load + external nav).
   - What's unclear: whether the new cluster provider needs its own debounce at all, given every call site is already a single discrete user action (button tap), not a stream of rapid events.
   - Recommendation: Mirror `mapTrailSearchProvider`'s existing debounce shape for consistency/low risk, even though it may be effectively a no-op given D-01's trigger model — cheap insurance against a future auto-debounce feature (explicitly deferred, not rejected forever) needing the scaffolding already in place.

2. **Whether `list_detail_screen`'s inline `_ListMap` needs the same tap-to-select behavior as `list_detail_map_screen`, or stays fully non-interactive**
   - What we know: today's `_ListMap` uses `InteractiveFlag.none` (fully static thumbnail, tap anywhere navigates to the full map screen) while `list_detail_map_screen` supports per-marker tap-to-select-and-fit.
   - What's unclear: CORE-08's requirement text doesn't distinguish the two screens' interactivity; CONTEXT.md doesn't discuss it either.
   - Recommendation: Preserve today's split exactly (inline map non-interactive/tap-through-to-full-map; full map screen interactive) — this is a straightforward port, not a design decision, and matches the "no UX change" spirit of the rest of this phase's decisions (D-01, D-02).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| `maplibre` (Flutter) | All of CORE-08, CLUS-01..05 | ✓ | 0.3.5 (locked) | — |
| `POST /search/trails/cluster` (backend) | CLUS-01..05 | ✓ | Already implemented, no changes needed | — |
| Flutter SDK | Build | ✓ | Project requires ^3.11.5, matches CLAUDE.md | — |

No missing dependencies. This phase is pure application code against already-available infrastructure.

## Security Domain

`security_enforcement` is `true` (ASVS level 1) per `.planning/config.json`. This phase's surface area is minimal from a security standpoint — it renders already-authenticated API responses on a map and re-uses existing filter-serialization and auth-token plumbing (`apiProvider`, `authProvider`) unchanged.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|-------------------|
| V2 Authentication | No | No new auth surface; requests go through the existing `apiProvider` (already authenticated) |
| V3 Session Management | No | Unchanged |
| V4 Access Control | Indirect (yes) | Category/subcategory preference filtering (CLUS-05) is enforced server-side by `withTrailPreferenceMeiliFilter` — already implemented, not touched by this phase. The app's only obligation is to keep sending the authenticated request (it already does via `apiProvider`); it must not attempt to bypass this by hand-building a request without the auth interceptor. |
| V5 Input Validation | Yes | The cluster endpoint's response (`GeoJSON FeatureCollection`) is server-controlled, but must still be `jsonDecode`d defensively (guard against malformed/partial responses) before being handed to `GeoJsonSource`/`updateGeoJsonSource` — same discipline as `offline_style_rewriter.dart`'s existing `try/catch` around `jsonDecode` in `WandererMap._composeStyle`. |
| V6 Cryptography | No | Not applicable |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Malformed/oversized cluster response causing a client-side parse crash | Denial of Service (client) | Wrap `jsonDecode`/`GeoJsonSource` construction in the same defensive `try/catch` pattern already used in `WandererMap._composeStyle` (log + fail soft, not crash) |
| GeoJSON `properties.id` used unsanitized as a trail lookup key | Tampering (low risk — server-controlled response) | The `id` originates from the same Meilisearch-backed `id` field `/search/trails` already trusts; no new injection surface, but still treat it as untrusted input when used to index into `mapTrailSearchProvider`'s results (`firstWhereOrNull`, not `firstWhere`, to avoid a crash on an unexpected id) |

## Sources

### Primary (HIGH confidence)

- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre-0.3.5/lib/maplibre.dart` — barrel export confirming `CircleStyleLayer`, `SymbolStyleLayer`, `GeoJsonSource`, `MapController`, `StyleController`, `RenderedFeature`, `QueriedLayer`, `MapGestures`, `MapEvent*` types are all present in the installed package
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/map_controller.dart` — `featuresAtPoint`, `queryLayers`, `fitBounds`, `animateCamera`, `getVisibleRegion`, `getCamera` signatures read in full
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/queried_layer.dart` — confirms `RenderedFeature` has no `geometry` field (only `id`/`properties`)
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/style_controller.dart` — confirms `updateGeoJsonSource`, `addSource`, `addLayer` signatures
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/style/layers/circle_style_layer.dart`, `symbol_style_layer.dart`, `style/sources/geo_json_source.dart`, `style/layers/style_layer.dart` — confirms `filter`/`paint`/`layout` typing (`List<Object>?`/`Map<String, Object>`) and `GeoJsonSource.data: String`
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/options/map_options.dart` — confirms no declarative bounds-fit-at-init field exists
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/map_events.dart` — full event catalog, confirms `MapEventStartMoveCamera(reason: CameraChangeReason)` replaces flutter_map's `MapEventMoveEnd`/`MapEventSource`
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre-0.3.5/lib/src/layer/polyline_layer.dart`, `circle_layer.dart` — confirms `ml.PolylineLayer`/`ml.CircleLayer` convenience widgets and their generated-source/layer behavior
- `web/src/lib/vendor/maplibre-layer-manager/cluster-layer.ts` — the exact verbatim port target (CONTEXT.md canonical ref), read in full
- `web/src/routes/api/v1/search/trails/cluster/+server.ts` — the actual backend route, read in full: confirms request body shape (`southWest`/`northEast`/`zoom`/`filterText`/`q`), response shape (`FeatureCollection` with `point_count`/`point_count_abbreviated`/`is_large`/`totalHits`), and that `attributesToRetrieve` is limited to `["id", "_geo", "bounding_box_diagonal"]`
- `app/lib/components/base/wanderer_map.dart`, `app/lib/components/map/trail_layer.dart`, `app/lib/provider/map_style_json_provider.dart` — Phase 15's established patterns, read in full
- `app/lib/routes/map_screen.dart`, `app/lib/routes/list_detail_map_screen.dart`, `app/lib/routes/list_detail_screen.dart`, `app/lib/provider/trail/map_trail_search_provider.dart` — current implementations being ported, read in full
- `app/pubspec.lock` — confirms installed `maplibre` version (0.3.5) directly

### Secondary (MEDIUM confidence)

None used — all technical claims in this research were verified directly against installed package source or the actual codebase, not via web search.

### Tertiary (LOW confidence)

None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages; existing pinned version confirmed via `pubspec.lock`
- Architecture: HIGH — every API surface cited (`featuresAtPoint`, `updateGeoJsonSource`, `GeoJsonSource.data`, `MapOptions`, `MapEvent*`, `PolylineLayer`) was read directly from the installed package source, not inferred from training data
- Pitfalls: HIGH — each pitfall traces to a specific, quoted source file (either the installed package or the actual backend route)

**Research date:** 2026-07-09
**Valid until:** 30 days (stable — no new dependencies, `maplibre` already pinned by lockfile; re-verify only if `pubspec.yaml`'s `maplibre` range is bumped before this phase executes)
