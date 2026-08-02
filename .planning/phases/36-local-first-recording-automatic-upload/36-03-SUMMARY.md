---
phase: 36-local-first-recording-automatic-upload
plan: 03
subsystem: mobile-storage
tags: [flutter, objectbox, local-first, sync-state, trail-download]

# Dependency graph
requires:
  - phase: 36-local-first-recording-automatic-upload
    plan: 01
    provides: "TrailEntity owner/localId/localPhotos/syncAttempts/syncNextAttemptAt/syncState schema, WaypointEntity localKey"
provides:
  - "local_trail_store.dart: the single sanctioned read/write layer for locally-captured TrailEntity rows"
  - "resolveLocalSaveMode/isDrainDue: pure save-mode routing and drain-due gating, test-pinned"
  - "saveNewLocalTrail/updateLocalTrail/deleteLocalTrailRow: the local write path"
  - "readLocalTrail/readOwnLocalTrails/countUnsyncedTrails/unsyncedLocalIds/selectDrainCandidates: owner-scoped reads"
  - "writeServerTrailId/writeServerWaypointId/markTrailSynced/recordDrainFailure/resetDrainBackoff: drain bookkeeping writes"
  - "TrailDownloadService no longer wipes local bookkeeping on a re-download"
affects: [36-04, 36-05, 36-06, 36-07, 36-08]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Plain-library store-function file (no Riverpod, no part directive), matching active_navigation_store.dart/current_account.dart"
    - "resolveDrainFailureOutcome extracted as recordDrainFailure's pure decision core, unit-tested without a live ObjectBox Store"

key-files:
  created:
    - app/lib/util/local_trail_store.dart
    - app/test/util/local_trail_store_test.dart
    - app/test/services/trail_download_service_carry_forward_test.dart
  modified:
    - app/lib/services/trail_download_service.dart

key-decisions:
  - "Every account-scoped function in local_trail_store.dart takes accountId as a required parameter rather than calling currentAccountId internally -- the file's own account-scoping invariant is enforced by requiring every caller to supply a fresh id, not by this file resolving it itself"
  - "recordDrainFailure's attempt-count/backoff decision extracted into a standalone pure function (resolveDrainFailureOutcome) so its boundary is unit-tested without a Store, since Phase 31 established there is no ObjectBox test harness for plain flutter test"
  - "unsyncedLocalIds is deliberately NOT account-scoped (D-13) -- its only consumer is the startup photo orphan sweep, which must not delete a signed-out account's still-pending photos"

requirements-completed: [REC-01, REC-02, REC-04, REC-05, SYNC-04, SYNC-05]

# Metrics
duration: 15min
completed: 2026-08-02
---

# Phase 36 Plan 03: Local trail store and download-carry-forward Summary

**`local_trail_store.dart` is now the single owner-scoped read/write layer for locally-captured trails (save-mode routing, drain-due gating, and the full drain bookkeeping lifecycle), and `TrailDownloadService` no longer erases a trail's local identity/ownership/sync state when the same trail is later re-downloaded.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-08-02T14:18:00+02:00 (approx.)
- **Completed:** 2026-08-02T14:32:38+02:00
- **Tasks:** 3
- **Files modified:** 4 (3 created, 1 modified)

## Accomplishments
- `LocalSaveMode` enum + `resolveLocalSaveMode(Trail)`: routes a save to `networkUpdate`/`createLocal`/`updateLocal`, correctly treating a mid-drain trail (non-empty server id, still `uploading`) as `updateLocal` rather than `networkUpdate`
- `isDrainDue(TrailEntity, DateTime)`: true only for `pending`/`uploading` rows whose `syncNextAttemptAt` has passed; a `failed` row is never due (D-07 manual-retry-only parking)
- `saveNewLocalTrail`/`updateLocalTrail`/`deleteLocalTrailRow`: the full local write path, each its own `runInTransaction(TxMode.write, ...)` block; `updateLocalTrail` carries the existing row's identity/ownership/sync bookkeeping forward and removes orphaned waypoint rows
- `readLocalTrail`/`readOwnLocalTrails`/`countUnsyncedTrails`/`unsyncedLocalIds`/`selectDrainCandidates`: every account-scoped read filters on `owner`, with `unsyncedLocalIds` deliberately cross-account for the photo orphan sweep
- `writeServerTrailId`/`writeServerWaypointId`/`markTrailSynced`/`recordDrainFailure`/`resetDrainBackoff`: the drain's five bookkeeping writes, each a standalone committed transaction
- `TrailDownloadService.downloadTrail`'s existing write transaction now also carries `owner`/`localId`/`syncState`/`syncAttempts`/`syncNextAttemptAt`/`localPhotos` forward from the pre-existing row, guarded by a source-level gate test

## Task Commits

Each task was committed atomically:

1. **Task 1: Pure save-mode and drain-due decisions, plus the local write path** - `09d05df6` (feat)
2. **Task 2: Owner-scoped reads and drain bookkeeping** - `4a9e87d9` (feat)
3. **Task 3: Stop a re-download from wiping local bookkeeping** - `18dbdc15` (fix)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/util/local_trail_store.dart` - the single sanctioned read/write layer for locally-captured `TrailEntity` rows: pure decisions, write path, owner-scoped reads, drain bookkeeping
- `app/test/util/local_trail_store_test.dart` - 10 pure-decision tests: `resolveLocalSaveMode` (4), `isDrainDue` (4), `resolveDrainFailureOutcome` (2 boundary tests)
- `app/test/services/trail_download_service_carry_forward_test.dart` - source-level gate test asserting all six carried fields appear inside `downloadTrail`'s write transaction
- `app/lib/services/trail_download_service.dart` - the existing `runInTransaction(TxMode.write` block now carries `owner`/`localId`/`syncState`/`syncAttempts`/`syncNextAttemptAt`/`localPhotos` forward alongside its pre-existing `savedByUserIds` carry-forward

## Decisions Made
- Account id resolution stays entirely with the CALLER (future plans 36-04..36-08): every account-scoped function here requires `accountId`/`ownerAccountId` as a parameter rather than reading `currentAccountId(store)` internally, so the D-13 "always fresh, never cached" invariant is enforced structurally at every call site
- `recordDrainFailure`'s attempt-count/backoff boundary decision was pulled out into a standalone pure function, `resolveDrainFailureOutcome`, specifically so it could be unit-tested without a live `Store` -- Phase 31 established there is no ObjectBox harness for plain `flutter test`
- `writeServerTrailId`'s body deliberately touches only `id` -- no `syncState` assignment -- so it can commit the instant `PUT /trail/form` returns, before any waypoint upload starts (SYNC-04, RESEARCH.md Pitfall 3)
- Avoided importing `trail_sync_state.dart` into `trail_download_service.dart`: `entity.syncState = existing?.syncState ?? entity.syncState` reads the freshly-built entity's own already-set default instead of writing a `TrailSyncState.synced` literal, keeping the diff scoped entirely to the write-transaction block and its doc comment (matching the plan's own acceptance criterion)

## Deviations from Plan

None - plan executed exactly as written. (Task 1 and Task 2 both target the same two files per the plan's own `files_modified` list; they were still committed as two separate atomic commits by writing the Task 1 subset first, committing, then extending with the Task 2 additions.)

## Issues Encountered

None. `flutter analyze --no-pub` reported zero errors on both changed files; the full `flutter test` suite (735 tests, 1 pre-existing skip unrelated to this plan) passed with no regressions.

## Device-Verified-Only Behaviours

Per the plan's own instruction, the following are covered only by source-level grep gates or the pure-decision tests above, not by a live-Store test (no ObjectBox harness exists for plain `flutter test`, per Phase 31):
- `saveNewLocalTrail`/`updateLocalTrail`/`deleteLocalTrailRow`'s actual ObjectBox write/put/remove behavior, including waypoint cascade-put and orphan-waypoint removal
- `readLocalTrail`/`readOwnLocalTrails`/`countUnsyncedTrails`/`unsyncedLocalIds`/`selectDrainCandidates`'s actual query execution against a live box
- `writeServerTrailId`/`writeServerWaypointId`/`markTrailSynced`/`recordDrainFailure`/`resetDrainBackoff`'s actual row mutation and commit behavior
- `TrailDownloadService.downloadTrail`'s carried-forward fields actually surviving a real re-download round trip

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `local_trail_store.dart` is ready for 36-04 (the upload drain), 36-05/36-06 (save-path wiring), and 36-07/36-08 (own-trails list, sign-out warning) to build on directly
- `TrailDownloadService` is now safe against the "capture, upload, re-download" sequence losing local bookkeeping
- No blockers

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-02*

## Self-Check: PASSED

All 4 files created/modified in this plan verified present on disk (`app/lib/util/local_trail_store.dart`, `app/test/util/local_trail_store_test.dart`, `app/test/services/trail_download_service_carry_forward_test.dart`, `app/lib/services/trail_download_service.dart`); all 3 task commits (`09d05df6`, `4a9e87d9`, `18dbdc15`) verified present in git log.
