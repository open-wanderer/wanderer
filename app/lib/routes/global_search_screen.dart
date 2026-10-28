import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/base/wanderer_no_results.dart';
import 'package:wanderer/components/search/search_history_list.dart';
import 'package:wanderer/components/search/search_result_tiles.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/search_history_item.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/search/global_search_provider.dart';
import 'package:wanderer/provider/search/search_history_provider.dart';

class GlobalSearchScreen extends ConsumerStatefulWidget {
  const GlobalSearchScreen({super.key});

  @override
  ConsumerState<GlobalSearchScreen> createState() => _GlobalSearchScreenState();
}

class _GlobalSearchScreenState extends ConsumerState<GlobalSearchScreen> {
  late final TextEditingController _controller;

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController();
    _controller.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final state = ref.watch(globalSearchProvider);
    final notifier = ref.read(globalSearchProvider.notifier);
    final unit = ref.watch(unitProvider);

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
                hintText: l10n.search_for_trails_places,
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
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Only once there is something to narrow down. A specific
          // category stays reachable even when it found nothing, so the
          // hiker can switch back to the others.
          if (state.query.isNotEmpty &&
              (state.hasResults ||
                  state.category != GlobalSearchCategory.all)) ...[
            _CategoryChips(state: state, notifier: notifier, l10n: l10n),
            const Divider(height: 1),
          ],
          Expanded(
            child: _ResultsList(state: state, l10n: l10n, unit: unit),
          ),
        ],
      ),
    );
  }
}

class _CategoryChips extends StatelessWidget {
  final GlobalSearchState state;
  final GlobalSearchNotifier notifier;
  final AppLocalizations l10n;

  const _CategoryChips({
    required this.state,
    required this.notifier,
    required this.l10n,
  });

  String _label(GlobalSearchCategory cat) => switch (cat) {
    GlobalSearchCategory.all => l10n.all,
    GlobalSearchCategory.trails => l10n.trail(2),
    GlobalSearchCategory.lists => l10n.list(2),
    GlobalSearchCategory.locations => l10n.locations,
    GlobalSearchCategory.actors => l10n.users,
  };

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: GlobalSearchCategory.values.map((cat) {
          final selected = state.category == cat;
          return Padding(
            padding: const EdgeInsets.only(right: 8),
            child: ChoiceChip(
              label: Text(_label(cat)),
              selected: selected,
              onSelected: (_) => notifier.setCategory(cat),
              backgroundColor: colorScheme.surfaceContainerHighest,
              selectedColor: colorScheme.primaryContainer,
              checkmarkColor: colorScheme.onPrimaryContainer,
              shape: StadiumBorder(
                side: BorderSide(
                  color: selected
                      ? colorScheme.outlineVariant
                      : colorScheme.outline,
                ),
              ),
              labelStyle: TextStyle(
                color: selected
                    ? colorScheme.onPrimaryContainer
                    : colorScheme.onSurface,
                fontWeight: selected ? FontWeight.w600 : FontWeight.normal,
              ),
            ),
          );
        }).toList(),
      ),
    );
  }
}

/// Opens [item] and remembers it in the search history -- the one path
/// for both a live result and a history entry, so tapping either moves it
/// to the top of the history.
void _open(BuildContext context, WidgetRef ref, SearchHistoryItem item) {
  ref.read(searchHistoryProvider.notifier).record(item);
  switch (item) {
    case TrailHistoryItem(:final trail):
      context.push('/trail/${trail.id}');
    case ListHistoryItem(:final list):
      context.push('/list/${list.id}');
    case LocationHistoryItem(:final location):
      context.go(
        '/map',
        extra: {'lat': location.lat, 'lon': location.lon, 'zoom': 13.0},
      );
    case ActorHistoryItem(:final actor):
      context.push(
        actor.isLocal
            ? '/profile/@${actor.preferredUsername}'
            : '/profile/@${actor.preferredUsername}@${actor.domain}',
      );
  }
}

class _ResultsList extends ConsumerWidget {
  final GlobalSearchState state;
  final AppLocalizations l10n;
  final String unit;

  const _ResultsList({
    required this.state,
    required this.l10n,
    required this.unit,
  });

  Widget _tile(BuildContext context, WidgetRef ref, SearchHistoryItem item) {
    void onTap() => _open(context, ref, item);
    return switch (item) {
      TrailHistoryItem(:final trail) => SearchTrailTile(
        trail: trail,
        unit: unit,
        onTap: onTap,
      ),
      ListHistoryItem(:final list) => SearchListTile(list: list, onTap: onTap),
      LocationHistoryItem(:final location) => SearchLocationTile(
        location: location,
        onTap: onTap,
      ),
      ActorHistoryItem(:final actor) => SearchActorTile(
        actor: actor,
        onTap: onTap,
      ),
    };
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (state.isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (state.query.isEmpty) {
      final history = ref.watch(searchHistoryProvider);
      if (history.isNotEmpty) {
        return SearchHistoryList(
          items: history,
          tileBuilder: (item) => _tile(context, ref, item),
        );
      }
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.search, size: 64, color: Colors.grey.shade300),
            const SizedBox(height: 12),
            Text(
              l10n.search_for_trails_places,
              style: TextStyle(color: Colors.grey.shade500),
            ),
          ],
        ),
      );
    }

    if (!state.hasResults) {
      return WandererNoResults(query: state.query);
    }

    return ListView(
      children: [
        for (final t in state.visibleTrails)
          _tile(context, ref, TrailHistoryItem(t)),
        for (final l in state.visibleLists)
          _tile(context, ref, ListHistoryItem(l)),
        for (final l in state.visibleLocations)
          _tile(context, ref, LocationHistoryItem(l)),
        for (final a in state.visibleActors)
          _tile(context, ref, ActorHistoryItem(a)),
      ],
    );
  }
}
