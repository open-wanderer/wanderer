---
phase: quick-260808-vct
plan: 01
subsystem: ui
tags: [flutter, riverpod, filters, offline, meilisearch, objectbox]

requires: []
provides:
  - "TrailFilter.offlineOnly, emitted server-side by toFilterText() as an `id IN [...]` whitelist"
  - "downloadedTrailIdsProvider: the whole-library id-set view over trailLibraryProvider"
  - "downloadedTrailIdsForAuthorProvider: the author-narrowed set a profile search sends"
  - "offlineTrailIdsForMapSearch: the map's id-set selector (author-narrowed when a profile is in scope)"
  - "applyTrailFilter honors offlineOnly client-side for the local-merge half (unsynced captures)"
  - "Available offline toggle chip in TrailQuickFilterBar, second position, with a showOfflineChip opt-out"
affects: [trail-filtering, profile-trails, map-search, library-screen, offline-library]

tech-stack:
  added: []
  patterns:
    - "Device-local facts the server cannot know are pushed server-side as an explicit id whitelist rather than post-filtered client-side, so pagination, sort and relevance keep working"
    - "Ids interpolated into a Meilisearch filter string are whitelisted to a safe alphabet at the emission site, never escaped"
    - "Request-scoped disclosure is narrowed at the source when a route may proxy the body to a third-party instance"

key-files:
  created: []
  modified:
    - app/lib/models/trail.dart
    - app/lib/util/trail/filter.dart
    - app/lib/provider/trail/trail_library_provider.dart
    - app/lib/provider/profile/profile_trails_provider.dart
    - app/lib/provider/trail/map_trail_search_provider.dart
    - app/lib/provider/trail/map_cluster_search_provider.dart
    - app/lib/components/trail/trail_quick_filter_bar.dart
    - app/lib/routes/library_screen.dart
    - app/lib/routes/profile_trail_map_screen.dart
    - app/test/models/trail_filter_test.dart
    - app/test/util/trail/filter_test.dart

key-decisions:
  - "Filter server-side via `id IN [...]` rather than client-side. `id` is already a filterable attribute on the trails index (db/main.go:413), toFilterText() already emits `difficulty IN [...]`, and both search endpoints are POST — so the id list rides in the body with no URL-length ceiling. This is what lets the map honor the chip and keeps pagination/sort/relevance normal."
  - "Narrow the id set to the profile's author before sending. Not required for correctness — /profile/{handle}/trails already ANDs `author = <actor>` server-side — but that route proxies the request body verbatim to the origin instance for a federated actor, so an unnarrowed set would disclose this device's whole library to a third party."
  - "Fail CLOSED on an unresolvable actor id: emit a never-matching `id IN ['']` rather than omitting the clause. Omitting it would silently widen the search to every trail, the opposite of what the chip asks for."
  - "Keep the client-side clause in applyTrailFilter. An unsynced local capture has no server id and is absent from the Meilisearch index entirely, so the server-side whitelist can never return it; the isLocal branch covers it on the local-merge half."
  - "Hide the offline chip on the Library tab (showOfflineChip: false) — every row there is already downloaded, so the toggle would be inert, which reads as a bug."
  - "available_offline l10n key already existed (added by quick task 260720-s7m, commit dc6b98a9) with exactly the right semantics. Reused verbatim — no .arb edits, no gen-l10n run."

requirements-completed: [QUICK-260808-vct]

completed: 2026-08-08
---

# Quick Task 260808-vct: Available Offline Filter Chip

**An "Available offline" toggle chip (second position, right after Sort) that narrows trail lists to trails whose data is already on this device, implemented as a server-side `id IN [...]` whitelist so pagination, sort and map clustering all keep working.**

## How it works

The server tracks nothing about downloads. `TrailEntity.savedByUserIds` is device-local and appears nowhere in `db/` or `web/`, so there is no server-side field to filter on. The device therefore names the ids itself:

1. `trailLibraryProvider` (already account-scoped, already returns `const []` when signed out) is the source of truth.
2. `downloadedTrailIdsForAuthorProvider` narrows that to the trails authored by the profile being viewed.
3. `TrailFilter.toFilterText(offlineTrailIds: ...)` emits `id IN ['a', 'b', ...]`.
4. Meilisearch does the filtering; the list paginates and sorts exactly as it does with any other filter.

The client-side clause in `applyTrailFilter` is retained, but only for the local-merge half — an unsynced capture has no server id and cannot appear in the index, so it is matched by the `isLocal` branch instead.

## Why server-side

An earlier client-side-only implementation was built and then reverted unpushed. It sourced the profile list entirely from device rows and pinned `totalPages = 1`. Three things were wrong with it:

- **It abandoned pagination.** The whole library came back in one page — the only filter in the app to do so.
- **It downgraded text search.** With the chip on, `q` went through a local substring match instead of Meilisearch, so ranking silently differed from every other filter combination.
- **The offline property it was built for was illusory.** The network fetch runs first either way, and `_fetchAndMerge` rethrows `DioException` for any non-own handle before a device-only path is reached. Both designs degrade identically offline.

It also forced the map to opt out, on the grounds that server-aggregated clusters cannot be intersected against a device-local id set. That is true of a client-side design and irrelevant to this one: with the whitelist evaluated server-side, clusters are aggregated over the already-filtered set.

## Security

| Threat | Disposition | Mitigation |
|---|---|---|
| Library disclosure to a **third-party instance** | mitigate | `/profile/{handle}/trails` proxies the request body verbatim to the origin instance for a federated actor. The id set is narrowed to that profile's author before it is sent, so the disclosure is bounded to "which of your trails I have cached". |
| Unnarrowed fallback on an unresolved actor | mitigate | `_resolveOfflineTrailIds` fails **closed** — an empty set emits a never-matching clause rather than an unfiltered search. Correctness is unaffected either way because the server ANDs `author = <actor>` itself. |
| Filter-syntax injection from a local row | mitigate | Ids are whitelisted to `[A-Za-z0-9_-]` before interpolation; a row carrying anything else is dropped, not escaped. Covered by a test that feeds a quote-breaking id. |
| Cross-account leakage | mitigate | Account scoping is inherited from `trailLibraryProvider`, never re-implemented. |

The map endpoints (`/search/trails`, `/search/trails/cluster`) are served by the user's own instance against its local index and are never proxied, which is why the whole-library fallback is safe there.

## Deviations / notes

### Pre-existing bug observed, deliberately NOT fixed

`TrailQuickFilterBar._isAnyActive` omits `_isDistanceActive` from its OR-chain, so a distance-only filter shows no "Reset" button. `_isOfflineActive` was added to the chain (required — otherwise Reset would not appear when only the new chip is active), but the distance omission was left exactly as found. Worth a separate task.

### Flaky test, watch it

`test/provider/trail/map_search_deletion_test.dart` failed twice early in this work and then passed on five consecutive runs. It waits a real wall-clock 600ms for a 400ms debounce, so it is load-sensitive. Baseline (455329a2) passed 3/3; this branch passed 3/5 during heavy local load and 3/3 since. The code path it exercises gains no work from this change when the chip is off (`offlineTrailIdsForMapSearch` returns `const {}` on its first line), so a causal link is unlikely — but it has not been positively excluded.

### Stale generated files in the repo

`dart run build_runner build` rewrites ~13 `.g.dart`/`.freezed.dart` files whose doc comments have drifted from their sources (they were committed before an unrelated comment edit). Those were restored to their committed state so this commit stays focused; a future codegen run will sync them.

## Verification performed

- `flutter analyze` — 5 issues, all pre-existing and all in files this task does not touch (`summit_log_card.dart`, `actor_entity.dart`, `navigation_stats_provider.dart`, `local_photo_store.dart`, `util/local/id.dart`)
- `flutter test` — 1053 passed, 1 pre-existing skip, 0 failures
- New coverage: 5 cases in `test/models/trail_filter_test.dart` for the `id IN [...]` emission (flag off, flag on, empty set, injection-shaped id, well-formed ids all survive), plus the 5 existing `applyTrailFilter` cases for the client-side half
- No `flutter build` or `adb install` was run

## Manual verification (not run — needs a device)

1. Own profile trails list: chip order reads Sort, Available offline, Categories, Difficulty, Distance, Elevation gain, Date, Completion status.
2. Chip is visually identical to its neighbours (height, radius, avatar size, active/inactive colours).
3. Tap it → turns active, list narrows to downloaded/recorded trails, "Reset" appears.
4. **Scroll to the bottom of the filtered list — it should paginate normally.** (This is the behaviour the reverted design got wrong.)
5. Type a search query with the chip on — ranking should match what you get with the chip off.
6. Tap "Reset" → chip goes inactive, full list returns.
7. Airplane mode, chip on → every trail shown actually opens.
8. Another hiker's profile (online), chip on → only their trails that you have downloaded.
9. **Open the map from a profile trails list → the chip IS present, and clusters shrink to match.**
10. Library tab → chip is NOT present.
