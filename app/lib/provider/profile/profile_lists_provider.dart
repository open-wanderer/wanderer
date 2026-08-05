import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/list/list_filter_provider.dart';
import 'package:wanderer/provider/paged_load_more.dart';
import 'package:wanderer/provider/profile/profile_constants.dart';

part 'profile_lists_provider.g.dart';
part 'profile_lists_provider.freezed.dart';

@freezed
abstract class ProfileListsState
    with _$ProfileListsState
    implements PagedState {
  const factory ProfileListsState({
    required List<ListSearchResult> lists,
    required int page,
    required int perPage,
    required int totalPages,
  }) = _ProfileListsState;

  const ProfileListsState._();

  factory ProfileListsState.mock() {
    return ProfileListsState(
      lists: List.generate(5, (_) => ListSearchResult.mock()),
      page: 1,
      perPage: 5,
      totalPages: 1,
    );
  }

  @override
  bool get hasMore => page < totalPages;
}

@riverpod
class ProfileListsNotifier extends _$ProfileListsNotifier
    with PagedLoadMore<ProfileListsState> {
  late String _handle;
  String _q = '';

  @override
  FutureOr<ProfileListsState> build(String handle) async {
    _handle = handle;
    resetPaging();
    ref.watch(listFilterProvider('profile_list_$handle'));
    return await _fetchPage(handle: handle, page: 1, q: _q);
  }

  Future<void> search(String q) async {
    _q = q;
    state = const AsyncLoading();
    state = await AsyncValue.guard(
      () => _fetchPage(handle: _handle, page: 1, q: _q),
    );
  }

  @override
  Future<ProfileListsState> appendPage(
    ProfileListsState current,
    int nextPage,
  ) async {
    final responseState = await _fetchPage(
      handle: _handle,
      page: nextPage,
      q: _q,
    );
    return current.copyWith(
      lists: [...current.lists, ...responseState.lists],
      page: responseState.page,
      totalPages: responseState.totalPages,
    );
  }

  Future<ProfileListsState> _fetchPage({
    required String handle,
    required int page,
    required String q,
  }) async {
    final api = ref.read(apiProvider);
    final filter = ref.read(listFilterProvider('profile_list_$handle')).value;
    const int perPage = kProfileSearchPerPage;

    final response = await api.post(
      '/profile/$handle/lists',
      data: {
        'q': q,
        'options': {
          'hitsPerPage': perPage,
          'page': page,
          if (filter != null)
            'sort': ['${filter.sort.name}:${filter.sortOrder.name}'],
        },
      },
    );

    if (response.data is! Map<String, dynamic>) {
      throw FormatException(
        'Unexpected response shape from /profile/$handle/lists: '
        '${response.data.runtimeType}',
      );
    }
    final data = response.data as Map<String, dynamic>;

    final List<dynamic> hits = data['hits'] ?? [];
    final int totalPages = data['totalPages'] ?? 1;

    return ProfileListsState(
      lists: hits.map((json) => ListSearchResult.fromJson(json)).toList(),
      page: page,
      perPage: perPage,
      totalPages: totalPages,
    );
  }
}
