import 'package:collection/collection.dart';
import 'package:maplibre/maplibre.dart' show Geographic, SphericalGreatCircle;
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/models/subcategory_preference.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/valhalla_profile.dart';
import 'package:wanderer/util/category_preference_sort.dart';
import 'package:wanderer/models/route_travel_bucket.dart';

/// Valhalla's own default `walking_speed`/`cycling_speed` (km/h, `Hybrid`
/// bike type) — used when a segment has no Valhalla-resolved time and no
/// explicit speed is set, mirroring what Valhalla applies server-side.
const _defaultWalkingSpeedKmh = 5.1;
const _defaultCyclingSpeedKmh = 18.0;

/// Estimates a segment's travel time, in seconds, from its geometry when no
/// Valhalla-resolved duration is available. Sums the great-circle length of
/// [polyline] and divides by the `walking_speed`/`cycling_speed` (km/h) on
/// [costingOptions] for [travelProfile], falling back to Valhalla's own
/// defaults when that key is absent.
double estimateSegmentDurationSeconds({
  required List<Geographic> polyline,
  required String travelProfile,
  Map<String, dynamic>? costingOptions,
}) {
  if (polyline.length < 2) return 0;

  var distanceMeters = 0.0;
  for (var i = 1; i < polyline.length; i++) {
    distanceMeters += SphericalGreatCircle(
      polyline[i - 1],
    ).distanceTo(polyline[i]);
  }

  final speedKmh = travelProfile == 'bicycle'
      ? ((costingOptions?['cycling_speed'] as num?)?.toDouble() ??
            _defaultCyclingSpeedKmh)
      : ((costingOptions?['walking_speed'] as num?)?.toDouble() ??
            _defaultWalkingSpeedKmh);
  if (speedKmh <= 0) return 0;

  final speedMetersPerSecond = speedKmh * 1000 / 3600;
  return distanceMeters / speedMetersPerSecond;
}

/// Resolves the Valhalla routing profile for a trail's category/subcategory
/// selection from the operator-configured `settings.valhalla_profile`.
///
/// This is data-driven end to end — no category *name* is ever inspected, so
/// translated and custom category names resolve identically.
///
/// Resolution order:
/// 1. **Subcategory tier** — [subcategoryId]'s own profile, but only when that
///    subcategory still *exists* in [subcategories]. The relation is declared
///    `cascadeDelete: false`, so deleting a subcategory leaves a dangling id on
///    every trail that referenced it; such a trail falls through to the tiers
///    below rather than resolving against a subcategory that is gone.
///
///    Deliberately NOT filtered by the user's subcategory *visibility*
///    preferences: those are a display setting driving which entries appear in
///    the category picker and settings screens. Hiding "MTB" to declutter a
///    picker must not silently change how existing MTB trails route.
/// 2. **Category tier** — [category]'s own profile.
/// 3. **Bicycle-hybrid tier** — when [category] is "biking-shaped" (at least
///    one of its subcategories resolves to a `bicycle` costing) but nothing
///    above resolved, defaults to `bicycle_hybrid`. This tier is deliberately
///    bicycle-only: Valhalla's 4 bike variants share one `bicycle` costing and
///    need a sane `bicycle_type`. It never fires for `auto`/`truck`/etc.,
///    which already resolve at the category tier.
/// 4. Otherwise `null` — callers apply their own default.
ValhallaProfile? resolveValhallaProfile({
  required Category? category,
  required String? subcategoryId,
  required List<Subcategory> subcategories,
}) {
  if (subcategoryId != null && subcategoryId.isNotEmpty) {
    final subcategory = subcategories.firstWhereOrNull(
      (s) => s.id == subcategoryId,
    );
    final profile = subcategory?.valhallaProfile;
    if (profile != null) return profile;
  }

  final categoryProfile = category?.valhallaProfile;
  if (categoryProfile != null) return categoryProfile;

  if (category != null) {
    final bikingShaped = subcategories
        .where((s) => s.category == category.id)
        .any((s) => s.valhallaProfile?.costing == 'bicycle');
    if (bikingShaped) return RouteTravelBucket.bikingHybrid.valhallaProfile;
  }

  return null;
}

/// The Valhalla `costing` string for a trail's category/subcategory selection.
///
/// Thin wrapper over [resolveValhallaProfile], defaulting to `'pedestrian'`
/// when nothing resolves. Note this returns only the plain `costing` value
/// used directly in `/valhalla/route` request bodies — the finer
/// `bicycle_type` lives on the resolved [ValhallaProfile], available via
/// [resolveValhallaProfile] for callers that also build `costing_options`.
String costingForCategory({
  required Category? category,
  required String? subcategoryId,
  required List<Subcategory> subcategories,
}) =>
    resolveValhallaProfile(
      category: category,
      subcategoryId: subcategoryId,
      subcategories: subcategories,
    )?.costing ??
    'pedestrian';

/// The Valhalla `costing` string for an existing [trail]'s own category and
/// subcategory selection.
///
/// Convenience wrapper over [costingForCategory] for the common "I have a
/// saved trail" case — every such caller otherwise repeats the identical
/// `category: trail.expand?.category, subcategoryId: trail.subcategory` pair.
/// Accepts a nullable [trail] (a trail-less recording session has none) and
/// resolves to the `'pedestrian'` default in that case.
///
/// Use [costingForCategory]/[resolveValhallaProfile] directly when the
/// category/subcategory do NOT come from a saved trail — most importantly the
/// route-planner edit hand-off, which must read the *live* form selection
/// rather than the trail's saved (and possibly stale) relation.
String costingForTrail(
  Trail? trail, {
  required List<Subcategory> subcategories,
}) => costingForCategory(
  category: trail?.expand?.category,
  subcategoryId: trail?.subcategory,
  subcategories: subcategories,
);

/// Narrows an open [ValhallaProfile] back to one of the route planner's 5
/// picker buckets, for icon/pre-fill purposes.
///
/// Returns `null` for `auto` and any other non-bucket costing — callers must
/// treat that as "no bucket to highlight", never as a reason to substitute a
/// default bucket.
RouteTravelBucket? bucketForProfile(ValhallaProfile? profile) {
  if (profile == null) return null;
  return RouteTravelBucket.values.firstWhereOrNull(
    (b) => b.valhallaProfile == profile,
  );
}

/// A category (and optionally subcategory) chosen to represent a travel
/// bucket — the pre-fill result of [categorySelectionForBucket].
class TravelBucketCategorySelection {
  final String categoryId;
  final String? subcategoryId;

  const TravelBucketCategorySelection({
    required this.categoryId,
    this.subcategoryId,
  });
}

/// Inverse of [resolveValhallaProfile]: given a bucket the user picked in the
/// route planner, finds the operator (sub)category that represents it.
///
/// Prefers a subcategory match (returning both ids) over a category-only
/// match. Returns `null` when nothing in the operator's loaded
/// (sub)categories maps to [bucket] — including for a category mapped to a
/// costing outside the 5 buckets (e.g. `auto`).
///
/// This is where the user's subcategory *visibility* preferences apply: a
/// subcategory hidden via [subcategoryPrefs] is never auto-assigned to a trail,
/// falling back to a matching category instead. That is the correct home for
/// the preference — it governs what we may newly assign on the user's behalf,
/// not how an already-tagged trail routes (see [resolveValhallaProfile]).
TravelBucketCategorySelection? categorySelectionForBucket(
  RouteTravelBucket bucket,
  List<Category> categories,
  List<Subcategory> subcategories, {
  List<SubcategoryPreference> subcategoryPrefs = const [],
}) {
  final target = bucket.valhallaProfile;

  final subcategory = subcategories.firstWhereOrNull(
    (s) =>
        s.valhallaProfile == target &&
        subcategoryVisible(s.id, subcategoryPrefs),
  );
  if (subcategory != null) {
    return TravelBucketCategorySelection(
      categoryId: subcategory.category,
      subcategoryId: subcategory.id,
    );
  }

  final category = categories.firstWhereOrNull(
    (c) => c.valhallaProfile == target,
  );
  if (category != null) {
    return TravelBucketCategorySelection(categoryId: category.id);
  }

  return null;
}
