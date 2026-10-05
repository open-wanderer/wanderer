// Tests for the user-installed CA trust helper (#1255).
//
// Covers the pure DER->PEM conversion, per-certificate failure isolation, and
// the platform and channel guards that keep app start from ever blocking or
// crashing. The channel is mocked and every case that adds trust passes a fresh
// `SecurityContext(withTrustedRoots: false)`, so the shared defaultContext is
// never touched by the test run.
import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/services/user_ca_trust.dart';

// A throwaway self-signed CA (CN=wanderer-test-user-ca, EC P-256, DER), valid
// until 2126. Inlined so the tests need neither a fixture file nor openssl.
const String _testCaDerBase64 =
    'MIIBmDCCAT2gAwIBAgIUMXa9p9iyysp4aHFbYLl7iQ8D320wCgYIKoZIzj0EAwIwIDEeMBwGA1UEAwwVd2FuZGVyZXItdGVzdC11c2VyLWNhMCAXDTI2MTAwNTEyNTU1MVoYDzIxMjYwOTExMTI1NTUxWjAgMR4wHAYDVQQDDBV3YW5kZXJlci10ZXN0LXVzZXItY2EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAATbl3w4CdEkoBryZbsNSD/CxEZfCmXBcuyZgPOipDaRTi7O9tjLdbYW3AVQC6qZTd5qVoa1Nm/ltB/NnqOxO+4Po1MwUTAdBgNVHQ4EFgQUUfvdTy71eskeLgfu9BkSwuBgib8wHwYDVR0jBBgwFoAUUfvdTy71eskeLgfu9BkSwuBgib8wDwYDVR0TAQH/BAUwAwEB/zAKBggqhkjOPQQDAgNJADBGAiEAgHgjWIL2Vvg5AExlpIv/ELkUSepmFJm/aN5YHgMjQtwCIQD2ZE4fEEAxXurdN4QUAobpfV03pQDKv+rPMWkxzIMOaQ==';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  final Uint8List validDer = base64Decode(_testCaDerBase64);
  final Uint8List garbage = Uint8List.fromList([1, 2, 3, 4]);
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

  SecurityContext freshContext() => SecurityContext(withTrustedRoots: false);

  int calls = 0;
  List<String> methods = <String>[];

  void mockChannel(Future<Object?> Function(MethodCall call) handler) {
    messenger.setMockMethodCallHandler(userCaChannel, (MethodCall call) {
      calls++;
      methods.add(call.method);
      return handler(call);
    });
  }

  setUp(() {
    calls = 0;
    methods = <String>[];
  });

  tearDown(() {
    messenger.setMockMethodCallHandler(userCaChannel, null);
  });

  group('derToPem', () {
    test('wraps the body in BEGIN/END CERTIFICATE lines', () {
      final pem = derToPem(validDer);
      expect(pem, startsWith('-----BEGIN CERTIFICATE-----\n'));
      expect(pem, endsWith('-----END CERTIFICATE-----\n'));
    });

    test('splits the body into 64-char lines', () {
      final der = Uint8List.fromList(List<int>.generate(130, (i) => i));
      final lines = derToPem(der).trimRight().split('\n');
      final body = lines.sublist(1, lines.length - 1);
      expect(body.map((l) => l.length).toList(), [64, 64, 48]);
    });

    test('body decodes back to the input bytes', () {
      final der = Uint8List.fromList(List<int>.generate(130, (i) => i));
      final lines = derToPem(der).trimRight().split('\n');
      final body = lines.sublist(1, lines.length - 1).join();
      expect(base64Decode(body), der);
    });
  });

  group('trustDerCertificates', () {
    test('adds a valid certificate', () {
      expect(trustDerCertificates([validDer], freshContext()), 1);
    });

    test('an invalid certificate does not abort the next one', () {
      expect(trustDerCertificates([garbage, validDer], freshContext()), 1);
    });

    test('empty input returns 0', () {
      expect(trustDerCertificates(<List<int>>[], freshContext()), 0);
    });

    test('only garbage returns 0 without throwing', () {
      expect(trustDerCertificates([garbage], freshContext()), 0);
    });
  });

  group('trustUserCaCertificates', () {
    test('isAndroid: false never touches the channel', () async {
      mockChannel((_) async => <Object?>[validDer]);
      final added = await trustUserCaCertificates(
        isAndroid: false,
        context: freshContext(),
      );
      expect(added, 0);
      expect(calls, 0);
    });

    test('default platform gate skips the channel off Android', () async {
      mockChannel((_) async => <Object?>[validDer]);
      final added = await trustUserCaCertificates(context: freshContext());
      expect(added, 0);
      expect(calls, 0);
    });

    test('trusts the valid certificate and skips the garbage one', () async {
      mockChannel((_) async => <Object?>[garbage, validDer]);
      final added = await trustUserCaCertificates(
        isAndroid: true,
        context: freshContext(),
      );
      expect(added, 1);
      expect(calls, 1);
      expect(methods, ['getUserCaCertificates']);
    });

    test('a PlatformException is swallowed', () async {
      mockChannel((_) async => throw PlatformException(code: 'boom'));
      final added = await trustUserCaCertificates(
        isAndroid: true,
        context: freshContext(),
      );
      expect(added, 0);
    });

    test('a missing handler (MissingPluginException) is swallowed', () async {
      final added = await trustUserCaCertificates(
        isAndroid: true,
        context: freshContext(),
      );
      expect(added, 0);
    });

    test('a null result returns 0', () async {
      mockChannel((_) async => null);
      final added = await trustUserCaCertificates(
        isAndroid: true,
        context: freshContext(),
      );
      expect(added, 0);
    });

    test('non-bytes elements are skipped', () async {
      mockChannel((_) async => <Object?>['x', validDer]);
      final added = await trustUserCaCertificates(
        isAndroid: true,
        context: freshContext(),
      );
      expect(added, 1);
    });

    test('a handler that never replies times out and returns 0', () async {
      mockChannel((_) => Completer<Object?>().future);
      final added = await trustUserCaCertificates(
        isAndroid: true,
        context: freshContext(),
        timeout: const Duration(milliseconds: 50),
      );
      expect(added, 0);
    });
  });
}
