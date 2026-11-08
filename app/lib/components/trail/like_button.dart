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
    // Guarded rather than degraded: liking is meaningless without an
    // identity, and `requireValue!` throws while `logout()` holds a
    // value-less AsyncLoading — reachable with a trail open now that a
    // rejected session resolves after routing (`Auth._validateInBackground`).
    final actorId = ref.watch(authProvider).value?.actorId;
    if (actorId == null) return const SizedBox.shrink();

    final isLiked = (trail.expand?.trailLikeViaTrail ?? [])
        .where((l) => l.actor == actorId)
        .isNotEmpty;

    final count = trail.likeCount;

    return Stack(
      clipBehavior: Clip.none,
      children: [
        IconButton(
          onPressed: () {
            final notifier = ref.read(trailProvider(trail.id).notifier);
            if (isLiked) {
              notifier.unlike(actorId);
            } else {
              notifier.like(actorId);
            }
          },
          icon: FaIcon(
            isLiked ? FontAwesomeIcons.solidHeart : FontAwesomeIcons.heart,
            size: 22,
            color: isLiked ? Colors.red : Colors.blueGrey,
          ),
          style: IconButton.styleFrom(
            backgroundColor: Theme.of(context).colorScheme.surface,
          ),
        ),
        if (count > 0)
          Positioned(
            top: 2,
            right: 2,
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
              decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.surface,
                borderRadius: BorderRadius.circular(10),
              ),
              child: Text(
                count > 99 ? '99+' : '$count',
                style: TextStyle(
                  color: Theme.of(context).colorScheme.onSurface,
                  fontSize: 9,
                  fontWeight: FontWeight.bold,
                ),
              ),
            ),
          ),
      ],
    );
  }
}
