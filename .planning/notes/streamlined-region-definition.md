---
title: Streamlined region definition — admin picks from a curated, seeded catalog
date: 2026-07-24
context: v1.6 Offline Region Tile Repository follow-up; un-defers the admin region UI that region-catalog-backend-decision-trail parked in a config file
---

## Problem

Today a server owner hand-authors `region_config.json` to define downloadable regions (see `region-catalog-backend-decision-trail.md`, which deliberately deferred any CRUD UI to a config file). This is cumbersome and error-prone: the admin must research and define bboxes themselves, and hand-drawn regions can overlap significantly. The decision to keep the admin in control stands — region choice corresponds directly to the instance's available hardware/storage — so the goal is not to *remove* the decision but to remove the *research and the bbox arithmetic* around it.

## Options considered

1. **Curated catalog + toggle (CHOSEN).** Seed a table with all regions derived from existing providers' extract trees (OsmAnd / CoMaps / Geofabrik lineage). The admin picks from this list via a UI. Because each entry corresponds to a *real, published extract*, the bbox is canonical, the size is known exactly, and "did I define this right?" disappears. Overlap ambiguity is gone (predefined, non-overlapping tree) and visible on a map.

2. **Freehand map draw.** Admin draws arbitrary rectangles/polygons. Maximum precision, but re-inherits every original pain: the admin must know what to draw, and you lose the "known extract = known size" guarantee.

**Why option 1 won:** Admins think in *places* ("cover this area"), and the provider trees already slice the world into named places at country / sub-country granularity — plenty for the broad instances we expect. The residual ceiling (can't pick an area smaller than any predefined region) was judged acceptable; freehand was not worth the guesswork it reintroduces. Storage-dedup and overlap-warning were explicitly ruled *not* real concerns.

## Chosen design

- **New `regions` table (PocketBase), seeded via migration.** Nested / hierarchical (parent–child, e.g. Europe › Germany › Bavaria), each row carrying its canonical bbox and an `enabled` flag. Master list is a **static snapshot** of a provider extract index; refreshing the "official" list is a deliberate patch (a new migration/release), not an automatic fetch.
- **Admin action = toggle `enabled`.** Nothing else. The existing archive-generation **cron job reads `enabled = true` on its next run** and pre-builds archives for those regions — replacing `region_config.json` parsing entirely.
- **Custom PocketBase admin page** for management, reusing the AlpineJS-bundle pattern already built on branch `feature/ap-instance-actors` (custom PB page + small bundled interactive UI). The page shows a **collapsible region tree** (toggle enabled per node) alongside a **live map rendering the bboxes** of enabled regions, so coverage is visible before commit.
- **App side must also change.** The Flutter settings screen currently lists downloadable regions **flat**; it needs to transition to the same **hierarchy** so the on-device download UX mirrors the admin-defined tree.

## Why this kills the original pains

- No research — admin picks from real extracts.
- No bbox arithmetic — bboxes are predefined and canonical.
- No overlap guesswork — predefined tree, visualized on the map.

## Reuse signals

- `feature/ap-instance-actors` — precedent for a custom PocketBase page with a bundled AlpineJS UI. This is the template for the region-picker page; not greenfield tooling.

## Provider source confirmed: CoMaps (2026-07-24)

Resolved the "exact provider index" open question from below. Source: [comaps/comaps](https://codeberg.org/comaps/comaps) (Codeberg), a fork of Organic Maps / Maps.me.

- **`data/countries.txt`** — JSON tree, the structural source of truth. Each node is either a **group** (has a `"g"` array of children, no size/hash of its own — e.g. `Germany`) or a **leaf/downloadable unit** (has `"s"` (size) + `"sha1_base64"`, no `"g"` — e.g. `Germany_Bavaria`, or `Luxembourg` for an undivided country). This distinction maps directly to a `kind: group | leaf` field.
- **`data/hierarchy.txt`** — flat, indentation-encoded tree carrying `Name;WikidataID;ISOcode;languages` — the source for human-readable display names (countries.txt's own ids are slugs like `Germany_Bavaria`, not display strings).
- **`data/borders/*.poly`** — per-leaf boundary polygons in the Osmosis `.poly` plaintext format (not GeoJSON) — geometry only exists for leaf nodes, not groups.
- **`countries.txt` itself is a generated build artifact** (sizes/hashes from CoMaps' own `.mwm` generation pipeline, `tools/python/maps_generator` + C++ `generator_tool`) — we do **not** run that pipeline (needs multi-TB disk + a planet.osm mount). What we actually snapshot is the much smaller, hand-curated `hierarchy.txt` + `.poly` files, which only change via manual PRs (see `data/HOW_TO_EDIT_HIERARCHY.TXT.md`-style workflow) — rare churn, confirming the "static snapshot, deliberate-patch refresh" design below was the right call, not something that needs automation.

## Licensing (2026-07-24)

CoMaps app code is Apache 2.0, but the geographic data (`countries.txt`/`hierarchy.txt`/`.poly` boundaries) has no separate license header — CoMaps' own `data/copyright.html` attributes it to **OpenStreetMap contributors under ODbL**, same as the rest of Wanderer's map data (Protomaps/Mapterhorn). Treat this as one more ODbL-sourced dataset covered by whatever attribution mechanism Wanderer already uses for its existing OSM-derived tiles — not a new compliance category. If reshaped into our own derivative dataset (which it is, per the schema below), keep it attributed/share-alike.

## Table schema — flattening the hierarchy (2026-07-24)

PocketBase has no recursive-CTE query support, so a bare adjacency list (`parent` pointer alone) would make "give me this whole subtree" expensive. Design:

| Field | Type | Purpose |
|---|---|---|
| `comaps_id` | text, unique | Stable external id (matches CoMaps' own naming) — the anchor for diffing future hierarchy refreshes |
| `parent` | relation (self) | Adjacency-list parent pointer, nullable for roots |
| `path` | text | Materialized path (e.g. `europe.germany.thuringia`) — cheap "all descendants" via prefix match |
| `depth` | number | Tree level, for indent rendering in the collapsible admin UI |
| `sort_order` | number | Preserves CoMaps' own sibling ordering |
| `name` | text | Display name, from `hierarchy.txt` |
| `kind` | select (`group` \| `leaf`) | From `countries.txt`'s `g`-array-vs-`s`/`sha1_base64` marker |
| `polygon` | JSON (GeoJSON) | **Leaf only.** Canonical boundary — the `pmtiles extract --region` input |
| `bbox` | JSON or 4 numbers | **Leaf only.** Derived from `polygon` — served to the app catalog for the client's existing bbox-based coverage math (Phase 26 guard etc.) |
| `enabled` | bool | **Leaf only** — cron reads `kind = 'leaf' AND enabled = true`, replacing `region_config.json` parsing entirely |

Group rows exist purely so the admin UI can render correct tree nesting — no geometry, no `enabled` semantics. Map preview only ever renders individual enabled **leaf** outlines; no aggregate/union boundary is computed for collapsed group nodes.

**Client stays bbox-only** (already decided, for efficiency) — the server holds the canonical polygon for accurate extraction, but the app-facing catalog (`GET /api/v1/regions`) continues to serve just the derived bbox, unchanged from the existing client-side coverage-check code.

## Extraction mechanism (2026-07-24)

`pmtiles extract` already supports a `--region` flag accepting a polygon (not just bbox) — using a real leaf polygon instead of its bounding box avoids pulling in the dead-weight of irregular/coastal shapes (a bbox around Norway's coastline, for example). This is a small, well-bounded backend change: point an existing pmtiles capability at better input geometry, not new clipping logic to build.

## Seeding tool (2026-07-24)

Two pieces, both matching this repo's existing Go/PocketBase conventions (not Python — the repo has zero Python footprint; Go matches both the migration language and the simple text-parsing/bbox-arithmetic the task actually needs):

1. **`db/commands/seed_regions.go`** — a maintainer-run, dev-time Cobra command (same pattern as `db/commands/dedup.go`: `*pocketbase.PocketBase` + `cobra.Command`). Run only by a Wanderer maintainer, only when CoMaps' hierarchy changes upstream — never by a self-hosted operator. Parses vendored `hierarchy.txt` + `.poly` files, converts `.poly` → GeoJSON, computes `path`/`depth`/`sort_order`/`bbox`, and writes the flattened result out as a **committed JSON seed file** (not a live `app.Save()` write against a running database).
2. **A normal schema migration** (`db/migrations/*.go`, same convention as every other migration here) creates the `regions` collection and bulk-inserts from that committed JSON seed file. Runs automatically on every instance startup like all the others — this is what actually satisfies "a new seeded `regions` table exists" for a fresh self-hosted install with zero admin action.

Refresh flow when CoMaps' hierarchy changes: re-run `seed_regions.go` → review the JSON diff → commit → ship as a normal migration in a release. Matches "deliberate patch, not automatic fetch" exactly; never touches a live production DB automatically. See seed `region-list-refresh-mechanism` for when this manual flow should be reconsidered/automated.

Raw CoMaps source files are **not** vendored wholesale — only referenced by URL + commit hash for provenance in this note. What's committed to this repo is the transform tool's *output* (the flattened JSON), not the raw upstream `.poly`/`hierarchy.txt` inputs.

## Still open

- Exact vendoring path/location for the raw CoMaps snapshot the tool reads from during a refresh run (a `db/tools/` scratch dir vs. fetched fresh from Codeberg at tool-run time, pinned to a commit hash) — implementation detail for Phase 28 planning, not blocking.
- Precise `.poly` → GeoJSON parsing approach in Go (small well-known format; find/adapt an existing minimal parser rather than hand-rolling one from scratch) — implementation detail for Phase 28 planning.
- Whether a spike is needed to prove `pmtiles extract --region <polygon>` behaves as expected against a real CoMaps `.poly`-derived polygon before Phase 28 planning commits to the approach (see pending todo).
