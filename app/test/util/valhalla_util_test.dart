import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/models/subcategory_preference.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/valhalla_profile.dart';
import 'package:wanderer/util/route_travel_bucket.dart';
import 'package:wanderer/util/valhalla_util.dart';

Category _category(String id, String name, {String? profile}) => Category(
  id: id,
  name: name,
  settings: profile == null ? null : {'valhalla_profile': profile},
);

Trail _trail({Category? category, String? subcategoryId}) => Trail(
  id: 't1',
  name: 'Test trail',
  created: DateTime(2026),
  updated: DateTime(2026),
  subcategory: subcategoryId,
  expand: TrailExpand(category: category),
);

Subcategory _subcategory(
  String id,
  String categoryId,
  String name, {
  String? profile,
}) => Subcategory(
  id: id,
  category: categoryId,
  name: name,
  settings: profile == null ? null : {'valhalla_profile': profile},
);

void main() {
  group('ValhallaProfile.parse', () {
    test('decomposes the 4 bicycle_<type> forms', () {
      const wanted = {
        'bicycle_road': 'Road',
        'bicycle_hybrid': 'Hybrid',
        'bicycle_cross': 'Cross',
        'bicycle_mountain': 'Mountain',
      };

      wanted.forEach((raw, bicycleType) {
        final profile = ValhallaProfile.parse(raw);
        expect(profile?.costing, 'bicycle', reason: raw);
        expect(profile?.bicycleType, bicycleType, reason: raw);
      });
    });

    test('pedestrian parses to a plain costing with no bicycle_type', () {
      final profile = ValhallaProfile.parse('pedestrian');

      expect(profile?.costing, 'pedestrian');
      expect(profile?.bicycleType, isNull);
    });

    test('an open-vocabulary costing (auto) passes through verbatim', () {
      final profile = ValhallaProfile.parse('auto');

      expect(profile?.costing, 'auto');
      expect(profile?.bicycleType, isNull);
    });

    test('motor_scooter survives as ONE costing name, never split', () {
      final profile = ValhallaProfile.parse('motor_scooter');

      expect(profile?.costing, 'motor_scooter');
      expect(profile?.bicycleType, isNull);
    });

    test('only the 4 known bicycle_types are decomposed', () {
      final profile = ValhallaProfile.parse('bicycle_banana');

      expect(profile?.costing, 'bicycle_banana');
      expect(profile?.bicycleType, isNull);
    });

    test('null / empty / non-String / malformed input returns null', () {
      expect(ValhallaProfile.parse(null), isNull);
      expect(ValhallaProfile.parse(''), isNull);
      expect(ValhallaProfile.parse('  '), isNull);
      expect(ValhallaProfile.parse(42), isNull);
      expect(ValhallaProfile.parse(const {'a': 1}), isNull);
      expect(ValhallaProfile.parse('Bad Value!'), isNull);
      expect(ValhallaProfile.parse('bicycle-road'), isNull);
      expect(ValhallaProfile.parse('_leading'), isNull);
    });

    test('has value equality', () {
      expect(
        ValhallaProfile.parse('bicycle_mountain'),
        const ValhallaProfile('bicycle', bicycleType: 'Mountain'),
      );
      expect(
        ValhallaProfile.parse('auto'),
        isNot(const ValhallaProfile('truck')),
      );
    });
  });

  group('RouteTravelBucket.valhallaProfileKey', () {
    test('derives its key from costing/costingOptions', () {
      expect(RouteTravelBucket.hiking.valhallaProfileKey, 'pedestrian');
      expect(
        RouteTravelBucket.bikingHybrid.valhallaProfileKey,
        'bicycle_hybrid',
      );
      expect(
        RouteTravelBucket.bikingMountain.valhallaProfileKey,
        'bicycle_mountain',
      );
      expect(RouteTravelBucket.bikingCross.valhallaProfileKey, 'bicycle_cross');
      expect(RouteTravelBucket.bikingRoad.valhallaProfileKey, 'bicycle_road');
    });
  });

  group('resolveValhallaProfile', () {
    test('a visible subcategory setting wins over the parent category', () {
      final category = _category('biking', 'Biking', profile: 'bicycle_hybrid');
      final subcategories = [
        _subcategory('mtb', 'biking', 'MTB', profile: 'bicycle_mountain'),
      ];

      final profile = resolveValhallaProfile(
        category: category,
        subcategoryId: 'mtb',
        subcategories: subcategories,
      );

      expect(profile, const ValhallaProfile('bicycle', bicycleType: 'Mountain'));
    });

    test('a null/empty subcategoryId falls through to the category', () {
      final category = _category('hiking', 'Hiking', profile: 'pedestrian');

      expect(
        resolveValhallaProfile(
          category: category,
          subcategoryId: null,
          subcategories: const [],
        ),
        const ValhallaProfile('pedestrian'),
      );
      expect(
        resolveValhallaProfile(
          category: category,
          subcategoryId: '',
          subcategories: const [],
        ),
        const ValhallaProfile('pedestrian'),
      );
    });

    test('a deleted subcategory falls back to the parent category', () {
      final category = _category('biking', 'Biking', profile: 'bicycle_road');

      final profile = resolveValhallaProfile(
        category: category,
        subcategoryId: 'gone',
        subcategories: const [],
      );

      expect(profile, const ValhallaProfile('bicycle', bicycleType: 'Road'));
    });

    test(
      'a subcategory hidden by user preferences STILL resolves — visibility '
      'is a display setting and must not change how a trail routes',
      () {
        final category = _category('biking', 'Biking', profile: 'bicycle_road');
        final subcategories = [
          _subcategory('mtb', 'biking', 'MTB', profile: 'bicycle_mountain'),
        ];

        // The trail is tagged MTB; the user has merely hidden MTB from their
        // picker. Routing must still use the MTB profile, not the parent's.
        final profile = resolveValhallaProfile(
          category: category,
          subcategoryId: 'mtb',
          subcategories: subcategories,
        );

        expect(
          profile,
          const ValhallaProfile('bicycle', bicycleType: 'Mountain'),
        );
      },
    );

    test(
      'an unmapped biking-shaped category falls back to bicycle_hybrid',
      () {
        final category = _category('biking', 'Biking');
        final subcategories = [
          _subcategory('mtb', 'biking', 'MTB', profile: 'bicycle_mountain'),
          _subcategory('ebike', 'biking', 'E-Bike'),
        ];

        final profile = resolveValhallaProfile(
          category: category,
          subcategoryId: null,
          subcategories: subcategories,
        );

        expect(profile, RouteTravelBucket.bikingHybrid.valhallaProfile);
        expect(profile?.bicycleType, 'Hybrid');
      },
    );

    test('returns null when nothing matches', () {
      expect(
        resolveValhallaProfile(
          category: null,
          subcategoryId: null,
          subcategories: const [],
        ),
        isNull,
      );
      expect(
        resolveValhallaProfile(
          category: _category('climbing', 'Climbing'),
          subcategoryId: 'nope',
          subcategories: const [],
        ),
        isNull,
      );
    });

    test('an auto category resolves to auto, never to bicycle_hybrid', () {
      final category = _category('car', 'Car', profile: 'auto');
      final subcategories = [
        // A bicycle-costing subcategory belonging to a *different* category
        // must not drag the Car category into the bicycle tier.
        _subcategory('mtb', 'biking', 'MTB', profile: 'bicycle_mountain'),
      ];

      final profile = resolveValhallaProfile(
        category: category,
        subcategoryId: null,
        subcategories: subcategories,
      );

      expect(profile, const ValhallaProfile('auto'));
      expect(profile?.bicycleType, isNull);
    });
  });

  group('costingForCategory', () {
    test('returns bicycle for any bike resolution', () {
      final category = _category('biking', 'Biking');
      final subcategories = [
        _subcategory('mtb', 'biking', 'MTB', profile: 'bicycle_mountain'),
      ];

      expect(
        costingForCategory(
          category: category,
          subcategoryId: 'mtb',
          subcategories: subcategories,
        ),
        'bicycle',
      );
    });

    test('returns pedestrian for hiking', () {
      expect(
        costingForCategory(
          category: _category('hiking', 'Hiking', profile: 'pedestrian'),
          subcategoryId: null,
          subcategories: const [],
        ),
        'pedestrian',
      );
    });

    test('returns auto for an auto-mapped category', () {
      expect(
        costingForCategory(
          category: _category('car', 'Car', profile: 'auto'),
          subcategoryId: null,
          subcategories: const [],
        ),
        'auto',
      );
    });

    test('defaults to pedestrian when nothing resolves', () {
      expect(
        costingForCategory(
          category: null,
          subcategoryId: null,
          subcategories: const [],
        ),
        'pedestrian',
      );
    });
  });

  group('costingForTrail', () {
    test('resolves from the trail\'s own subcategory', () {
      final trail = _trail(
        category: _category('biking', 'Biking'),
        subcategoryId: 'mtb',
      );

      expect(
        costingForTrail(
          trail,
          subcategories: [
            _subcategory('mtb', 'biking', 'MTB', profile: 'bicycle_mountain'),
          ],
        ),
        'bicycle',
      );
    });

    test('falls back to the trail\'s category when it has no subcategory', () {
      final trail = _trail(category: _category('car', 'Car', profile: 'auto'));

      expect(
        costingForTrail(
          trail,
          subcategories: const [],
        ),
        'auto',
      );
    });

    test('falls back to the category when the subcategory id is dangling', () {
      // cascadeDelete: false means a deleted subcategory leaves a dangling id
      // on the trail — it must not resolve against a subcategory that is gone.
      final trail = _trail(
        category: _category('car', 'Car', profile: 'auto'),
        subcategoryId: 'deleted',
      );

      expect(
        costingForTrail(trail, subcategories: const []),
        'auto',
      );
    });

    test('defaults to pedestrian for a null trail (trail-less recording)', () {
      expect(
        costingForTrail(
          null,
          subcategories: const [],
        ),
        'pedestrian',
      );
    });
  });

  group('bucketForProfile', () {
    test('narrows the 5 bucket profiles back to their bucket', () {
      for (final bucket in RouteTravelBucket.values) {
        expect(bucketForProfile(bucket.valhallaProfile), bucket);
      }
    });

    test('returns null for a non-bucket costing and for null', () {
      expect(bucketForProfile(const ValhallaProfile('auto')), isNull);
      expect(bucketForProfile(null), isNull);
    });
  });

  group('categorySelectionForBucket', () {
    test('prefers a matching subcategory over a matching category', () {
      final categories = [
        _category('biking', 'Biking', profile: 'bicycle_mountain'),
      ];
      final subcategories = [
        _subcategory('mtb', 'biking', 'MTB', profile: 'bicycle_mountain'),
      ];

      final selection = categorySelectionForBucket(
        RouteTravelBucket.bikingMountain,
        categories,
        subcategories,
      );

      expect(selection?.categoryId, 'biking');
      expect(selection?.subcategoryId, 'mtb');
    });

    test('falls back to a category-only match', () {
      final categories = [
        _category('hiking', 'Hiking', profile: 'pedestrian'),
      ];

      final selection = categorySelectionForBucket(
        RouteTravelBucket.hiking,
        categories,
        const [],
      );

      expect(selection?.categoryId, 'hiking');
      expect(selection?.subcategoryId, isNull);
    });

    test(
      'never auto-assigns a subcategory the user has hidden — falls back to '
      'a matching category instead',
      () {
        final categories = [
          _category('biking', 'Biking', profile: 'bicycle_mountain'),
        ];
        final subcategories = [
          _subcategory('mtb', 'biking', 'MTB', profile: 'bicycle_mountain'),
        ];

        final selection = categorySelectionForBucket(
          RouteTravelBucket.bikingMountain,
          categories,
          subcategories,
          subcategoryPrefs: const [
            SubcategoryPreference(user: 'u', subcategory: 'mtb', visible: false),
          ],
        );

        expect(selection?.categoryId, 'biking');
        expect(selection?.subcategoryId, isNull);
      },
    );

    test('assigns a visible subcategory normally', () {
      final categories = [_category('biking', 'Biking')];
      final subcategories = [
        _subcategory('mtb', 'biking', 'MTB', profile: 'bicycle_mountain'),
      ];

      final selection = categorySelectionForBucket(
        RouteTravelBucket.bikingMountain,
        categories,
        subcategories,
        subcategoryPrefs: const [
          SubcategoryPreference(user: 'u', subcategory: 'mtb', visible: true),
        ],
      );

      expect(selection?.subcategoryId, 'mtb');
    });

    test('returns null when nothing maps to the bucket', () {
      final categories = [_category('other', 'Other')];

      expect(
        categorySelectionForBucket(
          RouteTravelBucket.bikingRoad,
          categories,
          const [],
        ),
        isNull,
      );
      expect(
        categorySelectionForBucket(
          RouteTravelBucket.hiking,
          const [],
          const [],
        ),
        isNull,
      );
    });

    test('an auto-mapped category is never returned for any bucket', () {
      final categories = [_category('car', 'Car', profile: 'auto')];

      for (final bucket in RouteTravelBucket.values) {
        expect(
          categorySelectionForBucket(bucket, categories, const []),
          isNull,
          reason: bucket.name,
        );
      }
    });
  });
}
