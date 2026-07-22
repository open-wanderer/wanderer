import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/region_status.dart';

void main() {
  group('CatalogStatus.code', () {
    test('explicit code values', () {
      expect(CatalogStatus.building.code, 0);
      expect(CatalogStatus.ready.code, 1);
      expect(CatalogStatus.error.code, 2);
      expect(CatalogStatus.absent.code, 3);
    });

    test('decoding by code resolves the matching member', () {
      expect(
        CatalogStatus.values.firstWhere((s) => s.code == 0),
        CatalogStatus.building,
      );
      expect(
        CatalogStatus.values.firstWhere((s) => s.code == 1),
        CatalogStatus.ready,
      );
      expect(
        CatalogStatus.values.firstWhere((s) => s.code == 2),
        CatalogStatus.error,
      );
      expect(
        CatalogStatus.values.firstWhere((s) => s.code == 3),
        CatalogStatus.absent,
      );
    });
  });

  group('RegionStatus.code', () {
    test('explicit code values', () {
      expect(RegionStatus.notDownloaded.code, 0);
      expect(RegionStatus.downloading.code, 1);
      expect(RegionStatus.downloaded.code, 2);
      expect(RegionStatus.updateAvailable.code, 3);
    });

    test('decoding by code resolves the matching member', () {
      expect(
        RegionStatus.values.firstWhere((s) => s.code == 2),
        RegionStatus.downloaded,
      );
      expect(
        RegionStatus.values.firstWhere((s) => s.code == 3),
        RegionStatus.updateAvailable,
      );
    });
  });

  group('PackageStatus.code', () {
    test('explicit code values', () {
      expect(PackageStatus.notDownloaded.code, 0);
      expect(PackageStatus.downloading.code, 1);
      expect(PackageStatus.downloaded.code, 2);
    });

    test('decoding by code resolves the matching member', () {
      expect(
        PackageStatus.values.firstWhere((s) => s.code == 1),
        PackageStatus.downloading,
      );
    });
  });
}
