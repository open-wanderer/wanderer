import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/trail/profile_trail_bounding_box_provider.dart';

/// Routes every `/trail/bounding-box` request to a canned response chosen by
/// `handle`, or throws when [DioAdapter.throwOnFetch] is set — simulating a
/// network failure the shared Dio client would otherwise surface as a
/// [DioException].
class _StubAdapter implements HttpClientAdapter {
  _StubAdapter(this.responsesByHandle, {this.throwOnFetch = false});

  final Map<String, Map<String, dynamic>> responsesByHandle;
  final bool throwOnFetch;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    if (throwOnFetch) {
      throw DioException(
        requestOptions: options,
        type: DioExceptionType.connectionError,
        error: 'simulated network failure',
      );
    }

    final handle = options.queryParameters['handle'] as String?;
    final body = responsesByHandle[handle];
    if (body == null) {
      return ResponseBody.fromString('{}', 404);
    }

    return ResponseBody.fromString(
      jsonEncode(body),
      200,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
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

ProviderContainer _makeContainer(HttpClientAdapter adapter) {
  final dio = Dio(BaseOptions(baseUrl: 'https://example.test'))
    ..httpClientAdapter = adapter;

  final container = ProviderContainer(
    overrides: [apiProvider.overrideWith(() => _StubApi(dio))],
  );
  addTearDown(container.dispose);
  return container;
}

void main() {
  test(
    'parses a snake_case success payload into the right four doubles',
    () async {
      final container = _makeContainer(
        _StubAdapter({
          'tester': {
            'min_lat': 45.5,
            'max_lat': 47.25,
            'min_lon': 10.0,
            'max_lon': 12.75,
            'has_trails': true,
          },
        }),
      );

      final bbox = await container.read(
        profileTrailBoundingBoxProvider('tester').future,
      );

      expect(bbox.minLat, 45.5);
      expect(bbox.maxLat, 47.25);
      expect(bbox.minLon, 10.0);
      expect(bbox.maxLon, 12.75);
      expect(bbox.hasTrails, isTrue);
    },
  );

  test('a has_trails: false payload yields hasTrails == false', () async {
    final container = _makeContainer(
      _StubAdapter({
        'empty-profile': {
          'min_lat': 0,
          'max_lat': 0,
          'min_lon': 0,
          'max_lon': 0,
          'has_trails': false,
        },
      }),
    );

    final bbox = await container.read(
      profileTrailBoundingBoxProvider('empty-profile').future,
    );

    expect(bbox.hasTrails, isFalse);
  });

  test('whole-degree integer coordinates parse without a TypeError', () async {
    final container = _makeContainer(
      _StubAdapter({
        'integer-coords': {
          'min_lat': 45,
          'max_lat': 47,
          'min_lon': 10,
          'max_lon': 12,
          'has_trails': true,
        },
      }),
    );

    final bbox = await container.read(
      profileTrailBoundingBoxProvider('integer-coords').future,
    );

    expect(bbox.minLat, 45.0);
    expect(bbox.maxLat, 47.0);
    expect(bbox.minLon, 10.0);
    expect(bbox.maxLon, 12.0);
    expect(bbox.hasTrails, isTrue);
  });

  test('a request that throws a DioException resolves to hasTrails == false '
      'and does not put the provider into an error state', () async {
    final container = _makeContainer(
      _StubAdapter(const {}, throwOnFetch: true),
    );

    final bbox = await container.read(
      profileTrailBoundingBoxProvider('unreachable').future,
    );

    expect(bbox.hasTrails, isFalse);
    expect(
      container.read(profileTrailBoundingBoxProvider('unreachable')).hasError,
      isFalse,
    );
  });
}
