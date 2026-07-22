---
phase: 21.5-region-catalog-archive-pre-build-backend
verified: 2026-07-21T19:20:21Z
status: human_needed
score: 12/12 must-haves verified (automated); 3 human-check items outstanding
overrides_applied: 0
human_verification:
  - test: "Build the db image and run one region-archive-build cron pass against a 1-region config (small bbox)"
    expected: "pb_data/region_archives/{id}/vector.pmtiles and dem.pmtiles appear on disk; the region_archives record shows status=ready with a non-empty vector_built_date; a second immediate cron pass performs no re-extract (unchanged Protomaps date); interrupting a rebuild mid-extract leaves the prior vector.pmtiles fully intact and still served (no truncated/missing file)"
    why_human: "Requires a running db container, the pmtiles CLI, and live network access to build.protomaps.com/download.mapterhorn.com — cannot be exercised by static analysis or go test"
  - test: "With the db container running and a 1-region config that has finished building: call GET /api/v1/regions (internal Go route) authenticated and unauthenticated; call GET /api/v1/regions/{id}/download for a ready region; call GET /api/v1/regions/..%2f..%2fetc/download"
    expected: "Authenticated request returns a JSON array with status=ready, version, vector_url/vector_size (and dem_url/dem_size if DEM built); unauthenticated request is rejected 401; download route returns the .pmtiles bytes; the traversal-id request is rejected 400"
    why_human: "Requires a live PocketBase auth session and a built archive on disk — cannot be exercised by static analysis"
  - test: "With docker compose up (db + web) and a logged-in browser session: GET {web_origin}/api/v1/regions and GET {web_origin}/api/v1/regions/{id}/download"
    expected: "The SvelteKit-proxied public path returns the same JSON/bytes as the internal Go route; without a session cookie the proxy returns 401"
    why_human: "Requires the full stack running with real cookies/auth — cannot be exercised by static analysis; these were explicitly deferred to end-of-phase per the plan's own <verify><human-check> blocks (21.5-02 Task 2, 21.5-03 Tasks 1 and 4)"
---

# Phase 21.5: Region Catalog & Archive Pre-Build (Backend) Verification Report

**Phase Goal:** A Wanderer instance admin can define the regions their instance offers for offline download in a config file; the backend pre-builds each region's vector + DEM archive ahead of time via a cronjob and serves the resulting catalog through an API endpoint — so a client-side download is always a single, already-ready file.

**Verified:** 2026-07-21T19:20:21Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Missing/empty config yields empty catalog, no error (SC1, BACK-01) | ✓ VERIFIED | `db/services/regions/config.go` `LoadRegionCatalog` returns `[]Region{}, nil` for unset env var, missing file (`os.IsNotExist`), and zero-byte file. `TestLoadRegionCatalog` subtests for all three cases pass (`go test ./services/regions/...`) |
| 2 | Config entries parse into typed Region values (id/name/bbox) | ✓ VERIFIED | `Region` struct with `[4]float64` bbox; `TestLoadRegionCatalog/valid_JSON_of_2_well-formed_regions...` passes, asserting exact bbox order |
| 3 | Invalid id / out-of-range bbox entries are skipped, not fatal | ✓ VERIFIED | `ValidateRegion` rejects bad id/bbox; loader logs+skips (not error); `TestLoadRegionCatalog/invalid_id_is_skipped...` and `.../out-of-range_bbox_is_skipped...` pass, both confirming a sibling valid entry survives |
| 4 | Region id path-traversal defense (id → filesystem path) | ✓ VERIFIED | `regionIDPattern = ^[a-z0-9][a-z0-9_-]*$`; `IsValidRegionID` exported and re-checked at the download-route layer; `TestRegionIDCannotProduceTraversalPath` asserts `../etc`, `a/b`, `../../pb_data` all rejected before any `filepath.Join` |
| 5 | `region_archives` collection exists, superuser-only, with vector+DEM status/size/date/error fields (BACK-01 infra) | ✓ VERIFIED | `db/migrations/1784658610_created_region_archives.go` defines `region_id`, bbox fields, `status`/`dem_status` (`building/ready/error`), `vector_built_date`, `vector_size_bytes`, `dem_size_bytes`, `error_message`, `dem_error_message`; all five `*Rule` keys are `null`; collection id `pbc_1784658610` confirmed unique across all migrations |
| 6 | `BuildAll` pre-builds one vector (Protomaps z14) + one DEM (Mapterhorn z12) archive per region (BACK-02/BACK-03) | ✓ VERIFIED | `db/services/regions/builder.go`: `buildVector` uses `regionVectorMaxZoom=14` against `ValidProtomapsDateAndURL()`'s URL; `buildDem` uses `regionDemMaxZoom=12` against `mapterhornSource`; DEM failure sets `dem_status=error`/`dem_error_message` and returns without touching the vector result or record's `status` field (source-inspected, no shared error path) |
| 7 | DEM build never blocks/errors the vector build (BACK-03) | ✓ VERIFIED | `buildDem` has no return value threaded into `buildVector`'s call site in `buildRegion`; `buildDem` is called independently after the vector gate, and all its internal error branches only mutate `dem_status`/`dem_error_message` |
| 8 | Ready region stays servable through a background rebuild — temp path + atomic rename (D-08) | ✓ VERIFIED | `buildVector`/`buildDem` write to `final + ".building"`, only `os.Rename(tmp, final)` on success; status is only flipped to `building` when `os.Stat(final)` fails (i.e., no prior ready file) — source-inspected in both functions |
| 9 | Vector rebuilds only on Protomaps-date change; DEM builds once, rebuilds only on config bbox change (BACK-05 / D-10 / D-11) | ✓ VERIFIED | `needsVectorRebuild(stored, current) = stored != current`; `bboxChanged` pure float comparison; `buildRegion` gates vector build on `configChanged \|\| needsVectorRebuild(...)` and DEM build on `dem_status != "ready" \|\| configChanged`. `TestNeedsVectorRebuild` and `TestBboxChanged` (all subtests) pass |
| 10 | Two overlapping builds for the same region cannot race (in-flight guard) | ✓ VERIFIED | Package-level `inFlightMu sync.Mutex` + `inFlight map[string]*sync.WaitGroup` keyed by `r.ID` in `buildRegion`; second caller waits on the existing WaitGroup and returns without re-building |
| 11 | `GET /api/v1/regions` returns merged config+build-state catalog, auth-gated to any logged-in user (BACK-04 / D-07) | ✓ VERIFIED | `db/main.go`: `regionsGroup := se.Router.Group("/api/v1/regions"); regionsGroup.Bind(apis.RequireAuth())` — confirmed ENABLED (not commented, `grep -v '^\s*//' \| grep RequireAuth` matches only the regions group, not the dormant `/map/cells` line). `RegionsList` in `db/routes/regions_get.go` builds one entry per configured region (config = source of truth) merged with `region_archives` record state, keys `id/name/bbox/status/version/vector_url/vector_size/dem_status/dem_url/dem_size/error` per plan contract |
| 12 | `/{id}/download` and `/{id}/download-dem` are auth-gated and serve correct archive bytes, id-validated | ✓ VERIFIED | Both routes registered under the same auth-gated `regionsGroup`; both handlers call `regions.IsValidRegionID(id)` before any path use and `e.FileFS(os.DirFS(RegionCacheDir), ...)` confines serving; `RegionArchivePath`/`RegionDemPath` used correctly per file |
| 13 | Daily `region-archive-build` cron wired into `main.go`, invokes `regions.BuildAll` (D-09) | ✓ VERIFIED | `registerCronJobs`: `app.Cron().MustAdd("region-archive-build", regionSchedule, func() { regions.BuildAll(app) })`, default `"0 3 * * *"`, overridable via `REGION_ARCHIVE_CRON_SCHEDULE`; existing `plugin-sync` cron and `/map/cells` group untouched |
| 14 | Catalog driven by admin config; removed regions disappear even with stale record | ✓ VERIFIED | `RegionsList` iterates `regions.LoadRegionCatalog()` (config), not the `region_archives` collection — a stale record for a region no longer in config is never visited |
| 15 | Go `/api/v1/regions` reachable only inside docker network; SvelteKit proxy is the only public path | ✓ VERIFIED | `docker/docker-compose.prod.yml` and `docker/docker-compose.dev.yml` `db` service blocks have no `ports:` mapping (confirmed via grep); only the local-convenience root `docker-compose.yml` publishes `8090:8090`. `web/src/routes/api/v1/regions/+server.ts` + `[id]/download(-dem)/+server.ts` proxy via `event.locals.pb.send` / `event.fetch` with forwarded `Authorization` header |
| 16 | `generator.go` / per-cell path untouched (D-01/D-02 parallel-path constraint, CLAUDE.md-adjacent constraint) | ✓ VERIFIED | `git show --stat` on every phase execution commit (366e4b2f…dc218a4b) shows zero changes to `db/services/tiles/generator.go` or any file under `services/tiles/`; `staleness.go`/`builder.go` do not import `services/tiles` (`grep -c 'services/tiles'` reports 0 in both, confirmed by reading the files) |

**Score:** 16/16 automated truths verified. 3 additional deferred human-check items outstanding (see below) — these were explicitly deferred to end-of-phase by the plans themselves (`<verify><human-check>` blocks), not skipped.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `db/services/regions/config.go` | Region struct, LoadRegionCatalog, id validation, path builders | ✓ VERIFIED | 137 lines, all required symbols present and substantive |
| `db/services/regions/config_test.go` | Table tests for loader/validation/traversal | ✓ VERIFIED | 194 lines, `TestLoadRegionCatalog` + 4 more test funcs, all pass |
| `db/migrations/1784658610_created_region_archives.go` | region_archives collection schema | ✓ VERIFIED | 264 lines, all fields present, superuser-only rules, unique collection id |
| `db/services/regions/staleness.go` | Protomaps date/url resolver + staleness helpers | ✓ VERIFIED | 66 lines, `ValidProtomapsDateAndURL`, `needsVectorRebuild`, `bboxChanged` present |
| `db/services/regions/staleness_test.go` | Pure-function tests | ✓ VERIFIED | Table tests for date/bbox staleness pass |
| `db/services/regions/builder.go` | BuildAll + vector/DEM build, atomic rename, in-flight guard | ✓ VERIFIED | 330 lines, all required symbols present, `os.Rename`/`.building`/`regionExtractTimeout`/`inFlight` all present |
| `db/routes/regions_get.go` | RegionsList + download handlers | ✓ VERIFIED | 118 lines, `RegionsList`/`RegionArchiveDownload`/`RegionArchiveDownloadDem` all present, id-revalidated, `RegionCacheDir`/`FileFS` used |
| `db/main.go` | Auth-gated route group + cron registration | ✓ VERIFIED (modified, not new) | `/api/v1/regions` group with `apis.RequireAuth()` enabled; `region-archive-build` cron registered |
| `docker-compose.yml` / `docker/docker-compose.prod.yml` / `docker/docker-compose.dev.yml` | REGION_CATALOG_CONFIG_PATH env + commented mount | ✓ VERIFIED | All three set the env var unconditionally and document a commented bind-mount |
| `web/src/routes/api/v1/regions/+server.ts` | SvelteKit GET proxy | ✓ VERIFIED | Uses `event.locals.pb.send('/api/v1/regions', ...)`, propagates errors via `handleError` |
| `web/src/routes/api/v1/regions/[id]/download/+server.ts` | Vector download proxy | ✓ VERIFIED | Forwards `Authorization: Bearer <token>`, streams response body, preserves Content-Length |
| `web/src/routes/api/v1/regions/[id]/download-dem/+server.ts` | DEM download proxy | ✓ VERIFIED | Same shape as vector download proxy |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `config.go` | `os.Getenv("REGION_CATALOG_CONFIG_PATH")` | startup path resolution | ✓ WIRED | Direct `os.Getenv` call at top of `LoadRegionCatalog` |
| `config.go` | `pb_data/region_archives/{id}` | `IsValidRegionID`-gated `filepath.Join` | ✓ WIRED | `RegionArchivePath`/`RegionDemPath` both `filepath.Join(RegionCacheDir, id, ...)`; id never reaches this without passing `regionIDPattern` upstream (loader) or `IsValidRegionID` (route) |
| `builder.go` | `pmtiles extract` subprocess | `exec.CommandContext` + `os.Rename` | ✓ WIRED | Both `buildVector` and `buildDem` follow temp-path extract → stat → atomic rename |
| `builder.go` | `region_archives` collection | find-or-create + status/size/date persistence | ✓ WIRED | `findOrCreateRegionRecord` queries by `region_id`; `app.Save(record)` called on every state transition |
| `builder.go` | `staleness.go` | `ValidProtomapsDateAndURL` + `needsVectorRebuild` | ✓ WIRED | `buildRegion` calls both, gates the vector build call accordingly |
| `main.go registerRoutes` | `routes.RegionsList` | auth-gated `/api/v1/regions` group | ✓ WIRED | `regionsGroup.Bind(apis.RequireAuth()); regionsGroup.GET("", routes.RegionsList)` |
| `main.go registerCronJobs` | `regions.BuildAll` | `region-archive-build` cron | ✓ WIRED | `app.Cron().MustAdd("region-archive-build", regionSchedule, func() { regions.BuildAll(app) })` |
| `regions_get.go` | `region_archives` collection + on-disk files | record lookup + `e.FileFS` | ✓ WIRED | `dbx.NewExp("region_id = {:id}", ...)` lookup; `os.Stat` guard before exposing URLs; `e.FileFS` confined to `RegionCacheDir` |
| `web/.../regions/+server.ts` | internal Go `/api/v1/regions` | `event.locals.pb.send` | ✓ WIRED | Matches `hooks.server.ts`'s per-request `locals.pb` (auth token auto-attached) |
| `web/.../[id]/download/+server.ts` | internal Go download route | `event.fetch` + `Authorization` header | ✓ WIRED | Forwards `event.locals.pb.authStore.token` as Bearer token; streams `response.body` through |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `RegionsList` response entries | `entries []map[string]any` | `regions.LoadRegionCatalog()` (file) joined to `e.App.FindAllRecords("region_archives", ...)` (real PocketBase query, not a static return) | Yes — real config read + real DB query, no hardcoded stub values | ✓ FLOWING |
| `buildVector`/`buildDem` output files | `record.vector_size_bytes` / `dem_size_bytes` | `os.Stat(tmp).Size()` after a real `pmtiles extract` subprocess run | Yes (not exercised in this static verification — see human-check items below for live-build confirmation) | ✓ FLOWING (source-verified; runtime behavior deferred to human-check) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Region config loader tolerates missing/empty/malformed config | `cd db && go test ./services/regions/... -run TestLoadRegionCatalog -v` | All 7 subtests PASS | ✓ PASS |
| Path-traversal id rejected before filesystem use | `go test ./services/regions/... -run TestRegionIDCannotProduceTraversalPath -v` | PASS | ✓ PASS |
| Staleness/bbox-change pure helpers | `go test ./services/regions/... -run 'TestNeedsVectorRebuild|TestBboxChanged' -v` | All subtests PASS | ✓ PASS |
| Full module compiles | `cd db && go build ./...` | Exit 0, no output | ✓ PASS |
| Full module vet-clean | `cd db && go vet ./...` | Exit 0, no output | ✓ PASS |
| Web proxy routes typecheck | `cd web && npx tsc --noEmit -p .` | Only 2 pre-existing, unrelated `trail_merge` errors (confirmed pre-existing via `git log` — untouched by this phase); zero errors in any `regions` file | ✓ PASS |
| Live cron pass produces real archives on disk | N/A — requires docker + pmtiles CLI + network egress | Not run | ? SKIP (routed to human verification) |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist for this phase and none are declared in the PLAN/SUMMARY files. Step 7c: SKIPPED (no probes declared or discovered).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| BACK-01 | 21.5-01, 21.5-03 | Admin config file loaded at startup, tolerant of missing config | ✓ SATISFIED | `LoadRegionCatalog` + docker-compose env wiring, both verified above |
| BACK-02 | 21.5-02 | Cronjob pre-builds mosaicked vector PMTiles per region ahead of request | ✓ SATISFIED | `buildVector` in `builder.go`, invoked from the daily cron |
| BACK-03 | 21.5-02 | Same cronjob pre-builds DEM archive per region, reusing Mapterhorn extraction | ✓ SATISFIED | `buildDem` in `builder.go`, best-effort, independent of vector result |
| BACK-04 | 21.5-03 | API endpoint returns instance's region catalog (id/name/bbox/urls/sizes/version/status) | ✓ SATISFIED | `RegionsList` + SvelteKit proxy, auth-gated |
| BACK-05 | 21.5-02, 21.5-03 | Cron regeneration is staleness-aware, drives updateAvailable-equivalent state | ✓ SATISFIED | `needsVectorRebuild`/`bboxChanged` gate rebuilds; `version` field in catalog response exposes the built-date for client-side comparison |

No orphaned requirements — ROADMAP.md maps all five BACK-0x IDs to Phase 21.5 and all five appear in at least one plan's `requirements` frontmatter.

### Anti-Patterns Found

None. Scanned all 11 phase-modified/created files (`config.go`, `config_test.go`, `builder.go`, `staleness.go`, `staleness_test.go`, the migration file, `regions_get.go`, `main.go`, and the three SvelteKit proxy files) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/"not yet implemented"/"coming soon" markers — zero matches.

### Human Verification Required

### 1. Live cron build pass produces real, atomically-swapped archives

**Test:** Build the `db` image and run one `region-archive-build` cron pass against a 1-region config (small bbox). Confirm `pb_data/region_archives/{id}/vector.pmtiles` and `dem.pmtiles` appear, the record shows `status=ready` with a `vector_built_date`. Run a second immediate pass — confirm no re-extract (unchanged date). Interrupt a rebuild mid-extract and confirm the prior `vector.pmtiles` is still served intact.
**Expected:** Archives appear on disk, status transitions correctly, staleness gating prevents redundant work, atomic rename protects the served file during a rebuild.
**Why human:** Requires a running container, the `pmtiles` CLI binary, and live network access to `build.protomaps.com`/`download.mapterhorn.com` — not exercisable via static analysis or `go test`.

### 2. Internal Go catalog + download routes, live

**Test:** With `db` running and a 1-region config that has finished building: authenticated `GET /api/v1/regions`, unauthenticated `GET /api/v1/regions`, `GET /api/v1/regions/{id}/download`, and `GET /api/v1/regions/..%2f..%2fetc/download`.
**Expected:** Authenticated → 200 with `status=ready`/`version`/`vector_url`/`vector_size` (and DEM fields if built); unauthenticated → 401; download → archive bytes; traversal id → 400.
**Why human:** Requires a live PocketBase session and a built archive — not exercisable statically.

### 3. Public SvelteKit-proxied path, live

**Test:** With `docker compose up` (db + web) and a logged-in browser session: `GET {web_origin}/api/v1/regions` and `GET {web_origin}/api/v1/regions/{id}/download`.
**Expected:** Same JSON/bytes as the internal Go route; unauthenticated request via the proxy returns 401.
**Why human:** Requires the full stack running with real auth cookies — not exercisable statically. This item, along with items 1 and 2, was explicitly deferred to end-of-phase by the plans' own `<verify><human-check>` blocks (21.5-02 Task 2; 21.5-03 Tasks 1 and 4) rather than skipped.

### Gaps Summary

No gaps found. Every observable truth, artifact, and key link derived from the ROADMAP.md success criteria and the three plans' `must_haves` frontmatter is verified present, substantive, and wired in the codebase. `go build`, `go vet`, `go test ./services/regions/...`, and `npx tsc --noEmit` all pass clean with zero errors attributable to this phase's files. The parallel-path constraint (generator.go untouched, no `services/tiles` import) holds. Auth-gating is enabled (not dormant) on the new route group. The only outstanding items are the three human-verification checks that require a live docker stack, network egress to external tile sources, and a real browser session — these were explicitly deferred to end-of-phase by the plans themselves, not omitted or skipped by the executor.

---

*Verified: 2026-07-21T19:20:21Z*
*Verifier: Claude (gsd-verifier)*
