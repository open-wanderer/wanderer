import 'dart:math' show sqrt;

import 'package:flutter/foundation.dart' show compute;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:gpx/gpx.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/planned_gpx_provider.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/subcategory_preference_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/util/gpx/conversion.dart' show serializeGpxToXml;
import 'package:wanderer/util/gpx/gpx.dart';
import 'package:wanderer/models/route_travel_bucket.dart';
import 'package:wanderer/actions/import_trail_file.dart';
import 'package:wanderer/util/route/valhalla.dart';

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
/// [source], when given, is the ORIGINAL parsed document the [shape] was
/// derived from. Its non-track content — `metadata` (name/description/time),
/// `wpts` (the file's own `<wpt>` markers), `rtes`, `extensions`,
/// `version`/`creator`, and the first `<trk>`'s `name`/`desc`/`cmt`/`src`/
/// `number`/`type`/`links`/`extensions` — is carried onto the rebuilt
/// document, so only the track GEOMETRY is replaced.
///
/// This is mandatory for the file-import path: `trailFromGpx` reads
/// `metadata.name`, `metadata.desc`, `trks.first.name`, `rtes.first.name`
/// and `wpts` to build the draft trail, and the caller re-serialises the
/// result as the track file it uploads — so returning a bare document here
/// permanently discarded every imported waypoint, the GPX's own name, and
/// its description the moment either post-capture toggle was enabled.
/// Copying `rtes` through cannot double-count metrics: `computeTrailMetrics`
/// and `GpxMappingUtils.allPoints` both read `trks` only.
///
/// Omit [source] for a recording or a planner session, where there is no
/// source document and the geometry IS the whole document.
///
/// Returns a track-less [Gpx] when [shape] is empty — still carrying
/// [source]'s non-track content when one was given.
Gpx mergeHeightsIntoGpx(
  List<Map<String, double>> shape,
  List<num> heights, {
  DateTime? startTime,
  DateTime? endTime,
  Gpx? source,
}) {
  final gpx = Gpx();
  if (source != null) {
    gpx.version = source.version;
    gpx.creator = source.creator;
    gpx.metadata = source.metadata;
    // COPIES, not the source's own list objects. These were assigned by
    // reference, so the merged document and the document it was derived from
    // shared one growable list — `identical(merged.wpts, source.wpts)` was
    // true, and appending a waypoint to the merged trail silently appended it
    // to the original too.
    gpx.wpts = List.of(source.wpts);
    gpx.rtes = List.of(source.rtes);
    gpx.extensions = source.extensions;
  }
  if (shape.isEmpty) return gpx;

  final lastIndex = shape.length - 1;
  Wpt pointAt(int i) => Wpt(
    lat: shape[i]['lat'],
    lon: shape[i]['lon'],
    ele: i < heights.length ? heights[i].toDouble() : null,
    time: i == 0 ? startTime : (i == lastIndex ? endTime : null),
  );

  // Preserve the source's <trkseg> boundaries when the transform was
  // point-for-point.
  //
  // The shape arrives flattened across every segment, and this used to always
  // emit a single segment. For the route planner that is destructive rather
  // than cosmetic: each leg is its own <trkseg>, and `anchorsFromTrack`
  // recovers anchors from those boundaries — so a 3-anchor route came back as
  // a start/end pair once the user enabled either post-capture toggle.
  //
  // Only safe when the transform preserved the point count: a road-snap
  // returns map-matched geometry with its own point count, and there is no
  // honest way to map those back onto the original segments. In that case one
  // segment is the truthful answer — the leg boundaries genuinely no longer
  // exist in the returned geometry.
  final sourceSegments = source?.trks
      .expand((t) => t.trksegs)
      .map((s) => s.trkpts.length)
      .where((n) => n > 0)
      .toList();
  final sourcePointCount = sourceSegments?.fold<int>(0, (a, b) => a + b) ?? 0;
  final canPreserveSegments =
      sourceSegments != null &&
      sourceSegments.length > 1 &&
      sourcePointCount == shape.length;

  final segments = <Trkseg>[];
  if (canPreserveSegments) {
    var cursor = 0;
    for (final length in sourceSegments) {
      segments.add(
        Trkseg(
          trkpts: [for (var i = cursor; i < cursor + length; i++) pointAt(i)],
        ),
      );
      cursor += length;
    }
  } else {
    segments.add(
      Trkseg(trkpts: [for (var i = 0; i < shape.length; i++) pointAt(i)]),
    );
  }

  // Carry every source track's metadata, not just the first. The
  // geometry is necessarily consolidated into one track (the shape arrives
  // flattened), but silently discarding the name/desc/type of tracks 2..n was
  // avoidable data loss on a multi-track file.
  final sourceTrks = source?.trks ?? const <Trk>[];
  final sourceTrk = sourceTrks.isNotEmpty ? sourceTrks.first : null;
  gpx.trks = [
    Trk(
      name: sourceTrk?.name,
      cmt: sourceTrk?.cmt,
      desc: _joinTrackDescriptions(sourceTrks),
      src: sourceTrk?.src,
      links: sourceTrk?.links == null ? null : List.of(sourceTrk!.links),
      number: sourceTrk?.number,
      type: sourceTrk?.type,
      extensions: sourceTrk?.extensions,
      trksegs: segments,
    ),
  ];
  return gpx;
}

/// Merges the `desc` of every source track so a multi-track file does not
/// silently lose tracks 2..n's description when the geometry is consolidated.
String? _joinTrackDescriptions(List<Trk> trks) {
  final descriptions = trks
      .map((t) => t.desc)
      .whereType<String>()
      .where((d) => d.trim().isNotEmpty)
      .toList();
  if (descriptions.isEmpty) return null;
  return descriptions.join('\n\n');
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
/// [shape] is the OUTBOUND REQUEST HINT — every caller passes
/// [buildNavShape]'s ≤500-point decimation, because that is what Valhalla's
/// shape cap allows. [fallbackShape] is the caller's real, full-resolution
/// geometry and is what gets returned when no snap happens. Passing it is
/// mandatory wherever the result is persisted: returning [shape] on the
/// fallback path silently replaced a 5000-point recorded track with a
/// 500-point decimation because a network call failed, which is exactly the
/// invariant `navigation_screen.dart`'s "the cap applies only to this
/// outbound hint" comment states. It defaults to [shape] only so a
/// caller whose request hint IS its full geometry needs no extra argument.
///
/// On any error/timeout, or when [snapResultAcceptable] rejects the result
/// as a partial map-match truncation, returns the fallback unchanged —
/// mirrors [buildFinalPlannedGpx]'s silent-fallback precedent (no toast, no
/// rethrow). Use [snapShapeToRoadsResult] instead when the caller must be
/// able to tell a real snap from a silent fallback.
Future<List<Map<String, double>>> snapShapeToRoads(
  WidgetRef ref,
  List<Map<String, double>> shape,
  String costing, {
  List<Map<String, double>>? fallbackShape,
}) async {
  final result = await snapShapeToRoadsResult(
    ref,
    shape,
    costing,
    fallbackShape: fallbackShape,
  );
  return result.shape;
}

/// [snapShapeToRoads]'s underlying implementation, additionally reporting
/// whether a real snap actually happened.
///
/// `snapped: false` means the returned shape is the untouched fallback — the
/// request threw, timed out, or [snapResultAcceptable] rejected the result.
/// Callers that invalidate derived per-point data (elevation indices, for
/// instance) MUST gate that invalidation on this flag: doing it
/// unconditionally destroys good data whenever the network merely hiccuped
Future<({List<Map<String, double>> shape, bool snapped})>
snapShapeToRoadsResult(
  WidgetRef ref,
  List<Map<String, double>> shape,
  String costing, {
  List<Map<String, double>>? fallbackShape,
}) async {
  final fallback = fallbackShape ?? shape;
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
      return (shape: snapped, snapped: true);
    }
    return (shape: fallback, snapped: false);
  } catch (_) {
    // Silent fallback: proceed with the caller's real geometry.
    return (shape: fallback, snapped: false);
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
/// entirely on-device — no convert-endpoint request. [category]/
/// [subcategory] pre-fill the trail's operator classification;
/// [durationSeconds] is the planner's Valhalla-estimated fallback (see
/// below); [movingDuration] is the recording hand-off.
///
/// [movingDuration] is the ONLY value a recording takes from its own
/// session — distance, elevation gain/loss and `duration` all come from the
/// ported computation (`trailFromGpx`, reached via the local trail builder
/// below) over the recorded GPX, so every persisted stat besides moving time
/// stays reproducible by a later recompute over the same GPX.
Future<Trail> buildDraftTrail(
  WidgetRef ref,
  Gpx finalGpx, {
  String? category,
  String? subcategory,
  double? durationSeconds,
  Duration? movingDuration,
}) async {
  // Still produced: `expand.gpxData` is what `util/trail/form_data.dart` uploads
  // as the trail's track file on save. Serialized off the UI thread (compute,
  // never a capturing closure — the enclosing scope holds a WidgetRef) — this
  // runs at the recording Stop tap / planner Finish, over the full track.
  final xml = await compute(serializeGpxToXml, finalGpx);

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
    date: DateTime.now(),
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
/// The planner never road-snaps. Its geometry is already Valhalla-routed leg
/// by leg as the user builds the route, and the save-options sheet that used
/// to offer a "follow roads" pass was removed — it re-snapped an
/// already-snapped shape through a double decimation, discarding good
/// elevations to refetch them. A leg that is genuinely NOT road-following
/// (auto-routing switched off, or a leg left `SegmentState.blocked` by a
/// failed route call) stays as the user left it; re-enabling auto-routing or
/// retrying that segment in the planner is the way to route it, not a
/// post-hoc pass at save time.
///
/// This is why [anchorsFromTrack]/[segmentPolylinesFromTrack] can locate
/// anchors by EXACT coordinate match without a boundary re-pin step here:
/// nothing between the planner and the emitted GPX moves a leg boundary.
/// `elevationProfile` comes from [buildNavShape], which always preserves its
/// input's first and last point, so every boundary already lands exactly on
/// an anchor coordinate. Any future transform inserted here must preserve
/// that invariant or the route becomes unreconstructable on re-edit —
/// silently collapsing every intermediate anchor.
///
/// The height step is gated on connectivity (`onlineStatusProvider`), which
/// is what makes this function genuinely safe to call with no network access
/// at all (the offline path): offline it issues ZERO requests, rather than
/// issuing one per un-elevated leg and merely tolerating the failures.
Future<Gpx> buildFinalPlannedGpx(WidgetRef ref) async {
  final legs = ref.read(routeAnchorsProvider).orderedSegments;
  if (legs.isEmpty) return Gpx();

  // Prefer the height-resolved point set; fall back to the raw polyline for
  // a leg whose fetch is still pending or failed.
  final legPoints = [for (final s in legs) s.elevationProfile ?? s.polyline];
  final legElevations = [for (final s in legs) s.elevations];

  // The connectivity gate is what makes this function's "safe to call
  // with no network access at all" claim TRUE. The `pending` list below is
  // built from every leg whose elevations are unresolved, so a session with
  // any un-elevated leg would otherwise issue a `/valhalla/height` request on
  // exactly the offline path. The behaviour was safe (fetchHeightsForShape
  // swallows the failure) but the stated invariant was wrong and nothing
  // tested the real case.
  final online = ref.read(onlineStatusProvider);
  final pending = !online
      ? const <int>[]
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
      // the leg carrying whatever it already had, which for an entry in
      // `pending` is `null`, so it emits `ele: null` rather than blocking.
      if (heights.isEmpty) continue;
      legElevations[pending[k]] = [for (final h in heights) h.toDouble()];
    }
  }

  final gpx = Gpx();
  gpx.trks = [
    Trk(
      trksegs: [
        // A 0-point leg is skipped rather than emitted as an empty
        // `Trkseg`. `anchorsFromTrack` filters empty segments out, so an
        // emitted empty one silently deletes that anchor on re-edit — the
        // exact failure the 1-point special case below already guards
        // against. An empty leg is reachable through the defensive paths
        // (`segmentPolylinesFromTrack` can yield a 1-point polyline when
        // `prevIndex == idx`, and its not-found fallback feeds straight
        // lines into the tail), so this is not purely hypothetical. An empty
        // leg is deliberately unrepresentable in the round-trip format:
        // there is no anchor pair it could describe.
        for (var i = 0; i < legs.length; i++)
          if (legPoints[i].isNotEmpty)
            Trkseg(
              trkpts: [
                // Drop the closing point of every leg but the last: it is
                // the next leg's opening point. A degenerate 1-point leg
                // keeps its single point rather than emitting an empty
                // trkseg, which anchorsFromTrack would skip and so lose that
                // anchor.
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
/// Takes no save-options toggles. The planner used to route Finish through
/// the shared online gate (`resolve_track_save_options.dart`), which offered
/// "recalculate heights" and "follow roads"; both were removed because
/// neither could improve a planned route. Heights are already fetched per
/// segment from the same deterministic DEM endpoint, so a refetch returned
/// identical values, and the geometry is already Valhalla-routed — see
/// [buildFinalPlannedGpx]. No `movingDuration` is ever passed here: a planned
/// route was never traversed.
Future<void> finishPlanning({
  required WidgetRef ref,
  required BuildContext navContext,
}) async {
  final anchorsState = ref.read(routeAnchorsProvider);
  final bucket = bucketForState(
    anchorsState.travelProfile,
    anchorsState.costingOptions,
  );
  final finalGpx = await buildFinalPlannedGpx(ref);

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
      // Resume at the pair the main loop has NOT yet emitted. On reaching
      // anchor `i` the loop has emitted pairs (0,1)…(i-2,i-1) — `i-1`
      // entries — so the (i-1, i) pair is still outstanding and must be the
      // first straight line, not the second. Starting at `k = i` dropped it,
      // leaving `anchors.length - 2` polylines for `anchors.length - 1`
      // segments; since `seedFromTrack` indexes this list POSITIONALLY, every
      // segment from `i-1` on then received the next pair's polyline and the
      // last one silently degraded to a straight line.
      for (var k = i > 0 ? i - 1 : 0; k < anchors.length - 1; k++) {
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
/// The ONE exception is `movingDuration`, which is cleared. Moving
/// time describes a traversal of a specific track; once the geometry is
/// replaced it describes nothing, yet the display rule
/// (`moving_duration > 0 ? moving_duration : duration`) would keep PREFERRING
/// it over the correctly-recomputed `duration` everywhere the trail is
/// rendered. It is cleared as `0` rather than `null` because Freezed's
/// `copyWith` cannot assign null and `form_data_util`'s write guard skips a
/// null field — so only `0` actually reaches PocketBase and overwrites the
/// stale value. `0` is also exactly what the display rule reads as "no
/// moving time".
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
    // The geometry just changed, so any recorded moving time no longer
    // describes this track. Left alone it would keep winning the display rule
    // over the freshly-recomputed `duration`.
    movingDuration: 0,
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
