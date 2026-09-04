// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'glyph_sprite_cache_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The offline counterpart to [GlyphSpriteCache]: returns the local
/// glyph/sprite cache paths **without any network call or download**. Used by
/// the offline map render path, where the cache was already populated online at
/// download time and the style rewriter only needs the `file://` bases. Kept
/// separate from [GlyphSpriteCache] so an offline map open never awaits (or
/// hangs on) `mapStyleSourcesProvider`.

@ProviderFor(offlineGlyphSpritePaths)
final offlineGlyphSpritePathsProvider = OfflineGlyphSpritePathsProvider._();

/// The offline counterpart to [GlyphSpriteCache]: returns the local
/// glyph/sprite cache paths **without any network call or download**. Used by
/// the offline map render path, where the cache was already populated online at
/// download time and the style rewriter only needs the `file://` bases. Kept
/// separate from [GlyphSpriteCache] so an offline map open never awaits (or
/// hangs on) `mapStyleSourcesProvider`.

final class OfflineGlyphSpritePathsProvider
    extends
        $FunctionalProvider<
          AsyncValue<GlyphSpriteCachePaths>,
          GlyphSpriteCachePaths,
          FutureOr<GlyphSpriteCachePaths>
        >
    with
        $FutureModifier<GlyphSpriteCachePaths>,
        $FutureProvider<GlyphSpriteCachePaths> {
  /// The offline counterpart to [GlyphSpriteCache]: returns the local
  /// glyph/sprite cache paths **without any network call or download**. Used by
  /// the offline map render path, where the cache was already populated online at
  /// download time and the style rewriter only needs the `file://` bases. Kept
  /// separate from [GlyphSpriteCache] so an offline map open never awaits (or
  /// hangs on) `mapStyleSourcesProvider`.
  OfflineGlyphSpritePathsProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'offlineGlyphSpritePathsProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$offlineGlyphSpritePathsHash();

  @$internal
  @override
  $FutureProviderElement<GlyphSpriteCachePaths> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<GlyphSpriteCachePaths> create(Ref ref) {
    return offlineGlyphSpritePaths(ref);
  }
}

String _$offlineGlyphSpritePathsHash() =>
    r'b8ab4ff79ceb67b91398a59de02fdb1ce10fc8f5';

/// The one shared app-wide glyph/sprite cache.
///
/// On first read this `keepAlive` provider downloads every glyph range for the
/// 4 whitelisted fontstacks plus both light and dark sprite sheets into
/// `<app-docs>/map_cache`, idempotently (files already on disk are skipped, so a
/// second trail download / map open is a no-op). Every local path is built via
/// the path-safety helpers in `map_cache_path.dart`, so no operator-controlled
/// token is ever concatenated into a path.

@ProviderFor(GlyphSpriteCache)
final glyphSpriteCacheProvider = GlyphSpriteCacheProvider._();

/// The one shared app-wide glyph/sprite cache.
///
/// On first read this `keepAlive` provider downloads every glyph range for the
/// 4 whitelisted fontstacks plus both light and dark sprite sheets into
/// `<app-docs>/map_cache`, idempotently (files already on disk are skipped, so a
/// second trail download / map open is a no-op). Every local path is built via
/// the path-safety helpers in `map_cache_path.dart`, so no operator-controlled
/// token is ever concatenated into a path.
final class GlyphSpriteCacheProvider
    extends $AsyncNotifierProvider<GlyphSpriteCache, GlyphSpriteCachePaths> {
  /// The one shared app-wide glyph/sprite cache.
  ///
  /// On first read this `keepAlive` provider downloads every glyph range for the
  /// 4 whitelisted fontstacks plus both light and dark sprite sheets into
  /// `<app-docs>/map_cache`, idempotently (files already on disk are skipped, so a
  /// second trail download / map open is a no-op). Every local path is built via
  /// the path-safety helpers in `map_cache_path.dart`, so no operator-controlled
  /// token is ever concatenated into a path.
  GlyphSpriteCacheProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'glyphSpriteCacheProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$glyphSpriteCacheHash();

  @$internal
  @override
  GlyphSpriteCache create() => GlyphSpriteCache();
}

String _$glyphSpriteCacheHash() => r'925adc62e7d07832d61831b3ca24fb893a336499';

/// The one shared app-wide glyph/sprite cache.
///
/// On first read this `keepAlive` provider downloads every glyph range for the
/// 4 whitelisted fontstacks plus both light and dark sprite sheets into
/// `<app-docs>/map_cache`, idempotently (files already on disk are skipped, so a
/// second trail download / map open is a no-op). Every local path is built via
/// the path-safety helpers in `map_cache_path.dart`, so no operator-controlled
/// token is ever concatenated into a path.

abstract class _$GlyphSpriteCache
    extends $AsyncNotifier<GlyphSpriteCachePaths> {
  FutureOr<GlyphSpriteCachePaths> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<AsyncValue<GlyphSpriteCachePaths>, GlyphSpriteCachePaths>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<GlyphSpriteCachePaths>,
                GlyphSpriteCachePaths
              >,
              AsyncValue<GlyphSpriteCachePaths>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
