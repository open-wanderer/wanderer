import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/subcategory_entity.dart';
import 'package:wanderer/models/list_result.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';

part 'subcategory_provider.g.dart';

@Riverpod(keepAlive: true)
class SubcategoryNotifier extends _$SubcategoryNotifier {
  @override
  List<Subcategory> build() {
    // Cache-first: return cached subcategories immediately so they are
    // available at app start without waiting for an API call (D-03, CAT-03).
    final box = ref.watch(objectBoxProvider).box<SubcategoryEntity>();
    final cached = box.getAll().map((e) => e.toModel()).toList();

    // Kick off a background refresh that does not block the synchronous return.
    Future.microtask(_refresh);

    return cached;
  }

  /// Fetches `/subcategory` in the background, overwrites the ObjectBox cache,
  /// and updates provider state with the fresh list. Failures are swallowed so
  /// a missing network at startup leaves the cached rows and state intact (T-10-09).
  Future<void> _refresh() async {
    try {
      final response = await ref.read(apiProvider).get('/subcategory');

      if (response.data == null) {
        return;
      }

      final items = ListResult.fromJson(
        response.data,
        (json) => Subcategory.fromJson(json as Map<String, dynamic>),
      ).items;

      // Overwrite all cached subcategory rows on every successful refresh.
      // Both operations run in a single write transaction so a mid-operation
      // crash cannot leave the cache empty.
      final store = ref.read(objectBoxProvider);
      store.runInTransaction(TxMode.write, () {
        final box = store.box<SubcategoryEntity>();
        box.removeAll();
        box.putMany(items.map(SubcategoryEntity.fromModel).toList());
      });

      state = items;
    } catch (_) {
      // Swallow — keep the cached state and rows so startup stays resilient.
    }
  }
}
