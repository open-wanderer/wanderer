import 'package:cached_network_image/cached_network_image.dart';
import 'package:duration/duration.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/trail/stat_chip.dart';
import 'package:wanderer/models/list_summary.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/util/format.dart';

class ListListItem extends ConsumerWidget {
  final ListSummary list;
  final VoidCallback? onListSelect;

  const ListListItem({super.key, required this.list, this.onListSelect});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(authProvider).value;
    if (user == null) return const SizedBox.shrink();
    final unit = ref.watch(unitProvider);

    final String? avatarUrl = list.getFileUrl(
      user.serverUrl,
      list.avatar,
      thumb: "600x0",
    );

    final ImageProvider? imageProvider = avatarUrl != null
        ? CachedNetworkImageProvider(avatarUrl)
        : null;

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
              color: Theme.of(context).colorScheme.outline,
              width: 1,
            ),
          ),
          child: InkWell(
            onTap: onListSelect,
            borderRadius: BorderRadius.circular(16),
            child: Padding(
              padding: const EdgeInsets.all(10),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // ── Thumbnail ──────────────────────────────────────────
                  _Thumbnail(imageProvider: imageProvider),

                  const SizedBox(width: 12),

                  // ── Content ────────────────────────────────────────────
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        // Title
                        Text(
                          list.name,
                          style: const TextStyle(
                            fontWeight: FontWeight.bold,
                            fontSize: 15,
                          ),
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),

                        const SizedBox(height: 4),

                        // Trail count
                        Row(
                          children: [
                            FaIcon(
                              FontAwesomeIcons.route,
                              size: 12,
                              color: Theme.of(
                                context,
                              ).colorScheme.onSurface.withValues(alpha: 0.6),
                            ),
                            const SizedBox(width: 4),
                            Text(
                              '${list.trailCount} trails',
                              style: TextStyle(
                                color: Theme.of(
                                  context,
                                ).colorScheme.onSurface.withValues(alpha: 0.6),
                                fontSize: 12,
                              ),
                            ),
                          ],
                        ),

                        const SizedBox(height: 8),

                        // Stats row
                        _StatsRow(list: list, unit: unit),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

// ── Private helpers ────────────────────────────────────────────────────────────

class _Thumbnail extends StatelessWidget {
  final ImageProvider? imageProvider;

  const _Thumbnail({required this.imageProvider});

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(12),
      child: SizedBox(
        width: 90,
        height: 90,
        child: imageProvider != null
            ? Image(
                image: imageProvider!,
                fit: BoxFit.cover,
                errorBuilder: (context, _, _) => _placeholder(context),
              )
            : _placeholder(context),
      ),
    );
  }

  Widget _placeholder(BuildContext context) {
    return Container(
      color: Theme.of(context).colorScheme.secondaryContainer,
      child: Center(
        child: SvgPicture.asset(
          "assets/svgs/empty_state_trail_${Theme.of(context).brightness.name}.svg",
          semanticsLabel: 'wanderer logo',
          height: 40,
        ),
      ),
    );
  }
}

class _StatsRow extends StatelessWidget {
  final ListSummary list;
  final String unit;
  const _StatsRow({required this.list, required this.unit});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 6,
      runSpacing: 6,
      children: [
        if (list.distance != null)
          StatChip(
            icon: FontAwesomeIcons.ruler,
            label: formatDistance(list.distance!, unit: unit),
          ),
        if (list.duration != null)
          StatChip(
            icon: FontAwesomeIcons.clock,
            label: Duration(
              seconds: list.duration!.toInt(),
            ).pretty(abbreviated: true, tersity: DurationTersity.minute),
          ),
        if (list.elevationGain != null)
          StatChip(
            icon: FontAwesomeIcons.arrowTrendUp,
            label: formatElevation(list.elevationGain!, unit: unit),
          ),
        if (list.elevationLoss != null)
          StatChip(
            icon: FontAwesomeIcons.arrowTrendDown,
            label: formatElevation(list.elevationLoss!, unit: unit),
          ),
      ],
    );
  }
}
