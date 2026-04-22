import 'package:objectbox/objectbox.dart';
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
  final String created;
  @override
  final String updated;
  final String actorId;
  final String username;
  final String preferredUsername;
  final String email;
  final String iri;
  final String serverUrl;

  final String? avatar;

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
