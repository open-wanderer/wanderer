import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'profile_follows_provider.freezed.dart';
part 'profile_follows_provider.g.dart';

@freezed
abstract class ProfileFollowsState with _$ProfileFollowsState {
  const factory ProfileFollowsState({
    required List<Actor> items,
    required int page,
    required int totalPages,
  }) = _ProfileFollowsState;

  const ProfileFollowsState._();
  bool get hasMore => page < totalPages;
}

@riverpod
class ProfileFollowsNotifier extends _$ProfileFollowsNotifier {
  @override
  FutureOr<ProfileFollowsState> build(String handle, String type) async {
    return await _fetchPage(page: 1);
  }

  Future<void> loadNextPage() async {
    final current = state.value;
    if (current == null || state.isLoading || !current.hasMore) return;
    state = await AsyncValue.guard(() async {
      final next = await _fetchPage(page: current.page + 1);
      return current.copyWith(
        items: [...current.items, ...next.items],
        page: next.page,
        totalPages: next.totalPages,
      );
    });
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
