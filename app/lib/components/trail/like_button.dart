import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';

class LikeButton extends ConsumerWidget {
  const LikeButton({super.key, required this.trail});

  final Trail trail;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final actorId = ref.watch(authProvider).requireValue!.actorId;

    final isLiked = (trail.expand?.trailLikeViaTrail ?? [])
        .where((l) => l.actor == actorId)
        .isNotEmpty;

    final count = trail.likeCount;

    return InkWell(
      borderRadius: BorderRadius.circular(8),
      onTap: () {
        final notifier = ref.read(trailProvider(trail.id).notifier);
        if (isLiked) {
          notifier.unlike(actorId);
        } else {
          notifier.like(actorId);
        }
      },
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
        decoration: BoxDecoration(
          color: Colors.grey.withValues(alpha: 0.1),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            FaIcon(
              isLiked ? FontAwesomeIcons.solidHeart : FontAwesomeIcons.heart,
              size: 16,
              color: isLiked ? Colors.red : Colors.blueGrey,
            ),
            const SizedBox(height: 4),
            Text("$count", style: Theme.of(context).textTheme.bodySmall),
          ],
        ),
      ),
    );
  }
}
