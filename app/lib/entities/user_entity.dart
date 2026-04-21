import 'package:objectbox/objectbox.dart';

@Entity()
class UserEntity {
  @Id()
  int obxId = 0;

  @Index()
  @Unique(onConflict: ConflictStrategy.replace)
  final String id;
  final String actorId;
  final String username;
  final String preferredUsername;
  final String email;
  final String iri;
  final String serverUrl;
  final String created;
  final String updated;
  final String? avatar;

  UserEntity({
    required this.id,
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
