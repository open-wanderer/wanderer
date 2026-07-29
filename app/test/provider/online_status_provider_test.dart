import 'dart:io' show SocketException;

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/online_status_provider.dart';

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
  });
}
