import 'dart:async';

import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/trail_deletion_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';

part 'map_cluster_search_provider.g.dart';

/// Debounced, bounds+zoom-keyed provider hitting `POST /search/trails/cluster`.
/// Companion to [MapTrailSearch] — that provider still powers the
/// bottom-sheet trail list (full attributes); this one only feeds the
/// native cluster circle/count layers (`cluster_layer.dart`), so it returns
/// the server's already-clustered GeoJSON `FeatureCollection` verbatim.
@Riverpod(keepAlive: true)
class MapClusterSearch extends _$MapClusterSearch {
  LngLatBounds? _lastBounds;
  double? _lastZoom;
  Timer? _debounce;

  @override
  FutureOr<Map<String, dynamic>> build({
    String? authorId,
    required String filterId,
  }) async {
    ref.onDispose(() => _debounce?.cancel());

    ref.listen(trailFilterProvider(filterId), (previous, next) {
      if (_lastBounds != null && next.hasValue && !next.isLoading) {
        final currentFilter = next.value;
        if (currentFilter != null) {
          searchInBounds(
            _lastBounds!,
            _lastZoom ?? 0,
            passedFilter: currentFilter,
          );
        }
      }
    });

    ref.listen(trailDeletionsProvider, (previous, next) {
      if (next != null) _removeTrail(next.id);
    });

    return <String, dynamic>{
      'type': 'FeatureCollection',
      'features': <Object>[],
    };
  }

  /// Drops the deleted trail's marker from the feature collection so its
  /// category-icon pin disappears with the sheet entry.
  ///
  /// Only ever matches an unclustered feature: the server stamps `id` on
  /// single-trail (`point_count == 1`) features, and a real cluster carries a
  /// count instead of an id. A trail deleted out of a multi-trail cluster
  /// therefore leaves the circle's count one too high until the next bounds
  /// search — the alternative (decrementing a count we didn't compute) would
  /// desync the marker layer from the server's clustering.
  void _removeTrail(String id) {
    final current = state.value;
    if (current == null) return;
    final features = current['features'];
    if (features is! List) return;

    final remaining = features.where((feature) {
      if (feature is! Map) return true;
      final properties = feature['properties'];
      if (properties is! Map) return true;
      return properties['id'] != id;
    }).toList();
    if (remaining.length == features.length) return;

    state = AsyncData(<String, dynamic>{...current, 'features': remaining});
  }

  void searchInBounds(
    LngLatBounds bounds,
    double zoom, {
    TrailFilter? passedFilter,
  }) {
    _lastBounds = bounds;
    _lastZoom = zoom;
    _debounce?.cancel();
    _debounce = Timer(
      const Duration(milliseconds: 400),
      () => _executeSearch(bounds, zoom, passedFilter: passedFilter),
    );
  }

  Future<void> _executeSearch(
    LngLatBounds bounds,
    double zoom, {
    TrailFilter? passedFilter,
  }) async {
    final TrailFilter filter =
        passedFilter ?? await ref.read(trailFilterProvider(filterId).future);
    final user = await ref.read(authProvider.future);
    final api = ref.read(apiProvider);

    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final baseFilterText = filter.toFilterText(
        actor: user?.actorId ?? "",
        includeGeo: false,
        offlineTrailIds: offlineTrailIdsForMapSearch(
          ref,
          offlineOnly: filter.offlineOnly,
          authorId: authorId,
        ),
      );

      // The cluster endpoint takes `filterText` as a single string (unlike
      // `/search/trails`'s `filter` array), so the author clause must be
      // ` AND `-joined rather than appended as a third array element.
      final filterText = authorId == null
          ? baseFilterText
          : (baseFilterText.isEmpty
                ? 'author = $authorId'
                : '$baseFilterText AND author = $authorId');

      final response = await api.post(
        '/search/trails/cluster',
        data: {
          'southWest': {
            'lat': bounds.latitudeSouth,
            'lng': _wrapLng(bounds.longitudeWest),
          },
          'northEast': {
            'lat': bounds.latitudeNorth,
            'lng': _wrapLng(bounds.longitudeEast),
          },
          'zoom': zoom,
          'filterText': filterText,
          'q': '',
        },
      );

      return response.data as Map<String, dynamic>;
    });
  }

  // Wrap coordinate longitudes to [-180, 180]
  double _wrapLng(double lng) {
    return ((((lng + 180) % 360) + 360) % 360) - 180;
  }
}
