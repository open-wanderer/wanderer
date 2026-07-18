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
/// index-for-index — a public copy of `ElevationTab._buildEleMergedGpx`
/// (`elevation_tab.dart`). Merge must happen against [shape] (the
/// `buildNavShape`-downsampled array), never against the original point
/// list, so index alignment holds once a route exceeds Valhalla's 500-point
/// shape cap (Pitfall 2).
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

/// Builds an unsaved, in-memory draft [Trail] from a finished planner route
/// (HANDOFF-01). Carries only the synthesized GPX track — `waypoints: []`
/// (D-07) — the same shape a plain GPX-file import produces with no named
/// points. Route anchors never become [Waypoint] records; they stay strictly
/// internal to the Route Planner's own state.
///
/// Sets both `expand.gpxData` (the raw XML string `form_data_util.dart`'s
/// `toFormData()` actually uploads on create) and `expand.gpx` (the parsed
/// object used for client-side map preview) — Pitfall 1: setting only one of
/// the two produces a draft that renders correctly but saves with no track,
/// or vice versa.
///
/// Also sets `lat`/`lon`/`minLat`/`maxLat`/`minLon`/`maxLon` from the track's
/// bounds. For a plain GPX-file import these arrive pre-computed from the
/// server's `/trail/convert` (gpx2trail) response, but a planner draft never
/// makes that round-trip — without this, `Trail.empty()`'s zeroed bounds and
/// null lat/lon leave `TrailMap` centered on null island instead of fitting
/// the route (`trail_map.dart`'s `fitBounds` only fires when bounds have
/// spread).
Trail buildDraftTrail(Gpx finalGpx, {String? category}) {
  final xml = GpxWriter().asString(finalGpx);
  final bounds = finalGpx.getBounds();
  return Trail.empty().copyWith(
    category: category,
    lat: bounds != null
        ? (bounds.latitudeNorth + bounds.latitudeSouth) / 2
        : null,
    lon: bounds != null
        ? (bounds.longitudeEast + bounds.longitudeWest) / 2
        : null,
    maxLat: bounds?.latitudeNorth ?? 0,
    minLat: bounds?.latitudeSouth ?? 0,
    maxLon: bounds?.longitudeEast ?? 0,
    minLon: bounds?.longitudeWest ?? 0,
    expand: TrailExpand(
      gpxData: xml,
      gpx: finalGpx,
      waypointsViaTrail: const [],
    ),
  );
}

/// Builds the final, ele-merged [Gpx] for the current planner session
/// (extracted from [finishPlanning] so the edit-mode handoff — quick-260718-e9j
/// — can pop this same value instead of forward-pushing a draft trail).
///
/// Reads the final [plannedGpxProvider] snapshot, attempts a one-time
/// `POST /valhalla/height` elevation merge (silent best-effort — D-06: any
/// failure degrades to the pre-elevation `Gpx`, no error UI). Returns the
/// bare (pre-elevation) `Gpx` unchanged when fewer than 2 points are
/// available (D-05 backstop; the Finish action is already disabled below 2
/// anchors at the call site).
Future<Gpx> buildFinalPlannedGpx(WidgetRef ref) async {
  final gpx = ref.read(plannedGpxProvider);
  final points = gpx.allPoints;
  if (points.length < 2) return gpx;

  final shape = buildNavShape(points);
  var finalGpx = gpx; // fallback: pre-elevation, if the fetch fails (D-06)
  try {
    final response = await ref
        .read(apiProvider)
        .post('/valhalla/height', data: {'shape': shape});
    final heights = (response.data['height'] as List).cast<num>();
    finalGpx = mergeHeightsIntoGpx(shape, heights);
  } catch (_) {
    // D-06: proceed silently with the pre-elevation Gpx, no error UI.
  }

  return finalGpx;
}

/// Orchestrates the Route Planner's "Finish planning" handoff (HANDOFF-01) —
/// the plain GPX-import forward-push path, unchanged by quick-260718-e9j.
///
/// Builds the final ele-merged [Gpx] via [buildFinalPlannedGpx], resolves a
/// category pre-fill via [categoryForTravelProfile] (D-08, may be `null` —
/// deliberately UNCHANGED by the Settings-tab picker, quick-260717-t7q
/// CONTEXT: "explicitly OUT of scope"), builds the draft [Trail] via
/// [buildDraftTrail], then reuses the existing GPX-import handoff mechanism
/// verbatim ([pendingImportedTrail] + `navContext.push('/trail/create/edit')`)
/// — no parallel state-passing mechanism.
Future<void> finishPlanning({
  required WidgetRef ref,
  required BuildContext navContext,
}) async {
  final travelProfile = ref.read(routeAnchorsProvider).travelProfile;
  final finalGpx = await buildFinalPlannedGpx(ref);

  final categories = ref.read(categoryProvider).value ?? const [];
  final categoryId = categoryForTravelProfile(travelProfile, categories);

  final draftTrail = buildDraftTrail(finalGpx, category: categoryId);

  pendingImportedTrail = draftTrail;
  if (!navContext.mounted) return;
  navContext.push('/trail/create/edit', extra: draftTrail);
}

/// Derives segment-boundary [ml.Geographic] anchors from an existing track,
/// mirroring web's `initRouteAnchors` (`+page.svelte:553-577`) exactly for
/// the Route Planner's edit-mode seed (quick-260718-e9j, PLANNER-02): one
/// anchor at the first point of every `trkseg`, plus the last point of the
/// FINAL segment. No interior sampling, no reverse-geocoding at seed time.
///
/// Returns an empty list for a trackless (no `trks`) [gpx].
List<ml.Geographic> anchorsFromTrack(Gpx gpx) {
  final segs = gpx.trks.isNotEmpty ? gpx.trks.first.trksegs : const <Trkseg>[];
  // Only segments with at least one fully-coordinated point count — a
  // trkpt missing lat/lon (spec-valid but malformed data) is dropped rather
  // than force-unwrapped, and a trailing empty/coordinateless segment no
  // longer swallows the true final-point addition for the segment before it.
  final nonEmpty = segs
      .map((s) => s.trkpts.where((p) => p.lat != null && p.lon != null).toList())
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

/// Merges a Route Planner edit-mode result [finalGpx] onto an [existing]
/// in-memory [Trail] (quick-260718-e9j, PLANNER-02), modeled on
/// [buildDraftTrail] but as a `copyWith` — every non-track field (title,
/// description, id, visibility, photos, waypoints, category) carries through
/// untouched.
///
/// Sets both `expand.gpxData` (the raw XML `form_data_util.dart`'s
/// `toFormData()` actually uploads) AND `expand.gpx` (the parsed object used
/// for client-side map preview) — Pitfall 1: setting only one of the two
/// produces a trail that renders correctly but saves with no track, or vice
/// versa.
///
/// lat/lon/bounds are recomputed from [finalGpx]'s bounds when available,
/// falling back to [existing]'s prior values when the merged route has no
/// track (defensive; the Finish action guarantees >=2 anchors so this should
/// not occur in practice).
Trail mergeRouteIntoTrail(Trail existing, Gpx finalGpx) {
  final xml = GpxWriter().asString(finalGpx);
  final bounds = finalGpx.getBounds();
  return existing.copyWith(
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
