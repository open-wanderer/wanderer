import 'package:riverpod_annotation/riverpod_annotation.dart';

/// How much unseen content may remain below the viewport before the next page
/// is fetched, in logical pixels -- roughly three quarters of a phone screen.
///
/// Deliberately a DISTANCE, not the fraction (`pixels / maxScrollExtent >=
/// 0.8`) the scroll listeners used to compare. A fraction makes the trigger
/// distance scale with the list's current length, which is backwards at both
/// ends: on page 1 of a 20-row list, 20% left is only ~240px of runway, so the
/// user reaches the bottom and waits; after ten pages it is ~3800px, so pages
/// are fetched while the user is still dozens of rows away. A constant
/// distance behaves the same on page 1 and page 50.
const double kPagedPrefetchExtent = 600;

/// The shape every paged provider's state already had before this mixin
/// existed: a 1-based [page] cursor, a [totalPages] ceiling, and the
/// [hasMore] test derived from them.
///
/// Deliberately does NOT include the item list. The five paged states name it
/// differently (`items` / `lists` / `trails`) and two of them merge it in ways
/// nothing generic could express -- `ProfileTrailsState` splices in local
/// ObjectBox rows and dedupes network hits against them. Appending is the
/// per-provider half; see [PagedLoadMore.appendPage].
abstract interface class PagedState {
  int get page;
  int get totalPages;
  bool get hasMore;
}

/// Serialises "scroll to load more" so one gesture can only ever produce one
/// request.
///
/// ## Why an explicit flag rather than `state.isLoading`
///
/// Every `loadNextPage` here intentionally leaves the provider in `AsyncData`
/// while the next page is in flight, so the list stays on screen instead of
/// flickering to a spinner. That choice has a consequence that was missed
/// when it was made: `AsyncValue.guard` assigns to `state` only *after* its
/// future completes, so `state.isLoading` is false for the entire duration of
/// the fetch. It was also the only re-entrancy guard, which made it a no-op.
///
/// `ScrollController` listeners fire once per frame (60-120 Hz), so a single
/// fling past the trigger threshold called `loadNextPage` ~100+ times, each
/// call passing the dead guard and issuing a request for the *same*
/// `page + 1` -- `state.page` does not advance until the first response
/// lands. On 2026-08-05 that tripped the server's own rate limiting and
/// locked the developer's IP out of wanderer.to.
///
/// [_inFlight] is what `state.isLoading` was believed to be. `state.isLoading`
/// is still checked as well, because `search()` on some notifiers *does*
/// publish an `AsyncLoading` and a next-page fetch must not race it.
mixin PagedLoadMore<S extends PagedState> on $AsyncNotifier<S> {
  bool _inFlight = false;

  /// Clears the in-flight flag. Must be called at the top of every `build()`.
  ///
  /// The notifier instance survives rebuilds, so a flag left true by a fetch
  /// that was still running when the provider was invalidated would wedge
  /// pagination permanently -- every later `loadNextPage` returning at the
  /// guard, the list silently never growing again.
  void resetPaging() => _inFlight = false;

  /// Fetches [nextPage] and returns [current] with that page merged in.
  ///
  /// Implementations own the merge because they own the list field and any
  /// dedupe rules. They should NOT touch `state`, guard re-entrancy, or catch
  /// transport errors -- [loadNextPage] wraps this call in
  /// `AsyncValue.guard`, so a throw surfaces as `AsyncError` exactly as it
  /// did before.
  Future<S> appendPage(S current, int nextPage);

  /// Extra per-provider veto, checked after [PagedState.hasMore].
  ///
  /// Defaults to always allowing. `ProfileTrailsNotifier` overrides it to
  /// refuse paging while showing device-only content -- there is no server to
  /// page through.
  bool canLoadMore(S current) => true;

  Future<void> loadNextPage() async {
    final current = state.value;

    if (current == null ||
        _inFlight ||
        state.isLoading ||
        !current.hasMore ||
        !canLoadMore(current)) {
      return;
    }

    _inFlight = true;
    try {
      // Still no `AsyncLoading` here -- the seamless AsyncData -> AsyncData
      // transition is the point. It is safe now only because `_inFlight`,
      // not the state's own loading flag, is what holds the door shut.
      state = await AsyncValue.guard(
        () => appendPage(current, current.page + 1),
      );
    } finally {
      // `finally`, so a failed page does not wedge pagination the way a bare
      // reset after the await would have.
      _inFlight = false;
    }
  }
}
