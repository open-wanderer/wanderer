import 'package:gpx/gpx.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/util/gpx/conversion.dart' show haversineMeters;

// sanitizeGpxEmail lived here. The non-standard
// `<email>user@example.com</email>` text form is now handled where the
// coercion happens, in `lib/vendor/gpx/gpx_reader.dart`'s `_readEmail`
// (LOCAL MODIFICATION 4), so no pre-parse XML rewriting is needed. Parse
// third-party GPX with `parseGpxSafely` from `util/gpx/conversion.dart`.

/// Builds a Valhalla shape list from [points], downsampling to ≤500 entries
/// while always preserving the first and last point.
///
/// Shared by [launchNavigation] (online path) and [downloadTrail]
/// (cache-write path) so the two paths can never diverge on shape sampling.
///
/// When `points.length > 500`:
///   - `step = (points.length / 499).ceil()` → at most 499 regularly-sampled
///     entries, so appending the last point never exceeds 500 total.
///   - The last point is appended only if it is not already the last sampled
///     entry (dedup).
///
/// When `points.length ≤ 500`: all points are mapped without change.
List<Map<String, double>> buildNavShape(List<Geographic> points) {
  if (points.length > 500) {
    final step = (points.length / 499).ceil();
    final sampled = <Map<String, double>>[];
    for (int i = 0; i < points.length; i++) {
      if (i % step == 0) {
        sampled.add({'lat': points[i].lat, 'lon': points[i].lon});
      }
    }
    // Always include the last point, deduplicated by coordinate value (Map
    // equality here is identity-based, not structural).
    final lastLat = points.last.lat;
    final lastLon = points.last.lon;
    if (sampled.isEmpty ||
        sampled.last['lat'] != lastLat ||
        sampled.last['lon'] != lastLon) {
      sampled.add({'lat': lastLat, 'lon': lastLon});
    }
    return sampled;
  } else {
    return points.map((p) => {'lat': p.lat, 'lon': p.lon}).toList();
  }
}

Gpx buildGpxFromPoints(List<Wpt> points) {
  final gpx = Gpx();
  if (points.isEmpty) return gpx;
  gpx.trks = [
    Trk(trksegs: [Trkseg(trkpts: points)]),
  ];
  return gpx;
}

// D-17: trail metrics live in util/gpx/conversion.dart's
// computeTrailMetrics/GpxTrailMetrics — the port of the Phase 33-corrected
// TS algorithm. No second metrics implementation (a "totals"-style method or
// value class, or anything computing distance/elevation summaries) may be
// re-added to this extension: this file previously carried a second,
// CONV-01-buggy metrics implementation that disagreed with the ported one,
// so an unsaved-GPX preview showed one distance and the saved trail another
// for the same GPX (T-34-19). Both former consumers (elevation_profile.dart,
// trail_panel.dart) now call computeTrailMetrics.
extension GpxMappingUtils on Gpx {
  List<Wpt> get allWaypoints {
    return trks
        .expand((track) => track.trksegs)
        .expand((seg) => seg.trkpts)
        .where((pt) => pt.lat != null && pt.lon != null)
        .toList();
  }

  List<Geographic> get allPoints {
    return allWaypoints
        .map((pt) => Geographic(lat: pt.lat!, lon: pt.lon!))
        .toList();
  }

  LngLatBounds? getBounds() {
    final points = allPoints;
    if (points.isEmpty) return null;
    return LngLatBounds.fromPoints(points);
  }

  /// Approximates how far along the track [point] falls, in meters, by
  /// snapping to the nearest track vertex and returning its cumulative
  /// distance from the start. This is a lightweight estimate (not a true
  /// point-to-segment projection) meant to give locally-created waypoints a
  /// `distanceFromStart` immediately, before the server recomputes the exact
  /// value.
  double? distanceFromStartTo(Geographic point) {
    final points = allWaypoints;
    if (points.length < 2) return null;

    double cumDist = 0;
    double bestDist = double.infinity;
    double bestCumDist = 0;

    for (int i = 0; i < points.length; i++) {
      final wpt = points[i];
      if (i > 0) {
        // Shared haversine, not an open-coded SphericalGreatCircle loop: the
        // cumulative distance this returns is compared against distances the
        // metrics engine produces, so the two must use one implementation.
        cumDist += haversineMeters(points[i - 1], wpt);
      }

      final pointCalculator = SphericalGreatCircle(
        Geographic(lat: wpt.lat!, lon: wpt.lon!),
      );
      final dist = pointCalculator.distanceTo(point);
      if (dist < bestDist) {
        bestDist = dist;
        bestCumDist = cumDist;
      }
    }

    return bestCumDist;
  }
}
