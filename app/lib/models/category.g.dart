// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'category.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_CategoryTranslation _$CategoryTranslationFromJson(Map<String, dynamic> json) =>
    _CategoryTranslation(
      name: json['name'] as String?,
      shortName: json['short_name'] as String?,
    );

Map<String, dynamic> _$CategoryTranslationToJson(
  _CategoryTranslation instance,
) => <String, dynamic>{'name': instance.name, 'short_name': instance.shortName};

_Category _$CategoryFromJson(Map<String, dynamic> json) => _Category(
  id: json['id'] as String,
  name: json['name'] as String,
  shortName: json['short_name'] as String?,
  icon: json['icon'] as String?,
  translations: (json['translations'] as Map<String, dynamic>?)?.map(
    (k, e) =>
        MapEntry(k, CategoryTranslation.fromJson(e as Map<String, dynamic>)),
  ),
  settings: json['settings'] as Map<String, dynamic>?,
);

Map<String, dynamic> _$CategoryToJson(_Category instance) => <String, dynamic>{
  'id': instance.id,
  'name': instance.name,
  'short_name': instance.shortName,
  'icon': instance.icon,
  'translations': instance.translations?.map((k, e) => MapEntry(k, e.toJson())),
  'settings': instance.settings,
};
