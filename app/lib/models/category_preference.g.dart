// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'category_preference.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_CategoryPreference _$CategoryPreferenceFromJson(Map<String, dynamic> json) =>
    _CategoryPreference(
      id: json['id'] as String?,
      user: json['user'] as String,
      category: json['category'] as String,
      visible: json['visible'] as bool?,
      priority: (json['priority'] as num?)?.toInt(),
    );

Map<String, dynamic> _$CategoryPreferenceToJson(_CategoryPreference instance) =>
    <String, dynamic>{
      'id': instance.id,
      'user': instance.user,
      'category': instance.category,
      'visible': instance.visible,
      'priority': instance.priority,
    };
