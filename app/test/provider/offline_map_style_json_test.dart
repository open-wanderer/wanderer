import 'dart:convert';

import 'package:flutter/services.dart' show rootBundle;
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/map_style_json_provider.dart';
import 'package:wanderer/util/region/offline_style_rewriter.dart';

/// Collects the URL-bearing string fields of a composed style — every source's
/// `tiles` entries + `url`, plus top-level `glyphs`/`sprite`. Deliberately
/// excludes `attribution` HTML, which legitimately carries `https://` links
/// (mirrors offline_style_rewriter_test.dart's scheme-allowlist precedent).
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

  group('fillOfflineStyleSentinels', () {
    for (final asset in assets) {
      test('$asset — replaces every operator sentinel, no network value', () async {
        final raw = await rootBundle.loadString(asset);
        // Guard: the asset really does carry the sentinels this logic targets.
        expect(raw, contains('__TILE_URL__'));
        expect(raw, contains('__GLYPH_URL__'));
        expect(raw, contains('__SPRITE_URL__'));

        final filled = fillOfflineStyleSentinels(raw);

        expect(filled.contains('__TILE_URL__'), isFalse);
        expect(filled.contains('__GLYPH_URL__'), isFalse);
        expect(filled.contains('__SPRITE_URL__'), isFalse);
        // Parses cleanly — placeholders keep the JSON syntactically valid.
        expect(() => jsonDecode(filled), returnsNormally);
      });
    }
  });

  group('offline base style composed through rewriteStyleForProxy', () {
    for (final asset in assets) {
      test('$asset — no placeholder or https:// leaks into URL fields', () async {
        final raw = await rootBundle.loadString(asset);
        final filled = fillOfflineStyleSentinels(raw);
        final decoded = jsonDecode(filled) as Map<String, dynamic>;

        final result = rewriteStyleForProxy(
          decoded,
          cacheRoot: '/data/user/0/app.wanderer/app_flutter/map_cache',
          proxyBaseUrl: proxyBaseUrl,
        );

        // Vector tiles resolve through the loopback proxy.
        final protomaps = (result['sources'] as Map)['protomaps'] as Map;
        expect(protomaps['tiles'], <String>[
          '$proxyBaseUrl/vector/{z}/{x}/{y}.pbf',
        ]);

        // Glyphs + sprite resolve from the local file:// cache.
        expect(result['glyphs'], startsWith('file://'));
        expect(result['sprite'], startsWith('file://'));

        // The inert placeholder is fully overwritten — it never reaches
        // MapLibre — and no live https:// endpoint survives in any URL field.
        for (final url in _urlFields(result)) {
          expect(
            url.contains(offlineSentinelPlaceholder),
            isFalse,
            reason: 'placeholder leaked into $url',
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
