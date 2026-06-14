import 'package:flutter_map/flutter_map.dart';
import 'package:gpx/gpx.dart';
import 'package:latlong2/latlong.dart';

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
List<Map<String, double>> buildNavShape(List<LatLng> points) {
  if (points.length > 500) {
    final step = (points.length / 499).ceil();
    final sampled = <Map<String, double>>[];
    for (int i = 0; i < points.length; i++) {
      if (i % step == 0) {
        sampled.add({'lat': points[i].latitude, 'lon': points[i].longitude});
      }
    }
    // Always include the last point; deduplicate if already present.
    final lastPoint = {
      'lat': points.last.latitude,
      'lon': points.last.longitude,
    };
    if (sampled.isEmpty || sampled.last != lastPoint) {
      sampled.add(lastPoint);
    }
    return sampled;
  } else {
    return points
        .map((p) => {'lat': p.latitude, 'lon': p.longitude})
        .toList();
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

  List<LatLng> get allPoints {
    return allWaypoints.map((pt) => LatLng(pt.lat!, pt.lon!)).toList();
  }

  LatLngBounds? getBounds() {
    final points = allPoints;
    if (points.isEmpty) return null;
    return LatLngBounds.fromPoints(points);
  }

  GpxStats getTotals() {
    final points = allWaypoints;

    double totalDistance = 0;
    double totalElevationGain = 0;
    double totalElevationLoss = 0;
    Duration totalDuration = Duration.zero;

    const distanceCalc = Distance();

    for (int i = 1; i < points.length; i++) {
      final prev = points[i - 1];
      final curr = points[i];

      totalDistance += distanceCalc(
        LatLng(prev.lat!, prev.lon!),
        LatLng(curr.lat!, curr.lon!),
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
