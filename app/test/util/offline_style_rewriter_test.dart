import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/offline_style_rewriter.dart';

/// A minimal but representative online style: a `protomaps` vector source with an
/// `https://` tile template, an `https://` glyphs template + sprite base, one
/// source-less background layer, and two `protomaps`-sourced layers. The
/// `attribution` deliberately carries `https://` links (as the real Protomaps
/// theme does) so the scheme-allowlist assertions only inspect URL-bearing
/// fields, never the attribution HTML.
Map<String, dynamic> _onlineStyle() => <String, dynamic>{
  'version': 8,
  'glyphs': 'https://tiles.example.org/glyphs/{fontstack}/{range}.pbf',
  'sprite': 'https://tiles.example.org/sprite',
  'sources': <String, dynamic>{
    'protomaps': <String, dynamic>{
      'type': 'vector',
      'tiles': <String>['https://tiles.example.org/{z}/{x}/{y}.mvt'],
      'maxzoom': 15,
      'attribution':
          '<a href="https://github.com/protomaps/basemaps">Protomaps</a> '
          '© <a href="https://openstreetmap.org">OpenStreetMap</a>',
    },
  },
  'layers': <dynamic>[
    <String, dynamic>{'id': 'background', 'type': 'background'},
    <String, dynamic>{
      'id': 'earth',
      'type': 'fill',
      'source': 'protomaps',
      'source-layer': 'earth',
    },
    <String, dynamic>{
      'id': 'roads_labels',
      'type': 'symbol',
      'source': 'protomaps',
      'source-layer': 'roads',
      'layout': <String, dynamic>{
        'text-font': <String>['Noto Sans Regular'],
        'text-field': <String>['get', 'name'],
      },
    },
  ],
};

const String _cacheRoot = '/data/user/0/app.wanderer/app_flutter/map_cache';
const String _cellA =
    '/data/user/0/app.wanderer/app_flutter/library/t1/tiles/1600_1200.pmtiles';
const String _cellB =
    '/data/user/0/app.wanderer/app_flutter/library/t1/tiles/1601_1200.pmtiles';
const String _cellC =
    '/data/user/0/app.wanderer/app_flutter/library/t1/tiles/1600_1201.pmtiles';

void main() {
  group('rewriteStyleForOffline — single cell', () {
    test('rewrites glyphs, sprite, and the protomaps source to file://', () {
      final result = rewriteStyleForOffline(
        _onlineStyle(),
        cacheRoot: _cacheRoot,
        cellPaths: <String>[_cellA],
      );

      expect(
        result['glyphs'],
        'file://$_cacheRoot/glyphs/{fontstack}/{range}.pbf',
      );
      expect(result['sprite'], 'file://$_cacheRoot/sprite/light');

      final sources = result['sources'] as Map<String, dynamic>;
      expect(sources.keys, <String>['protomaps']);
      final protomaps = sources['protomaps'] as Map<String, dynamic>;
      expect(protomaps['url'], 'pmtiles://file://$_cellA');
      // The online tile template is dropped in favour of the pmtiles url.
      expect(protomaps.containsKey('tiles'), isFalse);
      // Non-URL metadata is preserved.
      expect(protomaps['type'], 'vector');
    });

    test('dark variant points the sprite at sprite/dark', () {
      final result = rewriteStyleForOffline(
        _onlineStyle(),
        cacheRoot: _cacheRoot,
        cellPaths: <String>[_cellA],
        dark: true,
      );
      expect(result['sprite'], 'file://$_cacheRoot/sprite/dark');
    });

    test('does not mutate the input style (deep copy)', () {
      final input = _onlineStyle();
      rewriteStyleForOffline(
        input,
        cacheRoot: _cacheRoot,
        cellPaths: <String>[_cellA],
      );
      // The shared online base JSON must be untouched.
      expect(
        input['glyphs'],
        'https://tiles.example.org/glyphs/{fontstack}/{range}.pbf',
      );
      final src = (input['sources'] as Map)['protomaps'] as Map;
      expect(src['tiles'], isNotNull);
      expect(src.containsKey('url'), isFalse);
    });
  });

  group('rewriteStyleForOffline — multi cell (OFFL-05)', () {
    test('emits one pmtiles source per cell and drops no cell', () {
      final result = rewriteStyleForOffline(
        _onlineStyle(),
        cacheRoot: _cacheRoot,
        cellPaths: <String>[_cellA, _cellB, _cellC],
      );

      final sources = result['sources'] as Map<String, dynamic>;
      expect(
        sources.keys.toSet(),
        <String>{'protomaps', 'protomaps-cell-1', 'protomaps-cell-2'},
      );

      // Every supplied cell path is referenced by exactly one source url —
      // no cell is silently dropped (OFFL-05).
      final urls = sources.values
          .map((dynamic s) => (s as Map<String, dynamic>)['url'] as String)
          .toSet();
      for (final cell in <String>[_cellA, _cellB, _cellC]) {
        expect(urls.contains('pmtiles://file://$cell'), isTrue,
            reason: 'cell $cell must be referenced by a source');
      }
    });

    test('clones every protomaps layer per extra cell, id-suffixed', () {
      final result = rewriteStyleForOffline(
        _onlineStyle(),
        cacheRoot: _cacheRoot,
        cellPaths: <String>[_cellA, _cellB],
      );

      final layers = (result['layers'] as List).cast<Map<String, dynamic>>();
      final ids = layers.map((l) => l['id'] as String).toList();

      // Original layers survive (background + the two protomaps layers).
      expect(ids, containsAll(<String>['background', 'earth', 'roads_labels']));
      // Cell 1 clones exist, suffixed and repointed at the cell-1 source.
      expect(ids, containsAll(<String>['earth__cell1', 'roads_labels__cell1']));

      final clone =
          layers.firstWhere((l) => l['id'] == 'earth__cell1');
      expect(clone['source'], 'protomaps-cell-1');
      // The background (source-less) layer is NOT cloned.
      expect(ids.where((id) => id.startsWith('background')).length, 1);
    });
  });

  group('rewriteStyleForOffline — path safety (T-15-06-01/02)', () {
    test('rejects a cell path with a .. traversal segment', () {
      expect(
        () => rewriteStyleForOffline(
          _onlineStyle(),
          cacheRoot: _cacheRoot,
          cellPaths: <String>['$_cacheRoot/../../etc/passwd.pmtiles'],
        ),
        throwsArgumentError,
      );
    });

    test('rejects a cacheRoot with a .. traversal segment', () {
      expect(
        () => rewriteStyleForOffline(
          _onlineStyle(),
          cacheRoot: '/data/user/0/app.wanderer/../map_cache',
          cellPaths: <String>[_cellA],
        ),
        throwsArgumentError,
      );
    });

    test('rejects a non-absolute cell path', () {
      expect(
        () => rewriteStyleForOffline(
          _onlineStyle(),
          cacheRoot: _cacheRoot,
          cellPaths: <String>['library/t1/tiles/a.pmtiles'],
        ),
        throwsArgumentError,
      );
    });

    test('rejects an empty cellPaths list', () {
      expect(
        () => rewriteStyleForOffline(
          _onlineStyle(),
          cacheRoot: _cacheRoot,
          cellPaths: const <String>[],
        ),
        throwsArgumentError,
      );
    });
  });

  group('rewriteStyleForOffline — scheme allowlist', () {
    test('rejects a foreign-scheme cell path (no https:// tile injected)', () {
      expect(
        () => rewriteStyleForOffline(
          _onlineStyle(),
          cacheRoot: _cacheRoot,
          cellPaths: <String>['https://evil.example.org/x.pmtiles'],
        ),
        throwsArgumentError,
      );
    });

    test('emits only file:// and pmtiles://file:// URL fields for offline', () {
      final result = rewriteStyleForOffline(
        _onlineStyle(),
        cacheRoot: _cacheRoot,
        cellPaths: <String>[_cellA, _cellB],
      );

      // Glyph + sprite URL fields are file://.
      expect((result['glyphs'] as String).startsWith('file://'), isTrue);
      expect((result['sprite'] as String).startsWith('file://'), isTrue);
      expect((result['glyphs'] as String).contains('http'), isFalse);
      expect((result['sprite'] as String).contains('http'), isFalse);

      // Every source url is pmtiles://file:// and never http(s).
      final sources = result['sources'] as Map<String, dynamic>;
      for (final dynamic s in sources.values) {
        final src = s as Map<String, dynamic>;
        final url = src['url'] as String;
        expect(url.startsWith('pmtiles://file://'), isTrue);
        expect(url.contains('http'), isFalse);
      }
    });
  });
}
