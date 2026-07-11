---
phase: quick-260711-lzb
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - db/migrations/1781900000_added_dem_fields_to_tile_cells.go
  - db/services/tiles/generator.go
  - db/routes/map_cells_id.go
  - db/main.go
  - app/lib/models/map_cell.dart
  - app/lib/models/trail.dart
  - app/lib/entities/trail_entity.dart
  - app/lib/services/trail_download_service.dart
  - app/lib/util/offline_style_rewriter.dart
  - app/test/util/offline_style_rewriter_test.dart
  - app/lib/components/base/trail_map.dart
  - app/lib/routes/navigation_screen.dart
autonomous: true
requirements: [HILLSHADE-OFFLINE-01]

must_haves:
  truths:
    - "Downloading a trail also downloads a companion per-cell DEM .pmtiles archive from the server"
    - "The offline style points hillshadeSource at the local DEM archive (pmtiles://file://…_dem.pmtiles), not the vector archive"
    - "The offline DEM source carries encoding:terrarium and tileSize:512 so relief decodes correctly"
    - "Opening a downloaded trail in airplane mode shows hillshade relief"
    - "The vector basemap still renders offline with no regression (existing rewriter tests still pass)"
  artifacts:
    - path: "db/migrations/1781900000_added_dem_fields_to_tile_cells.go"
      provides: "dem_status/dem_size_bytes/dem_error_message fields on tile_cells"
      contains: "dem_status"
    - path: "db/services/tiles/generator.go"
      provides: "per-cell DEM pmtiles extraction against download.mapterhorn.com"
      contains: "DemCellPath"
    - path: "db/routes/map_cells_id.go"
      provides: "GET /map/cells/{cellKey}/download-dem + dem_download_url in status JSON"
      contains: "MapCellsDownloadDem"
    - path: "app/lib/util/offline_style_rewriter.dart"
      provides: "raster-dem special-case pointing hillshade at DEM cells"
      contains: "raster-dem"
    - path: "app/test/util/offline_style_rewriter_test.dart"
      provides: "tests proving raster-dem sources point at DEM archives with terrarium/512"
      contains: "raster-dem"
    - path: "app/lib/services/trail_download_service.dart"
      provides: "parallel DEM cell download + entity.demPmTiles wiring"
      contains: "demPmTiles"
  key_links:
    - from: "app/lib/services/trail_download_service.dart"
      to: "/map/cells/<key>/download-dem"
      via: "MapCellStatusResponse.demDownloadUrl"
      pattern: "demDownloadUrl"
    - from: "app/lib/util/offline_style_rewriter.dart"
      to: "pmtiles://file://<dem cell>"
      via: "raster-dem source pointing"
      pattern: "pmtiles://file://"
    - from: "db/services/tiles/generator.go demMaxZoom"
      to: "app/lib/util/offline_style_rewriter.dart _offlineDemMaxZoom"
      via: "shared max-zoom constant (must be equal)"
      pattern: "12"
---

<objective>
Make hillshading work offline in the Flutter app by cloning the existing vector-cell
pipeline for a companion DEM archive, and by fixing the offline style rewriter bug that
currently repoints `hillshadeSource` (a `raster-dem` source) at the wrong (vector) archive.

Purpose: A downloaded trail viewed in airplane mode currently shows a flat basemap — hillshade
either fails to load or renders garbage because `offline_style_rewriter.dart` treats
`hillshadeSource` as a generic tiled source and points it at the vector `.pmtiles`. This
delivers real offline hillshade relief.

Output:
- Go: a second `pmtiles extract` per grid cell against `download.mapterhorn.com/planet.pmtiles`,
  parallel `dem_*` status fields, and a `/download-dem` route.
- Flutter: `demPmTiles` threaded through model/entity/download loop, and a `raster-dem`
  special case in the rewriter that injects `encoding:terrarium` + `tileSize:512` and points
  at the DEM archive.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/quick/260711-lzb-make-hillshading-work-offline-in-the-flu/260711-lzb-CONTEXT.md
@.planning/quick/260711-lzb-make-hillshading-work-offline-in-the-flu/260711-lzb-RESEARCH.md

# Go pipeline to extend (exact edit points in RESEARCH.md §1-2)
@db/services/tiles/generator.go
@db/routes/map_cells_id.go
@db/migrations/1778840749_created_tile_cells.go

# Flutter integration points (exact edit points in RESEARCH.md §3)
@app/lib/services/trail_download_service.dart
@app/lib/util/offline_style_rewriter.dart
@app/test/util/offline_style_rewriter_test.dart
@app/lib/entities/trail_entity.dart
@app/lib/models/map_cell.dart
@app/assets/map/wanderer_light.json
</context>

<tasks>

<task type="auto">
  <name>Task 1: Go — per-cell DEM extraction, tile_cells DEM fields, and download-dem route</name>
  <files>db/migrations/1781900000_added_dem_fields_to_tile_cells.go, db/services/tiles/generator.go, db/routes/map_cells_id.go, db/main.go</files>
  <action>
  Clone the existing vector-cell pipeline for a companion DEM archive (per HILLSHADE-OFFLINE-01,
  CONTEXT "DEM tile caching architecture").

  1. NEW additive migration `db/migrations/1781900000_added_dem_fields_to_tile_cells.go` (use any
  unix timestamp greater than the latest existing migration `1781890072`). Do NOT edit the existing
  `1778840749_created_tile_cells.go`. In the Up func, `FindCollectionByNameOrId("pbc_581507754")`
  (tile_cells) and append three fields, then `app.Save(collection)`; the Down func removes them:
  `dem_status` (select, values pending/ready/error, `required:false` — leave it optional so existing
  vector-only rows re-save cleanly and an absent value means "DEM not generated"), `dem_size_bytes`
  (number, optional), `dem_error_message` (text, optional). Give each a fresh unique field id.

  2. `db/services/tiles/generator.go`: add consts `demMaxZoom = 12` (per RESEARCH A1 — cap lower than
  vector's z14; MapLibre overzooms past it) and `mapterhornSource = "https://download.mapterhorn.com/planet.pmtiles"`
  (static; no date-rotation helper needed, unlike `getValidProtomapsURL`). Add
  `DemCellPath(cell GridCell) string` returning `filepath.Join(cacheDir, cell.CacheKey()+"_dem.pmtiles")`.
  In `findOrCreateRecord`, set `dem_status` to `"pending"` on the new record. Change `EnsureCell`'s
  early-return so it returns nil only when the vector `status=="ready"` AND `CellPath` exists AND
  `dem_status=="ready"` AND `DemCellPath` exists — otherwise fall through to `generateCell`, so
  pre-existing vector-only cells still get a DEM. In `generateCell`: keep the vector `pmtiles extract`
  exactly as-is but guard it to run only when `CellPath` is missing (preserve already-cached vector
  cells). Then, independently, run a second `exec.Command("pmtiles","extract", mapterhornSource,
  DemCellPath(cell), "--bbox="+bbox, fmt.Sprintf("--maxzoom=%d", demMaxZoom))` (same cell bbox). On
  DEM success: set `dem_status="ready"`, `dem_size_bytes` from `os.Stat`, clear `dem_error_message`.
  On DEM failure: `os.Remove` the partial DEM file, set `dem_status="error"` + `dem_error_message`, and
  LOG it but DO NOT return an error — hillshade is cosmetic and a DEM failure must never block the
  vector basemap from serving (CONTEXT: separate dem lifecycle). Save the record.

  3. `db/routes/map_cells_id.go`: add `MapCellsDownloadDem(e)` mirroring `MapCellsDownload` — parse the
  cell key, 404 if `DemCellPath` is missing, else `e.FileFS(os.DirFS("./pb_data/pmtiles_cache"),
  cell.CacheKey()+"_dem.pmtiles")` (no download-count increment needed). In `MapCellsGet` and
  `MapCellsStatus`, when returning the ready payload, also add a `"dem_download_url"` key
  (`"/map/cells/"+cell.CacheKey()+"/download-dem"`) — but ONLY when the record's `dem_status=="ready"`
  AND `DemCellPath` exists on disk, so a DEM-failed cell simply omits the field and the client skips it.

  4. `db/main.go`: register `g.GET("/{cellKey}/download-dem", routes.MapCellsDownloadDem)` alongside the
  existing `/{cellKey}/download` route (~line 219).
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/db && go build ./... && go vet ./services/tiles/... ./routes/... && grep -q "DemCellPath" services/tiles/generator.go && grep -q "MapCellsDownloadDem" routes/map_cells_id.go && grep -q "download-dem" main.go</automated>
  </verify>
  <done>`go build ./...` and `go vet` pass; generator.go runs a second pmtiles extract per cell against download.mapterhorn.com with --maxzoom=12; the migration adds dem_status/dem_size_bytes/dem_error_message; GET /map/cells/{cellKey}/download-dem is registered and serves the _dem.pmtiles file; dem_download_url appears in the cell status JSON only when the DEM archive is ready.</done>
</task>

<task type="auto">
  <name>Task 2: Flutter — thread DEM cell paths through model, entity, and the download loop</name>
  <files>app/lib/models/map_cell.dart, app/lib/models/trail.dart, app/lib/entities/trail_entity.dart, app/lib/services/trail_download_service.dart</files>
  <action>
  Carry the DEM archive paths from download to storage, mirroring how `pmTiles` already flows
  (RESEARCH §3 "Download loop").

  1. `app/lib/models/map_cell.dart`: add `@JsonKey(name: 'dem_download_url') String? demDownloadUrl`
  to `MapCellStatusResponse` (the server field added in Task 1).

  2. `app/lib/models/trail.dart`: add `@Default([]) List<String> demPmTiles` immediately after the
  existing `@Default([]) List<String> pmTiles` field (~line 101).

  3. `app/lib/entities/trail_entity.dart`: add `List<String> demPmTiles = [];` next to
  `List<String> pmTiles = [];` (line 41), and in the `toModel()` extension add `demPmTiles: demPmTiles`
  next to the existing `pmTiles: pmTiles` (line 157).

  4. `app/lib/services/trail_download_service.dart`: change `_downloadMapTiles` to also produce DEM
  paths. Inside the per-cell `downloadTasks` closure, after the vector `_api.download(readyCell.downloadUrl!, localPath)`,
  if `readyCell.demDownloadUrl != null` download it to `'${tilesDir.path}/${key}_dem.pmtiles'`; wrap the
  DEM download in its own try/catch and treat a DEM failure or a missing `demDownloadUrl` as best-effort
  (skip that cell's DEM path — hillshade is cosmetic and must not fail the trail download; a vector
  download failure stays fatal as today). Return both lists from `_downloadMapTiles` (use a record
  `(List<String> vector, List<String> dem)` or a small private class). In `downloadTrail`, set
  `entity.pmTiles = <vector paths>;` and `entity.demPmTiles = <dem paths>;` (replacing the single
  assignment at line 80). Also add DEM files to the cache-hit short-circuit (line 158) so a re-download
  reuses an already-present `${key}_dem.pmtiles`.

  5. Regenerate code: run build_runner so the freezed `Trail`/`MapCellStatusResponse` and the objectbox
  entity mapping pick up the new fields.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && dart run build_runner build --delete-conflicting-outputs && flutter analyze lib/models/map_cell.dart lib/models/trail.dart lib/entities/trail_entity.dart lib/services/trail_download_service.dart && grep -q "demPmTiles" lib/objectbox-model.json && grep -q "demDownloadUrl" lib/models/map_cell.freezed.dart</automated>
  </verify>
  <done>build_runner regenerates freezed + objectbox with no errors; `demPmTiles` exists on both `Trail` and `TrailEntity` and is persisted (present in objectbox-model.json); `MapCellStatusResponse.demDownloadUrl` deserializes `dem_download_url`; the download loop pulls a `${key}_dem.pmtiles` per cell and stores the paths in `entity.demPmTiles`; `flutter analyze` reports no new errors on the touched files.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 3: Flutter — raster-dem special case in the offline style rewriter (fix the hillshade bug) + wire callers</name>
  <files>app/lib/util/offline_style_rewriter.dart, app/test/util/offline_style_rewriter_test.dart, app/lib/components/base/trail_map.dart, app/lib/routes/navigation_screen.dart</files>
  <behavior>
    Write these tests FIRST (RED), against `rewriteStyleForOffline` with a `raster-dem`
    `hillshadeSource` (type `raster-dem`, only `type`+`url`) plus a `hillshade` layer, in addition
    to the existing `protomaps` vector source:
    - Single cell: with `cellPaths:[A]`, `demCellPaths:[demA]` the `hillshadeSource` `url` becomes
      `pmtiles://file://demA`, and it gains `encoding=='terrarium'`, `tileSize==512`, `maxzoom==12`
      (the dedicated DEM constant), while `protomaps` still points at A with `maxzoom==14`.
    - Multi cell: with `cellPaths:[A,B]`, `demCellPaths:[demA,demB]` a `hillshadeSource-cell-1`
      source points at demB and a `hillshade__cell1` layer clone references it (mirroring the vector
      cell-clone behavior).
    - Empty DEM: with `demCellPaths` empty, no `raster-dem` source and no hillshade layer remain, and
      NO `http(s)://` url leaks into the output (offline path-safety invariant holds); the call does
      not throw (hillshade is cosmetic).
    - Path safety: a `demCellPaths` entry with a `..` segment / non-absolute / foreign scheme throws
      ArgumentError (routed through the same `_assertSafePath`).
    Existing vector-source tests must keep passing unchanged (no regression).
  </behavior>
  <action>
  Fix the bug where `_rewriteSourcesAndLayers` sweeps `hillshadeSource` into the vector `tiledKeys`
  (because it has a `url`) and repoints it at a vector `.pmtiles` (CONTEXT "Existing style-rewriter
  bug"; RESEARCH §3 "Style rewriter").

  1. `offline_style_rewriter.dart`: add a `List<String> demCellPaths = const []` named param to
  `rewriteStyleForOffline`; assert each entry via the existing `_assertSafePath`. Add a dedicated
  `const int _offlineDemMaxZoom = 12;` (MUST equal the Go `demMaxZoom` from Task 1 — keep them in
  lockstep, same rationale as the existing `_offlinePmtilesMaxZoom` comment). In
  `_rewriteSourcesAndLayers`, split the candidate keys by `source['type']`: keys with
  `type=='raster-dem'` are DEM sources routed to `demCellPaths`; all other tiled keys keep going to
  `cellPaths`. Add a `_pointDemSourceAtCell(source, demPath)` that removes any `tiles`, sets
  `url='pmtiles://file://$demPath'`, and sets `encoding='terrarium'`, `tileSize=512`,
  `maxzoom=_offlineDemMaxZoom` (RESEARCH §4 pitfalls — these are absent from the style JSON because
  Mapterhorn's tilejson supplies them online; a `pmtiles://` source has no tilejson so they MUST be
  injected or MapLibre defaults to tileSize 256 / encoding mapbox → wrong relief). Apply the same
  per-extra-cell source + layer duplication used for vector cells, but keyed off `demCellPaths[i]`.
  When `demCellPaths` is empty, remove the `raster-dem` source keys and drop every layer that
  references them, so no `https://` hillshade url survives into the offline style (path-safety
  invariant). Update the function's doc comment to describe the raster-dem branch.

  2. `offline_style_rewriter_test.dart`: add the tests described in <behavior>.

  3. Wire the two callers to pass the new paths: `app/lib/components/base/trail_map.dart` `_composeStyle`
  (~line 150) add `demCellPaths: widget.trail.demPmTiles`; `app/lib/routes/navigation_screen.dart`
  `_composeStyle` (~line 429) add `demCellPaths: ref.read(trailProvider(widget.id)).value?.demPmTiles ?? const []`.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && flutter test test/util/offline_style_rewriter_test.dart && flutter analyze lib/util/offline_style_rewriter.dart lib/components/base/trail_map.dart lib/routes/navigation_screen.dart && grep -q "raster-dem" lib/util/offline_style_rewriter.dart && grep -q "_offlineDemMaxZoom = 12" lib/util/offline_style_rewriter.dart</automated>
    <human-check>On a physical device (A2/A3 smoke test, RESEARCH §4): download a trail with the app online, enable airplane mode, open the trail. Confirm hillshade relief renders offline (terrain shading visible over the basemap) and the basemap is not blank. If relief is garbage/blank, the webp-terrarium raster-dem native decode assumption failed — capture logs before deciding on a fallback.</human-check>
  </verify>
  <done>New rewriter tests pass and all pre-existing rewriter tests still pass; `hillshadeSource` (and any `raster-dem` source) points at the DEM `.pmtiles` with terrarium/512/maxzoom-12; empty `demCellPaths` drops hillshade with no https leak; both callers pass `demCellPaths` from `demPmTiles`; `flutter analyze` clean on touched files.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| downloaded cell paths → offline style JSON | DEM cell file paths are interpolated into `pmtiles://file://` urls inside the map style |
| Go `exec.Command` → shell/CLI | `pmtiles extract` is invoked with a bbox + static remote source URL |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-lzb-01 | Tampering | `offline_style_rewriter.dart` demCellPaths | mitigate | Route every `demCellPaths` entry through the existing `_assertSafePath` (rejects `..`, non-absolute, foreign scheme) before it enters a `file://` url — same guard already covering `cellPaths`. |
| T-lzb-02 | Injection | `generator.go` DEM `exec.Command` | accept | The DEM source URL is a hardcoded const; `--bbox` is `%f`-formatted floats from server-computed `GridCell`s (no user-controlled string). No shell interpolation (exec args, not `sh -c`). |
| T-lzb-03 | Information disclosure | offline style with empty DEM | mitigate | When `demCellPaths` is empty the rewriter drops raster-dem sources + layers so no `https://tiles.mapterhorn.com` url leaks into the offline style (preserves the existing no-network invariant). |
| T-lzb-SC | Tampering | npm/pip/cargo/pub installs | accept | No new packages added — reuses the already-present `pmtiles` CLI (Go side) and `pmtiles: ^1.2.0` Dart reader. No Package Legitimacy Gate required. |
</threat_model>

<verification>
- Go: `cd db && go build ./... && go vet ./...` pass; the second `pmtiles extract` targets
  `download.mapterhorn.com/planet.pmtiles` with `--maxzoom=12`; `/map/cells/{key}/download-dem` serves
  the `_dem.pmtiles` file.
- Flutter unit: `cd app && flutter test test/util/offline_style_rewriter_test.dart` — raster-dem
  sources point at DEM archives with `encoding:terrarium`/`tileSize:512`/`maxzoom:12`; vector sources
  unchanged (`maxzoom:14`); empty-DEM path leaks no `http(s)://`.
- Flutter codegen: `dart run build_runner build --delete-conflicting-outputs` clean; `demPmTiles`
  persisted in `objectbox-model.json`.
- Lockstep check: the Go `demMaxZoom` const and the Dart `_offlineDemMaxZoom` const are both `12`.
- Device smoke test (A2/A3): download a trail, airplane mode, open it — hillshade relief renders
  offline. Capture logs if the native webp-terrarium raster-dem decode fails.
</verification>

<success_criteria>
- Downloading a trail produces a `${cellKey}_dem.pmtiles` per grid cell alongside the vector cell.
- `tile_cells` tracks DEM status independently (`dem_status`) from the vector `status`.
- The offline style resolves `hillshadeSource` from the local DEM archive with terrarium/512 encoding
  — the pre-existing bug (hillshade repointed at the vector archive) is gone.
- A downloaded trail viewed offline (airplane mode) shows hillshade relief; the vector basemap shows
  no regression (all pre-existing rewriter tests still pass).
</success_criteria>

<output>
Create `.planning/quick/260711-lzb-make-hillshading-work-offline-in-the-flu/260711-lzb-SUMMARY.md` when done
</output>
