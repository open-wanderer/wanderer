import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/models/feed_item.dart';
import 'package:wanderer/components/trail/trail_list_item.dart';
import 'package:wanderer/components/list/list_list_item.dart';
import 'package:timeago/timeago.dart' as timeago;

String _formatCreated(String created) {
  try {
    return timeago.format(DateTime.parse(created));
  } catch (_) {
    return '';
  }
}

class FeedItemCard extends StatelessWidget {
  final FeedItem item;
  final Actor profileActor;

  const FeedItemCard({
    super.key,
    required this.item,
    required this.profileActor,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: switch (item) {
        FeedItemTrail(:final trail, :final created) => Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _AuthorRow(actor: profileActor, created: created, type: "trail"),
            TrailListItem(
              trail: trail,
              onTrailSelect: () => context.push('/trail/${trail.id}'),
            ),
          ],
        ),
        FeedItemList(:final list, :final created) => Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _AuthorRow(actor: profileActor, created: created, type: "list"),
            ListListItem(list: list),
          ],
        ),
      },
    );
  }
}

class _AuthorRow extends StatelessWidget {
  final Actor actor;
  final String created;
  final String type;

  const _AuthorRow({
    required this.actor,
    required this.created,
    required this.type,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(top: 12, bottom: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          CircleAvatar(
            radius: 14,
            backgroundColor: Theme.of(context).colorScheme.surfaceContainerHighest,
            backgroundImage: NetworkImage(
              actor.icon != null && actor.icon!.isNotEmpty
                  ? actor.icon!
                  : 'https://api.dicebear.com/7.x/initials/png?seed=${actor.preferredUsername}&backgroundType=gradientLinear',
            ),
            onBackgroundImageError: (_, _) =>
                const FaIcon(FontAwesomeIcons.user),
          ),
          const SizedBox(width: 8),
          Text(
            '@${actor.preferredUsername}',
            style: const TextStyle(fontWeight: FontWeight.w600),
          ),
          const SizedBox(width: 8),
          Text(
            _formatCreated(created),
            style: TextStyle(color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.6), fontSize: 12),
          ),
          Spacer(),
          if (type == "trail") ...[
            FaIcon(FontAwesomeIcons.route, size: 14, color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.5)),
            SizedBox(width: 4),
            Text(
              AppLocalizations.of(context)!.trail(1),
              style: Theme.of(context).textTheme.labelSmall,
            ),
          ] else if (type == "list") ...[
            FaIcon(FontAwesomeIcons.layerGroup, size: 14, color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.5)),
            SizedBox(width: 4),
            Text(
              AppLocalizations.of(context)!.list(1),
              style: Theme.of(context).textTheme.labelSmall,
            ),
          ],
        ],
      ),
    );
  }
}
