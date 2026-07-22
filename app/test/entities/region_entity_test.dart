import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/downloaded_tile_package_entity.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/models/region_catalog_entry.dart';
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

  group('RegionEntity.dbCatalogStatus / dbDemStatus', () {
    test('dbCatalogStatus round-trips CatalogStatus by .code', () {
      final entity = RegionEntity(id: 'de-nrw', name: 'NRW');

      for (final s in CatalogStatus.values) {
        entity.dbCatalogStatus = s.code;
        expect(entity.catalogStatus, s);
        expect(entity.dbCatalogStatus, s.code);
      }
    });

    test('out-of-range dbCatalogStatus falls back to building', () {
      final entity = RegionEntity(id: 'de-nrw', name: 'NRW');

      entity.dbCatalogStatus = 99;

      expect(entity.catalogStatus, CatalogStatus.building);
    });

    test('dbDemStatus round-trips CatalogStatus by .code', () {
      final entity = RegionEntity(id: 'de-nrw', name: 'NRW');

      for (final s in CatalogStatus.values) {
        entity.dbDemStatus = s.code;
        expect(entity.demStatus, s);
        expect(entity.dbDemStatus, s.code);
      }
    });

    test('out-of-range dbDemStatus falls back to absent', () {
      final entity = RegionEntity(id: 'de-nrw', name: 'NRW');

      entity.dbDemStatus = 99;

      expect(entity.demStatus, CatalogStatus.absent);
    });
  });

  group('RegionEntity.status getter', () {
    test('no vectorPackage target -> notDownloaded', () {
      final entity = RegionEntity(id: 'de-nrw', name: 'NRW');

      expect(entity.status, RegionStatus.notDownloaded);
    });

    test('vectorPackage downloading -> downloading', () {
      final entity = RegionEntity(id: 'de-nrw', name: 'NRW');
      entity.vectorPackage.target = DownloadedTilePackageEntity(
        status: PackageStatus.downloading,
      );

      expect(entity.status, RegionStatus.downloading);
    });

    test(
      'vectorPackage downloaded, catalogStatus != ready -> downloaded',
      () {
        final entity = RegionEntity(
          id: 'de-nrw',
          name: 'NRW',
          catalogStatus: CatalogStatus.building,
          version: '2026-07-01',
        );
        entity.lastDownloadedVersion = '2026-06-01';
        entity.vectorPackage.target = DownloadedTilePackageEntity(
          status: PackageStatus.downloaded,
        );

        expect(entity.status, RegionStatus.downloaded);
      },
    );

    test(
      'vectorPackage downloaded, version == lastDownloadedVersion -> downloaded',
      () {
        final entity = RegionEntity(
          id: 'de-nrw',
          name: 'NRW',
          catalogStatus: CatalogStatus.ready,
          version: '2026-07-01',
        );
        entity.lastDownloadedVersion = '2026-07-01';
        entity.vectorPackage.target = DownloadedTilePackageEntity(
          status: PackageStatus.downloaded,
        );

        expect(entity.status, RegionStatus.downloaded);
      },
    );

    test(
      'vectorPackage downloaded, catalogStatus == ready and version changed -> updateAvailable',
      () {
        final entity = RegionEntity(
          id: 'de-nrw',
          name: 'NRW',
          catalogStatus: CatalogStatus.ready,
          version: '2026-07-01',
        );
        entity.lastDownloadedVersion = '2026-06-01';
        entity.vectorPackage.target = DownloadedTilePackageEntity(
          status: PackageStatus.downloaded,
        );

        expect(entity.status, RegionStatus.updateAvailable);
      },
    );
  });

  group('RegionEntity.fromCatalogEntry', () {
    test('maps bbox/catalog fields and leaves local-only fields unset', () {
      final entry = RegionCatalogEntry(
        id: 'de-nrw',
        name: 'North Rhine-Westphalia',
        bbox: [5.9, 50.3, 9.5, 52.5],
        status: CatalogStatus.ready,
        version: '2026-07-01',
        vectorUrl: '/api/v1/regions/de-nrw/download',
        vectorSize: 123,
        demStatus: CatalogStatus.ready,
        demUrl: '/api/v1/regions/de-nrw/download-dem',
        demSize: 456,
      );

      final entity = RegionEntity.fromCatalogEntry(entry);

      expect(entity.id, 'de-nrw');
      expect(entity.name, 'North Rhine-Westphalia');
      expect(entity.minLon, 5.9);
      expect(entity.minLat, 50.3);
      expect(entity.maxLon, 9.5);
      expect(entity.maxLat, 52.5);
      expect(entity.catalogStatus, CatalogStatus.ready);
      expect(entity.demStatus, CatalogStatus.ready);
      expect(entity.version, '2026-07-01');
      expect(entity.vectorUrl, '/api/v1/regions/de-nrw/download');
      expect(entity.vectorSize, 123);
      expect(entity.demUrl, '/api/v1/regions/de-nrw/download-dem');
      expect(entity.demSize, 456);
      expect(entity.inCatalog, isTrue);
      expect(entity.lastDownloadedVersion, isNull);
      expect(entity.vectorPackage.target, isNull);
      expect(entity.demPackage.target, isNull);
    });

    test('missing demStatus maps to CatalogStatus.absent', () {
      final entry = RegionCatalogEntry(
        id: 'de-bay',
        name: 'Bavaria',
        bbox: [8.9, 47.2, 13.9, 50.6],
        status: CatalogStatus.building,
      );

      final entity = RegionEntity.fromCatalogEntry(entry);

      expect(entity.demStatus, CatalogStatus.absent);
    });

    test('throws FormatException on a malformed bbox', () {
      final entry = RegionCatalogEntry(
        id: 'de-bad',
        name: 'Bad',
        bbox: [1.0, 2.0, 3.0],
        status: CatalogStatus.building,
      );

      expect(
        () => RegionEntity.fromCatalogEntry(entry),
        throwsFormatException,
      );
    });
  });

  group('RegionEntity.applyCatalogEntry', () {
    test(
      'overwrites only catalog-owned fields, preserves obxId/links/lastDownloadedVersion',
      () {
        final entity = RegionEntity(id: 'de-nrw', name: 'Old Name');
        entity.obxId = 7;
        final vectorPkg = DownloadedTilePackageEntity(
          status: PackageStatus.downloaded,
        );
        entity.vectorPackage.target = vectorPkg;
        entity.lastDownloadedVersion = '2026-01-01';

        final entry = RegionCatalogEntry(
          id: 'de-nrw',
          name: 'New Name',
          bbox: [5.9, 50.3, 9.5, 52.5],
          status: CatalogStatus.ready,
          version: '2026-07-01',
        );

        entity.applyCatalogEntry(entry);

        expect(entity.name, 'New Name');
        expect(entity.version, '2026-07-01');
        expect(entity.catalogStatus, CatalogStatus.ready);
        expect(entity.obxId, 7);
        expect(entity.vectorPackage.target, same(vectorPkg));
        expect(entity.lastDownloadedVersion, '2026-01-01');
      },
    );

    test('throws FormatException on a malformed bbox', () {
      final entity = RegionEntity(id: 'de-nrw', name: 'NRW');
      final entry = RegionCatalogEntry(
        id: 'de-nrw',
        name: 'NRW',
        bbox: [1.0, 2.0],
        status: CatalogStatus.building,
      );

      expect(
        () => entity.applyCatalogEntry(entry),
        throwsFormatException,
      );
    });
  });
}
