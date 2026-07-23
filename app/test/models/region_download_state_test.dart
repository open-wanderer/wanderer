import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/region_download_state.dart';

void main() {
  group('RegionDownloadState', () {
    test('default state has null vector/DEM progress', () {
      const state = RegionDownloadState();

      expect(state.vectorProgress, isNull);
      expect(state.demProgress, isNull);
    });

    test('equal when both progress fields match', () {
      const a = RegionDownloadState(vectorProgress: 0.5, demProgress: 0.25);
      const b = RegionDownloadState(vectorProgress: 0.5, demProgress: 0.25);

      expect(a, equals(b));
      expect(a.hashCode, equals(b.hashCode));
    });

    test('unequal when vectorProgress differs', () {
      const a = RegionDownloadState(vectorProgress: 0.5);
      const b = RegionDownloadState(vectorProgress: 0.6);

      expect(a, isNot(equals(b)));
    });

    test('vector and DEM progress are independent of each other', () {
      const a = RegionDownloadState(vectorProgress: 0.5);
      final b = a.copyWith(demProgress: 0.1);

      expect(b.vectorProgress, 0.5);
      expect(b.demProgress, 0.1);
    });
  });
}
