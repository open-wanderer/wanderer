import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/profile/profile_follows_provider.dart';

// ---------------------------------------------------------------------------
// Regression harness for the 2026-08-05 request storm.
//
// `loadNextPage` guarded re-entrancy with `state.isLoading`, but deliberately
// never published an `AsyncLoading` (to avoid spinner flicker) and
// `AsyncValue.guard` only assigns to `state` after its future resolves -- so
// the flag was false for the whole fetch and the guard never fired. A
// ScrollController listener firing once per frame turned one fling into ~100+
// concurrent requests for the SAME page, which tripped server-side rate
// limiting.
//
// These tests drive `loadNextPage` the way a scroll listener does -- many
// synchronous calls with nothing awaited between them -- and assert on the
// number of requests the adapter actually saw.
//
// `profileFollowsProvider` is the vehicle because it is the thinnest of the
// five paged providers: a plain GET, no filter provider, no auth, no
// ObjectBox. The guard under test lives in the shared `PagedLoadMore` mixin,
// so it is the same code path all five now take.
// ---------------------------------------------------------------------------

/// A minimally complete `Actor` payload -- only the fields `Actor.fromJson`
/// requires non-null. The identity varies by [page] so a test can tell which
/// page a row came from.
Map<String, dynamic> _actorJson(int page) => {
  'id': 'actor-$page',
  'collectionId': 'actors',
  'collectionName': 'actors',
  'created': '2026-01-01 00:00:00.000Z',
  'updated': '2026-01-01 00:00:00.000Z',
  'username': 'hiker$page',
  'preferred_username': 'hiker$page',
  'iri': 'https://example.test/actor-$page',
  'inbox': 'https://example.test/actor-$page/inbox',
  'public_key': 'key-$page',
  'last_fetched': '2026-01-01 00:00:00.000Z',
  'user': 'user-$page',
};

/// Counts requests and holds each response open until [release] is called, so
/// a burst of `loadNextPage` calls overlaps the way frame-driven scroll
/// callbacks do in the real app.
class _CountingAdapter implements HttpClientAdapter {
  int totalPages = 5;

  /// When true, the next request throws instead of answering (then resets).
  bool failNext = false;

  final List<String?> requestedPages = [];
  final List<Completer<void>> _gates = [];

  int get requestCount => requestedPages.length;

  /// Lets every in-flight response complete.
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
    requestedPages.add(options.uri.queryParameters['page']);

    if (failNext) {
      failNext = false;
      throw DioException.connectionError(
        requestOptions: options,
        reason: 'simulated failure',
      );
    }

    final gate = Completer<void>();
    _gates.add(gate);
    await gate.future;

    final page = int.tryParse(options.uri.queryParameters['page'] ?? '1') ?? 1;
    return ResponseBody.fromString(
      jsonEncode({
        'items': [_actorJson(page)],
        'page': page,
        'totalPages': totalPages,
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
  final provider = profileFollowsProvider('@hiker', 'following');

  late _CountingAdapter adapter;
  late ProviderContainer container;

  setUp(() {
    adapter = _CountingAdapter();
    final dio = Dio(BaseOptions(baseUrl: 'https://example.test/api/v1'))
      ..httpClientAdapter = adapter;
    container = ProviderContainer(
      overrides: [apiProvider.overrideWith(() => _StubApi(dio))],
    );
    addTearDown(container.dispose);

    // These providers are autoDispose. Without a live listener the element is
    // torn down the moment each `read` returns, so the next `read` silently
    // rebuilds -- refetching page 1 and handing back a *different* notifier,
    // which makes every assertion below meaningless.
    final sub = container.listen(provider, (_, _) {});
    addTearDown(sub.close);
  });

  /// Resolves page 1 so the provider is in AsyncData with more pages to go --
  /// the state a scroll listener acts on.
  Future<void> settleFirstPage() async {
    final future = container.read(provider.future);
    await pumpEventQueue();
    adapter.release();
    await future;
    expect(adapter.requestedPages, ['1']);
  }

  test('a burst of loadNextPage calls issues exactly one request', () async {
    await settleFirstPage();
    final notifier = container.read(provider.notifier);

    // 100 synchronous calls with no await between them -- one fling's worth of
    // ScrollController callbacks. Before the fix this produced 100 requests,
    // all for page 2.
    final calls = [for (var i = 0; i < 100; i++) notifier.loadNextPage()];
    await pumpEventQueue();

    expect(
      adapter.requestCount,
      2,
      reason: 'page 1 + exactly one page-2 request, not one per call',
    );

    adapter.release();
    await Future.wait(calls);

    expect(adapter.requestedPages, ['1', '2']);

    final state = container.read(provider).value!;
    expect(state.page, 2);
    expect(state.items.length, 2, reason: 'page 2 appended once, not 100x');
  });

  test('pagination still advances one page per settled burst', () async {
    await settleFirstPage();
    final notifier = container.read(provider.notifier);

    for (var expectedPage = 2; expectedPage <= 4; expectedPage++) {
      final call = notifier.loadNextPage();
      await pumpEventQueue();
      adapter.release();
      await call;

      final state = container.read(provider).value!;
      expect(state.page, expectedPage);
      expect(state.items.length, expectedPage);
    }

    expect(adapter.requestedPages, ['1', '2', '3', '4']);
  });

  test('a failed page does not wedge pagination', () async {
    await settleFirstPage();
    final notifier = container.read(provider.notifier);

    // `_inFlight` is cleared in a `finally`, so the retry below must be let
    // through -- a bare reset after the await would have left the flag stuck
    // true and killed paging for the rest of the notifier's life.
    adapter.failNext = true;
    await notifier.loadNextPage();
    expect(container.read(provider), isA<AsyncError>());

    final retry = notifier.loadNextPage();
    await pumpEventQueue();
    adapter.release();
    await retry;

    expect(adapter.requestedPages, ['1', '2', '2']);
  });

  test('stops at the last page', () async {
    adapter.totalPages = 1;
    await settleFirstPage();
    final notifier = container.read(provider.notifier);

    for (var i = 0; i < 20; i++) {
      await notifier.loadNextPage();
    }

    expect(
      adapter.requestedPages,
      ['1'],
      reason: 'hasMore is false — never fetches',
    );
  });
}
