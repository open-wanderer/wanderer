import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/util/valhalla_util.dart';

void main() {
  group('categoryForTravelProfile', () {
    test('a bike-named category returns its id for bicycle', () {
      const categories = [
        Category(id: 'hike-id', name: 'Hiking Trails'),
        Category(id: 'bike-id', name: 'Mountain Bike Trails'),
      ];

      expect(
        categoryForTravelProfile('bicycle', categories),
        'bike-id',
      );
    });

    test('a hike-named category returns its id for pedestrian', () {
      const categories = [
        Category(id: 'hike-id', name: 'Hiking Trails'),
        Category(id: 'bike-id', name: 'Mountain Bike Trails'),
      ];

      expect(
        categoryForTravelProfile('pedestrian', categories),
        'hike-id',
      );
    });

    test('a list with no matching category returns null', () {
      const categories = [Category(id: 'other-id', name: 'Wildlife')];

      expect(categoryForTravelProfile('bicycle', categories), isNull);
      expect(categoryForTravelProfile('pedestrian', categories), isNull);
    });

    test('an empty category list returns null', () {
      expect(categoryForTravelProfile('bicycle', const []), isNull);
      expect(categoryForTravelProfile('pedestrian', const []), isNull);
    });

    test(
      'symmetry check: a category named to match costingForCategory\'s '
      '"bike" branch is returned by categoryForTravelProfile for bicycle',
      () {
        const category = Category(id: 'cycle-id', name: 'Cycling Routes');

        // Forward direction (existing heuristic).
        expect(costingForCategory(category.name), 'bicycle');
        // Reverse direction (this task's new heuristic) agrees.
        expect(
          categoryForTravelProfile('bicycle', const [category]),
          'cycle-id',
        );
      },
    );
  });

  group('categoryForBikeBucket', () {
    test(
      'returns the first Category whose name/shortName (case-insensitive) '
      'contains a keyword',
      () {
        const categories = [
          Category(id: 'hike-id', name: 'Hiking Trails'),
          Category(id: 'road-id', name: 'Road Racing'),
        ];

        final match = categoryForBikeBucket([
          'road',
          'race',
          'racing',
        ], categories);

        expect(match?.id, 'road-id');
      },
    );

    test('matches case-insensitively against shortName too', () {
      const categories = [
        Category(id: 'mtb-id', name: 'Off-road', shortName: 'MTB'),
      ];

      final match = categoryForBikeBucket(['mtb', 'mountain'], categories);

      expect(match?.id, 'mtb-id');
    });

    test('returns null when nothing matches', () {
      const categories = [Category(id: 'other-id', name: 'Wildlife')];

      expect(categoryForBikeBucket(['road', 'race'], categories), isNull);
    });

    test('never throws on an empty category list', () {
      expect(
        () => categoryForBikeBucket(['road', 'race'], const []),
        returnsNormally,
      );
      expect(categoryForBikeBucket(['road', 'race'], const []), isNull);
    });
  });
}
