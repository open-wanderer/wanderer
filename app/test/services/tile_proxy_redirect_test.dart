import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/services/tile_proxy_server.dart';

// ---------------------------------------------------------------------------
// Tests for the pure redirect-target validation and template substitution
// helpers only (isSafeRedirectTarget, buildUpstreamRedirect). Matches this
// repo's precedent (tile_repository_manager_test.dart, tile_proxy_identity_
// test.dart) of unit-testing @visibleForTesting pure helpers with no live
// Store and no live server.
// ---------------------------------------------------------------------------

void main() {
  group('isSafeRedirectTarget', () {
    test('accepts an https CDN host', () {
      expect(
        isSafeRedirectTarget(Uri.parse('https://api.protomaps.com/tiles/1/2/3.mvt')),
        isTrue,
      );
    });

    test('accepts an http CDN host', () {
      expect(
        isSafeRedirectTarget(Uri.parse('http://tiles.example.com/1/2/3.pbf')),
        isTrue,
      );
    });

    test('rejects a loopback host with a port', () {
      expect(isSafeRedirectTarget(Uri.parse('http://127.0.0.1:1234/x')), isFalse);
    });

    test('rejects the literal host "localhost"', () {
      expect(isSafeRedirectTarget(Uri.parse('http://localhost/x')), isFalse);
    });

    test('rejects the IPv6 loopback address', () {
      expect(isSafeRedirectTarget(Uri.parse('https://[::1]/x')), isFalse);
    });

    test('rejects a link-local address', () {
      expect(isSafeRedirectTarget(Uri.parse('http://169.254.1.1/x')), isFalse);
    });

    // 0.0.0.0 and :: are the unspecified addresses: neither loopback nor
    // link-local, so they slip past the checks catching 127.0.0.1 and ::1 --
    // yet they resolve to localhost, the same local relay this guard prevents.
    test('rejects the IPv4 unspecified address', () {
      expect(isSafeRedirectTarget(Uri.parse('http://0.0.0.0/x')), isFalse);
    });

    test('rejects the IPv4 unspecified address with a port', () {
      expect(isSafeRedirectTarget(Uri.parse('http://0.0.0.0:8080/x')), isFalse);
    });

    test('rejects the IPv6 unspecified address', () {
      expect(isSafeRedirectTarget(Uri.parse('http://[::]/x')), isFalse);
    });

    test('rejects a scheme-relative URI', () {
      expect(isSafeRedirectTarget(Uri.parse('//host/x')), isFalse);
    });

    test('rejects a relative path', () {
      expect(isSafeRedirectTarget(Uri.parse('/x/y.pbf')), isFalse);
    });

    test('rejects a file:// URI', () {
      expect(isSafeRedirectTarget(Uri.parse('file:///etc/passwd')), isFalse);
    });
  });

  group('buildUpstreamRedirect', () {
    test('substitutes all three tokens', () {
      final result = buildUpstreamRedirect(
        'https://tiles.example.com/{z}/{x}/{y}.pbf',
        z: 12,
        x: 34,
        y: 56,
      );
      expect(result, 'https://tiles.example.com/12/34/56.pbf');
    });

    test('substitutes tokens and preserves a query string', () {
      final result = buildUpstreamRedirect(
        'https://tiles.example.com/{z}/{x}/{y}.pbf?key=abc',
        z: 1,
        x: 2,
        y: 3,
      );
      expect(result, 'https://tiles.example.com/1/2/3.pbf?key=abc');
      expect(result, contains('?key=abc'));
    });

    test('returned URL contains no residual template tokens', () {
      final result = buildUpstreamRedirect(
        'https://tiles.example.com/{z}/{x}/{y}.pbf',
        z: 7,
        x: 8,
        y: 9,
      );
      expect(result, isNot(contains('{z}')));
      expect(result, isNot(contains('{x}')));
      expect(result, isNot(contains('{y}')));
    });

    test('returns null for a null template', () {
      expect(buildUpstreamRedirect(null, z: 1, x: 1, y: 1), isNull);
    });

    test('returns null for an empty template', () {
      expect(buildUpstreamRedirect('', z: 1, x: 1, y: 1), isNull);
    });

    test('returns null for a template missing {y}', () {
      final result = buildUpstreamRedirect(
        'https://tiles.example.com/{z}/{x}/fixed.pbf',
        z: 1,
        x: 1,
        y: 1,
      );
      expect(result, isNull);
    });

    test('returns null for a loopback template', () {
      final result = buildUpstreamRedirect(
        'http://127.0.0.1:8080/{z}/{x}/{y}.pbf',
        z: 1,
        x: 1,
        y: 1,
      );
      expect(result, isNull);
    });
  });
}
