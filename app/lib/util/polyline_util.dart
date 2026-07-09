import 'package:maplibre/maplibre.dart';

/// A utility to encode and decode Google Encoded Polyline strings
/// natively using MapLibre's [Geographic] positions.
class PolylineUtil {
  /// Encodes a list of [Geographic] points into a polyline string.
  static String encode(List<Geographic> points) {
    final StringBuffer encoded = StringBuffer();

    int lastLat = 0;
    int lastLng = 0;

    for (final point in points) {
      // Geographic uses .lat and .lon
      final int lat = (point.lat * 1E5).round();
      final int lng = (point.lon * 1E5).round();

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
  static List<Geographic> decode(String polyline) {
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
      points.add(Geographic(lat: lat / 1E5, lon: lng / 1E5));
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
