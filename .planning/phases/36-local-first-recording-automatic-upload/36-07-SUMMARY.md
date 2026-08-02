---
phase: 36-local-first-recording-automatic-upload
plan: 07
subsystem: mobile-profile
tags: [flutter, riverpod, objectbox, local-first, own-trails, offline]

# Dependency graph
requires:
  - phase: 36-local-first-recording-automatic-upload
    plan: 01
    provides: "TrailSyncState enum, isUnsyncedState, TrailEntity owner/localId/syncState schema"
  - phase: 36-local-first-recording-automatic-upload
    plan: 03
    provides: "local_trail_store.dart's readOwnLocalTrails (owner-scoped)"
provides:
  - "own_trails_merge.dart: mergeOwnTrails/filterOwnTrailsByQuery -- pure local+network merge with server-id dedupe"
  - "profile_trails_provider.dart: local-first own-trails state carrying offline/isOwnHandle flags"
  - "profile_trail_screen.dart: offline banner, offline empty state, unsynced-trail tap routing to /trail/create/edit"
affects: [36-08]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "mergeOwnTrails prepends local rows and dedupes network hits by non-empty local id -- same SYNC-05 identity-preservation idea local_trail_store.dart already uses for the write side, applied to the read/merge side"
    - "_fetchPage retyped to return a private (List<TrailSearchResult>, page, totalPages) record instead of a whole ProfileTrailsState, so the network fetch/parse step stays byte-identical while the state shape absorbs the new offline/isOwnHandle fields"
    - "own-handle comparison keyed on the exact '@$preferredUsername' string trail_dropdown.dart already invalidates with, not a separate handle-normalization helper"

key-files:
  created:
    - app/lib/util/own_trails_merge.dart
    - app/test/util/own_trails_merge_test.dart
  modified:
    - app/lib/provider/profile/profile_trails_provider.dart
    - app/lib/provider/profile/profile_trails_provider.freezed.dart
    - app/lib/provider/profile/profile_trails_provider.g.dart
    - app/lib/routes/profile_trail_screen.dart

key-decisions:
  - "ProfileTrailsNotifier keeps mutable (non-late-final) _isOwnHandle/_authorActorId fields set fresh at the top of every build(), matching the file's existing _handle field precedent -- the Notifier survives rebuilds, so a late final field would throw on the second build()"
  - "A failed network fetch for the signed-in hiker's own handle is swallowed inside a private _fetchAndMerge helper (never rethrown), setting offline: true regardless of whether the local list happens to be empty -- REC-06's offline empty state is itself a valid rendered outcome, not just the non-empty local case"
  - "loadNextPage dedupes each subsequent network page against the already-established local id set (re-read fresh via _readOwnLocal) rather than only page 1 -- an uploaded local trail's server search hit could land on any page, not just the first"
  - "The offline empty-state widget was duplicated into profile_trail_screen.dart as a new private _OwnTrailsEmptyState (icon-only variant of library_screen.dart's _LibraryEmptyState) rather than promoted to a shared file -- plan explicitly allowed either, and files_modified scoped this plan to profile_trail_screen.dart only"

requirements-completed: [REC-02, REC-04, REC-05, REC-06, SYNC-05]

# Metrics
duration: ~25min
completed: 2026-08-02
---

# Phase 36 Plan 07: Local-first own-trails list Summary

**`/profile/<handle>/trails` is now local-first for the signed-in hiker's own handle: `mergeOwnTrails` prepends owner-scoped ObjectBox rows ahead of the network search page and dedupes by server id, `ProfileTrailsNotifier` swallows a failed fetch into an `offline: true` state instead of an error, and the screen renders a persistent offline banner, a device-only empty state, and routes an unsynced trail's tap to the offline-capable edit screen.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-08-02 (approx.)
- **Completed:** 2026-08-02
- **Tasks:** 3
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments
- `own_trails_merge.dart`: `mergeOwnTrails({local, network})` puts local rows first and drops any network hit whose id matches a local row's non-empty id (SYNC-05); `filterOwnTrailsByQuery` is the local half of the search filter (case-insensitive name/location match, empty/whitespace query returns input unchanged)
- `own_trails_merge_test.dart`: 10 tests covering ordering, dedupe (including the "two empty-id local rows suppress nothing" edge case), both-empty-input identity cases, and the query filter's empty/case-insensitive behaviour
- `profile_trails_provider.dart`: `ProfileTrailsState.trails` retyped `List<TrailSummary>`, gained `offline`/`isOwnHandle` fields; `build(handle)` reads `currentAccountId(store)` and the signed-in user's `preferredUsername`/`actorId` fresh every call, computes `isOwnHandle` from the exact `'@$preferredUsername'` string, reads+filters local rows only for the own handle, and merges with the network page via `mergeOwnTrails`; a failed network fetch for the own handle sets `offline: true` without rethrowing (REC-06), while a non-own handle keeps today's error-state behaviour; `search`/`loadNextPage` re-derive the local half fresh via a new `_readOwnLocal` helper; `loadNextPage` is guarded off while `offline` is true and appends only network results deduped against the local id set
- `profile_trail_screen.dart`: a persistent, non-dismissable banner (`surfaceContainerHighest` background, `cloudArrowUp` icon, 16/12 padding) renders above the list when `offline && isOwnHandle`; a new `_OwnTrailsEmptyState` (duplicating `library_screen.dart`'s `_LibraryEmptyState.icon` structure) replaces the bare empty state for an empty own-handle list; tapping a `Trail` whose `syncState` is unsynced (`isUnsyncedState`) now pushes `/trail/create/edit` with the trail as `extra` instead of `/trail/<empty-id>`

## Task Commits

Each task was committed atomically:

1. **Task 1: Pure local+network merge with server-id dedupe** - `d912623b` (feat)
2. **Task 2: Make the own-trails provider local-first with an offline flag** - `9826c8dd` (feat)
3. **Task 3: Offline banner, offline empty state and unsynced tap routing** - `138f1df2` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/util/own_trails_merge.dart` - pure `mergeOwnTrails`/`filterOwnTrailsByQuery`, no Riverpod/ObjectBox/filesystem dependency
- `app/test/util/own_trails_merge_test.dart` - 10 tests, plain `Trail(...)`/`TrailSearchResult(...)` fixtures only
- `app/lib/provider/profile/profile_trails_provider.dart` - local-first `build`/`search`/`loadNextPage`, new `offline`/`isOwnHandle` state fields, `_fetchPage` retyped to a private record
- `app/lib/provider/profile/profile_trails_provider.freezed.dart` / `.g.dart` - regenerated by `build_runner`
- `app/lib/routes/profile_trail_screen.dart` - offline banner, `_OwnTrailsEmptyState`, unsynced-tap routing

## Decisions Made
- Kept `_isOwnHandle`/`_authorActorId` as plain mutable instance fields (not `late final`) on `ProfileTrailsNotifier`, reassigned at the top of every `build()` call, mirroring the file's pre-existing `_handle` field -- required because the Notifier instance survives rebuilds and a `late final` would throw `LateInitializationError` on the second `build()`
- `_fetchAndMerge`'s try/catch swallows any exception for the own handle unconditionally (never rethrows), independent of whether `local` is non-empty -- the offline empty state (nothing saved, no connection) is itself a valid REC-06 outcome, not just a degraded-but-populated one
- `loadNextPage` re-reads the local id set fresh via `_readOwnLocal` on every call (not cached from the initial `build()`) and dedupes each subsequent network page against it, since an uploaded local trail's server search hit is not guaranteed to land on page 1
- The `_OwnTrailsEmptyState` widget is a file-local duplicate of `library_screen.dart`'s `_LibraryEmptyState.icon` (icon-only variant, no `.artwork` constructor since this screen never needs it) rather than a promoted shared widget -- this plan's `files_modified` scoped changes to `profile_trail_screen.dart` only, and the plan text explicitly permitted either approach

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. `flutter analyze --no-pub` reports zero errors project-wide (only the same pre-existing `info`-level lints already logged in prior phase summaries: deprecated Font Awesome constants in `icon_util.dart`, two dangling-library-doc-comment infos). The full `flutter test` suite (759 tests, 1 pre-existing skip unrelated to this plan) passed with no regressions.

## Device-Verified-Only Behaviours

Per the plan's own scope, the following are covered only by source-level grep gates and unit tests, not by a live-device pass:
- The offline banner and offline empty state actually rendering with a real airplane-mode network failure (only the `catch`/`offline: true` code path is exercised by static verification)
- Tapping an unsynced trail actually opening `/trail/create/edit` with its title and photos populated on a real device
- The merged list's actual on-device appearance (local-first ordering, no visible duplicate after a real upload completes)

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `own_trails_merge.dart`'s pure functions and `profile_trails_provider.dart`'s `offline`/`isOwnHandle` fields are available for 36-08 (sign-out warning) if it needs to reference the own-trails list's local-first state
- No blockers

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-02*

## Self-Check: PASSED

All 4 files created/modified in this plan verified present on disk (`app/lib/util/own_trails_merge.dart`, `app/test/util/own_trails_merge_test.dart`, `app/lib/provider/profile/profile_trails_provider.dart`, `app/lib/routes/profile_trail_screen.dart`); all 3 task commits (`d912623b`, `9826c8dd`, `138f1df2`) verified present in git log.
