import 'package:flutter_map/flutter_map.dart';
import 'package:gpx/gpx.dart';
import 'package:latlong2/latlong.dart';

extension GpxMappingUtils on Gpx {
  List<LatLng> get allPoints {
    return trks
        .expand((track) => track.trksegs)
        .expand((seg) => seg.trkpts)
        .where((pt) => pt.lat != null && pt.lon != null)
        .map((pt) => LatLng(pt.lat!, pt.lon!))
        .toList();
  }

  LatLngBounds? getBounds() {
    final points = allPoints;
    if (points.isEmpty) return null;
    return LatLngBounds.fromPoints(points);
  }
}
