---
created: 2026-07-24T17:41:31.000Z
title: Prototype pmtiles extract --region with a CoMaps .poly-derived polygon
area: backend
files:
  - db/services/tiles/generator.go
  - .planning/notes/streamlined-region-definition.md
---

## Problem

The admin region picker design (`.planning/notes/streamlined-region-definition.md`) commits to server-side polygon-based extraction: instead of a bbox, each leaf region's `pmtiles extract` call would use the region's canonical boundary polygon (converted from CoMaps' `data/borders/*.poly` Osmosis-format files to GeoJSON), via `pmtiles extract`'s existing `--region` flag. This has not been proven end-to-end — only confirmed that the flag exists and accepts a polygon.

## Solution

Before Phase 28 planning commits to the polygon-extraction approach:

1. Pick one CoMaps `.poly` file (e.g. `Germany_Thuringia.poly`) and hand-convert it to GeoJSON (no need for the full Go conversion tool yet — a throwaway script or manual conversion is enough for this spike).
2. Run `pmtiles extract --region <geojson>` against the existing Protomaps source (same source `db/services/tiles/generator.go` already extracts from) and confirm:
   - The command succeeds and produces a valid PMTiles archive.
   - The output only contains data within the polygon boundary (not the polygon's bounding box) — i.e. confirm polygon clipping actually happens, not just a bbox pre-filter.
   - Archive size is smaller than the equivalent bbox-based extract for an irregularly-shaped region (a real, measurable win, not just theoretical).
3. Note any gotchas: coordinate order/winding requirements, multi-ring polygon support (some regions may have holes or be multi-part), performance vs. bbox extraction for a region of similar size.

## Verification

- A real `.pmtiles` archive is produced from a polygon input.
- Spot-check a few tile coordinates near the polygon boundary (but outside it) confirm they're absent from the output, proving true polygon clipping vs. bbox pre-filtering.
