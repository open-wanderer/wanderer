import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/downloaded_tile_package_entity.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/util/trail_coverage_util.dart';

/// Table-driven unit tests for the pure trail/region coverage functions
/// (GUARD-01/GUARD-04). Mirrors the group/test style of
/// `region_tile_status_util_test.dart`.
void main() {
  Trail trailWithBbox({
    required double minLon,
    required double minLat,
    required double maxLon,
    required double maxLat,
  }) {
    return Trail.empty().copyWith(
      minLon: minLon,
      minLat: minLat,
      maxLon: maxLon,
      maxLat: maxLat,
    );
  }

  RegionEntity regionWithBbox({
    required String id,
    required double minLon,
    required double minLat,
    required double maxLon,
    required double maxLat,
  }) {
    return RegionEntity(
      id: id,
      name: id,
      minLon: minLon,
      minLat: minLat,
      maxLon: maxLon,
      maxLat: maxLat,
    );
  }

  RegionEntity downloadedRegion(RegionEntity region) {
    region.vectorPackage.target = DownloadedTilePackageEntity(
      status: PackageStatus.downloaded,
    );
    return region;
  }

  group('bboxesOverlap', () {
    test('true when rectangles clearly overlap', () {
      expect(
        bboxesOverlap(
          aMinLon: 0,
          aMinLat: 0,
          aMaxLon: 10,
          aMaxLat: 10,
          bMinLon: 5,
          bMinLat: 5,
          bMaxLon: 15,
          bMaxLat: 15,
        ),
        true,
      );
    });

    test('false when rectangles are disjoint', () {
      expect(
        bboxesOverlap(
          aMinLon: 0,
          aMinLat: 0,
          aMaxLon: 10,
          aMaxLat: 10,
          bMinLon: 20,
          bMinLat: 20,
          bMaxLon: 30,
          bMaxLat: 30,
        ),
        false,
      );
    });

    test('true when rectangles only touch at an edge', () {
      expect(
        bboxesOverlap(
          aMinLon: 0,
          aMinLat: 0,
          aMaxLon: 10,
          aMaxLat: 10,
          bMinLon: 10,
          bMinLat: 0,
          bMaxLon: 20,
          bMaxLat: 10,
        ),
        true,
      );
    });

    test(
      'degenerate bbox (min > max) returns false without throwing',
      () {
        expect(
          () => bboxesOverlap(
            aMinLon: 10,
            aMinLat: 10,
            aMaxLon: 0, // degenerate: min > max
            aMaxLat: 0,
            bMinLon: 100,
            bMinLat: 100,
            bMaxLon: 120,
            bMaxLat: 120,
          ),
          returnsNormally,
        );
        expect(
          bboxesOverlap(
            aMinLon: 10,
            aMinLat: 10,
            aMaxLon: 0,
            aMaxLat: 0,
            bMinLon: 100,
            bMinLat: 100,
            bMaxLon: 120,
            bMaxLat: 120,
          ),
          false,
        );
      },
    );
  });

  group('overlappingRegions', () {
    test('returns every catalog region whose bbox overlaps the trail, '
        'regardless of status', () {
      final trail = trailWithBbox(
        minLon: 0,
        minLat: 0,
        maxLon: 10,
        maxLat: 10,
      );
      final overlapping = regionWithBbox(
        id: 'overlap',
        minLon: 5,
        minLat: 5,
        maxLon: 15,
        maxLat: 15,
      );
      final disjoint = regionWithBbox(
        id: 'disjoint',
        minLon: 50,
        minLat: 50,
        maxLon: 60,
        maxLat: 60,
      );

      final result = overlappingRegions(trail, [overlapping, disjoint]);

      expect(result.map((r) => r.id), ['overlap']);
    });

    test('empty catalog yields empty result, no throw', () {
      final trail = trailWithBbox(
        minLon: 0,
        minLat: 0,
        maxLon: 10,
        maxLat: 10,
      );

      expect(overlappingRegions(trail, []), isEmpty);
    });
  });

  group('missingCoverageRegions', () {
    test(
      'fully-covered case: overlapping-only downloaded/updateAvailable '
      'regions -> missing is empty but overlapping is non-empty',
      () {
        final trail = trailWithBbox(
          minLon: 0,
          minLat: 0,
          maxLon: 10,
          maxLat: 10,
        );
        final covered = downloadedRegion(
          regionWithBbox(
            id: 'covered',
            minLon: 5,
            minLat: 5,
            maxLon: 15,
            maxLat: 15,
          ),
        );

        final catalog = [covered];

        expect(overlappingRegions(trail, catalog), isNotEmpty);
        expect(missingCoverageRegions(trail, catalog), isEmpty);
      },
    );

    test(
      'no-region-gap case: trail overlaps no catalog region at all -> '
      'BOTH overlappingRegions and missingCoverageRegions are empty',
      () {
        final trail = trailWithBbox(
          minLon: 0,
          minLat: 0,
          maxLon: 10,
          maxLat: 10,
        );
        final farAway = regionWithBbox(
          id: 'far',
          minLon: 100,
          minLat: 100,
          maxLon: 110,
          maxLat: 110,
        );

        final catalog = [farAway];

        expect(overlappingRegions(trail, catalog), isEmpty);
        expect(missingCoverageRegions(trail, catalog), isEmpty);
      },
    );

    test('notDownloaded overlapping region is included in missing', () {
      final trail = trailWithBbox(
        minLon: 0,
        minLat: 0,
        maxLon: 10,
        maxLat: 10,
      );
      final notDownloaded = regionWithBbox(
        id: 'not-downloaded',
        minLon: 5,
        minLat: 5,
        maxLon: 15,
        maxLat: 15,
      );

      final result = missingCoverageRegions(trail, [notDownloaded]);

      expect(result.map((r) => r.id), ['not-downloaded']);
    });

    test(
      'updateAvailable region is EXCLUDED from missingCoverageRegions '
      '(GUARD-04)',
      () {
        final trail = trailWithBbox(
          minLon: 0,
          minLat: 0,
          maxLon: 10,
          maxLat: 10,
        );
        final stale = regionWithBbox(
          id: 'stale',
          minLon: 5,
          minLat: 5,
          maxLon: 15,
          maxLat: 15,
        )..version = 'v2';
        stale.vectorPackage.target = DownloadedTilePackageEntity(
          status: PackageStatus.downloaded,
        );
        stale.catalogStatus = CatalogStatus.ready;
        stale.lastDownloadedVersion = 'v1';

        // Sanity check the fixture actually resolves to updateAvailable.
        expect(stale.status, RegionStatus.updateAvailable);

        final result = missingCoverageRegions(trail, [stale]);

        expect(result, isEmpty);
      },
    );

    test('empty catalog -> empty missing list, no throw', () {
      final trail = trailWithBbox(
        minLon: 0,
        minLat: 0,
        maxLon: 10,
        maxLat: 10,
      );

      expect(missingCoverageRegions(trail, []), isEmpty);
    });
  });
}
