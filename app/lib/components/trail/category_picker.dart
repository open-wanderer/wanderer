import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/components/base/wanderer_select.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/provider/category_preference_provider.dart';
import 'package:wanderer/provider/subcategory_preference_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/util/category_icon_util.dart';
import 'package:wanderer/util/category_preference_sort.dart';

/// A resolved category/subcategory pair produced from a [CategoryPicker] value.
class CategorySelection {
  final String category;
  final String subcategory;

  const CategorySelection({required this.category, required this.subcategory});
}

/// Single-choice picker listing every category, each immediately followed by
/// its subcategories. Mirrors the web `category_picker`:
///
/// * form values are prefixed `category:<id>` / `subcategory:<id>`,
/// * a subcategory is shown as its `Category / Subcategory` path with the
///   subcategory icon + badge.
///
/// Wraps a plain [WandererSelect], so the chosen value is read back via
/// `formKey.currentState!.value['<name>']`. Resolve it into ids with
/// [CategoryPicker.resolve].
class CategoryPicker extends ConsumerWidget {
  final String name;
  final String? label;
  final String? placeholder;

  /// Currently selected category id (used to seed the initial value).
  final String? category;

  /// Currently selected subcategory id — takes precedence over [category] when
  /// seeding the initial value.
  final String? subcategory;

  const CategoryPicker({
    super.key,
    this.name = 'category',
    this.label,
    this.placeholder,
    this.category,
    this.subcategory,
  });

  /// Resolves a prefixed picker [value] into its category/subcategory ids. For
  /// a `subcategory:` value the parent category is looked up from
  /// [subcategories]. Returns `null` for an empty or unrecognised value.
  static CategorySelection? resolve(
    String? value,
    List<Subcategory> subcategories,
  ) {
    if (value == null) return null;

    if (value.startsWith('subcategory:')) {
      final id = value.substring('subcategory:'.length);
      final sub = subcategories.firstWhereOrNull((s) => s.id == id);
      if (sub == null) return null;
      return CategorySelection(category: sub.category, subcategory: sub.id);
    }

    if (value.startsWith('category:')) {
      return CategorySelection(
        category: value.substring('category:'.length),
        subcategory: '',
      );
    }

    return null;
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final locale = Localizations.localeOf(context);
    final List<Category> categories = ref.watch(categoryProvider).value ?? [];
    final List<Subcategory> subcategories = ref.watch(subcategoryProvider);
    final categoryPrefs = ref.watch(categoryPreferenceProvider).value ?? [];
    final subcategoryPrefs =
        ref.watch(subcategoryPreferenceProvider).value ?? [];

    // Order and filter by the user's category preferences (priority first, then
    // locale name), keeping the currently-selected items visible even if a
    // preference hides them. Mirrors the web `designSelectableCategories` +
    // `sortedSubcategoriesByPreference`.
    final orderedCategories = visibleSortedCategories(
      categories,
      categoryPrefs,
      locale,
      keepVisibleId: category,
    );

    final items = <SelectItem<String>>[];
    for (final cat in orderedCategories) {
      items.add(
        SelectItem(
          value: 'category:${cat.id}',
          label: cat.displayName(locale),
          icon: trailCategoryIcon(cat, size: 16),
        ),
      );

      final orderedSubs = visibleSortedSubcategories(
        subcategories.where((s) => s.category == cat.id).toList(),
        subcategoryPrefs,
        locale,
        keepVisibleId: subcategory,
      );

      for (final sub in orderedSubs) {
        items.add(
          SelectItem(
            value: 'subcategory:${sub.id}',
            label: '${cat.displayName(locale)} / ${sub.displayName(locale)}',
            icon: trailCategoryIcon(cat, subcategory: sub, size: 16),
          ),
        );
      }
    }

    final initialValue = (subcategory?.isNotEmpty ?? false)
        ? 'subcategory:$subcategory'
        : (category?.isNotEmpty ?? false)
        ? 'category:$category'
        : null;

    return WandererSelect<String>(
      name: name,
      label: label,
      placeholder: placeholder,
      initialValue: initialValue,
      items: items,
    );
  }
}
