import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/trail.dart';

void main() {
  group('TrailEntity movingDuration round-trip (D-10)', () {
    Trail buildSampleTrail({double? movingDuration}) {
      return Trail(
        id: 'trail-1',
        name: 'Sample Trail',
        created: DateTime(2026),
        updated: DateTime(2026),
        duration: 1234,
        movingDuration: movingDuration,
      );
    }

    test('a null movingDuration survives fromModel -> toModel unchanged', () {
      final trail = buildSampleTrail(movingDuration: null);

      final roundTripped = TrailEntity.fromModel(trail).toModel();

      expect(roundTripped.movingDuration, isNull);
    });

    test('a non-null movingDuration survives fromModel -> toModel unchanged', () {
      final trail = buildSampleTrail(movingDuration: 3600.0);

      final roundTripped = TrailEntity.fromModel(trail).toModel();

      expect(roundTripped.movingDuration, 3600.0);
    });

    test('duration is unaffected by movingDuration being null or set', () {
      final withoutMoving = TrailEntity.fromModel(
        buildSampleTrail(movingDuration: null),
      ).toModel();
      final withMoving = TrailEntity.fromModel(
        buildSampleTrail(movingDuration: 3600.0),
      ).toModel();

      expect(withoutMoving.duration, 1234);
      expect(withMoving.duration, 1234);
    });
  });
}
