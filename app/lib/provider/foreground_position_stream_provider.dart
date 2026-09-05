import 'dart:async';

import 'package:flutter/foundation.dart'
    show TargetPlatform, defaultTargetPlatform;
import 'package:flutter/widgets.dart'
    show AppLifecycleState, WidgetsBinding, WidgetsBindingObserver;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';

/// Plain data class carrying a foreground GPS fix for map location markers.
class LocationMarkerPosition {
  final double latitude;
  final double longitude;
  final double accuracy;
  final double? heading;
  final double? headingAccuracy;

  const LocationMarkerPosition({
    required this.latitude,
    required this.longitude,
    required this.accuracy,
    this.heading,
    this.headingAccuracy,
  });
}

/// Signals that the device's location service is disabled.
class ServiceDisabledException implements Exception {
  const ServiceDisabledException();
}

/// Signals that location permission was denied (or permanently denied).
class PermissionDeniedException implements Exception {
  const PermissionDeniedException();
}

/// A singleton foreground position stream shared across all map screens.
///
/// The ghost subscription keeps the broadcast controller alive between
/// navigations so [onListen] fires only once per app session — the GPS
/// activation dialog appears at most once. When the user declines and
/// later enables GPS manually, [getServiceStatusStream] fires [enabled]
/// and we restart [getPositionStream] so the location marker appears
/// without requiring another navigation or app restart.
///
/// The GPS hardware itself is *not* tied to that controller's lifetime — it
/// is reference counted via [ForegroundPositionStream.acquire]/[release].
/// Previously the ghost subscription meant nothing could ever stop the
/// stream, so the receiver ran at 100% duty cycle for the whole app session
/// (measured) even on screens with no map, which in turn drove continuous
/// Play Services WiFi scanning. Consumers now declare their demand instead:
/// use [liveLocationProvider] for a live marker, or
/// [ForegroundPositionStream.currentFix] for a single fix.
final foregroundPositionStreamProvider =
    NotifierProvider<ForegroundPositionStream, Stream<LocationMarkerPosition?>>(
      ForegroundPositionStream.new,
    );

/// Scopes a live GPS subscription to widget lifetime.
///
/// Watching this acquires the receiver; when the last watcher is disposed
/// the receiver stops. Returns the same broadcast stream as
/// [foregroundPositionStreamProvider] so error handling (service disabled /
/// permission denied) is unchanged for callers.
final liveLocationProvider =
    Provider.autoDispose<Stream<LocationMarkerPosition?>>((ref) {
      final notifier = ref.read(foregroundPositionStreamProvider.notifier);
      notifier.acquire();
      ref.onDispose(notifier.release);
      return ref.read(foregroundPositionStreamProvider);
    });

class ForegroundPositionStream extends Notifier<Stream<LocationMarkerPosition?>>
    with WidgetsBindingObserver {
  StreamSubscription<Position>? _positionSub;
  StreamSubscription<ServiceStatus>? _statusSub;
  late StreamController<LocationMarkerPosition?> _controller;

  /// Number of live consumers. The receiver runs only while this is > 0.
  int _consumers = 0;

  /// True while the app is backgrounded (paused/hidden/detached). The
  /// consumers that hold this receiver are map location markers — screens
  /// that stay mounted (and thus keep their acquire) underneath the app
  /// switcher, so without this gate the GPS receiver ran at full duty cycle
  /// for as long as a map screen sat anywhere in the back stack while the
  /// app was backgrounded. Recording is unaffected: tracelet owns its own
  /// native session, entirely separate from this receiver. `inactive` is
  /// deliberately not treated as background (transient — notification
  /// shade, incoming call).
  bool _uiSuspended = false;

  /// Whether a start attempt has already been allowed to raise the system
  /// "Turn on GPS?" dialog this session. See [_startPositionStream].
  bool _promptedForService = false;

  /// `best` requests the most expensive mode the hardware offers; `high`
  /// (~10m) is ample for a map dot. The distance filter is what actually
  /// stops the Play Services location churn — a stationary device produces
  /// no callbacks at all, rather than a fix every second.
  static LocationSettings get _settings {
    const accuracy = LocationAccuracy.high;
    const distanceFilter = 10; // metres
    switch (defaultTargetPlatform) {
      case TargetPlatform.android:
        return AndroidSettings(
          accuracy: accuracy,
          distanceFilter: distanceFilter,
          intervalDuration: const Duration(seconds: 2),
        );
      case TargetPlatform.iOS:
      case TargetPlatform.macOS:
        return AppleSettings(
          accuracy: accuracy,
          distanceFilter: distanceFilter,
          // Tells iOS this is human-locomotion tracking and lets it park the
          // GPS hardware when the user is genuinely still — Core Location
          // resumes on significant motion. Purely a map-marker receiver
          // (recording runs on tracelet's own session), so an auto-pause
          // while stationary costs nothing.
          activityType: ActivityType.fitness,
          pauseLocationUpdatesAutomatically: true,
        );
      default:
        return const LocationSettings(
          accuracy: accuracy,
          distanceFilter: distanceFilter,
        );
    }
  }

  /// Serializes start attempts. `Geolocator.requestPermission()` throws on
  /// iOS if a request is already in flight, and reference counting means an
  /// acquire can land while an earlier start is still awaiting the permission
  /// dialog — as happens when autoDispose churn produces a release/acquire
  /// pair on the first map screen. Chaining rather than dropping keeps a
  /// GPS-enabled restart from being swallowed by an in-flight start.
  Future<void> _startQueue = Future<void>.value();

  void _scheduleStart() {
    _startQueue = _startQueue
        .then((_) => _startPositionStream())
        .catchError((Object _) {});
  }

  /// Registers a live consumer, starting the receiver on the first one.
  void acquire() {
    if (_consumers++ == 0) _scheduleStart();
  }

  /// Releases a live consumer, stopping the receiver once none remain.
  void release() {
    if (_consumers > 0 && --_consumers == 0) _cancelPosition();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    switch (state) {
      case AppLifecycleState.resumed:
        if (!_uiSuspended) return;
        _uiSuspended = false;
        // Restore the receiver for whoever was holding it when the app
        // backgrounded — consumers kept their acquire the whole time.
        if (_consumers > 0) _scheduleStart();
      case AppLifecycleState.inactive:
        // Transient — see [_uiSuspended].
        break;
      case AppLifecycleState.paused:
      case AppLifecycleState.detached:
      case AppLifecycleState.hidden:
        _uiSuspended = true;
        _cancelPosition();
    }
  }

  /// Resolves a single GPS fix, holding the receiver open only as long as it
  /// takes. Returns `null` on timeout, denial, or a disabled location
  /// service — every existing caller already treats a miss as non-fatal.
  Future<LocationMarkerPosition?> currentFix({
    Duration timeout = const Duration(seconds: 10),
  }) async {
    acquire();
    try {
      return await state.firstWhere((p) => p != null).timeout(timeout);
    } catch (_) {
      return null;
    } finally {
      release();
    }
  }

  void _cancelPosition() {
    _positionSub?.cancel();
    _positionSub = null;
  }

  Future<void> _startPositionStream() async {
    _cancelPosition();

    var permission = await Geolocator.checkPermission();
    if (permission == LocationPermission.denied) {
      permission = await Geolocator.requestPermission();
    }
    if (permission == LocationPermission.denied ||
        permission == LocationPermission.deniedForever) {
      if (!_controller.isClosed) {
        _controller.addError(const PermissionDeniedException());
      }
      return;
    }
    if (_controller.isClosed) return;

    // Subscribing while location services are off is what raises the system
    // "Turn on GPS?" dialog. Reference counting means this can now run many
    // times per session, so only the first attempt is allowed to raise it —
    // later ones surface the disabled state instead. This preserves the
    // at-most-once-per-session dialog contract the ghost subscription below
    // was originally introduced to guarantee.
    if (_promptedForService && !await Geolocator.isLocationServiceEnabled()) {
      if (!_controller.isClosed) {
        _controller.addError(const ServiceDisabledException());
      }
      return;
    }
    _promptedForService = true;
    if (_controller.isClosed) return;

    // The awaits above can outlive the consumer that asked for this start
    // (a queued start whose acquirer has since released, or an app that
    // backgrounded mid-start). Starting anyway would run the receiver with
    // nobody watching / nothing visible.
    if (_consumers == 0 || _uiSuspended) return;

    _positionSub = Geolocator.getPositionStream(locationSettings: _settings)
        .listen(
          (pos) {
            if (!_controller.isClosed) {
              _controller.add(
                LocationMarkerPosition(
                  latitude: pos.latitude,
                  longitude: pos.longitude,
                  accuracy: pos.accuracy,
                  heading: pos.heading,
                  headingAccuracy: pos.headingAccuracy,
                ),
              );
            }
          },
          onError: (_) {
            // Swallow position stream errors (GPS declined or toggled off);
            // service status listener handles the disabled → enabled transition.
          },
        );
  }

  @override
  Stream<LocationMarkerPosition?> build() {
    _controller = StreamController<LocationMarkerPosition?>.broadcast(
      onListen: () async {
        // Mirror GPS on/off transitions to the layer.
        _statusSub = Geolocator.getServiceStatusStream().listen((status) {
          if (_controller.isClosed) return;
          if (status == ServiceStatus.enabled) {
            _controller.add(null); // reset layer to "locating"
            // Only resume if something is actually watching; otherwise the
            // receiver would restart on a GPS toggle with no consumer.
            if (_consumers > 0) {
              _scheduleStart();
            }
          } else {
            _cancelPosition();
            _controller.addError(const ServiceDisabledException());
          }
        });

        // Signal immediately if GPS is already off so the layer shows
        // "service disabled" while the activation dialog is pending.
        final serviceEnabled = await Geolocator.isLocationServiceEnabled();
        if (_controller.isClosed) return;
        if (!serviceEnabled) {
          _controller.addError(const ServiceDisabledException());
        }
      },
    );

    // Background gating — see [_uiSuspended].
    WidgetsBinding.instance.addObserver(this);

    ref.onDispose(() {
      WidgetsBinding.instance.removeObserver(this);
      _statusSub?.cancel();
      _cancelPosition();
      _controller.close();
    });

    // Ghost subscription: keeps the broadcast stream alive between
    // navigations so onListen (and therefore the service-status listener)
    // is set up exactly once per session. It deliberately does not start
    // the receiver — acquire() does.
    final ghost = _controller.stream.listen(null, onError: (_) {});
    ref.onDispose(ghost.cancel);

    return _controller.stream;
  }
}
