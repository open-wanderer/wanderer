import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:textfield_tags/textfield_tags.dart';
import 'package:wanderer/components/base/wanderer_autocomplete.dart';
import 'package:wanderer/components/base/wanderer_filter_chip.dart';
import 'package:wanderer/components/base/wanderer_searchbar.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/tag.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/tag_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';

class TrailFilterScreen extends ConsumerStatefulWidget {
  const TrailFilterScreen({super.key});

  @override
  ConsumerState<TrailFilterScreen> createState() => _TrailFilterScreenState();
}

class _TrailFilterScreenState extends ConsumerState<TrailFilterScreen> {
  @override
  Widget build(BuildContext context) {
    final filter = ref.watch(trailFilterProvider);
    final UserEntity user = ref.watch(authProvider).value!;
    final categories = ref.watch(categoryProvider);

    return Scaffold(
      appBar: AppBar(),
      body: filter.when(
        data: (f) {
          return Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24.0),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  filter.value?.toFilterText(actor: user.actorId) ??
                      "Filter loading...",
                ),
                Text(
                  AppLocalizations.of(context)?.categories ?? "Categories",
                  style: TextTheme.of(context).labelLarge,
                ),
                WandererFilterChip<Category>(
                  options: categories.value ?? [],
                  selectedValues: f.category,
                  labelBuilder: (c) => c.name,
                  onChanged: (categories) {
                    ref
                        .read(trailFilterProvider.notifier)
                        .updateFilter(
                          (filter) => filter.copyWith(category: categories),
                        );
                  },
                ),
                Text(
                  AppLocalizations.of(context)?.tags ?? "Tags",
                  style: TextTheme.of(context).labelLarge,
                ),
                WandererAutocomplete<Tag>(
                  hintText:
                      AppLocalizations.of(context)?.filter_tags ??
                      "Filter tags",
                  initialTags: filter.value?.tags
                      .map((t) => DynamicTagData(t.name, t))
                      .toList(),
                  optionsBuilder: (textEditingValue) async {
                    final tags = await ref
                        .read(tagProvider.notifier)
                        .searchByName(textEditingValue.text);
                    final tagData = tags
                        .map((t) => DynamicTagData(t.name, t))
                        .toList();

                    return tagData;
                  },
                  onSubmitted: (value) {
                    final newTag = Tag(name: value);

                    ref.read(trailFilterProvider.notifier).addTag(newTag);
                    return DynamicTagData(value, Tag(name: value));
                  },
                  onSelected: (value) =>
                      ref.read(trailFilterProvider.notifier).addTag(value.data),
                  onDeleted: (value) => ref
                      .read(trailFilterProvider.notifier)
                      .removeTag(value.data),
                ),
                Text(
                  AppLocalizations.of(context)?.author ?? "Author",
                  style: TextTheme.of(context).labelLarge,
                ),
                WandererSearchBar(
                  onChanged: (value) => ref
                      .read(trailFilterProvider.notifier)
                      .updateFilter((filter) => filter.copyWith(author: value)),
                ),
                Text(
                  AppLocalizations.of(context)?.visibilty_status ??
                      "Visibility status",
                  style: TextTheme.of(context).labelLarge,
                ),
                CheckboxListTile(
                  value: filter.value?.public,
                  title: Text(AppLocalizations.of(context)?.public ?? "Public"),
                  onChanged: (value) => ref
                      .read(trailFilterProvider.notifier)
                      .updateFilter((filter) => filter.copyWith(public: value)),
                ),
                CheckboxListTile(
                  value: filter.value?.private,
                  title: Text(
                    AppLocalizations.of(context)?.private ?? "Private",
                  ),
                  onChanged: (value) => ref
                      .read(trailFilterProvider.notifier)
                      .updateFilter(
                        (filter) => filter.copyWith(private: value),
                      ),
                ),
                CheckboxListTile(
                  value: filter.value?.shared,
                  title: Text(AppLocalizations.of(context)?.shared ?? "Shared"),

                  onChanged: (value) => ref
                      .read(trailFilterProvider.notifier)
                      .updateFilter((filter) => filter.copyWith(shared: value)),
                ),
                Text(
                  AppLocalizations.of(context)?.difficulty ?? "Difficulty",
                  style: TextTheme.of(context).labelLarge,
                ),
                WandererFilterChip<int>(
                  options: [0, 1, 2],
                  selectedValues: f.difficulty,
                  labelBuilder: (c) {
                    switch (c) {
                      case 0:
                        return AppLocalizations.of(context)?.easy ?? "Easy";
                      case 1:
                        return AppLocalizations.of(context)?.moderate ??
                            "Moderate";
                      case 2:
                        return AppLocalizations.of(context)?.difficult ??
                            "Difficult";
                    }

                    return "Unknown";
                  },
                  onChanged: (difficulty) {
                    ref
                        .read(trailFilterProvider.notifier)
                        .updateFilter(
                          (filter) => filter.copyWith(difficulty: difficulty),
                        );
                  },
                ),
                Text(
                  AppLocalizations.of(context)?.distance ?? "Distance",
                  style: TextTheme.of(context).labelLarge,
                ),
                RangeSlider(
                  values: RangeValues(
                    filter.value?.distanceMin ?? 0,
                    filter.value?.distanceMax ?? 0,
                  ),
                  min: filter.value?.distanceMin ?? 0,
                  max: filter.value?.distanceLimit ?? 0,
                  labels: RangeLabels(
                    (filter.value?.distanceMin ?? 0).round().toString(),
                    (filter.value?.distanceMax ?? 0).round().toString(),
                  ),
                  onChanged: (RangeValues values) {
                    ref
                        .read(trailFilterProvider.notifier)
                        .updateFilter(
                          (filter) => filter.copyWith(
                            distanceMin: values.start,
                            distanceMax: values.end,
                          ),
                        );
                  },
                ),
              ],
            ),
          );
        },
        loading: () => Center(child: CircularProgressIndicator()),
        error: (e, trace) => Text(e.toString()),
      ),
    );
  }
}
