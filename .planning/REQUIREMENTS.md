# Requirements: Wanderer Trail Navigation — v1.7 Admin Region Picker

**Defined:** 2026-07-24
**Core Value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.
**Milestone Goal:** A server owner defines downloadable regions by toggling entries in a curated, seeded catalog — sourced from CoMaps' extract hierarchy, with real known boundaries — instead of hand-authoring `region_config.json`; the app's settings screen presents the same hierarchy.

## v1 Requirements

### Region Catalog Data Model

- [x] **CATALOG-01**: Backend `regions` PocketBase collection stores each CoMaps hierarchy node with `comaps_id`, self-referencing `parent`, materialized `path`, `depth`, `sort_order`, `name`, and `kind` (`group`|`leaf`)
- [x] **CATALOG-02**: Leaf rows additionally store a canonical `polygon` (GeoJSON, converted from CoMaps `.poly`) and a derived `bbox`
- [x] **CATALOG-03**: Leaf rows carry an `enabled` boolean (default false); group rows carry no `enabled`/`polygon`/`bbox` semantics

### Region Catalog Seeding

- [x] **SEED-01**: A maintainer-run `db/commands/seed_regions.go` command parses vendored CoMaps `hierarchy.txt` + `.poly` files and writes a flattened JSON seed file matching the `regions` schema
- [x] **SEED-02**: A standard PocketBase migration creates the `regions` collection and bulk-inserts from the committed JSON seed automatically on every instance startup — a fresh self-hosted instance ships with a populated, toggleable catalog, zero admin action required

### Polygon-Based Extraction

- [x] **EXTRACT-01**: The archive-generation cron extracts each enabled leaf region via `pmtiles extract --region <polygon>` using its canonical polygon, replacing bbox-based extraction
- [x] **EXTRACT-02**: The cron reads `kind = 'leaf' AND enabled = true` from the `regions` table to determine build targets; `region_config.json` parsing is retired entirely
- [x] **EXTRACT-03**: `GET /api/v1/regions` includes hierarchy fields (`parent`, `path`, `depth`) alongside existing bbox/status/size fields, so the client can render a tree

### Admin Region Picker UI

- [x] **ADMINUI-01**: A custom PocketBase admin page (AlpineJS bundle, `feature/ap-instance-actors` pattern) renders the region catalog as a collapsible tree
- [x] **ADMINUI-02**: The admin toggles a leaf region's `enabled` flag directly from the tree; takes effect on the cron's next run — no other admin action required
- [x] **ADMINUI-03**: A live map on the same page renders boundary polygons of all currently-enabled leaf regions, so coverage is visible before committing

### Flutter Settings Hierarchy

- [x] **APPUI-01**: Settings → Offline Maps/Regions presents downloadable regions as a collapsible hierarchy matching the admin-defined tree, instead of a flat list
- [x] **APPUI-02**: Existing per-region download/cancel/delete actions and disk-usage summary continue working unchanged within the new hierarchical presentation — no download-UX regression

### Seed Slimming & On-Demand Geometry

Added 2026-07-28 after `/gsd-explore`. Revises the Phase 28 seeding approach: boundary geometry stops being distributed with the repo and is fetched at archive-build time instead. Supersedes CATALOG-02's stored `polygon` (its `bbox` half is retained) and changes the content of SEED-01/SEED-02.

- [ ] **SLIM-01**: `seed-regions` writes a geometry-free catalog — hierarchy fields plus leaf `bbox` only, under 100 KB — with the CoMaps commit SHA it fetched from recorded inside the artifact, so the fetcher cannot desync from the hierarchy
- [ ] **SLIM-02**: The migration creates `regions` only; the `region_polygons` collection is never created on fresh instances and is dropped on existing ones, eliminating the bulk geometry insert from first-boot startup
- [ ] **SLIM-03**: `buildRegion` fetches the target leaf's `.poly` on demand at the catalog's recorded commit and converts it via the existing `ParsePoly`, producing archives equivalent to today's — GitHub mirror primary, CoMaps' canonical Codeberg repository as fallback, with failures naming which upstreams were tried
- [ ] **SLIM-04**: A fresh self-hosted instance boots, migrates, and serves the full catalog through `GET /api/v1/regions` and the admin picker with no network access — only archive building, which is already network-gated, requires connectivity

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Region Catalog Enhancements

- **CATALOG-F01**: Automated refresh of the seeded catalog from CoMaps (currently a manual `seed_regions.go` re-run + reviewed diff + migration) — see seed `region-list-refresh-mechanism`
- **CATALOG-F02**: Group-level/cascading enable (toggle a whole country, enabling all its leaf sub-regions at once)
- **CATALOG-F03**: Map preview boundary for collapsed group nodes (currently leaf-only boundaries are rendered)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Freehand polygon drawing by admin | Rejected during design (`streamlined-region-definition.md`) — a curated catalog removes the research/bbox-arithmetic burden that freehand drawing reintroduces |
| Group-level/cascading enable | v1 toggles individual leaf regions only; tracked as CATALOG-F02 |
| Map preview for collapsed group nodes | Only individual enabled leaf boundaries are rendered; group nodes are tree-navigation only, no geometry |
| Automated CoMaps catalog refresh | Deferred — manual refresh procedure documented in seed `region-list-refresh-mechanism`; tracked as CATALOG-F01 |
| Client-side polygon precision | App-facing catalog and coverage math (trail-download guard) stay bbox-based for efficiency; only server-side extraction uses the canonical polygon |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CATALOG-01 | Phase 28 | Complete |
| CATALOG-02 | Phase 28 | Complete |
| CATALOG-03 | Phase 28 | Complete |
| SEED-01 | Phase 28 | Complete |
| SEED-02 | Phase 28 | Complete |
| EXTRACT-01 | Phase 29 | Complete |
| EXTRACT-02 | Phase 29 | Complete |
| EXTRACT-03 | Phase 29 | Complete |
| ADMINUI-01 | Phase 30 | Complete |
| ADMINUI-02 | Phase 30 | Complete |
| ADMINUI-03 | Phase 30 | Complete |
| APPUI-01 | Phase 31 | Complete |
| APPUI-02 | Phase 31 | Complete |
| SLIM-01 | Phase 32 | Not started |
| SLIM-02 | Phase 32 | Not started |
| SLIM-03 | Phase 32 | Not started |
| SLIM-04 | Phase 32 | Not started |

**Coverage:**

- v1 requirements: 17 total
- Mapped to phases: 17 (Phase 28: Region Catalog Data Model & Seeding, Phase 29: Polygon-Based Extraction & Region API, Phase 30: Admin Region Picker UI, Phase 31: Flutter Settings Hierarchy, Phase 32: On-Demand Polygon Fetch & Seed Slimming)
- Unmapped: 0 ✓

---
*Requirements defined: 2026-07-24*
*Last updated: 2026-07-28 — SLIM-01..04 added after `/gsd-explore` (Phase 32 assigned)*
