import 'dart:async';
import 'dart:convert';
import 'dart:math' show pi;
import 'dart:ui' show lerpDouble;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_rotation_sensor/flutter_rotation_sensor.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:geolocator/geolocator.dart' as geo;
import 'package:go_router/go_router.dart';
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
import 'package:wanderer/provider/glyph_sprite_cache_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/map_style_json_provider.dart';
import 'package:wanderer/provider/foreground_position_stream_provider.dart';
import 'package:wanderer/provider/navigation_provider.dart';
import 'package:wanderer/provider/navigation_stats_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/util/active_navigation_store.dart' as active_nav;
import 'package:wanderer/util/format_util.dart';
import 'package:wanderer/util/offline_style_rewriter.dart';
import 'package:wanderer/util/polyline_util.dart';
import 'package:wanderer/util/tracelet_position_source.dart';

class NavigationScreen extends ConsumerStatefulWidget {
  final String id;
  final NavigateResponse response;
  final bool isOffline;
  final ActiveNavigationEntity? resumeSession;

  const NavigationScreen({
    super.key,
    required this.id,
    required this.response,
    this.isOffline = false,
    this.resumeSession,
  });

  @override
  ConsumerState<NavigationScreen> createState() => _NavigationScreenState();
}

class _NavigationScreenState extends ConsumerState<NavigationScreen>
    with WidgetsBindingObserver, TickerProviderStateMixin {
  late final TraceletPositionSource _positionSource;
  late final Stream<geo.Position> _positionStream;
  StreamSubscription<geo.Position>? _sub;

  /// ObjectBox store, read once in [initState] — used to persist/clear the
  /// single active-session row via `active_navigation_store`.
  late final Store _store;

  /// Resume seeds computed once from [NavigationScreen.resumeSession]. Every
  /// `navigationProvider`/`navigationStatsProvider` call site below must pass
  /// the identical seed fields or the family resolves to a different
  /// (split-brain) provider instance.
  late final int? _resumeManeuverIndex;
  late final List<ml.Geographic>? _resumeBreadcrumb;
  late final NavigationStatsSeed? _resumeStats;

  /// obxId of the single active-session row this screen owns. 0 means "not
  /// yet inserted" — the first [_persistNow] call inserts and this is updated
  /// with the id `active_nav.save` returns so every later save updates the
  /// same row instead of inserting a duplicate.
  int _activeRowObxId = 0;

  /// Periodic best-effort persistence tick (in addition to maneuver-advance
  /// and pause-toggle saves).
  Timer? _persistTimer;

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
  /// writes this explicit value instead of `null`.
  double _mapBearing = 0;

  /// One-shot tween for the two discrete, user-triggered bearing transitions
  /// (compass tap to enter/exit heading-up). Continuous updates (GPS position
  /// ticks, heading sensor ticks) write [_mapBearing] and push directly
  /// instead — they're already smoothed by [_positionAnimController]/the
  /// sensor's own cadence — so this exists only to animate the discrete jump,
  /// and to be the *only* writer of [_mapBearing] while it's running (see the
  /// `!_bearingTransitionController.isAnimating` guard in [_onHeading]).
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

  /// Device heading in degrees (0 = north, clockwise), sourced from the
  /// orientation sensor ([_headingSub]) independently of GPS — GPS `heading`
  /// is only produced while moving and is useless when turning in place.
  /// Null until the first sensor event. Low-pass smoothed via [_lerpBearing]
  /// since the raw magnetometer signal is jittery.
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
    _resumeBreadcrumb = resumeSession?.breadcrumbPolyline != null
        ? PolylineUtil.decode(resumeSession!.breadcrumbPolyline!)
        : null;
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
        final navProviderInstance = navigationProvider(
          widget.response,
          resumeManeuverIndex: _resumeManeuverIndex,
          resumeBreadcrumb: _resumeBreadcrumb,
        );
        final beforeIndex = ref.read(navProviderInstance).currentManeuverIndex;
        ref
            .read(navProviderInstance.notifier)
            .onPosition(ml.Geographic(lat: pos.latitude, lon: pos.longitude));
        final afterIndex = ref.read(navProviderInstance).currentManeuverIndex;
        ref
            .read(
              navigationStatsProvider(
                widget.response,
                resume: _resumeStats,
              ).notifier,
            )
            .onPosition(pos);
        _onFix(pos);
        if (afterIndex > beforeIndex) {
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
    RotationSensor.samplingPeriod = SensorInterval.uiInterval;
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

  /// Low-pass smooths the raw sensor heading (jittery magnetometer) and pushes
  /// it to the marker and — in heading-up follow — the camera bearing, on the
  /// sensor's own ~15Hz cadence, independent of GPS.
  void _onHeading(double rawDegrees) {
    final normalized = rawDegrees % 360 + (rawDegrees < 0 ? 360 : 0);
    _smoothedHeading = _smoothedHeading == null
        ? normalized
        : _lerpBearing(_smoothedHeading!, normalized, _kHeadingSmoothingAlpha);
    _publishMarker();
    // Don't fight an in-flight compass-triggered transition — it owns
    // _mapBearing until it finishes, then live sensor updates resume here.
    if (_followEnabled &&
        _headingUp &&
        !_bearingTransitionController.isAnimating) {
      _mapBearing = _smoothedHeading!;
      _pushCamera();
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
        unawaited(_positionSource.setForeground(true));
        _startHeadingSub();
      case AppLifecycleState.paused:
      case AppLifecycleState.inactive:
      case AppLifecycleState.detached:
      case AppLifecycleState.hidden:
        unawaited(_positionSource.setForeground(false));
        _headingSub?.cancel();
        _headingSub = null;
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _sub?.cancel();
    _headingSub?.cancel();
    _persistTimer?.cancel();
    _positionAnimController.dispose();
    _bearingTransitionController.dispose();
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
    final navState = ref.read(
      navigationProvider(
        widget.response,
        resumeManeuverIndex: _resumeManeuverIndex,
        resumeBreadcrumb: _resumeBreadcrumb,
      ),
    );
    final statsNotifier = ref.read(
      navigationStatsProvider(widget.response, resume: _resumeStats).notifier,
    );
    final stats = ref.read(
      navigationStatsProvider(widget.response, resume: _resumeStats),
    );

    final entity = ActiveNavigationEntity(
      obxId: _activeRowObxId,
      sessionType: ActiveSessionType.nav,
      trailId: widget.id,
      isOffline: widget.isOffline,
      currentManeuverIndex: navState.currentManeuverIndex,
      breadcrumbPolyline: PolylineUtil.encode(navState.breadcrumb),
      distanceMeters: stats.distanceMeters,
      elevationGainMeters: stats.elevationGainMeters,
      elevationLossMeters: stats.elevationLossMeters,
      currentElapsedSeconds: stats.elapsed.inSeconds,
      pausedAccumSeconds: statsNotifier.pausedAccum.inSeconds,
      isPaused: stats.isPaused,
      updatedAtUtc: DateTime.now().toUtc(),
    );
    _activeRowObxId = active_nav.save(_store, entity);
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
    }
  }

  void _onRecenter() {
    setState(() => _followEnabled = true);
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

  /// Composes the style JSON to hand to the map from the two resolved inputs.
  ///
  /// Online: [baseJson] as-is. Offline: [baseJson] rewritten via
  /// [rewriteStyleForOffline] so `glyphs`/`sprite` resolve from [cache] and the
  /// protomaps tiles resolve from the trail's `.pmtiles` cells
  /// (`pmtiles://file://`). Returns null while a required input is still
  /// resolving or if the rewrite rejects an input — the caller then shows the
  /// loading passthrough.
  String? _composeStyle(String? baseJson, GlyphSpriteCachePaths? cache) {
    if (baseJson == null) return null;
    if (!widget.isOffline) return baseJson;
    if (cache == null) return null;
    final pmTiles = ref.read(trailProvider(widget.id)).value?.pmTiles;
    if (pmTiles == null || pmTiles.isEmpty) return null;
    try {
      final decoded = jsonDecode(baseJson) as Map<String, dynamic>;
      final offlineStyle = rewriteStyleForOffline(
        decoded,
        cacheRoot: cache.root,
        cellPaths: pmTiles,
        demCellPaths:
            ref.read(trailProvider(widget.id)).value?.demPmTiles ?? const [],
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
  /// state and swaps it onto the mounted controller in place.
  void _swapStyle() {
    final controller = _controller;
    if (controller == null) return;
    final baseJson = ref.read(mapStyleJsonProvider).value;
    final cache = widget.isOffline
        ? ref.read(glyphSpriteCacheProvider).value
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
  String _breadcrumbGeoJson(List<ml.Geographic> pts) {
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
          for (final p in pts) <double>[p.lon, p.lat],
        ],
      },
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
        await addTrailTrackLayers(style, trail!);
      }

      final breadcrumb = ref
          .read(
            navigationProvider(
              widget.response,
              resumeManeuverIndex: _resumeManeuverIndex,
              resumeBreadcrumb: _resumeBreadcrumb,
            ),
          )
          .breadcrumb;
      await style.addSource(
        ml.GeoJsonSource(
          id: 'breadcrumb',
          data: _breadcrumbGeoJson(breadcrumb),
        ),
      );
      await style.addLayer(
        const ml.LineStyleLayer(
          id: 'breadcrumb-route',
          sourceId: 'breadcrumb',
          paint: {'line-color': '#DC2626', 'line-width': 3.5},
          // Style-spec defaults to butt cap / miter join, which reads as
          // angular at turns — round both so the trail renders as a
          // continuously smooth line, matching the pre-migration
          // flutter_map `Polyline`'s auto-rounded rendering.
          layout: {'line-cap': 'round', 'line-join': 'round'},
        ),
      );
    } catch (e) {
      debugPrint('NavigationScreen: onStyleLoaded failed — $e');
    }
  }

  @override
  Widget build(BuildContext context) {
    // Cache warm on first open (mirrors TrailMap's caching pattern) —
    // idempotent against the trail-download trigger.
    if (!_cacheWarmed) {
      _cacheWarmed = true;
      ref.read(glyphSpriteCacheProvider.future).ignore();
    }

    // Live style swap: theme toggle or (offline) glyph/sprite cache warm
    // swaps the composed style in place on the already-mounted map.
    ref.listen(mapStyleJsonProvider, (_, _) => _swapStyle());
    if (widget.isOffline) {
      ref.listen(glyphSpriteCacheProvider, (_, _) => _swapStyle());
    }

    // Breadcrumb in-place update: swap the native source's data on every new
    // position fix, never remove/re-add the source.
    ref.listen(
      navigationProvider(
        widget.response,
        resumeManeuverIndex: _resumeManeuverIndex,
        resumeBreadcrumb: _resumeBreadcrumb,
      ),
      (prev, next) {
        if (prev?.breadcrumb == next.breadcrumb) return;
        final style = _controller?.style;
        if (style == null) return;
        style
            .updateGeoJsonSource(
              id: 'breadcrumb',
              data: _breadcrumbGeoJson(next.breadcrumb),
            )
            .catchError((Object e) {
              debugPrint('NavigationScreen: failed to update breadcrumb — $e');
            });
      },
    );

    final navState = ref.watch(
      navigationProvider(
        widget.response,
        resumeManeuverIndex: _resumeManeuverIndex,
        resumeBreadcrumb: _resumeBreadcrumb,
      ),
    );
    final stats = ref.watch(
      navigationStatsProvider(widget.response, resume: _resumeStats),
    );
    final trailAsync = ref.watch(trailProvider(widget.id));
    final user = ref.watch(authProvider).requireValue;
    final localizations = AppLocalizations.of(context)!;
    final unit = ref.watch(unitProvider);

    final baseAsync = ref.watch(mapStyleJsonProvider);
    final baseJson = baseAsync.value;
    Object? error = baseAsync.error;

    GlyphSpriteCachePaths? cache;
    if (widget.isOffline) {
      final cacheAsync = ref.watch(glyphSpriteCacheProvider);
      cache = cacheAsync.value;
      error ??= cacheAsync.error;
    }

    final composed = _composeStyle(baseJson, cache);
    if (composed != null) _lastStyleJson = composed;
    final styleJson = _lastStyleJson;

    final maneuvers = widget.response.maneuvers;
    final currentIndex = navState.currentManeuverIndex;
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
                      options: ml.MapOptions(
                        initStyle: styleJson,
                        initCenter: widget.response.shapeAsGeographic.isNotEmpty
                            ? widget.response.shapeAsGeographic.first
                            : const ml.Geographic(lat: 0, lon: 0),
                        initZoom: 15,
                        androidForegroundLoadColor: Theme.of(
                          context,
                        ).colorScheme.surface,
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
                          top: 128,
                          left: 8,
                          child: SafeArea(
                            child: const ml.MapScalebar(
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
                          top: 128,
                          right: 8,
                          child: SafeArea(
                            child: Column(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                ml.MapCompass(
                                  hideIfRotatedNorth: false,
                                  rotateNorthOnPressed: false,
                                  onPressed: () {
                                    // Continuous rotation in [_onHeading] is
                                    // gated on `_followEnabled && _headingUp`
                                    // — without re-enabling follow here, the
                                    // transition below would be the only
                                    // rotation that ever happens if follow
                                    // was already (or becomes) broken.
                                    setState(() {
                                      _headingUp = !_headingUp;
                                      if (_headingUp) _followEnabled = true;
                                    });
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

                  _buildStatsSheet(
                    context,
                    localizations,
                    stats,
                    trailAsync,
                    unit,
                  ),

                  Positioned(
                    left: 16,
                    right: 16,
                    bottom: MediaQuery.of(context).padding.bottom,
                    child: _buildButtonRow(context, localizations, stats),
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
    showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        content: Text(localizations.stop_navigation_confirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(localizations.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(localizations.exit_navigation),
          ),
        ],
      ),
    ).then((confirmed) {
      if (confirmed == true) {
        // Deliberate exit — best-effort clear so no stale resume prompt
        // appears on next launch.
        active_nav.clear(_store);
        if (context.mounted) {
          context.pop();
        }
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
    return Row(
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
    );
  }

  Widget _buildStatsSheet(
    BuildContext context,
    AppLocalizations localizations,
    NavigationStats stats,
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
            child: Column(
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
                        localizations.time,
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
            ),
          ),
        );
      },
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
  Widget _buildElevationPage(
    BuildContext context,
    AsyncValue<Trail> trailAsync,
  ) {
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

  Widget _buildButtonRow(
    BuildContext context,
    AppLocalizations localizations,
    NavigationStats stats,
  ) {
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
          ), // Center — dominant Pause/Resume, icon-only FAB.
          FloatingActionButton(
            heroTag: 'nav_pause',
            tooltip: stats.isPaused
                ? localizations.resume
                : localizations.pause,
            elevation: 2,
            shape: StadiumBorder(),
            onPressed: () {
              ref
                  .read(
                    navigationStatsProvider(
                      widget.response,
                      resume: _resumeStats,
                    ).notifier,
                  )
                  .togglePause();
              _persistNow();
            },
            child: FaIcon(
              stats.isPaused ? FontAwesomeIcons.play : FontAwesomeIcons.pause,
            ),
          ),

          // Right — toggle between additional stats and elevation profile.
          FloatingActionButton.small(
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
          ),
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
