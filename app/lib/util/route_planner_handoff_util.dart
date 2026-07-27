import 'dart:math' show sqrt;

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
/// [startTime]/[endTime], when given, are stamped onto the first/last
/// trackpoint only (interior points have no real time once [shape] has been
/// downsampled/snapped). The server's duration calc
/// (`gpx.ts`'s `getTotals`) only reads a segment's first and last point time,
/// so this is sufficient to recover a real recorded duration — pass `null`
/// for a planned (never-traversed) route, where no such times exist.
///
/// Returns a bare (trackless) [Gpx] when [shape] is empty.
Gpx mergeHeightsIntoGpx(
  List<Map<String, double>> shape,
  List<num> heights, {
  DateTime? startTime,
  DateTime? endTime,
}) {
  final gpx = Gpx();
  if (shape.isEmpty) return gpx;
  final lastIndex = shape.length - 1;
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
                time: i == 0
                    ? startTime
                    : (i == lastIndex ? endTime : null),
              ),
          ],
        ),
      ],
    ),
  ];
  return gpx;
}

/// Computes the bounding-box diagonal (in degrees) of a `{lat,lon}` shape
/// array. Pure helper backing [snapResultAcceptable].
double _bboxDiagonal(List<Map<String, double>> points) {
  var minLat = points.first['lat']!;
  var maxLat = minLat;
  var minLon = points.first['lon']!;
  var maxLon = minLon;
  for (final p in points) {
    final lat = p['lat']!;
    final lon = p['lon']!;
    if (lat < minLat) minLat = lat;
    if (lat > maxLat) maxLat = lat;
    if (lon < minLon) minLon = lon;
    if (lon > maxLon) maxLon = lon;
  }
  final dLat = maxLat - minLat;
  final dLon = maxLon - minLon;
  return sqrt(dLat * dLat + dLon * dLon);
}

/// Guards against Valhalla `trace_route`'s partial map-match truncation
/// (valhalla#4802), where a low-confidence trace can silently drop its tail
/// into `data.alternates` instead of `trip.legs`, yielding a materially
/// shorter snapped path.
///
/// PURE. Returns `false` when [snapped] is empty. Otherwise compares each
/// shape's bounding-box diagonal: [snapped] is rejected when its diagonal is
/// < 0.6x [original]'s. Point COUNT is deliberately not part of this check —
/// `trace_route` re-vertexes the path at Valhalla's own density, so a
/// shorter [snapped] list with a comparable bbox is expected and acceptable.
///
/// An empty (zero-length) [original] bbox is treated as trivially acceptable
/// (nothing to truncate).
bool snapResultAcceptable(
  List<Map<String, double>> original,
  List<Map<String, double>> snapped,
) {
  if (snapped.isEmpty) return false;
  if (original.isEmpty) return true;

  final originalDiagonal = _bboxDiagonal(original);
  if (originalDiagonal == 0) return true;

  final snappedDiagonal = _bboxDiagonal(snapped);
  return snappedDiagonal >= 0.6 * originalDiagonal;
}

/// Best-effort road-snap of a recorded [shape] via the authenticated
/// `POST /valhalla/trace-route` proxy, using [costing] (derived from the
/// trail's category via [costingForCategory] at the call site).
///
/// On any error/timeout, or when [snapResultAcceptable] rejects the result
/// as a partial map-match truncation, returns [shape] unchanged — mirrors
/// [buildFinalPlannedGpx]'s silent-fallback precedent (no toast, no
/// rethrow).
Future<List<Map<String, double>>> snapShapeToRoads(
  WidgetRef ref,
  List<Map<String, double>> shape,
  String costing,
) async {
  try {
    final response = await ref
        .read(apiProvider)
        .post(
          '/valhalla/trace-route',
          data: {'shape': shape, 'costing': costing},
        );
    final snapped = (response.data['shape'] as List)
        .cast<Map<String, dynamic>>()
        .map(
          (p) => {
            'lat': (p['lat'] as num).toDouble(),
            'lon': (p['lon'] as num).toDouble(),
          },
        )
        .toList();

    if (snapResultAcceptable(shape, snapped)) {
      return snapped;
    }
    return shape;
  } catch (e) {
    // Silent fallback: proceed with the pre-snap shape.
    return shape;
  }
}

/// Fetches elevation for a full-resolution recorded [shape] via
/// `POST /valhalla/height`, batching into ≤500-point chunks and
/// concatenating results — [buildNavShape]'s downsampling is appropriate for
/// a Valhalla routing *request* hint, but would silently truncate a *saved*
/// track's actual point count, which is what [shape] preserves here.
///
/// Best-effort: any chunk failing (network, non-2xx, malformed body, or a
/// response whose height count doesn't match the chunk it answered) drops
/// the whole result to an empty list, so [mergeHeightsIntoGpx] falls back to
/// no elevation rather than a partially-heighted track — mirrors
/// [snapShapeToRoads]'s silent-fallback precedent.
Future<List<num>> fetchHeightsForShape(
  WidgetRef ref,
  List<Map<String, double>> shape,
) async {
  if (shape.isEmpty) return const [];

  final heights = <num>[];
  try {
    for (var i = 0; i < shape.length; i += 500) {
      final chunk = shape.sublist(i, (i + 500).clamp(0, shape.length));
      final response = await ref
          .read(apiProvider)
          .post('/valhalla/height', data: {'shape': chunk});
      final chunkHeights = (response.data['height'] as List).cast<num>();
      if (chunkHeights.length != chunk.length) return const [];
      heights.addAll(chunkHeights);
    }
  } catch (_) {
    return const [];
  }
  return heights;
}

Future<Trail> buildDraftTrail(
  WidgetRef ref,
  Gpx finalGpx, {
  String? category,
  double? durationSeconds,
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

  // A timeless GPX (planner output, or a recording saved through the
  // height/snap transforms) yields `duration == 0` from the server's
  // timestamp-based `gpx2trail`. Fall back to the caller's known-good value
  // only in that case, so a GPX that carried real timestamps keeps the
  // server-derived duration.
  if (durationSeconds != null && durationSeconds > 0 && trail.duration <= 0) {
    trail = trail.copyWith(duration: durationSeconds);
  }

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

  final draftTrail = await buildDraftTrail(
    ref,
    finalGpx,
    category: categoryId,
    durationSeconds: anchorsState.estimatedDurationSeconds,
  );

  pendingImportedTrail = draftTrail;
  if (!navContext.mounted) return;
  navContext.pushReplacement('/trail/create/edit', extra: draftTrail);
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
