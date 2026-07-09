import 'package:gpx/gpx.dart';
import 'package:maplibre/maplibre.dart';

/// Derives the Valhalla costing string from a trail category name.
///
/// Returns `'bicycle'` when [category] (case-insensitive) contains `'bike'`,
/// `'cycling'`, or `'bicycle'`; otherwise returns `'pedestrian'`.
///
/// Shared by [launchNavigation] (online path) and [downloadTrail]
/// (cache-write path) so the two costing derivations can never diverge.
/// GPX 1.1 requires `<email id="user" domain="example.com"/>` but some files
/// use the non-standard text form `<email>user@example.com</email>`, which
/// crashes `GpxReader` with `Bad state: No element`. Rewrites the latter into
/// the attribute form expected by the `gpx` package before parsing.
String sanitizeGpxEmail(String xml) {
  return xml.replaceAllMapped(
    RegExp(r'<email>([^@<]+)@([^<]+)</email>'),
    (m) => '<email id="${m[1]}" domain="${m[2]}"/>',
  );
}

String costingForCategory(String? category) {
  final lower = (category ?? '').toLowerCase();
  if (lower.contains('bike') ||
      lower.contains('cycling') ||
      lower.contains('bicycle')) {
    return 'bicycle';
  }
  return 'pedestrian';
}

/// Builds a Valhalla shape list from [points], downsampling to ≤500 entries
/// while always preserving the first and last point (OFFLINE-01 / D-08).
///
/// Shared by [launchNavigation] (online path, plan 05-03) and
/// [downloadTrail] (cache-write path, plan 05-04) so the two paths can never
/// diverge on shape sampling.
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
    // Always include the last point; deduplicate if already present.
    // Note: Map<String, double> uses identity (not structural) equality with !=,
    // so compare coordinate values directly to detect whether the last sampled
    // entry already corresponds to the last input point.
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

class GpxStats {
  final double totalDistance;
  final double totalDuration;
  final double totalElevationGain;
  final double totalElevationloss;

  const GpxStats({
    required this.totalDistance,
    required this.totalDuration,
    required this.totalElevationGain,
    required this.totalElevationloss,
  });
}

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

  GpxStats getTotals() {
    final points = allWaypoints;

    double totalDistance = 0;
    double totalElevationGain = 0;
    double totalElevationLoss = 0;
    Duration totalDuration = Duration.zero;

    for (int i = 1; i < points.length; i++) {
      final prev = points[i - 1];
      final curr = points[i];

      final calculator = SphericalGreatCircle(
        Geographic(lat: prev.lat!, lon: prev.lon!),
      );
      totalDistance += calculator.distanceTo(
        Geographic(lat: curr.lat!, lon: curr.lon!),
      );

      if (prev.ele != null && curr.ele != null) {
        final diff = curr.ele! - prev.ele!;
        if (diff > 0) {
          totalElevationGain += diff;
        } else {
          totalElevationLoss += diff.abs();
        }
      }

      if (prev.time != null && curr.time != null) {
        totalDuration += curr.time!.difference(prev.time!);
      }
    }

    return GpxStats(
      totalDistance: totalDistance,
      totalDuration: totalDuration.inSeconds.toDouble(),
      totalElevationGain: totalElevationGain,
      totalElevationloss: totalElevationLoss,
    );
  }
}
