// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'region_geometry.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_RegionGeometry _$RegionGeometryFromJson(Map<String, dynamic> json) =>
    _RegionGeometry(
      path: json['path'] as String,
      polygon: json['polygon'] as Map<String, dynamic>,
      bbox: (json['bbox'] as List<dynamic>)
          .map((e) => (e as num).toDouble())
          .toList(),
    );

Map<String, dynamic> _$RegionGeometryToJson(_RegionGeometry instance) =>
    <String, dynamic>{
      'path': instance.path,
      'polygon': instance.polygon,
      'bbox': instance.bbox,
    };
