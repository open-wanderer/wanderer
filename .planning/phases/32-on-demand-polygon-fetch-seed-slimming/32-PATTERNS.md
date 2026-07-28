# Phase 32: On-Demand Polygon Fetch & Seed Slimming - Pattern Map

**Mapped:** 2026-07-28
**Files analyzed:** 9 (6 modified, 2 new, 1 rename-affected)
**Analogs found:** 9 / 9

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `db/commands/seed_regions.go` | config/utility (Cobra CLI) | batch/file-I/O | itself (in-place edit) — supporting analog `db/commands/dedup.go` for Cobra shape | exact (self) |
| `db/migrations/1785000000_create_regions_collection.go` | migration | batch | itself (in-place edit) | exact (self) |
| `db/services/regions/builder.go` (`buildRegion`) | service | CRUD + event-driven (cron) | itself (in-place edit) | exact (self) |
| NEW: geometry fetch helper (Go, e.g. `db/services/regions/geometry.go`) | service/utility | request-response (outbound HTTP) + transform | `db/commands/seed_regions.go` `fetch`/`doFetch`/`parseRetryAfter` | role-match (fetch shape), needs relocation from `commands` to `services/regions` or a shared location |
| NEW: backend route (e.g. `db/routes/regions_geometry_get.go`) | route/controller | request-response | `db/routes/regions_get.go` (`RegionArchiveDownload`/`RegionArchiveDownloadDem`) for path-param validation + `db/main.go`'s `apis.RequireSuperuserAuth()` routes for auth wiring | exact (role) |
| `db/routes/regions_ext/regions_ui.html` (`onLeafHoverStart`) | component (embedded JS) | request-response | itself, lines 1101-1194 | exact (self) |
| `db/main.go` lines 156-161, 194-271 | route registration / hook wiring | event-driven (hooks) + request-response (routes) | itself | exact (self) |
| `db/routes/regions_get.go` (`RegionsList` bbox emission) | controller | request-response | itself | exact (self) |
| `db/services/regions/staleness.go` (`bboxChanged`) | utility | transform | itself | exact (self) |
| `db/routes/regions_ui.go` (doc comment) | controller (static page server) | request-response | itself | exact (self) |

## Pattern Assignments

### `db/commands/seed_regions.go` (config/utility, batch)

**Analog:** self (edit in place) + `db/commands/dedup.go` for the Cobra command shape

**Cobra command skeleton** (`dedup.go` lines 14-24):
```go
func Dedup(app *pocketbase.PocketBase) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "dedup",
		Short: "Deduplicate trails by all matching fields",
		Run: func(cmd *cobra.Command, args []string) {
			records, err := app.FindAllRecords("trails")
			...
```
`SeedRegions()` already follows this shape (no `*pocketbase.PocketBase` needed — it never touches the DB). Keep the same `cmd.Flags().StringVar/IntVar` registration block (lines 164-166) but drop `limit`'s polygon-fetch semantics since there is no longer a polygon fetch loop to limit — `--limit` may be dropped entirely or kept for `hierarchy.txt`-node testing (Claude's Discretion applies to naming, not to whether the flag survives; check CONTEXT.md D-notes if ambiguous).

**Existing fetch/backoff pattern to reuse verbatim for the generator's now-single request** (lines 187-246):
```go
func fetch(rawURL string, maxBytes int64) ([]byte, error) { ... }
func doFetch(rawURL string, maxBytes int64) ([]byte, time.Duration, error) { ... }
func parseRetryAfter(v string) time.Duration { ... }
```
`seed_regions.go` keeps its own `fetch`/`doFetch`/`parseRetryAfter` (patient 10x budget, D-03 distinguishes this from the new build-path fetch) for the single `hierarchy.txt` GET. Delete: the `.poly` fetch loop (lines 112-138), `ParsePoly` import/call from this file, `gzip` import and the gzip-writing tail (lines 145-157), `Bbox`/`Polygon` fields from the emitted `SeedRow` (JSON `omitempty` fields become permanently absent — CONTEXT.md D-12).

**New field to add to `SeedRow`:** a `CommitSHA` (or similar) field, populated once from `commit` after the hierarchy fetch succeeds, per D-00b — "lives in the artifact," field name is Claude's Discretion. Simplest concrete shape: add it to every row, or lift `SeedRow` into a wrapper struct `{ Commit string; Rows []SeedRow }` — the wrapper is cleaner and avoids repeating the SHA ~1306 times in a pretty-printed diff. Recommend the wrapper given D-00a's stated diff-noise sensitivity.

**Marshal call to change** (line 140): replace `json.Marshal(rows)` with `json.MarshalIndent(rows, "", "  ")` (or the wrapper) for pretty-printing (D-00a), and remove the `gzip.NewWriter` wrapping around `outFile` — write `data` directly via `outFile.Write(data)`.

---

### `db/migrations/1785000000_create_regions_collection.go` (migration, batch)

**Analog:** self (edit in place, per D-05 — never shipped, no separate drop migration)

**Collection-creation + self-relation two-pass pattern to keep unchanged** (lines 53-104): the `regions` collection creation, including the two-`Save()` self-relation dance for `parent` — this is untouched by this phase except removing `bbox`/`enabled` fields from `regions` if D-12 relocates them (bbox already lived on `regions` at line 75 as a leaf-only field; move this field definition to the new `region_geometry` collection instead, alongside `enabled` staying on `regions` since it's a catalog toggle, not geometry).

**Second collection to rename and extend** (lines 106-125): `region_polygons` → `region_geometry`. Add a `bbox` field (`&core.JSONField{Name: "bbox", MaxSize: 1 << 10}`, moved from `regions`) alongside the existing `polygon` field. Ship it **empty** — remove the entire bulk-insert block for polygons (lines 198-205 inside pass 1) since D-09 mandates the table starts empty.

**Streaming decoder + decompression-bomb guard to delete entirely** (lines 130-160, 176-213 partially):
```go
gzReader, err := gzip.NewReader(seedFile)
...
dec := json.NewDecoder(io.LimitReader(gzReader, 512<<20))
if _, err := dec.Token(); err != nil { // consume opening '['
	...
for dec.More() {
	var row SeedRow
	if err := dec.Decode(&row); err != nil { ... }
	...
}
if _, err := dec.Token(); err != nil { // consume closing ']'
```
Replace with a plain `io.ReadAll` + `json.Unmarshal(data, &rows)` (or the wrapper struct if the generator adopted one) into a slice, since the artifact collapses to ~292 KB. Drop the `gzip` and `io` `LimitReader` imports; keep `encoding/json`, `os`, `fmt`, `strings`.

**Down-migration symmetry to preserve** (lines 242-253): keep dropping both collections by (new) name; `region_geometry` replaces `region_polygons` in the `FindCollectionByNameOrId` call.

**Idempotency guard to keep unchanged** (lines 44-51): `CountRecords("regions") > 0 → return nil`. D-05's accepted execution note (already-seeded dev boxes retain an orphaned `region_polygons` collection) requires no code change, just documentation.

---

### `db/services/regions/builder.go` — `buildRegion` (service, CRUD + event-driven)

**Analog:** self (edit in place)

**Current polygon lookup to replace** (lines 227-248):
```go
polyRecord, err := app.FindFirstRecordByFilter("region_polygons",
	"path = {:path}", dbx.Params{"path": regionID})
if err != nil {
	log.Printf("[regions] region %s has no region_polygons entry: %v", regionID, err)
	return
}
var polygon map[string]any
if err := polyRecord.UnmarshalJSONField("polygon", &polygon); err != nil || len(polygon) == 0 {
	log.Printf("[regions] region %s has an invalid or missing polygon (err=%v)", regionID, err)
	return
}
```
New shape per D-14's strict order:
1. `app.FindFirstRecordByFilter("region_geometry", "path = {:path}", ...)` — treat `err != nil` (absent) the same as an unmarshal failure/empty-geometry (malformed) — both branch to refetch, not to `return`.
2. On absent-or-malformed: call the new fetch helper (GitHub → Codeberg, `ParsePoly`), then persist to `region_geometry` (upsert by `path`) — this record's bbox+polygon become the new bbox/polygon for the rest of `buildRegion`.
3. Only if the refetch itself errors: call `setError(app, archive, err)` (existing helper, lines 429-434) and `return`.

**bbox source to change** (line 182-187): today reads `record.UnmarshalJSONField("bbox", ...)` off the `regions` record (`record` param). Per D-12, `bbox` now comes from the `region_geometry` row — but note `buildRegion`'s bbox-comparison logic (lines 195-213, `bboxChanged`) currently runs *before* the polygon lookup (which is lazy, gated by `!needsVector && !needsDem`). Since bbox now lives in `region_geometry`, the bbox read must move to occur after/alongside the geometry fetch — this reorders `buildRegion`'s control flow more than a one-line field swap. Read `region_geometry` **before** the early-return check if a cheap existence check needed to move bbox, or restructure so the geometry read happens first and feeds both the bbox-staleness comparison and the polygon build. Plan this restructuring carefully — it's the most invasive change in this file.

**Failure/continue pattern already in use, reuse verbatim** (`setError`, lines 429-434):
```go
func setError(app core.App, record *core.Record, err error) error {
	record.Set("status", "error")
	record.Set("error_message", err.Error())
	_ = app.Save(record)
	return err
}
```

**Per-region isolation to leave untouched** (`buildRegionSafely`, lines 137-148; `BuildAllLocked`, lines 110-135) — D-02 requires this unchanged.

---

### NEW: geometry/polygon fetch helper (Go)

**Analog:** `db/commands/seed_regions.go` `fetch`/`doFetch`/`parseRetryAfter` (lines 187-246) for the HTTP-GET-with-429-backoff shape; `ParsePoly` (`db/commands/poly_parser.go` lines 39-160) reused verbatim for conversion.

**Recommended placement:** `db/services/regions/` (e.g. `geometry_fetch.go`) since both `buildRegion` (in `services/regions`) and the new route need it, and `commands` importing into `services/regions` (or vice versa) should be avoided per the existing convention of not creating a `migrations -> commands` dependency (see migration file's own comment, lines 27-30, about avoiding cross-package deps). `ParsePoly` currently lives in package `commands` — either export/reuse it via import (`commands.ParsePoly`, if `services/regions` importing `commands` is acceptable) or, cleaner, relocate `ParsePoly` to a shared/neutral package (e.g. `services/regions` itself, or a new `geo`/`polyparse` package) since it's now consumed by two independent subsystems (generator CLI + backend service), not one.

**Backoff pattern to copy, tightened per D-03** (structure from `doFetch`/`parseRetryAfter`, `seed_regions.go` lines 209-246): same 429/`Retry-After` handling, same non-200 → descriptive error, but a much smaller retry budget/cap than `maxFetchRetries = 10` (30s sleeps) — e.g. 2-3 attempts, short fixed sleep or short cap, since this runs synchronously inside a per-region cron build (100+ regions) or inline in an HTTP request handler, not a patient one-off maintainer script.

**Two-host fallback shape (new — no direct precedent in this codebase for GitHub-then-Codeberg within one call):** structure as:
```go
func fetchPolygon(comapsID, commitSHA string) (geometry map[string]any, bbox [4]float64, err error) {
    data, err := fetchWithBudget(githubPolyURL(commitSHA, comapsID))
    if err != nil {
        data, err = fetchWithBudget(codebergPolyURL(commitSHA, comapsID))
        if err != nil {
            return nil, [4]float64{}, fmt.Errorf("fetch polygon for %s: tried github and codeberg: %w", comapsID, err)
        }
    }
    return ParsePoly(data)
}
```
Error message must name both upstreams tried (D-01/roadmap requirement). Codeberg's Forgejo raw URL form (`/{owner}/{repo}/raw/commit/{sha}/{path}`) needs verification against a live request during planning/implementation — flagged in ROADMAP design notes as not yet confirmed.

**URL construction to mirror** (`seed_regions.go` line 87, 124):
```go
baseURL := fmt.Sprintf("https://raw.githubusercontent.com/comaps/comaps/%s/data/", commit)
polyURL := baseURL + "borders/" + url.PathEscape(comapsID) + ".poly"
```
Reuse this exact GitHub URL-building logic; add a parallel Codeberg constant/function per D-04 (hardcoded, no env override).

**No test coverage for the network layer** (D-06) — only `ParsePoly`/fallback *decision* logic gets unit tests, following `db/commands/poly_parser_test.go`'s inline-fixture, `t.Run`-subtest style (no `testdata/` dirs):
```go
func TestParsePoly(t *testing.T) {
	t.Run("single outer ring yields a Polygon with correct bbox", func(t *testing.T) {
		fixture := `TestRegion
1
	0.000000E+00	0.000000E+00
	...
END
END
`
		geometry, bbox, err := ParsePoly([]byte(fixture))
		if err != nil { t.Fatalf(...) }
		...
	})
}
```

---

### NEW: backend route (e.g. `GET /api/v1/regions/{path}/geometry`)

**Analog for path-param validation + file/record existence checks:** `db/routes/regions_get.go` `RegionArchiveDownload`/`RegionArchiveDownloadDem` (lines 154-180):
```go
func RegionArchiveDownload(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if !regions.IsValidRegionID(id) {
		return e.BadRequestError("invalid region id", nil)
	}
	if _, err := os.Stat(regions.RegionArchivePath(id)); err != nil {
		return e.NotFoundError("Region archive not ready yet", nil)
	}
	return e.FileFS(...)
}
```
Reuse `regions.IsValidRegionID` (or the path-param convention) for input validation before any lookup/fetch. Return a `map[string]any{"type":..., "geometry":..., "bbox": ...}` JSON body (mirrors `RegionsList`'s `e.JSON(http.StatusOK, entries)` shape, `regions_get.go` line 148) rather than streaming a file.

**Auth pattern — the concrete, verified answer for superuser gating.** Custom Go routes in this codebase enforce superuser auth via `.Bind(apis.RequireSuperuserAuth())` chained directly on the route registration in `db/main.go`, exactly like the existing region-catalog admin routes:
```go
// db/main.go lines 244, 250-251
se.Router.DELETE("/region-catalog/{id}/archive", routes.RegionArchiveDelete).Bind(apis.RequireSuperuserAuth())
se.Router.POST("/region-catalog/sync", routes.RegionSyncStart).Bind(apis.RequireSuperuserAuth())
se.Router.GET("/region-catalog/sync", routes.RegionSyncStatus).Bind(apis.RequireSuperuserAuth())
```
vs. the group-level, any-authenticated-user pattern used for `/regions`:
```go
// db/main.go lines 266-269
regionsGroup := se.Router.Group("/regions")
regionsGroup.Bind(apis.RequireAuth())
regionsGroup.GET("", routes.RegionsList)
regionsGroup.GET("/{id}/download", routes.RegionArchiveDownload)
```
Per D-13, the new geometry endpoint needs `apis.RequireSuperuserAuth()`, **not** `apis.RequireAuth()` — it triggers outbound third-party fetches, so it must be superuser-only like the region-catalog admin routes, even though it lives under `/regions` conceptually. Register it either as a standalone route with its own `.Bind(apis.RequireSuperuserAuth())` (matching the delete/sync precedent) — this is the safer, more explicit choice since the existing `regionsGroup` is deliberately bound to the weaker `RequireAuth()` for its other members and mixing auth levels within one `Group()` is error-prone. Register outside `regionsGroup`, e.g.:
```go
se.Router.GET("/regions/{path}/geometry", routes.RegionGeometryGet).Bind(apis.RequireSuperuserAuth())
```
(exact path shape is planner's discretion per D-10 — CONTEXT.md suggests `GET /api/v1/regions/{path}/geometry` but the `/api/v1` prefix note at main.go lines 258-265 explains `/regions` is already effectively under that prefix via SvelteKit proxy).

**Persistence-on-enabled-only logic — no direct precedent, closest is `buildRegion`'s upsert-style `findOrCreateRegionRecord`** (`builder.go` lines 262-294) for the "find or create, then Save" shape:
```go
record := core.NewRecord(collection)
record.Set(...)
if err := app.Save(record); err != nil { return nil, err }
```
Apply this shape to `region_geometry`: look up the leaf's `regions` record by `path`, check `GetBool("enabled")`; only if true, upsert into `region_geometry` (find existing row by path, update in place, or create new).

**Error/response shape to follow:** `e.BadRequestError(...)`, `e.NotFoundError(...)`, `e.InternalServerError(...)` from `regions_get.go` — use these PocketBase `core.RequestEvent` helpers rather than hand-rolled `http.Error`.

---

### `db/routes/regions_ext/regions_ui.html` (`onLeafHoverStart`)

**Analog:** self, lines 1175-1194 — only this function needs repointing.

**Current implementation to change** (lines 1175-1194): replace the direct collection REST hit
```js
var res = await this.apiFetch('/api/collections/region_polygons/records?perPage=1&filter=' + encodeURIComponent(filter));
```
with a call to the new endpoint, e.g. `this.apiFetch('/api/v1/regions/' + row.path + '/geometry')` (or whatever path the route lands on), keeping the existing `_polygonCache` short-circuit and `_hoverToken` race-guard logic (lines 1176-1193) unchanged — it already absorbs repeat hovers, satisfying D-11's "no server-side cache" rationale.

**Two flows to leave untouched** — `loadEnabledPolygons` (lines 1101-1116) and `addPolygonForRow` (lines 1154-1162) keep reading `/api/collections/region_polygons/records` directly, only renaming the collection name in the URL string to `region_geometry` (D-09: enabled regions always have rows there).

**`fitToEnabled`** (lines 1135-1149) reads `r.bbox` off enabled leaf rows fetched via the (renamed) collection API — unaffected in shape, just depends on `region_geometry` now carrying `bbox` alongside `polygon`.

---

### `db/main.go` (route registration / hook wiring)

**Analog:** self

**Hook binding to rename** (lines 156-161):
```go
app.OnRecordCreate("region_archives", "region_polygons").BindFunc(hooks.ValidateRegionPathReferenceHandler())
app.OnRecordUpdate("region_archives", "region_polygons").BindFunc(hooks.ValidateRegionPathReferenceHandler())
```
→ replace `"region_polygons"` with `"region_geometry"` in both lines; `"region_archives"` untouched.

**New route registration** — see the backend-route section above for the exact `.Bind(apis.RequireSuperuserAuth())` idiom; insert near the existing region-catalog admin routes (lines 238-251) or immediately after the `regionsGroup` block (line 271), whichever the planner judges clearer given it needs different auth than the group.

---

### `db/routes/regions_get.go` (`RegionsList` bbox emission)

**Analog:** self, lines 96-101

**Current bbox read to change:**
```go
var bbox []float64
_ = r.UnmarshalJSONField("bbox", &bbox)
entry["bbox"] = bbox
entry["enabled"] = r.GetBool("enabled")
```
`r` here is the `regions` record — post-D-12, bbox no longer lives there. For an enabled leaf, join to `region_geometry` by `path` (same `dbx.NewExp("path = {:path}", ...)` pattern already used at lines 103-105 for `region_archives`) and read `bbox` from that record; omit the `bbox` key entirely when no `region_geometry` row exists (disabled region, or enabled-but-failed-fetch — D-12 consequence 2 requires a defined shape here, e.g. omit `bbox` and rely on `entry["status"] == "error"` to signal why).

---

### `db/services/regions/staleness.go` (`bboxChanged`)

**Analog:** self, lines 53-65 — the function signature and comparison logic (exact float64 equality, no epsilon) stay unchanged:
```go
func bboxChanged(configBbox [4]float64, storedMinLon, storedMinLat, storedMaxLon, storedMaxLat float64) bool {
	return configBbox[0] != storedMinLon || ...
}
```
Only the **caller** in `builder.go` changes what it passes as `configBbox` — now sourced from the freshly-fetched/read `region_geometry` bbox rather than the `regions` record's own bbox field (see `buildRegion` section above).

---

### `db/routes/regions_ui.go` (doc comment)

**Analog:** self, lines 33-38

**Comment to update:**
```go
// directly through PocketBase's own JWT-validated collection REST API
// (/api/collections/regions/records, /api/collections/region_polygons/records).
```
→ update the second collection name to `region_geometry`, and add a sentence noting the admin page's hover flow now also calls the new superuser-gated `/regions/{path}/geometry` (or equivalent) Go route directly, not just the collection REST API — since D-10/D-13 add a third privileged surface this comment should name.

## Shared Patterns

### HTTP fetch with retry/backoff
**Source:** `db/commands/seed_regions.go` lines 187-246 (`fetch`, `doFetch`, `parseRetryAfter`)
**Apply to:** the generator's slimmed single-request fetch (patient 10x budget, unchanged) AND the new build-path/route fetch helper (same shape, tighter budget per D-03 — do not literally share the function, since the retry *budget* must differ; share the *pattern*, not the code).

### Superuser-gated custom route
**Source:** `db/main.go` lines 244, 250-251 (`.Bind(apis.RequireSuperuserAuth())`)
**Apply to:** the new geometry-fetch route (D-13).

### Per-record path-keyed lookup + find-or-create
**Source:** `db/services/regions/builder.go` lines 262-294 (`findOrCreateRegionRecord`), and the `dbx.NewExp("path = {:path}", ...)` idiom used identically in `regions_get.go` lines 103-105 and `builder.go` line 231-232.
**Apply to:** `region_geometry` reads/writes in both `buildRegion` and the new route.

### Error/status state machine on archive-like records
**Source:** `db/services/regions/builder.go` `setError` (lines 429-434), used by `buildVector`/`buildDem`/callers.
**Apply to:** `buildRegion`'s new refetch-failure path (D-14/D-01) — call this exact helper, do not invent a new failure signal.

### Cobra maintainer command shape
**Source:** `db/commands/dedup.go` lines 14-24
**Apply to:** confirms `seed_regions.go`'s existing `SeedRegions()` shape needs no structural change, only its body.

### Go test conventions (inline fixtures, no testdata/)
**Source:** `db/commands/poly_parser_test.go` lines 1-42
**Apply to:** any new unit tests for the fallback *decision* logic or bbox/geometry equivalence assertions (D-06, D-07) — inline string/JSON fixtures, `t.Run` subtests, no `testdata/` directory.

## No Analog Found

None — every touched/new file has at least a role-match analog in the existing codebase (mostly itself, since 8 of 9 are in-place edits per this phase's design).

## Metadata

**Analog search scope:** `db/commands/`, `db/migrations/`, `db/services/regions/`, `db/routes/`, `db/main.go`
**Files scanned:** `seed_regions.go`, `poly_parser.go`, `poly_parser_test.go`, `dedup.go`, `1785000000_create_regions_collection.go`, `builder.go`, `staleness.go`, `regions_get.go`, `regions_ui.go`, `regions_ext/regions_ui.html`, `main.go`
**Pattern extraction date:** 2026-07-28
