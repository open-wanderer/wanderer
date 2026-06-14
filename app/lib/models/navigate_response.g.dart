// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'navigate_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_NavigateManeuver _$NavigateManeuverFromJson(Map<String, dynamic> json) =>
    _NavigateManeuver(
      instruction: json['instruction'] as String,
      length: (json['length'] as num).toDouble(),
      beginShapeIndex: (json['begin_shape_index'] as num).toInt(),
      bearing: (json['bearing'] as num?)?.toDouble() ?? 0.0,
      type: (json['type'] as num?)?.toInt() ?? 0,
    );

Map<String, dynamic> _$NavigateManeuverToJson(_NavigateManeuver instance) =>
    <String, dynamic>{
      'instruction': instance.instruction,
      'length': instance.length,
      'begin_shape_index': instance.beginShapeIndex,
      'bearing': instance.bearing,
      'type': instance.type,
    };

_NavigateResponse _$NavigateResponseFromJson(Map<String, dynamic> json) =>
    _NavigateResponse(
      maneuvers: (json['maneuvers'] as List<dynamic>)
          .map((e) => NavigateManeuver.fromJson(e as Map<String, dynamic>))
          .toList(),
      shape: (json['shape'] as List<dynamic>)
          .map(
            (e) =>
                (e as List<dynamic>).map((e) => (e as num).toDouble()).toList(),
          )
          .toList(),
    );

Map<String, dynamic> _$NavigateResponseToJson(_NavigateResponse instance) =>
    <String, dynamic>{'maneuvers': instance.maneuvers, 'shape': instance.shape};
