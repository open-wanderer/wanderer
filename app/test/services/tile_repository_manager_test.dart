import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart' show LngLatBounds;
import 'package:wanderer/entities/downloaded_tile_package_entity.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/services/tile_repository_manager.dart';

// ---------------------------------------------------------------------------
// Tests for tile_repository_manager's pure bboxOverlaps and
// splitRegionTilePaths helpers only.
//
// startVectorDownload/startDemDownload/cancelVectorDownload/
// cancelDemDownload/deleteRegion require a live Store + network and are
// exercised by the on-device harness instead, matching this codebase's
// precedent that upsertCatalog has no unit test. splitRegionTilePaths (the
// pure per-region body localTilePathsForBounds delegates to) takes an
// Iterable<RegionEntity> instead of a Store, so its vector/DEM-split
// contract IS unit-testable in memory (RENDER-01).
// ---------------------------------------------------------------------------

void main() {
  group('bboxOverlaps', () {
    test('overlapping region and query bounds -> true', () {
      final overlaps = bboxOverlaps(
        minLon: 0,
        minLat: 0,
        maxLon: 10,
        maxLat: 10,
        query: const LngLatBounds(
          longitudeWest: 5,
          longitudeEast: 15,
          latitudeSouth: 5,
          latitudeNorth: 15,
        ),
      );

      expect(overlaps, isTrue);
    });

    test('disjoint region and query bounds -> false', () {
      final overlaps = bboxOverlaps(
        minLon: 0,
        minLat: 0,
        maxLon: 10,
        maxLat: 10,
        query: const LngLatBounds(
          longitudeWest: 20,
          longitudeEast: 30,
          latitudeSouth: 20,
          latitudeNorth: 30,
        ),
      );

      expect(overlaps, isFalse);
    });

    test('edge-touching region and query bounds -> true', () {
      final overlaps = bboxOverlaps(
        minLon: 0,
        minLat: 0,
        maxLon: 10,
        maxLat: 10,
        query: const LngLatBounds(
          longitudeWest: 10,
          longitudeEast: 20,
          latitudeSouth: 10,
          latitudeNorth: 20,
        ),
      );

      expect(overlaps, isTrue);
    });
  });

  group('splitRegionTilePaths', () {
    // Overlaps the region built by `_region()` (bbox 0,0 -> 10,10).
    const insideQuery = LngLatBounds(
      longitudeWest: 1,
      longitudeEast: 9,
      latitudeSouth: 1,
      latitudeNorth: 9,
    );
    // Disjoint from `_region()`'s bbox.
    const outsideQuery = LngLatBounds(
      longitudeWest: 20,
      longitudeEast: 30,
      latitudeSouth: 20,
      latitudeNorth: 30,
    );

    RegionEntity region({String id = 'r1'}) => RegionEntity(
      path: id,
      id: 'rec-$id',
      name: id,
      minLon: 0,
      minLat: 0,
      maxLon: 10,
      maxLat: 10,
    );

    test(
      'vector-only region -> path lands in vectorPaths, demPaths stays empty',
      () {
        final r = region()
          ..vectorPackage.target = DownloadedTilePackageEntity(
            localFilePath: '/abs/vector.pmtiles',
          );

        final result = splitRegionTilePaths([r], insideQuery);

        expect(result.vectorPaths, ['/abs/vector.pmtiles']);
        expect(result.demPaths, isEmpty);
      },
    );

    test(
      'DEM-only region -> path lands in demPaths, NEVER vectorPaths '
      '(the anti-conflation guarantee this plan exists for)',
      () {
        final r = region()
          ..demPackage.target = DownloadedTilePackageEntity(
            localFilePath: '/abs/dem.pmtiles',
          );

        final result = splitRegionTilePaths([r], insideQuery);

        expect(result.demPaths, ['/abs/dem.pmtiles']);
        expect(result.vectorPaths, isEmpty);
      },
    );

    test('region with both vector and DEM packages -> each in its own list', () {
      final r = region()
        ..vectorPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/vector.pmtiles',
        )
        ..demPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/dem.pmtiles',
        );

      final result = splitRegionTilePaths([r], insideQuery);

      expect(result.vectorPaths, ['/abs/vector.pmtiles']);
      expect(result.demPaths, ['/abs/dem.pmtiles']);
    });

    test('non-overlapping region contributes nothing to either list', () {
      final r = region()
        ..vectorPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/vector.pmtiles',
        )
        ..demPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/dem.pmtiles',
        );

      final result = splitRegionTilePaths([r], outsideQuery);

      expect(result.vectorPaths, isEmpty);
      expect(result.demPaths, isEmpty);
    });

    test('region with null package targets contributes nothing', () {
      final r = region();

      final result = splitRegionTilePaths([r], insideQuery);

      expect(result.vectorPaths, isEmpty);
      expect(result.demPaths, isEmpty);
    });

    test('two overlapping regions accumulate into the same lists', () {
      final r1 = region(id: 'r1')
        ..vectorPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/v1.pmtiles',
        );
      final r2 = region(id: 'r2')
        ..demPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/d2.pmtiles',
        );

      final result = splitRegionTilePaths([r1, r2], insideQuery);

      expect(result.vectorPaths, ['/abs/v1.pmtiles']);
      expect(result.demPaths, ['/abs/d2.pmtiles']);
    });
  });

  group('resolveRegionForTile', () {
    // A tile bbox inside every region built below.
    const tileBounds = LngLatBounds(
      longitudeWest: 4,
      longitudeEast: 6,
      latitudeSouth: 4,
      latitudeNorth: 6,
    );
    // Disjoint from every region built below.
    const outsideTileBounds = LngLatBounds(
      longitudeWest: 50,
      longitudeEast: 51,
      latitudeSouth: 50,
      latitudeNorth: 51,
    );

    RegionEntity region({
      required String id,
      double minLon = 0,
      double minLat = 0,
      double maxLon = 10,
      double maxLat = 10,
    }) => RegionEntity(
      path: id,
      id: 'rec-$id',
      name: id,
      minLon: minLon,
      minLat: minLat,
      maxLon: maxLon,
      maxLat: maxLat,
    );

    test('smallest-bbox-area overlapping region wins (D-02)', () {
      final big = region(id: 'big', minLon: -50, minLat: -50, maxLon: 50, maxLat: 50)
        ..vectorPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/big.pmtiles',
        );
      final small = region(id: 'small', minLon: 0, minLat: 0, maxLon: 10, maxLat: 10)
        ..vectorPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/small.pmtiles',
        );

      final winner = resolveRegionForTile([big, small], tileBounds, dem: false);

      expect(winner?.path, 'small');
    });

    test('a region overlapping the tile with a null package target is not a candidate', () {
      final noPackage = region(id: 'no-package'); // vectorPackage.target stays null
      final withPackage =
          region(id: 'with-package', minLon: -50, minLat: -50, maxLon: 50, maxLat: 50)
            ..vectorPackage.target = DownloadedTilePackageEntity(
              localFilePath: '/abs/with-package.pmtiles',
            );

      final winner =
          resolveRegionForTile([noPackage, withPackage], tileBounds, dem: false);

      expect(winner?.path, 'with-package');
    });

    test('equal-area overlapping candidates: most-recent downloadedAtUtc wins (D-03)', () {
      final older = region(id: 'older')
        ..vectorPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/older.pmtiles',
          downloadedAtUtc: DateTime.utc(2026, 1, 1),
        );
      final newer = region(id: 'newer')
        ..vectorPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/newer.pmtiles',
          downloadedAtUtc: DateTime.utc(2026, 6, 1),
        );

      final winner = resolveRegionForTile([older, newer], tileBounds, dem: false);

      expect(winner?.path, 'newer');
    });

    test('equal-area tie: a null downloadedAtUtc sorts as epoch-zero and loses', () {
      final undated = region(id: 'undated')
        ..vectorPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/undated.pmtiles',
        );
      final dated = region(id: 'dated')
        ..vectorPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/dated.pmtiles',
          downloadedAtUtc: DateTime.utc(2020, 1, 1),
        );

      final winner = resolveRegionForTile([undated, dated], tileBounds, dem: false);

      expect(winner?.path, 'dated');
    });

    test('dem:true resolves against demPackage; a vector-only region is not a DEM candidate', () {
      final vectorOnly = region(id: 'vector-only')
        ..vectorPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/vector-only.pmtiles',
        );
      final demRegion = region(id: 'dem-region', minLon: -50, minLat: -50, maxLon: 50, maxLat: 50)
        ..demPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/dem-region.pmtiles',
        );

      final winner =
          resolveRegionForTile([vectorOnly, demRegion], tileBounds, dem: true);

      expect(winner?.path, 'dem-region');
    });

    test('dem:false resolves against vectorPackage; a DEM-only region is not a vector candidate', () {
      final demOnly = region(id: 'dem-only')
        ..demPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/dem-only.pmtiles',
        );

      final winner = resolveRegionForTile([demOnly], tileBounds, dem: false);

      expect(winner, isNull);
    });

    test('no overlapping downloaded region -> returns null', () {
      final r = region(id: 'r')
        ..vectorPackage.target = DownloadedTilePackageEntity(
          localFilePath: '/abs/r.pmtiles',
        );

      final winner =
          resolveRegionForTile([r], outsideTileBounds, dem: false);

      expect(winner, isNull);
    });
  });
}
