import 'package:gpx/gpx.dart';

import 'gpx_util.dart' show sanitizeGpxEmail;

/// Neutralises the four confirmed `GpxReader` crash inputs that the
/// corrected TS parser (via `parseElevation`/`Date` coercion) treats as
/// "no data": an empty-but-present `<ele></ele>`, a whitespace-only or
/// non-numeric `<ele>` body, and an empty `<time></time>`.
///
/// `GpxReader` (`package:gpx` 2.3.0) calls `double.parse`/`DateTime.parse`
/// directly on the accumulated element text with no empty/malformed guard
/// (`gpx_reader.dart`'s `_readDouble`/`_readDateTime`), so `<ele></ele>`
/// throws `FormatException: Invalid double` and `<time></time>` throws
/// `FormatException: Invalid date format`. This mirrors [sanitizeGpxEmail]'s
/// existing regex-rewrite precedent: rewrite the malformed body to a
/// self-closing tag (`<ele/>`, `<time/>`), which `GpxReader` already treats
/// as `null` (a self-closing start element short-circuits its `_readString`
/// helper to `null` before any `parse` call is reached).
///
/// A genuine `<ele>0</ele>` (real sea level) and a pretty-printed
/// `<ele>\n 1000.5\n</ele>` (`double.tryParse` trims surrounding whitespace)
/// both survive untouched — only a tag with no valid finite number left
/// after trimming is rewritten. The same trim-then-parse logic applies to
/// `<time>` via `DateTime.tryParse`.
String sanitizeGpxEleAndTime(String xml) {
  final withSanitizedEle = xml.replaceAllMapped(RegExp(r'<ele>([^<]*)</ele>'), (
    m,
  ) {
    final body = m[1] ?? '';
    final parsed = double.tryParse(body.trim());
    if (parsed != null && parsed.isFinite) {
      return m[0]!;
    }
    return '<ele/>';
  });

  return withSanitizedEle.replaceAllMapped(RegExp(r'<time>([^<]*)</time>'), (
    m,
  ) {
    final body = m[1] ?? '';
    if (DateTime.tryParse(body.trim()) != null) {
      return m[0]!;
    }
    return '<time/>';
  });
}

/// The single sanctioned parse entry point for any GPX this app did not
/// itself produce via [GpxWriter] — imported files, shared/received tracks,
/// or any other third-party GPX source.
///
/// Chains both pre-parse sanitize passes ([sanitizeGpxEmail],
/// [sanitizeGpxEleAndTime]) before handing the string to [GpxReader]. Later
/// plans redirect every existing `GpxReader().fromString(...)` call site in
/// the app through this function.
///
/// A `<trkpt>` missing its `lat`/`lon` attribute still throws `StateError`
/// from `GpxReader` (34-RESEARCH.md Pitfall 1) — this is a much rarer,
/// structurally broken input the GPX spec itself requires both attributes
/// for, and it is deliberately left to callers' existing try/catch-and-toast
/// paths rather than handled here via string surgery.
Gpx parseGpxSafely(String xml) {
  return GpxReader().fromString(sanitizeGpxEleAndTime(sanitizeGpxEmail(xml)));
}
