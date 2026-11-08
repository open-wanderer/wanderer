import 'dart:math';

import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/services/tile_proxy_identity.dart';

// ---------------------------------------------------------------------------
// Tests for the pure half of tile_proxy_identity.dart only (minting +
// validation). This repo's tests deliberately avoid live ObjectBox stores
// (see splitRegionTilePaths' precedent in tile_repository_manager_test.dart)
// — the store-backed half (resolveTileProxyIdentity / persistTileProxyPort)
// is exercised indirectly wherever a live Store is available.
// ---------------------------------------------------------------------------

void main() {
  group('mintTileProxyIdentity', () {
    test('minted ports always fall inside the constant range', () {
      final random = Random(1234);
      for (var i = 0; i < 200; i++) {
        final identity = mintTileProxyIdentity(random: random);
        expect(identity.port, greaterThanOrEqualTo(kTileProxyPortMin));
        expect(identity.port, lessThanOrEqualTo(kTileProxyPortMax));
      }
    });

    test('minted secrets always satisfy isValidProxySecret', () {
      final random = Random(1234);
      for (var i = 0; i < 200; i++) {
        final identity = mintTileProxyIdentity(random: random);
        expect(isValidProxySecret(identity.secret), isTrue);
      }
    });

    test('minted secrets are 32 characters, lowercase-hex-only', () {
      final identity = mintTileProxyIdentity(random: Random(1234));
      expect(identity.secret.length, 32);
      expect(RegExp(r'^[0-9a-f]{32}$').hasMatch(identity.secret), isTrue);
    });

    test('two draws from independent seeds differ', () {
      final a = mintTileProxyIdentity(random: Random(1));
      final b = mintTileProxyIdentity(random: Random(2));
      expect(a.secret, isNot(equals(b.secret)));
    });
  });

  group('isValidProxySecret', () {
    test('rejects the empty string', () {
      expect(isValidProxySecret(''), isFalse);
    });

    test('rejects a 31-char string', () {
      expect(isValidProxySecret('a' * 31), isFalse);
    });

    test('rejects a 33-char string', () {
      expect(isValidProxySecret('a' * 33), isFalse);
    });

    test('rejects an uppercase-hex string', () {
      expect(isValidProxySecret('A' * 32), isFalse);
    });

    test('rejects a string containing / (path-segment injection)', () {
      expect(isValidProxySecret('${'a' * 16}/${'a' * 15}'), isFalse);
    });

    test('rejects a string containing . (path-segment injection)', () {
      expect(isValidProxySecret('${'a' * 15}..${'a' * 15}'), isFalse);
    });

    test('accepts a well-formed 32-char lowercase-hex string', () {
      expect(isValidProxySecret('0' * 32), isTrue);
      expect(isValidProxySecret('abcdef0123456789' * 2), isTrue);
    });
  });
}
