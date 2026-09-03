import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/theme/icons.dart';

// ── Shared icon-data resolution ───────────────────────────────────────────────

FaIconData? _categoryIconData(Category? category) {
  if (category == null) return null;
  final raw = (category.icon ?? '').trim();
  final key = raw.startsWith('fa-') ? raw.substring(3) : raw;
  return fontAwesomeIconsMap[key];
}

FaIconData? _subcategoryIconData(Subcategory subcategory, Category? parent) {
  final raw =
      ((subcategory.icon?.trim().isNotEmpty ?? false)
              ? subcategory.icon!
              : (parent?.icon ?? ''))
          .trim();
  final key = raw.startsWith('fa-') ? raw.substring(3) : raw;
  return fontAwesomeIconsMap[key];
}

FaIconData? _subcategoryBadgeIconData(Subcategory subcategory) {
  final raw = (subcategory.badgeIcon ?? '').trim();
  final key = raw.startsWith('fa-') ? raw.substring(3) : raw;
  return fontAwesomeIconsMap[key];
}

// ── General-purpose inline icon (trail cards, etc.) ──────────────────────────

/// Icon for inline use (trail list items, detail pages). Uses the subcategory
/// icon when [subcategory] is provided, falling back to the category icon, then
/// to [FontAwesomeIcons.route].
Widget trailCategoryIcon(
  Category? category, {
  Subcategory? subcategory,
  double size = 12,
  Color? color,
}) {
  final iconData = subcategory != null
      ? _subcategoryIconData(subcategory, category)
      : _categoryIconData(category);
  final primary = FaIcon(
    iconData ?? FontAwesomeIcons.route,
    size: size,
    color: color,
  );

  if (subcategory == null) return primary;

  final badgeRaw = (subcategory.badgeIcon ?? '').trim();
  final badgeKey = badgeRaw.startsWith('fa-')
      ? badgeRaw.substring(3)
      : badgeRaw;
  final badgeFa = badgeKey.isNotEmpty ? fontAwesomeIconsMap[badgeKey] : null;
  if (badgeFa == null) return primary;

  return Stack(
    clipBehavior: Clip.none,
    children: [
      primary,
      Positioned(
        right: -3,
        top: -3,
        child: FaIcon(badgeFa, size: size * 0.4, color: color),
      ),
    ],
  );
}

// ── Filter chip avatars ───────────────────────────────────────────────────────

/// 16px avatar for a category [FilterChip]. Falls back to [Icons.category]
/// when the icon name is not in [fontAwesomeIconsMap].
Widget categoryFilterAvatar(Category c, {double size = 16, Color? color}) {
  final iconData = _categoryIconData(c);
  return iconData != null
      ? FaIcon(iconData, size: size, color: color)
      : Icon(Icons.category, size: size, color: color);
}

Widget? subcategoryBadgeAvatar(Subcategory c, {Color? color, double size = 8}) {
  final iconData = _subcategoryBadgeIconData(c);
  return iconData != null ? FaIcon(iconData, size: size, color: color) : null;
}

/// 16px avatar for a subcategory [FilterChip] with an optional FA badge icon
/// overlay at the top-right (mirrors the web's `displaySubcategoryIcon` +
/// badge logic). [Clip.none] keeps the badge from clipping.
Widget subcategoryFilterAvatar(
  Subcategory s,
  Category? parent,
  Locale locale, {
  Color? color,
}) {
  final primary = _subcategoryIconData(s, parent);

  Widget? badgeIconWidget;
  final badgeFa = subcategoryBadgeAvatar(s, color: color);
  if (badgeFa != null) {
    badgeIconWidget = Positioned(right: -6, top: -4, child: badgeFa);
  }

  return Stack(
    clipBehavior: Clip.none,
    children: [
      primary != null
          ? FaIcon(primary, size: 16, color: color)
          : Icon(Icons.category, size: 16, color: color),
      ?badgeIconWidget,
    ],
  );
}
