import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/base/wanderer_no_results.dart';
import 'package:wanderer/components/search/search_history_list.dart';
import 'package:wanderer/components/search/search_result_tiles.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/search_history_item.dart';
import 'package:wanderer/provider/search/global_search_provider.dart';
import 'package:wanderer/provider/search/search_history_provider.dart';

/// Location-only search shell mirroring [GlobalSearchScreen]'s visual
/// structure, minus the category-chip row and every non-location result
/// branch. Selecting a result pops the [LocationSearchResult] back to the
/// caller (the Route Planner) instead of navigating to `/map` — the planner
/// owns the camera animation.
class LocationSearchScreen extends ConsumerStatefulWidget {
  const LocationSearchScreen({super.key});

  @override
  ConsumerState<LocationSearchScreen> createState() =>
      _LocationSearchScreenState();
}

class _LocationSearchScreenState extends ConsumerState<LocationSearchScreen> {
  late final TextEditingController _controller;
  // Captured up front (not read inside dispose) — `ref` may already be torn
  // down by the time State.dispose runs (avoid_ref_inside_state_dispose).
  late final GlobalSearchNotifier _notifier;

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController();
    _controller.addListener(() => setState(() {}));
    _notifier = ref.read(globalSearchProvider.notifier);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      _notifier.setCategory(GlobalSearchCategory.locations);
      _notifier.setQuery('');
    });
  }

  @override
  void dispose() {
    // Reset the shared provider so this locations-only session doesn't leak
    // a stuck filter/state into the global search screen.
    _notifier.setCategory(GlobalSearchCategory.all);
    _notifier.setQuery('');
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(globalSearchProvider);
    final notifier = ref.read(globalSearchProvider.notifier);

    return Scaffold(
      appBar: AppBar(
        title: Builder(
          builder: (context) {
            final colorScheme = Theme.of(context).colorScheme;
            return TextField(
              controller: _controller,
              autofocus: true,
              onChanged: notifier.setQuery,
              cursorColor: colorScheme.onSurface,
              decoration: InputDecoration(
                hintText: AppLocalizations.of(context)!.search_for_a_location,
                enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(56),
                  borderSide: BorderSide(color: colorScheme.outline, width: 1),
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(56),
                  borderSide: BorderSide(
                    color: colorScheme.onSurface,
                    width: 1,
                  ),
                ),
                hintStyle: TextStyle(color: Colors.grey),
              ),
              style: const TextStyle(fontSize: 16),
            );
          },
        ),
        actions: [
          if (_controller.text.isNotEmpty)
            IconButton(
              icon: const Icon(Icons.close),
              onPressed: () {
                _controller.clear();
                notifier.setQuery('');
              },
            ),
        ],
      ),
      body: _LocationResultsList(state: state),
    );
  }
}

class _LocationResultsList extends ConsumerWidget {
  final GlobalSearchState state;

  const _LocationResultsList({required this.state});

  Widget _tile(BuildContext context, WidgetRef ref, LocationSearchResult l) {
    return SearchLocationTile(
      location: l,
      onTap: () {
        ref.read(searchHistoryProvider.notifier).record(LocationHistoryItem(l));
        context.pop(l);
      },
    );
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    if (state.isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (state.query.isEmpty) {
      final history = ref
          .watch(searchHistoryProvider)
          .whereType<LocationHistoryItem>()
          .toList();
      if (history.isNotEmpty) {
        return SearchHistoryList(
          items: history,
          tileBuilder: (item) =>
              _tile(context, ref, (item as LocationHistoryItem).location),
        );
      }
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.search, size: 64, color: Colors.grey.shade300),
            const SizedBox(height: 12),
            Text(
              l10n.search_for_a_location,
              style: TextStyle(color: Colors.grey.shade500),
            ),
          ],
        ),
      );
    }

    final locations = state.visibleLocations;

    if (locations.isEmpty) {
      return WandererNoResults(query: state.query);
    }

    return ListView(
      children: [for (final l in locations) _tile(context, ref, l)],
    );
  }
}
