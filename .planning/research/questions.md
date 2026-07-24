# Research Questions

## Region catalog source for seeded `regions` table (added 2026-07-24)

Blocks the seed migration for the streamlined region-definition feature (see `.planning/notes/streamlined-region-definition.md`).

- **Which provider index to snapshot?** OsmAnd, CoMaps, and Geofabrik all publish hierarchical extract trees of Geofabrik lineage. Which one gives the cleanest machine-readable index for seeding?
- **Does the index publish, per region:** a canonical **bounding box (or polygon)**, an explicit **parent/child relationship** (for the nested tree), and a **download size**? Which of these are present vs. need to be derived?
- **Licensing:** Is the region metadata (names, bboxes, hierarchy) redistributable — i.e. can we bake a snapshot into a PocketBase migration shipped with the app? (Data itself is ODbL/OSM; confirm the *index/metadata* terms.)
- **Granularity:** What is the finest slice available (sub-country regions? counties?), and does that granularity match the "cover this place" mental model for our expected admins?
- **Format for seeding:** Is there a stable JSON/XML index we can parse into migration seed data, or would we need to scrape/transform it?
