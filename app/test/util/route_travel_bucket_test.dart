import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/route_travel_bucket.dart';

void main() {
  group('RouteTravelBucket costing + costingOptions payloads', () {
    test('hiking has the pedestrian costing profile and fixed options', () {
      expect(RouteTravelBucket.hiking.costing, 'pedestrian');
      expect(RouteTravelBucket.hiking.costingOptions, {
        'max_hiking_difficulty': 6,
        'walking_speed': 5.1,
        'use_hills': 1,
        'shortest': false,
      });
    });

    test('bikingHybrid has the bicycle costing profile and fixed options', () {
      expect(RouteTravelBucket.bikingHybrid.costing, 'bicycle');
      expect(RouteTravelBucket.bikingHybrid.costingOptions, {
        'bicycle_type': 'Hybrid',
        'cycling_speed': 18,
        'use_roads': 0.5,
        'use_hills': 0.5,
        'avoid_bad_surfaces': 0.25,
        'shortest': false,
      });
    });

    test(
      'bikingRoad costingOptions has bicycle_type Road and cycling_speed 25',
      () {
        final opts = RouteTravelBucket.bikingRoad.costingOptions;
        expect(opts['bicycle_type'], 'Road');
        expect(opts['cycling_speed'], 25);
      },
    );

    test(
      'bikingCross costingOptions has bicycle_type Cross and cycling_speed 20',
      () {
        final opts = RouteTravelBucket.bikingCross.costingOptions;
        expect(opts['bicycle_type'], 'Cross');
        expect(opts['cycling_speed'], 20);
      },
    );

    test(
      'bikingMountain costingOptions has bicycle_type Mountain and '
      'cycling_speed 16',
      () {
        final opts = RouteTravelBucket.bikingMountain.costingOptions;
        expect(opts['bicycle_type'], 'Mountain');
        expect(opts['cycling_speed'], 16);
      },
    );

    test(
      'every bike bucket shares use_roads 0.5, use_hills 0.5, '
      'avoid_bad_surfaces 0.25; every bucket has shortest false',
      () {
        const bikeBuckets = [
          RouteTravelBucket.bikingHybrid,
          RouteTravelBucket.bikingMountain,
          RouteTravelBucket.bikingCross,
          RouteTravelBucket.bikingRoad,
        ];
        for (final bucket in bikeBuckets) {
          final opts = bucket.costingOptions;
          expect(opts['use_roads'], 0.5, reason: bucket.name);
          expect(opts['use_hills'], 0.5, reason: bucket.name);
          expect(opts['avoid_bad_surfaces'], 0.25, reason: bucket.name);
        }

        for (final bucket in RouteTravelBucket.values) {
          expect(bucket.costingOptions['shortest'], false, reason: bucket.name);
        }
      },
    );
  });

  group('bucketForState', () {
    test("pedestrian always resolves to hiking, regardless of options", () {
      expect(
        bucketForState('pedestrian', const {'bicycle_type': 'Road'}),
        RouteTravelBucket.hiking,
      );
      expect(bucketForState('pedestrian', null), RouteTravelBucket.hiking);
    });

    test('bicycle + bicycle_type Road resolves to bikingRoad', () {
      expect(
        bucketForState('bicycle', const {'bicycle_type': 'Road'}),
        RouteTravelBucket.bikingRoad,
      );
    });

    test('bicycle + bicycle_type Mountain resolves to bikingMountain', () {
      expect(
        bucketForState('bicycle', const {'bicycle_type': 'Mountain'}),
        RouteTravelBucket.bikingMountain,
      );
    });

    test('bicycle + bicycle_type Cross resolves to bikingCross', () {
      expect(
        bucketForState('bicycle', const {'bicycle_type': 'Cross'}),
        RouteTravelBucket.bikingCross,
      );
    });

    test('bicycle + null options falls back to bikingHybrid', () {
      expect(bucketForState('bicycle', null), RouteTravelBucket.bikingHybrid);
    });

    test(
      'bicycle + unrecognized bicycle_type falls back to bikingHybrid',
      () {
        expect(
          bucketForState('bicycle', const {'bicycle_type': 'Unknown'}),
          RouteTravelBucket.bikingHybrid,
        );
      },
    );

    test('a costing matching no bucket returns null, never bikingHybrid', () {
      expect(bucketForState('auto', null), isNull);
      expect(bucketForState('truck', const {'bicycle_type': 'Road'}), isNull);
      expect(bucketForState('motor_scooter', null), isNull);
    });
  });

  group('RouteTravelBucket.valhallaProfileKey / valhallaProfile', () {
    test('derives the settings string from costing/costingOptions', () {
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

    test('every bucket profile round-trips its own costing payload', () {
      for (final bucket in RouteTravelBucket.values) {
        final profile = bucket.valhallaProfile;
        expect(profile.costing, bucket.costing, reason: bucket.name);
        expect(
          profile.bicycleType,
          bucket.costingOptions['bicycle_type'],
          reason: bucket.name,
        );
      }
    });
  });
}
