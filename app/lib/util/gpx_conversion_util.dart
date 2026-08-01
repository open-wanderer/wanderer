import 'dart:math';

import 'package:collection/collection.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:gpx/gpx.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/util/icon_util.dart';

import 'gpx_util.dart' show sanitizeGpxEmail;

/// GPX elements whose text `GpxReader` hands to a THROWING `double.parse`
/// (`gpx_reader.dart:339-343`'s `_readDouble`, reached from `wpt.ele`,
/// `wpt.hdop`, `wpt.vdop`, `wpt.pdop`, `wpt.ageofdgpsdata`, `wpt.magvar`
/// and `wpt.geoidheight` at `gpx_reader.dart:292-316`).
const _gpxDoubleTags = <String>[
  'ele',
  'hdop',
  'vdop',
  'pdop',
  'magvar',
  'geoidheight',
  'ageofdgpsdata',
];

/// GPX elements whose text `GpxReader` hands to a THROWING `int.parse`
/// (`gpx_reader.dart:344-347`'s `_readInt`, reached from `wpt.sat` and
/// `wpt.dgpsid`).
///
/// `<number>` (rte/trk) and `<year>` (copyright) also reach `_readInt` and
/// are deliberately NOT rewritten here: both tag names are generic enough
/// that a blind, context-free rewrite could clobber same-named content
/// inside an `<extensions>` block, and neither is part of the commonly
/// emitted optional-element set this pass exists to survive.
const _gpxIntTags = <String>['sat', 'dgpsid'];

/// Spans whose contents are TEXT, not markup, and so must never be rewritten:
/// a rewrite inside a `<![CDATA[...]]>` section would silently mutate a
/// description/comment the user actually sees. Comments are protected on the
/// same principle (cheap, and keeps the pass a pure markup transform).
final _protectedXmlRegionPattern = RegExp(
  r'<!\[CDATA\[[\s\S]*?\]\]>|<!--[\s\S]*?-->',
);

final Map<String, RegExp> _sanitizableTagPatterns = {
  for (final tag in [..._gpxDoubleTags, ..._gpxIntTags, 'time'])
    tag: RegExp('<$tag>([^<]*)</$tag>'),
};

/// Applies [rewrite] to every span of [xml] that is OUTSIDE a CDATA section
/// or an XML comment, splicing those protected spans back verbatim.
String _rewriteOutsideProtectedRegions(
  String xml,
  String Function(String) rewrite,
) {
  final matches = _protectedXmlRegionPattern.allMatches(xml).toList();
  if (matches.isEmpty) return rewrite(xml);

  final buffer = StringBuffer();
  var cursor = 0;
  for (final match in matches) {
    buffer.write(rewrite(xml.substring(cursor, match.start)));
    buffer.write(match[0]);
    cursor = match.end;
  }
  buffer.write(rewrite(xml.substring(cursor)));
  return buffer.toString();
}

/// Neutralises every confirmed `GpxReader` crash input that the corrected TS
/// parser (via `parseElevation`/`Date` coercion, or `xml2js`'s
/// never-coercing behaviour) treats as "no data": an empty-but-present,
/// whitespace-only or non-numeric body on any of the ten numerically- or
/// temporally-parsed GPX elements.
///
/// `GpxReader` (`package:gpx` 2.3.0) calls `double.parse`/`int.parse`/
/// `DateTime.parse` directly on the accumulated element text with no
/// empty/malformed guard (`gpx_reader.dart`'s `_readDouble`/`_readInt`/
/// `_readDateTime`), and its `_readString` helper returns `''` — not `null` —
/// for any non-self-closing element. So `<ele></ele>`, `<hdop></hdop>`,
/// `<sat>   </sat>` and `<pdop>N/A</pdop>` all throw `FormatException` and
/// abort the whole import. `<hdop>`/`<vdop>`/`<pdop>`/`<sat>` in particular
/// are among the most commonly emitted optional GPX elements (Garmin, Locus,
/// OsmAnd and many track loggers), and empty-element forms are routine in
/// exporter output — before this app parsed GPX itself these files went
/// through the server's `xml2js`, which never coerces and so never threw.
///
/// This mirrors [sanitizeGpxEmail]'s existing regex-rewrite precedent:
/// rewrite the malformed body to a self-closing tag (`<ele/>`, `<hdop/>`,
/// `<time/>`, ...), which `GpxReader` already treats as `null` (a
/// self-closing start element short-circuits `_readString` to `null` before
/// any `parse` call is reached).
///
/// Deliberate non-corruption properties:
/// - Only an EXACT, attribute-less `<tag>` … `</tag>` pair matches, so a
///   namespaced `<gpx:ele>` or a longer same-suffix tag (`<myele>`) is never
///   touched.
/// - A body is rewritten only when it fails to parse, so a genuine
///   `<ele>0</ele>` (real sea level, CONV-03) and a pretty-printed
///   `<ele>\n 1000.5\n</ele>` (`double.tryParse`/`int.parse` both trim
///   surrounding whitespace) survive verbatim.
/// - CDATA sections and comments are excluded entirely — see
///   [_rewriteOutsideProtectedRegions]. XML forbids a raw `<` inside an
///   attribute value, so an attribute can never contain a matchable span.
String sanitizeGpxNumericAndTime(String xml) {
  return _rewriteOutsideProtectedRegions(xml, _sanitizeMarkupSpan);
}

String _sanitizeMarkupSpan(String span) {
  var out = span;

  for (final tag in _gpxDoubleTags) {
    out = out.replaceAllMapped(_sanitizableTagPatterns[tag]!, (m) {
      final parsed = double.tryParse((m[1] ?? '').trim());
      if (parsed != null && parsed.isFinite) {
        return m[0]!;
      }
      return '<$tag/>';
    });
  }

  for (final tag in _gpxIntTags) {
    out = out.replaceAllMapped(_sanitizableTagPatterns[tag]!, (m) {
      if (int.tryParse((m[1] ?? '').trim()) != null) {
        return m[0]!;
      }
      return '<$tag/>';
    });
  }

  return out.replaceAllMapped(_sanitizableTagPatterns['time']!, (m) {
    if (DateTime.tryParse((m[1] ?? '').trim()) != null) {
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
/// [sanitizeGpxNumericAndTime]) before handing the string to [GpxReader].
///
/// This is the ONLY place in `lib/` that may construct a [GpxReader] — the
/// import path, the server-download path (`trail_provider.dart`) and the
/// offline-cache read (`trail_entity.dart`) all route through here, and
/// `gpx_conversion_util_test.dart`'s "single GpxReader call site" gate fails
/// the build if a fourth appears. That gate exists because bypassing this
/// function silently opts a call site out of the sanitize chain: the two
/// non-import sites previously did, which made a server-authored track fail
/// to open (swallowed by a broad `catch`) and, once cached, made it
/// permanently un-openable offline.
///
/// A `<trkpt>` missing its `lat`/`lon` attribute still throws `StateError`
/// from `GpxReader` (34-RESEARCH.md Pitfall 1) — this is a much rarer,
/// structurally broken input the GPX spec itself requires both attributes
/// for, and it is deliberately left to callers' existing try/catch-and-toast
/// paths rather than handled here via string surgery.
Gpx parseGpxSafely(String xml) {
  return GpxReader().fromString(
    sanitizeGpxNumericAndTime(sanitizeGpxEmail(xml)),
  );
}

/// The Dart analogue of `parseElevation` (`gpx-metrics-computation.ts:15-24`).
///
/// `package:gpx` already coerces `<ele>` element text to `double?`, so the
/// string/empty-string branches from the TS original collapse; what must
/// survive the port is the finite check. A genuine `0.0` is real data,
/// never "missing".
double? parseGpxElevation(double? raw) {
  if (raw == null || !raw.isFinite) {
    return null;
  }
  return raw;
}

/// Line-for-line Dart port of `gpx-metrics-computation.ts`'s
/// `GpxMetricsComputation` class — the defer-then-publish elevation noise
/// filter plus the threshold-gated distance-smoothing accumulator.
///
/// The raw per-point cumulative-distance array from the TS original is
/// deliberately NOT ported (D-04 / Pitfall 3): its only consumer is the web
/// trail-edit crop slider, which has no Dart equivalent, so porting it
/// would be dead code.
class GpxMetricsComputation {
  final double _thresholdXYm; // Distance threshold for filtering on the XY axis (latitude / longitude)
  final double _thresholdZm; // Distance threshold for filtering on the Z axis (elevation)

  Wpt? _lastPointXY;
  Wpt? _lastFilteredPointXY;
  double? _lastFilteredZ;
  double? _lastZ;
  // Point at which _lastFilteredZ was last set. Used only to ask "has the
  // device moved horizontally since the elevation anchor?" on the discard
  // path below. Distinct from _lastFilteredPointXY, which is the
  // distance-smoothing anchor — the two must never be merged or reused.
  Wpt? _lastFilteredZPointXY;
  // Signed elevation delta of the most recent above-threshold move, held
  // back from the published totals until it is confirmed. 0 means nothing
  // is pending. See the defer-then-publish note in addAndFilter().
  double _pendingDelta = 0;
  // The value _lastFilteredZ held immediately before the pending excursion.
  double? _pendingAnchorZ;

  double totalElevationGain = 0;
  double totalElevationLoss = 0;

  // INVARIANT: these two are monotonically non-decreasing running totals.
  // The web's trail_anchor_list.svelte derives per-segment metrics by
  // subtracting consecutive snapshots of them, so any code path that
  // decrements them makes that component render negative elevation gain.
  // Only ever `+=`. The provisional (not-yet-confirmed) excursion lives in
  // _pendingDelta and is surfaced through finalElevationGain/
  // finalElevationLoss instead.
  double totalElevationGainSmoothed = 0;
  double totalElevationLossSmoothed = 0;

  double totalDistance = 0;
  double totalDistanceSmoothed = 0;

  GpxMetricsComputation(this._thresholdXYm, this._thresholdZm);

  /// Smoothed elevation gain including a still-pending excursion.
  ///
  /// This — not [totalElevationGainSmoothed] — is a completed track's
  /// reported gain. A track that ends mid-swing (climbs and stops) has its
  /// final climb sitting in the pending slot, unconfirmed but real.
  ///
  /// Consumers that difference successive readings to derive per-segment
  /// metrics must use the monotonic [totalElevationGainSmoothed] instead;
  /// this getter can decrease when a pending excursion is later discarded
  /// as noise.
  double get finalElevationGain =>
      totalElevationGainSmoothed + max(_pendingDelta, 0.0);

  /// Smoothed elevation loss including a still-pending excursion. See
  /// [finalElevationGain].
  double get finalElevationLoss =>
      totalElevationLossSmoothed + max(-_pendingDelta, 0.0);

  /// Moves the pending excursion into the published totals. Only ever adds,
  /// so the monotonicity invariant on those fields is preserved.
  void _publishPending() {
    if (_pendingDelta > 0) {
      totalElevationGainSmoothed += _pendingDelta;
    } else if (_pendingDelta < 0) {
      totalElevationLossSmoothed -= _pendingDelta;
    }
    _pendingDelta = 0;
    _pendingAnchorZ = null;
  }

  /// Great-circle distance between two waypoints, reused (not hand-rolled)
  /// from `package:geobase`'s `SphericalGreatCircle` (re-exported via
  /// `package:maplibre`), verified formula-identical to the TS
  /// `haversineDistance`. Returns `double.nan` when either coordinate is
  /// null, reproducing the TS behaviour where an `undefined` `$.lat`
  /// propagates `NaN` out of `haversineDistance`.
  double _haversine(Wpt a, Wpt b) {
    if (a.lat == null || a.lon == null || b.lat == null || b.lon == null) {
      return double.nan;
    }
    return SphericalGreatCircle(
      Geographic(lat: a.lat!, lon: a.lon!),
    ).distanceTo(Geographic(lat: b.lat!, lon: b.lon!));
  }

  /// Line-for-line port of `addAndFilter` (`gpx-metrics-computation.ts:97-235`).
  /// Do not simplify this into a plain accumulator — that reintroduces the
  /// CONV-04 defect Phase 33 fixed.
  void addAndFilter(Wpt point) {
    if (_lastPointXY == null || _lastFilteredPointXY == null) {
      // Initialize raw and smoothed anchors with the first point. When the
      // first point has no usable elevation, leave both anchors null so
      // the first point that *does* carry elevation becomes the anchor
      // instead of diffing against a fabricated 0.
      _lastPointXY = point;
      _lastFilteredPointXY = point;
      final initialElevation = parseGpxElevation(point.ele);
      if (initialElevation != null) {
        _lastFilteredZ = initialElevation;
        _lastFilteredZPointXY = point;
        _lastZ = initialElevation;
      }
      return;
    }

    final distance = _haversine(_lastPointXY!, point);
    final smoothedDistance = _haversine(_lastFilteredPointXY!, point);

    if (distance.isFinite) {
      totalDistance += distance;
    }

    _lastPointXY = point;

    final elevation = parseGpxElevation(point.ele);
    if (elevation != null) {
      if (_lastZ == null) {
        // This point establishes the raw anchor; no diff to record yet.
        _lastZ = elevation;
      } else {
        final elevationDiff = elevation - _lastZ!;
        _lastZ = elevation;
        if (elevationDiff > 0) {
          totalElevationGain += elevationDiff;
        }
        if (elevationDiff < 0) {
          totalElevationLoss -= elevationDiff;
        }
      }
    }

    // Defer-then-publish noise filter. An above-threshold elevation move is
    // held in a single pending slot rather than added to the published
    // totals straight away. It is *discarded* if the track then returns to
    // the pre-excursion elevation WITHOUT having moved horizontally — the
    // signature of altimeter/GPS noise on a paused or stationary device,
    // and the only thing that distinguishes it from rolling terrain (which
    // produces an identical elevation series). Any other subsequent move
    // confirms it, so it is published then.
    //
    // Deferring rather than committing-then-retracting is what keeps
    // totalElevationGainSmoothed/LossSmoothed monotonic (see the INVARIANT
    // note on those fields): a discarded excursion was never published, so
    // no published total ever decreases. Horizontal stillness is checked
    // only on the discard path, so a monotonic low-horizontal climb (the
    // CONV-04 case) is never affected.
    //
    // The pending excursion is not lost when a track ends mid-swing: it is
    // surfaced by the finalElevationGain/finalElevationLoss getters, which
    // are what a completed track's reported totals come from.
    if (elevation != null) {
      if (_lastFilteredZ == null) {
        // This point establishes the smoothed anchor; no diff to record yet.
        _lastFilteredZ = elevation;
        _lastFilteredZPointXY = point;
        _pendingDelta = 0;
        _pendingAnchorZ = null;
      } else {
        final elevationDiffSmoothed = elevation - _lastFilteredZ!;

        if (elevationDiffSmoothed.abs() < _thresholdZm) {
          // Below the noise floor — nothing to defer, publish or discard.
        } else {
          // Distance from where the pending excursion left off to here: "has
          // the device moved horizontally while the elevation came back?"
          final returnDistance = _lastFilteredZPointXY != null
              ? _haversine(_lastFilteredZPointXY!, point)
              : double.infinity;

          final cancelsPending =
              _pendingDelta != 0 &&
              _pendingAnchorZ != null &&
              elevationDiffSmoothed.sign == -_pendingDelta.sign &&
              (elevation - _pendingAnchorZ!).abs() < _thresholdZm &&
              returnDistance.isFinite &&
              returnDistance < _thresholdXYm;

          if (cancelsPending) {
            // Noise round-trip: drop the pending excursion unpublished and
            // rewind the anchor to where the excursion started.
            _lastFilteredZ = _pendingAnchorZ;
            _lastFilteredZPointXY = point;
            _pendingDelta = 0;
            _pendingAnchorZ = null;
          } else {
            // This move confirms whatever was pending — publish it — and
            // then becomes the new pending excursion itself.
            _publishPending();
            _pendingAnchorZ = _lastFilteredZ;
            _pendingDelta = elevationDiffSmoothed;
            _lastFilteredZ = elevation;
            _lastFilteredZPointXY = point;
          }
        }
      }
    }

    if (smoothedDistance >= _thresholdXYm) {
      totalDistanceSmoothed += smoothedDistance;
      _lastFilteredPointXY = point;
    }
  }
}

/// Immutable snapshot of a GPX track's public metrics. Deliberately narrower
/// than the TS `GPXFeature` shape — no per-point cumulative-distance array,
/// no `hash` (D-04, public metrics only; Dart internals may differ from TS).
class GpxTrailMetrics {
  final double centroidLat;
  final double centroidLon;
  final double minLat;
  final double maxLat;
  final double minLon;
  final double maxLon;
  final double distance;
  final double elevationGain;
  final double elevationLoss;
  final int durationMs;
  final int pointCount;

  const GpxTrailMetrics({
    required this.centroidLat,
    required this.centroidLon,
    required this.minLat,
    required this.maxLat,
    required this.minLon,
    required this.maxLon,
    required this.distance,
    required this.elevationGain,
    required this.elevationLoss,
    required this.durationMs,
    required this.pointCount,
  });
}

/// Line-for-line Dart port of the public-metrics assembly logic at
/// `gpx.ts:97-167`, minus the per-point cumulative-distance array and
/// `hash` (D-04).
///
/// One [GpxMetricsComputation] instance is constructed and shared across
/// EVERY [Trk] and EVERY [Trkseg] in [gpx] — no per-segment anchor reset —
/// so a multi-leg planner route measures through its anchors (Phase 33
/// 33-01 decision).
///
/// The point loop starts at index 0 (CONV-01) and the centroid divides by
/// the same count it summed, `summedPointCount` (CONV-02) — not by a
/// separately-collected point list. With no points, the centroid is
/// `0 / 0` = `double.nan` and the bounding box keeps its infinite
/// sentinels, matching the TS zero-point behaviour; this is deliberate, not
/// guarded.
GpxTrailMetrics computeTrailMetrics(Gpx gpx) {
  double totalLat = 0;
  double totalLon = 0;
  var summedPointCount = 0;
  var totalDurationMs = 0;

  var minLat = double.infinity;
  var maxLat = double.negativeInfinity;
  var minLon = double.infinity;
  var maxLon = double.negativeInfinity;

  final metrics = GpxMetricsComputation(5, 5);

  for (final track in gpx.trks) {
    for (final segment in track.trksegs) {
      final points = segment.trkpts;

      if (points.length >= 2) {
        final startTime = points.first.time;
        final endTime = points.last.time;
        if (startTime != null && endTime != null) {
          totalDurationMs += endTime.difference(startTime).inMilliseconds;
        }
      }

      for (var i = 0; i < points.length; i++) {
        final point = points[i];
        metrics.addAndFilter(point);

        totalLat += point.lat ?? 0;
        totalLon += point.lon ?? 0;
        summedPointCount++;

        minLat = min(minLat, point.lat ?? double.infinity);
        maxLat = max(maxLat, point.lat ?? double.negativeInfinity);
        minLon = min(minLon, point.lon ?? double.infinity);
        maxLon = max(maxLon, point.lon ?? double.negativeInfinity);
      }
    }
  }

  return GpxTrailMetrics(
    centroidLat: totalLat / summedPointCount,
    centroidLon: totalLon / summedPointCount,
    minLat: minLat,
    maxLat: maxLat,
    minLon: minLon,
    maxLon: maxLon,
    distance: metrics.totalDistanceSmoothed,
    elevationGain: metrics.finalElevationGain,
    elevationLoss: metrics.finalElevationLoss,
    durationMs: totalDurationMs.abs(),
    pointCount: summedPointCount,
  );
}

/// Line-for-line Dart port of `gpx2trail`'s trail assembly
/// (`web/src/lib/util/gpx_util.ts:37-89`) — the PORT-01 draft-trail step.
/// Builds a complete, unsaved [Trail] from a parsed [gpx] with no network
/// call (D-14): name, description, waypoints, start coordinates, date,
/// distance, elevation gain/loss, duration and bounding box.
///
/// [movingDuration] is the D-13 override: when a caller (a recording session)
/// supplies it, it becomes `Trail.movingDuration` — UNLESS it rounds down to
/// zero whole seconds, in which case it is treated as absent (`null`).
///
/// That zero-to-null mapping is D-10's "no value" state, and it has to live
/// here rather than at each call site: `NavigationStats.elapsed` starts at
/// `Duration.zero` and stays there until the 1-second tick begins, so a
/// recording saved immediately passed a zero elapsed straight through to
/// `moving_duration = 0` — precisely the state `form_data_util.dart`'s own
/// comment says must never be written ("sending an empty string for an
/// absent value would write 0 into PocketBase and defeat D-10's 'no value'
/// state"). Its write guard is `!= null`, not `> 0`, so nothing downstream
/// catches it, and the display rule's `> 0` fallback then MASKS the bad row
/// until something else reads the field. Note the truncation is the real
/// boundary, not `Duration.zero`: a 500 ms elapsed also yields 0 whole
/// seconds, which a `> Duration.zero` check at the call site would miss
/// (WR-08).
///
/// `Trail.duration` NEVER
/// comes from it — `duration` always comes from the GPX's own first/last
/// trkpt timestamps, so every persisted stat except moving time stays
/// reproducible by a later recompute over the same GPX (D-11). Omitting the
/// parameter leaves `movingDuration` null.
///
/// [gpxData] is the raw GPX string [gpx] was parsed from. When supplied it
/// is carried on `expand.gpxData`, which `form_data_util.dart` uploads as
/// the trail's track file on save — a caller that omits it produces a trail
/// that saves with no GPX.
///
/// `id`/`created`/`updated` are placeholders (`''`/`DateTime.now()`),
/// mirroring `trail_import_util.dart`'s `convertGpxToTrail` convention for a
/// not-yet-persisted trail — that function's own placeholder-injection is
/// unchanged by this plan; only its data source moves here in a later plan.
Trail trailFromGpx(
  Gpx gpx, {
  String? fallbackName,
  Duration? movingDuration,
  String? gpxData,
}) {
  // TS uses `||`, under which an empty string falls through to the next
  // candidate — replicate with an explicit isNotEmpty check per candidate,
  // not `??` (a `??` would stop at a present-but-empty string).
  final metadataName = gpx.metadata?.name;
  final trkName = gpx.trks.firstOrNull?.name;
  final rteName = gpx.rtes.firstOrNull?.name;
  final String name;
  if (metadataName != null && metadataName.isNotEmpty) {
    name = metadataName;
  } else if (trkName != null && trkName.isNotEmpty) {
    name = trkName;
  } else if (rteName != null && rteName.isNotEmpty) {
    name = rteName;
  } else if (fallbackName != null && fallbackName.isNotEmpty) {
    name = fallbackName;
  } else {
    name = 'trail-${DateTime.now().toIso8601String()}';
  }

  // The TS model leaves `description` `undefined` when `metadata.desc` is
  // absent; the Dart model's field is non-nullable with a `''` default —
  // treated as equivalent here and by the corpus's comparison helper.
  final description = gpx.metadata?.desc ?? '';

  // Hoisted once so every waypoint shares one placeholder created/updated
  // value, matching convertGpxToTrail's convention.
  final now = DateTime.now();

  final waypoints = <Waypoint>[
    for (final wpt in gpx.wpts)
      Waypoint(
        id: '',
        lat: wpt.lat ?? 0,
        lon: wpt.lon ?? 0,
        name: wpt.name ?? '',
        description: wpt.desc ?? '',
        // Closed-set lookup (Pitfall 6 / T-34-18): an unknown or hostile
        // `sym` resolves to the default circle and can never inject
        // arbitrary content. Never assign wpt.sym directly — Waypoint.icon
        // is FaIconData, not a String.
        icon: fontAwesomeIconsMap[wpt.sym] ?? FontAwesomeIcons.circle,
        created: now,
        updated: now,
      ),
  ];

  final trackPoints = gpx.trks.firstOrNull?.trksegs.firstOrNull?.trkpts;
  final routePoints = gpx.rtes.firstOrNull?.rtepts;
  final startPoint = trackPoints?.firstOrNull ?? routePoints?.firstOrNull;

  // Both the first AND last point of the segment must carry a time
  // (mirrors gpx.ts:65-71's guard exactly) — a time on an interior point
  // alone does not set the date.
  final startTime = trackPoints?.firstOrNull?.time;
  final endTime = trackPoints?.lastOrNull?.time;
  DateTime? date;
  if (startTime != null && endTime != null) {
    // TS takes `startTime.toISOString().substring(0, 10)` — a UTC calendar
    // date. Constructing from local time would drift by a day near
    // midnight, so convert to UTC first.
    final u = startTime.toUtc();
    date = DateTime.utc(u.year, u.month, u.day);
  }

  final metrics = computeTrailMetrics(gpx);

  // WR-08 / D-10: zero whole seconds is "no value", never a stored 0.
  final movingDurationSeconds =
      movingDuration != null && movingDuration.inSeconds > 0
      ? movingDuration.inSeconds.toDouble()
      : null;

  // Bounding box from the track. Normally computed server-side on save, but
  // an unsaved trail has none — set it here (mirroring gpx_util.ts:78-87)
  // so consumers can frame the whole track; otherwise leave the model's `0`
  // defaults.
  final hasFiniteBounds = metrics.minLat.isFinite && metrics.maxLat.isFinite;

  return Trail(
    id: '',
    name: name,
    description: description,
    lat: startPoint?.lat,
    lon: startPoint?.lon,
    date: date,
    distance: metrics.distance,
    elevationGain: metrics.elevationGain,
    elevationLoss: metrics.elevationLoss,
    duration: metrics.durationMs / 1000,
    movingDuration: movingDurationSeconds,
    minLat: hasFiniteBounds ? metrics.minLat : 0,
    maxLat: hasFiniteBounds ? metrics.maxLat : 0,
    minLon: hasFiniteBounds ? metrics.minLon : 0,
    maxLon: hasFiniteBounds ? metrics.maxLon : 0,
    created: now,
    updated: now,
    expand: TrailExpand(
      waypointsViaTrail: waypoints,
      gpx: gpx,
      gpxData: gpxData,
    ),
  );
}
