import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/region/proxy_style_rewriter.dart';

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

void main() {
  group('rewriteStyleForProxy', () {
    const proxyBaseUrl = 'http://127.0.0.1:54321';

    test(
      'vector source gets a static loopback XYZ tiles template, no url/pmtiles',
      () {
        final result = rewriteStyleForProxy(
          _onlineStyle(),
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

    test('glyphs/sprite rewritten to loopback proxy URLs', () {
      final result = rewriteStyleForProxy(
        _onlineStyle(),
        proxyBaseUrl: proxyBaseUrl,
      );

      expect(
        result['glyphs'],
        '$proxyBaseUrl/glyphs/{fontstack}/{range}.pbf',
      );
      expect(result['sprite'], '$proxyBaseUrl/sprite/light');
    });

    test('dark variant points the sprite at sprite/dark', () {
      final result = rewriteStyleForProxy(
        _onlineStyle(),
        proxyBaseUrl: proxyBaseUrl,
        dark: true,
      );

      expect(result['sprite'], '$proxyBaseUrl/sprite/dark');
    });

    test(
      '{z}/{x}/{y} and {fontstack}/{range} tokens survive verbatim for '
      'native runtime substitution',
      () {
        final result = rewriteStyleForProxy(
          _onlineStyle(),
          proxyBaseUrl: proxyBaseUrl,
        );

        final sources = result['sources'] as Map<String, dynamic>;
        final protomaps = sources['protomaps'] as Map<String, dynamic>;
        final hillshade = sources['hillshadeSource'] as Map<String, dynamic>;
        expect((protomaps['tiles'] as List).single, contains('{z}/{x}/{y}'));
        expect((hillshade['tiles'] as List).single, contains('{z}/{x}/{y}'));
        expect(result['glyphs'], contains('{fontstack}'));
        expect(result['glyphs'], contains('{range}'));
      },
    );

    test('does not mutate the input style (deep copy)', () {
      final input = _onlineStyle();
      rewriteStyleForProxy(input, proxyBaseUrl: proxyBaseUrl);

      expect(
        input['glyphs'],
        'https://tiles.example.org/glyphs/{fontstack}/{range}.pbf',
      );
      final src = (input['sources'] as Map)['protomaps'] as Map;
      expect(src['tiles'], <String>['https://tiles.example.org/{z}/{x}/{y}.mvt']);
      expect(src.containsKey('url'), isFalse);
    });

    test(
      'every URL-bearing field starts with http://127.0.0.1: — no https://, '
      'no file://, no pmtiles:// anywhere in the output; exactly one vector '
      'and one dem source, no __cellN duplication',
      () {
        final result = rewriteStyleForProxy(
          _onlineStyle(),
          proxyBaseUrl: proxyBaseUrl,
        );

        final sources = result['sources'] as Map<String, dynamic>;
        expect(sources.keys.toSet(), <String>{'protomaps', 'hillshadeSource'});

        // Only inspect URL-bearing fields (tiles/url/glyphs/sprite), never
        // the attribution HTML, which legitimately carries https:// links.
        expect(result['glyphs'], startsWith('http://127.0.0.1:'));
        expect(result['sprite'], startsWith('http://127.0.0.1:'));
        for (final dynamic s in sources.values) {
          final src = s as Map<String, dynamic>;
          expect(src.containsKey('url'), isFalse);
          final tiles = (src['tiles'] as List).cast<String>();
          for (final tile in tiles) {
            expect(tile, startsWith('http://127.0.0.1:'));
          }
        }

        final layers = (result['layers'] as List).cast<Map<String, dynamic>>();
        expect(
          layers.any((l) => (l['id'] as String).contains('__cell')),
          isFalse,
        );
      },
    );

    test('rejects a non-loopback proxyBaseUrl', () {
      expect(
        () => rewriteStyleForProxy(
          _onlineStyle(),
          proxyBaseUrl: 'https://evil.example.org',
        ),
        throwsArgumentError,
      );
    });
  });
}
