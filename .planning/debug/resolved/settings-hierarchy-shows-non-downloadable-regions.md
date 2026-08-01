---
status: resolved
trigger: "settings-hierarchy-shows-non-downloadable-regions"
created: 2026-07-27T00:00:00Z
updated: 2026-07-27T16:12:55Z
---

## Current Focus

hypothesis: CONFIRMED - Phase 31's hierarchy render pipeline never carries or filters on the backend's leaf-only `enabled` flag, so the full ~1306-row world CoMaps catalog (returned unconditionally by regions_get.go "regardless of enabled") renders in the Settings hierarchy, including every group-chain ancestor needed to reach non-enabled leaves.
test: Traced RegionHierarchyRow -> buildRegionTree -> flattenVisible -> ListView.builder pipeline; confirmed no field/step anywhere carries or checks `enabled`.
expecting: N/A - root cause confirmed, goal is find_root_cause_only
next_action: Return ROOT CAUSE FOUND diagnosis (no fix_and_verify per mode)

## Symptoms

expected: Settings → Offline Maps/Regions hierarchy only displays leaf regions that are actually downloadable (i.e., have real map data available to fetch), plus the ancestor group rows required to reach those leaves in the tree. Non-downloadable regions (and group branches containing only non-downloadable leaves) should not appear at all.
actual: The hierarchy renders the full region catalog fetched from the backend — every group and leaf node from `buildRegionTree`/`flattenVisible`, regardless of whether a given leaf region actually has downloadable content.
errors: None reported
reproduction: Test 1 in UAT (.planning/phases/31-flutter-settings-hierarchy/31-UAT.md) — open Settings → Offline Maps/Regions on-device.
started: Discovered during UAT of Phase 31, immediately after the collapsible hierarchy shipped.

## Eliminated

<!-- APPEND only -->

- hypothesis: "kind == 'leaf' already unambiguously means downloadable, so the tree needs no additional filtering"
  evidence: "regions_get.go's own doc comment (lines 21-29): every row (group AND leaf) is returned 'regardless of enabled', and a leaf's `enabled` field is completely separate from `kind`. `enabled` is what actually gates whether the region ever gets archived (db/services/regions/builder.go BuildAllLocked queries kind='leaf' AND enabled=true). A leaf can be kind=='leaf' and enabled==false forever, meaning it will never have real downloadable content."
  timestamp: 2026-07-27T00:20:00Z

## Evidence

<!-- APPEND only -->

- timestamp: 2026-07-27T00:05:00Z
  checked: app/lib/routes/settings_offline_regions_screen.dart (full file)
  found: "_refreshCatalog() calls fetchHierarchyRows() and passes every row straight into buildRegionTree(hierarchyRows) with zero filtering. build() then calls flattenVisible(_treeRoots!, _expandedIds, filterMatches) with no downloadable/enabled predicate anywhere in the pipeline. The only per-leaf filtering that exists is catalogStatus building/error -> _buildDisabledRow (still SHOWN, just visually disabled) vs ready -> _buildActiveRow. Nothing removes a row from visibleRows based on enabled/downloadable state."
  implication: "The render pipeline has no concept of 'not downloadable, hide entirely' - only 'downloadable but not ready yet, show disabled'."

- timestamp: 2026-07-27T00:07:00Z
  checked: app/lib/util/region_tree_util.dart (buildRegionTree, computeDefaultExpanded, flattenVisible, computeFilterMatches - full file)
  found: "None of the four ported functions accept or apply any 'downloadable'/'enabled' predicate for pruning. computeDefaultExpanded takes a hasDownloadOrInProgress predicate but that only controls default EXPAND state, not visibility/inclusion. flattenVisible only prunes by filterMatches (the user's search query), never by downloadability."
  implication: "This is the exact function ported 1:1 from the ADMIN region picker's regions_ui.html - which intentionally shows the WHOLE tree (enabled and disabled leaves) because the admin's job is to toggle enablement. Porting it unmodified to the end-user screen carries over that 'show everything' behavior inappropriately."

- timestamp: 2026-07-27T00:09:00Z
  checked: app/lib/models/region_hierarchy_row.dart and app/lib/models/region_tree_node.dart (full files)
  found: "RegionHierarchyRow's fields are exactly id/name/kind/parent/path/depth/sortOrder - no `enabled`, no `status`, no `bbox`. RegionTreeNode mirrors the same 7 fields. Neither model carries any signal of whether a leaf is actually downloadable."
  implication: "Even if a filter step were added, it can't currently distinguish enabled from disabled leaves - RegionHierarchyRow would need a new `enabled` field parsed from the backend's existing (but currently Flutter-unused) `enabled` JSON key."

- timestamp: 2026-07-27T00:11:00Z
  checked: db/routes/regions_get.go (full file, RegionsList handler)
  found: "Doc comment lines 21-29 states explicitly: 'Every row (both kind=group and kind=leaf) is returned, regardless of `enabled`, so a client can render group node names/labels for the whole tree... trimming is a trivial later filter, but omitting group rows now would mean re-adding data later.' Leaf rows DO carry `entry[\"enabled\"] = r.GetBool(\"enabled\")` (line 70) in the JSON response - the data exists on the wire, it's just never parsed/used by the Flutter app."
  implication: "The backend already deliberately punts 'downloadable' filtering to the client, and already exposes the exact field (`enabled`) needed to do it. The gap is entirely client-side: RegionHierarchyRow never parses `enabled`, so the client has no way to filter."

- timestamp: 2026-07-27T00:13:00Z
  checked: db/services/regions/builder.go (BuildAllLocked) and db/migrations/1785000000_create_regions_collection.go (schema comments)
  found: "BuildAllLocked only ever builds archives for `kind='leaf' AND enabled=true` regions. Migration comment: 'CATALOG-03: leaf-only enabled, default false' - every leaf is seeded with enabled=false by default; an admin must explicitly opt a region in via the admin picker (regions_ui.html) before it is ever built. A leaf with enabled=false never gets a region_archives row, so regions_get.go permanently reports status='building' for it (no archive record found) - it will NEVER transition to ready."
  implication: "`enabled` (not `kind`) is the real domain signal for 'has an admin actually made this region obtainable'. Of the ~1306 seeded CoMaps regions, only a small admin-curated subset is ever enabled=true; the rest are permanently non-downloadable placeholders that exist purely for hierarchy completeness."

- timestamp: 2026-07-27T00:15:00Z
  checked: app/lib/provider/region/region_provider.dart (parseRegionCatalog, RegionCatalogEntry, parseRegionHierarchyRows) and app/lib/models/region_catalog_entry.dart
  found: "grep across app/lib confirms `enabled` is NEVER parsed or referenced anywhere in the Flutter app in connection with regions (only unrelated `enabled:` widget-disable params in other screens). RegionCatalogEntry (the pre-existing flat-list model) also has no `enabled` field - it requires only bbox/status, both of which every leaf row carries unconditionally regardless of enabled state, so parseRegionCatalog never dropped disabled leaves either."
  implication: "The pre-Phase-31 flat list ALSO never filtered on enabled - meaning this specific 'show only downloadable' gap is not new to Phase 31's hierarchy per se, but Phase 31's tree view is what made the scale of the problem (nearly the full ~1306-row world CoMaps hierarchy, deeply nested) visually obvious enough for the user to flag it as 'unnecessary' - whereas a flat alphabetical list with search made the same underlying gap less noticeable."

- timestamp: 2026-07-27T00:17:00Z
  checked: .planning/phases/31-flutter-settings-hierarchy/31-CONTEXT.md and 31-RESEARCH.md (full files)
  found: "Phase 31's locked decisions (D-01 through D-07) cover tree-shape porting, expand defaults, group-node ephemerality, offline empty state, filter/search behavior, and sort_order - none of them mention filtering by `enabled`/downloadable status. CONTEXT.md's Phase Boundary explicitly frames the goal as rendering the hierarchy 'matching the shape of the admin-defined tree' and both docs' 'Deferred Ideas' sections say 'None - discussion stayed within phase scope.'"
  implication: "This is a genuine scope gap, not an implementation bug relative to the plan: the phase was scoped/researched/planned to mirror the ADMIN tree's rendering behavior (which intentionally shows all regions so an admin can toggle them) without ever separately deciding what the END-USER-facing screen's inclusion criteria should be."

## Resolution

root_cause: |
  Phase 31 ported the ADMIN region picker's tree-rendering algorithm (buildTree/flattenVisible/
  computeDefaultExpanded/computeFilterMatches from db/routes/regions_ext/regions_ui.html) verbatim
  into the end-user-facing Settings > Offline Maps/Regions screen, via a new RegionHierarchyRow model
  and buildRegionTree/flattenVisible in app/lib/util/region_tree_util.dart. That algorithm - by design,
  for its original admin-toggle-all-regions use case - renders every row the backend returns with no
  notion of "downloadable."

  The backend (db/routes/regions_get.go, RegionsList) deliberately and explicitly returns ALL ~1306
  seeded CoMaps region rows (both kind=group and kind=leaf) "regardless of enabled" (its own doc
  comment, lines 21-29), leaving inclusion-filtering to the client by design. A leaf's real
  "obtainable/downloadable" signal is the separate `enabled` boolean field (leaf-only, seeded
  default false per db/migrations/1785000000_create_regions_collection.go's CATALOG-03 comment) -
  only leaves with enabled=true are ever queried/built by db/services/regions/builder.go's
  BuildAllLocked, so an enabled=false leaf permanently reports status="building" and will never
  become real downloadable content.

  RegionHierarchyRow (app/lib/models/region_hierarchy_row.dart) and RegionTreeNode
  (app/lib/models/region_tree_node.dart) only carry id/name/kind/parent/path/depth/sortOrder - the
  `enabled` field, though already present in the backend's JSON response (regions_get.go line 70),
  is never parsed or referenced anywhere in the Flutter app. Consequently buildRegionTree/
  flattenVisible have no data available to prune non-enabled leaves or the group ancestors that
  exist solely to reach them, so the full world-region tree renders in Settings.

  This was never scoped as a decision in Phase 31's CONTEXT.md/RESEARCH.md (D-01..D-07 cover tree
  shape, expand defaults, filter/search, sort_order - none address inclusion-by-downloadability), so
  this is a genuine planning/scope gap rather than a deviation from the plan. The pre-existing flat
  list (pre-Phase-31) had the identical underlying gap (RegionCatalogEntry also never parses
  `enabled`) but its scale was less visually obvious in a flat, searchable, alphabetical list than in
  a nested hierarchy that now visibly surfaces near-complete continent/country group chains leading
  to disabled leaves.
fix:
verification:
files_changed: []
