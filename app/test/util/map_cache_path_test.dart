import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;
import 'package:wanderer/util/map_cache_path.dart';

// ---------------------------------------------------------------------------
// Tests for the app-wide glyph/sprite cache path-safety helpers. This is the
// load-bearing security control: cache filenames must derive ONLY from the 4
// whitelisted fontstacks + numeric ranges, rejecting any `..` traversal, with
// every path rooted at the supplied cache root.
// ---------------------------------------------------------------------------

void main() {
  const root = '/docs/map_cache';

  group('isAllowedFontstack', () {
    test('accepts all 4 whitelisted fontstacks', () {
      expect(isAllowedFontstack('Noto Sans Regular'), isTrue);
      expect(isAllowedFontstack('Noto Sans Medium'), isTrue);
      expect(isAllowedFontstack('Noto Sans Italic'), isTrue);
      expect(isAllowedFontstack('Noto Sans Devanagari Regular v1'), isTrue);
    });

    test('rejects traversal, unknown, and empty fontstacks', () {
      expect(isAllowedFontstack('../etc/passwd'), isFalse);
      expect(isAllowedFontstack('Comic Sans'), isFalse);
      expect(isAllowedFontstack(''), isFalse);
      // Whitelist is exact-match — near-misses are rejected.
      expect(isAllowedFontstack('noto sans regular'), isFalse);
      expect(isAllowedFontstack('Noto Sans Regular '), isFalse);
    });
  });

  group('glyphCacheFilePath', () {
    test('builds a root-rooted path for a whitelisted fontstack + range', () {
      final path = glyphCacheFilePath(root, 'Noto Sans Regular', '0-255');
      expect(path, p.join(root, 'glyphs', 'Noto Sans Regular', '0-255.pbf'));
      // Every successful path is prefixed by the supplied root (no escape).
      expect(path.startsWith(root), isTrue);
      expect(p.isWithin(root, path), isTrue);
    });

    test('builds paths for higher numeric ranges', () {
      final path = glyphCacheFilePath(root, 'Noto Sans Medium', '65280-65535');
      expect(
        path,
        p.join(root, 'glyphs', 'Noto Sans Medium', '65280-65535.pbf'),
      );
      expect(p.isWithin(root, path), isTrue);
    });

    test('throws ArgumentError for a non-whitelisted fontstack', () {
      expect(
        () => glyphCacheFilePath(root, 'evil', '0-255'),
        throwsArgumentError,
      );
      expect(
        () => glyphCacheFilePath(root, '../../etc', '0-255'),
        throwsArgumentError,
      );
    });

    test('throws ArgumentError for a traversal range', () {
      expect(
        () => glyphCacheFilePath(root, 'Noto Sans Regular', '../../0-255'),
        throwsArgumentError,
      );
    });

    test('throws ArgumentError for a non-numeric range', () {
      expect(
        () => glyphCacheFilePath(root, 'Noto Sans Regular', '0-255.pbf'),
        throwsArgumentError,
      );
      expect(
        () => glyphCacheFilePath(root, 'Noto Sans Regular', 'latin'),
        throwsArgumentError,
      );
      expect(
        () => glyphCacheFilePath(root, 'Noto Sans Regular', ''),
        throwsArgumentError,
      );
      expect(
        () => glyphCacheFilePath(root, 'Noto Sans Regular', '0-255/../..'),
        throwsArgumentError,
      );
    });
  });

  group('spriteCacheBasePath', () {
    test('builds light + dark sprite bases under the cache root', () {
      final light = spriteCacheBasePath(root, dark: false);
      final dark = spriteCacheBasePath(root, dark: true);
      expect(light, p.join(root, 'sprite', 'light'));
      expect(dark, p.join(root, 'sprite', 'dark'));
      expect(p.isWithin(root, light), isTrue);
      expect(p.isWithin(root, dark), isTrue);
    });
  });
}
