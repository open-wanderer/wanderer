---
phase: 36-local-first-recording-automatic-upload
plan: 01
subsystem: database
tags: [flutter, objectbox, freezed, dart, local-first, sync-state]

# Dependency graph
requires:
  - phase: 34-dart-conversion-port
    provides: on-device GPX->trail metrics computation
provides:
  - "TrailSyncState enum (synced/pending/uploading/failed), synced at index 0"
  - "local_id.dart: mintLocalId/isLocalId/localIdDirSegment collision-free identity helpers"
  - "TrailEntity owner/localId/localPhotos/syncAttempts/syncNextAttemptAt/syncState schema"
  - "WaypointEntity localKey schema, localPhotos no longer dropped on fromModel"
  - "Trail/Waypoint freezed models expose localId/syncState/localKey (non-serialized)"
  - "TrailSummary interface gains syncState/localId getters, TrailSearchResult overrides them"
affects: [36-02, 36-03, 36-04, 36-05, 36-06, 36-07, 36-08]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "dbSyncState int-shadow mirrors the existing dbDifficulty @Transient() pattern"
    - "local-sentinel ids (local-<micros>-<seq>) blanked to '' at the model layer via isLocalId(), same discipline map_cache_path.dart uses for path-safety whitelisting"

key-files:
  created:
    - app/lib/models/trail_sync_state.dart
    - app/lib/util/local_id.dart
    - app/test/util/local_id_test.dart
  modified:
    - app/lib/entities/trail_entity.dart
    - app/lib/entities/waypoint_entity.dart
    - app/lib/models/trail.dart
    - app/lib/models/waypoint.dart
    - app/lib/models/trail_summary.dart
    - app/lib/models/global_search_models.dart
    - app/lib/objectbox.g.dart
    - app/lib/objectbox-model.json
    - app/lib/models/trail.freezed.dart
    - app/lib/models/waypoint.freezed.dart
    - app/test/entities/trail_entity_test.dart

key-decisions:
  - "TrailSyncState.synced pinned to enum index 0 so every pre-existing TrailEntity row (int column defaults to 0) reads back Synced with no badge"
  - "owner kept strictly separate from savedByUserIds (1:1 authorship vs 1:N download-library membership), documented against future conflation (D-10)"
  - "fromModel never sets owner or localPhotos -- only writers that own a transaction set them explicitly, keeping the shared download-path conversion untouched"

patterns-established:
  - "Local sentinel id format local-<microsecondsSinceEpoch>-<seq>, validated only via localIdDirSegment before ever becoming a filesystem path segment"

requirements-completed: [REC-01, REC-03, REC-04, SYNC-04, SYNC-05]

# Metrics
duration: 10min
completed: 2026-08-02
---

# Phase 36 Plan 01: Sync-state schema and local identity foundation Summary

**TrailEntity/WaypointEntity/Trail/Waypoint gain a collision-free local identity, owning-account, sync state, and persisted local-photo fields, with the local-sentinel id blanked to `''` at the model layer and a regenerated ObjectBox/freezed schema pinned by round-trip tests.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-08-02T13:51:33+02:00
- **Completed:** 2026-08-02T14:01:04+02:00
- **Tasks:** 3
- **Files modified:** 14 (3 created, 11 modified)

## Accomplishments
- `TrailSyncState` enum + `isUnsyncedState()` predicate, with `synced` pinned to index 0 so every pre-existing row defaults to Synced
- `local_id.dart`: `mintLocalId()`/`isLocalId()`/`localIdDirSegment()` — a collision-free, path-traversal-safe local identity shared by trails and waypoints
- `TrailEntity` gains `owner`, `localId`, `localPhotos`, `syncAttempts`, `syncNextAttemptAt`, and a `dbSyncState` int-shadow mirroring the existing `dbDifficulty` pattern; `toModel()` blanks local-sentinel ids and prefers `localPhotos` over `photos`
- `WaypointEntity` gains `localKey`; `fromModel` now mints/reuses a stable key for waypoints with no server id (fixing the `@Unique(onConflict: replace)` collision risk) and no longer drops `localPhotos` (D-04 fix)
- `Trail`/`Waypoint` freezed models, `TrailSummary` interface, and `TrailSearchResult` all carry the new non-serialized fields/getters
- ObjectBox model and freezed code regenerated additively (no property removals); round-trip tests pin local-id blanking, `dbSyncState` defaults/fallback/round-trip, and the D-04 waypoint `localPhotos`/unique-collision guards

## Task Commits

Each task was committed atomically:

1. **Task 1: Sync-state enum and collision-free local identity helpers** - `2259af42` (feat)
2. **Task 2: Persist owner, local identity, sync state and local photos on both entities and their models** - `1e5558ea` (feat)
3. **Task 3: Regenerate ObjectBox and freezed output, pin the round trip with tests** - `933cfee1` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/models/trail_sync_state.dart` - `TrailSyncState` enum + `isUnsyncedState()`
- `app/lib/util/local_id.dart` - `mintLocalId`/`isLocalId`/`localIdDirSegment`
- `app/test/util/local_id_test.dart` - unit tests for the local-id helpers
- `app/lib/entities/trail_entity.dart` - owner/localId/localPhotos/syncAttempts/syncNextAttemptAt/syncState fields, `dbSyncState` shadow, `fromModel`/`toModel` updates
- `app/lib/entities/waypoint_entity.dart` - `localKey` field, `fromModel`/`toModel` updates (fixes dropped `localPhotos`)
- `app/lib/models/trail.dart` - non-serialized `localId`/`syncState` freezed fields
- `app/lib/models/waypoint.dart` - non-serialized `localKey` freezed field, `listKey` getter
- `app/lib/models/trail_summary.dart` - `syncState`/`localId` interface getters
- `app/lib/models/global_search_models.dart` - `TrailSearchResult` overrides for the two new getters
- `app/lib/objectbox.g.dart`, `app/lib/objectbox-model.json` - regenerated (additive only)
- `app/lib/models/trail.freezed.dart`, `app/lib/models/waypoint.freezed.dart` - regenerated
- `app/lib/provider/profile/profile_counts_provider.g.dart`, `app/lib/provider/trail/tag_provider.g.dart` - riverpod hash shift (side effect of the project-wide `build_runner` pass touching `Trail`'s dependency graph; logic unchanged, same as the Phase 27 precedent)
- `app/test/entities/trail_entity_test.dart` - new local-first round-trip test group

## Decisions Made
- `TrailSyncState.synced` pinned to enum index 0 (load-bearing for pre-existing rows) — documented in a doc comment above the enum
- `owner` kept strictly separate from `savedByUserIds`; `fromModel` deliberately never sets `owner` or `localPhotos` so every writer that owns those fields sets them explicitly inside its own transaction
- `localPhotos`/`photos` fallback in `TrailEntity.toModel()` relies on mutual exclusivity (a downloaded row's copies live in `photos`, an unsynced row's in `localPhotos`) rather than a merge

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. `build_runner` regeneration completed cleanly on the first pass; `flutter analyze --no-pub` reported zero errors (only pre-existing `deprecated_member_use` infos plus one expected `dangling_library_doc_comments` info on `local_id.dart`, matching the doc-comment-before-code style already used in `map_cache_path.dart`); the full `flutter test` suite (711 tests) passed with no regressions.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Schema and model round trip are in place; every later plan in this phase (recording capture, upload drain, sync badges, queries) can now read/write `owner`, `localId`, `syncState`, and `localPhotos` on both trails and waypoints
- No blockers

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-02*

## Self-Check: PASSED

All 10 files created/modified in this plan verified present on disk; all 3 task commits (`2259af42`, `1e5558ea`, `933cfee1`) verified present in git log.
