import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/profile/profile_constants.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';

part 'profile_trails_provider.freezed.dart';
part 'profile_trails_provider.g.dart';

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
  String _q = '';

  @override
  FutureOr<ProfileTrailsState> build(String handle) async {
    _handle = handle;

    // Watch the filter so we rebuild and re-fetch when it changes.
    final filterAsync = ref.watch(
      trailFilterProvider('profile_trail_$handle'),
    );
    final filter = filterAsync.value;

    return await _fetchPage(handle: handle, page: 1, q: _q, filter: filter);
  }

  Future<void> search(String q) async {
    _q = q;
    state = const AsyncLoading();
    final filter = ref
        .read(trailFilterProvider('profile_trail_$_handle'))
        .value;
    state = await AsyncValue.guard(
      () => _fetchPage(handle: _handle, page: 1, q: _q, filter: filter),
    );
  }

  Future<void> loadNextPage() async {
    final currentState = state.value;

    if (currentState == null || state.isLoading || !currentState.hasMore) {
      return;
    }

    state = await AsyncValue.guard(() async {
      final filter = ref
          .read(trailFilterProvider('profile_trail_$_handle'))
          .value;
      final nextPage = currentState.page + 1;
      final responseState = await _fetchPage(
        handle: _handle,
        page: nextPage,
        q: _q,
        filter: filter,
      );
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
    required String q,
    TrailFilter? filter,
  }) async {
    final api = ref.read(apiProvider);
    const int perPage = kProfileSearchPerPage;

    final filterText = filter?.toFilterText() ?? '';
    final sortParam = filter != null
        ? '${filter.sort.name}:${filter.sortOrder.name}'
        : 'created:desc';

    final response = await api.post(
      '/profile/$handle/trails',
      data: {
        'q': q,
        'options': {
          'hitsPerPage': perPage,
          'page': page,
          if (filterText.isNotEmpty) 'filter': filterText,
          'sort': [sortParam],
        },
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
