import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/base/wanderer_map.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/components/trail/photo_collage.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/util/format_util.dart';

class TrailDetailMapScreen extends ConsumerStatefulWidget {
  final String id;

  const TrailDetailMapScreen({super.key, required this.id});

  @override
  ConsumerState<TrailDetailMapScreen> createState() =>
      _TrailDetailMapScreenState();
}

class _TrailDetailMapScreenState extends ConsumerState<TrailDetailMapScreen> {
  bool showElevationProfile = true;
  Waypoint? selectedWaypoint;

  // Controller to programmatically open/reset the sheet when a waypoint changes
  final DraggableScrollableController _sheetController =
      DraggableScrollableController();

  void _onWaypointSelected(Waypoint waypoint) {
    setState(() {
      selectedWaypoint = waypoint;
      // If the profile was open, close it to avoid UI overcrowding
      showElevationProfile = false;
    });

    // Smoothly pop up the sheet to its initial size when a new waypoint is clicked
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
    final trailAsync = ref.watch(trailProvider(widget.id));
    final user = ref.watch(authProvider).requireValue!;

    return Scaffold(
      appBar: AppBar(
        title: trailAsync.value != null ? Text(trailAsync.value!.name) : null,
      ),
      floatingActionButton:
          trailAsync.value?.expand?.gpx != null && !showElevationProfile
          ? FloatingActionButton.small(
              onPressed: () {
                setState(() {
                  showElevationProfile = true;
                  selectedWaypoint = null;
                });
              },
              child: const FaIcon(FontAwesomeIcons.chartArea, size: 16),
            )
          : null,
      body: SafeArea(
        child: trailAsync.when(
          data: (trail) => Stack(
            children: [
              Positioned.fill(
                child: WandererMap(
                  trail: trail,
                  onWaypointTap: _onWaypointSelected,
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
                              icon: const FaIcon(
                                FontAwesomeIcons.xmark,
                                size: 16,
                              ),
                              style: IconButton.styleFrom(
                                backgroundColor: Colors.white,
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 16),
                        ElevationProfile(trail: trail, gpx: trail.expand!.gpx!),
                      ],
                    ),
                  ),
                ),

              if (selectedWaypoint != null)
                NotificationListener<DraggableScrollableNotification>(
                  onNotification: (notification) {
                    if (notification.extent <= notification.minExtent) {
                      setState(() => selectedWaypoint = null);
                    }
                    return true;
                  },
                  child: DraggableScrollableSheet(
                    controller: _sheetController,
                    initialChildSize: 0.35,
                    minChildSize: 0.0,
                    maxChildSize: 0.7,
                    snap: true,
                    builder: (context, scrollController) {
                      final theme = Theme.of(context);

                      return Container(
                        decoration: BoxDecoration(
                          color: theme.canvasColor,
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
                        child: ListView(
                          controller: scrollController,
                          children: [
                            Stack(
                              children: [
                                if (selectedWaypoint!.localPhotos.isNotEmpty ||
                                    selectedWaypoint!.photos.isNotEmpty)
                                  ClipRRect(
                                    borderRadius: const BorderRadius.vertical(
                                      top: Radius.circular(16),
                                    ),
                                    child: PhotoCollage(
                                      localPhotos:
                                          selectedWaypoint!.localPhotos,
                                      webPhotos: selectedWaypoint!.photos
                                          .map(
                                            (p) =>
                                                selectedWaypoint!.getFileUrl(
                                                  user.serverUrl,
                                                  p,
                                                  thumb: '200x0',
                                                ) ??
                                                '',
                                          )
                                          .toList(),
                                    ),
                                  ),
                                Padding(
                                  padding: const EdgeInsets.all(8.0),
                                  child: Row(
                                    mainAxisAlignment:
                                        MainAxisAlignment.spaceBetween,
                                    children: [
                                      const SizedBox(width: 40),
                                      Container(
                                        width: 20,
                                        height: 5,
                                        decoration: BoxDecoration(
                                          color: Colors.grey[300],
                                          borderRadius: BorderRadius.circular(
                                            10,
                                          ),
                                        ),
                                      ),
                                      IconButton(
                                        onPressed: () => setState(
                                          () => selectedWaypoint = null,
                                        ),
                                        icon: const FaIcon(
                                          FontAwesomeIcons.xmark,
                                          size: 14,
                                        ),
                                        visualDensity: VisualDensity.compact,
                                        style: IconButton.styleFrom(
                                          backgroundColor: Colors.white,
                                        ),
                                      ),
                                    ],
                                  ),
                                ),
                              ],
                            ),
                            const SizedBox(height: 8),
                            Padding(
                              padding: const EdgeInsets.symmetric(
                                horizontal: 16.0,
                              ),
                              child: Row(
                                children: [
                                  FaIcon(
                                    selectedWaypoint!.icon,
                                    color: theme.primaryColor,
                                    size: 22,
                                  ),
                                  const SizedBox(width: 12),
                                  Expanded(
                                    child: Text(
                                      selectedWaypoint!.name?.isNotEmpty == true
                                          ? selectedWaypoint!.name!
                                          : AppLocalizations.of(
                                              context,
                                            )!.waypoints(1),
                                      style: theme.textTheme.titleLarge
                                          ?.copyWith(
                                            fontWeight: FontWeight.bold,
                                          ),
                                    ),
                                  ),
                                ],
                              ),
                            ),
                            const SizedBox(height: 8),
                            if (selectedWaypoint!.distanceFromStart !=
                                null) ...[
                              Padding(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 16.0,
                                ),
                                child: Row(
                                  children: [
                                    const FaIcon(
                                      FontAwesomeIcons.route,
                                      size: 14,
                                      color: Colors.grey,
                                    ),
                                    const SizedBox(width: 8),
                                    Text(
                                      '${AppLocalizations.of(context)!.after} ${formatDistance(selectedWaypoint?.distanceFromStart ?? 0)}',
                                      style: theme.textTheme.bodyMedium
                                          ?.copyWith(color: Colors.grey[600]),
                                    ),
                                    const SizedBox(width: 24),
                                    const FaIcon(
                                      FontAwesomeIcons.locationDot,
                                      size: 14,
                                      color: Colors.grey,
                                    ),
                                    const SizedBox(width: 8),

                                    Text(
                                      '${selectedWaypoint!.lat.toStringAsFixed(5)}, ${selectedWaypoint!.lon.toStringAsFixed(5)}',
                                      style: theme.textTheme.bodyMedium
                                          ?.copyWith(color: Colors.grey[600]),
                                    ),
                                  ],
                                ),
                              ),
                            ],
                            const Divider(height: 24),

                            if (selectedWaypoint!.description?.isNotEmpty ==
                                true)
                              Padding(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 16.0,
                                ),
                                child: Text(
                                  selectedWaypoint!.description!,
                                  style: Theme.of(context).textTheme.bodyMedium
                                      ?.copyWith(
                                        height: 1.5,
                                        color: Colors.grey[800],
                                      ),
                                ),
                              ),
                          ],
                        ),
                      );
                    },
                  ),
                ),
            ],
          ),
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (err, stack) => WandererError(err: err, stack: stack),
        ),
      ),
    );
  }
}
