// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'subcategory.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_Subcategory _$SubcategoryFromJson(Map<String, dynamic> json) => _Subcategory(
  id: json['id'] as String,
  category: json['category'] as String,
  name: json['name'] as String,
  shortName: json['short_name'] as String?,
  icon: json['icon'] as String?,
  badgeIcon: json['badge_icon'] as String?,
  translations: (json['translations'] as Map<String, dynamic>?)?.map(
    (k, e) =>
        MapEntry(k, CategoryTranslation.fromJson(e as Map<String, dynamic>)),
  ),
);

Map<String, dynamic> _$SubcategoryToJson(
  _Subcategory instance,
) => <String, dynamic>{
  'id': instance.id,
  'category': instance.category,
  'name': instance.name,
  'short_name': instance.shortName,
  'icon': instance.icon,
  'badge_icon': instance.badgeIcon,
  'translations': instance.translations?.map((k, e) => MapEntry(k, e.toJson())),
};
