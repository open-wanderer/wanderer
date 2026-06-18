import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/base/wanderer_searchbar.dart';
import 'package:wanderer/components/trail/trail_list_item.dart';
import 'package:wanderer/components/trail/trail_quick_filter_bar.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/profile/profile_trails_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';

/// Applies TrailFilter to a list of TrailSearchResult objects.
/// TrailSearchResult.difficulty is int (0/1/2).
/// TrailSearchResult.category is the category name string.
/// TrailSearchResult.completed is bool.
/// TrailSearchResult.date is epoch seconds.
List<TrailSearchResult> applyProfileTrailFilter(
  List<TrailSearchResult> trails,
  TrailFilter filter,
) {
  List<TrailSearchResult> result = trails.where((trail) {
    // Difficulty filter
    if (!filter.difficulty.contains(trail.difficulty)) return false;

    // Category filter (by name)
    if (filter.category.isNotEmpty) {
      final match = filter.category.any((c) => c.name == trail.category);
      if (!match) return false;
    }

    // Completion filter
    if (filter.completed != null && trail.completed != filter.completed) {
      return false;
    }

    // Date range filter (trail.date is epoch seconds)
    final trailDate = DateTime.fromMillisecondsSinceEpoch(trail.date * 1000);
    if (filter.startDate != null && trailDate.isBefore(filter.startDate!)) {
      return false;
    }
    if (filter.endDate != null && trailDate.isAfter(filter.endDate!)) {
      return false;
    }

    return true;
  }).toList();

  // Sort
  result.sort((a, b) {
    int cmp;
    switch (filter.sort) {
      case TrailFilterSort.name:
        cmp = a.name.compareTo(b.name);
        break;
      case TrailFilterSort.date:
        cmp = a.date.compareTo(b.date);
        break;
      case TrailFilterSort.difficulty:
        cmp = a.difficulty.compareTo(b.difficulty);
        break;
      case TrailFilterSort.distance:
        cmp = a.distance.compareTo(b.distance);
        break;
      case TrailFilterSort.duration:
        cmp = a.duration.compareTo(b.duration);
        break;
      case TrailFilterSort.elevation_gain:
        cmp = a.elevationGain.compareTo(b.elevationGain);
        break;
      case TrailFilterSort.elevation_loss:
        cmp = a.elevationLoss.compareTo(b.elevationLoss);
        break;
      case TrailFilterSort.created:
        cmp = a.created.compareTo(b.created);
        break;
      case TrailFilterSort.like_count:
        cmp = a.likeCount.compareTo(b.likeCount);
        break;
    }
    return filter.sortOrder == SortOrder.asc ? cmp : -cmp;
  });

  return result;
}

class ProfileTrailScreen extends ConsumerStatefulWidget {
  final String handle;
  const ProfileTrailScreen({super.key, required this.handle});

  @override
  ConsumerState<ProfileTrailScreen> createState() => _ProfileTrailScreenState();
}

class _ProfileTrailScreenState extends ConsumerState<ProfileTrailScreen> {
  late final ScrollController _scrollController;

  @override
  void initState() {
    super.initState();
    _scrollController = ScrollController();
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    final pos = _scrollController.position;
    if (!pos.hasContentDimensions) return;
    if (pos.maxScrollExtent <= 0) return;
    if (pos.pixels / pos.maxScrollExtent >= 0.8) {
      final state = ref.read(profileTrailsProvider(widget.handle));
      if (state.value?.hasMore == true && !state.isLoading) {
        ref.read(profileTrailsProvider(widget.handle).notifier).loadNextPage();
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final trailsAsync = ref.watch(profileTrailsProvider(widget.handle));
    final filterAsync = ref.watch(
      trailFilterProvider('profile_trail_${widget.handle}'),
    );

    return Scaffold(
      appBar: AppBar(
        leading: BackButton(onPressed: () => context.pop()),
        title: Text(AppLocalizations.of(context)!.trail(2)),
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
            child: WandererSearchBar(
              hintText: AppLocalizations.of(context)!.search,
              onChanged: (q) => ref
                  .read(profileTrailsProvider(widget.handle).notifier)
                  .search(q),
            ),
          ),
          TrailQuickFilterBar(
            filterId: 'profile_trail_${widget.handle}',
          ),
          Expanded(
            child: trailsAsync.when(
              data: (state) {
                final visible = filterAsync.value != null
                    ? applyProfileTrailFilter(state.trails, filterAsync.value!)
                    : state.trails;

                return RefreshIndicator(
                  onRefresh: () async =>
                      ref.invalidate(profileTrailsProvider(widget.handle)),
                  child: visible.isEmpty
                      ? _EmptyState()
                      : ListView.builder(
                          controller: _scrollController,
                          padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
                          itemCount:
                              visible.length + (state.hasMore ? 1 : 0),
                          itemBuilder: (context, index) {
                            if (index == visible.length) {
                              return const Padding(
                                padding: EdgeInsets.symmetric(vertical: 16),
                                child: Center(
                                  child: CircularProgressIndicator(),
                                ),
                              );
                            }
                            final trail = visible[index];
                            return TrailListItem(
                              trail: trail,
                              onTrailSelect: () =>
                                  context.push('/trail/${trail.id}'),
                            );
                          },
                        ),
                );
              },
              loading: () => const Center(child: CircularProgressIndicator()),
              error: (err, stack) => WandererError(err: err, stack: stack),
            ),
          ),
        ],
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Center(
      child: Text('No trails yet.', style: TextStyle(color: Colors.grey[600])),
    );
  }
}
