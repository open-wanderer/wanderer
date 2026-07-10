import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/navigate_response.dart';

void main() {
  group('NavigateResponse.fromJson', () {
    final sampleJson = {
      'maneuvers': [
        {
          'instruction': 'Turn left',
          'length': 120.0,
          'begin_shape_index': 3,
          'bearing': 90.0,
        },
      ],
      'shape': [
        [47.1, 9.2],
        [47.2, 9.3],
      ],
    };

    test(
      'produces a NavigateResponse with one maneuver and two shape entries',
      () {
        final response = NavigateResponse.fromJson(sampleJson);

        expect(response.maneuvers.length, 1);
        expect(response.shape.length, 2);
      },
    );

    test('maneuver fields deserialize correctly', () {
      final response = NavigateResponse.fromJson(sampleJson);
      final maneuver = response.maneuvers[0];

      expect(maneuver.instruction, 'Turn left');
      expect(maneuver.length, 120.0);
      expect(maneuver.bearing, 90.0);
    });

    test(
      'maneuver.beginShapeIndex deserializes from snake_case begin_shape_index',
      () {
        final response = NavigateResponse.fromJson(sampleJson);
        expect(response.maneuvers[0].beginShapeIndex, 3);
      },
    );

    test(
      'shapeAsGeographic returns List<Geographic> where element 0 == Geographic(47.1, 9.2)',
      () {
        final response = NavigateResponse.fromJson(sampleJson);
        final latLngs = response.shapeAsGeographic;

        expect(latLngs.length, 2);
        expect(latLngs[0].lat, closeTo(47.1, 0.0001));
        expect(latLngs[0].lon, closeTo(9.2, 0.0001));
      },
    );

    test('shapeAsGeographic second element == Geographic(47.2, 9.3)', () {
      final response = NavigateResponse.fromJson(sampleJson);
      final latLngs = response.shapeAsGeographic;

      expect(latLngs[1].lat, closeTo(47.2, 0.0001));
      expect(latLngs[1].lon, closeTo(9.3, 0.0001));
    });
  });

  group('NavigateResponse roundtrip', () {
    final sampleJson = {
      'maneuvers': [
        {
          'instruction': 'Turn left',
          'length': 120.0,
          'begin_shape_index': 3,
          'bearing': 90.0,
          'type': 10,
        },
      ],
      'shape': [
        [47.1, 9.2],
        [47.2, 9.3],
      ],
    };

    /// Encode through real JSON (jsonEncode/jsonDecode) so serialization is
    /// exercised end to end, not just a map comparison.
    NavigateResponse roundtrip(NavigateResponse response) {
      final encoded = jsonEncode(response.toJson());
      final decoded = jsonDecode(encoded) as Map<String, dynamic>;
      return NavigateResponse.fromJson(decoded);
    }

    test('full field roundtrip preserves all maneuver and shape fields', () {
      final original = NavigateResponse.fromJson(sampleJson);
      final result = roundtrip(original);

      expect(result.maneuvers.length, original.maneuvers.length);
      final om = original.maneuvers[0];
      final rm = result.maneuvers[0];
      expect(rm.instruction, om.instruction);
      expect(rm.length, om.length);
      expect(rm.beginShapeIndex, om.beginShapeIndex);
      expect(rm.bearing, closeTo(om.bearing, 0.0001));
      expect(rm.type, om.type);

      expect(result.shape.length, original.shape.length);
      expect(result.shapeAsGeographic[0].lat, closeTo(47.1, 0.0001));
      expect(result.shapeAsGeographic[0].lon, closeTo(9.2, 0.0001));
    });

    test('empty maneuvers list roundtrips', () {
      const original = NavigateResponse(
        maneuvers: [],
        shape: [
          [47.1, 9.2],
        ],
      );
      final result = roundtrip(original);

      expect(result.maneuvers.isEmpty, isTrue);
      expect(result.shape.length, 1);
    });

    test('empty shape roundtrips', () {
      const original = NavigateResponse(
        maneuvers: [
          NavigateManeuver(instruction: 'Go', length: 10.0, beginShapeIndex: 0),
        ],
        shape: [],
      );
      final result = roundtrip(original);

      expect(result.shape.isEmpty, isTrue);
      expect(result.maneuvers.length, 1);
    });

    test('maneuver with only required fields uses @Default bearing/type '
        'after roundtrip', () {
      const original = NavigateResponse(
        maneuvers: [
          NavigateManeuver(instruction: 'Go', length: 10.0, beginShapeIndex: 0),
        ],
        shape: [
          [47.1, 9.2],
        ],
      );
      final result = roundtrip(original);

      expect(result.maneuvers[0].bearing, 0.0);
      expect(result.maneuvers[0].type, 0);
    });
  });
}
