// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'actor.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_Actor _$ActorFromJson(Map<String, dynamic> json) => _Actor(
  id: json['id'] as String,
  collectionId: json['collectionId'] as String,
  collectionName: json['collectionName'] as String,
  created: DateTime.parse(json['created'] as String),
  updated: DateTime.parse(json['updated'] as String),
  username: json['username'] as String,
  preferredUsername: json['preferred_username'] as String,
  domain: json['domain'] as String?,
  summary: json['summary'] as String?,
  published: json['published'] as String?,
  followerCount: (json['followerCount'] as num?)?.toInt(),
  followingCount: (json['followingCount'] as num?)?.toInt(),
  iri: json['iri'] as String,
  inbox: json['inbox'] as String,
  outbox: json['outbox'] as String?,
  icon: json['icon'] as String?,
  followers: json['followers'] as String?,
  following: json['following'] as String?,
  isLocal: json['isLocal'] as bool? ?? false,
  publicKey: json['public_key'] as String,
  lastFetched: json['last_fetched'] as String,
  user: json['user'] as String,
);

Map<String, dynamic> _$ActorToJson(_Actor instance) => <String, dynamic>{
  'id': instance.id,
  'collectionId': instance.collectionId,
  'collectionName': instance.collectionName,
  'created': instance.created.toIso8601String(),
  'updated': instance.updated.toIso8601String(),
  'username': instance.username,
  'preferred_username': instance.preferredUsername,
  'domain': instance.domain,
  'summary': instance.summary,
  'published': instance.published,
  'followerCount': instance.followerCount,
  'followingCount': instance.followingCount,
  'iri': instance.iri,
  'inbox': instance.inbox,
  'outbox': instance.outbox,
  'icon': instance.icon,
  'followers': instance.followers,
  'following': instance.following,
  'isLocal': instance.isLocal,
  'public_key': instance.publicKey,
  'last_fetched': instance.lastFetched,
  'user': instance.user,
};
