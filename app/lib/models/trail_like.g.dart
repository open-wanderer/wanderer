// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail_like.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_TrailLikeExpand _$TrailLikeExpandFromJson(Map<String, dynamic> json) =>
    _TrailLikeExpand(
      actor: Actor.fromJson(json['actor'] as Map<String, dynamic>),
      trail: Trail.fromJson(json['trail'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$TrailLikeExpandToJson(_TrailLikeExpand instance) =>
    <String, dynamic>{'actor': instance.actor, 'trail': instance.trail};

_TrailLike _$TrailLikeFromJson(Map<String, dynamic> json) => _TrailLike(
  id: json['id'] as String?,
  actor: json['actor'] as String,
  trail: json['trail'] as String,
  expand: json['expand'] == null
      ? null
      : TrailLikeExpand.fromJson(json['expand'] as Map<String, dynamic>),
);

Map<String, dynamic> _$TrailLikeToJson(_TrailLike instance) =>
    <String, dynamic>{
      'id': instance.id,
      'actor': instance.actor,
      'trail': instance.trail,
      'expand': instance.expand,
    };
