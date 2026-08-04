---
phase: 38-downloaded-trails-as-state-not-objects
plan: 03
subsystem: mobile-offline-sync
tags: [flutter, objectbox, riverpod, dio, trail-download]

# Dependency graph
requires:
  - phase: 38-01
    provides: forceOffline retirement (trailProvider is build(String id)); the two new l10n keys
provides:
  - applyServerTrailToLibraryRow(Store, {accountId, trail}) -- library-keyed stored-row refresh (local_trail_store.dart)
  - fetchServerTrail(Dio, String) -- network-only trail fetch with no ObjectBox fallback (trail_provider.dart)
  - TrailNotifier._refreshLibraryRow -- opportunistic D-14 refresh wired into build()
  - D-13 reconciliation call site in trail_create_screen.dart's _saveViaNetwork
affects: [38-05 (Update action reuses the same write path), 38-06 (Edit-fetches-server-copy D-15 needs fetchServerTrail as its seam)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Library-keyed vs local-capture-keyed ObjectBox writes are two separate functions with two separate query keys, never a loosened gate on one function"
    - "Best-effort writes inside a provider's build() get their own try/catch so a store failure can never be caught by build()'s existing catch(_) and silently downgrade a good server response to a stale cache"

key-files:
  created: []
  modified:
    - app/lib/store/local_trail_store.dart
    - app/lib/provider/trail/trail_provider.dart
    - app/lib/routes/trail_create_screen.dart

key-decisions:
  - "applyServerTrailToLibraryRow keys on TrailEntity_.id + savedByUserIds.containsElement(accountId), never localId/owner -- a downloaded row's localId is always null so applyNetworkEditToLocalRow could never reach it (D-13)"
  - "gpxData guarded: a response with no GPX keeps the stored track rather than blanking it"
  - "photos/localPhotos are always carried forward from the existing row, never touched by this automatic path (D-14a)"
  - "author/category relations and their scalar record ids fall back to the existing row's values when the incoming model carries no expand, so a partial response cannot drop them"
  - "waypoints are only rebuilt when trail.expand?.waypointsViaTrail is non-null; otherwise the existing children are carried forward untouched, since fromModel would otherwise leave the ToMany empty"

patterns-established:
  - "Two-account-scoping-keys trap: TrailEntity_.owner.equals(accountId) is local-capture ownership, TrailEntity_.savedByUserIds.containsElement(accountId) is library membership -- never conflate them in a query"

requirements-completed: [DL-02, DL-04, DL-05, DL-07]

# Metrics
duration: ~15min
completed: 2026-08-04
---

# Phase 38 Plan 03: Server-Trail-to-Library-Row Reconciliation Summary

**Added `applyServerTrailToLibraryRow` (a library-membership-keyed ObjectBox refresh), extracted `fetchServerTrail` as a network-only fetch seam, and wired both the hiker's own edit (D-13) and every successful online open of a downloaded trail (D-14) to write the known-good server model into the stored row at zero extra network cost.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-08-04T14:18Z
- **Tasks:** 3/3
- **Files modified:** 3

## Accomplishments

- A downloaded trail's stored metadata, description, waypoints and track now stay current merely by being viewed online (D-14), with zero additional network traffic since both the record and the GPX are already fetched by `TrailNotifier.build`.
- The hiker's own successful edit of a downloaded trail now lands on the stored copy without a re-download (D-13) -- previously `applyNetworkEditToLocalRow`'s `localId & owner` key could never match a downloaded row (`localId == null`), so the edit silently never reached ObjectBox.
- `fetchServerTrail` is a new network-only seam with no ObjectBox read and no cache fallback, ready for plan 38-06's Edit-fetches-server-copy flow (D-15).
- Downloaded photos, waypoint photos, and library membership are structurally untouched by every automatic path in this plan (D-14a).

## Task Commits

1. **Task 1: Add applyServerTrailToLibraryRow to local_trail_store.dart** - `e312ef8a` (feat)
2. **Task 2: Extract fetchServerTrail and wire the opportunistic refresh into TrailNotifier.build** - `824c1b06` (feat)
3. **Task 3: Reconcile the hiker's own edit onto the downloaded row (D-13)** - `310aca0c` (feat)

## Files Created/Modified

- `app/lib/store/local_trail_store.dart` - Added `applyServerTrailToLibraryRow`, a second write function (distinct from `applyNetworkEditToLocalRow`) keyed on `TrailEntity_.id & savedByUserIds.containsElement(accountId)`, carrying forward `obxId`/`id`/`owner`/`localId`/`syncState`/`syncAttempts`/`syncNextAttemptAt`/`savedByUserIds`/`photos`/`localPhotos`/`navCacheJson`, guarding `gpxData` against a blank overwrite, falling back on author/category relations, and conditionally rebuilding waypoints
- `app/lib/provider/trail/trail_provider.dart` - Extracted `fetchServerTrail(Dio, String)` as a top-level, network-only function; `build()` now calls it then `_refreshLibraryRow(trail)` (its own try/catch, `debugPrint`-and-swallow) before returning
- `app/lib/routes/trail_create_screen.dart` - Added an unconditional (not gated on `reconcileLocalId`) `applyServerTrailToLibraryRow` call in `_saveViaNetwork`, placed between the existing `applyNetworkEditToLocalRow` block and `_invalidateOwnTrailsList()`, preserving the source-text ordering the local-save-gate test asserts

## Decisions Made

- Query for the new function keys on library membership (`savedByUserIds`), never local-capture ownership (`owner`) -- these are two structurally distinct signals (D-10) and conflating them would either miss every downloaded row or risk cross-account writes.
- `gpxData`, `photos`, `localPhotos` are the three load-bearing carry-forward fields: dropping `savedByUserIds` would erase every account's library claim, and `photos` on a downloaded row holds LOCAL file paths (not server filenames), so overwriting it would strand downloaded images.
- Author/category fallback: relation targets fall back to the existing row when the incoming model's `expand` doesn't carry them; scalar record ids fall back via `??=` when the incoming value is null. This defends against a response fetched with a narrower expand than the phase's own `fetchServerTrail` query normally requests.
- Waypoints are only rebuilt when `trail.expand?.waypointsViaTrail` is non-null, since `TrailEntity.fromModel` otherwise leaves the `ToMany` empty and a blind carry-forward-nothing would silently delete every waypoint on a response with no waypoint expand.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `fetchServerTrail` is ready for plan 38-06's D-15 edit-fetches-server-copy flow -- it is already network-only with no cache fallback, exactly the seam that flow needs.
- `applyServerTrailToLibraryRow` is ready for plan 38-05's *Update* action, which is a re-download (D-12) that can reuse the same carry-forward discipline for its metadata/track half if needed, though `TrailDownloadService` already owns the photo half.
- Write-path verification for `applyServerTrailToLibraryRow` (does it actually persist correctly against a live ObjectBox `Store`) is deferred to plan 38-06's on-device human check, per this repo's testing constraint (host `flutter test` cannot open a real `Store`).
- `flutter analyze --no-pub lib test` reports only the 36 pre-existing info-level lines (deprecated FontAwesome icon names, dangling library doc comments, unnecessary imports); zero errors, zero warnings.
- `flutter test` is green at 960 passing tests, matching the pre-plan baseline.

---
*Phase: 38-downloaded-trails-as-state-not-objects*
*Completed: 2026-08-04*

## Self-Check: PASSED

- FOUND: app/lib/store/local_trail_store.dart
- FOUND: app/lib/provider/trail/trail_provider.dart
- FOUND: app/lib/routes/trail_create_screen.dart
- FOUND commit: e312ef8a
- FOUND commit: 824c1b06
- FOUND commit: 310aca0c
