import 'package:objectbox/objectbox.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/category_entity.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/list_result.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';

part 'category_provider.g.dart';

@Riverpod(keepAlive: true)
class CategoryNotifier extends _$CategoryNotifier {
  @override
  FutureOr<List<Category>> build() async {
    final api = ref.watch(apiProvider);

    try {
      final response = await api.get('/category');

      if (response.data == null) {
        throw Exception('No category data received from server');
      }

      ListResult<Category> categoryListResult = ListResult.fromJson(
        response.data,
        (json) => Category.fromJson(json as Map<String, dynamic>),
      );

      final items = categoryListResult.items;

      // Overwrite all cached category rows on every successful fetch (no
      // staleness tracking). Kept inside the try block so a failed fetch
      // leaves the prior cache intact. Both operations run in a single write
      // transaction so a mid-operation crash cannot leave the cache empty.
      final store = ref.read(objectBoxProvider);
      store.runInTransaction(TxMode.write, () {
        final box = store.box<CategoryEntity>();
        box.removeAll();
        box.putMany(items.map(CategoryEntity.fromModel).toList());
      });

      return items;
    } catch (e) {
      // Fall back to the ObjectBox cache so offline cold-starts can still
      // serve categories from a previous successful fetch.
      final box = ref.read(objectBoxProvider).box<CategoryEntity>();
      final cached = box.getAll();
      if (cached.isNotEmpty) return cached.map((e) => e.toModel()).toList();
      throw Exception('Failed to fetch categories: $e');
    }
  }
}
