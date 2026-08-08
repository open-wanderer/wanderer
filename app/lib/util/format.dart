import 'package:intl/intl.dart';
import 'package:wanderer/models/trail_summary.dart';

/// Memoized [DateFormat]s keyed by locale. Constructing a `DateFormat`
/// resolves the locale's full pattern data — doing that inside every list
/// item's `build()` (per card, per rebuild, during scroll) was pure waste;
/// the set of live locales is one, or two around a locale switch.
final Map<String, DateFormat> _yMMMMdCache = {};
final Map<String, DateFormat> _yMMMdCache = {};

DateFormat dateFormatYMMMMd(String locale) =>
    _yMMMMdCache.putIfAbsent(locale, () => DateFormat.yMMMMd(locale));

DateFormat dateFormatYMMMd(String locale) =>
    _yMMMdCache.putIfAbsent(locale, () => DateFormat.yMMMd(locale));

/// The single app-side display rule: show `movingDuration` when it is
/// present and positive, otherwise fall back to `duration`. A zero moving
/// time is not treated as a value (matches
/// `web/src/lib/util/format_util.ts`'s `trailDisplayDuration`, so the two
/// platforms never disagree about which value is shown).
///
/// [TrailSummary] rather than [Trail] so this compiles against every call
/// site: `trail_card.dart`/`trail_list_item.dart` hold a `TrailSummary`
/// (which may be a search-result summary with no moving-time concept, hence
/// `TrailSummary.movingDuration` defaulting to null), while `trail_panel.dart`
/// holds a `Trail` (a `TrailSummary` subtype). Nothing may write
/// `movingDuration` from a GPX recompute — it is a session-only field.
double? trailDisplayDuration(TrailSummary trail) {
  final movingDuration = trail.movingDuration;
  if (movingDuration != null && movingDuration > 0) {
    return movingDuration;
  }
  return trail.duration;
}

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

const int _kb = 1024;
const int _mb = _kb * 1024;
const int _gb = _mb * 1024;

/// Formats [bytes] as a human-readable string using 1024-based unit steps.
///
/// Human-readable byte formatting for the region tile repository's disk
/// usage displays.
///
/// Convention per `24-UI-SPEC.md`: one decimal place, unit steps at
/// KB/MB/GB (e.g. "45 MB", "2.4 GB"); a bare `'$bytes B'` below 1 KB (no
/// decimal -- a byte count is already an exact integer).
String formatBytes(int bytes) {
  if (bytes >= _gb) return '${(bytes / _gb).toStringAsFixed(1)} GB';
  if (bytes >= _mb) return '${(bytes / _mb).toStringAsFixed(1)} MB';
  if (bytes >= _kb) return '${(bytes / _kb).toStringAsFixed(1)} KB';
  return '$bytes B';
}
