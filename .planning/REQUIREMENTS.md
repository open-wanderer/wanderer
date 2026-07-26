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
- [ ] **EXTRACT-03**: `GET /api/v1/regions` includes hierarchy fields (`parent`, `path`, `depth`) alongside existing bbox/status/size fields, so the client can render a tree

### Admin Region Picker UI

- [ ] **ADMINUI-01**: A custom PocketBase admin page (AlpineJS bundle, `feature/ap-instance-actors` pattern) renders the region catalog as a collapsible tree
- [ ] **ADMINUI-02**: The admin toggles a leaf region's `enabled` flag directly from the tree; takes effect on the cron's next run — no other admin action required
- [ ] **ADMINUI-03**: A live map on the same page renders boundary polygons of all currently-enabled leaf regions, so coverage is visible before committing

### Flutter Settings Hierarchy

- [ ] **APPUI-01**: Settings → Offline Maps/Regions presents downloadable regions as a collapsible hierarchy matching the admin-defined tree, instead of a flat list
- [ ] **APPUI-02**: Existing per-region download/cancel/delete actions and disk-usage summary continue working unchanged within the new hierarchical presentation — no download-UX regression

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
| EXTRACT-03 | Phase 29 | Pending |
| ADMINUI-01 | Phase 30 | Pending |
| ADMINUI-02 | Phase 30 | Pending |
| ADMINUI-03 | Phase 30 | Pending |
| APPUI-01 | Phase 31 | Pending |
| APPUI-02 | Phase 31 | Pending |

**Coverage:**

- v1 requirements: 13 total
- Mapped to phases: 13 (Phase 28: Region Catalog Data Model & Seeding, Phase 29: Polygon-Based Extraction & Region API, Phase 30: Admin Region Picker UI, Phase 31: Flutter Settings Hierarchy)
- Unmapped: 0 ✓

---
*Requirements defined: 2026-07-24*
*Last updated: 2026-07-24 after roadmap creation (Phases 28-31 assigned)*
