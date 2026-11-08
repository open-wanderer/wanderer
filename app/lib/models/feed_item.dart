import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/list.dart';
import 'package:wanderer/models/trail.dart';

part 'feed_item.freezed.dart';

// Freezed 3.x sealed class — Dart 3 pattern matching applies.
// fromJson is hand-written (dispatch on 'type' field); no @JsonSerializable,
// so no .g.dart part directive is needed.
@freezed
sealed class FeedItem with _$FeedItem {
  const factory FeedItem.trail({
    required String id,
    required String actor,
    required String type,
    required String created,
    required Trail trail,
  }) = FeedItemTrail;

  const factory FeedItem.list({
    required String id,
    required String actor,
    required String type,
    required String created,
    required WandererList list,
  }) = FeedItemList;

  // Hand-written dispatch factory.
  // Callers MUST pre-filter summit_log items before calling fromJson
  // (see profile_feed_provider.dart). Unknown types throw UnsupportedError
  // rather than silently corrupting state.
  factory FeedItem.mock() => FeedItem.trail(
    id: 'mock-feed-id',
    actor: 'mock-actor-id',
    type: 'trail',
    created: '2024-01-01 00:00:00.000Z',
    trail: Trail(
      id: 'mock-trail-id',
      name: 'Mock Trail Name',
      created: DateTime(2024, 1, 1),
      updated: DateTime(2024, 1, 1),
      distance: 8000,
      elevationGain: 350,
      elevationLoss: 350,
      duration: 120,
    ),
  );

  factory FeedItem.fromJson(Map<String, dynamic> json) {
    final type = json['type'] as String?;
    final expandRaw = json['expand'];
    final expand = (expandRaw is Map)
        ? expandRaw.cast<String, dynamic>()
        : <String, dynamic>{};
    final item = expand['item'] as Map<String, dynamic>?;

    if (item == null) {
      throw FormatException(
        'FeedItem.fromJson: expand.item is absent for type "$type".',
      );
    }

    return switch (type) {
      'trail' => FeedItem.trail(
        id: json['id'] as String,
        actor: json['actor'] as String,
        type: type!, // type is narrowed to non-null by the switch arm
        created: json['created'] as String,
        trail: Trail.fromJson(item),
      ),
      'list' => FeedItem.list(
        id: json['id'] as String,
        actor: json['actor'] as String,
        type: type!, // type is narrowed to non-null by the switch arm
        created: json['created'] as String,
        list: WandererList.fromJson(item),
      ),
      _ => throw UnsupportedError('Unsupported feed item type: $type'),
    };
  }
}
