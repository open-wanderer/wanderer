import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:textfield_tags/textfield_tags.dart';
import 'package:wanderer/components/base/wanderer_autocomplete.dart';
import 'package:wanderer/components/base/wanderer_filter_chip.dart';
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
    final tags = ref.watch(tagProvider);

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
                Text(tags.value?.length.toString() ?? "Tags loading"),
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
