import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/base/wanderer_date_picker.dart';
import 'package:wanderer/components/base/wanderer_filter_chip.dart';
import 'package:wanderer/components/base/wanderer_radio_group.dart';
import 'package:wanderer/components/base/wanderer_range_slider.dart';
import 'package:wanderer/components/base/wanderer_sort_chip_group.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/category_preference_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/subcategory_preference_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/routes/trail_filter_screen.dart';
import 'package:wanderer/util/format.dart';
import 'package:wanderer/components/category/category_icon.dart';

class TrailQuickFilterBar extends ConsumerWidget {
  final String filterId;

  const TrailQuickFilterBar({super.key, required this.filterId});

  bool _isSortActive(TrailFilter filter) {
    return filter.sort != TrailFilterSort.created ||
        filter.sortOrder != SortOrder.desc;
  }

  bool _isCategoryActive(TrailFilter filter) {
    return filter.category.isNotEmpty || filter.subcategory.isNotEmpty;
  }

  bool _isDifficultyActive(TrailFilter filter) {
    return filter.difficulty.length != 3;
  }

  bool _isDistanceActive(TrailFilter filter) {
    return filter.distanceMin > 0 || filter.distanceMax < filter.distanceLimit;
  }

  bool _isElevationActive(TrailFilter filter) {
    return filter.elevationGainMin > 0 ||
        filter.elevationGainMax < filter.elevationGainLimit ||
        filter.elevationLossMin > 0 ||
        filter.elevationLossMax < filter.elevationLossLimit;
  }

  bool _isDateActive(TrailFilter filter) {
    return filter.startDate != null || filter.endDate != null;
  }

  bool _isCompletionActive(TrailFilter filter) {
    return filter.completed != null;
  }

  bool _isAnyActive(TrailFilter filter) {
    return _isSortActive(filter) ||
        _isCategoryActive(filter) ||
        _isDifficultyActive(filter) ||
        _isElevationActive(filter) ||
        _isDateActive(filter) ||
        _isCompletionActive(filter);
  }

  void _showSortSheet(BuildContext context, WidgetRef ref, TrailFilter filter) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (ctx) {
        return DraggableScrollableSheet(
          initialChildSize: 0.5,
          minChildSize: 0.3,
          maxChildSize: 0.9,
          expand: false,
          builder: (ctx, scrollController) {
            return Consumer(
              builder: (context, ref, _) {
                final filterAsync = ref.watch(trailFilterProvider(filterId));
                final currentFilter = filterAsync.value ?? filter;
                final l10n = AppLocalizations.of(context)!;

                final List<SortOption<TrailFilterSort>> sortOptions = [
                  SortOption(text: l10n.name, value: TrailFilterSort.name),
                  SortOption(
                    text: l10n.creation_date,
                    value: TrailFilterSort.created,
                  ),
                  SortOption(text: l10n.date, value: TrailFilterSort.date),
                  SortOption(
                    text: l10n.difficulty,
                    value: TrailFilterSort.difficulty,
                  ),
                  SortOption(
                    text: l10n.distance,
                    value: TrailFilterSort.distance,
                  ),
                  SortOption(
                    text: l10n.duration,
                    value: TrailFilterSort.duration,
                  ),
                  SortOption(
                    text: l10n.elevation_gain,
                    value: TrailFilterSort.elevation_gain,
                  ),
                  SortOption(
                    text: l10n.elevation_loss,
                    value: TrailFilterSort.elevation_loss,
                  ),
                  SortOption(
                    text: l10n.likes,
                    value: TrailFilterSort.like_count,
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
                            IconButton(
                              onPressed: () => Navigator.of(context).pop(),
                              icon: FaIcon(FontAwesomeIcons.xmark, size: 18),
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
                                .read(trailFilterProvider(filterId).notifier)
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

  void _showCategorySheet(
    BuildContext context,
    WidgetRef ref,
    TrailFilter filter,
  ) {
    final focusedCategoryId = ValueNotifier<String?>(null);
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (ctx) {
        return DraggableScrollableSheet(
          initialChildSize: 0.5,
          minChildSize: 0.3,
          maxChildSize: 0.9,
          expand: false,
          builder: (ctx, scrollController) {
            return Consumer(
              builder: (context, ref, _) {
                final filterAsync = ref.watch(trailFilterProvider(filterId));
                final currentFilter = filterAsync.value ?? filter;
                final categories = ref.watch(categoryProvider);
                final l10n = AppLocalizations.of(context)!;
                final locale = Localizations.localeOf(context);

                final catPrefs =
                    ref.watch(categoryPreferenceProvider).value ?? [];
                final visibleCategories = (categories.value ?? [])
                    .where(
                      (c) =>
                          catPrefs
                              .firstWhereOrNull((p) => p.category == c.id)
                              ?.visible !=
                          false,
                    )
                    .toList();

                final subcategories = ref.watch(subcategoryProvider);
                final subPrefs =
                    ref.watch(subcategoryPreferenceProvider).value ?? [];

                return ValueListenableBuilder<String?>(
                  valueListenable: focusedCategoryId,
                  builder: (context, focusedId, _) {
                    final visibleSubs = subcategories
                        .where((s) => s.category == focusedId)
                        .where(
                          (s) =>
                              subPrefs
                                  .firstWhereOrNull(
                                    (p) => p.subcategory == s.id,
                                  )
                                  ?.visible !=
                              false,
                        )
                        .toList();

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
                                  l10n.categories,
                                  style: Theme.of(
                                    context,
                                  ).textTheme.titleMedium,
                                ),
                                IconButton(
                                  onPressed: () => Navigator.of(context).pop(),
                                  icon: FaIcon(
                                    FontAwesomeIcons.xmark,
                                    size: 18,
                                  ),
                                ),
                              ],
                            ),
                            const SizedBox(height: 12),
                            WandererFilterChip<Category>(
                              options: visibleCategories,
                              selectedValues: currentFilter.category,
                              multiple: true,
                              keepSelectedOnTap: true,
                              labelBuilder: (c) => c.displayName(locale),
                              avatarBuilder: (c) => categoryFilterAvatar(
                                c,
                                color:
                                    Theme.of(context).brightness ==
                                        Brightness.light
                                    ? Theme.of(context).primaryColor
                                    : Theme.of(context).colorScheme.onSurface,
                              ),
                              badgeCountBuilder: (c) => currentFilter
                                  .subcategory
                                  .where((s) => s.category == c.id)
                                  .length,
                              onItemTap: (c) => focusedCategoryId.value = c.id,
                              onChanged: (selected) {
                                final removedIds = currentFilter.category
                                    .map((c) => c.id)
                                    .toSet()
                                    .difference(
                                      selected.map((c) => c.id).toSet(),
                                    );
                                final newSubs = currentFilter.subcategory
                                    .where(
                                      (s) => !removedIds.contains(s.category),
                                    )
                                    .toList();
                                ref
                                    .read(
                                      trailFilterProvider(filterId).notifier,
                                    )
                                    .updateFilter(
                                      (f) => f.copyWith(
                                        category: selected,
                                        subcategory: newSubs,
                                      ),
                                    );
                              },
                              onLongPress: (c) {
                                if (focusedCategoryId.value == c.id) {
                                  focusedCategoryId.value = null;
                                }
                                final newCats = currentFilter.category
                                    .where((cat) => cat.id != c.id)
                                    .toList();
                                final newSubs = currentFilter.subcategory
                                    .where((s) => s.category != c.id)
                                    .toList();
                                ref
                                    .read(
                                      trailFilterProvider(filterId).notifier,
                                    )
                                    .updateFilter(
                                      (f) => f.copyWith(
                                        category: newCats,
                                        subcategory: newSubs,
                                      ),
                                    );
                              },
                            ),

                            AnimatedSize(
                              duration: const Duration(milliseconds: 200),
                              alignment: Alignment.topCenter,
                              child: visibleSubs.isEmpty
                                  ? const SizedBox.shrink()
                                  : Column(
                                      crossAxisAlignment:
                                          CrossAxisAlignment.start,
                                      children: [
                                        const SizedBox(height: 12),
                                        Text(
                                          l10n.subcategories,
                                          style: Theme.of(
                                            context,
                                          ).textTheme.titleMedium,
                                        ),
                                        const SizedBox(height: 8),
                                        WandererFilterChip<Subcategory>(
                                          options: visibleSubs,
                                          selectedValues:
                                              currentFilter.subcategory,
                                          multiple: true,
                                          labelBuilder: (s) =>
                                              s.displayName(locale),
                                          avatarBuilder: (s) =>
                                              subcategoryFilterAvatar(
                                                s,
                                                currentFilter.category
                                                    .firstWhereOrNull(
                                                      (c) => c.id == s.category,
                                                    ),
                                                locale,
                                              ),
                                          onChanged: (sel) => ref
                                              .read(
                                                trailFilterProvider(
                                                  filterId,
                                                ).notifier,
                                              )
                                              .updateFilter(
                                                (f) => f.copyWith(
                                                  subcategory: sel,
                                                ),
                                              ),
                                        ),
                                      ],
                                    ),
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
      },
    );
  }

  void _showDifficultySheet(
    BuildContext context,
    WidgetRef ref,
    TrailFilter filter,
  ) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (ctx) {
        return DraggableScrollableSheet(
          initialChildSize: 0.4,
          minChildSize: 0.3,
          maxChildSize: 0.9,
          expand: false,
          builder: (ctx, scrollController) {
            return Consumer(
              builder: (context, ref, _) {
                final filterAsync = ref.watch(trailFilterProvider(filterId));
                final currentFilter = filterAsync.value ?? filter;
                final l10n = AppLocalizations.of(context)!;

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
                              l10n.difficulty,
                              style: Theme.of(context).textTheme.titleMedium,
                            ),
                            IconButton(
                              onPressed: () => Navigator.of(context).pop(),
                              icon: FaIcon(FontAwesomeIcons.xmark, size: 18),
                            ),
                          ],
                        ),
                        const SizedBox(height: 12),
                        WandererFilterChip<int>(
                          options: [0, 1, 2],
                          multiple: true,
                          selectedValues: currentFilter.difficulty,
                          labelBuilder: (c) {
                            switch (c) {
                              case 0:
                                return l10n.easy;
                              case 1:
                                return l10n.moderate;
                              case 2:
                                return l10n.difficult;
                            }
                            return 'Unknown';
                          },
                          onChanged: (selected) {
                            ref
                                .read(trailFilterProvider(filterId).notifier)
                                .updateFilter(
                                  (f) => f.copyWith(difficulty: selected),
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

  void _showElevationSheet(
    BuildContext context,
    WidgetRef ref,
    TrailFilter filter,
  ) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (ctx) {
        return DraggableScrollableSheet(
          initialChildSize: 0.5,
          minChildSize: 0.3,
          maxChildSize: 0.9,
          expand: false,
          builder: (ctx, scrollController) {
            return Consumer(
              builder: (context, ref, _) {
                final filterAsync = ref.watch(trailFilterProvider(filterId));
                final currentFilter = filterAsync.value ?? filter;
                final unit = ref.watch(unitProvider);
                final l10n = AppLocalizations.of(context)!;

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
                              l10n.distance,
                              style: Theme.of(context).textTheme.titleMedium,
                            ),
                            IconButton(
                              onPressed: () => Navigator.of(context).pop(),
                              icon: FaIcon(FontAwesomeIcons.xmark, size: 18),
                            ),
                          ],
                        ),
                        const SizedBox(height: 12),
                        Text(
                          l10n.elevation_gain,
                          style: Theme.of(context).textTheme.labelLarge,
                        ),
                        WandererRangeSlider(
                          values: RangeValues(
                            currentFilter.elevationGainMin,
                            currentFilter.elevationGainMax,
                          ),
                          min: 0,
                          max: currentFilter.elevationGainLimit > 0
                              ? currentFilter.elevationGainLimit
                              : 1,
                          labelsBuilder: (values) => RangeLabels(
                            formatElevation(values.start, unit: unit),
                            '${formatElevation(values.end, unit: unit)}${values.end >= currentFilter.elevationGainLimit ? "+" : ""}',
                          ),
                          onChangeEnd: (values) {
                            ref
                                .read(trailFilterProvider(filterId).notifier)
                                .updateFilter(
                                  (f) => f.copyWith(
                                    elevationGainMin: values.start,
                                    elevationGainMax: values.end,
                                  ),
                                );
                          },
                        ),
                        const SizedBox(height: 16),
                        Text(
                          l10n.elevation_loss,
                          style: Theme.of(context).textTheme.labelLarge,
                        ),
                        WandererRangeSlider(
                          values: RangeValues(
                            currentFilter.elevationLossMin,
                            currentFilter.elevationLossMax,
                          ),
                          min: 0,
                          max: currentFilter.elevationLossLimit > 0
                              ? currentFilter.elevationLossLimit
                              : 1,
                          labelsBuilder: (values) => RangeLabels(
                            formatElevation(values.start, unit: unit),
                            '${formatElevation(values.end, unit: unit)}${values.end >= currentFilter.elevationLossLimit ? "+" : ""}',
                          ),
                          onChangeEnd: (values) {
                            ref
                                .read(trailFilterProvider(filterId).notifier)
                                .updateFilter(
                                  (f) => f.copyWith(
                                    elevationLossMin: values.start,
                                    elevationLossMax: values.end,
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

  void _showDistanceSheet(
    BuildContext context,
    WidgetRef ref,
    TrailFilter filter,
  ) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (ctx) {
        return DraggableScrollableSheet(
          initialChildSize: 0.3,
          minChildSize: 0.3,
          maxChildSize: 0.5,
          expand: false,
          builder: (ctx, scrollController) {
            return Consumer(
              builder: (context, ref, _) {
                final filterAsync = ref.watch(trailFilterProvider(filterId));
                final currentFilter = filterAsync.value ?? filter;
                final unit = ref.watch(unitProvider);
                final l10n = AppLocalizations.of(context)!;

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
                              l10n.distance,
                              style: Theme.of(context).textTheme.titleMedium,
                            ),
                            IconButton(
                              onPressed: () => Navigator.of(context).pop(),
                              icon: FaIcon(FontAwesomeIcons.xmark, size: 18),
                            ),
                          ],
                        ),
                        const SizedBox(height: 12),
                        Text(
                          l10n.distance,
                          style: Theme.of(context).textTheme.labelLarge,
                        ),
                        WandererRangeSlider(
                          values: RangeValues(
                            currentFilter.distanceMin,
                            currentFilter.distanceMax,
                          ),
                          min: 0,
                          max: currentFilter.distanceLimit > 0
                              ? currentFilter.distanceLimit
                              : 1,
                          labelsBuilder: (values) => RangeLabels(
                            formatDistance(values.start, unit: unit),
                            '${formatDistance(values.end, unit: unit)}${values.end >= currentFilter.distanceLimit ? "+" : ""}',
                          ),
                          onChangeEnd: (values) {
                            ref
                                .read(trailFilterProvider(filterId).notifier)
                                .updateFilter(
                                  (f) => f.copyWith(
                                    distanceMin: values.start,
                                    distanceMax: values.end,
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

  void _showDateSheet(BuildContext context, WidgetRef ref, TrailFilter filter) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (ctx) {
        return DraggableScrollableSheet(
          initialChildSize: 0.5,
          minChildSize: 0.3,
          maxChildSize: 0.9,
          expand: false,
          builder: (ctx, scrollController) {
            return Consumer(
              builder: (context, ref, _) {
                final filterAsync = ref.watch(trailFilterProvider(filterId));
                final currentFilter = filterAsync.value ?? filter;
                final l10n = AppLocalizations.of(context)!;

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
                              l10n.date,
                              style: Theme.of(context).textTheme.titleMedium,
                            ),
                            IconButton(
                              onPressed: () => Navigator.of(context).pop(),
                              icon: FaIcon(FontAwesomeIcons.xmark, size: 18),
                            ),
                          ],
                        ),
                        const SizedBox(height: 12),
                        Text(
                          l10n.before,
                          style: Theme.of(context).textTheme.labelLarge,
                        ),
                        const SizedBox(height: 8),
                        FormBuilder(
                          child: WandererDatePicker(
                            name: 'startDate_$filterId',
                            initialValue: currentFilter.startDate,
                            onChanged: (value) {
                              ref
                                  .read(trailFilterProvider(filterId).notifier)
                                  .updateFilter(
                                    (f) => f.copyWith(startDate: value),
                                  );
                            },
                          ),
                        ),
                        const SizedBox(height: 16),
                        Text(
                          l10n.after,
                          style: Theme.of(context).textTheme.labelLarge,
                        ),
                        const SizedBox(height: 8),
                        FormBuilder(
                          child: WandererDatePicker(
                            name: 'endDate_$filterId',
                            initialValue: currentFilter.endDate,
                            onChanged: (value) {
                              ref
                                  .read(trailFilterProvider(filterId).notifier)
                                  .updateFilter(
                                    (f) => f.copyWith(endDate: value),
                                  );
                            },
                          ),
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

  void _showCompletionSheet(
    BuildContext context,
    WidgetRef ref,
    TrailFilter filter,
  ) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (ctx) {
        return DraggableScrollableSheet(
          initialChildSize: 0.4,
          minChildSize: 0.3,
          maxChildSize: 0.9,
          expand: false,
          builder: (ctx, scrollController) {
            return Consumer(
              builder: (context, ref, _) {
                final filterAsync = ref.watch(trailFilterProvider(filterId));
                final currentFilter = filterAsync.value ?? filter;
                final l10n = AppLocalizations.of(context)!;

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
                              l10n.completion_status,
                              style: Theme.of(context).textTheme.titleMedium,
                            ),
                            IconButton(
                              onPressed: () => Navigator.of(context).pop(),
                              icon: FaIcon(FontAwesomeIcons.xmark, size: 18),
                            ),
                          ],
                        ),
                        const SizedBox(height: 12),
                        FormBuilder(
                          child: WandererRadioGroup<CompletionStatus>(
                            name: 'completion_$filterId',
                            initialValue: CompletionStatus.fromBool(
                              currentFilter.completed,
                            ),
                            options: [
                              WandererRadioOption(
                                label: l10n.completed,
                                value: CompletionStatus.completed,
                              ),
                              WandererRadioOption(
                                label: l10n.not_completed,
                                value: CompletionStatus.notCompleted,
                              ),
                              WandererRadioOption(
                                label: l10n.no_preference,
                                value: CompletionStatus.noPreference,
                              ),
                            ],
                            onValueChanged: (status) {
                              ref
                                  .read(trailFilterProvider(filterId).notifier)
                                  .updateFilter(
                                    (f) =>
                                        f.copyWith(completed: status?.toBool()),
                                  );
                            },
                          ),
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
    final filterAsync = ref.watch(trailFilterProvider(filterId));
    final filter = filterAsync.value;

    final colorScheme = Theme.of(context).colorScheme;

    Color chipBg(bool active) =>
        active ? colorScheme.primaryContainer : colorScheme.secondaryContainer;

    Color chipLabelColor(bool active) =>
        active ? colorScheme.onPrimaryContainer : colorScheme.onSurface;

    Widget buildChip({
      required String label,
      required IconData icon,
      required bool active,
      required VoidCallback onPressed,
    }) {
      return ActionChip(
        backgroundColor: chipBg(active),
        labelStyle: TextStyle(color: chipLabelColor(active)),
        avatar: Icon(icon, size: 16, color: chipLabelColor(active)),
        label: Text(label),
        onPressed: onPressed,
        shape: StadiumBorder(side: BorderSide(color: Colors.transparent)),
      );
    }

    final sortActive = filter != null && _isSortActive(filter);
    final categoryActive = filter != null && _isCategoryActive(filter);
    final difficultyActive = filter != null && _isDifficultyActive(filter);
    final elevationActive = filter != null && _isElevationActive(filter);
    final distanceActive = filter != null && _isDistanceActive(filter);
    final dateActive = filter != null && _isDateActive(filter);
    final completionActive = filter != null && _isCompletionActive(filter);
    final anyActive = filter != null && _isAnyActive(filter);

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
              buildChip(
                label: l10n.sort,
                icon: Icons.swap_vert,
                active: sortActive,
                onPressed: filter != null
                    ? () => _showSortSheet(context, ref, filter)
                    : () {},
              ),
              const SizedBox(width: 8),
              buildChip(
                label: l10n.categories,
                icon: Icons.category_outlined,
                active: categoryActive,
                onPressed: filter != null
                    ? () => _showCategorySheet(context, ref, filter)
                    : () {},
              ),
              const SizedBox(width: 8),
              buildChip(
                label: l10n.difficulty,
                icon: Icons.signal_cellular_alt_outlined,
                active: difficultyActive,
                onPressed: filter != null
                    ? () => _showDifficultySheet(context, ref, filter)
                    : () {},
              ),
              const SizedBox(width: 8),
              buildChip(
                label: l10n.distance,
                icon: Icons.straighten,
                active: distanceActive,
                onPressed: filter != null
                    ? () => _showDistanceSheet(context, ref, filter)
                    : () {},
              ),
              const SizedBox(width: 8),
              buildChip(
                label: l10n.elevation_gain,
                icon: Icons.landscape_outlined,
                active: elevationActive,
                onPressed: filter != null
                    ? () => _showElevationSheet(context, ref, filter)
                    : () {},
              ),
              const SizedBox(width: 8),
              buildChip(
                label: l10n.date,
                icon: Icons.calendar_today_outlined,
                active: dateActive,
                onPressed: filter != null
                    ? () => _showDateSheet(context, ref, filter)
                    : () {},
              ),
              const SizedBox(width: 8),
              buildChip(
                label: l10n.completion_status,
                icon: Icons.check_circle_outline,
                active: completionActive,
                onPressed: filter != null
                    ? () => _showCompletionSheet(context, ref, filter)
                    : () {},
              ),
              if (anyActive) ...[
                const SizedBox(width: 4),
                TextButton(
                  onPressed: () => ref
                      .read(trailFilterProvider(filterId).notifier)
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
