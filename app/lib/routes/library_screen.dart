import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/trail/trail_card.dart';
import 'package:wanderer/components/trail/trail_quick_filter_bar.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/router_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';

class LibraryScreen extends ConsumerStatefulWidget {
  const LibraryScreen({super.key});

  @override
  ConsumerState<LibraryScreen> createState() => _LibraryScreenState();
}

class _LibraryScreenState extends ConsumerState<LibraryScreen> {
  final _searchController = TextEditingController();
  String _query = '';

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  List<Trail> _filtered(List<Trail> trails, TrailFilter? filter) {
    final q = _query.trim().toLowerCase();
    List<Trail> result = q.isEmpty
        ? trails
        : trails.where((t) {
            return t.name.toLowerCase().contains(q) ||
                t.description.toLowerCase().contains(q);
          }).toList();

    if (filter != null) {
      result = applyTrailFilter(result, filter);
    }

    return result;
  }

  @override
  Widget build(BuildContext context) {
    final trailLibrary = ref.watch(trailLibraryProvider);
    final router = ref.watch(routerProvider);
    final filterAsync = ref.watch(trailFilterProvider('library'));
    final visible = _filtered(trailLibrary, filterAsync.value);

    return SafeArea(
      bottom: false,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16.0),
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 8.0),
              child: TextField(
                controller: _searchController,
                onChanged: (v) => setState(() => _query = v),
                cursorColor: Theme.of(context).colorScheme.onSurface,
                decoration: InputDecoration(
                  hintText: 'Search library…',
                  prefixIcon: const Icon(Icons.search),
                  suffixIcon: _query.isNotEmpty
                      ? IconButton(
                          icon: const Icon(Icons.clear),
                          onPressed: () {
                            _searchController.clear();
                            setState(() => _query = '');
                          },
                        )
                      : null,
                  enabledBorder: OutlineInputBorder(
                    borderSide: BorderSide(
                      color: Theme.of(context).colorScheme.outline,
                    ),
                    borderRadius: BorderRadius.circular(56),
                  ),
                  focusedBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(56),
                    borderSide: BorderSide(
                      color: Theme.of(context).colorScheme.onSurface,
                      width: 1,
                    ),
                  ),
                  isDense: true,
                ),
              ),
            ),
            const TrailQuickFilterBar(filterId: 'library'),
            Expanded(
              child: ListView.builder(
                itemCount: visible.length,
                itemBuilder: (context, i) {
                  final trail = visible[i];
                  return TrailCard(
                    trail: trail,
                    onTrailSelect: () => router.push('/trail/${trail.id}'),
                    onLongPress: () => _showContextMenu(context, trail, router),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _showContextMenu(BuildContext context, Trail trail, router) {
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
                _confirmDelete(context, trail);
              },
            ),
          ],
        ),
      ),
    );
  }

  void _confirmDelete(BuildContext context, Trail trail) {
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
