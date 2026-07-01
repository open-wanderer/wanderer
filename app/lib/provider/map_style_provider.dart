import 'package:flutter/material.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';
import 'package:vector_tile_renderer/vector_tile_renderer.dart' as vtr;
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/tile_url_provider.dart';

part 'map_style_provider.g.dart';

@Riverpod(keepAlive: true)
Future<Style> mapStyle(Ref ref) async {
  final mode = ref.watch(themeModeProvider);
  final tileUrl = await ref.watch(tileUrlProvider.future);
  final brightness = effectiveBrightness(mode);
  final asset = brightness == Brightness.dark
      ? vtr.wandererDarkTheme(tileUrl)
      : vtr.wandererLightTheme(tileUrl);
  return StyleReader.map(asset).read();
}

Brightness effectiveBrightness(ThemeMode mode) {
  if (mode == ThemeMode.dark) return Brightness.dark;
  if (mode == ThemeMode.light) return Brightness.light;
  return WidgetsBinding.instance.platformDispatcher.platformBrightness;
}
