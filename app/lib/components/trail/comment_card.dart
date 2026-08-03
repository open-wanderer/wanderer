import 'package:flutter/material.dart';
import 'package:flutter_html/flutter_html.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/comment.dart';
import 'package:timeago/timeago.dart' as timeago;
import 'package:wanderer/components/base/actor_avatar.dart';

class CommentCard extends StatelessWidget {
  final Comment comment;
  const CommentCard({super.key, required this.comment});

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        ActorAvatar.fromActor(actor: comment.expand?.author, radius: 16),
        const SizedBox(width: 12),

        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text(
                  "${comment.expand!.author.username} (${comment.expand!.author.preferredUsername}@${comment.expand!.author.preferredUsername})",
                  style: Theme.of(context).textTheme.labelSmall,
                ),
                const SizedBox(width: 8),
                Text(
                  timeago.format(comment.created) +
                      (comment.created != comment.updated
                          ? " (${AppLocalizations.of(context)!.edited})"
                          : ""),
                  style: Theme.of(
                    context,
                  ).textTheme.labelSmall?.copyWith(color: Colors.grey),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Html(
              data: comment.text,
              style: {
                "body": Style(padding: HtmlPaddings.zero, margin: Margins.zero),
              },
            ),
          ],
        ),
      ],
    );
  }
}
