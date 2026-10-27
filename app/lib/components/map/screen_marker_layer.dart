import 'package:flutter/foundation.dart' show listEquals;
import 'package:flutter/material.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/util/map/batched_projection.dart';

/// Camera pitch (degrees) above which the screen position of a point is no
/// longer an affine function of its mercator position, so batching is off.
const _maxBatchPitch = 0.05;

/// How far outside the viewport a marker may sit and still be painted (and
/// hit-testable). Covers the marker's own growth animation at the edge.
const _cullMargin = 16.0;

/// Interactive widget markers over the native map, positioned at the camera
/// with O(1) work per camera frame in the number of markers.
///
/// A drop-in for `ml.WidgetLayer(allowInteraction: true)` in
/// [ml.MapLibreMap.children], minus its per-frame costs. For N markers
/// `WidgetLayer` makes N synchronous native projections, allocates and
/// re-lays-out N `Positioned`/`Transform` widgets, and repaints every marker
/// inside the page-level repaint boundary — which also repaints (and
/// re-rasterises) everything else on the page, such as a docked sheet, on
/// every pan frame. Here:
///
///  * positions come from [projectBatched] — five native projections
///    regardless of N, with a per-point fallback when the camera is tilted;
///  * a [Flow] moves the markers by paint-time translation: no `Positioned`
///    churn and no relayout while the camera moves;
///  * every marker is its own [RepaintBoundary], so its picture (shadow,
///    border, label) is recorded once and only re-parented per frame;
///  * `RenderFlow` is itself a repaint boundary, so camera frames never
///    repaint the rest of the page (a docked sheet, the app bar);
///  * markers outside the viewport are not painted and cannot be hit.
///
/// Like `WidgetLayer`, reading the controller/camera here subscribes to every
/// native camera event (the package's inherited model always notifies); the
/// rebuild is O(1) because [markers] is expected to be a stable list — pass
/// the same list instance while the markers have not changed.
///
/// [ml.Marker.rotate] and [ml.Marker.flat] are not supported (the layer is
/// upright and screen-aligned). A marker child's `key`, if any, keys its
/// slot, so a `ValueKey` per entity keeps per-marker state across inserts and
/// removals.
class ScreenMarkerLayer extends StatelessWidget {
  const ScreenMarkerLayer({super.key, required this.markers});

  final List<ml.Marker> markers;

  @override
  Widget build(BuildContext context) {
    final controller = ml.MapController.maybeOf(context);
    final camera = ml.MapCamera.maybeOf(context);
    if (controller == null || camera == null) return const SizedBox.shrink();

    return ScreenMarkerFlow(
      markers: markers,
      project: controller.toScreenLocation,
      batchable: camera.pitch.abs() < _maxBatchPitch,
    );
  }
}

/// The map-independent core of [ScreenMarkerLayer]: lays [markers] out at the
/// screen positions [project] returns. Split out so it can be exercised
/// without a map.
class ScreenMarkerFlow extends StatefulWidget {
  const ScreenMarkerFlow({
    super.key,
    required this.markers,
    required this.project,
    this.batchable = true,
  });

  final List<ml.Marker> markers;

  /// Projects one geographic point to a logical-pixel offset within this
  /// widget's box (the platform's `toScreenLocation`).
  final Offset Function(ml.Geographic point) project;

  /// Whether screen position is an affine function of geographic position
  /// right now (camera pitch 0). When false every point is projected natively.
  final bool batchable;

  @override
  State<ScreenMarkerFlow> createState() => _ScreenMarkerFlowState();
}

class _ScreenMarkerFlowState extends State<ScreenMarkerFlow> {
  List<ml.Marker>? _builtFor;
  List<ml.Geographic> _points = const [];
  List<Widget> _children = const [];

  /// Per-marker wrappers and the point list are derived from [markers] alone,
  /// so they are rebuilt only when the list instance changes — not on the
  /// per-camera-frame rebuilds.
  void _rebuildDerived(List<ml.Marker> markers) {
    assert(
      markers.every((m) => !m.rotate && !m.flat),
      'ScreenMarkerLayer does not support rotate/flat markers.',
    );
    _builtFor = markers;
    _points = [for (final m in markers) m.point];
    _children = [
      for (final m in markers)
        // Same keying as `Flow` itself would apply, kept distinct from the
        // child's own key so that key stays unique in the tree.
        RepaintBoundary(
          key: m.child.key == null ? null : ValueKey<Key>(m.child.key!),
          child: m.child,
        ),
    ];
  }

  @override
  Widget build(BuildContext context) {
    final markers = widget.markers;
    if (!identical(markers, _builtFor)) _rebuildDerived(markers);

    final offsets = widget.batchable
        ? projectBatched(_points, widget.project)
        : List<Offset>.generate(
            _points.length,
            (i) => widget.project(_points[i]),
            growable: false,
          );

    // `Flow.unwrapped`: the children are already boundaries (cached in
    // [_children]), so `Flow` need not re-wrap N of them on every frame.
    return Flow.unwrapped(
      // Culling is done in the delegate; a clip layer would only add cost.
      clipBehavior: Clip.none,
      delegate: _MarkerFlowDelegate(markers: markers, offsets: offsets),
      children: _children,
    );
  }
}

class _MarkerFlowDelegate extends FlowDelegate {
  const _MarkerFlowDelegate({required this.markers, required this.offsets});

  final List<ml.Marker> markers;

  /// Projected marker anchor points (the point each marker is aligned on).
  final List<Offset> offsets;

  @override
  BoxConstraints getConstraintsForChild(int i, BoxConstraints constraints) =>
      BoxConstraints.tight(markers[i].size);

  @override
  void paintChildren(FlowPaintingContext context) {
    final viewport = (Offset.zero & context.size).inflate(_cullMargin);
    for (var i = 0; i < markers.length; i++) {
      final marker = markers[i];
      final size = marker.size;
      final left = offsets[i].dx - size.width / 2 * (marker.alignment.x + 1);
      final top = offsets[i].dy - size.height / 2 * (marker.alignment.y + 1);
      if (!viewport.overlaps(
        Rect.fromLTWH(left, top, size.width, size.height),
      )) {
        continue;
      }
      context.paintChild(i, transform: Matrix4.translationValues(left, top, 0));
    }
  }

  @override
  bool shouldRelayout(_MarkerFlowDelegate oldDelegate) {
    if (identical(oldDelegate.markers, markers)) return false;
    if (oldDelegate.markers.length != markers.length) return true;
    for (var i = 0; i < markers.length; i++) {
      if (oldDelegate.markers[i].size != markers[i].size) return true;
    }
    return false;
  }

  @override
  bool shouldRepaint(_MarkerFlowDelegate oldDelegate) =>
      !identical(oldDelegate.markers, markers) ||
      !listEquals(oldDelegate.offsets, offsets);
}
