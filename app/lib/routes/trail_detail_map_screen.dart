import 'package:wanderer/components/map/map_ui_controls.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/base/trail_map.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/components/trail/waypoint_sheet.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/trail/local_trail_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/actions/launch_navigation.dart';

class TrailDetailMapScreen extends ConsumerStatefulWidget {
  final String id;

  /// The local identity of a not-yet-uploaded trail. When set, [id] is empty
  /// (a local-sentinel id is blanked at the model boundary) and the screen
  /// reads its data from [localTrailProvider] instead of [trailProvider].
  final String? localId;

  const TrailDetailMapScreen({super.key, required this.id, this.localId});

  @override
  ConsumerState<TrailDetailMapScreen> createState() =>
      _TrailDetailMapScreenState();
}

class _TrailDetailMapScreenState extends ConsumerState<TrailDetailMapScreen> {
  bool showElevationProfile = true;
  bool showTrail = true;
  ml.Geographic? elevationMarkerPosition;
  Waypoint? selectedWaypoint;
  bool _isLaunching = false;

  final DraggableScrollableController _sheetController =
      DraggableScrollableController();

  ml.MapController? _mapController;

  void _onWaypointSelected(Waypoint waypoint) {
    setState(() {
      selectedWaypoint = waypoint;
      showElevationProfile = false;
    });

    if (_sheetController.isAttached) {
      _sheetController.animateTo(
        0.35,
        duration: const Duration(milliseconds: 300),
        curve: Curves.easeOut,
      );
    }
  }

  @override
  void dispose() {
    _sheetController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final localId = widget.localId;
    final Trail? localTrailValue = localId != null
        ? ref.watch(localTrailProvider(localId))
        : null;
    final AsyncValue<Trail>? trailAsync = localId == null
        ? ref.watch(trailProvider(widget.id))
        : null;

    return Scaffold(
      appBar: AppBar(
        title: (localTrailValue?.name ?? trailAsync?.value?.name) != null
            ? Text(
                localTrailValue?.name ?? trailAsync!.value!.name,
                style: Theme.of(context).textTheme.titleMedium,
              )
            : null,
      ),
      body: SafeArea(
        child: localId != null
            ? (localTrailValue != null
                  ? _buildMap(context, localTrailValue)
                  : Center(
                      child: Text(
                        AppLocalizations.of(context)!.trail_not_on_this_device,
                      ),
                    ))
            : trailAsync!.when(
                data: (trail) => _buildMap(context, trail),
                loading: () => const Center(child: CircularProgressIndicator()),
                error: (err, stack) => WandererError(err: err, stack: stack),
              ),
      ),
    );
  }

  Widget _buildMap(BuildContext context, Trail trail) {
    final user = ref.watch(authProvider).requireValue;

    return Stack(
      children: [
        Positioned.fill(
          child: TrailMap(
            trail: trail,
            showTrail: showTrail,
            elevationMarkerPosition: elevationMarkerPosition,
            onWaypointTap: _onWaypointSelected,
            selectedWaypoint: selectedWaypoint,
            showLocation: true,
            // Connectivity, NOT trail.isOffline — see that field's doc
            // comment. Online we always prefer network tiles, even for
            // a downloaded trail.
            offline: !ref.watch(onlineStatusProvider),
            initialCameraFitPadding: EdgeInsets.only(
              bottom: 300,
              left: 40,
              right: 40,
              top: 40,
            ),
            controls: [
              _buildMapControls(context, trail),
              const WandererMapCompass(hideIfRotatedNorth: true),
            ],
            onMapCreated: (controller) => _mapController = controller,
          ),
        ),
        // Floating full-width Navigate button — floats above elevation
        // profile when it is visible, or at the very bottom otherwise.
        // Hidden for an unsynced trail: launchNavigation pushes
        // '/trail/${trail.id}/navigate', which for an unsynced trail is
        // '/trail//navigate' -- the same canonicalisation failure that made
        // the detail screen unreachable -- and its Valhalla request plus
        // readCachedNav/_recacheNav are all keyed on the server id.
        if (!isUnsyncedState(trail.syncState))
          Positioned(
            left: 16,
            right: 16,
            bottom:
                trail.expand?.gpx != null &&
                    (showElevationProfile || selectedWaypoint != null)
                ? showElevationProfile
                      ? 258
                      : 286
                : 16,
            child: SizedBox(
              width: double.infinity,
              child: ElevatedButton.icon(
                onPressed: _isLaunching
                    ? null
                    : () async {
                        setState(() => _isLaunching = true);
                        await launchNavigation(
                          context: context,
                          ref: ref,
                          trail: trail,
                        );
                        if (mounted) setState(() => _isLaunching = false);
                      },
                icon: _isLaunching
                    ? const SizedBox(
                        width: 18,
                        height: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : const FaIcon(FontAwesomeIcons.locationArrow),
                label: Text(AppLocalizations.of(context)!.navigate),
              ),
            ),
          ),

        if (trail.expand?.gpx != null && showElevationProfile)
          Align(
            alignment: Alignment.bottomCenter,
            child: Container(
              height: 250,
              decoration: BoxDecoration(
                color: Theme.of(context).canvasColor,
                borderRadius: const BorderRadius.vertical(
                  top: Radius.circular(16),
                ),
                boxShadow: [
                  BoxShadow(
                    color: Colors.black.withValues(alpha: 0.15),
                    blurRadius: 10,
                    offset: const Offset(0, -2),
                  ),
                ],
              ),
              padding: const EdgeInsets.all(8),
              child: Column(
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Text(
                        AppLocalizations.of(context)!.elevation_profile,
                        style: Theme.of(context).textTheme.titleMedium!
                            .copyWith(fontWeight: FontWeight.bold),
                      ),
                      IconButton(
                        onPressed: () =>
                            setState(() => showElevationProfile = false),
                        icon: const FaIcon(FontAwesomeIcons.xmark, size: 16),
                        style: IconButton.styleFrom(
                          backgroundColor: Theme.of(
                            context,
                          ).colorScheme.surfaceContainer,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  ElevationProfile(
                    trail: trail,
                    gpx: trail.expand!.gpx!,
                    onLineTouch: (p) {
                      setState(() => elevationMarkerPosition = p?.lonlat);
                    },
                  ),
                ],
              ),
            ),
          ),

        if (selectedWaypoint != null)
          WaypointSheet(
            waypoint: selectedWaypoint!,
            user: user,
            controller: _sheetController,
            onClose: () => setState(() => selectedWaypoint = null),
          ),
      ],
    );
  }

  Widget _buildMapControls(BuildContext context, Trail trail) {
    final bounds = trail.bounds;

    return Padding(
      padding: const EdgeInsets.all(8.0),
      child: Container(
        decoration: BoxDecoration(
          color: Theme.of(context).canvasColor,
          borderRadius: BorderRadius.all(Radius.circular(24)),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            IconButton(
              icon: FaIcon(
                showTrail ? FontAwesomeIcons.eye : FontAwesomeIcons.eyeSlash,
                size: 18,
              ),
              visualDensity: VisualDensity.compact,
              onPressed: () {
                setState(() {
                  showTrail = !showTrail;
                });
              },
            ),
            IconButton(
              icon: FaIcon(FontAwesomeIcons.expand, size: 18),
              visualDensity: VisualDensity.compact,
              onPressed: () {
                final padding =
                    !showElevationProfile && selectedWaypoint == null
                    ? EdgeInsets.all(40)
                    : EdgeInsets.only(
                        bottom: 300,
                        left: 40,
                        right: 40,
                        top: 40,
                      );
                _mapController?.fitBounds(bounds: bounds, padding: padding);
              },
            ),
            IconButton(
              icon: FaIcon(FontAwesomeIcons.mountain, size: 18),
              visualDensity: VisualDensity.compact,
              onPressed: () {
                setState(() {
                  showElevationProfile = !showElevationProfile;
                  selectedWaypoint = null;
                });
              },
            ),
          ],
        ),
      ),
    );
  }
}
