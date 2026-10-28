import 'package:objectbox/objectbox.dart';

/// One remembered search result, owned by the account that tapped it.
///
/// The result itself is stored as a JSON snapshot in [payload]; [kind]
/// says which result model to decode it with (see
/// `search_history_store.dart`).
@Entity()
class SearchHistoryEntity {
  @Id()
  int obxId = 0;

  @Index()
  String accountId;

  String key;
  String kind;
  String payload;

  @Property(type: PropertyType.dateUtc)
  DateTime usedAt;

  SearchHistoryEntity({
    required this.accountId,
    required this.key,
    required this.kind,
    required this.payload,
    required this.usedAt,
  });
}
