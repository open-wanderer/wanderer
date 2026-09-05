import 'package:cached_network_image/cached_network_image.dart';
import 'package:duration/duration.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/base/actor_avatar.dart';
import 'package:wanderer/components/trail/stat_chip.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/list_summary.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/util/format.dart';

class ListCard extends ConsumerWidget {
  final ListSummary list;
  final bool mini;
  final VoidCallback? onListSelect;

  const ListCard({
    super.key,
    required this.list,
    this.mini = false,
    this.onListSelect,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(authProvider).value;
    if (user == null) return const SizedBox.shrink();
    final unit = ref.watch(unitProvider);

    final avatarUrl = list.getFileUrl(
      user.serverUrl,
      list.avatar,
      thumb: "600x0",
    );

    return Container(
      margin: const EdgeInsets.symmetric(vertical: 8),
      width: mini ? 160 : double.infinity,
      child: Material(
        color: Theme.of(context).cardColor,
        borderRadius: BorderRadius.circular(20),
        child: Ink(
          decoration: BoxDecoration(
            color: Theme.of(context).cardColor,
            borderRadius: BorderRadius.circular(20),
            border: Border.all(
              color: Theme.of(
                context,
              ).colorScheme.outlineVariant.withValues(alpha: 0.1),
              width: 1,
            ),
          ),
          child: InkWell(
            onTap: onListSelect,
            borderRadius: BorderRadius.circular(20),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Stack(
                  children: [
                    ClipRRect(
                      borderRadius: const BorderRadius.vertical(
                        top: Radius.circular(20),
                      ),
                      child: AspectRatio(
                        aspectRatio: 16 / 9,
                        child: list.avatar?.isNotEmpty == true
                            ? Image(
                                image: CachedNetworkImageProvider(avatarUrl!),
                                fit: BoxFit.cover,
                                errorBuilder: (_, _, _) =>
                                    _placeholder(context),
                              )
                            : _placeholder(context),
                      ),
                    ),
                    if (list.public)
                      Positioned(
                        top: 12,
                        right: 12,
                        child: _BadgeWrapper(
                          child: FaIcon(FontAwesomeIcons.globe, size: 16),
                        ),
                      ),
                  ],
                ),
                Padding(
                  padding: const EdgeInsets.all(12),
                  child: mini
                      ? _MiniContent(list: list)
                      : _FullContent(list: list, unit: unit),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _placeholder(BuildContext context) {
    return SvgPicture.asset(
      "assets/svgs/empty_state_trail_${Theme.of(context).brightness.name}.svg",
      semanticsLabel: 'wanderer logo',
      height: 80,
    );
  }
}

class _MiniContent extends StatelessWidget {
  final ListSummary list;
  const _MiniContent({required this.list});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          list.name,
          style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 14),
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        const SizedBox(height: 2),
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
    );
  }
}

class _FullContent extends StatelessWidget {
  final ListSummary list;
  final String unit;
  const _FullContent({required this.list, required this.unit});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          list.name,
          style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 17),
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
        ),
        const SizedBox(height: 6),
        Text(
          '${list.trailCount} ${AppLocalizations.of(context)!.trail(list.trailCount)}',
          style: TextStyle(
            color: Theme.of(
              context,
            ).colorScheme.onSurface.withValues(alpha: 0.6),
            fontSize: 12,
          ),
        ),
        const SizedBox(height: 6),
        Row(
          children: [
            Text(
              AppLocalizations.of(context)?.by ?? "by",
              style: TextStyle(
                color: Theme.of(
                  context,
                ).colorScheme.onSurface.withValues(alpha: 0.6),
                fontSize: 12,
              ),
            ),
            const SizedBox(width: 6),
            ActorAvatar(
              actorId: list.summaryAuthorActorId,
              imageUrl: list.summaryAuthorAvatar,
              nameSeed: list.summaryAuthorName,
              radius: 12,
            ),
            const SizedBox(width: 6),
            Text(
              list.summaryAuthorName,
              style: TextStyle(
                color: Theme.of(
                  context,
                ).colorScheme.onSurface.withValues(alpha: 0.6),
                fontSize: 12,
              ),
            ),
          ],
        ),
        const SizedBox(height: 10),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            StatChip(
              icon: FontAwesomeIcons.ruler,
              label: formatDistance(list.distance, unit: unit),
            ),
            if (list.duration != null)
              StatChip(
                icon: FontAwesomeIcons.clock,
                label: Duration(
                  seconds: list.duration!.toInt(),
                ).pretty(abbreviated: true, tersity: DurationTersity.minute),
              ),
            StatChip(
              icon: FontAwesomeIcons.arrowTrendUp,
              label: formatElevation(list.elevationGain, unit: unit),
            ),
            StatChip(
              icon: FontAwesomeIcons.arrowTrendDown,
              label: formatElevation(list.elevationLoss, unit: unit),
            ),
          ],
        ),
        if (list.description?.isNotEmpty == true) ...[
          const SizedBox(height: 8),
          Text(
            list.description!.replaceAll(RegExp(r'<[^>]*>'), '').trim(),
            style: TextStyle(
              color: Theme.of(
                context,
              ).colorScheme.onSurface.withValues(alpha: 0.7),
              fontSize: 13,
            ),
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
          ),
        ],
      ],
    );
  }
}

class _BadgeWrapper extends StatelessWidget {
  final Widget child;
  const _BadgeWrapper({required this.child});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(6),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainer,
        shape: BoxShape.circle,
      ),
      child: child,
    );
  }
}
