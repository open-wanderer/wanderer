---
title: Region catalog & archive generation — decision trail
date: 2026-07-21
context: v1.6 Offline Region Tile Repository, Phase 22 planning follow-up
---

## What happened

Phase 22 (Region & Package Data Model) was researched, pattern-mapped, planned, and verified assuming the existing per-cell backend (`db/services/tiles/generator.go`, `GET /api/v1/map/cells?bbox=...`, `db/routes/map_cells_id.go`) needed no changes — the manifest's `vector_url`/`dem_url` fields would point at that endpoint, and the client would fan out into N per-cell requests at region-download time. This was flagged in research as an interpretation needing user confirmation, and confirmation surfaced the opposite conclusion.

## Decision trail (in order discussed)

1. **Reuse the existing per-cell endpoint, client orchestrates N requests.** (Original Phase 22 plan assumption.) Rejected: forces the client to orchestrate, retry, and account progress/size across many small requests instead of one resumable file; doesn't match REGN-01's literal "one URL per region" wording.

2. **Backend generates a single mosaicked archive per region, on-demand at first user request, then caches it.** Considered next. Rejected as the final design: first-request latency is unpredictable (seconds for a small region, much longer for something like Colorado), which would force the Flutter download engine (Phase 23) to handle a "still preparing" polling state that a purely file-download UX shouldn't need.

3. **Backend pre-builds every region's archive via a cronjob, ahead of any request — but which regions?** Landed here. A global bundled list (regions.json shipped in the app) doesn't fit a self-hostable app: each instance's admin should decide what their instance offers, and a small instance shouldn't have to store/generate archives for regions its userbase will never touch.

4. **Final design:** An admin defines their instance's regions in a config file, mounted via Docker volume (not a UI/API for CRUD — that's deferred, see REQUIREMENTS.md Out of Scope). A cronjob pre-builds one vector PMTiles archive and one DEM archive per configured region, only regenerating when source tiles changed. The app fetches the resulting catalog (id, name, bbox, archive URL + size per region) from a new API endpoint at runtime — replacing the bundled `regions.json` asset entirely.

## Why this won

- Downloads are instant from the user's perspective — no generate-then-wait UX to design.
- Storage and generation cost scale with what an instance's admin actually wants to offer, not a fixed global catalog.
- No new admin-facing UI/API surface needed for v1.6 — a config file redeploy is enough, keeping the milestone scoped.
- It's the natural pull-forward of REGN-F04 ("remote/updatable region manifest"), which had already been identified as inevitable, just deferred.

## What's still open

Cron cadence and the exact staleness-detection mechanic (how the cronjob knows a region's source tiles changed since the last build, and how that surfaces as `updateAvailable` to the client) were explicitly deferred to Phase 21.5's `discuss-phase` step rather than decided in this exploration.

## Consequences for already-planned work

Phase 22's plans (22-01-PLAN.md, 22-02-PLAN.md) were drafted and passed plan-check before this correction. They haven't executed. 22-01 (bundled asset + parse model) needs a replan; 22-02 (ObjectBox entities) is likely still structurally valid but should be re-verified once 22-01's replan settles the manifest's actual shape (API-fetched fields may differ slightly from the bundled-asset schema). See `.planning/todos/pending/replan-phase-22-region-manifest.md`.
