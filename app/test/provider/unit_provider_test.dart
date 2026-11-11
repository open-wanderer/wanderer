import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';

void main() {
  group('unitProvider', () {
    ProviderContainer containerWith(Settings? settings) {
      return ProviderContainer(
        overrides: [settingsProvider.overrideWithValue(settings)],
      );
    }

    test('returns "metric" when settings is null', () {
      final container = containerWith(null);
      addTearDown(container.dispose);
      expect(container.read(unitProvider), 'metric');
    });

    test('returns the stored unit when settings.unit is "imperial"', () {
      final container = containerWith(const Settings(unit: 'imperial'));
      addTearDown(container.dispose);
      expect(container.read(unitProvider), 'imperial');
    });

    test('returns "metric" when settings.unit is null', () {
      final container = containerWith(const Settings(unit: null));
      addTearDown(container.dispose);
      expect(container.read(unitProvider), 'metric');
    });
  });
}
