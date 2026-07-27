---
status: complete
phase: 31-flutter-settings-hierarchy
source: [31-VERIFICATION.md]
started: 2026-07-27T00:00:00Z
updated: 2026-07-27T16:20:00Z
---

## Current Test

[testing complete]

## Tests

### 1. On-device collapsible hierarchy walkthrough
expected: Group nodes expand/collapse to reveal child groups and leaf regions matching the admin tree; per-region Vector/DEM controls and disk-usage summary work unchanged inside the new hierarchical layout.
result: issue
reported: "It works but is unnecessary. Only show the regions (and their parents) that are actually downloadable."
severity: minor

### 2. On-device pruned-hierarchy walkthrough
expected: Collapsible hierarchy renders only admin-enabled downloadable regions plus ancestors; confirm an enabled-but-still-building region still appears (disabled row); confirm expand/collapse, per-region Vector/DEM controls, search/filter, and disk-usage total all still work; confirm the offline empty state appears only when genuinely offline, not merely because pruning yielded an empty tree.
result: pass

## Summary

total: 2
passed: 1
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- truth: "Settings → Offline Maps/Regions shows the collapsible hierarchy with group and leaf rows matching the admin-defined tree"
  status: resolved
  reason: "User reported: It works but is unnecessary. Only show the regions (and their parents) that are actually downloadable. Fixed by 31-03-PLAN.md: RegionHierarchyRow/RegionTreeNode now parse the backend enabled flag and pruneToDownloadable removes non-enabled leaves and childless group ancestors before flattenVisible runs. Verified against actual code/tests in 31-VERIFICATION.md (re-verification, 2026-07-27)."
  severity: minor
  test: 1
  root_cause: |
    Phase 31 ported the admin region-picker's tree algorithm (buildRegionTree/flattenVisible in
    app/lib/util/region_tree_util.dart) verbatim from db/routes/regions_ext/regions_ui.html, which was
    built for the admin's "toggle every region on/off" workflow and has no notion of "downloadable."
    The backend (db/routes/regions_get.go) deliberately returns all ~1306 seeded CoMaps rows regardless
    of the leaf-only `enabled` flag (doc comment: filtering is left to the client by design). Only
    kind='leaf' AND enabled=true rows are ever built (db/services/regions/builder.go BuildAllLocked);
    disabled leaves never get a region_archives row and are permanently stuck at status="building".
    RegionHierarchyRow/RegionTreeNode never parse `enabled` at all (verified via repo-wide grep), so
    buildRegionTree/flattenVisible have no data to prune non-downloadable leaves or the group ancestors
    that exist solely to reach them. Not a Phase 31 planning decision — 31-CONTEXT.md/31-RESEARCH.md
    (D-01–D-07) never address inclusion-by-downloadability; both "Deferred Ideas" sections read "None".
    Product decision (resolved via AskUserQuestion): filter to `enabled == true` only — keeps showing
    enabled-but-still-building leaves (matches existing disabled-row treatment), does not additionally
    require status == "ready".
  artifacts:
    - path: "app/lib/models/region_hierarchy_row.dart"
      issue: "Missing `enabled` field — backend already sends it for leaf rows, never parsed."
    - path: "app/lib/util/region_tree_util.dart"
      issue: "buildRegionTree/flattenVisible have no pruning step for non-downloadable leaves or now-childless group ancestors."
    - path: "app/lib/routes/settings_offline_regions_screen.dart"
      issue: "Passes every hierarchy row straight through to render with no downloadable filter."
  missing:
    - "Add `enabled` (nullable, leaf-only) to RegionHierarchyRow, parsed from the backend response."
    - "Insert a pruning step after buildRegionTree (or folded into it) that removes leaf nodes where enabled != true, then recursively removes any group node left with zero remaining children, before flattenVisible runs."
  debug_session: ".planning/debug/settings-hierarchy-shows-non-downloadable-regions.md"
