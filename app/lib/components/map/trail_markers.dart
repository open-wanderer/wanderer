import 'dart:convert';
import 'dart:ui' show PlatformDispatcher;

import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/category/category_icon.dart';
import 'package:wanderer/components/map/map_marker_gestures.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/subcategory.dart';

/// Saturation cap on rendered widget markers. Every marker in a
/// [ml.WidgetLayer] costs one JNI screen-projection per native camera frame
/// during a pan, so up to the search page size (100) of them made panning
/// measurably expensive. Beyond this many pins the map is visually saturated
/// anyway; denser areas are already folded into the native cluster circles
/// server-side, dropped trails remain in the result sheet, and zooming in
/// re-clusters. Approved trade-off (2026-08-08).
const _kMaxUnclusteredMarkers = 60;

/// GeoJSON source / symbol layer id for [addTrailMarkerLayer].
const String kTrailMarkerLayerId = 'trail-markers';

/// Logical size of a rendered marker image: the 36px pin plus room for its
/// drop shadow, which a tight 36px image would clip.
const double _kMarkerImageExtent = 48;

/// One unclustered (`point_count == 1`) cluster-endpoint feature joined with
/// its search result and resolved category.
typedef _UnclusteredTrail = ({
  String trailId,
  ml.Geographic point,
  Category? category,
  Subcategory? subcategory,
});

Iterable<_UnclusteredTrail> _unclusteredTrails({
  required List<dynamic> features,
  required List<TrailSearchResult> trails,
  required List<Category> categories,
  required List<Subcategory> subcategories,
}) sync* {
  for (final feature in features) {
    if (feature is! Map) continue;
    final properties = feature['properties'];
    if (properties is! Map) continue;

    final pointCount = properties['point_count'];
    if (pointCount is! num || pointCount.toInt() != 1) continue;
    // is_large not filtered here: full-polyline rendering for is_large
    // trails isn't implemented yet, so every unclustered point still
    // renders as a category-icon marker.

    final trailId = properties['id'];
    if (trailId is! String) continue;
    // The feature's id is untrusted input, so use firstWhereOrNull, not
    // firstWhere, so a missing match skips the marker instead of crashing.
    final trail = trails.firstWhereOrNull((t) => t.id == trailId);
    if (trail == null) continue;

    final geometry = feature['geometry'];
    if (geometry is! Map) continue;
    final coordinates = geometry['coordinates'];
    if (coordinates is! List || coordinates.length < 2) continue;
    final lon = (coordinates[0] as num).toDouble();
    final lat = (coordinates[1] as num).toDouble();

    yield (
      trailId: trailId,
      point: ml.Geographic(lat: lat, lon: lon),
      category: trail.categoryId != null
          ? categories.firstWhereOrNull((c) => c.id == trail.categoryId)
          : null,
      subcategory: trail.subcategoryId != null
          ? subcategories.firstWhereOrNull((s) => s.id == trail.subcategoryId)
          : null,
    );
  }
}

/// The category-icon pin both marker paths draw: as a widget in
/// [buildUnclusteredTrailMarkers], and rasterized into a style image for
/// [addTrailMarkerLayer].
Widget _trailMarkerPin(
  Category? category,
  Subcategory? subcategory,
  Color color,
) {
  return Container(
    width: 36,
    height: 36,
    decoration: BoxDecoration(
      color: color,
      shape: BoxShape.circle,
      border: Border.all(color: Colors.white, width: 2),
      boxShadow: [
        BoxShadow(
          color: Colors.black.withValues(alpha: 0.2),
          blurRadius: 4,
          offset: const Offset(0, 2),
        ),
      ],
    ),
    child: Center(
      child: trailCategoryIcon(
        category,
        subcategory: subcategory,
        color: Colors.white,
      ),
    ),
  );
}

/// Builds the category-icon markers for every unclustered (`point_count ==
/// 1`) feature in a cluster-endpoint `FeatureCollection`, capped at
/// [_kMaxUnclusteredMarkers].
///
/// Shared by `map_screen.dart` and `profile_trail_map_screen.dart` so the two
/// screens' marker rendering cannot silently drift apart.
List<ml.Marker> buildUnclusteredTrailMarkers({
  required List<dynamic> features,
  required List<TrailSearchResult> trails,
  required List<Category> categories,
  required List<Subcategory> subcategories,
  required BuildContext context,
  required void Function(String trailId) onTrailTap,
}) {
  final color = Theme.of(context).primaryColor;
  return [
    for (final t in _unclusteredTrails(
      features: features,
      trails: trails,
      categories: categories,
      subcategories: subcategories,
    ).take(_kMaxUnclusteredMarkers))
      ml.Marker(
        point: t.point,
        size: const Size(36, 36),
        // MapMarkerGestures, not GestureDetector: the marker must stay
        // tappable without stealing the map's pan/pinch.
        child: MapMarkerGestures(
          onTap: () => onTrailTap(t.trailId),
          child: _trailMarkerPin(t.category, t.subcategory, color),
        ),
      ),
  ];
}

/// Native counterpart of [buildUnclusteredTrailMarkers]: the GeoJSON for
/// [addTrailMarkerLayer]'s source, plus the pin widget behind every style
/// image id it references (one per category/subcategory/color combination).
///
/// Widget markers are positioned by Flutter from camera updates that arrive
/// separately from a texture-composited map's frames, so they visibly trail
/// it while panning. A symbol layer moves in the same native frame as the
/// map — and has no per-marker projection cost, hence no cap here.
({String geojson, Map<String, Widget> images}) buildUnclusteredTrailMarkerData({
  required List<dynamic> features,
  required List<TrailSearchResult> trails,
  required List<Category> categories,
  required List<Subcategory> subcategories,
  required Color color,
}) {
  final images = <String, Widget>{};
  final markers = <Map<String, Object?>>[];
  for (final t in _unclusteredTrails(
    features: features,
    trails: trails,
    categories: categories,
    subcategories: subcategories,
  )) {
    final imageId =
        'trail-marker-${color.toARGB32()}-${t.category?.id}-${t.subcategory?.id}';
    images.putIfAbsent(
      imageId,
      () => _trailMarkerPin(t.category, t.subcategory, color),
    );
    markers.add(<String, Object?>{
      'type': 'Feature',
      'properties': <String, Object?>{'id': t.trailId, 'icon': imageId},
      'geometry': <String, Object?>{
        'type': 'Point',
        'coordinates': <double>[t.point.lon, t.point.lat],
      },
    });
  }
  return (
    geojson: jsonEncode(<String, Object?>{
      'type': 'FeatureCollection',
      'features': markers,
    }),
    images: images,
  );
}

/// Adds the trail-marker source and symbol layer, empty. Call from
/// `onStyleLoaded` after [addClusterLayers] so the pins draw over the
/// cluster circles, then feed it via [addTrailMarkerImages] and
/// [updateTrailMarkerSource].
Future<void> addTrailMarkerLayer(ml.StyleController style) async {
  await style.addSource(
    const ml.GeoJsonSource(
      id: kTrailMarkerLayerId,
      data: '{"type":"FeatureCollection","features":[]}',
    ),
  );
  await style.addLayer(
    ml.SymbolStyleLayer(
      id: kTrailMarkerLayerId,
      sourceId: kTrailMarkerLayerId,
      layout: <String, Object>{
        'icon-image': <Object>['get', 'icon'],
        // Images are rendered at device resolution, and both platforms
        // register them as 1x (Android upscales by the device density, iOS
        // takes pixels as points) — scale back to the logical size.
        'icon-size':
            1 / PlatformDispatcher.instance.views.first.devicePixelRatio,
        'icon-allow-overlap': true,
        'icon-ignore-placement': true,
      },
    ),
  );
}

/// Rasterizes and registers [images] (id → pin widget) as style images.
Future<void> addTrailMarkerImages(
  ml.StyleController style,
  Map<String, Widget> images,
) async {
  final ratio = PlatformDispatcher.instance.views.first.devicePixelRatio;
  const logical = Size.square(_kMarkerImageExtent);
  for (final MapEntry(key: id, value: pin) in images.entries) {
    await style.addImageFromWidget(
      id: id,
      widget: Center(child: pin),
      logicalSize: logical,
      imageSize: logical * ratio,
    );
  }
}

/// Swaps the trail-marker source's data in place, like
/// [updateClusterSource].
Future<void> updateTrailMarkerSource(
  ml.StyleController style,
  String geojson,
) => style.updateGeoJsonSource(id: kTrailMarkerLayerId, data: geojson);
