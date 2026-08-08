import 'package:cached_network_image/cached_network_image.dart';
import 'package:duration/duration.dart';
import 'package:flutter/material.dart';
import 'package:flutter_html/flutter_html.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:intl/intl.dart';
import 'package:wanderer/components/base/actor_avatar.dart';
import 'package:wanderer/components/trail/photo_collage.dart';
import 'package:wanderer/components/trail/stat_chip.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/summit_log.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/routes/photo_gallery_screen.dart';
import 'package:wanderer/util/format.dart';

class SummitLogCard extends ConsumerWidget {
  final SummitLog summitLog;

  const SummitLogCard({super.key, required this.summitLog});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(authProvider).value;
    if (user == null) return const SizedBox.shrink();
    final unit = ref.watch(unitProvider);
    final author = summitLog.expand?.author;

    final bool hasStats =
        (summitLog.distance != null && summitLog.distance! > 0) ||
        (summitLog.duration != null && summitLog.duration! > 0) ||
        (summitLog.elevationGain != null && summitLog.elevationGain! > 0) ||
        (summitLog.elevationLoss != null && summitLog.elevationLoss! > 0);

    DateTime? parsedDate;
    try {
      parsedDate = DateTime.parse(summitLog.date);
    } catch (_) {}

    final String locale = Localizations.localeOf(context).toString();

    final allPhotos = summitLog.photos
        .map(
          (p) => PhotoSource.network(summitLog.getFileUrl(user.serverUrl, p)),
        )
        .toList();

    return Container(
      margin: const EdgeInsets.symmetric(vertical: 5),
      child: Material(
        color: Theme.of(context).cardColor,
        borderRadius: BorderRadius.circular(16),
        child: Ink(
          decoration: BoxDecoration(
            color: Theme.of(context).cardColor,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(
              color: Colors.grey.withValues(alpha: 0.2),
              width: 1,
            ),
          ),
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Stack(
                  children: [
                    GestureDetector(
                      onTap: () {
                        Navigator.of(context).push(
                          MaterialPageRoute(
                            builder: (_) => PhotoGalleryScreen(
                              photos: allPhotos,
                              initialIndex: 0,
                              title: '',
                            ),
                          ),
                        );
                      },
                      child: ClipRRect(
                        borderRadius: BorderRadius.circular(12),
                        child: SizedBox(
                          width: 80,
                          height: 80,
                          child: Image(
                            image: CachedNetworkImageProvider(
                              summitLog.photos.isNotEmpty
                                  ? summitLog.getFileUrl(
                                          user.serverUrl,
                                          summitLog.photos.first,
                                        ) ??
                                        ''
                                  : '',
                            ),
                            fit: BoxFit.cover,
                            errorBuilder: (_, _, _) =>
                                _buildPlaceholder(context),
                          ),
                        ),
                      ),
                    ),
                    if (summitLog.photos.length > 1)
                      Positioned(
                        bottom: 4,
                        right: 4,
                        child: Container(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 5,
                            vertical: 2,
                          ),
                          decoration: BoxDecoration(
                            color: Colors.black54,
                            borderRadius: BorderRadius.circular(6),
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              const FaIcon(
                                FontAwesomeIcons.images,
                                size: 9,
                                color: Colors.white,
                              ),
                              const SizedBox(width: 3),
                              Text(
                                '${summitLog.photos.length}',
                                style: const TextStyle(
                                  fontSize: 10,
                                  color: Colors.white,
                                  fontWeight: FontWeight.bold,
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                  ],
                ),

                const SizedBox(width: 12),

                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Expanded(
                            child: Text(
                              parsedDate != null
                                  ? dateFormatYMMMMd(locale).format(parsedDate)
                                  : summitLog.date,
                              style: const TextStyle(
                                fontWeight: FontWeight.bold,
                                fontSize: 15,
                              ),
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                          if (summitLog.gpx?.isNotEmpty == true)
                            FaIcon(
                              FontAwesomeIcons.route,
                              size: 13,
                              color: Colors.grey[500],
                            ),
                        ],
                      ),

                      const SizedBox(height: 3),
                      if (author != null) ...[
                        Row(
                          children: [
                            Text(
                              AppLocalizations.of(context)?.by ?? 'by',
                              style: TextStyle(
                                color: Colors.grey[500],
                                fontSize: 11,
                              ),
                            ),
                            const SizedBox(width: 4),
                            ActorAvatar.fromActor(actor: author, radius: 9),
                            const SizedBox(width: 4),
                            Expanded(
                              child: Text(
                                '${author.username} (${author.preferredUsername}@${author.preferredUsername})',
                                style: TextStyle(
                                  color: Colors.grey[500],
                                  fontSize: 11,
                                ),
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 4),
                      ],

                      if (summitLog.text.isNotEmpty) ...[
                        Html(data: summitLog.text),
                        const SizedBox(height: 6),
                      ],

                      if (hasStats)
                        Wrap(
                          spacing: 6,
                          runSpacing: 6,
                          children: [
                            if (summitLog.distance != null &&
                                summitLog.distance! > 0)
                              StatChip(
                                icon: FontAwesomeIcons.ruler,
                                label: formatDistance(
                                  summitLog.distance!,
                                  unit: unit,
                                ),
                              ),
                            if (summitLog.duration != null &&
                                summitLog.duration! > 0)
                              StatChip(
                                icon: FontAwesomeIcons.clock,
                                label:
                                    Duration(
                                      seconds: summitLog.duration!.toInt(),
                                    ).pretty(
                                      abbreviated: true,
                                      tersity: DurationTersity.minute,
                                    ),
                              ),
                            if (summitLog.elevationGain != null &&
                                summitLog.elevationGain! > 0)
                              StatChip(
                                icon: FontAwesomeIcons.arrowTrendUp,
                                label: formatElevation(
                                  summitLog.elevationGain!,
                                  unit: unit,
                                ),
                              ),
                            if (summitLog.elevationLoss != null &&
                                summitLog.elevationLoss! > 0)
                              StatChip(
                                icon: FontAwesomeIcons.arrowTrendDown,
                                label: formatElevation(
                                  summitLog.elevationLoss!,
                                  unit: unit,
                                ),
                              ),
                          ],
                        ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildPlaceholder(BuildContext context) {
    return SvgPicture.asset(
      "assets/svgs/empty_state_trail_${Theme.of(context).brightness.name}.svg",
      semanticsLabel: 'wanderer logo',
      height: 80,
    );
  }
}
