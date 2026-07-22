import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/util/region_disk_usage_util.dart';
import 'package:wanderer/util/region_file_path.dart';

// ---------------------------------------------------------------------------
// Tests for region_disk_usage_util's regionDiskUsageBytes, which must read
// REAL on-disk byte counts (final files AND partial .part files) rather than
// trust the persisted sizeBytesOnDisk field (D-06, Pitfall 1 -- a
// downloading/paused package never has sizeBytesOnDisk populated).
// ---------------------------------------------------------------------------

void main() {
  late Directory tempDir;

  setUp(() {
    tempDir = Directory.systemTemp.createTempSync('region_disk_usage_test');
  });

  tearDown(() {
    if (tempDir.existsSync()) tempDir.deleteSync(recursive: true);
  });

  RegionEntity buildRegion({String id = 'de-nrw'}) {
    return RegionEntity(id: id, name: 'Test Region');
  }

  group('regionDiskUsageBytes', () {
    test('final vector file present contributes its byte length', () {
      final region = buildRegion();
      final vectorPath = regionVectorPath(tempDir.path, region.id);
      Directory(File(vectorPath).parent.path).createSync(recursive: true);
      File(vectorPath).writeAsBytesSync(List.filled(100, 0));

      expect(regionDiskUsageBytes(tempDir.path, region), 100);
    });

    test(
      'only a .part partial vector file present contributes its byte length (Pitfall 1)',
      () {
        final region = buildRegion();
        final vectorPath = regionVectorPath(tempDir.path, region.id);
        Directory(File(vectorPath).parent.path).createSync(recursive: true);
        File('$vectorPath.part').writeAsBytesSync(List.filled(42, 0));

        expect(regionDiskUsageBytes(tempDir.path, region), 42);
      },
    );

    test('neither vector nor DEM file on disk contributes 0', () {
      final region = buildRegion();
      expect(regionDiskUsageBytes(tempDir.path, region), 0);
    });

    test('vector final (N) + DEM final (K) both present sums to N+K', () {
      final region = buildRegion();
      final vectorPath = regionVectorPath(tempDir.path, region.id);
      final demPath = regionDemPath(tempDir.path, region.id);
      Directory(File(vectorPath).parent.path).createSync(recursive: true);
      File(vectorPath).writeAsBytesSync(List.filled(100, 0));
      File(demPath).writeAsBytesSync(List.filled(30, 0));

      expect(regionDiskUsageBytes(tempDir.path, region), 130);
    });

    test('an invalid region id throws ArgumentError via assertValidRegionId', () {
      final region = buildRegion(id: '../etc');
      expect(
        () => regionDiskUsageBytes(tempDir.path, region),
        throwsArgumentError,
      );
    });
  });
}
