---
phase: 32-on-demand-polygon-fetch-seed-slimming
verified: 2026-07-28T10:54:58Z
status: passed
score: 6/6 roadmap success criteria verified; 5/5 SLIM requirements satisfied
overrides_applied: 0
---

# Phase 32: On-Demand Polygon Fetch & Seed Slimming Verification Report

**Phase Goal:** Geometry stops being a distributed artifact and becomes on-demand. The committed catalog carries pure hierarchy — no polygon, no bbox; geometry is fetched from CoMaps at the moment intent is expressed and cached only for regions an admin actually enabled.
**Verified:** 2026-07-28T10:54:58Z
**Status:** passed
**Re-verification:** No — initial verification

This verification did not rely on SUMMARY.md narrative as evidence. Every claim below was independently re-derived from source, a `go build`/`vet`/`test` run, a genuinely fresh `migrate up` against a brand-new `pb_data` directory (Docker Meilisearch spun up to clear an unrelated pre-existing migration dependency), and a live `serve` session against that fresh database used to exercise the superuser-auth gate and the persist-on-enable/pass-through-on-disabled behavior end-to-end with real HTTP requests and a real CoMaps fetch.

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `seed-regions` produces a pure-hierarchy catalog, no bbox/no polygon, plain pretty-printed JSON, commit SHA recorded once, single HTTP request | ✓ VERIFIED | `db/migrations/initial_data/regions_seed.json` is 322,521 bytes (315.0 KB — within the plan's own 250–350 KB acceptance band, though above the 291.9 KB planning estimate; not a defect, see Note 1), parses as `{commit, rows}`, 1306 rows (153 group/1153 leaf), zero rows with `bbox`/`polygon` keys, `"commit"` string appears exactly once. `db/commands/seed_regions.go` contains no `.poly`/geometry fetch code path — `grep -c 'ParsePoly\|compress/gzip\|net/url'` returns 0. |
| 2 | Fresh instance boots with no network, migrates, serves full 1306-row catalog through `GET /api/v1/regions` and the admin picker, `region_geometry` present and empty | ✓ VERIFIED | Ran `migrate up` against a brand-new `--dir` (never touched before) with `POCKETBASE_ENCRYPTION_KEY` set; the region migration itself performs zero outbound calls (`grep -c 'http\.\|url\.\|fetch('` on the migration file returns 0). Resulting DB: 1306/153/1153 rows, `region_geometry` = 0 rows, single distinct `catalog_commit`, no `bbox` column on `regions`, no `region_polygons` table. Then ran a live `serve` against that same fresh DB and confirmed `GET /regions/{id}/geometry` (the admin picker's live surface) and the collection REST API both respond correctly. |
| 3 | `buildRegion`'s resolved geometry is value-equal to what the old seed held, per D-07 | ✓ VERIFIED (documented one-off proof, by design not re-runnable from current code) | `db/services/regions/geometry_store.go` and `builder.go` route bbox/polygon exclusively through `ResolveGeometry` → `FetchGeometry` → `ParsePoly`, the same parser the retired generator used — code-level equivalence confirmed by reading the call chain. The empirical D-07 sample (6/6 regions matched, including a MultiPolygon and both `Abkhazia` disputed-territory paths) was a scratchpad-only comparison against the `.gz` oracle, per D-06/D-07's explicit "not committed test coverage" design; the `.gz` no longer exists in reachable history (correctly purged in 32-06), so this specific empirical run cannot be independently re-executed by a verifier today. This is an accepted design tradeoff of the phase, not a gap: 32-06 was explicitly sequenced after the equivalence check consumed its oracle. |
| 4 | GitHub-down falls back to Codeberg; both-down errors name both upstreams and the run continues | ✓ VERIFIED | `FetchGeometry` in `geometry_fetch.go` iterates `PolySourceURLs`'s two-URL list (GitHub then Codeberg) in order, only advancing on fetch failure (not parse failure); `describePolyFetchFailure` interpolates the full URL list into the error text — confirmed both by reading the source and by the passing unit test `TestDescribePolyFetchFailure/message_names_both_upstream_hosts_that_were_tried`. `buildRegion`'s `ResolveGeometry` failure branch calls `setError` (not a bare `log.Printf`) and returns, leaving `buildRegionSafely`/`BuildAllLocked` (confirmed unmodified) to continue the loop (D-02). |
| 5 | A deleted or corrupted `region_geometry` row rebuilds on the next cron run without admin intervention | ✓ VERIFIED | `ResolveGeometry` (`geometry_store.go:53-63`) computes cache validity from **one** combined condition — lookup error OR `len(polygon)==0` OR `len(bboxSlice)!=4` — and every branch that fails this condition falls through to the same `FetchGeometry` call at line 68. There is exactly one `return` of cached values and exactly one fall-through to refetch, closing the "absent vs malformed" trap the phase targets. Verified by direct code read, not by trusting the SUMMARY's manual-test narrative. |
| 6 | Toggling a region on caches its boundary; hovering a disabled region shows the true outline with no write | ✓ VERIFIED (live end-to-end test performed by this verifier) | Started the actual server against a fresh migrated DB, created a superuser, and: (a) unauthenticated `GET /regions/abkhazia/geometry` → **401**; (b) authenticated fetch against a still-disabled leaf → **200** with real GeoJSON from a live CoMaps request, and `select count(*) from region_geometry` stayed **0** afterward (pass-through, no write); (c) `PATCH`'d `regions/abkhazia` to `enabled=true`, re-fetched the same endpoint → **200**, and `region_geometry` now held exactly **1** row for `abkhazia`. This directly closes the "known-pending" human-check items flagged for this phase (see Human Verification section). |

**Score:** 6/6 truths verified.

### Requirements Coverage (SLIM-01 through SLIM-05)

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| SLIM-01 | 32-01, 32-06 | Pure-hierarchy catalog, single `hierarchy.txt` fetch, no gzip, commit recorded once | ✓ SATISFIED | Catalog content/shape verified directly (see Truth 1). `.gz` confirmed absent from working tree and from `feature/app`'s reachable history (`git rev-list --objects --all --not backup/pre-seed-purge-20260728` returns 0 hits for the `.gz` path; local backup tag deliberately not pushed, confirmed absent from `git ls-remote --tags origin`). |
| SLIM-02 | 32-02 | Migration creates hierarchy-only `regions` + empty `region_geometry`, eliminates bulk geometry insert | ✓ SATISFIED | Migration source reviewed line-by-line: `region_geometry` created with `bbox`+`polygon`, `regions` has `catalog_commit` and no `bbox`; zero geometry insert code remains (`grep -c 'record.Set("bbox"'` inside the leaf-insert loop returns 0). Confirmed empirically: fresh migrate produces `region_geometry` with 0 rows. |
| SLIM-03 | 32-03, 32-04 | `buildRegion` self-heals absent/malformed rows via `ResolveGeometry`/`FetchGeometry`, GitHub-primary/Codeberg-fallback, failures name upstreams | ✓ SATISFIED | See Truths 4 and 5 above. |
| SLIM-04 | 32-02, 32-04, 32-05 | Fresh instance boots/migrates/serves catalog + admin picker with no network (except archive building) | ✓ SATISFIED | See Truth 2, plus the live-server test in Truth 6 confirming the admin picker's backend surface (`/regions/{id}/geometry`, the `region_geometry`/`regions` collections) all function against a freshly migrated, geometry-empty database. |
| SLIM-05 | 32-05 | Superuser-gated on-demand geometry endpoint, persists only when enabled | ✓ SATISFIED | Confirmed live: unauthenticated request → 401; the route is registered with `.Bind(apis.RequireSuperuserAuth())` standalone, outside the weaker `apis.RequireAuth()`-bound `regionsGroup` (`db/main.go:281`, `db/main.go:266-267`). Persist-on-enable / pass-through-on-disabled confirmed live in Truth 6. |

No orphaned requirements: REQUIREMENTS.md's checklist (lines 41-45) marks all five SLIM items `[x]`, and each is claimed by at least one plan's `requirements:` frontmatter (32-01 → SLIM-01; 32-02 → SLIM-02/04; 32-03 → SLIM-03; 32-04 → SLIM-03/04; 32-05 → SLIM-05/04; 32-06 → SLIM-01/04). **Note:** REQUIREMENTS.md's separate coverage table (lines 88-92) still lists all five SLIM rows as "Not started" — this is stale bookkeeping inconsistent with the checklist items above it, and should be corrected, but it is a documentation artifact, not evidence of incomplete implementation (see Info Notes below).

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `db/migrations/initial_data/regions_seed.json` | Pure-hierarchy catalog, 1306 rows | ✓ VERIFIED | 322,521 bytes, `{commit, rows}` shape, no geometry keys |
| `db/services/regions/poly_parser.go` | `ParsePoly` relocated into package `regions` | ✓ VERIFIED | `package regions`, exports `ParsePoly`; `db/commands/poly_parser.go` no longer exists |
| `db/services/regions/geometry_fetch.go` | Two-host fetcher, `FetchGeometry`/`PolySourceURLs` | ✓ VERIFIED | Both exported functions present with exact signatures; hardcoded hosts, no `os.Getenv`, no `httptest` |
| `db/services/regions/geometry_store.go` | `ResolveGeometry` self-heal + enable-scoped persist | ✓ VERIFIED | Single combined validity condition, `persist := region.GetBool("enabled")` derived internally, no `persist` parameter anywhere in the file |
| `db/services/regions/builder.go` | `buildRegion` sourcing bbox/polygon from `ResolveGeometry` | ✓ VERIFIED | `ResolveGeometry(app, record)` called before the `!needsVector && !needsDem` early return; error path calls `setError`; `region_polygons` references gone |
| `db/routes/regions_get.go` | Leaf bbox joined from `region_geometry`, `polygon` never selected | ✓ VERIFIED | Single `Select("path", "bbox").From("region_geometry")` query built once before the loop; bbox set on `entry` only when a 4-element array is found, otherwise omitted |
| `db/routes/regions_geometry_get.go` | Superuser-gated geometry endpoint | ✓ VERIFIED | `RegionGeometryGet` validates id, rejects non-leaf, calls `ResolveGeometry` unchanged, never reads `enabled`/query params |
| `db/migrations/1785000000_create_regions_collection.go` | Hierarchy-only `regions` + empty `region_geometry` | ✓ VERIFIED | Confirmed via fresh `migrate up` producing the exact expected shape |
| `db/routes/regions_ext/regions_ui.html` | Only `onLeafHoverStart` repointed; others renamed | ✓ VERIFIED | `region_polygons` count 0, `region_geometry` count 2 (`loadEnabledPolygons`, `addPolygonForRow`), `.../geometry` fetch appears exactly once in `onLeafHoverStart`; `fitToEnabled` reads bbox from the `region_geometry` items `loadEnabledPolygons` fetched, with a 4-element-array guard |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `seed_regions.go` | `hierarchy.txt` | single `fetch()` call | ✓ WIRED | No `.poly` URL construction anywhere in `db/commands/` |
| `1785000000_create_regions_collection.go` | `regions_seed.json` | `os.ReadFile` + `json.Unmarshal` | ✓ WIRED | Confirmed live: fresh migrate reads the plain-JSON path with zero network |
| `main.go` | `routes.RegionGeometryGet` | standalone route, `RequireSuperuserAuth()` | ✓ WIRED | Confirmed live: 401 unauthenticated, 200 with superuser JWT |
| `regions_geometry_get.go` | `regions.ResolveGeometry` | direct call, no persistence hint | ✓ WIRED | Confirmed live: disabled region → no write; enabled region → one row written |
| `builder.go` | `ResolveGeometry` | replaces both the old bbox read and `region_polygons` lookup | ✓ WIRED | Confirmed by code read; `region_polygons` string absent from `builder.go` |
| `regions_ui.html onLeafHoverStart` | `/regions/{path}/geometry` | `apiFetch` | ✓ WIRED | Confirmed by grep and by the endpoint's live behavior matching the hover-preview contract (pass-through, no write) |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|---|---|---|---|---|
| `RegionsList` (`regions_get.go`) | `leafBboxes` map | `region_geometry` table via projected SQL query | Yes — confirmed empty on a geometry-empty fresh DB (correct, per D-12 consequence 1/2), and populated after a live geometry fetch persisted a row | ✓ FLOWING |
| `regions_geometry_get.go` response body | `polygon`/`bbox` | `ResolveGeometry` → `FetchGeometry` → live CoMaps HTTP fetch | Yes — confirmed with a real network round-trip returning real Abkhazia boundary GeoJSON | ✓ FLOWING |
| Admin SPA hover outline | `data.polygon` | `/regions/{path}/geometry` response | Yes — same live-verified endpoint | ✓ FLOWING |

### Behavioral Spot-Checks / Probe Execution

Not a probe-script-based phase (no `scripts/*/tests/probe-*.sh` declared or discovered). Behavioral verification was instead performed via a full live-server session (see below), which exceeds the spot-check bar for a phase of this shape.

| Behavior | Command | Result | Status |
|---|---|---|---|
| `go build`/`go vet`/`go test` across the whole module | `cd db && go build ./... && go vet ./... && go test ./...` | All exit 0; `services/regions` and `commands` package tests pass, including the new `TestPolySourceURLs`/`TestDescribePolyFetchFailure` suites | ✓ PASS |
| Fresh, network-free `migrate up` | `go run . migrate up --dir <new-empty-dir>` | 1306/153/1153 rows, `region_geometry` = 0, single `catalog_commit`, no `bbox` column, no `region_polygons` table | ✓ PASS |
| Unauthenticated geometry fetch | `curl /regions/abkhazia/geometry` (no auth header) | HTTP 401 | ✓ PASS |
| Superuser geometry fetch, disabled region | `curl /regions/abkhazia/geometry` (superuser JWT) | HTTP 200, real GeoJSON from live CoMaps fetch; `region_geometry` row count unchanged (0) afterward | ✓ PASS |
| Enable region, re-fetch geometry | `PATCH regions/{id} {enabled:true}` then re-`curl` the geometry endpoint | HTTP 200; `region_geometry` now holds exactly 1 row for `abkhazia` | ✓ PASS |
| History purge completeness | `git rev-list --objects --all --not backup/pre-seed-purge-20260728 \| grep regions_seed.json.gz` | 0 matches (blob only reachable via the local, unpushed backup tag) | ✓ PASS |
| Force-push landed | `git rev-parse feature/app origin/feature/app` | Identical SHAs | ✓ PASS |

### Anti-Patterns Found

None. Scanned all 13 files touched across the six plans (`seed_regions.go`, `poly_parser.go`, `geometry_fetch.go`, `geometry_store.go`, `builder.go`, `staleness.go`, `regions_get.go`, `regions_geometry_get.go`, the migration, `main.go`, `hooks/regions.go`, `regions_ui.go`, `regions_ui.html`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` — zero matches. No stub return patterns, no hardcoded empty responses feeding a UI, no disconnected props found.

### Info Notes (non-blocking)

1. **Catalog size is 315.0 KB, not the ~292 KB the ROADMAP/CONTEXT estimated.** Both figures are far below the old 54.65 MB gzipped artifact and within the plan's own explicit acceptance band (250–350 KB), so this is not a defect — it's a planning-time estimate vs. the real 1306-row catalog's actual size once regenerated with the final field set. No action needed.
2. **REQUIREMENTS.md's coverage table (lines 88-92) still reads "Not started" for all five SLIM requirements**, even though the checklist above it (lines 41-45) marks them `[x]` and the phase is functionally complete per every check in this report. This is stale bookkeeping that should be corrected as part of phase closure, not a code gap.
3. **STATE.md still says "Phase 32 execution started" / "current focus"** — also stale bookkeeping expected to be updated by the phase-completion workflow following this verification.
4. **D-07's empirical equivalence sample cannot be independently re-run today** because its oracle (`regions_seed.json.gz`) was correctly purged from history in 32-06, after 32-03 had already consumed it. This is the designed sequencing (32-06 depends on 32-03 having finished with the oracle), not an oversight, but it does mean this verifier could not re-execute that specific empirical check and is relying on the previously-recorded per-region match table in 32-03-SUMMARY.md for that one item. Everything else in Truth 3 (the code path routing bbox/polygon exclusively through the shared `ParsePoly`) was independently verified by reading the source.

### Human Verification Required

None outstanding. The task brief flagged plan 32-05's `<human-check>` items (live hover row-count delta, toggle-on caching, unauthenticated curl returning 401/403) as not yet exercised because no dev server had been run. This verifier stood up a genuine fresh instance (Docker Meilisearch + `migrate up` + `serve` + a live superuser) and exercised all three directly:

- Unauthenticated `curl` against `/regions/{path}/geometry` → 401 (not 200, not geometry)
- Fetching geometry for a **disabled** leaf → 200 with real boundary data, zero rows written to `region_geometry`
- Enabling that leaf then re-fetching → 200, exactly one row now present in `region_geometry`

The one remaining item — visually confirming the admin picker's map renders the hover outline and fits bounds correctly in a real browser — was not exercised (no browser/Playwright session was run against the admin SPA), but the backend contract every visual behavior depends on (the endpoint's auth gate, response shape, and persistence rule) is now proven live rather than merely asserted. If the developer wants pixel-level confirmation of the map rendering itself, that remains a genuinely visual check outside what a backend-focused verifier can close.

### Gaps Summary

No gaps. All six ROADMAP success criteria and all five SLIM requirements are satisfied by evidence gathered directly from the codebase and from live execution against a freshly migrated instance — not from SUMMARY.md narrative. `go build`/`go vet`/`go test` are clean, the catalog is genuinely geometry-free, the self-heal path structurally cannot skip the malformed-row case, persistence is structurally incapable of being caller-influenced, the superuser gate is enforced and independently confirmed with a live 401, and the ~55 MB historical blob is confirmed absent from `feature/app`'s reachable, pushed history.

---

*Verified: 2026-07-28T10:54:58Z*
*Verifier: Claude (gsd-verifier)*
