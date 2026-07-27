---
phase: 31-flutter-settings-hierarchy
plan: 01
subsystem: mobile-app
tags: [flutter, freezed, region-hierarchy, pocketbase, go, unit-testing]

# Dependency graph
requires:
  - phase: 29-polygon-based-extraction-and-region-api
    provides: "GET /api/v1/regions with hierarchy fields (id/name/kind/parent/path/depth)"
provides:
  - "sort_order field on every /api/v1/regions row (group and leaf)"
  - "RegionHierarchyRow: always-succeeding Freezed parse model for group+leaf rows"
  - "RegionTreeNode: mutable ephemeral tree node"
  - "buildRegionTree/computeDefaultExpanded/flattenVisible/computeFilterMatches pure tree functions"
  - "parseRegionHierarchyRows + RegionRepository.fetchHierarchyRows provider path"
affects: ["31-02 (hierarchical Settings screen build)"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Freezed API-response model with all-required fields except a @Default(0) forward-compat field (sortOrder)"
    - "Deliberate plain-mutable-class exception to Freezed-everything convention, documented inline (RegionTreeNode)"
    - "Dart record return type for a render-time tuple (flattenVisible), matching splitRegionTilePaths precedent"
    - "Per-element try/catch-and-skip parse posture extended to a second sibling parser (parseRegionHierarchyRows) without touching the original (parseRegionCatalog)"

key-files:
  created:
    - app/lib/models/region_hierarchy_row.dart
    - app/lib/models/region_tree_node.dart
    - app/lib/util/region_tree_util.dart
    - app/test/util/region_tree_util_test.dart
  modified:
    - db/routes/regions_get.go
    - app/lib/provider/region/region_provider.dart
    - app/test/provider/region_provider_test.dart

key-decisions:
  - "sort_order added to the base entry map in regions_get.go (above the leaf-only branch) so both group and leaf rows carry it, per D-07"
  - "RegionHierarchyRow declares parent as a required non-null String (never nullable) to match Go's r.GetString semantics, avoiding a null-vs-empty-string ambiguity bug class"
  - "flattenVisible returns a record type (List<({RegionTreeNode node, int depth})>) rather than mutating node.depth, matching tile_repository_manager.dart's splitRegionTilePaths precedent"
  - "fetchHierarchyRows is not covered by a repository-instantiation round-trip test — no established Store-construction pattern exists in this test harness for plain flutter test suites; covered by proxy via parseRegionHierarchyRows + the shared try/catch/rethrow shape"

patterns-established:
  - "A second sibling parser can extend the per-element try/catch/skip posture of an existing one without ever modifying the original (Pitfall 2 discipline)"

requirements-completed: [APPUI-01]

# Metrics
duration: 20min
completed: 2026-07-27
---

# Phase 31 Plan 01: Region Hierarchy Data Contracts & Pure Tree Algorithm Summary

**Backend sort_order fix + Freezed RegionHierarchyRow/RegionTreeNode models + 1:1-ported buildRegionTree/computeDefaultExpanded/flattenVisible/computeFilterMatches, all unit-tested, plus a provider fetch/parse path that recovers group rows the existing leaf parser silently drops.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-07-27
- **Tasks:** 3 completed
- **Files modified:** 3 (regions_get.go, region_provider.dart, region_provider_test.dart)
- **Files created:** 4 (region_hierarchy_row.dart + 2 generated parts, region_tree_node.dart, region_tree_util.dart, region_tree_util_test.dart) — 6 total including generated Freezed/JSON parts

## Accomplishments
- Fixed the D-07 backend gap: `GET /api/v1/regions` now emits `sort_order` on every row (group and leaf), not just implicitly via the DB column
- `RegionHierarchyRow` parses BOTH group and leaf rows without ever throwing (D-05) — the exact row `RegionCatalogEntry.fromJson` silently drops
- Ported all four tree functions 1:1 from the shipped admin page's JS (`regions_ui.html:515-667`), proven correct by 15 passing unit tests covering root/parent attachment, the `""`-root Pitfall 3 gotcha, sibling sortOrder ordering (Pitfall 4), expand/collapse visibility, and self-match vs. descendant-only-match filtering
- Added `parseRegionHierarchyRows`/`fetchHierarchyRows` as a parallel path in `region_provider.dart`, with zero changes to the existing `parseRegionCatalog`/ObjectBox pipeline

## Task Commits

Each task was committed atomically:

1. **Task 1: Backend sort_order field + RegionHierarchyRow + RegionTreeNode models** - `06a4a4a1` (feat)
2. **Task 2: Port the four pure tree functions + unit tests** - `dde775bf` (test, RED) → `892679ce` (feat, GREEN)
3. **Task 3: parseRegionHierarchyRows + fetchHierarchyRows provider path** - `093189aa` (feat)

**Plan metadata:** commit pending (docs: complete plan)

## Files Created/Modified
- `db/routes/regions_get.go` - Added `sort_order` to the base entry map (applies to group and leaf rows)
- `app/lib/models/region_hierarchy_row.dart` - `RegionNodeKind` enum + always-succeeding `RegionHierarchyRow` Freezed model
- `app/lib/models/region_tree_node.dart` - Plain mutable ephemeral tree node, documented Freezed exception
- `app/lib/util/region_tree_util.dart` - `buildRegionTree`/`computeDefaultExpanded`/`flattenVisible`/`computeFilterMatches`
- `app/test/util/region_tree_util_test.dart` - 15 unit tests covering all four functions' behavior
- `app/lib/provider/region/region_provider.dart` - Added `parseRegionHierarchyRows` + `RegionRepository.fetchHierarchyRows`
- `app/test/provider/region_provider_test.dart` - Added `parseRegionHierarchyRows` test group (group/leaf/malformed/sort_order-default/non-List cases)

## Decisions Made
- Kept `RegionHierarchyRow.parent` as a required non-null `String` (never `String?`), matching Go's `r.GetString("parent")` semantics exactly — avoids introducing a null-vs-empty-string distinction that doesn't exist on the wire.
- `fetchHierarchyRows` has no dedicated repository-instantiation round-trip test since constructing a `RegionRepository` requires an ObjectBox `Store`, and no existing `_test.dart` file in this codebase opens one (only on-device spike harnesses do). Documented in a code comment in the test file; the method's behavior is fully covered by proxy through `parseRegionHierarchyRows` and the shared try/catch/rethrow shape already proven by the `fetchRegionCatalog` tests.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Plan 02 (hierarchical Settings screen) can consume `RegionHierarchyRow`, `RegionTreeNode`, all four tree functions, and `RegionRepository.fetchHierarchyRows` exactly as built here — no further tree-contract work needed. The backend now serves `sort_order` on every row, so sibling ordering will be correct as soon as Plan 02 wires the fetch into the screen.

---
*Phase: 31-flutter-settings-hierarchy*
*Completed: 2026-07-27*

## Self-Check: PASSED

All created/modified files found on disk; all 4 task commit hashes (06a4a4a1, dde775bf, 892679ce, 093189aa) found in git log.
