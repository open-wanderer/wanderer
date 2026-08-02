---
phase: 36-local-first-recording-automatic-upload
plan: 06
subsystem: mobile-recording
tags: [flutter, riverpod, objectbox, local-first, offline-save, photo-copy]

# Dependency graph
requires:
  - phase: 36-local-first-recording-automatic-upload
    plan: 01
    provides: "TrailSyncState enum, local_id.dart's mintLocalId, Waypoint.localKey/listKey"
  - phase: 36-local-first-recording-automatic-upload
    plan: 02
    provides: "local_photo_store_util.dart's reconcileLocalPhotos/unsyncedTrailPhotoDir/unsyncedWaypointPhotoDir"
  - phase: 36-local-first-recording-automatic-upload
    plan: 03
    provides: "local_trail_store.dart's resolveLocalSaveMode/LocalSaveMode/saveNewLocalTrail/updateLocalTrail/readLocalTrail"
  - phase: 36-local-first-recording-automatic-upload
    plan: 04
    provides: "trail_sync_provider.dart's drainIfOnline, trailSaveProvider.resolveTags"
provides:
  - "trail_create_screen.dart: three-way local-first _onSave (networkUpdate/createLocal/updateLocal), never network-dependent on a local save"
  - "_copyPhotosForLocalSave: copies picked trail/waypoint photos into app-owned storage before the local row is written, counting failures (D-03)"
  - "Every not-yet-uploaded waypoint carries id: '' plus a minted localKey instead of a timestamp-derived synthetic id (D-06)"
  - "trail_save_provider.updateTrail's waypoint diff keyed on listKey instead of id"
affects: [36-07, 36-08]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shared local-save tail (_finishLocalSave) factored out of both LocalSaveMode.createLocal/updateLocal branches so the drain kick, photo-failure toast, and post-save bookkeeping exist exactly once in source, each branch still keeping its own try/catch"

key-files:
  created:
    - app/test/routes/trail_create_screen_local_save_gate_test.dart
  modified:
    - app/lib/routes/trail_create_screen.dart
    - app/lib/provider/trail/trail_save_provider.dart

key-decisions:
  - "Both local-first _onSave branches are fully local-first -- they never touch the network, online or offline. A fire-and-forget unawaited(trailSyncProvider.notifier.drainIfOnline()) runs immediately after a successful local write, resolving RESEARCH.md's Open Question 1 in favour of one code path (matching the Komoot/AllTrails model) instead of a network-first branch with an offline-catch fallback"
  - "_finishLocalSave is a private shared helper (not a required plan artifact) holding the drain kick + photo-failure toast + re-read/setState/reset/success-toast sequence common to createLocal and updateLocal, so that logic exists exactly once in source while each branch still gets its own try/catch around its own write call"
  - "The empty-id convention now also drives waypoint list identity: _onDeleteWaypoint/_replaceWaypoint match on Waypoint.listKey (id when present, else localKey) instead of .id, and trail_save_provider.updateTrail's compareObjectArrays diffs on listKey too -- keying on id after ids can legitimately be empty would collapse every new waypoint onto the same '' key"

requirements-completed: [REC-01, REC-05, SYNC-04]

# Metrics
duration: ~20min
completed: 2026-08-02
---

# Phase 36 Plan 06: Local-first _onSave, photo copy, and the empty-id waypoint convention Summary

**`trail_create_screen.dart`'s `_onSave` is now a three-way branch routed on `resolveLocalSaveMode`: an already-synced trail still goes through the unchanged network `PUT`/`POST` path, while a captured or still-unsynced trail writes straight to ObjectBox (photos copied into app-owned storage first) and never touches the network at all, with a fire-and-forget upload-drain kick immediately after.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-08-02T14:00:00Z (approx.)
- **Completed:** 2026-08-02T14:20:17Z
- **Tasks:** 3
- **Files modified:** 3 (1 created, 2 modified)

## Accomplishments
- Both synthetic-id waypoint stubs (`_onCreateWaypoint`'s manual stub, `_onCreateWaypointsFromPhotos`'s EXIF-derived construction) now mint `id: ''` plus `localKey: mintLocalId()` instead of a `DateTime.now().microsecondsSinceEpoch`-derived id, so a manually-added or photo-derived waypoint no longer looks already-uploaded to the drain's `id.isEmpty` resume check
- In-memory waypoint list identity (`_onDeleteWaypoint`'s `removeWhere`, `_replaceWaypoint`'s comprehension) switched from matching on `.id` to matching on `.listKey`; `trail_save_provider.updateTrail`'s `compareObjectArrays<Waypoint>` call does the same
- `_onSave` restructured around `resolveLocalSaveMode(updatedTrail)`: `LocalSaveMode.networkUpdate` keeps the pre-existing `updateTrail` call byte-for-byte; `LocalSaveMode.createLocal` mints a `localId`, copies photos, and calls `saveNewLocalTrail`; `LocalSaveMode.updateLocal` reconciles photos against the existing local directory and calls `updateLocalTrail` with the resumed `_localId`
- `_copyPhotosForLocalSave` copies the form's picked trail photos plus every waypoint's `localPhotos` into app-owned storage via `reconcileLocalPhotos`, summing failures across all copies
- `_finishLocalSave` (shared tail of both local branches): fires `trailSyncProvider.notifier.drainIfOnline()` unawaited, shows the `photo_copy_failed_toast` when any photo failed to copy, re-reads the saved trail via `readLocalTrail`, mirrors the network path's `setState`/post-frame `reset()`/success-toast bookkeeping
- `_localId` field added to `_TrailCreateScreenState`, seeded from `widget.trail.localId` in `initState`, so re-opening an unsynced trail resumes against the same local row (REC-05, SYNC-04)
- Source gate test (`trail_create_screen_local_save_gate_test.dart`, 4 tests / 12 `expect`s all carrying a `reason`) protects: no `microsecondsSinceEpoch` outside comments, both `Waypoint(` constructions pass `id: ''`, `_onSave` references `resolveLocalSaveMode` and all three `LocalSaveMode` values, and neither local branch references `trailSaveProvider`

## Task Commits

Each task was committed atomically:

1. **Task 1: Give every not-yet-uploaded waypoint an empty id and a local key** - `cfa0020b` (fix)
2. **Task 2: Three-way _onSave with a local-first write, photo copy and D-03 reporting** - `fea11932` (feat)
3. **Task 3: Source gate for the empty-id convention and the save-mode branch** - `fe6e0252` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/routes/trail_create_screen.dart` - three-way `_onSave`, `_localId` field, `_copyPhotosForLocalSave`/`_finishLocalSave` helpers, empty-id waypoint stubs, `.listKey`-based list identity
- `app/lib/provider/trail/trail_save_provider.dart` - `updateTrail`'s `compareObjectArrays<Waypoint>` now diffs on `.listKey`
- `app/test/routes/trail_create_screen_local_save_gate_test.dart` - source-level gate protecting the empty-id convention and the three-way save branch

## Decisions Made
- Both local-first branches are structurally incapable of a connectivity failure (REC-01): the network call happens only in the `networkUpdate` branch, resolved via `resolveLocalSaveMode` before either local branch's code runs at all, rather than a try/catch around a network call with an offline fallback
- The drain kick (`drainIfOnline()`) and the photo-failure toast were factored into a single shared `_finishLocalSave` helper called from both `createLocal` and `updateLocal`, rather than duplicated inline in each branch, satisfying the plan's acceptance criteria that these appear exactly once in source while still keeping each local branch's own `try`/`catch` as the plan's action text specified
- `_copyPhotosForLocalSave` only calls `reconcileLocalPhotos` for a waypoint when that waypoint's `localPhotos` is non-empty (per the plan's literal instruction: "once per waypoint that has `localPhotos`"), so a waypoint with no picked photos never gets an empty directory created for it

## Deviations from Plan

None - plan executed exactly as written. The gate test's initial 200-character look-ahead window (for the "next thing after `Waypoint(` is `id: ''`" check) was too narrow for the photo-EXIF construction's two-line doc comment and was widened to 300 characters before committing Task 3 -- this was iteration on the test's own implementation detail, not a deviation from the plan's specified assertions.

## Issues Encountered

None. `flutter analyze --no-pub` reports zero errors project-wide on every changed/created file (only the same pre-existing `info`-level lints already logged in 36-04's SUMMARY: deprecated Font Awesome constants in `icon_util.dart`, two dangling-library-doc-comment infos). The full `flutter test` suite (763 tests, 1 pre-existing skip unrelated to this plan) passed with no regressions.

## Device-Verified-Only Behaviours

Per the plan's own `<verify><human-check>` instruction, the following is covered only by the source-level gate test and `flutter analyze`, not by a live-device pass (no device available in this execution environment):
- An airplane-mode save actually succeeding end-to-end with the success toast and no error
- Re-opening and re-saving the same unsynced draft actually updating the same trail row rather than producing a second one, observed on-device
- The drain actually kicking off and completing an upload moments after an online save

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `trail_create_screen.dart`'s local-first save path is ready for 36-07 (own-trails list UI) and 36-08 (sign-out warning) to build on directly -- both already consume `local_trail_store.dart`/`trail_sync_provider.dart` from earlier plans, and this plan is the one that finally wires the screen's save button to those primitives
- No blockers

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-02*

## Self-Check: PASSED

All 3 files created/modified in this plan verified present on disk (`app/lib/routes/trail_create_screen.dart`, `app/lib/provider/trail/trail_save_provider.dart`, `app/test/routes/trail_create_screen_local_save_gate_test.dart`); all 3 task commits (`cfa0020b`, `fea11932`, `fe6e0252`) verified present in git log.
