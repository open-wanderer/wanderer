// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'glyph_sprite_cache_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The one shared app-wide glyph/sprite cache (GLYPH-04, D-08).
///
/// On first read this `keepAlive` provider downloads every glyph range for the
/// 4 whitelisted fontstacks plus both light and dark sprite sheets into
/// `<app-docs>/map_cache`, idempotently (files already on disk are skipped, so a
/// second trail download / map open is a no-op — OFFL-01, D-10). Every local
/// path is built via the Task-1 path-safety helpers, so no operator-controlled
/// token is ever concatenated into a path (T-15-03-01).

@ProviderFor(GlyphSpriteCache)
final glyphSpriteCacheProvider = GlyphSpriteCacheProvider._();

/// The one shared app-wide glyph/sprite cache (GLYPH-04, D-08).
///
/// On first read this `keepAlive` provider downloads every glyph range for the
/// 4 whitelisted fontstacks plus both light and dark sprite sheets into
/// `<app-docs>/map_cache`, idempotently (files already on disk are skipped, so a
/// second trail download / map open is a no-op — OFFL-01, D-10). Every local
/// path is built via the Task-1 path-safety helpers, so no operator-controlled
/// token is ever concatenated into a path (T-15-03-01).
final class GlyphSpriteCacheProvider
    extends $AsyncNotifierProvider<GlyphSpriteCache, GlyphSpriteCachePaths> {
  /// The one shared app-wide glyph/sprite cache (GLYPH-04, D-08).
  ///
  /// On first read this `keepAlive` provider downloads every glyph range for the
  /// 4 whitelisted fontstacks plus both light and dark sprite sheets into
  /// `<app-docs>/map_cache`, idempotently (files already on disk are skipped, so a
  /// second trail download / map open is a no-op — OFFL-01, D-10). Every local
  /// path is built via the Task-1 path-safety helpers, so no operator-controlled
  /// token is ever concatenated into a path (T-15-03-01).
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

String _$glyphSpriteCacheHash() => r'91ec39748eb03fddaea591ab3dc45f85150d928b';

/// The one shared app-wide glyph/sprite cache (GLYPH-04, D-08).
///
/// On first read this `keepAlive` provider downloads every glyph range for the
/// 4 whitelisted fontstacks plus both light and dark sprite sheets into
/// `<app-docs>/map_cache`, idempotently (files already on disk are skipped, so a
/// second trail download / map open is a no-op — OFFL-01, D-10). Every local
/// path is built via the Task-1 path-safety helpers, so no operator-controlled
/// token is ever concatenated into a path (T-15-03-01).

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
