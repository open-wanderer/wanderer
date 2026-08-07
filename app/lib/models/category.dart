// ignore_for_file: invalid_annotation_target

import 'dart:ui';

import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/valhalla_profile.dart';

part 'category.freezed.dart';
part 'category.g.dart';

@freezed
abstract class CategoryTranslation with _$CategoryTranslation {
  const factory CategoryTranslation({
    String? name,
    @JsonKey(name: 'short_name') String? shortName,
  }) = _CategoryTranslation;

  factory CategoryTranslation.fromJson(Map<String, dynamic> json) =>
      _$CategoryTranslationFromJson(json);
}

@freezed
abstract class Category with _$Category {
  @JsonSerializable(explicitToJson: true)
  const factory Category({
    required String id,
    required String name,
    @JsonKey(name: 'short_name') String? shortName,
    String? icon,
    Map<String, CategoryTranslation>? translations,
    Map<String, dynamic>? settings,
  }) = _Category;

  factory Category.fromJson(Map<String, dynamic> json) =>
      _$CategoryFromJson(json);
}

/// Resolves the locale-aware display name for a [Category].
///
/// Fallback chain: active locale → English (`'en'`) → raw [name].
extension CategoryDisplay on Category {
  String displayName(Locale? locale) =>
      translations?[locale?.languageCode]?.name ??
      translations?['en']?.name ??
      name;
}

/// Reads the operator-configured Valhalla routing profile from this
/// category's `settings.valhalla_profile`.
///
/// Returns `null` when unset or unparseable — [ValhallaProfile.parse] is
/// total, so an operator-edited settings blob can never make this throw.
extension CategoryValhallaProfile on Category {
  ValhallaProfile? get valhallaProfile =>
      ValhallaProfile.parse(settings?['valhalla_profile']);
}
