import 'dart:io' show SocketException;

import 'package:cookie_jar/cookie_jar.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';

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
