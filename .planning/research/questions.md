# Research Questions

## Region catalog source for seeded `regions` table (added 2026-07-24) — RESOLVED 2026-07-24

Blocked the seed migration for the streamlined region-definition feature (see `.planning/notes/streamlined-region-definition.md`, "Provider source confirmed" section for full detail). Answered via `/gsd-explore` session:

- **Which provider index to snapshot?** **CoMaps** ([codeberg.org/comaps/comaps](https://codeberg.org/comaps/comaps)) — chosen over OsmAnd/Geofabrik.
- **Does the index publish bbox/polygon, parent/child, and size?** `data/countries.txt` (JSON) publishes parent/child (via a `g` children array) and size (`s`/`sha1_base64`, leaf nodes only) — but sizes are for CoMaps' own `.mwm` format, not directly usable for our PMTiles archive sizes. `data/hierarchy.txt` publishes display names/ISO codes. `data/borders/*.poly` publishes canonical **polygon** boundaries (leaf nodes only, Osmosis `.poly` format, not GeoJSON — needs conversion) — richer than a bare bbox, and directly usable with `pmtiles extract --region`.
- **Licensing:** ODbL (OpenStreetMap-derived), per CoMaps' own `data/copyright.html`. Redistributable as a derivative dataset under the same existing attribution mechanism Wanderer already uses for its other OSM-derived tile data — not a new compliance category, but must stay attributed/share-alike.
- **Granularity:** Country-and-below where CoMaps' extract tree defines sub-divisions (e.g. Germany → Bavaria, Thuringia, ...); country-level where it doesn't (e.g. Luxembourg is itself the leaf). Matches the "cover this place" mental model.
- **Format for seeding:** No single ready-to-seed file — combine `countries.txt` (structure + leaf/group marker) + `hierarchy.txt` (names) + `.poly` files (leaf geometry) via a one-time transform tool. See `.planning/notes/streamlined-region-definition.md`'s "Seeding tool" section for the full `db/commands/seed_regions.go` → committed JSON → migration design.
