import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:maplibre/maplibre.dart' as ml;
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

  void _clearDrag() {
    setState(() {
      _draggingAnchorId = null;
      _dragOffset = null;
    });
  }

  @override
  Widget build(BuildContext context) {
    final anchors = ref.watch(routeAnchorsProvider).anchors;
    final controller = ml.MapController.maybeOf(context);
    // Subscribe to camera changes so a drag-in-progress marker recomputes as
    // the user pans/zooms.
    ml.MapCamera.maybeOf(context);

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
          child: GestureDetector(
            onTap: () => widget.onAnchorTap?.call(anchor.id),
            onPanStart: (details) {
              final c = ml.MapController.maybeOf(context);
              if (c == null) return;
              setState(() {
                _draggingAnchorId = anchor.id;
                _dragOffset = c.toScreenLocation(anchor.point);
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
