import 'dart:ui';

import 'package:collection/collection.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/category_preference.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/models/subcategory_preference.dart';

/// Sort helpers + visibility resolution for the settings categories screens.
///
/// Ported from `web/src/lib/util/category_util.ts`. These are
/// pure functions with no Riverpod dependency, preserving the Phase-10
/// read-only-provider boundary — callers pass already-fetched data.
///
/// Sort order (Pattern 4): prioritized items (`priority != null && priority > 0`)
/// come first, ascending by priority; non-prioritized items follow; ties break
/// on the lowercased locale-resolved display name. `String.compareTo` on
/// lowercased names stands in for `localeCompare(sensitivity: "base")` — no
/// intl-collation package is added (A4).

/// Returns [categories] sorted by preference priority then locale name.
List<Category> sortedCategoriesByPreference(
  List<Category> categories,
  List<CategoryPreference> prefs,
  Locale? locale,
) {
  int? priorityOf(String id) =>
      prefs.where((p) => p.category == id).map((p) => p.priority).firstOrNull;

  final list = [...categories];
  list.sort((a, b) {
    final ap = priorityOf(a.id);
    final bp = priorityOf(b.id);
    final aPrioritized = ap != null && ap > 0;
    final bPrioritized = bp != null && bp > 0;
    if (aPrioritized && bPrioritized) return ap.compareTo(bp);
    if (aPrioritized) return -1;
    if (bPrioritized) return 1;
    return a
        .displayName(locale)
        .toLowerCase()
        .compareTo(b.displayName(locale).toLowerCase());
  });
  return list;
}

/// Returns [subs] sorted by preference priority then locale name.
List<Subcategory> sortedSubcategoriesByPreference(
  List<Subcategory> subs,
  List<SubcategoryPreference> prefs,
  Locale? locale,
) {
  int? priorityOf(String id) => prefs
      .where((p) => p.subcategory == id)
      .map((p) => p.priority)
      .firstOrNull;

  final list = [...subs];
  list.sort((a, b) {
    final ap = priorityOf(a.id);
    final bp = priorityOf(b.id);
    final aPrioritized = ap != null && ap > 0;
    final bPrioritized = bp != null && bp > 0;
    if (aPrioritized && bPrioritized) return ap.compareTo(bp);
    if (aPrioritized) return -1;
    if (bPrioritized) return 1;
    return a
        .displayName(locale)
        .toLowerCase()
        .compareTo(b.displayName(locale).toLowerCase());
  });
  return list;
}

/// [categories] sorted by preference and filtered to visible ones — the
/// "what should the picker show, in order" query. Optionally keeps
/// [keepVisibleId] in the result even if hidden, so a picker doesn't drop the
/// currently-selected category out from under the user.
List<Category> visibleSortedCategories(
  List<Category> categories,
  List<CategoryPreference> prefs,
  Locale? locale, {
  String? keepVisibleId,
}) => sortedCategoriesByPreference(
  categories,
  prefs,
  locale,
).where((c) => categoryVisible(c.id, prefs) || c.id == keepVisibleId).toList();

/// [subs] sorted by preference and filtered to visible ones. Optionally keeps
/// [keepVisibleId] in the result even if hidden, so a picker doesn't drop the
/// currently-selected subcategory out from under the user.
List<Subcategory> visibleSortedSubcategories(
  List<Subcategory> subs,
  List<SubcategoryPreference> prefs,
  Locale? locale, {
  String? keepVisibleId,
}) => sortedSubcategoriesByPreference(subs, prefs, locale)
    .where((s) => subcategoryVisible(s.id, prefs) || s.id == keepVisibleId)
    .toList();

/// A category is visible unless a preference row exists with `visible == false`.
/// No-preference and a null `visible` both mean visible (Pitfall 4).
bool categoryVisible(String categoryId, List<CategoryPreference> prefs) =>
    prefs
        .where((p) => p.category == categoryId)
        .map((p) => p.visible)
        .firstOrNull !=
    false;

/// A subcategory is visible unless a preference row exists with
/// `visible == false`. No-preference and a null `visible` both mean visible.
bool subcategoryVisible(
  String subcategoryId,
  List<SubcategoryPreference> prefs,
) =>
    prefs
        .where((p) => p.subcategory == subcategoryId)
        .map((p) => p.visible)
        .firstOrNull !=
    false;
