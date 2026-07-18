import 'package:gpx/gpx.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';

part 'planned_gpx_provider.g.dart';

/// Derives an ordered, pre-elevation [Gpx] skeleton (points only, no `ele`)
/// from the in-progress route held by [routeAnchorsProvider] (PLANUI-02).
///
/// Walks the anchor-id chain starting at `anchors.first`, following each
/// segment's `beforeAnchorId -> afterAnchorId` link (not `state.segments`
/// array order, Pitfall 3), appending each segment's polyline via
/// `skip(1)` so the shared boundary point between two consecutive segments
/// is not duplicated.
///
/// Recomputes whenever `routeAnchorsProvider`'s anchors/segments identity
/// changes. Never sets `ele` (D-10): the elevation tab owns the
/// elevation-merged copy.
@riverpod
Gpx plannedGpx(Ref ref) {
  final state = ref.watch(routeAnchorsProvider);

  final gpx = Gpx();
  if (state.anchors.isEmpty) return gpx;

  final segByBefore = {for (final s in state.segments) s.beforeAnchorId: s};

  final first = state.anchors.first;
  final trackSegments = <Trkseg>[
    Trkseg(trkpts: [Wpt(lat: first.lat, lon: first.lon)]),
  ];
  var currentId = first.id;

  while (segByBefore.containsKey(currentId)) {
    final seg = segByBefore[currentId]!;

    trackSegments.add(
      Trkseg(
        trkpts: [
          for (final p in seg.polyline.skip(1)) Wpt(lat: p.lat, lon: p.lon),
        ],
      ),
    );

    currentId = seg.afterAnchorId;
  }

  gpx.trks = [Trk(trksegs: trackSegments)];

  return gpx;
}

/// Sibling of [plannedGpx]: an elevation-bearing `Gpx` for the Elevation
/// tab's chart, built entirely from data already living on
/// `routeAnchorsProvider`'s segments — no network call of its own (that
/// fetch now lives on `route_anchor_provider.dart`'s `_resolveElevation`,
/// fired fire-and-forget per segment on creation/update).
///
/// Same anchor-chain walk as [plannedGpx] (`beforeAnchorId -> afterAnchorId`
/// links, not array order), but each segment contributes its own
/// [RouteSegment.elevationProfile] points (falling back to [RouteSegment.polyline]
/// when a segment's height fetch hasn't resolved yet) paired with
/// [RouteSegment.elevations] — `ele` stays `null` for any point beyond
/// what's been fetched so far, which the elevation chart already treats as
/// `0` (transient, until that segment's fetch resolves).
@riverpod
Gpx plannedElevationGpx(Ref ref) {
  final state = ref.watch(routeAnchorsProvider);

  final gpx = Gpx();
  if (state.anchors.isEmpty) return gpx;

  final segByBefore = {for (final s in state.segments) s.beforeAnchorId: s};

  final first = state.anchors.first;
  // The seed anchor's own elevation is never known (heights live on
  // segments, not anchors) — `ele: null` here degrades the same way the
  // chart already treats any not-yet-fetched point.
  final trackSegments = <Trkseg>[
    Trkseg(trkpts: [Wpt(lat: first.lat, lon: first.lon)]),
  ];
  var currentId = first.id;

  while (segByBefore.containsKey(currentId)) {
    final seg = segByBefore[currentId]!;
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

    currentId = seg.afterAnchorId;
  }

  gpx.trks = [Trk(trksegs: trackSegments)];

  return gpx;
}
