import 'dart:convert';

import 'package:wanderer/entities/search_history_entity.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/search_history_item.dart';
import 'package:wanderer/objectbox.g.dart';

/// How many entries one account's history keeps; the oldest fall off.
const searchHistoryCap = 20;

/// Encodes [item] as the `(kind, payload)` pair a [SearchHistoryEntity]
/// stores.
({String kind, String payload}) encodeSearchHistoryItem(
  SearchHistoryItem item,
) {
  final (kind, json) = switch (item) {
    TrailHistoryItem(:final trail) => ('trail', trail.toJson()),
    ListHistoryItem(:final list) => ('list', list.toJson()),
    LocationHistoryItem(:final location) => ('location', location.toJson()),
    ActorHistoryItem(:final actor) => ('actor', actor.toJson()),
  };
  return (kind: kind, payload: jsonEncode(json));
}

/// Decodes a stored `(kind, payload)` pair, or null when the kind is
/// unknown or the payload no longer matches its model -- a snapshot written
/// by an older app version must never break the whole history.
SearchHistoryItem? decodeSearchHistoryItem(String kind, String payload) {
  try {
    final json = jsonDecode(payload) as Map<String, dynamic>;
    return switch (kind) {
      'trail' => TrailHistoryItem(TrailSearchResult.fromJson(json)),
      'list' => ListHistoryItem(ListSearchResult.fromJson(json)),
      'location' => LocationHistoryItem(LocationSearchResult.fromJson(json)),
      'actor' => ActorHistoryItem(ActorSearchResult.fromJson(json)),
      _ => null,
    };
  } catch (_) {
    return null;
  }
}

/// [accountId]'s history, most recently used first.
List<SearchHistoryItem> readSearchHistory(
  Store store, {
  required String accountId,
}) {
  final query = store
      .box<SearchHistoryEntity>()
      .query(SearchHistoryEntity_.accountId.equals(accountId))
      .order(SearchHistoryEntity_.usedAt, flags: Order.descending)
      .build();
  final rows = query.find();
  query.close();
  return [
    for (final row in rows) ?decodeSearchHistoryItem(row.kind, row.payload),
  ];
}

/// Puts [item] at the top of [accountId]'s history, replacing an earlier
/// entry for the same result and trimming the history to
/// [searchHistoryCap].
void recordSearchHistory(
  Store store, {
  required String accountId,
  required SearchHistoryItem item,
}) {
  final box = store.box<SearchHistoryEntity>();
  final encoded = encodeSearchHistoryItem(item);
  store.runInTransaction(TxMode.write, () {
    _removeKeys(store, accountId: accountId, keys: [item.key]);
    box.put(
      SearchHistoryEntity(
        accountId: accountId,
        key: item.key,
        kind: encoded.kind,
        payload: encoded.payload,
        usedAt: DateTime.now().toUtc(),
      ),
    );

    final query = box
        .query(SearchHistoryEntity_.accountId.equals(accountId))
        .order(SearchHistoryEntity_.usedAt, flags: Order.descending)
        .build();
    final ids = query.findIds();
    query.close();
    if (ids.length > searchHistoryCap) {
      box.removeMany(ids.sublist(searchHistoryCap));
    }
  });
}

/// Removes the entries for [items] from [accountId]'s history.
void removeSearchHistory(
  Store store, {
  required String accountId,
  required Iterable<SearchHistoryItem> items,
}) {
  _removeKeys(
    store,
    accountId: accountId,
    keys: [for (final item in items) item.key],
  );
}

void _removeKeys(
  Store store, {
  required String accountId,
  required List<String> keys,
}) {
  if (keys.isEmpty) return;
  final query = store
      .box<SearchHistoryEntity>()
      .query(
        SearchHistoryEntity_.accountId
            .equals(accountId)
            .and(SearchHistoryEntity_.key.oneOf(keys)),
      )
      .build();
  query.remove();
  query.close();
}
