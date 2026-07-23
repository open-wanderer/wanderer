import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/region_download_state.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/util/region_row_status_util.dart';

void main() {
  group('resolveRowStatus', () {
    test('idle (no mutation in flight) returns the persisted status', () {
      expect(
        resolveRowStatus(RegionStatus.downloaded, null),
        RegionStatus.downloaded,
      );
    });

    test('terminal idle: error persists when no mutation is in flight', () {
      expect(resolveRowStatus(RegionStatus.error, null), RegionStatus.error);
    });

    test('vector download in flight returns the live downloading status', () {
      const live = RegionDownloadState(
        status: RegionStatus.downloading,
        vectorProgress: 0,
      );
      expect(
        resolveRowStatus(RegionStatus.notDownloaded, live),
        RegionStatus.downloading,
      );
    });

    test('resume in flight (no progress yet) returns live downloading', () {
      const live = RegionDownloadState(status: RegionStatus.downloading);
      expect(
        resolveRowStatus(RegionStatus.paused, live),
        RegionStatus.downloading,
      );
    });

    test('DEM-only download keeps a downloaded row at downloaded', () {
      const live = RegionDownloadState(
        status: RegionStatus.downloading,
        demProgress: 0,
      );
      expect(
        resolveRowStatus(RegionStatus.downloaded, live),
        RegionStatus.downloaded,
      );
    });

    test('DEM-only download keeps a not-downloaded row at notDownloaded', () {
      const live = RegionDownloadState(
        status: RegionStatus.downloading,
        demProgress: 0,
      );
      expect(
        resolveRowStatus(RegionStatus.notDownloaded, live),
        RegionStatus.notDownloaded,
      );
    });
  });
}
