# Phase 16: List & Map Screens on MapLibre - Pattern Map

**Mapped:** 2026-07-09
**Files analyzed:** 6 (2 new, 4 modified)
**Analogs found:** 6 / 6

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `app/lib/components/base/search_map.dart` (new) | component (map host) | request-response (style load) | `app/lib/components/base/wanderer_map.dart` | role-match (strip trail/offline specifics) |
| `app/lib/components/map/cluster_layer.dart` (new) | component (style-layer builder) | streaming (GeoJSON source updates) | `app/lib/components/map/trail_layer.dart`'s `addTrailTrackLayers` + `web/src/lib/vendor/maplibre-layer-manager/cluster-layer.ts` | exact (web is verbatim port source; trail_layer.dart is the Dart-side structural analog) |
| `app/lib/provider/trail/map_cluster_search_provider.dart` (new) | provider (Riverpod) | request-response | `app/lib/provider/trail/map_trail_search_provider.dart` | exact |
| `app/lib/routes/map_screen.dart` (modified) | route/screen | event-driven + request-response | itself (current flutter_map version, being replaced in place) | exact (self-port) |
| `app/lib/routes/list_detail_map_screen.dart` (modified) | route/screen | CRUD (bounds-fit render) | itself + `wanderer_map.dart`'s `_fitInitialCamera` pattern | exact (self-port + Pattern 3) |
| `app/lib/routes/list_detail_screen.dart` (modified, `_ListMap` widget) | component (inline non-interactive map) | CRUD (bounds-fit render) | itself (`_ListMap`) + `list_detail_map_screen.dart` | exact (self-port) |

## Pattern Assignments

### `app/lib/components/base/search_map.dart` (new component, request-response)

**Analog:** `app/lib/components/base/wanderer_map.dart` (full file read — 333 lines)

This is the file to strip down, not copy wholesale. Take the *style-loading + live-swap* skeleton only; drop everything trail/offline-specific.

**Imports pattern** (lines 1-18 of `wanderer_map.dart`):
```dart
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/provider/map_style_json_provider.dart';
```
(Drop `glyph_sprite_cache_provider`, `offline_style_rewriter`, `GlyphSpriteCachePaths`, `LocationMarkerPosition` — all offline/trail-only.)

**Keep verbatim — style resolve + loading/error passthrough** (lines 104-131):
```dart
final baseAsync = ref.watch(mapStyleJsonProvider);
final baseJson = baseAsync.value;
Object? error = baseAsync.error;
// search_map.dart has no offline branch — baseJson IS the style, no _composeStyle needed
if (baseJson == null) {
  if (error != null) return Center(child: Text(error.toString()));
  return ColoredBox(color: Theme.of(context).colorScheme.surface);
}
return _buildMap(context, baseJson);
```

**Keep verbatim — live theme swap on style change** (lines 94-99, 160-172):
```dart
ref.listen(mapStyleJsonProvider, (_, _) => _swapStyle());
...
void _swapStyle() {
  final controller = _controller;
  if (controller == null) return;
  final json = ref.read(mapStyleJsonProvider).value;
  if (json != null && json != _lastStyleJson) {
    _lastStyleJson = json;
    controller.setStyle(json);
  }
}
```

**Keep verbatim — the `ml.MapLibreMap` widget shape** (lines 180-236), minus the trail-specific `layers:`/`TrailMarkerLayer`/elevation-marker children — replace with a `layers`/`children` param the caller supplies (this is exactly what CONTEXT.md's "no Trail param" note calls for):
```dart
return ml.MapLibreMap(
  options: ml.MapOptions(
    initStyle: styleJson,
    initCenter: widget.initCenter ?? const ml.Geographic(lat: 0, lon: 0),
    initZoom: widget.initZoom ?? 3,
    gestures: widget.disabled ? const ml.MapGestures.none() : const ml.MapGestures.all(),
    androidForegroundLoadColor: Theme.of(context).colorScheme.surface,
  ),
  onMapCreated: (controller) {
    _controller = controller;
    widget.onMapCreated?.call(controller);
  },
  onStyleLoaded: (style) => widget.onStyleLoaded?.call(style),
  onEvent: (event) => widget.onMapEvent?.call(event),
  layers: widget.layers ?? const [],
  children: widget.children ?? const [
    ml.MapScalebar(),
    ml.SourceAttribution(),
  ],
);
```

**Drop entirely:** `_composeStyle` offline branch, `_cacheWarmed`/`glyphSpriteCacheProvider` warm-up, `_buildElevationMarker`, `_buildLocationLayer`, `widget.trail`/`widget.offline`/`widget.showTrail`/`widget.selectedWaypoint` fields, `_fitInitialCamera` (each caller does its own bounds-fit imperatively per RESEARCH.md Pattern 3 — `search_map.dart` itself stays trail/bounds-agnostic, exposing only `onStyleLoaded` for the caller to fit against).

---

### `app/lib/components/map/cluster_layer.dart` (new component, streaming)

**Analog 1 (verbatim style/paint values):** `web/src/lib/vendor/maplibre-layer-manager/cluster-layer.ts` (full file read — 134 lines) — port ONLY the `clusters` circle layer and `cluster-count` symbol layer verbatim per CONTEXT.md D-04/D-05. Do NOT port the `unclustered-point` circle layer — per D-05 that stays a `WidgetLayer` marker (see TrailMarkerLayer analog below).

**Analog 2 (Dart structural pattern — `StyleController.addLayer`/`addSource` sequencing, GeoJSON serialization discipline):** `app/lib/components/map/trail_layer.dart`'s `addTrailTrackLayers` (lines 53-129).

**Source data → source add** (mirrors `trail_layer.dart` lines 76-87, using `jsonEncode`, never string concat):
```dart
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

  // D-05: unclustered-point native circle layer is NOT ported — WidgetLayer marker instead.

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

**Re-query update — do NOT remove/re-add** (RESEARCH.md Pattern 1's `updateGeoJsonSource` note):
```dart
Future<void> updateClusterSource(ml.StyleController style, String geojson) =>
    style.updateGeoJsonSource(id: kClusterSourceId, data: geojson);
```

**Naming convention analog:** `trail_layer.dart`'s `_kTrailArrowImageId = 'trail-arrow'` comment (lines 39-45) documents why custom image/layer ids must be distinct from bare/generic basemap sprite names — no custom image is registered by cluster_layer.dart, but if any is added later, follow the same `'cluster-'`-prefixed convention, not a bare id.

---

### `app/lib/provider/trail/map_cluster_search_provider.dart` (new provider, request-response)

**Analog:** `app/lib/provider/trail/map_trail_search_provider.dart` (full file read — 91 lines). Mirror shape and debounce (per RESEARCH.md Open Question 1 — keep the same 400ms `Timer` debounce shape for consistency even though D-01 makes every call site a discrete button tap).

**Imports pattern** (lines 1-9):
```dart
import 'dart:async';

import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/trail.dart'; // for TrailFilter.toFilterText
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';

part 'map_cluster_search_provider.g.dart';
```

**Provider class shape + debounce** (lines 13-41, copy verbatim structure):
```dart
@Riverpod(keepAlive: true)
class MapClusterSearch extends _$MapClusterSearch {
  LngLatBounds? _lastBounds;
  double? _lastZoom;
  Timer? _debounce;

  @override
  FutureOr<Map<String, dynamic>> build() async {
    ref.onDispose(() => _debounce?.cancel());
    return <String, dynamic>{'type': 'FeatureCollection', 'features': <Object>[]};
  }

  void searchInBounds(LngLatBounds bounds, double zoom) {
    _lastBounds = bounds;
    _lastZoom = zoom;
    _debounce?.cancel();
    _debounce = Timer(
      const Duration(milliseconds: 400),
      () => _executeSearch(bounds, zoom),
    );
  }
  // ...
}
```

**Request body + `wrapLng` — reuse verbatim from `map_trail_search_provider.dart` lines 59-67** (per Pitfall 6, this MUST be duplicated or shared, not reinvented):
```dart
double _wrapLng(double lng) => ((((lng + 180) % 360) + 360) % 360) - 180;

Future<void> _executeSearch(LngLatBounds bounds, double zoom) async {
  final filter = await ref.read(trailFilterProvider('map').future);
  final user = await ref.read(authProvider.future);
  final api = ref.read(apiProvider);

  state = const AsyncLoading();
  state = await AsyncValue.guard(() async {
    final filterText = filter.toFilterText(actor: user?.actorId ?? '', includeGeo: false);
    final response = await api.post('/search/trails/cluster', data: {
      'southWest': {'lat': bounds.latitudeSouth, 'lng': _wrapLng(bounds.longitudeWest)},
      'northEast': {'lat': bounds.latitudeNorth, 'lng': _wrapLng(bounds.longitudeEast)},
      'zoom': zoom,
      'filterText': filterText,
      'q': '',
    });
    return response.data as Map<String, dynamic>; // GeoJSON FeatureCollection
  });
}
```

**Error handling / async guard pattern** — identical to analog (`AsyncValue.guard`, `AsyncLoading()` before the call), no deviation needed.

**Reactive re-filter on filter change** (lines 22-29 of analog) — same `ref.listen(trailFilterProvider('map'), ...)` pattern, re-triggering `searchInBounds` with the last bounds/zoom when filters change.

---

### `app/lib/routes/map_screen.dart` (modified, event-driven + request-response)

**Analog:** itself — current implementation read in full (681 lines), being replaced in place. Key excerpts of what is being ported/replaced:

**REPLACED — flutter_map imports → maplibre imports.** Current (lines 1-31) uses `flutter_map`, `flutter_map_animations`, `flutter_map_location_marker`, `flutter_map_marker_cluster`, `vector_map_tiles`. New version should follow `wanderer_map.dart`'s import shape (`maplibre` as `ml`, no `vector_map_tiles`, no flutter_map_* packages) plus the new `search_map.dart`, `cluster_layer.dart`, `map_cluster_search_provider.dart`.

**KEPT AS-IS — "Search this area" button trigger** (D-01, lines 355-379): the `_searchAreaController`/`_searchAreaScale` `AnimationController` fields (lines 55, 78-87) and the `FilledButton.icon` reveal/tap handler are UI-only and untouched by the engine swap; only the event that *reveals* the button changes (see Pitfall 4 below).

**REPLACED — button reveal trigger.** Current (lines 226-243) uses `onMapEvent: (event) { if (event is MapEventMoveEnd) { ...userGestures.contains(event.source)... _searchAreaController.forward(); } }`. Per RESEARCH.md Pitfall 4, the native `onEvent` replacement is:
```dart
onEvent: (event) {
  if (event is ml.MapEventStartMoveCamera &&
      event.reason == ml.CameraChangeReason.apiGesture) {
    _searchAreaController.forward();
  }
  // ... cluster/unclustered tap handling (Pattern 2, adjusted per D-05 below)
},
```

**KEPT AS-IS — fetch-then-fit camera pattern on trail selection** (D-02's explicit reference, lines 295-324): this exact `trailPolylineProvider(trailId).future`-then-`animateCamera`/`fitBounds` sequence is reused unchanged for the unclustered-point tap handler; only the trigger changes from `MarkerClusterLayerWidget.onMarkerTap` to the `WidgetLayer` marker's own `GestureDetector.onTap` (per D-05's tap-handling consequence note):
```dart
// KEPT VERBATIM (was inside onMarkerTap, lines 295-324) — now inside the
// WidgetLayer marker's onTap for point_count==1 features:
setState(() {
  _selectedTrail = trail;
  _selectedPolyline = null;
});
ref.read(trailPolylineProvider(trailId).future).then((polyline) {
  if (!mounted || _selectedTrail?.id != trailId) return;
  if (polyline != null) {
    controller.animateCamera( // was _animatedMapController.animatedFitCamera
      bounds: ml.LngLatBounds.fromPoints(polyline),
      padding: const EdgeInsets.fromLTRB(40, 56, 40, 248),
      duration: const Duration(milliseconds: 750),
    );
  }
  setState(() => _selectedPolyline = polyline);
});
```

**REPLACED — cluster tap handling.** Current: `MarkerClusterLayerWidget.onMarkerTap` (lines 285-350) does client clustering + tap. New: RESEARCH.md Pattern 2, adjusted per D-05 — only `layerIds: ['clusters']` is queried via `featuresAtPoint`; the `unclustered-point` branch is dropped (unclustered points are now `WidgetLayer` markers with their own `onTap`, not a native layer requiring `featuresAtPoint`):
```dart
onEvent: (event) {
  if (event is! ml.MapEventClick) return;
  final controller = _controller;
  if (controller == null) return;
  final clusterHits = controller.featuresAtPoint(event.screenPoint, layerIds: const ['clusters']);
  if (clusterHits.isNotEmpty) {
    final currentZoom = controller.getCamera().zoom;
    controller.animateCamera(center: event.point, zoom: currentZoom + 2);
  }
  // no unclustered-point featuresAtPoint branch — WidgetLayer marker.onTap handles it directly
},
```

**REPLACED — unclustered marker rendering.** Current `markers` list (lines 172-208) builds flutter_map `Marker`s with `trailCategoryIcon`. New version keeps the exact same `Container`/`trailCategoryIcon` visual (per D-05, category icon preserved) but as an `ml.Marker` inside an `ml.WidgetLayer`, filtered to cluster-response features where `point_count == 1` — direct analog is `TrailMarkerLayer`'s waypoint marker construction in `trail_layer.dart` (lines 166-191, the `GestureDetector`-wrapped `ml.Marker` pattern):
```dart
ml.Marker(
  point: ml.Geographic(lat: trail.geo.lat, lon: trail.geo.lng),
  size: const Size(36, 36),
  child: GestureDetector(
    onTap: () => _selectTrail(trail), // D-02 fetch-then-fit, see above
    child: Container( /* identical decoration to current lines 186-206 */
      decoration: BoxDecoration(
        color: Theme.of(context).primaryColor,
        shape: BoxShape.circle,
        border: Border.all(color: Colors.white, width: 2),
        boxShadow: [BoxShadow(color: Colors.black.withValues(alpha: 0.2), blurRadius: 4, offset: const Offset(0, 2))],
      ),
      child: Center(child: trailCategoryIcon(category, subcategory: subcategory, color: Colors.white)),
    ),
  ),
),
```

**Data source split (Pitfall 2):** keep `mapTrailSearchProvider` (line 22, 167-170) watched exactly as today for the bottom sheet `TrailCard` list AND for resolving tapped-trail metadata (`trails.firstWhereOrNull((t) => t.id == trailId)`, use `firstWhereOrNull` not `firstWhere` per the Known Threat Patterns table); add `mapClusterSearchProvider` watched in parallel to drive `cluster_layer.dart`'s `GeoJsonSource`.

**`is_large` filtering (D-03):** apply `["!=", ["get", "is_large"], true]` server-response-side filter already baked into the ported `clusters`/`cluster-count` layer filters above — no additional client-side filtering needed in `map_screen.dart` itself.

---

### `app/lib/routes/list_detail_map_screen.dart` (modified, CRUD/bounds-fit)

**Analog:** itself — current implementation read in full (255 lines), being replaced in place; and `wanderer_map.dart`'s `_fitInitialCamera` (RESEARCH.md Pattern 3) for the imperative bounds-fit.

**KEPT VERBATIM — `_combinedBounds` logic** (lines 229-253): pure Dart math over `List<Trail>`, no flutter_map/maplibre API surface, ports unchanged:
```dart
ml.LngLatBounds? _combinedBounds(List<Trail> trails) {
  final withBounds = trails.where(
    (t) => t.minLat != 0 || t.maxLat != 0 || t.minLon != 0 || t.maxLon != 0,
  );
  if (withBounds.isEmpty) return null;
  double minLat = double.infinity, maxLat = double.negativeInfinity;
  double minLon = double.infinity, maxLon = double.negativeInfinity;
  for (final t in withBounds) {
    if (t.minLat < minLat) minLat = t.minLat;
    if (t.maxLat > maxLat) maxLat = t.maxLat;
    if (t.minLon < minLon) minLon = t.minLon;
    if (t.maxLon > maxLon) maxLon = t.maxLon;
  }
  return ml.LngLatBounds(longitudeEast: maxLon, longitudeWest: minLon, latitudeNorth: maxLat, latitudeSouth: minLat);
}
```

**REPLACED — declarative `initialCameraFit` → imperative `onStyleLoaded` fit** (current lines 180-185 used flutter_map's `MapOptions.initialCameraFit`; no maplibre equivalent exists per RESEARCH.md Pitfall 3). New pattern, directly from `wanderer_map.dart`'s `_fitInitialCamera` (lines 239-264):
```dart
onStyleLoaded: (style) async {
  final bounds = combinedBounds;
  if (bounds != null) {
    await _controller?.fitBounds(
      bounds: bounds,
      padding: const EdgeInsets.all(40),
      nativeDuration: Duration.zero,
    );
  }
},
```

**REPLACED — `_onMarkerTap`/`_deselect` bounds-fit** (current lines 44-75) — same `CameraFit.bounds`-then-`animatedFitCamera` intent, now via `ml.MapController.fitBounds` with a non-zero animated duration:
```dart
void _onMarkerTap(Trail trail) {
  setState(() => _selectedTrail = trail);
  _controller?.fitBounds(
    bounds: ml.LngLatBounds(longitudeEast: trail.maxLon, longitudeWest: trail.minLon, latitudeNorth: trail.maxLat, latitudeSouth: trail.minLat),
    padding: const EdgeInsets.fromLTRB(40, 56, 40, 120),
    // animate over 750ms — check ml.MapController.fitBounds's animation param name
  );
}
```

**REPLACED — polyline/marker rendering:** current `PolylineLayer(polylines: ...)`/`MarkerLayer(markers: ...)` (lines 199-201) → `ml.PolylineLayer(polylines: List<Feature<LineString>>)` (RESEARCH.md "Don't Hand-Roll" table, `ml.PolylineLayer` already ships in the installed package) + `ml.WidgetLayer` with markers built exactly like current lines 100-146's `trailCategoryIcon`-based `Container`, direct analog `TrailMarkerLayer` in `trail_layer.dart` (lines 145-237) for the `GestureDetector`-wrapped `ml.Marker` shape.

**Host widget:** use the new `search_map.dart` (no `Trail` param needed — this screen shows N trails, not one), NOT `WandererMap`.

---

### `app/lib/routes/list_detail_screen.dart` (modified, `_ListMap` inner widget, CRUD/bounds-fit)

**Analog:** itself — `_ListMap` class read in full (lines 320-436), and `list_detail_map_screen.dart`'s post-port version above (near-identical minus interactivity).

**KEPT VERBATIM — `_combinedBounds`** (lines 411-435): identical duplicate of `list_detail_map_screen.dart`'s version — port unchanged (or, if planner opts to dedupe into a shared util, both ports point to the same source).

**KEPT AS-IS — non-interactive/tap-through behavior** (RESEARCH.md Open Question 2's recommendation to preserve exactly): current `MapOptions.interactionOptions: InteractionOptions(flags: InteractiveFlag.none)` + `onTap: (_, _) => context.push('/list/${list.id}/map')` (lines 382-396). New maplibre equivalent:
```dart
ml.MapGestures.none() // disabled: true on search_map.dart, matches WandererMap.disabled pattern
// tap-through:
onEvent: (event) {
  if (event is ml.MapEventClick) context.push('/list/${list.id}/map');
},
```
No per-marker tap handling needed here (unlike `list_detail_map_screen.dart`) — markers render but are not individually tappable, matching today's `MarkerLayer` (no `GestureDetector` wrapper in the current `_ListMap.build`, lines 347-373 — markers today are plain, non-tappable `Container`s, confirmed by absence of `GestureDetector` in this method vs. its presence in `list_detail_map_screen.dart`'s `_onMarkerTap`-wired markers).

**REPLACED — declarative → imperative bounds-fit-on-init:** identical `onStyleLoaded` pattern as `list_detail_map_screen.dart` above (Pattern 3), applied to `_combinedBounds(trails)`.

**Host widget:** new `search_map.dart`, embedded inline (this widget is already wrapped in a fixed-height `SizedBox`/`ClipRRect` by its caller in `_ListHeader`, lines 262-296 — no change needed there).

---

## Shared Patterns

### Style hosting / live theme swap
**Source:** `app/lib/components/base/wanderer_map.dart` lines 94-131, 160-172
**Apply to:** `search_map.dart` (new host used by all three screens)
```dart
ref.listen(mapStyleJsonProvider, (_, _) => _swapStyle());
final baseAsync = ref.watch(mapStyleJsonProvider);
// ... loading/error passthrough, then controller.setStyle(json) on change
```

### Imperative bounds-fit (no declarative option exists)
**Source:** `app/lib/components/base/wanderer_map.dart` lines 239-264 (`_fitInitialCamera`)
**Apply to:** `list_detail_map_screen.dart`, `list_detail_screen.dart`'s `_ListMap`, and (for polyline-fit-after-tap) `map_screen.dart`
```dart
onStyleLoaded: (style) async {
  await controller.fitBounds(bounds: ..., padding: ..., nativeDuration: Duration.zero);
},
```

### Category-icon marker construction (WidgetLayer)
**Source:** `app/lib/components/map/trail_layer.dart`'s `TrailMarkerLayer.build` (lines 166-191) and `_buildCircularMarker` (lines 242-268); also the current `trailCategoryIcon`-based `Container` in `map_screen.dart` (lines 181-208) and `list_detail_map_screen.dart` (lines 111-144)
**Apply to:** unclustered-point markers in `map_screen.dart` (D-05), and all markers in `list_detail_map_screen.dart`/`list_detail_screen.dart`'s `_ListMap`
```dart
ml.Marker(
  point: ml.Geographic(lat: ..., lon: ...),
  size: const Size(36, 36),
  child: GestureDetector(
    onTap: () => ...,
    child: Container(
      decoration: BoxDecoration(color: ..., shape: BoxShape.circle, border: Border.all(color: Colors.white, width: 2), boxShadow: [...]),
      child: Center(child: trailCategoryIcon(category, subcategory: subcategory, color: Colors.white)),
    ),
  ),
)
```

### GeoJSON source discipline (jsonEncode, never string concat)
**Source:** `app/lib/components/map/trail_layer.dart` lines 76-87
**Apply to:** `cluster_layer.dart`, `map_cluster_search_provider.dart`
```dart
final data = jsonEncode(<String, Object?>{'type': 'Feature', ...});
await style.addSource(ml.GeoJsonSource(id: '...', data: data));
```

### Debounced bounds-keyed search provider
**Source:** `app/lib/provider/trail/map_trail_search_provider.dart` (full file, 91 lines)
**Apply to:** `map_cluster_search_provider.dart`
```dart
@Riverpod(keepAlive: true)
class X extends _$X {
  LngLatBounds? _lastBounds;
  Timer? _debounce;
  void searchInBounds(...) { _debounce?.cancel(); _debounce = Timer(const Duration(milliseconds: 400), () => _executeSearch(...)); }
  Future<void> _executeSearch(...) async { state = const AsyncLoading(); state = await AsyncValue.guard(() async { ... }); }
}
```

### Longitude wrap-around helper
**Source:** `app/lib/provider/trail/map_trail_search_provider.dart` lines 60-62
**Apply to:** `map_cluster_search_provider.dart` (Pitfall 6 — must not be skipped)
```dart
double wrapLng(double lng) => ((((lng + 180) % 360) + 360) % 360) - 180;
```

## No Analog Found

None — all 6 files have strong same-repo analogs (self-ports of the current flutter_map version, plus Phase 15's `wanderer_map.dart`/`trail_layer.dart` for the new MapLibre-native pieces, plus the web `cluster-layer.ts` for the verbatim style-layer port).

## Metadata

**Analog search scope:** `app/lib/routes/`, `app/lib/components/base/`, `app/lib/components/map/`, `app/lib/provider/trail/`, `app/lib/util/`, `web/src/lib/vendor/maplibre-layer-manager/`
**Files scanned:** 8 (map_screen.dart, list_detail_map_screen.dart, list_detail_screen.dart, wanderer_map.dart, trail_layer.dart, map_trail_search_provider.dart, category_icon_util.dart, cluster-layer.ts)
**Pattern extraction date:** 2026-07-09
