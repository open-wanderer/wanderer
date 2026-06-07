import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/feed_item.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/global_search_models.dart';

// Minimal valid JSON for TrailSearchResult
// Required fields from TrailSearchResult factory (non-default, non-nullable)
Map<String, dynamic> _minimalTrailJson() => {
      'id': 'trail-1',
      'author': 'user-1',
      'author_name': 'Alice',
      'author_avatar': 'avatar.jpg',
      'name': 'Test Trail',
      'description': 'A test trail',
      'location': 'Test Location',
      'distance': 10.0,
      'elevation_gain': 100.0,
      'elevation_loss': 90.0,
      'duration': 3600.0,
      'difficulty': 1,
      'category': 'hiking',
      'completed': false,
      'date': 1700000000,
      'created': 1700000000,
      'public': true,
      'thumbnail': 'thumb.jpg',
      'like_count': 0,
      'gpx': 'trail.gpx',
      '_geo': {'lat': 48.0, 'lng': 11.0},
    };

// Minimal valid JSON for ListSearchResult
// Required fields from ListSearchResult factory (non-default, non-nullable)
Map<String, dynamic> _minimalListJson() => {
      'id': 'list-1',
      'author': 'user-1',
      'author_name': 'Alice',
      'author_avatar': 'avatar.jpg',
      'name': 'Test List',
      'description': 'A test list',
      'elevation_gain': 100.0,
      'elevation_loss': 90.0,
      'distance': 10.0,
      'duration': 3600.0,
      'public': true,
      'trails': 3,
    };

void main() {
  group('FeedItem.fromJson', () {
    test('type "trail" returns FeedItemTrail with TrailSearchResult', () {
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
      expect(trailItem.trail, isA<TrailSearchResult>());
      expect(trailItem.trail.id, 'trail-1');
    });

    test('type "list" returns FeedItemList with ListSearchResult', () {
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
      expect(listItem.list, isA<ListSearchResult>());
      expect(listItem.list.id, 'list-1');
    });

    test('type "summit_log" throws UnsupportedError', () {
      final json = {
        'id': 'feed-3',
        'actor': 'actor-1',
        'type': 'summit_log',
        'created': '2024-01-01 00:00:00.000Z',
        'expand': {},
      };

      expect(() => FeedItem.fromJson(json), throwsUnsupportedError);
    });

    test('unknown type throws UnsupportedError', () {
      final json = {
        'id': 'feed-4',
        'actor': 'actor-1',
        'type': 'unknown_type',
        'created': '2024-01-01 00:00:00.000Z',
        'expand': {},
      };

      expect(() => FeedItem.fromJson(json), throwsUnsupportedError);
    });
  });
}
