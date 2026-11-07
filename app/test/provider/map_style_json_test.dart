import 'dart:convert';

import 'package:flutter/services.dart' show rootBundle;
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/region/proxy_style_rewriter.dart';

/// Collects the URL-bearing string fields of a composed style — every source's
/// `tiles` entries + `url`, plus top-level `glyphs`/`sprite`. Deliberately
/// excludes `attribution` HTML, which legitimately carries `https://` links
/// (mirrors proxy_style_rewriter_test.dart's scheme-allowlist precedent).
List<String> _urlFields(Map<String, dynamic> style) {
  final urls = <String>[];
  final glyphs = style['glyphs'];
  final sprite = style['sprite'];
  if (glyphs is String) urls.add(glyphs);
  if (sprite is String) urls.add(sprite);

  final sources = style['sources'];
  if (sources is Map) {
    for (final source in sources.values) {
      if (source is! Map) continue;
      final url = source['url'];
      if (url is String) urls.add(url);
      final tiles = source['tiles'];
      if (tiles is List) {
        for (final t in tiles) {
          if (t is String) urls.add(t);
        }
      }
    }
  }
  return urls;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const proxyBaseUrl = 'http://127.0.0.1:54321';
  const assets = <String>[
    'assets/map/wanderer_light.json',
    'assets/map/wanderer_dark.json',
  ];

  group('shipped style assets composed through rewriteStyleForProxy', () {
    for (final asset in assets) {
      test('$asset — emits only loopback URL fields', () async {
        final raw = await rootBundle.loadString(asset);
        final decoded = jsonDecode(raw) as Map<String, dynamic>;

        final result = rewriteStyleForProxy(
          decoded,
          proxyBaseUrl: proxyBaseUrl,
        );

        // Vector tiles resolve through the loopback proxy.
        final protomaps = (result['sources'] as Map)['protomaps'] as Map;
        expect(protomaps['tiles'], <String>[
          '$proxyBaseUrl/vector/{z}/{x}/{y}.pbf',
        ]);

        // Glyphs + sprite resolve through the loopback proxy too.
        expect(result['glyphs'], startsWith('http://127.0.0.1:'));
        expect(result['sprite'], startsWith('http://127.0.0.1:'));

        // No shipped-asset sentinel token or live https:// endpoint survives
        // in any URL field — rewriteStyleForProxy overwrites every one.
        for (final url in _urlFields(result)) {
          expect(
            url.contains('__TILE_URL__') ||
                url.contains('__GLYPH_URL__') ||
                url.contains('__SPRITE_URL__'),
            isFalse,
            reason: 'unsubstituted sentinel leaked into $url',
          );
          expect(
            url.contains('https://'),
            isFalse,
            reason: 'live endpoint leaked into $url',
          );
        }
      });
    }
  });
}
