import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';

part 'trail_search_provider.freezed.dart';
part 'trail_search_provider.g.dart';

@freezed
abstract class TrailSearchState with _$TrailSearchState {
  const factory TrailSearchState({
    required List<TrailSearchResult> trails,
    required int page,
    required int perPage,
    required int totalPages,
  }) = _TrailSearchState;

  const TrailSearchState._();
  bool get hasMore => page < totalPages;
}

@riverpod
class TrailSearchNotifier extends _$TrailSearchNotifier {
  @override
  FutureOr<TrailSearchState> build() async {
    return await _fetchPage(page: 1);
  }

  Future<void> loadNextPage() async {
    final currentState = state.value;

    if (currentState == null || state.isLoading || !currentState.hasMore) {
      return;
    }

    state = AsyncLoading();
    state = await AsyncValue.guard(() async {
      final nextPage = currentState.page + 1;
      final responseState = await _fetchPage(page: nextPage);
      return currentState.copyWith(
        trails: [...currentState.trails, ...responseState.trails],
        page: responseState.page,
        totalPages: responseState.totalPages,
      );
    });
  }

  Future<TrailSearchState> _fetchPage({required int page}) async {
    final api = ref.read(apiProvider);
    final filter = await ref.watch(trailFilterProvider.future);
    final user = await ref.watch(authProvider.future);

    const int perPage = 20;

    final response = await api.post(
      '/search/trails',
      data: {
        'q': filter.q,
        'options': {
          'filter': filter.toFilterText(actor: user?.actorId ?? ""),
          'sort': ["${filter.sort.name}:${filter.sortOrder.name}"],
          'hitsPerPage': perPage,
          'page': page,
        },
      },
    );

    final List<dynamic> hits = response.data['hits'] ?? [];

    final int totalPages = response.data['totalPages'] ?? 1;

    return TrailSearchState(
      trails: hits.map((json) => TrailSearchResult.fromJson(json)).toList(),
      page: page,
      perPage: perPage,
      totalPages: totalPages,
    );
  }
}
