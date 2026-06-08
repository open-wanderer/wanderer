// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'list.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_ListExpand _$ListExpandFromJson(Map<String, dynamic> json) => _ListExpand(
  trails: (json['trails'] as List<dynamic>?)
      ?.map((e) => Trail.fromJson(e as Map<String, dynamic>))
      .toList(),
  author: json['author'] == null
      ? null
      : Actor.fromJson(json['author'] as Map<String, dynamic>),
);

Map<String, dynamic> _$ListExpandToJson(_ListExpand instance) =>
    <String, dynamic>{'trails': instance.trails, 'author': instance.author};

_WandererList _$WandererListFromJson(Map<String, dynamic> json) =>
    _WandererList(
      id: json['id'] as String,
      collectionId: json['collectionId'] as String? ?? 'lists',
      name: json['name'] as String,
      public: json['public'] as bool? ?? false,
      description: json['description'] as String?,
      elevationGain: (json['elevation_gain'] as num?)?.toDouble(),
      elevationLoss: (json['elevation_loss'] as num?)?.toDouble(),
      distance: (json['distance'] as num?)?.toDouble(),
      duration: (json['duration'] as num?)?.toDouble(),
      avatar: json['avatar'] as String?,
      trails:
          (json['trails'] as List<dynamic>?)
              ?.map((e) => e as String)
              .toList() ??
          const [],
      iri: json['iri'] as String?,
      expand: json['expand'] == null
          ? null
          : ListExpand.fromJson(json['expand'] as Map<String, dynamic>),
      created: json['created'] == null
          ? null
          : DateTime.parse(json['created'] as String),
      updated: json['updated'] == null
          ? null
          : DateTime.parse(json['updated'] as String),
      author: json['author'] as String? ?? '000000000000000',
    );

Map<String, dynamic> _$WandererListToJson(_WandererList instance) =>
    <String, dynamic>{
      'id': instance.id,
      'collectionId': instance.collectionId,
      'name': instance.name,
      'public': instance.public,
      'description': instance.description,
      'elevation_gain': instance.elevationGain,
      'elevation_loss': instance.elevationLoss,
      'distance': instance.distance,
      'duration': instance.duration,
      'avatar': instance.avatar,
      'trails': instance.trails,
      'iri': instance.iri,
      'expand': instance.expand,
      'created': instance.created?.toIso8601String(),
      'updated': instance.updated?.toIso8601String(),
      'author': instance.author,
    };
