import 'dart:async';

import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';

part 'map_trail_search_provider.g.dart';

@Riverpod(keepAlive: true)
class MapTrailSearch extends _$MapTrailSearch {
  LngLatBounds? _lastBounds;
  Timer? _debounce;

  @override
  FutureOr<List<TrailSearchResult>> build() async {
    ref.onDispose(() => _debounce?.cancel());

    ref.listen(trailFilterProvider('map'), (previous, next) {
      if (_lastBounds != null && next.hasValue && !next.isLoading) {
        final currentFilter = next.value;
        if (currentFilter != null) {
          searchInBounds(_lastBounds!, passedFilter: currentFilter);
        }
      }
    });

    return [];
  }

  void searchInBounds(LngLatBounds bounds, {TrailFilter? passedFilter}) {
    _lastBounds = bounds;
    _debounce?.cancel();
    _debounce = Timer(
      const Duration(milliseconds: 400),
      () => _executeSearch(bounds, passedFilter: passedFilter),
    );
  }

  Future<void> _executeSearch(
    LngLatBounds bounds, {
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

      // Wrap coordinate longitudes to [-180, 180]
      double wrapLng(double lng) {
        return ((((lng + 180) % 360) + 360) % 360) - 180;
      }

      final neLat = bounds.latitudeNorth;
      final neLng = wrapLng(bounds.longitudeEast);
      final swLat = bounds.latitudeSouth;
      final swLng = wrapLng(bounds.longitudeWest);

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
