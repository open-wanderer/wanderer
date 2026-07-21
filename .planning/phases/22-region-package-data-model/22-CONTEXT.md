# Phase 22: Region & Package Data Model - Context

**Gathered:** 2026-07-21
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers the foundation data model for v1.6's offline region tile repository: a bundled `assets/map/regions.json` manifest, plus ObjectBox `Region` and `DownloadedTilePackage` entities. Nothing downstream (download engine, UI, rendering) reads from these entities yet — the app builds and runs unchanged. Purely additive; zero UI.

</domain>

<decisions>
## Implementation Decisions

### Status Enum Persistence (REGN-02, REGN-03)
- **D-01:** Both `RegionStatus` and any `DownloadedTilePackage` status use Dart enhanced enums with an explicit `code` int field per value, e.g.:
  ```dart
  enum RegionStatus {
    notDownloaded(0),
    downloading(1),
    downloaded(2),
    updateAvailable(3);
    const RegionStatus(this.code);
    final int code;
  }
  ```
- **D-02:** Persist via the codebase's existing shadow-property pattern (`@Transient()` enum field + `int get/set` shadow property), but the getter/setter reads/writes `.code` — **never `.index`**. This is a deliberate deviation from `TrailEntity`/`ActiveNavigationEntity`, which both use `.index` today (flagged by research as the exact anti-pattern REGN-02 forbids). New statuses added later just pick an unused int; no ordering dependency.

### regions.json Initial Content (REGN-01)
- **D-03:** Ship real regions with real bbox coordinates and real vector-PMTiles/DEM URLs — not placeholder/test data. Model region granularity after how OsmAnd and CoMaps split the world into downloadable regions (country/state or sub-country admin-boundary level), not arbitrary custom boxes.
- **D-04:** Include 3-4 regions in this initial manifest.
- **D-05 (flag for research phase):** Exact region boundaries and the concrete URL-sourcing mechanism (how vector-PMTiles/DEM URLs map onto the existing backend pipeline — `db/services/tiles/generator.go` / build.protomaps.com-derived cells + Mapterhorn DEM) are NOT fully locked here. The phase researcher should investigate OsmAnd/CoMaps' region-splitting convention and propose 3-4 concrete regions + real URLs before planning finalizes the manifest content.

### DownloadedTilePackage Shape (REGN-03)
- **D-06:** `Region` has two nullable `ToOne<DownloadedTilePackage>` fields: `vectorPackage` and `demPackage` (unset when no DEM downloaded). No package-type discriminator field, no `ToMany`/`@Backlink` — direct field access (`region.vectorPackage.target?.status`). Rejected alternative: `ToMany<DownloadedTilePackage>` + a `PackageType` enum discriminator (more extensible but adds a filter step to every read; not worth it for exactly two known package types).

### Region vs Package Status Relationship (REGN-02, REGN-03)
- **D-07:** `Region.status` is a **computed getter, not a stored field.** It derives from `vectorPackage.target?.status` (folding in `demPackage` status when a DEM is required/present), defaulting to `RegionStatus.notDownloaded` when no package rows exist yet (the pre-download state). This guarantees Region and package status can never drift out of sync — no dual-write step needed when `TileRepositoryManager` (Phase 23) updates package state.
  - **Note for planner:** REGN-02's literal wording says Region "persists... a live status... and the status survives an app restart" as if it were a stored field. The computed-getter approach still satisfies restart-survival (it reads from persisted `DownloadedTilePackage` rows), but does not add a literal stored column on `Region`. This is an intentional, discussed interpretation — not an oversight. If future phases need to query/filter/sort by Region status directly in ObjectBox (e.g., "list all downloaded regions"), that will require either a stored+synced field or an in-memory filter after fetching all regions; flag this to the user if Phase 23/24 planning finds ObjectBox query-by-computed-field to be a blocker.

### Claude's Discretion
- None — every gray area identified was explicitly decided above.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Roadmap
- `.planning/ROADMAP.md` — Phase 22 section (goal, success criteria, dependencies)
- `.planning/REQUIREMENTS.md` — REGN-01, REGN-02, REGN-03

### Research (already completed — this phase was flagged "skip research-phase" in SUMMARY.md, but D-05 reopens a narrow research need)
- `.planning/research/SUMMARY.md` — Executive summary, Phase 1 rationale, Critical Pitfall 5 (index-backed enum), Gaps to Address
- `.planning/research/ARCHITECTURE.md`
- `.planning/research/PITFALLS.md`
- `.planning/research/STACK.md`
- `.planning/research/FEATURES.md`

### Project Context
- `.planning/PROJECT.md` — v1.6 milestone scope, DEM pipeline description (§ "Offline tiles today are trail-scoped...")

### Existing Backend Tile Pipeline (for D-05 URL sourcing)
- `db/services/tiles/generator.go` — Mapterhorn DEM extraction, `mapterhornSource`, `demMaxZoom = 12`
- `db/routes/map_cells_id.go` — `dem_download_url` / `MapCellsDownloadDem`
- `web/src/routes/api/v1/map/cells/[cellKey]/download-dem/+server.ts` — DEM download proxy

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `app/lib/entities/trail_entity.dart` — dual-id convention (`@Id() int obxId = 0` + separate business `@Index() @Unique(onConflict: ConflictStrategy.replace) String id`), `ToOne`/`@Backlink` relation patterns, manual `fromModel`/`toModel` mapping extension — mirror this structure for `Region`/`DownloadedTilePackage`.
- `app/lib/entities/active_navigation_entity.dart` — closest existing analog for a status-enum entity; shows the shadow-property pattern to adapt (swap `.index` for `.code`).
- `app/lib/models/*.dart` (e.g. `trail.dart`) — `@freezed` + `part '*.freezed.dart'` + `part '*.g.dart'` + `factory fromJson` pattern; use this for the `regions.json` manifest parse model (no existing precedent for hand-parsing bundled JSON — `map_style_json_provider.dart` loads a raw string, not a typed model, so don't follow that example).

### Established Patterns
- **Anti-pattern to avoid:** `TrailEntity.dbDifficulty` / `ActiveNavigationEntity.dbSessionType` both persist via `enum.index` — this is the exact anti-pattern REGN-02 forbids. Do not copy this pattern verbatim; only copy its structural shape (`@Transient()` + shadow int property), swapping `.index` for the new enhanced-enum `.code`.
- ObjectBox Store/box registration is code-gen driven (`objectbox.g.dart`, `objectbox-model.json`) — new `@Entity()` classes under `app/lib/entities/` just need `build_runner` re-run; no manual registry list to update.

### Integration Points
- None yet — this phase is purely additive with no read/write integration. Phase 23 (`TileRepositoryManager`) is the first consumer of these entities.

</code_context>

<specifics>
## Specific Ideas

- Region granularity should be modeled after OsmAnd and CoMaps' approach to splitting the world into downloadable regions — the user explicitly named these two apps as the reference point (not a generic "pick reasonable regions" instruction).

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 22-region-package-data-model*
*Context gathered: 2026-07-21*
