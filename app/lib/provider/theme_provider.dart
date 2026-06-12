import 'package:flutter/material.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/settings_entity.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';

part 'theme_provider.g.dart';

@Riverpod(keepAlive: true)
class ThemeModeNotifier extends _$ThemeModeNotifier {
  @override
  ThemeMode build() {
    final box = ref.watch(objectBoxProvider).box<SettingsEntity>();
    final settings = box.getAll().firstOrNull;
    return _parseMode(settings?.themeMode);
  }

  Future<void> setMode(ThemeMode mode) async {
    final box = ref.read(objectBoxProvider).box<SettingsEntity>();
    final existing = box.getAll().firstOrNull;
    final settings = existing ?? SettingsEntity();
    settings.themeMode = _modeToString(mode);
    box.put(settings);
    state = mode;
  }

  ThemeMode _parseMode(String? value) {
    switch (value) {
      case 'light':
        return ThemeMode.light;
      case 'dark':
        return ThemeMode.dark;
      default:
        return ThemeMode.system;
    }
  }

  String _modeToString(ThemeMode mode) {
    switch (mode) {
      case ThemeMode.light:
        return 'light';
      case ThemeMode.dark:
        return 'dark';
      case ThemeMode.system:
        return 'system';
    }
  }
}
