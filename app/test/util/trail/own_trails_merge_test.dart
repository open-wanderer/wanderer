import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/util/trail/own_trails_merge.dart';

// ---------------------------------------------------------------------------
// Pure-function tests for own_trails_merge.dart. No Store, no provider
// container -- plain Trail(...)/TrailSearchResult(...) fixtures only.
// ---------------------------------------------------------------------------

/// `Trail.empty()` defaults to `TrailSyncState.synced`, so every fixture that
/// is meant to stand for a not-yet-uploaded row says so explicitly -- the
/// online half of the merge now turns on exactly that field.
Trail _unsynced({String id = '', required String name}) =>
    Trail.empty().copyWith(
      id: id,
      name: name,
      syncState: TrailSyncState.pending,
    );

void main() {
  group('mergeOwnTrails online', () {
    test('unsynced local rows come first, in the order supplied', () {
      final local1 = _unsynced(name: 'Local First');
      final local2 = _unsynced(name: 'Local Second');
      final network = TrailSearchResult.mock().copyWith(
        id: 'server-1',
        name: 'Network Trail',
      );

      final merged = mergeOwnTrails(
        local: [local1, local2],
        network: [network],
        offline: false,
      );

      expect(merged.map((t) => t.name).toList(), [
        'Local First',
        'Local Second',
        'Network Trail',
      ]);
    });

    test('a synced local row is dropped -- the network hit renders', () {
      final local = Trail.empty().copyWith(
        id: 'shared-id',
        name: 'Downloaded copy',
      );
      final network = TrailSearchResult.mock().copyWith(
        id: 'shared-id',
        name: 'Server copy',
      );

      final merged = mergeOwnTrails(
        local: [local],
        network: [network],
        offline: false,
      );

      expect(merged.length, 1);
      expect(
        merged.single.name,
        'Server copy',
        reason:
            'online, a trail that already reached the server is represented '
            'by the network half only -- the device copy must not pin it '
            'above the online results',
      );
    });

    test(
      'a synced local row with no matching network hit is dropped entirely',
      () {
        final local = Trail.empty().copyWith(id: 'downloaded-1');
        final network = TrailSearchResult.mock().copyWith(id: 'other-id');

        final merged = mergeOwnTrails(
          local: [local],
          network: [network],
          offline: false,
        );

        expect(merged.map((t) => t.id).toList(), ['other-id']);
      },
    );

    test(
      'two unsynced local rows with empty ids do not suppress any network '
      'result',
      () {
        final local1 = _unsynced(name: 'Unsynced 1');
        final local2 = _unsynced(name: 'Unsynced 2');
        final network = TrailSearchResult.mock().copyWith(id: '');

        // A network result would never realistically carry an empty id, but
        // this pins that the dedupe set is built from non-empty local ids
        // only -- two empty-id local rows can never suppress it.
        final merged = mergeOwnTrails(
          local: [local1, local2],
          network: [network],
          offline: false,
        );

        expect(merged.length, 3);
      },
    );

    test('an empty local list returns the network list unchanged', () {
      final network = TrailSearchResult.mock();

      final merged = mergeOwnTrails(
        local: const [],
        network: [network],
        offline: false,
      );

      expect(merged, [network]);
    });

    // A local row can carry a real server id while its syncState is
    // still not `synced` (the `alreadyUploaded` window: the drain's create
    // step stamps a server id well before the row is retired). Such a row
    // survives the online narrowing -- it is still unsynced -- and the
    // network hit for that same id is dropped by the id dedupe. That is
    // correct only because `applyNetworkEditToLocalRow` reconciles
    // this exact row onto the server's accepted result right after a
    // successful network save, before the own-trails list is ever re-read.
    test(
      'an unsynced local row carrying a real server id still suppresses the '
      'matching network hit -- correct only because '
      'applyNetworkEditToLocalRow keeps that row current',
      () {
        final local = _unsynced(id: 'server-1', name: 'Reconciled Name');
        final network = TrailSearchResult.mock().copyWith(
          id: 'server-1',
          name: 'Stale Server-Side Search Index Name',
        );

        final merged = mergeOwnTrails(
          local: [local],
          network: [network],
          offline: false,
        );

        final matching = merged.where((t) => t.id == 'server-1');
        expect(
          matching.length,
          1,
          reason:
              'exactly one entry for "server-1" -- the network hit must be '
              'deduped against the local row',
        );
        expect(
          matching.single.name,
          'Reconciled Name',
          reason:
              'the local row wins the dedupe. This is only correct because '
              'applyNetworkEditToLocalRow keeps it reconciled to the '
              'server-accepted edit; without that reconciliation this row '
              'would still show its pre-edit name.',
        );
      },
    );
  });

  group('mergeOwnTrails offline', () {
    test('every local row is kept, synced or not', () {
      final downloaded = Trail.empty().copyWith(
        id: 'downloaded-1',
        name: 'Downloaded',
      );
      final unsynced = _unsynced(name: 'Unsynced');

      final merged = mergeOwnTrails(
        local: [unsynced, downloaded],
        network: const [],
        offline: true,
      );

      expect(merged.map((t) => t.name).toList(), ['Unsynced', 'Downloaded']);
    });

    test('an empty network list returns the local list unchanged', () {
      final local = Trail.empty().copyWith(id: 'local-1');

      final merged = mergeOwnTrails(
        local: [local],
        network: const [],
        offline: true,
      );

      expect(merged, [local]);
    });
  });

  group('ownTrailsLocalHalf', () {
    test('online keeps only rows whose syncState is not synced', () {
      final synced = Trail.empty().copyWith(id: 'a', name: 'Synced');
      final pending = _unsynced(name: 'Pending');
      final uploading = Trail.empty().copyWith(
        name: 'Uploading',
        syncState: TrailSyncState.uploading,
      );
      final failed = Trail.empty().copyWith(
        name: 'Failed',
        syncState: TrailSyncState.failed,
      );

      final half = ownTrailsLocalHalf([
        synced,
        pending,
        uploading,
        failed,
      ], offline: false);

      expect(half.map((t) => t.name).toList(), [
        'Pending',
        'Uploading',
        'Failed',
      ]);
    });

    test('offline is a passthrough', () {
      final rows = [
        Trail.empty().copyWith(id: 'a'),
        _unsynced(name: 'b'),
      ];

      expect(ownTrailsLocalHalf(rows, offline: true), rows);
    });
  });
}
