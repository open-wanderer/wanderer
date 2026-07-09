import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';
import 'package:maplibre/maplibre.dart' as ml;

/// Temporary Geographic/LngLatBounds <-> LatLng/LatLngBounds adapters for
/// flutter_map call sites not yet migrated to MapLibre widgets (Phase 15+).
/// Each call site deletes its own usage of these as that screen migrates.

LatLng toLatLng(ml.Geographic g) => LatLng(g.lat, g.lon);

ml.Geographic toGeographic(LatLng ll) =>
    ml.Geographic(lat: ll.latitude, lon: ll.longitude);

LatLngBounds toLatLngBounds(ml.LngLatBounds b) => LatLngBounds(
  LatLng(b.latitudeSouth, b.longitudeWest),
  LatLng(b.latitudeNorth, b.longitudeEast),
);

List<LatLng> toLatLngList(List<ml.Geographic> points) =>
    points.map(toLatLng).toList();

ml.LngLatBounds toLngLatBounds(LatLngBounds b) => ml.LngLatBounds(
  longitudeWest: b.southWest.longitude,
  longitudeEast: b.northEast.longitude,
  latitudeSouth: b.southWest.latitude,
  latitudeNorth: b.northEast.latitude,
);
