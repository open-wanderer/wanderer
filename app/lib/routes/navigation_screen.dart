import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_map_animations/flutter_map_animations.dart';
import 'package:flutter_map_location_marker/flutter_map_location_marker.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:geolocator/geolocator.dart';
import 'package:go_router/go_router.dart';
import 'package:latlong2/latlong.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';
import 'package:wanderer/components/map/map_compass.dart';
import 'package:wanderer/components/map/trail_layer.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/provider/map_style_provider.dart';
import 'package:wanderer/provider/navigation_provider.dart';
import 'package:wanderer/provider/navigation_stats_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/util/format_util.dart';

class NavigationScreen extends ConsumerStatefulWidget {
  final String id;
  final NavigateResponse response;

  const NavigationScreen({super.key, required this.id, required this.response});

  @override
  ConsumerState<NavigationScreen> createState() => _NavigationScreenState();
}

class _NavigationScreenState extends ConsumerState<NavigationScreen>
    with TickerProviderStateMixin {
  late final _animatedMapController = AnimatedMapController(vsync: this);

  final StreamController<double?> _recenterTrigger =
      StreamController<double?>.broadcast();

  late final Stream<Position> _positionStream;
  StreamSubscription<Position>? _sub;

  final DraggableScrollableController _sheetController =
      DraggableScrollableController();

  static const _kSheetMinSize = 0.2;
  static const _kSheetStatsSize = 0.3;
  static const _kSheetElevationSize = 0.45;

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

    // ONE broadcast geolocator stream shared by CurrentLocationLayer + notifier
    // (D-13: no second stream allowed — battery + position divergence risk)
    _positionStream = Geolocator.getPositionStream().asBroadcastStream();

    // Subscribe the navigation notifier to raw Position (lat/lon for D-12 + D-18).
    // Also drive camera follow: emit to _recenterTrigger when _followEnabled so
    // CurrentLocationLayer snaps to each new GPS fix. This avoids AlignOnUpdate.always,
    // which races against setState(_followEnabled=false) and re-centers the camera
    // before the rebuild lands — making the recenter button impossible to reach.
    _sub = _positionStream.listen(
      (pos) {
        ref
            .read(navigationProvider(widget.response).notifier)
            .onPosition(LatLng(pos.latitude, pos.longitude));
        // Feed the SAME single GPS fix to the stats notifier (D-13 — no second
        // stream). The stats provider needs altitude + speed, so pass the raw
        // geolocator Position rather than a LatLng.
        ref
            .read(navigationStatsProvider(widget.response).notifier)
            .onPosition(pos);
        if (_followEnabled) {
          _recenterTrigger.add(null);
        }
      },
      onError: (Object error) {
        // PlatformException from Geolocator (e.g. permission denied mid-session)
        // is swallowed here so it does not escape to the Flutter error handler.
        // The GPS dot simply stops updating — the user can exit via the X button.
        debugPrint('NavigationScreen: GPS stream error — $error');
      },
    );
  }

  @override
  void dispose() {
    // T-02-04 mitigation: cancel subscription, close controllers, dispose map
    // controller to prevent background GPS drain and setState-after-dispose
    // (Pitfall 6 — mirror map_screen.dart:103-111)
    _sub?.cancel();
    _recenterTrigger.close();
    _sheetController.dispose();
    _animatedMapController.dispose();
    super.dispose();
  }

  void _onPanStart() {
    if (_followEnabled) {
      setState(() => _followEnabled = false);
    }
  }

  void _onRecenter() {
    setState(() => _followEnabled = true);
    _recenterTrigger.add(null);
  }

  @override
  Widget build(BuildContext context) {
    final styleAsync = ref.watch(mapStyleProvider);
    final navState = ref.watch(navigationProvider(widget.response));
    final stats = ref.watch(navigationStatsProvider(widget.response));
    final trailAsync = ref.watch(trailProvider(widget.id));
    final localizations = AppLocalizations.of(context)!;

    final maneuvers = widget.response.maneuvers;
    final currentIndex = navState.currentManeuverIndex;
    final isArrived =
        currentIndex >= maneuvers.length - 1 && maneuvers.isNotEmpty;

    return Scaffold(
      body: styleAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text(e.toString())),
        data: (style) {
          return Stack(
            children: [
              // ----------------------------------------------------------------
              // Full-screen map
              // ----------------------------------------------------------------
              FlutterMap(
                key: ObjectKey(style),
                mapController: _animatedMapController.mapController,
                options: MapOptions(
                  initialCenter: widget.response.shapeAsLatLng.isNotEmpty
                      ? widget.response.shapeAsLatLng.first
                      : const LatLng(0, 0),
                  // D-10: sensible hiking zoom ~15–16; do NOT lock with minZoom
                  // so pinch-zoom stays user-adjustable
                  initialZoom: 15,
                  maxZoom: 22,
                  interactionOptions: const InteractionOptions(
                    enableMultiFingerGestureRace: true,
                  ),
                  onMapEvent: (event) {
                    // Only drag events disable follow — pinch-zoom events must
                    // NOT pause follow (D-09 free-pan; D-10 zoom must stay free)
                    if (event is MapEventMoveStart &&
                        event.source == MapEventSource.dragStart) {
                      _onPanStart();
                    }
                  },
                ),
                children: [
                  // (1) Vector tile layer (map background)
                  SizedBox.expand(
                    child: VectorTileLayer(
                      tileProviders: style.providers,
                      theme: style.theme,
                      tileOffset: TileOffset.DEFAULT,
                      concurrency: kDebugMode
                          ? 0
                          : VectorTileLayer.defaultConcurrency,
                    ),
                  ),

                  // (2) Trail polyline (planned route — blue #3549BB, 5px)
                  trailAsync.when(
                    data: (trail) {
                      if (trail.expand?.gpx != null) {
                        return TrailLayer(trail: trail, showWaypoints: false);
                      }
                      return const SizedBox.shrink();
                    },
                    loading: () => const SizedBox.shrink(),
                    error: (err, st) => const SizedBox.shrink(),
                  ),

                  // (3) Breadcrumb polyline (actual traveled path — crimson
                  // #DC2626, 3.5px, thinner than trail so trail stays dominant
                  // D-18, D-20, T-02-05)
                  // Guard: PolylineLayer crashes on empty points list (flutter_map
                  // calls LatLngBounds.fromPoints which asserts isNotEmpty).
                  if (navState.breadcrumb.isNotEmpty)
                    PolylineLayer(
                      polylines: [
                        Polyline(
                          points: navState.breadcrumb,
                          color: const Color(0xFFDC2626),
                          strokeWidth: 3.5,
                        ),
                      ],
                    ),

                  // (4) GPS dot — position stream shared with notifier (D-13)
                  // alignPositionOnUpdate is always Never; follow is driven
                  // manually by emitting to _recenterTrigger from the GPS
                  // listener when _followEnabled=true. This avoids the race
                  // where AlignOnUpdate.always fires a mapController move
                  // before the setState(_followEnabled=false) rebuild lands.
                  CurrentLocationLayer(
                    positionStream: const LocationMarkerDataStreamFactory()
                        .fromGeolocatorPositionStream(stream: _positionStream),
                    alignPositionStream: _recenterTrigger.stream,
                    alignPositionOnUpdate: AlignOnUpdate.never,
                    alignDirectionOnUpdate: _headingUp
                        ? AlignOnUpdate.always
                        : AlignOnUpdate.never,
                  ),

                  // (5) Map controls column — bottom-right.
                  // Must live inside FlutterMap.children so MapCamera.of()
                  // resolves for MapCompass. Recenter is disabled (not hidden)
                  // when the camera is already following the user.
                  Positioned(
                    top: 96,
                    right: 8,
                    child: SafeArea(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          MapCompass(
                            hideIfRotatedNorth: false,
                            onPressed: () {
                              setState(() => _headingUp = !_headingUp);
                              if (!_headingUp) {
                                _animatedMapController.animateTo(rotation: 0);
                              }
                            },
                          ),
                          const SizedBox(height: 4),
                          IconButton(
                            onPressed: _followEnabled ? null : _onRecenter,
                            icon: const FaIcon(
                              FontAwesomeIcons.locationCrosshairs,
                            ),
                            style: IconButton.styleFrom(
                              backgroundColor: Theme.of(
                                context,
                              ).colorScheme.surface,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ],
              ),

              // ----------------------------------------------------------------
              // Overlay chrome (above map)
              // ----------------------------------------------------------------

              // Maneuver / completion banner — top, SafeArea, 24px horizontal
              SafeArea(
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(24, 8, 24, 0),
                  child: _buildBanner(
                    context,
                    localizations,
                    maneuvers,
                    currentIndex,
                    isArrived,
                  ),
                ),
              ),

              // ----------------------------------------------------------------
              // Stats sheet — bottom band only (Pitfall 4: must NOT cover the
              // whole screen, so the map + bottom-right controls stay reachable).
              // ----------------------------------------------------------------
              _buildStatsSheet(context, localizations, stats, trailAsync),

              // Button row floats above the sheet as a Positioned overlay so the
              // sheet itself can be a plain SingleChildScrollView (no LayoutBuilder
              // hack needed to anchor the row at the sheet bottom).
              Positioned(
                left: 16,
                right: 16,
                bottom: MediaQuery.of(context).padding.bottom,
                child: _buildButtonRow(context, localizations, stats),
              ),
            ],
          );
        },
      ),
    );
  }

  /// Builds the maneuver banner (active nav) or completion banner (arrived).
  Widget _buildBanner(
    BuildContext context,
    AppLocalizations localizations,
    List<NavigateManeuver> maneuvers,
    int currentIndex,
    bool isArrived,
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
              ),
      ),
    );
  }

  /// Active maneuver: leading accent icon + instruction + distance sub-label.
  Widget _buildActiveBannerContent(
    BuildContext context,
    AppLocalizations localizations,
    List<NavigateManeuver> maneuvers,
    int currentIndex,
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
          child: FaIcon(
            FontAwesomeIcons.locationArrow,
            color: Theme.of(context).colorScheme.primary,
            size: 18,
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
                  formatDistance(maneuver.length * 1000),
                ),
                style: Theme.of(context).textTheme.bodyMedium,
              ),
            ],
          ),
        ),
      ],
    );
  }

  /// Completion banner: "You've arrived" heading + body text (D-14, D-15).
  Widget _buildCompletionBannerContent(
    BuildContext context,
    AppLocalizations localizations,
  ) {
    return Column(
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
    );
  }

  /// Bottom stats sheet. Standard [DraggableScrollableSheet] usage: the sheet's
  /// [scrollController] drives a [SingleChildScrollView] so drag-to-expand works
  /// out of the box. The button row lives in the outer [Stack] as a [Positioned]
  /// overlay, so no [LayoutBuilder]/[Stack] trick is needed here.
  Widget _buildStatsSheet(
    BuildContext context,
    AppLocalizations localizations,
    NavigationStats stats,
    AsyncValue<dynamic> trailAsync,
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
                        formatDistance(stats.distanceMeters),
                      ),
                      _buildStatCell(
                        context,
                        localizations.elevation_gain,
                        formatElevation(stats.elevationGainMeters),
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

  /// Additional stats that fade in as the sheet expands: elevation loss,
  /// current speed, and average speed.
  Widget _buildAdditionalStats(
    BuildContext context,
    AppLocalizations localizations,
    NavigationStats stats,
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
                formatElevation(stats.elevationLossMeters),
              ),
              _buildStatCell(
                context,
                localizations.speed,
                formatSpeed(stats.currentSpeedKmh),
              ),
              _buildStatCell(
                context,
                localizations.average_speed,
                formatSpeed(stats.averageSpeedKmh),
              ),
            ],
          ),
        ],
      ),
    );
  }

  /// A single stat cell: subdued label above, large bold value below
  /// (CONTEXT specifics: value fontSize 24, FontWeight.bold).
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
    AsyncValue<dynamic> trailAsync,
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
      error: (e, _) => const SizedBox.shrink(),
    );
  }

  /// Button row (always visible below the PageView): elevation-profile switch
  /// (left), dominant Pause/Resume (center), Exit (right). Exit consolidates
  /// the old top-left X overlay (NAV-07).
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
          // Left — Exit pops the route.
          FloatingActionButton.small(
            heroTag: 'nav_exit',
            tooltip: localizations.exit_navigation,
            elevation: 2,
            shape: StadiumBorder(),
            backgroundColor: Theme.of(context).colorScheme.surface,
            onPressed: () => context.pop(),
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
