import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/record.dart';

part 'actor.freezed.dart';
part 'actor.g.dart';

@Freezed()
abstract class Actor with _$Actor, RecordFunctions implements IRecord {
  const Actor._();

  const factory Actor({
    required String id,
    required String collectionId,
    required String collectionName,
    required DateTime created,
    required DateTime updated,
    required String username,
    @JsonKey(name: 'preferred_username') required String preferredUsername,
    String? domain,
    String? summary,
    String? published,
    int? followerCount,
    int? followingCount,
    required String iri,
    required String inbox,
    String? outbox,
    String? icon,
    String? followers,
    String? following,
    @Default(false) bool isLocal,
    @JsonKey(name: 'public_key') required String publicKey,
    @JsonKey(name: 'last_fetched') required String lastFetched,
    required String user,
  }) = _Actor;

  factory Actor.fromJson(Map<String, dynamic> json) => _$ActorFromJson(json);
}
