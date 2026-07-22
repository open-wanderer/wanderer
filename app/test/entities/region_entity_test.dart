import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/downloaded_tile_package_entity.dart';
import 'package:wanderer/models/region_status.dart';

void main() {
  group('DownloadedTilePackageEntity.dbStatus', () {
    test('dbStatus getter returns status.code for every member', () {
      for (final s in PackageStatus.values) {
        final entity = DownloadedTilePackageEntity(status: s);
        expect(entity.dbStatus, s.code);
      }
    });

    test('setting dbStatus to 0/1/2 restores the matching member', () {
      final entity = DownloadedTilePackageEntity();

      entity.dbStatus = 0;
      expect(entity.status, PackageStatus.notDownloaded);

      entity.dbStatus = 1;
      expect(entity.status, PackageStatus.downloading);

      entity.dbStatus = 2;
      expect(entity.status, PackageStatus.downloaded);
    });

    test('an out-of-range dbStatus falls back to notDownloaded', () {
      final entity = DownloadedTilePackageEntity();

      entity.dbStatus = 99;

      expect(entity.status, PackageStatus.notDownloaded);
    });
  });

  group('DownloadedTilePackageEntity defaults', () {
    test('a default-constructed entity has notDownloaded and null fields', () {
      final entity = DownloadedTilePackageEntity();

      expect(entity.status, PackageStatus.notDownloaded);
      expect(entity.localFilePath, isNull);
      expect(entity.downloadedAtUtc, isNull);
      expect(entity.sizeBytesOnDisk, isNull);
    });
  });
}
