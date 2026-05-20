import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/trail/comment_card.dart';
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
          children: comments.map((c) => CommentCard(comment: c)).toList(),
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
