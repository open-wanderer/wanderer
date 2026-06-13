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

  // Stats sheet controllers: PageView (button-driven page switch — stats vs
  // elevation profile) and the DraggableScrollableSheet drag controller.
  final PageController _pageController = PageController();
  final DraggableScrollableController _sheetController =
      DraggableScrollableController();

  bool _followEnabled = true;
  bool _headingUp = false;

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
    _pageController.dispose();
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
                    bottom: 24,
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
      crossAxisAlignment: CrossAxisAlignment.start,
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

  /// Bottom stats sheet: a [DraggableScrollableSheet] covering only the bottom
  /// band (Pitfall 4 — the map and its bottom-right control column remain
  /// reachable). Snaps between a collapsed (0.18) and expanded (0.45) extent
  /// (A2 — tune if controls get covered). The scrollable child is a [ListView]
  /// so the vertical drag-to-expand gesture is forwarded (Pitfall 1).
  Widget _buildStatsSheet(
    BuildContext context,
    AppLocalizations localizations,
    NavigationStats stats,
    AsyncValue<dynamic> trailAsync,
  ) {
    final theme = Theme.of(context);

    return DraggableScrollableSheet(
      controller: _sheetController,
      initialChildSize: 0.18,
      minChildSize: 0.18,
      maxChildSize: 0.45,
      snap: true,
      snapSizes: const [0.18, 0.45],
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
          child: ListView(
            controller: scrollController,
            physics: const ClampingScrollPhysics(),
            padding: EdgeInsets.zero,
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

              // Page area: page 0 = live stats, page 1 = elevation profile.
              // Button-driven only (NeverScrollableScrollPhysics — no swipe).
              SizedBox(
                height: 220,
                child: PageView(
                  controller: _pageController,
                  physics: const NeverScrollableScrollPhysics(),
                  children: [
                    _buildStatsPage(context, localizations, stats),
                    _buildElevationPage(context, trailAsync),
                  ],
                ),
              ),

              // Button row: [Elevation profile | Pause/Resume | Exit].
              _buildButtonRow(context, localizations, stats),
            ],
          ),
        );
      },
    );
  }

  /// Page 0: live stats. Compact row (collapsed view) shows Time / Distance /
  /// Elevation gain; the expanded row (revealed as the sheet grows) adds
  /// Elevation loss / Current speed / Average speed.
  Widget _buildStatsPage(
    BuildContext context,
    AppLocalizations localizations,
    NavigationStats stats,
  ) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16.0),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
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
          const SizedBox(height: 16),
          const Divider(height: 1),
          const SizedBox(height: 16),
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
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Align(
              alignment: Alignment.topLeft,
              child: IconButton(
                onPressed: () => _pageController.animateToPage(
                  0,
                  duration: const Duration(milliseconds: 250),
                  curve: Curves.easeInOut,
                ),
                icon: const FaIcon(FontAwesomeIcons.arrowLeft, size: 18),
              ),
            ),
            Flexible(
              child: ElevationProfile(
                trail: trail,
                gpx: gpx,
                enableLineTouch: false,
              ),
            ),
          ],
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
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          // Left — switch to elevation profile page.
          IconButton(
            tooltip: localizations.elevation_profile,
            onPressed: () => _pageController.animateToPage(
              1,
              duration: const Duration(milliseconds: 250),
              curve: Curves.easeInOut,
            ),
            icon: const FaIcon(FontAwesomeIcons.chartArea),
          ),

          // Center — dominant Pause/Resume bound to the stats provider.
          FilledButton.icon(
            onPressed: () => ref
                .read(navigationStatsProvider(widget.response).notifier)
                .togglePause(),
            icon: FaIcon(
              stats.isPaused ? FontAwesomeIcons.play : FontAwesomeIcons.pause,
              size: 18,
            ),
            label: Text(
              stats.isPaused ? localizations.resume : localizations.pause,
            ),
          ),

          // Right — Exit pops the route (replaces the old top-left X, NAV-07).
          IconButton(
            tooltip: localizations.exit_navigation,
            onPressed: () => context.pop(),
            icon: const FaIcon(FontAwesomeIcons.xmark),
          ),
        ],
      ),
    );
  }
}
