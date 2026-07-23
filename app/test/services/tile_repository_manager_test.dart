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
      id: id,
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
}
