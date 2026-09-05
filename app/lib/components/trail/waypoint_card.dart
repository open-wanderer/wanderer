import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/trail/photo_collage.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/auth_provider.dart';

class WaypointCard extends ConsumerWidget {
  final Waypoint waypoint;

  /// When set, an edit action is shown; tapping it invokes this callback.
  final VoidCallback? onEdit;

  /// When set, a delete action is shown; tapping it invokes this callback.
  final VoidCallback? onDelete;

  const WaypointCard({
    super.key,
    required this.waypoint,
    this.onEdit,
    this.onDelete,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final hasPhotos =
        waypoint.localPhotos.isNotEmpty || waypoint.photos.isNotEmpty;

    final user = ref.watch(authProvider).requireValue!;

    // A waypoint whose photos have been downloaded for offline use (library)
    // carries BOTH `photos` (server filenames) and `localPhotos` (downloaded
    // file paths) at the same time -- unlike `Trail`, which leaves `photos`
    // empty for any DB-backed row. When local copies exist, prefer them
    // exclusively: building network URLs alongside them would duplicate every
    // photo in PhotoCollage (network copy first, local copy second) and,
    // offline, the leading network copies fail to load while the trailing
    // local ones succeed -- showing placeholders for photos that are in fact
    // available on-device.
    final webPhotos = waypoint.localPhotos.isNotEmpty
        ? const <String>[]
        : waypoint.photos
              .map(
                (p) =>
                    waypoint.getFileUrl(user.serverUrl, p, thumb: '200x0') ??
                    '',
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
                  if (waypoint.name?.isNotEmpty == true ||
                      onEdit != null ||
                      onDelete != null)
                    Row(
                      children: [
                        if (waypoint.name?.isNotEmpty == true)
                          Expanded(
                            child: Text(
                              waypoint.name!,
                              style: Theme.of(context).textTheme.titleMedium
                                  ?.copyWith(fontWeight: FontWeight.bold),
                            ),
                          )
                        else
                          const Spacer(),
                        if (onEdit != null)
                          IconButton(
                            icon: const FaIcon(FontAwesomeIcons.pen, size: 14),
                            visualDensity: VisualDensity.compact,
                            onPressed: onEdit,
                          ),
                        if (onDelete != null)
                          IconButton(
                            icon: const FaIcon(
                              FontAwesomeIcons.trash,
                              size: 14,
                              color: Colors.red,
                            ),
                            visualDensity: VisualDensity.compact,
                            onPressed: onDelete,
                          ),
                      ],
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
                        color: Theme.of(context).brightness == Brightness.light
                            ? Colors.grey[800]
                            : Colors.grey[400],
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
