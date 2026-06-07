import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/feed_item.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/profile/profile_constants.dart';

part 'profile_feed_provider.g.dart';
part 'profile_feed_provider.freezed.dart';

@freezed
abstract class ProfileFeedState with _$ProfileFeedState {
  const factory ProfileFeedState({
    required List<FeedItem> items,
    required int page,
    required int perPage,
    required int totalPages,
    required int totalItems,
  }) = _ProfileFeedState;

  const ProfileFeedState._();
  bool get hasMore => page < totalPages;
}

@riverpod
class ProfileFeedNotifier extends _$ProfileFeedNotifier {
  late String _handle;

  @override
  FutureOr<ProfileFeedState> build(String handle) async {
    _handle = handle;
    return await _fetchPage(handle: handle, page: 1);
  }

  Future<void> loadNextPage() async {
    final currentState = state.value;

    if (currentState == null || state.isLoading || !currentState.hasMore) {
      return;
    }

    // Do NOT set AsyncLoading here — stay in AsyncData while fetching the
    // next page so the UI keeps showing the existing list without flickering
    // to an empty spinner. State transitions directly AsyncData -> AsyncData.
    state = await AsyncValue.guard(() async {
      final nextPage = currentState.page + 1;
      final responseState = await _fetchPage(handle: _handle, page: nextPage);
      return currentState.copyWith(
        items: [...currentState.items, ...responseState.items],
        page: responseState.page,
        totalPages: responseState.totalPages,
      );
    });
  }

  Future<ProfileFeedState> _fetchPage({
    required String handle,
    required int page,
  }) async {
    final api = ref.read(apiProvider);
    const int perPage = kProfileFeedPerPage;

    final response = await api.get(
      '/profile/$handle/feed',
      queryParameters: {'page': page, 'perPage': perPage, 'sort': '-created'},
    );

    if (response.data is! Map<String, dynamic>) {
      throw FormatException(
        'Unexpected response shape from /profile/$handle/feed: '
        '${response.data.runtimeType}',
      );
    }
    final data = response.data as Map<String, dynamic>;

    final rawItems = (data['items'] as List? ?? []);
    final List<FeedItem> feedItems = rawItems
        .where((json) {
          final type = (json as Map<String, dynamic>)['type'] as String?;
          return type == 'trail' || type == 'list';
        })
        .map((json) => FeedItem.fromJson(json as Map<String, dynamic>))
        .toList();

    return ProfileFeedState(
      items: feedItems,
      page: data['page'] as int? ?? page,
      perPage: data['perPage'] as int? ?? perPage,
      totalPages: data['totalPages'] as int? ?? 1,
      totalItems: data['totalItems'] as int? ?? 0,
    );
  }
}
