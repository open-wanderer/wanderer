import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/feed_item.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/list.dart';

// Minimal valid JSON for Trail
// Required fields from the Trail factory (non-default, non-nullable)
Map<String, dynamic> _minimalTrailJson() => {
      'id': 'trail-1',
      'name': 'Test Trail',
      'created': '2024-01-01T00:00:00.000Z',
      'updated': '2024-01-01T00:00:00.000Z',
    };

// Minimal valid JSON for WandererList
// Required fields from the WandererList factory (non-default, non-nullable)
Map<String, dynamic> _minimalListJson() => {
      'id': 'list-1',
      'name': 'Test List',
    };

void main() {
  group('FeedItem.fromJson', () {
    test('type "trail" returns FeedItemTrail with Trail', () {
      final json = {
        'id': 'feed-1',
        'actor': 'actor-1',
        'type': 'trail',
        'created': '2024-01-01 00:00:00.000Z',
        'expand': {
          'item': _minimalTrailJson(),
        },
      };

      final result = FeedItem.fromJson(json);

      expect(result, isA<FeedItemTrail>());
      final trailItem = result as FeedItemTrail;
      expect(trailItem.id, 'feed-1');
      expect(trailItem.actor, 'actor-1');
      expect(trailItem.type, 'trail');
      expect(trailItem.created, '2024-01-01 00:00:00.000Z');
      expect(trailItem.trail, isA<Trail>());
      expect(trailItem.trail.id, 'trail-1');
    });

    test('type "list" returns FeedItemList with WandererList', () {
      final json = {
        'id': 'feed-2',
        'actor': 'actor-1',
        'type': 'list',
        'created': '2024-01-01 00:00:00.000Z',
        'expand': {
          'item': _minimalListJson(),
        },
      };

      final result = FeedItem.fromJson(json);

      expect(result, isA<FeedItemList>());
      final listItem = result as FeedItemList;
      expect(listItem.id, 'feed-2');
      expect(listItem.actor, 'actor-1');
      expect(listItem.type, 'list');
      expect(listItem.created, '2024-01-01 00:00:00.000Z');
      expect(listItem.list, isA<WandererList>());
      expect(listItem.list.id, 'list-1');
    });

    // Replaces the old 'summit_log throws UnsupportedError' test.
    // summit_log items are pre-filtered at the provider level and must never
    // reach fromJson in production. The meaningful defensive contract is that
    // a feed entry arriving WITHOUT expand.item (the field the server may omit
    // for unresolved federated items) throws a typed FormatException — not a
    // null-pointer crash — so the provider's AsyncValue.guard can catch it.
    test('absent expand.item throws FormatException', () {
      final json = {
        'id': 'feed-3',
        'actor': 'actor-1',
        'type': 'trail',
        'created': '2024-01-01 00:00:00.000Z',
        'expand': <String, dynamic>{}, // item key is missing
      };

      expect(() => FeedItem.fromJson(json), throwsA(isA<FormatException>()));
    });

    test('unknown type with expand.item present throws UnsupportedError', () {
      final json = {
        'id': 'feed-4',
        'actor': 'actor-1',
        'type': 'unknown_type',
        'created': '2024-01-01 00:00:00.000Z',
        'expand': {
          'item': {'id': 'x'},
        },
      };

      expect(() => FeedItem.fromJson(json), throwsUnsupportedError);
    });
  });
}
