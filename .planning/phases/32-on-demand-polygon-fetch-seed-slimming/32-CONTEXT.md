# Phase 32: On-Demand Polygon Fetch & Seed Slimming - Context

**Gathered:** 2026-07-28
**Status:** ⛔ BLOCKED — one open decision must be resolved before planning

<blocker>
## ⛔ Open Blocker — resolve before `/gsd-plan-phase 32`

**The admin UI reads `region_polygons` directly. Dropping the collection breaks shipped Phase 30 behavior.**

Discovered at the end of this discussion, after decisions D-00c and D-05 had already been recorded. `db/routes/regions_ext/regions_ui.html` calls the PocketBase collection REST API in three flows:

| Line | Flow | What it needs |
|---|---|---|
| 1110 | `loadEnabledPolygons(rows)` | Batch polygons for every **enabled** leaf — draws the coverage map (ADMINUI-03) |
| 1157 | `addPolygonForRow(row)` | One leaf's polygon the instant an admin toggles it on (ADMINUI-02/03) |
| 1183 | `onLeafHoverStart(row)` | Any leaf's polygon on hover — **including disabled regions that have never been built** |

The hover flow is decisive: it rules out any scheme that populates geometry only when a region is built or enabled, because it needs polygons for regions nobody has ever enabled.

**How this was missed:** the claim "`buildRegion` is the only reader of polygon geometry" came from a grep restricted to `*.go`. The admin UI is JavaScript hitting `/api/collections/region_polygons/records` directly, so it did not appear. That claim is **false** and is corrected in `<code_context>` below.

**What this does NOT change:** every original motivation for the phase — repo weight, GitHub push friction, Docker image size, migration runtime — is satisfied by the *seed* no longer carrying geometry. None of them require the collection to be absent. The seed-slimming core of this phase is unaffected.

**What it does change:** D-00c ("`region_polygons` is dropped entirely") is invalid as written, and D-05's "the collection is never created" clause is contingent on how this resolves.

**Options presented, decision deferred by the user on 2026-07-28:**

1. **Keep the table as a lazy cache** — stays in the schema, never seeded; rows written on first demand by `buildRegion` or by a small backend endpoint the admin UI calls on cache miss. Smallest deviation from shipped Phase 30 code.
2. **Backend proxy endpoint, no table** — drop the collection, add `GET /api/v1/regions/{path}/polygon` fetching from CoMaps server-side with a cache; rewire three JS call sites. Keeps geometry out of the database entirely.
3. **Admin map falls back to bbox** — drop polygons from the admin surface, draw rectangles from the catalog's `bbox`. No fetch, no table, no endpoint — but ADMINUI-03 specifies "boundary polygons," and a bbox is wildly misleading for Chile or Norway. Requires amending a shipped requirement.

</blocker>

<domain>
## Phase Boundary

Boundary geometry stops being a distributed artifact. This phase delivers:

- `db/commands/seed_regions.go` writes a **geometry-free** catalog — hierarchy fields plus leaf `bbox` only — as plain, pretty-printed JSON with no gzip layer (~387 KB measured, down from 54.65 MB), with the CoMaps commit SHA it fetched from recorded inside the artifact.
- `db/migrations/1785000000_create_regions_collection.go` no longer bulk-inserts geometry — the streaming decoder and the decompression-bomb guard both come out. **Whether `region_polygons` is still created (empty, as a lazy cache) or removed outright is the open blocker above.**
- `db/services/regions/builder.go`'s `buildRegion` fetches the one leaf `.poly` it needs at build time from the commit recorded in the catalog, converting it via the existing `ParsePoly`. GitHub mirror primary, CoMaps' canonical Codeberg repository as fallback.
- The ~55 MB blob is purged from `feature/app` git history (folded todo, see below).

Out of scope: any change to what the region catalog *contains* (the 1306-row hierarchy is unchanged), the admin picker, the Flutter hierarchy, and the archive-extraction pipeline itself. `ParsePoly` and the existing `fetch` helper are reused as-is, not rewritten.

</domain>

<decisions>
## Implementation Decisions

### Locked before this discussion (from `/gsd-explore`, same day — do not re-litigate)

- **D-00a:** Seed artifact is plain, pretty-printed JSON. No gzip. Measured on the real 1306-row catalog after stripping geometry: compact 281.8 KB, pretty-printed 386.9 KB, one-object-per-line 283.0 KB. Pretty-printed was chosen deliberately for readability, accepting that it expands a single renamed region into ~10 diff lines where one-object-per-line would show exactly one.
- **D-00b:** The pinned CoMaps SHA travels **inside the catalog artifact**, not as a shared Go const. A const goes stale the moment someone regenerates with `--commit X`, silently desyncing geometry from hierarchy; storing what `seed-regions` actually used makes that desync structurally impossible.
- **D-00c:** ⛔ **INVALIDATED — see `<blocker>` above.** As originally recorded: "`region_polygons` is dropped entirely — not retained as a lazily-populated cache. The cache would save a ~165 KB fetch inside a function that already downloads hundreds of MB to GB from Mapterhorn and Protomaps." That reasoning only accounted for `buildRegion`. The admin UI is a second consumer requiring on-demand geometry for arbitrary (including disabled) leaves, which the cost/benefit argument never weighed. Do not act on D-00c until the blocker is resolved.
- **D-00d:** `bbox` stays committed. Non-negotiable: `app/lib/util/trail_coverage_util.dart` runs the on-device trail-download-guard overlap math against a local catalog snapshot with no network.
- **D-00e:** `seed-regions` still fetches all ~1153 `.poly` files at maintainer time — `ParsePoly` is the only bbox source, so it computes the bbox and discards the geometry. The maintainer run's cost is unchanged; only its output shrinks.

### Cron failure behavior

- **D-01:** When a polygon fetch fails for one region after both hosts have been tried, call the existing `setError` — sets `status: "error"` and `error_message`, which the admin UI already surfaces — then continue to the next region. This also fixes a latent wart: today's polygon-missing path at `builder.go:231-241` does a bare `log` + `return` without `setError`, leaving the archive record at the `"building"` status that `findOrCreateRegionRecord` assigned. That is nearly unreachable today (the migration inserted every polygon in one transaction) but becomes reachable on any network blip.
- **D-02:** A fetch failure must **never** abort the whole cron run. `BuildAllLocked`'s existing per-region isolation via `buildRegionSafely` is preserved — every reachable region still builds even when one upstream is flaky.
- **D-03:** The build path gets its own **tighter retry budget** (small number of attempts, short cap) rather than reusing `fetch`'s maintainer-tuned 10× retries with 30s `Retry-After` sleeps. At ~100 enabled regions, the existing budget lets a flaky upstream grind a nightly run for hours while building nothing. `seed-regions` keeps its patient budget — it needs to survive Codeberg's ~250-requests/600s quota across a ~1150-file run.

### Source configurability

- **D-04:** Both hosts are **hardcoded constants**. No env var, despite `NOMINATIM_URL` / `OVERPASS_API_URL` / `VALHALLA_URL` establishing an override convention for external services. Rationale: those point at *services* an admin might self-host, whereas an override here would have to replicate a git repo's `.../data/borders/{comaps_id}.poly` file-tree shape, and the pinned-SHA design makes the URLs stable. Explicitly considered and rejected; the follow-up question about replace-vs-prepend override semantics is therefore moot.

### Migration delivery

- **D-05:** **Edit `1785000000_create_regions_collection.go` in place.** No separate drop migration. Verified during discussion: this migration has never shipped — it is absent from `origin/main` and from every release tag (v1.6, v1.5, v0.20.0 all contain zero matching files), existing only on `feature/app`. There is no fleet of production instances carrying `region_polygons`.
  - **Execution note:** the migration's idempotency guard is `CountRecords("regions") > 0 → return`, so any dev box that already seeded will **not** re-run it and will retain both a populated `regions` table and an orphaned `region_polygons` collection. Resetting local `pb_data` (or manually dropping the collection) is a required step on already-seeded dev instances. This was accepted knowingly over shipping a defensive drop migration.

### Test strategy

- **D-06:** **No automated coverage of the network layer.** `ParsePoly` and the fallback *decision* logic are covered by pure unit tests; HTTP fetching is verified manually. This was chosen with the cost stated: the GitHub→Codeberg fallback is the one genuinely new behavior in this phase and it ships without automated coverage. **Do not add `httptest` coverage on your own initiative** — this is a deliberate decision, not an oversight.
  - Clean consequence: because nothing needs to redirect the fetcher in tests, the base URLs can remain `const` with no injectable seam. D-04 and D-06 resolve each other.
- **D-07:** The equivalence bar is **polygon value-equality**, not byte-identical archives. Assert that the fetched-and-parsed GeoJSON deep-equals what the old seed held for that region, following the precedent set by commit `490a685f` (which proved the gzip artifact by deep value-equality against the 730 MB original rather than comparing downstream outputs). Byte-identical `.pmtiles` comparison was rejected as slow, requiring multi-gigabyte downloads per run, and possibly unachievable if pmtiles embeds generation metadata.
  - **⚠ Required roadmap correction:** ROADMAP.md's Phase 32 success criterion 3 currently reads "produces a byte-equivalent archive to today's." That wording is now superseded by D-07 and must be reworded before or during planning. It was left unedited because this workflow does not mutate ROADMAP.md.

### Claude's Discretion

- Exact retry counts and timeout caps for D-03's tighter build-path budget.
- The concrete field name and placement of the commit SHA inside the catalog artifact (D-00b fixes *that* it lives there, not what it's called).
- Whether the build-path fetch is a new function or a parameterized variant of the existing `fetch`.
- Error message wording, subject to the D-01/ROADMAP requirement that a failure names which upstreams were tried.
- Whether the slim catalog keeps the `regions_seed.json.gz` filename lineage or takes a new name (the `.gz` suffix must go regardless).

### Folded Todos

- **`.planning/todos/pending/2026-07-28-purge-regions-seed-blob-from-git-history.md`** (matched 0.9) — the ~55 MB `regions_seed.json.gz` sits in pushed history on `feature/app` as of 2026-07-28. Deleting the file does not shrink the repository; the blob persists in history and every clone keeps paying for it. Folded because the phase is not meaningfully complete until the blob is gone, and the timing window is exactly this phase: the purge needs a `filter-branch` plus a force-push, which is cheap while `feature/app` is effectively single-owner and expensive once it merges to `main`. Sequence: land the code change first, then purge. Note this is the one part of the phase that rewrites *published* history — unlike the 730 MB purge on 2026-07-28, which needed no force-push because those commits were unpushed.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### This phase's scope and rationale
- `.planning/ROADMAP.md` (Phase 32 section) — goal, success criteria, sequencing, and the full **Design notes for planning** block. Contains material not repeated here: why the gzip layer retires, which two migration-side guards retire with it, the disputed-territory `comaps_id` special case dissolving, and the Codeberg-as-fallback rationale. **Note success criterion 3 is superseded by D-07 above.**
- `.planning/REQUIREMENTS.md` — SLIM-01 through SLIM-04 map to this phase. Also records that this supersedes CATALOG-02 (leaf `polygon` storage; its `bbox` half is retained) and changes SEED-01/SEED-02.

### Prior phase decisions still binding
- `.planning/phases/28-region-catalog-data-model-seeding/28-CONTEXT.md` — D-01 (nothing raw is vendored; only flattened output is committed), D-02 (commit hash is a CLI flag with a baked-in default), D-03/D-04 (hand-rolled multi-ring `.poly` parser supporting holes and multi-part regions). All still hold; `ParsePoly` is reused unchanged.
- `.planning/notes/streamlined-region-definition.md` — the original design trail for the region catalog: table schema, group/leaf semantics, and the ODbL licensing/attribution treatment for CoMaps-derived data. Relevant because this phase changes *when* that data is fetched, not the attribution obligation.

### Source files this phase modifies
- `db/commands/seed_regions.go` — the generator. Note lines 19-31 (why GitHub over Codeberg for the maintainer run) and 64-69 (why gzip existed, including its now-obsolete "do not drop below level 6" warning).
- `db/migrations/1785000000_create_regions_collection.go` — edited in place per D-05.
- `db/services/regions/builder.go` — `buildRegion`, specifically the polygon lookup at lines 227-248 that becomes a fetch.
- `db/main.go` lines 156-161 — path-reference validation hooks currently wired for `region_polygons`; must be unwired.

### Related pending work
- `.planning/todos/pending/2026-07-28-purge-regions-seed-blob-from-git-history.md` — folded into scope, see above.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `db/commands/poly_parser.go` `ParsePoly` — returns `(map[string]any, [4]float64, error)`. The sole source of both geometry and bbox; reused verbatim by both the generator and the new build-path fetch.
- `db/commands/seed_regions.go` `fetch` / `doFetch` / `parseRetryAfter` — existing HTTP GET with 429 handling and `Retry-After` backoff. Reused in spirit by the build path but with a tighter budget per D-03.
- `db/services/regions/builder.go` `setError` — existing helper that sets `status: "error"` + `error_message`; D-01 reuses it rather than inventing a new failure signal.
- `db/services/regions/builder.go` `writePolygonTempFile` — already converts an in-memory polygon to the temp file `pmtiles extract --region` consumes. Unchanged; only its input source moves.

### Established Patterns
- **Per-region failure isolation:** `BuildAllLocked` loops leaf records through `buildRegionSafely`, which recovers from panics so one region cannot abort the run. D-02 preserves this.
- **Lazy polygon access:** `buildRegion` already defers polygon lookup until after the `!needsVector && !needsDem` early return, so a no-op build never touches geometry. The fetch inherits this position and inherits the property that unchanged regions cost zero network.
- **Go test conventions:** inline string fixtures, subtests via `t.Run`, no `testdata/` directories anywhere in `db/` (see `db/commands/poly_parser_test.go`). `httptest` has exactly one precedent, `db/pluginsystem/host_http_test.go` — not to be followed here, per D-06.
- **Migrations** are `m.Register(up, down)` in `db/migrations/*.go`, auto-run on startup, with idempotency guards.

### Integration Points
- **CORRECTION — do not trust the earlier claim that `buildRegion` is the only reader of polygon geometry.** That came from a `*.go`-only grep and is false. There are **two** consumers: `buildRegion` (Go, at archive-build time) and the admin SPA (JavaScript, hitting `/api/collections/region_polygons/records` directly). See `<blocker>`.
- `db/routes/regions_ext/regions_ui.html` lines 1110 / 1157 / 1183 — the three admin-UI polygon flows. Any resolution of the blocker must keep all three working or explicitly amend ADMINUI-02/ADMINUI-03.
- `db/main.go:160-161` registers `ValidateRegionPathReferenceHandler` on `region_archives` **and** `region_polygons`; if the collection goes, the latter binding must be removed without disturbing the former.
- `db/routes/regions_ui.go:35` — the doc comment naming both collections as the admin page's privileged API surface; keep it accurate to whatever the blocker resolves to.

</code_context>

<specifics>
## Specific Ideas

- The gzip removal is not a performance change and must not be planned as one. The entire speed win comes from dropping ~216 MB of polygon JSON parsing; whether the remaining ~387 KB arrives compressed is lost in the noise. Plain JSON is justified by making CATALOG-F01's stated refresh procedure ("manual re-run + **reviewed diff** + migration") actually true for the first time — today that diff is an opaque 55 MB binary blob change nobody can inspect.
- A 404 from either host is effectively impossible by construction: the catalog was *generated* from the pinned commit, so if a region is in the catalog, its `.poly` exists at that commit. Realistic failures are host availability and transport errors — transient by nature. Planning should not build elaborate permanent-failure handling.

</specifics>

<deferred>
## Deferred Ideas

- **One-object-per-line seed format** — measured at 283.0 KB vs pretty-printed's 386.9 KB, and would render a catalog refresh as exactly one changed diff line per added/removed/renamed region instead of ~10. Deliberately not chosen (D-00a). Revisit only if catalog-refresh diffs prove annoying to review in practice.
- **Env-configurable polygon source** — rejected in D-04. If a self-hoster behind restricted egress actually asks, this is a small, self-contained follow-up.
- **Automated fallback coverage** — D-06 ships the GitHub→Codeberg fallback untested. If the fallback ever misfires in the wild, adding `httptest` coverage (with base URLs promoted from `const` to package-level `var`) is the known remedy.

### Reviewed Todos (not folded)
- **`.planning/todos/pending/2026-07-24-comaps-poly-region-extraction-spike.md`** — matched at 0.9 on keywords but is stale. It was a pre-planning validation spike for Phase 29 (`resolves_phase: 29`), which has shipped and is Complete. Not folded; it should probably be moved to `completed/` independently of this phase.
- **`.planning/todos/pending/2026-07-18-way-types-and-surfaces-breakdown.md`** — matched at 0.6, mobile area, unrelated to region seeding.

</deferred>

---

*Phase: 32-on-demand-polygon-fetch-seed-slimming*
*Context gathered: 2026-07-28*
