import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/local/sync_backoff.dart';

void main() {
  group('kMaxSyncAttempts', () {
    test('is greater than 1', () {
      expect(kMaxSyncAttempts, greaterThan(1));
    });
  });

  group('syncBackoffDelay', () {
    test('attempt 1 is 30 seconds', () {
      expect(syncBackoffDelay(1), const Duration(seconds: 30));
    });

    test('attempt 2 is 2 minutes', () {
      expect(syncBackoffDelay(2), const Duration(minutes: 2));
    });

    test('attempt 3 is 10 minutes', () {
      expect(syncBackoffDelay(3), const Duration(minutes: 10));
    });

    test('attempt 4 and beyond stay clamped at 10 minutes', () {
      expect(syncBackoffDelay(4), const Duration(minutes: 10));
      expect(syncBackoffDelay(10), const Duration(minutes: 10));
      expect(syncBackoffDelay(1000), const Duration(minutes: 10));
    });

    test('attempt 0 and negative inputs do not throw', () {
      expect(() => syncBackoffDelay(0), returnsNormally);
      expect(() => syncBackoffDelay(-1), returnsNormally);
      expect(() => syncBackoffDelay(-1000), returnsNormally);
    });

    test('attempt 0 and negative inputs behave like attempt 1', () {
      expect(syncBackoffDelay(0), const Duration(seconds: 30));
      expect(syncBackoffDelay(-1), const Duration(seconds: 30));
    });

    test('curve is monotonically non-decreasing', () {
      final samples = [
        for (var attempt = -5; attempt <= 20; attempt++)
          syncBackoffDelay(attempt),
      ];
      for (var i = 1; i < samples.length; i++) {
        expect(
          samples[i] >= samples[i - 1],
          isTrue,
          reason:
              'syncBackoffDelay must never decrease as attempts increase '
              '(index $i: ${samples[i]} < ${samples[i - 1]})',
        );
      }
    });
  });
}
