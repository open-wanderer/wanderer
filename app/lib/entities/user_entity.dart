import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/actor_entity.dart';
import 'package:wanderer/entities/settings_entity.dart';
import 'package:wanderer/models/record.dart';

@Entity()
class UserEntity with RecordFunctions implements IRecord {
  @Id()
  int obxId = 0;

  @override
  @Index()
  @Unique(onConflict: ConflictStrategy.replace)
  final String id;
  @override
  final String collectionId;
  @override
  final String collectionName;
  @override
  @Property(type: PropertyType.dateUtc)
  final DateTime created;
  @override
  @Property(type: PropertyType.dateUtc)
  final DateTime updated;
  final String actorId;
  final String username;
  final String preferredUsername;
  final String email;
  final String iri;
  final String serverUrl;

  final String? avatar;

  final settings = ToOne<SettingsEntity>();

  /// Cached ActivityPub actor for this user, populated from the
  /// `activitypub_actors_via_user` expand at auth time and refreshed by
  /// `OwnProfile`. Lets the own-profile screen render its real layout offline
  /// rather than a stand-in scaffold.
  ///
  /// The target-id column is named explicitly because the default (`actorId`)
  /// would collide with the existing [actorId] field above — which holds the
  /// actor's PocketBase record id, not an ObjectBox row id.
  @TargetIdProperty('actorObxId')
  final actor = ToOne<ActorEntity>();

  UserEntity({
    required this.id,
    required this.collectionId,
    required this.collectionName,
    required this.actorId,
    required this.username,
    required this.preferredUsername,
    required this.email,
    required this.iri,
    required this.serverUrl,
    required this.created,
    required this.updated,
    this.avatar,
  });
}
