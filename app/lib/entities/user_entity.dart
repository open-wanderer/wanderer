import 'package:objectbox/objectbox.dart';
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
