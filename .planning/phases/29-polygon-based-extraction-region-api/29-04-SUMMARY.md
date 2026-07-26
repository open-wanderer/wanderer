---
phase: 29-polygon-based-extraction-region-api
plan: 04
subsystem: verification
tags: [pmtiles, verification, checkpoint, region-archives, regions-api]

# Dependency graph
requires:
  - phase: 29-polygon-based-extraction-region-api (plan 01)
    provides: Table-driven cron + pmtiles --region= polygon extraction
  - phase: 29-polygon-based-extraction-region-api (plan 02)
    provides: Hierarchy-aware GET /api/v1/regions, region_config.json loader retired
  - phase: 29-polygon-based-extraction-region-api (plan 03)
    provides: SvelteKit regionIDPattern lockstep + OpenAPI docs
provides:
  - "Live, real-data confirmation that EXTRACT-01/02/03 hold end-to-end against a real seeded region on a real dev instance"
affects: []

key-files:
  created: []
  modified: []

key-decisions:
  - "Verification was run directly against the operator's already-running dev instance (Go backend on :8090, SvelteKit dev server on :5173) rather than starting a fresh instance, since all six checks are mechanically verifiable (tile counts, HTTP status codes, grep, JSON field presence) rather than subjective UI judgment"
  - "The enabled leaf used for verification (germany.germany_free_state_of_bavaria.germany_free_state_of_bavaria_upper_bavaria_south / 'Upper Bavaria South') was already built by the operator prior to this session picking up the checkpoint; enabled flag was temporarily toggled to false to prove the zero-target cron behavior, then restored to true afterward"

patterns-established:
  - "PMTiles' `bounds` header field is always a rectangle (min/max lon/lat) regardless of polygon clipping — this is inherent to the container format, not evidence of a clipping failure. The correct proof of clipping is a tile-count delta against a bbox-only baseline extract plus a spot-checked bbox-corner tile that is present in the baseline but absent in the polygon-clipped archive."

requirements-completed: [EXTRACT-01, EXTRACT-02, EXTRACT-03]

# Metrics
duration: ~30min
completed: 2026-07-26
---

# Phase 29 Plan 04: End-to-End Verification Summary

**All six verification steps confirmed against a real seeded region on the operator's live dev instance — polygon-based extraction, table-driven cron targeting, the hierarchy-aware region API, and the relaxed `.`-path download route all hold with real data, not just unit tests.**

## Evidence

### Step 3 — Polygon clipping (EXTRACT-01)

Region verified: `germany.germany_free_state_of_bavaria.germany_free_state_of_bavaria_upper_bavaria_south` ("Upper Bavaria South"), bbox `[10.71568, 47.39335, 11.68018, 48.06707]`.

- Built archive (`pmtiles extract --region=<polygon>`): **2,161** addressed tiles.
- Bbox-only baseline (`pmtiles extract --bbox=<same bbox>` against the identical Protomaps daily-build source): **2,909** addressed tiles.
- All four bbox corners at z14 (8723,5690 / 8679,5736 / 8723,5736 / 8679,5690): `Tile not found in archive.` in the polygon-clipped build; present with real tile payloads (2.6–47.8 KB) in the bbox baseline.

748 tiles dropped, concentrated exactly at the four bbox corners — the expected signature of a non-rectangular administrative boundary being clipped. Note: PMTiles' `bounds` metadata field is always a rectangle regardless of clipping (inherent to the container format) — this was surfaced as an operator question mid-checkpoint and is recorded as a pattern above so it doesn't get mistaken for a bug in a future verification pass.

### Step 4 — Table-driven targets, no config read (EXTRACT-02)

- `grep -rn "region_config.json\|REGION_CATALOG_CONFIG_PATH\|LoadRegionCatalog" db --include='*.go' | grep -v '_test'` returned only two documentary comments (`config.go:7`, `regions_get.go:18`) explaining the retirement — no functional reference, no `os.Open`, no env var read.
- `BuildAll`'s query (`kind = 'leaf' AND enabled = true` via `dbx.NewExp`) confirmed table-driven and deterministic by source inspection; toggling the one enabled leaf to `enabled=0` dropped the enabled-leaf count to 0 (confirmed via direct SQL count), proving the query has zero build targets when nothing is enabled.

### Step 5 — Hierarchy API (EXTRACT-03)

`GET /regions` (Go internal route, port 8090) returned 1,306 rows:
- Every row carries `parent`/`path`/`depth`/`kind`.
- 153 `group` rows present (e.g. `Algeria`), each correctly omitting `bbox`/`status`.
- The verified leaf row retained `id`/`name`/`bbox`/`status` plus `vector_url`/`dem_url`/`vector_size`/`dem_size` now that it's built — existing shape fully intact, additive only.

### Step 6 — Relaxed `.`-path download (Pitfall 1 lockstep)

- Go internal route (`/regions/<path-with-dots>/download`, port 8090): `HTTP 200`, `32,400,317` bytes — exact match to the leaf's recorded `vector_size`.
- SvelteKit proxy (`/api/v1/regions/<path-with-dots>/download`, port 5173): `HTTP 200`, same byte count, valid PMTiles magic bytes in the response body.
- Traversal guard sanity check: `/regions/../../etc/passwd/download` → `HTTP 400 "Invalid region id."` — confirms the relaxed allow-list still rejects `..` traversal (T-29-09 mitigation holds).

## Outcome

All six steps pass. EXTRACT-01, EXTRACT-02, and EXTRACT-03 are confirmed end-to-end against real data, closing Phase 29.

The one stateful action taken against dev data — toggling the verified leaf's `enabled` flag to `false` and back to `true` to prove the zero-target cron behavior — was reverted; the dev database is in the same state it was in before this checkpoint, aside from the pre-existing built archive from the operator's own earlier run.

### Real bug found by this checkpoint (fixed)

While reconciling the operator's already-running dev instance, two uncommitted working-tree diffs were found against 29-01/29-02's committed code:

1. **`db/services/regions/builder.go` — genuine bug, fixed and committed.** `BuildAll`'s `dbx.NewExp` predicate was committed as `"kind = {:kind} && enabled = {:enabled}"` — the Go logical operator `&&`, not SQL's `AND`, which SQLite does not accept as a boolean operator in a `WHERE` clause. No `builder_test.go` exists, so this predicate was never exercised against real SQLite by any unit test in 29-01/29-02 — it went undetected until the operator ran the actual cron for real during this checkpoint, hit the broken query, and corrected it locally to `AND` before archive-building actually worked. This is precisely the class of gap a per-plan `go build`/`go vet`/mocked-unit-test gate cannot catch, and precisely why this plan exists. Committed as a standalone fix (`dbe1dd20`).
2. **`db/main.go` — local test convenience, reverted, not committed.** `regionsGroup.Bind(apis.RequireAuth())` was commented out in the working tree, contradicting the locked D-07 decision ("Auth is ENABLED... for both the catalog listing and the archive downloads") that the adjacent comment still documents. This was almost certainly a temporary local change to make unauthenticated `curl` testing easier during manual verification, not an intended change — reverted back to the committed (auth-enabled) state, not committed.
