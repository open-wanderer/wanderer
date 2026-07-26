# Phase 29: Polygon-Based Extraction & Region API - Research

**Researched:** 2026-07-26
**Domain:** Go/PocketBase backend — cron target discovery, `pmtiles extract --region` polygon clipping, hierarchy-aware REST API
**Confidence:** HIGH

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| EXTRACT-01 | The archive-generation cron extracts each enabled leaf region via `pmtiles extract --region <polygon>` using its canonical polygon, replacing bbox-based extraction | Directly verified on this machine: ran real `pmtiles extract --region=<geojson>` against a real archive, confirmed true polygon clipping (not a bbox pre-filter) — see Code Examples and Common Pitfalls. Cross-verified against official docs and the exact seeded `polygon` JSON shape from Phase 28. |
| EXTRACT-02 | The cron reads `kind = 'leaf' AND enabled = true` from the `regions` table to determine build targets; `region_config.json` parsing is retired entirely | Architecture Patterns Pattern 2 gives the exact `dbx` query + the full list of `db/services/regions/config.go` call sites that must be deleted/replaced (verified via `grep` against real source) |
| EXTRACT-03 | `GET /api/v1/regions` includes hierarchy fields (`parent`, `path`, `depth`) alongside existing bbox/status/size fields, so the client can render a tree | Architecture Patterns Pattern 3 + Open Question 1 (whether group rows must also be returned) give the exact current handler shape (`db/routes/regions_get.go`, read in full) and what must be added |
</phase_requirements>

## Summary

This phase replaces the admin-hand-authored `region_config.json` (a flat array of `{id, name, bbox}`) with direct queries against the `regions` PocketBase collection Phase 28 seeded (1306 rows: 153 groups, 1153 leaves, keyed uniquely by materialized `path`, not by `comaps_id` — 5 real disputed-territory leaves collide on `comaps_id`). Three existing files own all of the logic being replaced: `db/services/regions/config.go` (`LoadRegionCatalog`/`Region`/`ValidateRegion`/`IsValidRegionID`/path builders), `db/services/regions/builder.go` (`BuildAll`/`buildRegion`/`buildVector`/`buildDem`, the cron entrypoint registered in `db/main.go`'s `registerCronJobs`), and `db/routes/regions_get.go` (`RegionsList`, the handler behind the internal `/regions` route that `web/src/routes/api/v1/regions/+server.ts` proxies verbatim as `GET /api/v1/regions`).

The riskiest unknown going into this phase — whether `pmtiles extract --region <polygon>` actually performs true polygon clipping against a real CoMaps-shaped GeoJSON boundary, not just a bbox pre-filter — was **directly resolved in this research session**, not left as a pending spike. Using the repo's own already-built `munich` test archive (`db/pb_data/region_archives/munich/vector.pmtiles`, bounds 11.358,48.061 to 11.731,48.226, 340 tiles) as a source, a triangular GeoJSON polygon covering roughly the SW half of that bbox was extracted with `pmtiles extract --region=triangle.geojson`: the output contained only 209 of the 340 tiles (61%), and a specific tile just outside the triangle but inside the bbox came back "Tile not found in archive" from the region-extracted output while present (and byte-identical between a bbox-extract and a region-extract) for a tile genuinely inside the polygon. `--region` also accepted a `MultiPolygon` (Fiji-style antimeridian-split leaves are common in the seeded catalog) and a GeoJSON `Feature`-wrapped geometry without error. This confirms EXTRACT-01's mechanism end-to-end with HIGH confidence — the pending todo `.planning/todos/pending/2026-07-24-comaps-poly-region-extraction-spike.md` can be marked resolved once this phase ships.

Two non-obvious, concretely-verified risks surfaced during this research that the plan must address, not just the two "obvious" requirement-mapped pieces: (1) the existing `regionIDPattern` regex (`^[a-z0-9][a-z0-9_-]*$`, enforced in **both** the Go backend `IsValidRegionID` and the SvelteKit proxy's `RegionIdSchema`) will reject real seeded `path` values — 29 of 1306 rows contain a `.` (the path separator) or an apostrophe (e.g. `people's_republic_of_china`) that this regex does not allow; and (2) the existing hand-tested `munich` region (`region_archives.region_id = "munich"`) has **no corresponding row anywhere in the seeded CoMaps catalog** (CoMaps' finest granularity in Bavaria is the Regierungsbezirk/district level, e.g. "Upper Bavaria" — there is no city-level "Munich" leaf), so it becomes orphaned dead data + orphaned `.pmtiles` files on disk the moment `region_config.json` parsing is retired.

**Primary recommendation:** Query `regions` directly with `dbx.NewExp("kind = {:kind} && enabled = {:enabled}", ...)` for cron targets (EXTRACT-02); write each leaf's `polygon` JSON field to a per-build temp `.geojson` file and pass it via `--region=<tmpfile>` instead of `--bbox=` (EXTRACT-01); and change `region_archives.region_id` to store the region's `path` (not a free-text admin ID), updating the id-validation regex in **both** Go and SvelteKit to safely allow `.` and `'` while still rejecting path-traversal characters. Return the full `regions` table (both `kind=group` and `kind=leaf` rows) from `GET /api/v1/regions`, not leaf-only, so a client has the group node names/depths needed to render a real tree (EXTRACT-03) — flagged below as an assumption since it's not explicitly locked by any prior design doc.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Cron build-target discovery (`regions` query) | API/Backend (Go, `db/services/regions/builder.go`) | Database/Storage (PocketBase/SQLite `regions` table) | Same tier as the code it replaces; no new tier introduced |
| Polygon-based `pmtiles extract` subprocess invocation | API/Backend (Go, shells out to the `pmtiles` CLI binary already vendored in the Docker image) | — | Identical mechanism to the existing bbox-based extract, only the input geometry source changes |
| `GET /api/v1/regions` hierarchy fields | API/Backend (Go `db/routes/regions_get.go`) | Frontend Server (SvelteKit proxy, `web/src/routes/api/v1/regions/+server.ts` — pure pass-through, no reshaping needed) | The SvelteKit route is a byte-for-byte JSON proxy; all new fields flow through automatically once the Go handler emits them — no SvelteKit code change required for the data itself, only its OpenAPI doc comment (optional) |
| Region-id validation (`.`/`'` in `path`) | API/Backend (Go `regionIDPattern`) | Frontend Server (SvelteKit `RegionIdSchema` — duplicated validation, must stay in lockstep) | Two independent copies of the same regex exist today; both must change together or the download route breaks for any path containing `.`/`'` |

## Package Legitimacy Audit

No new external packages are installed this phase. The `pmtiles` CLI binary is already vendored into the production Docker image (`db/Dockerfile`, pinned `PMTILES_VERSION=1.28.0`, downloaded from `github.com/protomaps/go-pmtiles`'s official GitHub releases) and invoked via `exec.Command` exactly as `db/services/regions/builder.go` already does for bbox-based extraction — only the CLI flag (`--region=` instead of `--bbox=`) and the argument's file content change. No `go.mod` changes are required.

**Packages removed due to slopcheck `[SLOP]` verdict:** none (nothing installed)
**Packages flagged as suspicious `[SUS]`:** none (nothing installed)

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/protomaps/go-pmtiles` (CLI binary, not a Go import) | v1.28.0 `[VERIFIED: db/Dockerfile ENV PMTILES_VERSION]` | `pmtiles extract --region=<geojson-file>` polygon-clipped extraction | Already the project's only PMTiles extraction mechanism (bbox-based today); this phase only changes which flag is used, not the tool |
| `github.com/pocketbase/pocketbase` | v0.38.0 `[VERIFIED: db/go.mod, same as Phase 28]` | `dbx` query construction, `core.Record`/`core.App` for reading the `regions` collection | Already the project's backend framework |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `os` (stdlib) `CreateTemp`/`WriteFile`/`Remove` | Go 1.25 | Writing a leaf's `polygon` JSON field out to a per-build temp `.geojson` file before invoking `pmtiles extract --region=` | `--region` requires a **local file path** argument, not inline JSON or stdin — confirmed via `pmtiles extract --help` and official docs |
| `encoding/json` (stdlib) | Go 1.25 | Marshal the `polygon` field's `map[string]any` value (already the shape produced by Phase 28's `record.UnmarshalJSONField`) directly to the temp file | Verified: a bare `{"type":"Polygon"/"MultiPolygon","coordinates":[...]}` object (no `Feature` wrapper needed) is accepted as-is by `pmtiles extract --region=` |
| `crypto/sha1` or similar (stdlib) | Go 1.25 | Optional: hash the polygon geometry to detect a rare hierarchy-refresh-triggered boundary change, replacing the old `bboxChanged`-triggered forced-rebuild role | Discretionary — see Common Pitfalls "Staleness trigger gap" below |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Temp-file `--region=` argument | Piping GeoJSON via stdin | Not supported — `pmtiles extract --region` explicitly documents "local GeoJSON Polygon or MultiPolygon **file**"; no stdin mode observed in `--help` or docs |
| `region_id` text field storing `path` | A proper PocketBase relation field from `region_archives` to `regions` | A relation field is more "correct" (referential integrity, avoids duplicating a string key) but is a larger schema change touching an existing production collection with live data (the orphaned `munich` row); recommend deferring to a future phase unless the planner has reason to do it now — text-field-storing-`path` is the lower-risk, requirement-satisfying choice for this phase |

**Installation:**
No new packages to install — `pmtiles` CLI is already baked into the Docker image; `pocketbase`/stdlib are already present.

**Version verification:** `pmtiles` CLI version pinned at `PMTILES_VERSION=1.28.0` in `db/Dockerfile` (downloaded from `github.com/protomaps/go-pmtiles`'s GitHub releases) `[VERIFIED: db/Dockerfile]`. This research's own hands-on `--region` verification (below) ran against a locally-installed `pmtiles` binary (Homebrew-installed, reported version 1.17.0 via `brew info`) — the `--region` flag's existence, file-input contract, and true-clipping behavior were confirmed directly by executing it, and cross-checked against `docs.protomaps.com/pmtiles/cli` (official docs, describing the same flag/behavior for the current CLI) `[CITED: docs.protomaps.com/pmtiles/cli]`. The flag has existed in go-pmtiles across multiple releases per its GitHub issue history; no version-specific behavior change between 1.17.0 and 1.28.0 was found. Confidence: HIGH for flag existence/contract, MEDIUM-HIGH for "byte-identical CLI behavior at exactly 1.28.0" (not the literal binary tested).

## Architecture Patterns

### System Architecture Diagram

```text
db/main.go registerCronJobs()
  "region-archive-build" cron (REGION_ARCHIVE_CRON_SCHEDULE, default "0 3 * * *")
       │
       ▼
db/services/regions/builder.go  BuildAll(app)
       │  REPLACES: LoadRegionCatalog() (region_config.json)
       │  WITH:      app.FindAllRecords("regions",
       │               dbx.NewExp("kind = {:k} && enabled = {:e}",
       │                 dbx.Params{"k":"leaf","e":true}))
       ▼
  for each enabled leaf `regions` record:
       │
       ├─ 1. write record's `polygon` JSON field to a temp file
       │       tmpPolyPath := os.CreateTemp("", "region-*.geojson")
       │       json.Marshal(polygon) → tmpPolyPath
       │
       ├─ 2. resolve region_archives record
       │       (region_id now = regions.path, was admin config .ID)
       │
       ├─ 3. staleness check (UNCHANGED: needsVectorRebuild via
       │       Protomaps daily-build date; DEM builds once)
       │       + NEW: polygon-changed check replaces old bboxChanged
       │
       ├─ 4. pmtiles extract <source> <tmp.pmtiles>
       │       --region=<tmpPolyPath>          (was --bbox=<bbox string>)
       │       --maxzoom=<14 vector / 12 DEM>  (UNCHANGED)
       │
       ├─ 5. atomic rename tmp → final (UNCHANGED discipline)
       │
       └─ 6. os.Remove(tmpPolyPath)             (NEW cleanup step)

GET /regions  (internal-only, Go)  ─── proxied verbatim by ───▶  GET /api/v1/regions (SvelteKit)
db/routes/regions_get.go  RegionsList(e)
       │  REPLACES: regions.LoadRegionCatalog() iteration (leaf-only)
       │  WITH:      app.FindAllRecords("regions")  -- ALL rows: group + leaf
       │             (see Open Question 1 re: full-catalog vs enabled-only)
       ▼
  for each `regions` record:
       entry := { id, name, kind, parent, path, depth }   // NEW hierarchy fields
       if kind == "leaf":
         entry += { bbox (from regions.bbox, NOT region_archives columns),
                     enabled, status, version, vector_url, vector_size,
                     dem_status, dem_url, dem_size, error }
                     // UNCHANGED shape/semantics, sourced from region_archives
                     // joined on region_id == regions.path
```

### Recommended Project Structure
```
db/
├── services/regions/
│   ├── config.go       # MODIFIED — remove LoadRegionCatalog/Region/ValidateRegion
│   │                    #   (region_config.json parsing retired per EXTRACT-02);
│   │                    #   KEEP RegionArchivePath/RegionDemPath/RegionCacheDir;
│   │                    #   UPDATE regionIDPattern to allow `.`/`'` (see Pitfalls)
│   ├── builder.go       # MODIFIED — BuildAll queries `regions` table directly;
│   │                    #   buildRegion/buildVector/buildDem take a `regions`
│   │                    #   core.Record (or a small struct derived from one)
│   │                    #   instead of the old catalog `Region` struct
│   ├── staleness.go      # MODIFIED (minor) — bboxChanged's role replaced by a
│   │                    #   polygon-hash comparison, OR removed if the planner
│   │                    #   judges the rare-refresh case not worth tracking
│   └── polygon_extract.go   # NEW (or a function in builder.go) — writes a
│                          #   region's polygon JSON to a temp .geojson file,
│                          #   returns its path + a cleanup func
├── routes/
│   └── regions_get.go    # MODIFIED — RegionsList iterates `regions` table
│                          #   (group + leaf), joins region_archives by path
└── main.go                # UNCHANGED registration points (same route group,
                           #   same cron registration call)
```

### Pattern 1: Verified true polygon clipping via `pmtiles extract --region=<file>`
**What:** `--region` takes a local file path to a GeoJSON `Polygon`, `MultiPolygon`, `Feature`, or `FeatureCollection` and clips to the polygon boundary, not its bounding box.
**When to use:** Every leaf region build in `buildVector`/`buildDem`, replacing `--bbox=`.
**Verified directly in this session** (not assumed from docs):
```bash
# Source: pmtiles extract --help (local install) + hands-on run against
# db/pb_data/region_archives/munich/vector.pmtiles (real repo archive,
# bounds 11.358,48.061 to 11.731,48.226, 340 tiles total)

# Full-bbox extract (baseline): 340 tiles, 13MB
pmtiles extract munich/vector.pmtiles bbox_out.pmtiles \
  --bbox=11.358,48.061,11.731,48.226 --maxzoom=14

# Triangular polygon covering ~SW half of the same bbox: 209 tiles, 8.7MB
cat > triangle.geojson <<'EOF'
{"type":"Polygon","coordinates":[[
  [11.358,48.061],[11.731,48.061],[11.358,48.226],[11.358,48.061]
]]}
EOF
pmtiles extract munich/vector.pmtiles triangle_out.pmtiles \
  --region=triangle.geojson --maxzoom=14
# -> "Region tiles 209, result tile entries 209" (vs 340 for full bbox)

# Spot-check: a z14 tile near the excluded NE corner (11.72,48.22 -> tile
# 8725,5680) is genuinely absent from the region-extracted output:
pmtiles tile triangle_out.pmtiles 14 8725 5680
# -> "Tile not found in archive."
pmtiles tile bbox_out.pmtiles 14 8725 5680
# -> present, 4016 bytes (same tile, present when using the full bbox)

# A tile inside both the bbox AND the polygon (8709,5690, near the SW
# corner) is present and byte-identical in both archives (37066 bytes),
# confirming no content corruption from region-based clipping.
```
**MultiPolygon and Feature-wrapped geometry also verified accepted** (no error, correct tile subsetting) in the same session — relevant because the seeded catalog contains real `MultiPolygon` leaves (e.g. `Caribisch Nederland`, and Fiji-style antimeridian splits per Phase 28's research).

### Pattern 2: Querying `regions` for cron build targets (EXTRACT-02)
**What:** Replace `regions.LoadRegionCatalog()` with a direct `dbx`-filtered `FindAllRecords` call against the seeded `regions` collection.
**When to use:** `BuildAll`'s entrypoint.
**Example:**
```go
// Source: pattern adapted from db/routes/regions_get.go's existing
// dbx.NewExp("region_id = {:id}", ...) idiom (same package, same dbx import)
leafRecords, err := app.FindAllRecords("regions",
    dbx.NewExp("kind = {:kind} && enabled = {:enabled}",
        dbx.Params{"kind": "leaf", "enabled": true}),
)
if err != nil {
    log.Printf("[regions] failed to query enabled leaf regions: %v", err)
    return
}
for _, record := range leafRecords {
    buildRegionSafely(app, record) // signature changes from (app, Region) to (app, *core.Record)
}
```
**Every call site referencing the retired catalog must be found and removed** — verified via direct `grep` in this session, the full list is:
- `db/services/regions/config.go`: `LoadRegionCatalog`, `Region` struct, `ValidateRegion` — DELETE (comment doc references to `REGION_CATALOG_CONFIG_PATH`/BACK-01/D-05 become stale and should be removed with the code)
- `db/services/regions/builder.go` line 63: `regionsList, err := LoadRegionCatalog()` — REPLACE with the query above
- `db/routes/regions_get.go` line 27: `catalog, err := regions.LoadRegionCatalog()` — REPLACE with a `regions` table read (see Pattern 3)
- **KEEP unchanged:** `RegionArchivePath`/`RegionDemPath`/`RegionCacheDir`/`IsValidRegionID` (function signature stays, only the underlying regex changes — see Pitfall "Path-unsafe characters")

### Pattern 3: `GET /api/v1/regions` hierarchy fields (EXTRACT-03)
**What:** `RegionsList` currently iterates the admin catalog (leaf-only) and merges in `region_archives` build state. It must instead iterate the `regions` table (both `kind`s) and add `parent`/`path`/`depth`.
**Current handler shape (verified, read in full this session):** `db/routes/regions_get.go` builds one `map[string]any` per catalog entry with `id`/`name`/`bbox` always present, then conditionally adds `status`/`version`/`vector_url`/`vector_size`/`dem_status`/`dem_url`/`dem_size`/`error` based on the joined `region_archives` record.
**Example of the minimal addition (illustrative, not prescriptive on every field):**
```go
// Source: pattern extends db/routes/regions_get.go's existing entry-building
// loop; regions.parent/path/depth per Phase 28's 1785000000_create_regions_collection.go
records, err := e.App.FindAllRecords("regions") // ALL rows: group AND leaf
// ...
for _, r := range records {
    entry := map[string]any{
        "id":     r.Id,
        "name":   r.GetString("name"),
        "kind":   r.GetString("kind"),
        "parent": r.GetString("parent"), // relation field's raw value = parent record id, "" for roots
        "path":   r.GetString("path"),
        "depth":  r.GetInt("depth"),
    }
    if r.GetString("kind") == "leaf" {
        var bbox []float64
        _ = r.UnmarshalJSONField("bbox", &bbox)
        entry["bbox"] = bbox
        entry["enabled"] = r.GetBool("enabled")
        // ...then merge in region_archives build-state exactly as today,
        // joined on region_archives.region_id == r.GetString("path")
    }
    entries = append(entries, entry)
}
```
**Open question, not resolved by any prior design doc:** whether to return every row (group + leaf, regardless of `enabled`) or filter down. See Open Questions below — recommend returning everything so Phase 31's Flutter tree has group labels to render, but flag for confirmation since Phase 31 isn't this phase's concern to lock in.

### Anti-Patterns to Avoid
- **Reusing `comaps_id` as a join/lookup key anywhere in this phase's new code.** Phase 28's own migration found 5 real duplicate `comaps_id` values (disputed territories). `path` is the only field proven globally unique across all 1306 seed rows — use it exclusively for joins (`region_archives.region_id`, any map-keyed lookups).
- **Interpolating `polygon` JSON directly into a shell string.** Always write it to a temp file via `os.CreateTemp` + `json.Marshal`/`os.WriteFile`, then pass the file *path* as `--region=`'s argument via `exec.CommandContext`'s argument slice (never through a shell) — mirrors the existing `--bbox=` argument's safe construction (`fmt.Sprintf` into a Go string, passed as a discrete `exec.Command` arg, never through `sh -c`).
- **Leaving the old `regionIDPattern` regex unchanged while switching `region_id` to store `path`.** This silently breaks the download route for the 29 rows (of 1306) whose `path` contains `.` or `'` — a real, verified-in-this-session data mismatch, not a theoretical edge case.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Polygon-to-tile clipping | A custom point-in-polygon tile filter before calling `pmtiles extract --bbox=` | `pmtiles extract --region=<geojson-file>` (already does true polygon clipping, verified this session) | Reinventing this would duplicate go-pmtiles' own R-tree/spatial-index-backed clipping logic — a correctness and performance regression versus the tool's built-in flag |
| Materialized-path tree reconstruction for the API response | A recursive query or in-memory tree-builder inside the Go handler | Return every row flat with `parent`/`path`/`depth`; let the **client** build the tree (the client already has to do this in Phase 31 regardless — building it twice, once server-side and once client-side, is wasted, divergence-prone work) | Matches the existing design note's stated intent ("materialized path... cheap 'all descendants' via prefix match") — the server's job is to expose the raw shape, not pre-render a nested JSON tree |
| Region-id path-safety validation | A brand-new regex from scratch | Extend the existing `regionIDPattern`/`RegionIdSchema` pair (both already exist, both already enforce this exact contract) | Two independent hand-rolled regexes already exist and are proven — the fix is updating both in lockstep, not building a third validation mechanism |

**Key insight:** Nothing in this phase requires a new abstraction. Every piece — cron target discovery, subprocess invocation, ID validation, hierarchy-field serialization — already has a working, in-repo precedent from the bbox-based system being replaced. The work is substitution, not invention.

## Common Pitfalls

### Pitfall 1: `regionIDPattern`/`RegionIdSchema` reject real seeded `path` values
**What goes wrong:** If `region_archives.region_id` (and the download route's `{id}` path param) switches to storing `regions.path`, both the Go `regionIDPattern` (`^[a-z0-9][a-z0-9_-]*$`) and the SvelteKit `RegionIdSchema` (identical regex, `web/src/routes/api/v1/regions/[id]/download/+server.ts`) will reject any path containing `.` (the materialized-path separator, e.g. `algeria.algeria_central`) or `'` (e.g. `people's_republic_of_china`, and its 28 descendant rows).
**Why it happens:** The old regex was written for admin-hand-typed slugs (`munich`, `germany-bavaria`) that never needed a hierarchy separator or an apostrophe. It was never re-examined against real seeded data.
**How to avoid:** Directly counted in this session against the real committed `regions_seed.json.gz`: **29 of 1306 rows** fail the current regex (all due to `.` or `'`). Update both regexes to something like `^[a-z0-9](?:[a-z0-9_.'-])*$` (still rejects `/`, `\`, `..` path-traversal sequences, spaces, and control characters — verify the updated pattern still blocks `../` before shipping) or, more simply, keep using PocketBase's own opaque `record.Id` (guaranteed `^[a-z0-9]{15}$`) as the URL-facing id instead of `path`, sidestepping the regex problem entirely (see "Alternatives Considered" above).
**Warning signs:** A previously-working download URL 400s (`BadRequestError`/Zod validation failure) only for specific regions — specifically ones whose display name contains an apostrophe or that are nested more than one level deep.

### Pitfall 2: The existing `munich` region becomes orphaned dead data
**What goes wrong:** `region_archives` currently has (in dev/test environments that ran the old cron) a record with `region_id = "munich"`, plus real built files at `db/pb_data/region_archives/munich/{vector,dem}.pmtiles`. Directly verified this session: **no row in the seeded 1306-row CoMaps catalog corresponds to "Munich"** — CoMaps' finest granularity under Bavaria is Regierungsbezirk/district level (e.g. "Upper Bavaria"), not city level. Once `region_config.json` parsing is retired (EXTRACT-02), nothing will ever query or rebuild this record again — it's orphaned, not migrated.
**Why it happens:** `region_config.json` was hand-authored ad hoc for local testing/dev and was never a real CoMaps entity; the new catalog has no equivalent granularity.
**How to avoid:** This is not a data-loss risk (the seeded catalog doesn't reference "munich" at all, so nothing breaks), but the plan should explicitly decide: leave the orphaned record + files in place (harmless, just stale disk usage) or add a one-time cleanup step. Not a blocking requirement — flagging so it isn't a surprise during verification (e.g. a test asserting "region_archives table only contains valid catalog entries" would need to account for or clean up this row).
**Warning signs:** A `region_archives` row whose `region_id` doesn't match any `regions.path` value — a useful invariant check for a verification step.

### Pitfall 3: Staleness/rebuild trigger gap once bbox is no longer admin-editable
**What goes wrong:** The current `buildRegion` forces a full rebuild whenever `bboxChanged(r.Bbox, ...)` is true — this was the mechanism for "admin edited the config file, boundaries changed, must rebuild both vector and DEM even though DEM normally builds once" (D-11 from Phase 21.5). Once bbox/polygon comes from the seeded, effectively-static catalog, this trigger will almost never fire in normal operation — but it CAN still fire in the rare case of a Phase 28-style catalog refresh (D-02's "bump `--commit`, re-run, review diff, ship as a migration" flow), which could shift a region's polygon/bbox slightly (e.g. an upstream boundary correction).
**Why it happens:** It's tempting to simply delete the `bboxChanged` check entirely since "the bbox never changes now" — true in the common case, but not true across a hierarchy refresh.
**How to avoid:** Keep an equivalent check, just re-keyed on the seeded catalog's data instead of admin config: compare `regions.bbox` (or a hash of `regions.polygon`) against what's stored in `region_archives`, exactly mirroring the old `bboxChanged` role. Don't remove the check outright.
**Warning signs:** After a future hierarchy-refresh migration ships, a region's boundary visibly changed but its already-built archive silently keeps serving the old shape until the next Protomaps-date-triggered vector rebuild (which could be up to a day later) and DEM never rebuilds at all (DEM has no date-based staleness trigger, only the bbox-changed one).

### Pitfall 4: `--region` requires a real file path, not inline JSON or stdin
**What goes wrong:** Assuming `--region=<json-string>` or piping GeoJSON via stdin will work, based on how some other CLI tools accept inline data.
**Why it happens:** Natural assumption given `--bbox=` takes an inline comma-separated string; `--region` looks similar syntactically but isn't.
**How to avoid:** Confirmed directly (`pmtiles extract --help`: "local GeoJSON Polygon or MultiPolygon file for area of interest") and by successfully running it only ever with a real file path in this session. Always write the polygon to a temp file first (`os.CreateTemp`, write, defer `os.Remove`).
**Warning signs:** `pmtiles extract` failing immediately with a file-not-found-style error if a bare JSON string is passed as `--region=`'s value.

### Pitfall 5: Large `MultiPolygon` leaves and temp-file concurrency
**What goes wrong:** Some leaf polygons are large (Phase 28 research flagged up to 8-16MB JSON for complex coastlines) and multiple regions could theoretically build concurrently in a future refactor (today's `BuildAll` is sequential, bounded to 1 concurrent subprocess via the existing `inFlight` guard — this remains true and doesn't need to change).
**Why it happens:** Temp file names must be unique per build to avoid collision if concurrency is ever introduced later.
**How to avoid:** Use `os.CreateTemp("", "region-*.geojson")` (Go's stdlib already guarantees a unique name) rather than a fixed/predictable filename; always `defer os.Remove(path)` immediately after creation, even on early-return error paths.
**Warning signs:** Not observable today (sequential builds), but would surface as a corrupted/overwritten polygon file mid-extract if a future change introduces concurrent region builds without also parameterizing the temp file per-build.

## Code Examples

### Verified `pmtiles extract --region` invocation shape
```go
// Adapts the EXISTING db/services/regions/builder.go buildVector shape —
// only the flag and its argument source change; everything else (temp path,
// atomic rename, timeout context, Stdout/Stderr wiring) is unchanged.
tmpFile, err := os.CreateTemp("", "region-*.geojson")
if err != nil {
    return fmt.Errorf("create temp polygon file: %w", err)
}
defer os.Remove(tmpFile.Name())

var polygon map[string]any
if err := record.UnmarshalJSONField("polygon", &polygon); err != nil {
    return fmt.Errorf("unmarshal polygon for region %s: %w", record.GetString("path"), err)
}
polyBytes, err := json.Marshal(polygon)
if err != nil {
    return fmt.Errorf("marshal polygon for region %s: %w", record.GetString("path"), err)
}
if _, err := tmpFile.Write(polyBytes); err != nil {
    return fmt.Errorf("write temp polygon file: %w", err)
}
tmpFile.Close()

cmd := exec.CommandContext(ctx, "pmtiles", "extract",
    url,
    tmp, // output path, same as today
    "--region="+tmpFile.Name(),
    fmt.Sprintf("--maxzoom=%d", regionVectorMaxZoom),
)
```

### Existing `dbx` filter idiom, extended for the new query (verified against real in-repo usage)
```go
// Source: db/routes/regions_get.go line 34-36 (existing dbx.NewExp usage,
// same package) — extended with the kind/enabled predicate for EXTRACT-02
leafRecords, err := app.FindAllRecords("regions",
    dbx.NewExp("kind = {:kind} && enabled = {:enabled}",
        dbx.Params{"kind": "leaf", "enabled": true}),
)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Admin hand-authors `region_config.json`, bbox-based `pmtiles extract --bbox=` | Curated seeded `regions` catalog, polygon-based `pmtiles extract --region=` | This phase (Phase 29) | Retires `LoadRegionCatalog`/`ValidateRegion`/`Region` entirely; archive tile coverage now follows real political/geographic boundaries instead of a rectangle, reducing archive size for irregular regions (verified: ~35% smaller for a simple triangular test case, larger irregular regions will see bigger wins) |
| `region_id` = admin-typed free-text slug | `region_id` = seeded `regions.path` (materialized path, globally unique) | This phase | Requires the `regionIDPattern`/`RegionIdSchema` regex update (Pitfall 1) |

**Deprecated/outdated:** `region_config.json` (`db/pb_data/region_config.json`) and the `REGION_CATALOG_CONFIG_PATH` env var become fully unused after this phase — safe to delete the file and stop documenting the env var (verify no deployment docs/`.env.example` still reference it before removing).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `GET /api/v1/regions` should return **all** `regions` rows (both `kind=group` and `kind=leaf`), not leaf-only, so a client can render group labels in a tree | Architecture Patterns Pattern 3 / Open Questions | If the planner instead keeps leaf-only, Phase 31's Flutter tree would have parent IDs pointing at group records it never received — the client couldn't render group names/labels at all, only a flat list of leaves with no visible nesting names. Medium risk — not locked by any CONTEXT.md (none exists for this phase) or the design note, which is silent on this exact point. |
| A2 | `region_archives.region_id` should switch from an admin-typed slug to storing `regions.path` (not a new relation field) | Standard Stack "Alternatives Considered" | If the planner instead adds a proper relation field, more schema migration work is needed but referential integrity improves; if `path` is used and the regex fix (Pitfall 1) is skipped, 29 real regions become undownloadable. Medium risk — this is the most consequential open design choice in this phase. |
| A3 | `regions.parent` should be exposed in the API response as the raw relation-field value (parent's record `id`), not the parent's `path` | Architecture Patterns Pattern 3 | If a client actually needs the parent's `path` string (e.g., to build its own materialized-path-based tree rather than an id-keyed one), the field would need reshaping. Low risk — either shape lets a client build a tree, just with a different join key; `path` itself is already present per-row regardless, so a client can derive the parent's path from its own path by string-stripping if needed. |
| A4 | The old `bboxChanged`-triggered forced-rebuild role should be kept (re-keyed to the seeded catalog), not deleted outright | Common Pitfalls "Staleness trigger gap" | If deleted, a future hierarchy-refresh migration that corrects a region's boundary would silently not trigger a rebuild — low near-term risk (refreshes are rare/manual per Phase 28's own research) but a real latent gap. |

**If this table is empty:** N/A — see entries above; none are locked decisions from a CONTEXT.md since none exists for this phase yet.

## Open Questions (RESOLVED)

1. **Should `GET /api/v1/regions` return every `regions` row, or only `enabled` leaves (plus their ancestor groups)?**
   - What we know: EXTRACT-03's literal wording ("returns each region's `parent`, `path`, and `depth`... so a client can reconstruct the tree") is satisfied by any superset that includes enough rows to build a connected tree. Phase 30 (admin UI) reads the `regions` table directly, not through this API, so it doesn't force an answer here. Phase 31 (Flutter) is the only consumer that needs this resolved, and it's a later phase.
   - What's unclear: Whether the app's Settings hierarchy (Phase 31) should show every catalog region (including ones the admin hasn't enabled — greyed out?) or only what's actually downloadable today.
   - Recommendation: Return everything (both kinds, `enabled` or not) in this phase — it's the maximal, most-flexible response shape and doesn't foreclose either future Phase 31 design. Filtering down is a trivial client-side or later-server-side change; adding fields back in later (if under-scoped now) is not.
   - **RESOLVED: Return every row (both `group` and `leaf`, regardless of `enabled`) — adopted in 29-02 Task 1.**

2. **Exact temp-file cleanup discipline on early-return error paths in `buildVector`/`buildDem`**
   - What we know: The existing functions have several early `return` points on error (`setError`, DEM error paths) before reaching the end of the function.
   - What's unclear: Whether every one of those early returns needs an explicit `os.Remove(tmpPolyPath)` or whether a single `defer` at the top of the function (before any error path) suffices.
   - Recommendation: A single `defer os.Remove(tmpFile.Name())` immediately after successful temp-file creation, before any error-prone step, covers every subsequent early return — no per-branch cleanup needed.
   - **RESOLVED: Single top-level `defer os.Remove(tmpFile.Name())` after temp-file creation — adopted in 29-01 Task 3.**

3. **Whether the orphaned `munich` `region_archives` row/files (Pitfall 2) need an explicit cleanup task**
   - What we know: Nothing in the new catalog will ever reference "munich" again; it's inert, not actively harmful.
   - What's unclear: Whether a verification step or future admin-UI listing would be confused by a `region_archives` row with no matching `regions.path`.
   - Recommendation: Leave it — cleaning it up is optional polish, not a requirement. Flag it in the plan's verification notes so a "stale test data" surprise doesn't get misdiagnosed as a bug later.
   - **RESOLVED: Leave `munich` orphaned, no cleanup task — documented in 29-01's `<notes>` for verification awareness.**

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `pmtiles` CLI | `pmtiles extract --region=` (EXTRACT-01) | ✓ (verified — used directly in this research session) | Production: 1.28.0 pinned in `db/Dockerfile`; local dev machine: 1.17.0 (Homebrew) | — |
| Go toolchain | Building/testing the modified `db/services/regions/*` and `db/routes/regions_get.go` | ✓ | 1.25.0 (per `db/go.mod`) | — |
| A real `.pmtiles` source archive to extract from (Protomaps daily build, or a local test archive) | Verifying `buildVector`/`buildDem` end-to-end in tests | ✓ (the existing `db/pb_data/region_archives/munich/vector.pmtiles` test archive was used for this research's hands-on verification) | — | For a full local test against a real seeded region's real-shaped polygon, the plan may need to fetch a small Protomaps daily-build slice, or continue using the existing munich archive as a stand-in source |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None — everything needed is already present in the dev environment and the production Docker image.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | No new auth surface — the existing `/regions` route group's `apis.RequireAuth()` binding is unchanged; this phase only changes what data flows through already-authenticated handlers |
| V3 Session Management | No | Unchanged |
| V4 Access Control | No | No new PocketBase collection rules — `regions`' access rules were already set in Phase 28 (unchanged here); the cron runs with full `core.App` privileges as it already does |
| V5 Input Validation | Yes | The updated `regionIDPattern`/`RegionIdSchema` regex (Pitfall 1) is the single most important input-validation surface this phase touches — it gates both a filesystem path (`RegionArchivePath`/`RegionDemPath`) and a `dbx` query parameter. Any relaxation to allow `.`/`'` must be re-verified to still reject `..`, `/`, `\`, and control characters (path traversal) |
| V6 Cryptography | No | No cryptographic operations in this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Path traversal via a relaxed `region_id`/`path`-derived id reaching `filepath.Join` (`RegionArchivePath`/`RegionDemPath`) | Tampering | Keep the allow-list regex approach (don't switch to a deny-list) — explicitly test that the updated pattern still rejects `../`, absolute paths, and null bytes before shipping (existing `IsValidRegionID`/`ValidateRegion` precedent, just with an expanded but still-bounded character class) |
| Command injection via the polygon temp-file path or the region's polygon content reaching `exec.Command` | Tampering | Already mitigated by construction: `exec.CommandContext` takes a discrete argument slice (no shell interpolation, matching the existing `--bbox=` argument's safe construction) — the temp file's path comes from `os.CreateTemp` (Go stdlib, never influenced by untrusted input), and the file's *content* (polygon JSON) is never itself passed as a shell argument, only its file path is |
| A malformed/oversized `polygon` JSON field causing an unbounded temp-file write or `pmtiles extract` hang | Denial of Service | The `regionExtractTimeout()` context (already exists, 30min default, `REGION_ARCHIVE_EXTRACT_TIMEOUT`-overridable) already bounds subprocess runtime; Phase 28's `MaxSize: 8 << 20` cap on the `polygon` JSONField already bounds the field's stored size, so the temp file written from it is bounded too |

## Sources

### Primary (HIGH confidence)
- Direct hands-on execution in this session: `pmtiles extract --region=<geojson>` against `db/pb_data/region_archives/munich/vector.pmtiles` (real repo archive) — confirmed true polygon clipping (209/340 tiles), confirmed absent-tile spot-check via `pmtiles tile`, confirmed `MultiPolygon` and `Feature`-wrapped input acceptance
- `pmtiles extract --help` (local install) — flag list, `--region`'s documented "local GeoJSON Polygon or MultiPolygon file" contract
- `db/services/regions/config.go`, `builder.go`, `staleness.go`, `db/routes/regions_get.go` — read in full, this session
- `db/main.go` — cron/route registration, read directly (`registerCronJobs`, `registerRoutes`)
- `web/src/routes/api/v1/regions/+server.ts`, `.../[id]/download/+server.ts` — read directly, confirms the SvelteKit proxy is a pure pass-through (no reshaping needed for EXTRACT-03's new fields) and confirms the duplicated `RegionIdSchema` regex (Pitfall 1)
- `db/migrations/1785000000_create_regions_collection.go`, `db/migrations/1784658610_created_region_archives.go` — read directly, confirms exact field names/types on both collections
- `db/migrations/initial_data/regions_seed.json.gz` — decompressed and inspected directly in this session (1306 rows; confirmed `path` field format, confirmed 29 rows with `.`/`'`, confirmed no "munich" row exists, confirmed real `MultiPolygon` leaf example)
- `db/Dockerfile` — `pmtiles` CLI version pinning (`PMTILES_VERSION=1.28.0`) confirmed directly
- Phase 28 artifacts (`28-CONTEXT.md`, `28-RESEARCH.md`, `28-PATTERNS.md`, all four `28-0X-SUMMARY.md` files) — read in full, establish the exact schema/seed shape this phase builds on

### Secondary (MEDIUM confidence)
- `https://docs.protomaps.com/pmtiles/cli` `[CITED]` — official docs confirming `--region` accepts Polygon/MultiPolygon/Feature/FeatureCollection, cross-verifying this session's hands-on findings
- WebSearch results referencing `github.com/protomaps/go-pmtiles/issues/68` (pmtiles extract feedback thread) — corroborates `--region`'s documented format acceptance, not independently fetched/read in full this session

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; `pmtiles` CLI version pinned and already vendored
- Architecture: HIGH — every replaced call site was located via direct `grep`/`Read`, not inferred
- Pitfalls: HIGH for 1/2/4 (directly counted/verified against real seed data and real CLI behavior this session); MEDIUM for 3/5 (reasoned from existing code's documented intent, not independently reproduced)
- EXTRACT-01's core mechanism (polygon clipping via `--region`): HIGH — directly verified end-to-end in this session against a real archive, not left as an unresolved spike

**Research date:** 2026-07-26
**Valid until:** No fixed expiry — re-verify only if `db/Dockerfile`'s `PMTILES_VERSION` changes (a major go-pmtiles version bump could in theory change `--region`'s flag contract, though no such change is documented across the versions checked), or if Phase 28's `regions` seed data is regenerated with a different `path`-uniqueness or character-set profile than observed here.
