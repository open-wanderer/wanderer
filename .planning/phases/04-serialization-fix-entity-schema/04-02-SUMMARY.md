---
phase: 04-serialization-fix-entity-schema
plan: 02
subsystem: database
tags: [objectbox, flutter, dart, entity-schema, navigation-cache, build_runner]

# Dependency graph
requires:
  - phase: 04-serialization-fix-entity-schema (plan 04-01)
    provides: NavigateResponse.toJson() serialization fix (nested maneuvers serialized) — makes navCacheJson storable
provides:
  - String? navCacheJson field + constructor parameter on TrailEntity (entity-only cache slot)
  - Regenerated objectbox-model.json with navCacheJson property UID under TrailEntity
  - Regenerated objectbox.g.dart reflecting the new property
affects: [05, offline-navigation, navCacheJson, trail-cache, objectbox]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Entity-only ObjectBox field: nullable String stored on TrailEntity but deliberately absent from Trail/TrailExpand model and both fromModel/toModel conversion paths"

key-files:
  created: []
  modified:
    - app/lib/entities/trail_entity.dart
    - app/lib/objectbox-model.json
    - app/lib/objectbox.g.dart

key-decisions:
  - "navCacheJson follows the gpxData precedent: plain nullable String, optional named constructor param, no annotation, no default (D-06)"
  - "navCacheJson is entity-only — not added to Trail/TrailExpand and not referenced in fromModel/toModel (D-07, D-08)"
  - "objectbox-model.json committed in the SAME commit as the entity change to prevent UID conflicts (D-02)"

patterns-established:
  - "Entity-only cache column: store serialized payload on the ObjectBox entity without exposing it through the domain model conversion paths"

requirements-completed: []

# Metrics
duration: 13min
completed: 2026-06-14
---

# Phase 4 Plan 02: TrailEntity navCacheJson Schema Summary

**Added an entity-only `String? navCacheJson` cache slot to TrailEntity and regenerated the ObjectBox model, giving Phase 5 a place to persist `jsonEncode(NavigateResponse.toJson())` without leaking into the Trail domain model.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-06-14T10:31:46Z
- **Completed:** 2026-06-14T10:44:52Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Added `String? navCacheJson;` field declaration to `TrailEntity` directly after `gpxData`, with no ObjectBox annotation (native nullable String, no migration guard).
- Added `this.navCacheJson,` as an optional named constructor parameter next to `this.gpxData,` (no default — matches the gpxData precedent).
- Ran `build_runner` which assigned a fresh property UID in `objectbox-model.json` and regenerated `objectbox.g.dart`.
- Verified the field stays entity-only: it appears exactly twice in `trail_entity.dart` (field + constructor param), is absent from `trail.dart`, and is not referenced in `fromModel`/`toModel`.
- Confirmed no Phase 4 regression: `flutter analyze` reports no error-level issues, and the navigate_response roundtrip group (from 04-01) passes.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add navCacheJson field and constructor parameter to TrailEntity, regenerate ObjectBox model** - `69c01707` (feat)
2. **Task 2: Verify full Flutter analyze + test suite passes (no regression)** - verification-only, no code changes (no commit)

**Plan metadata:** see final docs commit (or skipped — `commit_docs: false`)

## Files Created/Modified
- `app/lib/entities/trail_entity.dart` - Added `String? navCacheJson;` field and `this.navCacheJson,` constructor parameter; conversion paths untouched.
- `app/lib/objectbox-model.json` - build_runner added the `navCacheJson` property entry with a fresh UID under the TrailEntity model.
- `app/lib/objectbox.g.dart` - Regenerated to reflect the new property (binding/reader/writer for navCacheJson).

## Decisions Made
- Followed plan as specified — navCacheJson added as a plain nullable String matching the gpxData precedent (D-06), kept entity-only (D-07, D-08), and objectbox-model.json committed in the same commit (D-02).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

**Pre-existing test failures in `app/test/models/feed_item_test.dart` (NOT caused by this plan, NOT fixed — out of scope).**

The full `flutter test` run surfaced 2 failures in `feed_item_test.dart`:
- `FeedItem.fromJson type "trail"` — `type 'int' is not a subtype of type 'String' in type cast` at `trail.g.dart 55:67`
- `FeedItem.fromJson type "list"` — `type 'int' is not a subtype of type 'List<dynamic>?' in type cast` at `list.g.dart 34:27`

These originate in `trail.g.dart` / `list.g.dart` / `feed_item.dart`, none of which were modified by this plan (verified via `git status`/`git diff` — all unchanged). They are test-fixture type mismatches predating Phase 4. Per the scope boundary rule, they were logged to `deferred-items.md` and left unfixed. The Phase-4-relevant tests (navigate_response roundtrip group) all pass.

`flutter analyze` returns non-zero only because of pre-existing `info`/`warning` items (deprecated FontAwesome icon aliases, an unused import in `feed_item_test.dart`) — no error-level issues were introduced.

## Known Stubs

None — `navCacheJson` is an intentional schema slot to be populated by Phase 5; it is not a UI-facing stub and does not block this plan's goal (establishing the cache column).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The `navCacheJson` cache column is in place and the ObjectBox model is regenerated — Phase 5 can persist `jsonEncode(NavigateResponse.toJson())` (made serializable by 04-01) to this field.
- No blockers introduced. The pre-existing `feed_item_test.dart` failures are tracked in `deferred-items.md` for a future fix outside Phase 4 scope.

## Self-Check: PASSED

- FOUND: app/lib/entities/trail_entity.dart
- FOUND: app/lib/objectbox-model.json
- FOUND: app/lib/objectbox.g.dart
- FOUND: .planning/phases/04-serialization-fix-entity-schema/04-02-SUMMARY.md
- FOUND commit: 69c01707 (Task 1)

---
*Phase: 04-serialization-fix-entity-schema*
*Completed: 2026-06-14*
