---
phase: 28-region-catalog-data-model-seeding
plan: 02
subsystem: database
tags: [go, cobra, comaps, fetch, seeding, github-mirror]

# Dependency graph
requires: [28-01]
provides:
  - "seed-regions CLI command (db/commands/seed_regions.go) — fetches CoMaps hierarchy.txt + leaf .poly files and writes a flattened JSON seed"
  - "SeedRow struct + defaultCommitHash const + --commit/--out/--limit flags"
  - "db/migrations/initial_data/regions_seed.json — committed 1306-row flattened CoMaps region catalog"
affects: [28-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Stdlib-only fetch-transform-write Cobra command (no *pocketbase.PocketBase param — never touches a live DB)"
    - "429 retry-with-backoff honoring Retry-After, bounded by maxFetchRetries, non-429 failures surfaced as fatal immediately"
    - "io.LimitReader-bounded HTTP reads (8MB hierarchy.txt / 32MB per .poly) to cap memory against unexpectedly large upstream responses"

key-files:
  created:
    - db/commands/seed_regions.go
    - db/migrations/initial_data/regions_seed.json
  modified:
    - db/main.go

key-decisions:
  - "Fetch source switched from Codeberg (comaps/comaps, per D-01's original wording) to comaps/comaps's GitHub mirror (raw.githubusercontent.com) at the same pinned commit — Codeberg's raw-file endpoint enforces a ~250 requests/600s quota that a full ~1,150-leaf run routinely exhausted (observed: two separate runs stalled in 10-minute rate-limit backoffs, one of which was lost entirely to an unrelated machine restart since output is only written at the very end). GitHub's mirror serves byte-identical content for the same commit hash with no rate limiting observed across a 20-request burst test and the full 1,153-leaf run. D-01's substance (fetch fresh at run time, commit only the JSON output, never vendor raw upstream files) is unchanged — only the specific host serving that content changed."
  - "defaultCommitHash baked in as 2528fbb91977201cf6d16b1b01ebf27eea342e85 (CoMaps main HEAD at implementation time, resolved via Codeberg's Forgejo commits API), reused unchanged for the GitHub mirror since both hosts serve the same repository history"
  - "Added a retry-with-backoff (fix commit d5e1ef1f, now largely moot after the GitHub switch but kept as defensive behavior) — a 429 is retried up to 10 times honoring Retry-After, other non-200 statuses fail immediately via log.Fatalf naming the offending file"

requirements-completed: [SEED-01, CATALOG-01, CATALOG-02]

# Metrics
duration: ~90min (including two full-generation attempts against Codeberg's rate limit and the GitHub-mirror switch)
completed: 2026-07-25
---

# Phase 28 Plan 02: seed_regions.go Fetch-Transform-Write Command Summary

**`seed_regions.go` — a maintainer-run Cobra command that fetches CoMaps' `hierarchy.txt` and every leaf `.poly` file from comaps/comaps's GitHub mirror at a pinned commit, converts them via the plan 28-01 parsers, and writes a flattened JSON seed array — plus the committed `regions_seed.json` output itself (1306 rows: 153 groups, 1153 leaves).**

## Performance

- **Duration:** ~90 min (dominated by working around Codeberg's rate limiting, not implementation time)
- **Completed:** 2026-07-25
- **Tasks:** 2/2 completed
- **Files modified:** 3 (2 new, 1 modified)

## Accomplishments
- `SeedRegions()` returns a `seed-regions` Cobra command registered in `db/main.go`'s `setupCommands`, taking no `*pocketbase.PocketBase` (never touches a live DB).
- Validates `--commit` against `^[0-9a-f]{7,40}$` before URL interpolation (Threat T-28-03), bounds every HTTP read via `io.LimitReader` (Threat T-28-04), and calls the plan 28-01 parsers directly (`ParseHierarchy`, `ParsePoly`) rather than re-implementing parsing.
- A committed `db/migrations/initial_data/regions_seed.json` holds the full flattened CoMaps catalog: 1306 rows, every leaf carrying a GeoJSON `Polygon`/`MultiPolygon` + 4-element `[minLon,minLat,maxLon,maxLat]` bbox, every group row carrying neither, no `World`/`WorldCoasts` header rows.
- Spot-checked: Fiji serializes as a `MultiPolygon` leaf (antimeridian split), Germany is a group with 16 children, no header rows present.

## Task Commits

Each task was committed atomically, plus two follow-up fixes required to get the full generation to actually complete:

1. **Task 1: seed_regions.go fetch-transform-write command + main.go registration** — `639248fd` (feat)
2. **Rate-limit retry-with-backoff** — `d5e1ef1f` (fix) — added after the first full-generation attempt against Codeberg hit repeated 429s
3. **Fetch source switch to GitHub mirror** — `7391fcf6` (fix) — Codeberg's quota made a full run impractical (see Decisions); switched `baseURL` to `raw.githubusercontent.com/comaps/comaps` at the same pinned commit
4. **Task 2: Generate and commit the full regions_seed.json** — `67d056c9` (feat)

## Files Created/Modified
- `db/commands/seed_regions.go` — `SeedRegions() *cobra.Command`, `SeedRow` struct, `defaultCommitHash` const, `fetch`/`doFetch`/`parseRetryAfter` helpers
- `db/main.go` — one line added to `setupCommands`: `app.RootCmd.AddCommand(commands.SeedRegions())`
- `db/migrations/initial_data/regions_seed.json` — generated, committed output (not hand-edited)

## Decisions Made
- See `key-decisions` above (GitHub-mirror source switch, baked commit hash reuse, retry-with-backoff).
- Honored `--limit` for a fast 3-leaf smoke test (Task 1's verify step) before committing to the full ~1,150-leaf run (Task 2).

## Deviations from Plan

### Auto-fixed Issues

**1. [Environmental — not a plan defect] Fetch source changed from Codeberg to GitHub mirror**
- **Found during:** Task 2 (full generation run)
- **Issue:** Codeberg's raw-file endpoint enforces a tight per-window quota (~250 requests/600s) that the full run — 1 hierarchy.txt + 1,153 leaf `.poly` fetches — routinely exhausted, causing repeated 10-minute rate-limit stalls. One run was lost entirely to an unrelated machine restart mid-wait, since the command only writes output once at the very end (no incremental checkpointing).
- **Fix:** Verified comaps/comaps is mirrored to GitHub and that `raw.githubusercontent.com` serves identical content for the same pinned commit (confirmed via direct byte-identical fetch checks and a 20-request burst test with no rate limiting). Switched `baseURL` in `seed_regions.go` to the GitHub mirror; re-ran the full generation, which completed in a couple of minutes with zero rate-limit hits.
- **Files modified:** `db/commands/seed_regions.go`
- **Verification:** Full run completed cleanly (`seed-regions: wrote 1306 rows (153 groups, 1153 leaves, 1153 leaf .poly files fetched)`); all Task 2 structural assertions and human-check spot-checks pass; `go build`/`go vet`/`go test` all green.
- **Committed in:** `7391fcf6` (fix), followed by `67d056c9` (the actual generated seed file)

---

**Total deviations:** 1 (environmental fetch-source substitution, not a code defect in the original plan or implementation)
**Impact on plan:** D-01's substance (fetch fresh at run time, never vendor raw upstream files, commit only the JSON output) is fully preserved — only the specific upstream host changed, and it serves byte-identical content for the same commit.

## Issues Encountered
Codeberg rate limiting made two consecutive full-generation attempts impractical (one lost to an unrelated laptop power-down mid-wait). Resolved by switching to GitHub's mirror of the same repository at the same commit — see Deviations above.

## User Setup Required

None — no external service configuration required. (Re-running `seed-regions` in the future requires network access to `raw.githubusercontent.com`.)

## Next Phase Readiness
- `db/migrations/initial_data/regions_seed.json` is committed and ready for plan 28-03's migration to bulk-insert.
- No blockers.

---
*Phase: 28-region-catalog-data-model-seeding*
*Completed: 2026-07-25*

## Self-Check: PASSED

All created files and commit hashes verified present on disk / in git log.
