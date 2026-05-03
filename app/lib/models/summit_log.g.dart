// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'summit_log.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_SummitLog _$SummitLogFromJson(Map<String, dynamic> json) => _SummitLog(
  id: json['id'] as String?,
  date: json['date'] as String,
  text: json['text'] as String? ?? "",
  gpx: json['gpx'] as String?,
  photos:
      (json['photos'] as List<dynamic>?)?.map((e) => e as String).toList() ??
      const [],
  distance: (json['distance'] as num?)?.toDouble(),
  elevationGain: (json['elevation_gain'] as num?)?.toDouble(),
  elevationLoss: (json['elevation_loss'] as num?)?.toDouble(),
  duration: (json['duration'] as num?)?.toDouble(),
  author: json['author'] as String? ?? "000000000000000",
  trail: json['trail'] as String?,
  iri: json['iri'] as String?,
  created: json['created'] as String?,
  expand: json['expand'] == null
      ? null
      : SummitLogExpand.fromJson(json['expand'] as Map<String, dynamic>),
);

Map<String, dynamic> _$SummitLogToJson(_SummitLog instance) =>
    <String, dynamic>{
      'id': instance.id,
      'date': instance.date,
      'text': instance.text,
      'gpx': instance.gpx,
      'photos': instance.photos,
      'distance': instance.distance,
      'elevation_gain': instance.elevationGain,
      'elevation_loss': instance.elevationLoss,
      'duration': instance.duration,
      'author': instance.author,
      'trail': instance.trail,
      'iri': instance.iri,
      'created': instance.created,
      'expand': instance.expand,
    };

_SummitLogExpand _$SummitLogExpandFromJson(Map<String, dynamic> json) =>
    _SummitLogExpand(
      gpxData: json['gpx_data'] as String?,
      trail: json['trail'] == null
          ? null
          : Trail.fromJson(json['trail'] as Map<String, dynamic>),
      author: json['author'] == null
          ? null
          : Actor.fromJson(json['author'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$SummitLogExpandToJson(_SummitLogExpand instance) =>
    <String, dynamic>{
      'gpx_data': instance.gpxData,
      'trail': instance.trail,
      'author': instance.author,
    };
