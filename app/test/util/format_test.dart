import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/util/format.dart';

// ---------------------------------------------------------------------------
// Tests for formatSpeed and formatElapsed (stats formatters).
//
// Mirrors the import/group/test structure of
// test/provider/navigation_provider_test.dart.
// ---------------------------------------------------------------------------

void main() {
  group('formatDistance', () {
    test('metric (default) formats meters >= 1000 as km', () {
      expect(formatDistance(1000), '1.00 km');
    });

    test('imperial converts meters to miles (×0.000621371)', () {
      // 1000 * 0.000621371 = 0.621371 → "0.62 mi"
      expect(formatDistance(1000, unit: 'imperial'), '0.62 mi');
    });
  });

  group('formatElevation', () {
    test('metric (default) formats meters with m suffix', () {
      expect(formatElevation(100), '100 m');
    });

    test('imperial converts meters to feet (×3.28084)', () {
      // 100 * 3.28084 = 328.084 → "328 ft"
      expect(formatElevation(100, unit: 'imperial'), '328 ft');
    });
  });

  group('formatSpeed', () {
    test('null returns "-"', () {
      expect(formatSpeed(null), '-');
    });

    test('NaN returns "-"', () {
      expect(formatSpeed(double.nan), '-');
    });

    test('negative returns "-"', () {
      expect(formatSpeed(-1.0), '-');
    });

    test('metric (default) formats to one decimal with km/h suffix', () {
      expect(formatSpeed(12.34), '12.3 km/h');
    });

    test('zero metric is "0.0 km/h"', () {
      expect(formatSpeed(0.0), '0.0 km/h');
    });

    test('imperial converts km/h to mph (×0.621371) with one decimal', () {
      // 10.0 km/h * 0.621371 = 6.21371 → "6.2 mph"
      expect(formatSpeed(10.0, unit: 'imperial'), '6.2 mph');
    });
  });

  group('formatElapsed', () {
    test('seconds only → MM:SS with zero padding', () {
      expect(formatElapsed(const Duration(seconds: 5)), '00:05');
    });

    test('minutes and seconds → MM:SS zero padded', () {
      expect(
        formatElapsed(const Duration(minutes: 3, seconds: 7)),
        '03:07',
      );
    });

    test('hours present → H:MM:SS with minutes/seconds zero padded', () {
      expect(
        formatElapsed(const Duration(hours: 1, minutes: 2, seconds: 3)),
        '1:02:03',
      );
    });

    test('zero duration → 00:00', () {
      expect(formatElapsed(Duration.zero), '00:00');
    });
  });

  group('trailDisplayDuration', () {
    Trail buildTrail({double duration = 0, double? movingDuration}) {
      return Trail(
        id: 'trail-1',
        name: 'Sample Trail',
        duration: duration,
        movingDuration: movingDuration,
        created: DateTime(2026),
        updated: DateTime(2026),
      );
    }

    test('movingDuration null returns duration', () {
      final trail = buildTrail(duration: 3600, movingDuration: null);
      expect(trailDisplayDuration(trail), 3600);
    });

    test('movingDuration 0 returns duration (a zero moving time is not a value)', () {
      final trail = buildTrail(duration: 3600, movingDuration: 0);
      expect(trailDisplayDuration(trail), 3600);
    });

    test('movingDuration 1800 and duration 3600 returns 1800', () {
      final trail = buildTrail(duration: 3600, movingDuration: 1800);
      expect(trailDisplayDuration(trail), 1800);
    });

    test('movingDuration 1800 and duration 0 returns 1800', () {
      final trail = buildTrail(duration: 0, movingDuration: 1800);
      expect(trailDisplayDuration(trail), 1800);
    });
  });

  // Convention per 24-UI-SPEC.md: one decimal place, unit steps at KB/MB/GB
  // (e.g. "45 MB", "2.4 GB"); bare "N B" below 1 KB.
  group('formatBytes', () {
    test('0 bytes formats as "0 B"', () {
      expect(formatBytes(0), '0 B');
    });

    test('512 bytes formats as "512 B"', () {
      expect(formatBytes(512), '512 B');
    });

    test('1024 bytes formats as "1.0 KB"', () {
      expect(formatBytes(1024), '1.0 KB');
    });

    test('1536 bytes formats as "1.5 KB"', () {
      expect(formatBytes(1536), '1.5 KB');
    });

    test('1 MB formats as "1.0 MB"', () {
      expect(formatBytes(1024 * 1024), '1.0 MB');
    });

    test('2.4 GB formats as "2.4 GB"', () {
      expect(formatBytes((2.4 * 1024 * 1024 * 1024).round()), '2.4 GB');
    });
  });

  // -------------------------------------------------------------------------
  // Mirrors web/src/lib/util/format_util.test.ts case for case. When one side
  // gains a case, the other should gain it too — the two implementations only
  // stay interchangeable as long as both suites agree.
  // -------------------------------------------------------------------------

  group('formatHtmlAsText', () {
    test('returns an empty string for missing input', () {
      expect(formatHtmlAsText(null), '');
      expect(formatHtmlAsText(''), '');
    });

    test('strips inline markup but keeps the text', () {
      expect(formatHtmlAsText('<p>a <strong>bold</strong> word</p>'),
          'a bold word');
    });

    test('separates block elements by a blank line', () {
      expect(formatHtmlAsText('<p>one</p><p>two</p>'), 'one\n\ntwo');
      expect(formatHtmlAsText('<ul><li>one</li><li>two</li></ul>'), 'one\n\ntwo');
    });

    test('turns line breaks into single newlines', () {
      expect(formatHtmlAsText('one<br>two<br />three'), 'one\ntwo\nthree');
    });

    test('keeps torn markup out of the output', () {
      // What `summary.substring(0, bioMaxLength)` used to produce (web #1128)
      expect(formatHtmlAsText('<p>text <strong>bold and it'),
          'text bold and it');
    });

    test('survives attribute values containing angle brackets', () {
      expect(
        formatHtmlAsText('<a href="/x?a=1&b=2" title="a > b">link</a>'),
        'link',
      );
    });

    test('drops script and style content', () {
      expect(
        formatHtmlAsText("<p>keep</p><script>alert('x')</script>"),
        'keep',
      );
      expect(formatHtmlAsText('<style>p { color: red; }</style>keep'), 'keep');
    });

    test('decodes named and numeric entities', () {
      expect(formatHtmlAsText('a &amp; b'), 'a & b');
      expect(formatHtmlAsText('&lt;p&gt;'), '<p>');
      expect(
        formatHtmlAsText('&quot;q&quot; &#39;a&#39; &#x27;b&#x27;'),
        '"q" \'a\' \'b\'',
      );
      expect(formatHtmlAsText('caf&#233; &#x2014; open'), 'café — open');
    });

    test('decodes entities exactly once', () {
      // The author wrote "&lt;p&gt;", which the editor stored as
      // "&amp;lt;p&amp;gt;". Decoding twice would turn that back into a tag
      // and lose it.
      expect(formatHtmlAsText('<p>&amp;lt;p&amp;gt; stays</p>'),
          '&lt;p&gt; stays');
    });

    test('leaves unknown entities alone', () {
      expect(
        formatHtmlAsText('&unknownentity; &#xZZ;'),
        '&unknownentity; &#xZZ;',
      );
    });

    test('collapses redundant whitespace', () {
      expect(formatHtmlAsText('<p>a</p><p></p><p></p><p>b</p>'), 'a\n\nb');
      expect(formatHtmlAsText('<p>  padded  </p>'), 'padded');
      expect(formatHtmlAsText('a    b'), 'a  b');
    });
  });

  group('formatHtmlAsTextPreview', () {
    test('reports untruncated text unchanged', () {
      final result = formatHtmlAsTextPreview('<p>short</p>', 100);

      expect(result.text, 'short');
      expect(result.truncated, isFalse);
    });

    test('truncates to the visible character count, not the HTML length', () {
      // 30 characters of text behind far more than 30 characters of markup
      final html = '<p><strong><em>${'a' * 30}</em></strong></p>';
      final result = formatHtmlAsTextPreview(html, 10);

      expect(result.text, 'a' * 10);
      expect(result.truncated, isTrue);
    });

    test('counts code points so the cutoff never splits an astral character',
        () {
      final result = formatHtmlAsTextPreview('<p>🏔️🏔️🏔️</p>', 2);

      expect(result.truncated, isTrue);
      expect(result.text.runes.length, 2);
      expect(result.text, '🏔️');
    });

    test('treats a boundary-length description as untruncated', () {
      final result = formatHtmlAsTextPreview('<p>${'a' * 100}</p>', 100);

      expect(result.truncated, isFalse);
    });

    test('handles missing descriptions', () {
      final result = formatHtmlAsTextPreview(null, 100);

      expect(result.text, '');
      expect(result.truncated, isFalse);
    });
  });
}
