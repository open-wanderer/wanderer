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
