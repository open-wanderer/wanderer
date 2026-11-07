import 'package:flutter/material.dart';

/// Tears down a Hybrid Composition platform view before its route pops.
///
/// With `androidMode: hc` a MapLibre map is a real Android `SurfaceView` in
/// the platform view hierarchy rather than a texture composited into the
/// Flutter scene. That is what makes it cheap to render — it avoids the
/// GPU→CPU→GPU copy that cost 76% of a core — but it also means its teardown
/// is not synchronized with Flutter's frame pipeline. On pop the native
/// surface stays composited over the *destination* screen for a few hundred
/// milliseconds while Flutter has already moved on, and the compositor
/// juggling a live map surface mid-transition shows up as raster jank.
///
/// Watching the route's own animation and dropping the platform view the
/// moment the pop starts removes the surface up front, so there is nothing
/// left to linger. A cancelled back-swipe restores it automatically, since
/// the status returns to [AnimationStatus.forward].
///
/// Note there is no way to leave a snapshot behind in its place: platform
/// views under Hybrid Composition do not render into `RepaintBoundary`
/// captures. The replacement has to be a flat fill, so use the same colour
/// as the map's `androidForegroundLoadColor` to keep the swap invisible.
mixin PlatformViewPopGuard<T extends StatefulWidget> on State<T> {
  ModalRoute<dynamic>? _route;
  bool _popping = false;

  /// Whether the platform view should be replaced by a placeholder because
  /// the enclosing route is currently popping.
  bool get platformViewPopping => _popping;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final route = ModalRoute.of(context);
    if (identical(route, _route)) return;
    _route?.animation?.removeStatusListener(_onRouteStatus);
    _route = route;
    // Null when this widget is not hosted in a route (e.g. embedded in a
    // tab or a sheet) — nothing to guard against in that case.
    _route?.animation?.addStatusListener(_onRouteStatus);
  }

  void _onRouteStatus(AnimationStatus status) {
    final popping = status == AnimationStatus.reverse;
    if (popping == _popping || !mounted) return;
    setState(() => _popping = popping);
  }

  @override
  void dispose() {
    _route?.animation?.removeStatusListener(_onRouteStatus);
    super.dispose();
  }
}
