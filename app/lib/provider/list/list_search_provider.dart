import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/list/list_filter_provider.dart';
import 'package:wanderer/provider/paged_load_more.dart';

part 'list_search_provider.freezed.dart';
part 'list_search_provider.g.dart';

@freezed
abstract class ListSearchState with _$ListSearchState implements PagedState {
  const factory ListSearchState({
    required List<ListSearchResult> lists,
    required int page,
    required int perPage,
    required int totalPages,
  }) = _ListSearchState;

  const ListSearchState._();
  @override
  bool get hasMore => page < totalPages;

  factory ListSearchState.mock() => ListSearchState(
    lists: List.generate(5, (_) => ListSearchResult.mock()),
    page: 1,
    perPage: 20,
    totalPages: 1,
  );
}

@riverpod
class ListSearchNotifier extends _$ListSearchNotifier
    with PagedLoadMore<ListSearchState> {
  @override
  FutureOr<ListSearchState> build() async {
    resetPaging();
    return await _fetchPage(page: 1);
  }

  @override
  Future<ListSearchState> appendPage(
    ListSearchState current,
    int nextPage,
  ) async {
    final responseState = await _fetchPage(page: nextPage);
    return current.copyWith(
      lists: [...current.lists, ...responseState.lists],
      page: responseState.page,
      totalPages: responseState.totalPages,
    );
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
