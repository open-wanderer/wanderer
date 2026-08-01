// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'map_style_json_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
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

@ProviderFor(mapStyleJson)
final mapStyleJsonProvider = MapStyleJsonProvider._();

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

final class MapStyleJsonProvider
    extends $FunctionalProvider<AsyncValue<String>, String, FutureOr<String>>
    with $FutureModifier<String>, $FutureProvider<String> {
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

/// The network-free offline counterpart to [mapStyleJson]: loads the same
/// theme-appropriate style asset but fills the three endpoint sentinels with an
/// inert placeholder instead of awaiting [mapStyleSourcesProvider]. Safe because
/// the offline render path rewrites every one of those sentinels via
/// `rewriteStyleForProxy` before the style is used. Watches only
/// [themeModeProvider] so a theme toggle still triggers a live style swap.

@ProviderFor(offlineMapStyleJson)
final offlineMapStyleJsonProvider = OfflineMapStyleJsonProvider._();

/// The network-free offline counterpart to [mapStyleJson]: loads the same
/// theme-appropriate style asset but fills the three endpoint sentinels with an
/// inert placeholder instead of awaiting [mapStyleSourcesProvider]. Safe because
/// the offline render path rewrites every one of those sentinels via
/// `rewriteStyleForProxy` before the style is used. Watches only
/// [themeModeProvider] so a theme toggle still triggers a live style swap.

final class OfflineMapStyleJsonProvider
    extends $FunctionalProvider<AsyncValue<String>, String, FutureOr<String>>
    with $FutureModifier<String>, $FutureProvider<String> {
  /// The network-free offline counterpart to [mapStyleJson]: loads the same
  /// theme-appropriate style asset but fills the three endpoint sentinels with an
  /// inert placeholder instead of awaiting [mapStyleSourcesProvider]. Safe because
  /// the offline render path rewrites every one of those sentinels via
  /// `rewriteStyleForProxy` before the style is used. Watches only
  /// [themeModeProvider] so a theme toggle still triggers a live style swap.
  OfflineMapStyleJsonProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'offlineMapStyleJsonProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$offlineMapStyleJsonHash();

  @$internal
  @override
  $FutureProviderElement<String> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<String> create(Ref ref) {
    return offlineMapStyleJson(ref);
  }
}

String _$offlineMapStyleJsonHash() =>
    r'ac89f1f239a141494836a20217d74379bdeafa9f';
