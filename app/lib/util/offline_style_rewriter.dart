import 'dart:convert';

import 'package:path/path.dart' as p;
import 'package:wanderer/util/map_cache_path.dart';

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
///  * repoints every remote (tiled) source at a native `pmtiles://file://`
///    archive.
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
  bool dark = false,
}) {
  if (cellPaths.isEmpty) {
    throw ArgumentError.value(cellPaths, 'cellPaths', 'must not be empty');
  }
  _assertSafePath(cacheRoot, 'cacheRoot');
  for (final cell in cellPaths) {
    _assertSafePath(cell, 'cellPath');
  }

  // Deep copy so the shared online base style is never mutated in place.
  final out = jsonDecode(jsonEncode(style)) as Map<String, dynamic>;

  // Glyphs + sprite resolve from the app-wide file:// cache. The
  // literal {fontstack}/{range} tokens are kept for native substitution.
  out['glyphs'] =
      'file://${p.join(cacheRoot, 'glyphs', '{fontstack}', '{range}.pbf')}';
  out['sprite'] = 'file://${spriteCacheBasePath(cacheRoot, dark: dark)}';

  // Repoint every remote (tiled) source at a pmtiles://file://
  // archive, duplicating sources + layers per extra cell.
  final sources = out['sources'];
  if (sources is Map<String, dynamic>) {
    _rewriteSourcesAndLayers(out, sources, cellPaths);
  }

  return out;
}

/// Repoints [sources] (and clones the [style]'s layers) onto the offline
/// `pmtiles://file://` cells. The first cell reuses each source's original key;
/// each extra cell gets a `<key>-cell-<i>` source plus `__cell<i>` layer clones.
void _rewriteSourcesAndLayers(
  Map<String, dynamic> style,
  Map<String, dynamic> sources,
  List<String> cellPaths,
) {
  // A "tiled" source is a remote data source carrying a `tiles` template or a
  // `url` — exactly the sources that must become local pmtiles archives.
  final tiledKeys = sources.keys.where((key) {
    final source = sources[key];
    return source is Map &&
        (source.containsKey('tiles') || source.containsKey('url'));
  }).toList();

  if (tiledKeys.isEmpty) return;

  // Snapshot each original source definition before repointing cell 0, so the
  // per-cell clones inherit the original metadata (type, maxzoom, ...).
  final originals = <String, Map<String, dynamic>>{
    for (final key in tiledKeys)
      key: Map<String, dynamic>.from(sources[key] as Map),
  };

  // Cell 0 — repoint each original source in place.
  for (final key in tiledKeys) {
    _pointSourceAtCell(sources[key] as Map<String, dynamic>, cellPaths.first);
  }

  // Extra cells — one duplicated source + one duplicated layer set each.
  final extraLayers = <Map<String, dynamic>>[];
  final layers = style['layers'];
  final layerList = layers is List ? layers : const <dynamic>[];

  for (var i = 1; i < cellPaths.length; i++) {
    for (final key in tiledKeys) {
      final clonedSource = jsonDecode(jsonEncode(originals[key]))
          as Map<String, dynamic>;
      _pointSourceAtCell(clonedSource, cellPaths[i]);
      sources['$key-cell-$i'] = clonedSource;
    }

    for (final layer in layerList) {
      if (layer is! Map) continue;
      final source = layer['source'];
      if (source is String && tiledKeys.contains(source)) {
        final clone =
            jsonDecode(jsonEncode(layer)) as Map<String, dynamic>;
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
