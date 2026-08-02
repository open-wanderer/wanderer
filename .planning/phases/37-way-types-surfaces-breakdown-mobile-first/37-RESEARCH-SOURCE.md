# Way Types & Surfaces breakdown for trails

## Context

Komoot shows, for any GPX track, a breakdown of **way types** (Hiking Path, Path, Footpath, Road, Street…) and **surfaces** (Unpaved, Gravel, Asphalt, Paved…) as a stacked bar + legend with distance per category. We want to replicate this for Wanderer trails using **public, free APIs** — ideally the infrastructure Wanderer already uses.

The obvious source is **Valhalla `trace_attributes`** (map-matching), which Wanderer already points at via the free public FOSSGIS server (`VALHALLA_URL: https://valhalla1.openstreetmap.de`). It map-matches a track and returns per-edge `use` (way type), `surface`, and `length`. **This is confirmed working on the public server** — no new API, no cost.

### The POC bottleneck — root-caused and solved

The earlier POC failed on hiking trails that don't follow major roads. I reproduced this against a real alpine trail (OSM way 39669166, `sac_scale=mountain_hiking`, `surface=rock`, 73 points):

| Request | Matched length |
|---|---|
| Default `pedestrian` costing | **0.016 km** (trail dropped) |
| `trace_options` tuned only (search_radius/gps_accuracy) | 0.016 km (no change) |
| **`costing_options.pedestrian.max_hiking_difficulty: 6`** | **1.088 km** (full match) ✅ |

**Root cause:** default pedestrian costing uses `max_hiking_difficulty ≈ 1`, which **excludes `sac_scale ≥ mountain_hiking` paths from the routable graph**, so map-matching has no candidate edge to snap to and silently drops those segments — exactly "trails that don't go along major roads." Raising `max_hiking_difficulty` to `6` (the max) makes those paths routable candidates and matching succeeds. This is a single-line fix in the costing options. (Wanderer's `ValhallaPedestrianCostingOptions` already exposes `max_hiking_difficulty`.)

**Known limitations (design into the feature, not blockers):**
- **Surface fidelity is coarse.** Valhalla normalizes OSM `surface` into 8 buckets (`paved_smooth, paved, paved_rough, compacted, dirt, gravel, path, impassable`) — e.g. the `surface=rock` trail came back as `path`. Way-type `use` likewise can't distinguish "Hiking Path" vs "Path" (both are `path`). So categories are approximations of komoot's exact OSM labels. Acceptable for v1; an optional Overpass enrichment (below) recovers exact tags later.
- **Unmatched gaps.** When a segment genuinely isn't in OSM, matched length < trail distance. Bucket the remainder as **Unknown** (komoot does the same — "Unknown: 183 m").
- **Fair-use.** The public server is best-effort/no-SLA. Compute **once on save and persist**; throttle any bulk backfill.

## Decisions (confirmed with user)
- **Data source:** Valhalla `trace_attributes` only (with the `max_hiking_difficulty` fix). No Overpass in v1.
- **Compute + store:** server-side, computed once on trail create/update, persisted as JSON on the trail record.
- **Client:** **Mobile-first — the Flutter app (`app/lib/`) is built first. The SvelteKit web UI follows afterwards as a separate phase** (same shared field, deferred; sketched at the end).
- **Deliverable:** full implementation plan (shared backend compute + persistence + Flutter trail-detail UI).

## Approach — compute in the web API on trail save, persist JSON, render in the Flutter app

**All clients (web + mobile) write trails through the SvelteKit web API — never directly to PocketBase — and Valhalla already lives in the web layer.** So the web trail create/update endpoint is the universal chokepoint: compute there, persist the result on the trail record, and every client (including the Flutter app, which just reads the field) gets it. **No Go hook and no backend Valhalla config** — the only backend change is a migration adding the field.

### 1. Backend — schema only (Go migration)

- **New migration** in [db/migrations/](db/migrations/) adding a `json` field `way_type_surface` to the `trails` collection (`e864strfxo14pm4`). Follow `1781900000_added_dem_fields_to_tile_cells.go` / `1778583800_persist_trail_bounds.go` (`AddMarshaledJSONAt` + down-migration). No hook logic, no `VALHALLA_URL` in the backend, no `db`-service compose change.

### 2. Web API — Valhalla trace proxy + compute on trail save (TypeScript, server-side)

All in the web layer where Valhalla config already resolves (`resolveBaseUrl` in [web/src/lib/server/url.ts](web/src/lib/server/url.ts), getters in [web/src/lib/server/valhalla.ts](web/src/lib/server/valhalla.ts)):

- **New Valhalla URL key + getter:** add `VALHALLA_TRACE_ATTRIBUTES_URL` to `ExternalServiceUrlKey` in `url.ts` and `getValhallaTraceAttributesUrl()` in `server/valhalla.ts`, mirroring the existing route/height/navigate getters (falls back to the configured Valhalla base).
- **New server helper `computeWayTypeSurface(shape)`** (e.g. `web/src/lib/server/way_types.ts`):
  1. Takes the track coordinates, **downsamples** to ~1000–1500 points (min spacing) — keeps payload small, within the public server's trace limits, and matching robust.
  2. POSTs to `<valhalla>/trace_attributes` with the map-matching fix that resolves the POC bottleneck:
     ```json
     {"shape":[{"lat":..,"lon":..}, ...],
      "costing":"pedestrian",
      "costing_options":{"pedestrian":{"max_hiking_difficulty":6,"use_hills":1}},
      "shape_match":"map_snap",
      "trace_options":{"search_radius":50,"gps_accuracy":10,"breakage_distance":2000},
      "filters":{"attributes":["edge.use","edge.surface","edge.length"],"action":"include"}}
     ```
  3. Aggregates: sum `edge.length` (km) grouped by mapped **way-type** and **surface** category; `unknownKm = max(0, trailDistanceKm - Σ edge.length)` → **Unknown** bucket in both breakdowns. Returns `{ way_types: [{key,meters}], surfaces: [{key,meters}] }`.
- **Category mapping** (constant maps, tunable):
  - Way type from `edge.use`: `footway|sidewalk|pedestrian_crossing → footpath`, `path → path`, `track → track`, `steps → steps`, `cycleway → cycleway`, `road|driveway|alley → road`, else `other`.
  - Surface from `edge.surface`: `paved_smooth|paved|paved_rough → paved`, `compacted → compacted`, `gravel → gravel`, `dirt → dirt`, `path → unpaved`, `impassable|<gap> → unknown`.
- **Wire into the trail create/update handler** in [web/src/routes/api/v1/trail/+server.ts](web/src/routes/api/v1/trail/+server.ts) (and `[id]/+server.ts`): when a GPX is present/changed, parse it (reuse `GPX.parse` / `getTotals().cumulativeDistance` from [web/src/lib/models/gpx/gpx.ts](web/src/lib/models/gpx/gpx.ts) for coords + total distance), call `computeWayTypeSurface`, and write `way_type_surface` onto the record. **Best-effort with a short timeout** — if Valhalla is unreachable, log and save the trail anyway (field stays empty, recomputed on next save). Only recompute when the GPX changed, to avoid redundant external calls.
  - *Latency note:* this adds one Valhalla round-trip to trail save (~0.5–2 s). If that proves too slow, split it into a dedicated `POST /api/v1/trail/[id]/way-types` endpoint that the save flow calls fire-and-forget — still entirely web-API. Start synchronous for simplicity.

### 3. Frontend — Flutter model + render (mobile-first; the primary UI deliverable)

Uses **freezed + json_serializable** (like the rest of the app). After model edits, **regenerate** with `dart run build_runner build --delete-conflicting-outputs` — never hand-edit `*.freezed.dart` / `*.g.dart`.

- **Model:** in [app/lib/models/trail.dart](app/lib/models/trail.dart), add two new `@freezed` classes — `WayTypeSurface` (`{ List<WayTypeSegment> wayTypes; List<WayTypeSegment> surfaces; }`, with `@JsonKey(name: 'way_types')`/`'surfaces'`) and `WayTypeSegment` (`{ String key; double meters; }`) — following the existing nested-object precedent (`TrailExpand`). Add to `Trail`:
  ```dart
  @JsonKey(name: 'way_type_surface') WayTypeSurface? wayTypeSurface,
  ```
- **UI widget:** new `app/lib/components/trail/way_type_surface_section.dart` (a `ConsumerWidget`), inserted into the **"About" tab `Column`** of [app/lib/components/trail/trail_panel.dart](app/lib/components/trail/trail_panel.dart), right after the `ElevationProfile` block. Reuse the existing section-header style (`titleMedium` + `FontWeight.bold`, as the "Route" header). Layout per screenshot, one block each for Way Types and Surfaces:
  - **Stacked bar:** `ClipRRect` wrapping a `Row` of `Expanded(flex: meters, child: Container(color: catColor))` segments (flex proportional to meters). No fl_chart needed — matches existing `Row`/`Expanded` usage.
  - **Legend rows:** swatch + label + `formatDistance(meters, unit: unit)` (from [app/lib/util/format_util.dart](app/lib/util/format_util.dart); `unit` via `ref.watch(unitProvider)`, as `trail_panel.dart` already does). Reuse `StatChip` styling for the rows if convenient.
  - Render nothing if `wayTypeSurface` is null/empty (older trails until backfilled).
- **Category colors:** add a dedicated categorical map keyed by segment `key` to [app/lib/theme/colors.dart](app/lib/theme/colors.dart) (the `ColorScheme` is monochrome and has no categorical ramp — follow the `_gradientColor` hardcoded-`Color` precedent in `elevation_profile.dart`). Same map drives both bar segments and legend swatches.
- **i18n:** add keys to [app/lib/i18n/app_en.arb](app/lib/i18n/app_en.arb) (+ sibling locale ARBs): `way_types`, `surfaces`, and value labels (`way_type_path`, `way_type_footpath`, `surface_paved`, `surface_gravel`, `surface_unpaved`, `surface_unknown`, …); reference via `AppLocalizations.of(context)!`. Regenerated by build (`generate: true`).

### 4. Web (SvelteKit) UI — deferred follow-up phase (not built now)
The compute already lives in the web API (§2), so this phase is **rendering only**: read the persisted `way_type_surface` field ([web/src/lib/models/trail.ts](web/src/lib/models/trail.ts)) and render it in the trail-detail **Route section** of [web/src/lib/components/trail/trail_info_panel.svelte](web/src/lib/components/trail/trail_info_panel.svelte) — a CSS flexbox stacked bar + legend, `formatDistance` from [web/src/lib/util/format_util.ts](web/src/lib/util/format_util.ts), i18n keys in [web/src/lib/i18n/locales/en.json](web/src/lib/i18n/locales/en.json). Do this after the mobile feature ships.

### Optional future enhancement (not v1): exact OSM tags via Overpass
Add `edge.way_id` to the trace filter, dedupe the returned way IDs, and batch-query Overpass for raw `highway`/`surface`/`sac_scale` tags to reproduce komoot's exact categories (Hiking Path vs Path, rock/scree surfaces, etc.). Adds a rate-limited public API — keep it opt-in and cached.

## Files to change

**Backend (schema only, built first):**
- **New:** `db/migrations/<ts>_added_way_type_surface_to_trails.go` (adds the `way_type_surface` json field). No hook, no Valhalla client, no compose/env change.

**Web API (compute, built with backend):**
- **New:** `web/src/lib/server/way_types.ts` (Valhalla trace call + aggregation)
- **Edit:** [web/src/lib/server/url.ts](web/src/lib/server/url.ts) + [web/src/lib/server/valhalla.ts](web/src/lib/server/valhalla.ts) (trace_attributes URL key + getter), [web/src/routes/api/v1/trail/+server.ts](web/src/routes/api/v1/trail/+server.ts) + `[id]/+server.ts` (wire compute into create/update)
- **Reuse (no change):** `GPX.parse` / `getTotals().cumulativeDistance` ([web/src/lib/models/gpx/gpx.ts](web/src/lib/models/gpx/gpx.ts)), existing Valhalla proxy pattern ([web/src/routes/api/v1/valhalla/route/+server.ts](web/src/routes/api/v1/valhalla/route/+server.ts))

**Mobile / Flutter (the primary UI deliverable):**
- **New:** `app/lib/components/trail/way_type_surface_section.dart`
- **Edit:** [app/lib/models/trail.dart](app/lib/models/trail.dart) (+ regenerate `trail.freezed.dart` / `trail.g.dart` via build_runner), [app/lib/components/trail/trail_panel.dart](app/lib/components/trail/trail_panel.dart), [app/lib/theme/colors.dart](app/lib/theme/colors.dart), [app/lib/i18n/app_en.arb](app/lib/i18n/app_en.arb) (+ sibling ARBs)
- **Reuse (no change):** `formatDistance` ([app/lib/util/format_util.dart](app/lib/util/format_util.dart)), `unitProvider`, `StatChip`, `ElevationProfile` patterns

**Web / SvelteKit (deferred follow-up phase):** `trail.ts`, a new `way_type_surface.svelte`, `trail_info_panel.svelte`, `web/src/lib/i18n/locales/en.json`.

> Note (config discrepancy found during research): code reads per-endpoint keys `VALHALLA_ROUTE_URL`/`HEIGHT`/`NAVIGATE` in `web/src/lib/server/valhalla.ts`, but compose files still set only the older `VALHALLA_URL`. Make `getValhallaTraceAttributesUrl()` fall back to `VALHALLA_URL` (the key actually present in deployments) so the new endpoint works out of the box, matching the existing `resolveBaseUrl` fallback chain.

## Verification
1. **Reproduce & confirm the fix (already done, re-runnable):** POST the alpine trail (OSM way 39669166) geometry to `https://valhalla1.openstreetmap.de/trace_attributes` with `max_hiking_difficulty:6` → expect a full ~1.09 km match vs 0.016 km without it.
2. **Web API unit:** feed a known GPX (a road walk + an alpine path) through `computeWayTypeSurface`; assert non-trivial matched length, sane category buckets, and that the trail-distance remainder lands in **Unknown**.
3. **End-to-end (mobile):** run the backend + web (docker compose) and the Flutter app; create/edit a trail (through the web API, as all clients do) with a GPX that includes off-road hiking, confirm `way_type_surface` is persisted on the record (via PocketBase admin/API), and that the Flutter trail-detail "About" tab renders the stacked bar + legend with distances that sum to the trail length. Verify an older trail (no field) renders nothing (no crash).
4. **Regression:** confirm trail save still succeeds when Valhalla is unreachable (best-effort: field empty, save completes, error logged) — existing user federation unaffected.
