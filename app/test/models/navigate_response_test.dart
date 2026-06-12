import 'package:flutter_test/flutter_test.dart';
import 'package:latlong2/latlong.dart';
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
        }
      ],
      'shape': [
        [47.1, 9.2],
        [47.2, 9.3],
      ],
    };

    test('produces a NavigateResponse with one maneuver and two shape entries',
        () {
      final response = NavigateResponse.fromJson(sampleJson);

      expect(response.maneuvers.length, 1);
      expect(response.shape.length, 2);
    });

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
    });

    test(
        'shapeAsLatLng returns List<LatLng> where element 0 == LatLng(47.1, 9.2)',
        () {
      final response = NavigateResponse.fromJson(sampleJson);
      final latLngs = response.shapeAsLatLng;

      expect(latLngs.length, 2);
      expect(latLngs[0].latitude, closeTo(47.1, 0.0001));
      expect(latLngs[0].longitude, closeTo(9.2, 0.0001));
    });

    test('shapeAsLatLng second element == LatLng(47.2, 9.3)', () {
      final response = NavigateResponse.fromJson(sampleJson);
      final latLngs = response.shapeAsLatLng;

      expect(latLngs[1].latitude, closeTo(47.2, 0.0001));
      expect(latLngs[1].longitude, closeTo(9.3, 0.0001));
    });
  });
}
