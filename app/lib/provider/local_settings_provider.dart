import 'package:flutter/material.dart';
import 'package:objectbox/objectbox.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/local_settings_entity.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';

part 'local_settings_provider.g.dart';

@Riverpod(keepAlive: true)
class LocalSettingsNotifier extends _$LocalSettingsNotifier {
  Box<LocalSettingsEntity> get _box =>
      ref.read(objectBoxProvider).box<LocalSettingsEntity>();

  @override
  LocalSettingsEntity build() {
    final box = ref.watch(objectBoxProvider).box<LocalSettingsEntity>();
    return box.getAll().firstOrNull ?? LocalSettingsEntity();
  }

  Future<void> setThemeMode(ThemeMode mode) async {
    final entity = _box.getAll().firstOrNull ?? LocalSettingsEntity();
    entity.themeMode = _modeToString(mode);
    _box.put(entity);
    ref.invalidateSelf();
  }

  Future<void> markBackgroundLocationAsked() async {
    final entity = _box.getAll().firstOrNull ?? LocalSettingsEntity();
    if (entity.backgroundLocationAsked) return;
    entity.backgroundLocationAsked = true;
    _box.put(entity);
    ref.invalidateSelf();
  }

  String _modeToString(ThemeMode mode) => switch (mode) {
    ThemeMode.light => 'light',
    ThemeMode.dark => 'dark',
    ThemeMode.system => 'system',
  };
}

@Riverpod(keepAlive: true)
ThemeMode themeMode(Ref ref) {
  final settings = ref.watch(localSettingsProvider);
  return switch (settings.themeMode) {
    'light' => ThemeMode.light,
    'dark' => ThemeMode.dark,
    _ => ThemeMode.system,
  };
}

@Riverpod(keepAlive: true)
Locale? locale(Ref ref) {
  final lang = ref.watch(settingsProvider)?.language;
  if (lang == null) return null;
  return Locale(lang.name);
}

@Riverpod(keepAlive: true)
String unit(Ref ref) {
  final settings = ref.watch(settingsProvider);
  return settings?.unit ?? 'metric';
}
