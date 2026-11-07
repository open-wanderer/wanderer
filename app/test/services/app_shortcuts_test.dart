// The shortcut icons are looked up by name at runtime (Android
// getIdentifier(), iOS UIApplicationShortcutIcon(templateImageName:)), so a
// renamed or missing resource only shows up as a blank icon on a device. These
// tests pin the names to the files that ship.

import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/i18n/app_localizations_en.dart';
import 'package:wanderer/provider/pending_launch_location_provider.dart';
import 'package:wanderer/services/app_shortcuts.dart';

void main() {
  final items = shortcutItems(AppLocalizationsEn());

  test('every shortcut opens a route the router defines', () {
    final router = File('lib/provider/router_provider.dart').readAsStringSync();
    for (final item in items) {
      final location = shortcutLocation(item.type);
      expect(location, isNotNull, reason: item.type);
      expect(router, contains("path: '$location'"), reason: item.type);
    }
  });

  test('a shortcut type this build does not know opens nothing', () {
    expect(shortcutLocation('pinned_by_a_later_version'), isNull);
  });

  test('the Android shortcut icons ship and are kept from the shrinker', () {
    const res = 'android/app/src/main/res';
    final keep = File('$res/raw/keep.xml').readAsStringSync();
    expect(keep, contains('@drawable/ic_shortcut_*'));
    for (final item in items) {
      expect(item.icon, startsWith('ic_shortcut_'));
      expect(File('$res/drawable/${item.icon}.xml').existsSync(), isTrue);
    }
  });

  test('the iOS shortcut icons ship as template images', () {
    for (final name in ['ShortcutNewTrail', 'ShortcutLibrary']) {
      final set = 'ios/Runner/Assets.xcassets/$name.imageset';
      expect(File('$set/$name.svg').existsSync(), isTrue, reason: name);
      expect(
        File('$set/Contents.json').readAsStringSync(),
        contains('"template-rendering-intent" : "template"'),
        reason: name,
      );
    }
  });

  test('a pending launch location is handed out once', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    final pending = container.read(pendingLaunchLocationProvider.notifier);

    pending.set('/library');
    expect(pending.take(), '/library');
    expect(pending.take(), isNull);
  });
}
