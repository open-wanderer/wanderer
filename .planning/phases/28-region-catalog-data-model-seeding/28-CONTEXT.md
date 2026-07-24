# Phase 28: Region Catalog Data Model & Seeding - Context

**Gathered:** 2026-07-24
**Status:** Ready for planning

<domain>
## Phase Boundary

A fresh, self-hosted Wanderer instance boots with a fully populated, hierarchical, toggleable region catalog — sourced from CoMaps' extract hierarchy — with zero admin action required. This phase delivers:

- The `regions` PocketBase collection schema (`comaps_id`, self-referencing `parent`, materialized `path`, `depth`, `sort_order`, `name`, `kind`, and leaf-only `polygon`/`bbox`/`enabled`).
- `db/commands/seed_regions.go` — a maintainer-run Cobra command that fetches CoMaps' `hierarchy.txt` + `.poly` files, converts them, and writes a flattened JSON seed file.
- A standard auto-run PocketBase migration that creates the `regions` collection and bulk-inserts from the committed JSON seed on every instance startup.

Out of scope for this phase: the archive-generation cron switch, the region API's hierarchy fields, and the admin/Flutter UIs (Phases 29-31). `region_config.json` and the existing `db/services/regions/` package are untouched here.

</domain>

<decisions>
## Implementation Decisions

### CoMaps Snapshot Sourcing
- **D-01:** `seed_regions.go` fetches `hierarchy.txt` + `.poly` files fresh from Codeberg (`comaps/comaps`) at tool-run time — nothing raw is vendored/committed to this repo. Only the tool's flattened JSON output is committed.
- **D-02:** The commit hash to fetch is a CLI flag (e.g. `--commit=<hash>`) with a sensible default baked in, so a maintainer can override ad hoc without editing source. Refresh flow: bump the default (or pass `--commit`), re-run, review the JSON diff, commit, ship as a normal migration.

### .poly → GeoJSON Parsing
- **D-03:** Hand-roll a minimal Go parser for CoMaps' Osmosis-format `.poly` files (~50 lines: name header, ring blocks terminated by `END`, `!`-prefixed inner rings) rather than pulling in a new dependency for this one-off maintainer tool.
- **D-04:** The parser must support multi-ring geometry from day one — both holes (`!`-prefixed inner rings, e.g. lakes) and multi-part regions (e.g. a country with islands/exclaves). Output GeoJSON `Polygon` for single-ring regions, `MultiPolygon` when a region has multiple outer rings. Skipping this would silently produce wrong/simplified boundaries for coastal or multi-part regions with no easy way to notice.

### Claude's Discretion
- Exact seed JSON file name/location under `db/migrations/initial_data/` (or similar) — follow existing migration conventions (see `db/migrations/1780000005_add_other_category.go` for the `initial_data`-relative-file pattern).
- Internal structure of the fetch step (HTTP client choice, caching of fetched files during a single tool run, error handling for network failures) — this is a maintainer-run dev-time tool, not production code, so keep it simple.
- `sort_order` derivation — preserve CoMaps' own sibling ordering as encountered in `hierarchy.txt`.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Design decisions (locked)
- `.planning/notes/streamlined-region-definition.md` — the full design trail: table schema, group/leaf semantics, seeding tool split (Cobra command + migration), licensing/attribution treatment (ODbL, same as existing OSM-derived tile data), and why a curated catalog was chosen over freehand drawing. This is the primary source of truth for Phase 28 — read it in full.
- `.planning/notes/region-catalog-backend-decision-trail.md` (referenced by the above; check if present) — original decision to defer region CRUD to `region_config.json`, which this phase supersedes.

### Requirements
- `.planning/REQUIREMENTS.md` — CATALOG-01, CATALOG-02, CATALOG-03, SEED-01, SEED-02 map to this phase.
- `.planning/ROADMAP.md` (Phase 28 section) — goal, success criteria, and sequencing rationale (Phase 28 gates Phases 29 and 30).

### Related pending work
- `.planning/todos/pending/2026-07-24-comaps-poly-region-extraction-spike.md` — validates `pmtiles extract --region <polygon>` end-to-end. Tagged `resolves_phase: 29`, not this phase — the polygon geometry produced here just needs to be structurally correct GeoJSON; the extraction spike is Phase 29's concern.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `db/commands/dedup.go` — the existing pattern for a maintainer-run Cobra command taking `*pocketbase.PocketBase`, useful as a structural template for `seed_regions.go` (though dedup operates on a live DB; seed_regions.go instead writes a JSON file, per the design note).
- `db/migrations/1780000005_add_other_category.go` — the existing pattern for a migration that seeds data from a committed file (`filesystem.NewFileFromPath("migrations/initial_data/...")`) with idempotency guards (`FindFirstRecordByData` before inserting). The `regions` seeding migration should follow the same idempotency-guard shape, adapted for bulk insert.

### Established Patterns
- Migrations live in `db/migrations/*.go`, registered via `m.Register(up, down)`, run automatically on every instance startup — no separate "seed" step exists outside this mechanism.
- `db/services/regions/` (`config.go`, `builder.go`, `staleness.go`) is the existing bbox/`region_config.json`-based system Phase 29 will replace — do not modify it in Phase 28.

### Integration Points
- None yet — this phase only creates the new `regions` collection and seed data. Phase 29 wires the cron and API to read from it; Phase 30 wires the admin UI.

</code_context>

<specifics>
## Specific Ideas

No specific UI/UX references — this is a backend-only, data-model phase. The two implementation choices locked above (fetch-at-runtime sourcing, hand-rolled multi-ring `.poly` parser) are the load-bearing specifics for planning.

</specifics>

<deferred>
## Deferred Ideas

None raised during this discussion — it stayed within phase scope. (Broader deferred items — CATALOG-F01 automated refresh, CATALOG-F02 cascading enable, CATALOG-F03 group-level map preview — are already tracked in `.planning/REQUIREMENTS.md` and don't need re-capturing here.)

</deferred>

---

*Phase: 28-region-catalog-data-model-seeding*
*Context gathered: 2026-07-24*
