---
phase: 36-local-first-recording-automatic-upload
plan: 17
subsystem: mobile-sync
tags: [flutter, riverpod, objectbox, i18n, gap-closure]

# Dependency graph
requires:
  - phase: 36-local-first-recording-automatic-upload (plans 01-16)
    provides: local-first trail capture, deferred-upload drain, TrailSync/local_trail_store,
      retired-id carry-forward and refusal message (CR-01/WR-01, 36-16)
provides:
  - "applyNetworkEditToLocalRow: owner-scoped metadata reconciliation onto the local
    ObjectBox row after a successful network save, called from _saveViaNetwork after
    result.trail is adopted and before _invalidateOwnTrailsList() re-reads the row (closes
    CR-03)"
  - "resolveLocalSaveModeForRow routes a persisted row that is already synced OR already
    carries a real server id (the alreadyUploaded window) to networkUpdate up front, before
    _copyPhotosForLocalSave can touch the filesystem (closes WR-14)"
  - "resolveLocalSaveMode's doc comment now agrees with updateLocalTrail's actual refusal
    behaviour instead of contradicting it (closes WR-16)"
  - "photosNotYetOnServer: a pure, location-based diff (not filename-based -- PocketBase
    renames every upload) excluding picked paths already inside unsynced/<localId>/ from
    the network photo payload (closes WR-13)"
  - "trail_create_screen's photo picker suppresses initialWebPhotos for any unsynced trail,
    so trail.photos and trail.localPhotos no longer render the same images twice"
affects: [mobile-trail-create-flow, mobile-own-trails-list]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A save-target routing decision must anticipate a downstream refusal before any side
      effect runs, not discover it afterward -- resolveLocalSaveModeForRow now duplicates
      updateLocalTrail's alreadyUploaded/alreadySynced refusal logic ahead of the photo
      copy, rather than letting the copy run and then discarding its result."
    - "A local-first write reconciliation that runs AFTER a network write succeeded needs no
      dirty flag and no schema change -- it is reconciling to a known-good server state, not
      remembering something still owed to the server. That asymmetry (push-then-reconcile vs
      write-then-push) is the dividing line for when a dirty flag is actually needed."
    - "Diff photo upload payloads on filesystem LOCATION (inside vs outside the app-owned
      unsynced directory), never on filename, whenever the server is known to rename
      uploads on ingest."

key-files:
  created: []
  modified:
    - app/lib/store/local_trail_store.dart
    - app/lib/store/local_photo_store.dart
    - app/lib/routes/trail_create_screen.dart
    - app/test/store/local_trail_store_test.dart
    - app/test/store/local_photo_store_test.dart
    - app/test/util/trail/own_trails_merge_test.dart

key-decisions:
  - "No dirty flag, no ObjectBox schema change for the CR-03 fix -- applyNetworkEditToLocalRow
    runs only after POST /trail/form/{id} already succeeded, reconciling the row to a
    known-good server state. There is nothing new to track and nothing left to push. A
    reconciliation that throws leaves the row stale (logged, best-effort) until
    retireUploadedLocalTrail removes it -- accepted as unchanged from today's behaviour
    (nothing currently reconciles the row at all), not a regression."
  - "photosNotYetOnServer diffs on filesystem location (p.isWithin the unsynced/<localId>/
    directory), not on filename -- the review's own suggested basename diff against
    trail.photos would exclude nothing, since PocketBase renames every uploaded file."
  - "The photo picker's initialWebPhotos is suppressed entirely (not partially filtered) for
    any unsynced trail (isUnsyncedState(trail.syncState)) -- in that window trail.photos and
    trail.localPhotos name the same images, and the local copies are the ones the hiker can
    actually remove, so they win."
  - "unsyncedTrailPhotoDir's ArgumentError (a malformed persistedLocalId) is caught and falls
    back to the unfiltered photo list rather than blocking the save -- WR-13's fix must never
    itself become a new failure mode for an otherwise-valid save."

requirements-completed: [REC-01, REC-02, REC-05, SYNC-05]

# Metrics
duration: ~20min
completed: 2026-08-03
---

# Phase 36 Plan 17: Local-row reconciliation after a network save, and duplicate-photo prevention (CR-03, WR-13, WR-14, WR-16) Summary

**A network save for an already-uploaded-but-not-yet-synced trail now writes its edit onto the local ObjectBox row too, before the own-trails list is re-read -- closing the CR-03 window where the server held the new name but the device kept showing the old one under a success toast -- and the network photo payload is filtered to files the drain has never sent, so repeated saves in that window stop doubling the server-side photo set.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-08-03T17:11:00Z
- **Completed:** 2026-08-03T17:27:00Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- Closed CR-03: `applyNetworkEditToLocalRow` writes the hiker's edited metadata (`name`,
  `location`, `date`, `public`, `completed`, `difficulty`, `description`, `categoryRecordId`,
  `subcategoryRecordId`, `tagsJson`, `updated`) onto the owner-scoped local row, called from
  `_saveViaNetwork` immediately after `trail = result.trail` is adopted and BEFORE
  `_invalidateOwnTrailsList()` runs -- so `mergeOwnTrails`' local-first dedupe now surfaces the
  server-accepted edit instead of a stale pre-edit row shadowing it. Deliberately never touches
  `id`/`owner`/`localId`/`syncState`/`syncAttempts`/`syncNextAttemptAt`/`savedByUserIds`/
  `photos`/`localPhotos`/`gpxData`/`waypoints` and never constructs via `TrailEntity.fromModel`
  -- verified by a negative grep across the function body.
- Closed WR-14: `resolveLocalSaveModeForRow` now anticipates the `alreadyUploaded`/
  `alreadySynced` refusal `updateLocalTrail` would otherwise discover only after
  `_copyPhotosForLocalSave` already ran, routing a persisted row that is synced or already
  carries a real server id straight to `networkUpdate` up front. `_copyPhotosForLocalSave` is
  now unreachable for a row the store is going to refuse -- a save the store refuses can no
  longer delete a photo file the row still lists.
- Closed WR-16: `resolveLocalSaveMode`'s doc comment no longer claims a non-synced row with a
  real server id is "deliberately routed to `updateLocal` ... so the safe write target is
  still the local row" -- that was false and contradicted `updateLocalTrail`'s own refusal
  logic 250 lines below. It now states plainly that `resolveLocalSaveMode` is the coarse,
  single-`Trail` decision, `resolveLocalSaveModeForRow` is the authority when a persisted row
  is available, and `updateLocalTrail` is the final authority on whether a write is legal.
- Closed WR-13: new pure `photosNotYetOnServer` diffs picked photo paths against the app-owned
  `unsynced/<localId>/` directory by LOCATION (`p.isWithin`), not by filename -- a basename diff
  against `trail.photos` would exclude nothing, since PocketBase renames every uploaded file on
  ingest. `_onSave` now builds the network photo payload from this filtered list at both
  `_saveViaNetwork` call sites, so a save in the `alreadyUploaded` window no longer re-sends
  photos the drain already uploaded under the append-only `photos+` key. The photo picker's
  `initialWebPhotos` is also suppressed for any unsynced trail (`isUnsyncedState`), closing the
  second half of WR-13 (each photo now renders once, not twice, before the save even runs).
- `own_trails_merge_test.dart` gained the case the review explicitly asked for: a local row
  with a real server id and a non-synced `syncState`, asserting the network hit is still
  deduped against it -- with a `reason:` documenting that this is correct only because
  `applyNetworkEditToLocalRow` now keeps that row reconciled; before this plan the same
  assertion would have documented CR-03 itself.

## Task Commits

1. **Task 1: The store reconciles a network edit onto the row, and routes an already-uploaded
   row before any side effect** - `d2d6df16` (fix)
2. **Task 2: A pure diff for "which picked photos are not already on the server"** - `9841667a`
   (feat)
3. **Task 3: The screen reconciles after a successful network save and stops re-sending
   photos** - `a5aecc83` (fix)

**Plan metadata:** pending (this commit)

## Files Created/Modified

- `app/lib/store/local_trail_store.dart` - new `applyNetworkEditToLocalRow`;
  `resolveLocalSaveModeForRow` routes `synced`/`alreadyUploaded`-shaped persisted rows to
  `networkUpdate` up front; `resolveLocalSaveMode`'s doc comment corrected (WR-16)
- `app/lib/store/local_photo_store.dart` - new `photosNotYetOnServer`
- `app/lib/routes/trail_create_screen.dart` - `_saveViaNetwork` gained `reconcileLocalId`,
  calling `applyNetworkEditToLocalRow` between adopting `result.trail` and
  `_invalidateOwnTrailsList()`; `_onSave` filters the network photo payload via
  `photosNotYetOnServer` (with an `ArgumentError` fallback to the unfiltered list); the photo
  picker's `initialWebPhotos` is empty for an unsynced trail; a WR-14 invariant comment added
  above `_copyPhotosForLocalSave` in the `updateLocal` branch
- `app/test/store/local_trail_store_test.dart` - `resolveLocalSaveModeForRow` group extended
  with the CR-03 routing cases (pending+server-id, failed+server-id) and the REC-01 control
  case (empty-id pending row still routes local)
- `app/test/store/local_photo_store_test.dart` - new `photosNotYetOnServer` group (6 cases:
  in-dir, waypoint-subdir, outside-dir, canonicalization, order preservation, empty input)
- `app/test/util/trail/own_trails_merge_test.dart` - new case: local row with a real server id
  and non-synced state still suppresses the matching network hit, with the reasoning that this
  is correct only because of this plan's reconciliation

## Decisions Made

- No dirty flag, no ObjectBox schema change -- `applyNetworkEditToLocalRow` runs only after
  the network write already succeeded, reconciling to a known-good server state rather than
  tracking something still owed to the server. A reconciliation failure is best-effort and
  logged; the row stays stale until retirement removes it, which is unchanged from today's
  behaviour (nothing currently reconciles the row at all).
- `photosNotYetOnServer` diffs on filesystem location, not filename, per the review's own
  correction that a basename diff cannot work against PocketBase's renamed uploads.
- `initialWebPhotos` is suppressed entirely (not selectively filtered) for any unsynced trail,
  since in that window it names exactly the same images as `initialLocalPhotos`.
- `unsyncedTrailPhotoDir`'s `ArgumentError` on a malformed `persistedLocalId` is caught and
  falls back to the unfiltered photo list, so the WR-13 fix itself can never block an
  otherwise-valid save.

## Deviations from Plan

None - plan executed exactly as written, including the `own_trails_merge_test.dart` case with
its explicit `reason:` documenting the CR-03 connection, and the WR-14 invariant comment above
`_copyPhotosForLocalSave` recording that the pre-routing guard should not be re-hoisted.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CR-03, WR-13, WR-14 and WR-16 are closed at the source level: `flutter analyze --no-pub`
  reports 0 errors (36 pre-existing info lints, unchanged from the 36-16 baseline), and
  `flutter test` reports 918 passing, 1 skipped, 0 failures (908 baseline + 11 new cases across
  `local_trail_store_test.dart`, `local_photo_store_test.dart` and `own_trails_merge_test.dart`).
- **Not provable in this repo -- device UAT owns it**, per the plan's own `<verification>`
  note: `applyNetworkEditToLocalRow` mutates a live ObjectBox `Store`, and its effect is only
  observable through `readOwnLocalTrails` -> `mergeOwnTrails` -> the rendered own-trails list --
  none of which `flutter test` can reach. This is `36-VERIFICATION.md` `human_verification`
  item 4, and per the plan must be extended to: force a trail into the `alreadyUploaded` window
  (let `PUT /trail/form` succeed, then make a waypoint upload keep failing), edit its title, and
  confirm (a) the own-trails list shows the NEW title immediately, (b) the server holds the new
  title, (c) the trail's photo count on the server did not change across repeated saves in that
  window, and (d) the photo picker shows each photo once.
- All four defects (CR-03, WR-13, WR-14, WR-16) `36-REVIEW.md` raised against this plan's scope
  are closed. The review's two other criticals (CR-01, CR-02) were closed by earlier
  gap-closure plans (36-15, 36-16).

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-03*

## Self-Check: PASSED
