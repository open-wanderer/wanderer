import 'package:flutter/foundation.dart' show compute;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/util/gpx/conversion.dart';
import 'package:wanderer/util/gpx/gpx.dart';
import 'package:wanderer/util/gpx/replace_elevations.dart';
import 'package:wanderer/util/route/planner_handoff.dart';

/// A captured track after [applyTrackTransforms], with the transforms that
/// actually took effect. Both transforms fall back silently when their
/// network call fails, so `followsRoads`/`heightsRecalculated` can be false
/// even when the caller asked for them.
typedef TransformedTrack = ({
  Gpx gpx,
  String xml,
  bool followsRoads,
  bool heightsRecalculated,
});

/// Applies the post-capture "Follow roads" and "Recalculate heights"
/// transforms to [originalXml], the track exactly as it was recorded or
/// imported. The trail editor re-runs it from the kept original whenever the
/// user flips either switch, so switching both off restores the capture.
///
/// Snapping runs before the height fetch so elevation reflects the final
/// shape. A snap replaces the geometry through [mergeHeightsIntoGpx], keeping
/// only the first/last point time (enough for the GPX-derived duration) and
/// the source document's waypoints and names. Heights alone write the new
/// altitudes into the original text, so per-point times, extensions and
/// route anchor marks all survive.
///
/// [costing] is the Valhalla profile the snap matches against.
Future<TransformedTrack> applyTrackTransforms(
  WidgetRef ref, {
  required String originalXml,
  required bool followRoads,
  required bool recalcHeights,
  required String costing,
}) async {
  final source = await compute(parseGpxSafely, originalXml);
  final unchanged = (
    gpx: source,
    xml: originalXml,
    followsRoads: false,
    heightsRecalculated: false,
  );
  final points = source.allPoints;
  if (points.length < 2 || (!followRoads && !recalcHeights)) {
    return unchanged;
  }

  final shape = [
    for (final point in points) {'lat': point.lat, 'lon': point.lon},
  ];

  if (followRoads) {
    // `fallbackShape` is the full-resolution geometry, never the ≤500-point
    // request hint, so a failed snap can't persist a decimated track.
    final snap = await snapShapeToRoadsResult(
      ref,
      buildNavShape(points),
      costing,
      fallbackShape: shape,
    );
    if (snap.snapped) {
      final heights = recalcHeights
          ? await fetchHeightsForShape(ref, snap.shape)
          : const <num>[];
      final wpts = source.allWaypoints;
      final gpx = mergeHeightsIntoGpx(
        snap.shape,
        heights,
        startTime: wpts.firstOrNull?.time,
        endTime: wpts.lastOrNull?.time,
        source: source,
      );
      return (
        gpx: gpx,
        xml: await compute(serializeGpxToXml, gpx),
        followsRoads: true,
        heightsRecalculated: heights.isNotEmpty,
      );
    }
  }

  if (!recalcHeights) return unchanged;
  final heights = await fetchHeightsForShape(ref, shape);
  if (heights.isEmpty) return unchanged;
  final xml = await compute(replaceTrackElevationsMessage, (
    originalXml,
    heights,
  ));
  return (
    gpx: await compute(parseGpxSafely, xml),
    xml: xml,
    followsRoads: false,
    heightsRecalculated: true,
  );
}

/// Measures [track] the way a fresh capture is measured, for
/// [withTransformedTrack]. Runs [trailFromGpx] off the UI thread: the metrics
/// walk every track point.
Future<Trail> measureTrack(TransformedTrack track) {
  // An explicit args record, never a capturing closure.
  return compute(trailFromGpxIsolate, (
    gpx: track.gpx,
    fallbackName: null,
    movingDuration: null,
    gpxData: null,
  ));
}

/// Swaps [track] in as [trail]'s track, taking every value derived from it
/// (distance, elevation, duration, start point, bounds) from [measured], see
/// [measureTrack]. Everything else — name, description, category, waypoints,
/// photos, moving time — stays as it is, and so does the kept original, so
/// the transforms can be switched off again.
///
/// Synchronous on purpose: a caller that awaited the measurement applies this
/// to its trail as it is now, so an edit made meanwhile is not lost.
Trail withTransformedTrack(
  Trail trail,
  TransformedTrack track, {
  required Trail measured,
}) {
  return trail.copyWith(
    distance: measured.distance,
    elevationGain: measured.elevationGain,
    elevationLoss: measured.elevationLoss,
    duration: measured.duration,
    lat: measured.lat,
    lon: measured.lon,
    minLat: measured.minLat,
    maxLat: measured.maxLat,
    minLon: measured.minLon,
    maxLon: measured.maxLon,
    trackFollowsRoads: track.followsRoads,
    trackHeightsRecalculated: track.heightsRecalculated,
    expand: (trail.expand ?? const TrailExpand()).copyWith(
      gpx: track.gpx,
      gpxData: track.xml,
    ),
  );
}
