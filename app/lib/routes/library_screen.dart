import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/trail/trail_card.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
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
              final trail = trailLibrary[i];
              return TrailCard(
                trail: trail,
                onTrailSelect: () => router.push('/trail/${trail.id}'),
                onLongPress: () =>
                    _showContextMenu(context, ref, trail, router),
              );
            },
          ),
        ),
      ),
    );
  }

  void _showContextMenu(
    BuildContext context,
    WidgetRef ref,
    Trail trail,
    router,
  ) {
    final l18n = AppLocalizations.of(context)!;

    showModalBottomSheet(
      context: context,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const FaIcon(
                FontAwesomeIcons.arrowUpRightFromSquare,
                size: 18,
              ),
              title: Text(l18n.open),
              onTap: () {
                Navigator.of(ctx).pop();
                router.push('/trail/${trail.id}');
              },
            ),
            ListTile(
              leading: const FaIcon(
                FontAwesomeIcons.trash,
                color: Colors.red,
                size: 18,
              ),
              title: Text(
                l18n.delete,
                style: const TextStyle(color: Colors.red),
              ),
              onTap: () {
                Navigator.of(ctx).pop();
                _confirmDelete(context, ref, trail);
              },
            ),
          ],
        ),
      ),
    );
  }

  void _confirmDelete(BuildContext context, WidgetRef ref, Trail trail) {
    final l18n = AppLocalizations.of(context)!;

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        content: Text(l18n.delete_trail_confirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: Text(l18n.cancel),
          ),
          TextButton(
            onPressed: () {
              Navigator.of(ctx).pop();
              ref.read(trailLibraryProvider.notifier).deleteTrail(trail.id);
            },
            child: Text(l18n.delete, style: const TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );
  }
}
