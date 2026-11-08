// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'map_style_json_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
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

@ProviderFor(mapStyleJson)
final mapStyleJsonProvider = MapStyleJsonProvider._();

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

final class MapStyleJsonProvider
    extends $FunctionalProvider<AsyncValue<String>, String, FutureOr<String>>
    with $FutureModifier<String>, $FutureProvider<String> {
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
