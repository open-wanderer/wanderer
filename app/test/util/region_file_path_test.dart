import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;
import 'package:wanderer/util/region_file_path.dart';

// ---------------------------------------------------------------------------
// Tests for the region tile repository's path-safety helpers. This is the
// load-bearing security control: a catalog region id must derive ONLY from
// the backend's own `^[a-z0-9][a-z0-9_-]*$` allow-list, rejecting any `..`
// traversal, uppercase, or foreign character before it becomes a filesystem
// path — every resulting path stays rooted at the supplied root.
// ---------------------------------------------------------------------------

void main() {
  const root = '/docs';

  group('isValidRegionId', () {
    test('accepts lowercase alphanumeric ids with hyphen/underscore', () {
      expect(isValidRegionId('de-nrw'), isTrue);
      expect(isValidRegionId('de_bay_1'), isTrue);
      expect(isValidRegionId('a'), isTrue);
      expect(isValidRegionId('a1b2'), isTrue);
    });

    test('rejects traversal, uppercase, empty, and malformed ids', () {
      expect(isValidRegionId('../etc'), isFalse);
      expect(isValidRegionId('..'), isFalse);
      expect(isValidRegionId(''), isFalse);
      expect(isValidRegionId('De-NRW'), isFalse);
      expect(isValidRegionId('a/b'), isFalse);
      expect(isValidRegionId('a b'), isFalse);
      expect(isValidRegionId('-de-nrw'), isFalse);
      expect(isValidRegionId('_de-nrw'), isFalse);
    });
  });

  group('assertValidRegionId', () {
    test('returns the id unchanged when valid', () {
      expect(assertValidRegionId('de-nrw'), 'de-nrw');
    });

    test('throws ArgumentError for an invalid id', () {
      expect(() => assertValidRegionId('../evil'), throwsArgumentError);
      expect(() => assertValidRegionId(''), throwsArgumentError);
      expect(() => assertValidRegionId('De-NRW'), throwsArgumentError);
    });
  });

  group('isValidRegionPath', () {
    test('accepts dotted materialized paths the id validator rejects', () {
      expect(isValidRegionPath('canada.alberta.south'), isTrue);
      expect(isValidRegionPath('germany.north_rhine_westphalia'), isTrue);
      expect(isValidRegionPath("people's_republic_of_china"), isTrue);
      expect(isValidRegionPath('a'), isTrue);
      // A plain record-id-shaped value is still a valid path.
      expect(isValidRegionPath('de-nrw'), isTrue);
    });

    test('rejects traversal, uppercase, empty, and leading-separator paths', () {
      expect(isValidRegionPath('../etc'), isFalse);
      expect(isValidRegionPath('a..b'), isFalse);
      expect(isValidRegionPath('..'), isFalse);
      expect(isValidRegionPath(''), isFalse);
      expect(isValidRegionPath('Canada.Alberta'), isFalse);
      expect(isValidRegionPath('a/b'), isFalse);
      expect(isValidRegionPath('.canada'), isFalse);
    });
  });

  group('assertValidRegionPath', () {
    test('returns the path unchanged when valid', () {
      expect(
        assertValidRegionPath('canada.alberta.south'),
        'canada.alberta.south',
      );
    });

    test('throws ArgumentError for an invalid path', () {
      expect(() => assertValidRegionPath('../evil'), throwsArgumentError);
      expect(() => assertValidRegionPath('a..b'), throwsArgumentError);
      expect(() => assertValidRegionPath(''), throwsArgumentError);
      expect(
        () => assertValidRegionPath('Canada.Alberta'),
        throwsArgumentError,
      );
    });
  });

  group('regionStorageDir', () {
    test('builds a root-rooted region directory', () {
      final dir = regionStorageDir(root, 'de-nrw');
      expect(dir, p.join(root, 'regions', 'de-nrw'));
      expect(p.isWithin(root, dir), isTrue);
    });

    test('throws ArgumentError for a traversal id, no path returned', () {
      expect(() => regionStorageDir(root, '../evil'), throwsArgumentError);
    });
  });

  group('regionVectorPath', () {
    test('builds a root-rooted vector.pmtiles path', () {
      final path = regionVectorPath(root, 'de-nrw');
      expect(
        path,
        p.join(root, 'regions', 'de-nrw', 'vector.pmtiles'),
      );
      expect(p.isWithin(root, path), isTrue);
      expect(path.endsWith('vector.pmtiles'), isTrue);
    });

    test('throws ArgumentError for a traversal id, no path returned', () {
      expect(() => regionVectorPath(root, '../evil'), throwsArgumentError);
    });
  });

  group('regionDemPath', () {
    test('builds a root-rooted dem.pmtiles path under the same region dir', () {
      final path = regionDemPath(root, 'de-nrw');
      expect(
        path,
        p.join(root, 'regions', 'de-nrw', 'dem.pmtiles'),
      );
      expect(p.isWithin(root, path), isTrue);
      expect(path.endsWith('dem.pmtiles'), isTrue);
      expect(
        p.dirname(path),
        p.dirname(regionVectorPath(root, 'de-nrw')),
      );
    });

    test('throws ArgumentError for a traversal id, no path returned', () {
      expect(() => regionDemPath(root, '../evil'), throwsArgumentError);
    });
  });
}
