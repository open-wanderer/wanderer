import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/trail/comment_card.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/trail/comment_provider.dart';

class CommentList extends ConsumerWidget {
  final Trail trail;
  const CommentList({super.key, required this.trail});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final commentListAsync = ref.watch(commentProvider(trail.id));

    return commentListAsync.when(
      data: (comments) => Padding(
        padding: const EdgeInsets.all(12.0),
        child: Column(
          children: [
            if (comments.isEmpty) ...[
              const SizedBox(height: 16),
              Center(
                child: SvgPicture.asset(
                  "assets/svgs/empty_state_comment_${Theme.of(context).brightness.name}.svg",
                  semanticsLabel: 'wanderer comment empty state',
                  height: 120,
                ),
              ),
              const SizedBox(height: 8),
              Center(
                child: Text(
                  AppLocalizations.of(context)!.no_comments_so_far,
                  style: Theme.of(
                    context,
                  ).textTheme.titleMedium!.copyWith(color: Colors.grey),
                ),
              ),
            ] else ...[
              ...comments.map((c) => CommentCard(comment: c)),
            ],
          ],
        ),
      ),
      error: (err, stack) => WandererError(err: err, stack: stack),
      loading: () => Padding(
        padding: const EdgeInsets.symmetric(vertical: 24.0),
        child: Center(child: CircularProgressIndicator()),
      ),
    );
  }
}
