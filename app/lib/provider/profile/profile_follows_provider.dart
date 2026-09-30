import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/paged_load_more.dart';

part 'profile_follows_provider.freezed.dart';
part 'profile_follows_provider.g.dart';

@freezed
abstract class ProfileFollowsState
    with _$ProfileFollowsState
    implements PagedState {
  const factory ProfileFollowsState({
    required List<Actor> items,
    required int page,
    required int totalPages,

    /// Cursor for the next page, null on the last page.
    String? next,

    /// Whether the backend sent a `next` key at all.
    @Default(false) bool cursorPaging,
  }) = _ProfileFollowsState;

  const ProfileFollowsState._();

  /// Older backends send no `next`, so they keep the totalPages test.
  @override
  bool get hasMore => cursorPaging ? next != null : page < totalPages;
}

@riverpod
class ProfileFollowsNotifier extends _$ProfileFollowsNotifier
    with PagedLoadMore<ProfileFollowsState> {
  @override
  FutureOr<ProfileFollowsState> build(String handle, String type) async {
    resetPaging();
    return await _fetchPage(page: 1);
  }

  @override
  Future<ProfileFollowsState> appendPage(
    ProfileFollowsState current,
    int nextPage,
  ) async {
    final fetched = await _fetchPage(
      page: nextPage,
      cursor: current.cursorPaging ? current.next : null,
    );
    return current.copyWith(
      items: [...current.items, ...fetched.items],
      page: fetched.page,
      totalPages: fetched.totalPages,
      next: fetched.next,
      cursorPaging: fetched.cursorPaging,
    );
  }

  Future<ProfileFollowsState> _fetchPage({
    required int page,
    String? cursor,
  }) async {
    final api = ref.read(apiProvider);
    final response = await api.get(
      '/profile/$handle/follows',
      queryParameters: {
        'type': type,
        'page': page,
        'cursor': ?cursor,
      },
    );

    final data = response.data as Map<String, dynamic>;
    final List<dynamic> rawItems = data['items'] ?? [];
    final int totalPages = (data['totalPages'] as num?)?.toInt() ?? 1;
    final rawNext = data['next'];

    return ProfileFollowsState(
      items: rawItems
          .whereType<Map<String, dynamic>>()
          .map(Actor.fromJson)
          .toList(),
      page: page,
      totalPages: totalPages,
      next: rawNext is String && rawNext.isNotEmpty ? rawNext : null,
      cursorPaging: data.containsKey('next'),
    );
  }
}
