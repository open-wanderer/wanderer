import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/models/feed_item.dart';
import 'package:wanderer/components/trail/trail_card.dart';
import 'package:wanderer/components/profile/list_card.dart';
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
            _AuthorRow(actor: profileActor, created: created),
            TrailCard(
              trail: trail,
              fullWidth: true,
              onTrailSelect: () => context.push('/trail/${trail.id}'),
            ),
          ],
        ),
        FeedItemList(:final list, :final created) => Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _AuthorRow(actor: profileActor, created: created),
            ListCard(list: list),
          ],
        ),
      },
    );
  }
}

class _AuthorRow extends StatelessWidget {
  final Actor actor;
  final String created;

  const _AuthorRow({required this.actor, required this.created});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(top: 12, bottom: 4),
      child: Row(
        children: [
          CircleAvatar(
            radius: 16,
            backgroundColor: Colors.grey.shade300,
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
            style: TextStyle(color: Colors.grey[600], fontSize: 12),
          ),
        ],
      ),
    );
  }
}
