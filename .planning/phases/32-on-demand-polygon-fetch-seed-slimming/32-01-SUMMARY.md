---
phase: 32-on-demand-polygon-fetch-seed-slimming
plan: 01
subsystem: database
tags: [go, pocketbase, cobra, geojson, comaps]

# Dependency graph
requires:
  - phase: 28-region-catalog-data-model-seeding
    provides: ParsePoly (Osmosis .poly -> GeoJSON parser), ParseHierarchy, the original gzip-compressed regions_seed.json.gz seed
provides:
  - Hierarchy-only seed_regions.go generator (single hierarchy.txt fetch, no .poly scrape)
  - SeedCatalog{Commit, Rows} wrapper shape for the committed catalog
  - ParsePoly relocated into package regions, importable by the runtime fetch path with no services->commands dependency
  - Regenerated db/migrations/initial_data/regions_seed.json (plain pretty-printed, pure hierarchy, ~315 KB)
affects: [32-02-migration-region-geometry-collection, 32-03-runtime-polygon-fetch, 32-06-purge-gz-blob]

# Tech tracking
tech-stack:
  added: []
  patterns: ["wrapper struct carrying provenance (Commit) once instead of per-row", "package relocation to avoid services->commands dependency direction"]

key-files:
  created:
    - db/services/regions/poly_parser.go
    - db/services/regions/poly_parser_test.go
    - db/migrations/initial_data/regions_seed.json
  modified:
    - db/commands/seed_regions.go
  deleted:
    - db/commands/poly_parser.go
    - db/commands/poly_parser_test.go

key-decisions:
  - "SeedCatalog wrapper (Commit + Rows) chosen over per-row commit field, per D-00b and the plan's diff-noise rationale"
  - "ParsePoly moved verbatim (no logic change) into package regions so plan 32-03's runtime fetch path avoids a services/regions -> commands import"
  - "regions_seed.json.gz deliberately retained on disk -- migration still reads it until 32-02, and 32-03 uses it as the D-07 value-equality oracle"

patterns-established:
  - "Provenance-in-artifact: a generated catalog records the exact input (pinned commit) it was generated from, once, inside the artifact itself rather than as a separate shared const"

requirements-completed: [SLIM-01]

# Metrics
duration: 10min
completed: 2026-07-28
---

# Phase 32 Plan 01: Slim Seed Generator & Relocate ParsePoly Summary

**Collapsed `seed-regions` from a ~1153-request polygon scrape into a single `hierarchy.txt` fetch, moved `ParsePoly` from package `commands` to `services/regions`, and regenerated the committed catalog as pure-hierarchy, plain pretty-printed JSON (~315 KB, 1306 rows, no gzip layer).**

## Performance

- **Duration:** 10 min
- **Started:** 2026-07-28T09:49:00Z (approx.)
- **Completed:** 2026-07-28T09:51:21Z
- **Tasks:** 3 completed
- **Files modified:** 5 (1 modified, 3 created, 2 deleted/renamed)

## Accomplishments
- `seed_regions.go` now issues exactly one HTTP request per run (`hierarchy.txt` only); the `.poly` fetch loop, `gzip` layer, and `--limit` flag are gone
- New `SeedCatalog{Commit, Rows}` wrapper records the pinned CoMaps commit SHA exactly once inside the artifact (D-00b)
- `ParsePoly` and its full test suite relocated from `db/commands/` to `db/services/regions/` via `git mv`, package renamed, logic untouched — clears the way for plan 32-03's runtime fetch path to consume it without a `services/regions -> commands` dependency
- `db/migrations/initial_data/regions_seed.json` regenerated: 1306 rows (153 groups, 1153 leaves), plain pretty-printed JSON, ~315 KB, no `bbox`/`polygon` keys, commit recorded once

## Task Commits

Each task was committed atomically:

1. **Task 1: Slim seed_regions.go to a single hierarchy.txt fetch with a wrapper artifact** - `8b07fbcd` (feat)
2. **Task 2: Relocate ParsePoly and its tests from package commands to package regions** - `2f8f5de2` (refactor, add side) + `f8349e05` (refactor, delete side — see Deviations)
3. **Task 3: Regenerate the committed catalog artifact** - `d6c0bafd` (feat)

_Note: Task 2's `git mv` rename was split across two commits (see Deviations) because the first commit's pathspec filter only captured the add side of the rename._

## Files Created/Modified
- `db/commands/seed_regions.go` - Rewritten `SeedRegions()`: single `hierarchy.txt` fetch, `SeedCatalog` wrapper, `MarshalIndent` output, no gzip, no `--limit` flag
- `db/services/regions/poly_parser.go` - `ParsePoly` and its package-private helpers, relocated verbatim from `db/commands/poly_parser.go`, `package regions`
- `db/services/regions/poly_parser_test.go` - Full `TestParsePoly` suite, relocated verbatim, `package regions`
- `db/migrations/initial_data/regions_seed.json` - New pure-hierarchy catalog artifact, 1306 rows, ~315 KB

## Decisions Made
- Used a `SeedCatalog` wrapper struct (`Commit string`, `Rows []SeedRow`) rather than adding a commit field to every row, matching the plan's diff-noise rationale (D-00a/D-00b)
- Kept `fetch`/`doFetch`/`parseRetryAfter`/`maxFetchRetries = 10` unchanged in the generator per D-03 — the generator keeps its patient retry budget; the build path gets its own tighter one in a later plan
- Did not touch `db/migrations/1785000000_create_regions_collection.go`'s local `SeedRow` reader struct (out of this plan's `files_modified` scope; that's plan 32-02)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Split rename commit left the deletion side of `git mv` uncommitted**
- **Found during:** Task 2 (relocate ParsePoly)
- **Issue:** The task commit protocol's per-task commit used a `-- <pathspec>` filter naming only the new file paths. Since `git mv`'s deletion of the old `db/commands/poly_parser*.go` paths wasn't included in that pathspec, `git commit -- <new paths>` committed only the "add" half of the rename, leaving the old files staged as deleted in the index (`git status` showed `D` for both old paths afterward).
- **Fix:** Committed the remaining staged deletions in a follow-up commit (`f8349e05`) so the full rename is captured across the two commits (git still resolves this as a rename in `git log --follow`/`git show -M`).
- **Files modified:** `db/commands/poly_parser.go`, `db/commands/poly_parser_test.go` (deletion committed)
- **Commit:** `f8349e05`

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** No scope creep — purely a commit-mechanics correction; the net working-tree state after both commits matches the plan's `git mv` instruction exactly (verified: `test ! -f db/commands/poly_parser.go && test ! -f db/commands/poly_parser_test.go` passes).

## Issues Encountered
None beyond the deviation above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `ParsePoly` lives in `db/services/regions/` and is ready for plan 32-03's runtime fetch path to import with no cross-package dependency issue
- `db/migrations/initial_data/regions_seed.json.gz` remains on disk, unmodified, for plan 32-02 (migration switch-over) and plan 32-03 (D-07 value-equality oracle)
- `db/migrations/1785000000_create_regions_collection.go` still reads the old `.gz` artifact and its own local `SeedRow` (with `Polygon`/`Bbox`) — unchanged in this plan, to be updated in plan 32-02
- No blockers identified

---
*Phase: 32-on-demand-polygon-fetch-seed-slimming*
*Completed: 2026-07-28*

## Self-Check: PASSED

All created files confirmed present on disk; all 4 task/deviation commits confirmed present in git history.
