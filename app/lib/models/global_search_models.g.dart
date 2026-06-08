// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'global_search_models.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_GeoLocation _$GeoLocationFromJson(Map<String, dynamic> json) => _GeoLocation(
  lat: (json['lat'] as num).toDouble(),
  lng: (json['lng'] as num).toDouble(),
);

Map<String, dynamic> _$GeoLocationToJson(_GeoLocation instance) =>
    <String, dynamic>{'lat': instance.lat, 'lng': instance.lng};

_TrailSearchResult _$TrailSearchResultFromJson(
  Map<String, dynamic> json,
) => _TrailSearchResult(
  id: json['id'] as String,
  collectionId: json['collectionId'] as String? ?? 'trails',
  author: json['author'] as String,
  authorName: json['author_name'] as String,
  authorAvatar: json['author_avatar'] as String,
  name: json['name'] as String,
  description: json['description'] as String,
  location: json['location'] as String,
  distance: (json['distance'] as num).toDouble(),
  elevationGain: (json['elevation_gain'] as num).toDouble(),
  elevationLoss: (json['elevation_loss'] as num).toDouble(),
  duration: (json['duration'] as num).toDouble(),
  difficulty: (json['difficulty'] as num).toInt(),
  category: json['category'] as String,
  completed: json['completed'] as bool,
  date: (json['date'] as num).toInt(),
  created: (json['created'] as num).toInt(),
  public: json['public'] as bool,
  thumbnail: json['thumbnail'] as String,
  polyline: json['polyline'] as String?,
  likes: (json['likes'] as List<dynamic>?)?.map((e) => e as String).toList(),
  likeCount: (json['like_count'] as num).toInt(),
  shares: (json['shares'] as List<dynamic>?)?.map((e) => e as String).toList(),
  tags: (json['tags'] as List<dynamic>?)?.map((e) => e as String).toList(),
  domain: json['domain'] as String?,
  iri: json['iri'] as String?,
  gpx: json['gpx'] as String,
  geo: GeoLocation.fromJson(json['_geo'] as Map<String, dynamic>),
);

Map<String, dynamic> _$TrailSearchResultToJson(_TrailSearchResult instance) =>
    <String, dynamic>{
      'id': instance.id,
      'collectionId': instance.collectionId,
      'author': instance.author,
      'author_name': instance.authorName,
      'author_avatar': instance.authorAvatar,
      'name': instance.name,
      'description': instance.description,
      'location': instance.location,
      'distance': instance.distance,
      'elevation_gain': instance.elevationGain,
      'elevation_loss': instance.elevationLoss,
      'duration': instance.duration,
      'difficulty': instance.difficulty,
      'category': instance.category,
      'completed': instance.completed,
      'date': instance.date,
      'created': instance.created,
      'public': instance.public,
      'thumbnail': instance.thumbnail,
      'polyline': instance.polyline,
      'likes': instance.likes,
      'like_count': instance.likeCount,
      'shares': instance.shares,
      'tags': instance.tags,
      'domain': instance.domain,
      'iri': instance.iri,
      'gpx': instance.gpx,
      '_geo': instance.geo,
    };

_ListSearchResult _$ListSearchResultFromJson(Map<String, dynamic> json) =>
    _ListSearchResult(
      id: json['id'] as String,
      collectionId: json['collectionId'] as String? ?? 'lists',
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
      'collectionId': instance.collectionId,
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
