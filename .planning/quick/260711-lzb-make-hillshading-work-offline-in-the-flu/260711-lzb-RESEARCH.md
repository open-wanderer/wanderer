# Quick Task 260711-lzb: Make hillshading work offline - Research

**Researched:** 2026-07-11
**Domain:** Go/PocketBase tile pipeline + Flutter/MapLibre offline style rewriting
**Confidence:** HIGH (mirrors existing vector-cell pattern verbatim; two `[ASSUMED]` items flagged for the renderer)

## Summary

This is a near-exact clone of the existing per-cell vector pipeline, applied to a companion
DEM archive. The Go side adds a second `pmtiles extract` call against
`https://download.mapterhorn.com/planet.pmtiles` per grid cell; the `tile_cells` schema gains
parallel DEM status/size fields; the Flutter download loop pulls a second `.pmtiles` per cell;
and `offline_style_rewriter.dart` must **stop** treating `hillshadeSource` as a vector source
(the current bug) and instead point `raster-dem` sources at the DEM archives with explicit
`encoding`/`tileSize`.

**Primary recommendation:** Extend, don't invent. Add `--maxzoom=12` DEM extraction, parallel
`_dem`-suffixed fields/files/routes, and a `raster-dem` special case in the rewriter that injects
`"encoding": "terrarium"` and `"tileSize": 512` (both currently supplied online by Mapterhorn's
tilejson and therefore absent from the style JSON).

## 1. `pmtiles extract` invocation for generator.go

**Current vector call** (`db/services/tiles/generator.go:139-144`), verbatim:
```go
cmd := exec.Command("pmtiles", "extract",
    pmtilesSource,                       // getValidProtomapsURL() -> build.protomaps.com/<date>.pmtiles
    outputPath,                          // CellPath(cell) -> ./pb_data/pmtiles_cache/<key>.pmtiles
    "--bbox="+bbox,                      // "minLon,minLat,maxLon,maxLat"
    fmt.Sprintf("--maxzoom=%d", maxZoom),// maxZoom const = 14 (generator.go:18)
)
```

**Analogous DEM call** (same bbox, static source, lower cap):
```go
demCmd := exec.Command("pmtiles", "extract",
    "https://download.mapterhorn.com/planet.pmtiles",  // static; no date-rotation needed
    demOutputPath,                                       // e.g. ./pb_data/pmtiles_cache/<key>_dem.pmtiles
    "--bbox="+bbox,                                      // identical cell bbox
    fmt.Sprintf("--maxzoom=%d", demMaxZoom),             // recommend demMaxZoom = 12
)
```
Notes:
- The vector source rotates daily via `getValidProtomapsURL()`; Mapterhorn's planet archive is a
  single static URL, so no equivalent helper is needed — hardcode it (or a const).
- `--bbox` and cell iteration are unchanged; DEM cells align 1:1 with vector cells (same
  `grid.go` `GridSize = 0.5`, same `CacheKey()`).

**`--maxzoom` recommendation: `12`.** [ASSUMED]
Reasoning about archive growth for one 0.5°×0.5° cell (tile count quadruples per zoom, so the
deepest level dominates): at mid-latitudes ~2^z/360 tiles per degree, a 0.5° span is roughly
`z10 ≈ 3×3`, `z11 ≈ 6×6`, `z12 ≈ 11×11`. Full pyramid to z12 ≈ ~150 tiles; at ~50-150 KB per
512px webp DEM tile that is ~a few MB per cell — comparable to a vector cell, acceptable inside
the existing per-trail download. Capping at **z11** roughly quarters that (~40 tiles, ~1-2 MB) if
size becomes a concern; hillshading is cosmetic relief and reads fine at z11-12. `pmtiles extract`
caps gracefully if Mapterhorn's own max zoom is below the requested value, so an over-request is
safe. Mapterhorn's tilejson advertises no `maxzoom`, so the true source ceiling is unverified —
z12 is a safe, small target regardless.

## 2. `tile_cells` schema addition

Migration `db/migrations/1778840749_created_tile_cells.go` defines (collection `pbc_581507754`):
`cell_key` (text), `status` (select: `pending`/`ready`/`error`), `min_lon`/`min_lat`/`max_lon`/
`max_lat`/`size_bytes`/`download_count` (number), `error_message` (text), `created`/`updated`.

**Minimal addition** (new migration; PocketBase migrations are additive — do not edit the existing
file). Follow the existing status/size convention with a `dem_` prefix so vector and DEM lifecycles
track independently within the same row:

| New field | Type | Mirrors |
|-----------|------|---------|
| `dem_status` | select `pending`/`ready`/`error` | `status` |
| `dem_size_bytes` | number (optional) | `size_bytes` |
| `dem_error_message` | text (optional) | `error_message` |

Rationale for separate `dem_status` (not reusing `status`): the two extracts can succeed/fail
independently, and a vector-ready / DEM-failed cell should still serve the basemap. `generateCell`
sets `dem_status` alongside `status`; `MapCellsList`/`MapCellsGet`/`MapCellsStatus` expose a DEM
download URL only when `dem_status == "ready"`. A single new field could work if you accept
all-or-nothing coupling, but separate status matches the codebase's existing per-artifact
status pattern and is the safer minimal choice.

## 3. Flutter integration points

**Download loop** — `app/lib/services/trail_download_service.dart`:
- `_downloadMapTiles` (lines 130-195) is the vector cell loop. Per cell it hits `cell.url`
  (`/map/cells/<key>`), polls `_pollUntilReady`, then `_api.download(readyCell.downloadUrl!, localPath)`
  where `localPath = '${tilesDir.path}/$key.pmtiles'` (line 156).
- Add a parallel DEM download in the same `downloadTasks` map: download the cell's DEM URL to
  `'${tilesDir.path}/${key}_dem.pmtiles'`. Simplest wiring: extend `MapCellStatusResponse` /
  the cell status JSON to carry a `dem_download_url`, or add a fixed `/map/cells/<key>/download-dem`
  route (mirrors `MapCellsDownload`, `map_cells_id.go:80` + registration `main.go:219`). Return a
  parallel `demCellPaths` list.
- Line 80 `entity.pmTiles = cellPaths;` — add `entity.demPmTiles = demCellPaths;`.
- `TrailEntity` (`app/lib/entities/trail_entity.dart:41` `List<String> pmTiles = [];`) needs a new
  `List<String> demPmTiles = [];` field → regenerate objectbox (`objectbox.g.dart`,
  `objectbox-model.json`) via build_runner. The `Trail` freezed model similarly needs a `demPmTiles`
  field if the paths flow through it (currently `pmTiles` is on both entity and `Trail`).

**Style rewriter** — `app/lib/util/offline_style_rewriter.dart`:
- The bug: `_rewriteSourcesAndLayers` (lines 83-139) selects `tiledKeys` as any source with a
  `tiles` **or** `url` key (line 90-94). `hillshadeSource` has `"url": ".../tilejson.json"`
  (style `wanderer_light.json:12-15`), so it is swept in and `_pointSourceAtCell` (155-159)
  repoints it at a **vector** protomaps `.pmtiles` — wrong data type → broken/blank hillshade.
- Fix: split `tiledKeys` by `source['type']`. Route `type == 'raster-dem'` sources to the DEM
  archive paths (`demCellPaths`), everything else to `cellPaths`. Add a parallel
  `required List<String> demCellPaths` param to `rewriteStyleForOffline` (signature line 47-52).
- DEM-specific pointing function (a `raster-dem` variant of `_pointSourceAtCell`) must set:
  `source['url'] = 'pmtiles://file://$demCellPath'`, `source['encoding'] = 'terrarium'`,
  `source['tileSize'] = 512`, and `source['maxzoom'] = <DEM max, e.g. 12>` — a **separate**
  constant from `_offlinePmtilesMaxZoom = 14` (line 148), which must match the Go DEM `--maxzoom`.

**Multi-cell duplication:** Apply the **same** `<key>-cell-<i>` source + `__cell<i>` layer-clone
pattern (lines 110-138) to the hillshade source, using `demCellPaths[i]`. DEM cells are 1:1 with
vector cells (same grid/bbox/count), so `demCellPaths.length == cellPaths.length`. A trail spanning
2-4 cells would otherwise show hillshade over only one cell. Since it aligns cleanly and the
machinery already exists, do the duplication rather than special-casing to a single source — the
cost is negligible and correctness is free. (Single-source is an acceptable *degraded* fallback
only if a `demCellPaths` mismatch forces it, since hillshade is cosmetic.)

## 4. Pitfalls

- **`tileSize` / `encoding` must be injected explicitly.** The style JSON's `hillshadeSource`
  carries only `type` + `url` (`wanderer_light.json:12-15`); `encoding: terrarium` and
  `tileSize: 512` are supplied online by Mapterhorn's `tilejson.json`. A `pmtiles://` source has no
  tilejson, so both must be written into the source during rewrite or MapLibre defaults to
  `tileSize: 256` and `encoding: mapbox` → wrong elevation decode and misaligned relief. **[CITED: Mapterhorn tilejson in CONTEXT.md canonical_refs — `encoding:"terrarium"`, `tileSize:512`]**
- **`terrarium` encoding is required, not default.** MapLibre's default raster-dem encoding is
  `mapbox`; decoding terrarium-encoded tiles as mapbox produces garbage hillshade. Set
  `"encoding": "terrarium"` on the offline source explicitly. [ASSUMED — MapLibre default is
  `mapbox`; verify against `maplibre` 0.3.5 native behavior]
- **WebP raster-dem tile support in the native renderer.** `pmtiles extract` copies tiles verbatim,
  so the offline DEM archive contains Mapterhorn's `.webp` terrarium tiles (pmtiles header tile_type
  = webp). MapLibre GL native supports webp raster-dem, and the online style already renders these
  same webp tiles today — so parity is expected. The `pmtiles: ^1.2.0` Dart reader is format-agnostic
  (returns raw tile bytes; decoding is the native renderer's job). Flag for a device smoke test
  after wiring. [ASSUMED — relies on `maplibre` 0.3.5 / MapLibre-native webp raster-dem decode;
  not independently verified this session]
- **maxzoom clamp mismatch.** As with vector (rewriter comment lines 141-148), the DEM source's
  `maxzoom` in the style must equal the Go `--maxzoom` DEM cap, else MapLibre requests z13+ DEM
  tiles that were never extracted → blank relief above the cap instead of overzoom. Keep a dedicated
  DEM constant and the Go `demMaxZoom` in lockstep.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | DEM `--maxzoom=12` is a good size/detail tradeoff | 1 | Larger archives (raise) or coarse relief (lower); tunable, low risk |
| A2 | MapLibre native default raster-dem encoding is `mapbox`, so `terrarium` must be set | 4 | If default already terrarium, injection is a harmless no-op |
| A3 | `maplibre` 0.3.5 native renders webp terrarium raster-dem from a local pmtiles archive | 4 | Hillshade renders blank/wrong offline; needs device smoke test |

## Sources

- **Primary (HIGH):** codebase — `db/services/tiles/generator.go`, `grid.go`,
  `db/routes/map_cells.go` + `map_cells_id.go`, `db/main.go:213-220`,
  `db/migrations/1778840749_created_tile_cells.go`, `app/assets/map/wanderer_light.json:1-16,477-480`,
  `app/lib/services/trail_download_service.dart`, `app/lib/util/offline_style_rewriter.dart`,
  `app/lib/entities/trail_entity.dart`, `app/pubspec.yaml` (`maplibre: 0.3.5`, `pmtiles: ^1.2.0`).
- **Secondary (MEDIUM):** CONTEXT.md canonical refs — Mapterhorn tilejson (`terrarium`, `tileSize:512`),
  `download.mapterhorn.com/planet.pmtiles` reachability.

## Ready for Planning

Research complete. The change is a symmetric duplication of the existing vector-cell pipeline
across four layers (Go extract, migration, Flutter download, style rewrite) plus one explicit
raster-dem special case. Confirm A2/A3 with a quick device render test during execution.
