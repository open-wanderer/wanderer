import 'package:flutter/material.dart';
import 'package:flutter/services.dart' show rootBundle;
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/map_style_sources_provider.dart';

part 'map_style_json_provider.g.dart';

/// Loads the theme-appropriate MapLibre style asset and injects the operator's
/// tile, glyph, and sprite endpoints (from [mapStyleSourcesProvider]) into the
/// raw style-JSON String that MapLibre GL Native consumes.
///
/// Watches [themeModeProvider] so the provider re-runs on theme change,
/// enabling a live theme swap.
///
/// The assets embed three unique sentinel tokens (`__TILE_URL__`,
/// `__GLYPH_URL__`, `__SPRITE_URL__`); a plain [String.replaceAll] on each is
/// lossless because the tokens never collide with legitimate style content.
///
/// This provider exists alongside the legacy `mapStyleProvider` (returning
/// `vtr.Style`), which stays in place because some flutter_map screens still
/// rely on it.
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

/// Inert placeholder substituted for the operator tile/glyph/sprite sentinels
/// on the offline path. Every one of these is overwritten by
/// `rewriteStyleForProxy` (tiles → loopback proxy, glyphs/sprite → `file://`)
/// before the style reaches MapLibre, so the value is never used at render
/// time — it exists only to keep the style JSON syntactically valid.
const String offlineSentinelPlaceholder = 'about:blank';

/// Fills the three operator-endpoint sentinels in a raw style asset with
/// [offlineSentinelPlaceholder]. Pure counterpart to [mapStyleJson]'s
/// network-sourced substitution — used by [offlineMapStyleJson] so the offline
/// path never depends on `/map/style-sources`. The placeholders are safe
/// because the offline render path overwrites every one via
/// `rewriteStyleForProxy`.
String fillOfflineStyleSentinels(String raw) {
  return raw
      .replaceAll('__TILE_URL__', offlineSentinelPlaceholder)
      .replaceAll('__GLYPH_URL__', offlineSentinelPlaceholder)
      .replaceAll('__SPRITE_URL__', offlineSentinelPlaceholder);
}

/// The network-free offline counterpart to [mapStyleJson]: loads the same
/// theme-appropriate style asset but fills the three endpoint sentinels with an
/// inert placeholder instead of awaiting [mapStyleSourcesProvider]. Safe because
/// the offline render path rewrites every one of those sentinels via
/// `rewriteStyleForProxy` before the style is used. Watches only
/// [themeModeProvider] so a theme toggle still triggers a live style swap.
@Riverpod(keepAlive: true)
Future<String> offlineMapStyleJson(Ref ref) async {
  final mode = ref.watch(themeModeProvider);
  final brightness = effectiveBrightness(mode);
  final assetPath = brightness == Brightness.dark
      ? 'assets/map/wanderer_dark.json'
      : 'assets/map/wanderer_light.json';

  final raw = await rootBundle.loadString(assetPath);
  return fillOfflineStyleSentinels(raw);
}

/// Resolves the effective [Brightness] for a given [ThemeMode], falling back
/// to the platform's current brightness when the mode is [ThemeMode.system].
Brightness effectiveBrightness(ThemeMode mode) {
  if (mode == ThemeMode.dark) return Brightness.dark;
  if (mode == ThemeMode.light) return Brightness.light;
  return WidgetsBinding.instance.platformDispatcher.platformBrightness;
}
