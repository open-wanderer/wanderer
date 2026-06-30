import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/models/trail.dart';

// Builds a baseline TrailFilter equivalent to defaultFilter, with empty
// category/subcategory lists. Individual tests use copyWith to set the
// category/subcategory selections under test.
TrailFilter _baseFilter() => const TrailFilter(
      q: "",
      category: [],
      subcategory: [],
      tags: [],
      difficulty: [],
      near: TrailNear(radius: 2000),
      distanceMin: 0,
      distanceMax: 100000,
      distanceLimit: 100000,
      elevationGainMin: 0,
      elevationGainMax: 10000,
      elevationGainLimit: 10000,
      elevationLossMin: 0,
      elevationLossMax: 10000,
      elevationLossLimit: 10000,
      sort: TrailFilterSort.created,
      sortOrder: SortOrder.desc,
    );

const _category = Category(id: 'c1', name: 'Hiking');
const _subcategory = Subcategory(id: 's1', category: 'c1', name: 'Day Hike');

void main() {
  group('TrailFilter.toFilterText category/subcategory clause', () {
    test('one category, no subcategories emits category_id IN', () {
      final filter = _baseFilter().copyWith(category: [_category]);

      final text = filter.toFilterText();

      expect(text, contains("(category_id IN ['c1'])"));
    });

    test('one subcategory with no parent category selected emits subcategory_id IN', () {
      final filter = _baseFilter().copyWith(subcategory: [_subcategory]);

      final text = filter.toFilterText();

      expect(text, contains("(subcategory_id IN ['s1'])"));
    });

    test(
        'category with a matching subcategory selected narrows to subcategory '
        'only — parent category_id IN is omitted', () {
      final filter = _baseFilter()
          .copyWith(category: [_category], subcategory: [_subcategory]);

      final text = filter.toFilterText();

      expect(text, contains("(subcategory_id IN ['s1'])"));
      expect(text, isNot(contains("category_id IN ['c1']")));
    });

    test(
        'two categories: one with subcategory selected, one without — mixed '
        'clause mirrors web trail_store.ts', () {
      const cat2 = Category(id: 'c2', name: 'Nature');
      final filter = _baseFilter().copyWith(
        category: [_category, cat2],
        subcategory: [_subcategory], // belongs to c1 only
      );

      final text = filter.toFilterText();

      // c1 has a sub → only subcategory_id IN clause; c1 excluded from category_id IN
      expect(text, contains("subcategory_id IN ['s1']"));
      expect(text, contains("category_id IN ['c2']"));
      expect(text, isNot(contains("category_id IN ['c1']")));
    });

    test('neither category nor subcategory emits neither clause', () {
      final filter = _baseFilter();

      final text = filter.toFilterText();

      expect(text, isNot(contains('category_id')));
      expect(text, isNot(contains('subcategory_id')));
    });
  });
}
