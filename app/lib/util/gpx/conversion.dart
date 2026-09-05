import 'dart:math';

import 'package:collection/collection.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
// `hide GpxReader` so the vendored, tolerant reader below is unambiguously
// the one in scope — the models (Gpx, Wpt, Trk, Trkseg, ...) and GpxWriter
// still come from the published package, so types are shared with the rest of
// the app.
import 'package:gpx/gpx.dart' hide GpxReader;
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/theme/icons.dart';
import 'package:wanderer/vendor/gpx/gpx_reader.dart';

/// The single sanctioned parse entry point for any GPX this app did not
/// itself produce via [GpxWriter] — imported files, shared/received tracks,
/// or any other third-party GPX source.
///
/// Uses the vendored [GpxReader] (`lib/vendor/gpx/gpx_reader.dart`), which
/// coerces with `tryParse` instead of the published package's throwing
/// `parse`. That is what makes this function safe: real exporter output
/// routinely carries `<hdop></hdop>`, `<pdop>N/A</pdop>`, `<time></time>`,
/// the non-standard `<email>user@example.com</email>` text form, and even a
/// `<trkpt>` missing `lat`/`lon` — every one of which aborted the entire
/// document with the published reader. They now degrade to "no reading" on
/// the affected field and leave the rest of the track intact.
///
/// This replaced a pair of pre-parse regex sanitizers. Rewriting the XML
/// could only ever chase symptoms one tag at a time and risked corrupting
/// CDATA and comments; fixing the coercion where it happens covers every
/// numerically-parsed element at once, including the `<number>` and `<year>`
/// tags the sanitizer deliberately would not touch.
///
/// This is the ONLY place in `lib/` that may construct a [GpxReader] —
/// import, the server-download path (`trail_provider.dart`) and the
/// offline-cache read (`trail_entity.dart`) all route through here, and
/// `gpx_conversion_util_test.dart`'s "single GpxReader call site" gate fails
/// the build if a fourth appears. Bypassing this function silently opts a
/// call site out of that tolerance: the two non-import sites previously did,
/// which made a server-authored track fail to open (swallowed by a broad
/// `catch`) and, once cached, permanently un-openable offline.
Gpx parseGpxSafely(String xml) {
  return GpxReader().fromString(xml);
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

/// Great-circle distance between two waypoints, in METRES.
///
/// The single haversine in the app. Delegates to `package:geobase`'s
/// `SphericalGreatCircle` (re-exported via `package:maplibre`), verified
/// formula-identical to the TS `haversineDistance` that the Dart/TS parity
/// corpus pins — so this is also the function that must not drift.
///
/// Returns `double.nan` when either coordinate is null, reproducing the TS
/// behaviour where an `undefined` `$.lat` propagates `NaN` out of
/// `haversineDistance`. Do NOT "fix" this to `0`: a fabricated 0 would be
/// silently absorbed into a distance total, whereas NaN is caught by
/// `addAndFilter`'s `Number.isFinite` guard and skipped.
///
/// Two hand-rolled copies previously existed — one here inside
/// `GpxMetricsComputation`, one in `elevation_profile.dart` (its own
/// `const r = 6371.0` haversine, which agreed to ~1e-10 m but was free to
/// drift) — plus an open-coded `SphericalGreatCircle` loop in
/// `GpxUtils.distanceFromStartTo`. All three now route here.
/// Whether [p] carries a position that can actually be used.
///
/// THE single rule for "is this a real point?", deliberately shared by every
/// consumer — the metrics accumulators, the trail's start coordinate, the
/// waypoint mapping, the centroid and the bounding box. A point without a
/// usable position is DISCARDED everywhere; it is never defaulted to (0, 0),
/// which would plant it in the Gulf of Guinea and drag the centroid, the
/// bounding box and the map pin with it.
///
/// `isFinite`, not merely non-null: the vendored reader rejects non-finite
/// coordinate ATTRIBUTES, but a `Wpt` can also be built programmatically, and
/// `NaN`/`Infinity` silently poison every accumulator they reach.
///
/// Three separate answers to this question previously coexisted — null-only in
/// one place, `?? 0` in another, unfiltered in a third — which is how a single
/// malformed point could zero a track's distance, fabricate 2 km of climb, and
/// still show up as a marker off the coast of Africa.
bool hasUsablePosition(Wpt p) =>
    p.lat != null && p.lon != null && p.lat!.isFinite && p.lon!.isFinite;

double haversineMeters(Wpt a, Wpt b) {
  if (a.lat == null || a.lon == null || b.lat == null || b.lon == null) {
    return double.nan;
  }
  return SphericalGreatCircle(
    Geographic(lat: a.lat!, lon: a.lon!),
  ).distanceTo(Geographic(lat: b.lat!, lon: b.lon!));
}

/// Line-for-line Dart port of `gpx-metrics-computation.ts`'s
/// `GpxMetricsComputation` class — the defer-then-publish elevation noise
/// filter plus the threshold-gated distance-smoothing accumulator.
///
/// The raw per-point cumulative-distance array from the TS original is
/// deliberately NOT ported: its only consumer is the web
/// trail-edit crop slider, which has no Dart equivalent, so porting it
/// would be dead code.
class GpxMetricsComputation {
  final double
  _thresholdXYm; // Distance threshold for filtering on the XY axis (latitude / longitude)
  final double
  _thresholdZm; // Distance threshold for filtering on the Z axis (elevation)

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
  // NOT REPORTED: no consumer reads this for a trail's distance anymore —
  // computeTrailMetrics reports the raw totalDistance instead.
  // Kept, and _thresholdXYm/GpxMetricsComputation(5, 5) untouched, because
  // _lastFilteredPointXY sits in a class whose other anchors are
  // elevation-critical, and a future speed-aware filter would build on it.
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

  double _haversine(Wpt a, Wpt b) => haversineMeters(a, b);

  /// Line-for-line port of `addAndFilter` (`gpx-metrics-computation.ts:97-235`).
  /// Do not simplify this into a plain accumulator — that reintroduces a
  /// defect this shape exists to avoid.
  void addAndFilter(Wpt point) {
    // A point with no usable POSITION is not a trackpoint, and is dropped
    // before it can touch any accumulator.
    //
    // This guard belongs here, at the top, covering every point — an earlier
    // version sat inside the initialisation branch below and checked only for
    // `null`, which left two holes:
    //
    //   * a first `<trkpt lat="NaN" lon="NaN">` sailed through (`tryParse`
    //     accepts "NaN"/"Infinity"), re-creating the exact poisoning it was
    //     written to stop: every haversine from that anchor is NaN, so
    //     `smoothedDistance >= _thresholdXYm` never fires, the filtered anchor
    //     is never replaced, and a ~2 km track measured 0.0 m;
    //   * a coordinate-less point MID-track was never checked at all. Its
    //     elevation still entered the diff chain, so a single
    //     `<trkpt ele="3000">` dropped into a flat 1000 m track fabricated
    //     2000 m of gain AND 2000 m of loss.
    //
    // Skipping also keeps this in step with every other consumer of the same
    // data — `GpxUtils.allWaypoints`, the elevation chart, and
    // `importTrailFile`'s usability check all ignore such points — so the
    // chart and the stats cannot disagree about the same file.
    //
    // No parity risk: the TS `Waypoint` falls lat/lon back to -1, so a
    // position-less point cannot arise on that side, and no corpus fixture
    // contains one.
    if (!hasUsablePosition(point)) {
      return;
    }

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
    // only on the discard path, so a monotonic low-horizontal climb is
    // never affected.
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
/// no `hash` (public metrics only; Dart internals may differ from TS).
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
/// `hash`.
///
/// One [GpxMetricsComputation] instance is constructed and shared across
/// EVERY [Trk] and EVERY [Trkseg] in [gpx] — no per-segment anchor reset —
/// so a multi-leg planner route measures through its anchors.
///
/// The point loop starts at index 0 and the centroid divides by
/// the same count it summed, `summedPointCount` — not by a
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

        // A position-less point contributed 0 to the sums but still
        // incremented the divisor, dragging the centroid toward (0, 0) in
        // proportion to how many such points the file had. Discarded here for
        // the same reason addAndFilter discards it — one rule, one predicate.
        if (!hasUsablePosition(point)) continue;

        totalLat += point.lat!;
        totalLon += point.lon!;
        summedPointCount++;

        minLat = min(minLat, point.lat!);
        maxLat = max(maxLat, point.lat!);
        minLon = min(minLon, point.lon!);
        maxLon = max(maxLon, point.lon!);
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
    distance: metrics.totalDistance,
    elevationGain: metrics.finalElevationGain,
    elevationLoss: metrics.finalElevationLoss,
    durationMs: totalDurationMs.abs(),
    pointCount: summedPointCount,
  );
}

/// Line-for-line Dart port of `gpx2trail`'s trail assembly
/// (`web/src/lib/util/gpx_util.ts`) — the draft-trail step.
/// Builds a complete, unsaved [Trail] from a parsed [gpx] with no network
/// call: name, description, waypoints, start coordinates, date,
/// distance, elevation gain/loss, duration and bounding box.
///
/// [movingDuration] is an override: when a caller (a recording session)
/// supplies it, it becomes `Trail.movingDuration` — UNLESS it rounds down to
/// zero whole seconds, in which case it is treated as absent (`null`).
///
/// That zero-to-null mapping is the "no value" state, and it has to live
/// here rather than at each call site: `NavigationStats.elapsed` starts at
/// `Duration.zero` and stays there until the 1-second tick begins, so a
/// recording saved immediately passed a zero elapsed straight through to
/// `moving_duration = 0` — precisely the state `util/trail/form_data.dart`'s own
/// comment says must never be written ("sending an empty string for an
/// absent value would write 0 into PocketBase and defeat the 'no value'
/// state"). Its write guard is `!= null`, not `> 0`, so nothing downstream
/// catches it, and the display rule's `> 0` fallback then MASKS the bad row
/// until something else reads the field. Note the truncation is the real
/// boundary, not `Duration.zero`: a 500 ms elapsed also yields 0 whole
/// seconds, which a `> Duration.zero` check at the call site would miss
///
/// `Trail.duration` NEVER
/// comes from it — `duration` always comes from the GPX's own first/last
/// trkpt timestamps, so every persisted stat except moving time stays
/// reproducible by a later recompute over the same GPX. Omitting the
/// parameter leaves `movingDuration` null.
///
/// [gpxData] is the raw GPX string [gpx] was parsed from. When supplied it
/// is carried on `expand.gpxData`, which `util/trail/form_data.dart` uploads as
/// the trail's track file on save — a caller that omits it produces a trail
/// that saves with no GPX.
///
/// `id`/`created`/`updated` are placeholders (`''`/`DateTime.now()`),
/// mirroring `import_trail_file.dart`'s `convertGpxToTrail` convention for a
/// not-yet-persisted trail — that function's own placeholder-injection is
/// unchanged by this plan; only its data source moves here in a later plan.
/// Top-level `compute()` entry point for [trailFromGpx] — takes one sendable
/// record so no capturing closure (which would drag its enclosing scope,
/// WidgetRef and all, into the isolate message) is ever needed at call sites.
Trail trailFromGpxIsolate(
  ({Gpx gpx, String? fallbackName, Duration? movingDuration, String? gpxData})
  args,
) {
  return trailFromGpx(
    args.gpx,
    fallbackName: args.fallbackName,
    movingDuration: args.movingDuration,
    gpxData: args.gpxData,
  );
}

/// Top-level `compute()` entry point for GPX serialization — the writer walk
/// over a full track is pure CPU that must not run on the UI thread at the
/// recording-Stop / planner-Finish / import moments that call it.
String serializeGpxToXml(Gpx gpx) => GpxWriter().asString(gpx);

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

  // A <wpt> with no usable position is DROPPED, not planted at (0, 0).
  // The old `?? 0` produced a marker in the Gulf of Guinea that the user could
  // neither explain nor remove, and re-serialised as an attribute-less <wpt>
  // that the next reader would reject.
  final waypoints = <Waypoint>[
    for (final wpt in gpx.wpts)
      if (hasUsablePosition(wpt))
        Waypoint(
          id: '',
          lat: wpt.lat!,
          lon: wpt.lon!,
          name: wpt.name ?? '',
          description: wpt.desc ?? '',
          // Closed-set lookup: an unknown or hostile
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
  // The FIRST USABLE point, not merely the first. A leading malformed
  // point otherwise produced a trail with null lat/lon — no map pin, no
  // reverse geocode — even though every other point in the file was fine.
  final startPoint =
      trackPoints?.firstWhereOrNull(hasUsablePosition) ??
      routePoints?.firstWhereOrNull(hasUsablePosition);

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

  // Zero whole seconds is "no value", never a stored 0.
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
