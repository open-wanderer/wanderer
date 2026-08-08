import 'dart:async';

import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/trail_deletion_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';

part 'map_trail_search_provider.g.dart';

@Riverpod(keepAlive: true)
class MapTrailSearch extends _$MapTrailSearch {
  LngLatBounds? _lastBounds;
  Timer? _debounce;

  @override
  FutureOr<List<TrailSearchResult>> build({
    String? authorId,
    required String filterId,
  }) async {
    ref.onDispose(() => _debounce?.cancel());

    ref.listen(trailFilterProvider(filterId), (previous, next) {
      if (_lastBounds != null && next.hasValue && !next.isLoading) {
        final currentFilter = next.value;
        if (currentFilter != null) {
          searchInBounds(_lastBounds!, passedFilter: currentFilter);
        }
      }
    });

    // A deleted trail must leave the bottom-sheet list immediately rather than
    // waiting for the next bounds search: the user is popped straight back
    // onto the map from the detail screen without moving the camera, so
    // nothing would re-run the search. Re-running it wouldn't be reliable
    // anyway — Meilisearch's index drops the document asynchronously, so a
    // search fired right after the DELETE can still return the hit.
    ref.listen(trailDeletionsProvider, (previous, next) {
      if (next != null) _removeTrail(next.id);
    });

    return [];
  }

  void _removeTrail(String id) {
    final current = state.value;
    if (current == null) return;
    final remaining = current.where((t) => t.id != id).toList();
    if (remaining.length == current.length) return;
    // Written as AsyncData, not `state = AsyncData(...)` on a loading state:
    // if a bounds search is in flight it will overwrite this shortly anyway,
    // and the deleted trail is gone from the server by then.
    state = AsyncData(remaining);
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
        passedFilter ?? await ref.read(trailFilterProvider(filterId).future);
    final user = await ref.read(authProvider.future);
    final api = ref.read(apiProvider);

    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final filterText = filter.toFilterText(
        actor: user?.actorId ?? "",
        includeGeo: false,
        offlineTrailIds: offlineTrailIdsForMapSearch(
          ref,
          offlineOnly: filter.offlineOnly,
          authorId: authorId,
        ),
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
              if (authorId != null) 'author = $authorId',
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
