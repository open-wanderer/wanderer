// ignore_for_file: invalid_annotation_target

import 'package:freezed_annotation/freezed_annotation.dart';

part 'subcategory_preference.freezed.dart';
part 'subcategory_preference.g.dart';

@freezed
abstract class SubcategoryPreference with _$SubcategoryPreference {
  const factory SubcategoryPreference({
    String? id,
    required String user,
    required String subcategory,
    bool? visible,
    int? priority,
  }) = _SubcategoryPreference;

  factory SubcategoryPreference.fromJson(Map<String, dynamic> json) =>
      _$SubcategoryPreferenceFromJson(json);
}
