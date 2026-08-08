import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/util/trail/filter.dart';

// ---------------------------------------------------------------------------
// Pure-function tests for util/trail/filter.dart. No Store, no provider
// container, no widgets -- plain Trail(...)/TrailFilter(...) fixtures only.
//
// The reference behaviour is TrailFilter.toFilterText(): every assertion here
// asks "would Meilisearch have kept this row?", because the whole point of
// applyTrailFilter is that a locally-held trail and a server-side hit in the
// same list are narrowed by the same rules.
// ---------------------------------------------------------------------------

/// An all-pass filter: every difficulty selected, every range at its limit
/// (so no upper bound is imposed, per buildDefaultTrailFilter's convention),
/// no category, no dates.
TrailFilter _baseFilter({
  List<int> difficulty = const [0, 1, 2],
  List<Category> category = const [],
  List<Subcategory> subcategory = const [],
  double distanceMin = 0,
  double distanceMax = 100000,
  double distanceLimit = 100000,
  double elevationGainMin = 0,
  double elevationGainMax = 10000,
  double elevationGainLimit = 10000,
  double elevationLossMin = 0,
  double elevationLossMax = 10000,
  double elevationLossLimit = 10000,
  DateTime? startDate,
  DateTime? endDate,
  bool offlineOnly = false,
  TrailFilterSort sort = TrailFilterSort.created,
  SortOrder sortOrder = SortOrder.desc,
}) {
  return TrailFilter(
    q: '',
    category: category,
    subcategory: subcategory,
    tags: const [],
    difficulty: difficulty,
    near: const TrailNear(radius: 2000),
    distanceMin: distanceMin,
    distanceMax: distanceMax,
    distanceLimit: distanceLimit,
    elevationGainMin: elevationGainMin,
    elevationGainMax: elevationGainMax,
    elevationGainLimit: elevationGainLimit,
    elevationLossMin: elevationLossMin,
    elevationLossMax: elevationLossMax,
    elevationLossLimit: elevationLossLimit,
    startDate: startDate,
    endDate: endDate,
    offlineOnly: offlineOnly,
    sort: sort,
    sortOrder: sortOrder,
  );
}

Trail _trail({
  String id = '',
  String name = 'Trail',
  TrailDifficulty difficulty = TrailDifficulty.easy,
  double distance = 0,
  double elevationGain = 0,
  double elevationLoss = 0,
  double duration = 0,
  int likeCount = 0,
  String? category,
  String? subcategory,
  DateTime? date,
  DateTime? created,
  bool isLocal = false,
}) {
  return Trail.empty().copyWith(
    id: id,
    name: name,
    difficulty: difficulty,
    distance: distance,
    elevationGain: elevationGain,
    elevationLoss: elevationLoss,
    duration: duration,
    likeCount: likeCount,
    category: category,
    subcategory: subcategory,
    date: date,
    created: created ?? DateTime(2024, 1, 1),
    isLocal: isLocal,
  );
}

const _hiking = Category(id: 'cat-hiking', name: 'Hiking');
const _cycling = Category(id: 'cat-cycling', name: 'Cycling');
const _viaFerrata = Subcategory(
  id: 'sub-via-ferrata',
  category: 'cat-hiking',
  name: 'Via Ferrata',
);

void main() {
  group('difficulty', () {
    test('drops trails whose difficulty index is not selected', () {
      final trails = [
        _trail(name: 'Easy', difficulty: TrailDifficulty.easy),
        _trail(name: 'Moderate', difficulty: TrailDifficulty.moderate),
        _trail(name: 'Difficult', difficulty: TrailDifficulty.difficult),
      ];

      final result = applyTrailFilter(trails, _baseFilter(difficulty: [1]));

      expect(result.map((t) => t.name), ['Moderate']);
    });
  });

  group('ranges', () {
    test('the minimum always applies', () {
      final trails = [
        _trail(name: 'Short', distance: 1000),
        _trail(name: 'Long', distance: 9000),
      ];

      final result = applyTrailFilter(
        trails,
        _baseFilter(distanceMin: 5000),
      );

      expect(result.map((t) => t.name), ['Long']);
    });

    test('the maximum applies only while it is below its limit', () {
      final trails = [
        _trail(name: 'Short', distance: 1000),
        _trail(name: 'Long', distance: 9000),
      ];

      // max == limit: an untouched slider must not drop anything, not even a
      // trail longer than the device-derived offline bound.
      expect(
        applyTrailFilter(
          trails,
          _baseFilter(distanceMax: 5000, distanceLimit: 5000),
        ).length,
        2,
      );

      // max < limit: the upper bound is real.
      expect(
        applyTrailFilter(
          trails,
          _baseFilter(distanceMax: 5000, distanceLimit: 100000),
        ).map((t) => t.name),
        ['Short'],
      );
    });

    test('elevation gain and loss are bounded independently', () {
      final trails = [
        _trail(name: 'Flat', elevationGain: 50, elevationLoss: 50),
        _trail(name: 'Climb', elevationGain: 1200, elevationLoss: 50),
        _trail(name: 'Descent', elevationGain: 50, elevationLoss: 1200),
      ];

      expect(
        applyTrailFilter(
          trails,
          _baseFilter(elevationGainMin: 1000),
        ).map((t) => t.name),
        ['Climb'],
      );
      expect(
        applyTrailFilter(
          trails,
          _baseFilter(elevationLossMax: 100, elevationLossLimit: 10000),
        ).map((t) => t.name).toList()..sort(),
        ['Climb', 'Flat'],
      );
    });
  });

  group('category and subcategory', () {
    test('no category and no subcategory selected imposes no constraint', () {
      final trails = [
        _trail(name: 'Uncategorised'),
        _trail(name: 'Hike', category: 'cat-hiking'),
      ];

      expect(applyTrailFilter(trails, _baseFilter()).length, 2);
    });

    test('a selected category matches on the trail category id', () {
      final trails = [
        _trail(name: 'Hike', category: 'cat-hiking'),
        _trail(name: 'Ride', category: 'cat-cycling'),
        _trail(name: 'Uncategorised'),
      ];

      final result = applyTrailFilter(
        trails,
        _baseFilter(category: const [_hiking]),
      );

      expect(result.map((t) => t.name), ['Hike']);
    });

    test('a selected subcategory narrows its own parent category', () {
      final trails = [
        _trail(name: 'Plain hike', category: 'cat-hiking'),
        _trail(
          name: 'Ferrata',
          category: 'cat-hiking',
          subcategory: 'sub-via-ferrata',
        ),
      ];

      // Both the parent category and one of its subcategories are selected --
      // toFilterText() represents that as the subcategory alone, so the plain
      // hike must NOT be widened back in by its parent.
      final result = applyTrailFilter(
        trails,
        _baseFilter(
          category: const [_hiking],
          subcategory: const [_viaFerrata],
        ),
      );

      expect(result.map((t) => t.name), ['Ferrata']);
    });

    test('category and subcategory clauses are OR-ed across parents', () {
      final trails = [
        _trail(name: 'Plain hike', category: 'cat-hiking'),
        _trail(
          name: 'Ferrata',
          category: 'cat-hiking',
          subcategory: 'sub-via-ferrata',
        ),
        _trail(name: 'Ride', category: 'cat-cycling'),
      ];

      // Cycling has no selected subcategory, so it matches by category; the
      // ferrata matches by subcategory. The plain hike matches neither.
      final result = applyTrailFilter(
        trails,
        _baseFilter(
          category: const [_hiking, _cycling],
          subcategory: const [_viaFerrata],
        ),
      );

      expect(result.map((t) => t.name).toList()..sort(), ['Ferrata', 'Ride']);
    });
  });

  group('date range', () {
    test('keeps trails inside the range and drops those outside', () {
      final trails = [
        _trail(name: 'Before', date: DateTime(2024, 1, 10)),
        _trail(name: 'Inside', date: DateTime(2024, 6, 10)),
        _trail(name: 'After', date: DateTime(2024, 12, 10)),
      ];

      final result = applyTrailFilter(
        trails,
        _baseFilter(
          startDate: DateTime(2024, 3, 1),
          endDate: DateTime(2024, 9, 1),
        ),
      );

      expect(result.map((t) => t.name), ['Inside']);
    });

    test('a trail with no date is never dropped by the date range', () {
      final trails = [_trail(name: 'Undated')];

      final result = applyTrailFilter(
        trails,
        _baseFilter(
          startDate: DateTime(2024, 3, 1),
          endDate: DateTime(2024, 9, 1),
        ),
      );

      expect(result.map((t) => t.name), ['Undated']);
    });
  });

  group('offlineOnly', () {
    test('flag off changes nothing, even for a non-downloaded, non-local trail', () {
      final trails = [_trail(name: 'A', id: 'a', isLocal: false)];

      final result = applyTrailFilter(
        trails,
        _baseFilter(offlineOnly: false),
        downloadedIds: const {},
      );

      expect(result.map((t) => t.name), ['A']);
    });

    test('drops a non-downloaded, non-local trail when the flag is on', () {
      final trails = [_trail(name: 'A', id: 'a', isLocal: false)];

      final result = applyTrailFilter(
        trails,
        _baseFilter(offlineOnly: true),
        downloadedIds: const {},
      );

      expect(result, isEmpty);
    });

    test('keeps a trail whose id is in downloadedIds', () {
      final trails = [_trail(name: 'A', id: 'a', isLocal: false)];

      final result = applyTrailFilter(
        trails,
        _baseFilter(offlineOnly: true),
        downloadedIds: {'a'},
      );

      expect(result.map((t) => t.name), ['A']);
    });

    test('keeps a local trail even when downloadedIds is empty', () {
      final trails = [_trail(name: 'A', id: 'a', isLocal: true)];

      final result = applyTrailFilter(
        trails,
        _baseFilter(offlineOnly: true),
        downloadedIds: const {},
      );

      expect(result.map((t) => t.name), ['A']);
    });

    test('composes with an existing clause: a downloaded trail excluded by '
        'difficulty is still dropped', () {
      final trails = [
        _trail(
          name: 'A',
          id: 'a',
          isLocal: false,
          difficulty: TrailDifficulty.difficult,
        ),
      ];

      final result = applyTrailFilter(
        trails,
        _baseFilter(offlineOnly: true, difficulty: const [0, 1]),
        downloadedIds: {'a'},
      );

      expect(result, isEmpty);
    });
  });

  group('sort', () {
    final trails = [
      _trail(
        name: 'B',
        distance: 2000,
        elevationGain: 300,
        elevationLoss: 100,
        duration: 7200,
        likeCount: 1,
        difficulty: TrailDifficulty.moderate,
        date: DateTime(2024, 5, 1),
        created: DateTime(2024, 2, 1),
      ),
      _trail(
        name: 'A',
        distance: 3000,
        elevationGain: 100,
        elevationLoss: 300,
        duration: 3600,
        likeCount: 9,
        difficulty: TrailDifficulty.difficult,
        date: DateTime(2024, 4, 1),
        created: DateTime(2024, 3, 1),
      ),
    ];

    void expectOrder(TrailFilterSort sort, List<String> ascending) {
      expect(
        applyTrailFilter(
          trails,
          _baseFilter(sort: sort, sortOrder: SortOrder.asc),
        ).map((t) => t.name),
        ascending,
        reason: '$sort ascending',
      );
      expect(
        applyTrailFilter(
          trails,
          _baseFilter(sort: sort, sortOrder: SortOrder.desc),
        ).map((t) => t.name),
        ascending.reversed,
        reason: '$sort descending',
      );
    }

    test('every sort key orders both ways', () {
      expectOrder(TrailFilterSort.name, ['A', 'B']);
      expectOrder(TrailFilterSort.date, ['A', 'B']);
      expectOrder(TrailFilterSort.difficulty, ['B', 'A']);
      expectOrder(TrailFilterSort.distance, ['B', 'A']);
      expectOrder(TrailFilterSort.duration, ['A', 'B']);
      expectOrder(TrailFilterSort.elevation_gain, ['A', 'B']);
      expectOrder(TrailFilterSort.elevation_loss, ['B', 'A']);
      expectOrder(TrailFilterSort.created, ['B', 'A']);
      expectOrder(TrailFilterSort.like_count, ['B', 'A']);
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
