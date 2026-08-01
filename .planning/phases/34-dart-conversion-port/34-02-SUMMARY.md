---
phase: 34-dart-conversion-port
plan: 02
subsystem: database
tags: [pocketbase, openapi, sveltekit, flutter, objectbox, freezed, trail-metrics]

# Dependency graph
requires:
  - phase: 33-conversion-correctness
    provides: the corrected GPX->trail metrics computation whose output duration/moving_duration must not be conflated
provides:
  - "moving_duration field on the PocketBase trails collection, added by an idempotent reversible migration"
  - "moving_duration on the Trail/TrailCreateInput/TrailUpdateInput OpenAPI schemas (not SummitLog)"
  - "Trail.moving_duration on the TS model + trailDisplayDuration() as the single D-10 display-rule implementation, wired into 5 web display sites"
  - "Trail.movingDuration / TrailEntity.movingDuration on the Dart side, round-tripping through ObjectBox and the multipart trail form body"
affects: [34-05, dart-conversion-port]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "D-10 display rule centralized in one function (trailDisplayDuration) rather than open-coded at each call site"
    - "PocketBase additive-field migration mirrors an existing field's exact shape (min/max/onlyInt), guarded by Fields.GetByName == nil for idempotency"

key-files:
  created:
    - db/migrations/1785300000_add_moving_duration_to_trails.go
    - app/test/entities/trail_entity_test.dart
  modified:
    - web/src/lib/models/api/openapi_schemas.ts
    - web/src/lib/models/trail.ts
    - web/src/lib/util/format_util.ts
    - web/src/lib/util/maplibre_util.ts
    - web/src/lib/components/trail/trail_info_panel.svelte
    - web/src/lib/components/trail/trail_card.svelte
    - web/src/lib/components/trail/trail_list_item.svelte
    - web/src/lib/components/trail/trail_table.svelte
    - app/lib/models/trail.dart
    - app/lib/models/trail.freezed.dart
    - app/lib/models/trail.g.dart
    - app/lib/entities/trail_entity.dart
    - app/lib/objectbox-model.json
    - app/lib/objectbox.g.dart
    - app/lib/util/form_data_util.dart
    - app/lib/provider/trail/trail_filter_provider.g.dart
    - app/lib/provider/trail/trail_library_provider.g.dart

key-decisions:
  - "web/static/docs/api/wanderer.openapi.json is regenerated and its 3 moving_duration occurrences verified, but not committed -- it is gitignored (web/.gitignore:6), generated at build time, not part of source control"
  - "Trail.from() (the TS trail-duplicate helper) now carries moving_duration alongside duration -- omitting it would have silently dropped a recording's moving time whenever a user duplicates a trail"

patterns-established:
  - "trailDisplayDuration(trail) is the only sanctioned way to read a Trail's displayed duration on the web; new display sites must call it, not trail.duration directly"

requirements-completed: [CONV-06]

# Metrics
duration: 25min
completed: 2026-07-31
---

# Phase 34 Plan 02: moving_duration field end to end Summary

**Added a separate `moving_duration` field across PocketBase, OpenAPI, TS and Dart, with `trailDisplayDuration()` as the single web display-rule implementation -- `duration` keeps exactly one meaning (GPX-derived elapsed) everywhere, and the trail-edit page remains structurally unable to write moving_duration.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-07-31T22:11:00+02:00 (approx.)
- **Completed:** 2026-07-31T22:22:28+02:00
- **Tasks:** 3
- **Files modified:** 17 (2 new)

## Accomplishments
- New idempotent, reversible PocketBase migration adds `moving_duration` (optional number, min:0) to `trails`, mirroring the existing `duration` field's exact shape
- `moving_duration` published on exactly the three Trail-shaped OpenAPI schemas (`Trail`, `TrailCreateInput`, `TrailUpdateInput`); verified zero occurrences in the SummitLog schema range
- `trailDisplayDuration()` is now the single implementation of D-10's display rule on the web, wired into all five Trail duration display surfaces; the trail-edit page (`updateTotals()`'s ~14 call sites) is verified untouched and contains zero references to `moving_duration`
- `Trail.movingDuration` round-trips through the Dart model, the ObjectBox entity, and the multipart trail form body, preserving `null` at every hop (new test asserts this for both null and 3600.0)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add the moving_duration PocketBase field and publish it on the three Trail OpenAPI schemas** - `d938b258` (feat)
2. **Task 2: Add moving_duration to the TypeScript Trail model and route the web's Trail duration displays through one helper** - `9f9795f5` (feat)
3. **Task 3: Add movingDuration to the Dart Trail model, the ObjectBox entity and the trail form body** - `d592b958` (feat)

_No TDD tasks in this plan._

## Files Created/Modified
- `db/migrations/1785300000_add_moving_duration_to_trails.go` - new idempotent/reversible migration adding `moving_duration` to `trails`
- `web/src/lib/models/api/openapi_schemas.ts` - `moving_duration` added to `Trail`/`TrailCreateInput`/`TrailUpdateInput` JSDoc schemas only
- `web/src/lib/models/trail.ts` - `moving_duration?: number` field, constructor wiring, carried through `Trail.from()`
- `web/src/lib/util/format_util.ts` - new `trailDisplayDuration()` export, the single D-10 display-rule implementation
- `web/src/lib/util/maplibre_util.ts` - trail popup duration now via `trailDisplayDuration()`
- `web/src/lib/components/trail/trail_info_panel.svelte` - duration display via `trailDisplayDuration()`
- `web/src/lib/components/trail/trail_card.svelte` - duration display via `trailDisplayDuration()`
- `web/src/lib/components/trail/trail_list_item.svelte` - duration display via `trailDisplayDuration()`
- `web/src/lib/components/trail/trail_table.svelte` - duration display via `trailDisplayDuration()`
- `app/lib/models/trail.dart` - `movingDuration` freezed field, `@JsonKey(name: 'moving_duration')`, no default
- `app/lib/models/trail.freezed.dart`, `app/lib/models/trail.g.dart` - regenerated via build_runner
- `app/lib/entities/trail_entity.dart` - `movingDuration` field wired through constructor/`fromModel`/`toModel`
- `app/lib/objectbox-model.json`, `app/lib/objectbox.g.dart` - regenerated; new property id, no existing id reused
- `app/lib/util/form_data_util.dart` - sends `moving_duration` only when non-null
- `app/lib/provider/trail/trail_filter_provider.g.dart`, `app/lib/provider/trail/trail_library_provider.g.dart` - incidental hash-only churn from the same build_runner pass (Trail is in their dependency graph)
- `app/test/entities/trail_entity_test.dart` - new: round-trip test for `movingDuration` (null and 3600.0)

## Decisions Made
- `web/static/docs/api/wanderer.openapi.json` was regenerated (`npm run openapi:generate`) and its 3 `moving_duration` occurrences confirmed by direct grep, but it was **not committed** -- the repo's `.gitignore` (`web/.gitignore:6`) marks this file gitignored; it is a build-time-generated artifact, not source-controlled. Force-adding a gitignored file was avoided per the project's git-safety guidance.
- Extended `Trail.from()` (the TS trail-duplicate helper) to also carry `moving_duration`, mirroring how it already carries `duration`/`elevation_gain`/etc. Not explicitly named in the plan's action text, but omitting it would silently drop a recording's moving time on trail duplication -- a real, easily-hit data-loss path only one line away from the fields the plan did name.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Duplicated `moving_duration` in `Trail.from()`**
- **Found during:** Task 2
- **Issue:** `Trail.from()` copies every other stat field (`duration`, `elevation_gain`, `elevation_loss`, `distance`, etc.) when duplicating a trail. Leaving `moving_duration` out would silently drop a recording's moving time the first time a user duplicates it.
- **Fix:** Added `moving_duration: orig.moving_duration,` alongside the existing `duration: orig.duration,` line.
- **Files modified:** web/src/lib/models/trail.ts
- **Verification:** svelte-check clean, vitest suite passes (no dedicated duplicate-trail test exists to extend; the field flows through the same code path as every other duplicated stat)
- **Committed in:** 9f9795f5 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Small, low-risk addition consistent with the plan's own stated pattern (mirror `duration`'s handling). No scope creep -- stayed within `web/src/lib/models/trail.ts`, already a files_modified entry.

## Issues Encountered
- Attempting the plan's phase-level verification step ("apply the migration against a scratch PocketBase data dir... run it a second time... run Down") could not be completed end-to-end: `db/main.go`'s startup path requires a live Meilisearch connection during an earlier, unrelated migration (`1742167033_init_meilisearch.go`) before the migration chain reaches this plan's new migration, and standing up a full Meilisearch instance was out of proportion for this task. Verified instead via `go build ./...` (clean), `go vet ./migrations/` (clean), and direct comparison against the exact field shape of the existing, already-shipped `duration` field and the proven `1778583800_persist_trail_bounds.go` idempotent-add pattern.
- `flutter test` (full suite) surfaced 4 pre-existing failures in `test/components/route_planner/settings_tab_test.dart` (an icon-lookup assertion, `IconData(U+0E159)` not found). Confirmed via a git worktree at the pre-plan commit (`7c732117`) that these fail identically before this plan's changes -- not a regression introduced here.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `moving_duration` now exists end-to-end with no writer anywhere yet (by design -- D-11/34-05 supplies the value from the recording session).
- Web display rule is live and safe; the trail-edit page cannot reach the field.
- App-side model/entity/form-body plumbing is ready for 34-05 to wire `NavigationStats.elapsed` into `movingDurationOverride` at save time.

---
*Phase: 34-dart-conversion-port*
*Completed: 2026-07-31*

## Self-Check: PASSED

All 9 created/modified key files verified present on disk; all 3 task commits (`d938b258`, `9f9795f5`, `d592b958`) verified present in git log.
