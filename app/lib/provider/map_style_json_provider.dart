import 'package:flutter/material.dart';
import 'package:flutter/services.dart' show rootBundle;
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/map_style_provider.dart' show effectiveBrightness;
import 'package:wanderer/provider/map_style_sources_provider.dart';

part 'map_style_json_provider.g.dart';

/// Loads the theme-appropriate MapLibre style asset and injects the operator's
/// tile, glyph, and sprite endpoints (from [mapStyleSourcesProvider]) into the
/// raw style-JSON String that MapLibre GL Native consumes (STYLE-02/03/04).
///
/// Watches [themeModeProvider] so the provider re-runs on theme change — the
/// enabling half of the live theme swap consumed in 15-04 (CORE-02).
///
/// The assets embed three unique sentinel tokens (`__TILE_URL__`,
/// `__GLYPH_URL__`, `__SPRITE_URL__`); a plain [String.replaceAll] on each is
/// lossless because the tokens never collide with legitimate style content.
///
/// This ADDS a provider alongside the legacy `mapStyleProvider` (returning
/// `vtr.Style`), which stays untouched so the four not-yet-migrated flutter_map
/// screens keep rendering.
@Riverpod(keepAlive: true)
Future<String> mapStyleJson(Ref ref) async {
  final mode = ref.watch(themeModeProvider);
  final sources = await ref.watch(mapStyleSourcesProvider.future);
  final brightness = effectiveBrightness(mode);
  final assetPath = brightness == Brightness.dark
      ? 'assets/map/wanderer_dark.json'
      : 'assets/map/wanderer_light.json';

  final raw = await rootBundle.loadString(assetPath);
  return raw
      .replaceAll('__TILE_URL__', sources.tileUrl)
      .replaceAll('__GLYPH_URL__', sources.glyphUrl)
      .replaceAll('__SPRITE_URL__', sources.spriteUrl);
}
