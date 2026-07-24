---
phase: 27-legacy-cleanup
plan: 02
subsystem: mobile-app
tags: [flutter, objectbox, freezed, build_runner, cleanup, dead-code-removal]

# Dependency graph
requires:
  - phase: 27-legacy-cleanup
    plan: 01
    provides: Trail-scoped tile-download methods, generating-state wiring, showGenerating(), and map_cell.dart all removed with zero remaining references
provides:
  - TrailEntity without pmTiles/demPmTiles field declarations or TrailEntityMapping bridge references
  - Trail freezed model without pmTiles/demPmTiles fields
  - Regenerated trail.freezed.dart/trail.g.dart/objectbox-model.json/objectbox.g.dart with the two ObjectBox property UIDs retired
  - Zero remaining references to the legacy trail-scoped tile system across app/lib and app/test
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - app/lib/entities/trail_entity.dart
    - app/lib/models/trail.dart
    - app/lib/models/trail.freezed.dart
    - app/lib/models/trail.g.dart
    - app/lib/objectbox-model.json
    - app/lib/objectbox.g.dart
    - app/lib/provider/map_style_json_provider.g.dart
    - app/lib/provider/trail/trail_download_state_provider.g.dart

key-decisions:
  - "Regeneration touched two riverpod provider hash files (map_style_json_provider.g.dart, trail_download_state_provider.g.dart) as an unavoidable side effect of the single project-wide build_runner pass -- their logic is unchanged, only the compile-time source hash used for provider identity shifted"
  - "No ObjectBox migration step performed -- field removal is a supported regeneration for a pre-production app (D-06); build_runner's own log confirms both properties were cleanly retired from the model (\"not found in the code, removing from the model\")"

patterns-established: []

requirements-completed: [CLEAN-01]

# Metrics
duration: 4min
completed: 2026-07-24
---

# Phase 27 Plan 02: Remove pmTiles/demPmTiles from TrailEntity/Trail and Regenerate Summary

**Deleted the persisted `pmTiles`/`demPmTiles` fields from `TrailEntity` (ObjectBox) and `Trail` (freezed) together, then ran a single `build_runner build --delete-conflicting-outputs` pass to regenerate `trail.freezed.dart`, `trail.g.dart`, `objectbox-model.json`, and `objectbox.g.dart` with both property UIDs cleanly retired.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-07-24T15:40:38Z
- **Completed:** 2026-07-24T15:43:54Z
- **Tasks:** 2 completed
- **Files modified:** 8

## Accomplishments
- Removed `TrailEntity.pmTiles`/`demPmTiles` field declarations and both `TrailEntityMapping.toModel()` bridge references (D-06)
- Removed `Trail.pmTiles`/`demPmTiles` freezed fields from the `Trail` constructor
- Ran `build_runner build --delete-conflicting-outputs` exactly once, at the end of the phase, after all Phase 27 source edits (27-01's service surgery + `map_cell.dart` deletion, and this plan's Task 1 field removal) — matching constraint 4 / Pitfall 2
- Confirmed via `build_runner`'s own log that both ObjectBox properties were cleanly retired ("Property TrailEntity.pmTiles(...) not found in the code, removing from the model"; same for `demPmTiles`) — no migration step needed
- `grep -rn "pmTiles\|demPmTiles\|showGenerating\|onGeneratingChanged\|MapCellInfo\|map_cell\|_downloadMapTiles\|_fetchCellList\|_pollUntilReady" app/lib app/test --include="*.dart"` returns zero lines — the whole phase's legacy trail-scoped tile system is gone with zero remaining references (ROADMAP success criterion #1)
- Completes CLEAN-01 (27-01 handled the runtime portion; this plan completes the persistence portion)

## Task Commits

Each task was committed atomically:

1. **Task 1: Remove pmTiles/demPmTiles from TrailEntity and Trail together (no regeneration yet)** - `4fbc5c8b` (refactor)
2. **Task 2: Regenerate generated code once and run the final regression gate** - `7007d5b9` (refactor)

## Files Created/Modified
- `app/lib/entities/trail_entity.dart` - `pmTiles`/`demPmTiles` field declarations and their `TrailEntityMapping.toModel()` bridge references deleted
- `app/lib/models/trail.dart` - `pmTiles`/`demPmTiles` freezed fields deleted from the `Trail` constructor
- `app/lib/models/trail.freezed.dart` - regenerated; `pmTiles`/`demPmTiles` removed from `copyWith`/`==`/`hashCode`/`toString`/`when`/`maybeWhen`/`whenOrNull`
- `app/lib/models/trail.g.dart` - regenerated; `pmTiles`/`demPmTiles` removed from `fromJson`/`toJson`
- `app/lib/objectbox-model.json` - regenerated; both property UIDs retired from the `TrailEntity` model definition
- `app/lib/objectbox.g.dart` - regenerated; both property bindings removed from the ObjectBox entity binding
- `app/lib/provider/map_style_json_provider.g.dart` - riverpod provider source hash shifted (no logic change) as a side effect of the whole-project regeneration pass
- `app/lib/provider/trail/trail_download_state_provider.g.dart` - riverpod provider source hash shifted (no logic change) as a side effect of the whole-project regeneration pass

## Decisions Made
- Followed the plan's Pitfall-2 guidance exactly: edited `trail_entity.dart` and `trail.dart` together in Task 1 (their hand-written `TrailEntityMapping` bridge references both sides) and deliberately left the tree non-compiling with stale generated code between Task 1 and Task 2 — this is expected, not a bug
- Did not hand-edit any generated file; `build_runner --delete-conflicting-outputs` produced all four regenerated files in one pass
- Included the two incidentally-touched riverpod `.g.dart` hash files in Task 2's commit rather than reverting them, since they are a direct, unavoidable output of the single sanctioned regeneration pass (not a separate edit)

## Deviations from Plan

None — plan executed exactly as written. Both tasks completed with no auto-fixes required.

## Issues Encountered

`flutter analyze` and `flutter test` both reproduced the exact same pre-existing, out-of-scope baseline that 27-01-SUMMARY.md already confirmed unrelated to this phase's file scope:
- 37 analyzer issues (info/warning level, no errors) in `map_screen.dart`, `icon_util.dart`, `navigation_stats_provider.dart`, `settings_categories_screen.dart`, `settings_subcategories_screen.dart` — identical file list and issue count to 27-01's confirmed baseline
- 4 failing tests in `test/components/route_planner/settings_tab_test.dart` — identical to 27-01's confirmed baseline; re-confirmed here via a targeted grep showing this test file has zero references to `pmTiles`, `demPmTiles`, or any file this plan modified

Both are logged here rather than fixed, per the executor's scope-boundary rule (out of scope for this plan's `files_modified`, and already independently confirmed pre-existing by the prior wave).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CLEAN-01 is now fully complete: both the runtime portion (27-01) and the persistence portion (this plan) are done. Zero remaining references to the legacy trail-scoped tile system exist across `app/lib` and `app/test`.
- ROADMAP Phase 27 success criterion #1 (zero remaining references, app builds/runs) and #3 (no functional regression — region-based offline download still works, 26-04/26-05 invariants green) are both satisfied.
- ROADMAP Phase 27 success criterion #2 (legacy-file sweep, CLEAN-02) remains intentionally descoped per D-05 — no task in either 27-01 or 27-02 addresses it.
- No blockers for closing out Phase 27 / milestone v1.6.

---
*Phase: 27-legacy-cleanup*
*Completed: 2026-07-24*

## Self-Check: PASSED

- FOUND: app/lib/entities/trail_entity.dart
- FOUND: app/lib/models/trail.dart
- FOUND: app/lib/objectbox-model.json
- FOUND commit: 4fbc5c8b
- FOUND commit: 7007d5b9
