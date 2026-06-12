import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';

class SortOption {
  final String text;
  final TrailFilterSort value;

  SortOption({required this.text, required this.value});
}

class TrailSortScreen extends ConsumerWidget {
  const TrailSortScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final filterState = ref.watch(trailFilterProvider);

    final currentSort = filterState.value?.sort ?? TrailFilterSort.name;
    final currentOrder =
        filterState.value?.sortOrder ?? TrailFilterSortOrder.desc;

    final List<SortOption> sortOptions = [
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

            _TrailSortChipGroup(
              options: sortOptions,
              currentSort: currentSort,
              sortOrder: currentOrder,
              onSortChanged: (newSort, newOrder) {
                ref
                    .read(trailFilterProvider.notifier)
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

class _TrailSortChipGroup extends StatelessWidget {
  final List<SortOption> options;
  final TrailFilterSort currentSort;
  final TrailFilterSortOrder sortOrder;
  final void Function(TrailFilterSort sort, TrailFilterSortOrder order)
  onSortChanged;

  const _TrailSortChipGroup({
    required this.options,
    required this.currentSort,
    required this.sortOrder,
    required this.onSortChanged,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return Wrap(
      spacing: 8.0,
      runSpacing: 4.0,
      children: options.map((option) {
        final isSelected = option.value == currentSort;

        final isDescending = sortOrder == TrailFilterSortOrder.desc;
        final turns = isDescending ? 0.5 : 0.0;

        return RawChip(
          label: Text(option.text),
          avatar: AnimatedSize(
            duration: const Duration(milliseconds: 300),
            curve: Curves.easeOut,
            child: isSelected
                ? SizedBox(
                    width: 18,
                    height: 18,
                    child: AnimatedRotation(
                      turns: turns,
                      duration: const Duration(milliseconds: 650),
                      curve: Curves.elasticOut,
                      child: Icon(
                        Icons.arrow_upward,
                        size: 16,
                        color: colorScheme.onPrimaryContainer,
                      ),
                    ),
                  )
                : const SizedBox.shrink(),
          ),
          selected: isSelected,
          showCheckmark: false,
          onPressed: () {
            if (isSelected) {
              final nextOrder = sortOrder == TrailFilterSortOrder.asc
                  ? TrailFilterSortOrder.desc
                  : TrailFilterSortOrder.asc;
              onSortChanged(option.value, nextOrder);
            } else {
              onSortChanged(option.value, TrailFilterSortOrder.asc);
            }
          },
          backgroundColor: colorScheme.surfaceContainerHighest,
          selectedColor: colorScheme.primaryContainer,
          shape: StadiumBorder(
            side: BorderSide(
              color: isSelected
                  ? colorScheme.outlineVariant
                  : colorScheme.outline,
            ),
          ),
          labelStyle: TextStyle(
            color: isSelected
                ? colorScheme.onPrimaryContainer
                : colorScheme.onSurface,
            fontWeight: isSelected ? FontWeight.w600 : FontWeight.normal,
          ),
        );
      }).toList(),
    );
  }
}
