import 'dart:async';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/util/trail/offline_filter_bounds.dart';

// ---------------------------------------------------------------------------
// Regression harness: a non-connection failure (a 500, not a dropped
// connection) from `/trail/filter` must still leave `resetFilter()` safe to
// call. Mirrors test/provider/trail_filter_provider_test.dart's
// ProviderContainer + stubbed apiProvider pattern.
// ---------------------------------------------------------------------------

/// Always answers `/trail/filter` with a 500 `Response` -- a genuine server
/// failure, NOT a `DioExceptionType.connectionError` -- so `build()` takes
/// the "every other error" branch and rethrows without ever assigning
/// `defaultFilter` from inside `build()`.
class _ServerErrorAdapter implements HttpClientAdapter {
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    return ResponseBody.fromString('Internal Server Error', 500);
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

/// Never reaches a server -- throws a genuine `DioExceptionType.
/// connectionError`, so `isConnectionFailure` returns true and `build()`
/// must take the offline-fallback branch (`AsyncData`, never `AsyncError`).
class _ConnectionErrorAdapter implements HttpClientAdapter {
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    throw DioException.connectionError(
      requestOptions: options,
      reason: 'simulated offline',
    );
  }

  @override
  void close({bool force = false}) {}
}

// ---------------------------------------------------------------------------
// Pure-input tests for trail_filter_provider.dart's fallback filter and
// retry policy. No widget pump, no ProviderScope, no Store. Imports
// computeOfflineTrailFilterValues/kOfflineTrailFilterValues from
// util/trail/offline_filter_bounds.dart to build the `computed` fixture, so this
// file and offline_trail_filter_bounds_test.dart stay in step by
// construction rather than by duplicated literals.
// ---------------------------------------------------------------------------

void main() {
  // (15000, 500, 250) -- differs from kOfflineTrailFilterValues on all three
  // axes.
  final computed = computeOfflineTrailFilterValues((
    distances: [12345.0],
    elevationGains: [321.0],
    elevationLosses: [98.0],
  ));

  // A server-shaped fixture, matching the payload
  // test/provider/trail_filter_provider_test.dart already stubs.
  const server = TrailFilterValues(
    minDistance: 0,
    maxDistance: 42000,
    minElevationGain: 0,
    maxElevationGain: 3000,
    minElevationLoss: 0,
    maxElevationLoss: 2500,
    minDuration: 0,
    maxDuration: 86400,
  );

  group('buildDefaultTrailFilter', () {
    test('max == limit on every axis, proven with COMPUTED values -- this '
        'is the property the entire spinner fix rests on', () {
      final filter = buildDefaultTrailFilter(computed);

      expect(filter.distanceMax, filter.distanceLimit);
      expect(filter.elevationGainMax, filter.elevationGainLimit);
      expect(filter.elevationLossMax, filter.elevationLossLimit);
    });

    test('max == limit also holds for the constant', () {
      final filter = buildDefaultTrailFilter(kOfflineTrailFilterValues);

      expect(filter.distanceMax, filter.distanceLimit);
      expect(filter.elevationGainMax, filter.elevationGainLimit);
      expect(filter.elevationLossMax, filter.elevationLossLimit);
    });

    test('the invariant observed through filter text: computed and server '
        'bounds produce byte-identical filter text', () {
      // Guard against silently degenerating: prove the equality across
      // three genuinely different bound triples, not three copies of one.
      expect(computed, isNot(kOfflineTrailFilterValues));
      expect(computed.maxDistance, isNot(server.maxDistance));

      expect(
        buildDefaultTrailFilter(computed).toFilterText(),
        buildDefaultTrailFilter(server).toFilterText(),
      );
    });

    test('the fallback never narrows a search', () {
      final text = buildDefaultTrailFilter(computed).toFilterText();

      expect(text, contains('distance >= 0'));
      expect(text, isNot(contains('distance <=')));
      expect(text, isNot(contains('elevation_gain <=')));
      expect(text, isNot(contains('elevation_loss <=')));
    });

    test('empty-store slider bounds are 100 km / 5000 m / 5000 m -- the '
        'user-visible half. On a device with no trails, these are the max: '
        'of every filter sheet slider app-wide while offline.', () {
      final filter = buildDefaultTrailFilter(kOfflineTrailFilterValues);

      expect(filter.distanceLimit, 100000);
      expect(filter.elevationGainLimit, 5000);
      expect(filter.elevationLossLimit, 5000);
    });
  });

  group('trailFilterRetry', () {
    test('retries the first two attempts', () {
      expect(trailFilterRetry(0, Exception('x')), isNotNull);
      expect(trailFilterRetry(1, Exception('x')), isNotNull);
    });

    test('exhausts the retry budget after two attempts', () {
      expect(trailFilterRetry(2, Exception('x')), isNull);
    });
  });

  group('ResetFilter after a non-connection build failure', () {
    late ProviderContainer container;

    setUp(() {
      final dio = Dio(BaseOptions(baseUrl: 'https://example.test/api/v1'))
        ..httpClientAdapter = _ServerErrorAdapter();
      container = ProviderContainer(
        overrides: [apiProvider.overrideWith(() => _StubApi(dio))],
      );
    });

    tearDown(() => container.dispose());

    test(
      'resetFilter does not throw after a 500 (not a connection failure), '
      'and leaves the provider holding the offline default',
      () async {
        // The build fails with a real DioException (badResponse), not a
        // connection failure -- so `build()` rethrows without ever
        // assigning `defaultFilter` from inside `build()` itself.
        //
        // Deliberately not `await container.read(provider.future)`: with
        // `trailFilterRetry` configured, `.future` never completes once a
        // retry has occurred (a pre-existing riverpod 3.3.1 behaviour,
        // confirmed independent of this fix). Wait for the retry budget
        // (400ms + 800ms) to exhaust and observe the settled state instead.
        final settled = Completer<void>();
        final sub = container.listen(trailFilterProvider('wr-03'), (
          prev,
          next,
        ) {
          if (!next.isLoading && !settled.isCompleted) settled.complete();
        }, fireImmediately: true);
        await settled.future.timeout(const Duration(seconds: 5));
        sub.close();

        final failedState = container.read(trailFilterProvider('wr-03'));
        expect(failedState, isA<AsyncError<TrailFilter>>());

        // Before the fix, defaultFilter was `late` and unassigned here --
        // calling resetFilter() threw LateInitializationError out of this
        // button callback.
        expect(
          () => container
              .read(trailFilterProvider('wr-03').notifier)
              .resetFilter(),
          returnsNormally,
        );

        final state = container.read(trailFilterProvider('wr-03'));
        expect(state, isA<AsyncData<TrailFilter>>());
        expect(
          state.value,
          buildDefaultTrailFilter(kOfflineTrailFilterValues),
        );
      },
    );
  });

  group('connection-failure fallback (regression guard)', () {
    test(
      'a connectionError build yields AsyncData, never AsyncError',
      () async {
        final dio = Dio(BaseOptions(baseUrl: 'https://example.test/api/v1'))
          ..httpClientAdapter = _ConnectionErrorAdapter();
        final container = ProviderContainer(
          overrides: [apiProvider.overrideWith(() => _StubApi(dio))],
        );
        addTearDown(container.dispose);

        // No retry is ever triggered on this path -- build()'s catch block
        // resolves the connection-failure branch itself, so `.future`
        // completes normally (unlike the group above, which only
        // exercises `.future` after a genuine retry has occurred).
        final value = await container.read(
          trailFilterProvider('wr-03-conn').future,
        );

        final state = container.read(trailFilterProvider('wr-03-conn'));
        expect(state, isA<AsyncData<TrailFilter>>());
        expect(value, buildDefaultTrailFilter(kOfflineTrailFilterValues));
      },
    );
  });
}
