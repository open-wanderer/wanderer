---
phase: 32-on-demand-polygon-fetch-seed-slimming
plan: 02
subsystem: database
tags: [go, pocketbase, migrations, sqlite]

# Dependency graph
requires:
  - phase: 32-on-demand-polygon-fetch-seed-slimming
    plan: 32-01
    provides: SeedCatalog{Commit, Rows} wrapper shape, regenerated plain-JSON regions_seed.json (1306 rows, no bbox/polygon)
provides:
  - Edited-in-place migration (1785000000_create_regions_collection.go) creating a hierarchy-only regions collection with catalog_commit, plus an empty region_geometry collection (bbox + polygon)
  - Path-reference integrity hook rebound from region_polygons to region_geometry (db/main.go, db/hooks/regions.go)
  - Verified fresh-boot proof: a network-free migrate up reseeds the full 1306-row catalog with catalog_commit populated and region_geometry empty
affects: [32-03-runtime-polygon-fetch, 32-04-build-region-geometry-read, 32-05-geometry-route, 32-06-purge-gz-blob]

# Tech tracking
tech-stack:
  added: []
  patterns: ["plain os.ReadFile + json.Unmarshal replacing a streaming gzip decoder once the artifact shrinks below ~1MB", "commit-SHA-on-every-row rather than a shared const, mirroring 32-01's artifact-carries-provenance pattern"]

key-files:
  created: []
  modified:
    - db/migrations/1785000000_create_regions_collection.go
    - db/main.go
    - db/hooks/regions.go

key-decisions:
  - "D-05 honored exactly: migration edited in place, no separate drop/rename migration, filename unchanged, CountRecords idempotency guard kept byte-for-byte"
  - "region_polygons -> region_geometry rename carries bbox onto the new collection alongside polygon; regions loses its bbox field entirely, gains catalog_commit (required text)"
  - "Fresh-boot proof required a locally-running Meilisearch (Docker) plus ORIGIN/MEILI_* env vars for unrelated earlier migrations in the same migrate-up chain -- this is pre-existing infrastructure, not part of this plan's change"

patterns-established:
  - "Reworded in-file comments to avoid the literal retired collection name entirely (not just in logic) so a plain grep for the old name returns zero hits, matching the plan's own acceptance-criteria grep gates"

requirements-completed: [SLIM-02, SLIM-04]

# Metrics
duration: ~20min
completed: 2026-07-28
---

# Phase 32 Plan 02: Migration Region-Geometry Collection Summary

**Edited `1785000000_create_regions_collection.go` in place to drop geometry from the seed path entirely: `regions` now carries `catalog_commit` and no `bbox`, `region_geometry` (renamed from `region_polygons`) ships empty with both `bbox` and `polygon`, and a verified fresh, network-free `migrate up` reseeds the full 1306-row hierarchy.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-07-28T09:55:00Z (approx.)
- **Completed:** 2026-07-28T10:00:56Z
- **Tasks:** 3 completed
- **Files modified:** 3 (db/migrations/1785000000_create_regions_collection.go, db/main.go, db/hooks/regions.go)

## Accomplishments
- Migration's seed reader replaced: the gzip layer, the token-by-token streaming `json.Decoder`, and the 512 MB `io.LimitReader` decompression-bomb bound are all gone, replaced by a plain `os.ReadFile` + `json.Unmarshal` into the local `SeedCatalog{Commit, Rows}` struct (mirroring 32-01's writer-side shape byte-for-byte)
- `regions` collection loses its `bbox` field and gains a required `catalog_commit` text field, set from `catalog.Commit` on every inserted row (group and leaf alike)
- `region_polygons` renamed to `region_geometry`, now holding both `bbox` (moved from `regions`) and `polygon`; the migration's entire geometry bulk-insert block is deleted, so the collection ships genuinely empty
- Down migration, indexes, and the two-`Save()` self-relation dance for `parent` are all preserved unchanged, per D-05's edit-in-place instruction
- Path-reference integrity hook (`ValidateRegionPathReferenceHandler`) rebound in `db/main.go` from `region_archives, region_polygons` to `region_archives, region_geometry`; the admin-page privileged-surface comment updated to match; `db/hooks/regions.go`'s doc comment updated (function body untouched)
- Verified end-to-end on a genuinely fresh `pb_data`: `migrate up` from `db/` with no code touching the network beyond what pre-existing, unrelated migrations already required (Meilisearch index creation) produced exactly 1306 `regions` rows (153 group / 1153 leaf), a single distinct `catalog_commit` value (`2528fbb91977201cf6d16b1b01ebf27eea342e85`), zero rows in `region_geometry`, no `bbox` column on `regions`, zero pre-enabled leaves, and zero orphaned `parent` links

## Task Commits

Each task was committed atomically:

1. **Task 1: Rewrite the migration for a hierarchy-only regions table and an empty region_geometry** - `e3b17677` (feat)
2. **Task 2: Rebind the path-reference integrity hook to region_geometry** - `f6f69792` (refactor)
3. **Task 3: Prove a fresh, network-free boot reseeds the full catalog with an empty region_geometry** - verification-only; no committable change (`db/pb_data*` is gitignored per `db/.gitignore:2`)

## Files Created/Modified
- `db/migrations/1785000000_create_regions_collection.go` - Edited in place: local `SeedRow`/`SeedCatalog` reader structs match 32-01's writer; `regions` gains `catalog_commit`, loses `bbox`; `region_geometry` (renamed) holds `bbox` + `polygon`, ships empty; gzip/streaming-decoder/bound all removed; down migration targets `region_geometry`
- `db/main.go` - `OnRecordCreate`/`OnRecordUpdate` hook bindings and the admin-page doc comment repointed from `region_polygons` to `region_geometry`
- `db/hooks/regions.go` - `ValidateRegionPathReferenceHandler` doc comment updated to name `region_geometry` and the on-demand geometry fetch's persist-on-enabled write instead of the retired seed-migration insert; handler body unchanged

## Decisions Made
- Kept every comment reference to the retired collection name reworded (not just logic), since the plan's own acceptance criteria grep for zero occurrences of `region_polygons` in the migration file, and the phase-level verification greps the same across `db/main.go`/`db/hooks/regions.go`
- Used a local Docker Meilisearch container plus `MEILI_URL`/`MEILI_MASTER_KEY`/`ORIGIN` env vars to satisfy earlier, unrelated migrations in the same `migrate up` chain (e.g. `1742167033_init_meilisearch.go`, `1780566579_add_iri_to_local_resources.go`) that had to run before this plan's target migration could execute on a genuinely fresh database — this is pre-existing infrastructure dependency, not a change introduced by this plan

## Deviations from Plan

None - plan executed exactly as written. The only unplanned work was environment setup (starting Docker/Meilisearch and setting `ORIGIN`) required to exercise the *earlier*, unrelated migrations that run before this plan's target migration in the same `migrate up` invocation — no code changed as a result, and this is explicitly the kind of one-time local environment step the plan's Task 3 verification anticipated ("run the migration ... from db/").

## Issues Encountered
- The first `migrate up` attempt failed on `1742167033_init_meilisearch.go` (`unsupported protocol scheme ""`) because no `MEILI_URL`/`MEILI_MASTER_KEY` was set and Docker wasn't running. Started Docker Desktop, ran a throwaway `getmeili/meilisearch:v1.11.3` container on `127.0.0.1:17700`, and set `MEILI_URL`/`MEILI_MASTER_KEY` accordingly.
- The second attempt failed on `1780566579_add_iri_to_local_resources.go` (`ORIGIN not set`) — added `ORIGIN=http://localhost:8090` and re-ran. Third attempt succeeded, applying every pending migration including this plan's target one. The temporary Meilisearch container was stopped and removed after verification.

## User Setup Required
None - no external service configuration required for this plan's own change. (The temporary Meilisearch container used only for local verification has already been torn down.)

## Next Phase Readiness
- `region_geometry` exists, empty, with both `bbox` and `polygon` fields, ready for plan 32-03's runtime fetch path and plan 32-04's `buildRegion` read/upsert
- `regions.catalog_commit` is populated on every row, ready for the geometry fetch path to read the pinned commit off the leaf's own record with zero extra lookups
- The path-reference hook already guards `region_geometry`, so plan 32-03/32-05's geometry-persist writes will be validated for free
- **Repo-wide `region_polygons` references remain** outside this plan's `files_modified` scope, by design — deferred to their owning plans: `db/services/regions/builder.go` (lines ~228-234, plan 32-04's `buildRegion` rewrite), `db/routes/regions_ui.go` (doc comment, plan 32-05/30 admin-page work), `db/migrations/1785100000_rename_region_archives_region_id_to_path.go` (historical comment, out of scope for any plan — pre-existing migration text describing the old shape at the time it was written)
- The pre-existing dev `pb_data` was preserved (not deleted) at `db/pb_data.pre32.1785232728/` before Task 3's fresh-boot proof; the freshly migrated `pb_data` (regenerated during this plan) is left in place as the working dev database. The backup directory can be removed once no longer needed for comparison.
- No blockers identified

---
*Phase: 32-on-demand-polygon-fetch-seed-slimming*
*Completed: 2026-07-28*

## Self-Check: PASSED

All modified files confirmed present on disk; all 3 commits (`e3b17677`, `f6f69792`, `cc6f1a59`) confirmed present in git history.
