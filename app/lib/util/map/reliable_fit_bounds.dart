/// Shared, verified wrapper around [ml.MapController.fitBounds] used by every
/// screen that fits the camera to a set of bounds right after a map's style
/// loads (`trail_map.dart`, `list_detail_map_screen.dart`,
/// `list_detail_screen.dart`, `route_planner_screen.dart`,
/// `settings_offline_regions_map_screen.dart`, `profile_trail_map_screen.dart`).
library;

import 'dart:async';

import 'package:flutter/widgets.dart' show EdgeInsets;
import 'package:maplibre/maplibre.dart' as ml;

/// Extra attempts made if the camera hasn't visibly moved after a
/// `fitBounds()` call.
const int _kMaxAttempts = 5;

/// Delay between retry attempts. Short enough to be imperceptible once the
/// fit has actually applied, long enough to give iOS a chance to finish
/// laying out the platform view before the next attempt.
const Duration _kRetryDelay = Duration(milliseconds: 100);

/// Calls [ml.MapController.fitBounds] and verifies the camera actually
/// changed, retrying a few times if it didn't.
///
/// Works around a known Flutter-iOS platform-view timing race (the same
/// class of bug as flutter/flutter#59502 for `google_maps_flutter`'s
/// `newLatLngBounds`): the very first camera command issued right after a
/// map's style loads can silently no-op on iOS, because
/// `maplibre_ios`'s `fitBounds()` (a thin wrapper around the deprecated
/// native `-[MLNMapView setVisibleCoordinateBounds:edgePadding:animated:]`)
/// can be invoked before the native view has been laid out to its final
/// on-screen frame — our maps load their style from inline JSON rather
/// than a URL, which lets `onStyleLoaded` fire essentially synchronously
/// with platform-view creation. With no valid viewport to fit against, the
/// native call leaves the camera exactly where it started.
///
/// Android's `fitBounds()` goes through a different, non-deprecated code
/// path (`CameraUpdateFactory.newLatLngBounds` + `animateCamera`) and its
/// platform view is already sized when the map is created, so the first
/// attempt there always succeeds and this resolves after one retry-delay
/// wait — no behaviour change from calling `fitBounds()` directly.
Future<void> fitBoundsReliably(
  ml.MapController controller, {
  required ml.LngLatBounds bounds,
  EdgeInsets padding = EdgeInsets.zero,
  Duration nativeDuration = const Duration(milliseconds: 1),
}) async {
  final before = controller.getCamera();

  for (var attempt = 0; attempt < _kMaxAttempts; attempt++) {
    await controller.fitBounds(
      bounds: bounds,
      padding: padding,
      nativeDuration: nativeDuration,
    );
    await Future<void>.delayed(_kRetryDelay);

    if (controller.getCamera() != before) return;
  }
}
