import 'package:flutter/gestures.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/widgets.dart';

/// Gesture wrapper for widgets rendered inside an
/// `ml.WidgetLayer(allowInteraction: true)` over the native map.
///
/// A plain [GestureDetector] there swallows every pointer that lands on the
/// marker: the marker sits in the normal hit-test path, so the map's platform
/// view underneath never sees the event and a pan or pinch started on a marker
/// does nothing. This wrapper keeps the marker tappable (and optionally
/// draggable) while staying hit-test *translucent*, so the pointer reaches both
/// the marker and the map surface below.
///
/// Why that resolves correctly: the map's platform view is created with an
/// empty `gestureRecognizers` set, which means it only claims pointers no
/// Flutter recognizer wins. A tap won here stays here; anything the recognizers
/// below reject — drags, flings, pinches — flushes through to the map.
///
/// Multi-finger gestures are always the map's: both recognizers refuse to start
/// while another pointer is down, and bail out if a second finger arrives
/// mid-gesture.
class MapMarkerGestures extends StatelessWidget {
  const MapMarkerGestures({
    super.key,
    required this.child,
    this.onTap,
    this.onPanStart,
    this.onPanUpdate,
    this.onPanEnd,
    this.onPanCancel,
  });

  final Widget child;

  final VoidCallback? onTap;

  /// Drag callbacks. Pass them only for markers that are actually draggable —
  /// a registered pan recognizer takes single-finger drags away from the map.
  final GestureDragStartCallback? onPanStart;
  final GestureDragUpdateCallback? onPanUpdate;
  final GestureDragEndCallback? onPanEnd;
  final GestureDragCancelCallback? onPanCancel;

  bool get _hasPan =>
      onPanStart != null ||
      onPanUpdate != null ||
      onPanEnd != null ||
      onPanCancel != null;

  @override
  Widget build(BuildContext context) {
    final gestures = <Type, GestureRecognizerFactory>{
      if (onTap != null)
        _SingleFingerTapRecognizer:
            GestureRecognizerFactoryWithHandlers<_SingleFingerTapRecognizer>(
              _SingleFingerTapRecognizer.new,
              (recognizer) => recognizer.onTap = onTap,
            ),
      if (_hasPan)
        _SingleFingerPanRecognizer:
            GestureRecognizerFactoryWithHandlers<_SingleFingerPanRecognizer>(
              _SingleFingerPanRecognizer.new,
              (recognizer) => recognizer
                ..onStart = onPanStart
                ..onUpdate = onPanUpdate
                ..onEnd = onPanEnd
                ..onCancel = onPanCancel,
            ),
    };

    return _PointerPassThrough(
      child: RawGestureDetector(
        // The marker's full box is the target — its child is usually a
        // DecoratedBox, which would not absorb hits on its own.
        behavior: HitTestBehavior.opaque,
        gestures: gestures,
        child: child,
      ),
    );
  }
}

/// Hit-tests its child normally — so the child's recognizers join the hit-test
/// path — but reports a miss, letting whatever sits below (the map's platform
/// view) receive the same pointer.
///
/// This is the translucent counterpart to maplibre's own `TranslucentPointer`,
/// which reports a miss *without* descending and so drops taps entirely.
/// Entries added to the [BoxHitTestResult] survive a `false` return; that is
/// what [HitTestBehavior.translucent] relies on too.
class _PointerPassThrough extends SingleChildRenderObjectWidget {
  const _PointerPassThrough({required super.child});

  @override
  RenderProxyBox createRenderObject(BuildContext context) =>
      _RenderPointerPassThrough();
}

class _RenderPointerPassThrough extends RenderProxyBox {
  _RenderPointerPassThrough() : super(null);

  @override
  bool hitTest(BoxHitTestResult result, {required Offset position}) {
    super.hitTest(result, position: position);
    return false;
  }
}

/// Shared bookkeeping of every pointer currently touching the screen.
///
/// A recognizer only ever learns about the pointers routed to it, so it cannot
/// tell a single-finger tap from the first finger of a pinch. One global
/// pointer route — installed once, on first use, and kept for the app's
/// lifetime — closes that gap for a set insert per pointer event.
class _PointerTracker {
  _PointerTracker._() {
    GestureBinding.instance.pointerRouter.addGlobalRoute(_handleEvent);
  }

  static final _PointerTracker instance = _PointerTracker._();

  final Set<int> _downPointers = <int>{};
  final Set<VoidCallback> _listeners = <VoidCallback>{};

  /// Whether a finger other than [pointer] is currently down. Order-independent
  /// on purpose: the global route runs after the hit-test path, so a
  /// recognizer's own pointer may or may not be registered yet.
  bool hasOtherPointer(int pointer) =>
      _downPointers.any((other) => other != pointer);

  void addListener(VoidCallback listener) => _listeners.add(listener);

  void removeListener(VoidCallback listener) => _listeners.remove(listener);

  void _handleEvent(PointerEvent event) {
    final bool changed;
    if (event is PointerDownEvent) {
      changed = _downPointers.add(event.pointer);
    } else if (event is PointerUpEvent ||
        event is PointerCancelEvent ||
        // Belt and braces: a pointer that vanishes without an up/cancel must
        // not stick around and block every later tap.
        event is PointerRemovedEvent) {
      changed = _downPointers.remove(event.pointer);
    } else {
      changed = false;
    }
    if (!changed || _listeners.isEmpty) return;
    for (final listener in List<VoidCallback>.of(_listeners)) {
      listener();
    }
  }
}

/// Refuses to start while another finger is down, and rejects itself the moment
/// a second finger joins — so the map, not the marker, owns every multi-finger
/// gesture.
mixin _SingleFingerOnly on OneSequenceGestureRecognizer {
  int? _activePointer;
  bool _listening = false;

  @override
  void addAllowedPointer(PointerDownEvent event) {
    if (_PointerTracker.instance.hasOtherPointer(event.pointer)) return;
    _activePointer ??= event.pointer;
    super.addAllowedPointer(event);
    if (!_listening) {
      _listening = true;
      _PointerTracker.instance.addListener(_onPointersChanged);
    }
  }

  void _onPointersChanged() {
    final pointer = _activePointer;
    if (pointer == null) return;
    if (_PointerTracker.instance.hasOtherPointer(pointer)) {
      resolve(GestureDisposition.rejected);
    }
  }

  void _stopListening() {
    if (!_listening) return;
    _listening = false;
    _PointerTracker.instance.removeListener(_onPointersChanged);
  }

  @override
  void didStopTrackingLastPointer(int pointer) {
    _activePointer = null;
    _stopListening();
    super.didStopTrackingLastPointer(pointer);
  }

  @override
  void dispose() {
    _stopListening();
    super.dispose();
  }
}

class _SingleFingerTapRecognizer extends TapGestureRecognizer
    with _SingleFingerOnly {}

class _SingleFingerPanRecognizer extends PanGestureRecognizer
    with _SingleFingerOnly {}
