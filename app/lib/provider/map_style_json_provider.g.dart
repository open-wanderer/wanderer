// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'map_style_json_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
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

@ProviderFor(mapStyleJson)
final mapStyleJsonProvider = MapStyleJsonProvider._();

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

final class MapStyleJsonProvider
    extends $FunctionalProvider<AsyncValue<String>, String, FutureOr<String>>
    with $FutureModifier<String>, $FutureProvider<String> {
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
  MapStyleJsonProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'mapStyleJsonProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$mapStyleJsonHash();

  @$internal
  @override
  $FutureProviderElement<String> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<String> create(Ref ref) {
    return mapStyleJson(ref);
  }
}

String _$mapStyleJsonHash() => r'153afd3e727c8270d857c1d1ec0d874cbb8a6bab';
