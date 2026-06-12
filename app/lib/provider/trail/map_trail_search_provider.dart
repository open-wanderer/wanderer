import 'dart:async';

import 'package:flutter_map/flutter_map.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';

part 'map_trail_search_provider.g.dart';

@riverpod
class MapTrailSearch extends _$MapTrailSearch {
  LatLngBounds? _lastBounds;
  Timer? _debounce;

  @override
  FutureOr<List<TrailSearchResult>> build() async {
    ref.onDispose(() => _debounce?.cancel());

    ref.listen(trailFilterProvider, (previous, next) {
      if (_lastBounds != null && next.hasValue && !next.isLoading) {
        final currentFilter = next.value;
        if (currentFilter != null) {
          searchInBounds(_lastBounds!, passedFilter: currentFilter);
        }
      }
    });

    return [];
  }

  void searchInBounds(LatLngBounds bounds, {TrailFilter? passedFilter}) {
    _lastBounds = bounds;
    _debounce?.cancel();
    _debounce = Timer(
      const Duration(milliseconds: 400),
      () => _executeSearch(bounds, passedFilter: passedFilter),
    );
  }

  Future<void> _executeSearch(
    LatLngBounds bounds, {
    TrailFilter? passedFilter,
  }) async {
    final TrailFilter filter =
        passedFilter ?? await ref.read(trailFilterProvider.future);
    final user = await ref.read(authProvider.future);
    final api = ref.read(apiProvider);

    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final filterText = filter.toFilterText(
        actor: user?.actorId ?? "",
        includeGeo: false,
      );

      // Wrap coordinate longitudes to [-180, 180]
      double wrapLng(double lng) {
        return ((((lng + 180) % 360) + 360) % 360) - 180;
      }

      final neLat = bounds.northEast.latitude;
      final neLng = wrapLng(bounds.northEast.longitude);
      final swLat = bounds.southWest.latitude;
      final swLng = wrapLng(bounds.southWest.longitude);

      final response = await api.post(
        '/search/trails',
        data: {
          'q': '',
          'options': {
            'filter': [
              '_geoBoundingBox([$neLat, $neLng], [$swLat, $swLng])',
              if (filterText.isNotEmpty) filterText,
            ],
            'sort': ["${filter.sort.name}:${filter.sortOrder.name}"],
            'attributesToRetrieve': [...defaultTrailSearchAttributes],
            'hitsPerPage': 100,
            'page': 1,
          },
        },
      );

      final List<dynamic> hits = response.data['hits'] ?? [];
      return hits.map((json) => TrailSearchResult.fromJson(json)).toList();
    });
  }
}
