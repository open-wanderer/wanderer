import 'package:flutter/material.dart';
import 'package:flutter/services.dart' show rootBundle;
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/map_style_sources_provider.dart';

part 'map_style_json_provider.g.dart';

/// The app's single style-JSON provider. Loads the theme-appropriate
/// MapLibre style asset and substitutes the operator's
/// tile, glyph, and sprite endpoints from [mapStyleSourcesProvider], which
/// resolves from the network when reachable and from its persisted copy when
/// not — so this provider never blocks on connectivity.
///
/// Watches [themeModeProvider] so the provider re-runs on theme change,
/// enabling a live theme swap.
///
/// The assets embed three unique sentinel tokens (`__TILE_URL__`,
/// `__GLYPH_URL__`, `__SPRITE_URL__`); a plain [String.replaceAll] on each is
/// lossless because the tokens never collide with legitimate style content.
/// The three sentinels are always filled with the real operator values, even
/// though the composed style is subsequently rewritten to loopback URLs by
/// `rewriteStyleForProxy` — the operator values are exactly what the proxy
/// redirects a cache miss to.
@Riverpod(keepAlive: true)
Future<String> mapStyleJson(Ref ref) async {
  final mode = ref.watch(themeModeProvider);
  final sources = await ref.watch(mapStyleSourcesProvider.future);
  final brightness = effectiveBrightness(mode);
  final assetPath = brightness == Brightness.dark
      ? 'assets/map/wanderer_dark.json'
      : 'assets/map/wanderer_light.json';

  // spriteUrl is a theme-agnostic base (".../sprites/v4"); MapLibre resolves
  // "<sprite>@2x.png"/"<sprite>.json" etc. directly against it, so the
  // per-theme variant ("/light" or "/dark") must be appended here.
  final spriteVariant = brightness == Brightness.dark ? 'dark' : 'light';

  final raw = await rootBundle.loadString(assetPath);
  return raw
      .replaceAll('__TILE_URL__', sources.tileUrl)
      .replaceAll('__GLYPH_URL__', sources.glyphUrl)
      .replaceAll('__SPRITE_URL__', '${sources.spriteUrl}/$spriteVariant');
}

/// Resolves the effective [Brightness] for a given [ThemeMode], falling back
/// to the platform's current brightness when the mode is [ThemeMode.system].
Brightness effectiveBrightness(ThemeMode mode) {
  if (mode == ThemeMode.dark) return Brightness.dark;
  if (mode == ThemeMode.light) return Brightness.light;
  return WidgetsBinding.instance.platformDispatcher.platformBrightness;
}
