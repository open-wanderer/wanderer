import 'package:flutter_test/flutter_test.dart';
import 'package:geolocator/geolocator.dart' as geo;
import 'package:wanderer/provider/foreground_position_stream_provider.dart'
    show LocationMarkerPosition;
import 'package:wanderer/services/tracelet_position_source.dart';

geo.Position _pos({
  required double altitude,
  required double altitudeAccuracy,
}) {
  return geo.Position(
    latitude: 48.137,
    longitude: 11.575,
    timestamp: DateTime(2026, 1, 1),
    accuracy: 5.0,
    altitude: altitude,
    altitudeAccuracy: altitudeAccuracy,
    heading: 0.0,
    headingAccuracy: 5.0,
    speed: 0.0,
    speedAccuracy: 5.0,
  );
}

void main() {
  group('hasUsableAltitude', () {
    test('rejects the both-zero "no reading" sentinel', () {
      expect(
        hasUsableAltitude(_pos(altitude: 0, altitudeAccuracy: 0)),
        isFalse,
      );
    });

    test('accepts a real altitude reported with vertical accuracy', () {
      expect(
        hasUsableAltitude(_pos(altitude: 519.0, altitudeAccuracy: 3.0)),
        isTrue,
      );
    });

    test(
      'accepts a real altitude with NO vertical accuracy (Android < API 26)',
      () {
        // The whole reason this is not a plain `altitudeAccuracy > 0` check —
        // requiring accuracy would disable elevation entirely on API 21-25.
        expect(
          hasUsableAltitude(_pos(altitude: 519.0, altitudeAccuracy: 0)),
          isTrue,
        );
      },
    );

    test('accepts a below-sea-level reading', () {
      expect(
        hasUsableAltitude(_pos(altitude: -12.0, altitudeAccuracy: 0)),
        isTrue,
      );
    });

    test('rejects non-finite altitudes', () {
      expect(
        hasUsableAltitude(_pos(altitude: double.nan, altitudeAccuracy: 3.0)),
        isFalse,
      );
      expect(
        hasUsableAltitude(
          _pos(altitude: double.infinity, altitudeAccuracy: 3.0),
        ),
        isFalse,
      );
    });
  });

  group('seedPositionFrom', () {
    test('produces a position that hasUsableAltitude rejects', () {
      // Locks the two together: the seed exists to give the map marker an
      // immediate position, and must never be mistaken for an elevation
      // reading. If seedPositionFrom ever starts carrying a real altitude,
      // this test is what forces that decision to be made deliberately.
      final seed = seedPositionFrom(
        const LocationMarkerPosition(
          latitude: 48.137,
          longitude: 11.575,
          accuracy: 8.0,
        ),
      );

      expect(hasUsableAltitude(seed), isFalse);
      // Position itself is real and must survive — only elevation is absent.
      expect(seed.latitude, 48.137);
      expect(seed.longitude, 11.575);
      expect(seed.accuracy, 8.0);
    });
  });
}
