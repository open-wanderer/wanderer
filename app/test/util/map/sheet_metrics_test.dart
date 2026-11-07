import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/map/sheet_metrics.dart';

/// Two size-triple shapes exercised for both functions: `map_screen.dart`'s
/// (bottom-nav-inclusive `sheetMinSize`) and the profile map's (smaller
/// `sheetMinSize`, no bottom nav to account for).
const _mapScreenShape = (min: 0.35, medium: 0.5, max: 1.0);
const _profileScreenShape = (min: 0.18, medium: 0.5, max: 1.0);

void main() {
  group('sheetHeaderOpacity', () {
    for (final shape in [_mapScreenShape, _profileScreenShape]) {
      test('returns 1.0 at and below the medium threshold (${shape.min})', () {
        expect(
          sheetHeaderOpacity(shape.medium, sheetMediumsize: shape.medium),
          1.0,
        );
        expect(
          sheetHeaderOpacity(shape.min, sheetMediumsize: shape.medium),
          1.0,
        );
      });

      test('returns 0.0 at max size (${shape.min})', () {
        expect(
          sheetHeaderOpacity(shape.max, sheetMediumsize: shape.medium),
          0.0,
        );
      });

      test(
        'clamps a monotonic interpolation between medium and max (${shape.min})',
        () {
          final midway = (shape.medium + shape.max) / 2;
          final opacity = sheetHeaderOpacity(
            midway,
            sheetMediumsize: shape.medium,
          );
          expect(opacity, greaterThan(0.0));
          expect(opacity, lessThan(1.0));

          final closerToMax = (midway + shape.max) / 2;
          final opacityCloserToMax = sheetHeaderOpacity(
            closerToMax,
            sheetMediumsize: shape.medium,
          );
          expect(opacityCloserToMax, lessThan(opacity));
        },
      );
    }
  });

  group('dynamicSheetPadding', () {
    for (final shape in [_mapScreenShape, _profileScreenShape]) {
      test('returns the max bottom padding at min size (${shape.min})', () {
        final atMin = dynamicSheetPadding(
          shape.min,
          sheetMinSize: shape.min,
          sheetMediumsize: shape.medium,
          sheetMaxSize: shape.max,
        );
        final belowMin = dynamicSheetPadding(
          shape.min - 0.1,
          sheetMinSize: shape.min,
          sheetMediumsize: shape.medium,
          sheetMaxSize: shape.max,
        );
        expect(atMin, belowMin);
        expect(atMin, greaterThan(0.0));
      });

      test('returns the max top padding at max size (${shape.min})', () {
        final atMax = dynamicSheetPadding(
          shape.max,
          sheetMinSize: shape.min,
          sheetMediumsize: shape.medium,
          sheetMaxSize: shape.max,
        );
        final aboveMax = dynamicSheetPadding(
          shape.max + 0.1,
          sheetMinSize: shape.min,
          sheetMediumsize: shape.medium,
          sheetMaxSize: shape.max,
        );
        expect(atMax, aboveMax);
        expect(atMax, greaterThan(0.0));
      });

      test(
        'interpolates monotonically on both sides of medium (${shape.min})',
        () {
          final belowMediumNearMin = dynamicSheetPadding(
            shape.min + (shape.medium - shape.min) * 0.25,
            sheetMinSize: shape.min,
            sheetMediumsize: shape.medium,
            sheetMaxSize: shape.max,
          );
          final belowMediumNearMedium = dynamicSheetPadding(
            shape.min + (shape.medium - shape.min) * 0.75,
            sheetMinSize: shape.min,
            sheetMediumsize: shape.medium,
            sheetMaxSize: shape.max,
          );
          // Padding shrinks from min toward medium.
          expect(belowMediumNearMedium, lessThan(belowMediumNearMin));

          final atMedium = dynamicSheetPadding(
            shape.medium,
            sheetMinSize: shape.min,
            sheetMediumsize: shape.medium,
            sheetMaxSize: shape.max,
          );
          expect(atMedium, 0.0);

          final aboveMediumNearMedium = dynamicSheetPadding(
            shape.medium + (shape.max - shape.medium) * 0.25,
            sheetMinSize: shape.min,
            sheetMediumsize: shape.medium,
            sheetMaxSize: shape.max,
          );
          final aboveMediumNearMax = dynamicSheetPadding(
            shape.medium + (shape.max - shape.medium) * 0.75,
            sheetMinSize: shape.min,
            sheetMediumsize: shape.medium,
            sheetMaxSize: shape.max,
          );
          // Padding grows from medium toward max.
          expect(aboveMediumNearMax, greaterThan(aboveMediumNearMedium));
        },
      );
    }
  });
}
