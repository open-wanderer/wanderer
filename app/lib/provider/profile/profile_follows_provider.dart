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
  }) = _ProfileFollowsState;

  const ProfileFollowsState._();
  @override
  bool get hasMore => page < totalPages;
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
    final next = await _fetchPage(page: nextPage);
    return current.copyWith(
      items: [...current.items, ...next.items],
      page: next.page,
      totalPages: next.totalPages,
    );
  }

  Future<ProfileFollowsState> _fetchPage({required int page}) async {
    final api = ref.read(apiProvider);
    final response = await api.get(
      '/profile/$handle/follows',
      queryParameters: {'type': type, 'page': page},
    );

    final data = response.data as Map<String, dynamic>;
    final List<dynamic> rawItems = data['items'] ?? [];
    final int totalPages = (data['totalPages'] as num?)?.toInt() ?? 1;

    return ProfileFollowsState(
      items: rawItems
          .whereType<Map<String, dynamic>>()
          .map(Actor.fromJson)
          .toList(),
      page: page,
      totalPages: totalPages,
    );
  }
}
