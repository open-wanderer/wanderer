import 'package:collection/collection.dart';
import 'package:wanderer/models/category.dart';

/// Derives the Valhalla costing string from a trail category name.
///
/// Returns `'bicycle'` when [category] (case-insensitive) contains `'bike'`,
/// `'cycling'`, or `'bicycle'`; otherwise returns `'pedestrian'`.
///
/// Shared by [launchNavigation] (online path) and [downloadTrail]
/// (cache-write path) so the two costing derivations can never diverge.
String costingForCategory(String? category) {
  final lower = (category ?? '').toLowerCase();
  if (lower.contains('bike') ||
      lower.contains('cycling') ||
      lower.contains('bicycle')) {
    return 'bicycle';
  }
  return 'pedestrian';
}

/// Inverse of [costingForCategory] (D-08): given a Valhalla travel profile
/// (`'bicycle'` or `'pedestrian'`) and the operator's loaded category list,
/// returns the id of the first category whose name/short name matches the
/// corresponding heuristic, or `null` when no category matches.
///
/// Mirrors [costingForCategory]'s own check order (`'bike'` → `'cycling'` →
/// `'bicycle'`) for symmetry (Pitfall 5) — categories are fully
/// operator-managed runtime content with no reserved hiking/biking id, so a
/// substring match against `name`/`shortName` is the only available
/// heuristic. Never throws on an empty [categories] list; degrades
/// gracefully to `null` (no pre-fill) when nothing matches (A1 — expected,
/// not a bug).
String? categoryForTravelProfile(
  String travelProfile,
  List<Category> categories,
) {
  final wantsBike = travelProfile == 'bicycle';
  bool matches(Category c) {
    final name = c.name.toLowerCase();
    final short = (c.shortName ?? '').toLowerCase();
    final hay = '$name $short';
    return wantsBike
        ? (hay.contains('bike') ||
              hay.contains('cycling') ||
              hay.contains('bicycle'))
        : (hay.contains('hik') || hay.contains('walk') || hay.contains('foot'));
  }

  return categories.firstWhereOrNull(matches)?.id;
}

/// Sibling of [categoryForTravelProfile]: given a bike-bucket keyword set
/// (`RouteTravelBucket.keywords`, `route_travel_bucket.dart`) and the
/// operator's loaded category list, returns the first [Category] whose
/// name/short name (case-insensitive) contains any keyword — for ICON
/// RESOLUTION ONLY. Never throws on an empty [categories] list; degrades
/// gracefully to `null` (caller falls back to a hardcoded FontAwesome icon)
/// when nothing matches.
///
/// A bucket's `bicycle_type` costing payload is fixed regardless of what
/// categories the operator happens to have (RESEARCH.md Q2 key finding) — it
/// is carried on `RouteTravelBucket` itself and is NEVER synthesized from a
/// matched `Category` here.
Category? categoryForBikeBucket(
  List<String> keywords,
  List<Category> categories,
) {
  bool matches(Category c) {
    final name = c.name.toLowerCase();
    final short = (c.shortName ?? '').toLowerCase();
    final hay = '$name $short';
    return keywords.any(hay.contains);
  }

  return categories.firstWhereOrNull(matches);
}
