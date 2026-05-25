import 'dart:convert';

import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/models/record.dart';

part 'user.freezed.dart';
part 'user.g.dart';

class ActorListConverter implements JsonConverter<Actor?, Object?> {
  const ActorListConverter();

  @override
  Actor? fromJson(Object? json) {
    if (json == null) return null;
    if (json is List && json.isNotEmpty) {
      return Actor.fromJson(json.first as Map<String, dynamic>);
    }
    if (json is Map<String, dynamic>) {
      return Actor.fromJson(json);
    }
    return null;
  }

  @override
  Object? toJson(Actor? object) => object?.toJson();
}

@Freezed()
abstract class User with _$User, RecordFunctions implements IRecord {
  const User._();

  const factory User({
    required String id,
    required String collectionId,
    required String collectionName,
    required DateTime created,
    required DateTime updated,
    required String username,
    required String email,
    required bool emailVisibility,
    required bool verified,

    String? avatar,
    UserExpand? expand,
  }) = _User;

  factory User.fromJson(Map<String, dynamic> json) => _$UserFromJson(json);

  factory User.fromCookie(String cookieValue) {
    final decoded = Uri.decodeComponent(cookieValue);
    final map = jsonDecode(decoded) as Map<String, dynamic>;
    return User.fromJson(map['record'] as Map<String, dynamic>);
  }

  UserEntity toEntity() {
    if (expand?.actor == null) {
      throw Exception(
        "Cannot convert User to UserEntity without expanded actor",
      );
    }

    final userIri = Uri.parse(expand!.actor!.iri);
    final rootUri = Uri(scheme: userIri.scheme, host: userIri.host);

    return UserEntity(
      id: id,
      collectionId: collectionId,
      collectionName: collectionName,
      actorId: expand!.actor!.id,
      username: expand!.actor!.username,
      preferredUsername: expand!.actor!.preferredUsername,
      email: email,
      created: created,
      updated: updated,
      avatar: avatar,
      iri: expand!.actor!.iri,
      serverUrl: rootUri.toString(),
    );
  }
}

@Freezed()
abstract class UserExpand with _$UserExpand {
  const factory UserExpand({
    @JsonKey(name: 'activitypub_actors_via_user') @ActorListConverter() Actor? actor,
  }) = _UserExpand;

  factory UserExpand.fromJson(Map<String, dynamic> json) =>
      _$UserExpandFromJson(json);
}
