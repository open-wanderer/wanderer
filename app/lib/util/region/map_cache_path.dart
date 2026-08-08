import 'package:path/path.dart' as p;

/// Path-safety helpers for the app-wide glyph/sprite cache.
///
/// The glyph/sprite URLs are operator-controlled (`/map/style-sources`) and the
/// `{fontstack}`/`{range}` tokens flow into on-disk path segments under
/// `<app-docs>/map_cache`. To prevent path traversal, every cache filename
/// derives ONLY from the 4 known fontstacks + a strictly numeric range,
/// joined via `package:path` and rooted at the supplied cache root — an
/// unexpected fontstack or a `..`-bearing / non-numeric range is rejected
/// with an `ArgumentError` before any path is built.

/// The 4 whitelisted fontstacks (confirmed against the theme `text-font`
/// expressions). Any other fontstack is rejected.
const List<String> allowedFontstacks = <String>[
  'Noto Sans Regular',
  'Noto Sans Medium',
  'Noto Sans Italic',
  'Noto Sans Devanagari Regular v1',
];

/// A glyph range must be two non-negative integers joined by a hyphen
/// (e.g. `0-255`). This rejects `../../0-255`, `0-255/../..`, and anything else
/// that could escape the cache root.
final RegExp _rangePattern = RegExp(r'^\d+-\d+$');

/// Whether [fontstack] is one of the 4 whitelisted fontstacks.
bool isAllowedFontstack(String fontstack) =>
    allowedFontstacks.contains(fontstack);

/// Build the on-disk cache path for a single glyph range .pbf under [root].
///
/// Returns `<root>/glyphs/<fontstack>/<range>.pbf`, joined via `package:path`.
/// Throws [ArgumentError] if [fontstack] is not whitelisted or [range] is not a
/// strictly numeric `\d+-\d+` range — no path is returned in that case.
String glyphCacheFilePath(String root, String fontstack, String range) {
  if (!isAllowedFontstack(fontstack)) {
    throw ArgumentError.value(
      fontstack,
      'fontstack',
      'not a whitelisted fontstack',
    );
  }
  if (!_rangePattern.hasMatch(range)) {
    throw ArgumentError.value(range, 'range', r'range must match ^\d+-\d+$');
  }
  return p.join(root, 'glyphs', fontstack, '$range.pbf');
}

/// Build the sprite base path (light or dark) under [root].
///
/// Returns `<root>/sprite/<light|dark>` — a base to which the MapLibre sprite
/// loader appends `.json` / `.png` / `@2x.png`. Both variants live under the
/// same `<root>/sprite/` directory. The variant name is a
/// hard-coded literal, so no external input reaches this path segment.
String spriteCacheBasePath(String root, {required bool dark}) {
  return p.join(root, 'sprite', dark ? 'dark' : 'light');
}
