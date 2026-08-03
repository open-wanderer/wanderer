import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/util/trail/own_trails_merge.dart';

// ---------------------------------------------------------------------------
// Pure-function tests for own_trails_merge.dart. No Store, no provider
// container -- plain Trail(...)/TrailSearchResult(...) fixtures only.
// ---------------------------------------------------------------------------

void main() {
  group('mergeOwnTrails', () {
    test('local rows come first, in the order supplied', () {
      final local1 = Trail.empty().copyWith(id: '', name: 'Local First');
      final local2 = Trail.empty().copyWith(id: '', name: 'Local Second');
      final network = TrailSearchResult.mock().copyWith(
        id: 'server-1',
        name: 'Network Trail',
      );

      final merged = mergeOwnTrails(
        local: [local1, local2],
        network: [network],
      );

      expect(merged.map((t) => t.name).toList(), [
        'Local First',
        'Local Second',
        'Network Trail',
      ]);
    });

    test('a network result whose id matches a local row is dropped', () {
      final local = Trail.empty().copyWith(id: 'shared-id', name: 'Uploaded');
      final network = TrailSearchResult.mock().copyWith(
        id: 'shared-id',
        name: 'Uploaded (server copy)',
      );

      final merged = mergeOwnTrails(local: [local], network: [network]);

      expect(merged.length, 1);
      expect(merged.single.name, 'Uploaded');
    });

    test('a network result with a different id is kept', () {
      final local = Trail.empty().copyWith(id: 'local-server-id');
      final network = TrailSearchResult.mock().copyWith(id: 'other-id');

      final merged = mergeOwnTrails(local: [local], network: [network]);

      expect(merged.length, 2);
    });

    test(
      'two local rows with empty ids do not suppress any network result',
      () {
        final local1 = Trail.empty().copyWith(id: '', name: 'Unsynced 1');
        final local2 = Trail.empty().copyWith(id: '', name: 'Unsynced 2');
        final network = TrailSearchResult.mock().copyWith(id: '');

        // A network result would never realistically carry an empty id, but
        // this pins that the dedupe set is built from non-empty local ids
        // only -- two empty-id local rows can never suppress it.
        final merged = mergeOwnTrails(
          local: [local1, local2],
          network: [network],
        );

        expect(merged.length, 3);
      },
    );

    test('an empty local list returns the network list unchanged', () {
      final network = TrailSearchResult.mock();

      final merged = mergeOwnTrails(local: const [], network: [network]);

      expect(merged, [network]);
    });

    test('an empty network list returns the local list unchanged', () {
      final local = Trail.empty().copyWith(id: 'local-1');

      final merged = mergeOwnTrails(local: [local], network: const []);

      expect(merged, [local]);
    });

    // CR-03. A local row can carry a real server id while its syncState is
    // still not `synced` (the `alreadyUploaded` window: the drain's create
    // step stamps a server id well before the row is retired). The network
    // hit for that same id is dropped here -- and that is now CORRECT only
    // because `applyNetworkEditToLocalRow` (36-17) reconciles this exact row
    // onto the server's accepted result right after a successful network
    // save, before the own-trails list is ever re-read. Before that
    // reconciliation existed, this same assertion documented the bug: the
    // local row's PRE-EDIT name would win over the server's up-to-date one.
    test(
      'a local row carrying a real server id but a non-synced syncState '
      'still suppresses the matching network hit -- correct only because '
      'applyNetworkEditToLocalRow keeps that row current (CR-03)',
      () {
        final local = Trail.empty().copyWith(
          id: 'server-1',
          syncState: TrailSyncState.pending,
          name: 'Reconciled Name',
        );
        final network = TrailSearchResult.mock().copyWith(
          id: 'server-1',
          name: 'Stale Server-Side Search Index Name',
        );

        final merged = mergeOwnTrails(local: [local], network: [network]);

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
              'applyNetworkEditToLocalRow (36-17) keeps it reconciled to the '
              'server-accepted edit; without that reconciliation this row '
              'would still show its pre-edit name, which is exactly CR-03.',
        );
      },
    );
  });
}
