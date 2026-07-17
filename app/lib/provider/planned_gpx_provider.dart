import 'package:gpx/gpx.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/util/gpx_util.dart';

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

  if (state.anchors.isEmpty) return buildGpxFromPoints(const []);

  final segByBefore = {
    for (final s in state.segments) s.beforeAnchorId: s,
  };

  final points = [state.anchors.first.point];
  var currentId = state.anchors.first.id;

  while (segByBefore.containsKey(currentId)) {
    final seg = segByBefore[currentId]!;
    points.addAll(seg.polyline.skip(1));
    currentId = seg.afterAnchorId;
  }

  return buildGpxFromPoints(points);
}
