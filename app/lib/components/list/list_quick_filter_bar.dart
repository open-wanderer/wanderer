import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/components/base/wanderer_sort_chip_group.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/list.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/list/list_filter_provider.dart';

class ListQuickFilterBar extends ConsumerWidget {
  final String filterId;

  const ListQuickFilterBar({super.key, required this.filterId});

  bool _isSortActive(ListFilter filter) {
    return filter.sort != ListFilterSort.created ||
        filter.sortOrder != SortOrder.desc;
  }

  void _showSortSheet(BuildContext context, WidgetRef ref, ListFilter filter) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (ctx) {
        return DraggableScrollableSheet(
          initialChildSize: 0.4,
          minChildSize: 0.25,
          maxChildSize: 0.6,
          expand: false,
          builder: (ctx, scrollController) {
            return Consumer(
              builder: (context, ref, _) {
                final filterAsync = ref.watch(listFilterProvider(filterId));
                final currentFilter = filterAsync.value ?? filter;
                final l10n = AppLocalizations.of(context)!;

                final sortOptions = [
                  SortOption(text: l10n.name, value: ListFilterSort.name),
                  SortOption(
                    text: l10n.creation_date,
                    value: ListFilterSort.created,
                  ),
                ];

                return SingleChildScrollView(
                  controller: scrollController,
                  child: Padding(
                    padding: const EdgeInsets.fromLTRB(16, 16, 16, 32),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            Text(
                              l10n.sort,
                              style: Theme.of(context).textTheme.titleMedium,
                            ),
                            TextButton(
                              onPressed: () => Navigator.of(context).pop(),
                              child: Text(l10n.close),
                            ),
                          ],
                        ),
                        const SizedBox(height: 12),
                        WandererSortChipGroup(
                          options: sortOptions,
                          currentSort: currentFilter.sort,
                          sortOrder: currentFilter.sortOrder,
                          onSortChanged: (newSort, newOrder) {
                            ref
                                .read(listFilterProvider(filterId).notifier)
                                .updateFilter(
                                  (f) => f.copyWith(
                                    sort: newSort,
                                    sortOrder: newOrder,
                                  ),
                                );
                          },
                        ),
                      ],
                    ),
                  ),
                );
              },
            );
          },
        );
      },
    );
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final filterAsync = ref.watch(listFilterProvider(filterId));
    final filter = filterAsync.value;

    final colorScheme = Theme.of(context).colorScheme;

    Color chipBg(bool active) =>
        active ? colorScheme.primaryContainer : colorScheme.secondaryContainer;

    Color chipLabelColor(bool active) =>
        active ? colorScheme.onPrimaryContainer : colorScheme.onSurface;

    final sortActive = filter != null && _isSortActive(filter);

    final l10n = AppLocalizations.of(context)!;

    return SizedBox(
      height: 48,
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: Padding(
          padding: const EdgeInsets.only(left: 8),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              ActionChip(
                backgroundColor: chipBg(sortActive),
                labelStyle: TextStyle(color: chipLabelColor(sortActive)),
                avatar: Icon(
                  Icons.swap_vert,
                  size: 16,
                  color: chipLabelColor(sortActive),
                ),
                label: Text(l10n.sort),
                onPressed: filter != null
                    ? () => _showSortSheet(context, ref, filter)
                    : () {},
                shape: const StadiumBorder(
                  side: BorderSide(color: Colors.transparent),
                ),
              ),
              if (sortActive) ...[
                const SizedBox(width: 4),
                TextButton(
                  onPressed: () => ref
                      .read(listFilterProvider(filterId).notifier)
                      .resetFilter(),
                  child: Text(l10n.reset),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
