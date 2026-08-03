import 'package:objectbox/objectbox.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/objectbox.g.dart';

@Entity()
class ActorEntity {
  @Id()
  int obxId = 0;

  @Index()
  @Unique(onConflict: ConflictStrategy.replace)
  String id;

  String collectionId;
  String collectionName;
  @Property(type: PropertyType.dateUtc)
  DateTime created;
  @Property(type: PropertyType.dateUtc)
  DateTime updated;
  String username;
  String preferredUsername;
  String? domain;
  String? summary;
  String? published;
  int? followerCount;
  int? followingCount;
  String iri;
  String inbox;
  String? outbox;
  String? icon;
  String? followers;
  String? following;
  bool isLocal;
  String publicKey;
  String lastFetched;
  String user;

  ActorEntity({
    required this.id,
    required this.collectionId,
    required this.collectionName,
    required this.created,
    required this.updated,
    required this.username,
    required this.preferredUsername,
    this.domain,
    this.summary,
    this.published,
    this.followerCount,
    this.followingCount,
    required this.iri,
    required this.inbox,
    this.outbox,
    this.icon,
    this.followers,
    this.following,
    this.isLocal = false,
    required this.publicKey,
    required this.lastFetched,
    required this.user,
  });

  factory ActorEntity.fromModel(Actor a) {
    return ActorEntity(
      id: a.id,
      collectionId: a.collectionId,
      collectionName: a.collectionName,
      created: a.created,
      updated: a.updated,
      username: a.username,
      preferredUsername: a.preferredUsername,
      domain: a.domain,
      summary: a.summary,
      published: a.published,
      followerCount: a.followerCount,
      followingCount: a.followingCount,
      iri: a.iri,
      inbox: a.inbox,
      outbox: a.outbox,
      icon: a.icon,
      followers: a.followers,
      following: a.following,
      isLocal: a.isLocal,
      publicKey: a.publicKey,
      lastFetched: a.lastFetched,
      user: a.user,
    );
  }
}

/// Builds the [ActorEntity] for [actor] carrying the ObjectBox id of the row
/// that already holds it, so a put UPDATES that row instead of replacing it.
///
/// [ActorEntity.id] is `@Unique(onConflict: replace)`. A plain
/// `ActorEntity.fromModel(...)` has `obxId == 0`, so putting it deletes the
/// existing row and inserts a new one under a new ObjectBox id — and every
/// `ToOne<ActorEntity>` pointing at the old id (notably
/// `TrailEntity.author`) silently resolves to null from then on. Reusing the
/// id keeps those relations intact across sign-in and profile refresh.
ActorEntity actorEntityForUpsert(Store store, Actor actor) {
  final entity = ActorEntity.fromModel(actor);
  final query = store
      .box<ActorEntity>()
      .query(ActorEntity_.id.equals(actor.id))
      .build();
  final existing = query.findFirst();
  query.close();
  if (existing != null) entity.obxId = existing.obxId;
  return entity;
}

extension ActorEntityMapping on ActorEntity {
  Actor toModel() {
    return Actor(
      id: id,
      collectionId: collectionId,
      collectionName: collectionName,
      created: created,
      updated: updated,
      username: username,
      preferredUsername: preferredUsername,
      domain: domain,
      summary: summary,
      published: published,
      followerCount: followerCount,
      followingCount: followingCount,
      iri: iri,
      inbox: inbox,
      outbox: outbox,
      icon: icon,
      followers: followers,
      following: following,
      isLocal: isLocal,
      publicKey: publicKey,
      lastFetched: lastFetched,
      user: user,
    );
  }
}
