// One-off CLI: dumps the flomp `vector_tile_renderer` fork's wanderer light/dark
// themes to plain MapLibre Style Spec v8 JSON assets, replacing the tile, glyph,
// and sprite endpoints with sentinel placeholder tokens that the runtime
// `mapStyleJsonProvider` string-replaces with operator-provided URLs.
//
// These assets are BUILD INPUTS, not runtime-generated: regenerate them only if
// the flomp `vector_tile_renderer` fork's wanderer themes change (see
// 15-RESEARCH.md § Runtime State Inventory). To regenerate, from `app/`:
//
//     dart run tool/extract_map_styles.dart
//
// The themes already carry real Protomaps `glyphs`/`sprite` values today; this
// CLI OVERWRITES them (and the source `tiles[0]` URL) with the sentinel tokens
// `__TILE_URL__` / `__GLYPH_URL__` / `__SPRITE_URL__` so the runtime provider can
// inject the operator-configured endpoints losslessly.

import 'dart:convert';
import 'dart:io';

// Import the theme source files directly rather than the public
// `vector_tile_renderer.dart` barrel. The barrel transitively pulls in the
// package's FFI/Flutter-engine code, which crashes the pure-Dart `dart run` VM
// compiler ("type 'InvalidType' is not a subtype of type 'FunctionType'"). The
// wanderer theme files are self-contained `Map<String, dynamic>` literals with
// no imports, so importing them in isolation keeps this CLI runnable under
// `dart run` without a Flutter engine.
import 'package:vector_tile_renderer/src/themes/wanderer/wanderer_light_theme.dart'
    show wandererLightTheme;
import 'package:vector_tile_renderer/src/themes/wanderer/wanderer_dark_theme.dart'
    show wandererDarkTheme;

const _tilePlaceholder = '__TILE_URL__';
const _glyphPlaceholder = '__GLYPH_URL__';
const _spritePlaceholder = '__SPRITE_URL__';

void main() {
  _write('assets/map/wanderer_light.json', wandererLightTheme(_tilePlaceholder));
  _write('assets/map/wanderer_dark.json', wandererDarkTheme(_tilePlaceholder));
  stdout.writeln('Wrote assets/map/wanderer_light.json and assets/map/wanderer_dark.json');
}

void _write(String path, Map<String, dynamic> style) {
  // Force the glyph/sprite endpoints to sentinel tokens so the runtime provider
  // injects operator URLs. `tiles[0]` already carries `__TILE_URL__` because the
  // theme function was called with the placeholder above.
  style['glyphs'] = _glyphPlaceholder;
  style['sprite'] = _spritePlaceholder;

  final json = const JsonEncoder.withIndent('  ').convert(style);
  final file = File(path);
  file.parent.createSync(recursive: true);
  file.writeAsStringSync('$json\n');
}
