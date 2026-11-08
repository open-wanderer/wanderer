import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/components/base/wanderer_sort_chip_group.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';

class TrailSortScreen extends ConsumerWidget {
  final String filterId;
  const TrailSortScreen({super.key, this.filterId = 'map'});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final filterState = ref.watch(trailFilterProvider(filterId));

    final currentSort = filterState.value?.sort ?? TrailFilterSort.name;
    final currentOrder = filterState.value?.sortOrder ?? SortOrder.desc;

    final List<SortOption<TrailFilterSort>> sortOptions = [
      SortOption(text: l10n.name, value: TrailFilterSort.name),
      SortOption(text: l10n.creation_date, value: TrailFilterSort.created),
      SortOption(text: l10n.date, value: TrailFilterSort.date),
      SortOption(text: l10n.difficulty, value: TrailFilterSort.difficulty),
      SortOption(text: l10n.distance, value: TrailFilterSort.distance),
      SortOption(text: l10n.duration, value: TrailFilterSort.duration),
      SortOption(
        text: l10n.elevation_gain,
        value: TrailFilterSort.elevation_gain,
      ),
      SortOption(
        text: l10n.elevation_loss,
        value: TrailFilterSort.elevation_loss,
      ),
      SortOption(text: l10n.likes, value: TrailFilterSort.like_count),
    ];

    return Scaffold(
      appBar: AppBar(title: Text(l10n.sort)),
      body: Padding(
        padding: const EdgeInsets.symmetric(vertical: 8.0, horizontal: 16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(l10n.sort, style: Theme.of(context).textTheme.labelLarge),
            const SizedBox(height: 8),

            WandererSortChipGroup(
              options: sortOptions,
              currentSort: currentSort,
              sortOrder: currentOrder,
              onSortChanged: (newSort, newOrder) {
                ref
                    .read(trailFilterProvider(filterId).notifier)
                    .updateFilter(
                      (filter) =>
                          filter.copyWith(sort: newSort, sortOrder: newOrder),
                    );
              },
            ),
          ],
        ),
      ),
    );
  }
}
