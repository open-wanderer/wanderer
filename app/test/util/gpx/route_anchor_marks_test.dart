import 'package:flutter_test/flutter_test.dart';
import 'package:gpx/gpx.dart' hide GpxReader;
import 'package:wanderer/util/gpx/conversion.dart';
import 'package:wanderer/util/gpx/gpx.dart';
import 'package:wanderer/util/gpx/route_anchor_marks.dart';
import 'package:wanderer/util/route/planner_handoff.dart';
import 'package:wanderer/vendor/gpx/gpx_reader.dart';

/// A 3-leg loop as the web's `trail2gpx` exports it since #1370: one
/// segment, the two inner leg boundaries marked with `wanderer:anchor`.
const _joinedLoop =
    '<?xml version="1.0" encoding="UTF-8"?>'
    '<gpx version="1.1" creator="wanderer" xmlns="http://www.topografix.com/GPX/1/1" '
    'xmlns:wanderer="https://wanderer.to/xmlns/gpx/1"><trk><trkseg>'
    '<trkpt lat="47.0" lon="11.0"/>'
    '<trkpt lat="47.1" lon="11.1"/>'
    '<trkpt lat="47.2" lon="11.2"><extensions><wanderer:anchor/></extensions></trkpt>'
    '<trkpt lat="47.3" lon="11.3"><extensions><wanderer:anchor></wanderer:anchor></extensions></trkpt>'
    '<trkpt lat="47.4" lon="11.4"/>'
    '<trkpt lat="47.0" lon="11.0"/>'
    '</trkseg></trk></gpx>';

List<List<List<double>>> _coordinates(Gpx gpx) => [
  for (final seg in gpx.trks.single.trksegs)
    [
      for (final p in seg.trkpts) [p.lat!, p.lon!],
    ],
];

/// The planner's in-memory layout: one Trkseg per leg, the anchor opening
/// the next leg. Leg 1 is a 1-point interior leg.
Gpx _plannedLoop() => Gpx()
  ..creator = 'wanderer'
  ..trks = [
    Trk(
      trksegs: [
        Trkseg(
          trkpts: [
            Wpt(
              lat: 47.0,
              lon: 11.0,
              ele: 500,
              time: DateTime.utc(2026, 10, 14, 8),
            ),
            Wpt(lat: 47.1, lon: 11.1, ele: 510),
          ],
        ),
        Trkseg(trkpts: [Wpt(lat: 47.2, lon: 11.2, ele: 520)]),
        Trkseg(
          trkpts: [
            Wpt(lat: 47.3, lon: 11.3, ele: 530),
            Wpt(lat: 47.4, lon: 11.4, ele: 540),
            Wpt(
              lat: 47.0,
              lon: 11.0,
              ele: 500,
              time: DateTime.utc(2026, 10, 14, 9),
            ),
          ],
        ),
      ],
    ),
  ];

Gpx _legs(List<List<List<double>>> legs) => Gpx()
  ..trks = [
    Trk(
      trksegs: [
        for (final leg in legs)
          Trkseg(
            trkpts: [for (final c in leg) Wpt(lat: c[0], lon: c[1])],
          ),
      ],
    ),
  ];

/// Flattened indices of the points carrying the anchor mark, read WITHOUT
/// the split so the marks are still there.
List<int> _markedIndices(String xml) {
  final points = [
    for (final seg in GpxReader().fromString(xml).trks.single.trksegs)
      ...seg.trkpts,
  ];
  return [
    for (var i = 0; i < points.length; i++)
      if (points[i].extensions.containsKey(routeAnchorExtension)) i,
  ];
}

List<Wpt> _points(Gpx gpx) => [
  for (final seg in gpx.trks.single.trksegs) ...seg.trkpts,
];

List<List<double>> _flat(Gpx gpx) => [
  for (final seg in gpx.trks.single.trksegs)
    for (final p in seg.trkpts) [p.lat!, p.lon!],
];

List<List<double>> _latLon(Iterable<dynamic> geo) => [
  for (final g in geo) [g.lat as double, g.lon as double],
];

int _count(String haystack, String needle) =>
    needle.allMatches(haystack).length;

void main() {
  group('parseGpxSafely with route anchor marks', () {
    test('restores the legs of a joined web export', () {
      final gpx = parseGpxSafely(_joinedLoop);

      expect(_coordinates(gpx), [
        [
          [47.0, 11.0],
          [47.1, 11.1],
        ],
        [
          [47.2, 11.2],
        ],
        [
          [47.3, 11.3],
          [47.4, 11.4],
          [47.0, 11.0],
        ],
      ]);
    });

    test('recovers every drawn anchor for the route planner', () {
      final gpx = parseGpxSafely(_joinedLoop);
      final anchors = anchorsFromTrack(gpx);

      expect(
        [
          for (final a in anchors) [a.lat, a.lon],
        ],
        [
          [47.0, 11.0],
          [47.2, 11.2],
          [47.3, 11.3],
          [47.0, 11.0],
        ],
      );
      expect(segmentPolylinesFromTrack(gpx, anchors).map((l) => l.length), [
        3,
        2,
        3,
      ]);
    });

    test('keeps the flattened point stream identical to the file', () {
      final gpx = parseGpxSafely(_joinedLoop);

      expect(
        [
          for (final p in gpx.allPoints) [p.lat, p.lon],
        ],
        [
          [47.0, 11.0],
          [47.1, 11.1],
          [47.2, 11.2],
          [47.3, 11.3],
          [47.4, 11.4],
          [47.0, 11.0],
        ],
      );
    });

    test('strips the marks so GpxWriter output stays well-formed', () {
      final xml = GpxWriter().asString(parseGpxSafely(_joinedLoop));

      expect(xml, isNot(contains('wanderer:')));
      expect(() => parseGpxSafely(xml), returnsNormally);
    });

    test('keeps other extensions on a marked point', () {
      final gpx = parseGpxSafely(
        '<gpx><trk><trkseg>'
        '<trkpt lat="1" lon="1"/>'
        '<trkpt lat="2" lon="2"><extensions><ns3:hr>90</ns3:hr><wanderer:anchor/></extensions></trkpt>'
        '<trkpt lat="3" lon="3"/>'
        '</trkseg></trk></gpx>',
      );

      expect(gpx.trks.single.trksegs, hasLength(2));
      expect(gpx.trks.single.trksegs[1].trkpts.first.extensions, {
        'ns3:hr': '90',
      });
    });

    test('ignores marks on the ends of a segment but still strips them', () {
      final gpx = parseGpxSafely(
        '<gpx><trk><trkseg>'
        '<trkpt lat="1" lon="1"><extensions><wanderer:anchor/></extensions></trkpt>'
        '<trkpt lat="2" lon="2"/>'
        '<trkpt lat="3" lon="3"><extensions><wanderer:anchor/></extensions></trkpt>'
        '</trkseg></trk></gpx>',
      );

      expect(_coordinates(gpx), [
        [
          [1.0, 1.0],
          [2.0, 2.0],
          [3.0, 3.0],
        ],
      ]);
      expect(
        gpx.trks.single.trksegs.single.trkpts.every(
          (p) => p.extensions.isEmpty,
        ),
        isTrue,
      );
    });

    test('leaves an unmarked multi-segment track unchanged', () {
      final gpx = parseGpxSafely(
        '<gpx><trk>'
        '<trkseg><trkpt lat="1" lon="1"/><trkpt lat="2" lon="2"/></trkseg>'
        '<trkseg><trkpt lat="2" lon="2"/><trkpt lat="3" lon="3"/></trkseg>'
        '</trk></gpx>',
      );

      expect(_coordinates(gpx), [
        [
          [1.0, 1.0],
          [2.0, 2.0],
        ],
        [
          [2.0, 2.0],
          [3.0, 3.0],
        ],
      ]);
    });
  });
  group('serializePlannedRoute', () {
    test('stores one marked segment under the wanderer namespace', () {
      final xml = serializePlannedRoute(_plannedLoop());

      expect(_count(xml, '<trkseg>'), 1);
      expect(_count(xml, '<wanderer:anchor'), 2);
      expect(xml, contains('xmlns:wanderer="$routeAnchorNamespace"'));
      expect(routeAnchorNamespace, 'https://wanderer.to/xmlns/gpx/1');
      expect(
        GpxReader().fromString(xml).trks.single.trksegs.single.trkpts,
        hasLength(6),
      );
      expect(_markedIndices(xml), [2, 3]);
    });

    test('round-trips through parseGpxSafely, ele and time included', () {
      final input = _plannedLoop();
      final parsed = parseGpxSafely(serializePlannedRoute(input));

      expect(parsed.trks.single.trksegs, input.trks.single.trksegs);
      expect(
        [for (final p in _points(parsed)) p.ele],
        [500, 510, 520, 530, 540, 500],
      );
      final pts = _points(parsed).toList();
      expect(pts.first.time, DateTime.utc(2026, 10, 14, 8));
      expect(pts.last.time, DateTime.utc(2026, 10, 14, 9));
      expect(pts.every((p) => p.extensions.isEmpty), isTrue);

      final anchors = anchorsFromTrack(parsed);
      expect(_latLon(anchors), _latLon(anchorsFromTrack(input)));
      expect(
        [
          for (final l in segmentPolylinesFromTrack(parsed, anchors))
            _latLon(l),
        ],
        [
          for (final l in segmentPolylinesFromTrack(
            input,
            anchorsFromTrack(input),
          ))
            _latLon(l),
        ],
      );
    });

    test('never mutates the input', () {
      final input = _plannedLoop();
      final before = GpxWriter().asString(input);
      final wpts = [for (final s in input.trks.single.trksegs) ...s.trkpts];
      final maps = [for (final p in wpts) p.extensions];

      serializePlannedRoute(input);

      expect(GpxWriter().asString(input), before);
      expect(input.trks.single.trksegs, hasLength(3));
      final after = [for (final s in input.trks.single.trksegs) ...s.trkpts];
      for (var i = 0; i < wpts.length; i++) {
        expect(identical(after[i], wpts[i]), isTrue);
        expect(identical(after[i].extensions, maps[i]), isTrue);
        expect(after[i].extensions, isEmpty);
      }
    });

    test('round-trips a 1-point interior leg', () {
      final input = _legs([
        [
          [1, 1],
          [2, 2],
        ],
        [
          [3, 3],
        ],
        [
          [3, 3],
          [4, 4],
          [5, 5],
        ],
      ]);
      final xml = serializePlannedRoute(input);

      expect(_markedIndices(xml), [2, 3]);
      expect(
        parseGpxSafely(xml).trks.single.trksegs,
        input.trks.single.trksegs,
      );
    });

    test('writes plain output when there is nothing to mark', () {
      final single = _legs([
        [
          [1, 1],
          [2, 2],
        ],
      ]);
      for (final gpx in [single, Gpx()]) {
        final xml = serializePlannedRoute(gpx);
        expect(xml, GpxWriter().asString(gpx));
        expect(xml, isNot(contains('xmlns:wanderer')));
      }
    });

    test('skips empty legs', () {
      final input = Gpx()
        ..trks = [
          Trk(
            trksegs: [
              Trkseg(),
              Trkseg(trkpts: [Wpt(lat: 1, lon: 1), Wpt(lat: 2, lon: 2)]),
              Trkseg(),
              Trkseg(trkpts: [Wpt(lat: 3, lon: 3), Wpt(lat: 4, lon: 4)]),
            ],
          ),
        ];
      final xml = serializePlannedRoute(input);

      expect(_count(xml, '<trkseg>'), 1);
      expect(_markedIndices(xml), [2]);
      expect(_coordinates(parseGpxSafely(xml)), [
        [
          [1.0, 1.0],
          [2.0, 2.0],
        ],
        [
          [3.0, 3.0],
          [4.0, 4.0],
        ],
      ]);
    });

    test('re-storing a web-saved route reproduces the web form', () {
      final restored = serializePlannedRoute(parseGpxSafely(_joinedLoop));

      expect(
        _flat(GpxReader().fromString(restored)),
        _flat(GpxReader().fromString(_joinedLoop)),
      );
      expect(_markedIndices(restored), [2, 3]);
      expect(_markedIndices(_joinedLoop), [2, 3]);
    });
  });
}
