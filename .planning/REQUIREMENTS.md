# Requirements: Wanderer Trail Navigation — v1.6 Offline Region Tile Repository

**Defined:** 2026-07-21
**Core Value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.
**Milestone Goal:** Replace trail-scoped PMTiles downloads with an app-wide, region-based offline tile repository (vector + optional Mapterhorn DEM), managed in Settings, so map rendering and offline trail recording work anywhere within a downloaded region instead of only within a specific trail's cached cells.

## v1 Requirements

### Region Manifest & Data Model

- [ ] **REGN-01**: A bundled `regions.json` app asset defines regions (id, name, bbox, vector PMTiles URL + size, optional DEM URL + size)
- [ ] **REGN-02**: ObjectBox `Region` entity stores manifest fields plus live status (notDownloaded/downloading/downloaded/updateAvailable), using explicit stable int constants (not index-backed enum persistence)
- [ ] **REGN-03**: ObjectBox `DownloadedTilePackage` entity tracks vector and DEM as independent packages per region — local file path, timestamp, size on disk, status

### Tile Repository (Download Engine)

- [ ] **TILE-01**: `TileRepositoryManager` service owns region download lifecycle (start/pause/resume/delete), fully decoupled from `Trail`
- [ ] **TILE-02**: Region downloads are resumable within a session via HTTP Range requests + Dio `FileAccessMode.append` (no cross-restart resume)
- [ ] **TILE-03**: Disk-space is checked with a safety margin before each file download in a region, before writing begins
- [ ] **TILE-04**: App backgrounding mid-download (iOS suspension / Android Doze) is treated as a deliberate pause, not a silent failure
- [ ] **TILE-05**: A bbox-based query (`localTilePathsForBounds`) returns local vector/DEM file paths covering a given area, for use by map rendering

### DEM Support

- [ ] **DEM-01**: Per-region optional DEM toggle reuses the existing Mapterhorn DEM pipeline (`generator.go` / download-dem endpoint), re-keyed to regions instead of trail cells
- [ ] **DEM-02**: DEM download/deletion is tracked as its own `DownloadedTilePackage` per region, independent of the vector package's status

### Settings — Offline Maps/Regions UI

- [ ] **SETUI-01**: Settings → Offline Maps/Regions screen shows a flat, searchable region list (no hierarchical tree)
- [ ] **SETUI-02**: Each region row shows name, 4-state status, and size breakdown (vector vs DEM) shown before download starts
- [ ] **SETUI-03**: Download / pause / resume / delete actions available per region
- [ ] **SETUI-04**: Per-region DEM toggle control, clearly presented as the optional/adds-size choice
- [ ] **SETUI-05**: Total disk usage summary shown on the region list screen
- [ ] **SETUI-06**: `updateAvailable` regions show a non-blocking badge/label with an optional user-triggered update action — region continues to render/route normally while the badge is shown

### Trail Download Guard

- [ ] **GUARD-01**: On trail download tap, the app checks the trail's bbox coverage against downloaded (or updateAvailable) regions before proceeding
- [ ] **GUARD-02**: If coverage is missing, a dialog names the specific missing region(s) and their size, with a direct in-dialog "Download region" CTA per region
- [ ] **GUARD-03**: Partial-coverage handling: a trail spanning multiple regions lists all missing regions with individual + combined size, lets the user download any subset, and does not force full coverage before allowing the trail download to proceed
- [ ] **GUARD-04**: `updateAvailable` regions satisfy the coverage check identically to `downloaded` — the guard never re-fires for a region that's merely stale

### Map Rendering Integration

- [ ] **RENDER-01**: `TrailMap` and `navigation_screen` read offline tiles from the region registry via `TileRepositoryManager`, replacing trail-bound cache reads
- [ ] **RENDER-02**: Style composition is viewport-scoped — only regions intersecting the current viewport contribute style sources, not every downloaded region unconditionally
- [ ] **RENDER-03**: Before finalizing the rendering approach, verify maplibre 0.3.5's incremental source add/remove behavior (vs. full style reload) and layer-count scaling with a spike against the pinned package version

### Legacy Cleanup

- [ ] **CLEAN-01**: Trail-scoped tile download code is removed outright — `trail_download_service.dart` tile-download methods, `TrailEntity.pmTiles`/`demPmTiles` fields, and related UI — no dual-run, no migration path (app is pre-production)
- [ ] **CLEAN-02**: A one-time on-device cleanup sweep deletes orphaned legacy tile files left on existing dev/test installs, so the new disk-usage figure is accurate

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Region Management Enhancements

- **REGN-F01**: Map boundary highlight overlay showing downloaded region coverage directly on the map
- **REGN-F02**: Bulk download/delete actions (add once the manifest grows past ~5+ regions)
- **REGN-F03**: Auto-download the region containing the user's current GPS location on first launch
- **REGN-F04**: Remote/updatable region manifest (revisit only if the bundled-asset manifest proves stale in practice)
- **REGN-F05**: User-drawn custom download areas
- **REGN-F06**: Offline search within downloaded regions

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Legacy trail-cache migration | App is pre-production; old trail-scoped tile/DEM cache is deleted outright, no conversion path |
| Remote/server-fetched region manifest | v1.6 ships a bundled `regions.json` app asset only |
| Polygon region geometries | v1.6 regions are bounding-box only; arbitrary polygon boundaries add geometry-processing complexity with no functional download benefit |
| 3D terrain/hillshade rendering redesign | v1.6 only relocates the existing DEM download/storage pipeline to be region-based; `offline_style_rewriter.dart`'s hillshade rendering is reused as-is |
| Background/resumable downloads across app restarts | Session-scoped pause/resume only — cross-restart resume is a documented source of bugs even in mature apps (OsmAnd) |
| Hierarchical region tree navigation | Manifest is tens of entries, not thousands — a flat searchable list is the right complexity |
| Granular per-layer toggles beyond vector/DEM | No reviewed hiking app exposes finer-grained toggles (roads/POIs/water) at the region-download level |
| Map boundary highlight overlay | Nice differentiator, deferred to v1.x (REGN-F01) |
| Region entitlement/paywall model | No paywall exists in Wanderer; guard dialog borrows Komoot's messaging pattern only, not its unlock/purchase logic |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| REGN-01 | Phase 22 | Pending |
| REGN-02 | Phase 22 | Pending |
| REGN-03 | Phase 22 | Pending |
| TILE-01 | Phase 23 | Pending |
| TILE-02 | Phase 23 | Pending |
| TILE-03 | Phase 23 | Pending |
| TILE-04 | Phase 23 | Pending |
| TILE-05 | Phase 23 | Pending |
| DEM-01 | Phase 23 | Pending |
| DEM-02 | Phase 23 | Pending |
| SETUI-01 | Phase 24 | Pending |
| SETUI-02 | Phase 24 | Pending |
| SETUI-03 | Phase 24 | Pending |
| SETUI-04 | Phase 24 | Pending |
| SETUI-05 | Phase 24 | Pending |
| SETUI-06 | Phase 24 | Pending |
| GUARD-01 | Phase 26 | Pending |
| GUARD-02 | Phase 26 | Pending |
| GUARD-03 | Phase 26 | Pending |
| GUARD-04 | Phase 26 | Pending |
| RENDER-01 | Phase 25 | Pending |
| RENDER-02 | Phase 25 | Pending |
| RENDER-03 | Phase 25 | Pending |
| CLEAN-01 | Phase 27 | Pending |
| CLEAN-02 | Phase 27 | Pending |

**Coverage:**
- v1 requirements: 25 total
- Mapped to phases: 25
- Unmapped: 0 ✓

---
*Requirements defined: 2026-07-21*
*Last updated: 2026-07-21 after roadmap creation (24-item count in this file corrected to 25 — the Coverage summary had undercounted the requirement list by one)*
