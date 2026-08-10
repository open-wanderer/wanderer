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

// ---------------------------------------------------------------------------
// Rich text → plain text
//
// Port of `formatHTMLAsText`/`formatHTMLAsTextPreview` from
// `web/src/lib/util/format_util.ts`, kept string-for-string identical (the web
// tests in `format_util.test.ts` are mirrored in `test/util/format_test.dart`)
// so a description previewed on both platforms cuts at the same character.
//
// Deliberately not `package:html`: a parser would normalise whitespace and
// repair torn markup differently, and the two platforms would then disagree
// about what the first N characters of a description are.
// ---------------------------------------------------------------------------

const _blockTags =
    'address|article|aside|blockquote|div|figure|footer|h[1-6]|header|li|main|'
    'nav|ol|p|pre|section|table|td|th|tr|ul';

final _scriptStyleRegex = RegExp(
  r'<(script|style)\b[^>]*>[\s\S]*?</\1\s*>',
  caseSensitive: false,
);
final _brRegex = RegExp(r'<br\s*/?>', caseSensitive: false);
final _blockTagRegex = RegExp(
  '</?(?:$_blockTags)(?:\\s[^>]*)?/?>',
  caseSensitive: false,
);
// Attribute values may contain ">", so match quoted runs explicitly
final _tagRegex = RegExp('<(?:[^>"\']|"[^"]*"|\'[^\']*\')*>');
final _entityRegex = RegExp(r'&(#\d+|#[xX][0-9a-fA-F]+|[a-zA-Z][a-zA-Z0-9]*);');

final _crlfRegex = RegExp(r'\r\n?');
final _trailingSpaceRegex = RegExp(r'[ \t]+\n');
final _leadingSpaceRegex = RegExp(r'\n[ \t]+');
final _blankLinesRegex = RegExp(r'\n{3,}');
final _runOfSpacesRegex = RegExp(r'[ \t]{2,}');

const _namedEntities = <String, String>{
  'amp': '&',
  'apos': "'",
  'gt': '>',
  'lt': '<',
  'nbsp': ' ',
  'quot': '"',
};

/// Single pass, so a decoded "&amp;lt;" stays as the literal text "&lt;".
String _decodeEntities(String text) {
  return text.replaceAllMapped(_entityRegex, (match) {
    final entity = match[0]!;
    final body = match[1]!;
    if (body.startsWith('#')) {
      final isHex = body[1] == 'x' || body[1] == 'X';
      final codePoint = isHex
          ? int.tryParse(body.substring(2), radix: 16)
          : int.tryParse(body.substring(1));
      if (codePoint == null ||
          codePoint < 0 ||
          codePoint > 0x10ffff ||
          (codePoint >= 0xd800 && codePoint <= 0xdfff)) {
        return entity;
      }
      return String.fromCharCode(codePoint);
    }
    return _namedEntities[body.toLowerCase()] ?? entity;
  });
}

/// Converts rich text to plain text.
String formatHtmlAsText(String? html) {
  if (html == null || html.isEmpty) {
    return '';
  }

  return _decodeEntities(
        html
            .replaceAll(_scriptStyleRegex, '')
            .replaceAll(_brRegex, '\n')
            .replaceAll(_blockTagRegex, '\n')
            .replaceAll(_tagRegex, ''),
      )
      .replaceAll(_crlfRegex, '\n')
      .replaceAll(_trailingSpaceRegex, '\n')
      .replaceAll(_leadingSpaceRegex, '\n')
      .replaceAll(_blankLinesRegex, '\n\n')
      .replaceAll(_runOfSpacesRegex, '  ')
      .trim();
}

/// Plain-text preview of rich text, truncated to [maxLength] characters.
///
/// Truncating the HTML itself would tear tags in half (web #1128 — the same
/// bug the profile bio had). Counts code points via [String.runes] so the
/// cutoff never splits an astral character such as an emoji.
({String text, bool truncated}) formatHtmlAsTextPreview(
  String? html,
  int maxLength,
) {
  final text = formatHtmlAsText(html);
  final characters = text.runes.toList();

  if (characters.length <= maxLength) {
    return (text: text, truncated: false);
  }

  return (
    text: String.fromCharCodes(characters.take(maxLength)),
    truncated: true,
  );
}
