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

## Still open

- Exact provider index to snapshot and its format (see `.planning/research/questions.md`).
- When/whether to automate refreshing the seeded list (see seed `region-list-refresh-mechanism`).
