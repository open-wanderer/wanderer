import 'package:objectbox/objectbox.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/search_history_item.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/store/search_history_store.dart';

part 'search_history_provider.g.dart';

/// The signed-in account's search history, most recently used first.
///
/// Every write re-reads `currentAccountId(store)` rather than remembering
/// the id `build()` saw, so an entry can never land in another account's
/// history.
@riverpod
class SearchHistoryNotifier extends _$SearchHistoryNotifier {
  @override
  List<SearchHistoryItem> build() {
    final store = ref.watch(objectBoxProvider);
    final accountId = currentAccountId(store);
    if (accountId == null) return const [];
    return readSearchHistory(store, accountId: accountId);
  }

  void record(SearchHistoryItem item) {
    _write(
      (store, accountId) =>
          recordSearchHistory(store, accountId: accountId, item: item),
    );
  }

  void remove(Iterable<SearchHistoryItem> items) {
    _write(
      (store, accountId) =>
          removeSearchHistory(store, accountId: accountId, items: items),
    );
  }

  void _write(void Function(Store store, String accountId) write) {
    final store = ref.read(objectBoxProvider);
    final accountId = currentAccountId(store);
    if (accountId == null) return;
    write(store, accountId);
    state = readSearchHistory(store, accountId: accountId);
  }
}
