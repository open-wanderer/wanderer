import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/util/region_disk_reconcile_util.dart';

// ---------------------------------------------------------------------------
// Tests for region_disk_reconcile_util's pure helpers: resolvePackageRepair
// (D-08's package-vs-disk truth table) and orphanedRegionDirNames (the
// empty-catalog / invalid-id sweep guards). reconcileRegionPackagesWithDisk
// itself needs a real ObjectBox Store and is covered by flutter analyze plus
// on-device verification (see PLAN.md's Task 3 human-check).
// ---------------------------------------------------------------------------

void main() {
  group('resolvePackageRepair', () {
    test('archive exists, no package row at all -> markDownloaded', () {
      expect(
        resolvePackageRepair(archiveExists: true, persisted: null),
        PackageRepair.markDownloaded,
      );
    });

    test('archive exists, persisted notDownloaded -> markDownloaded', () {
      expect(
        resolvePackageRepair(
          archiveExists: true,
          persisted: PackageStatus.notDownloaded,
        ),
        PackageRepair.markDownloaded,
      );
    });

    test('archive exists, persisted error -> markDownloaded', () {
      expect(
        resolvePackageRepair(
          archiveExists: true,
          persisted: PackageStatus.error,
        ),
        PackageRepair.markDownloaded,
      );
    });

    test('archive exists, persisted downloaded -> none', () {
      expect(
        resolvePackageRepair(
          archiveExists: true,
          persisted: PackageStatus.downloaded,
        ),
        PackageRepair.none,
      );
    });

    test('archive absent, persisted downloaded -> markNotDownloaded', () {
      expect(
        resolvePackageRepair(
          archiveExists: false,
          persisted: PackageStatus.downloaded,
        ),
        PackageRepair.markNotDownloaded,
      );
    });

    test('archive absent, no package row at all -> none', () {
      expect(
        resolvePackageRepair(archiveExists: false, persisted: null),
        PackageRepair.none,
      );
    });
  });

  group('orphanedRegionDirNames', () {
    test('a dir matching no known region id is orphaned', () {
      expect(orphanedRegionDirNames(['de-nrw', 'fr-idf'], ['de-nrw']), {
        'fr-idf',
      });
    });

    test('an empty local catalog can never justify deleting archives', () {
      expect(orphanedRegionDirNames(['de-nrw'], []), isEmpty);
    });

    test('only names that are valid region ids are ever sweep candidates', () {
      expect(orphanedRegionDirNames(['..', 'Not-An-Id'], ['de-nrw']), isEmpty);
    });
  });
}
