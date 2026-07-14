// Native-looking selection chrome for the editor: drag handles and the
// drag magnifier.
//
// "Native" here means reusing Flutter's platform-correct building blocks —
// materialTextSelectionControls / cupertinoTextSelectionControls for the
// handle visuals, and the Material/Cupertino magnifier widgets — while the
// positioning and gesture logic are driven by the editor through the
// PositionRegistry. No built-in Flutter widget understands ProseMirror
// positions, so the wiring is ours; the pixels are the platform's.
//
// The handles widget is a passive renderer: it draws handles at the
// endpoint geometry it is given and reports raw drag events back through
// callbacks. All selection logic — which endpoint is fixed, preview state,
// committing to the engine — lives in the editor widget.
//
// IMPORTANT — why handles take precomputed geometry instead of querying the
// PositionRegistry themselves: the registry resolves positions through
// RenderParagraph text-layout queries (getOffsetForCaret), which assert that
// the paragraph has been laid out. The document renderer creates fresh
// GlobalKeys (and therefore fresh RenderParagraphs) on every rebuild, so
// during the build phase those render objects are never laid out yet —
// querying them from a widget's build() throws the !debugNeedsLayout
// assertion. The editor therefore computes a [SelectionChromeGeometry] in a
// post-frame callback (after layout, when the registry is safe to read) and
// passes the cached result here. This widget must never call the registry
// during build.
//
// Layout convention: the editor places these widgets inside its overlay
// Stack; all geometry offsets are in that Stack's local space. Handle
// positions follow the SelectionOverlay convention: the handle's anchor
// point (from getHandleAnchor) is aligned to the selection endpoint, which
// is the BOTTOM of the caret at that position.

import 'package:flutter/cupertino.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';

/// Resolve the platform-appropriate [TextSelectionControls] for native
/// handle visuals: Cupertino on iOS/macOS, Material elsewhere.
TextSelectionControls adaptiveTextSelectionControls(BuildContext context) {
  switch (Theme.of(context).platform) {
    case TargetPlatform.iOS:
    case TargetPlatform.macOS:
      return cupertinoTextSelectionControls;
    default:
      return materialTextSelectionControls;
  }
}

/// The pixel geometry of a selection's two endpoints, in the editor's
/// overlay Stack's local coordinate space.
///
/// Computed by the editor in a post-frame callback — after the frame's
/// layout pass, when the position registry's RenderParagraph queries are
/// valid — and cached across builds. Handles and the context toolbar render
/// from this cached value so the build phase never touches text layout.
///
/// [startTop] / [endTop] are the caret TOP offsets at the selection's from
/// and to positions; the caret heights give the corresponding bottoms, which
/// are the endpoints handles anchor to.
///
/// Value equality is implemented so the editor can skip a setState when a
/// recomputation produces identical geometry, letting the post-frame update
/// loop settle instead of rebuilding forever.
class SelectionChromeGeometry {
  /// Caret top offset at the selection's from position (overlay-local).
  final Offset startTop;

  /// Caret line height at the selection's from position.
  final double startCaretHeight;

  /// Caret top offset at the selection's to position (overlay-local).
  final Offset endTop;

  /// Caret line height at the selection's to position.
  final double endCaretHeight;

  const SelectionChromeGeometry({
    required this.startTop,
    required this.startCaretHeight,
    required this.endTop,
    required this.endCaretHeight,
  });

  /// The selection's start endpoint: the bottom of the caret at from.
  /// This is the point the start handle's anchor aligns to.
  Offset get startEndpoint => startTop + Offset(0, startCaretHeight);

  /// The selection's end endpoint: the bottom of the caret at to.
  /// This is the point the end handle's anchor aligns to.
  Offset get endEndpoint => endTop + Offset(0, endCaretHeight);

  @override
  bool operator ==(Object other) {
    return other is SelectionChromeGeometry &&
        other.startTop == startTop &&
        other.startCaretHeight == startCaretHeight &&
        other.endTop == endTop &&
        other.endCaretHeight == endCaretHeight;
  }

  @override
  int get hashCode =>
      Object.hash(startTop, startCaretHeight, endTop, endCaretHeight);

  @override
  String toString() =>
      'SelectionChromeGeometry(startTop: $startTop, h: $startCaretHeight, '
      'endTop: $endTop, h: $endCaretHeight)';
}

/// Callback for handle drag events. [isStartHandle] identifies which handle
/// is being dragged; [globalPosition] is the pointer's global position.
typedef HandleDragCallback =
    void Function(bool isStartHandle, Offset globalPosition);

/// Renders the two draggable selection handles for a non-empty selection
/// from precomputed endpoint geometry, using the platform's native handle
/// visuals.
///
/// Must be placed inside the editor's overlay Stack (it returns a
/// Positioned.fill containing its own inner Stack). All positions come from
/// the supplied [SelectionChromeGeometry]; this widget performs no text
/// layout queries of its own (see the file header for why).
///
/// Handle direction currently assumes LTR text: the start handle gets the
/// left type and the end handle the right type. RTL support is a follow-up.
class EditorSelectionHandles extends StatelessWidget {
  /// The endpoint geometry to render handles at, in the overlay Stack's
  /// local coordinate space.
  final SelectionChromeGeometry geometry;

  /// Drag lifecycle callbacks, reported with which handle is involved.
  final HandleDragCallback onDragStart;
  final HandleDragCallback onDragUpdate;
  final void Function(bool isStartHandle) onDragEnd;

  const EditorSelectionHandles({
    super.key,
    required this.geometry,
    required this.onDragStart,
    required this.onDragUpdate,
    required this.onDragEnd,
  });

  @override
  Widget build(BuildContext context) {
    final controls = adaptiveTextSelectionControls(context);

    return Positioned.fill(
      child: Stack(
        children: [
          _buildHandle(
            context: context,
            controls: controls,
            isStartHandle: true,
            endpoint: geometry.startEndpoint,
            caretHeight: geometry.startCaretHeight,
            type: TextSelectionHandleType.left,
          ),
          _buildHandle(
            context: context,
            controls: controls,
            isStartHandle: false,
            endpoint: geometry.endEndpoint,
            caretHeight: geometry.endCaretHeight,
            type: TextSelectionHandleType.right,
          ),
        ],
      ),
    );
  }

  /// Build one positioned handle at a precomputed endpoint.
  Widget _buildHandle({
    required BuildContext context,
    required TextSelectionControls controls,
    required bool isStartHandle,
    required Offset endpoint,
    required double caretHeight,
    required TextSelectionHandleType type,
  }) {
    /// getHandleAnchor returns the point within the handle widget that
    /// must coincide with the endpoint; subtracting it positions the
    /// handle's top-left corner.
    final anchor = controls.getHandleAnchor(type, caretHeight);

    return Positioned(
      left: endpoint.dx - anchor.dx,
      top: endpoint.dy - anchor.dy,
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        dragStartBehavior: DragStartBehavior.down,
        onPanStart: (details) =>
            onDragStart(isStartHandle, details.globalPosition),
        onPanUpdate: (details) =>
            onDragUpdate(isStartHandle, details.globalPosition),
        onPanEnd: (_) => onDragEnd(isStartHandle),
        onPanCancel: () => onDragEnd(isStartHandle),
        child: controls.buildHandle(context, type, caretHeight),
      ),
    );
  }
}

/// The platform-adaptive magnifier shown above the finger during a
/// long-press selection drag or a handle drag.
///
/// Built on the framework's Magnifier / CupertinoMagnifier widgets, which
/// magnify whatever is rendered beneath them. [focalPoint] is in the
/// overlay Stack's local coordinates (the editor converts the pointer's
/// global position before passing it in); the magnifier is positioned a
/// fixed gap above the finger with its focal point shifted down so the
/// magnified content is the text under the finger.
///
/// The platform magnifier widgets apply some internal focal-point shifting
/// of their own, so the vertical calibration here is an approximation that
/// may need device tuning; it is deliberately kept in one place (the two
/// constants below) for that reason.
class EditorDragMagnifier extends StatelessWidget {
  /// The point being magnified, in the overlay Stack's local coordinates.
  final Offset focalPoint;

  const EditorDragMagnifier({super.key, required this.focalPoint});

  /// Default sizes of the platform magnifiers, used for positioning math.
  static const Size _materialSize = Size(77.37, 37.9);
  static const Size _cupertinoSize = Size(80.0, 47.5);

  /// Vertical gap between the magnifier's bottom edge and the finger.
  static const double _gapAboveFinger = 24.0;

  @override
  Widget build(BuildContext context) {
    final platform = Theme.of(context).platform;
    final isCupertino =
        platform == TargetPlatform.iOS || platform == TargetPlatform.macOS;
    final size = isCupertino ? _cupertinoSize : _materialSize;

    /// Shift the focal point from the magnifier's center down to the
    /// finger position beneath it.
    final focalShift = Offset(0, size.height / 2 + _gapAboveFinger);

    return Positioned(
      left: focalPoint.dx - size.width / 2,
      top: focalPoint.dy - size.height - _gapAboveFinger,
      child: IgnorePointer(
        child: isCupertino
            ? CupertinoMagnifier(additionalFocalPointOffset: focalShift)
            : Magnifier(additionalFocalPointOffset: focalShift),
      ),
    );
  }
}
