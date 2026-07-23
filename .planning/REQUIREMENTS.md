# Requirements: Wanderer Trail Navigation — v1.6 Offline Region Tile Repository

**Defined:** 2026-07-21
**Core Value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.
**Milestone Goal:** Replace trail-scoped PMTiles downloads with an app-wide, region-based offline tile repository (vector + optional Mapterhorn DEM), managed in Settings, so map rendering and offline trail recording work anywhere within a downloaded region instead of only within a specific trail's cached cells.

## v1 Requirements

### Region Manifest & Data Model

- [x] **REGN-01**: The app fetches its region catalog from this Wanderer instance's backend API at runtime — id, name, bbox, vector archive URL + size, optional DEM archive URL + size, per region. No bundled `regions.json` asset: the catalog reflects only the regions *this* instance's admin has configured (see BACK-01/BACK-04), so a fresh/default instance with no admin config yet returns an empty catalog rather than a fixed global list.
- [x] **REGN-02**: ObjectBox `Region` entity stores fetched-catalog fields plus live status (notDownloaded/downloading/downloaded/updateAvailable), using explicit stable int constants (not index-backed enum persistence)
- [x] **REGN-03**: ObjectBox `DownloadedTilePackage` entity tracks vector and DEM as independent packages per region — local file path, timestamp, size on disk, status

### Backend — Region Catalog & Archive Pre-Build

**Why this exists:** This corrects two assumptions made during Phase 22 planning. First, that the existing per-cell backend (`db/services/tiles/generator.go`, `GET /api/v1/map/cells?bbox=...`, `db/routes/map_cells_id.go`) could be reused as-is by pointing the manifest's `vector_url`/`dem_url` at that endpoint and letting the client fan out into N per-cell requests at download time — wrong shape for a "download this region for offline" feature (forces the client to orchestrate/retry/account many small requests instead of one resumable file). Second, that `regions.json` should be a bundled app asset at all — for a self-hostable app, the set of offline-downloadable regions is an admin decision per instance (a small instance may only want to serve the regions its own trails cover), not something fixed at Flutter build time. The design that replaces both: an admin defines their instance's regions in a config file mounted via Docker volume; a cronjob pre-builds each region's archives ahead of any user request so downloads are instant; the app fetches the resulting catalog from an API endpoint instead of parsing a bundled asset.

- [x] **BACK-01**: Backend loads a region catalog from an admin-supplied config file (mounted via Docker volume) at startup — each entry defines a region's id, name, and bbox; no per-region URL/size in the config, since those are generated, not admin-supplied
- [x] **BACK-02**: A cronjob pre-builds a single mosaicked vector PMTiles archive per configured region — merging the grid cells covering the region's bbox into one file — ahead of any user download request
- [x] **BACK-03**: The same cronjob pre-builds a single DEM archive per configured region on the same basis, reusing the existing Mapterhorn extraction (`generator.go`, `mapterhornSource`, `demMaxZoom = 12`) as its data source but mosaicked to the region's bbox instead of served per grid cell
- [x] **BACK-04**: Backend exposes an API endpoint returning this instance's region catalog (id, name, bbox, vector archive URL + size, DEM archive URL + size, version/status) for the app to fetch at runtime, per REGN-01
- [x] **BACK-05**: Cron regeneration is staleness-aware — it only rebuilds a region's archive when the underlying source tiles have changed since the last build, and that changed-since check is what drives the client-visible `updateAvailable` status (exact cadence/staleness-detection mechanics: open question, revisit at discuss-phase for this phase)

### Tile Repository (Download Engine)

> **Amended 2026-07-23** (post-Phase-24, no new phase): pause/resume was removed from the download engine entirely and replaced with cancel-deletes-and-restarts-from-0. Root cause: Dio's native `deleteOnError` handler deletes the `.part` file on ANY cancellation — including a deliberate pause — not just genuine errors, so the very first pause of a fresh download silently destroyed its own resume progress (see commit `3adeb11c`). Rather than work around that footgun, pause/resume/Range-resume/backgrounding-auto-pause were dropped in favor of the simpler cancel semantics below (commit `4732d20e`). TILE-01/02/04 below reflect the current, amended behavior — do NOT flag pause/resume as missing in a later phase.

- [x] **TILE-01**: `TileRepositoryManager` service owns region download lifecycle (start/cancel/delete), fully decoupled from `Trail` — no pause/resume (amended 2026-07-23, see note above)
- [x] **TILE-02**: ~~Region downloads are resumable within a session via HTTP Range requests + Dio `FileAccessMode.append` (no cross-restart resume)~~ — **superseded 2026-07-23**: downloads are never resumable now, at any level. Cancelling (deliberate or a genuine transfer error) always deletes the `.part` file via `deleteOnError: true`; a later download attempt always restarts from byte 0. See amendment note above.
- [x] **TILE-03**: Disk-space is checked with a safety margin before each file download in a region, before writing begins
- [x] **TILE-04**: ~~App backgrounding mid-download (iOS suspension / Android Doze) is treated as a deliberate pause, not a silent failure~~ — **superseded 2026-07-23**: there is no pause state to enter. The `AppLifecycleListener`-driven auto-pause-on-background was removed; downloads keep running in the background as long as the OS allows the transfer to continue, and if the OS kills it, it simply ends in an `error` state the user can retry. See amendment note above.
- [x] **TILE-05**: A bbox-based query (`localTilePathsForBounds`) returns local vector/DEM file paths covering a given area, for use by map rendering

### DEM Support

- [x] **DEM-01**: Per-region optional DEM toggle downloads the pre-built region-scoped DEM archive produced by BACK-03, served via BACK-04's catalog endpoint — not the existing per-cell `download-dem` endpoint pointed at a region bbox
- [x] **DEM-02**: DEM download/deletion is tracked as its own `DownloadedTilePackage` per region, independent of the vector package's status

### Settings — Offline Maps/Regions UI

> **Amended 2026-07-23** (post-Phase-24, no new phase): the single-row-with-DEM-toggle design was replaced by two independent list tiles per region — Vector and Elevation data — each with its own download/cancel/delete action and progress bar (commit `4732d20e`). The DEM tile's download action is additionally gated on the Vector tile being `downloaded`/`updateAvailable`: hillshading without a basemap underneath it is meaningless, so DEM can no longer be downloaded before Vector (commit `663f049a`). SETUI-03/04 below reflect the current, amended behavior — do NOT flag "no DEM toggle" or "no pause/resume" as gaps in a later phase.

- [x] **SETUI-01**: Settings → Offline Maps/Regions screen shows a flat, searchable region list (no hierarchical tree)
- [x] **SETUI-02**: Each region row shows name, 4-state status, and size breakdown (vector vs DEM) shown before download starts
- [x] **SETUI-03**: ~~Download / pause / resume / delete actions available per region~~ — **superseded 2026-07-23**: Download / cancel / delete actions available per region, independently for Vector and DEM (no pause/resume — see TILE-01/02/04 amendment). Cancelling always deletes progress; a later download restarts from byte 0.
- [x] **SETUI-04**: ~~Per-region DEM toggle control, clearly presented as the optional/adds-size choice~~ — **superseded 2026-07-23**: DEM is its own list tile (not a toggle), with its own download/cancel/delete action, disabled with an explanatory subtitle ("Download map data first") until the Vector tile is downloaded. Deleting Vector still cascades to delete DEM.
- [x] **SETUI-05**: Total disk usage summary shown on the region list screen
- [x] **SETUI-06**: `updateAvailable` regions show a non-blocking badge/label with an optional user-triggered update action — region continues to render/route normally while the badge is shown

### Trail Download Guard

- [ ] **GUARD-01**: On trail download tap, the app checks the trail's bbox coverage against downloaded (or updateAvailable) regions before proceeding
- [ ] **GUARD-02**: If coverage is missing, a dialog names the specific missing region(s) and their size, with a direct in-dialog "Download region" CTA per region
- [ ] **GUARD-03**: Partial-coverage handling: a trail spanning multiple regions lists all missing regions with individual + combined size, lets the user download any subset, and does not force full coverage before allowing the trail download to proceed
- [ ] **GUARD-04**: `updateAvailable` regions satisfy the coverage check identically to `downloaded` — the guard never re-fires for a region that's merely stale

### Map Rendering Integration

- [x] **RENDER-01**: `TrailMap` and `navigation_screen` read offline tiles from the region registry via `TileRepositoryManager`, replacing trail-bound cache reads
- [ ] **RENDER-02**: Style composition is viewport-scoped — only regions intersecting the current viewport contribute style sources, not every downloaded region unconditionally
- [x] **RENDER-03**: Before finalizing the rendering approach, verify maplibre 0.3.5's incremental source add/remove behavior (vs. full style reload) and layer-count scaling with a spike against the pinned package version

### Legacy Cleanup

- [ ] **CLEAN-01**: Trail-scoped tile download code is removed outright — `trail_download_service.dart` tile-download methods, `TrailEntity.pmTiles`/`demPmTiles` fields, and related UI — no dual-run, no migration path (app is pre-production)
- [ ] **CLEAN-02**: A one-time on-device cleanup sweep deletes orphaned legacy tile files left on existing dev/test installs, so the new disk-usage figure is accurate

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Region Management Enhancements

- **REGN-F01**: Map boundary highlight overlay showing downloaded region coverage directly on the map
- **REGN-F02**: Bulk download/delete actions (add once the manifest grows past ~5+ regions)
- **REGN-F03**: Auto-download the region containing the user's current GPS location on first launch
- ~~**REGN-F04**: Remote/updatable region manifest~~ — superseded, pulled into v1 as REGN-01/BACK-01/BACK-04 (2026-07-21)
- **REGN-F05**: User-drawn custom download areas
- **REGN-F06**: Offline search within downloaded regions

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Legacy trail-cache migration | App is pre-production; old trail-scoped tile/DEM cache is deleted outright, no conversion path |
| Admin UI/API for region catalog CRUD | v1.6 region definition is a config file mounted via Docker volume, edited by redeploying; a settings/admin screen for live editing is deferred |
| Polygon region geometries | v1.6 regions are bounding-box only; arbitrary polygon boundaries add geometry-processing complexity with no functional download benefit |
| 3D terrain/hillshade rendering redesign | v1.6 only relocates the existing DEM download/storage pipeline to be region-based; `offline_style_rewriter.dart`'s hillshade rendering is reused as-is |
| Resumable downloads, of any kind | Originally session-scoped pause/resume only (cross-restart resume excluded as a documented OsmAnd bug source); amended 2026-07-23 to drop resume entirely — Dio's `deleteOnError` treats a deliberate pause identically to a genuine transfer error and deletes the `.part` file either way (see TILE-01/02/04 amendment note), so cancel-and-restart-from-0 replaced pause/resume outright |
| Hierarchical region tree navigation | Manifest is tens of entries, not thousands — a flat searchable list is the right complexity |
| Granular per-layer toggles beyond vector/DEM | No reviewed hiking app exposes finer-grained toggles (roads/POIs/water) at the region-download level |
| Map boundary highlight overlay | Nice differentiator, deferred to v1.x (REGN-F01) |
| Region entitlement/paywall model | No paywall exists in Wanderer; guard dialog borrows Komoot's messaging pattern only, not its unlock/purchase logic |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| REGN-01 | Phase 22 (replan needed — see todo) | Complete |
| REGN-02 | Phase 22 | Complete |
| REGN-03 | Phase 22 | Complete |
| BACK-01 | Phase 21.5 | Complete |
| BACK-02 | Phase 21.5 | Complete |
| BACK-03 | Phase 21.5 | Complete |
| BACK-04 | Phase 21.5 | Complete |
| BACK-05 | Phase 21.5 | Complete |
| TILE-01 | Phase 23 | Complete |
| TILE-02 | Phase 23 | Complete |
| TILE-03 | Phase 23 | Complete |
| TILE-04 | Phase 23 | Complete |
| TILE-05 | Phase 23 | Complete |
| DEM-01 | Phase 23 | Complete |
| DEM-02 | Phase 23 | Complete |
| SETUI-01 | Phase 24 | Complete |
| SETUI-02 | Phase 24 | Complete |
| SETUI-03 | Phase 24 | Complete |
| SETUI-04 | Phase 24 | Complete |
| SETUI-05 | Phase 24 | Complete |
| SETUI-06 | Phase 24 | Complete |
| GUARD-01 | Phase 26 | Pending |
| GUARD-02 | Phase 26 | Pending |
| GUARD-03 | Phase 26 | Pending |
| GUARD-04 | Phase 26 | Pending |
| RENDER-01 | Phase 25 | Complete |
| RENDER-02 | Phase 25 | Pending |
| RENDER-03 | Phase 25 | Complete |
| CLEAN-01 | Phase 27 | Pending |
| CLEAN-02 | Phase 27 | Pending |

**Coverage:**

- v1 requirements: 30 total
- Mapped to phases: 30
- Unmapped: 0 ✓ (REGN-01 maps to Phase 22, but that phase's existing plans need replanning — see `.planning/todos/pending/replan-phase-22-region-manifest.md`)

---
*Requirements defined: 2026-07-21*
*Last updated: 2026-07-21 — reworked region catalog design from "bundled app asset" to "per-instance API, admin-configured via a Docker-mounted config file, pre-built by cronjob" (REGN-01 rewritten; BACK-01..05 replace the earlier on-demand-generation framing; REGN-F04 superseded and pulled into v1). See `.planning/notes/region-catalog-backend-decision-trail.md` for the full rationale. New requirements mapped to Phase 21.5 in ROADMAP.md.*
