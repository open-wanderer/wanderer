import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/trail/photo_collage.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/auth_provider.dart';

class WaypointCard extends ConsumerWidget {
  final Waypoint waypoint;

  const WaypointCard({super.key, required this.waypoint});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final hasPhotos =
        waypoint.localPhotos.isNotEmpty || waypoint.photos.isNotEmpty;

    final user = ref.watch(authProvider).requireValue!;

    final webPhotos = waypoint.photos
        .map(
          (p) => waypoint.getFileUrl(user.serverUrl, p, thumb: '200x0') ?? '',
        )
        .toList();

    return Container(
      decoration: BoxDecoration(
        color: Theme.of(context).cardColor,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: Colors.grey.withValues(alpha: 0.2), width: 1),
      ),
      child: Card(
        clipBehavior: Clip.antiAlias,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        elevation: 0,

        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            if (hasPhotos)
              PhotoCollage(
                localPhotos: waypoint.localPhotos,
                webPhotos: webPhotos,
              ),

            Padding(
              padding: const EdgeInsets.fromLTRB(16, 14, 16, 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  if (waypoint.name?.isNotEmpty == true)
                    Text(
                      waypoint.name!,
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                    ),

                  const SizedBox(height: 6),

                  // Coordinates
                  Row(
                    children: [
                      FaIcon(
                        FontAwesomeIcons.locationDot,
                        size: 13,
                        color: Theme.of(context).colorScheme.onSurface,
                      ),
                      const SizedBox(width: 4),
                      Text(
                        '${waypoint.lat.toStringAsFixed(5)}, '
                        '${waypoint.lon.toStringAsFixed(5)}',
                        style: TextStyle(
                          fontSize: 12,
                          color: Theme.of(context).colorScheme.onSurface,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    ],
                  ),

                  // Description
                  if ((waypoint.description ?? '').isNotEmpty) ...[
                    const SizedBox(height: 10),
                    Text(
                      waypoint.description!,
                      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                        height: 1.5,
                        color: Colors.grey[800],
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
