import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/feed_item.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/paged_load_more.dart';
import 'package:wanderer/provider/profile/profile_constants.dart';

part 'profile_feed_provider.g.dart';
part 'profile_feed_provider.freezed.dart';

@freezed
abstract class ProfileFeedState with _$ProfileFeedState implements PagedState {
  const factory ProfileFeedState({
    required List<FeedItem> items,
    required int page,
    required int perPage,
    required int totalPages,
    required int totalItems,

    /// True only while a next-page fetch is actually in flight.
    ///
    /// Distinct from [hasMore], which merely says another page *exists*. The
    /// feed's trailing spinner is gated on this: keyed on [hasMore] it stayed
    /// mounted from first paint on any multi-page profile, and since a
    /// `CircularProgressIndicator` drives a repeating ticker, that scheduled a
    /// frame every vsync forever on an idle screen.
    @Default(false) bool isLoadingMore,
  }) = _ProfileFeedState;

  const ProfileFeedState._();
  @override
  bool get hasMore => page < totalPages;

  factory ProfileFeedState.mock() => ProfileFeedState(
    items: List.generate(3, (_) => FeedItem.mock()),
    page: 1,
    perPage: 10,
    totalPages: 1,
    totalItems: 3,
  );
}

@riverpod
class ProfileFeedNotifier extends _$ProfileFeedNotifier
    with PagedLoadMore<ProfileFeedState> {
  late String _handle;

  @override
  FutureOr<ProfileFeedState> build(String handle) async {
    _handle = handle;
    resetPaging();
    return await _fetchPage(handle: handle, page: 1);
  }

  @override
  ProfileFeedState withLoadingMore(ProfileFeedState current, bool value) =>
      current.copyWith(isLoadingMore: value);

  @override
  Future<ProfileFeedState> appendPage(
    ProfileFeedState current,
    int nextPage,
  ) async {
    final responseState = await _fetchPage(handle: _handle, page: nextPage);
    return current.copyWith(
      items: [...current.items, ...responseState.items],
      page: responseState.page,
      totalPages: responseState.totalPages,
    );
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
