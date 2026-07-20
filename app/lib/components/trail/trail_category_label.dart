import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/util/category_icon_util.dart';

/// Inline icon + locale-resolved name for a category/subcategory pair.
///
/// Looks up [categoryId] and [subcategoryId] from [categoryProvider] and
/// [subcategoryProvider]. Returns [SizedBox.shrink] when the category is not
/// yet loaded.
class TrailCategoryLabel extends ConsumerWidget {
  final String categoryId;
  final String? subcategoryId;
  final double iconSize;
  final double fontSize;
  final Color? color;

  const TrailCategoryLabel({
    super.key,
    required this.categoryId,
    this.subcategoryId,
    this.iconSize = 12,
    this.fontSize = 12,
    this.color,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final locale = Localizations.localeOf(context);
    final categories = ref.watch(categoryProvider).value ?? [];
    final subcategories = ref.watch(subcategoryProvider);

    final Category? category = categories.firstWhereOrNull(
      (c) => c.id == categoryId,
    );
    final Subcategory? subcategory = subcategoryId != null
        ? subcategories.firstWhereOrNull((s) => s.id == subcategoryId)
        : null;

    if (category == null) return const SizedBox.shrink();

    final label = subcategory != null
        ? '${category.displayName(locale)} / ${subcategory.displayName(locale)}'
        : category.displayName(locale);
    final labelColor = color ?? Theme.of(context).colorScheme.onSurface;

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        trailCategoryIcon(
          category,
          subcategory: subcategory,
          size: iconSize,
          color: color ?? Colors.blueGrey,
        ),
        const SizedBox(width: 4),
        Text(
          label,
          style: TextStyle(fontSize: fontSize, color: labelColor),
        ),
      ],
    );
  }
}
