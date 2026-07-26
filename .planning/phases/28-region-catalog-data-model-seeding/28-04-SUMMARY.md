---
phase: 28-region-catalog-data-model-seeding
plan: 04
subsystem: database
tags: [go, pocketbase, gzip, compress/gzip, seed-data, migration]

# Dependency graph
requires:
  - phase: 28-region-catalog-data-model-seeding (plan 02)
    provides: seed_regions.go writer (fetch/parse/write flattened JSON seed)
  - phase: 28-region-catalog-data-model-seeding (plan 03)
    provides: regions collection migration + bulk-insert from committed seed
provides:
  - A <100MB gzip-compressed regions_seed.json.gz replacing the undistributable 730MB regions_seed.json
  - seed_regions.go writer emits compact+gzip by default (--out .gz)
  - migration reader gzip-decompresses the seed before json.Unmarshal
affects: [29-polygon-based-extraction-and-region-api, 30-admin-region-picker-ui]

# Tech tracking
tech-stack:
  added: []  # compress/gzip is Go stdlib, no new dependency
  patterns:
    - "Large committed JSON seed artifacts must ship gzip-compressed (compress/gzip, stdlib) rather than raw-indented JSON — GitHub's 100MB hard per-file push limit is a real distribution constraint, not just a local-disk concern"

key-files:
  created: []
  modified:
    - db/commands/seed_regions.go
    - db/migrations/1785000000_create_regions_collection.go
    - db/migrations/initial_data/regions_seed.json.gz

key-decisions:
  - "Regenerated the .gz from the already-verified 730MB regions_seed.json (re-encode, not a full CoMaps network re-fetch) — the seed data was already proven byte-correct by 28-03's live migrate-up test; only the on-disk encoding needed to change"
  - "gzip DefaultCompression (level 6) selected — matches the ~57MB measured point from this session's earlier ad-hoc sizing pass, comfortably under GitHub's 100MB limit without going to a slower/marginal higher compression level"
  - "Reader bounds decompression with a 512MB io.LimitReader (T-28-09 discretionary mitigation) even though the seed is a trusted in-repo artifact of known ~216MB decompressed size"

patterns-established:
  - "Any future large committed data seed in this repo should default to gzip-compressed compact JSON, matching this plan's writer/reader shape"

requirements-completed: [SEED-02, SEED-01, CATALOG-01, CATALOG-02, CATALOG-03]

# Metrics
duration: ~100min
completed: 2026-07-26
---

# Phase 28 Plan 04: Gzip-Compress the Regions Seed Artifact Summary

**Replaced the 730MB committed regions_seed.json with a 57MB gzip-compressed compact JSON artifact, closing the sole failing truth (#10/10) from 28-VERIFICATION.md — the seed is now within GitHub's 100MB hard per-file push limit and pushable to the project's actual git remote.**

## Performance

- **Duration:** ~100 min (dominated by two long-running background processes: a ~35min pure-Python re-encode of the 730MB JSON into gzip, and a ~40min fresh `migrate up` re-proving the 1306-row catalog against a from-scratch pb_data)
- **Tasks:** 2
- **Files modified:** 3 (2 Go source files, 1 binary seed artifact)

## Accomplishments

- `db/commands/seed_regions.go`'s writer now emits compact `json.Marshal` piped through `gzip.NewWriter` (DefaultCompression) instead of `json.MarshalIndent` + raw `os.WriteFile`; `--out` default renamed to `migrations/initial_data/regions_seed.json.gz`
- `db/migrations/1785000000_create_regions_collection.go`'s reader gzip-decompresses (`gzip.NewReader` + bounded `io.LimitReader`) before `json.Unmarshal`, replacing the bare `os.ReadFile` of the removed 730MB file
- Regenerated `db/migrations/initial_data/regions_seed.json.gz` (57,308,025 bytes) from the existing verified seed data; proved zero data loss via a deep-equality diff against the original decompressed content
- Live-tested: a fresh `migrate up` against an empty scratch `pb_data` (with a locally-run Meilisearch to satisfy an unrelated pre-existing migration dependency) reproduced exactly 1306 rows (153 group, 1153 leaf), zero orphaned parent references
- `git rm`'d the 730MB `regions_seed.json`; `git ls-files db/migrations/initial_data/` now lists only `other.jpg` and `regions_seed.json.gz`

## Task Commits

1. **Task 1: Switch the seed writer to compact+gzip and the migration reader to gzip-decompress** - `ef0d15d5` (feat)
2. **Task 2: Regenerate the compressed artifact, prove round-trip + fresh-boot parity, git-rm the 730MB file** - `490a685f` (fix)

**Plan metadata:** commit pending (this SUMMARY + STATE/ROADMAP update)

_Note: no TDD tasks in this plan; each task is a single atomic commit._

## Files Created/Modified

- `db/commands/seed_regions.go` - Writer: compact `json.Marshal` + `gzip.NewWriter` (level 6); `--out` default is now `migrations/initial_data/regions_seed.json.gz`
- `db/migrations/1785000000_create_regions_collection.go` - Reader: `os.Open` + `gzip.NewReader` + bounded `io.ReadAll` before `json.Unmarshal`; all new failure points wrapped with `%w`
- `db/migrations/initial_data/regions_seed.json.gz` - New 57,308,025-byte gzip-compressed compact JSON seed, replacing the deleted 730,548,521-byte `regions_seed.json`

## Decisions Made

- Re-encoded the existing 730MB seed data directly (Python `json.load` + `json.dump(separators=(",",":"))` + `gzip.open(..., compresslevel=6)`) rather than re-running the full CoMaps network fetch — per the plan's explicit guidance, the seed data was already verified byte-correct by 28-03's live testing; only the on-disk encoding needed to change, and re-encoding is both faster and lower-risk than a ~1,150-file network refetch
- Kept `gzip.DefaultCompression` (maps to zlib level 6) rather than a higher compression level — this is the exact level this session's earlier ad-hoc sizing pass measured at ~57MB, comfortably under the 100MB gate; the plan explicitly warned not to drop below this level
- Added a 512MB `io.LimitReader` bound around the migration's gzip decompression (T-28-09 discretionary mitigation) even though the seed is a trusted in-repo artifact — defense in depth against a corrupted/oversized `.gz` at boot

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Reworded a doc comment to avoid the literal substring the plan's own negative grep checks for**
- **Found during:** Task 1 verification
- **Issue:** The new doc comment on `SeedRegions()` explained the format change using the literal phrase `json.MarshalIndent`, which caused the plan's own acceptance-criteria gate (`grep -c 'MarshalIndent' db/commands/seed_regions.go` must return 0) to fail on the comment text alone, even though the actual marshal call had already been switched to compact `json.Marshal`
- **Fix:** Reworded the comment to say "indenting the marshaled output" instead of naming `json.MarshalIndent` literally — same intent, no literal substring collision
- **Files modified:** db/commands/seed_regions.go
- **Verification:** `grep -c 'MarshalIndent' db/commands/seed_regions.go` now returns 0; `go build ./... && go vet ./...` clean
- **Committed in:** ef0d15d5 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1)
**Impact on plan:** Cosmetic-only (comment wording); no functional or scope change. Same precedent as several prior phases in this project (16-02, 25.1-01) where a doc comment is reworded to avoid colliding with the plan's own literal negative-grep gate.

## Issues Encountered

- The fresh `migrate up` initially failed with `unaccepted status code found: 403 ... invalid_api_key` because the locally-run Meilisearch container's `MEILI_MASTER_KEY` didn't match the value passed to the Go process via env var — fixed by restarting the container with a matching (32+ byte) key.
- A second failure, `Error: failed to apply migration 1780566579_add_iri_to_local_resources.go: ORIGIN not set`, required adding an `ORIGIN` env var (unrelated pre-existing migration in the chain, not part of this plan's scope) to let the full `migrate up` chain reach the regions migration.
- The Docker daemon was not running at the start of the session; started `Docker.app` and waited for `docker info` to succeed before spinning up the Meilisearch container needed for the live migrate-up gate.
- Both the ~216MB pure-Python JSON re-encode and the ~2,388-row PocketBase bulk insert/link transaction were slow (per-row `Save()` overhead, no batching in the migration's existing two-pass loop) — each took tens of minutes. This is pre-existing behavior from 28-03's migration code (unchanged by this plan) and not a regression introduced here; noted for awareness, not fixed (out of this plan's `files_modified` scope).

## User Setup Required

None - no external service configuration required. The gzip-compressed seed and updated writer/reader are self-contained; no Git LFS, no CI change, no new dependency (`compress/gzip` is Go stdlib).

## Next Phase Readiness

- SEED-02's "zero admin action, real distributable artifact" promise now holds: the committed `regions_seed.json.gz` (57,308,025 bytes) is comfortably under GitHub's 100MB hard per-file push limit, so the branch can be pushed and every real `git clone` will contain a working seed.
- CATALOG-01/02/03 and SEED-01 remain unregressed — re-proven via the same live `migrate up` test that 28-VERIFICATION.md ran, now against the distributable `.gz` artifact instead of the raw 730MB file.
- Phase 29 (Polygon-Based Extraction & Region API) and Phase 30 (Admin Region Picker UI) can both proceed against the seeded `regions` table with no remaining Phase 28 gaps.
- **Claude's Discretion / follow-up (not a task, per the plan's own output note):** the original 730MB blob remains reachable in unpushed local history via commit `67d056c9` (and any other earlier local-only commit that carried it). This plan's `git rm` only removes it from the working tree/index going forward; GSD's commit protocol never amends/rebases prior commits. If the cumulative unpushed push size becomes a concern before this branch is first pushed, a separate maintainer decision (e.g. `git filter-repo`/BFG on the local-only commits, or squashing the local branch) can address it — out of scope here.

---
*Phase: 28-region-catalog-data-model-seeding*
*Completed: 2026-07-26*

## Self-Check: PASSED

- FOUND: db/commands/seed_regions.go
- FOUND: db/migrations/1785000000_create_regions_collection.go
- FOUND: db/migrations/initial_data/regions_seed.json.gz (57,308,025 bytes)
- CONFIRMED REMOVED: db/migrations/initial_data/regions_seed.json
- FOUND commit: ef0d15d5
- FOUND commit: 490a685f
- FOUND commit: 4598d5dc
