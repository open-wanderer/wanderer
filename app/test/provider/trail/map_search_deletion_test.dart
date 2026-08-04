import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/map_cluster_search_provider.dart';
import 'package:wanderer/provider/trail/map_trail_search_provider.dart';
import 'package:wanderer/provider/trail/trail_deletion_provider.dart';

// ---------------------------------------------------------------------------
// A trail deleted from the detail screen must leave the map's results
// immediately. The map screen is popped back to without a camera move, so
// nothing re-runs the bounds search -- and re-running it would not be a fix
// anyway, since Meilisearch drops the document from its index
// asynchronously and can still return the hit right after the DELETE.
//
// Both map providers are `keepAlive` and hold their last bounds in instance
// fields, so `ref.invalidate` is NOT an option: it resets them to their empty
// build() value and leaves the user staring at an empty list and an empty map
// until they pan far enough to trigger a fresh search. They subscribe to
// `trailDeletionsProvider` and splice the id out of the state they hold.
// ---------------------------------------------------------------------------

const _deletedId = 'trail_deleted';
const _keptId = 'trail_kept';

Map<String, dynamic> _hit(String id) => {
  'id': id,
  'author': 'actor_1',
  'author_name': 'tester',
  'author_avatar': '',
  'name': id,
  'description': '',
  'location': '',
  'distance': 1000.0,
  'elevation_gain': 100.0,
  'elevation_loss': 100.0,
  'duration': 3600.0,
  'difficulty': 0,
  'completed': false,
  'date': 0,
  'created': 0,
  'public': true,
  'thumbnail': '',
  'like_count': 0,
  'gpx': '',
  '_geo': {'lat': 47.0, 'lng': 11.0},
};

Map<String, dynamic> _feature(String id, {int pointCount = 1}) => {
  'type': 'Feature',
  'geometry': {
    'type': 'Point',
    'coordinates': [11.0, 47.0],
  },
  'properties': {'id': id, 'point_count': pointCount},
};

/// A real cluster circle: server-computed count, no trail id to match on.
Map<String, dynamic> _clusterFeature() => {
  'type': 'Feature',
  'geometry': {
    'type': 'Point',
    'coordinates': [11.5, 47.5],
  },
  'properties': {'point_count': 4},
};

/// Serves the three endpoints the two providers' `build()`/search path touch:
/// `/trail/filter` (for the real `trailFilterProvider('map')`),
/// `/search/trails`, and `/search/trails/cluster`.
class _RoutingAdapter implements HttpClientAdapter {
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    final path = options.uri.path;

    if (path.endsWith('/trail/filter')) {
      return ResponseBody.fromString(
        jsonEncode({
          'min_distance': 0.0,
          'max_distance': 42000.0,
          'min_elevation_gain': 0.0,
          'max_elevation_gain': 3000.0,
          'min_elevation_loss': 0.0,
          'max_elevation_loss': 2500.0,
          'min_duration': 0.0,
          'max_duration': 86400.0,
        }),
        200,
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
        },
      );
    }

    if (path.endsWith('/search/trails/cluster')) {
      return ResponseBody.fromString(
        jsonEncode({
          'type': 'FeatureCollection',
          'features': [_feature(_deletedId), _feature(_keptId), _clusterFeature()],
        }),
        200,
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
        },
      );
    }

    if (path.endsWith('/search/trails')) {
      return ResponseBody.fromString(
        jsonEncode({
          'hits': [_hit(_deletedId), _hit(_keptId)],
        }),
        200,
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
        },
      );
    }

    return ResponseBody.fromString('{}', 404);
  }

  @override
  void close({bool force = false}) {}
}

class _StubApi extends Api {
  _StubApi(this._dio);

  final Dio _dio;

  @override
  Dio build() => _dio;
}

/// `authProvider`'s real `build()` needs ObjectBox; the search path only reads
/// `user?.actorId` off it, so a null user is enough.
class _StubAuth extends Auth {
  @override
  Future<UserEntity?> build() async => null;
}

ProviderContainer _makeContainer() {
  final dio = Dio(BaseOptions(baseUrl: 'https://example.test'))
    ..httpClientAdapter = _RoutingAdapter();

  final container = ProviderContainer(
    overrides: [
      apiProvider.overrideWith(() => _StubApi(dio)),
      authProvider.overrideWith(_StubAuth.new),
    ],
  );
  addTearDown(container.dispose);
  return container;
}

/// Both providers debounce `searchInBounds` by 400ms before firing.
Future<void> _settleSearch() =>
    Future<void>.delayed(const Duration(milliseconds: 600));

final _bounds = LngLatBounds(
  longitudeWest: 10.0,
  latitudeSouth: 46.0,
  longitudeEast: 12.0,
  latitudeNorth: 48.0,
);

void main() {
  test('deleted trail leaves the map result list, others stay', () async {
    final container = _makeContainer();

    await container.read(mapTrailSearchProvider.future);
    container.read(mapTrailSearchProvider.notifier).searchInBounds(_bounds);
    await _settleSearch();

    expect(
      container.read(mapTrailSearchProvider).value!.map((t) => t.id),
      containsAll([_deletedId, _keptId]),
      reason: 'search fixture must land before the deletion is announced',
    );

    container.read(trailDeletionsProvider.notifier).announce(_deletedId);

    final ids = container
        .read(mapTrailSearchProvider)
        .value!
        .map((t) => t.id)
        .toList();
    expect(ids, isNot(contains(_deletedId)));
    expect(ids, contains(_keptId));
  });

  test("deleted trail's marker leaves the cluster feature collection", () async {
    final container = _makeContainer();

    await container.read(mapClusterSearchProvider.future);
    container
        .read(mapClusterSearchProvider.notifier)
        .searchInBounds(_bounds, 12);
    await _settleSearch();

    expect(
      (container.read(mapClusterSearchProvider).value!['features'] as List),
      hasLength(3),
      reason: 'cluster fixture must land before the deletion is announced',
    );

    container.read(trailDeletionsProvider.notifier).announce(_deletedId);

    final features =
        container.read(mapClusterSearchProvider).value!['features'] as List;
    final ids = features
        .map((f) => (f as Map)['properties'] as Map)
        .map((p) => p['id'])
        .toList();

    expect(ids, isNot(contains(_deletedId)));
    expect(ids, contains(_keptId));
    // The multi-trail cluster circle survives untouched: it carries a
    // server-computed count and no id, so there is nothing to match against
    // and decrementing a count we did not compute would desync the circle
    // from the server's clustering.
    expect(features, hasLength(2));
  });

  test('an unrelated deletion leaves both providers alone', () async {
    final container = _makeContainer();

    await container.read(mapTrailSearchProvider.future);
    container.read(mapTrailSearchProvider.notifier).searchInBounds(_bounds);
    await _settleSearch();

    container.read(trailDeletionsProvider.notifier).announce('some_other_id');

    expect(container.read(mapTrailSearchProvider).value, hasLength(2));
  });

  test('an empty id is never announced', () async {
    final container = _makeContainer();

    container.read(trailDeletionsProvider.notifier).announce('');

    expect(container.read(trailDeletionsProvider), isNull);
  });

  test('the same id announced twice still notifies listeners', () async {
    final container = _makeContainer();

    final seen = <String>[];
    container.listen(trailDeletionsProvider, (previous, next) {
      if (next != null) seen.add(next.id);
    });

    container.read(trailDeletionsProvider.notifier)
      ..announce(_deletedId)
      ..announce(_deletedId);

    expect(
      seen,
      [_deletedId, _deletedId],
      reason: 'Riverpod skips equal states -- hence TrailDeletion.seq',
    );
  });
}
