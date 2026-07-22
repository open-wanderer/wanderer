# Phase 24: Settings — Offline Maps/Regions UI - Context

**Gathered:** 2026-07-22
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers a single new Settings screen — "Offline Maps/Regions" — where a user discovers, downloads, manages, and monitors offline regions end-to-end. It is the first consumer of Phase 22's `RegionRepository`/`RegionEntity` data model and Phase 23's `TileRepositoryManager`/`TileRepositoryStatus` download engine; nothing about the map rendering pipeline (Phase 25) or trail download guard (Phase 26) is touched here. In scope: flat searchable region list, per-row 4-state-framed status display (in practice 6 `RegionStatus` values), download/pause/resume/delete actions, an independent per-region DEM toggle, a total disk-usage summary, and the `updateAvailable` badge. One net-new capability surfaces here that Phase 23 didn't build: a DEM-only delete path on `TileRepositoryManager` (see D-01 below) — this is additive to the existing engine, not new scope creep, since SETUI-04 already required an independent DEM control and the engine was one method short of supporting it cleanly.

</domain>

<decisions>
## Implementation Decisions

### DEM Toggle, Delete & Error Handling
- **D-01:** Toggling a region's DEM switch OFF after the DEM package is downloaded deletes the DEM file **immediately** — no confirmation dialog on toggle-off, mirroring toggle-on's immediate `downloadDem()` call. This requires a **new `TileRepositoryManager` method** (e.g. `deleteDemPackage(regionId)`) that removes only the DEM `DownloadedTilePackageEntity` row and on-disk file, clearing `region.demPackage.target`, and explicitly leaves `region.vectorPackage` and its file untouched. `deleteRegion()` (Phase 23) intentionally deletes both packages together and is NOT reused for this — it's the wrong granularity.
- **D-02:** Deleting an entire downloaded region (the row-level delete action, not the DEM toggle) DOES require a confirmation dialog before calling `deleteRegion()` — matches the existing own-trail-disable-confirm pattern in `settings_categories_screen.dart`. This is deliberately asymmetric with D-01: full-region delete is a bigger, harder-to-reverse action (re-downloading a multi-hundred-MB archive vs. a DEM toggle) and warrants the extra guard; the DEM toggle does not.
- **D-03:** A region row with `RegionStatus.error` shows an inline error badge/indicator (distinct from the normal downloading/downloaded look) plus a tappable **Retry** action that re-invokes `downloadVector()`/`downloadDem()` from scratch for whichever package failed. No separate "view error detail" step — retry is the primary and only action surfaced.

### Status Display Mapping
- **D-04:** `RegionStatus` has 6 values (`notDownloaded`/`downloading`/`downloaded`/`updateAvailable`/`paused`/`error`) even though the roadmap's success criteria describe a "4-state status." Resolution: show **6 distinct visual states**, each with its own icon/label/action (paused shows a resume icon + dimmed progress; error shows D-03's badge+retry). Do not fold paused/error into the "downloading" look — treat the roadmap's "4-state" phrasing as describing the baseline lifecycle, not a hard visual cap.
- **D-05:** The `updateAvailable` badge (per Phase 22's D-12: region keeps behaving as fully "downloaded" while the badge shows) renders as **persistent banner text under the region name** (e.g. "Update available") plus an inline update button — always visible, not a small tap-to-discover badge/chip. This is more prominent than a bare badge by design, since staleness is otherwise invisible until the user notices stale map data.

### Disk Usage & Progress Detail
- **D-06:** The total disk-usage summary at the top of the screen sums `DownloadedTilePackageEntity.sizeBytesOnDisk` across **every package regardless of status** — downloaded, paused (partial `.part` file), and downloading (partial bytes written so far) — because partial files genuinely occupy disk space and the summary should reflect real usage, not just "usable offline data."
- **D-07:** When vector and DEM download concurrently for the same region, the row shows **one combined progress bar**, not two separate bars — combine/average `vectorProgress` and `demProgress` from `RegionDownloadState` into a single visual indicator per row. Simpler row layout was prioritized over per-package progress transparency.

### List Content & Sorting
- **D-08:** The search box filters on **region name only** (`RegionEntity.name` substring match) — not the internal catalog `id` (an internal slug like `de-nrw`, never shown in the UI, so searching it would offer no discoverable benefit).
- **D-09:** Default sort order is **alphabetical (A-Z) by name**, always — no downloaded-first or status-based reordering. A region whose `catalogStatus` is `building` or `error` (backend hasn't produced a downloadable archive yet, distinct from a *local* download error per D-03) is still **shown in the list, disabled**, with a label ("Not yet available" for `building`, "Build failed" for `error`) and no download action — not hidden from the list entirely.

### Claude's Discretion
- None — every gray area identified in this discussion was explicitly decided above.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Roadmap
- `.planning/ROADMAP.md` — Phase 24 section (goal, 6 success criteria, "Depends on: Phase 23")
- `.planning/REQUIREMENTS.md` — SETUI-01 through SETUI-06 (all currently `Pending`)

### Prior Phase Context (data model + download engine this phase consumes)
- `.planning/phases/22-region-package-data-model/22-CONTEXT.md` — D-01 through D-12: catalog upsert strategy, `catalogStatus` vs local `status` split, `updateAvailable` staleness mechanism (D-06/D-12), `inCatalog` flag, `ToOne` package relations. This phase's status/badge decisions (D-04/D-05 above) build directly on D-12's design.
- `.planning/phases/23-tilerepositorymanager-download-engine/23-RESEARCH.md` — resumable download, pmtiles validation, app-lifecycle pause patterns (background for understanding what the engine already handles)
- `.planning/phases/23-tilerepositorymanager-download-engine/23-PATTERNS.md` — file-by-file analog mapping for the download engine; useful precedent for how this phase's new UI files should be mapped

### Data Model (read before touching status/size fields)
- `app/lib/entities/region_entity.dart` — `RegionEntity.status` computed getter (D-12 from Phase 22), `catalogStatus`/`demStatus` enum shadows, `vectorPackage`/`demPackage` `ToOne` fields
- `app/lib/entities/downloaded_tile_package_entity.dart` — `sizeBytesOnDisk`, `status`, `localFilePath` — the fields D-06's disk-usage sum reads
- `app/lib/models/region_status.dart` — `CatalogStatus` (building/ready/error/absent), `RegionStatus` (6 values, D-04 above), `PackageStatus` (5 values) — explicit `.code` int enums, never `.index`

### Download Engine (this phase's primary integration point)
- `app/lib/services/tile_repository_manager.dart` — `startVectorDownload`, `startDemDownload`, `pauseRegion`, `resumeRegion`, `deleteRegion` (lines ~324-366, deletes vector+DEM together — D-01 requires a new sibling method, not reuse of this one)
- `app/lib/provider/region/tile_repository_provider.dart` — `TileRepositoryStatus` Riverpod notifier (`downloadVector`, `downloadDem`, `pause`, `resume`, `delete`) — this phase's screen subscribes to this for per-region ephemeral progress state; D-01's new DEM-only delete needs a corresponding notifier method here too
- `app/lib/provider/region/region_provider.dart` — `RegionRepository`, `fetchRegionCatalog`, `RegionCatalogException` — catalog refresh entry point; per Phase 22's D-02, this phase decides when `refreshCatalog()` is invoked (current intent: on-demand when this screen opens)
- `app/lib/models/region_download_state.dart` — `RegionDownloadState` (`status`, `vectorProgress`, `demProgress`) — the ephemeral UI state D-07's combined progress bar reads

### Existing Settings Screen Patterns (structural precedent for the new screen)
- `app/lib/routes/settings_categories_screen.dart` — closest analog: `ConsumerStatefulWidget` list screen with per-row toggle (auto-save, error-toast-only per its D-08 note), confirm-before-disable dialog for a destructive action (the pattern D-02 above explicitly reuses for region delete), drag-reorder (not needed here, but the dialog/toast conventions are)
- `app/lib/routes/settings_subcategories_screen.dart` — secondary analog for a filterable/searchable list within Settings
- `app/lib/routes/settings_screen.dart` — the parent Settings list this new screen's entry point is added to

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `app/lib/routes/settings_categories_screen.dart`'s confirm-before-disable dialog pattern and error-toast-only save wrapper (`_save` helper) — reuse directly for D-02's region-delete confirmation and for surfacing D-01/D-03 failures.
- `app/lib/provider/toast_provider.dart` — used by `settings_categories_screen.dart` for error-only toasts; same convention applies to DEM-delete/retry failures in this phase.
- `TileRepositoryStatus` (Riverpod `keepAlive` notifier) already exists and already exposes `downloadVector`/`downloadDem`/`pause`/`resume`/`delete` with per-region progress tracking — this phase's screen is primarily a consumer, not a from-scratch state-management build.

### Established Patterns
- Explicit `.code` int enum persistence (never `.index`) — already established by Phase 22/23; this phase adds no new persisted enums, but D-01's new engine method must follow the same status-transition discipline `pauseRegion`/`deleteRegion` already use (batched `runInTransaction` writes).
- Per-region `CancelToken` map keyed by `'$id:vector'`/`'$id:dem'` in `TileRepositoryManager` — D-01's new `deleteDemPackage` method must cancel only the `'$id:dem'` token, not `'$id:vector'`, mirroring `deleteRegion`'s existing selective-cancel logic (lines 327-331).

### Integration Points
- This phase adds a new route/screen entry from `settings_screen.dart` (go_router), following the same navigation pattern as `settings_categories_screen.dart`.
- `RegionRepository.refreshCatalog()` (Phase 22) needs a call site for the first time — this phase decides exactly when (on screen open, per Phase 22's D-02 stated intent).
- `TileRepositoryManager` needs one new method (D-01) plus a corresponding `TileRepositoryStatus` notifier method — the only engine-side change this phase makes; everything else in Phase 23's engine is consumed as-is.

</code_context>

<specifics>
## Specific Ideas

- Error rows: badge + Retry only, no secondary "view detail" step (D-03) — keep it to one tap.
- Update-available treatment should be more prominent than a bare badge — persistent banner text, not a small chip (D-05).
- Building/error catalog-status regions stay visible in the list as disabled rows with a specific label per state, never hidden (D-09).

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed entirely within phase scope. No new capabilities were proposed; all four discussed areas were implementation-mechanics questions about the already-scoped Settings screen.

</deferred>

---

*Phase: 24-settings-offline-maps-regions-ui*
*Context gathered: 2026-07-22*
