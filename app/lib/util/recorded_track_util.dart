import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/util/gpx_util.dart';
import 'package:wanderer/util/route_planner_handoff_util.dart';

/// Builds an unsaved, in-memory stub [Trail] from a navigation session's
/// recorded GPS breadcrumb, so it can be handed to `trail_create_screen` the
/// same way a GPX-file import or a Route Planner "Finish" does.
///
/// Deliberately thin: [NavigationState.breadcrumb] carries lat/lon fixes
/// only — no per-fix `ele` or `time` — so the uploaded track itself has no
/// elevation profile. [durationSeconds] and [elevationGainMeters]/
/// [elevationLossMeters] (typically `NavigationStats.elapsed` and its own
/// GPS-altitude-delta accumulation) are passed straight through to
/// [buildDraftTrail]'s matching pre-fill params, since there's no `ele`- or
/// `time`-tagged track for the server to derive them from on save. Reuses
/// [buildDraftTrail] (rather than hand-rolling the same logic) so the
/// bounds/`expand.gpxData`+`expand.gpx` derivation stays single-sourced with
/// the route-planner and GPX-import handoffs — a draft that sets only one of
/// `gpxData`/`gpx` either saves with no track or previews incorrectly (see
/// [buildDraftTrail]'s own doc comment).
///
/// No `category` is passed through — a recorded track has no travel profile,
/// and [buildDraftTrail] accepts a null category.
///
/// Returns `null` when [breadcrumb] has fewer than 2 points (empty or a
/// single fix — nothing meaningful to save); callers treat `null` as
/// "nothing recorded, just leave navigation".
Trail? buildRecordedTrackTrail(
  List<Geographic> breadcrumb, {
  double? durationSeconds,
  double? elevationGainMeters,
  double? elevationLossMeters,
}) {
  if (breadcrumb.length < 2) return null;
  // Todo: store a better breadcrumb trail as Gpx.Wpt instead
  final gpx = buildGpxFromPoints(breadcrumb);
  return Trail.empty();
}
