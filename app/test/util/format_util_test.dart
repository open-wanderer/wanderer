import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/format_util.dart';

// ---------------------------------------------------------------------------
// Tests for formatSpeed and formatElapsed (Phase 3 stats formatters).
//
// Mirrors the import/group/test structure of
// test/provider/navigation_provider_test.dart.
// ---------------------------------------------------------------------------

void main() {
  group('formatSpeed', () {
    test('null returns "-"', () {
      expect(formatSpeed(null), '-');
    });

    test('NaN returns "-"', () {
      expect(formatSpeed(double.nan), '-');
    });

    test('negative returns "-"', () {
      expect(formatSpeed(-1.0), '-');
    });

    test('metric (default) formats to one decimal with km/h suffix', () {
      expect(formatSpeed(12.34), '12.3 km/h');
    });

    test('zero metric is "0.0 km/h"', () {
      expect(formatSpeed(0.0), '0.0 km/h');
    });

    test('imperial converts km/h to mph (×0.621371) with one decimal', () {
      // 10.0 km/h * 0.621371 = 6.21371 → "6.2 mph"
      expect(formatSpeed(10.0, unit: 'imperial'), '6.2 mph');
    });
  });

  group('formatElapsed', () {
    test('seconds only → MM:SS with zero padding', () {
      expect(formatElapsed(const Duration(seconds: 5)), '00:05');
    });

    test('minutes and seconds → MM:SS zero padded', () {
      expect(
        formatElapsed(const Duration(minutes: 3, seconds: 7)),
        '03:07',
      );
    });

    test('hours present → H:MM:SS with minutes/seconds zero padded', () {
      expect(
        formatElapsed(const Duration(hours: 1, minutes: 2, seconds: 3)),
        '1:02:03',
      );
    });

    test('zero duration → 00:00', () {
      expect(formatElapsed(Duration.zero), '00:00');
    });
  });
}
