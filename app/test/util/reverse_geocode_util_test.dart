import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/reverse_geocode_util.dart';

void main() {
  group('getLocationDescription', () {
    test(
      'includeRoad:true builds comma-joined "road, city, state, country" '
      'in that exact order',
      () {
        final address = <String, dynamic>{
          'road': 'Bahnhofstrasse',
          'city': 'Zürich',
          'state': 'Zürich',
          'country': 'Switzerland',
        };

        final description = getLocationDescription(address, includeRoad: true);

        expect(description, 'Bahnhofstrasse, Zürich, Zürich, Switzerland');
      },
    );

    test(
      'includeCountry:false omits the country segment (used for label)',
      () {
        final address = <String, dynamic>{
          'road': 'Bahnhofstrasse',
          'city': 'Zürich',
          'state': 'Zürich',
          'country': 'Switzerland',
        };

        final description = getLocationDescription(
          address,
          includeRoad: true,
          includeCountry: false,
        );

        expect(description, 'Bahnhofstrasse, Zürich, Zürich');
      },
    );

    test(
      'default includeCountry includes the country segment (used for fullLabel)',
      () {
        final address = <String, dynamic>{'city': 'Zürich', 'country': 'Switzerland'};

        final description = getLocationDescription(address);

        expect(description, 'Zürich, Switzerland');
      },
    );

    test(
      'city fallback chain uses city, else town, else hamlet, else village '
      '(city present wins)',
      () {
        final address = <String, dynamic>{
          'city': 'Zürich',
          'town': 'ShouldNotAppear',
          'hamlet': 'ShouldNotAppear',
          'village': 'ShouldNotAppear',
        };

        expect(getLocationDescription(address), 'Zürich');
      },
    );

    test('city fallback chain falls to town when city absent', () {
      final address = <String, dynamic>{
        'town': 'SomeTown',
        'hamlet': 'ShouldNotAppear',
        'village': 'ShouldNotAppear',
      };

      expect(getLocationDescription(address), 'SomeTown');
    });

    test('city fallback chain falls to hamlet when city and town absent', () {
      final address = <String, dynamic>{
        'hamlet': 'SomeHamlet',
        'village': 'ShouldNotAppear',
      };

      expect(getLocationDescription(address), 'SomeHamlet');
    });

    test(
      'city fallback chain falls to village when city, town, hamlet absent',
      () {
        final address = <String, dynamic>{'village': 'SomeVillage'};

        expect(getLocationDescription(address), 'SomeVillage');
      },
    );

    test(
      'includeRoad:false (the default) omits the road segment entirely, '
      'even when address[road] is present',
      () {
        final address = <String, dynamic>{'road': 'Bahnhofstrasse', 'city': 'Zürich'};

        expect(getLocationDescription(address), 'Zürich');
      },
    );

    test(
      'missing/absent address keys are skipped (no empty commas): an '
      'address with only country yields just the country',
      () {
        final address = <String, dynamic>{'country': 'Switzerland'};

        expect(getLocationDescription(address), 'Switzerland');
      },
    );

    test('an entirely empty address yields an empty string', () {
      expect(getLocationDescription(<String, dynamic>{}), '');
    });
  });

  group('getReverseLocationResult', () {
    test(
      'label omits country, fullLabel includes country; country = '
      'address[country] ?? empty string',
      () {
        final address = <String, dynamic>{
          'road': 'Bahnhofstrasse',
          'city': 'Zürich',
          'country': 'Switzerland',
        };

        final result = getReverseLocationResult(address, includeRoad: true);

        expect(result.label, 'Bahnhofstrasse, Zürich');
        expect(result.fullLabel, 'Bahnhofstrasse, Zürich, Switzerland');
        expect(result.country, 'Switzerland');
      },
    );

    test('country defaults to empty string when address[country] is absent', () {
      final address = <String, dynamic>{'city': 'Zürich'};

      final result = getReverseLocationResult(address);

      expect(result.country, '');
    });

    test(
      'label falls back to fullLabel when the country-less description is '
      'empty (address has only country: label == fullLabel == country)',
      () {
        final address = <String, dynamic>{'country': 'Switzerland'};

        final result = getReverseLocationResult(address);

        expect(result.label, 'Switzerland');
        expect(result.fullLabel, 'Switzerland');
        expect(result.country, 'Switzerland');
      },
    );
  });
}
