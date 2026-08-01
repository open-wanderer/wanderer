// Vendored from `package:gpx` 2.3.0 (`lib/src/gpx_reader.dart`).
// Upstream: https://github.com/kb0/dart-gpx — Apache License 2.0.
// The upstream LICENSE is retained beside this file (Apache 2.0 §4(a)); the
// LOCAL MODIFICATIONS list below is the §4(b) notice of changes. See README.md.
//
// WHY THIS IS VENDORED
//
// Upstream's reader coerces eagerly with the THROWING `double.parse`,
// `int.parse` and `DateTime.parse`, and reads required attributes with
// `firstWhere`. Real-world GPX from real exporters routinely contains
// `<hdop></hdop>`, `<pdop>N/A</pdop>`, `<time></time>` and the non-standard
// `<email>user@example.com</email>` text form, every one of which aborts the
// whole parse with a `FormatException` or `StateError`. The app previously
// worked around this by rewriting the XML with regexes before parsing, which
// could only ever chase symptoms one tag at a time — the fix belongs where the
// coercion happens.
//
// Only the MODELS are imported from the published package, so `Gpx`, `Wpt`,
// `Trk`, `Trkseg` etc. remain the exact same classes used everywhere else in
// the app and by `GpxWriter`. This file replaces the reader, it does not fork
// the library. `gpx_tag.dart` is vendored alongside only because upstream does
// not export it.
//
// LOCAL MODIFICATIONS (keep this list current when re-syncing upstream)
//
//  1. `_readDouble` / `_readInt` / `_readDateTime` use `tryParse`, so an
//     unparseable value becomes `null` ("no reading") instead of throwing.
//     Upstream's own `!= null` guard and `double?` return type show this was
//     the intent; `_readString` returning `''` rather than `null` for a
//     present-but-empty element is what defeated it. Note the upstream
//     asymmetry this produced: `<hdop/>` parsed fine while `<hdop></hdop>`
//     crashed.
//  2. `_readPoint` tolerates a `<trkpt>`/`<wpt>`/`<rtept>` with missing or
//     unparseable `lat`/`lon` instead of throwing `StateError`.
//  3. `_readBounds` tolerates missing/unparseable bounds attributes.
//  4. `_readEmail` accepts the non-standard `<email>user@example.com</email>`
//     text form in addition to the spec's `<email id="user" domain="..."/>`.
//  5. `_readCopyright` tolerates a `<copyright>` with no `author` attribute
//     (same `firstWhere`-with-no-orElse crash as 4, over a field the app
//     never reads).
//  6. `fromString` throws a documented `FormatException` when the document
//     contains no `<gpx>` element, instead of an opaque `_TypeError` from
//     `iterator.current as XmlStartElementEvent` (or a `StateError` from
//     `current` on empty input). Callers cannot reasonably catch `_TypeError`.
//
// Behaviour is otherwise byte-for-byte upstream. Coercion of `<ele>` to a
// trail metric still happens in `gpx_conversion_util.dart`'s
// `parseGpxElevation`, which is the layer the TS/Dart parity corpus pins.

import 'package:gpx/gpx.dart';
import 'package:xml/xml_events.dart';

import 'gpx_tag.dart';

/// Read Gpx from string.
///
/// Tolerant vendored replacement for `package:gpx`'s `GpxReader` — see the
/// file header. Reach for this through `parseGpxSafely` rather than
/// constructing it directly; a repo-guard test enforces the single call site.
class GpxReader {
  //  // @TODO
  //  Gpx fromStream(Stream<int> stream) {
  //
  //  }

  /// Parse xml string and create Gpx object.
  ///
  /// Throws [FormatException] when [xml] contains no `<gpx>` element.
  Gpx fromString(String xml) {
    final iterator = parseEvents(xml).iterator;

    XmlStartElementEvent? found;
    while (iterator.moveNext()) {
      final val = iterator.current;

      if (val is XmlStartElementEvent && val.name == GpxTagV11.gpx) {
        found = val;
        break;
      }
    }

    // LOCAL MODIFICATION 6 (see file header): a documented FormatException
    // instead of upstream's raw `iterator.current as XmlStartElementEvent`,
    // which threw an opaque `_TypeError` (and, on empty input, a
    // `StateError` from `current` before that). Callers already catch parse
    // failure and toast; what they could not do was distinguish it from a bug,
    // and `_TypeError` is not something a caller can reasonably name in a
    // catch clause.
    if (found == null) {
      throw const FormatException('No <gpx> element found in the document');
    }

    final gpxTag = found;
    final gpx = Gpx();

    gpx.version = gpxTag.attributes
        .firstWhere(
          (attr) => attr.name == GpxTagV11.version,
          orElse: () => XmlEventAttribute(
            GpxTagV11.version,
            '1.1',
            XmlAttributeType.DOUBLE_QUOTE,
          ),
        )
        .value;
    gpx.creator = gpxTag.attributes
        .firstWhere(
          (attr) => attr.name == GpxTagV11.creator,
          orElse: () => XmlEventAttribute(
            GpxTagV11.creator,
            'unknown',
            XmlAttributeType.DOUBLE_QUOTE,
          ),
        )
        .value;

    while (iterator.moveNext()) {
      final val = iterator.current;
      if (val is XmlEndElementEvent && val.name == GpxTagV11.gpx) {
        break;
      }

      if (val is XmlStartElementEvent) {
        switch (val.name) {
          case GpxTagV11.metadata:
            gpx.metadata = _parseMetadata(iterator);
            break;
          case GpxTagV11.wayPoint:
            gpx.wpts.add(_readPoint(iterator, val.name));
            break;
          case GpxTagV11.route:
            gpx.rtes.add(_parseRoute(iterator));
            break;
          case GpxTagV11.track:
            gpx.trks.add(_parseTrack(iterator));
            break;

          case GpxTagV11.extensions:
            gpx.extensions = _readExtensions(iterator);
            break;
        }
      }
    }

    return gpx;
  }

  Metadata _parseMetadata(Iterator<XmlEvent> iterator) {
    final metadata = Metadata();
    final elm = iterator.current;

    if ((elm is XmlStartElementEvent) && !elm.isSelfClosing) {
      while (iterator.moveNext()) {
        final val = iterator.current;

        if (val is XmlStartElementEvent) {
          switch (val.name) {
            case GpxTagV11.name:
              metadata.name = _readString(iterator, val.name);
              break;
            case GpxTagV11.desc:
              metadata.desc = _readString(iterator, val.name);
              break;
            case GpxTagV11.author:
              metadata.author = _readPerson(iterator);
              break;
            case GpxTagV11.copyright:
              metadata.copyright = _readCopyright(iterator);
              break;
            case GpxTagV11.link:
              metadata.links.add(_readLink(iterator));
              break;
            case GpxTagV11.time:
              metadata.time = _readDateTime(iterator, val.name);
              break;
            case GpxTagV11.keywords:
              metadata.keywords = _readString(iterator, val.name);
              break;
            case GpxTagV11.bounds:
              metadata.bounds = _readBounds(iterator);
              break;
            case GpxTagV11.extensions:
              metadata.extensions = _readExtensions(iterator);
              break;
          }
        }

        if (val is XmlEndElementEvent && val.name == GpxTagV11.metadata) {
          break;
        }
      }
    }

    return metadata;
  }

  Rte _parseRoute(Iterator<XmlEvent> iterator) {
    final rte = Rte();
    final elm = iterator.current;

    if ((elm is XmlStartElementEvent) && !elm.isSelfClosing) {
      while (iterator.moveNext()) {
        final val = iterator.current;

        if (val is XmlStartElementEvent) {
          switch (val.name) {
            case GpxTagV11.routePoint:
              rte.rtepts.add(_readPoint(iterator, val.name));
              break;

            case GpxTagV11.name:
              rte.name = _readString(iterator, val.name);
              break;
            case GpxTagV11.desc:
              rte.desc = _readString(iterator, val.name);
              break;
            case GpxTagV11.comment:
              rte.cmt = _readString(iterator, val.name);
              break;
            case GpxTagV11.src:
              rte.src = _readString(iterator, val.name);
              break;

            case GpxTagV11.link:
              rte.links.add(_readLink(iterator));
              break;

            case GpxTagV11.number:
              rte.number = _readInt(iterator, val.name);
              break;
            case GpxTagV11.type:
              rte.type = _readString(iterator, val.name);
              break;

            case GpxTagV11.extensions:
              rte.extensions = _readExtensions(iterator);
              break;
          }
        }

        if (val is XmlEndElementEvent && val.name == GpxTagV11.route) {
          break;
        }
      }
    }

    return rte;
  }

  Trk _parseTrack(Iterator<XmlEvent> iterator) {
    final trk = Trk();
    final elm = iterator.current;

    if ((elm is XmlStartElementEvent) && !elm.isSelfClosing) {
      while (iterator.moveNext()) {
        final val = iterator.current;

        if (val is XmlStartElementEvent) {
          switch (val.name) {
            case GpxTagV11.trackSegment:
              trk.trksegs.add(_readSegment(iterator));
              break;

            case GpxTagV11.name:
              trk.name = _readString(iterator, val.name);
              break;
            case GpxTagV11.desc:
              trk.desc = _readString(iterator, val.name);
              break;
            case GpxTagV11.comment:
              trk.cmt = _readString(iterator, val.name);
              break;
            case GpxTagV11.src:
              trk.src = _readString(iterator, val.name);
              break;

            case GpxTagV11.link:
              trk.links.add(_readLink(iterator));
              break;

            case GpxTagV11.number:
              trk.number = _readInt(iterator, val.name);
              break;
            case GpxTagV11.type:
              trk.type = _readString(iterator, val.name);
              break;

            case GpxTagV11.extensions:
              trk.extensions = _readExtensions(iterator);
              break;
          }
        }

        if (val is XmlEndElementEvent && val.name == GpxTagV11.track) {
          break;
        }
      }
    }

    return trk;
  }

  Wpt _readPoint(Iterator<XmlEvent> iterator, String tagName) {
    final wpt = Wpt();
    final elm = iterator.current;

    if (elm is XmlStartElementEvent) {
      // LOCAL MODIFICATION 2 (see file header): tolerate missing/unparseable
      // lat/lon instead of throwing StateError out of `firstWhere`.
      //
      // The GPX spec requires both attributes, so a point without them is
      // genuinely malformed — but one such point should not cost the caller
      // the entire file. Leaving lat/lon null lets downstream code drop the
      // point: `computeTrailMetrics` skips non-finite coordinates, and
      // `GpxMappingUtils.allWaypoints` filters on `lat != null && lon != null`.
      wpt.lat = _readCoordinateAttribute(elm, GpxTagV11.latitude);
      wpt.lon = _readCoordinateAttribute(elm, GpxTagV11.longitude);
    }

    if ((elm is XmlStartElementEvent) && !elm.isSelfClosing) {
      while (iterator.moveNext()) {
        final val = iterator.current;

        if (val is XmlStartElementEvent) {
          switch (val.name) {
            case GpxTagV11.sym:
              wpt.sym = _readString(iterator, val.name);
              break;

            case GpxTagV11.fix:
              final fixAsString = _readString(iterator, val.name);
              wpt.fix = FixType.values.firstWhere(
                (e) =>
                    e.toString().replaceFirst('.fix_', '.') ==
                    'FixType.$fixAsString',
                orElse: () => FixType.unknown,
              );

              if (wpt.fix == FixType.unknown) {
                wpt.fix = null;
              }
              break;

            case GpxTagV11.dGPSId:
              wpt.dgpsid = _readInt(iterator, val.name);
              break;

            case GpxTagV11.name:
              wpt.name = _readString(iterator, val.name);
              break;
            case GpxTagV11.desc:
              wpt.desc = _readString(iterator, val.name);
              break;
            case GpxTagV11.comment:
              wpt.cmt = _readString(iterator, val.name);
              break;
            case GpxTagV11.src:
              wpt.src = _readString(iterator, val.name);
              break;
            case GpxTagV11.link:
              wpt.links.add(_readLink(iterator));
              break;
            case GpxTagV11.hDOP:
              wpt.hdop = _readDouble(iterator, val.name);
              break;
            case GpxTagV11.vDOP:
              wpt.vdop = _readDouble(iterator, val.name);
              break;
            case GpxTagV11.pDOP:
              wpt.pdop = _readDouble(iterator, val.name);
              break;
            case GpxTagV11.ageOfData:
              wpt.ageofdgpsdata = _readDouble(iterator, val.name);
              break;

            case GpxTagV11.magVar:
              wpt.magvar = _readDouble(iterator, val.name);
              break;
            case GpxTagV11.geoidHeight:
              wpt.geoidheight = _readDouble(iterator, val.name);
              break;

            case GpxTagV11.sat:
              wpt.sat = _readInt(iterator, val.name);
              break;

            case GpxTagV11.elevation:
              wpt.ele = _readDouble(iterator, val.name);
              break;
            case GpxTagV11.time:
              wpt.time = _readDateTime(iterator, val.name);
              break;
            case GpxTagV11.type:
              wpt.type = _readString(iterator, val.name);
              break;
            case GpxTagV11.extensions:
              wpt.extensions = _readExtensions(iterator);
              break;
          }
        }

        if (val is XmlEndElementEvent && val.name == tagName) {
          break;
        }
      }
    }

    return wpt;
  }

  /// Value of [attributeName] on [elm], or null when absent.
  ///
  /// Upstream reads required-by-spec attributes with a bare `firstWhere`,
  /// which throws `StateError: Bad state: No element` the moment a real file
  /// omits one. Every local modification that fixes such a site goes through
  /// here.
  String? _attributeOrNull(XmlStartElementEvent elm, String attributeName) {
    for (final attr in elm.attributes) {
      if (attr.name == attributeName) return attr.value;
    }
    return null;
  }

  /// Reads a required-by-spec numeric attribute without throwing when it is
  /// absent or unparseable. Added by LOCAL MODIFICATIONS 2 and 3; shared by
  /// `_readPoint` and `_readBounds`.
  double? _readCoordinateAttribute(
    XmlStartElementEvent elm,
    String attributeName,
  ) {
    final raw = _attributeOrNull(elm, attributeName);
    return raw == null ? null : double.tryParse(raw);
  }

  // LOCAL MODIFICATION 1 (see file header): `tryParse`, not `parse`.
  //
  // `_readString` returns `''` — never `null` — for a present-but-empty
  // element, so upstream's `!= null` guard never fired and `double.parse('')`
  // threw, aborting the entire document over one empty `<hdop></hdop>`. These
  // fields are all `double?`/`int?`/`DateTime?`: absent and unparseable both
  // mean "no reading", which is exactly what `tryParse` returns.
  double? _readDouble(Iterator<XmlEvent> iterator, String tagName) {
    final doubleString = _readString(iterator, tagName);
    return doubleString != null ? double.tryParse(doubleString) : null;
  }

  int? _readInt(Iterator<XmlEvent> iterator, String tagName) {
    final intString = _readString(iterator, tagName);
    return intString != null ? int.tryParse(intString) : null;
  }

  DateTime? _readDateTime(Iterator<XmlEvent> iterator, String tagName) {
    final dateTimeString = _readString(iterator, tagName);
    return dateTimeString != null ? DateTime.tryParse(dateTimeString) : null;
  }

  String? _readString(Iterator<XmlEvent> iterator, String tagName) {
    final elm = iterator.current;
    if (!(elm is XmlStartElementEvent &&
        elm.name == tagName &&
        !elm.isSelfClosing)) {
      return null;
    }

    var string = '';
    while (iterator.moveNext()) {
      final val = iterator.current;

      if (val is XmlTextEvent) {
        string += val.value;
      }

      if (val is XmlCDATAEvent) {
        string += val.value;
      }

      if (val is XmlEndElementEvent && val.name == tagName) {
        break;
      }
    }

    return string;
  }

  Object? _readMap(Iterator<XmlEvent> iterator, String tagName) {
    final elm = iterator.current;
    if (!(elm is XmlStartElementEvent &&
        elm.name == tagName &&
        !elm.isSelfClosing)) {
      return null;
    }

    final valueMap = <String, Object>{};
    String? valueText;
    while (iterator.moveNext()) {
      final val = iterator.current;

      if (val is XmlStartElementEvent) {
        valueMap[val.name] = _readMap(iterator, val.name) ?? {};
      }

      if (val is XmlTextEvent) {
        valueText = val.value;
      }

      if (val is XmlCDATAEvent) {
        valueText = val.value;
      }

      if (val is XmlEndElementEvent && val.name == tagName) {
        break;
      }
    }

    return valueMap.isNotEmpty ? valueMap : valueText;
  }

  Trkseg _readSegment(Iterator<XmlEvent> iterator) {
    final trkseg = Trkseg();
    final elm = iterator.current;

    if ((elm is XmlStartElementEvent) && !elm.isSelfClosing) {
      while (iterator.moveNext()) {
        final val = iterator.current;

        if (val is XmlStartElementEvent) {
          switch (val.name) {
            case GpxTagV11.trackPoint:
              trkseg.trkpts.add(_readPoint(iterator, val.name));
              break;
            case GpxTagV11.extensions:
              trkseg.extensions = _readExtensions(iterator);
              break;
          }
        }

        if (val is XmlEndElementEvent && val.name == GpxTagV11.trackSegment) {
          break;
        }
      }
    }

    return trkseg;
  }

  Map<String, Object> _readExtensions(Iterator<XmlEvent> iterator) {
    final exts = _readMap(iterator, GpxTagV11.extensions) ?? {};
    return (exts is Map<String, Object>) ? exts : {};
  }

  Link _readLink(Iterator<XmlEvent> iterator) {
    final link = Link();
    final elm = iterator.current;

    if (elm is XmlStartElementEvent) {
      final hrefs = elm.attributes.where((attr) => attr.name == GpxTagV11.href);

      if (hrefs.isNotEmpty) {
        link.href = hrefs.first.value;
      }
    }

    if ((elm is XmlStartElementEvent) && !elm.isSelfClosing) {
      while (iterator.moveNext()) {
        final val = iterator.current;

        if (val is XmlStartElementEvent) {
          switch (val.name) {
            case GpxTagV11.text:
              link.text = _readString(iterator, val.name);
              break;
            case GpxTagV11.type:
              link.type = _readString(iterator, val.name);
              break;
          }
        }

        if (val is XmlEndElementEvent && val.name == GpxTagV11.link) {
          break;
        }
      }
    }

    return link;
  }

  Person _readPerson(Iterator<XmlEvent> iterator) {
    final person = Person();
    final elm = iterator.current;

    if ((elm is XmlStartElementEvent) && !elm.isSelfClosing) {
      while (iterator.moveNext()) {
        final val = iterator.current;

        if (val is XmlStartElementEvent) {
          switch (val.name) {
            case GpxTagV11.name:
              person.name = _readString(iterator, val.name);
              break;
            case GpxTagV11.email:
              person.email = _readEmail(iterator);
              break;
            case GpxTagV11.link:
              person.link = _readLink(iterator);
              break;
          }
        }

        if (val is XmlEndElementEvent && val.name == GpxTagV11.author) {
          break;
        }
      }
    }

    return person;
  }

  Copyright _readCopyright(Iterator<XmlEvent> iterator) {
    final copyright = Copyright();
    final elm = iterator.current;

    if (elm is XmlStartElementEvent) {
      // LOCAL MODIFICATION 5 (see file header): tolerate a <copyright> with no
      // author attribute. Same `firstWhere`-with-no-orElse crash as _readEmail
      // had — StateError: Bad state: No element — over a metadata field the app
      // never reads.
      copyright.author = _attributeOrNull(elm, GpxTagV11.author) ?? '';

      if (!elm.isSelfClosing) {
        while (iterator.moveNext()) {
          final val = iterator.current;

          if (val is XmlStartElementEvent) {
            switch (val.name) {
              case GpxTagV11.year:
                copyright.year = _readInt(iterator, val.name);
                break;
              case GpxTagV11.license:
                copyright.license = _readString(iterator, val.name);
                break;
            }
          }

          if (val is XmlEndElementEvent && val.name == GpxTagV11.copyright) {
            break;
          }
        }
      }
    }

    return copyright;
  }

  Bounds _readBounds(Iterator<XmlEvent> iterator) {
    final bounds = Bounds();
    final elm = iterator.current;

    if (elm is XmlStartElementEvent) {
      // LOCAL MODIFICATION 3 (see file header): tolerate missing/unparseable
      // bounds attributes rather than throwing. `<bounds>` is optional
      // metadata the app does not read — losing it must never cost the track.
      //
      // Unlike `Wpt.lat`, `Bounds`' fields are non-nullable, so an unparseable
      // attribute leaves the model's own 0.0 default rather than being nulled.
      // That is indistinguishable from a genuine 0.0, which is acceptable here
      // only because nothing in the app reads `Bounds` — the trail's bounding
      // box is recomputed from the track by `computeTrailMetrics`.
      final minlat = _readCoordinateAttribute(elm, GpxTagV11.minLatitude);
      final minlon = _readCoordinateAttribute(elm, GpxTagV11.minLongitude);
      final maxlat = _readCoordinateAttribute(elm, GpxTagV11.maxLatitude);
      final maxlon = _readCoordinateAttribute(elm, GpxTagV11.maxLongitude);
      if (minlat != null) bounds.minlat = minlat;
      if (minlon != null) bounds.minlon = minlon;
      if (maxlat != null) bounds.maxlat = maxlat;
      if (maxlon != null) bounds.maxlon = maxlon;

      if (!elm.isSelfClosing) {
        while (iterator.moveNext()) {
          final val = iterator.current;

          if (val is XmlEndElementEvent && val.name == GpxTagV11.bounds) {
            break;
          }
        }
      }
    }

    return bounds;
  }

  // LOCAL MODIFICATION 4 (see file header): accept the non-standard
  // `<email>user@example.com</email>` text form.
  //
  // GPX 1.1 specifies `<email id="user" domain="example.com"/>`, but plenty of
  // exporters emit the address as element text instead. Upstream reads both
  // attributes with `firstWhere`, so the text form threw
  // `StateError: Bad state: No element` and killed the whole document over a
  // metadata field nothing depends on. Attributes still win when present; the
  // text form is only consulted as a fallback, and an address with no `@` (or
  // none at all) simply leaves the fields empty.
  Email _readEmail(Iterator<XmlEvent> iterator) {
    final email = Email();
    final elm = iterator.current;

    if (elm is XmlStartElementEvent) {
      String? id;
      String? domain;
      for (final attr in elm.attributes) {
        if (attr.name == GpxTagV11.id) id = attr.value;
        if (attr.name == GpxTagV11.domain) domain = attr.value;
      }

      var text = '';
      if (!elm.isSelfClosing) {
        while (iterator.moveNext()) {
          final val = iterator.current;

          if (val is XmlTextEvent) text += val.value;
          if (val is XmlCDATAEvent) text += val.value;

          if (val is XmlEndElementEvent && val.name == GpxTagV11.email) {
            break;
          }
        }
      }

      if (id == null || domain == null) {
        final at = text.trim().lastIndexOf('@');
        if (at > 0 && at < text.trim().length - 1) {
          final trimmed = text.trim();
          id ??= trimmed.substring(0, at);
          domain ??= trimmed.substring(at + 1);
        }
      }

      email.id = id ?? '';
      email.domain = domain ?? '';
    }

    return email;
  }
}
