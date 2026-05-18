import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/components/trail/trail_card.dart';
import 'package:wanderer/provider/router_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';

class LibraryScreen extends ConsumerWidget {
  const LibraryScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final trailLibrary = ref.watch(trailLibraryProvider);
    final router = ref.watch(routerProvider);

    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16.0),
          child: ListView.builder(
            itemCount: trailLibrary.length,
            itemBuilder: (context, i) {
              return TrailCard(
                trail: trailLibrary[i],
                onTrailSelect: () =>
                    router.push('/library/${trailLibrary[i].id}'),
              );
            },
          ),
        ),
      ),
    );
  }
}
