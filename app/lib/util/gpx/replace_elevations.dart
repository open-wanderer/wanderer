import 'package:xml/xml.dart';

/// Writes [heights] into the `<ele>` of every track point of [xml] and leaves
/// the rest of the document as it was: times, extensions (sensor data,
/// `wanderer:anchor` marks), names and segment layout all survive, unlike a
/// rebuild through `mergeHeightsIntoGpx`, which keeps only lat/lon/ele.
///
/// [heights] is index-aligned with `allWaypoints` of the parsed document: the
/// `<trkpt>`s of every `<trk>` in document order, skipping a point whose
/// `lat`/`lon` the vendored reader would drop (missing, unparseable or
/// non-finite). A count mismatch returns [xml] unchanged rather than shifting
/// heights onto the wrong points.
///
/// An existing `<ele>` is overwritten in place; a missing one is inserted as
/// the point's first child, where the GPX 1.1 schema requires it.
String replaceTrackElevations(String xml, List<num> heights) {
  final XmlDocument doc;
  try {
    doc = XmlDocument.parse(xml);
  } on XmlException {
    return xml;
  }
  final root = doc.findAllElements('gpx').firstOrNull;
  if (root == null) return xml;

  final points = [
    for (final trk in root.findElements('trk'))
      for (final seg in trk.findElements('trkseg'))
        for (final pt in seg.findElements('trkpt'))
          if (_coordinate(pt, 'lat') != null && _coordinate(pt, 'lon') != null)
            pt,
  ];
  if (points.length != heights.length) return xml;

  for (var i = 0; i < points.length; i++) {
    final value = heights[i].toString();
    final ele = points[i].getElement('ele');
    if (ele != null) {
      ele.innerText = value;
    } else {
      points[i].children.insert(
        0,
        XmlElement(XmlName('ele'), const [], [XmlText(value)]),
      );
    }
  }
  return doc.toXmlString();
}

/// [replaceTrackElevations] taking one message, so `compute()` can run it.
String replaceTrackElevationsMessage((String, List<num>) message) =>
    replaceTrackElevations(message.$1, message.$2);

double? _coordinate(XmlElement point, String name) {
  final raw = point.getAttribute(name);
  if (raw == null) return null;
  final parsed = double.tryParse(raw);
  return (parsed != null && parsed.isFinite) ? parsed : null;
}
