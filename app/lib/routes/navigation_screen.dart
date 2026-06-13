import 'dart:async';

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
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/provider/map_style_provider.dart';
import 'package:wanderer/provider/navigation_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/util/format_util.dart';

class NavigationScreen extends ConsumerStatefulWidget {
  final String id;
  final NavigateResponse response;

  const NavigationScreen({
    super.key,
    required this.id,
    required this.response,
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

  bool _followEnabled = true;
  bool _headingUp = false;

  late final AnimationController _recenterButtonController = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 500),
    reverseDuration: const Duration(milliseconds: 200),
  );
  late final Animation<double> _recenterButtonScale = CurvedAnimation(
    parent: _recenterButtonController,
    curve: Curves.elasticOut,
    reverseCurve: Curves.easeOut,
  );

  @override
  void initState() {
    super.initState();

    // ONE broadcast geolocator stream shared by CurrentLocationLayer + notifier
    // (D-13: no second stream allowed — battery + position divergence risk)
    _positionStream = Geolocator.getPositionStream().asBroadcastStream();

    // Subscribe the navigation notifier to raw Position (lat/lon for D-12 + D-18)
    _sub = _positionStream.listen(
      (pos) {
        ref
            .read(navigationProvider(widget.response).notifier)
            .onPosition(LatLng(pos.latitude, pos.longitude));
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
    _recenterButtonController.dispose();
    _animatedMapController.dispose();
    super.dispose();
  }

  void _onPanStart() {
    // Only a drag (pan) disables follow — pinch-zoom must NOT disable follow
    // (D-09 free-pan; D-10 user-adjustable zoom)
    if (_followEnabled) {
      setState(() => _followEnabled = false);
      _recenterButtonController.forward();
    }
  }

  void _onRecenter() {
    setState(() => _followEnabled = true);
    _recenterButtonController.reverse();
    // One-shot realign event to CurrentLocationLayer.alignPositionStream
    _recenterTrigger.add(null);
  }

  @override
  Widget build(BuildContext context) {
    final styleAsync = ref.watch(mapStyleProvider);
    final navState = ref.watch(navigationProvider(widget.response));
    final trailAsync = ref.watch(trailProvider(widget.id));
    final localizations = AppLocalizations.of(context)!;

    final maneuvers = widget.response.maneuvers;
    final currentIndex = navState.currentManeuverIndex;
    final isArrived = currentIndex >= maneuvers.length - 1 &&
        maneuvers.isNotEmpty;

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
                        event.source == MapEventSource.onDrag) {
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
                    ),
                  ),

                  // (2) Trail polyline (planned route — blue #3549BB, 5px)
                  trailAsync.when(
                    data: (trail) {
                      if (trail.expand?.gpx != null) {
                        return TrailLayer(
                          trail: trail,
                          showWaypoints: false,
                        );
                      }
                      return const SizedBox.shrink();
                    },
                    loading: () => const SizedBox.shrink(),
                    error: (err, st) => const SizedBox.shrink(),
                  ),

                  // (3) Breadcrumb polyline (actual traveled path — crimson
                  // #DC2626, 3.5px, thinner than trail so trail stays dominant
                  // D-18, D-20, T-02-05)
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
                  // Camera follow and heading-up are purely configuration:
                  // alignPositionOnUpdate controls follow; alignDirectionOnUpdate
                  // controls heading-up rotation — no manual animateTo per frame
                  // (avoids camera fight — Pitfall 2)
                  CurrentLocationLayer(
                    positionStream: const LocationMarkerDataStreamFactory()
                        .fromGeolocatorPositionStream(
                          stream: _positionStream,
                        ),
                    alignPositionStream: _recenterTrigger.stream,
                    alignPositionOnUpdate: _followEnabled
                        ? AlignOnUpdate.always
                        : AlignOnUpdate.never,
                    alignDirectionOnUpdate: _headingUp
                        ? AlignOnUpdate.always
                        : AlignOnUpdate.never,
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

              // Exit button — top-left, 16px inset (NAV-07)
              SafeArea(
                child: Align(
                  alignment: Alignment.topLeft,
                  child: Padding(
                    padding: const EdgeInsets.all(16.0),
                    child: _buildExitButton(context),
                  ),
                ),
              ),

              // Compass toggle — top-right, 16px inset (NAV-05, D-11)
              SafeArea(
                child: Align(
                  alignment: Alignment.topRight,
                  child: Padding(
                    padding: const EdgeInsets.all(16.0),
                    child: MapCompass(
                      hideIfRotatedNorth: false,
                      onPressed: () {
                        setState(() => _headingUp = !_headingUp);
                        // When switching back to north-up, animate map to 0°
                        if (!_headingUp) {
                          _animatedMapController.animateTo(rotation: 0);
                        }
                      },
                    ),
                  ),
                ),
              ),

              // Recenter button — bottom-center, visible only when follow paused
              // (D-09, reuse ScaleTransition pattern from map_screen.dart:490-513)
              Positioned(
                bottom: 24,
                left: 0,
                right: 0,
                child: Center(
                  child: ScaleTransition(
                    scale: _recenterButtonScale,
                    child: FilledButton.icon(
                      onPressed: _onRecenter,
                      icon: const FaIcon(
                        FontAwesomeIcons.locationCrosshairs,
                      ),
                      label: const SizedBox.shrink(),
                    ),
                  ),
                ),
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

  /// Exit button (top-left): circular IconButton with canvasColor bg, xmark
  /// icon, 48px minimum touch target. Pops route with no confirmation (NAV-07).
  Widget _buildExitButton(BuildContext context) {
    return Material(
      color: Theme.of(context).canvasColor,
      shape: const CircleBorder(),
      elevation: 4,
      child: InkWell(
        customBorder: const CircleBorder(),
        onTap: () => context.pop(),
        child: Padding(
          padding: const EdgeInsets.all(12.0),
          child: FaIcon(
            FontAwesomeIcons.xmark,
            color: Theme.of(context).colorScheme.onSurface,
            size: 20,
          ),
        ),
      ),
    );
  }
}
