import 'dart:io';

import 'package:exif/exif.dart';

/// Reads GPS coordinates from an image file's EXIF data.
///
/// Mirrors the web client's `getCoordinatesFromPhoto` + `convertDMSToDD`
/// (`web/src/lib/models/gpx/utils.ts`): GPS latitude/longitude are stored as
/// degrees/minutes/seconds rationals plus a N/S/E/W reference. Returns null when
/// the file has no GPS EXIF or can't be parsed.
Future<({double lat, double lon})?> readGpsFromImage(String path) async {
  final bytes = await File(path).readAsBytes();
  final tags = await readExifFromBytes(bytes);

  final lat = _dmsToDecimal(
    tags['GPS GPSLatitude'],
    tags['GPS GPSLatitudeRef'],
  );
  final lon = _dmsToDecimal(
    tags['GPS GPSLongitude'],
    tags['GPS GPSLongitudeRef'],
  );

  if (lat == null || lon == null) return null;
  return (lat: lat, lon: lon);
}

/// Converts a [degrees, minutes, seconds] rational triple + hemisphere ref into
/// decimal degrees. Returns null if the tags are missing or malformed.
double? _dmsToDecimal(IfdTag? dms, IfdTag? ref) {
  if (dms == null || ref == null) return null;

  final parts = dms.values.toList();
  if (parts.length < 3) return null;

  final degrees = parts[0];
  final minutes = parts[1];
  final seconds = parts[2];
  if (degrees is! Ratio || minutes is! Ratio || seconds is! Ratio) return null;

  var dd =
      degrees.toDouble() + minutes.toDouble() / 60 + seconds.toDouble() / 3600;

  final direction = ref.printable.trim().toUpperCase();
  if (direction == 'S' || direction == 'W') dd = -dd;

  return dd;
}
