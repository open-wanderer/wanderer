import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/util/icon_util.dart';

/// Returns a 16px FontAwesome icon avatar for a [Category], falling back to
/// [Icons.category] when the icon name is unknown. No explicit color is set so
/// the chip's foreground color drives theming.
Widget categoryFilterAvatar(Category c) {
  final raw = (c.icon ?? '').trim();
  final key = raw.startsWith('fa-') ? raw.substring(3) : raw;
  final faData = fontAwesomeIconsMap[key];
  return faData != null
      ? FaIcon(faData, size: 16)
      : const Icon(Icons.category, size: 16);
}

/// Returns a subcategory chip avatar: a 16px primary icon with an optional
/// badge overlay at the bottom-right.
///
/// Primary icon: the subcategory's own icon when set, otherwise the parent
/// category's icon (mirrors the web's `displaySubcategoryIcon`).
///
/// Badge (mirrors web `displaySubcategoryShortBadge`):
/// - If `s.badgeIcon` maps to a known FA icon → renders a 10px [FaIcon].
///
/// [Clip.none] prevents the badge from being clipped by chip avatar bounds.
Widget subcategoryFilterAvatar(
  BuildContext context,
  Subcategory s,
  Category? parent,
  Locale locale,
) {
  final primaryRaw =
      ((s.icon?.trim().isNotEmpty ?? false) ? s.icon! : (parent?.icon ?? ''))
          .trim();
  final primaryKey = primaryRaw.startsWith('fa-')
      ? primaryRaw.substring(3)
      : primaryRaw;
  final primary = fontAwesomeIconsMap[primaryKey];

  Widget? badgeIconWidget;
  final badgeRaw = (s.badgeIcon ?? '').trim();
  final badgeKey = badgeRaw.startsWith('fa-')
      ? badgeRaw.substring(3)
      : badgeRaw;
  final badgeFa = badgeKey.isNotEmpty ? fontAwesomeIconsMap[badgeKey] : null;
  if (badgeFa != null) {
    badgeIconWidget = Positioned(
      right: -6,
      top: -4,
      child: FaIcon(badgeFa, size: 8),
    );
  }

  return Stack(
    clipBehavior: Clip.none,
    children: [
      primary != null
          ? FaIcon(primary, size: 16)
          : const Icon(Icons.category, size: 16),
      ?badgeIconWidget,
    ],
  );
}
