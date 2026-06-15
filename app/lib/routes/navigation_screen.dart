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
import 'package:wanderer/components/trail/waypoint_sheet.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/map_style_provider.dart';
import 'package:wanderer/provider/navigation_provider.dart';
import 'package:wanderer/provider/navigation_stats_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/util/format_util.dart';
import 'package:wanderer/vendor/vector_map_tiles/pm_tile_provider.dart';

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

class _NavigationScreenState extends ConsumerState<NavigationScreen>
    with TickerProviderStateMixin {
  late final _animatedMapController = AnimatedMapController(vsync: this);

  final StreamController<double?> _recenterTrigger =
      StreamController<double?>.broadcast();

  late final Stream<Position> _positionStream;
  StreamSubscription<Position>? _sub;

  final DraggableScrollableController _sheetController =
      DraggableScrollableController();
  final DraggableScrollableController _waypointSheetController =
      DraggableScrollableController();

  Waypoint? _selectedWaypoint;

  static const _kSheetMinSize = 0.2;
  static const _kSheetStatsSize = 0.3;
  static const _kSheetElevationSize = 0.45;

  MultiPmTilesVectorTileProvider? _offlineTileProvider;
  bool _offlineInitialized = false;

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

    _positionStream = Geolocator.getPositionStream().asBroadcastStream();
    _sub = _positionStream.listen(
      (pos) {
        ref
            .read(navigationProvider(widget.response).notifier)
            .onPosition(LatLng(pos.latitude, pos.longitude));
        ref
            .read(navigationStatsProvider(widget.response).notifier)
            .onPosition(pos);
        if (_followEnabled) {
          _recenterTrigger.add(null);
        }
      },
      onError: (Object error) {
        debugPrint('NavigationScreen: GPS stream error — $error');
      },
    );
  }

  Future<void> _initOffline(List<String> pmTiles) async {
    if (_offlineInitialized || pmTiles.isEmpty) return;
    _offlineInitialized = true;
    try {
      final provider = await MultiPmTilesVectorTileProvider.fromSources(
        pmTiles,
      );
      if (mounted) setState(() => _offlineTileProvider = provider);
    } catch (e) {
      debugPrint(e.toString());
    }
  }

  @override
  void dispose() {
    _sub?.cancel();
    _recenterTrigger.close();
    _sheetController.dispose();
    _waypointSheetController.dispose();
    _animatedMapController.dispose();
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
      setState(() => _followEnabled = false);
    }
  }

  void _onRecenter() {
    setState(() => _followEnabled = true);
    _recenterTrigger.add(null);
  }

  VectorTileLayer _buildTileLayer(Style style) {
    if (widget.isOffline && _offlineTileProvider != null) {
      return VectorTileLayer(
        fileCacheTtl: Duration.zero,
        theme: style.theme,
        tileProviders: TileProviders({'protomaps': _offlineTileProvider!}),
        tileOffset: TileOffset.DEFAULT,
        concurrency: kDebugMode ? 0 : VectorTileLayer.defaultConcurrency,
      );
    }
    return VectorTileLayer(
      tileProviders: style.providers,
      theme: style.theme,
      tileOffset: TileOffset.DEFAULT,
      concurrency: kDebugMode ? 0 : VectorTileLayer.defaultConcurrency,
    );
  }

  @override
  Widget build(BuildContext context) {
    final styleAsync = ref.watch(mapStyleProvider);
    final navState = ref.watch(navigationProvider(widget.response));
    final stats = ref.watch(navigationStatsProvider(widget.response));
    final trailAsync = ref.watch(trailProvider(widget.id));
    final user = ref.watch(authProvider).requireValue;
    final localizations = AppLocalizations.of(context)!;

    if (widget.isOffline && !_offlineInitialized) {
      trailAsync.whenData((trail) {
        if (trail.pmTiles.isNotEmpty) {
          WidgetsBinding.instance.addPostFrameCallback((_) {
            if (mounted) _initOffline(trail.pmTiles);
          });
        }
      });
    }

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
                    backgroundColor: Theme.of(context).colorScheme.surface,
                    initialCenter: widget.response.shapeAsLatLng.isNotEmpty
                        ? widget.response.shapeAsLatLng.first
                        : const LatLng(0, 0),
                    initialZoom: 15,
                    maxZoom: 22,
                    interactionOptions: const InteractionOptions(
                      enableMultiFingerGestureRace: true,
                    ),
                    onTap: (_, _) {
                      setState(() {
                        _selectedWaypoint = null;
                      });
                    },
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
                    SizedBox.expand(child: _buildTileLayer(style)),

                    // (2) Trail polyline + waypoints
                    trailAsync.when(
                      data: (trail) {
                        if (trail.expand?.gpx != null) {
                          return TrailLayer(
                            trail: trail,
                            showWaypoints: true,
                            selectedWaypoint: _selectedWaypoint,
                            onWaypointTap: _onWaypointSelected,
                          );
                        }
                        return const SizedBox.shrink();
                      },
                      loading: () => const SizedBox.shrink(),
                      error: (err, st) => const SizedBox.shrink(),
                    ),

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

                    CurrentLocationLayer(
                      positionStream: const LocationMarkerDataStreamFactory()
                          .fromGeolocatorPositionStream(
                            stream: _positionStream,
                          ),
                      alignPositionStream: _recenterTrigger.stream,
                      alignPositionOnUpdate: AlignOnUpdate.never,
                      alignDirectionOnUpdate: _headingUp
                          ? AlignOnUpdate.always
                          : AlignOnUpdate.never,
                    ),

                    Positioned(
                      top: 128,
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

                _buildStatsSheet(context, localizations, stats, trailAsync),

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
            );
          },
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
                  formatDistance(maneuver.length * 1000),
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

  Widget _buildStatsSheet(
    BuildContext context,
    AppLocalizations localizations,
    NavigationStats stats,
    AsyncValue<Trail> trailAsync,
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
      error: (e, _) => const SizedBox.shrink(),
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
