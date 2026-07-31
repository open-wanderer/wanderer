import 'package:flutter_test/flutter_test.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/util/gpx_conversion_util.dart';

void main() {
  group('sanitizeGpxEleAndTime', () {
    test('preserves a genuine <ele>0</ele> (real sea level, CONV-03)', () {
      final xml = _gpxXml(['<trkpt lat="47.0" lon="11.0"><ele>0</ele></trkpt>']);
      expect(sanitizeGpxEleAndTime(xml), contains('<ele>0</ele>'));
    });

    test(
      'preserves a pretty-printed <ele>\\n 1000.5\\n</ele> (double.tryParse trims)',
      () {
        final xml = _gpxXml([
          '<trkpt lat="47.0" lon="11.0"><ele>\n 1000.5\n </ele></trkpt>',
        ]);
        expect(sanitizeGpxEleAndTime(xml), contains('<ele>\n 1000.5\n </ele>'));
      },
    );

    test('rewrites an empty <ele></ele> to a self-closing <ele/>', () {
      final xml = _gpxXml(['<trkpt lat="47.0" lon="11.0"><ele></ele></trkpt>']);
      final sanitized = sanitizeGpxEleAndTime(xml);
      expect(sanitized, contains('<ele/>'));
      expect(sanitized, isNot(contains('<ele></ele>')));
    });

    test('rewrites a whitespace-only <ele>   </ele> to a self-closing <ele/>', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><ele>   </ele></trkpt>',
      ]);
      expect(sanitizeGpxEleAndTime(xml), contains('<ele/>'));
    });

    test('rewrites a non-numeric <ele>N/A</ele> to a self-closing <ele/>', () {
      final xml = _gpxXml(['<trkpt lat="47.0" lon="11.0"><ele>N/A</ele></trkpt>']);
      expect(sanitizeGpxEleAndTime(xml), contains('<ele/>'));
    });

    test('rewrites <ele>NaN</ele> and <ele>Infinity</ele> to self-closing tags', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><ele>NaN</ele></trkpt>',
        '<trkpt lat="47.001" lon="11.0"><ele>Infinity</ele></trkpt>',
      ]);
      final sanitized = sanitizeGpxEleAndTime(xml);
      expect(sanitized, isNot(contains('<ele>NaN</ele>')));
      expect(sanitized, isNot(contains('<ele>Infinity</ele>')));
    });

    test('rewrites an empty <time></time> to a self-closing <time/>', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><time></time></trkpt>',
      ]);
      final sanitized = sanitizeGpxEleAndTime(xml);
      expect(sanitized, contains('<time/>'));
      expect(sanitized, isNot(contains('<time></time>')));
    });

    test('preserves a valid <time>...</time>', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><time>2024-01-01T00:00:00Z</time></trkpt>',
      ]);
      expect(
        sanitizeGpxEleAndTime(xml),
        contains('<time>2024-01-01T00:00:00Z</time>'),
      );
    });
  });

  group('parseGpxSafely', () {
    test('an empty <ele></ele> parses to a null ele, not throwing', () {
      final xml = _gpxXml(['<trkpt lat="47.0" lon="11.0"><ele></ele></trkpt>']);
      final gpx = parseGpxSafely(xml);
      expect(gpx.trks.single.trksegs.single.trkpts.single.ele, isNull);
    });

    test('a whitespace-only <ele>   </ele> parses to a null ele', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><ele>   </ele></trkpt>',
      ]);
      final gpx = parseGpxSafely(xml);
      expect(gpx.trks.single.trksegs.single.trkpts.single.ele, isNull);
    });

    test('a non-numeric <ele>N/A</ele> parses to a null ele', () {
      final xml = _gpxXml(['<trkpt lat="47.0" lon="11.0"><ele>N/A</ele></trkpt>']);
      final gpx = parseGpxSafely(xml);
      expect(gpx.trks.single.trksegs.single.trkpts.single.ele, isNull);
    });

    test('an empty <time></time> parses to a null time', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><time></time></trkpt>',
      ]);
      final gpx = parseGpxSafely(xml);
      expect(gpx.trks.single.trksegs.single.trkpts.single.time, isNull);
    });

    test('a genuine <ele>0</ele> parses to 0.0, not "missing"', () {
      final xml = _gpxXml(['<trkpt lat="47.0" lon="11.0"><ele>0</ele></trkpt>']);
      final gpx = parseGpxSafely(xml);
      expect(gpx.trks.single.trksegs.single.trkpts.single.ele, 0.0);
    });

    test('a pretty-printed <ele> with surrounding newlines/indent parses to 1000.5', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><ele>\n 1000.5\n </ele></trkpt>',
      ]);
      final gpx = parseGpxSafely(xml);
      expect(gpx.trks.single.trksegs.single.trkpts.single.ele, 1000.5);
    });

    test('an omitted <ele> parses to a null ele', () {
      final xml = _gpxXml(['<trkpt lat="47.0" lon="11.0"></trkpt>']);
      final gpx = parseGpxSafely(xml);
      expect(gpx.trks.single.trksegs.single.trkpts.single.ele, isNull);
    });

    test(
      'the unsanitised parser throws on the <ele></ele> fixture but parseGpxSafely does not',
      () {
        final xml = _gpxXml(['<trkpt lat="47.0" lon="11.0"><ele></ele></trkpt>']);

        expect(() => GpxReader().fromString(xml), throwsA(anything));
        expect(() => parseGpxSafely(xml), returnsNormally);
      },
    );
  });
}

/// Produces a GPX 1.1 document with one `<trk>` and one `<trkseg>` holding
/// [trkpts] (already-formed `<trkpt>...</trkpt>` XML strings), matching the
/// `gpxXml` helper in `gpx-metrics-computation.test.ts:325-333` so the two
/// suites stay comparable.
String _gpxXml(List<String> trkpts) {
  return '''<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="wanderer-test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <trkseg>
      ${trkpts.join('\n      ')}
    </trkseg>
  </trk>
</gpx>''';
}
