import 'package:cached_network_image/cached_network_image.dart';
import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:textfield_tags/textfield_tags.dart';
import 'package:wanderer/components/base/wanderer_actor_search.dart';
import 'package:wanderer/components/base/wanderer_autocomplete.dart';
import 'package:wanderer/components/base/wanderer_date_picker.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/base/wanderer_filter_chip.dart';
import 'package:wanderer/components/base/wanderer_radio_group.dart';
import 'package:wanderer/components/base/wanderer_range_slider.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/models/tag.dart';
import 'package:wanderer/provider/category_preference_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/subcategory_preference_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/trail/tag_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/components/category/category_icon.dart';
import 'package:wanderer/util/format.dart';

class TrailFilterScreen extends ConsumerStatefulWidget {
  final String filterId;
  const TrailFilterScreen({super.key, this.filterId = 'map'});

  @override
  ConsumerState<TrailFilterScreen> createState() => _TrailFilterScreenState();
}

enum CompletionStatus {
  completed,
  notCompleted,
  noPreference;

  bool? toBool() => switch (this) {
    CompletionStatus.completed => true,
    CompletionStatus.notCompleted => false,
    CompletionStatus.noPreference => null,
  };

  static CompletionStatus fromBool(bool? value) => switch (value) {
    true => CompletionStatus.completed,
    false => CompletionStatus.notCompleted,
    null => CompletionStatus.noPreference,
  };
}

class _TrailFilterScreenState extends ConsumerState<TrailFilterScreen> {
  String? _focusedCategoryId;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    final filter = ref.watch(trailFilterProvider(widget.filterId));
    final categories = ref.watch(categoryProvider);
    final unit = ref.watch(unitProvider);

    // Hide categories the user has marked visible:false in
    // preferences. A null/absent preference means visible (`!= false`).
    final catPrefs = ref.watch(categoryPreferenceProvider).value ?? [];
    final visibleCategories = (categories.value ?? [])
        .where(
          (c) =>
              catPrefs.firstWhereOrNull((p) => p.category == c.id)?.visible !=
              false,
        )
        .toList();

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.filter_trails),
        actions: [
          IconButton(
            onPressed: () => ref
                .read(trailFilterProvider(widget.filterId).notifier)
                .resetFilter(),
            icon: FaIcon(FontAwesomeIcons.filterCircleXmark, size: 18),
          ),
        ],
      ),
      body: filter.when(
        data: (f) {
          // Subcategory options: scoped to the selected
          // categories and filtered by visibility preferences (null pref
          // = visible). `subcategoryProvider` is a synchronous List, NOT an
          // AsyncValue, so no `.value`.
          final subcategories = ref.watch(subcategoryProvider);
          final subPrefs = ref.watch(subcategoryPreferenceProvider).value ?? [];
          final visibleSubs = subcategories
              .where((s) => s.category == _focusedCategoryId)
              .where(
                (s) =>
                    subPrefs
                        .firstWhereOrNull((p) => p.subcategory == s.id)
                        ?.visible !=
                    false,
              )
              .toList();

          return Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24.0),
            child: ListView(
              children: [
                Text(l10n.categories, style: TextTheme.of(context).labelLarge),
                const SizedBox(height: 8),

                WandererFilterChip<Category>(
                  options: visibleCategories,
                  selectedValues: f.category,
                  multiple: true,
                  keepSelectedOnTap: true,
                  labelBuilder: (c) =>
                      c.displayName(Localizations.localeOf(context)),
                  avatarBuilder: (c) => _categoryAvatar(c),
                  badgeCountBuilder: (c) =>
                      f.subcategory.where((s) => s.category == c.id).length,
                  onItemTap: (c) => setState(() => _focusedCategoryId = c.id),
                  onChanged: (categories) {
                    final removedIds = f.category
                        .map((c) => c.id)
                        .toSet()
                        .difference(categories.map((c) => c.id).toSet());
                    final newSubs = f.subcategory
                        .where((s) => !removedIds.contains(s.category))
                        .toList();
                    ref
                        .read(trailFilterProvider(widget.filterId).notifier)
                        .updateFilter(
                          (filter) => filter.copyWith(
                            category: categories,
                            subcategory: newSubs,
                          ),
                        );
                  },
                  onLongPress: (c) {
                    if (_focusedCategoryId == c.id) {
                      setState(() => _focusedCategoryId = null);
                    }
                    final newCats = f.category
                        .where((cat) => cat.id != c.id)
                        .toList();
                    final newSubs = f.subcategory
                        .where((s) => s.category != c.id)
                        .toList();
                    ref
                        .read(trailFilterProvider(widget.filterId).notifier)
                        .updateFilter(
                          (filter) => filter.copyWith(
                            category: newCats,
                            subcategory: newSubs,
                          ),
                        );
                  },
                ),
                const SizedBox(height: 16),

                // Subcategories section. Animates in only when ≥1
                // category is selected and at least one scoped, visible
                // subcategory exists; otherwise it silently collapses to
                // nothing (UI-SPEC: no placeholder text).
                AnimatedSize(
                  duration: const Duration(milliseconds: 200),
                  alignment: Alignment.topCenter,
                  child: f.category.isEmpty || visibleSubs.isEmpty
                      ? const SizedBox.shrink()
                      : Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              l10n.subcategories,
                              style: TextTheme.of(context).labelLarge,
                            ),
                            const SizedBox(height: 8),
                            WandererFilterChip<Subcategory>(
                              options: visibleSubs,
                              selectedValues: f.subcategory,
                              multiple: true,
                              labelBuilder: (s) => s.displayName(
                                Localizations.localeOf(context),
                              ),
                              avatarBuilder: (s) => _subcategoryAvatar(
                                s,
                                f.category.firstWhereOrNull(
                                  (c) => c.id == s.category,
                                ),
                              ),
                              onChanged: (sel) => ref
                                  .read(
                                    trailFilterProvider(
                                      widget.filterId,
                                    ).notifier,
                                  )
                                  .updateFilter(
                                    (flt) => flt.copyWith(subcategory: sel),
                                  ),
                            ),
                            const SizedBox(height: 16),
                          ],
                        ),
                ),

                Text(l10n.tags, style: TextTheme.of(context).labelLarge),
                const SizedBox(height: 8),

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

                    ref
                        .read(trailFilterProvider(widget.filterId).notifier)
                        .addTag(newTag);
                    return DynamicTagData(value, Tag(name: value));
                  },
                  onSelected: (value) => ref
                      .read(trailFilterProvider(widget.filterId).notifier)
                      .addTag(value.data),
                  onDeleted: (value) => ref
                      .read(trailFilterProvider(widget.filterId).notifier)
                      .removeTag(value.data),
                ),
                const SizedBox(height: 16),

                Text(l10n.author, style: TextTheme.of(context).labelLarge),
                const SizedBox(height: 8),

                if (f.author != null)
                  Align(
                    alignment: Alignment.centerLeft,
                    child: _AuthorChip(
                      actor: f.author!,
                      onDeleted: () => ref
                          .read(trailFilterProvider(widget.filterId).notifier)
                          .updateFilter(
                            (filter) => filter.copyWith(author: null),
                          ),
                    ),
                  )
                else
                  WandererActorSearch(
                    hintText: l10n.author,
                    onSelected: (actor) => ref
                        .read(trailFilterProvider(widget.filterId).notifier)
                        .updateFilter(
                          (filter) => filter.copyWith(author: actor),
                        ),
                    onCleared: () => ref
                        .read(trailFilterProvider(widget.filterId).notifier)
                        .updateFilter(
                          (filter) => filter.copyWith(author: null),
                        ),
                  ),
                const SizedBox(height: 16),

                Text(
                  AppLocalizations.of(context)?.visibilty_status ??
                      "Visibility status",
                  style: TextTheme.of(context).labelLarge,
                ),
                CheckboxListTile(
                  dense: true,
                  value: filter.value?.public,
                  title: Text(l10n.public),
                  onChanged: (value) => ref
                      .read(trailFilterProvider(widget.filterId).notifier)
                      .updateFilter((filter) => filter.copyWith(public: value)),
                ),
                CheckboxListTile(
                  dense: true,
                  value: filter.value?.private,
                  title: Text(l10n.private),
                  onChanged: (value) => ref
                      .read(trailFilterProvider(widget.filterId).notifier)
                      .updateFilter(
                        (filter) => filter.copyWith(private: value),
                      ),
                ),
                CheckboxListTile(
                  dense: true,
                  value: filter.value?.shared,
                  title: Text(l10n.shared),

                  onChanged: (value) => ref
                      .read(trailFilterProvider(widget.filterId).notifier)
                      .updateFilter((filter) => filter.copyWith(shared: value)),
                ),
                const SizedBox(height: 16),

                Text(l10n.difficulty, style: TextTheme.of(context).labelLarge),
                const SizedBox(height: 8),

                WandererFilterChip<int>(
                  options: [0, 1, 2],
                  multiple: true,
                  selectedValues: f.difficulty,
                  labelBuilder: (c) {
                    switch (c) {
                      case 0:
                        return l10n.easy;
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
                        .read(trailFilterProvider(widget.filterId).notifier)
                        .updateFilter(
                          (filter) => filter.copyWith(difficulty: difficulty),
                        );
                  },
                ),
                const SizedBox(height: 16),

                Text(l10n.distance, style: TextTheme.of(context).labelLarge),
                WandererRangeSlider(
                  values: RangeValues(
                    filter.value?.distanceMin ?? 0,
                    filter.value?.distanceMax ?? 0,
                  ),

                  min: 0,
                  max: filter.value?.distanceLimit ?? 0,
                  labelsBuilder: (values) => RangeLabels(
                    formatDistance(values.start, unit: unit),
                    "${formatDistance(values.end, unit: unit)}${values.end == filter.value?.distanceLimit ? "+" : ""}",
                  ),
                  onChangeEnd: (RangeValues values) {
                    ref
                        .read(trailFilterProvider(widget.filterId).notifier)
                        .updateFilter(
                          (filter) => filter.copyWith(
                            distanceMin: values.start,
                            distanceMax: values.end,
                          ),
                        );
                  },
                ),
                const SizedBox(height: 16),

                Text(
                  l10n.elevation_gain,
                  style: TextTheme.of(context).labelLarge,
                ),
                WandererRangeSlider(
                  values: RangeValues(
                    filter.value?.elevationGainMin ?? 0,
                    filter.value?.elevationGainMax ?? 0,
                  ),

                  min: 0,
                  max: filter.value?.elevationGainLimit ?? 0,
                  labelsBuilder: (values) => RangeLabels(
                    formatElevation(values.start, unit: unit),
                    "${formatElevation(values.end, unit: unit)}${values.end == filter.value?.elevationGainLimit ? "+" : ""}",
                  ),
                  onChangeEnd: (RangeValues values) {
                    ref
                        .read(trailFilterProvider(widget.filterId).notifier)
                        .updateFilter(
                          (filter) => filter.copyWith(
                            elevationGainMin: values.start,
                            elevationGainMax: values.end,
                          ),
                        );
                  },
                ),
                const SizedBox(height: 16),

                Text(
                  l10n.elevation_loss,
                  style: TextTheme.of(context).labelLarge,
                ),
                WandererRangeSlider(
                  values: RangeValues(
                    filter.value?.elevationLossMin ?? 0,
                    filter.value?.elevationLossMax ?? 0,
                  ),
                  min: 0,
                  max: filter.value?.elevationLossLimit ?? 0,
                  labelsBuilder: (values) => RangeLabels(
                    formatElevation(values.start, unit: unit),
                    "${formatElevation(values.end, unit: unit)}${values.end == filter.value?.elevationLossLimit ? "+" : ""}",
                  ),
                  onChangeEnd: (RangeValues values) {
                    ref
                        .read(trailFilterProvider(widget.filterId).notifier)
                        .updateFilter(
                          (filter) => filter.copyWith(
                            elevationLossMin: values.start,
                            elevationLossMax: values.end,
                          ),
                        );
                  },
                ),
                const SizedBox(height: 16),

                Text(l10n.before, style: TextTheme.of(context).labelLarge),
                const SizedBox(height: 8),

                WandererDatePicker(
                  name: "start",
                  initialValue: filter.value?.startDate,
                  onChanged: (value) {
                    ref
                        .read(trailFilterProvider(widget.filterId).notifier)
                        .updateFilter(
                          (filter) => filter.copyWith(startDate: value),
                        );
                  },
                ),
                const SizedBox(height: 16),

                Text(l10n.after, style: TextTheme.of(context).labelLarge),
                const SizedBox(height: 8),

                WandererDatePicker(
                  name: "end",
                  initialValue: filter.value?.endDate,
                  onChanged: (value) {
                    ref
                        .read(trailFilterProvider(widget.filterId).notifier)
                        .updateFilter(
                          (filter) => filter.copyWith(endDate: value),
                        );
                  },
                ),
                const SizedBox(height: 16),

                Text(l10n.like_status, style: TextTheme.of(context).labelLarge),
                CheckboxListTile(
                  dense: true,
                  value: filter.value?.liked,
                  title: Text(l10n.liked),
                  onChanged: (value) => ref
                      .read(trailFilterProvider(widget.filterId).notifier)
                      .updateFilter((filter) => filter.copyWith(liked: value)),
                ),
                const SizedBox(height: 16),

                Text(
                  l10n.completion_status,
                  style: TextTheme.of(context).labelLarge,
                ),
                const SizedBox(height: 8),

                WandererRadioGroup<CompletionStatus>(
                  name: "completion_status",
                  initialValue: CompletionStatus.fromBool(
                    filter.value?.completed,
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
                        .read(trailFilterProvider(widget.filterId).notifier)
                        .updateFilter(
                          (f) => f.copyWith(completed: status?.toBool()),
                        );
                  },
                ),
                const SizedBox(height: 16),
              ],
            ),
          );
        },
        loading: () => Center(child: CircularProgressIndicator()),
        error: (err, stack) => WandererError(err: err, stack: stack),
      ),
    );
  }

  Widget _categoryAvatar(Category c) => categoryFilterAvatar(c);

  Widget _subcategoryAvatar(Subcategory s, Category? parent) =>
      subcategoryFilterAvatar(s, parent, Localizations.localeOf(context));
}

class _AuthorChip extends StatelessWidget {
  final ActorSearchResult actor;
  final VoidCallback onDeleted;

  const _AuthorChip({required this.actor, required this.onDeleted});

  @override
  Widget build(BuildContext context) {
    final displayName = actor.username.isNotEmpty
        ? actor.username
        : actor.preferredUsername;
    final handle =
        '@${actor.preferredUsername}${actor.isLocal ? "" : "@${actor.domain}"}';
    final ImageProvider avatarImage =
        actor.icon != null && actor.icon!.isNotEmpty
        ? CachedNetworkImageProvider(actor.icon!)
        : CachedNetworkImageProvider(
            'https://api.dicebear.com/7.x/initials/png?seed=$displayName',
          );

    return InputChip(
      avatar: CircleAvatar(
        backgroundImage: avatarImage,
        backgroundColor: Colors.grey.shade300,
      ),
      shape: StadiumBorder(),
      label: Text(handle),
      onDeleted: onDeleted,
    );
  }
}
