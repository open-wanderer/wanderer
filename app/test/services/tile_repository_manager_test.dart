import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart' show LngLatBounds;
import 'package:wanderer/services/tile_repository_manager.dart';

// ---------------------------------------------------------------------------
// Tests for tile_repository_manager's pure bboxOverlaps helper only.
//
// startVectorDownload/startDemDownload/cancelVectorDownload/
// cancelDemDownload/localTilePathsForBounds/deleteRegion require a live
// Store + network and are exercised by the on-device harness instead,
// matching this codebase's precedent that upsertCatalog has no unit test.
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
}
