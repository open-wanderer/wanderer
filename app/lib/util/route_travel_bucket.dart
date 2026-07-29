import 'package:collection/collection.dart';
import 'package:flutter/widgets.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/models/valhalla_profile.dart';
import 'package:wanderer/util/category_icon_util.dart';
import 'package:wanderer/util/valhalla_util.dart';

/// The route planner's 5 fixed travel buckets, each carrying a hardcoded
/// Valhalla costing profile and a fixed, non-user-adjustable
/// `costing_options` payload (mirrors `route_editor.svelte` on web).
///
/// There is no manual-override UI (no sliders); `shortest` is always `false`.
enum RouteTravelBucket { hiking, bikingHybrid, bikingMountain, bikingCross, bikingRoad }

/// Resolved display/behavior data for each [RouteTravelBucket].
extension RouteTravelBucketData on RouteTravelBucket {
  /// Hardcoded English label (no l10n key for this picker).
  String get label {
    switch (this) {
      case RouteTravelBucket.hiking:
        return 'Hiking';
      case RouteTravelBucket.bikingHybrid:
        return 'Biking / Hybrid';
      case RouteTravelBucket.bikingMountain:
        return 'Biking / Mountain';
      case RouteTravelBucket.bikingCross:
        return 'Biking / Cross';
      case RouteTravelBucket.bikingRoad:
        return 'Biking / Road';
    }
  }

  /// One-line hardcoded English description.
  String get description {
    switch (this) {
      case RouteTravelBucket.hiking:
        return 'On-foot routing over trails and footpaths.';
      case RouteTravelBucket.bikingHybrid:
        return 'Balanced cycling on mixed surfaces.';
      case RouteTravelBucket.bikingMountain:
        return 'Rugged off-road cycling on trails.';
      case RouteTravelBucket.bikingCross:
        return 'Mixed-terrain cycling, gravel-ready.';
      case RouteTravelBucket.bikingRoad:
        return 'Fast cycling on paved roads.';
    }
  }

  /// The Valhalla `costing` profile string — `'pedestrian'` for hiking,
  /// `'bicycle'` for every bike bucket (only `bicycle_type` inside
  /// [costingOptions] distinguishes the 4 bike buckets from each other).
  String get costing =>
      this == RouteTravelBucket.hiking ? 'pedestrian' : 'bicycle';

  /// Fixed, immutable Valhalla `costing_options` payload for this bucket.
  /// Int/double literals match the web reference exactly (hiking
  /// `use_hills` is int `1`; bike `use_hills` is `0.5`) so serialized JSON
  /// mirrors the web verbatim.
  Map<String, Object> get costingOptions {
    switch (this) {
      case RouteTravelBucket.hiking:
        return const {
          'max_hiking_difficulty': 6,
          'walking_speed': 5.1,
          'use_hills': 1,
          'shortest': false,
        };
      case RouteTravelBucket.bikingHybrid:
        return const {
          'bicycle_type': 'Hybrid',
          'cycling_speed': 18,
          'use_roads': 0.5,
          'use_hills': 0.5,
          'avoid_bad_surfaces': 0.25,
          'shortest': false,
        };
      case RouteTravelBucket.bikingRoad:
        return const {
          'bicycle_type': 'Road',
          'cycling_speed': 25,
          'use_roads': 0.5,
          'use_hills': 0.5,
          'avoid_bad_surfaces': 0.25,
          'shortest': false,
        };
      case RouteTravelBucket.bikingCross:
        return const {
          'bicycle_type': 'Cross',
          'cycling_speed': 20,
          'use_roads': 0.5,
          'use_hills': 0.5,
          'avoid_bad_surfaces': 0.25,
          'shortest': false,
        };
      case RouteTravelBucket.bikingMountain:
        return const {
          'bicycle_type': 'Mountain',
          'cycling_speed': 16,
          'use_roads': 0.5,
          'use_hills': 0.5,
          'avoid_bad_surfaces': 0.25,
          'shortest': false,
        };
    }
  }

  /// The Valhalla-native string stored in `settings.valhalla_profile` —
  /// derived from [costing]/[costingOptions] so it can never drift from the
  /// bucket's actual Valhalla request payload. `'pedestrian'` for hiking;
  /// `'bicycle_<type>'` (lowercase `bicycle_type`) for every bike bucket.
  String get valhallaProfileKey => this == RouteTravelBucket.hiking
      ? 'pedestrian'
      : 'bicycle_${(costingOptions['bicycle_type']! as String).toLowerCase()}';

  /// This bucket's own resolved profile — the bridge between the closed
  /// picker enum and the open profile space.
  ValhallaProfile get valhallaProfile =>
      ValhallaProfile.parse(valhallaProfileKey)!;

  /// Keyword set used to resolve an operator [Category] to this bucket for
  /// icon purposes only — never used to derive the costing payload.
  List<String> get keywords {
    switch (this) {
      case RouteTravelBucket.hiking:
        return const ['hik', 'walk', 'foot'];
      case RouteTravelBucket.bikingHybrid:
        return const [
          'hybrid',
          'city',
          'commut',
          'touring',
          'trekking',
          'urban',
        ];
      case RouteTravelBucket.bikingMountain:
        return const ['mountain', 'mtb', 'downhill', 'enduro'];
      case RouteTravelBucket.bikingCross:
        return const ['cross', 'cyclocross', 'gravel', 'cx'];
      case RouteTravelBucket.bikingRoad:
        return const ['road', 'race', 'racing'];
    }
  }

  /// Fallback icon used when no operator [Category] matches [keywords] —
  /// matches `travel_profile_sheet.dart`'s pre-existing icons.
  FaIconData get fallbackIcon {
    return this == RouteTravelBucket.hiking
        ? FontAwesomeIcons.personHiking
        : FontAwesomeIcons.bicycle;
  }

  /// Small badge icon overlaid on the bike buckets' primary icon so the 4
  /// bicycle_type options stay visually distinct when no operator [Category]
  /// matches (they'd otherwise all share the same plain-bicycle
  /// [fallbackIcon]). `null` for hiking, which needs no badge.
  FaIconData? get badgeIcon {
    switch (this) {
      case RouteTravelBucket.hiking:
        return null;
      case RouteTravelBucket.bikingHybrid:
        return FontAwesomeIcons.city;
      case RouteTravelBucket.bikingRoad:
        return FontAwesomeIcons.gaugeHigh;
      case RouteTravelBucket.bikingCross:
        return FontAwesomeIcons.mound;
      case RouteTravelBucket.bikingMountain:
        return FontAwesomeIcons.mountain;
    }
  }
}

/// Resolves the leading icon for [bucket]'s picker card — shared by
/// `settings_tab.dart` and `travel_profile_sheet.dart`.
///
/// Prefers the operator (sub)category mapped to this bucket via
/// `settings.valhalla_profile` (resolved by [categorySelectionForBucket], so
/// icons and routing agree by construction), falling back to
/// [RouteTravelBucket.fallbackIcon]. Always overlays
/// [RouteTravelBucket.badgeIcon] so the 4 bike buckets stay visually
/// distinct even when no category match is found.
Widget bucketIcon(
  RouteTravelBucket bucket,
  List<Category> categories,
  List<Subcategory> subcategories, {
  double size = 20,
}) {
  final selection = categorySelectionForBucket(
    bucket,
    categories,
    subcategories,
  );

  Category? matchedCategory;
  Subcategory? matchedSubcategory;
  if (selection != null) {
    matchedCategory = categories.firstWhereOrNull(
      (c) => c.id == selection.categoryId,
    );
    if (selection.subcategoryId != null) {
      matchedSubcategory = subcategories.firstWhereOrNull(
        (s) => s.id == selection.subcategoryId,
      );
    }
  }

  final primary = matchedCategory != null || matchedSubcategory != null
      ? trailCategoryIcon(
          matchedCategory,
          subcategory: matchedSubcategory,
          size: size,
        )
      : FaIcon(bucket.fallbackIcon, size: size);

  final badge = bucket.badgeIcon;
  if (badge == null) return primary;

  return Stack(
    clipBehavior: Clip.none,
    children: [
      primary,
      Positioned(
        right: -3,
        top: -3,
        child: FaIcon(badge, size: size * 0.4),
      ),
    ],
  );
}

/// Resolves the [RouteTravelBucket] matching the current routing state —
/// drives the picker's current-selection highlight (both the entry sheet and
/// the Settings tab).
///
/// `'pedestrian'` always resolves to [RouteTravelBucket.hiking]. For
/// `'bicycle'`, matches on `costingOptions['bicycle_type']` (capitalized
/// exactly as Valhalla emits it: `Road`/`Mountain`/`Cross`/`Hybrid`),
/// defaulting to [RouteTravelBucket.bikingHybrid] when [costingOptions] is
/// `null` or `bicycle_type` is absent/unrecognized — a bicycle costing with
/// no usable `bicycle_type` genuinely is Valhalla's Hybrid default.
///
/// Returns `null` for any other costing (e.g. `auto`, from a category mapped
/// to a costing outside the 5 picker buckets). Callers must treat that as
/// "nothing highlighted" and must NOT substitute a default bucket — doing so
/// would mis-seed the planner for e.g. a car trail.
RouteTravelBucket? bucketForState(
  String travelProfile,
  Map<String, dynamic>? costingOptions,
) {
  if (travelProfile == 'pedestrian') return RouteTravelBucket.hiking;
  if (travelProfile != 'bicycle') return null;

  switch (costingOptions?['bicycle_type']) {
    case 'Road':
      return RouteTravelBucket.bikingRoad;
    case 'Mountain':
      return RouteTravelBucket.bikingMountain;
    case 'Cross':
      return RouteTravelBucket.bikingCross;
    default:
      return RouteTravelBucket.bikingHybrid;
  }
}
