import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/trail.dart';
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
  });

  group('filterOwnTrailsByQuery', () {
    final trails = [
      Trail.empty().copyWith(
        id: '1',
        name: 'Mountain Ridge',
        location: 'Alps',
      ),
      Trail.empty().copyWith(
        id: '2',
        name: 'Coastal Path',
        location: 'Brittany',
      ),
    ];

    test("returns the input unchanged for ''", () {
      expect(filterOwnTrailsByQuery(trails, ''), trails);
    });

    test("returns the input unchanged for '   '", () {
      expect(filterOwnTrailsByQuery(trails, '   '), trails);
    });

    test('matches case-insensitively on name', () {
      final filtered = filterOwnTrailsByQuery(trails, 'mountain');

      expect(filtered.length, 1);
      expect(filtered.single.name, 'Mountain Ridge');
    });

    test('matches case-insensitively on location, excludes non-matching', () {
      final filtered = filterOwnTrailsByQuery(trails, 'BRITTANY');

      expect(filtered.length, 1);
      expect(filtered.single.name, 'Coastal Path');
    });
  });
}
