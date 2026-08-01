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

    test(
      'a non-null movingDuration survives fromModel -> toModel unchanged',
      () {
        final trail = buildSampleTrail(movingDuration: 3600.0);

        final roundTripped = TrailEntity.fromModel(trail).toModel();

        expect(roundTripped.movingDuration, 3600.0);
      },
    );

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

  group('toModel() and malformed cached GPX', () {
    // Why TrailLibraryNotifier.build() wraps each toModel() in a try/catch
    // rather than mapping in bulk: this throws, and a throw inside build()
    // used to fail the ENTIRE offline library — one unopenable cached trail
    // hid every other downloaded trail, with no way for the user to identify
    // or clear it. The guard's blast radius is now the single bad row.
    //
    // The guard itself is verified by inspection, not by this test: driving
    // TrailLibraryNotifier.build() needs an ObjectBox store, and no trail
    // harness exists (test/services/ has tile/region ones only).
    test('a cached entity with non-GPX gpxData throws out of toModel', () {
      final entity = TrailEntity(
        id: 'bad-1',
        name: 'Corrupt',
        created: DateTime(2026),
        updated: DateTime(2026),
        gpxData: '<?xml version="1.0"?><kml><foo/></kml>',
      );

      expect(() => entity.toModel(), throwsFormatException);
    });

    test('a cached entity with well-formed gpxData still parses', () {
      final entity = TrailEntity(
        id: 'good-1',
        name: 'Fine',
        created: DateTime(2026),
        updated: DateTime(2026),
        gpxData:
            '<?xml version="1.0"?>'
            '<gpx version="1.1" creator="t"><trk><trkseg>'
            '<trkpt lat="47.0" lon="11.0"><ele>1000</ele></trkpt>'
            '</trkseg></trk></gpx>',
      );

      expect(entity.toModel().expand?.gpx, isNotNull);
    });
  });
}
