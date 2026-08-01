import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/disk_space_util.dart';

// ---------------------------------------------------------------------------
// Tests for disk_space_util's pure decision function, hasEnoughSpace, and
// for the injectable path -> device-wide fallback orchestrator,
// resolveFreeDiskSpaceBytes.
//
// hasEnoughSpace's pure margin math + fail-closed behavior is unit-tested
// directly. resolveFreeDiskSpaceBytes is the pure, injectable decision logic
// behind freeDiskSpaceBytes (the plugin wrapper) — it takes fake
// pathQuery/deviceQuery closures so the fallback ordering is deterministic
// under `flutter test`, with no platform channel involved. The real
// `disk_space_2` plugin wiring in freeDiskSpaceBytes itself still needs a
// device/platform channel and is covered by the phase's on-device
// human-check instead.
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

  group('resolveFreeDiskSpaceBytes', () {
    test(
      'path query succeeds -> returns its mebibytes as bytes; '
      'device query never called',
      () async {
        var deviceCallCount = 0;
        final result = await resolveFreeDiskSpaceBytes(
          forPath: '/app/regions/munich',
          pathQuery: (path) async => 100.0,
          deviceQuery: () async {
            deviceCallCount++;
            return 999.0;
          },
        );
        expect(result, 100 * mb);
        expect(deviceCallCount, 0);
      },
    );

    test(
      'path query throws -> falls back to device query and returns its '
      'value converted to bytes',
      () async {
        final result = await resolveFreeDiskSpaceBytes(
          forPath: '/app/regions/munich',
          pathQuery: (path) async =>
              throw Exception('Specified path does not exist'),
          deviceQuery: () async => 250.0,
        );
        expect(result, 250 * mb);
      },
    );

    test(
      'path query throws AND device query throws -> returns null '
      '(fail closed, TILE-03 preserved)',
      () async {
        final result = await resolveFreeDiskSpaceBytes(
          forPath: '/app/regions/munich',
          pathQuery: (path) async =>
              throw Exception('Specified path does not exist'),
          deviceQuery: () async => throw Exception('device query failed'),
        );
        expect(result, isNull);
      },
    );

    test(
      'forPath == null -> device query used directly; path query never '
      'called',
      () async {
        var pathCallCount = 0;
        final result = await resolveFreeDiskSpaceBytes(
          forPath: null,
          pathQuery: (path) async {
            pathCallCount++;
            return 999.0;
          },
          deviceQuery: () async => 400.0,
        );
        expect(result, 400 * mb);
        expect(pathCallCount, 0);
      },
    );

    test(
      'forPath == null AND device query throws -> returns null '
      '(no further fallback exists)',
      () async {
        final result = await resolveFreeDiskSpaceBytes(
          forPath: null,
          pathQuery: (path) async => 999.0,
          deviceQuery: () async => throw Exception('device query failed'),
        );
        expect(result, isNull);
      },
    );

    test('query returns null mebibytes -> returns null bytes', () async {
      final result = await resolveFreeDiskSpaceBytes(
        forPath: '/app/regions/munich',
        pathQuery: (path) async => null,
        deviceQuery: () async => 400.0,
      );
      expect(result, isNull);
    });
  });
}
