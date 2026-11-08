// ignore_for_file: invalid_annotation_target

import 'package:freezed_annotation/freezed_annotation.dart';

part 'category_preference.freezed.dart';
part 'category_preference.g.dart';

@freezed
abstract class CategoryPreference with _$CategoryPreference {
  const factory CategoryPreference({
    String? id,
    required String user,
    required String category,
    bool? visible,
    int? priority,
  }) = _CategoryPreference;

  factory CategoryPreference.fromJson(Map<String, dynamic> json) =>
      _$CategoryPreferenceFromJson(json);
}
