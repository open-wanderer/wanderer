import 'dart:math' as math;

import 'package:maplibre/maplibre.dart';

/// A utility to encode and decode Google Encoded Polyline strings
/// natively using MapLibre's [Geographic] positions.
class PolylineUtil {
  /// Encodes a list of [Geographic] points into a polyline string.
  ///
  /// [precision] controls the coordinate scaling factor (10^precision).
  /// Defaults to 5, matching this app's existing trail GPX-derived
  /// polylines. Valhalla's `/route` endpoint encodes at precision 6.
  static String encode(List<Geographic> points, {int precision = 5}) {
    final num factor = math.pow(10, precision);
    final StringBuffer encoded = StringBuffer();

    int lastLat = 0;
    int lastLng = 0;

    for (final point in points) {
      // Geographic uses .lat and .lon
      final int lat = (point.lat * factor).round();
      final int lng = (point.lon * factor).round();

      final int dLat = lat - lastLat;
      final int dLng = lng - lastLng;

      encoded.write(_encodeValue(dLat));
      encoded.write(_encodeValue(dLng));

      lastLat = lat;
      lastLng = lng;
    }

    return encoded.toString();
  }

  /// Decodes an encoded polyline string into a list of MapLibre [Geographic] points.
  ///
  /// [precision] controls the coordinate scaling factor (10^precision).
  /// Defaults to 5, matching this app's existing trail GPX-derived
  /// polylines. Valhalla's `/route` endpoint encodes at precision 6 —
  /// callers decoding a Valhalla shape MUST pass `precision: 6`.
  static List<Geographic> decode(String polyline, {int precision = 5}) {
    final num factor = math.pow(10, precision);
    final List<Geographic> points = [];
    int index = 0;
    final int len = polyline.length;
    int lat = 0;
    int lng = 0;

    while (index < len) {
      int b;
      int shift = 0;
      int result = 0;

      do {
        b = polyline.codeUnitAt(index++) - 63;
        result |= (b & 0x1F) << shift;
        shift += 5;
      } while (b >= 0x20);

      final int dlat = ((result & 1) != 0 ? ~(result >> 1) : (result >> 1));
      lat += dlat;

      shift = 0;
      result = 0;

      do {
        b = polyline.codeUnitAt(index++) - 63;
        result |= (b & 0x1F) << shift;
        shift += 5;
      } while (b >= 0x20);

      final int dlng = ((result & 1) != 0 ? ~(result >> 1) : (result >> 1));
      lng += dlng;

      // Natively populating MapLibre's Geographic positions
      points.add(Geographic(lat: lat / factor, lon: lng / factor));
    }

    return points;
  }

  static String _encodeValue(int value) {
    value = value < 0 ? ~(value << 1) : (value << 1);
    final StringBuffer encoded = StringBuffer();

    while (value >= 0x20) {
      encoded.writeCharCode((0x20 | (value & 0x1f)) + 63);
      value >>= 5;
    }
    encoded.writeCharCode(value + 63);

    return encoded.toString();
  }
}
