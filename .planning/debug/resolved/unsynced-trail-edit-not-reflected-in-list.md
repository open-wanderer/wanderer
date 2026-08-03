---
status: resolved
trigger: "After saving edits on a non synced trail the 'own trail' list needs a manual reload before edits are shown"
created: 2026-08-02T00:00:00Z
updated: 2026-08-03T16:45:00Z
---

## Current Focus

hypothesis: CONFIRMED — the local-first save path in `trail_create_screen.dart` never invalidates `profileTrailsProvider`, and the own-trails list reads ObjectBox as a one-shot query with no reactive watcher, so the already-mounted list keeps its cached `ProfileTrailsState` when the edit screen pops.
test: exhaustive grep of every `invalidate(profileTrailsProvider(...))` call site + trace of the save path + confirmation that ObjectBox holds the new value (proved by "manual reload shows the edits")
expecting: zero invalidation on the local-save path — confirmed
next_action: hand off to plan-phase --gaps; no fix applied (goal: find_root_cause_only)

## Symptoms

expected: After saving edits to an unsynced (not-yet-uploaded) trail, the own-trails list reflects the new values immediately on navigating back — no manual pull-to-refresh or re-navigation required.
actual: The own-trails list shows stale values (e.g. the old title) until the user pull-to-refreshes.
errors: none reported
reproduction: Test 2 in `.planning/phases/36-local-first-recording-automatic-upload/36-UAT.md` — open an unsynced trail from `/profile/<handle>/trails`, change its title, save, pop back to the list.
started: Discovered during UAT of Phase 36 (local-first recording + automatic upload)

## Eliminated

- hypothesis: The ObjectBox write never lands (e.g. `updateLocalTrail` returns `LocalUpdateOutcome.missing`, which `_onSave` treats identically to `updated` and still shows the success toast).
  evidence: The user's own report says a MANUAL RELOAD shows the edits. Pull-to-refresh (`profile_trail_screen.dart:127`) is a bare `ref.invalidate(profileTrailsProvider(handle))` — it re-runs the same `readOwnLocalTrails` read against the same store, no network involved when offline. If ObjectBox did not hold the new values, the manual reload would show stale data too. The write path is sound; only propagation fails.
  timestamp: 2026-08-02

- hypothesis: Invalidation targets a different provider than the one the list watches (e.g. the remote-backed provider is invalidated while the local/merged one is not).
  evidence: There is exactly ONE own-trails provider (`profileTrailsProvider`, `profile_trails_provider.dart:53`), and `profile_trail_screen.dart:66` watches it directly. The save path invalidates nothing at all, so there is no mis-targeted invalidation — there is no invalidation.
  timestamp: 2026-08-02

- hypothesis: The merge drops the updated local row (`mergeOwnTrails` dedupe suppressing it).
  evidence: `own_trails_merge.dart:40` builds the dedupe set from NON-EMPTY local ids and always emits `[...local, ...dedupedNetwork]` — the local half is never filtered out. Also eliminated by the manual-reload observation.
  timestamp: 2026-08-02

- hypothesis: The provider auto-disposes when the edit screen is pushed and simply fails to rebuild correctly on return.
  evidence: `isAutoDispose: true` (`profile_trails_provider.g.dart:23`), but `context.push('/trail/create/edit', ...)` (`profile_trail_screen.dart:58`) uses a default `GoRoute` builder → `MaterialPage` with `maintainState: true`. The list screen's `ConsumerState` stays mounted underneath, so its `ref.watch` subscription persists and auto-dispose never fires. The provider is kept alive holding its stale `AsyncData` — which is precisely why nothing rebuilds.
  timestamp: 2026-08-02

## Evidence

- timestamp: 2026-08-02
  checked: `app/lib/provider/profile/profile_trails_provider.dart` build() (lines 67-97) and `_readOwnLocal` (166-182)
  found: The local half is a one-shot synchronous ObjectBox read — `readOwnLocalTrails(store, ...)` — evaluated once per `build()`. `build()` watches only `trailFilterProvider`, `objectBoxProvider` and `authProvider`; none of those change when a trail row is written.
  implication: A row mutation in ObjectBox produces no signal. The list can only pick up local writes when the provider is explicitly invalidated (or first built).

- timestamp: 2026-08-02
  checked: `grep -rn "\.watch()" app/lib` for ObjectBox `Query.watch()` / reactive streams
  found: Zero results — the app has no ObjectBox reactive watchers anywhere. Every read is a one-shot `query.find()`.
  implication: There is no stream-based propagation path in this codebase at all; `ref.invalidate` is the only mechanism, so an omitted invalidate is a total propagation failure, not a delayed one.

- timestamp: 2026-08-02
  checked: `grep -n "invalidate\|refresh(" app/lib/routes/trail_create_screen.dart`
  found: NO MATCHES. The edit/save screen never invalidates any provider.
  implication: The save path has no notification side effect whatsoever.

- timestamp: 2026-08-02
  checked: `_onSave` local branches (`trail_create_screen.dart:451-506` createLocal, `:508-569` updateLocal) and their shared tail `_finishLocalSave` (`:714-762`)
  found: `_finishLocalSave` does exactly four things — fires `trailSyncProvider.drainIfOnline()`, reports photo-copy failures, re-reads the row into THIS screen's own `trail` field via `setState`, resets the form, shows the success toast. It refreshes only local widget state; it never touches `profileTrailsProvider` or `trailLibraryProvider`.
  implication: This is the exact missing link. The write commits, the editing screen updates itself, and every other consumer of that row is left holding a stale snapshot.

- timestamp: 2026-08-02
  checked: exhaustive `grep -rn "invalidate(profileTrailsProvider" app/lib` — all call sites
  found: Only three: `trail_sync_provider.dart:303` (after a drain uploads a trail successfully), `trail_sync_provider.dart:377` (after `deleteUnsynced`), `trail_dropdown.dart:318` (after a server-side delete). Plus the user-driven `RefreshIndicator` at `profile_trail_screen.dart:127`.
  implication: There is no edit-save invalidation site. Offline (UAT Test 2 is run in airplane mode) the drain never succeeds, so `:303` never fires either — leaving pull-to-refresh as the ONLY route to a fresh list, which is verbatim what the user reported.

- timestamp: 2026-08-02
  checked: `trail_dropdown.dart:107-110` — the other entry point into the same edit screen
  found: `await context.push('/trail/create/edit', extra: trail); ref.invalidate(trailProvider(trail.id));` — the caller compensates for the screen's silence, but only for the single-trail provider, never for the list provider.
  implication: Confirms the invalidation responsibility was left to callers and was implemented inconsistently. `profile_trail_screen.dart:58` pushes the same route with no `await` and no invalidate at all.

- timestamp: 2026-08-02
  checked: Why UAT Test 1 ("trail appears once in the own-trails list after first save") passed while this fails
  found: After a fresh capture the own-trails list is not yet mounted, so `profileTrailsProvider` is built for the first time when the user navigates to it and the local read naturally includes the new row. In the edit case the list is already mounted beneath the pushed edit screen (`maintainState: true`), keeping the auto-dispose provider alive with its stale value.
  implication: Explains the create-works/update-fails asymmetry without needing a second mechanism — one root cause covers both observations.

- timestamp: 2026-08-02
  checked: `grep -rn "RouteAware|RouteObserver|didPopNext" app/lib`
  found: No results. No route-observer-based refresh-on-return exists.
  implication: Nothing else can rescue the stale state on pop. Rules out the last alternative propagation path.

## Resolution

root_cause: |
  `TrailCreateScreen`'s local-first save path commits the edit to ObjectBox
  (`updateLocalTrail`, `local_trail_store.dart:234`) but its shared post-save
  tail `_finishLocalSave` (`trail_create_screen.dart:714-762`) never calls
  `ref.invalidate(profileTrailsProvider('@<handle>'))` — the file contains no
  `invalidate` call at all. Because `ProfileTrailsNotifier.build()`
  (`profile_trails_provider.dart:85-96`) reads the local half via a one-shot
  `readOwnLocalTrails()` query and the codebase has no ObjectBox
  `Query.watch()` streams anywhere, an ObjectBox row mutation emits no signal.
  The list screen also stays mounted beneath the pushed edit route
  (`maintainState: true`), so its `ref.watch` keeps the auto-dispose provider
  alive with the pre-edit `ProfileTrailsState`. The only three invalidation
  sites (`trail_sync_provider.dart:303`/`:377`, `trail_dropdown.dart:318`)
  cover successful upload and deletion, none of which occurs on an offline
  edit — leaving the `RefreshIndicator` at `profile_trail_screen.dart:127` as
  the sole path to fresh data, exactly the "manual reload" the user reported.
fix: NOT APPLIED (goal: find_root_cause_only)
verification: n/a
files_changed: []
