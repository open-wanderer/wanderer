// ignore_for_file: invalid_annotation_target

import 'dart:ui';

import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/valhalla_profile.dart';

part 'subcategory.freezed.dart';
part 'subcategory.g.dart';

@freezed
abstract class Subcategory with _$Subcategory {
  @JsonSerializable(explicitToJson: true)
  const factory Subcategory({
    required String id,
    required String category,
    required String name,
    @JsonKey(name: 'short_name') String? shortName,
    String? icon,
    @JsonKey(name: 'badge_icon') String? badgeIcon,
    Map<String, CategoryTranslation>? translations,
    Map<String, dynamic>? settings,
  }) = _Subcategory;

  factory Subcategory.fromJson(Map<String, dynamic> json) =>
      _$SubcategoryFromJson(json);
}

/// Resolves the locale-aware display name for a [Subcategory].
///
/// Fallback chain: active locale → English (`'en'`) → raw [name].
extension SubcategoryDisplay on Subcategory {
  String displayName(Locale? locale) =>
      translations?[locale?.languageCode]?.name ??
      translations?['en']?.name ??
      name;
}

/// Reads the operator-configured Valhalla routing profile from this
/// subcategory's `settings.valhalla_profile`.
///
/// Returns `null` when unset or unparseable — [ValhallaProfile.parse] is
/// total, so an operator-edited settings blob can never make this throw.
extension SubcategoryValhallaProfile on Subcategory {
  ValhallaProfile? get valhallaProfile =>
      ValhallaProfile.parse(settings?['valhalla_profile']);
}
