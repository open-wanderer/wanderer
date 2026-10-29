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
  final markers = <ml.Marker>[];

  for (final feature in features) {
    if (markers.length >= _kMaxUnclusteredMarkers) break;
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

    final Category? category = trail.categoryId != null
        ? categories.firstWhereOrNull((c) => c.id == trail.categoryId)
        : null;
    final subcategory = trail.subcategoryId != null
        ? subcategories.firstWhereOrNull((s) => s.id == trail.subcategoryId)
        : null;

    markers.add(
      ml.Marker(
        point: ml.Geographic(lat: lat, lon: lon),
        size: const Size(36, 36),
        // MapMarkerGestures, not GestureDetector: the marker must stay
        // tappable without stealing the map's pan/pinch.
        child: MapMarkerGestures(
          onTap: () => onTrailTap(trailId),
          child: Container(
            decoration: BoxDecoration(
              color: Theme.of(context).primaryColor,
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
          ),
        ),
      ),
    );
  }

  return markers;
}
