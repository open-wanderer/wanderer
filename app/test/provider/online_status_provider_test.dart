import 'dart:io' show SocketException;
import 'dart:typed_data';

import 'package:cookie_jar/cookie_jar.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/util/connectivity.dart';

/// In-memory cookie storage — `cookie_jar` ships only a `FileStorage`, and
/// these tests must not touch disk. Only exists to satisfy the
/// `cookieJarProvider` override that `apiProvider` depends on; no cookie is
/// ever read or written here.
class _MemoryCookieStorage implements Storage {
  final Map<String, String> _values = {};

  @override
  Future<void> init(bool persistSession, bool ignoreExpires) async {}

  @override
  Future<String?> read(String key) async => _values[key];

  @override
  Future<void> write(String key, String value) async => _values[key] = value;

  @override
  Future<void> delete(String key) async => _values.remove(key);

  @override
  Future<void> deleteAll(List<String> keys) async => _values.clear();
}

DioException _exception({
  required DioExceptionType type,
  Object? error,
  int? statusCode,
}) {
  final requestOptions = RequestOptions(path: '/health');
  return DioException(
    requestOptions: requestOptions,
    type: type,
    error: error,
    response: statusCode == null
        ? null
        : Response(requestOptions: requestOptions, statusCode: statusCode),
  );
}

/// Answers every request with [status] and an empty JSON body — the probe only
/// ever looks at the status code.
class _StatusAdapter implements HttpClientAdapter {
  _StatusAdapter(this.status);

  final int status;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async => ResponseBody.fromString('{}', status);

  @override
  void close({bool force = false}) {}
}

/// Never reaches a server: throws a genuine transport-level failure of [type],
/// the shape `isConnectionFailure` is meant to recognise.
class _FailingAdapter implements HttpClientAdapter {
  _FailingAdapter(this.type);

  final DioExceptionType type;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async => throw DioException(requestOptions: options, type: type);

  @override
  void close({bool force = false}) {}
}

Dio _probeDio(HttpClientAdapter adapter) {
  final dio = Dio(BaseOptions(baseUrl: 'https://server.example/api/v1'));
  dio.httpClientAdapter = adapter;
  return dio;
}

/// A container holding the REAL `apiProvider` (interceptors included) with its
/// transport stubbed and a real server URL applied, so the online-status
/// interceptor can be exercised end to end.
ProviderContainer _apiContainer(HttpClientAdapter adapter) {
  final container = ProviderContainer(
    overrides: [
      cookieJarProvider.overrideWithValue(
        PersistCookieJar(storage: _MemoryCookieStorage()),
      ),
    ],
  );
  container.read(apiProvider.notifier).updateBaseUrl('https://server.example');
  container.read(apiProvider).httpClientAdapter = adapter;
  return container;
}

/// One of the ~1000 glyph URLs the shared client fetches on a fresh install.
/// Third-party host by default (`MAP_ASSETS_URL` unset on the server).
const _thirdPartyUrl =
    'https://protomaps.github.io/basemaps-assets/fonts/Noto%20Sans%20Regular/0-255.pbf';

void main() {
  group('isConnectionFailure', () {
    test('true for connectionError', () {
      expect(
        isConnectionFailure(
          _exception(type: DioExceptionType.connectionError),
        ),
        isTrue,
      );
    });

    test('true for connectionTimeout', () {
      expect(
        isConnectionFailure(
          _exception(type: DioExceptionType.connectionTimeout),
        ),
        isTrue,
      );
    });

    test('true for unknown wrapping a SocketException', () {
      expect(
        isConnectionFailure(
          _exception(
            type: DioExceptionType.unknown,
            error: const SocketException('no route to host'),
          ),
        ),
        isTrue,
      );
    });

    test('false for unknown when error is not a SocketException', () {
      expect(
        isConnectionFailure(
          _exception(type: DioExceptionType.unknown, error: 'boom'),
        ),
        isFalse,
      );
    });

    for (final status in [404, 422, 500, 503]) {
      test('false for badResponse at $status — a server answered', () {
        expect(
          isConnectionFailure(
            _exception(
              type: DioExceptionType.badResponse,
              statusCode: status,
            ),
          ),
          isFalse,
        );
      });
    }

    test('false for badCertificate', () {
      expect(
        isConnectionFailure(
          _exception(type: DioExceptionType.badCertificate),
        ),
        isFalse,
      );
    });

    test('false for cancel', () {
      expect(
        isConnectionFailure(_exception(type: DioExceptionType.cancel)),
        isFalse,
      );
    });
  });

  group('OnlineStatus notifier', () {
    test('build() returns true (optimistic default)', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      expect(container.read(onlineStatusProvider), isTrue);
    });

    test('markOffline() then markOnline() flips state as expected', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      container.read(onlineStatusProvider.notifier).markOffline();
      expect(container.read(onlineStatusProvider), isFalse);

      container.read(onlineStatusProvider.notifier).markOnline();
      expect(container.read(onlineStatusProvider), isTrue);
    });

    test(
      'refresh() does not probe (or report offline) while the api client is '
      'still on the placeholder host',
      () async {
        final container = ProviderContainer(
          overrides: [
            cookieJarProvider.overrideWithValue(
              PersistCookieJar(storage: _MemoryCookieStorage()),
            ),
          ],
        );
        addTearDown(container.dispose);

        // The cold-start ordering: main.dart seeds the status before
        // Auth.build() has applied a real server URL. Probing here would hit a
        // host that cannot resolve and mark the app offline on every launch.
        expect(container.read(apiProvider.notifier).isConfigured, isFalse);

        await expectLater(
          container.read(onlineStatusProvider.notifier).refresh(),
          completion(isTrue),
        );
        expect(container.read(onlineStatusProvider), isTrue);
      },
    );

    test('isBackendReachable: 200 is online', () async {
      expect(await isBackendReachable(_probeDio(_StatusAdapter(200))), isTrue);
    });

    test(
      'isBackendReachable: 404 is ONLINE — a server without the /health route '
      'still answered',
      () async {
        expect(
          await isBackendReachable(_probeDio(_StatusAdapter(404))),
          isTrue,
        );
      },
    );

    test('isBackendReachable: 500 is online — the far end answered', () async {
      expect(await isBackendReachable(_probeDio(_StatusAdapter(500))), isTrue);
    });

    test(
      'isBackendReachable: 503 is offline — /health says the database is down',
      () async {
        expect(
          await isBackendReachable(_probeDio(_StatusAdapter(503))),
          isFalse,
        );
      },
    );

    for (final type in [
      DioExceptionType.connectionError,
      DioExceptionType.connectionTimeout,
    ]) {
      test('isBackendReachable: $type is offline — nothing answered', () async {
        expect(
          await isBackendReachable(_probeDio(_FailingAdapter(type))),
          isFalse,
        );
      });
    }

    test(
      'isBackendReachable: badCertificate is online — a TLS handshake means '
      'the far end responded',
      () async {
        expect(
          await isBackendReachable(
            _probeDio(_FailingAdapter(DioExceptionType.badCertificate)),
          ),
          isTrue,
        );
      },
    );
  });

  group('online-status interceptor host gating', () {
    test('a failed BACKEND request marks the app offline', () async {
      final container = _apiContainer(
        _FailingAdapter(DioExceptionType.connectionError),
      );
      addTearDown(container.dispose);

      await expectLater(
        container.read(apiProvider).get('/trail'),
        throwsA(isA<DioException>()),
      );
      expect(container.read(onlineStatusProvider), isFalse);
    });

    test(
      'a failed THIRD-PARTY request leaves the status alone — the glyph/sprite '
      'warm must not report the backend down',
      () async {
        final container = _apiContainer(
          _FailingAdapter(DioExceptionType.connectionError),
        );
        addTearDown(container.dispose);

        await expectLater(
          container.read(apiProvider).get(_thirdPartyUrl),
          throwsA(isA<DioException>()),
        );
        expect(container.read(onlineStatusProvider), isTrue);
      },
    );

    test('a successful BACKEND response marks the app online', () async {
      final container = _apiContainer(_StatusAdapter(200));
      addTearDown(container.dispose);

      container.read(onlineStatusProvider.notifier).markOffline();
      await container.read(apiProvider).get('/trail');
      expect(container.read(onlineStatusProvider), isTrue);
    });

    test(
      'a successful THIRD-PARTY response does not clear an offline status — it '
      'says nothing about the backend',
      () async {
        final container = _apiContainer(_StatusAdapter(200));
        addTearDown(container.dispose);

        container.read(onlineStatusProvider.notifier).markOffline();
        await container.read(apiProvider).get(_thirdPartyUrl);
        expect(container.read(onlineStatusProvider), isFalse);
      },
    );

    test(
      'a failure against the placeholder host leaves the status alone',
      () async {
        final container = ProviderContainer(
          overrides: [
            cookieJarProvider.overrideWithValue(
              PersistCookieJar(storage: _MemoryCookieStorage()),
            ),
          ],
        );
        addTearDown(container.dispose);
        // No updateBaseUrl: still on kUnconfiguredApiHost, as on a cold start
        // before Auth.build() applies the real server URL.
        container.read(apiProvider).httpClientAdapter = _FailingAdapter(
          DioExceptionType.connectionError,
        );

        await expectLater(
          container.read(apiProvider).get('/trail'),
          throwsA(isA<DioException>()),
        );
        expect(container.read(onlineStatusProvider), isTrue);
      },
    );
  });

  group('OnlineStatus api client', () {
    test('isConfigured flips once a real server URL is applied', () {
      final container = ProviderContainer(
        overrides: [
          cookieJarProvider.overrideWithValue(
            PersistCookieJar(storage: _MemoryCookieStorage()),
          ),
        ],
      );
      addTearDown(container.dispose);

      final api = container.read(apiProvider.notifier);
      expect(api.isConfigured, isFalse);

      api.updateBaseUrl('https://wanderer.example');
      expect(api.isConfigured, isTrue);
    });
  });
}
