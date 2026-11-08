# Vendored `GpxReader` (from `package:gpx` 2.3.0)

`gpx_reader.dart` and `gpx_tag.dart` are copied from
[`package:gpx`](https://github.com/kb0/dart-gpx/) 2.3.0
(`lib/src/gpx_reader.dart`, `lib/src/model/gpx_tag.dart`) and **modified**.

Upstream is licensed **Apache License 2.0** — see `LICENSE` in this directory,
retained per §4(a). Apache 2.0 §4(b) requires modified files to carry prominent
notices stating that they changed; `gpx_reader.dart`'s header does that, listing
every local modification. Keep that list current when re-syncing upstream.

## Why

Upstream's reader coerces eagerly with the **throwing** `double.parse`,
`int.parse` and `DateTime.parse`, and reads required-by-spec attributes with a
bare `firstWhere`. Ordinary exporter output — `<hdop></hdop>`,
`<pdop>N/A</pdop>`, `<time></time>`, `<email>user@example.com</email>`, a
`<trkpt>` missing `lat`/`lon` — therefore aborts the entire document. Wanderer
previously worked around this by rewriting the XML with regexes before parsing,
which could only chase symptoms one tag at a time and risked corrupting CDATA.

Only the **reader** is vendored. Models (`Gpx`, `Wpt`, `Trk`, `Trkseg`, …) and
`GpxWriter` still come from the published package, so types remain shared with
the rest of the app — this replaces one class, it does not fork the library.
`gpx_tag.dart` is vendored alongside solely because upstream does not export it.

## Re-syncing

```
diff <(cat ~/.pub-cache/hosted/pub.dev/gpx-<ver>/lib/src/gpx_reader.dart) gpx_reader.dart
```

Expect exactly the modifications listed in `gpx_reader.dart`'s header, plus
`dart format` churn. Anything else is drift and should be reconciled.

Upstream fix worth tracking: the `parse` → `tryParse` change (modification 1) is
a bug in its own right — `<hdop/>` parses fine while `<hdop></hdop>` crashes,
and the `!= null` guard plus `double?` return type show null was the intent. If
that lands upstream, this vendoring shrinks or goes away.
