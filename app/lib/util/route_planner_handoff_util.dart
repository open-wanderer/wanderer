import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:gpx/gpx.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/planned_gpx_provider.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/util/gpx_util.dart';
import 'package:wanderer/util/trail_import_util.dart';
import 'package:wanderer/util/valhalla_util.dart';

/// Builds a fresh, ele-merged [Gpx] by zipping [heights] onto [shape]
/// index-for-index (a public copy of `ElevationTab._buildEleMergedGpx`).
/// Must merge against [shape] (the downsampled array), not the original
/// points, so indices stay aligned once a route exceeds Valhalla's 500-point
/// shape cap.
///
/// Returns a bare (trackless) [Gpx] when [shape] is empty.
Gpx mergeHeightsIntoGpx(List<Map<String, double>> shape, List<num> heights) {
  final gpx = Gpx();
  if (shape.isEmpty) return gpx;
  gpx.trks = [
    Trk(
      trksegs: [
        Trkseg(
          trkpts: [
            for (var i = 0; i < shape.length; i++)
              Wpt(
                lat: shape[i]['lat'],
                lon: shape[i]['lon'],
                ele: i < heights.length ? heights[i].toDouble() : null,
              ),
          ],
        ),
      ],
    ),
  ];
  return gpx;
}

Future<Trail> buildDraftTrail(
  WidgetRef ref,
  Gpx finalGpx, {
  String? category,
}) async {
  final xml = GpxWriter().asString(finalGpx);
  final formData = FormData.fromMap({
    'file': MultipartFile.fromString(
      xml,
      filename: 'route.gpx',
      contentType: DioMediaType('application', 'gpx+xml'),
    ),
  });
  Trail trail = await convertGpxToTrail(ref, formData);

  trail = trail.copyWith(
    category: category,
    expand: (trail.expand ?? const TrailExpand()).copyWith(gpx: finalGpx),
  );

  return trail;
}

/// Builds the final, ele-merged [Gpx] for the current planner session
/// (extracted from [finishPlanning] so the edit-mode handoff can pop this
/// same value instead of forward-pushing a draft trail).
///
/// Reads the final [plannedGpxProvider] snapshot and attempts a one-time
/// `POST /valhalla/height` elevation merge — best-effort; any failure
/// degrades to the pre-elevation `Gpx` with no error UI. Returns the bare
/// `Gpx` unchanged below 2 points (the Finish action is already disabled
/// below that at the call site).
Future<Gpx> buildFinalPlannedGpx(WidgetRef ref) async {
  final gpx = ref.read(plannedGpxProvider);
  final points = gpx.allPoints;
  if (points.length < 2) return gpx;

  final shape = buildNavShape(points);
  var finalGpx = gpx; // fallback if the height fetch fails
  try {
    final response = await ref
        .read(apiProvider)
        .post('/valhalla/height', data: {'shape': shape});
    final heights = (response.data['height'] as List).cast<num>();
    finalGpx = mergeHeightsIntoGpx(shape, heights);
  } catch (_) {
    // Silent fallback: proceed with the pre-elevation Gpx.
  }

  return finalGpx;
}

/// Orchestrates the Route Planner's "Finish planning" handoff — the plain
/// GPX-import forward-push path.
///
/// Builds the final ele-merged [Gpx] via [buildFinalPlannedGpx], resolves a
/// category pre-fill via [categoryForTravelProfile] (may be `null`), builds
/// the draft [Trail] via [buildDraftTrail], then reuses the existing
/// GPX-import handoff mechanism ([pendingImportedTrail] +
/// `navContext.push('/trail/create/edit')`).
Future<void> finishPlanning({
  required WidgetRef ref,
  required BuildContext navContext,
}) async {
  final anchorsState = ref.read(routeAnchorsProvider);
  final travelProfile = anchorsState.travelProfile;
  final finalGpx = await buildFinalPlannedGpx(ref);

  final categories = ref.read(categoryProvider).value ?? const [];
  final categoryId = categoryForTravelProfile(travelProfile, categories);

  final draftTrail = await buildDraftTrail(ref, finalGpx, category: categoryId);

  pendingImportedTrail = draftTrail;
  if (!navContext.mounted) return;
  navContext.push('/trail/create/edit', extra: draftTrail);
}

/// Derives segment-boundary [ml.Geographic] anchors from an existing track,
/// mirroring web's `initRouteAnchors` (`+page.svelte:553-577`): one anchor
/// at the first point of every `trkseg`, plus the last point of the final
/// segment. No interior sampling, no reverse-geocoding at seed time.
///
/// Returns an empty list for a trackless (no `trks`) [gpx].
List<ml.Geographic> anchorsFromTrack(Gpx gpx) {
  final segs = gpx.trks.isNotEmpty ? gpx.trks.first.trksegs : const <Trkseg>[];
  // Only points with both lat and lon count; a malformed trkpt is dropped
  // rather than force-unwrapped, and a trailing empty segment no longer
  // swallows the true final point of the segment before it.
  final nonEmpty = segs
      .map(
        (s) => s.trkpts.where((p) => p.lat != null && p.lon != null).toList(),
      )
      .where((pts) => pts.isNotEmpty)
      .toList();
  final out = <ml.Geographic>[];
  for (var i = 0; i < nonEmpty.length; i++) {
    final pts = nonEmpty[i];
    out.add(ml.Geographic(lat: pts.first.lat!, lon: pts.first.lon!));
    if (i == nonEmpty.length - 1) {
      out.add(ml.Geographic(lat: pts.last.lat!, lon: pts.last.lon!));
    }
  }
  return out;
}

/// Derives the full-resolution polyline for each inter-anchor span produced
/// by [anchorsFromTrack].
///
/// Uses the track's own recorded points rather than a straight line +
/// immediate Valhalla resolve — an earlier version did that and silently
/// snapped off-road recordings onto nearby roads before the user touched
/// anything.
///
/// Anchors are located by exact coordinate match via a forward-only search
/// cursor (this and [anchorsFromTrack] traverse points in the same order),
/// so a track that crosses/retraces itself is sliced in traversal order
/// rather than matching an earlier occurrence of the same coordinate.
///
/// Returns one polyline per consecutive anchor pair; falls back to a
/// straight 2-point line for any remaining pair if a coordinate is
/// unexpectedly not found (defensive).
List<List<ml.Geographic>> segmentPolylinesFromTrack(
  Gpx gpx,
  List<ml.Geographic> anchors,
) {
  if (anchors.length < 2) return const [];

  final segs = gpx.trks.isNotEmpty ? gpx.trks.first.trksegs : const <Trkseg>[];
  final allPoints = [
    for (final seg in segs)
      for (final p in seg.trkpts)
        if (p.lat != null && p.lon != null)
          ml.Geographic(lat: p.lat!, lon: p.lon!),
  ];

  final polylines = <List<ml.Geographic>>[];
  var cursor = 0;
  var prevIndex = 0;
  for (var i = 0; i < anchors.length; i++) {
    final anchor = anchors[i];
    var idx = -1;
    for (var j = cursor; j < allPoints.length; j++) {
      if (allPoints[j].lat == anchor.lat && allPoints[j].lon == anchor.lon) {
        idx = j;
        break;
      }
    }
    if (idx == -1) {
      for (var k = i; k < anchors.length - 1; k++) {
        polylines.add([anchors[k], anchors[k + 1]]);
      }
      return polylines;
    }
    if (i > 0) {
      polylines.add(allPoints.sublist(prevIndex, idx + 1));
    }
    prevIndex = idx;
    cursor = idx + 1;
  }
  return polylines;
}

/// Merges a Route Planner edit-mode result [finalGpx] onto an [existing]
/// in-memory [Trail], modeled on [buildDraftTrail] but as a `copyWith` —
/// every non-track field (title, description, id, visibility, photos,
/// waypoints, category) carries through untouched.
///
/// Sets both `expand.gpxData` and `expand.gpx` — setting only one produces a
/// trail that saves with no track, or previews incorrectly.
///
/// lat/lon/bounds are recomputed from [finalGpx]'s bounds when available,
/// falling back to [existing]'s prior values when the merged route has no
/// track (defensive; the Finish action guarantees >=2 anchors).
Trail mergeRouteIntoTrail(
  Trail existing,
  Gpx finalGpx, {
  double? estimatedDurationSeconds,
}) {
  final xml = GpxWriter().asString(finalGpx);
  final bounds = finalGpx.getBounds();
  return existing.copyWith(
    // An edited route's Gpx still carries no `time`, so re-derive `duration`
    // from the planner's own estimate rather than the stale pre-edit value.
    duration: estimatedDurationSeconds ?? existing.duration,
    lat: bounds != null
        ? (bounds.latitudeNorth + bounds.latitudeSouth) / 2
        : existing.lat,
    lon: bounds != null
        ? (bounds.longitudeEast + bounds.longitudeWest) / 2
        : existing.lon,
    maxLat: bounds?.latitudeNorth ?? existing.maxLat,
    minLat: bounds?.latitudeSouth ?? existing.minLat,
    maxLon: bounds?.longitudeEast ?? existing.maxLon,
    minLon: bounds?.longitudeWest ?? existing.minLon,
    expand: (existing.expand ?? const TrailExpand()).copyWith(
      gpx: finalGpx,
      gpxData: xml,
    ),
  );
}
