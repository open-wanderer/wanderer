import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/profile/profile_follows_provider.dart';

// The follows endpoint pages remote collections with a cursor: the `next` of
// one response is sent back as `cursor` for the following page. Backends that
// send no `next` key keep the page < totalPages test.

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

/// Records each request's query and answers with one actor for that page.
class _RecordingAdapter implements HttpClientAdapter {
  _RecordingAdapter(this.nextFor);

  /// Returns the `next` value for a page, or null to omit the key.
  final String? Function(int page) nextFor;
  int totalPages = 5;

  final List<Map<String, String>> queries = [];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    queries.add(options.uri.queryParameters);
    final page = int.parse(options.uri.queryParameters['page'] ?? '1');
    final next = nextFor(page);
    return ResponseBody.fromString(
      jsonEncode({
        'items': [_actorJson(page)],
        'page': page,
        'totalPages': totalPages,
        'next': ?next,
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
  final provider = profileFollowsProvider('@hiker', 'followers');

  late ProviderContainer container;

  _RecordingAdapter setUpWith(String? Function(int page) nextFor) {
    final adapter = _RecordingAdapter(nextFor);
    final dio = Dio(BaseOptions(baseUrl: 'https://example.test/api/v1'))
      ..httpClientAdapter = adapter;
    container = ProviderContainer(
      overrides: [apiProvider.overrideWith(() => _StubApi(dio))],
    );
    addTearDown(container.dispose);
    // autoDispose: keep the element alive between reads.
    final sub = container.listen(provider, (_, _) {});
    addTearDown(sub.close);
    return adapter;
  }

  test('sends the previous next as cursor together with the page', () async {
    final adapter = setUpWith(
      (page) => page < 3 ? 'https://r.test/followers?page=${page + 1}' : '',
    );

    await container.read(provider.future);
    final notifier = container.read(provider.notifier);
    await notifier.loadNextPage();
    await notifier.loadNextPage();

    expect(adapter.queries, hasLength(3));
    expect(adapter.queries[0].containsKey('cursor'), isFalse);
    expect(adapter.queries[1]['cursor'], 'https://r.test/followers?page=2');
    expect(adapter.queries[1]['page'], '2');
    expect(adapter.queries[2]['cursor'], 'https://r.test/followers?page=3');
    expect(adapter.queries[2]['page'], '3');
  });

  test('an empty next ends paging even when totalPages says more', () async {
    final adapter = setUpWith((page) => '');

    final first = await container.read(provider.future);
    expect(adapter.totalPages, 5);
    expect(first.hasMore, isFalse);

    await container.read(provider.notifier).loadNextPage();
    expect(adapter.queries, hasLength(1));
  });

  test('without a next key it sends no cursor and uses totalPages', () async {
    final adapter = setUpWith((page) => null);
    adapter.totalPages = 2;

    final first = await container.read(provider.future);
    expect(first.hasMore, isTrue);

    final notifier = container.read(provider.notifier);
    await notifier.loadNextPage();
    expect(adapter.queries, hasLength(2));
    expect(adapter.queries[1].containsKey('cursor'), isFalse);
    expect(adapter.queries[1]['page'], '2');

    expect(container.read(provider).requireValue.hasMore, isFalse);
    await notifier.loadNextPage();
    expect(adapter.queries, hasLength(2));
  });
}
