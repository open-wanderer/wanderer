import 'dart:io';

import 'package:quick_actions/quick_actions.dart';
import 'package:wanderer/i18n/app_localizations.dart';

/// Launcher shortcuts: Android app shortcuts (long-press on the app icon,
/// pinnable to the home screen) and iOS Home Screen Quick Actions.
///
/// The type strings are stored in shortcuts the user pinned, so they are
/// stable ids rather than the routes themselves — a route can move without
/// breaking a pinned icon.
const _newTrail = 'new_trail';
const _library = 'library';

/// The route a shortcut opens, or null for a type this build does not know
/// (a shortcut pinned by an older or newer version).
String? shortcutLocation(String type) => switch (type) {
  _newTrail => '/trail/create',
  _library => '/library',
  _ => null,
};

/// The shortcuts, titled in the app's current language. Icons are
/// `ic_shortcut_*` drawables on Android and `Shortcut*` template images in the
/// iOS asset catalog.
List<ShortcutItem> shortcutItems(AppLocalizations l10n) => [
  ShortcutItem(
    type: _newTrail,
    localizedTitle: l10n.shortcut_new_trail,
    icon: Platform.isIOS ? 'ShortcutNewTrail' : 'ic_shortcut_new_trail',
  ),
  ShortcutItem(
    type: _library,
    localizedTitle: l10n.library,
    icon: Platform.isIOS ? 'ShortcutLibrary' : 'ic_shortcut_library',
  ),
];
