// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'comment.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_CommentExpand _$CommentExpandFromJson(Map<String, dynamic> json) =>
    _CommentExpand(
      author: Actor.fromJson(json['author'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$CommentExpandToJson(_CommentExpand instance) =>
    <String, dynamic>{'author': instance.author};

_Comment _$CommentFromJson(Map<String, dynamic> json) => _Comment(
  id: json['id'] as String,
  text: json['text'] as String,
  author: json['author'] as String,
  trail: json['trail'] as String,
  created: DateTime.parse(json['created'] as String),
  updated: DateTime.parse(json['updated'] as String),
  iri: json['iri'] as String?,
  expand: json['expand'] == null
      ? null
      : CommentExpand.fromJson(json['expand'] as Map<String, dynamic>),
);

Map<String, dynamic> _$CommentToJson(_Comment instance) => <String, dynamic>{
  'id': instance.id,
  'text': instance.text,
  'author': instance.author,
  'trail': instance.trail,
  'created': instance.created.toIso8601String(),
  'updated': instance.updated.toIso8601String(),
  'iri': instance.iri,
  'expand': instance.expand,
};
