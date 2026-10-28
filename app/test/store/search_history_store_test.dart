import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/search_history_item.dart';
import 'package:wanderer/store/search_history_store.dart';

// ---------------------------------------------------------------------------
// Encode/decode and identity tests for the search history. There is no
// ObjectBox test harness for plain `flutter test`, so the Store-touching
// functions in search_history_store.dart are verified on-device.
// ---------------------------------------------------------------------------

const _location = LocationSearchResult(
  name: 'Zugspitze',
  description: 'Garmisch-Partenkirchen, Bayern, Deutschland',
  lat: 47.421066,
  lon: 10.985365,
  category: 'natural',
  type: 'peak',
);

const _actor = ActorSearchResult(
  id: 'actor-1',
  username: 'Hiker',
  preferredUsername: 'hiker',
  domain: 'example.org',
  iri: 'https://example.org/api/v1/activitypub/user/hiker',
);

SearchHistoryItem _roundTrip(SearchHistoryItem item) {
  final encoded = encodeSearchHistoryItem(item);
  return decodeSearchHistoryItem(encoded.kind, encoded.payload)!;
}

void main() {
  group('encode/decode round trip', () {
    test('trail', () {
      final trail = TrailSearchResult.mock();
      final decoded = _roundTrip(TrailHistoryItem(trail));
      expect(decoded, isA<TrailHistoryItem>());
      expect((decoded as TrailHistoryItem).trail, trail);
    });

    test('list', () {
      final list = ListSearchResult.mock();
      final decoded = _roundTrip(ListHistoryItem(list));
      expect((decoded as ListHistoryItem).list, list);
    });

    test('location', () {
      final decoded = _roundTrip(const LocationHistoryItem(_location));
      expect((decoded as LocationHistoryItem).location, _location);
    });

    test('actor', () {
      final decoded = _roundTrip(const ActorHistoryItem(_actor));
      expect((decoded as ActorHistoryItem).actor, _actor);
    });
  });

  group('decodeSearchHistoryItem', () {
    test('returns null for an unknown kind', () {
      expect(decodeSearchHistoryItem('photo', '{}'), isNull);
    });

    test('returns null for a payload that no longer matches its model', () {
      expect(decodeSearchHistoryItem('trail', '{"id": 1}'), isNull);
      expect(decodeSearchHistoryItem('trail', 'not json'), isNull);
    });
  });

  group('SearchHistoryItem.key', () {
    test('is stable for the same result', () {
      expect(
        const LocationHistoryItem(_location).key,
        _roundTrip(const LocationHistoryItem(_location)).key,
      );
    });

    test('differs between kinds sharing an id', () {
      final trail = TrailSearchResult.mock().copyWith(id: 'same');
      final list = ListSearchResult.mock().copyWith(id: 'same');
      expect(TrailHistoryItem(trail).key, isNot(ListHistoryItem(list).key));
    });

    test('tells apart two places with the same name', () {
      final elsewhere = _location.copyWith(lat: 46.0, lon: 11.0);
      expect(
        const LocationHistoryItem(_location).key,
        isNot(LocationHistoryItem(elsewhere).key),
      );
    });
  });
}
