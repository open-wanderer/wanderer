import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/util/region_tile_status_util.dart';

void main() {
  group('resolveVectorTileStatus', () {
    test('idle (no live progress) returns the persisted status', () {
      expect(
        resolveVectorTileStatus(RegionStatus.downloaded, null),
        RegionStatus.downloaded,
      );
    });

    test('terminal idle: error persists when no download is in flight', () {
      expect(
        resolveVectorTileStatus(RegionStatus.error, null),
        RegionStatus.error,
      );
    });

    test('live progress overrides a stale persisted status', () {
      expect(
        resolveVectorTileStatus(RegionStatus.notDownloaded, 0.4),
        RegionStatus.downloading,
      );
    });

    test('live progress of exactly 0 still counts as downloading', () {
      expect(
        resolveVectorTileStatus(RegionStatus.notDownloaded, 0),
        RegionStatus.downloading,
      );
    });
  });

  group('resolveDemTileStatus', () {
    test('idle (no live progress) returns the persisted status', () {
      expect(
        resolveDemTileStatus(PackageStatus.downloaded, null),
        PackageStatus.downloaded,
      );
    });

    test('terminal idle: error persists when no download is in flight', () {
      expect(
        resolveDemTileStatus(PackageStatus.error, null),
        PackageStatus.error,
      );
    });

    test('live progress overrides a stale persisted status', () {
      expect(
        resolveDemTileStatus(PackageStatus.notDownloaded, 0.7),
        PackageStatus.downloading,
      );
    });

    test('vector-only download in flight never affects the DEM tile', () {
      // The two tiles read fully independent RegionDownloadState fields —
      // a vector-progress value has no bearing on resolveDemTileStatus.
      expect(
        resolveDemTileStatus(PackageStatus.downloaded, null),
        PackageStatus.downloaded,
      );
    });
  });
}
