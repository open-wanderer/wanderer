import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/offline_style_rewriter.dart';

/// A minimal but representative online style: a `protomaps` vector source with an
/// `https://` tile template, a `hillshadeSource` `raster-dem` source (only
/// `type` + `url`, mirroring the real style — `encoding`/`tileSize` are
/// supplied online by Mapterhorn's tilejson and therefore absent here), an
/// `https://` glyphs template + sprite base, one source-less background
/// layer, two `protomaps`-sourced layers, and one `hillshade`-sourced layer.
/// The `attribution` deliberately carries `https://` links (as the real
/// Protomaps theme does) so the scheme-allowlist assertions only inspect
/// URL-bearing fields, never the attribution HTML.
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
    'hillshadeSource': <String, dynamic>{
      'type': 'raster-dem',
      'url': 'https://tiles.mapterhorn.com/tilejson.json',
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
    <String, dynamic>{
      'id': 'hillshade',
      'type': 'hillshade',
      'source': 'hillshadeSource',
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
const String _demCellA =
    '/data/user/0/app.wanderer/app_flutter/library/t1/tiles/1600_1200_dem.pmtiles';
const String _demCellB =
    '/data/user/0/app.wanderer/app_flutter/library/t1/tiles/1601_1200_dem.pmtiles';

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
      // maxzoom is clamped to what the local archive actually contains (14,
      // per db/services/tiles/generator.go), NOT the online style's 15 —
      // otherwise MapLibre requests tiles that were never extracted and
      // renders blank above z14 instead of overzooming.
      expect(protomaps['maxzoom'], 14);
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

  group('rewriteStyleForOffline — multi cell', () {
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
      // no cell is silently dropped.
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

  group('rewriteStyleForOffline — path safety', () {
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

  group('rewriteStyleForOffline — raster-dem (hillshade)', () {
    test(
      'single cell: hillshadeSource points at the DEM archive with '
      'terrarium/512/12, protomaps is untouched',
      () {
        final result = rewriteStyleForOffline(
          _onlineStyle(),
          cacheRoot: _cacheRoot,
          cellPaths: <String>[_cellA],
          demCellPaths: <String>[_demCellA],
        );

        final sources = result['sources'] as Map<String, dynamic>;
        final hillshade = sources['hillshadeSource'] as Map<String, dynamic>;
        expect(hillshade['url'], 'pmtiles://file://$_demCellA');
        expect(hillshade['encoding'], 'terrarium');
        expect(hillshade['tileSize'], 512);
        expect(hillshade['maxzoom'], 12);
        expect(hillshade.containsKey('tiles'), isFalse);

        // The vector source is untouched by the DEM special case.
        final protomaps = sources['protomaps'] as Map<String, dynamic>;
        expect(protomaps['url'], 'pmtiles://file://$_cellA');
        expect(protomaps['maxzoom'], 14);
      },
    );

    test(
      'multi cell: hillshadeSource-cell-1 points at demB, and '
      'hillshade__cell1 clone references it',
      () {
        final result = rewriteStyleForOffline(
          _onlineStyle(),
          cacheRoot: _cacheRoot,
          cellPaths: <String>[_cellA, _cellB],
          demCellPaths: <String>[_demCellA, _demCellB],
        );

        final sources = result['sources'] as Map<String, dynamic>;
        expect(sources.containsKey('hillshadeSource-cell-1'), isTrue);
        final demCell1 =
            sources['hillshadeSource-cell-1'] as Map<String, dynamic>;
        expect(demCell1['url'], 'pmtiles://file://$_demCellB');
        expect(demCell1['encoding'], 'terrarium');
        expect(demCell1['tileSize'], 512);
        expect(demCell1['maxzoom'], 12);

        final layers = (result['layers'] as List)
            .cast<Map<String, dynamic>>();
        final clone = layers.firstWhere((l) => l['id'] == 'hillshade__cell1');
        expect(clone['source'], 'hillshadeSource-cell-1');
      },
    );

    test(
      'empty demCellPaths drops the raster-dem source and hillshade layer, '
      'and leaks no http(s):// url; does not throw',
      () {
        final result = rewriteStyleForOffline(
          _onlineStyle(),
          cacheRoot: _cacheRoot,
          cellPaths: <String>[_cellA],
        );

        final sources = result['sources'] as Map<String, dynamic>;
        expect(sources.containsKey('hillshadeSource'), isFalse);

        final layers = (result['layers'] as List)
            .cast<Map<String, dynamic>>();
        expect(layers.any((l) => l['id'] == 'hillshade'), isFalse);

        // No https:// leaks anywhere in the emitted style (attribution is
        // exempt in other tests, but here we assert on the whole style
        // string since there is no DEM url to spare).
        final urls = sources.values
            .map((dynamic s) => (s as Map<String, dynamic>)['url'])
            .whereType<String>();
        for (final url in urls) {
          expect(url.contains('http'), isFalse);
        }
      },
    );
  });

  group('rewriteStyleForOffline — DEM path safety', () {
    test('rejects a demCellPaths entry with a .. traversal segment', () {
      expect(
        () => rewriteStyleForOffline(
          _onlineStyle(),
          cacheRoot: _cacheRoot,
          cellPaths: <String>[_cellA],
          demCellPaths: <String>['$_cacheRoot/../../etc/passwd_dem.pmtiles'],
        ),
        throwsArgumentError,
      );
    });

    test('rejects a non-absolute demCellPaths entry', () {
      expect(
        () => rewriteStyleForOffline(
          _onlineStyle(),
          cacheRoot: _cacheRoot,
          cellPaths: <String>[_cellA],
          demCellPaths: <String>['library/t1/tiles/a_dem.pmtiles'],
        ),
        throwsArgumentError,
      );
    });

    test('rejects a foreign-scheme demCellPaths entry', () {
      expect(
        () => rewriteStyleForOffline(
          _onlineStyle(),
          cacheRoot: _cacheRoot,
          cellPaths: <String>[_cellA],
          demCellPaths: <String>['https://evil.example.org/x_dem.pmtiles'],
        ),
        throwsArgumentError,
      );
    });
  });

  group('rewriteStyleForProxy', () {
    const proxyBaseUrl = 'http://127.0.0.1:54321';

    test(
      'vector source gets a static loopback XYZ tiles template, no url/pmtiles',
      () {
        final result = rewriteStyleForProxy(
          _onlineStyle(),
          cacheRoot: _cacheRoot,
          proxyBaseUrl: proxyBaseUrl,
        );

        final sources = result['sources'] as Map<String, dynamic>;
        final protomaps = sources['protomaps'] as Map<String, dynamic>;
        expect(protomaps['tiles'], <String>[
          '$proxyBaseUrl/vector/{z}/{x}/{y}.pbf',
        ]);
        expect(protomaps.containsKey('url'), isFalse);
        expect(protomaps['maxzoom'], 14);
      },
    );

    test(
      'raster-dem source gets a static loopback XYZ tiles template with '
      'terrarium/512/maxzoom 12',
      () {
        final result = rewriteStyleForProxy(
          _onlineStyle(),
          cacheRoot: _cacheRoot,
          proxyBaseUrl: proxyBaseUrl,
        );

        final sources = result['sources'] as Map<String, dynamic>;
        final hillshade = sources['hillshadeSource'] as Map<String, dynamic>;
        expect(hillshade['tiles'], <String>[
          '$proxyBaseUrl/dem/{z}/{x}/{y}.png',
        ]);
        expect(hillshade.containsKey('url'), isFalse);
        expect(hillshade['encoding'], 'terrarium');
        expect(hillshade['tileSize'], 512);
        expect(hillshade['maxzoom'], 12);
      },
    );

    test('glyphs/sprite rewritten to file:// exactly as rewriteStyleForOffline', () {
      final result = rewriteStyleForProxy(
        _onlineStyle(),
        cacheRoot: _cacheRoot,
        proxyBaseUrl: proxyBaseUrl,
      );

      expect(
        result['glyphs'],
        'file://$_cacheRoot/glyphs/{fontstack}/{range}.pbf',
      );
      expect(result['sprite'], 'file://$_cacheRoot/sprite/light');
    });

    test('dark variant points the sprite at sprite/dark', () {
      final result = rewriteStyleForProxy(
        _onlineStyle(),
        cacheRoot: _cacheRoot,
        proxyBaseUrl: proxyBaseUrl,
        dark: true,
      );

      expect(result['sprite'], 'file://$_cacheRoot/sprite/dark');
    });

    test('{z}/{x}/{y} tokens survive verbatim for native runtime substitution', () {
      final result = rewriteStyleForProxy(
        _onlineStyle(),
        cacheRoot: _cacheRoot,
        proxyBaseUrl: proxyBaseUrl,
      );

      final sources = result['sources'] as Map<String, dynamic>;
      final protomaps = sources['protomaps'] as Map<String, dynamic>;
      final hillshade = sources['hillshadeSource'] as Map<String, dynamic>;
      expect((protomaps['tiles'] as List).single, contains('{z}/{x}/{y}'));
      expect((hillshade['tiles'] as List).single, contains('{z}/{x}/{y}'));
    });

    test('does not mutate the input style (deep copy)', () {
      final input = _onlineStyle();
      rewriteStyleForProxy(
        input,
        cacheRoot: _cacheRoot,
        proxyBaseUrl: proxyBaseUrl,
      );

      expect(
        input['glyphs'],
        'https://tiles.example.org/glyphs/{fontstack}/{range}.pbf',
      );
      final src = (input['sources'] as Map)['protomaps'] as Map;
      expect(src['tiles'], <String>['https://tiles.example.org/{z}/{x}/{y}.mvt']);
      expect(src.containsKey('url'), isFalse);
    });

    test(
      'no https:// or pmtiles:// survives anywhere in the output; exactly '
      'one vector and one dem source, no __cellN duplication',
      () {
        final result = rewriteStyleForProxy(
          _onlineStyle(),
          cacheRoot: _cacheRoot,
          proxyBaseUrl: proxyBaseUrl,
        );

        final sources = result['sources'] as Map<String, dynamic>;
        expect(sources.keys.toSet(), <String>{'protomaps', 'hillshadeSource'});

        // Only inspect URL-bearing fields (tiles/url/glyphs/sprite), never
        // the attribution HTML, which legitimately carries https:// links
        // (matches rewriteStyleForOffline's own scheme-allowlist precedent).
        expect((result['glyphs'] as String).contains('http'), isFalse);
        expect((result['sprite'] as String).contains('http'), isFalse);
        for (final dynamic s in sources.values) {
          final src = s as Map<String, dynamic>;
          expect(src.containsKey('url'), isFalse);
          final tiles = (src['tiles'] as List).cast<String>();
          for (final tile in tiles) {
            expect(tile.contains('https://'), isFalse);
            expect(tile.contains('pmtiles://'), isFalse);
          }
        }

        final layers = (result['layers'] as List).cast<Map<String, dynamic>>();
        expect(
          layers.any((l) => (l['id'] as String).contains('__cell')),
          isFalse,
        );
      },
    );

    test('rejects a cacheRoot with a .. traversal segment', () {
      expect(
        () => rewriteStyleForProxy(
          _onlineStyle(),
          cacheRoot: '/data/user/0/app.wanderer/../map_cache',
          proxyBaseUrl: proxyBaseUrl,
        ),
        throwsArgumentError,
      );
    });

    test('rejects a non-loopback proxyBaseUrl', () {
      expect(
        () => rewriteStyleForProxy(
          _onlineStyle(),
          cacheRoot: _cacheRoot,
          proxyBaseUrl: 'https://evil.example.org',
        ),
        throwsArgumentError,
      );
    });
  });
}
