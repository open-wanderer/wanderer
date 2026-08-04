import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/components/trail/trail_panel.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';

class LibraryDetailScreen extends ConsumerWidget {
  final String id;

  const LibraryDetailScreen({super.key, required this.id});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final trail = ref.watch(trailLibraryProvider).firstWhere((t) => t.id == id);

    return Stack(
      children: [
        DraggableScrollableSheet(
          initialChildSize: 0.3,
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
                // Every trail on this screen came straight out of
                // trailLibraryProvider, so library membership is a
                // tautology here -- without this the stored-on-device
                // badge (D-10) would silently disappear from the Library
                // detail sheet.
                availableOffline: true,
              ),
            );
          },
        ),
      ],
    );
  }
}
