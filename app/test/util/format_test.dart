import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/util/format.dart';

// ---------------------------------------------------------------------------
// Tests for formatSpeed and formatElapsed (Phase 3 stats formatters).
//
// Mirrors the import/group/test structure of
// test/provider/navigation_provider_test.dart.
// ---------------------------------------------------------------------------

void main() {
  group('formatDistance', () {
    test('metric (default) formats meters >= 1000 as km', () {
      expect(formatDistance(1000), '1.00 km');
    });

    test('imperial converts meters to miles (×0.000621371)', () {
      // 1000 * 0.000621371 = 0.621371 → "0.62 mi"
      expect(formatDistance(1000, unit: 'imperial'), '0.62 mi');
    });
  });

  group('formatElevation', () {
    test('metric (default) formats meters with m suffix', () {
      expect(formatElevation(100), '100 m');
    });

    test('imperial converts meters to feet (×3.28084)', () {
      // 100 * 3.28084 = 328.084 → "328 ft"
      expect(formatElevation(100, unit: 'imperial'), '328 ft');
    });
  });

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

  group('trailDisplayDuration', () {
    Trail buildTrail({double duration = 0, double? movingDuration}) {
      return Trail(
        id: 'trail-1',
        name: 'Sample Trail',
        duration: duration,
        movingDuration: movingDuration,
        created: DateTime(2026),
        updated: DateTime(2026),
      );
    }

    test('movingDuration null returns duration', () {
      final trail = buildTrail(duration: 3600, movingDuration: null);
      expect(trailDisplayDuration(trail), 3600);
    });

    test('movingDuration 0 returns duration (a zero moving time is not a value)', () {
      final trail = buildTrail(duration: 3600, movingDuration: 0);
      expect(trailDisplayDuration(trail), 3600);
    });

    test('movingDuration 1800 and duration 3600 returns 1800', () {
      final trail = buildTrail(duration: 3600, movingDuration: 1800);
      expect(trailDisplayDuration(trail), 1800);
    });

    test('movingDuration 1800 and duration 0 returns 1800', () {
      final trail = buildTrail(duration: 0, movingDuration: 1800);
      expect(trailDisplayDuration(trail), 1800);
    });
  });

  // Convention per 24-UI-SPEC.md: one decimal place, unit steps at KB/MB/GB
  // (e.g. "45 MB", "2.4 GB"); bare "N B" below 1 KB.
  group('formatBytes', () {
    test('0 bytes formats as "0 B"', () {
      expect(formatBytes(0), '0 B');
    });

    test('512 bytes formats as "512 B"', () {
      expect(formatBytes(512), '512 B');
    });

    test('1024 bytes formats as "1.0 KB"', () {
      expect(formatBytes(1024), '1.0 KB');
    });

    test('1536 bytes formats as "1.5 KB"', () {
      expect(formatBytes(1536), '1.5 KB');
    });

    test('1 MB formats as "1.0 MB"', () {
      expect(formatBytes(1024 * 1024), '1.0 MB');
    });

    test('2.4 GB formats as "2.4 GB"', () {
      expect(formatBytes((2.4 * 1024 * 1024 * 1024).round()), '2.4 GB');
    });
  });
}
