// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'server_instance.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_ServerInstance _$ServerInstanceFromJson(
  Map<String, dynamic> json,
) => _ServerInstance(
  name: json['name'] as String,
  url: json['url'] as String,
  description: json['description'] as String,
  image: json['image'] as String,
  region:
      (json['region'] as List<dynamic>?)?.map((e) => e as String).toList() ??
      const [],
  language:
      (json['language'] as List<dynamic>?)?.map((e) => e as String).toList() ??
      const [],
  category:
      (json['category'] as List<dynamic>?)?.map((e) => e as String).toList() ??
      const [],
);

Map<String, dynamic> _$ServerInstanceToJson(_ServerInstance instance) =>
    <String, dynamic>{
      'name': instance.name,
      'url': instance.url,
      'description': instance.description,
      'image': instance.image,
      'region': instance.region,
      'language': instance.language,
      'category': instance.category,
    };
