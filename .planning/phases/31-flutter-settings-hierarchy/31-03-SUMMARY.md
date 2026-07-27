---
phase: 31-flutter-settings-hierarchy
plan: 03
subsystem: mobile-app
tags: [flutter, freezed, region-hierarchy, gap-closure]

# Dependency graph
requires:
  - phase: 31-flutter-settings-hierarchy (Plan 01)
    provides: "RegionHierarchyRow/RegionTreeNode models, buildRegionTree/flattenVisible pure tree functions"
  - phase: 31-flutter-settings-hierarchy (Plan 02)
    provides: "SettingsOfflineRegionsScreen renders the region catalog as a collapsible group/leaf hierarchy"
provides:
  - "RegionHierarchyRow.enabled (nullable, leaf-only) parsed from the backend's already-emitted enabled JSON key"
  - "RegionTreeNode.enabled carried through buildRegionTree"
  - "pruneToDownloadable pure function: removes non-enabled leaves and any group left childless"
  - "Settings > Offline Maps/Regions now renders only downloadable regions plus their ancestor chain"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Post-order retainWhere prune of an ephemeral, in-place-mutable tree (pruneToDownloadable), matching the codebase's existing 'ephemeral tree rebuilt per fetch' precedent (RegionTreeNode/D-03)"
    - "Nullable-no-@Default Freezed field to distinguish 'group row / legacy row' (null) from an explicit false/true leaf flag"

key-files:
  created: []
  modified:
    - app/lib/models/region_hierarchy_row.dart
    - app/lib/models/region_hierarchy_row.freezed.dart
    - app/lib/models/region_hierarchy_row.g.dart
    - app/lib/models/region_tree_node.dart
    - app/lib/util/region_tree_util.dart
    - app/lib/routes/settings_offline_regions_screen.dart
    - app/test/util/region_tree_util_test.dart
    - app/test/provider/region_provider_test.dart
    - app/test/routes/settings_offline_regions_screen_test.dart

key-decisions:
  - "enabled parses as a nullable bool? with NO @Default -- an absent key (group rows, legacy rows) reads as null, which pruneToDownloadable treats identically to false (both dropped); only an explicit true survives"
  - "pruneToDownloadable ignores build/status entirely, per the resolved product decision -- an enabled==true leaf that is still status=='building' continues to render, matching the screen's existing disabled-row treatment for in-progress builds"
  - "No Go backend changes required -- db/routes/regions_get.go already emitted entry[\"enabled\"] = r.GetBool(\"enabled\") for every leaf row before this plan started; the gap was entirely client-side (unparsed field)"

requirements-completed: [APPUI-01]

# Metrics
duration: ~12min
completed: 2026-07-27
---

# Phase 31 Plan 03: Prune Settings Hierarchy to Downloadable Regions Summary

**Client-side gap closure: parse the backend's already-on-the-wire leaf `enabled` flag onto `RegionHierarchyRow`/`RegionTreeNode`, add a pure `pruneToDownloadable` step, and wire it into `_refreshCatalog` so Settings > Offline Maps/Regions renders only the admin-enabled downloadable leaves (plus their ancestor group chain) instead of the full ~1306-row seeded CoMaps catalog.**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-07-27T15:35:00Z
- **Completed:** 2026-07-27T15:47:37Z
- **Tasks:** 2 completed
- **Files modified:** 9 (6 lib files including 2 regenerated Freezed/JSON parts, 3 test files)

## Accomplishments
- `RegionHierarchyRow` now parses the backend's leaf-only `enabled` key (`bool?`, no `@Default`, so an absent key — every group row, and any legacy row — reads as `null`); regenerated `region_hierarchy_row.freezed.dart`/`.g.dart` via `build_runner`
- `RegionTreeNode` carries `enabled` through `buildRegionTree`'s node construction
- New pure `pruneToDownloadable(List<RegionTreeNode> roots)` in `region_tree_util.dart`: a leaf survives iff `enabled == true`; a group survives iff it has at least one surviving child after post-order `retainWhere` pruning of its own descendants — proven by 8 new unit tests covering every bullet in the plan's behavior spec (enabled-leaf survives, disabled/null-leaf dropped, childless-group dropped, mixed group keeps only enabled children, deep nested chains in both directions, and explicit confirmation that pruning never inspects a status/build field)
- Wired into `_refreshCatalog`'s single tree-build call site: `_treeRoots = pruneToDownloadable(buildRegionTree(hierarchyRows))` — the only change to `settings_offline_regions_screen.dart`; every render method, the offline empty-state branch, and the `byId` render-time join are byte-for-byte unchanged
- Widget test fixture extended with an enabled leaf (`de-nrw`, now explicit `enabled: true`) and a disabled leaf/group pair (`asia`/`jp-x`, `enabled: false`) with no backing `RegionEntity`; new widget test proves the disabled leaf and its now-childless group never render while the enabled subtree (`Europe`) still does
- `parseRegionHierarchyRows` extended with a round-trip test: leaf `true`/`false` parse correctly, a group fixture (no `enabled` key) parses `null`

## Task Commits

Each task was committed atomically:

1. **Task 1: Parse `enabled`, carry it onto the tree node, add pruneToDownloadable + unit tests** - `da6f7cd6` (feat)
2. **Task 2: Wire pruning into the screen + extend widget tests to prove pruning** - `1d566b5b` (feat)

**Plan metadata:** commit pending (docs: complete plan)

## Files Created/Modified
- `app/lib/models/region_hierarchy_row.dart` - Added nullable `enabled` field (`@JsonKey(name: 'enabled') bool? enabled`, no `@Default`)
- `app/lib/models/region_hierarchy_row.freezed.dart` / `.g.dart` - Regenerated via `dart run build_runner build --delete-conflicting-outputs`
- `app/lib/models/region_tree_node.dart` - Added `final bool? enabled;` + required constructor param
- `app/lib/util/region_tree_util.dart` - `buildRegionTree` now copies `row.enabled` onto each node; new `pruneToDownloadable` function
- `app/lib/routes/settings_offline_regions_screen.dart` - `_refreshCatalog`'s single `_treeRoots` assignment now wraps `buildRegionTree(hierarchyRows)` in `pruneToDownloadable(...)`
- `app/test/util/region_tree_util_test.dart` - `buildRow` fixture gained an `enabled` param; new `pruneToDownloadable` group (8 tests)
- `app/test/provider/region_provider_test.dart` - New `enabled` round-trip test in the `parseRegionHierarchyRows` group
- `app/test/routes/settings_offline_regions_screen_test.dart` - Fixture leaf `de-nrw` gained `enabled: true`; new disabled group/leaf (`asia`/`jp-x`) added; new widget test asserting the pruning outcome

## Decisions Made
- `enabled` declared nullable with no `@Default` (not defaulting to `false`) so the model's own shape documents the wire contract precisely: `null` genuinely means "no signal" (group row or pre-this-plan legacy row), which `pruneToDownloadable` treats the same as an explicit `false` — both are dropped, only `true` survives.
- Confirmed via direct read of `db/routes/regions_get.go` (lines 65-70, already present in the working tree before this plan started) that `entry["enabled"] = r.GetBool("enabled")` is set for every leaf row before the archive/build-status lookup — so an enabled leaf's `enabled` value never depends on or interacts with its `status` field. No backend change was needed or made.

## Deviations from Plan

None - plan executed exactly as written. The single git-hygiene wrinkle: `app/lib/routes/settings_offline_regions_screen.dart` had a pre-existing, unrelated uncommitted change (two `FaIcon` size-16 tweaks) in the working tree when this plan started. That change was left untouched and unstaged — only the plan's one-line `_treeRoots` hunk was staged (via a hand-built single-hunk patch applied with `git apply --cached`) and committed, so the unrelated icon-size edit remains exactly as it was, uncommitted, out of this plan's scope.

## Issues Encountered
- Pre-existing `unused_import` warning (`package:wanderer/models/region_tree_node.dart`) in `app/test/util/region_tree_util_test.dart` — confirmed via `git show` against the pre-plan commit (`f3da9548`) that this warning already existed before Task 1's changes (the import was already unused in that file). Out of scope per the executor's scope-boundary rule; not fixed.
- Working tree at plan start also had several unrelated in-progress changes (a `region_id`→`path` region-archives migration touching `db/`, `app/lib/entities/region_entity.dart`, `app/lib/services/tile_repository_manager.dart`, etc., plus icon-size tweaks in the screen file). Mid-session, these were committed by a separate concurrent process as `e756777a fix(regions): download by path, add path FK integrity + disk reconcile` — none of that work is part of this plan's task scope and none of it was touched or re-committed here.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 31 (Flutter Settings Hierarchy) now closes its single remaining UAT gap: Settings > Offline Maps/Regions renders only admin-enabled downloadable regions plus the ancestor group chain needed to reach them, matching the resolved product decision (filter on `enabled == true` only; enabled-but-still-building leaves keep rendering). All 42 tests across `region_tree_util_test.dart`/`region_provider_test.dart`/`settings_offline_regions_screen_test.dart` pass; `flutter analyze` is clean on all touched lib files and the touched test file. The plan's Task 2 `<human-check>` (on-device confirmation of the pruned hierarchy, still-building enabled regions, and unregressed expand/filter/disk-usage behavior) was not performed in this autonomous execution — per `workflow.human_verify_mode = end-of-phase`, this is deferred to the phase-level verification pass, matching the precedent already set by Plan 02.

---
*Phase: 31-flutter-settings-hierarchy*
*Completed: 2026-07-27*

## Self-Check: PASSED

All modified files found on disk; both task commit hashes (da6f7cd6, 1d566b5b) found in git log.
