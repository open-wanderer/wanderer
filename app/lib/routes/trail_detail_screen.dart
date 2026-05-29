import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/trail/trail_dropdown.dart';
import 'package:wanderer/components/trail/trail_panel.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';

class TrailDetailScreen extends ConsumerWidget {
  final String id;
  const TrailDetailScreen({super.key, required this.id});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final trailAsync = ref.watch(trailProvider(id));

    return Scaffold(
      body: SafeArea(
        child: trailAsync.when(
          data: (trail) => TrailPanel(
            trail: trail,
            scrollController: ScrollController(),
            actionMenu: TrailDropdown(trail: trail),
          ),
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (err, stack) => WandererError(err: err, stack: stack),
        ),
      ),
    );
  }
}
