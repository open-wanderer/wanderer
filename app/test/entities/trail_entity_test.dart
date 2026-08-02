import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/waypoint_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/util/local_id.dart';

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

  group('local-first round trip (D-04/D-06/D-10, 36-01)', () {
    test(
      'a TrailEntity with a minted local id round-trips to id == "" and '
      'preserves localId',
      () {
        final localId = mintLocalId();
        final entity = TrailEntity(
          id: localId,
          name: 'Unsynced',
          created: DateTime(2026),
          updated: DateTime(2026),
          localId: localId,
          syncState: TrailSyncState.pending,
        );

        final model = entity.toModel();

        expect(model.id, '');
        expect(model.localId, localId);
      },
    );

    test('a TrailEntity with a server-shaped id round-trips with that id', () {
      const serverId = 'abc123xyz456789';
      final entity = TrailEntity(
        id: serverId,
        name: 'Downloaded',
        created: DateTime(2026),
        updated: DateTime(2026),
      );

      expect(entity.toModel().id, serverId);
    });

    test(
      'a default-constructed TrailEntity reports TrailSyncState.synced',
      () {
        final entity = TrailEntity(
          id: 'abc123xyz456789',
          name: 'Default',
          created: DateTime(2026),
          updated: DateTime(2026),
        );

        expect(entity.syncState, TrailSyncState.synced);
      },
    );

    test('dbSyncState falls back to synced for an out-of-range value', () {
      final entity = TrailEntity(
        id: 'abc123xyz456789',
        name: 'OutOfRange',
        created: DateTime(2026),
        updated: DateTime(2026),
      );

      entity.dbSyncState = 99;

      expect(entity.syncState, TrailSyncState.synced);
    });

    test('dbSyncState round-trips each of the four enum values', () {
      final entity = TrailEntity(
        id: 'abc123xyz456789',
        name: 'RoundTrip',
        created: DateTime(2026),
        updated: DateTime(2026),
      );

      for (final state in TrailSyncState.values) {
        entity.dbSyncState = state.index;
        expect(entity.syncState, state);
        expect(entity.dbSyncState, state.index);
      }
    });

    test(
      'a Trail carrying localId/syncState: pending survives fromModel and '
      'back',
      () {
        final localId = mintLocalId();
        final trail = Trail(
          id: localId,
          name: 'Pending Trail',
          created: DateTime(2026),
          updated: DateTime(2026),
          localId: localId,
          syncState: TrailSyncState.pending,
        );

        final entity = TrailEntity.fromModel(trail);

        expect(entity.localId, localId);
        expect(entity.syncState, TrailSyncState.pending);

        final roundTripped = entity.toModel();

        expect(roundTripped.localId, localId);
        expect(roundTripped.syncState, TrailSyncState.pending);
      },
    );

    test(
      'a Waypoint with id: "", a localKey and two localPhotos survives '
      'fromModel -> toModel intact (D-04 regression guard)',
      () {
        final waypoint = Waypoint(
          id: '',
          lat: 47.0,
          lon: 11.0,
          author: '000000000000000',
          created: DateTime(2026),
          updated: DateTime(2026),
          localKey: mintLocalId(),
          localPhotos: const ['photo1.jpg', 'photo2.jpg'],
        );

        final entity = WaypointEntity.fromModel(waypoint);
        final roundTripped = entity.toModel();

        expect(roundTripped.id, '');
        expect(roundTripped.localKey, waypoint.localKey);
        expect(roundTripped.localPhotos, ['photo1.jpg', 'photo2.jpg']);
      },
    );

    test(
      'two Waypoints with id: "" and different localKeys produce two '
      'WaypointEntity rows with different ids (unique-collision guard)',
      () {
        final waypointA = Waypoint(
          id: '',
          lat: 47.0,
          lon: 11.0,
          author: '000000000000000',
          created: DateTime(2026),
          updated: DateTime(2026),
          localKey: mintLocalId(),
        );
        final waypointB = Waypoint(
          id: '',
          lat: 47.1,
          lon: 11.1,
          author: '000000000000000',
          created: DateTime(2026),
          updated: DateTime(2026),
          localKey: mintLocalId(),
        );

        final entityA = WaypointEntity.fromModel(waypointA);
        final entityB = WaypointEntity.fromModel(waypointB);

        expect(entityA.id, isNot(equals(entityB.id)));
      },
    );
  });
}
