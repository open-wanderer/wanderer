import 'package:flutter_map/flutter_map.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';

part 'map_trail_search_provider.g.dart';

@riverpod
class MapTrailSearch extends _$MapTrailSearch {
  @override
  FutureOr<List<TrailSearchResult>> build() async {
    return [];
  }

  Future<void> searchInBounds(LatLngBounds bounds) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final api = ref.read(apiProvider);
      final filter = await ref.read(trailFilterProvider.future);
      final user = await ref.read(authProvider.future);

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
            'sort': [
              "${filter.sort}:${filter.sortOrder == "+" ? "asc" : "desc"}",
            ],
            'attributesToRetrieve': [
              ...defaultTrailSearchAttributes,
              'polyline',
            ],
            'hitsPerPage': 500,
            'page': 1,
          },
        },
      );

      final List<dynamic> hits = response.data['hits'] ?? [];
      return hits.map((json) => TrailSearchResult.fromJson(json)).toList();
    });
  }
}
