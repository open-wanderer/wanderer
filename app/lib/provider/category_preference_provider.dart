import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/category_preference.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';

part 'category_preference_provider.g.dart';

@Riverpod(keepAlive: true)
class CategoryPreferenceNotifier extends _$CategoryPreferenceNotifier {
  @override
  Future<List<CategoryPreference>> build() async {
    final user = ref.watch(authProvider).value;
    if (user == null) {
      // Graceful degradation: anonymous users have no preferences and
      // consumers treat the empty list as "all visible" — no API call made.
      return <CategoryPreference>[];
    }

    try {
      final response = await ref
          .read(apiProvider)
          .get('/user-category-preference');

      // The endpoint returns a BARE JSON array (not a paginated list wrapper).
      // Guard against null or unexpected shape (e.g. error object) before
      // casting so failures surface with a meaningful message rather than an
      // opaque TypeError.
      final data = response.data;
      if (data == null || data is! List) {
        throw Exception(
          'Unexpected response shape from /user-category-preference: $data',
        );
      }
      return data
          .map((e) => CategoryPreference.fromJson(e as Map<String, dynamic>))
          .toList();
    } catch (e) {
      throw Exception('Failed to fetch category preferences: $e');
    }
  }

  /// Creates or updates a single category preference. The server injects the
  /// owning `user` from the session — never send it.
  Future<void> upsert(String categoryId, bool visible) async {
    await ref
        .read(apiProvider)
        .put(
          '/user-category-preference',
          data: {'category': categoryId, 'visible': visible},
        );
    ref.invalidateSelf();
  }

  /// Persists a new ordering of category preferences. The server injects the
  /// owning `user` from the session — never send it.
  /// Callers wrap this in try/catch + toast; no error handling here.
  Future<void> reorder(List<String> orderedCategoryIds) async {
    await ref
        .read(apiProvider)
        .post(
          '/user-category-preference/reorder',
          data: {'categories': orderedCategoryIds},
        );
    ref.invalidateSelf();
  }
}
