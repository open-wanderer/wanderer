// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'waypoint.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_Waypoint _$WaypointFromJson(Map<String, dynamic> json) => _Waypoint(
  id: json['id'] as String?,
  name: json['name'] as String? ?? "",
  description: json['description'] as String? ?? "",
  lat: (json['lat'] as num).toDouble(),
  lon: (json['lon'] as num).toDouble(),
  distanceFromStart: (json['distanceFromStart'] as num?)?.toDouble(),
  icon: json['icon'] == null
      ? FontAwesomeIcons.circle
      : const FaIconDataConverter().fromJson(
          json['icon'] as Map<String, dynamic>,
        ),
  photos:
      (json['photos'] as List<dynamic>?)?.map((e) => e as String).toList() ??
      const [],
  author: json['author'] as String? ?? "000000000000000",
  trail: json['trail'] as String?,
);

Map<String, dynamic> _$WaypointToJson(_Waypoint instance) => <String, dynamic>{
  'id': instance.id,
  'name': instance.name,
  'description': instance.description,
  'lat': instance.lat,
  'lon': instance.lon,
  'distanceFromStart': instance.distanceFromStart,
  'icon': const FaIconDataConverter().toJson(instance.icon),
  'photos': instance.photos,
  'author': instance.author,
  'trail': instance.trail,
};
