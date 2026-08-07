import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/map/map_marker_gestures.dart';
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';

/// Interactive route-anchor markers rendered as Flutter widgets over the
/// native map via a single [ml.WidgetLayer]: numbered, draggable markers
/// whose displayed number is always derived from the anchor's current
/// position in the anchor list — never a stored field.
///
/// Reads its data directly from `ref.watch(routeAnchorsProvider)` since it
/// renders the in-progress, unsaved route plan rather than a persisted `Trail`.
///
/// While dragging, the marker shows at a temporary screen position; connected
/// segments only re-resolve once the gesture ends (`dragAnchor` is called
/// from `onPanEnd`, never `onPanUpdate`).
///
/// Place this in [ml.MapLibreMap.children]. It reads [ml.MapController]/
/// [ml.MapCamera] from context so a drag-in-progress marker stays aligned
/// with the map as the user pans/zooms mid-gesture.
class RouteAnchorLayer extends ConsumerStatefulWidget {
  final String? selectedAnchorId;
  final void Function(String anchorId)? onAnchorTap;

  const RouteAnchorLayer({super.key, this.selectedAnchorId, this.onAnchorTap});

  @override
  ConsumerState<RouteAnchorLayer> createState() => _RouteAnchorLayerState();
}

class _RouteAnchorLayerState extends ConsumerState<RouteAnchorLayer> {
  String? _draggingAnchorId;
  Offset? _dragOffset;

  /// Memoized markers, and the inputs they were built from.
  ///
  /// This widget is rebuilt on every camera frame and cannot avoid it:
  /// `MapController.maybeOf` and `MapCamera.maybeOf` both resolve to
  /// `InheritedModel.inheritFrom<MapLibreInheritedModel>` with no aspect,
  /// and that model's `updateShouldNotify`/`updateShouldNotifyDependent`
  /// both return `true` unconditionally — so reading the controller at all
  /// subscribes to every camera update.
  ///
  /// Rebuilding N marker subtrees per frame (gesture detectors,
  /// AnimatedScales, blurred-shadow containers) is the expensive part, and
  /// that *is* avoidable: [ml.WidgetLayer] repositions markers itself via
  /// `toScreenLocations`, so as long as it receives the same `Widget`
  /// instances, element diffing skips their subtrees entirely.
  List<ml.Marker>? _cachedMarkers;
  List<RouteAnchor>? _cachedAnchors;
  String? _cachedSelectedId;
  Brightness? _cachedBrightness;

  void _invalidateMarkers() => _cachedMarkers = null;

  void _clearDrag() {
    setState(() {
      _draggingAnchorId = null;
      _dragOffset = null;
      _invalidateMarkers();
    });
  }

  @override
  Widget build(BuildContext context) {
    final anchors = ref.watch(routeAnchorsProvider).anchors;
    final controller = ml.MapController.maybeOf(context);
    final brightness = Theme.of(context).brightness;

    // A drag re-projects a fixed screen offset through the live camera, so
    // its marker must be rebuilt every frame. Only the static case caches.
    final cached = _cachedMarkers;
    if (_draggingAnchorId == null &&
        cached != null &&
        identical(anchors, _cachedAnchors) &&
        _cachedSelectedId == widget.selectedAnchorId &&
        _cachedBrightness == brightness) {
      return ml.WidgetLayer(allowInteraction: true, markers: cached);
    }

    final markers = <ml.Marker>[];

    for (var i = 0; i < anchors.length; i++) {
      final anchor = anchors[i];
      final number = i + 1; // derived from list order, never stored.
      final isSelected = widget.selectedAnchorId == anchor.id;
      final isDragging = _draggingAnchorId == anchor.id;
      final point = (isDragging && controller != null && _dragOffset != null)
          ? controller.toLngLat(_dragOffset!)
          : anchor.point;

      markers.add(
        ml.Marker(
          point: point,
          size: const Size(32, 32),
          // Anchors keep their single-finger drag; MapMarkerGestures only
          // hands two-finger gestures back to the map.
          child: MapMarkerGestures(
            onTap: () => widget.onAnchorTap?.call(anchor.id),
            onPanStart: (details) {
              final c = ml.MapController.maybeOf(context);
              if (c == null) return;
              setState(() {
                _draggingAnchorId = anchor.id;
                _dragOffset = c.toScreenLocation(anchor.point);
                _invalidateMarkers();
              });
            },
            onPanUpdate: (details) {
              if (_draggingAnchorId != anchor.id || _dragOffset == null) {
                return;
              }
              setState(() => _dragOffset = _dragOffset! + details.delta);
            },
            onPanEnd: (details) {
              if (_draggingAnchorId != anchor.id) return;
              final c = ml.MapController.maybeOf(context);
              final offset = _dragOffset;
              _clearDrag();
              if (c != null && offset != null) {
                // Called only at gesture end, never onPanUpdate — re-resolves
                // the anchor's adjacent segments to the current routing mode.
                ref
                    .read(routeAnchorsProvider.notifier)
                    .dragAnchor(anchor.id, c.toLngLat(offset));
              }
            },
            onPanCancel: () {
              if (_draggingAnchorId == anchor.id) _clearDrag();
            },
            child: AnimatedScale(
              scale: (isSelected || isDragging) ? 1.0 : 0.875,
              duration: const Duration(milliseconds: 200),
              curve: Curves.easeOutBack,
              child: _buildNumberedMarker(
                context,
                number,
                selected: isSelected || isDragging,
              ),
            ),
          ),
        ),
      );
    }

    // Only the static case is reusable — a drag's marker position is derived
    // from the live camera, so caching it would freeze it mid-gesture.
    if (_draggingAnchorId == null) {
      _cachedMarkers = markers;
      _cachedAnchors = anchors;
      _cachedSelectedId = widget.selectedAnchorId;
      _cachedBrightness = brightness;
    } else {
      _invalidateMarkers();
    }

    return ml.WidgetLayer(allowInteraction: true, markers: markers);
  }
}

/// Numbered circular marker showing a 1-based display number. Selected/
/// dragging state inverts to a white fill with an accent-colored border.
Widget _buildNumberedMarker(
  BuildContext context,
  int number, {
  required bool selected,
}) {
  final accent = Theme.of(context).brightness == Brightness.dark
      ? const Color(0xff3E435B)
      : const Color(0xff242734);

  return Container(
    decoration: BoxDecoration(
      color: selected ? Colors.white : accent,
      shape: BoxShape.circle,
      boxShadow: [
        BoxShadow(
          color: Colors.black.withValues(alpha: .2),
          blurRadius: 4,
          offset: const Offset(0, 2),
        ),
      ],
      border: Border.all(color: selected ? accent : Colors.white, width: 2),
    ),
    child: Center(
      child: Text(
        '$number',
        style: TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w600,
          color: selected ? accent : Colors.white,
        ),
      ),
    ),
  );
}
