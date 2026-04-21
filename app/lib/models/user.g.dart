// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'user.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_User _$UserFromJson(Map<String, dynamic> json) => _User(
  id: json['id'] as String,
  username: json['username'] as String,
  email: json['email'] as String,
  emailVisibility: json['emailVisibility'] as bool,
  verified: json['verified'] as bool,
  created: json['created'] as String,
  updated: json['updated'] as String,
  avatar: json['avatar'] as String?,
  expand: json['expand'] == null
      ? null
      : UserExpand.fromJson(json['expand'] as Map<String, dynamic>),
);

Map<String, dynamic> _$UserToJson(_User instance) => <String, dynamic>{
  'id': instance.id,
  'username': instance.username,
  'email': instance.email,
  'emailVisibility': instance.emailVisibility,
  'verified': instance.verified,
  'created': instance.created,
  'updated': instance.updated,
  'avatar': instance.avatar,
  'expand': instance.expand,
};

_UserExpand _$UserExpandFromJson(Map<String, dynamic> json) => _UserExpand(
  actor: json['activitypub_actors_via_user'] == null
      ? null
      : Actor.fromJson(
          json['activitypub_actors_via_user'] as Map<String, dynamic>,
        ),
);

Map<String, dynamic> _$UserExpandToJson(_UserExpand instance) =>
    <String, dynamic>{'activitypub_actors_via_user': instance.actor};
