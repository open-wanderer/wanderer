import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/list/list_filter_provider.dart';

part 'list_search_provider.freezed.dart';
part 'list_search_provider.g.dart';

@freezed
abstract class ListSearchState with _$ListSearchState {
  const factory ListSearchState({
    required List<ListSearchResult> lists,
    required int page,
    required int perPage,
    required int totalPages,
  }) = _ListSearchState;

  const ListSearchState._();
  bool get hasMore => page < totalPages;

  factory ListSearchState.mock() => ListSearchState(
        lists: List.generate(5, (_) => ListSearchResult.mock()),
        page: 1,
        perPage: 20,
        totalPages: 1,
      );
}

@riverpod
class ListSearchNotifier extends _$ListSearchNotifier {
  @override
  FutureOr<ListSearchState> build() async {
    return await _fetchPage(page: 1);
  }

  Future<void> loadNextPage() async {
    final currentState = state.value;

    if (currentState == null || state.isLoading || !currentState.hasMore) {
      return;
    }

    // Do NOT set AsyncLoading here — stay in AsyncData while fetching the
    // next page so the existing list remains visible.
    state = await AsyncValue.guard(() async {
      final nextPage = currentState.page + 1;
      final responseState = await _fetchPage(page: nextPage);
      return currentState.copyWith(
        lists: [...currentState.lists, ...responseState.lists],
        page: responseState.page,
        totalPages: responseState.totalPages,
      );
    });
  }

  Future<ListSearchState> _fetchPage({required int page}) async {
    final api = ref.read(apiProvider);
    final filter = await ref.watch(listFilterProvider('lists').future);
    final user = ref.read(authProvider).value;

    const int perPage = 20;

    final response = await api.post(
      '/search/lists',
      data: {
        'q': filter.q,
        'options': {
          'filter': filter.toFilterText(actorId: user?.actorId),
          'sort': ["${filter.sort.name}:${filter.sortOrder.name}"],
          'hitsPerPage': perPage,
          'page': page,
        },
      },
    );

    final List<dynamic> hits = response.data['hits'] ?? [];

    final int totalPages = response.data['totalPages'] ?? 1;

    return ListSearchState(
      lists: hits.map((json) => ListSearchResult.fromJson(json)).toList(),
      page: page,
      perPage: perPage,
      totalPages: totalPages,
    );
  }
}
