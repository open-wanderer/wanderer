// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'waypoint.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_Waypoint _$WaypointFromJson(Map<String, dynamic> json) => _Waypoint(
  id: json['id'] as String,
  collectionId: json['collectionId'] as String? ?? 'waypoints',
  collectionName: json['collectionName'] as String? ?? 'waypoints',
  name: json['name'] as String? ?? "",
  description: json['description'] as String? ?? "",
  lat: (json['lat'] as num).toDouble(),
  lon: (json['lon'] as num).toDouble(),
  distanceFromStart: (json['distance_from_start'] as num?)?.toDouble(),
  icon: json['icon'] == null
      ? FontAwesomeIcons.circle
      : const FaIconDataConverter().fromJson(json['icon'] as String),
  photos:
      (json['photos'] as List<dynamic>?)?.map((e) => e as String).toList() ??
      const [],
  author: json['author'] as String? ?? "000000000000000",
  trail: json['trail'] as String?,
  created: DateTime.parse(json['created'] as String),
  updated: DateTime.parse(json['updated'] as String),
);

Map<String, dynamic> _$WaypointToJson(_Waypoint instance) => <String, dynamic>{
  'id': instance.id,
  'collectionId': instance.collectionId,
  'collectionName': instance.collectionName,
  'name': instance.name,
  'description': instance.description,
  'lat': instance.lat,
  'lon': instance.lon,
  'distance_from_start': instance.distanceFromStart,
  'icon': const FaIconDataConverter().toJson(instance.icon),
  'photos': instance.photos,
  'author': instance.author,
  'trail': instance.trail,
  'created': instance.created.toIso8601String(),
  'updated': instance.updated.toIso8601String(),
};
