---
status: diagnosed
trigger: "The own trails list shows a spinner every so often"
created: 2026-08-02
updated: 2026-08-02
---

## Current Focus

hypothesis: CONFIRMED — `trailFilterProvider` throws on a failed `GET /trail/filter`; Riverpod 3's automatic retry (`ProviderContainer.defaultRetry`, 10 attempts, 200ms→6400ms backoff) re-emits its state repeatedly. `ProfileTrailsNotifier.build` **watches** that provider, so each emission is a dependency change (`isReload: true`), which makes the new state a real `AsyncLoading` (not a seamless `AsyncData(isLoading: true)`), and `AsyncValue.when`'s `skipLoadingOnReload` defaults to `false` — so the screen renders the full-screen `CircularProgressIndicator` on every one.
test: Traced riverpod 3.2.1 source for reload-vs-refresh transition semantics and the default retry policy; traced every watched dependency of `ProfileTrailsNotifier.build`.
expecting: A dependency that re-emits on a timer during offline/flaky use.
next_action: Diagnosis complete — hand to plan-phase --gaps. No fix applied (goal: find_root_cause_only).

## Symptoms

expected: `/profile/<handle>/trails` own-trails list renders stably, no repeated spinner flashes.
actual: "The own trails list shows a spinner every so often"
errors: none reported (the failing `GET /trail/filter` is swallowed into an `Exception` that only the filter bar surfaces)
reproduction: Phase 36 UAT Test 2 — in airplane mode, open your own profile's own-trails list (`/profile/<handle>/trails`) and watch for ~45 s.
started: Discovered during UAT of Phase 36 (local-first recording + automatic upload). The offline own-trails path is what made the screen reachable-and-populated while offline, which is what exposes the pre-existing retry/reload interaction.

## Eliminated

- hypothesis: The background upload drain's `ref.invalidate(profileTrailsProvider(...))` causes the spinner.
  evidence: `Ref.invalidate` defaults to `asReload: false` (riverpod-3.2.1 `lib/src/core/ref.dart:322`), so `_didChangeDependency` stays false, `_performRebuild` passes `isReload: false` (`element.dart:572`), `asyncTransition` runs with `seamless: true`, and `AsyncLoading.copyWithPrevious(previous, isRefresh: true)` returns an `AsyncData` with `loading: true` (`async_value.dart:54-61`). That is `isRefreshing`, and `when`'s `skipLoadingOnRefresh` defaults to **true** — so invalidation is silent. `trail_sync_provider.dart:303,377` and `trail_dropdown.dart:318` are all plain `ref.invalidate`.
  timestamp: 2026-08-02

- hypothesis: `objectBoxProvider` or an ObjectBox stream/query subscription re-emits.
  evidence: `objectbox_store_provider.dart` is a `@Riverpod(keepAlive: true) Notifier<Store>` whose `build()` throws and is replaced by `overrideWithValue(store)` in `main.dart:58` — it never emits again. `readOwnLocalTrails` is a synchronous one-shot query, not a `Stream`/watcher.
  timestamp: 2026-08-02

- hypothesis: `authProvider` re-emits periodically (token refresh).
  evidence: `Auth` is `keepAlive`; its state only changes on `login`/`register`/`loginWithOAuth`/`logout`/`refresh()`. The only `refresh()` callers are `settings_account_screen.dart:54` and `email_change_sheet.dart:64` — both user-initiated. No timer, no lifecycle hook touches it.
  timestamp: 2026-08-02

- hypothesis: `onlineStatusProvider` connectivity transitions retrigger the list.
  evidence: `ProfileTrailsNotifier.build` does not watch `onlineStatusProvider` (deliberately — see its own doc comment at `profile_trails_provider.dart:29`). No transitive watch either: `apiProvider` only `ref.read`s the notifier lazily inside interceptor callbacks.
  timestamp: 2026-08-02

- hypothesis: Un-debounced search bar fires `search()` per keystroke.
  evidence: `wanderer_searchbar.dart:21-32` debounces at 500 ms. (`search()` is still a spinner source — see Evidence — but not a per-keystroke one, and it is user-driven, not "every so often".)
  timestamp: 2026-08-02

## Evidence

- timestamp: 2026-08-02
  checked: `app/lib/provider/profile/profile_trails_provider.dart` (full read)
  found: `build()` is async and watches exactly three providers — `trailFilterProvider('profile_trail_$handle')` (:71), `objectBoxProvider` (:74), `authProvider` (:76). Any emission from any of them re-runs the whole async build, including the network `POST /profile/$handle/trails`.
  implication: The spinner source must be whichever of these three re-emits during idle use.

- timestamp: 2026-08-02
  checked: `app/lib/routes/profile_trail_screen.dart:90` and `:173`
  found: The screen renders `trailsAsync.when(data: ..., loading: () => const Center(child: CircularProgressIndicator()), error: ...)` with **no** `skipLoadingOnReload` argument.
  implication: This is the exact widget the user sees flash. `when`'s defaults are the deciding factor.

- timestamp: 2026-08-02
  checked: riverpod-3.2.1 `lib/src/core/async_value.dart:86,97,242-260`
  found: `isRefreshing => _hasState && isLoading && this is! AsyncLoading`; `isReloading => _hasState && isLoading && this is AsyncLoading`. `when(skipLoadingOnReload = false, skipLoadingOnRefresh = true)` — a **reload** renders `loading()`, a **refresh** does not.
  implication: Dependency-driven rebuilds show the spinner; `ref.invalidate` does not. The two paths are not interchangeable.

- timestamp: 2026-08-02
  checked: riverpod-3.2.1 `lib/src/core/element.dart:50,572,749-752` and `lib/src/core/ref.dart:322,657`
  found: `ref.watch` is implemented as `listen(... => invalidateSelf(asReload: true))` (ref.dart:657). `invalidateSelf(asReload: true)` sets `_didChangeDependency = true` (element.dart:752), `_performRebuild` builds a `$Ref(isReload: _didChangeDependency)` (element.dart:572), and the async transition uses `seamless: !ref.isReload` (element.dart:50). With `seamless: false`, `AsyncLoading.copyWithPrevious(previous, isRefresh: false)` returns a genuine `AsyncLoading` carrying the previous value (async_value.dart, else branch).
  implication: CONFIRMED MECHANISM — every emission of a watched dependency puts `profileTrailsProvider` into `isReloading`, which `when` renders as the full-screen spinner even though the previous trail list is still held.

- timestamp: 2026-08-02
  checked: `app/lib/provider/trail/trail_filter_provider.dart:20-61`
  found: `TrailFilterNotifier.build` does `await api.get('/trail/filter')` inside a `try`, and its `catch (e)` at :60 rethrows as `throw Exception('Failed to fetch trail filters: $e')`. Offline (UAT Test 2 is explicitly airplane mode) or on any flaky request, this build **fails**.
  implication: The filter provider enters `AsyncError`. It is `@Riverpod(keepAlive: true)`, so it stays alive and keeps re-emitting.

- timestamp: 2026-08-02
  checked: riverpod-3.2.1 `lib/src/core/element.dart:699-720` and `lib/src/core/provider_container.dart:831-845`
  found: On a failed async build, `triggerRetry` schedules `Timer(duration, () { _retryCount++; invalidateSelf(asReload: false); })`. `ProviderContainer.defaultRetry` retries up to `maxRetries = 10` with `minDelay = 200ms`, doubling, capped at `maxDelay = 6400ms` — bailing out only when `error is ProviderException || error is Error`. `Exception('Failed to fetch trail filters: ...')` is a plain `Exception`, so it is fully retryable.
  implication: `trailFilterProvider` re-emits on a 200ms / 400ms / 800ms / 1.6s / 3.2s / 6.4s / 6.4s ... schedule — ~10 retry cycles spread over roughly 45 seconds. Each cycle emits at least twice (error→loading, loading→error, with a fresh `Exception` instance each time so `updateShouldNotify` always fires). That is ~20 dependency changes, i.e. ~20 spinner flashes at irregular, lengthening intervals — the literal shape of "shows a spinner every so often".

- timestamp: 2026-08-02
  checked: `app/lib/main.dart:56-64` (ProviderScope) and a repo-wide grep for `skipLoadingOnReload` / `skipLoadingOnRefresh` / `@Riverpod(retry:)`
  found: `ProviderScope` passes only `overrides:` — no `retry:` override, so `defaultRetry` applies container-wide. Zero occurrences of `skipLoadingOnReload`/`skipLoadingOnRefresh` anywhere in `app/lib`, and no provider sets a custom `retry`.
  implication: Nothing in the app opts out of either half of the mechanism. Every `.when()` in the codebase has the same latent exposure; the own-trails list is where a retrying dependency actually sits underneath one.

- timestamp: 2026-08-02
  checked: `app/lib/provider/profile/profile_trails_provider.dart:99-110` (`search`) and :71-73 (initial filter watch)
  found: (a) `search()` assigns a **bare** `state = const AsyncLoading()` (:101) with no previous value, so `hasValue` is false and `when` renders `loading()` unconditionally — a guaranteed full-screen spinner per (debounced) search burst. (b) `trailFilterProvider` is async, so on first entry it emits `AsyncLoading` then `AsyncData`; `build()` runs once with `filter == null` and again with the real filter — one guaranteed spinner plus a wasted `POST /profile/$handle/trails` on every screen entry.
  implication: Two additional, independent spinner sources sharing the same root shape (list state dropped to loading while data is in hand). A complete fix should address all three, but the retry storm is the one matching "every so often".

- timestamp: 2026-08-02
  checked: `app/lib/provider/trail/trail_sync_provider.dart:96` and `app/lib/main.dart:129-136`
  found: Each `profileTrailsProvider` reload re-runs `_fetchAndMerge` → a fresh failing `POST`, whose Dio interceptor (`api_provider.dart:63-70`) calls `markOffline()`. `main.dart`'s `_onlineStatusSub` kicks `drainIfOnline()` on any false→true transition.
  implication: Secondary blast radius — the retry storm also generates ~20 pointless network attempts and can drive spurious upload-drain passes while the user just sits on the list.

## Resolution

root_cause: |
  `ProfileTrailsNotifier.build` (`app/lib/provider/profile/profile_trails_provider.dart:71`) `ref.watch`es
  `trailFilterProvider('profile_trail_$handle')`, whose `build` throws a plain `Exception` whenever
  `GET /trail/filter` fails (`app/lib/provider/trail/trail_filter_provider.dart:60`) — which it always does
  offline. Riverpod 3's automatic retry (`ProviderContainer.defaultRetry`: 10 attempts, 200ms doubling to a
  6400ms cap) then re-emits the filter provider's state ~20 times over ~45 s. Because those emissions arrive
  through `ref.watch`, riverpod classifies each resulting rebuild as a RELOAD, not a refresh
  (`element.dart:572` `isReload: _didChangeDependency` → `element.dart:50` `seamless: !ref.isReload`), so
  `profileTrailsProvider` transitions to a genuine `AsyncLoading` rather than a seamless
  `AsyncData(isLoading: true)`. `profile_trail_screen.dart:90` uses `AsyncValue.when` with default
  `skipLoadingOnReload: false`, so each of those reloads renders
  `loading: () => const Center(child: CircularProgressIndicator())` — the spinner the user sees, flashing at
  irregular, lengthening intervals until the retry budget is exhausted.
fix: (not applied — goal: find_root_cause_only)
verification: (n/a)
files_changed: []
