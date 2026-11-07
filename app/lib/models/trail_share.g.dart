// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail_share.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_TrailShareExpand _$TrailShareExpandFromJson(Map<String, dynamic> json) =>
    _TrailShareExpand(
      actor: Actor.fromJson(json['actor'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$TrailShareExpandToJson(_TrailShareExpand instance) =>
    <String, dynamic>{'actor': instance.actor};

_TrailShare _$TrailShareFromJson(Map<String, dynamic> json) => _TrailShare(
  id: json['id'] as String?,
  actor: json['actor'] as String,
  trail: json['trail'] as String,
  permission: $enumDecode(_$TrailPermissionEnumMap, json['permission']),
  expand: json['expand'] == null
      ? null
      : TrailShareExpand.fromJson(json['expand'] as Map<String, dynamic>),
);

Map<String, dynamic> _$TrailShareToJson(_TrailShare instance) =>
    <String, dynamic>{
      'id': instance.id,
      'actor': instance.actor,
      'trail': instance.trail,
      'permission': _$TrailPermissionEnumMap[instance.permission]!,
      'expand': instance.expand,
    };

const _$TrailPermissionEnumMap = {
  TrailPermission.view: 'view',
  TrailPermission.edit: 'edit',
};
