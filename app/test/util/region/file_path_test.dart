import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;
import 'package:wanderer/util/region/file_path.dart';

// ---------------------------------------------------------------------------
// Tests for the region tile repository's path-safety helpers. This is the
// load-bearing security control: a catalog region path must derive ONLY from
// the backend's own `^[a-z0-9][a-z0-9_.'-]*$` allow-list, rejecting any `..`
// traversal, uppercase, or foreign character before it becomes a filesystem
// path — every resulting path stays rooted at the supplied root.
//
// There is deliberately no region-*id* validator to test: the backend re-mints
// region record ids on every rebuild, so an id may never name a directory.
// ---------------------------------------------------------------------------

void main() {
  const root = '/docs';

  group('isValidRegionPath', () {
    test('accepts dotted materialized paths the id validator rejects', () {
      expect(isValidRegionPath('canada.alberta.south'), isTrue);
      expect(isValidRegionPath('germany.north_rhine_westphalia'), isTrue);
      expect(isValidRegionPath("people's_republic_of_china"), isTrue);
      expect(isValidRegionPath('a'), isTrue);
      // A plain record-id-shaped value is still a valid path.
      expect(isValidRegionPath('de-nrw'), isTrue);
    });

    test(
      'rejects traversal, uppercase, empty, and leading-separator paths',
      () {
        expect(isValidRegionPath('../etc'), isFalse);
        expect(isValidRegionPath('a..b'), isFalse);
        expect(isValidRegionPath('..'), isFalse);
        expect(isValidRegionPath(''), isFalse);
        expect(isValidRegionPath('Canada.Alberta'), isFalse);
        expect(isValidRegionPath('a/b'), isFalse);
        expect(isValidRegionPath('.canada'), isFalse);
      },
    );
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

  // A realistic materialized path — dotted, which the retired id allow-list
  // would have rejected outright.
  const regionPath = 'canada.canada_alberta.canada_alberta_south';

  group('regionStorageDir', () {
    test('builds a root-rooted region directory from a dotted path', () {
      final dir = regionStorageDir(root, regionPath);
      expect(dir, p.join(root, 'regions', regionPath));
      expect(p.isWithin(root, dir), isTrue);
    });

    test('throws ArgumentError for a traversal path, no path returned', () {
      expect(() => regionStorageDir(root, '../evil'), throwsArgumentError);
    });
  });

  group('regionVectorPath', () {
    test('builds a root-rooted vector.pmtiles path', () {
      final path = regionVectorPath(root, regionPath);
      expect(path, p.join(root, 'regions', regionPath, 'vector.pmtiles'));
      expect(p.isWithin(root, path), isTrue);
      expect(path.endsWith('vector.pmtiles'), isTrue);
    });

    test('throws ArgumentError for a traversal path, no path returned', () {
      expect(() => regionVectorPath(root, '../evil'), throwsArgumentError);
    });
  });

  group('regionDemPath', () {
    test('builds a root-rooted dem.pmtiles path under the same region dir', () {
      final path = regionDemPath(root, regionPath);
      expect(path, p.join(root, 'regions', regionPath, 'dem.pmtiles'));
      expect(p.isWithin(root, path), isTrue);
      expect(path.endsWith('dem.pmtiles'), isTrue);
      expect(p.dirname(path), p.dirname(regionVectorPath(root, regionPath)));
    });

    test('throws ArgumentError for a traversal path, no path returned', () {
      expect(() => regionDemPath(root, '../evil'), throwsArgumentError);
    });
  });
}
