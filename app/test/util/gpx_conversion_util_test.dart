import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:gpx/gpx.dart';
import 'package:gpx/gpx.dart' as pub;
import 'package:wanderer/util/gpx_conversion_util.dart';

void main() {
  // These groups used to assert on a pair of pre-parse regex sanitizers. The
  // tolerance now lives in the vendored reader (lib/vendor/gpx/gpx_reader.dart),
  // so they assert on PARSED VALUES instead of rewritten strings — which is the
  // property that actually matters and survives a change of mechanism.
  //
  // Each group pins the same contract twice: the PUBLISHED package:gpx reader
  // throws on the input, and parseGpxSafely does not. That pairing is what
  // proves the vendored reader is doing work; without it these tests would
  // still pass if the vendoring were reverted.
  group('vendored reader tolerance - <ele> and <time>', () {
    test('preserves a genuine <ele>0</ele> (real sea level, CONV-03)', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><ele>0</ele></trkpt>',
      ]);
      expect(
        parseGpxSafely(xml).trks.single.trksegs.single.trkpts.single.ele,
        0.0,
      );
    });

    test('preserves a pretty-printed <ele> with surrounding whitespace', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><ele>\n 1000.5\n </ele></trkpt>',
      ]);
      expect(
        parseGpxSafely(xml).trks.single.trksegs.single.trkpts.single.ele,
        1000.5,
      );
    });

    test(
      'an empty <ele></ele> becomes null where the package reader throws',
      () {
        final xml = _gpxXml([
          '<trkpt lat="47.0" lon="11.0"><ele></ele></trkpt>',
        ]);

        expect(() => GpxReader().fromString(xml), throwsFormatException);
        expect(
          parseGpxSafely(xml).trks.single.trksegs.single.trkpts.single.ele,
          isNull,
        );
      },
    );

    test('a non-numeric <ele>N/A</ele> becomes null', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><ele>N/A</ele></trkpt>',
      ]);
      expect(
        parseGpxSafely(xml).trks.single.trksegs.single.trkpts.single.ele,
        isNull,
      );
    });

    test(
      'an empty <time></time> becomes null where the package reader throws',
      () {
        final xml = _gpxXml([
          '<trkpt lat="47.0" lon="11.0"><time></time></trkpt>',
        ]);

        expect(() => GpxReader().fromString(xml), throwsFormatException);
        expect(
          parseGpxSafely(xml).trks.single.trksegs.single.trkpts.single.time,
          isNull,
        );
      },
    );

    test('a valid <time> still parses', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><time>2024-01-01T10:00:00Z</time></trkpt>',
      ]);
      expect(
        parseGpxSafely(xml).trks.single.trksegs.single.trkpts.single.time,
        DateTime.utc(2024, 1, 1, 10),
      );
    });
  });

  group('vendored reader tolerance - the other numerically-parsed tags', () {
    // <sat>/<dgpsid> reach _readInt; the rest reach _readDouble.
    const doubleTags = [
      'hdop',
      'vdop',
      'pdop',
      'magvar',
      'geoidheight',
      'ageofdgpsdata',
    ];
    const intTags = ['sat', 'dgpsid'];

    double? readDouble(Wpt p, String tag) => switch (tag) {
      'hdop' => p.hdop,
      'vdop' => p.vdop,
      'pdop' => p.pdop,
      'magvar' => p.magvar,
      'geoidheight' => p.geoidheight,
      'ageofdgpsdata' => p.ageofdgpsdata,
      _ => throw ArgumentError(tag),
    };
    int? readInt(Wpt p, String tag) => switch (tag) {
      'sat' => p.sat,
      'dgpsid' => p.dgpsid,
      _ => throw ArgumentError(tag),
    };

    for (final tag in [...doubleTags, ...intTags]) {
      for (final (label, body) in [
        ('empty', ''),
        ('whitespace-only', '   '),
        ('non-numeric', 'N/A'),
      ]) {
        test('a $label <$tag> becomes null instead of throwing', () {
          final xml = _gpxXml([
            '<trkpt lat="47.0" lon="11.0"><$tag>$body</$tag></trkpt>',
          ]);

          // Non-vacuity: prove the published reader really does throw here.
          expect(() => GpxReader().fromString(xml), throwsFormatException);

          final p = parseGpxSafely(
            xml,
          ).trks.single.trksegs.single.trkpts.single;
          expect(
            doubleTags.contains(tag) ? readDouble(p, tag) : readInt(p, tag),
            isNull,
          );
        });
      }
    }

    for (final tag in doubleTags) {
      test('a valid <$tag>1.5</$tag> still parses', () {
        final xml = _gpxXml([
          '<trkpt lat="47.0" lon="11.0"><$tag>1.5</$tag></trkpt>',
        ]);
        final p = parseGpxSafely(xml).trks.single.trksegs.single.trkpts.single;
        expect(readDouble(p, tag), 1.5);
      });
    }

    for (final tag in intTags) {
      test('a valid <$tag>7</$tag> still parses', () {
        final xml = _gpxXml([
          '<trkpt lat="47.0" lon="11.0"><$tag>7</$tag></trkpt>',
        ]);
        final p = parseGpxSafely(xml).trks.single.trksegs.single.trkpts.single;
        expect(readInt(p, tag), 7);
      });
    }

    test('a realistic Garmin-style trkpt carrying every optional tag empty '
        'parses instead of throwing', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0">'
            '<ele></ele><time></time><magvar></magvar>'
            '<geoidheight></geoidheight><sat></sat><hdop></hdop>'
            '<vdop></vdop><pdop></pdop><ageofdgpsdata></ageofdgpsdata>'
            '<dgpsid></dgpsid>'
            '</trkpt>',
        '<trkpt lat="47.001" lon="11.001"><ele>500</ele><sat>8</sat>'
            '<hdop>1.2</hdop></trkpt>',
      ]);

      expect(() => GpxReader().fromString(xml), throwsA(anything));

      final points = parseGpxSafely(xml).trks.single.trksegs.single.trkpts;
      expect(points, hasLength(2));
      expect(points.first.ele, isNull);
      expect(points.first.sat, isNull);
      expect(points.first.hdop, isNull);
      expect(points.last.ele, 500.0);
      expect(points.last.sat, 8);
      expect(points.last.hdop, 1.2);
    });
  });

  group('vendored reader tolerance - structural and metadata inputs', () {
    test(
      'a <trkpt> missing lat/lon yields null coords instead of StateError',
      () {
        final xml = _gpxXml([
          '<trkpt lat="47.0" lon="11.0"><ele>100</ele></trkpt>',
          '<trkpt><ele>200</ele></trkpt>',
        ]);

        expect(() => GpxReader().fromString(xml), throwsStateError);

        final points = parseGpxSafely(xml).trks.single.trksegs.single.trkpts;
        expect(points, hasLength(2));
        expect(points.first.lat, 47.0);
        expect(points.last.lat, isNull);
        expect(points.last.lon, isNull);
        // The rest of the point still parsed — tolerance is per-field.
        expect(points.last.ele, 200.0);
      },
    );

    test('the non-standard <email>a@b.com</email> text form parses', () {
      const xml =
          '<?xml version="1.0"?>'
          '<gpx version="1.1" creator="t">'
          '<metadata><author><name>A</name>'
          '<email>user@example.com</email>'
          '</author></metadata></gpx>';

      expect(() => GpxReader().fromString(xml), throwsStateError);

      final email = parseGpxSafely(xml).metadata!.author!.email!;
      expect(email.id, 'user');
      expect(email.domain, 'example.com');
    });

    test('the spec <email id domain/> attribute form still wins', () {
      const xml =
          '<?xml version="1.0"?>'
          '<gpx version="1.1" creator="t">'
          '<metadata><author><name>A</name>'
          '<email id="spec" domain="example.org"/>'
          '</author></metadata></gpx>';

      final email = parseGpxSafely(xml).metadata!.author!.email!;
      expect(email.id, 'spec');
      expect(email.domain, 'example.org');
    });

    test(
      'WR-06: a partial-attribute <email> does not splice with the text',
      () {
        // `<email domain="a.com">u@b.com</email>` used to yield `u@a.com` — an
        // address present in neither the attributes nor the text. A document
        // that supplies one attribute has chosen the attribute form; an
        // incomplete one is malformed, and empty is the honest answer.
        const xml =
            '<?xml version="1.0"?>'
            '<gpx version="1.1" creator="t">'
            '<metadata><author><name>A</name>'
            '<email domain="a.com">user@b.com</email>'
            '</author></metadata></gpx>';

        final email = parseGpxSafely(xml).metadata!.author!.email!;
        expect(email.domain, 'a.com');
        expect(
          email.id,
          isEmpty,
          reason: 'must not borrow the local part from the text form',
        );
      },
    );

    test('an <email> with no usable address leaves the fields empty', () {
      const xml =
          '<?xml version="1.0"?>'
          '<gpx version="1.1" creator="t">'
          '<metadata><author><name>A</name><email>not-an-address</email>'
          '</author></metadata></gpx>';

      final email = parseGpxSafely(xml).metadata!.author!.email!;
      expect(email.id, isEmpty);
      expect(email.domain, isEmpty);
    });

    test('a CDATA description survives parsing verbatim', () {
      // The retired regex sanitizer had to explicitly exclude CDATA to avoid
      // mutating user-visible text. Parsing has no such hazard by
      // construction; this pins that it stays true.
      const xml =
          '<?xml version="1.0"?>'
          '<gpx version="1.1" creator="t"><metadata>'
          '<desc><![CDATA[readings: <hdop></hdop> and <ele>N/A</ele>]]></desc>'
          '</metadata></gpx>';

      expect(
        parseGpxSafely(xml).metadata!.desc,
        'readings: <hdop></hdop> and <ele>N/A</ele>',
      );
    });

    test('a namespaced <gpx:hdop> is not mistaken for <hdop>', () {
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><gpx:hdop></gpx:hdop>'
            '<hdop>2.5</hdop></trkpt>',
      ]);
      final p = parseGpxSafely(xml).trks.single.trksegs.single.trkpts.single;
      expect(p.hdop, 2.5);
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
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><ele>N/A</ele></trkpt>',
      ]);
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
      final xml = _gpxXml([
        '<trkpt lat="47.0" lon="11.0"><ele>0</ele></trkpt>',
      ]);
      final gpx = parseGpxSafely(xml);
      expect(gpx.trks.single.trksegs.single.trkpts.single.ele, 0.0);
    });

    test(
      'a pretty-printed <ele> with surrounding newlines/indent parses to 1000.5',
      () {
        final xml = _gpxXml([
          '<trkpt lat="47.0" lon="11.0"><ele>\n 1000.5\n </ele></trkpt>',
        ]);
        final gpx = parseGpxSafely(xml);
        expect(gpx.trks.single.trksegs.single.trkpts.single.ele, 1000.5);
      },
    );

    test('an omitted <ele> parses to a null ele', () {
      final xml = _gpxXml(['<trkpt lat="47.0" lon="11.0"></trkpt>']);
      final gpx = parseGpxSafely(xml);
      expect(gpx.trks.single.trksegs.single.trkpts.single.ele, isNull);
    });

    test(
      'the unsanitised parser throws on the <ele></ele> fixture but parseGpxSafely does not',
      () {
        final xml = _gpxXml([
          '<trkpt lat="47.0" lon="11.0"><ele></ele></trkpt>',
        ]);

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
      final trail = trailFromGpx(parseGpxSafely(xml), fallbackName: 'Fallback');
      expect(trail.name, 'Metadata Name');
    });

    test('name: an empty metadata name falls through to the trk name', () {
      final xml = '''<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="wanderer-test" xmlns="http://www.topografix.com/GPX/1/1">
  <metadata><name></name></metadata>
  <trk><name>Trk Name</name><trkseg><trkpt lat="47.0" lon="11.0"></trkpt></trkseg></trk>
</gpx>''';
      final trail = trailFromGpx(parseGpxSafely(xml), fallbackName: 'Fallback');
      expect(trail.name, 'Trk Name');
    });

    test('name: empty metadata and trk names fall through to fallbackName', () {
      final xml = '''<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="wanderer-test" xmlns="http://www.topografix.com/GPX/1/1">
  <metadata><name></name></metadata>
  <trk><name></name><trkseg><trkpt lat="47.0" lon="11.0"></trkpt></trkseg></trk>
</gpx>''';
      final trail = trailFromGpx(parseGpxSafely(xml), fallbackName: 'Fallback');
      expect(trail.name, 'Fallback');
    });

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

    // WR-08 / D-10: `moving_duration = 0` is the exact state D-10 forbids —
    // `form_data_util.dart`'s write guard is `!= null`, not `> 0`, so a zero
    // override used to be persisted and then masked by the display rule's own
    // `> 0` fallback.
    test(
      'D-10: a zero-length override is "no value" (null), never a stored 0',
      () {
        final xml = _gpxXml([_trkptXml(47.0, 11.0)]);
        final trail = trailFromGpx(
          parseGpxSafely(xml),
          movingDuration: Duration.zero,
        );
        expect(trail.movingDuration, isNull);
      },
    );

    test('D-10: a sub-second override, which truncates to 0 whole seconds, '
        'is also "no value"', () {
      final xml = _gpxXml([_trkptXml(47.0, 11.0)]);
      final trail = trailFromGpx(
        parseGpxSafely(xml),
        movingDuration: const Duration(milliseconds: 500),
      );
      expect(trail.movingDuration, isNull);
    });

    test('a one-second override is real data and survives', () {
      final xml = _gpxXml([_trkptXml(47.0, 11.0)]);
      final trail = trailFromGpx(
        parseGpxSafely(xml),
        movingDuration: const Duration(seconds: 1),
      );
      expect(trail.movingDuration, 1.0);
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

  group('vendored reader tolerance - copyright and malformed documents', () {
    test('a <copyright> with no author attribute parses', () {
      const xml =
          '<?xml version="1.0"?>'
          '<gpx version="1.1" creator="t"><metadata>'
          '<copyright><year>2024</year></copyright>'
          '</metadata></gpx>';

      // Non-vacuity: the published reader really does die on this.
      expect(() => pub.GpxReader().fromString(xml), throwsStateError);

      final gpx = parseGpxSafely(xml);
      expect(gpx.metadata!.copyright!.year, 2024);
      expect(gpx.metadata!.copyright!.author, isEmpty);
    });

    test('a <copyright author> still wins when present', () {
      const xml =
          '<?xml version="1.0"?>'
          '<gpx version="1.1" creator="t"><metadata>'
          '<copyright author="OSM"><year>2024</year></copyright>'
          '</metadata></gpx>';
      expect(parseGpxSafely(xml).metadata!.copyright!.author, 'OSM');
    });

    test('a document with no <gpx> element throws FormatException', () {
      // Callers already catch-and-toast, but they could not name the old
      // `_TypeError` in a catch clause or tell it apart from a bug.
      expect(
        () => parseGpxSafely('<?xml version="1.0"?><kml><foo/></kml>'),
        throwsFormatException,
      );
      expect(() => parseGpxSafely(''), throwsFormatException);
      expect(() => parseGpxSafely('not xml at all'), throwsFormatException);
    });
  });

  group('points without a usable position are dropped, not absorbed', () {
    // Round-3 finding: the first version of this guard sat inside
    // addAndFilter's INITIALISATION branch and tested only for `null`, so it
    // covered neither a NaN coordinate nor a bad point mid-track. The original
    // fixture put the bad point at index 0, where the guard did fire — so the
    // assertion passed while both real holes stayed open.
    Gpx flatTrack({Wpt? leading, Wpt? atIndex10}) {
      final pts = <Wpt>[];
      if (leading != null) pts.add(leading);
      for (var i = 0; i < 40; i++) {
        pts.add(Wpt(lat: 47.0 + i * 0.0002, lon: 11.0, ele: 1000));
      }
      if (atIndex10 != null) pts.insert(leading != null ? 11 : 10, atIndex10);
      return Gpx()
        ..trks = [
          Trk(trksegs: [Trkseg(trkpts: pts)]),
        ];
    }

    Gpx clean() => flatTrack();

    test('a leading null-coordinate point does not zero the distance', () {
      final m = computeTrailMetrics(
        flatTrack(leading: Wpt(lat: null, lon: null, ele: 1000)),
      );
      expect(m.distance, closeTo(computeTrailMetrics(clean()).distance, 1e-9));
    });

    test('a leading NaN-coordinate point does not zero the distance', () {
      // double.tryParse accepts "NaN" and "Infinity", so these are NOT null
      // and slipped past a null-only guard — reproducing the exact poisoning
      // it was written to prevent (measured: distance 0.0 on a ~2 km track).
      final m = computeTrailMetrics(
        flatTrack(
          leading: Wpt(lat: double.nan, lon: double.nan, ele: 1000),
        ),
      );
      expect(m.distance, closeTo(computeTrailMetrics(clean()).distance, 1e-9));
    });

    test('an infinite coordinate is treated the same way', () {
      final m = computeTrailMetrics(
        flatTrack(
          leading: Wpt(lat: double.infinity, lon: double.infinity, ele: 1000),
        ),
      );
      expect(m.distance, closeTo(computeTrailMetrics(clean()).distance, 1e-9));
    });

    test('a MID-track bad point does not fabricate elevation', () {
      // The hole the original fixture could not see: the point still entered
      // the elevation diff chain, so one <trkpt ele="3000"> dropped into a flat
      // 1000 m track produced gain 2000 AND loss 2000.
      final m = computeTrailMetrics(
        flatTrack(atIndex10: Wpt(lat: null, lon: null, ele: 3000)),
      );

      expect(m.elevationGain, 0.0);
      expect(m.elevationLoss, 0.0);
      expect(m.distance, closeTo(computeTrailMetrics(clean()).distance, 1e-9));
    });

    test('a MID-track NaN-coordinate point is equally inert', () {
      final m = computeTrailMetrics(
        flatTrack(
          atIndex10: Wpt(lat: double.nan, lon: double.nan, ele: 3000),
        ),
      );

      expect(m.elevationGain, 0.0);
      expect(m.elevationLoss, 0.0);
    });

    test('the vendored reader maps a NaN lat attribute to null', () {
      // Closed one layer earlier too: "NaN" is not a coordinate, so it should
      // never reach the metrics as a number at all.
      const xml =
          '<?xml version="1.0"?>'
          '<gpx version="1.1" creator="t"><trk><trkseg>'
          '<trkpt lat="NaN" lon="Infinity"><ele>1000</ele></trkpt>'
          '</trkseg></trk></gpx>';

      final wpt = parseGpxSafely(xml).trks.single.trksegs.single.trkpts.single;
      expect(wpt.lat, isNull);
      expect(wpt.lon, isNull);
    });
  });

  group('the discard rule reaches every consumer, not just the metrics', () {
    // One predicate — hasUsablePosition — now answers "is this a real point?"
    // for the accumulators, the start coordinate, the waypoints, the centroid
    // and the bounding box. Three different answers used to coexist, which is
    // how one malformed point could zero a distance, fabricate climb, AND
    // leave a marker off the coast of Africa.
    test('WR-01: the start coordinate skips a leading malformed point', () {
      final pts = <Wpt>[
        Wpt(lat: null, lon: null, ele: 1000),
        Wpt(lat: 47.5, lon: 11.5, ele: 1000),
        Wpt(lat: 47.6, lon: 11.6, ele: 1000),
      ];
      final gpx = Gpx()
        ..trks = [
          Trk(trksegs: [Trkseg(trkpts: pts)]),
        ];

      final trail = trailFromGpx(gpx);

      expect(trail.lat, 47.5);
      expect(trail.lon, 11.5);
    });

    test('WR-02: the centroid ignores position-less points entirely', () {
      Gpx build({int padding = 0}) {
        final pts = <Wpt>[
          for (var i = 0; i < padding; i++) Wpt(lat: null, lon: null),
          Wpt(lat: 47.0, lon: 11.0),
          Wpt(lat: 47.2, lon: 11.2),
        ];
        return Gpx()
          ..trks = [
            Trk(trksegs: [Trkseg(trkpts: pts)]),
          ];
      }

      final clean = computeTrailMetrics(build());
      final padded = computeTrailMetrics(build(padding: 8));

      // Previously the 8 bad points summed 0 but still counted, dragging the
      // centroid from 47.1 down toward 9.4.
      expect(padded.centroidLat, closeTo(clean.centroidLat, 1e-9));
      expect(padded.centroidLon, closeTo(clean.centroidLon, 1e-9));
      expect(padded.centroidLat, closeTo(47.1, 1e-9));
    });

    test('WR-02: the bounding box is unaffected by them too', () {
      final pts = <Wpt>[
        Wpt(lat: null, lon: null),
        Wpt(lat: 47.0, lon: 11.0),
        Wpt(lat: 47.2, lon: 11.2),
      ];
      final m = computeTrailMetrics(
        Gpx()
          ..trks = [
            Trk(trksegs: [Trkseg(trkpts: pts)]),
          ],
      );

      expect(m.minLat, closeTo(47.0, 1e-9));
      expect(m.maxLat, closeTo(47.2, 1e-9));
      expect(m.minLon, closeTo(11.0, 1e-9));
      expect(m.maxLon, closeTo(11.2, 1e-9));
    });

    test('WR-03: a coordinate-less <wpt> is dropped, not placed at (0, 0)', () {
      final gpx = Gpx()
        ..wpts = [
          Wpt(lat: null, lon: null, name: 'ghost'),
          Wpt(lat: double.nan, lon: 11.0, name: 'nan'),
          Wpt(lat: 47.0, lon: 11.0, name: 'real'),
        ]
        ..trks = [
          Trk(
            trksegs: [
              Trkseg(trkpts: [Wpt(lat: 47.0, lon: 11.0)]),
            ],
          ),
        ];

      final waypoints = trailFromGpx(gpx).expand!.waypointsViaTrail!;

      expect(waypoints, hasLength(1));
      expect(waypoints.single.name, 'real');
      expect(
        waypoints.any((w) => w.lat == 0 && w.lon == 0),
        isFalse,
        reason: 'a dropped waypoint must never resurface at (0, 0)',
      );
    });
  });

  group('single GpxReader call site gate', () {
    // parseGpxSafely is the only sanctioned parse entry point, but nothing
    // enforced that: trail_provider.dart and trail_entity.dart both constructed
    // a bare GpxReader and so silently opted out of the sanitize chain. A
    // server-authored track with, say, <hdop></hdop> failed to open (the
    // exception swallowed by a broad catch) and, once cached, was permanently
    // un-openable offline. This gate makes a fourth call site fail the build
    // rather than quietly reintroducing that class of defect.
    test('exactly one GpxReader() construction exists in app/lib/, inside '
        'util/gpx_conversion_util.dart', () {
      final libDir = Directory('lib');
      expect(
        libDir.existsSync(),
        isTrue,
        reason:
            'This test must be run with `flutter test`\'s working '
            'directory set to "app/" (e.g. "cd app && flutter test").',
      );

      final matches = <String>[];
      var dartFilesScanned = 0;
      for (final entity in libDir.listSync(recursive: true)) {
        if (entity is! File || !entity.path.endsWith('.dart')) continue;
        dartFilesScanned++;
        for (final line in entity.readAsLinesSync()) {
          // Skip doc/line comments so prose naming GpxReader (including
          // parseGpxSafely's own contract comment) doesn't trip the gate.
          if (line.trimLeft().startsWith('//')) continue;
          if (line.contains('GpxReader(')) matches.add(entity.path);
        }
      }

      // Non-vacuity: a typo'd glob or a wrong working directory would scan
      // nothing and make the hasLength(1) assertion below unreachable.
      expect(
        dartFilesScanned,
        greaterThan(100),
        reason:
            'Expected to scan the whole app/lib tree; only '
            '$dartFilesScanned .dart files were seen, so this gate is not '
            'actually checking anything.',
      );

      expect(
        matches,
        hasLength(1),
        reason:
            'Every GPX the app did not itself produce must be parsed via '
            'parseGpxSafely so it gets the sanitize chain. Found: $matches',
      );
      expect(
        matches.single.replaceAll('\\', '/'),
        endsWith('util/gpx_conversion_util.dart'),
        reason:
            'The sole GpxReader construction must live inside '
            'parseGpxSafely in util/gpx_conversion_util.dart',
      );
    });

    // WR-07: counting constructions is not enough. The construction site could
    // stay put while its IMPORT silently flips to the published package's
    // reader — the throwing one — and every tolerance test above would keep
    // passing, because they exercise parseGpxSafely, whose import is exactly
    // what changed.
    //
    // Scoped to files that actually CONSTRUCT a reader. The other nine lib/
    // files import package:gpx for its models and never touch GpxReader;
    // forcing `hide` on them would be an unexplainable rule that fails the
    // build the next time someone imports a model.
    test('every GpxReader construction resolves to the vendored reader', () {
      final libDir = Directory('lib');
      final offenders = <String>[];
      var constructors = 0;

      for (final entity in libDir.listSync(recursive: true)) {
        if (entity is! File || !entity.path.endsWith('.dart')) continue;
        final path = entity.path.replaceAll(r'\\', '/');
        // The vendored reader DEFINES GpxReader rather than resolving one.
        if (path.endsWith('vendor/gpx/gpx_reader.dart')) continue;

        final lines = entity.readAsLinesSync();
        final constructs = lines.any(
          (l) => !l.trimLeft().startsWith('//') && l.contains('GpxReader('),
        );
        if (!constructs) continue;
        constructors++;

        final importsVendored = lines.any(
          (l) => l.contains("vendor/gpx/gpx_reader.dart"),
        );
        final hidesPublished = lines.any(
          (l) =>
              l.trimLeft().startsWith("import 'package:gpx/gpx.dart'") &&
              l.contains('hide GpxReader'),
        );

        if (!importsVendored || !hidesPublished) {
          offenders.add(
            '${entity.path} '
            '(importsVendored=$importsVendored, hidesPublished=$hidesPublished)',
          );
        }
      }

      // Non-vacuity: if nothing constructs a reader, the loop above asserted
      // nothing at all.
      expect(
        constructors,
        greaterThan(0),
        reason: 'no GpxReader construction found — this gate checked nothing',
      );
      expect(
        offenders,
        isEmpty,
        reason:
            'A file constructing GpxReader must import the vendored reader AND '
            'hide the published one, or it silently gets the throwing reader: '
            '$offenders',
      );
    });
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
        (trkpts) =>
            '''<trkseg>
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
