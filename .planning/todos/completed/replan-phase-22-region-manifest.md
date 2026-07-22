---
title: Replan Phase 22 (region manifest) for the new backend-fetched catalog design
date: 2026-07-21
priority: high
---

## What

Phase 22's existing plans (`22-01-PLAN.md`, `22-02-PLAN.md`) were planned and passed plan-check under the assumption that `regions.json` is a bundled Flutter app asset. Exploration after planning (see `.planning/notes/region-catalog-backend-decision-trail.md`) determined regions should instead be an admin-configured, per-instance catalog fetched from a new backend API (Phase 21.5) at runtime.

Neither plan has executed yet, so nothing needs to be reverted in code — but the plans as written build the wrong thing.

## What needs to change

- **22-01-PLAN.md**: currently scaffolds a bundled `assets/map/regions.json` asset + a parse model for it. Needs to become: an API client/fetch for Phase 21.5's catalog endpoint + a parse model for that response shape instead. The `RegionManifest`/`RegionManifestEntry` freezed types may be reusable if the API response shape matches closely — verify field-for-field once Phase 21.5's endpoint contract is planned (BACK-04).
- **22-02-PLAN.md**: ObjectBox `RegionEntity`/`DownloadedTilePackageEntity` schema is likely still valid as-is (it doesn't care whether the source data came from a bundled asset or an API fetch) — but re-verify `fromManifestEntry` against whatever type 22-01's replan produces.
- **22-CONTEXT.md** decisions D-03/D-04/D-05 (regions.json content, region-splitting convention, URL-sourcing) are now partially obsolete — D-05's "bbox-cells query endpoint" interpretation is superseded by Phase 21.5's single-archive-per-region design.

## When

Block on Phase 21.5 being planned first (its `discuss-phase`/`plan-phase` will settle the catalog API's exact response shape — field names, whether it's a REST endpoint or something else — which 22-01's replan needs to match).

## Suggested sequence

1. `/gsd-discuss-phase 21.5` then `/gsd-plan-phase 21.5` — lock the catalog API contract.
2. `/gsd-plan-phase 22 --reviews` or a fresh discuss-phase pass for 22-01 specifically, once the API contract is known.
3. Re-run plan-check on the replanned Phase 22 before executing.
