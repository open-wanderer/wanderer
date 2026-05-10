import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/list_result.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'category_provider.g.dart';

@riverpod
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

      return categoryListResult.items;
    } catch (e) {
      throw Exception('Failed to fetch categories: $e');
    }
  }
}
