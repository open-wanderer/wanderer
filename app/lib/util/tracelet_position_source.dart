import 'dart:async';

import 'package:geolocator/geolocator.dart' as geo;
import 'package:tracelet/tracelet.dart' as tl;

/// Bridges tracelet's location engine into a [geo.Position] stream so the
/// navigation screen's existing consumers (maneuver provider, stats
/// provider, live marker/camera) remain type-compatible.
///
/// Drives BOTH recording/stats and the live UI off this single stream via
/// two reconfigurable profiles, swapped live with [setForeground] as the app
/// foregrounds/backgrounds — no separate GPS session needed for the UI:
/// - Foreground (navigating): continuous, no distance filter, stop-detection
///   disabled so fixes never throttle down while the user is briefly still.
/// - Background: battery-conscious, 5 m distance filter, tracelet's default
///   adaptive/stationary handling.
///
/// Lifecycle: call [start] once in initState, [setForeground] from
/// `didChangeAppLifecycleState`, [dispose] in dispose().
class TraceletPositionSource {
  final _controller = StreamController<geo.Position>.broadcast();
  StreamSubscription<tl.Location>? _locationSub;

  String? _notificationTitle;
  String? _notificationText;

  Stream<geo.Position> get stream => _controller.stream;

  /// Config for while the navigation screen is foregrounded — continuous
  /// tracking with no distance/stationary throttling. Based on tracelet's
  /// own `highAccuracy` preset, with nested configs chained via their own
  /// `copyWith` rather than replaced wholesale — replacing `geo`/`android`
  /// entirely (as this used to) silently drops the preset's other tuned
  /// values, falling back to each nested config's own class defaults.
  tl.Config _foregroundConfig() => tl.Config.highAccuracy().copyWith(
    geo: tl.Config.highAccuracy().geo.copyWith(distanceFilter: 0.0),
    // No `MotionConfig.copyWith` exists, so this is fully specified rather
    // than chained — but with stop detection disabled, stationary-tracking
    // fields never trigger, so the rest don't matter here.
    motion: const tl.MotionConfig(
      motionDetectionMode: tl.MotionDetectionMode.accelerometer,
      stopTimeout: 3,
      disableStopDetection: true,
    ),
    app: const tl.AppConfig(stopOnTerminate: false),
    android: tl.Config.highAccuracy().android.copyWith(
      foregroundService: _foregroundServiceConfig(),
    ),
  );

  /// Config for while the app is backgrounded — battery-conscious, tolerant
  /// of tracelet's adaptive/stationary-tracking behavior, matching the
  /// values this screen originally intended (see [_foregroundConfig] doc).
  tl.Config _backgroundConfig() => tl.Config.balanced().copyWith(
    geo: tl.Config.balanced().geo.copyWith(
      desiredAccuracy: tl.DesiredAccuracy.high,
      distanceFilter: 5.0,
    ),
    app: const tl.AppConfig(stopOnTerminate: false),
    android: tl.Config.balanced().android.copyWith(
      foregroundService: _foregroundServiceConfig(),
    ),
  );

  tl.ForegroundServiceConfig _foregroundServiceConfig() =>
      tl.ForegroundServiceConfig(
        channelId: 'wanderer_tracking',
        channelName: 'Wanderer Tracking',
        notificationTitle: _notificationTitle!,
        notificationText: _notificationText!,
        notificationSmallIcon: 'ic_notification_icon',
      );

  Future<void> start({
    required String notificationTitle,
    required String notificationText,
  }) async {
    _notificationTitle = notificationTitle;
    _notificationText = notificationText;
    _locationSub = tl.Tracelet.onLocation(_onLocation);

    await tl.Tracelet.ready(_foregroundConfig());
    await tl.Tracelet.start();
  }

  /// Swaps the live config between the foreground (continuous) and
  /// background (battery-conscious) profiles without stopping/restarting
  /// the underlying tracking session.
  Future<void> setForeground(bool foreground) async {
    await tl.Tracelet.setConfig(
      foreground ? _foregroundConfig() : _backgroundConfig(),
    );
  }

  void _onLocation(tl.Location location) {
    if (_controller.isClosed) return;
    final c = location.coords;
    _controller.add(
      geo.Position(
        latitude: c.latitude,
        longitude: c.longitude,
        altitude: c.altitude,
        altitudeAccuracy: c.altitudeAccuracy,
        speed: c.speed,
        speedAccuracy: c.speedAccuracy,
        heading: c.heading,
        headingAccuracy: c.headingAccuracy,
        accuracy: c.accuracy,
        timestamp: DateTime.now(),
      ),
    );
  }

  Future<void> dispose() async {
    await _locationSub?.cancel();
    _locationSub = null;
    await tl.Tracelet.stop();
    await _controller.close();
  }
}
