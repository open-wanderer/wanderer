/// Rewrites a MapLibre style so every URL-bearing field — vector/DEM tiles,
/// glyphs and sprite — resolves through the loopback tile proxy
/// (`tile_proxy_server.dart`). Emits no filesystem path.
library;

import 'dart:convert';

/// Deepest zoom present in a locally-extracted `.pmtiles` cell. Must match
/// `maxZoom` in `db/services/tiles/generator.go`.
///
/// Applied online too: the proxy cannot fake overzoom — MVT coordinates are
/// tile-local, so a z14 parent served at z15 crushes its geometry into the
/// child tile — and a deeper cap would blank a downloaded region above its
/// local depth. Costs a little online sharpness.
const int _proxyVectorMaxZoom = 14;

/// Deepest zoom present in a locally-extracted DEM cell. Must match
/// `demMaxZoom` in `db/services/tiles/generator.go`. Lower than
/// [_proxyVectorMaxZoom] — hillshading needs less detail — and applied for
/// the same reason.
const int _proxyDemMaxZoom = 12;

/// Routes every URL-bearing field in [style] at [proxyBaseUrl]:
///
///  * vector sources → `/vector/{z}/{x}/{y}.pbf`, capped at
///    [_proxyVectorMaxZoom];
///  * `raster-dem` sources → `/dem/{z}/{x}/{y}.png`, capped at
///    [_proxyDemMaxZoom], plus `encoding: terrarium` and `tileSize: 512`
///    (online those come from Mapterhorn's tilejson, so the style lacks them);
///  * `glyphs` → `/glyphs/{fontstack}/{range}.pbf`, tokens preserved for
///    native substitution;
///  * `sprite` → `/sprite/<light|dark>`, no suffix — MapLibre appends
///    `.json`/`.png`/`@2x.*` itself.
///
/// Coverage is resolved per tile inside the proxy, not here: an uncovered
/// tile is redirected upstream, and glyphs/sprites are served local-first
/// with write-through caching.
///
/// [proxyBaseUrl] must be loopback — defense in depth against a style ever
/// pointing at a remote host. [style] is deep-copied before mutation.
Map<String, dynamic> rewriteStyleForProxy(
  Map<String, dynamic> style, {
  required String proxyBaseUrl,
  bool dark = false,
}) {
  if (!proxyBaseUrl.startsWith('http://127.0.0.1:')) {
    throw ArgumentError.value(
      proxyBaseUrl,
      'proxyBaseUrl',
      'must start with "http://127.0.0.1:" (loopback-only)',
    );
  }

  final out = jsonDecode(jsonEncode(style)) as Map<String, dynamic>;

  out['glyphs'] = '$proxyBaseUrl/glyphs/{fontstack}/{range}.pbf';
  out['sprite'] = '$proxyBaseUrl/sprite/${dark ? 'dark' : 'light'}';

  final sources = out['sources'];
  if (sources is Map<String, dynamic>) {
    for (final key in sources.keys.toList()) {
      final source = sources[key];
      if (source is! Map) continue;
      final isTiled = source.containsKey('tiles') || source.containsKey('url');
      if (!isTiled) continue;

      final sourceMap = source as Map<String, dynamic>;
      sourceMap.remove('url');
      if (sourceMap['type'] == 'raster-dem') {
        sourceMap['tiles'] = ['$proxyBaseUrl/dem/{z}/{x}/{y}.png'];
        sourceMap['encoding'] = 'terrarium';
        sourceMap['tileSize'] = 512;
        sourceMap['maxzoom'] = _proxyDemMaxZoom;
      } else {
        sourceMap['tiles'] = ['$proxyBaseUrl/vector/{z}/{x}/{y}.pbf'];
        sourceMap['maxzoom'] = _proxyVectorMaxZoom;
      }
    }
  }

  return out;
}
