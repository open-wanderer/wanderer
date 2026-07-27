# Phase 31: Flutter Settings Hierarchy - Context

**Gathered:** 2026-07-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Settings → Offline Maps/Regions in the Flutter app renders the region catalog as a collapsible hierarchy (group/leaf tree) matching the shape of the admin-defined tree (Phase 30), replacing today's flat list — while every existing per-region Vector/DEM download/cancel/delete action and the disk-usage summary keep working exactly as they do today, now nested inside the tree. Covers APPUI-01, APPUI-02 only. No changes to the archive cron, extraction pipeline, or the PocketBase admin page (those are Phase 29/30, already shipped). One small, surgical backend touch is in scope as a decision below (adding `sort_order` to `GET /api/v1/regions`), but it does not expand this phase's UI/UX scope.

</domain>

<decisions>
## Implementation Decisions

### Tree rendering
- **D-01:** Port the admin page's flatten-list algorithm (`buildTree` / `computeDefaultExpanded` / `flattenVisible` from `db/routes/regions_ext/regions_ui.html:500-582`) to Dart, rather than nesting Flutter's built-in `ExpansionTile`. Build the tree once from the flat API response, compute a flat depth-tagged array of currently-visible rows based on expand state, and render it via `ListView.builder` with depth-based indentation. Leaf rows keep their existing `_buildRegionRow`/`_buildVectorTile`/`_buildDemTile` rendering, just indented — no rewrite of the download-action UI.
- **D-02:** Default expand state mirrors the admin's rule (its D-07): a branch auto-expands only if it contains a leaf the user has already downloaded (Vector and/or DEM present) or mid-download; everything else starts collapsed. Gives an at-a-glance view of existing state on a small screen without a wall of nodes.

### Group node data handling
- **D-03:** Group rows (`kind != "leaf"`) are NOT persisted to ObjectBox. They exist only as an ephemeral client-side tree structure, rebuilt from the API's flat list on each fetch, used purely to organize/nest the existing `RegionEntity` leaf rows for rendering. No new ObjectBox entity, no group-specific local state to keep in sync — groups own no downloadable content, disk usage, or version, so there is nothing to persist offline.
- **D-04 (deliberate scoped exception to APPUI-02):** When offline (no fresh catalog fetch succeeds and no cached hierarchy shape exists), the screen shows an **empty state**, not a fallback flat list. The user confirmed this explicitly: previously-downloaded regions become unmanageable (no cancel/delete, no disk-usage view) while offline. This is a deliberate, scoped regression from "no download-UX regression" for the offline-only case — call this out explicitly to the researcher/planner/verifier as an accepted trade-off, not an oversight.
- **D-05:** Because `RegionCatalogEntry.fromJson` currently requires `bbox`/`status` and silently drops group rows, the parsing layer needs a lightweight in-memory group-node representation (separate from `RegionCatalogEntry`, which stays leaf/download-focused) so group rows survive parsing long enough to build the tree.

### Search/filter
- **D-06:** Filter behavior mirrors the admin page's `computeFilterMatches`: typing narrows the tree to matching leaves/groups plus their full ancestor chain, auto-expanded, so matches are visible without extra taps. Not a "flatten to matches only" simplification — keep the hierarchy visible during filtering, same as the admin page.

### Sort order
- **D-07:** Add `sort_order` to `GET /api/v1/regions`'s response (`db/routes/regions_get.go`'s entry map) — the DB column already exists (`db/migrations/1785000000_create_regions_collection.go:36,67`) but the handler doesn't surface it yet, a real gap not a style choice. The Flutter tree sorts siblings within a group by `sort_order` (matching the admin tree's `buildTree`, which already sorts this way), giving CoMaps-canonical ordering and visual consistency between admin and app. This is a small, surgical Go change flagged for research/planning — it does not otherwise expand this phase's scope.

### Claude's Discretion
- Exact debounce timing on the filter input, if any.
- Visual treatment of the empty state shown offline (D-04) — icon, copy, whether it explains why (e.g. "Connect to the internet to manage regions").
- Whether the depth-indentation uses fixed padding per level or some other visual grouping (borders, background tint) — as long as it reads clearly on a phone-width screen.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Design rationale
- `.planning/notes/streamlined-region-definition.md` — full design decision trail for the region catalog + hierarchy across the whole v1.7 milestone.

### Admin tree algorithm to port (Phase 30, already shipped, reference only)
- `db/routes/regions_ext/regions_ui.html:500-582` — `buildTree(rows)`, `computeDefaultExpanded(roots)`, `flattenVisible(roots, expandedSet, filterMatchSet)`, `computeFilterMatches(roots, query)`. This is the algorithm to port to Dart per D-01/D-02/D-06, not the Alpine.js code itself.
- `.planning/phases/30-admin-region-picker-ui/30-CONTEXT.md` — D-07/D-08 (default expand + filter decisions) that this phase's D-02/D-06 explicitly mirror.

### Data model / API (Phase 29, already shipped — needs a small extension per D-07)
- `db/routes/regions_get.go` — `RegionsList` handler (lines ~37-117); every record already returns `id/name/kind/parent/path/depth`, leaf rows additionally carry `bbox/enabled/status/version/vector_url/...`. D-07 requires adding `sort_order` to this handler's entry map.
- `db/migrations/1785000000_create_regions_collection.go` — schema source of truth; `sort_order` column already exists (lines 36, 67).

### Flutter code to modify
- `app/lib/routes/settings_offline_regions_screen.dart` (722 lines) — the screen being converted from flat `ListView.separated` to the hierarchical `ListView.builder`. Existing per-row methods (`_buildRegionRow`, `_buildActiveRow`, `_buildVectorTile`, `_buildDemTile`, `_buildVectorTrailing`, `_buildDemTrailing`, `_tileLeadingIcon`, `_tileSubtitle`) stay as-is per D-01, just called from the new flattened-row renderer instead of a flat iteration.
- `app/lib/models/region_catalog_entry.dart:20-34` (Freezed) — currently requires `bbox`/`status`, silently drops group rows; needs the group-node handling from D-05.
- `app/lib/entities/region_entity.dart` — ObjectBox entity for leaf regions; NOT extended for groups per D-03.
- `app/lib/provider/region/region_provider.dart` — `regionRepositoryProvider`, `regionListNotifierProvider` (line ~165); tree-building logic sits on top of this, doesn't replace it.
- `app/lib/util/region_disk_usage_util.dart:45` — `totalRegionDiskUsageBytes(regions)`; keeps operating over the full flat leaf set unchanged, per phase goal (disk-usage summary continues to work unchanged).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `db/routes/regions_ext/regions_ui.html:500-582` — tree-flattening algorithm to port to Dart (see canonical refs).
- Existing per-row Flutter methods in `settings_offline_regions_screen.dart` — download/cancel/delete UI is already decoupled enough (plain methods taking a `RegionEntity`) to be called from nested/indented rows without extraction into new widgets.

### Established Patterns
- No prior tree/`ExpansionTile`/collapsible UI exists anywhere in `app/lib` — this phase establishes the first one, using the ported flatten-array approach (D-01) rather than Flutter's built-in `ExpansionTile`.
- `region_polygons`-style data-splitting precedent from Phase 30 (keep heavy/irrelevant data out of hierarchy reads) doesn't apply here — the app-facing API never returns raw polygons, only bbox for leaves.

### Integration Points
- Tree is built client-side from the existing `GET /api/v1/regions` flat response (now also carrying group rows once D-05's parsing gap is fixed) — no new endpoint needed, matching how Phase 29 already ships the hierarchy fields.
- `regionListNotifierProvider` continues to be the source of leaf `RegionEntity` data; a new layer above it (not a replacement) merges in ephemeral group nodes to build the renderable tree.

</code_context>

<specifics>
## Specific Ideas

No new visual references beyond "match the admin tree's expand/filter/sort behavior, keep the existing leaf row UI untouched." The admin page (Phase 30) is the explicit behavioral reference for D-01, D-02, D-06, D-07.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 31-flutter-settings-hierarchy*
*Context gathered: 2026-07-27*
