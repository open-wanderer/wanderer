import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/trail/photo_collage.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/util/format.dart';

class WaypointSheet extends ConsumerWidget {
  final Waypoint waypoint;
  final UserEntity? user;
  final DraggableScrollableController controller;
  final VoidCallback onClose;

  const WaypointSheet({
    super.key,
    required this.waypoint,
    required this.user,
    required this.controller,
    required this.onClose,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final localizations = AppLocalizations.of(context)!;
    final unit = ref.watch(unitProvider);

    // A waypoint whose photos have been downloaded for offline use (library)
    // carries BOTH `photos` (server filenames) and `localPhotos` (downloaded
    // file paths) at the same time -- unlike `Trail`, which leaves `photos`
    // empty for any DB-backed row. When local copies exist, prefer them
    // exclusively: building network URLs alongside them would duplicate every
    // photo in PhotoCollage (network copy first, local copy second) and,
    // offline, the leading network copies fail to load while the trailing
    // local ones succeed -- showing placeholders for photos that are in fact
    // available on-device. This also sidesteps `user!` below when `user` is
    // null but local copies already make the collage worth showing.
    final currentUser = user;
    final webPhotos = waypoint.localPhotos.isNotEmpty || currentUser == null
        ? const <String>[]
        : waypoint.photos
              .map(
                (p) =>
                    waypoint.getFileUrl(
                      currentUser.serverUrl,
                      p,
                      thumb: '200x0',
                    ) ??
                    '',
              )
              .toList();

    return NotificationListener<DraggableScrollableNotification>(
      onNotification: (n) {
        if (n.extent <= n.minExtent) onClose();
        return true;
      },
      child: DraggableScrollableSheet(
        controller: controller,
        initialChildSize: 0.35,
        minChildSize: 0.0,
        maxChildSize: 0.7,
        snap: true,
        builder: (context, scrollController) {
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
              padding: EdgeInsets.zero,
              children: [
                Stack(
                  children: [
                    if (waypoint.localPhotos.isNotEmpty ||
                        waypoint.photos.isNotEmpty && user != null)
                      ClipRRect(
                        borderRadius: const BorderRadius.vertical(
                          top: Radius.circular(16),
                        ),
                        child: PhotoCollage(
                          localPhotos: waypoint.localPhotos,
                          webPhotos: webPhotos,
                        ),
                      ),
                    Padding(
                      padding: const EdgeInsets.all(8.0),
                      child: Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          const SizedBox(width: 40),
                          Container(
                            width: 30,
                            height: 5,
                            decoration: BoxDecoration(
                              borderRadius: const BorderRadius.all(
                                Radius.circular(24),
                              ),
                              color: theme.colorScheme.secondaryContainer,
                            ),
                          ),
                          IconButton(
                            onPressed: onClose,
                            icon: const FaIcon(
                              FontAwesomeIcons.xmark,
                              size: 14,
                            ),
                            visualDensity: VisualDensity.compact,
                            style: IconButton.styleFrom(
                              backgroundColor:
                                  theme.colorScheme.secondaryContainer,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 16.0),
                  child: Row(
                    children: [
                      FaIcon(
                        waypoint.icon,
                        color: theme.colorScheme.onSurface,
                        size: 22,
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          waypoint.name?.isNotEmpty == true
                              ? waypoint.name!
                              : localizations.waypoints(1),
                          style: theme.textTheme.titleLarge?.copyWith(
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 8),
                if (waypoint.distanceFromStart != null) ...[
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 16.0),
                    child: Row(
                      children: [
                        const FaIcon(
                          FontAwesomeIcons.route,
                          size: 14,
                          color: Colors.grey,
                        ),
                        const SizedBox(width: 8),
                        Text(
                          '${localizations.after} ${formatDistance(waypoint.distanceFromStart ?? 0, unit: unit)}',
                          style: theme.textTheme.bodyMedium?.copyWith(
                            color: Colors.grey[600],
                          ),
                        ),
                        const SizedBox(width: 24),
                        const FaIcon(
                          FontAwesomeIcons.locationDot,
                          size: 14,
                          color: Colors.grey,
                        ),
                        const SizedBox(width: 8),
                        Text(
                          '${waypoint.lat.toStringAsFixed(5)}, ${waypoint.lon.toStringAsFixed(5)}',
                          style: theme.textTheme.bodyMedium?.copyWith(
                            color: Colors.grey[600],
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
                const Divider(height: 24),
                if (waypoint.description?.isNotEmpty == true)
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 16.0),
                    child: Text(
                      waypoint.description!,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        height: 1.5,
                        color: Theme.of(
                          context,
                        ).colorScheme.onSurface.withValues(alpha: 0.8),
                      ),
                    ),
                  ),
              ],
            ),
          );
        },
      ),
    );
  }
}
