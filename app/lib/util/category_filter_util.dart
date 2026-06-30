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
/// - Otherwise → text abbreviation from [subcategoryShortBadge].
///
/// [Clip.none] prevents the badge from being clipped by chip avatar bounds.
Widget subcategoryFilterAvatar(Subcategory s, Category? parent, Locale locale) {
  final primaryRaw =
      ((s.icon?.trim().isNotEmpty ?? false) ? s.icon! : (parent?.icon ?? ''))
          .trim();
  final primaryKey =
      primaryRaw.startsWith('fa-') ? primaryRaw.substring(3) : primaryRaw;
  final primary = fontAwesomeIconsMap[primaryKey];

  Widget? badgeWidget;
  final badgeRaw = (s.badgeIcon ?? '').trim();
  final badgeKey =
      badgeRaw.startsWith('fa-') ? badgeRaw.substring(3) : badgeRaw;
  final badgeFa = badgeKey.isNotEmpty ? fontAwesomeIconsMap[badgeKey] : null;
  if (badgeFa != null) {
    badgeWidget = Positioned(
      right: -2,
      bottom: -2,
      child: FaIcon(badgeFa, size: 10),
    );
  } else {
    final badgeText = subcategoryShortBadge(s, locale);
    if (badgeText.isNotEmpty) {
      badgeWidget = Positioned(
        right: -4,
        bottom: -5,
        child: Text(
          badgeText,
          style: const TextStyle(
            fontSize: 7,
            fontWeight: FontWeight.bold,
            height: 1,
          ),
        ),
      );
    }
  }

  return Stack(
    clipBehavior: Clip.none,
    children: [
      primary != null
          ? FaIcon(primary, size: 16)
          : const Icon(Icons.category, size: 16),
      ?badgeWidget,
    ],
  );
}

/// Computes the short badge text for a subcategory chip, mirroring the web's
/// `displaySubcategoryShortBadge` function:
/// 1. Use `short_name` as-is (uppercased) when set.
/// 2. If display name ≤ 5 chars → return it uppercased.
/// 3. If multi-word → join the first letter of each word (max 5).
/// 4. Otherwise → first 4 characters uppercased.
String subcategoryShortBadge(Subcategory s, Locale locale) {
  final rawShort = (s.shortName ?? '').trim();
  if (rawShort.isNotEmpty) return rawShort.toUpperCase();

  final label = s.displayName(locale).trim();
  if (label.isEmpty) return '';
  if (label.length <= 5) return label.toUpperCase();

  final words =
      RegExp(r'[\p{L}\p{N}]+', unicode: true).allMatches(label).toList();
  if (words.length > 1) {
    return words.map((m) => m.group(0)![0]).take(5).join('').toUpperCase();
  }
  return label.substring(0, 4).toUpperCase();
}
