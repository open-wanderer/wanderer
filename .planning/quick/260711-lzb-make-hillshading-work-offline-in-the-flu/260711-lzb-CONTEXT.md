# Quick Task 260711-lzb: Make hillshading work offline in the Flutter app - Context

**Gathered:** 2026-07-11
**Status:** Ready for planning

<domain>
## Task Boundary

Make hillshading work offline in the Flutter app. Currently `app/assets/map/wanderer_light.json` and `wanderer_dark.json` define `hillshadeSource` as a `raster-dem` source pointing at `https://tiles.mapterhorn.com/tilejson.json` (Mapterhorn, XYZ tile API, `.webp` terrarium-encoded tiles, `tileSize: 512`, no pmtiles archive available). This works online but is not currently downloaded/cached for offline trail viewing, and is in fact actively broken offline today (see decisions below).

</domain>

<decisions>
## Implementation Decisions

### DEM tile caching architecture (Go backend)
- **Correction (supersedes initial discussion):** Mapterhorn publishes a downloadable global pmtiles archive at `https://download.mapterhorn.com/planet.pmtiles`, confirmed reachable with `Accept-Ranges: bytes` (HTTP range request support via Cloudflare) — the same shape as `build.protomaps.com`'s vector archive already used by `EnsureCell`. The `https://tiles.mapterhorn.com/tilejson.json` XYZ endpoint referenced in the style JSON is a separate *online-serving* API; it is not the source used for offline extraction.
- Extend the existing per-cell tile generation pipeline (`db/services/tiles/generator.go`, `EnsureCell`) to run a second `pmtiles extract` invocation per cell — same pattern as the existing vector extraction, just pointed at `download.mapterhorn.com/planet.pmtiles` with the cell's bbox — producing a companion per-cell DEM `.pmtiles` archive. No custom Go pmtiles-writing library, tile-fetch loop, or archive assembly is needed; the existing `pmtiles extract` CLI dependency already handles this.
- Cap the extraction zoom for the DEM archive lower than the vector basemap's z14 (hillshading doesn't need that much detail) — exact value left to planning/implementation.
- Rejected: per-tile proxy/cache endpoint and manual archive-writing approaches — unnecessary now that direct bbox extraction from a remote pmtiles source is available, mirroring the existing vector pipeline exactly.

### Offline download scope
- Bundle DEM tile download with the existing per-trail download flow (`TrailDownloadService.downloadTrail` in `app/lib/services/trail_download_service.dart`), consistent with how vector basemap `.pmtiles` cells are downloaded today. No separate/optional toggle.

### Existing style-rewriter bug (in scope)
- `app/lib/util/offline_style_rewriter.dart` currently mishandles `hillshadeSource`: because it has a `url` key, `_rewriteSourcesAndLayers` treats it as a generic tiled source and incorrectly repoints it at the local vector `.pmtiles` archive (wrong data type) whenever a trail is downloaded and viewed offline today. Fixing this — so `hillshadeSource` (or any `type: raster-dem` source) is special-cased to point at its own local DEM `.pmtiles` archive instead — is in scope for this task.

### Claude's Discretion
- Exact capped max zoom level for offline DEM tiles.
- Exact PocketBase schema changes needed to `tile_cells` (or a new collection) to track DEM archive status alongside the existing vector cell status.

</decisions>

<specifics>
## Specific Ideas

No specific requirements beyond the above — open to standard implementation approaches within the locked architecture.

</specifics>

<canonical_refs>
## Canonical References

- `db/services/tiles/generator.go` — existing `EnsureCell` vector tile generation/caching pipeline (pattern to extend)
- `db/routes/map_cells.go` — existing `GET /map/cells?bbox=` grid-cell status/list endpoint
- `app/lib/services/trail_download_service.dart` — existing per-trail offline download flow
- `app/lib/util/offline_style_rewriter.dart` — existing offline style source rewriting (has the hillshadeSource bug)
- `app/assets/map/wanderer_light.json`, `wanderer_dark.json` — style definitions with `hillshadeSource`
- Mapterhorn tilejson (online serving, NOT the extraction source): `{"tilejson":"3.0.0","scheme":"xyz","tiles":["https://tiles.mapterhorn.com/{z}/{x}/{y}.webp"],"encoding":"terrarium","tileSize":512,"bounds":[-180,-85.0511287,180,85.0511287]}`
- Mapterhorn downloadable planet archive (offline extraction source): `https://download.mapterhorn.com/planet.pmtiles` — confirmed via `curl -sI` to return `HTTP 200`, `accept-ranges: bytes`, `content-length: 705726897585` (~700GB), served via Cloudflare. Usable directly with `pmtiles extract --bbox=... https://download.mapterhorn.com/planet.pmtiles <output>`, same CLI pattern already used in `db/services/tiles/generator.go` for the vector basemap.

</canonical_refs>
