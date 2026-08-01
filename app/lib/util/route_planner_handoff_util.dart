import 'dart:math' show sqrt;

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
import 'package:wanderer/provider/subcategory_preference_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/util/gpx_util.dart';
import 'package:wanderer/util/route_travel_bucket.dart';
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

/// Builds a draft [Trail] from a planner/recording session's [finalGpx]
/// entirely on-device (PORT-03) — no convert-endpoint request. [category]/
/// [subcategory] pre-fill the trail's operator classification;
/// [durationSeconds] is the planner's Valhalla-estimated fallback (see
/// below); [movingDuration] is the D-11 recording hand-off.
///
/// [movingDuration] is the ONLY value a recording takes from its own
/// session — distance, elevation gain/loss and `duration` all come from the
/// ported computation (`trailFromGpx`, reached via the local trail builder
/// below) over the recorded GPX, so every persisted stat besides moving time
/// stays reproducible by a later recompute over the same GPX (D-11).
Future<Trail> buildDraftTrail(
  WidgetRef ref,
  Gpx finalGpx, {
  String? category,
  String? subcategory,
  double? durationSeconds,
  Duration? movingDuration,
}) async {
  // Still produced: `expand.gpxData` is what `form_data_util.dart` uploads
  // as the trail's track file on save.
  final xml = GpxWriter().asString(finalGpx);

  Trail trail = await buildLocalTrail(
    ref,
    finalGpx,
    movingDuration: movingDuration,
    gpxData: xml,
  );

  trail = trail.copyWith(
    category: category,
    // `''` clears the relation — matches trail_create_screen's own
    // subcategory-on-clear convention.
    subcategory: subcategory ?? '',
    expand: (trail.expand ?? const TrailExpand()).copyWith(gpx: finalGpx),
  );

  // A timeless GPX (planner output, or a recording saved through the
  // height/snap transforms) yields `duration == 0` from the LOCAL
  // `trailFromGpx` computation rather than from the server's `gpx2trail`.
  // Fall back to the caller's known-good value only in that case, so a GPX
  // that carried real timestamps keeps its own GPX-derived duration. This is
  // the planner's answer to what it writes into `duration`: the Valhalla
  // estimate is used only when the planned route carries no timestamps at
  // all, which is always the case for a never-traversed route.
  if (durationSeconds != null && durationSeconds > 0 && trail.duration <= 0) {
    trail = trail.copyWith(duration: durationSeconds);
  }

  return trail;
}

/// Builds the final, ele-merged [Gpx] for the current planner session
/// (extracted from [finishPlanning] so the edit-mode handoff can pop this
/// same value instead of forward-pushing a draft trail).
///
/// Builds the route's export [Gpx]: **one `Trkseg` per leg**, each starting
/// exactly on its opening anchor.
///
/// A leg omits its closing point — that point opens the next leg's `Trkseg`,
/// and only the final leg emits the route's last anchor. So the flattened
/// point stream stays duplicate-free while every `trkseg` boundary lands on
/// an anchor, which is precisely the pair of properties [anchorsFromTrack]
/// and [segmentPolylinesFromTrack] read back. Repeating both endpoints per
/// leg would also round-trip, but would leave a duplicated coordinate at
/// every boundary that compounds on each successive save/re-edit cycle.
///
/// That layout is the entire mechanism by which a route's structure survives
/// a save/reload — nothing anchor-related is persisted on `Trail`, so
/// [anchorsFromTrack] rebuilds anchors from `trkseg` boundaries on the way
/// back in. Emitting a single flattened segment here (as an earlier version
/// did, via [buildNavShape] + [mergeHeightsIntoGpx]) collapsed every
/// intermediate anchor to a bare start/end pair on re-edit. Deliberately does
/// NOT reuse [plannedGpxProvider], whose `skip(1)` layout puts boundaries one
/// point after each anchor and so is not round-trippable.
///
/// Elevations are read off each segment's already-fetched
/// [RouteSegment.elevationProfile]/[RouteSegment.elevations] — the route
/// planner resolves heights per segment as the user builds the route, so no
/// whole-route refetch is needed. Because `elevationProfile` comes from
/// [buildNavShape], which always preserves its input's first and last point,
/// every leg boundary lands exactly on an anchor coordinate — which is what
/// [segmentPolylinesFromTrack]'s exact-match slicing requires.
///
/// A segment whose fire-and-forget height fetch never landed is backfilled
/// here (in parallel, best-effort); one that still fails emits `ele: null`
/// rather than blocking the save. Returns a bare (trackless) [Gpx] when the
/// route has no legs — the Finish action is already disabled below 2 anchors
/// at the call site.
///
/// [snapCosting], when non-null, road-snaps every leg with at least 2 points
/// via the authenticated trace-route proxy, using [buildNavShape] as the
/// outbound request hint — mirrors `navigation_screen.dart`'s
/// `_saveRecordedTrack` pipeline. **After snapping, each leg's first and
/// last point is forced back onto its original opening/closing anchor
/// coordinates.** This is mandatory, not cosmetic: [anchorsFromTrack] and
/// [segmentPolylinesFromTrack] locate anchors by EXACT coordinate match, so
/// a snapped boundary that drifted even a fraction would make the route
/// unreconstructable — and silently collapse every intermediate anchor — on
/// re-edit. A snapped leg's previously-fetched elevations are invalidated
/// (its point indices no longer align with the pre-snap heights) and
/// re-fetched below regardless of [refetchAllHeights].
///
/// [refetchAllHeights] set to `true` refetches heights for every leg over
/// its final (possibly snapped) shape; `false` keeps the existing
/// fetch-only-what's-missing behaviour. Both flags default `false`, and
/// both underlying network steps are skipped entirely when their flag is
/// off — the combination that makes this function safe to call with no
/// network access at all (D-15's offline path). Snapping always runs before
/// heights are read, matching the recording path's own ordering, so
/// elevation reflects the final (possibly snapped) shape.
Future<Gpx> buildFinalPlannedGpx(
  WidgetRef ref, {
  bool refetchAllHeights = false,
  String? snapCosting,
}) async {
  final legs = ref.read(routeAnchorsProvider).orderedSegments;
  if (legs.isEmpty) return Gpx();

  // Prefer the height-resolved point set; fall back to the raw polyline for
  // a leg whose fetch is still pending or failed. Mutated in place below by
  // the snap step, so this must stay a growable per-leg list, not a const.
  final legPoints = [for (final s in legs) s.elevationProfile ?? s.polyline];
  final legElevations = [for (final s in legs) s.elevations];

  final costing = snapCosting;
  if (costing != null) {
    final toSnap = <int>[
      for (var i = 0; i < legs.length; i++)
        if (legPoints[i].length >= 2) i,
    ];
    if (toSnap.isNotEmpty) {
      final snapped = await Future.wait([
        for (final i in toSnap)
          snapShapeToRoads(ref, buildNavShape(legPoints[i]), costing),
      ]);
      for (var k = 0; k < toSnap.length; k++) {
        final i = toSnap[k];
        final shape = snapped[k];
        if (shape.length < 2) continue;
        final original = legPoints[i];
        final rePinned = [
          for (final p in shape) ml.Geographic(lat: p['lat']!, lon: p['lon']!),
        ];
        // Force the boundary back onto the leg's real anchors — see this
        // function's doc comment for why this is mandatory.
        rePinned[0] = original.first;
        rePinned[rePinned.length - 1] = original.last;
        legPoints[i] = rePinned;
        legElevations[i] = null;
      }
    }
  }

  final pending = refetchAllHeights
      ? [for (var i = 0; i < legs.length; i++) i]
      : <int>[
          for (var i = 0; i < legs.length; i++)
            if (legElevations[i] == null) i,
        ];
  if (pending.isNotEmpty) {
    final fetched = await Future.wait([
      for (final i in pending)
        fetchHeightsForShape(ref, [
          for (final p in legPoints[i]) {'lat': p.lat, 'lon': p.lon},
        ]),
    ]);
    for (var k = 0; k < pending.length; k++) {
      final heights = fetched[k];
      // fetchHeightsForShape returns an empty list on any failure — leaves
      // whatever the leg already carried (null for a newly-snapped/pending
      // leg, or its prior good elevations for a refetch that failed).
      if (heights.isEmpty) continue;
      legElevations[pending[k]] = [for (final h in heights) h.toDouble()];
    }
  }

  final gpx = Gpx();
  gpx.trks = [
    Trk(
      trksegs: [
        for (var i = 0; i < legs.length; i++)
          Trkseg(
            trkpts: [
              // Drop the closing point of every leg but the last: it is the
              // next leg's opening point. A degenerate 1-point leg keeps its
              // single point rather than emitting an empty trkseg, which
              // anchorsFromTrack would skip and so lose that anchor.
              for (
                var j = 0;
                j <
                    (i == legs.length - 1 || legPoints[i].length < 2
                        ? legPoints[i].length
                        : legPoints[i].length - 1);
                j++
              )
                Wpt(
                  lat: legPoints[i][j].lat,
                  lon: legPoints[i][j].lon,
                  ele: j < (legElevations[i]?.length ?? 0)
                      ? legElevations[i]![j]
                      : null,
                ),
            ],
          ),
      ],
    ),
  ];
  return gpx;
}

/// Orchestrates the Route Planner's "Finish planning" handoff — the plain
/// GPX-import forward-push path.
///
/// Builds the final ele-merged [Gpx] via [buildFinalPlannedGpx], resolves the
/// exact bucket the session is running ([bucketForState]) and maps it back to
/// an operator (sub)category pre-fill via [categorySelectionForBucket] (both
/// may be `null`), builds the draft [Trail] via [buildDraftTrail], then reuses
/// the existing GPX-import handoff mechanism ([pendingImportedTrail] +
/// `navContext.push('/trail/create/edit')`).
///
/// A session running a costing outside the 5 picker buckets simply yields no
/// category pre-fill — no default bucket is ever substituted.
///
/// [recalcHeights]/[followRoads] are the two toggles resolved by the shared
/// online gate (`track_save_options_util.dart`) — both default `false`,
/// matching that gate's offline/declined-both outcome. [followRoads] maps
/// onto [buildFinalPlannedGpx]'s `snapCosting`, derived from the session's
/// own bucket, falling back to `'pedestrian'` when the bucket doesn't
/// resolve one — the same fallback [costingForTrail] itself uses. No
/// `movingDuration` is ever passed here: a planned route was never
/// traversed (unchanged since plan 34-05).
Future<void> finishPlanning({
  required WidgetRef ref,
  required BuildContext navContext,
  bool recalcHeights = false,
  bool followRoads = false,
}) async {
  final anchorsState = ref.read(routeAnchorsProvider);
  final bucket = bucketForState(
    anchorsState.travelProfile,
    anchorsState.costingOptions,
  );
  final finalGpx = await buildFinalPlannedGpx(
    ref,
    refetchAllHeights: recalcHeights,
    snapCosting: followRoads ? (bucket?.costing ?? 'pedestrian') : null,
  );

  final categories = ref.read(categoryProvider).value ?? const [];
  final subcategories = ref.read(subcategoryProvider);
  final selection = bucket == null
      ? null
      : categorySelectionForBucket(
          bucket,
          categories,
          subcategories,
          // Never auto-assign a subcategory the user has hidden.
          subcategoryPrefs:
              ref.read(subcategoryPreferenceProvider).value ?? const [],
        );

  final draftTrail = await buildDraftTrail(
    ref,
    finalGpx,
    category: selection?.categoryId,
    subcategory: selection?.subcategoryId,
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
