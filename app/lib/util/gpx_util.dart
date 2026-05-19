import 'package:flutter_map/flutter_map.dart';
import 'package:gpx/gpx.dart';
import 'package:latlong2/latlong.dart';

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
