import 'dart:async';
import 'dart:io' show Platform;

import 'package:flutter_local_notifications/flutter_local_notifications.dart';
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

/// Whether [pos] carries a REAL altitude reading, as opposed to the
/// "no altitude available" placeholder that both [seedPositionFrom] above and
/// geolocator itself express as `altitude == 0` together with
/// `altitudeAccuracy == 0`.
///
/// This is the ONE answer to "does this fix have usable elevation?", the
/// position-stream counterpart to `parseGpxElevation`'s answer for waypoints.
/// Every consumer that anchors or diffs elevation must ask it, because
/// anchoring on a fabricated 0 makes the next genuine reading register as a
/// single-step climb of the device's full absolute altitude (~500 m in
/// Munich) — see the `recording-elevation-gain-jump` debug session.
///
/// Deliberately NOT `altitudeAccuracy > 0` alone: Android only reports
/// vertical accuracy on API 26+ (`Location.hasVerticalAccuracy`), and this app
/// supports API 21+, so a real fix on an older device carries a genuine
/// altitude with `altitudeAccuracy == 0`. Requiring accuracy would silently
/// disable elevation tracking outright on those devices — a worse bug than
/// the one being prevented. A genuine reading of exactly 0 m with no accuracy
/// (true sea level, no vertical accuracy) is the only false negative, and it
/// is harmless: the reference simply anchors on the next fix instead.
bool hasUsableAltitude(geo.Position pos) =>
    pos.altitude.isFinite && (pos.altitudeAccuracy > 0 || pos.altitude != 0);

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

  /// Mirrors the profile last applied via [setForeground] (the session starts
  /// on the foreground config, see [start]), so [setNotificationText] can
  /// re-apply the *current* profile rather than silently demoting a
  /// backgrounded session back to continuous tracking.
  bool _foreground = true;

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
      // The hardware-pedometer path needs ACTIVITY_RECOGNITION, which is
      // stripped from the merged manifest (see AndroidManifest.xml), so ask
      // tracelet for its location-based stationary detection outright rather
      // than letting it attempt the pedometer and fall back.
      disableMotionActivityUpdates: true,
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

    // Ask before the service exists: on Android 13+ the foreground-service
    // notification is suppressed without POST_NOTIFICATIONS, and granting it
    // afterwards does not retroactively surface the running service's
    // notification. Only the trail-download service ever requested it, so on a
    // fresh install a recording ran with no visible notification at all —
    // which is also the only thing telling the user tracking survived the app
    // being swiped away.
    await _ensureNotificationPermission();

    await tl.Tracelet.ready(_foregroundConfig());
    await tl.Tracelet.start();
  }

  /// Best-effort POST_NOTIFICATIONS request for tracelet's foreground-service
  /// notification.
  ///
  /// A no-op below Android 13, where the permission does not exist, and on
  /// iOS, where the notification is not permission-gated the same way.
  /// Failures are swallowed deliberately: a denied or unavailable permission
  /// costs visibility, never the recording itself.
  static Future<void> _ensureNotificationPermission() async {
    if (!Platform.isAndroid) return;
    try {
      await FlutterLocalNotificationsPlugin()
          .resolvePlatformSpecificImplementation<
            AndroidFlutterLocalNotificationsPlugin
          >()
          ?.requestNotificationsPermission();
    } catch (_) {
      // Plugin unavailable or the request threw — tracking proceeds either
      // way, just without a visible notification.
    }
  }

  /// Swaps the live config between the foreground (continuous) and
  /// background (battery-conscious) profiles without stopping/restarting
  /// the underlying tracking session.
  Future<void> setForeground(bool foreground) async {
    _foreground = foreground;
    await tl.Tracelet.setConfig(
      foreground ? _foregroundConfig() : _backgroundConfig(),
    );
  }

  /// Rewrites the foreground-service notification body of the running
  /// session, leaving the tracking profile itself untouched.
  ///
  /// Exists because the navigating notification names the trail (see
  /// `NavigationScreen._notificationText`), and the trail model can still be
  /// resolving when [start] fires — a resumed session pushed straight to the
  /// navigation route at launch has nothing cached to read. A no-op before
  /// [start] has configured the service — [start]'s own argument is the
  /// initial text either way.
  Future<void> setNotificationText(String text) async {
    if (_locationSub == null || _notificationText == text) return;
    _notificationText = text;
    await tl.Tracelet.setConfig(
      _foreground ? _foregroundConfig() : _backgroundConfig(),
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

  /// Tear down the Dart-side listeners but leave the native session running.
  ///
  /// For when the screen goes away while the session is still live — the task
  /// was swiped off the recents list, or the route was popped mid-recording.
  /// `stopOnTerminate: false` (see [_foregroundConfig]) is what lets tracelet
  /// keep recording through it, notification and all; the startup
  /// reconciliation in `main.dart` owns the session from here, resuming it or
  /// calling [stopOrphanedTracking].
  ///
  /// Stopping here instead is what used to end a recording the moment Android
  /// tore the widget tree down: the foreground notification vanished and
  /// nothing was recorded until the app was reopened, while the persisted
  /// session row still offered a resume that silently skipped the gap.
  Future<void> detach() async {
    await _locationSub?.cancel();
    _locationSub = null;
    await _motionSub?.cancel();
    _motionSub = null;
    await _controller.close();
    await _movingController.close();
  }

  /// Tear down everything, native tracking session included. Only correct
  /// when the session is genuinely over — use [detach] otherwise.
  Future<void> dispose() async {
    await detach();
    await tl.Tracelet.stop();
  }

  /// Whether a native tracking session is currently running.
  ///
  /// True after the app process was killed while `stopOnTerminate: false` kept
  /// tracelet recording — the case where the foreground notification is still
  /// up and the user is looking at a live session, so relaunching should drop
  /// them straight back into it rather than asking whether to resume.
  /// Conservatively false when the state cannot be read.
  static Future<bool> isTracking() async {
    try {
      return (await tl.Tracelet.getState()).enabled;
    } catch (_) {
      return false;
    }
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
