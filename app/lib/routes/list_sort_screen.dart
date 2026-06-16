import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/components/base/wanderer_sort_chip_group.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/list.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/list/list_filter_provider.dart';

class ListSortScreen extends ConsumerWidget {
  const ListSortScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final filterState = ref.watch(listFilterProvider);

    final currentSort = filterState.value?.sort ?? ListFilterSort.name;
    final currentOrder = filterState.value?.sortOrder ?? SortOrder.desc;

    final List<SortOption<ListFilterSort>> listSortOptions = [
      SortOption(text: l10n.name, value: ListFilterSort.name),
      SortOption(text: l10n.creation_date, value: ListFilterSort.created),
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
              options: listSortOptions,
              currentSort: currentSort,
              sortOrder: currentOrder,
              onSortChanged: (newSort, newOrder) {
                ref
                    .read(listFilterProvider.notifier)
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
