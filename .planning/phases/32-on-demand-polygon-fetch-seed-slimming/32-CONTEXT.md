# Phase 32: On-Demand Polygon Fetch & Seed Slimming - Context

**Gathered:** 2026-07-28
**Status:** Ready for planning (blocker resolved 2026-07-28 via `/gsd-explore`)

<domain>
## Phase Boundary

Geometry stops being a distributed artifact and becomes on-demand, fetched from CoMaps at the moment intent is expressed. This phase delivers:

- `db/commands/seed_regions.go` writes a **pure-hierarchy** catalog — no `bbox`, no `polygon` — as plain, pretty-printed JSON with no gzip layer (291.9 KB measured, down from 54.65 MB), with the CoMaps commit SHA it fetched from recorded inside the artifact. It fetches **only `hierarchy.txt`**: one HTTP request, down from ~1153.
- `db/migrations/1785000000_create_regions_collection.go` creates `regions` (hierarchy only) plus an **empty** `region_geometry` collection — renamed from `region_polygons`, now holding both `bbox` and `polygon`. No bulk insert; the streaming decoder and decompression-bomb guard both come out.
- A new backend endpoint fetches a leaf's `.poly` from CoMaps (GitHub primary, Codeberg fallback), converts it via `ParsePoly`, and returns GeoJSON + bbox. It **persists to `region_geometry` only when that region is currently enabled**.
- `db/services/regions/builder.go`'s `buildRegion` reads `region_geometry`, fetching and persisting on miss.
- The ~55 MB blob is purged from `feature/app` git history (folded todo, see below).

Out of scope: any change to what the catalog *contains* (the 1306-row hierarchy is unchanged), the Flutter hierarchy, and the archive-extraction pipeline itself. `ParsePoly` is reused verbatim.

</domain>

<decisions>
## Implementation Decisions

### Locked before this discussion (from `/gsd-explore`, same day — do not re-litigate)

- **D-00a:** Seed artifact is plain, pretty-printed JSON. No gzip. Measured on the real 1306-row catalog: with bbox, pretty-printed 386.9 KB; **without bbox (the shipping shape, per D-12), 291.9 KB**. Pretty-printed was chosen deliberately for readability, accepting that it expands a single renamed region into ~10 diff lines where one-object-per-line would show exactly one.
- **D-00b:** The pinned CoMaps SHA travels **inside the catalog artifact**, not as a shared Go const. A const goes stale the moment someone regenerates with `--commit X`, silently desyncing geometry from hierarchy; storing what `seed-regions` actually used makes that desync structurally impossible.
- **D-00c:** ⚠️ **SUPERSEDED by D-09.** As originally recorded: "`region_polygons` is dropped entirely — not retained as a lazily-populated cache." That reasoning rested on a `*.go`-only grep that identified `buildRegion` as the sole geometry consumer. It is false — the admin SPA reads the collection over the PocketBase REST API from JavaScript. The table survives, empty. See D-09.
- **D-00d:** ⚠️ **SUPERSEDED by D-12.** As originally recorded: "`bbox` stays committed." Verified false on re-check: every consumer needs bbox only for *enabled* regions. bbox moves into `region_geometry`.
- **D-00e:** ⚠️ **SUPERSEDED by D-12.** As originally recorded: "`seed-regions` still fetches all ~1153 `.poly` files." With bbox out of the catalog, the generator needs only `hierarchy.txt` — one request.

### Geometry storage and retrieval (resolves the former blocker)

- **D-09:** `region_geometry` (renamed from `region_polygons`) **stays in the schema but ships empty.** It is never seeded. This preserves the admin SPA's three shipped flows while still solving every original motivation for the phase — repo weight, push friction, image size, migration runtime — because the win comes from the *seed* carrying no geometry, not from the table being absent.
  - The admin SPA reads geometry in three flows: `loadEnabledPolygons` ([regions_ui.html:1110](../../db/routes/regions_ext/regions_ui.html)) batch-loads enabled leaves for the coverage map; `addPolygonForRow` (line 1157) draws a region the instant it is toggled on; `onLeafHoverStart` (line 1183) previews **any** leaf on hover, **including disabled regions that have never been built**. The hover flow is why "populate when built" is not sufficient.
- **D-10:** A new backend endpoint (shape at planner's discretion, e.g. `GET /api/v1/regions/{path}/geometry`) fetches the leaf's `.poly` from CoMaps, converts via `ParsePoly`, and returns GeoJSON + bbox. **It persists to `region_geometry` only when that region is currently enabled.**
  - This single rule handles both UI flows with no client cooperation, no `?persist=` flag, and no new state: hovering a disabled region is a pass-through with no write; toggling a region on and drawing it writes, because by then the region *is* enabled. Persistence keys off actual catalog state rather than which gesture happened to call it.
  - **Why the client cannot fetch CoMaps directly:** CoMaps serves Osmosis-format `.poly`, not GeoJSON. Converting it is `ParsePoly` — Go, with the multi-ring hole/exclave support Phase 28 deliberately built (28-CONTEXT D-03/D-04). Doing it client-side means reimplementing that parser in JavaScript and widening the admin page's CSP to two more hosts. A backend endpoint is therefore required regardless of the persistence rule.
- **D-11:** **Hover on a disabled region is a pass-through with no write, and no server-side cache.** Rejected: persisting on hover. `region_geometry` has no eviction policy, so hover-writes would let an admin idly scrolling ~1153 leaves grow the table toward the same ~55 MB just removed from git — relocating the weight into `pb_data` rather than shedding it. Persist-on-enable is naturally bounded by deliberate admin intent; persist-on-hover is bounded by curiosity. The existing client-side `_polygonCache` in `onLeafHoverStart` already absorbs repeat hovers within a session, so a server cache buys little; a bounded in-memory LRU was offered and declined.
- **D-14:** **`buildRegion` self-heals before it errors.** The build-time resolution order is strictly:
  1. Read the row from `region_geometry`.
  2. If the row is **absent** *or* its geometry is **malformed/unusable**, refetch from CoMaps (GitHub → Codeberg) and re-persist, overwriting the bad row if one existed.
  3. Only if that refetch fails does D-01 apply — `setError` and continue to the next region.
  - The malformed case is called out deliberately. Today's `builder.go:238-241` treats "invalid or missing polygon" as a dead end (`log` + `return`), and a literal "fetch on cache miss" implementation would inherit that: a corrupt row *exists*, so there is no miss to trigger the refetch, and the region stays permanently broken with no self-healing path. Both conditions must route to the refetch.
  - A region reaching `buildRegion` is enabled by definition (`BuildAllLocked` only iterates enabled leaves), so the D-10 persistence rule is always satisfied here — the refetch always writes.
  - Rationale: geometry is now derived state with an authoritative upstream, so a bad or missing local copy is a cache fault, not a data loss. Erroring without re-deriving would strand a region until an admin noticed and manually re-toggled it.
- **D-13:** The endpoint **must require superuser auth.** It is otherwise an unauthenticated shape that triggers outbound third-party requests — both an open proxy and a way to burn the CoMaps rate limit from outside. The admin SPA already sends a superuser JWT via its `apiFetch` helper, so gating costs nothing.

### Catalog contents

- **D-12:** **`bbox` moves out of the committed catalog into `region_geometry`**, fetched together with the polygon. The catalog becomes pure hierarchy.
  - **Verified before deciding — every consumer needs bbox only for enabled regions:** `fitToEnabled` unions bbox over `enabledLeafRows` only (regions_ui.html:1136); `buildRegion:183` runs only for enabled leaves via `BuildAllLocked`; the Flutter guard reads the catalog fetched with `?enabled=true` (commit `407b767c`). The sole exception is `RegionsList` *without* `?enabled=true`, whose own comment scopes it to "the dev harness and any admin tooling."
  - **The payoff is the generator, not the 95 KB.** With no bbox to derive, `seed-regions` needs only `hierarchy.txt` — one request instead of ~1153. That deletes the Codeberg rate-limit problem (the entire reason GitHub was chosen as the generator's primary host, seed_regions.go:19-31), the 10×/`Retry-After` retry machinery in the generator, and `ParsePoly` from the generator's path. A maintainer run drops from multi-minute and retry-prone to about a second. It also makes CATALOG-F01 (automated refresh) genuinely tractable — fetch one text file, diff a 292 KB JSON, something CI could run.
  - **Three accepted consequences:**
    1. Unfiltered `/api/v1/regions` (without `?enabled=true`) returns disabled leaves with no bbox. The Flutter parser drops entries missing required bbox, so an older client in unfiltered mode silently drops disabled regions. Disabled regions aren't downloadable, so this is arguably correct — but it *is* a behavior change to a shipped endpoint.
    2. An enabled region whose geometry fetch failed has no bbox at all, where today it always had one. It is in `status: "error"` per D-01; the API needs a deliberate answer for what it emits in that state.
    3. `bboxChanged` staleness comparison moves source from the catalog record to the geometry fetch. Semantics unchanged — bbox can still only change when the pinned commit changes — but the code moves.

### Cron failure behavior

- **D-01:** *(sequenced by D-14 — `setError` is the last resort, only after a refetch attempt has failed)* When a polygon fetch fails for one region after both hosts have been tried, call the existing `setError` — sets `status: "error"` and `error_message`, which the admin UI already surfaces — then continue to the next region. This also fixes a latent wart: today's polygon-missing path at `builder.go:231-241` does a bare `log` + `return` without `setError`, leaving the archive record at the `"building"` status that `findOrCreateRegionRecord` assigned. That is nearly unreachable today (the migration inserted every polygon in one transaction) but becomes reachable on any network blip.
- **D-02:** A fetch failure must **never** abort the whole cron run. `BuildAllLocked`'s existing per-region isolation via `buildRegionSafely` is preserved — every reachable region still builds even when one upstream is flaky.
- **D-03:** The build path gets its own **tighter retry budget** (small number of attempts, short cap) rather than reusing `fetch`'s maintainer-tuned 10× retries with 30s `Retry-After` sleeps. At ~100 enabled regions, the existing budget lets a flaky upstream grind a nightly run for hours while building nothing. `seed-regions` keeps its patient budget — it needs to survive Codeberg's ~250-requests/600s quota across a ~1150-file run.

### Source configurability

- **D-04:** Both hosts are **hardcoded constants**. No env var, despite `NOMINATIM_URL` / `OVERPASS_API_URL` / `VALHALLA_URL` establishing an override convention for external services. Rationale: those point at *services* an admin might self-host, whereas an override here would have to replicate a git repo's `.../data/borders/{comaps_id}.poly` file-tree shape, and the pinned-SHA design makes the URLs stable. Explicitly considered and rejected; the follow-up question about replace-vs-prepend override semantics is therefore moot.

### Migration delivery

- **D-05:** **Edit `1785000000_create_regions_collection.go` in place.** No separate drop migration. Verified during discussion: this migration has never shipped — it is absent from `origin/main` and from every release tag (v1.6, v1.5, v0.20.0 all contain zero matching files), existing only on `feature/app`. There is no fleet of production instances carrying `region_polygons`. Editing in place also makes the `region_polygons` → `region_geometry` rename (D-09) free: no rename migration, the collection is simply created under its new name with `bbox` added.
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
- **CORRECTION — do not trust the earlier claim that `buildRegion` is the only reader of polygon geometry.** That came from a `*.go`-only grep and is false. There are **two** consumers: `buildRegion` (Go, at archive-build time) and the admin SPA (JavaScript, hitting `/api/collections/region_polygons/records` directly). Resolved by D-09/D-10.
- `db/routes/regions_ext/regions_ui.html` lines 1110 / 1157 / 1183 — the three admin-UI geometry flows. Under D-10, `loadEnabledPolygons` and `addPolygonForRow` can keep reading the collection (enabled regions always have rows); only `onLeafHoverStart` must be repointed at the new endpoint. Line 1136's `fitToEnabled` reads `r.bbox` off the enabled rows and is affected by D-12's bbox move.
- `db/main.go:160-161` registers `ValidateRegionPathReferenceHandler` on `region_archives` **and** `region_polygons`; the latter binding follows the rename to `region_geometry` without disturbing the former.
- `db/routes/regions_ui.go:35` — doc comment naming both collections as the admin page's privileged API surface; update for the rename and the new endpoint.
- `db/routes/regions_get.go` — `RegionsList`'s `filtering` flag (`?enabled=true`) and the bbox emission at lines 99-100; directly affected by D-12 consequence 1.
- `db/services/regions/staleness.go` `bboxChanged` — affected by D-12 consequence 3.

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
