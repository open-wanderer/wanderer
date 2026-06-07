import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'profile_trails_provider.g.dart';
part 'profile_trails_provider.freezed.dart';

@freezed
abstract class ProfileTrailsState with _$ProfileTrailsState {
  const factory ProfileTrailsState({
    required List<TrailSearchResult> trails,
    required int page,
    required int perPage,
    required int totalPages,
  }) = _ProfileTrailsState;

  const ProfileTrailsState._();
  bool get hasMore => page < totalPages;
}

@riverpod
class ProfileTrailsNotifier extends _$ProfileTrailsNotifier {
  late String _handle;

  @override
  FutureOr<ProfileTrailsState> build(String handle) async {
    _handle = handle;
    return await _fetchPage(handle: handle, page: 1);
  }

  Future<void> loadNextPage() async {
    final currentState = state.value;

    if (currentState == null || state.isLoading || !currentState.hasMore) {
      return;
    }

    // Preserve previous data in the loading state so the UI does not flicker.
    state = AsyncLoading<ProfileTrailsState>()..copyWithPrevious(state);
    state = await AsyncValue.guard(() async {
      final nextPage = currentState.page + 1;
      final responseState = await _fetchPage(handle: _handle, page: nextPage);
      return currentState.copyWith(
        trails: [...currentState.trails, ...responseState.trails],
        page: responseState.page,
        totalPages: responseState.totalPages,
      );
    });
  }

  Future<ProfileTrailsState> _fetchPage({
    required String handle,
    required int page,
  }) async {
    final api = ref.read(apiProvider);
    const int perPage = 20;

    final response = await api.post(
      '/profile/$handle/trails',
      data: {
        'q': '',
        'options': {'hitsPerPage': perPage, 'page': page},
      },
    );

    if (response.data is! Map<String, dynamic>) {
      throw FormatException(
        'Unexpected response shape from /profile/$handle/trails: '
        '${response.data.runtimeType}',
      );
    }
    final data = response.data as Map<String, dynamic>;

    final List<dynamic> hits = data['hits'] ?? [];
    final int totalPages = data['totalPages'] ?? 1;

    return ProfileTrailsState(
      trails: hits.map((json) => TrailSearchResult.fromJson(json)).toList(),
      page: page,
      perPage: perPage,
      totalPages: totalPages,
    );
  }
}
