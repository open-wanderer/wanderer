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

    test('setting dbStatus to 0/1/2/4 restores the matching member', () {
      final entity = DownloadedTilePackageEntity();

      entity.dbStatus = 0;
      expect(entity.status, PackageStatus.notDownloaded);

      entity.dbStatus = 1;
      expect(entity.status, PackageStatus.downloading);

      entity.dbStatus = 2;
      expect(entity.status, PackageStatus.downloaded);

      entity.dbStatus = 4;
      expect(entity.status, PackageStatus.error);
    });

    test(
      'the retired paused code (3) falls back to notDownloaded',
      () {
        final entity = DownloadedTilePackageEntity();
        entity.dbStatus = 3;
        expect(entity.status, PackageStatus.notDownloaded);
      },
    );

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
      final entity = RegionEntity(path: 'de.nrw', id: 'rec1', name: 'NRW');

      for (final s in CatalogStatus.values) {
        entity.dbCatalogStatus = s.code;
        expect(entity.catalogStatus, s);
        expect(entity.dbCatalogStatus, s.code);
      }
    });

    test('out-of-range dbCatalogStatus falls back to building', () {
      final entity = RegionEntity(path: 'de.nrw', id: 'rec1', name: 'NRW');

      entity.dbCatalogStatus = 99;

      expect(entity.catalogStatus, CatalogStatus.building);
    });

    test('dbDemStatus round-trips CatalogStatus by .code', () {
      final entity = RegionEntity(path: 'de.nrw', id: 'rec1', name: 'NRW');

      for (final s in CatalogStatus.values) {
        entity.dbDemStatus = s.code;
        expect(entity.demStatus, s);
        expect(entity.dbDemStatus, s.code);
      }
    });

    test('out-of-range dbDemStatus falls back to absent', () {
      final entity = RegionEntity(path: 'de.nrw', id: 'rec1', name: 'NRW');

      entity.dbDemStatus = 99;

      expect(entity.demStatus, CatalogStatus.absent);
    });
  });

  group('RegionEntity.status getter', () {
    test('no vectorPackage target -> notDownloaded', () {
      final entity = RegionEntity(path: 'de.nrw', id: 'rec1', name: 'NRW');

      expect(entity.status, RegionStatus.notDownloaded);
    });

    test('vectorPackage downloading -> downloading', () {
      final entity = RegionEntity(path: 'de.nrw', id: 'rec1', name: 'NRW');
      entity.vectorPackage.target = DownloadedTilePackageEntity(
        status: PackageStatus.downloading,
      );

      expect(entity.status, RegionStatus.downloading);
    });

    test('vectorPackage downloaded, catalogStatus != ready -> downloaded', () {
      final entity = RegionEntity(
        path: 'de.nrw',
        id: 'rec1',
        name: 'NRW',
        catalogStatus: CatalogStatus.building,
        version: '2026-07-01',
      );
      entity.lastDownloadedVersion = '2026-06-01';
      entity.vectorPackage.target = DownloadedTilePackageEntity(
        status: PackageStatus.downloaded,
      );

      expect(entity.status, RegionStatus.downloaded);
    });

    test(
      'vectorPackage downloaded, version == lastDownloadedVersion -> downloaded',
      () {
        final entity = RegionEntity(
          path: 'de.nrw',
          id: 'rec1',
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
          path: 'de.nrw',
          id: 'rec1',
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

    test('vectorPackage error -> error', () {
      final entity = RegionEntity(path: 'de.nrw', id: 'rec1', name: 'NRW');
      entity.vectorPackage.target = DownloadedTilePackageEntity(
        status: PackageStatus.error,
      );

      expect(entity.status, RegionStatus.error);
    });
  });

  group('RegionEntity.fromCatalogEntry', () {
    test('maps bbox/catalog fields and leaves local-only fields unset', () {
      final entry = RegionCatalogEntry(
        id: 'de-nrw',
        path: 'germany.north_rhine_westphalia',
        name: 'North Rhine-Westphalia',
        bbox: [5.9, 50.3, 9.5, 52.5],
        status: CatalogStatus.ready,
        version: '2026-07-01',
        vectorUrl: '/api/v1/regions/germany.north_rhine_westphalia/download',
        vectorSize: 123,
        demStatus: CatalogStatus.ready,
        demUrl: '/api/v1/regions/germany.north_rhine_westphalia/download-dem',
        demSize: 456,
      );

      final entity = RegionEntity.fromCatalogEntry(entry);

      expect(entity.id, 'de-nrw');
      expect(entity.path, 'germany.north_rhine_westphalia');
      expect(entity.name, 'North Rhine-Westphalia');
      expect(entity.minLon, 5.9);
      expect(entity.minLat, 50.3);
      expect(entity.maxLon, 9.5);
      expect(entity.maxLat, 52.5);
      expect(entity.catalogStatus, CatalogStatus.ready);
      expect(entity.demStatus, CatalogStatus.ready);
      expect(entity.version, '2026-07-01');
      expect(
        entity.vectorUrl,
        '/api/v1/regions/germany.north_rhine_westphalia/download',
      );
      expect(entity.vectorSize, 123);
      expect(
        entity.demUrl,
        '/api/v1/regions/germany.north_rhine_westphalia/download-dem',
      );
      expect(entity.demSize, 456);
      expect(entity.inCatalog, isTrue);
      expect(entity.lastDownloadedVersion, isNull);
      expect(entity.vectorPackage.target, isNull);
      expect(entity.demPackage.target, isNull);
    });

    test('missing demStatus maps to CatalogStatus.absent', () {
      final entry = RegionCatalogEntry(
        id: 'de-bay',
        path: 'germany.bavaria',
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
        path: 'germany.bad',
        name: 'Bad',
        bbox: [1.0, 2.0, 3.0],
        status: CatalogStatus.building,
      );

      expect(() => RegionEntity.fromCatalogEntry(entry), throwsFormatException);
    });
  });

  group('RegionEntity.applyCatalogEntry', () {
    test(
      'overwrites only catalog-owned fields, preserves obxId/path/links/lastDownloadedVersion',
      () {
        final entity = RegionEntity(
          path: 'germany.north_rhine_westphalia',
          id: 'oldrecordid1234',
          name: 'Old Name',
        );
        entity.obxId = 7;
        final vectorPkg = DownloadedTilePackageEntity(
          status: PackageStatus.downloaded,
        );
        entity.vectorPackage.target = vectorPkg;
        entity.lastDownloadedVersion = '2026-01-01';

        final entry = RegionCatalogEntry(
          id: 'newrecordid5678',
          path: 'germany.north_rhine_westphalia',
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

        // `path` is the key this row was matched by and must never be
        // reassigned; `id` MUST be adopted, because a differing id means the
        // backend re-minted the record and this row has to follow it while
        // keeping its download state.
        expect(entity.path, 'germany.north_rhine_westphalia');
        expect(entity.id, 'newrecordid5678');
      },
    );

    test('adopts a re-minted record id without dropping download state', () {
      // The exact regression: `ReconcileArchives` deletes a region_archives
      // row whose files are gone and the next build recreates it with a fresh
      // PocketBase id. Matched by path, that must be an in-place update.
      final entity = RegionEntity(
        path: 'canada.canada_alberta.canada_alberta_south',
        id: '1ani4n8myc8rh2m',
        name: 'South',
      );
      final vectorPkg = DownloadedTilePackageEntity(
        status: PackageStatus.downloaded,
        localFilePath:
            '/docs/regions/canada.canada_alberta'
            '.canada_alberta_south/vector.pmtiles',
      );
      entity.vectorPackage.target = vectorPkg;
      entity.lastDownloadedVersion = '20260727';

      entity.applyCatalogEntry(
        RegionCatalogEntry(
          id: '3v3rsaxejr4yihp',
          path: 'canada.canada_alberta.canada_alberta_south',
          name: 'South',
          bbox: [-114.0, 49.0, -110.0, 51.0],
          status: CatalogStatus.ready,
          version: '20260729',
        ),
      );

      expect(entity.id, '3v3rsaxejr4yihp');
      expect(entity.vectorPackage.target, same(vectorPkg));
      expect(entity.lastDownloadedVersion, '20260727');
      // A genuinely newer server version still reads as an available update
      // rather than silently masquerading as up to date.
      expect(entity.status, RegionStatus.updateAvailable);
    });

    test('throws FormatException on a malformed bbox', () {
      final entity = RegionEntity(path: 'de.nrw', id: 'rec1', name: 'NRW');
      final entry = RegionCatalogEntry(
        id: 'de-nrw',
        path: 'germany.north_rhine_westphalia',
        name: 'NRW',
        bbox: [1.0, 2.0],
        status: CatalogStatus.building,
      );

      expect(() => entity.applyCatalogEntry(entry), throwsFormatException);
    });
  });
}
