import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';

// ---------------------------------------------------------------------------
// Regression test for `LateInitializationError: Field 'defaultFilter' has
// already been initialized`, thrown on every account switch.
//
// Riverpod keeps ONE Notifier instance alive across rebuilds and only re-runs
// `build()`. `defaultFilter` was `late final` and assigned inside `build()`,
// so the second build threw. An account switch hits this every time because
// `trailFilterProvider` is invalidated from `accountScopedProviders`, but any
// other refresh would have hit it too.
//
// The assertion that matters is simply that a SECOND build on the same
// notifier succeeds.
// ---------------------------------------------------------------------------

const _filterValuesJson = {
  'min_distance': 0.0,
  'max_distance': 42000.0,
  'min_elevation_gain': 0.0,
  'max_elevation_gain': 3000.0,
  'min_elevation_loss': 0.0,
  'max_elevation_loss': 2500.0,
  'min_duration': 0.0,
  'max_duration': 86400.0,
};

/// Serves a canned `/trail/filter` payload so `build()` can complete without a
/// network. Counts calls so the test can prove `build()` really did re-run.
class _StubAdapter implements HttpClientAdapter {
  int requestCount = 0;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requestCount++;
    return ResponseBody.fromString(
      jsonEncode(_filterValuesJson),
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

void main() {
  late _StubAdapter adapter;
  late ProviderContainer container;

  setUp(() {
    adapter = _StubAdapter();
    final dio = Dio(BaseOptions(baseUrl: 'https://example.test/api/v1'))
      ..httpClientAdapter = adapter;

    container = ProviderContainer(
      overrides: [apiProvider.overrideWith(() => _StubApi(dio))],
    );
  });

  tearDown(() => container.dispose());

  test('rebuilding after an invalidate does not throw', () async {
    final first = await container.read(trailFilterProvider('default').future);
    expect(first.distanceLimit, 42000.0);

    // What an account switch does (see `accountScopedProviders`).
    container.invalidate(trailFilterProvider);

    // Before the fix this threw LateInitializationError instead of resolving.
    final second = await container.read(trailFilterProvider('default').future);
    expect(second.distanceLimit, 42000.0);
    expect(
      adapter.requestCount,
      greaterThanOrEqualTo(2),
      reason: 'build() must actually have re-run, or this proves nothing',
    );
  });

  test('survives repeated invalidations', () async {
    for (var i = 0; i < 3; i++) {
      await container.read(trailFilterProvider('default').future);
      container.invalidate(trailFilterProvider);
    }

    await expectLater(
      container.read(trailFilterProvider('default').future),
      completes,
    );
  });
}
