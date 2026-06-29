// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'subcategory_preference.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_SubcategoryPreference _$SubcategoryPreferenceFromJson(
  Map<String, dynamic> json,
) => _SubcategoryPreference(
  id: json['id'] as String?,
  user: json['user'] as String,
  subcategory: json['subcategory'] as String,
  visible: json['visible'] as bool?,
  priority: (json['priority'] as num?)?.toInt(),
);

Map<String, dynamic> _$SubcategoryPreferenceToJson(
  _SubcategoryPreference instance,
) => <String, dynamic>{
  'id': instance.id,
  'user': instance.user,
  'subcategory': instance.subcategory,
  'visible': instance.visible,
  'priority': instance.priority,
};
