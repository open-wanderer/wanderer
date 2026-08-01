import 'package:flutter_test/flutter_test.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
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

  group('computeTrailMetrics', () {
    test('CONV-01: a two-point segment reports the full hop instead of 0', () {
      // Pre-fix value on this exact fixture: exactly 0 -- the loop started
      // at i = 1, so only the second point was ever fed to addAndFilter().
      final xml = _gpxXml([_trkptXml(47.0, 11.0), _trkptXml(47.001, 11.001)]);
      final metrics = computeTrailMetrics(parseGpxSafely(xml));

      expect(metrics.distance, closeTo(134.59, 0.5));
    });

    test(
      'CONV-01/CONV-02: bounding box and centroid include the geographic-extreme first point',
      () {
        // Pre-fix: minLat 47.0, centroid 31.33/7.33 -- the first point
        // (also the geographic extreme) was skipped by the i = 1 loop
        // bound.
        final xml = _gpxXml([
          _trkptXml(40.0, 10.0),
          _trkptXml(47.0, 11.0),
          _trkptXml(48.0, 12.0),
        ]);
        final metrics = computeTrailMetrics(parseGpxSafely(xml));

        expect(metrics.minLat, 40.0);
        expect(metrics.minLon, 10.0);
        expect(metrics.centroidLat, closeTo(45.0, 1e-9));
        expect(metrics.centroidLon, closeTo(11.0, 1e-9));
      },
    );

    test(
      'CONV-03: an omitted <ele> is carried forward as "no data", not sea level',
      () {
        // Pre-fix: elevationGain 1015, elevationLoss 1005 -- the missing
        // tag coerced to 0, fabricating a plunge to sea level and back.
        final xml = _gpxXml([
          _trkptXml(47.000, 11.0, ele: '1000'),
          _trkptXml(47.001, 11.0, ele: '1005'),
          _trkptXml(47.002, 11.0),
          _trkptXml(47.003, 11.0, ele: '1010'),
          _trkptXml(47.004, 11.0, ele: '1015'),
        ]);
        final metrics = computeTrailMetrics(parseGpxSafely(xml));

        expect(metrics.elevationGain, 15.0);
        expect(metrics.elevationLoss, 0.0);
      },
    );

    test(
      'CONV-03: an empty <ele></ele> is the parser landmine canary, treated identically to an omitted tag',
      () {
        // Pre-fix: elevationGain 1015, elevationLoss 1005. This is the
        // exact fixture that crashes the raw parser (34-RESEARCH.md
        // Pitfall 1).
        final xml = _gpxXml([
          _trkptXml(47.000, 11.0, ele: '1000'),
          _trkptXml(47.001, 11.0, ele: '1005'),
          _trkptXml(47.002, 11.0, ele: ''),
          _trkptXml(47.003, 11.0, ele: '1010'),
          _trkptXml(47.004, 11.0, ele: '1015'),
        ]);
        final metrics = computeTrailMetrics(parseGpxSafely(xml));

        expect(metrics.elevationGain, 15.0);
        expect(metrics.elevationLoss, 0.0);
      },
    );

    test(
      'CONV-03: a genuine <ele>0</ele> counts as real sea-level data, not missing',
      () {
        final xml = _gpxXml([
          _trkptXml(47.0, 11.0, ele: '0'),
          _trkptXml(47.001, 11.0, ele: '10'),
        ]);
        final metrics = computeTrailMetrics(parseGpxSafely(xml));

        expect(metrics.elevationGain, 10.0);
        expect(metrics.elevationLoss, 0.0);
      },
    );

    test(
      'CONV-04: registers the full climb of an 88 m scramble spread over ~4.4 m of horizontal movement',
      () {
        // Pre-fix: elevationGain 0 -- the smoothed elevation diff was
        // gated behind the horizontal threshold, which this stretch never
        // clears.
        final trkpts = [
          for (var i = 0; i < 12; i++)
            _trkptXml(47 + i * 0.0000036, 11.0, ele: '${1000 + i * 8}'),
        ];
        final metrics = computeTrailMetrics(parseGpxSafely(_gpxXml(trkpts)));

        expect(metrics.elevationGain, 88.0);
        expect(metrics.elevationLoss, 0.0);
        // Distance smoothing must stay gated for the same stretch.
        expect(metrics.distance, 0.0);
      },
    );

    test(
      'D-04: a completed track reports finalElevationGain (88), not the monotonic totalElevationGainSmoothed (80)',
      () {
        // The single test that catches porting the wrong one of the pair
        // (Pitfall 2): this monotonic climb ends without a confirming
        // move, so its last 8 m step is still sitting in the noise
        // filter's pending slot.
        final trkpts = [
          for (var i = 0; i < 12; i++)
            _trkptXml(47 + i * 0.0000036, 11.0, ele: '${1000 + i * 8}'),
        ];
        final points = _flatten(parseGpxSafely(_gpxXml(trkpts)));
        final metrics = GpxMetricsComputation(5, 5);
        for (final point in points) {
          metrics.addAndFilter(point);
        }

        expect(metrics.finalElevationGain, 88.0);
        expect(metrics.totalElevationGainSmoothed, 80.0);
      },
    );

    test(
      'CONV-04: a fully-stationary track whose altitude oscillates +/-7 m and returns to start reports 0/0',
      () {
        // Pre-fix: elevationGain 210, elevationLoss 210 -- the flat
        // threshold commit rule ratchets on every +/-7 m swing even
        // though the track never moves and returns exactly to its
        // starting elevation.
        final trkpts = [
          for (var i = 0; i <= 60; i++)
            _trkptXml(47.0, 11.0, ele: i.isEven ? '1000' : '1007'),
        ];
        final metrics = computeTrailMetrics(parseGpxSafely(_gpxXml(trkpts)));

        expect(metrics.elevationGain, 0.0);
        expect(metrics.elevationLoss, 0.0);
      },
    );

    test(
      'CONV-04: the same stationary oscillation ending mid-swing reports exactly the one un-cancelled excursion',
      () {
        // Pre-fix: elevationGain 210, elevationLoss 203.
        final trkpts = [
          for (var i = 0; i < 60; i++)
            _trkptXml(47.0, 11.0, ele: i.isEven ? '1000' : '1007'),
        ];
        final metrics = computeTrailMetrics(parseGpxSafely(_gpxXml(trkpts)));

        expect(metrics.elevationGain, 7.0);
        expect(metrics.elevationLoss, 0.0);
      },
    );

    test(
      'CONV-04: rejects a stationary out-and-back bump but measures the genuine climb that follows in full',
      () {
        // Pre-fix: elevationGain 32, elevationLoss 8.
        final trkpts = [
          _trkptXml(47.0, 11.0, ele: '1000'),
          _trkptXml(47.0, 11.0, ele: '1008'),
          _trkptXml(47.0, 11.0, ele: '1000'),
          _trkptXml(47.0, 11.0, ele: '1008'),
          _trkptXml(47.0, 11.0, ele: '1016'),
          _trkptXml(47.0, 11.0, ele: '1024'),
        ];
        final metrics = computeTrailMetrics(parseGpxSafely(_gpxXml(trkpts)));

        expect(metrics.elevationGain, 24.0);
        expect(metrics.elevationLoss, 0.0);
      },
    );

    test(
      'CONV-04 rolling-terrain guard: noise rejection never eats real terrain',
      () {
        const elevations = [1000, 1008, 1000, 1008, 1000, 1008];
        final trkpts = [
          for (var i = 0; i < 6; i++)
            _trkptXml(47 + i * 0.0009, 11.0, ele: '${elevations[i]}'),
        ];
        final metrics = computeTrailMetrics(parseGpxSafely(_gpxXml(trkpts)));

        expect(metrics.elevationGain, 24.0);
        expect(metrics.elevationLoss, 16.0);
      },
    );

    test(
      'CONV-05: suppresses GPS jitter in totalDistanceSmoothed while totalDistance stays raw',
      () {
        // Pre-33-01 (i = 1 loop bug): 90.068 m -- a different defect
        // entirely. Real forward travel is ~100.075 m; the raw haversine
        // sum over every consecutive pair is ~110.083 m.
        final trkpts = [_trkptXml(47.0, 11.0)];
        var lat = 47.0;
        for (var i = 0; i < 5; i++) {
          lat += 0.00018;
          trkpts.add(_trkptXml(lat, 11.0));
          lat += 0.000009;
          trkpts.add(_trkptXml(lat, 11.0));
          lat -= 0.000009;
          trkpts.add(_trkptXml(lat, 11.0));
        }
        final points = _flatten(parseGpxSafely(_gpxXml(trkpts)));
        final metrics = GpxMetricsComputation(5, 5);
        for (final point in points) {
          metrics.addAndFilter(point);
        }

        expect(metrics.totalDistanceSmoothed, closeTo(100.075, 1.0));
        expect(metrics.totalDistance, closeTo(110.083, 1.0));
      },
    );

    test(
      'cross-segment continuity: a two-leg planner route measures through its shared anchor, no per-segment reset',
      () {
        // Pre-fix: 333.585 -- the i = 1 loop bound dropped each segment's
        // own first point (leg1's opening hop and leg2's zero-length
        // anchor duplicate), losing one real hop's worth of distance.
        final leg1 = [
          _trkptXml(47.0, 11.0),
          _trkptXml(47.001, 11.0),
          _trkptXml(47.002, 11.0),
        ];
        final leg2 = [
          _trkptXml(47.002, 11.0),
          _trkptXml(47.003, 11.0),
          _trkptXml(47.004, 11.0),
        ];
        final xml = _gpxXmlSegments([leg1, leg2]);
        final metrics = computeTrailMetrics(parseGpxSafely(xml));

        expect(metrics.distance, closeTo(444.78, 1.0));
      },
    );

    test('duration: two trkpts 30 minutes apart report durationMs == 1800000', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><time>2024-01-01T00:00:00Z</time></trkpt>',
        '<trkpt lat="47.001" lon="11.0"><time>2024-01-01T00:30:00Z</time></trkpt>',
      ]);
      final metrics = computeTrailMetrics(parseGpxSafely(xml));

      expect(metrics.durationMs, 1800000);
    });

    test(
      'duration: a segment where only the middle point carries a <time> reports durationMs == 0',
      () {
        // gpx.ts:116-123 requires both the first AND last point of a
        // segment to carry a time; a time on an interior point alone does
        // not contribute.
        final xml = _gpxXml([
          '<trkpt lat="47.0" lon="11.0"></trkpt>',
          '<trkpt lat="47.001" lon="11.0"><time>2024-01-01T00:30:00Z</time></trkpt>',
          '<trkpt lat="47.002" lon="11.0"></trkpt>',
        ]);
        final metrics = computeTrailMetrics(parseGpxSafely(xml));

        expect(metrics.durationMs, 0);
      },
    );

    test(
      'empty-document guard: a track with no points keeps the pre-existing sentinel behavior, not a fix',
      () {
        final xml = _gpxXml(const []);
        final metrics = computeTrailMetrics(parseGpxSafely(xml));

        expect(metrics.pointCount, 0);
        expect(metrics.distance, 0.0);
        expect(metrics.centroidLat.isNaN, isTrue);
        expect(metrics.minLat, double.infinity);
        expect(metrics.maxLat, double.negativeInfinity);
      },
    );
  });

  group('trailFromGpx', () {
    test('name: metadata name wins over the trk name and fallbackName', () {
      final xml = '''<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="wanderer-test" xmlns="http://www.topografix.com/GPX/1/1">
  <metadata><name>Metadata Name</name></metadata>
  <trk><name>Trk Name</name><trkseg><trkpt lat="47.0" lon="11.0"></trkpt></trkseg></trk>
</gpx>''';
      final trail = trailFromGpx(
        parseGpxSafely(xml),
        fallbackName: 'Fallback',
      );
      expect(trail.name, 'Metadata Name');
    });

    test('name: an empty metadata name falls through to the trk name', () {
      final xml = '''<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="wanderer-test" xmlns="http://www.topografix.com/GPX/1/1">
  <metadata><name></name></metadata>
  <trk><name>Trk Name</name><trkseg><trkpt lat="47.0" lon="11.0"></trkpt></trkseg></trk>
</gpx>''';
      final trail = trailFromGpx(
        parseGpxSafely(xml),
        fallbackName: 'Fallback',
      );
      expect(trail.name, 'Trk Name');
    });

    test(
      'name: empty metadata and trk names fall through to fallbackName',
      () {
        final xml = '''<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="wanderer-test" xmlns="http://www.topografix.com/GPX/1/1">
  <metadata><name></name></metadata>
  <trk><name></name><trkseg><trkpt lat="47.0" lon="11.0"></trkpt></trkseg></trk>
</gpx>''';
        final trail = trailFromGpx(
          parseGpxSafely(xml),
          fallbackName: 'Fallback',
        );
        expect(trail.name, 'Fallback');
      },
    );

    test(
      'description defaults to an empty string when metadata has no desc',
      () {
        final xml = _gpxXml([_trkptXml(47.0, 11.0)]);
        final trail = trailFromGpx(parseGpxSafely(xml));
        expect(trail.description, '');
      },
    );

    test('lat/lon come from the first track point', () {
      final xml = _gpxXml([_trkptXml(47.5, 11.5), _trkptXml(47.6, 11.6)]);
      final trail = trailFromGpx(parseGpxSafely(xml));
      expect(trail.lat, 47.5);
      expect(trail.lon, 11.5);
    });

    test(
      'date is set only when both the first and last point carry a time, as the UTC calendar date of the first',
      () {
        final xml = _gpxXml([
          '<trkpt lat="47.0" lon="11.0"><time>2024-06-15T23:30:00Z</time></trkpt>',
          '<trkpt lat="47.001" lon="11.0"><time>2024-06-16T00:30:00Z</time></trkpt>',
        ]);
        final trail = trailFromGpx(parseGpxSafely(xml));
        expect(trail.date, DateTime.utc(2024, 6, 15));
      },
    );

    test('date stays null when only the first point carries a time', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><time>2024-06-15T23:30:00Z</time></trkpt>',
        '<trkpt lat="47.001" lon="11.0"></trkpt>',
      ]);
      final trail = trailFromGpx(parseGpxSafely(xml));
      expect(trail.date, isNull);
    });

    test('duration equals durationMs / 1000', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><time>2024-01-01T00:00:00Z</time></trkpt>',
        '<trkpt lat="47.001" lon="11.0"><time>2024-01-01T00:30:00Z</time></trkpt>',
      ]);
      final trail = trailFromGpx(parseGpxSafely(xml));
      expect(trail.duration, 1800.0);
    });

    test(
      'D-13: the movingDuration override populates movingDuration, never duration',
      () {
        final xml = _gpxXml([
          '<trkpt lat="47.0" lon="11.0"><time>2024-01-01T00:00:00Z</time></trkpt>',
          '<trkpt lat="47.001" lon="11.0"><time>2024-01-01T00:30:00Z</time></trkpt>',
        ]);
        final trail = trailFromGpx(
          parseGpxSafely(xml),
          movingDuration: const Duration(minutes: 42),
        );
        expect(trail.movingDuration, 2520.0);
        expect(trail.duration, 1800.0);
      },
    );

    test('movingDuration is null when the override is omitted', () {
      final xml = _gpxXml([_trkptXml(47.0, 11.0)]);
      final trail = trailFromGpx(parseGpxSafely(xml));
      expect(trail.movingDuration, isNull);
    });

    test(
      'waypoint mapping: a known sym resolves through fontAwesomeIconsMap, an unknown sym falls back to the default circle',
      () {
        final xml = '''<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="wanderer-test" xmlns="http://www.topografix.com/GPX/1/1">
  <wpt lat="47.1" lon="11.1"><name>Camp</name><desc>A camp</desc><sym>campground</sym></wpt>
  <wpt lat="47.2" lon="11.2"><name>Mystery</name><sym>not-a-real-icon</sym></wpt>
  <trk><trkseg><trkpt lat="47.0" lon="11.0"></trkpt></trkseg></trk>
</gpx>''';
        final trail = trailFromGpx(parseGpxSafely(xml));
        final waypoints = trail.expand!.waypointsViaTrail!;
        expect(waypoints.length, 2);
        expect(waypoints[0].icon, FontAwesomeIcons.campground);
        expect(waypoints[1].icon, FontAwesomeIcons.circle);
      },
    );

    test(
      "bounding-box guard leaves the model's 0 defaults for a trackless GPX",
      () {
        final xml = '''<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="wanderer-test" xmlns="http://www.topografix.com/GPX/1/1">
  <metadata><name>No Track</name></metadata>
</gpx>''';
        final trail = trailFromGpx(parseGpxSafely(xml));
        expect(trail.minLat, 0);
        expect(trail.maxLat, 0);
        expect(trail.minLon, 0);
        expect(trail.maxLon, 0);
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

/// Produces a GPX 1.1 document with one `<trk>` holding one `<trkseg>` per
/// entry in [segments], mirroring `valhalla_store.svelte.ts`'s
/// `insertIntoRoute()` output shape: each planner leg is its own track
/// segment, with the shared anchor point deliberately repeated as the next
/// leg's first point.
String _gpxXmlSegments(List<List<String>> segments) {
  final trksegs = segments
      .map(
        (trkpts) => '''<trkseg>
      ${trkpts.join('\n      ')}
    </trkseg>''',
      )
      .join('\n    ');

  return '''<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="wanderer-test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    $trksegs
  </trk>
</gpx>''';
}

/// Produces a single `<trkpt>` XML string at [lat]/[lon], with an `<ele>`
/// element only when [ele] is non-null (mirroring `trkptXml` in
/// `gpx-metrics-computation.test.ts:319-321`).
String _trkptXml(double lat, double lon, {String? ele}) {
  final eleElement = ele != null ? '<ele>$ele</ele>' : '';
  return '<trkpt lat="$lat" lon="$lon">$eleElement</trkpt>';
}

/// Flattens every `<trkpt>` across every track/segment in [gpx] into a
/// single ordered list, mirroring `gpx.ts`'s `flatten()`.
List<Wpt> _flatten(Gpx gpx) {
  return [
    for (final track in gpx.trks)
      for (final segment in track.trksegs) ...segment.trkpts,
  ];
}
