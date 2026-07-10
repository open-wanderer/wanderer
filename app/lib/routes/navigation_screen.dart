import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:geolocator/geolocator.dart' as geo;
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/map/trail_layer.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/components/trail/waypoint_sheet.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/glyph_sprite_cache_paths.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/glyph_sprite_cache_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/map_style_json_provider.dart';
import 'package:wanderer/provider/map_style_provider.dart'
    show effectiveBrightness;
import 'package:wanderer/provider/navigation_provider.dart';
import 'package:wanderer/provider/navigation_stats_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/util/format_util.dart';
import 'package:wanderer/util/offline_style_rewriter.dart';
import 'package:wanderer/util/tracelet_position_source.dart';

class NavigationScreen extends ConsumerStatefulWidget {
  final String id;
  final NavigateResponse response;
  final bool isOffline;

  const NavigationScreen({
    super.key,
    required this.id,
    required this.response,
    this.isOffline = false,
  });

  @override
  ConsumerState<NavigationScreen> createState() => _NavigationScreenState();
}

class _NavigationScreenState extends ConsumerState<NavigationScreen> {
  late final TraceletPositionSource _positionSource;
  late final Stream<geo.Position> _positionStream;
  StreamSubscription<geo.Position>? _sub;

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
  /// `onStyleLoaded` (same race `SearchMap` guards against, Phase 16-03).
  ml.StyleController? _pendingStyle;

  /// The last successfully-resolved (and possibly offline-rewritten) style
  /// JSON. Cached so a provider refresh (e.g. a theme toggle) never drops us
  /// back to the loading state and remounts the map — the live swap goes
  /// through [ml.MapController.setStyle] instead (CORE-02).
  String? _lastStyleJson;

  bool _cacheWarmed = false;

  /// Active pointer count on the map surface. `CameraChangeReason.apiGesture`
  /// fires identically for pan/pinch/rotate (no native sub-classification —
  /// RESEARCH Pitfall 1), so this heuristic is what narrows follow-break to a
  /// single-finger drag (NAV-02). Read synchronously inside `onEvent` only —
  /// never mutated via `setState`.
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

    _positionSource = TraceletPositionSource();
    _positionStream = _positionSource.stream;
    unawaited(_positionSource.start());
    _sub = _positionStream.listen(
      (pos) {
        ref
            .read(navigationProvider(widget.response).notifier)
            .onPosition(ml.Geographic(lat: pos.latitude, lon: pos.longitude));
        ref
            .read(navigationStatsProvider(widget.response).notifier)
            .onPosition(pos);
      },
      onError: (Object error) {
        debugPrint('NavigationScreen: GPS stream error — $error');
      },
    );
  }

  @override
  void dispose() {
    _sub?.cancel();
    unawaited(_positionSource.dispose());
    _sheetController.dispose();
    _waypointSheetController.dispose();
    super.dispose();
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
      _controller?.trackLocation(trackLocation: false);
      setState(() => _followEnabled = false);
    }
  }

  void _onRecenter() {
    setState(() => _followEnabled = true);
    _controller?.trackLocation(
      trackLocation: true,
      // D-03: restores prior heading-up state — recenter never forces north.
      trackBearing: _headingUp
          ? ml.BearingTrackMode.gps
          : ml.BearingTrackMode.none,
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
  /// state and swaps it onto the mounted controller in place (CORE-02).
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
  /// concatenation — keeps the geometry structurally isolated (T-17-01).
  String _breadcrumbGeoJson(List<ml.Geographic> pts) {
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

  /// Re-arms everything that binds to the current native `Style` object
  /// (Pattern 1, 17-RESEARCH.md): `setStyle` (CORE-02 theme swap) drops added
  /// layers/sources AND the location component, so this must run after every
  /// style load, not just once at `onMapCreated`.
  Future<void> _onStyleLoaded(ml.StyleController style) async {
    try {
      final trail = ref.read(trailProvider(widget.id)).value;
      if (trail?.expand?.gpx != null) {
        await addTrailTrackLayers(style, trail!);
      }

      final breadcrumb = ref
          .read(navigationProvider(widget.response))
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
        ),
      );

      final controller = _controller;
      if (controller != null) {
        await controller.enableLocation(
          bearingRenderMode: ml.BearingRenderMode.gps, // D-04: GPS heading
        );
        await controller.trackLocation(
          trackLocation: _followEnabled,
          trackBearing: _headingUp
              ? ml.BearingTrackMode.gps
              : ml.BearingTrackMode.none,
        );
      }
    } catch (e) {
      debugPrint('NavigationScreen: onStyleLoaded failed — $e');
    }
  }

  @override
  Widget build(BuildContext context) {
    // D-09-style cache warm on first open (mirrors WandererMap's CORE-02
    // pattern) — idempotent against the trail-download trigger.
    if (!_cacheWarmed) {
      _cacheWarmed = true;
      ref.read(glyphSpriteCacheProvider.future).ignore();
    }

    // Live style swap (CORE-02): theme toggle or (offline) glyph/sprite cache
    // warm swaps the composed style in place on the already-mounted map.
    ref.listen(mapStyleJsonProvider, (_, _) => _swapStyle());
    if (widget.isOffline) {
      ref.listen(glyphSpriteCacheProvider, (_, _) => _swapStyle());
    }

    // Breadcrumb in-place update (T-17-01 fail-soft): swap the native source's
    // data on every new position fix, never remove/re-add the source.
    ref.listen(navigationProvider(widget.response), (prev, next) {
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
    });

    final navState = ref.watch(navigationProvider(widget.response));
    final stats = ref.watch(navigationStatsProvider(widget.response));
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
                          // CameraChangeReason.apiGesture fires identically for
                          // pan/pinch/rotate (RESEARCH Pitfall 1 — no native
                          // sub-classification exists); _activePointers <= 1 is
                          // the compensating heuristic so only a single-finger
                          // drag breaks follow (NAV-02). Needs on-device
                          // verification (RESEARCH Open Question 1 — 17-03
                          // checkpoint).
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

                        const ml.MapScalebar(), // CORE-04
                        const ml.SourceAttribution(), // CORE-04
                        Positioned(
                          top: 128,
                          right: 8,
                          child: SafeArea(
                            child: Column(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                ml.MapCompass(
                                  hideIfRotatedNorth: false, // D-02
                                  rotateNorthOnPressed: false,
                                  onPressed: () {
                                    setState(() => _headingUp = !_headingUp);
                                    final controller = _controller;
                                    if (controller == null) return;
                                    controller.trackLocation(
                                      trackLocation: _followEnabled,
                                      trackBearing: _headingUp
                                          ? ml.BearingTrackMode.gps
                                          : ml.BearingTrackMode.none,
                                    );
                                    if (!_headingUp) {
                                      controller.animateCamera(
                                        bearing: 0,
                                        nativeDuration: const Duration(
                                          milliseconds: 400,
                                        ),
                                      );
                                    }
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
      if (confirmed == true && context.mounted) {
        context.pop();
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
            onPressed: () => ref
                .read(navigationStatsProvider(widget.response).notifier)
                .togglePause(),
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
