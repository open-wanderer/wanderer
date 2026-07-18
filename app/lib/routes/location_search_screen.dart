import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/provider/search/global_search_provider.dart';

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
                hintText: 'Search for a location',
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

class _LocationResultsList extends StatelessWidget {
  final GlobalSearchState state;

  const _LocationResultsList({required this.state});

  @override
  Widget build(BuildContext context) {
    if (state.isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (state.query.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.search, size: 64, color: Colors.grey.shade300),
            const SizedBox(height: 12),
            Text(
              'Search for a location',
              style: TextStyle(color: Colors.grey.shade500),
            ),
          ],
        ),
      );
    }

    final locations = state.visibleLocations;

    if (locations.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            SvgPicture.asset(
              "assets/svgs/empty_state_search_${Theme.of(context).brightness.name}.svg",
              semanticsLabel: 'wanderer comment empty state',
              height: 120,
            ),
            const SizedBox(height: 12),
            Text(
              'No results for "${state.query}"',
              style: TextStyle(color: Colors.grey.shade500),
            ),
          ],
        ),
      );
    }

    return ListView(
      children: locations
          .map((location) => _LocationResultTile(location: location))
          .toList(),
    );
  }
}

class _LocationResultTile extends StatelessWidget {
  final LocationSearchResult location;

  const _LocationResultTile({required this.location});

  @override
  Widget build(BuildContext context) {
    return ListTile(
      onTap: () => context.pop(location),
      leading: Container(
        width: 40,
        height: 40,
        decoration: BoxDecoration(
          color: Colors.orange.withValues(alpha: 0.15),
          borderRadius: BorderRadius.circular(8),
        ),
        child: const FaIcon(
          FontAwesomeIcons.locationDot,
          size: 18,
          color: Colors.orange,
        ),
      ),
      title: Text(location.name, maxLines: 1, overflow: TextOverflow.ellipsis),
      subtitle: location.description.isNotEmpty
          ? Text(
              location.description,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            )
          : null,
    );
  }
}
