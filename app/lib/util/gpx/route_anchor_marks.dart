import 'package:gpx/gpx.dart';
import 'package:xml/xml.dart' show XmlDocument;

/// The namespace [routeAnchorExtension] lives in; identical to the web's
/// (`web/src/lib/util/gpx_segment_util.ts`).
const routeAnchorNamespace = 'https://wanderer.to/xmlns/gpx/1';

/// The mark on each interior leg boundary of a STORED drawn route. The web
/// writes it on save and export, the app through [serializePlannedRoute],
/// and both declare it under [routeAnchorNamespace]. `<wanderer:anchor/>`
/// reaches the vendored reader as the extension key `wanderer:anchor`.
const routeAnchorExtension = 'wanderer:anchor';

/// Splits every `trkseg` of [gpx] at points marked with
/// [routeAnchorExtension] and removes the marks, in place.
///
/// A drawn route is stored (by the web and by [serializePlannedRoute]) as
/// ONE continuous segment (Garmin shows a multi-segment track one segment at
/// a time) with each former leg boundary marked instead. In memory the
/// planner keeps one `Trkseg` per leg and recovers anchors from those
/// boundaries (`anchorsFromTrack`); this split, run inside `parseGpxSafely`,
/// restores that layout. Without it a stored route would collapse to a bare
/// start/end pair.
///
/// The legs use the app's own layout (`buildFinalPlannedGpx`): the anchor
/// point opens the next leg and is not repeated as the previous leg's last
/// point, so the flattened point stream — and every metric computed from it
/// — is identical to the joined file's.
///
/// The marks are always stripped, including on a segment's first or last
/// point (which never splits). Left in place, `GpxWriter` would write them
/// back without the `xmlns:wanderer` declaration, producing XML with an
/// unbound prefix.
void splitTrackSegmentsAtAnchors(Gpx gpx) {
  for (final track in gpx.trks) {
    final split = <Trkseg>[];

    for (final segment in track.trksegs) {
      final points = segment.trkpts;
      var start = 0;

      for (var i = 0; i < points.length; i++) {
        if (points[i].extensions.remove(routeAnchorExtension) == null ||
            i == 0 ||
            i == points.length - 1) {
          continue;
        }
        split.add(
          Trkseg(
            trkpts: points.sublist(start, i),
            extensions: Map.of(segment.extensions),
          ),
        );
        start = i;
      }

      if (start == 0) {
        split.add(segment);
      } else {
        split.add(
          Trkseg(
            trkpts: points.sublist(start),
            extensions: Map.of(segment.extensions),
          ),
        );
      }
    }

    track.trksegs = split;
  }
}

Wpt _copyWpt(Wpt p) => Wpt(
  lat: p.lat,
  lon: p.lon,
  ele: p.ele,
  time: p.time,
  magvar: p.magvar,
  geoidheight: p.geoidheight,
  name: p.name,
  cmt: p.cmt,
  desc: p.desc,
  src: p.src,
  links: List.of(p.links),
  sym: p.sym,
  type: p.type,
  fix: p.fix,
  sat: p.sat,
  hdop: p.hdop,
  vdop: p.vdop,
  pdop: p.pdop,
  ageofdgpsdata: p.ageofdgpsdata,
  dgpsid: p.dgpsid,
  extensions: Map.of(p.extensions),
);

/// Serializes the planner's per-leg [gpx] as it is stored: ONE `trkseg` per
/// `trk`, with [routeAnchorExtension] on the first point of every non-empty
/// leg after the first. The inverse of [splitTrackSegmentsAtAnchors].
///
/// For the planner's layout only; recordings and imports keep their own
/// segments, whose boundaries are real gaps. The input is never mutated: the
/// in-memory [Gpx] is reused by the edit-mode handoff and mergeRouteIntoTrail,
/// so marks go on copied points. The namespace is declared only when
/// something was marked, so a single-leg or trackless route is byte-identical
/// to the plain writer. Top-level so `compute()` can run it.
///
/// A trailing 1-point leg cannot be marked (readers ignore marks on a
/// segment's last point) and folds into the previous leg.
String serializePlannedRoute(Gpx gpx) {
  var marked = false;

  final copy = Gpx()
    ..version = gpx.version
    ..creator = gpx.creator
    ..metadata = gpx.metadata
    ..wpts = List.of(gpx.wpts)
    ..rtes = List.of(gpx.rtes)
    ..extensions = Map.of(gpx.extensions);

  for (final track in gpx.trks) {
    final joined = <Wpt>[];
    var firstLeg = true;

    for (final leg in track.trksegs) {
      if (leg.trkpts.isEmpty) continue;
      if (firstLeg) {
        joined.addAll(leg.trkpts);
        firstLeg = false;
        continue;
      }
      joined.add(
        _copyWpt(leg.trkpts.first)..extensions[routeAnchorExtension] = '',
      );
      joined.addAll(leg.trkpts.skip(1));
      marked = true;
    }

    copy.trks.add(
      Trk(
        name: track.name,
        cmt: track.cmt,
        desc: track.desc,
        src: track.src,
        links: List.of(track.links),
        number: track.number,
        type: track.type,
        extensions: Map.of(track.extensions),
        trksegs: joined.isEmpty ? [] : [Trkseg(trkpts: joined)],
      ),
    );
  }

  final doc = GpxWriter().asXml(copy) as XmlDocument;
  if (marked) {
    doc.rootElement.setAttribute('xmlns:wanderer', routeAnchorNamespace);
  }
  return doc.toXmlString();
}
