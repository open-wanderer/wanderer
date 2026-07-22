import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/region_download_state.dart';
import 'package:wanderer/models/region_status.dart';

void main() {
  group('RegionDownloadState', () {
    test('default state reads as notDownloaded with null progress', () {
      const state = RegionDownloadState();

      expect(state.status, RegionStatus.notDownloaded);
      expect(state.vectorProgress, isNull);
      expect(state.demProgress, isNull);
    });

    test('equal when status and progress match', () {
      const a = RegionDownloadState(
        status: RegionStatus.downloading,
        vectorProgress: 0.5,
        demProgress: 0.25,
      );
      const b = RegionDownloadState(
        status: RegionStatus.downloading,
        vectorProgress: 0.5,
        demProgress: 0.25,
      );

      expect(a, equals(b));
      expect(a.hashCode, equals(b.hashCode));
    });

    test('unequal when progress differs', () {
      const a = RegionDownloadState(
        status: RegionStatus.downloading,
        vectorProgress: 0.5,
      );
      const b = RegionDownloadState(
        status: RegionStatus.downloading,
        vectorProgress: 0.6,
      );

      expect(a, isNot(equals(b)));
    });
  });
}
