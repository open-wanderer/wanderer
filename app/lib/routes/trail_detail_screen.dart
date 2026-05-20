import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/base/wanderer_map.dart';
import 'package:wanderer/components/trail/trail_dropdown.dart';
import 'package:wanderer/components/trail/trail_panel.dart';
import 'package:wanderer/models/trail.dart';
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
          data: (trail) => buildMap(trailAsync.requireValue),
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (err, stack) => WandererError(err: err, stack: stack),
        ),
      ),
    );
  }

  Widget buildMap(Trail trail) {
    return Stack(
      children: [
        WandererMap(trail: trail),
        DraggableScrollableSheet(
          initialChildSize: 0.4,
          minChildSize: 0.15,
          maxChildSize: 0.9,
          builder: (context, scrollController) {
            return Container(
              decoration: BoxDecoration(
                color: Theme.of(context).canvasColor,
                borderRadius: const BorderRadius.vertical(
                  top: Radius.circular(20),
                ),
                boxShadow: [
                  BoxShadow(
                    blurRadius: 10,
                    color: Colors.black.withValues(alpha: 0.1),
                  ),
                ],
              ),
              // The extracted content function
              child: TrailPanel(
                trail: trail,
                scrollController: scrollController,
                actionMenu: TrailDropdown(trail: trail),
              ),
            );
          },
        ),
      ],
    );
  }
}
