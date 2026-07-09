import 'dart:async';

import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';

part 'map_cluster_search_provider.g.dart';

/// Debounced, bounds+zoom-keyed provider hitting `POST /search/trails/cluster`
/// (CLUS-01/04/05). Companion to [MapTrailSearch] — that provider still powers
/// the bottom-sheet trail list (full attributes); this one only feeds the
/// native cluster circle/count layers (`cluster_layer.dart`), so it returns
/// the server's already-clustered GeoJSON `FeatureCollection` verbatim.
@Riverpod(keepAlive: true)
class MapClusterSearch extends _$MapClusterSearch {
  LngLatBounds? _lastBounds;
  double? _lastZoom;
  Timer? _debounce;

  @override
  FutureOr<Map<String, dynamic>> build() async {
    ref.onDispose(() => _debounce?.cancel());

    ref.listen(trailFilterProvider('map'), (previous, next) {
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

    return <String, dynamic>{'type': 'FeatureCollection', 'features': <Object>[]};
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
        passedFilter ?? await ref.read(trailFilterProvider('map').future);
    final user = await ref.read(authProvider.future);
    final api = ref.read(apiProvider);

    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final filterText = filter.toFilterText(
        actor: user?.actorId ?? "",
        includeGeo: false,
      );

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
