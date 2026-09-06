import 'dart:convert';

import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/entities/actor_entity.dart';
import 'package:wanderer/entities/settings_entity.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/models/record.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/util/server_url.dart';

part 'user.freezed.dart';
part 'user.g.dart';

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

    // Port and subpath prefix included: an instance on a non-default port or
    // under a path prefix is unreachable without them (see serverUrlFromActorIri).
    final iri = expand!.actor!.iri;
    final rootUrl = serverUrlFromActorIri(iri);
    if (rootUrl == null) {
      throw Exception("Actor IRI is not an absolute http(s) URL: $iri");
    }

    final entity = UserEntity(
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
      serverUrl: rootUrl,
    );

    if (expand?.settings != null) {
      entity.settings.target = SettingsEntity.fromModel(expand!.settings!);
    }

    entity.actor.target = ActorEntity.fromModel(expand!.actor!);

    return entity;
  }
}

@Freezed()
abstract class UserExpand with _$UserExpand {
  const factory UserExpand({
    @JsonKey(name: 'activitypub_actors_via_user') Actor? actor,
    @JsonKey(name: 'settings_via_user') Settings? settings,
  }) = _UserExpand;

  factory UserExpand.fromJson(Map<String, dynamic> json) =>
      _$UserExpandFromJson(json);
}
