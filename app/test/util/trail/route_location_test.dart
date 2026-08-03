import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/util/trail/route_location.dart';

// ---------------------------------------------------------------------------
// Pure-function tests for trail_route_location.dart. No ProviderScope, no
// Store -- plain Trail.empty().copyWith(...) fixtures only.
// ---------------------------------------------------------------------------

void main() {
  group('trailDetailLocation / trailMapLocation', () {
    test('synced trail with id "abc" -> /trail/abc; map -> /trail/abc/map', () {
      final trail = Trail.empty().copyWith(
        id: 'abc',
        syncState: TrailSyncState.synced,
      );

      expect(trailDetailLocation(trail), '/trail/abc');
      expect(trailMapLocation(trail), '/trail/abc/map');
    });

    test(
      'pending trail with empty id and localId "local-1-0" -> '
      '/trail/local/local-1-0; map -> /trail/local/local-1-0/map',
      () {
        final trail = Trail.empty().copyWith(
          id: '',
          localId: 'local-1-0',
          syncState: TrailSyncState.pending,
        );

        expect(trailDetailLocation(trail), '/trail/local/local-1-0');
        expect(trailMapLocation(trail), '/trail/local/local-1-0/map');
      },
    );

    test('uploading and failed states behave identically to pending', () {
      for (final state in [
        TrailSyncState.uploading,
        TrailSyncState.failed,
      ]) {
        final trail = Trail.empty().copyWith(
          id: '',
          localId: 'local-1-0',
          syncState: state,
        );

        expect(
          trailDetailLocation(trail),
          '/trail/local/local-1-0',
          reason: 'state=$state',
        );
        expect(
          trailMapLocation(trail),
          '/trail/local/local-1-0/map',
          reason: 'state=$state',
        );
      }
    });

    test('unsynced trail with a null localId -> null (both functions)', () {
      final trail = Trail.empty().copyWith(
        id: '',
        localId: null,
        syncState: TrailSyncState.pending,
      );

      expect(trailDetailLocation(trail), isNull);
      expect(trailMapLocation(trail), isNull);
    });

    test('synced trail with an empty id -> null (both functions)', () {
      final trail = Trail.empty().copyWith(
        id: '',
        syncState: TrailSyncState.synced,
      );

      expect(trailDetailLocation(trail), isNull);
      expect(trailMapLocation(trail), isNull);
    });

    test('no returned location ever contains a double slash', () {
      final fixtures = [
        Trail.empty().copyWith(id: 'abc', syncState: TrailSyncState.synced),
        Trail.empty().copyWith(
          id: '',
          localId: 'local-1-0',
          syncState: TrailSyncState.pending,
        ),
        Trail.empty().copyWith(
          id: '',
          localId: 'local-1-0',
          syncState: TrailSyncState.uploading,
        ),
        Trail.empty().copyWith(
          id: '',
          localId: 'local-1-0',
          syncState: TrailSyncState.failed,
        ),
        Trail.empty().copyWith(
          id: '',
          localId: null,
          syncState: TrailSyncState.pending,
        ),
        Trail.empty().copyWith(id: '', syncState: TrailSyncState.synced),
      ];

      for (final trail in fixtures) {
        final detail = trailDetailLocation(trail);
        final map = trailMapLocation(trail);
        if (detail != null) {
          expect(
            detail.contains('//'),
            isFalse,
            reason: 'trailDetailLocation emitted a double slash: $detail',
          );
        }
        if (map != null) {
          expect(
            map.contains('//'),
            isFalse,
            reason: 'trailMapLocation emitted a double slash: $map',
          );
        }
      }
    });
  });
}
