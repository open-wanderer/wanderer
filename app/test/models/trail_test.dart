import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/trail.dart';

Map<String, dynamic> _json({Object? difficulty}) => {
  'id': 'trail-1',
  'name': 'Sample Trail',
  'created': '2026-01-01T00:00:00Z',
  'updated': '2026-01-01T00:00:00Z',
  'difficulty': difficulty,
};

void main() {
  group('Trail.difficulty from JSON', () {
    test('an unset select ("") is an unknown difficulty', () {
      expect(Trail.fromJson(_json(difficulty: '')).difficulty, isNull);
    });

    test('a missing difficulty is unknown', () {
      expect(Trail.fromJson(_json()..remove('difficulty')).difficulty, isNull);
    });

    test('an unsupported value is unknown', () {
      expect(Trail.fromJson(_json(difficulty: 'extreme')).difficulty, isNull);
    });

    test('a known grade decodes', () {
      expect(
        Trail.fromJson(_json(difficulty: 'moderate')).difficulty,
        TrailDifficulty.moderate,
      );
    });
  });

  group('TrailEntity difficulty round-trip', () {
    test('an unknown difficulty stays unknown', () {
      final trail = Trail.fromJson(_json(difficulty: ''));

      final entity = TrailEntity.fromModel(trail);

      expect(entity.dbDifficulty, -1);
      expect(entity.toModel().difficulty, isNull);
    });

    test('a known grade survives', () {
      final trail = Trail.fromJson(_json(difficulty: 'difficult'));

      expect(
        TrailEntity.fromModel(trail).toModel().difficulty,
        TrailDifficulty.difficult,
      );
    });
  });

  group('Trail original track', () {
    test('never reaches the JSON the upload is built from', () {
      final trail = Trail(
        id: 'trail-1',
        name: 'Sample Trail',
        created: DateTime(2026),
        updated: DateTime(2026),
        originalGpxData: '<gpx/>',
        trackFollowsRoads: true,
        trackHeightsRecalculated: true,
      );

      final json = trail.toJson();

      expect(json.keys, isNot(contains('originalGpxData')));
      expect(json.keys, isNot(contains('trackFollowsRoads')));
      expect(json.keys, isNot(contains('trackHeightsRecalculated')));
      expect(json.values, isNot(contains('<gpx/>')));
    });
  });
}
