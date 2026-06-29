import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/subcategory_preference.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';

part 'subcategory_preference_provider.g.dart';

@Riverpod(keepAlive: true)
class SubcategoryPreferenceNotifier extends _$SubcategoryPreferenceNotifier {
  @override
  Future<List<SubcategoryPreference>> build() async {
    final user = ref.watch(authProvider).value;
    if (user == null) {
      // Graceful degradation (D-07): anonymous users have no preferences and
      // consumers treat the empty list as "all visible" — no API call made.
      return <SubcategoryPreference>[];
    }

    try {
      final response = await ref
          .read(apiProvider)
          .get('/user-subcategory-preference');

      // The endpoint returns a BARE JSON array (not a paginated list wrapper).
      return (response.data as List)
          .map((e) => SubcategoryPreference.fromJson(e as Map<String, dynamic>))
          .toList();
    } catch (e) {
      throw Exception('Failed to fetch subcategory preferences: $e');
    }
  }

  /// Creates or updates a single subcategory preference. The server injects the
  /// owning `user` from the session (Security V4 / T-10-05) — never send it.
  Future<void> upsert(String subcategoryId, bool visible) async {
    await ref.read(apiProvider).put(
      '/user-subcategory-preference',
      data: {'subcategory': subcategoryId, 'visible': visible},
    );
    ref.invalidateSelf();
  }
}
