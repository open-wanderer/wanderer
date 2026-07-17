import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/planned_gpx_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/util/gpx_util.dart';
import 'package:wanderer/util/trail_import_util.dart';

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

/// Orchestrates the Route Planner's "Finish planning" handoff (HANDOFF-01).
///
/// Reads the final [plannedGpxProvider] snapshot for [travelProfile],
/// attempts a one-time `POST /valhalla/height` elevation merge (silent
/// best-effort — D-06: any failure degrades to the pre-elevation `Gpx`, no
/// error UI), resolves a category pre-fill via [categoryForTravelProfile]
/// (D-08, may be `null`), builds the draft [Trail] via [buildDraftTrail],
/// then reuses the existing GPX-import handoff mechanism verbatim
/// ([pendingImportedTrail] + `navContext.push('/trail/create/edit')`) — no
/// parallel state-passing mechanism.
///
/// Returns early (no-op) when fewer than 2 points are available (D-05
/// backstop; the Finish action is already disabled below 2 anchors at the
/// call site).
Future<void> finishPlanning({
  required WidgetRef ref,
  required BuildContext navContext,
  required String travelProfile,
}) async {
  final gpx = ref.read(plannedGpxProvider(travelProfile));
  final points = gpx.allPoints;
  if (points.length < 2) return;

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

  final categories = ref.read(categoryProvider).value ?? const [];
  final categoryId = categoryForTravelProfile(travelProfile, categories);

  final draftTrail = buildDraftTrail(finalGpx, category: categoryId);

  pendingImportedTrail = draftTrail;
  if (!navContext.mounted) return;
  navContext.push('/trail/create/edit', extra: draftTrail);
}
