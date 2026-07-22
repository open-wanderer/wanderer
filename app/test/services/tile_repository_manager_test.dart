import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart' show LngLatBounds;
import 'package:wanderer/services/tile_repository_manager.dart';

// ---------------------------------------------------------------------------
// Tests for tile_repository_manager's pure resumePlanFor/bboxOverlaps
// helpers only.
//
// startVectorDownload/startDemDownload/pauseRegion/resumeRegion/
// localTilePathsForBounds/deleteRegion require a live Store + network and are
// exercised by Plan 06's on-device checkpoint instead, matching this
// codebase's precedent that upsertCatalog has no unit test.
// ---------------------------------------------------------------------------

void main() {
  group('resumePlanFor', () {
    test('0 existing bytes -> fresh write, no range, deleteOnError true', () {
      final plan = resumePlanFor(0);

      expect(plan.offset, 0);
      expect(plan.mode, FileAccessMode.write);
      expect(plan.sendRange, isFalse);
      expect(plan.deleteOnError, isTrue);
    });

    test(
      '>0 existing bytes -> resumed append, range, deleteOnError false',
      () {
        const existingBytes = 1048576;

        final plan = resumePlanFor(existingBytes);

        expect(plan.offset, existingBytes);
        expect(plan.mode, FileAccessMode.append);
        expect(plan.sendRange, isTrue);
        expect(plan.deleteOnError, isFalse);
      },
    );
  });

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
}
