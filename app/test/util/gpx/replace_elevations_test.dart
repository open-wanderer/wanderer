import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/gpx/conversion.dart';
import 'package:wanderer/util/gpx/gpx.dart';
import 'package:wanderer/util/gpx/replace_elevations.dart';

/// A recorded track with per-point time, a sensor extension and a name.
const _recording =
    '<?xml version="1.0" encoding="UTF-8"?>'
    '<gpx version="1.1" creator="Garmin Connect" xmlns="http://www.topografix.com/GPX/1/1" '
    'xmlns:ns3="http://www.garmin.com/xmlschemas/TrackPointExtension/v1">'
    '<metadata><name>Morning hike</name></metadata>'
    '<trk><name>Track</name><trkseg>'
    '<trkpt lat="47.0" lon="11.0"><ele>500</ele><time>2026-10-01T08:00:00Z</time>'
    '<extensions><ns3:TrackPointExtension><ns3:hr>92</ns3:hr></ns3:TrackPointExtension></extensions></trkpt>'
    '<trkpt lat="47.1" lon="11.1"><time>2026-10-01T08:10:00Z</time><name>Bridge</name></trkpt>'
    '<trkpt lat="47.2" lon="11.2"><ele>520</ele><time>2026-10-01T08:20:00Z</time></trkpt>'
    '</trkseg></trk></gpx>';

/// A drawn route as the web stores it: one segment, B marked.
const _webRoute =
    '<?xml version="1.0" encoding="UTF-8"?>'
    '<gpx version="1.1" creator="wanderer" xmlns="http://www.topografix.com/GPX/1/1" '
    'xmlns:wanderer="https://wanderer.to/xmlns/gpx/1"><trk><trkseg>'
    '<trkpt lat="1" lon="1"><ele>10</ele></trkpt>'
    '<trkpt lat="2" lon="2"><ele>20</ele></trkpt>'
    '<trkpt lat="3" lon="3"><ele>30</ele><extensions><wanderer:anchor/></extensions></trkpt>'
    '<trkpt lat="4" lon="4"><ele>40</ele></trkpt>'
    '<trkpt lat="5" lon="5"><ele>50</ele></trkpt>'
    '</trkseg></trk></gpx>';

List<double?> _elevations(String xml) => [
  for (final p in parseGpxSafely(xml).allWaypoints) p.ele,
];

void main() {
  group('replaceTrackElevations', () {
    test('overwrites existing ele and inserts missing ones', () {
      final out = replaceTrackElevations(_recording, [601, 602.5, 603]);

      expect(_elevations(out), [601, 602.5, 603]);
      expect(out, contains('<trkpt lat="47.1" lon="11.1"><ele>602.5</ele>'));
    });

    test('leaves times, extensions and names untouched', () {
      final out = replaceTrackElevations(_recording, [601, 602, 603]);
      final points = parseGpxSafely(out).allWaypoints;

      expect(points.map((p) => p.time), [
        DateTime.utc(2026, 10, 1, 8),
        DateTime.utc(2026, 10, 1, 8, 10),
        DateTime.utc(2026, 10, 1, 8, 20),
      ]);
      expect(points[1].name, 'Bridge');
      expect(out, contains('<ns3:hr>92</ns3:hr>'));
    });

    test('changes nothing in the text but the ele values', () {
      final out = replaceTrackElevations(_recording, [601, 602, 603]);

      expect(
        out,
        _recording
            .replaceFirst('<ele>500</ele>', '<ele>601</ele>')
            .replaceFirst(
              '<trkpt lat="47.1" lon="11.1">',
              '<trkpt lat="47.1" lon="11.1"><ele>602</ele>',
            )
            .replaceFirst('<ele>520</ele>', '<ele>603</ele>'),
      );
    });

    test('keeps route anchor marks, so the legs survive', () {
      final out = replaceTrackElevations(_webRoute, [11, 21, 31, 41, 51]);

      expect(out, contains('xmlns:wanderer="https://wanderer.to/xmlns/gpx/1"'));
      expect(out, contains('<wanderer:anchor/>'));
      expect(parseGpxSafely(out).trks.single.trksegs, hasLength(2));
      expect(_elevations(out), [11, 21, 31, 41, 51]);
    });

    test('skips points the reader drops, keeping heights aligned', () {
      const xml =
          '<gpx><trk><trkseg>'
          '<trkpt lat="1" lon="1"/>'
          '<trkpt lat="NaN" lon="2"/>'
          '<trkpt lon="3"/>'
          '<trkpt lat="4" lon="4"/>'
          '</trkseg></trk></gpx>';

      final out = replaceTrackElevations(xml, [100, 400]);

      expect(_elevations(out), [100, 400]);
      expect(out, contains('<trkpt lat="NaN" lon="2"/>'));
    });

    test('walks every track and segment in document order', () {
      const xml =
          '<gpx><trk><trkseg><trkpt lat="1" lon="1"/></trkseg>'
          '<trkseg><trkpt lat="2" lon="2"/></trkseg></trk>'
          '<trk><trkseg><trkpt lat="3" lon="3"/></trkseg></trk></gpx>';

      expect(_elevations(replaceTrackElevations(xml, [1, 2, 3])), [1, 2, 3]);
    });

    test('leaves the file unchanged when the counts disagree', () {
      expect(replaceTrackElevations(_webRoute, [1, 2]), _webRoute);
    });

    test('leaves unparseable input unchanged', () {
      expect(replaceTrackElevations('<gpx><trk>', [1]), '<gpx><trk>');
      expect(replaceTrackElevations('<kml/>', [1]), '<kml/>');
    });
  });
}
