// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'global_search_models.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_ListSearchResult _$ListSearchResultFromJson(Map<String, dynamic> json) =>
    _ListSearchResult(
      id: json['id'] as String,
      author: json['author'] as String,
      authorName: json['author_name'] as String,
      authorAvatar: json['author_avatar'] as String,
      avatar: json['avatar'] as String?,
      name: json['name'] as String,
      description: json['description'] as String,
      elevationGain: (json['elevation_gain'] as num).toDouble(),
      elevationLoss: (json['elevation_loss'] as num).toDouble(),
      distance: (json['distance'] as num).toDouble(),
      duration: (json['duration'] as num).toDouble(),
      domain: json['domain'] as String?,
      public: json['public'] as bool,
      trails: (json['trails'] as num).toInt(),
      shares: (json['shares'] as List<dynamic>?)
          ?.map((e) => e as String)
          .toList(),
      iri: json['iri'] as String?,
    );

Map<String, dynamic> _$ListSearchResultToJson(_ListSearchResult instance) =>
    <String, dynamic>{
      'id': instance.id,
      'author': instance.author,
      'author_name': instance.authorName,
      'author_avatar': instance.authorAvatar,
      'avatar': instance.avatar,
      'name': instance.name,
      'description': instance.description,
      'elevation_gain': instance.elevationGain,
      'elevation_loss': instance.elevationLoss,
      'distance': instance.distance,
      'duration': instance.duration,
      'domain': instance.domain,
      'public': instance.public,
      'trails': instance.trails,
      'shares': instance.shares,
      'iri': instance.iri,
    };

_ActorSearchResult _$ActorSearchResultFromJson(Map<String, dynamic> json) =>
    _ActorSearchResult(
      id: json['id'] as String? ?? '',
      username: json['username'] as String? ?? '',
      preferredUsername: json['preferred_username'] as String? ?? '',
      isLocal: json['is_local'] as bool? ?? false,
      domain: json['domain'] as String? ?? '',
      icon: json['icon'] as String?,
      iri: json['iri'] as String? ?? '',
    );

Map<String, dynamic> _$ActorSearchResultToJson(_ActorSearchResult instance) =>
    <String, dynamic>{
      'id': instance.id,
      'username': instance.username,
      'preferred_username': instance.preferredUsername,
      'is_local': instance.isLocal,
      'domain': instance.domain,
      'icon': instance.icon,
      'iri': instance.iri,
    };
