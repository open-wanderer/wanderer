import 'package:collection/collection.dart';
import 'package:flutter/widgets.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/util/category_icon_util.dart';
import 'package:wanderer/util/valhalla_util.dart';

/// The route planner's 5 fixed travel buckets (CONTEXT: "ONE picker, five
/// options"), each carrying a hardcoded Valhalla costing profile + a fixed,
/// non-user-adjustable `costing_options` payload (RESEARCH.md Q1 — values
/// copied verbatim from `web/src/lib/components/trail/route_editor.svelte`).
///
/// There is deliberately no manual-override UI (no sliders) — every value
/// below is fixed at implementation time (CONTEXT "Explicitly excluded").
/// `shortest` is always `false`.
enum RouteTravelBucket { hiking, bikingHybrid, bikingMountain, bikingCross, bikingRoad }

/// Resolved display/behavior data for each [RouteTravelBucket].
extension RouteTravelBucketData on RouteTravelBucket {
  /// Hardcoded English label (CONTEXT: matches the planner's other
  /// hardcoded strings; no l10n key required for this picker).
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

  /// Fixed, immutable Valhalla `costing_options` payload for this bucket
  /// (RESEARCH.md Q1). Int/double literals are written exactly as the web
  /// reference emits them (hiking `use_hills` is int `1`; bike `use_hills`
  /// is `0.5`) so JSON serialization mirrors the web verbatim.
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

  /// Keyword set used to resolve an operator [Category] to this bucket for
  /// icon purposes only (RESEARCH.md Q2) — never used to derive the costing
  /// payload, which is always carried on the bucket itself.
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
  /// bicycle_type options stay visually discernible even when no operator
  /// [Category] matches (all 4 share the identical [fallbackIcon] otherwise
  /// — a plain bicycle glyph, indistinguishable from one another). `null`
  /// for hiking, which has only one option and needs no badge.
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

/// Resolves the leading icon for [bucket]'s picker card — the single shared
/// implementation for both `settings_tab.dart` and `travel_profile_sheet.dart`
/// (previously duplicated verbatim between the two, quick-260717-t7q code
/// review IN-02).
///
/// Prefers a matching operator [Category] icon (via [trailCategoryIcon]),
/// falling back to [RouteTravelBucket.fallbackIcon] when no category
/// matches (RESEARCH.md Q2) — icon resolution only, never the costing
/// payload. Always overlays [RouteTravelBucket.badgeIcon] (mirroring
/// `trailCategoryIcon`'s own subcategory-badge Stack shape) so the 4 bike
/// buckets stay discernible from each other regardless of whether a
/// category match was found — the fallback icon alone is identical
/// (a plain bicycle) across all 4.
Widget bucketIcon(
  RouteTravelBucket bucket,
  List<Category> categories, {
  double size = 20,
}) {
  Category? matched;
  if (bucket == RouteTravelBucket.hiking) {
    final id = categoryForTravelProfile('pedestrian', categories);
    matched = id == null
        ? null
        : categories.firstWhereOrNull((c) => c.id == id);
  } else {
    matched = categoryForBikeBucket(bucket.keywords, categories);
  }

  final primary = matched != null
      ? trailCategoryIcon(matched, size: size)
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
/// `null` or `bicycle_type` is absent/unrecognized.
RouteTravelBucket bucketForState(
  String travelProfile,
  Map<String, dynamic>? costingOptions,
) {
  if (travelProfile == 'pedestrian') return RouteTravelBucket.hiking;

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
