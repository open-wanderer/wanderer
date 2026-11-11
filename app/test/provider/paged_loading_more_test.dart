import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/profile/profile_feed_provider.dart';

// ---------------------------------------------------------------------------
// Regression harness for the idle-frame churn on the profile screen.
//
// The feed's trailing `CircularProgressIndicator` was gated on
// `PagedState.hasMore` -- "another page exists" -- rather than on a fetch being
// in flight. On any multi-page profile that mounted the spinner from first
// paint and never unmounted it, and a `CircularProgressIndicator` drives a
// repeating ticker, so it scheduled a frame every vsync forever on a screen
// nobody was touching. `debugPrintScheduleFrameStacks` showed a self-sustaining
// `Ticker._tick -> Ticker.scheduleTick -> scheduleFrame` loop in the
// transientCallbacks phase, with no `markNeedsBuild` anywhere.
//
// The fix is `ProfileFeedState.isLoadingMore`, published by `PagedLoadMore`
// around the fetch. These tests pin both halves: that the flag tracks the
// actual request window, and that the obvious-looking alternative
// (`AsyncValue.isLoading`) genuinely cannot work.
// ---------------------------------------------------------------------------

/// Minimal `expand.item` payload for a feed row -- only the fields
/// `Trail.fromJson` requires non-null.
Map<String, dynamic> _feedRowJson(int page) => {
  'id': 'feed-$page',
  'actor': 'actor-1',
  'type': 'trail',
  'created': '2024-01-01 00:00:00.000Z',
  'expand': {
    'item': {
      'id': 'trail-$page',
      'name': 'Trail $page',
      'created': '2024-01-01T00:00:00.000Z',
      'updated': '2024-01-01T00:00:00.000Z',
    },
  },
};

/// Holds each response open until [release], so the test can observe the state
/// mid-fetch rather than only after it settles.
class _GatedFeedAdapter implements HttpClientAdapter {
  int totalPages = 3;

  final List<Completer<void>> _gates = [];

  void release() {
    for (final gate in _gates) {
      if (!gate.isCompleted) gate.complete();
    }
    _gates.clear();
  }

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    final gate = Completer<void>();
    _gates.add(gate);
    await gate.future;

    final page = int.tryParse(options.uri.queryParameters['page'] ?? '1') ?? 1;
    return ResponseBody.fromString(
      jsonEncode({
        'items': [_feedRowJson(page)],
        'page': page,
        'perPage': 10,
        'totalPages': totalPages,
        'totalItems': totalPages,
      }),
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
  final provider = profileFeedProvider('@hiker');

  late _GatedFeedAdapter adapter;
  late ProviderContainer container;

  setUp(() {
    adapter = _GatedFeedAdapter();
    final dio = Dio(BaseOptions(baseUrl: 'https://example.test/api/v1'))
      ..httpClientAdapter = adapter;
    container = ProviderContainer(
      overrides: [apiProvider.overrideWith(() => _StubApi(dio))],
    );
    addTearDown(container.dispose);

    // autoDispose: without a live listener the element is torn down between
    // reads and every assertion below would be talking to a fresh notifier.
    final sub = container.listen(provider, (_, _) {});
    addTearDown(sub.close);
  });

  Future<void> settleFirstPage() async {
    final future = container.read(provider.future);
    await pumpEventQueue();
    adapter.release();
    await future;
  }

  test('isLoadingMore is false on a settled page', () async {
    await settleFirstPage();

    final state = container.read(provider).value!;
    expect(state.hasMore, isTrue, reason: 'precondition: more pages exist');
    expect(
      state.isLoadingMore,
      isFalse,
      reason: 'nothing in flight — this is the state the spinner must NOT show '
          'on, and hasMore being true is exactly what used to mount it',
    );
  });

  test('isLoadingMore is true only for the duration of the fetch', () async {
    await settleFirstPage();
    final notifier = container.read(provider.notifier);

    final call = notifier.loadNextPage();
    await pumpEventQueue();

    expect(
      container.read(provider).value!.isLoadingMore,
      isTrue,
      reason: 'request is open — the spinner should be up',
    );

    adapter.release();
    await call;

    final settled = container.read(provider).value!;
    expect(settled.page, 2, reason: 'precondition: the page actually landed');
    expect(
      settled.isLoadingMore,
      isFalse,
      reason: 'flag must clear on the way out, or the spinner is immortal '
          'again by a different route',
    );
  });

  test('AsyncValue.isLoading stays false for the whole fetch', () async {
    await settleFirstPage();
    final notifier = container.read(provider.notifier);

    final call = notifier.loadNextPage();
    await pumpEventQueue();

    // `loadNextPage` deliberately never publishes an AsyncLoading -- an
    // AsyncData -> AsyncData transition is what keeps `AsyncLoader` from
    // dropping its skeleton over the whole list mid-scroll. The consequence is
    // that `isLoading` is false even with a request open, so gating the
    // spinner on it would render it never rather than always. This test exists
    // so that dead end is caught here instead of on a device.
    expect(container.read(provider), isA<AsyncData<ProfileFeedState>>());
    expect(container.read(provider).isLoading, isFalse);

    adapter.release();
    await call;
  });

  test('the appended page is not polluted by the marker', () async {
    await settleFirstPage();
    final notifier = container.read(provider.notifier);

    final call = notifier.loadNextPage();
    await pumpEventQueue();
    adapter.release();
    await call;

    final state = container.read(provider).value!;
    expect(state.items.length, 2, reason: 'page 2 appended to page 1');
    expect(
      state.isLoadingMore,
      isFalse,
      reason: 'appendPage merges onto the pre-marker state, so the true flag '
          'cannot survive into the merged result',
    );
  });
}
