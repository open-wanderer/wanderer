import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/disk_space_util.dart';

// ---------------------------------------------------------------------------
// Tests for disk_space_util's pure decision function, hasEnoughSpace.
//
// Only the pure margin math + fail-closed behavior is unit-tested here.
// freeDiskSpaceBytes (the plugin wrapper) needs a device/platform channel
// and is covered by Plan 06's on-device human-check instead (per this
// plan's own action text).
// ---------------------------------------------------------------------------

void main() {
  const int mb = 1024 * 1024;

  group('hasEnoughSpace', () {
    test('300 MB free vs 100 MB declared at default 1.75x margin is true', () {
      // 300 MB > 100 MB * 1.75 = 175 MB
      expect(
        hasEnoughSpace(freeBytes: 300 * mb, declaredSizeBytes: 100 * mb),
        isTrue,
      );
    });

    test('150 MB free vs 100 MB declared at default 1.75x margin is false', () {
      // 150 MB < 100 MB * 1.75 = 175 MB
      expect(
        hasEnoughSpace(freeBytes: 150 * mb, declaredSizeBytes: 100 * mb),
        isFalse,
      );
    });

    test('null free space is false (fail-closed) regardless of declared size', () {
      expect(
        hasEnoughSpace(freeBytes: null, declaredSizeBytes: 100 * mb),
        isFalse,
      );
      expect(
        hasEnoughSpace(freeBytes: null, declaredSizeBytes: 0),
        isFalse,
      );
    });

    test('default safetyMultiplier is 1.75', () {
      // Exactly at the 1.75x boundary is NOT enough (strict >).
      final declared = 100 * mb;
      final exactlyAtMargin = (declared * 1.75).round();
      expect(
        hasEnoughSpace(freeBytes: exactlyAtMargin, declaredSizeBytes: declared),
        isFalse,
      );
      expect(
        hasEnoughSpace(
          freeBytes: exactlyAtMargin + 1,
          declaredSizeBytes: declared,
        ),
        isTrue,
      );
    });

    test('a custom safetyMultiplier overrides the default', () {
      // 120 MB free vs 100 MB declared fails the default 1.75x margin...
      expect(
        hasEnoughSpace(freeBytes: 120 * mb, declaredSizeBytes: 100 * mb),
        isFalse,
      );
      // ...but passes a relaxed 1.0x margin.
      expect(
        hasEnoughSpace(
          freeBytes: 120 * mb,
          declaredSizeBytes: 100 * mb,
          safetyMultiplier: 1.0,
        ),
        isTrue,
      );
    });
  });
}
