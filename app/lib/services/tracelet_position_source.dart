import 'dart:async';

import 'package:geolocator/geolocator.dart' as geo;
import 'package:tracelet/tracelet.dart' as tl;
import 'package:wanderer/provider/foreground_position_stream_provider.dart'
    show LocationMarkerPosition;

/// Converts an already-resolved [LocationMarkerPosition] (from
/// `foregroundPositionStreamProvider`, typically resolved by the caller
/// before pushing to `NavigationScreen` — see `_openRecorder` and
/// `launchNavigation`) into a seed [geo.Position] for
/// [TraceletPositionSource.start]. Altitude/speed aren't tracked by
/// [LocationMarkerPosition], so they're zeroed rather than left to fall back
/// on stale values.
geo.Position seedPositionFrom(LocationMarkerPosition pos) => geo.Position(
  latitude: pos.latitude,
  longitude: pos.longitude,
  altitude: 0,
  altitudeAccuracy: 0,
  speed: 0,
  speedAccuracy: 0,
  heading: pos.heading ?? 0,
  headingAccuracy: pos.headingAccuracy ?? 0,
  accuracy: pos.accuracy,
  timestamp: DateTime.now(),
);

/// Bridges tracelet's location engine into a [geo.Position] stream so the
/// navigation screen's existing consumers (maneuver provider, stats
/// provider, live marker/camera) remain type-compatible.
///
/// Drives BOTH recording/stats and the live UI off this single stream via
/// two reconfigurable profiles, swapped live with [setForeground] as the app
/// foregrounds/backgrounds — no separate GPS session needed for the UI:
/// - Foreground (navigating): continuous while moving. Motion is detected by
///   tracelet's native GPS-speed state machine
///   ([tl.MotionDetectionMode.speed]), which automatically drops to
///   low-power periodic fixes while stationary and resumes continuous
///   tracking the moment GPS speed confirms movement again — no manual
///   config swap needed for this dimension.
/// - Background: battery-conscious, 5 m distance filter, tracelet's default
///   adaptive/stationary handling.
///
/// Lifecycle: call [start] once in initState, [setForeground] from
/// `didChangeAppLifecycleState`, [dispose] in dispose().
class TraceletPositionSource {
  final _controller = StreamController<geo.Position>.broadcast();
  final _movingController = StreamController<bool>.broadcast();
  StreamSubscription<tl.Location>? _locationSub;
  StreamSubscription<tl.SpeedMotionEvent>? _motionSub;

  String? _notificationTitle;
  String? _notificationText;

  Stream<geo.Position> get stream => _controller.stream;

  /// Emits `true` while tracelet's native speed-motion state machine
  /// considers the user moving (`moving`/`slowing`) and `false` only once it
  /// has confirmed a `stationary` transition — this is the sole "is the user
  /// moving" signal for the rest of the app; nothing here recomputes motion
  /// from raw GPS speed/displacement independently.
  Stream<bool> get isMovingStream => _movingController.stream;

  /// Config for while the navigation screen is foregrounded — continuous
  /// tracking with no distance filter while moving. Based on tracelet's own
  /// `highAccuracy` preset, with nested configs chained via their own
  /// `copyWith` rather than replaced wholesale — replacing `geo`/`android`
  /// entirely (as this used to) silently drops the preset's other tuned
  /// values, falling back to each nested config's own class defaults.
  tl.Config _foregroundConfig() => tl.Config.highAccuracy().copyWith(
    geo: tl.Config.highAccuracy().geo.copyWith(distanceFilter: 0.0),
    // No `MotionConfig.copyWith` exists, so this is fully specified rather
    // than chained. Uses tracelet's native GPS-speed motion state machine
    // (not the accelerometer `stopTimeout`, which is minute-grained) tuned
    // for walking pace rather than tracelet's vehicle-oriented defaults
    // (`speedMovingThreshold: 1.5` m/s, `speedStationaryDelay: 180`s). When
    // the engine declares `stationary` it automatically switches to
    // low-power periodic fixes ([tl.StationaryTrackingMode.periodic]) and
    // switches back to continuous the moment it re-confirms motion — no
    // second, manually-swapped config is needed for this.
    motion: const tl.MotionConfig(
      motionDetectionMode: tl.MotionDetectionMode.speed,
      speedMovingThreshold: 0.4, // m/s (~1.5 km/h) — slow walking still moves
      speedStationaryDelay: 10, // seconds below threshold before stationary
      stationaryTrackingMode: tl.StationaryTrackingMode.periodic,
      stationaryPeriodicInterval: 20, // seconds between fixes while stopped
      stationaryPeriodicAccuracy: tl.DesiredAccuracy.medium,
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
    geo.Position? seed,
  }) async {
    _notificationTitle = notificationTitle;
    _notificationText = notificationText;
    _locationSub = tl.Tracelet.onLocation(_onLocation);
    _motionSub = tl.Tracelet.onSpeedMotionChange(_onSpeedMotionChange);

    // Emit the caller's already-resolved fix immediately so the live marker
    // doesn't sit blank through tracelet's own cold GPS acquisition —
    // overwritten the moment `_onLocation` fires for real.
    if (seed != null && !_controller.isClosed) {
      _controller.add(seed);
    }

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

  /// Maps tracelet's native `moving → slowing → stationary` state machine to
  /// a single is-moving boolean. `slowing` still counts as moving — it's a
  /// grace window before the engine commits to `stationary`, and freezing
  /// during it would cut off accumulation prematurely on every brief slow-down.
  void _onSpeedMotionChange(tl.SpeedMotionEvent event) {
    if (_movingController.isClosed) return;
    _movingController.add(event.state != tl.SpeedMotionState.stationary);
  }

  Future<void> dispose() async {
    await _locationSub?.cancel();
    _locationSub = null;
    await _motionSub?.cancel();
    _motionSub = null;
    await tl.Tracelet.stop();
    await _controller.close();
    await _movingController.close();
  }

  /// Best-effort stop of a native tracking session left running by a killed
  /// app process. `stopOnTerminate: false` (see [_foregroundConfig]) is what
  /// lets an in-progress recording survive termination — but it also means
  /// that when the persisted session row is dropped instead of resumed
  /// (user declines the resume prompt, or the row is unresolvable), nothing
  /// ever told the native service to stop, and it kept tracking — GPS,
  /// foreground notification and all — until the next recording session
  /// reconfigured it. Called from the startup resume-reconciliation paths in
  /// `main.dart`; a no-op (or swallowed error) when nothing is running.
  static Future<void> stopOrphanedTracking() async {
    try {
      await tl.Tracelet.stop();
    } catch (_) {
      // Not running / plugin unavailable — nothing to stop.
    }
  }
}
