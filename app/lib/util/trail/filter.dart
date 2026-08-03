/// Client-side equivalent of the server's Meilisearch filter.
///
/// Deliberately dependency-light: no Riverpod, no ObjectBox, no Flutter
/// widgets — it is a pure `List<Trail>` → `List<Trail>` transform, unit-tested
/// in `trail_filter_util_test.dart` against plain `Trail(...)` fixtures.
///
/// Used wherever a list of trails is held locally rather than fetched already
/// filtered: the library screen, and the local half of the own-trails list in
/// `ProfileTrailsNotifier`.
library;

import 'package:wanderer/models/trail.dart';

/// Applies [filter] to [trails], then sorts by [TrailFilter.sort]/
/// [TrailFilter.sortOrder].
///
/// Mirrors `TrailFilter.toFilterText()` (`models/trail.dart`) clause for
/// clause, so a locally-held trail is kept or dropped by the same rules
/// Meilisearch would apply to the server-side half of the same list. Two
/// deliberate exceptions, both because [Trail] carries no matching field:
/// - `completed` is skipped.
/// - `tags`, `author`, visibility (`public`/`private`/`shared`), `liked` and
///   the `_geoRadius` clause are skipped — they are server-side concerns.
///
/// Range clauses follow `toFilterText()`'s convention: the minimum always
/// applies, the maximum only when it is below its limit. `buildDefaultTrailFilter`
/// constructs every bounded axis with `max == limit`, so an untouched slider
/// imposes no upper bound (and cannot silently drop a trail whose distance
/// exceeds device-derived offline bounds).
///
/// `Trail.difficulty` is a `TrailDifficulty` enum whose index maps to the
/// filter's 0/1/2. `Trail.category`/`Trail.subcategory` hold record IDs.
List<Trail> applyTrailFilter(List<Trail> trails, TrailFilter filter) {
  // Category/subcategory form ONE OR-group, matching toFilterText(): a
  // category whose subcategories are also selected is represented by those
  // subcategories rather than by itself, so selecting a subcategory narrows
  // its parent instead of being widened back out by it.
  final selectedSubcategoryParentIds = filter.subcategory
      .map((s) => s.category)
      .toSet();
  final categoryIdsWithoutSub = filter.category
      .where((c) => !selectedSubcategoryParentIds.contains(c.id))
      .map((c) => c.id)
      .toSet();
  final subcategoryIds = filter.subcategory.map((s) => s.id).toSet();
  final hasCategoryConstraint =
      categoryIdsWithoutSub.isNotEmpty || subcategoryIds.isNotEmpty;

  List<Trail> result = trails.where((trail) {
    // Difficulty
    if (!filter.difficulty.contains(trail.difficulty.index)) return false;

    // Distance / elevation ranges
    if (trail.distance < filter.distanceMin) return false;
    if (filter.distanceMax < filter.distanceLimit &&
        trail.distance > filter.distanceMax) {
      return false;
    }

    if (trail.elevationGain < filter.elevationGainMin) return false;
    if (filter.elevationGainMax < filter.elevationGainLimit &&
        trail.elevationGain > filter.elevationGainMax) {
      return false;
    }

    if (trail.elevationLoss < filter.elevationLossMin) return false;
    if (filter.elevationLossMax < filter.elevationLossLimit &&
        trail.elevationLoss > filter.elevationLossMax) {
      return false;
    }

    // Category / subcategory
    if (hasCategoryConstraint) {
      final matchesCategory =
          trail.category != null &&
          categoryIdsWithoutSub.contains(trail.category);
      final matchesSubcategory =
          trail.subcategory != null &&
          subcategoryIds.contains(trail.subcategory);
      if (!matchesCategory && !matchesSubcategory) return false;
    }

    // Date range
    final trailDate = trail.date;
    if (trailDate != null) {
      if (filter.startDate != null && trailDate.isBefore(filter.startDate!)) {
        return false;
      }
      if (filter.endDate != null && trailDate.isAfter(filter.endDate!)) {
        return false;
      }
    }

    return true;
  }).toList();

  // Sort
  result.sort((a, b) {
    int cmp;
    switch (filter.sort) {
      case TrailFilterSort.name:
        cmp = a.name.compareTo(b.name);
        break;
      case TrailFilterSort.date:
        final aDate = a.date ?? DateTime(0);
        final bDate = b.date ?? DateTime(0);
        cmp = aDate.compareTo(bDate);
        break;
      case TrailFilterSort.difficulty:
        cmp = a.difficulty.index.compareTo(b.difficulty.index);
        break;
      case TrailFilterSort.distance:
        cmp = a.distance.compareTo(b.distance);
        break;
      case TrailFilterSort.duration:
        cmp = a.duration.compareTo(b.duration);
        break;
      case TrailFilterSort.elevation_gain:
        cmp = a.elevationGain.compareTo(b.elevationGain);
        break;
      case TrailFilterSort.elevation_loss:
        cmp = a.elevationLoss.compareTo(b.elevationLoss);
        break;
      case TrailFilterSort.created:
        cmp = a.created.compareTo(b.created);
        break;
      case TrailFilterSort.like_count:
        cmp = a.likeCount.compareTo(b.likeCount);
        break;
    }
    return filter.sortOrder == SortOrder.asc ? cmp : -cmp;
  });

  return result;
}
