String formatDistance(double? meters, {String unit = 'metric'}) {
  if (meters == null) {
    return "-";
  }

  if (unit == "metric") {
    if (meters >= 1000) {
      return "${(meters / 1000).toStringAsFixed(2)} km";
    } else {
      return meters % 1 == 0 ? "${meters.toInt()} m" : "${meters.round()} m";
    }
  } else {
    const double milesConversion = 0.000621371;
    final miles = meters * milesConversion;
    final roundedMiles = miles.toStringAsFixed(2);

    return "$roundedMiles mi";
  }
}

String formatElevation(double? meters, {String unit = 'metric'}) {
  if (meters == null) {
    return "-";
  }

  if (unit == "metric") {
    return "${meters.round()} m";
  } else {
    final feet = meters * 3.28084;
    return "${feet.round()} ft";
  }
}

/// Formats a speed value (km/h) for display.
///
/// Returns "-" when [kmh] is null, NaN, or negative (Pitfall 3: guard invalid
/// GPS-derived speed). Metric → one-decimal "km/h"; imperial → converted to
/// mph (× 0.621371), one decimal, "mph" suffix.
String formatSpeed(double? kmh, {String unit = 'metric'}) {
  if (kmh == null || kmh.isNaN || kmh < 0) {
    return "-";
  }

  if (unit == "metric") {
    return "${kmh.toStringAsFixed(1)} km/h";
  } else {
    const double mphConversion = 0.621371;
    return "${(kmh * mphConversion).toStringAsFixed(1)} mph";
  }
}

/// Formats an elapsed [Duration] as a stopwatch string.
///
/// Hours are shown only when > 0 ("H:MM:SS"); otherwise "MM:SS". Minutes and
/// seconds are always zero-padded to two digits (CONTEXT format spec).
String formatElapsed(Duration d) {
  final h = d.inHours;
  final m = d.inMinutes.remainder(60);
  final s = d.inSeconds.remainder(60);
  final mm = m.toString().padLeft(2, '0');
  final ss = s.toString().padLeft(2, '0');
  return h > 0 ? "$h:$mm:$ss" : "$mm:$ss";
}
