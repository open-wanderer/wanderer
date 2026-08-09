import 'package:wanderer/components/map/map_ui_controls.dart';
import 'dart:async';
import 'dart:convert';
import 'dart:math' show exp, pi;
import 'dart:ui' show lerpDouble;

import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter/scheduler.dart' show Ticker;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_rotation_sensor/flutter_rotation_sensor.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:geolocator/geolocator.dart' as geo;
import 'package:go_router/go_router.dart';
import 'package:gpx/gpx.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:objectbox/objectbox.dart';
import 'package:wanderer/components/base/wanderer_attribution.dart';
import 'package:wanderer/components/map/location_marker_layer.dart';
import 'package:wanderer/components/map/trail_layer.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/components/trail/waypoint_sheet.dart';
import 'package:wanderer/entities/active_navigation_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/glyph_sprite_cache_paths.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/foreground_position_stream_provider.dart';
import 'package:wanderer/provider/glyph_sprite_cache_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/map_style_json_provider.dart';
import 'package:wanderer/provider/navigation_provider.dart';
import 'package:wanderer/provider/navigation_stats_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/region/tile_proxy_provider.dart';
import 'package:wanderer/provider/subcategory_preference_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/store/active_navigation_store.dart' as active_nav;
import 'package:wanderer/util/format.dart';
import 'package:wanderer/util/gpx/gpx.dart';
import 'package:wanderer/util/region/offline_style_rewriter.dart';
import 'package:wanderer/util/geo/polyline.dart';
import 'package:wanderer/util/route/planner_handoff.dart';
import 'package:wanderer/models/route_travel_bucket.dart';
import 'package:wanderer/actions/resolve_track_save_options.dart';
import 'package:wanderer/services/tracelet_position_source.dart';
import 'package:wanderer/actions/import_trail_file.dart';
import 'package:wanderer/util/route/valhalla.dart';

/// The three actions offered by [_NavigationScreenState._confirmExit]'s
/// premature-exit dialog.
enum _NavExitChoice { cancel, exit, saveTrack }

class NavigationScreen extends ConsumerStatefulWidget {
  final String id;
  final NavigateResponse response;
  final bool isOffline;
  final ActiveNavigationEntity? resumeSession;

  /// True for a trail-less GPS-recording session (pushed via the top-level
  /// `/record` route) — false (default) preserves every existing
  /// turn-by-turn navigation call site unchanged. See the recording-mode
  /// branches in `_buildButtonRow`, `_confirmExit`, `_persistNow`, and
  /// `_buildElevationPage`.
  final bool isRecording;

  /// Initial map camera center for a fresh recording session (recording mode
  /// has no `response.shape` to derive one from). Resolved by the caller
  /// (`trail_source_select_screen.dart`'s `_openRecorder`) from a real GPS
  /// fix before pushing `/record`, so the map never opens at the
  /// `Geographic(0, 0)` fallback. Ignored outside recording mode.
  final ml.Geographic? initialCenter;

  /// Already-resolved GPS fix — from `_openRecorder` (recording) or
  /// `launchNavigation` (turn-by-turn) — seeded into [TraceletPositionSource]
  /// so the live marker renders instantly instead of waiting for tracelet's
  /// own cold GPS acquisition. Null for a resumed session (no fresh fix to
  /// hand off) — the marker waits for tracelet's fix same as before.
  final geo.Position? initialPosition;

  /// Valhalla costing (`'pedestrian'`/`'bicycle'`) chosen via
  /// `showTravelProfileSheet` at record start (`_openRecorder`) — gives
  /// "Follow roads" the same profile picker the route planner already has,
  /// instead of always costing as pedestrian for a trail-less recording
  /// (which has no trail category for [costingForCategory] to read). Ignored
  /// outside recording mode; a resumed session reads its own persisted
  /// [ActiveNavigationEntity.recordingCosting] instead (see `initState`).
  final String? recordingCosting;

  const NavigationScreen({
    super.key,
    required this.id,
    required this.response,
    this.isOffline = false,
    this.resumeSession,
    this.isRecording = false,
    this.initialCenter,
    this.initialPosition,
    this.recordingCosting,
  });

  @override
  ConsumerState<NavigationScreen> createState() => _NavigationScreenState();
}

class _NavigationScreenState extends ConsumerState<NavigationScreen>
    with WidgetsBindingObserver, TickerProviderStateMixin {
  static const _trailLayer = TrailLayer();

  late final TraceletPositionSource _positionSource;
  late final Stream<geo.Position> _positionStream;
  StreamSubscription<geo.Position>? _sub;

  /// Drives `NavigationStatsNotifier.setStationary` from tracelet's native
  /// speed-motion engine (via [TraceletPositionSource.isMovingStream]), so
  /// the timer/GPS/stats auto-freeze while stationary and auto-resume on
  /// motion.
  StreamSubscription<bool>? _movingSub;

  /// ObjectBox store, read once in [initState] — used to persist/clear the
  /// single active-session row via `active_navigation_store`.
  late final Store _store;

  /// Resume seeds computed once from [NavigationScreen.resumeSession]. Every
  /// `navigationProvider`/`navigationStatsProvider` call site below must pass
  /// the identical seed fields or the family resolves to a different
  /// (split-brain) provider instance.
  late final int? _resumeManeuverIndex;
  late final List<Wpt>? _resumeBreadcrumb;
  late final NavigationStatsSeed? _resumeStats;

  /// Resolved once in [initState]: a resumed recording's own persisted
  /// costing wins over [NavigationScreen.recordingCosting] (a fresh push
  /// only ever supplies one of the two — resume seeds have no fresh sheet
  /// selection to carry). Null outside recording mode.
  late final String? _recordingCosting;

  /// Family provider instances constructed ONCE (in [initState], after the
  /// resume seeds resolve) and reused for every watch/read/listen in this
  /// file. Constructing them inline per call site both risks split-brain
  /// seeds and re-creates the argument record each time. Note the container
  /// still hashes the argument per lookup (and that hash deep-walks the full
  /// route shape — freezed's DeepCollectionEquality), which is why the
  /// per-GPS-fix path below goes through the cached NOTIFIER references
  /// instead of any provider read.
  late final NavigationProvider _navProviderInstance;
  late final NavigationStatsNotifierProvider _statsProviderInstance;

  /// Notifier references cached once in [initState] — the per-fix hot path
  /// calls these directly, doing ZERO provider lookups per GPS fix. Safe for
  /// the session: both families are kept alive by this screen's own
  /// watch/listen subscriptions and nothing invalidates them mid-session.
  late final Navigation _navNotifier;
  late final NavigationStatsNotifier _statsNotifier;

  /// Mirror of `stats.isPaused || stats.isStationary`, maintained by a
  /// [ref.listenManual] subscription — so the per-fix handler never has to
  /// `ref.read` the stats provider (see [_statsProviderInstance] docs).
  bool _frozen = false;

  /// False while the app is backgrounded (paused/hidden/detached). Gates the
  /// per-fix breadcrumb GeoJSON serialization + platform-channel push — the
  /// native map is paused and invisible, so feeding it geometry is pure
  /// battery waste. Recording itself (breadcrumb, stats, persistence) is
  /// UNAFFECTED: background recording is first-class, only map-feeding
  /// stops. On resume, [_pushBreadcrumbSources] catches the map up in one
  /// update.
  bool _uiVisible = true;

  /// Number of leading breadcrumb points currently baked into the native
  /// `breadcrumb` (frozen) GeoJSON source. Points past this index live in the
  /// small `breadcrumb-tail` source, re-serialized per fix — bounding the
  /// per-fix cost to O(tail) instead of O(whole track), which over a
  /// multi-hour hike made the old single-source push quadratic. The tail is
  /// rolled into the frozen source every [_kBreadcrumbTailMax] points.
  int _frozenCount = 0;
  static const _kBreadcrumbTailMax = 250;

  /// obxId of the single active-session row this screen owns. 0 means "not
  /// yet inserted" — the first [_persistNow] call inserts and this is updated
  /// with the id `active_nav.save` returns so every later save updates the
  /// same row instead of inserting a duplicate.
  int _activeRowObxId = 0;

  /// Periodic best-effort persistence tick (in addition to maneuver-advance
  /// and pause-toggle saves).
  Timer? _persistTimer;

  /// Guards [_saveRecordedTrack] against a double-tap firing two concurrent
  /// conversions/navigations, and drives the completion banner's loading
  /// state.
  bool _savingTrack = false;

  /// Latest *animated* GPS fix, driving both the custom [_LocationMarkerLayer]
  /// marker and camera-follow. A [ValueNotifier] so the marker rebuilds in a
  /// scoped `ValueListenableBuilder` without a full-screen `setState`. Sourced
  /// from [_positionSource] (tracelet), which runs a continuous, unfiltered
  /// foreground config while the screen is active (see
  /// [TraceletPositionSource.setForeground]) — no separate GPS stream needed.
  /// Values are interpolated between raw fixes by [_positionAnimController]
  /// — see [_onFix].
  final ValueNotifier<LocationMarkerPosition?> _currentPosition = ValueNotifier(
    null,
  );

  /// Short per-fix position tween smoothing the marker/camera between raw GPS
  /// fixes — mirrors the 200ms `fastOutSlowIn` `flutter_map_location_marker`'s
  /// `CurrentLocationLayer` used pre-MapLibre-migration, which this screen lost
  /// when native `trackLocation` was replaced by one-shot `animateCamera` calls
  /// per fix (that native animation cancels/restarts on every call, causing
  /// stutter once fixes arrive faster than its duration).
  late final AnimationController _positionAnimController = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 200),
  )..addListener(_applyAnimatedFrame);
  late final CurvedAnimation _positionCurve = CurvedAnimation(
    parent: _positionAnimController,
    curve: Curves.fastOutSlowIn,
  );

  double? _animStartLat;
  double? _animStartLon;
  double? _animTargetLat;
  double? _animTargetLon;
  double _animAccuracy = 0;

  /// Latest interpolated position, updated every position-tween frame and
  /// reused when a heading-sensor event needs to re-publish the marker/camera.
  double? _lastLat;
  double? _lastLon;

  /// Camera bearing we last told MapLibre to use — the single source of
  /// truth for every `moveCamera` bearing argument. This MapLibre version's
  /// camera-update builder does not reliably treat an omitted (`null`) field
  /// as "leave unchanged", so passing `bearing: null` while a heading-up/
  /// north-return transition is mid-flight can reset or fight the in-flight
  /// rotation, occasionally leaving it stuck partway. Every camera push below
  /// writes this explicit value instead of `null`. Written by either
  /// [_bearingTransitionController] (the two discrete transitions) or
  /// [_bearingFollowTicker] (continuous heading-up follow) — never both at
  /// once, see the latter's `isAnimating` guard in [_onBearingFollowTick].
  double _mapBearing = 0;

  /// One-shot tween for the two discrete, user-triggered bearing transitions
  /// (compass tap to enter/exit heading-up). Continuous heading-up follow is
  /// driven separately by [_bearingFollowTicker], which defers to this
  /// controller while it's animating — so this is the only writer of
  /// [_mapBearing] during those 400ms transitions.
  late final AnimationController _bearingTransitionController =
      AnimationController(
        vsync: this,
        duration: const Duration(milliseconds: 400),
      )..addListener(_applyBearingTransition);
  late final CurvedAnimation _bearingTransitionCurve = CurvedAnimation(
    parent: _bearingTransitionController,
    curve: Curves.easeInOut,
  );
  double _bearingTransitionStart = 0;
  double _bearingTransitionTarget = 0;

  /// Per-frame ticker that continuously eases [_mapBearing] toward
  /// [_smoothedHeading] while following in heading-up mode — decouples the
  /// camera's update rate from the heading sensor's throttled ~15Hz cadence
  /// so rotation reads as smooth as a native gesture instead of stepping.
  /// Started/stopped by [_syncBearingFollowTicker].
  late final Ticker _bearingFollowTicker = createTicker(_onBearingFollowTick);

  /// Elapsed timestamp of the previous [_bearingFollowTicker] tick that
  /// actually advanced [_mapBearing] — used to compute a frame-rate
  /// independent smoothing factor. Reset to `null` whenever the ticker
  /// (re)starts or is gated off, so resuming never applies one giant
  /// catch-up jump from a stale baseline.
  Duration? _lastBearingFollowElapsed;

  /// Time constant (seconds) for [_bearingFollowTicker]'s exponential
  /// smoothing — smaller converges faster but jitters more.
  static const double _kBearingFollowTauSeconds = 0.15;

  /// Device heading in degrees (0 = north, clockwise), sourced from the
  /// orientation sensor ([_headingSub]) independently of GPS — GPS `heading`
  /// is only produced while moving and is useless when turning in place.
  /// Null until the first sensor event. Low-pass smoothed via [_lerpBearing]
  /// since the raw magnetometer signal is jittery. Also the convergence
  /// target [_bearingFollowTicker] eases [_mapBearing] toward in heading-up
  /// follow, rather than something snapped straight to the camera.
  double? _smoothedHeading;
  StreamSubscription<OrientationEvent>? _headingSub;
  static const _kHeadingSmoothingAlpha = 0.35;

  final DraggableScrollableController _sheetController =
      DraggableScrollableController();
  final DraggableScrollableController _waypointSheetController =
      DraggableScrollableController();

  Waypoint? _selectedWaypoint;

  static const _kSheetMinSize = 0.2;
  static const _kSheetStatsSize = 0.3;
  static const _kSheetElevationSize = 0.45;

  ml.MapController? _controller;

  /// Buffers a style-loaded event that arrives before [_controller] is set —
  /// the native platform channel does not reliably fire `onMapCreated` before
  /// `onStyleLoaded` (the same race `TrailCollectionMap` guards against).
  ml.StyleController? _pendingStyle;

  /// The last successfully-resolved (and possibly offline-rewritten) style
  /// JSON. Cached so a provider refresh (e.g. a theme toggle) never drops us
  /// back to the loading state and remounts the map — the live swap goes
  /// through [ml.MapController.setStyle] instead.
  String? _lastStyleJson;

  /// Identity-keyed memo of [_composeStyle]'s last inputs/output — see the
  /// build() comment at the compose call site.
  String? _composeBaseInput;
  GlyphSpriteCachePaths? _composeCacheInput;
  String? _composeOutput;

  /// [ml.MapOptions] built exactly once (first build with a resolved style):
  /// `MapOptions` has no value equality, so a fresh instance per build
  /// defeats the plugin's `didUpdateWidget` early-out and re-issues its
  /// min/max zoom + pitch JNI setters on every rebuild. Every field in it is
  /// init-only anyway (style/theme changes post-creation go through
  /// [_swapStyle], never through options).
  ml.MapOptions? _mapOptions;

  bool _cacheWarmed = false;

  /// Active pointer count on the map surface. `CameraChangeReason.apiGesture`
  /// fires identically for pan/pinch/rotate (no native sub-classification
  /// exists), so this heuristic narrows follow-break to a single-finger
  /// drag. Read synchronously inside `onEvent` only — never mutated via
  /// `setState`.
  int _activePointers = 0;

  bool _followEnabled = true;
  bool _headingUp = false;
  bool _showingElevation = false;
  // Ceiling for maxChildSize — raised before animating to elevation and lowered
  // only after the shrink animation finishes, so the sheet never gets clamped
  // mid-animation.
  bool _sheetAtElevationSize = false;

  @override
  void initState() {
    super.initState();

    WidgetsBinding.instance.addObserver(this);

    _store = ref.read(objectBoxProvider);
    final resumeSession = widget.resumeSession;
    _resumeManeuverIndex = resumeSession?.currentManeuverIndex;
    _activeRowObxId = resumeSession?.obxId ?? 0;
    _recordingCosting =
        resumeSession?.recordingCosting ?? widget.recordingCosting;
    final resumePos = resumeSession?.breadcrumbPolyline != null
        ? PolylineUtil.decode(resumeSession!.breadcrumbPolyline!)
        : null;
    // elevations/timestampsUtc may be null or shorter than the decoded
    // polyline (e.g. a row persisted before these fields existed) — index
    // defensively rather than assuming they're always the same length.
    final resumeElevations = resumeSession?.elevations;
    final resumeTimestamps = resumeSession?.timestampsUtc;
    _resumeBreadcrumb = resumePos
        ?.mapIndexed(
          (i, pos) => Wpt(
            lat: pos.lat,
            lon: pos.lon,
            ele: (resumeElevations != null && i < resumeElevations.length)
                ? resumeElevations[i]
                : null,
            time: (resumeTimestamps != null && i < resumeTimestamps.length)
                ? DateTime.fromMillisecondsSinceEpoch(resumeTimestamps[i])
                : null,
          ),
        )
        .toList();
    _resumeStats = resumeSession != null
        ? NavigationStatsSeed(
            distanceMeters: resumeSession.distanceMeters,
            elevationGainMeters: resumeSession.elevationGainMeters,
            elevationLossMeters: resumeSession.elevationLossMeters,
            elapsed: Duration(seconds: resumeSession.currentElapsedSeconds),
            pausedAccum: Duration(seconds: resumeSession.pausedAccumSeconds),
            isPaused: resumeSession.isPaused,
          )
        : null;

    // Family instances + notifiers cached once — see the field docs. Every
    // later read/watch/listen in this file MUST go through these so the
    // family seeds can never split-brain.
    _navProviderInstance = navigationProvider(
      widget.response,
      resumeManeuverIndex: _resumeManeuverIndex,
      resumeBreadcrumb: _resumeBreadcrumb,
    );
    _statsProviderInstance = navigationStatsProvider(
      widget.response,
      resume: _resumeStats,
    );
    _navNotifier = ref.read(_navProviderInstance.notifier);
    _statsNotifier = ref.read(_statsProviderInstance.notifier);
    // Keeps the per-fix handler free of provider reads: mirror the frozen
    // flag on every stats emission instead of re-deriving it per fix.
    ref.listenManual(_statsProviderInstance, fireImmediately: true, (_, next) {
      _frozen = next.isPaused || next.isStationary;
    });

    if (resumeSession == null) {
      // Fresh session: clear any stale prior-trail row, then write an
      // initial zeroed row for this trail.
      active_nav.clear(_store);
      _persistNow();
    }
    _persistTimer = Timer.periodic(
      const Duration(seconds: 10),
      (_) => _persistNow(),
    );

    _positionSource = TraceletPositionSource();
    _positionStream = _positionSource.stream;
    // Couples tracelet's native speed-motion engine to the stats notifier:
    // stationary → freeze timer/GPS-power/stats; moving → auto-resume. The
    // notifier never computes motion itself, it only reacts to this stream.
    _movingSub = _positionSource.isMovingStream.listen((moving) {
      _statsNotifier.setStationary(!moving);
    });
    // AppLocalizations.of(context) isn't safe to call synchronously here —
    // inherited-widget dependencies aren't established until after the first
    // frame — so the notification-text lookup (and thus `start()`) is
    // deferred by one frame.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      final localizations = AppLocalizations.of(context)!;
      unawaited(
        _positionSource.start(
          notificationTitle: localizations.location_tracking_notification_title,
          notificationText: localizations.location_tracking_notification_text,
          seed: widget.initialPosition,
        ),
      );
    });
    // Single stream drives both recording/stats and the live marker/camera —
    // TraceletPositionSource swaps between a continuous foreground config and
    // a battery-conscious background config (see [_positionSource.setForeground],
    // called from [didChangeAppLifecycleState]) rather than running a second,
    // separate GPS session just for the UI.
    _sub = _positionStream.listen(
      (pos) {
        // HOT PATH — one call per GPS fix for the whole session. Zero
        // provider lookups here by design: every ref.read hashes the family
        // argument, which deep-walks the full route shape (see the
        // _navProviderInstance field docs).
        //
        // Frozen (manually paused and/or tracelet-detected stationary) must
        // not append to the breadcrumb — that's the data persisted to disk
        // and exported as the saved trail's GPX. Maneuver-advance detection
        // is unrelated bookkeeping and must keep running even while frozen,
        // so it's not gated here — only the breadcrumb append is.
        final advanced = _navNotifier.onPosition(
          ml.Geographic(lat: pos.latitude, lon: pos.longitude),
          // `null`, never a fabricated 0, when the fix carries no real
          // altitude (see [hasUsableAltitude]). The breadcrumb IS the saved
          // trail's GPX, and computeTrailMetrics deliberately skips waypoints
          // with no usable `ele` so the first point that does carry elevation
          // becomes the anchor. Passing 0 here instead would bake a
          // ~absolute-altitude phantom climb into every saved recording that
          // started from an already-resolved map-marker position.
          altitude: hasUsableAltitude(pos) ? pos.altitude : null,
          heading: pos.heading,
          headingAccuracy: pos.headingAccuracy,
          speed: pos.speed,
          accuracy: pos.accuracy,
          recordBreadcrumb: !_frozen,
        );
        _statsNotifier.onPosition(pos);
        _onFix(pos);
        if (advanced) {
          _persistNow();
        }
      },
      onError: (Object error) {
        debugPrint('NavigationScreen: GPS stream error — $error');
      },
    );

    _startHeadingSub();
  }

  /// Subscribes to the device orientation sensor for heading — decoupled from
  /// GPS so the marker/map keep rotating when the user turns in place (GPS
  /// `heading` only updates while moving). Same source `flutter_map_location_
  /// marker` uses. Foreground-only: paused/resumed alongside [_foregroundSub].
  void _startHeadingSub() {
    _headingSub?.cancel();
    if (!RotationSensor.isPlatformSupported) return;
    // ~15 Hz — always this file's documented intent, but `uiInterval` is
    // 16.7ms (~60 Hz on most devices): 4× the sensor callbacks and marker
    // publishes this screen was designed for, for the whole session. 66ms
    // is plenty: the low-pass smoothing plus [_bearingFollowTicker]'s
    // per-frame easing already decouple perceived rotation smoothness from
    // the sensor cadence.
    RotationSensor.samplingPeriod = const Duration(milliseconds: 66);
    _headingSub = RotationSensor.orientationStream.listen(
      (event) {
        // azimuth: radians, 0 = north, clockwise — same convention as the
        // MapLibre camera bearing. Package already normalises it to 0–2π.
        _onHeading(event.eulerAngles.azimuth * 180 / pi);
      },
      onError: (Object error) {
        debugPrint('NavigationScreen: heading sensor error — $error');
      },
    );
  }

  /// Records a new raw GPS fix as the position-tween target and (re)starts the
  /// short tween toward it — mirrors `flutter_map_location_marker`'s
  /// `CurrentLocationLayer`, which disposed/restarted its own per-fix tween
  /// the same way, so overlapping fixes never fight each other.
  void _onFix(geo.Position pos) {
    _animStartLat = _lastLat ?? pos.latitude;
    _animStartLon = _lastLon ?? pos.longitude;
    _animTargetLat = pos.latitude;
    _animTargetLon = pos.longitude;
    _animAccuracy = pos.accuracy;
    _positionAnimController.forward(from: 0);
  }

  /// Low-pass smooths the raw sensor heading (jittery magnetometer) into
  /// [_smoothedHeading] and pushes it to the marker, on the sensor's own
  /// ~15Hz cadence, independent of GPS. In heading-up follow, the camera
  /// bearing itself is eased toward this value every rendered frame by
  /// [_bearingFollowTicker] rather than snapped here — see that ticker's
  /// docs for why.
  void _onHeading(double rawDegrees) {
    final normalized = rawDegrees % 360 + (rawDegrees < 0 ? 360 : 0);
    final previous = _smoothedHeading;
    _smoothedHeading = previous == null
        ? normalized
        : _lerpBearing(previous, normalized, _kHeadingSmoothingAlpha);
    // A rotation under ~0.3° is invisible at marker size — skip the
    // republish (and its marker rebuild) while the heading is effectively
    // still. Position changes republish independently via the tween, so this
    // can never starve position updates.
    if (previous != null &&
        _bearingDelta(previous, _smoothedHeading!).abs() < 0.3) {
      return;
    }
    _publishMarker();
  }

  /// Per-frame tick for [_bearingFollowTicker] — eases [_mapBearing] toward
  /// the live [_smoothedHeading] target using an elapsed-time-based alpha
  /// (frame-rate independent), decoupling the camera's update rate from the
  /// heading sensor's throttled ~15Hz cadence so rotation reads as smooth as
  /// a native gesture instead of stepping once per sensor sample.
  void _onBearingFollowTick(Duration elapsed) {
    final target = _smoothedHeading;
    // Don't fight an in-flight compass-triggered transition — it owns
    // _mapBearing until it finishes, then this ticker resumes from there.
    if (target == null || _bearingTransitionController.isAnimating) {
      _lastBearingFollowElapsed = elapsed;
      return;
    }
    final last = _lastBearingFollowElapsed;
    _lastBearingFollowElapsed = elapsed;
    if (last == null) return;
    final dtSeconds =
        (elapsed - last).inMicroseconds / Duration.microsecondsPerSecond;
    if (dtSeconds <= 0) return;
    final alpha = 1 - exp(-dtSeconds / _kBearingFollowTauSeconds);
    final previous = _mapBearing;
    _mapBearing = _lerpBearing(_mapBearing, target, alpha);
    // Converged: once the per-frame easing step falls below ~0.02° the
    // rotation is done to sub-pixel precision — skip the JNI camera push.
    // Without this the ticker issues a native moveCamera every display frame
    // (60–120 Hz) for the entire heading-up session even while standing
    // still. The position tween pushes the camera independently, so centering
    // never depends on this.
    if (_bearingDelta(previous, _mapBearing).abs() < 0.02) return;
    _pushCamera();
  }

  /// Starts/stops [_bearingFollowTicker] to match whether continuous
  /// heading-up follow should currently be live — called after every change
  /// to [_followEnabled], [_headingUp], or [_headingSub].
  void _syncBearingFollowTicker() {
    final shouldRun = _followEnabled && _headingUp && _headingSub != null;
    if (shouldRun && !_bearingFollowTicker.isTicking) {
      _lastBearingFollowElapsed = null;
      _bearingFollowTicker.start();
    } else if (!shouldRun && _bearingFollowTicker.isTicking) {
      _bearingFollowTicker.stop();
    }
  }

  /// Shortest-path angular interpolation so the marker/camera never spin the
  /// long way around a 359°→1° wraparound. Reused for both per-frame bearing
  /// lerp and per-event heading smoothing.
  double _lerpBearing(double from, double to, double t) {
    var diff = (to - from) % 360;
    if (diff > 180) diff -= 360;
    if (diff < -180) diff += 360;
    return (from + diff * t) % 360;
  }

  /// Shortest-path signed angular difference in degrees ([-180, 180]) —
  /// the wraparound-safe companion to [_lerpBearing], used for the
  /// below-visible-threshold guards in [_onHeading] and
  /// [_onBearingFollowTick].
  double _bearingDelta(double from, double to) {
    var diff = (to - from) % 360;
    if (diff > 180) diff -= 360;
    if (diff < -180) diff += 360;
    return diff;
  }

  /// Composes the latest interpolated position with the latest smoothed
  /// heading into the marker's [ValueNotifier]. Called from both the position
  /// tween ([_applyAnimatedFrame]) and heading sensor ([_onHeading]) so marker
  /// position and rotation stay independently live.
  void _publishMarker() {
    final lat = _lastLat;
    final lon = _lastLon;
    if (lat == null || lon == null) return;
    _currentPosition.value = LocationMarkerPosition(
      latitude: lat,
      longitude: lon,
      accuracy: _animAccuracy,
      heading: _smoothedHeading,
      // LocationPuck.hasValidHeading requires a non-negative accuracy; the
      // sensor reports -1 on iOS, so pass a synthetic 0 whenever a heading is
      // available — the wedge is a direction indicator, not an accuracy gauge.
      headingAccuracy: _smoothedHeading == null ? null : 0,
    );
  }

  /// Ticks every frame of the position tween, advancing the interpolated
  /// position and (when following) the camera center — `moveCamera` is an
  /// instant, non-animated set so nothing fights this Flutter-side tween the
  /// way native `animateCamera` did.
  void _applyAnimatedFrame() {
    final targetLat = _animTargetLat;
    final targetLon = _animTargetLon;
    if (targetLat == null || targetLon == null) return;

    _lastLat = lerpDouble(_animStartLat, targetLat, _positionCurve.value)!;
    _lastLon = lerpDouble(_animStartLon, targetLon, _positionCurve.value)!;
    _publishMarker();
    _pushCamera();
  }

  /// Advances the bearing-transition tween, driving the same explicit
  /// [_mapBearing]/[_pushCamera] path everything else uses.
  void _applyBearingTransition() {
    _mapBearing = _lerpBearing(
      _bearingTransitionStart,
      _bearingTransitionTarget,
      _bearingTransitionCurve.value,
    );
    _pushCamera();
  }

  /// Smoothly animates [_mapBearing] to [target] — used for the two discrete
  /// compass-triggered transitions (enter/exit heading-up).
  void _animateBearingTo(double target) {
    _bearingTransitionStart = _mapBearing;
    _bearingTransitionTarget = target;
    _bearingTransitionController.forward(from: 0);
  }

  /// Pushes an explicit center+bearing to the map when following — never a
  /// partial update, since this MapLibre version doesn't reliably treat an
  /// omitted field as "unchanged" (see [_mapBearing]'s doc comment).
  void _pushCamera() {
    if (!_followEnabled) return;
    final lat = _lastLat;
    final lon = _lastLon;
    if (lat == null || lon == null) return;
    _controller?.moveCamera(
      center: ml.Geographic(lat: lat, lon: lon),
      bearing: _mapBearing,
    );
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    switch (state) {
      case AppLifecycleState.resumed:
        _uiVisible = true;
        unawaited(_positionSource.setForeground(true));
        _startHeadingSub();
        _syncBearingFollowTicker();
        // Catch the paused native map up on everything recorded while
        // backgrounded — per-fix pushes are skipped while !_uiVisible.
        _pushBreadcrumbSources(ref.read(_navProviderInstance).breadcrumb);
      case AppLifecycleState.inactive:
        // Transient (notification shade, incoming call, app-switcher peek) —
        // treating it as background used to bounce tracelet through a full
        // native setConfig round-trip and kill/restart the heading sensor on
        // every shade pull. A real background always delivers paused/hidden
        // right after, so doing nothing here loses nothing.
        break;
      case AppLifecycleState.paused:
      case AppLifecycleState.detached:
      case AppLifecycleState.hidden:
        _uiVisible = false;
        unawaited(_positionSource.setForeground(false));
        _headingSub?.cancel();
        _headingSub = null;
        _syncBearingFollowTicker();
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _sub?.cancel();
    _movingSub?.cancel();
    _headingSub?.cancel();
    _persistTimer?.cancel();
    _positionAnimController.dispose();
    _bearingTransitionController.dispose();
    _bearingFollowTicker.dispose();
    unawaited(_positionSource.dispose());
    _currentPosition.dispose();
    _sheetController.dispose();
    _waypointSheetController.dispose();
    super.dispose();
  }

  /// Persists a snapshot of the current navigation progress + stats +
  /// breadcrumb to the single active-session row, updating (never
  /// duplicating) the row via [_activeRowObxId]. Best-effort — see
  /// `active_navigation_store`'s swallow-all semantics.
  void _persistNow() {
    final navState = ref.read(_navProviderInstance);
    final stats = ref.read(_statsProviderInstance);

    final entity = ActiveNavigationEntity(
      obxId: _activeRowObxId,
      sessionType: widget.isRecording
          ? ActiveSessionType.rec
          : ActiveSessionType.nav,
      trailId: widget.isRecording ? null : widget.id,
      isOffline: widget.isOffline,
      recordingCosting: widget.isRecording ? _recordingCosting : null,
      currentManeuverIndex: navState.currentManeuverIndex,
      breadcrumbPolyline: PolylineUtil.encode(
        navState.breadcrumb
            .map((wpt) => ml.Geographic(lon: wpt.lon!, lat: wpt.lat!))
            .toList(),
      ),
      // ele/time can be null for a point rehydrated from a resumed session
      // whose persisted elevations/timestampsUtc were null or short (see
      // initState) — default rather than force-unwrap so a resumed session
      // can always be persisted again.
      elevations: navState.breadcrumb.map((wpt) => wpt.ele ?? 0.0).toList(),
      timestampsUtc: navState.breadcrumb
          .map(
            (wpt) =>
                (wpt.time ?? DateTime.now()).toUtc().millisecondsSinceEpoch,
          )
          .toList(),
      distanceMeters: stats.distanceMeters,
      elevationGainMeters: stats.elevationGainMeters,
      elevationLossMeters: stats.elevationLossMeters,
      currentElapsedSeconds: stats.elapsed.inSeconds,
      pausedAccumSeconds: _statsNotifier.pausedAccum.inSeconds,
      isPaused: stats.isPaused,
      updatedAtUtc: DateTime.now().toUtc(),
    );
    _activeRowObxId = active_nav.save(_store, entity);
  }

  /// Whether the recorded breadcrumb has enough points for "Save track" to be
  /// offered at all — a 0-1 point track has nothing meaningful to convert.
  /// Gates the option's *visibility* on the completion banner and exit
  /// dialog, rather than accepting the tap and no-op'ing inside
  /// [_saveRecordedTrack].
  bool _hasSavableTrack() {
    return ref.read(_navProviderInstance).breadcrumb.length >= 2;
  }

  /// Builds a stub [Trail] from the recorded breadcrumb (via the same
  /// on-device conversion the Route Planner and GPX-file import use — see
  /// `buildDraftTrail`) and hands off to `trail_create_screen`.
  /// Callers must only invoke this with a breadcrumb of >=2 points (see the
  /// completion-banner and exit-dialog guards) — the conversion isn't
  /// meaningful for a near-empty track.
  ///
  /// Reads `navState` via `ref.read` with the IDENTICAL family seed args used
  /// everywhere else in this file — a different seed would resolve a
  /// different (split-brain) provider instance.
  ///
  /// Guarded by [_savingTrack] so a double-tap can't fire two concurrent
  /// conversions/navigations (mirrors `route_planner_screen.dart`'s
  /// `_finishing`). On failure (e.g. offline), shows an error toast and
  /// leaves the session intact so the user can retry — matching
  /// `import_trail_file.dart`'s `importTrailFile` precedent for this same
  /// toast-and-stay behaviour.
  ///
  /// Opens the shared online gate (see `resolve_track_save_options.dart`)
  /// FIRST, before the [_savingTrack] guard, so both call sites
  /// (exit-dialog and completion-banner) inherit it with no change. The
  /// sheet itself is now shown only when online; offline, the save
  /// proceeds straight through with both transforms off.
  /// Cancelling/dismissing an online sheet still aborts the save entirely —
  /// no change to the session.
  ///
  /// When "Follow roads" is on, the breadcrumb is snapped to the road
  /// network via [snapShapeToRoads] BEFORE "Recalculate heights" runs, so
  /// elevation reflects the final (possibly snapped) shape. Both transforms
  /// are best-effort with silent fallback (see [snapShapeToRoads]'s and
  /// `/valhalla/height`'s own fallback behavior) — a failure never blocks
  /// the save nor surfaces an error toast.
  ///
  /// Any transform path (snap and/or heights) yields a timeless track — the
  /// merge/handoff helpers here are elevation-only, matching the planner
  /// handoff. Only the no-transform path preserves the recorded breadcrumb's
  /// timestamps verbatim.
  Future<void> _saveRecordedTrack(BuildContext context) async {
    final options = await resolveTrackSaveOptions(
      ref,
      context,
      TrackSaveOptionsSource.recording,
    );
    if (options == null) return;
    if (_savingTrack) return;
    setState(() => _savingTrack = true);

    try {
      final navState = ref.read(_navProviderInstance);
      final navStats = ref.read(_statsProviderInstance);

      final originalTrail = ref.read(trailProvider(widget.id)).value;

      Gpx gpx;
      final (recalcHeights, followRoads) = options;
      if (recalcHeights || followRoads) {
        final breadcrumbPoints = [
          for (final wpt in navState.breadcrumb)
            if (wpt.lat != null && wpt.lon != null)
              ml.Geographic(lat: wpt.lat!, lon: wpt.lon!),
        ];
        // Full recorded resolution — NOT [buildNavShape]'s downsampled form.
        // Downsampling to Valhalla's 500-point cap is fine for an outbound
        // routing *request* hint but must never define what gets saved.
        var workingShape = [
          for (final p in breadcrumbPoints) {'lat': p.lat, 'lon': p.lon},
        ];

        if (followRoads && workingShape.length >= 2) {
          // `_recordingCosting` (from `showTravelProfileSheet` at record
          // start, see `_openRecorder`) wins for a trail-less recording,
          // where `originalTrail` is always null and `costingForCategory`
          // would otherwise always fall back to pedestrian regardless of
          // the recorded activity. Falls through to the trail's own
          // category for a real trail's navigate/save flow.
          final costing =
              _recordingCosting ??
              costingForTrail(
                originalTrail,
                subcategories: ref.read(subcategoryProvider),
              );
          // buildNavShape's cap applies only to this outbound hint — the
          // matched path Valhalla returns replaces workingShape entirely.
          // `fallbackShape` is what enforces that: without it a failed or
          // rejected snap returned the hint itself, so a flaky connection
          // silently saved a 500-point decimation of a full-resolution
          // recording.
          workingShape = await snapShapeToRoads(
            ref,
            buildNavShape(breadcrumbPoints),
            costing,
            fallbackShape: workingShape,
          );
        }

        // Real recorded start/end times survive the transform (snap/height
        // merge builds a fresh trackless Gpx from lat/lon/ele only) so the
        // server can still derive an accurate duration instead of 0.
        final startTime = navState.breadcrumb.firstOrNull?.time;
        final endTime = navState.breadcrumb.lastOrNull?.time;

        if (recalcHeights && workingShape.length >= 2) {
          final heights = await fetchHeightsForShape(ref, workingShape);
          gpx = mergeHeightsIntoGpx(
            workingShape,
            heights,
            startTime: startTime,
            endTime: endTime,
          );
        } else {
          gpx = mergeHeightsIntoGpx(
            workingShape,
            const [],
            startTime: startTime,
            endTime: endTime,
          );
        }
      } else {
        gpx = buildGpxFromPoints(navState.breadcrumb);
      }

      // For a real trail, keep its own category/subcategory. For a trail-less
      // recording, derive a (sub)category from the chosen travel profile
      // (the same settings-driven pre-fill the route planner uses) rather
      // than always leaving it unset.
      //
      // Precision limit: `_recordingCosting` only ever carries the binary
      // `'bicycle'`/`'pedestrian'` Valhalla costing string (captured via
      // `bucket.costing` in trail_source_select_screen's `_openRecorder`),
      // not the full RouteTravelBucket — so a recording started on
      // MTB/Gravel/Road collapses to the generic Hybrid bike bucket here.
      // That is a pre-existing limitation, not a new regression: widening it
      // needs ActiveNavigationEntity/router_provider schema changes, which
      // are out of scope for this task.
      final recordingBucket = _recordingCosting == null
          ? null
          : (_recordingCosting == 'bicycle'
                ? RouteTravelBucket.bikingHybrid
                : RouteTravelBucket.hiking);
      final selection = recordingBucket == null
          ? null
          : categorySelectionForBucket(
              recordingBucket,
              ref.read(categoryProvider).value ?? const [],
              ref.read(subcategoryProvider),
              // Never auto-assign a subcategory the user has hidden.
              subcategoryPrefs:
                  ref.read(subcategoryPreferenceProvider).value ?? const [],
            );

      final category = originalTrail?.categoryId ?? selection?.categoryId;
      final subcategory =
          originalTrail?.subcategoryId ?? selection?.subcategoryId;

      // NavigationStats.elapsed is already moving time — the 1-second
      // tick is a no-op while isPaused || isStationary, and it is already
      // the value shown to the user during the session, so no second
      // derivation is invented here. No planner-style estimated duration
      // fallback is passed either — `duration` must come from the
      // GPX so a later web recompute reproduces it.
      final trail = await buildDraftTrail(
        ref,
        gpx,
        category: category,
        subcategory: subcategory,
        movingDuration: navStats.elapsed,
      );

      active_nav.clear(_store);

      pendingImportedTrail = trail;
      if (context.mounted) {
        context.pushReplacement('/trail/create/edit', extra: trail);
      }
    } catch (_) {
      if (!context.mounted) return;
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: AppLocalizations.of(context)!.error_saving_trail,
            ),
          );
    } finally {
      if (mounted) setState(() => _savingTrack = false);
    }
  }

  void _onWaypointSelected(Waypoint wp) {
    setState(() => _selectedWaypoint = wp);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted && _waypointSheetController.isAttached) {
        _waypointSheetController.animateTo(
          0.35,
          duration: const Duration(milliseconds: 300),
          curve: Curves.easeOut,
        );
      }
    });
  }

  void _onPanStart() {
    if (_followEnabled) {
      setState(() => _followEnabled = false);
      _syncBearingFollowTicker();
    }
  }

  void _onRecenter() {
    setState(() => _followEnabled = true);
    _syncBearingFollowTicker();
    final pos = _currentPosition.value;
    if (pos == null) return;
    // Restores prior heading-up state — recenter never forces north. Synced
    // into _mapBearing immediately so a position tick landing mid-animation
    // pushes the same value instead of a stale/ambiguous one (see
    // _mapBearing's doc comment).
    _mapBearing = _headingUp ? (_smoothedHeading ?? _mapBearing) : 0;
    _controller?.animateCamera(
      center: ml.Geographic(lat: pos.latitude, lon: pos.longitude),
      bearing: _mapBearing,
      nativeDuration: const Duration(milliseconds: 300),
    );
  }

  /// Composes the style JSON to hand to the map from the two resolved
  /// inputs.
  ///
  /// Online: [baseJson] as-is. Offline: [baseJson] rewritten via
  /// [rewriteStyleForProxy] so `glyphs`/`sprite` resolve from [cache] and the
  /// protomaps/hillshade tiles resolve from the loopback tile proxy
  /// — a single static XYZ source, with per-tile region coverage
  /// resolved server-side (`resolveRegionForTile`) rather than a live
  /// viewport query here. This screen's camera moves freely across a
  /// session, unlike `TrailMap`'s fixed trail bounds, but the proxy's
  /// per-request resolution means no viewport parameter is needed at all.
  /// Returns null while a required input is still resolving — the caller
  /// then shows the loading passthrough (initStyle path) or leaves the
  /// mounted style unchanged (`_swapStyle` path). An uncovered viewport
  /// resolves to a blank basemap via the proxy's own 404 responses.
  String? _composeStyle(String? baseJson, GlyphSpriteCachePaths? cache) {
    if (baseJson == null) return null;
    if (!widget.isOffline) return baseJson;
    if (cache == null) return null;
    try {
      final decoded = jsonDecode(baseJson) as Map<String, dynamic>;
      final offlineStyle = rewriteStyleForProxy(
        decoded,
        cacheRoot: cache.root,
        proxyBaseUrl: ref.read(tileProxyBaseUrlProvider),
        dark:
            effectiveBrightness(ref.read(themeModeProvider)) == Brightness.dark,
      );
      return jsonEncode(offlineStyle);
    } catch (e) {
      debugPrint('NavigationScreen: offline style rewrite failed — $e');
      return null;
    }
  }

  /// Recomposes the (possibly offline-rewritten) style from current provider
  /// state and swaps it onto the mounted controller in place. Theme/glyph
  /// path only (unchanged mechanism) — the static proxy source is baked into
  /// every composed style by construction, so no separate region-swap path
  /// is needed here.
  void _swapStyle() {
    final controller = _controller;
    if (controller == null) return;
    final baseJson = widget.isOffline
        ? ref.read(offlineMapStyleJsonProvider).value
        : ref.read(mapStyleJsonProvider).value;
    final cache = widget.isOffline
        ? ref.read(offlineGlyphSpritePathsProvider).value
        : null;
    final json = _composeStyle(baseJson, cache);
    if (json != null && json != _lastStyleJson) {
      _lastStyleJson = json;
      controller.setStyle(json);
    }
  }

  /// JSON-encodes the breadcrumb as a single LineString Feature. Never string
  /// concatenation — keeps the geometry structurally isolated.
  ///
  /// A GeoJSON `LineString` requires at least 2 coordinates (RFC 7946
  /// §3.1.4) — the native MapLibre parser rejects a 0/1-point one, which
  /// would throw on the very first `addSource` call (before any GPS fix has
  /// landed) and permanently skip creating the `breadcrumb`
  /// source/`breadcrumb-route` layer for the rest of the session. So below
  /// 2 points this returns an empty (but valid) FeatureCollection instead,
  /// which renders nothing until real geometry is available.
  String _breadcrumbGeoJson(List<Wpt> pts) {
    if (pts.length < 2) {
      return jsonEncode(<String, Object?>{
        'type': 'FeatureCollection',
        'features': <Object?>[],
      });
    }
    return jsonEncode(<String, Object?>{
      'type': 'Feature',
      'properties': <String, Object?>{},
      'geometry': <String, Object?>{
        'type': 'LineString',
        'coordinates': <List<double>>[
          for (final p in pts) <double>[p.lon!, p.lat!],
        ],
      },
    });
  }

  /// Pushes the breadcrumb to the map across its two native sources: the
  /// `breadcrumb` (frozen) source holding points `[0.._frozenCount)`, updated
  /// only when the tail rolls over, and the `breadcrumb-tail` source holding
  /// the still-growing remainder (sharing one overlap point so the line reads
  /// as continuous), re-serialized per fix. Bounds per-fix serialization +
  /// platform-channel traffic to O([_kBreadcrumbTailMax]) — pushing the whole
  /// track per fix was quadratic over a session. The roll itself is O(track),
  /// amortized to O(1)-per-fix by the chunk size.
  ///
  /// Idempotent against arbitrary gaps: passing the full current list after
  /// any number of skipped pushes (style reload, backgrounded interval)
  /// converges both sources — callers never need to replay missed fixes.
  void _pushBreadcrumbSources(List<Wpt> pts) {
    final style = _controller?.style;
    if (style == null) return;
    if (pts.length - _frozenCount >= _kBreadcrumbTailMax) {
      style
          .updateGeoJsonSource(id: 'breadcrumb', data: _breadcrumbGeoJson(pts))
          .catchError((Object e) {
            debugPrint('NavigationScreen: failed to update breadcrumb — $e');
          });
      _frozenCount = pts.length;
    }
    final tailStart = _frozenCount == 0 ? 0 : _frozenCount - 1;
    style
        .updateGeoJsonSource(
          id: 'breadcrumb-tail',
          data: _breadcrumbGeoJson(pts.sublist(tailStart)),
        )
        .catchError((Object e) {
          debugPrint('NavigationScreen: failed to update breadcrumb tail — $e');
        });
  }

  /// Re-arms everything that binds to the current native `Style` object:
  /// `setStyle` (used for theme swaps) drops added layers/sources, so this
  /// must run after every style load, not just once at `onMapCreated`. The
  /// location marker itself is a Flutter `_LocationMarkerLayer` (not a
  /// native style layer), so it survives style swaps untouched.
  Future<void> _onStyleLoaded(ml.StyleController style) async {
    try {
      final trail = ref.read(trailProvider(widget.id)).value;
      if (trail?.expand?.gpx != null) {
        await _trailLayer.add(style, trail!);
      }

      final breadcrumb = ref.read(_navProviderInstance).breadcrumb;
      // Two-source split: everything so far seeds the frozen source; the
      // tail source starts empty and takes the per-fix updates — see
      // [_pushBreadcrumbSources].
      await style.addSource(
        ml.GeoJsonSource(
          id: 'breadcrumb',
          data: _breadcrumbGeoJson(breadcrumb),
        ),
      );
      _frozenCount = breadcrumb.length;
      await style.addSource(
        ml.GeoJsonSource(
          id: 'breadcrumb-tail',
          data: _breadcrumbGeoJson(const []),
        ),
      );
      // Style-spec defaults to butt cap / miter join, which reads as
      // angular at turns — round both so the trail renders as a
      // continuously smooth line, matching the pre-migration
      // flutter_map `Polyline`'s auto-rounded rendering. Both breadcrumb
      // layers share identical paint so the frozen/tail split is invisible.
      const breadcrumbPaint = {'line-color': '#DC2626', 'line-width': 3.5};
      const breadcrumbLayout = {'line-cap': 'round', 'line-join': 'round'};
      await style.addLayer(
        const ml.LineStyleLayer(
          id: 'breadcrumb-route',
          sourceId: 'breadcrumb',
          paint: breadcrumbPaint,
          layout: breadcrumbLayout,
        ),
      );
      await style.addLayer(
        const ml.LineStyleLayer(
          id: 'breadcrumb-route-tail',
          sourceId: 'breadcrumb-tail',
          paint: breadcrumbPaint,
          layout: breadcrumbLayout,
        ),
      );
    } catch (e) {
      debugPrint('NavigationScreen: onStyleLoaded failed — $e');
    }
  }

  @override
  Widget build(BuildContext context) {
    // Cache warm on first open (mirrors TrailMap's caching pattern) —
    // idempotent against the trail-download trigger. Skipped when offline:
    // the warm is a network download, and the cache is already populated by
    // the time an offline navigation session renders from it.
    if (!_cacheWarmed && !widget.isOffline) {
      _cacheWarmed = true;
      ref.read(glyphSpriteCacheProvider.future).ignore();
    }

    // Live style swap: theme toggle or (offline) glyph/sprite cache warm
    // swaps the composed style in place on the already-mounted map. Region
    // coverage is resolved by the loopback tile proxy per-tile:
    // a newly-downloaded region's tiles resolve the next time MapLibre
    // requests them (confirmed on-device, no remount needed) — no separate
    // region-change listener is required.
    // Offline reads the network-free providers so no `/map/style-sources`
    // call is ever made.
    if (widget.isOffline) {
      ref.listen(offlineMapStyleJsonProvider, (_, _) => _swapStyle());
      ref.listen(offlineGlyphSpritePathsProvider, (_, _) => _swapStyle());
    } else {
      ref.listen(mapStyleJsonProvider, (_, _) => _swapStyle());
    }

    // Breadcrumb in-place update: swap the native tail source's data on every
    // new position fix, never remove/re-add sources. Keyed on
    // breadcrumbLength — the list object itself is identity-stable by design
    // (see NavigationState.breadcrumb). Skipped while backgrounded: the
    // native map is paused, [_pushBreadcrumbSources] catches it up on resume.
    ref.listen(_navProviderInstance, (prev, next) {
      if (prev?.breadcrumbLength == next.breadcrumbLength) return;
      if (!_uiVisible) return;
      _pushBreadcrumbSources(next.breadcrumb);
    });

    // Deliberately NOT a whole-state watch: navigation state changes per GPS
    // fix and stats tick at 1 Hz — a whole-state watch rebuilt this entire
    // screen (map included) at ≥1 Hz for the whole session. The banner only
    // needs the maneuver index; the stats sheet and pause button watch their
    // own slices via scoped Consumers below.
    final currentIndex = ref.watch(
      _navProviderInstance.select((s) => s.currentManeuverIndex),
    );
    final trailAsync = ref.watch(trailProvider(widget.id));
    final user = ref.watch(authProvider).requireValue;
    final localizations = AppLocalizations.of(context)!;
    final unit = ref.watch(unitProvider);

    final baseAsync = widget.isOffline
        ? ref.watch(offlineMapStyleJsonProvider)
        : ref.watch(mapStyleJsonProvider);
    final baseJson = baseAsync.value;
    Object? error = baseAsync.error;

    GlyphSpriteCachePaths? cache;
    if (widget.isOffline) {
      final cacheAsync = ref.watch(offlineGlyphSpritePathsProvider);
      cache = cacheAsync.value;
      error ??= cacheAsync.error;
    }

    // Memoized on input identity: the offline path's compose is a full
    // style-JSON decode → rewrite → encode round-trip (100s of KB), far too
    // heavy to re-run on every incidental rebuild of this screen.
    if (!identical(baseJson, _composeBaseInput) ||
        !identical(cache, _composeCacheInput)) {
      _composeBaseInput = baseJson;
      _composeCacheInput = cache;
      _composeOutput = _composeStyle(baseJson, cache);
    }
    final composed = _composeOutput;
    if (composed != null) _lastStyleJson = composed;
    final styleJson = _lastStyleJson;

    final maneuvers = widget.response.maneuvers;
    final isArrived =
        currentIndex >= maneuvers.length - 1 && maneuvers.isNotEmpty;

    return PopScope(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) _confirmExit(context, localizations);
      },
      child: Scaffold(
        body: styleJson == null
            ? (error != null
                  ? Center(child: Text(error.toString()))
                  : const Center(child: CircularProgressIndicator()))
            : Stack(
                children: [
                  // ----------------------------------------------------------
                  // Full-screen map
                  // ----------------------------------------------------------
                  Listener(
                    onPointerDown: (_) => _activePointers++,
                    onPointerUp: (_) =>
                        _activePointers = (_activePointers - 1).clamp(0, 10),
                    onPointerCancel: (_) =>
                        _activePointers = (_activePointers - 1).clamp(0, 10),
                    child: ml.MapLibreMap(
                      options: _mapOptions ??= ml.MapOptions(
                        initStyle: styleJson,
                        initCenter:
                            widget.initialCenter ??
                            (widget.response.shapeAsGeographic.isNotEmpty
                                ? widget.response.shapeAsGeographic.first
                                : const ml.Geographic(lat: 0, lon: 0)),
                        initZoom: 15,
                        androidForegroundLoadColor: Theme.of(
                          context,
                        ).colorScheme.surface,
                        // See TrailMap for why texture mode is off and `hc`
                        // is pinned.
                        androidTextureMode: false,
                        androidMode: ml.AndroidPlatformViewMode.hc,
                      ),
                      onMapCreated: (controller) {
                        _controller = controller;
                        final pending = _pendingStyle;
                        if (pending != null) {
                          _pendingStyle = null;
                          _onStyleLoaded(pending);
                        }
                      },
                      onStyleLoaded: (style) {
                        if (_controller == null) {
                          _pendingStyle = style;
                          return;
                        }
                        _onStyleLoaded(style);
                      },
                      onEvent: (event) {
                        if (event is ml.MapEventClick) {
                          setState(() => _selectedWaypoint = null);
                        } else if (event is ml.MapEventStartMoveCamera &&
                            event.reason == ml.CameraChangeReason.apiGesture &&
                            _activePointers <= 1 &&
                            _followEnabled) {
                          // CameraChangeReason.apiGesture fires identically
                          // for pan/pinch/rotate (no native
                          // sub-classification exists); _activePointers <= 1
                          // is the compensating heuristic so only a
                          // single-finger drag breaks follow.
                          _onPanStart();
                        }
                      },
                      children: [
                        if (trailAsync.value?.expand?.gpx != null)
                          TrailMarkerLayer(
                            trail: trailAsync.value!,
                            selectedWaypoint: _selectedWaypoint,
                            onWaypointTap: _onWaypointSelected,
                          ),
                        _LocationMarkerLayer(
                          position: _currentPosition,
                          headingUp: _headingUp,
                        ),

                        Positioned(
                          top: widget.isRecording ? 8 : 128,
                          left: 8,
                          child: SafeArea(
                            child: const WandererMapScalebar(
                              alignment: Alignment.topLeft,
                            ),
                          ),
                        ),
                        WandererAttribution(
                          alignment: Alignment.bottomLeft,
                          padding: EdgeInsets.only(
                            left: 10,
                            bottom:
                                MediaQuery.of(context).size.height *
                                _kSheetMinSize,
                          ),
                        ), //
                        Positioned(
                          top: widget.isRecording ? 8 : 128,
                          right: 8,
                          child: SafeArea(
                            child: Column(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                WandererMapCompass(
                                  hideIfRotatedNorth: false,
                                  rotateNorthOnPressed: false,
                                  onPressed: () {
                                    // Continuous rotation via
                                    // [_bearingFollowTicker] is gated on
                                    // `_followEnabled && _headingUp` — without
                                    // re-enabling follow here, the transition
                                    // below would be the only rotation that
                                    // ever happens if follow was already (or
                                    // becomes) broken.
                                    setState(() {
                                      _headingUp = !_headingUp;
                                      if (_headingUp) _followEnabled = true;
                                    });
                                    _syncBearingFollowTicker();
                                    // Animates via [_mapBearing]/[_pushCamera]
                                    // rather than a native `animateCamera` —
                                    // that native animation could get
                                    // interrupted/reset mid-flight by a
                                    // concurrent position-tick camera push,
                                    // occasionally leaving it stuck partway.
                                    _animateBearingTo(
                                      _headingUp ? (_smoothedHeading ?? 0) : 0,
                                    );
                                  },
                                ),
                                const SizedBox(height: 8),
                                IconButton(
                                  onPressed: _followEnabled
                                      ? null
                                      : _onRecenter,
                                  icon: const FaIcon(
                                    FontAwesomeIcons.locationCrosshairs,
                                  ),
                                  style: IconButton.styleFrom(
                                    backgroundColor: Theme.of(
                                      context,
                                    ).colorScheme.surface,
                                    disabledBackgroundColor: Theme.of(
                                      context,
                                    ).colorScheme.surface,
                                    disabledForegroundColor: Colors.grey,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),

                  if (!widget.isRecording)
                    SafeArea(
                      child: Padding(
                        padding: const EdgeInsets.fromLTRB(24, 8, 24, 0),
                        child: _buildBanner(
                          context,
                          localizations,
                          maneuvers,
                          currentIndex,
                          isArrived,
                          unit,
                        ),
                      ),
                    ),

                  _buildStatsSheet(context, localizations, trailAsync, unit),

                  Positioned(
                    left: 16,
                    right: 16,
                    bottom: MediaQuery.of(context).padding.bottom,
                    child: _buildButtonRow(context, localizations),
                  ),

                  if (_selectedWaypoint != null)
                    WaypointSheet(
                      waypoint: _selectedWaypoint!,
                      user: user,
                      controller: _waypointSheetController,
                      onClose: () => setState(() => _selectedWaypoint = null),
                    ),
                ],
              ),
      ),
    );
  }

  void _confirmExit(BuildContext context, AppLocalizations localizations) {
    showDialog<_NavExitChoice>(
      context: context,
      builder: (ctx) => AlertDialog(
        content: Text(
          widget.isRecording
              ? localizations.stop_recording_confirm
              : localizations.stop_navigation_confirm,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(_NavExitChoice.cancel),
            child: Text(localizations.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(_NavExitChoice.exit),
            child: Text(localizations.exit_navigation),
          ),
          if (_hasSavableTrack())
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(_NavExitChoice.saveTrack),
              child: Text(localizations.save_track),
            ),
        ],
      ),
    ).then((choice) {
      switch (choice) {
        case _NavExitChoice.saveTrack:
          if (context.mounted) _saveRecordedTrack(context);
        case _NavExitChoice.exit:
          // Deliberate exit — best-effort clear so no stale resume prompt
          // appears on next launch.
          active_nav.clear(_store);
          if (context.mounted) {
            context.pop();
          }
        case _NavExitChoice.cancel:
        case null:
          // Cancel, or barrier dismiss — do nothing.
          break;
      }
    });
  }

  /// Maps a Valhalla maneuver type (0–38) to the closest Material icon.
  /// https://valhalla.github.io/valhalla/api/turn-by-turn/api-reference/#maneuver-types
  static IconData _iconForManeuverType(int type) => switch (type) {
    1 || 2 || 3 => Icons.navigation, // start / start-right / start-left
    4 || 5 || 6 => Icons.flag, // destination
    7 || 8 => Icons.straight, // becomes / continue
    9 => Icons.turn_slight_right,
    10 => Icons.turn_right,
    11 => Icons.turn_sharp_right,
    12 => Icons.u_turn_right,
    13 => Icons.u_turn_left,
    14 => Icons.turn_sharp_left,
    15 => Icons.turn_left,
    16 => Icons.turn_slight_left,
    17 || 22 => Icons.straight, // ramp straight / stay straight
    18 ||
    20 ||
    23 => Icons.turn_slight_right, // ramp-right / exit-right / stay-right
    19 ||
    21 ||
    24 => Icons.turn_slight_left, // ramp-left / exit-left / stay-left
    26 || 27 => Icons.roundabout_right, // roundabout enter / exit
    28 || 29 => Icons.directions_boat, // ferry
    37 => Icons.turn_slight_right, // merge right
    38 => Icons.turn_slight_left, // merge left
    _ => Icons.navigation,
  };

  Widget _buildBanner(
    BuildContext context,
    AppLocalizations localizations,
    List<NavigateManeuver> maneuvers,
    int currentIndex,
    bool isArrived,
    String unit,
  ) {
    return Card(
      elevation: 4,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      color: Theme.of(context).colorScheme.surface,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16.0, vertical: 12.0),
        child: isArrived
            ? _buildCompletionBannerContent(context, localizations)
            : _buildActiveBannerContent(
                context,
                localizations,
                maneuvers,
                currentIndex,
                unit,
              ),
      ),
    );
  }

  Widget _buildActiveBannerContent(
    BuildContext context,
    AppLocalizations localizations,
    List<NavigateManeuver> maneuvers,
    int currentIndex,
    String unit,
  ) {
    if (maneuvers.isEmpty) {
      return const SizedBox.shrink();
    }
    final safeIndex = currentIndex.clamp(0, maneuvers.length - 1);
    final maneuver = maneuvers[safeIndex];

    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Padding(
          padding: const EdgeInsets.only(top: 2.0, right: 8.0),
          child: Icon(
            _iconForManeuverType(maneuver.type),
            color: Theme.of(context).colorScheme.onSurface,
            size: 24,
          ),
        ),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                maneuver.instruction,
                style: Theme.of(context).textTheme.titleLarge,
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
              ),
              const SizedBox(height: 4),
              Text(
                localizations.in_distance(
                  formatDistance(maneuver.length * 1000, unit: unit),
                ),
                style: Theme.of(context).textTheme.bodyMedium,
              ),
            ],
          ),
        ),
        if (widget.isOffline) ...[
          const SizedBox(width: 8),
          Icon(
            Icons.cloud_off,
            color: Theme.of(context).colorScheme.onSurface,
            size: 20,
          ),
        ],
      ],
    );
  }

  Widget _buildCompletionBannerContent(
    BuildContext context,
    AppLocalizations localizations,
  ) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        Row(
          children: [
            FaIcon(FontAwesomeIcons.circleCheck, color: Colors.greenAccent),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    localizations.you_have_arrived,
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                  const SizedBox(height: 4),
                  Text(
                    localizations.reached_end_of_trail,
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                ],
              ),
            ),
          ],
        ),
        if (_hasSavableTrack()) ...[
          const SizedBox(height: 12),
          FilledButton.icon(
            onPressed: _savingTrack ? null : () => _saveRecordedTrack(context),
            icon: _savingTrack
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const FaIcon(FontAwesomeIcons.floppyDisk, size: 16),
            label: Text(localizations.save_track),
          ),
        ],
      ],
    );
  }

  Widget _buildStatsSheet(
    BuildContext context,
    AppLocalizations localizations,
    AsyncValue<Trail> trailAsync,
    String unit,
  ) {
    final theme = Theme.of(context);
    final maxSize = _sheetAtElevationSize
        ? _kSheetElevationSize
        : _kSheetStatsSize;

    return DraggableScrollableSheet(
      controller: _sheetController,
      initialChildSize: _kSheetMinSize,
      minChildSize: _kSheetMinSize,
      maxChildSize: maxSize,
      snap: true,
      snapSizes: [_kSheetMinSize, maxSize],
      builder: (context, scrollController) {
        return Container(
          decoration: BoxDecoration(
            color: theme.canvasColor,
            borderRadius: const BorderRadius.vertical(top: Radius.circular(16)),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withValues(alpha: 0.15),
                blurRadius: 10,
                offset: const Offset(0, -2),
              ),
            ],
          ),
          child: SingleChildScrollView(
            controller: scrollController,
            // Scoped stats watch: the 1 Hz tick and per-fix stat updates
            // rebuild only this sheet content, never the screen (see the
            // build() comment where the whole-state watches used to live).
            child: Consumer(
              builder: (context, ref, _) {
                final stats = ref.watch(_statsProviderInstance);
                return _buildStatsSheetContent(
                  context,
                  localizations,
                  stats,
                  trailAsync,
                  unit,
                );
              },
            ),
          ),
        );
      },
    );
  }

  Widget _buildStatsSheetContent(
    BuildContext context,
    AppLocalizations localizations,
    NavigationStats stats,
    AsyncValue<Trail> trailAsync,
    String unit,
  ) {
    final theme = Theme.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        // Drag handle.
        Center(
          child: Container(
            margin: const EdgeInsets.symmetric(vertical: 8),
            width: 40,
            height: 4,
            decoration: BoxDecoration(
              color: theme.colorScheme.onSurface.withValues(alpha: 0.3),
              borderRadius: BorderRadius.circular(2),
            ),
          ),
        ),

        // Always-visible top stats row.
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
          child: Row(
            children: [
              _buildStatCell(
                context,
                localizations.time_in_motion,
                formatElapsed(stats.elapsed),
              ),
              _buildStatCell(
                context,
                localizations.distance,
                formatDistance(stats.distanceMeters, unit: unit),
              ),
              _buildStatCell(
                context,
                localizations.elevation_gain,
                formatElevation(stats.elevationGainMeters, unit: unit),
              ),
            ],
          ),
        ),

        // Additional content fades in as the sheet expands.
        AnimatedBuilder(
          animation: _sheetController,
          builder: (ctx, child) {
            final targetSize = _showingElevation
                ? _kSheetElevationSize
                : _kSheetStatsSize;
            final t = _sheetController.isAttached
                ? ((_sheetController.size - _kSheetMinSize) /
                          (targetSize - _kSheetMinSize))
                      .clamp(0.0, 1.0)
                : 0.0;
            return Opacity(opacity: t, child: child);
          },
          child: AnimatedSwitcher(
            duration: const Duration(milliseconds: 250),
            child: _showingElevation
                ? SizedBox(
                    key: const ValueKey('elevation'),
                    height: 216,
                    child: _buildElevationPage(context, trailAsync),
                  )
                : SizedBox(
                    key: const ValueKey('stats'),
                    child: _buildAdditionalStats(
                      context,
                      localizations,
                      stats,
                      unit,
                    ),
                  ),
          ),
        ),

        // Clearance so content does not hide behind the button overlay.
        const SizedBox(height: 80),
      ],
    );
  }

  Widget _buildAdditionalStats(
    BuildContext context,
    AppLocalizations localizations,
    NavigationStats stats,
    String unit,
  ) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 0),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const SizedBox(height: 12),
          Row(
            children: [
              _buildStatCell(
                context,
                localizations.elevation_loss,
                formatElevation(stats.elevationLossMeters, unit: unit),
              ),
              _buildStatCell(
                context,
                localizations.speed,
                formatSpeed(stats.currentSpeedKmh, unit: unit),
              ),
              _buildStatCell(
                context,
                localizations.average_speed,
                formatSpeed(stats.averageSpeedKmh, unit: unit),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildStatCell(BuildContext context, String label, String value) {
    final theme = Theme.of(context);
    return Expanded(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            label,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
            ),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 4),
          Text(
            value,
            style: const TextStyle(fontSize: 24, fontWeight: FontWeight.bold),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }

  /// Page 1: the reused [ElevationProfile] chart with a top-left back control
  /// returning to the stats page. The map behind the sheet stays interactive.
  ///
  /// Recording mode has no saved trail/GPX to profile (`trailProvider('')`
  /// would resolve to AsyncError) — it builds a LIVE `Gpx` from the
  /// in-progress `navState.breadcrumb` instead, via the same
  /// [buildGpxFromPoints] helper `_saveRecordedTrack` uses for the identical
  /// breadcrumb. Scoped in its own [Consumer] selecting `breadcrumbLength`
  /// (never `breadcrumb` itself — see [NavigationState.breadcrumb]'s
  /// stable-identity doc comment, a `select` on the list would never fire)
  /// so only this page rebuilds per GPS fix, not the whole screen — mirrors
  /// every other scoped watch in this file (see the build() comment on
  /// whole-state watches).
  Widget _buildElevationPage(
    BuildContext context,
    AsyncValue<Trail> trailAsync,
  ) {
    if (widget.isRecording) {
      return Consumer(
        builder: (context, ref, _) {
          ref.watch(_navProviderInstance.select((s) => s.breadcrumbLength));
          // SNAPSHOT, not the live view. `breadcrumb` is an identity-stable
          // UnmodifiableListView over a grow-in-place list, so feeding it
          // straight in makes every rebuild's Gpx wrap the SAME list instance
          // as the previous one — and gpx 2.3.0's Gpx/Trkseg `==` delegates to
          // ListEquality, which short-circuits on `identical`. The two Gpx
          // objects would compare EQUAL no matter how many fixes arrived, so
          // ElevationProfile.didUpdateWidget would never re-parse and the
          // chart would freeze at its first two points. Copying gives each
          // rebuild a distinct list whose differing length fails that
          // equality check cheaply (length is compared before any element).
          final breadcrumb = List<Wpt>.of(
            ref.read(_navProviderInstance).breadcrumb,
          );
          final gpx = buildGpxFromPoints(breadcrumb);
          // Matches elevation_tab.dart's own empty-state gate for the same
          // "not enough points yet" case (route planner's live GPX).
          if (gpx.allPoints.length < 2) {
            return const SizedBox.shrink();
          }
          final elapsed = ref.watch(
            _statsProviderInstance.select((s) => s.elapsed),
          );
          return Padding(
            padding: const EdgeInsets.all(16.0),
            child: ElevationProfile(
              trail: null,
              gpx: gpx,
              enableLineTouch: false,
              durationOverride: elapsed,
            ),
          );
        },
      );
    }
    return trailAsync.when(
      data: (trail) {
        final gpx = trail.expand?.gpx;
        if (gpx == null) {
          return const SizedBox.shrink();
        }
        return Padding(
          padding: const EdgeInsets.all(16.0),
          child: ElevationProfile(
            trail: trail,
            gpx: gpx,
            enableLineTouch: false,
          ),
        );
      },
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(
        child: Text(
          AppLocalizations.of(context)!.error_reading_file,
          style: TextStyle(color: Theme.of(context).colorScheme.error),
        ),
      ),
    );
  }

  /// Dominant Pause/Resume, icon-only FAB — shared verbatim between nav mode
  /// (center slot) and recording mode (left slot). Reflects only the manual
  /// [NavigationStats.isPaused] flag — isStationary (auto-freeze by
  /// tracelet's motion engine) still freezes the timer/breadcrumb but must
  /// not flip this button to its "paused" state, since the user never asked
  /// to pause and onPressed always toggles the manual flag regardless of
  /// isStationary.
  Widget _buildPauseFab(AppLocalizations localizations) {
    // Scoped isPaused watch — the FAB flips on pause toggles without the
    // screen watching the whole (1 Hz-ticking) stats state.
    return Consumer(
      builder: (context, ref, _) {
        final isPaused = ref.watch(
          _statsProviderInstance.select((s) => s.isPaused),
        );
        return FloatingActionButton(
          heroTag: 'nav_pause',
          tooltip: isPaused ? localizations.resume : localizations.pause,
          elevation: 2,
          shape: StadiumBorder(),
          onPressed: () {
            _statsNotifier.togglePause();
            _persistNow();
          },
          child: FaIcon(
            isPaused ? FontAwesomeIcons.play : FontAwesomeIcons.pause,
          ),
        );
      },
    );
  }

  /// Toggle between additional stats and elevation profile — unchanged
  /// between nav and recording mode, always the right-most button.
  Widget _buildElevationFab(AppLocalizations localizations) {
    return FloatingActionButton.small(
      heroTag: 'nav_elevation',
      tooltip: localizations.elevation_profile,
      shape: const StadiumBorder(),
      elevation: 2,
      backgroundColor: Theme.of(context).colorScheme.surface,
      onPressed: () {
        if (!_showingElevation) {
          // Stats → elevation: raise ceiling first, then expand + switch.
          setState(() {
            _sheetAtElevationSize = true;
            _showingElevation = true;
          });
          WidgetsBinding.instance.addPostFrameCallback((_) {
            if (mounted) {
              _sheetController.animateTo(
                _kSheetElevationSize,
                duration: const Duration(milliseconds: 300),
                curve: Curves.easeInOut,
              );
            }
          });
        } else {
          // Elevation → stats: switch content + shrink simultaneously;
          // lower the ceiling only after the animation completes so the
          // sheet is never clamped mid-animation.
          setState(() => _showingElevation = false);
          _sheetController.animateTo(
            _kSheetStatsSize,
            duration: const Duration(milliseconds: 300),
            curve: Curves.easeInOut,
          );
          Future.delayed(const Duration(milliseconds: 300), () {
            if (mounted) setState(() => _sheetAtElevationSize = false);
          });
        }
      },
      child: FaIcon(
        _showingElevation
            ? FontAwesomeIcons.chartSimple
            : FontAwesomeIcons.chartArea,
        color: Theme.of(context).colorScheme.onSurface,
        size: 18,
      ),
    );
  }

  Widget _buildButtonRow(BuildContext context, AppLocalizations localizations) {
    if (widget.isRecording) {
      return Padding(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceAround,
          children: [
            // Left — Pause/Resume toggle.
            _buildPauseFab(localizations),

            // Center — dominant red Stop button; the ONLY finish trigger for
            // a recording session (isArrived is structurally always false
            // with empty maneuvers, so there is no auto-arrival banner).
            FloatingActionButton(
              heroTag: 'rec_stop',
              tooltip: localizations.stop_recording,
              elevation: 2,
              shape: StadiumBorder(),
              backgroundColor: Colors.redAccent,
              onPressed: _savingTrack
                  ? null
                  : () => _confirmExit(context, localizations),
              child: _savingTrack
                  ? SizedBox(
                      height: 24,
                      width: 24,
                      child: const CircularProgressIndicator(
                        color: Colors.white,
                        strokeWidth: 2,
                      ),
                    )
                  : const FaIcon(FontAwesomeIcons.stop),
            ),

            // Right — toggle between additional stats and elevation profile.
            _buildElevationFab(localizations),
          ],
        ),
      );
    }

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceAround,
        children: [
          // Left — Exit prompts for confirmation before popping.
          FloatingActionButton.small(
            heroTag: 'nav_exit',
            tooltip: localizations.exit_navigation,
            elevation: 2,
            shape: StadiumBorder(),
            backgroundColor: Theme.of(context).colorScheme.surface,
            onPressed: () => _confirmExit(context, localizations),
            child: FaIcon(
              FontAwesomeIcons.xmark,
              color: Theme.of(context).colorScheme.onSurface,
              size: 18,
            ),
          ),

          // Center — dominant Pause/Resume.
          _buildPauseFab(localizations),

          // Right — toggle between additional stats and elevation profile.
          _buildElevationFab(localizations),
        ],
      ),
    );
  }
}

/// Custom, appropriately-sized (~22px) location marker replacing the native
/// MapLibre location puck — which has no supported size control on either
/// Android or iOS (`maplibre` 0.3.5). Mirrors the `ml.WidgetLayer`/`ml.Marker`
/// structure shared with `trail_map.dart`/`map_screen.dart` via
/// [LocationMarkerLayer], but much smaller and heading-aware via
/// [LocationPuck].
class _LocationMarkerLayer extends StatelessWidget {
  const _LocationMarkerLayer({required this.position, required this.headingUp});

  /// Latest GPS fix; null until the first fix lands, in which case nothing
  /// renders.
  final ValueNotifier<LocationMarkerPosition?> position;
  final bool headingUp;

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<LocationMarkerPosition?>(
      valueListenable: position,
      builder: (context, pos, _) {
        if (pos == null) return const SizedBox.shrink();
        return ml.WidgetLayer(
          markers: [
            ml.Marker(
              point: ml.Geographic(lat: pos.latitude, lon: pos.longitude),
              size: const Size(44, 44),
              child: LocationPuck(
                size: 44,
                dotSize: 20,
                heading: headingUp ? 0 : pos.heading,
                headingAccuracy: pos.headingAccuracy,
                showHeading: true,
              ),
            ),
          ],
        );
      },
    );
  }
}
