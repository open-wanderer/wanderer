import 'dart:convert';

import 'package:path/path.dart' as p;
import 'package:wanderer/util/region/map_cache_path.dart';

/// Rewrites a downloaded trail's MapLibre style so it resolves entirely from
/// on-device resources — no network.
///
/// Given the online base style, the app-wide glyph/sprite cache [cacheRoot]
/// (`<app-docs>/map_cache`) and the trail's downloaded
/// `.pmtiles` [cellPaths], this pure transform:
///
///  * points `glyphs` at `file://<cacheRoot>/glyphs/{fontstack}/{range}.pbf`
///    (the literal `{fontstack}`/`{range}` tokens are preserved for native
///    runtime substitution);
///  * points `sprite` at `file://<cacheRoot>/sprite/<light|dark>`;
///  * repoints every remote (tiled) vector/raster source at a native
///    `pmtiles://file://` archive using [cellPaths];
///  * special-cases any `type: raster-dem` source (e.g. `hillshadeSource`) —
///    it is routed to its own [demCellPaths] archive instead, and gains
///    `encoding: terrarium` + `tileSize: 512` + a dedicated
///    [_offlineDemMaxZoom], all three of which the online style relies on
///    Mapterhorn's tilejson to supply and therefore does not carry itself.
///    When [demCellPaths] is empty, the raster-dem source and every layer
///    that references it are dropped entirely (hillshade is cosmetic, so a
///    trail with no downloaded DEM archive just renders without relief —
///    it must never leak the source's `https://` url into the offline
///    style).
///
/// ## Multi-cell strategy
///
/// A single trail's offline tiles are split across one `.pmtiles` archive per
/// 0.5° grid cell (`db/services/tiles/generator.go` runs `pmtiles extract` per
/// `grid.go` `GridSize = 0.5`, so a realistic trail spans ~1-4 cells). The
/// installed `pmtiles` 1.2.0 Dart package is **read-only** (`PmTilesArchive`
/// exposes only `from`/`fromFile`/`fromReadAt` — no write/merge API), and a
/// server-side merge endpoint is out of this Flutter phase's scope. So a
/// client-side or download-time merge is infeasible; instead this transform
/// emits **N native `pmtiles://file://` sources + N duplicated style-layer
/// sets**: the first cell keeps the original source key, each extra cell `i`
/// gets a `<source>-cell-<i>` source and a `<layerId>__cell<i>` clone of every
/// layer that referenced that source. Source-less layers (e.g. `background`)
/// are never cloned. This is bounded by `cellPaths.length` (realistic cell
/// counts are small; `is_large` full-polyline trails are deferred).
///
/// ## Path safety
///
/// Every emitted URL is rooted at the supplied [cacheRoot] / [cellPaths] and
/// carries only the `file://` or `pmtiles://file://` scheme. Any `cellPath` or
/// `cacheRoot` that is not absolute, contains a `..` traversal segment, or
/// carries a foreign URL scheme is rejected with an [ArgumentError] before any
/// path enters the style — a downloaded trail can never read outside the
/// app sandbox, and no `https://` URL is ever produced for the offline style.
///
/// The input [style] is deep-copied before any mutation, so the shared online
/// base JSON (from the `keepAlive` style provider) is never corrupted.
Map<String, dynamic> rewriteStyleForOffline(
  Map<String, dynamic> style, {
  required String cacheRoot,
  required List<String> cellPaths,
  List<String> demCellPaths = const [],
  bool dark = false,
}) {
  if (cellPaths.isEmpty) {
    throw ArgumentError.value(cellPaths, 'cellPaths', 'must not be empty');
  }
  _assertSafePath(cacheRoot, 'cacheRoot');
  for (final cell in cellPaths) {
    _assertSafePath(cell, 'cellPath');
  }
  for (final cell in demCellPaths) {
    _assertSafePath(cell, 'demCellPath');
  }

  // Deep copy so the shared online base style is never mutated in place.
  final out = jsonDecode(jsonEncode(style)) as Map<String, dynamic>;

  // Glyphs + sprite resolve from the app-wide file:// cache. The
  // literal {fontstack}/{range} tokens are kept for native substitution.
  out['glyphs'] =
      'file://${p.join(cacheRoot, 'glyphs', '{fontstack}', '{range}.pbf')}';
  out['sprite'] = 'file://${spriteCacheBasePath(cacheRoot, dark: dark)}';

  // Repoint every remote (tiled) source at a pmtiles://file://
  // archive, duplicating sources + layers per extra cell. raster-dem
  // sources are special-cased onto demCellPaths (see doc comment above).
  final sources = out['sources'];
  if (sources is Map<String, dynamic>) {
    _rewriteSourcesAndLayers(out, sources, cellPaths, demCellPaths);
  }

  return out;
}

/// Repoints [sources] (and clones the [style]'s layers) onto the offline
/// `pmtiles://file://` cells. Vector/raster (tiled) sources are routed to
/// [cellPaths] via [_pointSourceAtCell]; any `type: raster-dem` source (e.g.
/// `hillshadeSource`) is routed separately to [demCellPaths] via
/// [_pointDemSourceAtCell] — when [demCellPaths] is empty, raster-dem
/// sources and every layer referencing them are dropped instead (see the
/// function doc comment on [rewriteStyleForOffline]).
void _rewriteSourcesAndLayers(
  Map<String, dynamic> style,
  Map<String, dynamic> sources,
  List<String> cellPaths,
  List<String> demCellPaths,
) {
  // A "tiled" source is a remote data source carrying a `tiles` template or a
  // `url` — exactly the sources that must become local pmtiles archives.
  final allTiledKeys = sources.keys.where((key) {
    final source = sources[key];
    return source is Map &&
        (source.containsKey('tiles') || source.containsKey('url'));
  }).toList();

  // raster-dem sources (hillshade) have an independent DEM archive lifecycle
  // and must never be repointed at a vector cell (the bug this fixes).
  final demKeys = allTiledKeys.where((key) {
    final source = sources[key] as Map;
    return source['type'] == 'raster-dem';
  }).toList();
  final tiledKeys = allTiledKeys
      .where((key) => !demKeys.contains(key))
      .toList();

  final layers = style['layers'];
  final layerList = layers is List ? layers : const <dynamic>[];

  if (tiledKeys.isNotEmpty) {
    _rewriteSourceGroup(
      sources,
      layerList,
      layers,
      tiledKeys,
      cellPaths,
      _pointSourceAtCell,
    );
  }

  if (demKeys.isNotEmpty) {
    if (demCellPaths.isEmpty) {
      // Hillshade is cosmetic: with no downloaded DEM archive, drop the
      // raster-dem source(s) and every layer that referenced them, so no
      // https:// hillshade url survives into the offline style (path-safety
      // invariant) and the call never throws for a missing DEM.
      for (final key in demKeys) {
        sources.remove(key);
      }
      if (layers is List) {
        layers.removeWhere((layer) {
          if (layer is! Map) return false;
          final source = layer['source'];
          return source is String && demKeys.contains(source);
        });
      }
    } else {
      _rewriteSourceGroup(
        sources,
        layerList,
        layers,
        demKeys,
        demCellPaths,
        _pointDemSourceAtCell,
      );
    }
  }
}

/// Shared cell-duplication machinery for a group of source [keys] (either the
/// vector/raster [tiledKeys] or the raster-dem [demKeys]) against a parallel
/// list of local archive [paths]. The first path reuses each source's
/// original key; each extra path gets a `<key>-cell-<i>` source plus
/// `__cell<i>` layer clones. [pointFn] performs the source-specific field
/// rewrite ([_pointSourceAtCell] or [_pointDemSourceAtCell]).
void _rewriteSourceGroup(
  Map<String, dynamic> sources,
  List<dynamic> layerList,
  dynamic layers,
  List<String> keys,
  List<String> paths,
  void Function(Map<String, dynamic> source, String path) pointFn,
) {
  // Snapshot each original source definition before repointing cell 0, so the
  // per-cell clones inherit the original metadata (type, maxzoom, ...).
  final originals = <String, Map<String, dynamic>>{
    for (final key in keys) key: Map<String, dynamic>.from(sources[key] as Map),
  };

  // Cell 0 — repoint each original source in place.
  for (final key in keys) {
    pointFn(sources[key] as Map<String, dynamic>, paths.first);
  }

  // Extra cells — one duplicated source + one duplicated layer set each.
  final extraLayers = <Map<String, dynamic>>[];

  for (var i = 1; i < paths.length; i++) {
    for (final key in keys) {
      final clonedSource =
          jsonDecode(jsonEncode(originals[key])) as Map<String, dynamic>;
      pointFn(clonedSource, paths[i]);
      sources['$key-cell-$i'] = clonedSource;
    }

    for (final layer in layerList) {
      if (layer is! Map) continue;
      final source = layer['source'];
      if (source is String && keys.contains(source)) {
        final clone = jsonDecode(jsonEncode(layer)) as Map<String, dynamic>;
        clone['id'] = '${layer['id']}__cell$i';
        clone['source'] = '$source-cell-$i';
        extraLayers.add(clone);
      }
    }
  }

  if (extraLayers.isNotEmpty && layers is List) {
    layers.addAll(extraLayers);
  }
}

/// The deepest zoom level actually present in a locally-extracted `.pmtiles`
/// cell. Must match `maxZoom` in `db/services/tiles/generator.go` (currently
/// 14) — the server runs `pmtiles extract --maxzoom=14`, which is shallower
/// than the online style's `maxzoom: 15` (inherited from the live Protomaps
/// CDN's own, deeper tile pyramid). If the offline source keeps the online
/// `maxzoom`, MapLibre requests nonexistent z15+ tiles directly from the local
/// archive instead of overzooming the z14 tile — rendering blank above z14.
const int _offlinePmtilesMaxZoom = 14;

/// The deepest zoom level actually present in a locally-extracted DEM
/// `.pmtiles` cell. MUST equal the Go `demMaxZoom` const in
/// `db/services/tiles/generator.go` (currently 12) — kept in lockstep for the
/// same reason as [_offlinePmtilesMaxZoom]: a mismatch means MapLibre
/// requests DEM tiles that were never extracted, leaving relief blank above
/// the cap instead of overzooming. Deliberately a separate, lower constant
/// than the vector basemap's z14 — hillshading doesn't need that much detail.
const int _offlineDemMaxZoom = 12;

/// Rewrites the online base [style] into a static-loopback-XYZ offline style
/// backed by the client-local tile proxy (`tile_proxy_server.dart`)
/// instead of per-cell `pmtiles://file://` archives.
///
/// Unlike [rewriteStyleForOffline]'s N-cell duplication (see that function's
/// doc comment), this transform emits exactly ONE vector source and ONE
/// `raster-dem` source, each pointed at the proxy's fixed
/// `<proxyBaseUrl>/vector/{z}/{x}/{y}.pbf` / `.../dem/{z}/{x}/{y}.png` XYZ
/// template — the literal `{z}`/`{x}`/`{y}` tokens survive verbatim for
/// native runtime substitution. No `__cellN` source/layer cloning is
/// produced, because the proxy resolves per-tile coverage server-side
/// ([resolveRegionForTile]) rather than the style needing to
/// enumerate every downloaded region's archive up front.
///
/// `glyphs`/`sprite` are rewritten to `file://<cacheRoot>/...` identically to
/// [rewriteStyleForOffline] (light vs dark honored via [dark]), and
/// [cacheRoot] is validated via the same [_assertSafePath] used there.
/// [proxyBaseUrl] must start with `http://127.0.0.1:` — any other base is
/// rejected (defense in depth: this style must never point at a non-loopback
/// host).
///
/// The input [style] is deep-copied before any mutation, matching
/// [rewriteStyleForOffline]'s "never mutate the shared base style" invariant.
///
/// The legacy `rewriteStyleForOffline(cellPaths:...)` path is left intact
/// pending a separate cleanup.
Map<String, dynamic> rewriteStyleForProxy(
  Map<String, dynamic> style, {
  required String cacheRoot,
  required String proxyBaseUrl,
  bool dark = false,
}) {
  _assertSafePath(cacheRoot, 'cacheRoot');
  if (!proxyBaseUrl.startsWith('http://127.0.0.1:')) {
    throw ArgumentError.value(
      proxyBaseUrl,
      'proxyBaseUrl',
      'must start with "http://127.0.0.1:" (loopback-only)',
    );
  }

  // Deep copy so the shared online base style is never mutated in place.
  final out = jsonDecode(jsonEncode(style)) as Map<String, dynamic>;

  out['glyphs'] =
      'file://${p.join(cacheRoot, 'glyphs', '{fontstack}', '{range}.pbf')}';
  out['sprite'] = 'file://${spriteCacheBasePath(cacheRoot, dark: dark)}';

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
        sourceMap['maxzoom'] = _offlineDemMaxZoom;
      } else {
        sourceMap['tiles'] = ['$proxyBaseUrl/vector/{z}/{x}/{y}.pbf'];
        sourceMap['maxzoom'] = _offlinePmtilesMaxZoom;
      }
    }
  }

  return out;
}

/// Repoints a single source [source] at the pmtiles archive at [cellPath]:
/// drops any remote `tiles` template, sets `url` to `pmtiles://file://…`, and
/// clamps `maxzoom` to the archive's actual depth (see
/// [_offlinePmtilesMaxZoom]) so MapLibre overzooms past it instead of
/// requesting tiles that were never extracted.
void _pointSourceAtCell(Map<String, dynamic> source, String cellPath) {
  source.remove('tiles');
  source['url'] = 'pmtiles://file://$cellPath';
  source['maxzoom'] = _offlinePmtilesMaxZoom;
}

/// Repoints a `raster-dem` source [source] at the DEM pmtiles archive at
/// [demPath]: drops any remote `tiles` template, sets `url` to
/// `pmtiles://file://…`, and injects `encoding: terrarium` + `tileSize: 512`
/// + `maxzoom: `[_offlineDemMaxZoom]. These three fields are supplied online
/// by Mapterhorn's tilejson and are therefore absent from the style JSON's
/// `hillshadeSource` definition (which carries only `type` + `url`) — a
/// `pmtiles://` source has no tilejson, so they must be written explicitly or
/// MapLibre falls back to its `mapbox` encoding / 256px tile size default,
/// producing garbage or misaligned relief.
void _pointDemSourceAtCell(Map<String, dynamic> source, String demPath) {
  source.remove('tiles');
  source['url'] = 'pmtiles://file://$demPath';
  source['encoding'] = 'terrarium';
  source['tileSize'] = 512;
  source['maxzoom'] = _offlineDemMaxZoom;
}

/// Rejects any [path] that is not an absolute, traversal-free local path.
///
/// Guards against a foreign URL scheme, a relative path, or a `..`
/// segment that would let a downloaded trail escape the app sandbox or inject
/// an `https://` URL into the offline style. None is ever emitted.
void _assertSafePath(String path, String label) {
  if (path.contains('://')) {
    throw ArgumentError.value(path, label, 'must not carry a URL scheme');
  }
  if (!p.isAbsolute(path)) {
    throw ArgumentError.value(path, label, 'must be an absolute path');
  }
  if (p.split(path).contains('..')) {
    throw ArgumentError.value(path, label, 'must not contain a ".." segment');
  }
}
