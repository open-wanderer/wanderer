import 'package:dio/dio.dart';
import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_summary.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/profile/profile_constants.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/util/current_account.dart';
import 'package:wanderer/util/local_trail_store.dart';
import 'package:wanderer/util/own_trails_merge.dart';

part 'profile_trails_provider.freezed.dart';
part 'profile_trails_provider.g.dart';

@freezed
abstract class ProfileTrailsState with _$ProfileTrailsState {
  const factory ProfileTrailsState({
    required List<TrailSummary> trails,
    required int page,
    required int perPage,
    required int totalPages,

    /// True when the last network fetch failed and this state is showing
    /// only what's on this device (REC-06). Decided from the fetch outcome
    /// itself, never from `onlineStatusProvider`'s optimistic default
    /// (RESEARCH.md Pitfall 5).
    @Default(false) bool offline,

    /// True when this state is for the signed-in hiker's own handle --
    /// only then does the local half of the merge run (T-36-07-02).
    @Default(false) bool isOwnHandle,
  }) = _ProfileTrailsState;

  const ProfileTrailsState._();
  bool get hasMore => page < totalPages;
}

/// One page of the network half, kept separate from [ProfileTrailsState] so
/// `_fetchPage` can stay a pure fetch/parse step reused verbatim by
/// `build`/`search`/`loadNextPage` (36-PATTERNS.md: the network half is
/// unchanged by this plan).
typedef _NetworkPage = ({
  List<TrailSearchResult> trails,
  int page,
  int totalPages,
});

@riverpod
class ProfileTrailsNotifier extends _$ProfileTrailsNotifier {
  late String _handle;
  String _q = '';

  // Recomputed at the top of every `build()`. `_isOwnHandle` is published on
  // the state and read by `_fetchAndMerge`'s rethrow decision; the local read
  // path does NOT use it -- `_readOwnLocal` re-derives both the own-handle
  // test and the actor id from a fresh `authProvider`/`currentAccountId`
  // read, which is D-13's "always fresh, never cached" invariant actually
  // enforced rather than merely asserted.
  bool _isOwnHandle = false;
  String? _authorActorId;

  @override
  FutureOr<ProfileTrailsState> build(String handle) async {
    _handle = handle;

    // Watch the filter so we rebuild and re-fetch when it changes.
    final filterAsync = ref.watch(trailFilterProvider('profile_trail_$handle'));
    final filter = filterAsync.value;

    final store = ref.watch(objectBoxProvider);
    final accountId = currentAccountId(store);
    final user = ref.watch(authProvider).value;
    final preferredUsername = user?.preferredUsername;

    _isOwnHandle =
        accountId != null &&
        preferredUsername != null &&
        handle == '@$preferredUsername';
    _authorActorId = user?.actorId;

    final local = _isOwnHandle
        ? filterOwnTrailsByQuery(
            readOwnLocalTrails(
              store,
              accountId: accountId!,
              authorActorId: _authorActorId,
            ),
            _q,
          )
        : const <Trail>[];

    return _fetchAndMerge(local: local, page: 1, q: _q, filter: filter);
  }

  Future<void> search(String q) async {
    _q = q;
    state = const AsyncLoading();

    final local = _readOwnLocal(_q);
    final filter = ref
        .read(trailFilterProvider('profile_trail_$_handle'))
        .value;
    state = await AsyncValue.guard(
      () => _fetchAndMerge(local: local, page: 1, q: _q, filter: filter),
    );
  }

  Future<void> loadNextPage() async {
    final currentState = state.value;

    if (currentState == null || state.isLoading || !currentState.hasMore) {
      return;
    }
    // Never poll for a next page while showing device-only content -- there
    // is no server to page through.
    if (currentState.offline) return;

    state = await AsyncValue.guard(() async {
      final filter = ref
          .read(trailFilterProvider('profile_trail_$_handle'))
          .value;
      final nextPage = currentState.page + 1;
      final fetched = await _fetchPage(
        handle: _handle,
        page: nextPage,
        q: _q,
        filter: filter,
      );

      // Keep the deduped local prefix intact -- only append network
      // results, and only ones not already represented by a local row.
      final local = _readOwnLocal(_q);
      final localIds = local
          .map((t) => t.id)
          .where((id) => id.isNotEmpty)
          .toSet();
      final dedupedNetwork = fetched.trails.where(
        (t) => !localIds.contains(t.id),
      );

      return currentState.copyWith(
        trails: [...currentState.trails, ...dedupedNetwork],
        page: fetched.page,
        totalPages: fetched.totalPages,
      );
    });
  }

  /// Re-reads [ProfileTrailsState]'s local half fresh from the store, gated
  /// on the handle being the signed-in hiker's own -- a different handle's
  /// profile, or a signed-out account, always gets an empty local list
  /// (T-36-07-01, T-36-07-02).
  ///
  /// Derives BOTH the own-handle test and the author actor id here rather
  /// than reading [_isOwnHandle]/[_authorActorId], which is what the fields'
  /// own doc comment promised and the code did not do. The actor id is a
  /// matching key in `readOwnLocalTrails`' second clause
  /// (`entity.author.target?.id == authorActorId`), so a stale one paired the
  /// NEW account's id with the PREVIOUS account's actor -- the exact shape of
  /// the leak D-13 exists to prevent. `build()` watching `authProvider` kept
  /// the window narrow, but narrow is not the guarantee the comment claimed.
  List<Trail> _readOwnLocal(String q) {
    final store = ref.read(objectBoxProvider);
    final accountId = currentAccountId(store);
    final user = ref.read(authProvider).value;
    if (accountId == null || user == null) return const <Trail>[];

    if (_handle != '@${user.preferredUsername}') return const <Trail>[];

    return filterOwnTrailsByQuery(
      readOwnLocalTrails(
        store,
        accountId: accountId,
        authorActorId: user.actorId,
      ),
      q,
    );
  }

  /// Fetches network page 1, merges it with [local], and reports whether the
  /// fetch failed.
  ///
  /// A failed TRANSPORT fetch for the signed-in hiker's own handle is
  /// swallowed (never rethrown) so the local half still renders with
  /// `offline: true` (REC-06) -- regardless of whether [local] happens to be
  /// empty, since the offline empty state is itself a valid rendered outcome.
  /// The same failure for another hiker's handle is rethrown, keeping today's
  /// error behaviour so that profile does not silently render empty.
  ///
  /// Only [DioException] counts. A blanket `catch (_)` also swallowed the
  /// `FormatException` `_fetchPage` throws deliberately for an unexpected
  /// response shape, and the implicit `TypeError` from a non-int
  /// `totalPages` -- so a client- or server-side BUG rendered as
  /// "Showing what's saved on this device — connect to see everything" on a
  /// fully-online device, and `loadNextPage` then refused to page because
  /// `offline` was true. [ProfileTrailsState.offline] is documented as
  /// "decided from the fetch outcome itself"; it was decided from any
  /// outcome.
  Future<ProfileTrailsState> _fetchAndMerge({
    required List<Trail> local,
    required int page,
    required String q,
    TrailFilter? filter,
  }) async {
    List<TrailSearchResult> networkTrails = const [];
    int resultPage = 1;
    int totalPages = 1;
    bool offline = false;

    try {
      final fetched = await _fetchPage(
        handle: _handle,
        page: page,
        q: q,
        filter: filter,
      );
      networkTrails = fetched.trails;
      resultPage = fetched.page;
      totalPages = fetched.totalPages;
    } on DioException catch (_) {
      if (!_isOwnHandle) rethrow;
      offline = true;
    }

    return ProfileTrailsState(
      trails: mergeOwnTrails(local: local, network: networkTrails),
      page: resultPage,
      perPage: kProfileSearchPerPage,
      totalPages: totalPages,
      offline: offline,
      isOwnHandle: _isOwnHandle,
    );
  }

  Future<_NetworkPage> _fetchPage({
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

    return (
      trails: hits.map((json) => TrailSearchResult.fromJson(json)).toList(),
      page: page,
      totalPages: totalPages,
    );
  }
}
