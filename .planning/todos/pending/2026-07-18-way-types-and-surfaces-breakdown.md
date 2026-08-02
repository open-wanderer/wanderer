---
created: 2026-07-18T13:47:01.903Z
title: Way Types & Surfaces breakdown feature (mobile-first)
area: mobile
files:
  - /Users/christianbeutel/.claude/plans/komoot-is-able-to-enchanted-truffle.md
  - app/lib/models/trail.dart
  - app/lib/components/trail/trail_panel.dart
  - app/lib/theme/colors.dart
  - app/lib/i18n/app_en.arb
  - web/src/lib/server/way_types.ts
  - web/src/lib/server/url.ts
  - web/src/lib/server/valhalla.ts
  - web/src/routes/api/v1/trail/+server.ts
  - db/migrations/
---

> **Promoted 2026-08-02 → Phase 37** (`.planning/phases/37-way-types-surfaces-breakdown-mobile-first/`).
> Unscheduled, outside v1.8, blocked on Phase 36 by file overlap. The session-local plan file
> referenced below has been rescued to `37-RESEARCH-SOURCE.md` in that directory.

## Problem

Komoot shows, for any GPX track, a "Way Types & Surfaces" breakdown: a horizontal stacked bar + legend listing OSM way types (Hiking Path, Path, Footpath, Road, Street…) and surfaces (Unpaved, Gravel, Asphalt, Paved, Unknown…) with distance per category. Wanderer has no equivalent. A prior POC attempted this with Valhalla `trace_attributes` map-matching and hit a bottleneck: off-road hiking trails (not along major roads) were not matched — segments silently dropped.

**Root cause (verified during this research session):** Valhalla's default `pedestrian` costing uses `max_hiking_difficulty ≈ 1`, which excludes `sac_scale >= mountain_hiking` paths from the routable graph entirely. With no candidate edge, map-matching drops those segments. Reproduced against a real OSM way (id 39669166, `sac_scale=mountain_hiking`, `surface=rock`, 73 points): default costing matched only 16 m; setting `costing_options.pedestrian.max_hiking_difficulty: 6` matched the full 1.09 km. Isolated to that single lever (tuning `trace_options` alone did nothing).

Full research + a complete file-level implementation plan already exist: **`/Users/christianbeutel/.claude/plans/komoot-is-able-to-enchanted-truffle.md`** (this is a session-local plan-mode file — read it and fold its content into this todo/phase before the plan file is lost to cleanup; it will not persist on its own).

## Solution

**Data source:** Valhalla `trace_attributes` (already configured for free via the public FOSSGIS server `valhalla1.openstreetmap.de` that Wanderer's web layer already points at) — no new API needed. Request `costing_options.pedestrian.max_hiking_difficulty: 6` to include alpine/off-road paths in matching. Filter response to `edge.use`, `edge.surface`, `edge.length`; aggregate matched length by category; bucket unmatched remainder (trail distance − Σ matched length) as **Unknown** in both breakdowns.

**Architecture (confirmed with user):** All clients (web + the Flutter mobile app) write trails through the SvelteKit **web API** — never directly to PocketBase — and Valhalla config already lives in the web layer. So **compute entirely in the web API** on trail create/update; **no Go/PocketBase hook, no `VALHALLA_URL` in the backend**. The only backend (Go) change is a migration adding a `way_type_surface` json field to the `trails` collection (`e864strfxo14pm4`).

**Client priority: mobile-first.** Build the Flutter app (`app/lib/`) UI first; the SvelteKit web UI is a deferred follow-up phase (rendering only, same persisted field).

### Backend (schema only)
- New migration in `db/migrations/` adding `way_type_surface` json field to `trails`. Follow `1781900000_added_dem_fields_to_tile_cells.go` / `1778583800_persist_trail_bounds.go` pattern (`AddMarshaledJSONAt` + down-migration).

### Web API (compute)
- Add `VALHALLA_TRACE_ATTRIBUTES_URL` to `ExternalServiceUrlKey` in `web/src/lib/server/url.ts` + getter `getValhallaTraceAttributesUrl()` in `web/src/lib/server/valhalla.ts` (fall back to `VALHALLA_URL`, the key actually present in compose files — there's a discrepancy where code reads `VALHALLA_ROUTE_URL`/`HEIGHT`/`NAVIGATE` but compose only sets `VALHALLA_URL`).
- New `web/src/lib/server/way_types.ts`: downsample track coords (~1000–1500 pts), POST to `<valhalla>/trace_attributes` with the request shape below, aggregate into `{ way_types: [{key,meters}], surfaces: [{key,meters}] }`.
  ```json
  {"shape":[{"lat":..,"lon":..}, ...],
   "costing":"pedestrian",
   "costing_options":{"pedestrian":{"max_hiking_difficulty":6,"use_hills":1}},
   "shape_match":"map_snap",
   "trace_options":{"search_radius":50,"gps_accuracy":10,"breakage_distance":2000},
   "filters":{"attributes":["edge.use","edge.surface","edge.length"],"action":"include"}}
  ```
- Category mapping (tunable): way type from `edge.use` — `footway|sidewalk|pedestrian_crossing→footpath`, `path→path`, `track→track`, `steps→steps`, `cycleway→cycleway`, `road|driveway|alley→road`, else `other`. Surface from `edge.surface` — `paved_smooth|paved|paved_rough→paved`, `compacted→compacted`, `gravel→gravel`, `dirt→dirt`, `path→unpaved`, `impassable|gap→unknown`.
- Wire into `web/src/routes/api/v1/trail/+server.ts` (+ `[id]/+server.ts`) create/update: when GPX present/changed, parse via `GPX.parse`/`getTotals().cumulativeDistance` (`web/src/lib/models/gpx/gpx.ts`), call `computeWayTypeSurface`, write `way_type_surface` onto the record. Best-effort with short timeout — Valhalla unreachable → log, save trail anyway, field stays empty. Only recompute when GPX changed.
- Latency note: if the extra Valhalla round-trip (~0.5–2s) proves too slow inline, split into a dedicated endpoint the save flow calls fire-and-forget.

### Mobile / Flutter (primary UI deliverable)
- `app/lib/models/trail.dart`: add freezed classes `WayTypeSurface { wayTypes, surfaces }` and `WayTypeSegment { key, meters }` (mirror `TrailExpand` nested-object precedent); add `@JsonKey(name: 'way_type_surface') WayTypeSurface? wayTypeSurface` to `Trail`. Regenerate via `dart run build_runner build --delete-conflicting-outputs` — never hand-edit `*.freezed.dart`/`*.g.dart`.
- New `app/lib/components/trail/way_type_surface_section.dart` (ConsumerWidget), inserted into the "About" tab `Column` of `app/lib/components/trail/trail_panel.dart`, right after `ElevationProfile`. Stacked bar: `ClipRRect` + `Row` of `Expanded(flex: meters, ...)` segments (no fl_chart needed). Legend: swatch + label + `formatDistance(meters, unit: unit)` (`app/lib/util/format_util.dart`, `unit` via `ref.watch(unitProvider)`). Render nothing if `wayTypeSurface` null/empty.
- Category colors: new map keyed by segment `key` in `app/lib/theme/colors.dart` (app's `ColorScheme` is monochrome, no categorical ramp — follow `_gradientColor` hardcoded-`Color` precedent in `elevation_profile.dart`).
- i18n: add keys to `app/lib/i18n/app_en.arb` (+ sibling locale ARBs): `way_types`, `surfaces`, `way_type_path`, `way_type_footpath`, `surface_paved`, `surface_gravel`, `surface_unpaved`, `surface_unknown`, etc.

### Web (SvelteKit) — deferred follow-up phase
Rendering only, same persisted field: `web/src/lib/models/trail.ts`, new `way_type_surface.svelte` rendered in the Route section of `trail_info_panel.svelte`, i18n keys in `web/src/lib/i18n/locales/en.json`.

### Optional future enhancement (not v1)
Add `edge.way_id` to the trace filter, batch-query Overpass for raw OSM tags (`highway`/`surface`/`sac_scale`) to reproduce komoot's exact categories (currently Valhalla only exposes 8 coarse surface buckets and can't distinguish "Hiking Path" vs "Path"). Keep opt-in/cached — Overpass is rate-limited.

### Verification
1. Reproduce the fix: POST OSM way 39669166 geometry to `https://valhalla1.openstreetmap.de/trace_attributes` with `max_hiking_difficulty:6` → expect ~1.09 km match vs 0.016 km without it.
2. Web API unit test: feed a GPX with a road walk + an alpine path through `computeWayTypeSurface`; assert sane category buckets and Unknown captures the remainder.
3. End-to-end (mobile): create/edit a trail via the web API with an off-road GPX, confirm `way_type_surface` persists, confirm Flutter "About" tab renders stacked bar + legend summing to trail length; older trail (no field) renders nothing.
4. Regression: trail save still succeeds when Valhalla is unreachable (best-effort); existing user federation unaffected.
