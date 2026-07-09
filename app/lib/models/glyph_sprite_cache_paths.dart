/// On-disk layout of the shared app-wide glyph/sprite cache (GLYPH-04, D-08).
///
/// All paths are rooted at `<app-docs>/map_cache`. The 15-06 offline style
/// rewriter targets these bases when pointing the MapLibre style's `glyphs` /
/// `sprite` keys at `file://` URLs.
///
/// Layout (`root` == `<app-docs>/map_cache`):
/// ```
///   root/
///     glyphs/{fontstack}/{range}.pbf
///     sprite/light.json | light.png | light@2x.png
///     sprite/dark.json  | dark.png  | dark@2x.png
/// ```
class GlyphSpriteCachePaths {
  const GlyphSpriteCachePaths({
    required this.root,
    required this.glyphBase,
    required this.spriteLightBase,
    required this.spriteDarkBase,
  });

  /// The cache root, `<app-docs>/map_cache`.
  final String root;

  /// The glyph base directory, `<root>/glyphs`. Individual ranges live under
  /// `<glyphBase>/<fontstack>/<range>.pbf`.
  final String glyphBase;

  /// The light sprite base, `<root>/sprite/light` — the MapLibre sprite loader
  /// appends `.json` / `.png` / `@2x.png`.
  final String spriteLightBase;

  /// The dark sprite base, `<root>/sprite/dark`.
  final String spriteDarkBase;
}
