---
phase: 11-trail-filter-subcategory-support
plan: 01
subsystem: ui
tags: [flutter, freezed, riverpod, meilisearch, trail-filter, subcategory]

requires:
  - phase: 10-category-subcategory-data-layer
    provides: Subcategory freezed model and subcategoryProvider; locale-aware Category names
provides:
  - TrailFilter.subcategory field (List<Subcategory>, defaults empty)
  - ID-based combined (category_id IN [...] OR subcategory_id IN [...]) clause in toFilterText()
  - defaultFilter initializes subcategory to empty list
  - Unit tests for all four category/subcategory selection permutations
affects: [11-03 TrailFilterScreen chips, 11-04 quick filter bar]

tech-stack:
  added: []
  patterns:
    - "Combined OR-group filter clause mirroring web trail_store.ts (ID-based category/subcategory)"
    - "@Default(<T>[]) list field addition to a freezed factory to avoid touching unrelated constructors"

key-files:
  created:
    - app/test/models/trail_filter_test.dart
  modified:
    - app/lib/models/trail.dart
    - app/lib/models/trail.freezed.dart
    - app/lib/provider/trail/trail_filter_provider.dart

key-decisions:
  - "Used @Default(<Subcategory>[]) form (not required) so only defaultFilter needed updating — RESEARCH Open Question 2 / Pitfall 3"
  - "Rewrote category clause to use record IDs (c.id/s.id) instead of names, targeting filterable category_id/subcategory_id attributes — fixes latent broken-category-filter bug (RESEARCH Pitfall 1)"
  - "Composed category and subcategory into a single OR group so both can be active simultaneously"

patterns-established:
  - "Pattern 1: ID-based combined OR-group category/subcategory filter clause mirroring web trail_store.ts"

requirements-completed: [FILTER-01, FILTER-03]

duration: 4min
completed: 2026-06-30
---

# Phase 11 Plan 01: TrailFilter Subcategory Support Summary

**TrailFilter now carries a subcategory list and emits an ID-based `(category_id IN [...] OR subcategory_id IN [...])` Meilisearch clause, replacing the broken name-based category filter.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-06-30T07:13:35Z
- **Completed:** 2026-06-30T07:17:28Z
- **Tasks:** 2 completed
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- Added `@Default(<Subcategory>[]) List<Subcategory> subcategory` to the `TrailFilter` freezed factory, with the matching `subcategory.dart` import.
- Set `subcategory: []` explicitly in `defaultFilter`.
- Regenerated `trail.freezed.dart` so `copyWith(subcategory:)` compiles.
- Rewrote the category block in `toFilterText()` to build a single OR group from record IDs: categories add `category_id IN [...]`, subcategories add `subcategory_id IN [...]`, joined with ` OR ` and wrapped in parentheses.
- Removed the legacy name-based `category IN ['<name>']` clause that targeted a now-unfilterable Meilisearch attribute (latent bug fix).
- Added `app/test/models/trail_filter_test.dart` covering category-only, subcategory-only, combined, and empty selection permutations (TDD RED then GREEN).

## Task Commits

| Task | Name | Type | Commit |
| ---- | ---- | ---- | ------ |
| 1 | Add TrailFilter.subcategory field and default | feat | 4b478312 |
| 2 (RED) | Failing tests for ID-based filter clause | test | 0d6c96c3 |
| 2 (GREEN) | Rewrite category clause to ID-based OR group | feat | 13247ec9 |

## Verification Results

- `flutter test test/models/trail_filter_test.dart` — all 4 tests pass.
- `flutter analyze lib/models/trail.dart lib/provider/trail/trail_filter_provider.dart` — no issues.
- `grep "category IN \[" app/lib/models/trail.dart` — no matches (legacy clause removed).
- `grep -E "(category_id|subcategory_id) IN" app/lib/models/trail.dart` — both clauses present.
- `dart run build_runner build --delete-conflicting-outputs` — succeeded (50 outputs).

## TDD Gate Compliance

- RED gate: `test(11-01)` commit `0d6c96c3` (3 behavioral tests failing before implementation; the empty-selection test passed trivially because the legacy clause used `category`, not `category_id`).
- GREEN gate: `feat(11-01)` commit `13247ec9` (all 4 tests passing after implementation).
- REFACTOR gate: not needed; implementation was clean.

## Deviations from Plan

None - plan executed exactly as written.

## Decisions Made

- Kept the `@Default(<Subcategory>[])` form so no constructor beyond `defaultFilter` required changes; a grep for `TrailFilter(` confirmed `defaultFilter` was the only constructor literal.
- Used single-quoted record IDs (`'c1'`, `'s1'`) in the emitted clause, consistent with the existing tag clause style and the web OR-group builder.

## Notes for Future Plans

- Plans 03 and 04 should read `TrailFilter.subcategory` and call `copyWith(subcategory: ...)`; the field and `defaultFilter` default are now in place.
- Several `*.g.dart` files (`category_preference_provider.g.dart`, `subcategory_preference_provider.g.dart`, `category_provider.g.dart`, `subcategory_provider.g.dart`, `trail_filter_provider.g.dart`) carry uncommitted riverpod hash-only regeneration noise that predates this plan; they were left out of scope (no functional change).

## Known Stubs

None.

## Self-Check: PASSED

- All created/modified files exist on disk.
- All three task commits (4b478312, 0d6c96c3, 13247ec9) present in git history.
