import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'profile_lists_provider.g.dart';
part 'profile_lists_provider.freezed.dart';

@freezed
abstract class ProfileListsState with _$ProfileListsState {
  const factory ProfileListsState({
    required List<ListSearchResult> lists,
    required int page,
    required int perPage,
    required int totalPages,
  }) = _ProfileListsState;

  const ProfileListsState._();
  bool get hasMore => page < totalPages;
}

@riverpod
class ProfileListsNotifier extends _$ProfileListsNotifier {
  late String _handle;

  @override
  FutureOr<ProfileListsState> build(String handle) async {
    _handle = handle;
    return await _fetchPage(handle: handle, page: 1);
  }

  Future<void> loadNextPage() async {
    final currentState = state.value;

    if (currentState == null || state.isLoading || !currentState.hasMore) {
      return;
    }

    // Preserve previous data in the loading state so the UI does not flicker.
    state = AsyncLoading<ProfileListsState>()..copyWithPrevious(state);
    state = await AsyncValue.guard(() async {
      final nextPage = currentState.page + 1;
      final responseState = await _fetchPage(handle: _handle, page: nextPage);
      return currentState.copyWith(
        lists: [...currentState.lists, ...responseState.lists],
        page: responseState.page,
        totalPages: responseState.totalPages,
      );
    });
  }

  Future<ProfileListsState> _fetchPage({
    required String handle,
    required int page,
  }) async {
    final api = ref.read(apiProvider);
    const int perPage = 20;

    final response = await api.post(
      '/profile/$handle/lists',
      data: {
        'q': '',
        'options': {'hitsPerPage': perPage, 'page': page},
      },
    );

    final List<dynamic> hits = response.data['hits'] ?? [];
    final int totalPages = response.data['totalPages'] ?? 1;

    return ProfileListsState(
      lists: hits.map((json) => ListSearchResult.fromJson(json)).toList(),
      page: page,
      perPage: perPage,
      totalPages: totalPages,
    );
  }
}
