import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/global_search_models.dart';
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
    required TrailSearchResult trail,
  }) = FeedItemTrail;

  const factory FeedItem.list({
    required String id,
    required String actor,
    required String type,
    required String created,
    required ListSearchResult list,
  }) = FeedItemList;

  // Hand-written dispatch factory.
  // Callers MUST pre-filter summit_log items before calling fromJson
  // (see profile_feed_provider.dart). Unknown types throw UnsupportedError
  // rather than silently corrupting state (mitigates T-01-01).
  factory FeedItem.fromJson(Map<String, dynamic> json) {
    final type = json['type'] as String?;
    final expandRaw = json['expand'];
    final expand = (expandRaw is Map)
        ? expandRaw.cast<String, dynamic>()
        : <String, dynamic>{};
    final item = expand['item'] as Map<String, dynamic>?;

    return switch (type) {
      'trail' => FeedItem.trail(
          id: json['id'] as String,
          actor: json['actor'] as String,
          type: type!,
          created: json['created'] as String,
          trail: TrailSearchResult.fromJson(item!),
        ),
      'list' => FeedItem.list(
          id: json['id'] as String,
          actor: json['actor'] as String,
          type: type!,
          created: json['created'] as String,
          list: ListSearchResult.fromJson(item!),
        ),
      _ => throw UnsupportedError('Unsupported feed item type: $type'),
    };
  }
}
