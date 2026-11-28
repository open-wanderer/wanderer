import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/services/server_capabilities.dart';

// The old BFF error page renders its status and message. Discovery treats
// its surrounding HTML as opaque; the HTTP 404 is what selects legacy mode.
const _legacy404Page =
    '<!doctype html><html><head>'
    '<title>404 | wanderer</title></head><body>'
    '<h1>404</h1><p>Not Found</p></body></html>';

void main() {
  late Dio api;
  late ServerCapabilities capabilities;
  late List<RequestOptions> requests;
  late int status;
  late String contentType;
  dynamic body;
  late DateTime now;

  setUp(() {
    now = DateTime.utc(2026);
    requests = [];
    status = 200;
    contentType = Headers.jsonContentType;
    body = {'assetApi': true};
    api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (request, handler) {
          requests.add(request);
          handler.resolve(
            Response(
              requestOptions: request,
              statusCode: status,
              data: body,
              headers: Headers.fromMap({
                Headers.contentTypeHeader: [contentType],
              }),
            ),
          );
        },
      ),
    );
    capabilities = ServerCapabilities(api, now: () => now);
  });
  tearDown(() {
    capabilities.dispose();
    api.close();
  });

  test(
    'marker is discovered once for concurrent callers in a session',
    () async {
      expect(
        await Future.wait([capabilities.discover(), capabilities.discover()]),
        [true, true],
      );
      expect(await capabilities.discover(), true);
      expect(requests, hasLength(1));
      expect(
        requests.single.uri.toString(),
        'https://a.example/api/v1/capabilities',
      );
      expect(requests.single.followRedirects, false);
      expect(requests.single.receiveTimeout, const Duration(seconds: 8));
      expect(requests.single.sendTimeout, const Duration(seconds: 8));
    },
  );

  for (final response in [(200, <String, dynamic>{}), (404, null)]) {
    test('only explicit absence selects legacy: ${response.$1}', () async {
      status = response.$1;
      body = response.$2;
      expect(await capabilities.discover(), false);
      expect(await capabilities.discover(), false);
      expect(requests, hasLength(1));
    });
  }

  for (final code in [201, 301, 400, 401, 403, 429, 500, 503]) {
    test(
      'status $code stays unknown until its discovery cooldown ends',
      () async {
        status = code;
        expect(await capabilities.discover(), isNull);
        expect(capabilities.assetApi, isNull);
        status = 200;
        expect(await capabilities.discover(), isNull);
        expect(requests, hasLength(1));
        now = now.add(const Duration(seconds: 30));
        expect(await capabilities.discover(), true);
        expect(requests, hasLength(2));
      },
    );
  }

  for (final value in [
    'html',
    null,
    {'assetApi': null},
    {'assetApi': false},
    {'assetApi': 'true'},
  ]) {
    test(
      'inconclusive body $value stays unknown without repeated probes',
      () async {
        body = value;
        expect(await capabilities.discover(), isNull);
        expect(await capabilities.discover(), isNull);
        expect(requests, hasLength(1));
      },
    );
  }

  test(
    'a network failure uses the same cooldown as an inconclusive response',
    () async {
      var attempts = 0;
      api.interceptors.insert(
        0,
        InterceptorsWrapper(
          onRequest: (request, handler) {
            attempts++;
            handler.reject(
              DioException(
                requestOptions: request,
                type: DioExceptionType.connectionError,
              ),
            );
          },
        ),
      );
      expect(await capabilities.discover(), isNull);
      expect(await capabilities.discover(), isNull);
      expect(attempts, 1);
      api.interceptors.removeAt(0);
      now = now.add(const Duration(seconds: 30));
      expect(await capabilities.discover(), true);
    },
  );

  test(
    'unknown discovery backs off across repeated save and resume triggers',
    () async {
      status = 503;
      expect(await capabilities.discover(), isNull);
      for (var i = 0; i < 10; i++) {
        expect(await capabilities.discover(), isNull);
      }
      capabilities.background();
      expect(await capabilities.resume(), isNull);
      expect(requests, hasLength(1));
      now = now.add(const Duration(seconds: 30));
      expect(await capabilities.discover(), isNull);
      expect(requests, hasLength(2));
      now = now.add(const Duration(seconds: 30));
      expect(await capabilities.discover(), isNull);
      expect(requests, hasLength(2));
      now = now.add(const Duration(seconds: 30));
      status = 200;
      expect(await capabilities.discover(), true);
      expect(requests, hasLength(3));
    },
  );

  test('unconfigured servers never start probes or retries', () async {
    for (final base in ['', 'https://unknown-server.local/api/v1']) {
      capabilities.dispose();
      api.options.baseUrl = base;
      capabilities = ServerCapabilities(api, now: () => now);
      capabilities.newSession();
      for (var i = 0; i < 10; i++) {
        expect(await capabilities.discover(), isNull);
      }
      now = now.add(const Duration(hours: 1));
      expect(await capabilities.resume(), isNull);
    }
    expect(requests, isEmpty);
  });

  test(
    'logged-out discovery stays dormant until a new authenticated session',
    () async {
      capabilities.dispose();
      capabilities = ServerCapabilities(
        api,
        initiallyActive: false,
        now: () => now,
      );
      for (var i = 0; i < 10; i++) {
        expect(await capabilities.discover(), isNull);
      }
      expect(requests, isEmpty);
      capabilities.newSession();
      expect(await capabilities.discover(), true);
      expect(requests, hasLength(1));
      capabilities.endSession();
      now = now.add(const Duration(days: 1));
      expect(await capabilities.resume(), isNull);
      expect(await capabilities.discover(), isNull);
      expect(requests, hasLength(1));
    },
  );

  test(
    'login resets a known capability and an inconclusive cooldown',
    () async {
      expect(await capabilities.discover(), true);
      status = 401;
      capabilities.newSession();
      expect(await capabilities.discover(), isNull);
      status = 404;
      body = {'message': 'not_found'};
      capabilities.newSession();
      expect(await capabilities.discover(), false);
      expect(requests, hasLength(3));
    },
  );

  test(
    'legacy decisions expire before an in-session server upgrade can stick',
    () async {
      status = 404;
      expect(await capabilities.discover(), false);
      status = 200;
      now = now.add(const Duration(minutes: 4));
      expect(await capabilities.discover(), false);
      expect(requests, hasLength(1));
      now = now.add(const Duration(minutes: 1));
      expect(await capabilities.discover(), true);
      expect(requests, hasLength(2));
    },
  );

  test(
    'an explicit legacy rejection selects assets without another probe',
    () async {
      status = 404;
      expect(await capabilities.discover(), false);
      var resolved = 0;
      capabilities.onResolved = () => resolved++;
      capabilities.markAssetApi();
      expect(capabilities.assetApi, true);
      expect(await capabilities.discover(), true);
      expect(resolved, 1);
      expect(requests, hasLength(1));
    },
  );

  test('marking the asset API invalidates an older in-flight probe', () async {
    final release = Completer<void>();
    api.interceptors.insert(
      0,
      InterceptorsWrapper(
        onRequest: (request, handler) async {
          await release.future;
          handler.resolve(Response(requestOptions: request, statusCode: 404));
        },
      ),
    );
    final pending = capabilities.discover();
    await Future<void>.delayed(Duration.zero);
    capabilities.markAssetApi();
    release.complete();
    expect(await pending, isNull);
    expect(await capabilities.discover(), true);
  });

  test(
    'switching servers never reuses the other server marker or cooldown',
    () async {
      status = 503;
      expect(await capabilities.discover(), isNull);
      api.options.baseUrl = 'https://b.example/api/v1';
      status = 404;
      expect(await capabilities.discover(), false);
      expect(requests.last.uri.host, 'b.example');
      api.options.baseUrl = 'https://a.example/api/v1';
      status = 200;
      expect(capabilities.assetApi, isNull);
      expect(await capabilities.discover(), true);
      expect(requests, hasLength(3));
    },
  );

  test(
    'a late response from the previous server cannot set its marker',
    () async {
      final first = Completer<void>();
      api.interceptors.insert(
        0,
        InterceptorsWrapper(
          onRequest: (request, handler) async {
            if (request.uri.host == 'a.example') {
              await first.future;
              handler.resolve(
                Response(
                  requestOptions: request,
                  statusCode: 200,
                  data: {'assetApi': true},
                ),
              );
            } else {
              handler.resolve(
                Response(requestOptions: request, statusCode: 404),
              );
            }
          },
        ),
      );
      final pending = capabilities.discover();
      await Future<void>.delayed(Duration.zero);
      api.options.baseUrl = 'https://b.example/api/v1';
      expect(await capabilities.discover(), false);
      first.complete();
      expect(await pending, isNull);
      expect(capabilities.assetApi, false);
    },
  );

  test('a long background interval discovers server upgrades again', () async {
    body = {};
    expect(await capabilities.discover(), false);
    capabilities.background();
    now = now.add(const Duration(minutes: 2));
    expect(await capabilities.resume(), false);
    expect(requests, hasLength(1));
    capabilities.background();
    now = now.add(const Duration(minutes: 16));
    body = {'assetApi': true};
    expect(await capabilities.resume(), true);
    expect(requests, hasLength(2));
  });

  test(
    'unknown discovery retries automatically and notifies when resolved',
    () async {
      capabilities.dispose();
      capabilities = ServerCapabilities(
        api,
        retryDelay: const Duration(milliseconds: 5),
        now: () => now,
      );
      var resolved = 0;
      capabilities.onResolved = () => resolved++;
      status = 503;
      expect(await capabilities.discover(), isNull);
      status = 200;
      now = now.add(const Duration(milliseconds: 5));
      await Future<void>.delayed(const Duration(milliseconds: 30));
      expect(capabilities.assetApi, true);
      expect(resolved, 1);
    },
  );

  test(
    'a short background interval rearms unknown discovery after cooldown',
    () async {
      capabilities.dispose();
      capabilities = ServerCapabilities(
        api,
        retryDelay: const Duration(milliseconds: 20),
        now: () => now,
      );
      status = 503;
      expect(await capabilities.discover(), isNull);
      capabilities.background();
      status = 200;
      expect(await capabilities.resume(), isNull);
      expect(requests, hasLength(1));
      now = now.add(const Duration(milliseconds: 20));
      await Future<void>.delayed(const Duration(milliseconds: 50));
      expect(capabilities.assetApi, true);
      expect(requests, hasLength(2));
    },
  );

  for (final context in [
    'public logged out',
    'public logged in',
    'private logged out',
    'private logged in',
  ]) {
    test(
      'released beta.25 and v0.21.2 404 JSON selects legacy: $context',
      () async {
        status = 404;
        body = {'message': 'not_found'};
        expect(await capabilities.discover(), false);
        expect(await capabilities.discover(), false);
        expect(requests, hasLength(1));
      },
    );
  }

  for (final context in [
    'public logged out',
    'public logged in',
    'private logged in',
  ]) {
    test('the feature app BFF 404 page selects legacy: $context', () async {
      status = 404;
      contentType = 'text/html';
      body = _legacy404Page;
      expect(await capabilities.discover(), false);
      expect(await capabilities.discover(), false);
      expect(requests, hasLength(1));
    });
  }

  test(
    'a private old server is discovered again after its pre-login 401',
    () async {
      status = 401;
      body = {'message': 'Unauthorized'};
      expect(await capabilities.discover(), isNull);
      expect(await capabilities.discover(), isNull);
      expect(requests, hasLength(1));
      status = 404;
      contentType = 'text/html';
      body = _legacy404Page;
      capabilities.newSession();
      expect(await capabilities.discover(), false);
      expect(requests, hasLength(2));
    },
  );
}
