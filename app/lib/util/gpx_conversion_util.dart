import 'dart:math';

import 'package:gpx/gpx.dart';
import 'package:maplibre/maplibre.dart';

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
