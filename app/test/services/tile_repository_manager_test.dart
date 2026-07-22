import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/services/tile_repository_manager.dart';

// ---------------------------------------------------------------------------
// Tests for tile_repository_manager's pure resumePlanFor helper only.
//
// startVectorDownload/startDemDownload/pauseRegion/resumeRegion require a
// live Store + network and are exercised by Plan 06's on-device checkpoint
// instead, matching this codebase's precedent that upsertCatalog has no
// unit test.
// ---------------------------------------------------------------------------

void main() {
  group('resumePlanFor', () {
    test('0 existing bytes -> fresh write, no range, deleteOnError true', () {
      final plan = resumePlanFor(0);

      expect(plan.offset, 0);
      expect(plan.mode, FileAccessMode.write);
      expect(plan.sendRange, isFalse);
      expect(plan.deleteOnError, isTrue);
    });

    test(
      '>0 existing bytes -> resumed append, range, deleteOnError false',
      () {
        const existingBytes = 1048576;

        final plan = resumePlanFor(existingBytes);

        expect(plan.offset, existingBytes);
        expect(plan.mode, FileAccessMode.append);
        expect(plan.sendRange, isTrue);
        expect(plan.deleteOnError, isFalse);
      },
    );
  });
}
