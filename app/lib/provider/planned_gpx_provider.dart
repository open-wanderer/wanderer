import 'package:gpx/gpx.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';

part 'planned_gpx_provider.g.dart';

/// Derives an ordered, pre-elevation [Gpx] skeleton (points only, no `ele`)
/// from the in-progress route held by [routeAnchorsProvider].
///
/// Walks [RouteAnchorsState.orderedSegments], appending each segment's
/// polyline via `skip(1)` so the shared boundary point between two
/// consecutive segments isn't duplicated. The leading single-point `Trkseg`
/// carries the start anchor.
///
/// This layout is for in-app consumers that read the route as one continuous
/// point stream (`gpx.allPoints`) — its `trkseg` boundaries sit one point
/// *after* each anchor, so it is NOT round-trippable through
/// `anchorsFromTrack`. Route export uses [buildFinalPlannedGpx] instead,
/// which emits whole legs including both endpoints.
///
/// Never sets `ele`: the elevation tab owns the elevation-merged copy.
@riverpod
Gpx plannedGpx(Ref ref) {
  final state = ref.watch(routeAnchorsProvider);

  final gpx = Gpx();
  if (state.anchors.isEmpty) return gpx;

  final first = state.anchors.first;
  final trackSegments = <Trkseg>[
    Trkseg(
      trkpts: [Wpt(lat: first.lat, lon: first.lon)],
    ),
  ];

  for (final seg in state.orderedSegments) {
    trackSegments.add(
      Trkseg(
        trkpts: [
          for (final p in seg.polyline.skip(1)) Wpt(lat: p.lat, lon: p.lon),
        ],
      ),
    );
  }

  gpx.trks = [Trk(trksegs: trackSegments)];

  return gpx;
}

/// Sibling of [plannedGpx]: an elevation-bearing `Gpx` for the Elevation
/// tab's chart, built from data already on `routeAnchorsProvider`'s
/// segments (elevation fetches happen there, fire-and-forget per segment).
///
/// Same layout and [RouteAnchorsState.orderedSegments] walk as [plannedGpx]
/// (and the same non-round-trippable caveat), but each segment contributes its
/// [RouteSegment.elevationProfile] points (falling back to
/// [RouteSegment.polyline] while a segment's height fetch is pending) paired
/// with [RouteSegment.elevations]; `ele` stays `null` for points not yet
/// fetched, which the chart treats as `0` until that segment resolves.
@riverpod
Gpx plannedElevationGpx(Ref ref) {
  final state = ref.watch(routeAnchorsProvider);

  final gpx = Gpx();
  if (state.anchors.isEmpty) return gpx;

  final first = state.anchors.first;
  // Heights live on segments, not anchors, so the seed anchor's elevation
  // is always unknown; `ele: null` degrades the same as any unfetched point.
  final trackSegments = <Trkseg>[
    Trkseg(
      trkpts: [Wpt(lat: first.lat, lon: first.lon)],
    ),
  ];

  for (final seg in state.orderedSegments) {
    final points = seg.elevationProfile ?? seg.polyline;
    final elevations = seg.elevations;

    trackSegments.add(
      Trkseg(
        trkpts: [
          for (var i = 1; i < points.length; i++)
            Wpt(
              lat: points[i].lat,
              lon: points[i].lon,
              ele: elevations != null && i < elevations.length
                  ? elevations[i]
                  : null,
            ),
        ],
      ),
    );
  }

  gpx.trks = [Trk(trksegs: trackSegments)];

  return gpx;
}
