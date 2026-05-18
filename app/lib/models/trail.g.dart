// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_TrailExpand _$TrailExpandFromJson(Map<String, dynamic> json) => _TrailExpand(
  tags: (json['tags'] as List<dynamic>?)
      ?.map((e) => Tag.fromJson(e as Map<String, dynamic>))
      .toList(),
  category: json['category'] == null
      ? null
      : Category.fromJson(json['category'] as Map<String, dynamic>),
  waypointsViaTrail: (json['waypoints_via_trail'] as List<dynamic>?)
      ?.map((e) => Waypoint.fromJson(e as Map<String, dynamic>))
      .toList(),
  summitLogsViaTrail: (json['summit_logs_via_trail'] as List<dynamic>?)
      ?.map((e) => SummitLog.fromJson(e as Map<String, dynamic>))
      .toList(),
  author: json['author'] == null
      ? null
      : Actor.fromJson(json['author'] as Map<String, dynamic>),
  commentsViaTrail: (json['comments_via_trail'] as List<dynamic>?)
      ?.map((e) => Comment.fromJson(e as Map<String, dynamic>))
      .toList(),
  gpxData: json['gpx_data'] as String?,
  trailShareViaTrail: (json['trail_share_via_trail'] as List<dynamic>?)
      ?.map((e) => TrailShare.fromJson(e as Map<String, dynamic>))
      .toList(),
  trailLikeViaTrail: (json['trail_like_via_trail'] as List<dynamic>?)
      ?.map((e) => TrailLike.fromJson(e as Map<String, dynamic>))
      .toList(),
);

Map<String, dynamic> _$TrailExpandToJson(_TrailExpand instance) =>
    <String, dynamic>{
      'tags': instance.tags,
      'category': instance.category,
      'waypoints_via_trail': instance.waypointsViaTrail,
      'summit_logs_via_trail': instance.summitLogsViaTrail,
      'author': instance.author,
      'comments_via_trail': instance.commentsViaTrail,
      'gpx_data': instance.gpxData,
      'trail_share_via_trail': instance.trailShareViaTrail,
      'trail_like_via_trail': instance.trailLikeViaTrail,
    };

_Trail _$TrailFromJson(Map<String, dynamic> json) => _Trail(
  id: json['id'] as String,
  collectionId: json['collectionId'] as String? ?? 'trails',
  name: json['name'] as String,
  location: json['location'] as String?,
  date: json['date'] == null ? null : DateTime.parse(json['date'] as String),
  public: json['public'] as bool? ?? false,
  distance: (json['distance'] as num?)?.toDouble() ?? 0,
  elevationGain: (json['elevation_gain'] as num?)?.toDouble() ?? 0,
  elevationLoss: (json['elevation_loss'] as num?)?.toDouble() ?? 0,
  duration: (json['duration'] as num?)?.toDouble() ?? 0,
  difficulty:
      $enumDecodeNullable(_$TrailDifficultyEnumMap, json['difficulty']) ??
      TrailDifficulty.easy,
  lat: (json['lat'] as num?)?.toDouble(),
  lon: (json['lon'] as num?)?.toDouble(),
  thumbnail: (json['thumbnail'] as num?)?.toInt() ?? 0,
  photos:
      (json['photos'] as List<dynamic>?)?.map((e) => e as String).toList() ??
      const [],
  gpx: json['gpx'] as String?,
  created: json['created'] as String?,
  updated: json['updated'] as String?,
  category: json['category'] as String?,
  tags:
      (json['tags'] as List<dynamic>?)?.map((e) => e as String).toList() ??
      const [],
  polyline: json['polyline'] as String?,
  domain: json['domain'] as String?,
  iri: json['iri'] as String?,
  likeCount: (json['like_count'] as num?)?.toInt() ?? 0,
  expand: json['expand'] == null
      ? null
      : TrailExpand.fromJson(json['expand'] as Map<String, dynamic>),
  description: json['description'] as String? ?? "",
  author: json['author'] as String? ?? "000000000000000",
  isOffline: json['isOffline'] as bool? ?? false,
  localPhotos:
      (json['localPhotos'] as List<dynamic>?)
          ?.map((e) => e as String)
          .toList() ??
      const [],
  pmTiles:
      (json['pmTiles'] as List<dynamic>?)?.map((e) => e as String).toList() ??
      const [],
);

Map<String, dynamic> _$TrailToJson(_Trail instance) => <String, dynamic>{
  'id': instance.id,
  'collectionId': instance.collectionId,
  'name': instance.name,
  'location': instance.location,
  'date': instance.date?.toIso8601String(),
  'public': instance.public,
  'distance': instance.distance,
  'elevation_gain': instance.elevationGain,
  'elevation_loss': instance.elevationLoss,
  'duration': instance.duration,
  'difficulty': _$TrailDifficultyEnumMap[instance.difficulty]!,
  'lat': instance.lat,
  'lon': instance.lon,
  'thumbnail': instance.thumbnail,
  'photos': instance.photos,
  'gpx': instance.gpx,
  'created': instance.created,
  'updated': instance.updated,
  'category': instance.category,
  'tags': instance.tags,
  'polyline': instance.polyline,
  'domain': instance.domain,
  'iri': instance.iri,
  'like_count': instance.likeCount,
  'expand': instance.expand,
  'description': instance.description,
  'author': instance.author,
  'isOffline': instance.isOffline,
  'localPhotos': instance.localPhotos,
  'pmTiles': instance.pmTiles,
};

const _$TrailDifficultyEnumMap = {
  TrailDifficulty.easy: 'easy',
  TrailDifficulty.moderate: 'moderate',
  TrailDifficulty.difficult: 'difficult',
};

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

_TrailFilterValues _$TrailFilterValuesFromJson(Map<String, dynamic> json) =>
    _TrailFilterValues(
      minDistance: (json['min_distance'] as num).toDouble(),
      maxDistance: (json['max_distance'] as num).toDouble(),
      minElevationGain: (json['min_elevation_gain'] as num).toDouble(),
      maxElevationGain: (json['max_elevation_gain'] as num).toDouble(),
      minElevationLoss: (json['min_elevation_loss'] as num).toDouble(),
      maxElevationLoss: (json['max_elevation_loss'] as num).toDouble(),
      minDuration: (json['min_duration'] as num).toDouble(),
      maxDuration: (json['max_duration'] as num).toDouble(),
    );

Map<String, dynamic> _$TrailFilterValuesToJson(_TrailFilterValues instance) =>
    <String, dynamic>{
      'min_distance': instance.minDistance,
      'max_distance': instance.maxDistance,
      'min_elevation_gain': instance.minElevationGain,
      'max_elevation_gain': instance.maxElevationGain,
      'min_elevation_loss': instance.minElevationLoss,
      'max_elevation_loss': instance.maxElevationLoss,
      'min_duration': instance.minDuration,
      'max_duration': instance.maxDuration,
    };
