# Wanderer Trail Navigation

## Current State

**Shipped:** v1.8 Offline Recording & Deferred Upload (2026-08-07) — a hiker who records a trail or
imports a GPX with no signal can save it, review it, and fill in its details on the spot, and it
uploads itself the next time the phone has a connection. The GPX→trail metrics computation was
corrected (CONV-01…06) and then ported to Dart, pinned to the TypeScript by a ten-fixture on-disk
corpus both languages read; `POST /api/v1/trail/convert` is transcode-only. Locally-captured trails
carry a collision-free local id, owning account, sync state, and app-owned photo storage, appear
in `/profile/<handle>/trails` behind a four-state sync chip, and are drained by a resume-from-step
uploader that cannot duplicate a trail. Phases 38/38.1 additionally separated downloaded-trail
*state* from trail *identity*: *Remove download* and *Delete trail* are distinct actions derived
from library membership and authorship rather than `Trail.isLocal`.

**Previously shipped:** v1.7 Admin Region Picker (2026-07-28) — a server owner defines downloadable
regions by toggling entries in a seeded 1,306-row CoMaps catalog with real polygon boundaries.

**Known gaps carried forward** (accepted at close):
- Phase 36 has an unresolved VERIFICATION.md (`human_needed`) and one pending UAT scenario — the
  only two v1.8-scoped items among the 51 deferred at close (see STATE.md → Deferred Items — v1.8 Close)
- No `v1.8-MILESTONE-AUDIT.md` was run before close, unlike v1.7
- Phase 29 has no VERIFICATION.md; Phase 31's on-device pass is still `human_needed` (both from v1.7)

## Next Milestone Goals

**Not yet defined.** Run `/gsd-new-milestone` to scope v1.9.

Already scoped and unscheduled, available to claim:
- **Phase 37: Way Types & Surfaces Breakdown (mobile-first)** — only `37-RESEARCH-SOURCE.md` exists;
  it was blocked behind Phase 36 (shared `trail.dart`, `api/v1/trail` handlers, `trails` migrations)
  and is now unblocked

Standing candidates from the deferred backlog: dark mode for the Flutter app (quick task
260612-gmg), the untranslated destructive-action strings from Phase 36 (WR-06), and CONV-F01/F02
plus REC-F01/F02 from the v1.8 requirements archive.

<details>
<summary>v1.8 milestone scope as originally stated (archived)</summary>

**Goal:** A hiker who records a trail with no signal can save it, review it, and fill in its details on the spot — and it uploads itself when the phone next has a connection, without the hiker doing anything.

**Deliberate scope exceptions taken:**
- **Web frontend changes were in scope**, overriding the standing app-only boundary — the metrics defects live in shared conversion logic used by both, and fixing only one side would make a single GPX yield two different answers.
- **Trail metrics were re-baselined.** Newly converted trails report different distance/elevation/duration than the same file converted before v1.8. Already-saved trails are unaffected (`PUT /trail/form` stores client values and never recomputes). Accepted: the app is pre-production.

</details>

## What This Is

Turn-by-turn trail navigation, full settings management, category-aware trail discovery, and native-GL map rendering for the Wanderer Flutter mobile app. Users launch navigation from a trail's detail or map screen and get Valhalla-powered maneuver instructions, a live map centered on their position, and a stats sheet tracking distance, elevation, and speed. The app includes a complete settings suite (language, units, privacy, account, notifications, categories), subcategory-aware trail filters, and a `maplibre`-native map stack across all map surfaces (detail, list, map screen, navigation) with offline `.pmtiles` + glyph/sprite caching for airplane-mode trail use.

## Core Value

A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.

## Requirements

### Validated

- [x] Navigate button on trail_detail_screen and trail_detail_map_screen (v1.0 — Phase 2)
- [x] Full-screen navigation screen with map centered on current GPS position (v1.0 — Phase 2)
- [x] Valhalla turn-by-turn maneuver instructions shown at the top of the screen (v1.0 — Phase 2)
- [x] North-up / heading-up map orientation toggle button (v1.0 — Phase 2)
- [x] DraggableScrollableSheet with live stats: distance, elevation gain/loss, speed (v1.0 — Phase 3)
- [x] New SvelteKit API endpoint POST /api/v1/valhalla/navigate (v1.0 — Phase 1)
- [x] Progress tracking: advance through maneuvers as user moves along the trail (v1.0 — Phase 2)
- [x] Offline navigation fallback: serve maneuvers from ObjectBox cache when network is unavailable (v1.1 — Phase 5)
- [x] Silent cache write at trail download time and re-cache after online sessions (v1.1 — Phase 5)
- [x] SettingsScreen: Privacy, Language, Notifications list entries with sub-routes (v1.2 — Phase 6)
- [x] SettingsLanguageScreen: 14-locale language picker + metric/imperial unit toggle, persists and auto-saves (v1.2 — Phase 6)
- [x] Unit toggle propagates to all format_util call sites app-wide (v1.2 — Phase 6)
- [x] SettingsPrivacyScreen: account, trails, and lists visibility radio groups, auto-save (v1.2 — Phase 7)
- [x] SettingsAccountScreen: avatar upload, bio editor, email change, password change, account deletion with confirmation (v1.2 — Phase 8)
- [x] SettingsNotificationsScreen: 9 notification types × web + email toggles, auto-save (v1.2 — Phase 9)
- [x] Category model updated with `icon`, `short_name`, `translations`; locale-aware display name resolution (v1.3 — Phase 10)
- [x] Subcategory freezed model, ObjectBox entity, and provider fetching from `/subcategory` (v1.3 — Phase 10)
- [x] Category and subcategory preference Riverpod providers backed by `/user-category-preference` and `/user-subcategory-preference` (v1.3 — Phase 10)
- [x] `Settings.category` field removed (replaced by priority-based category preferences) (v1.3 — Phase 10)
- [x] `TrailFilter` gains `subcategory` list; TrailFilterScreen and quick filter bar show subcategory chips (v1.3 — Phase 11)
- [x] New SettingsCategoriesScreen: per-category and per-subcategory visibility toggles + priority reordering (v1.3 — Phase 12)
- [x] Wanderer style JSONs (light + dark) live as plain app assets; both `flomp/*` forks removed from `dependency_overrides` (v1.4 — Phase 15/18)
- [x] Wanderer server serves glyph PBFs and a sprite sheet via unified `/api/v1/map/style-sources`; app resolves them like it resolves `TILE_SERVER_URL` (v1.4 — Phase 13)
- [x] Trail download fetches glyphs + sprite once, caches them, and rewrites the style to `file://` for offline label rendering (v1.4 — Phase 15)
- [x] All 6 map screens render via `MapLibreMap`; `flutter_map` + 4 plugins gone from `pubspec.yaml` (v1.4 — Phases 15-18)
- [x] Map screen clustering reads `POST /search/trails/cluster` and renders native circle/symbol layers (v1.4 — Phase 16)
- [x] Offline trails render from `.pmtiles` via native `pmtiles://`; `pm_tile_provider.dart` deleted (v1.4 — Phase 15/17)
- [x] Navigation screen keeps heading-up follow mode, compass reset, and live location puck on maplibre (v1.4 — Phase 17)
- [x] Route Planner screen: tap-to-add, drag, insert-mid-route, delete, reorder waypoints with undo/redo (v1.5 — Phase 19)
- [x] Auto-routing toggle: Valhalla-routed segments (fixed foot/bike profile set at entry) vs straight-line segments, re-resolved on toggle (v1.5 — Phase 19)
- [x] Waypoint list sheet and live elevation profile, mutually exclusive map-control-toggled views (v1.5 — Phase 20)
- [x] Search-to-focus map panning via existing GlobalSearchScreen flow (v1.5 — Phase 20)
- [x] Handoff to trail_create_screen as a draft Trail (synthesized GPX + waypoints) (v1.5 — Phase 21)
- [x] Hike/bike selection dialog on "Open trail planner" tap, before the planner screen opens, sets initial travel profile (v1.5 — Phase 21)
- [x] Backend region catalog loaded from an admin-supplied config file (Docker-volume mounted); cronjob pre-builds one mosaicked vector + one DEM PMTiles archive per region ahead of any user request; API endpoint serves the catalog (v1.6 — Phase 21.5)
- [x] App fetches its region catalog from the instance's backend API at runtime — no bundled asset, empty catalog on a fresh instance with no admin config (v1.6 — Phase 22)
- [x] ObjectBox `Region` + `DownloadedTilePackage` entities tracking download status (explicit int constants, not index-backed enum), paths, timestamps, disk usage (v1.6 — Phase 22)
- [x] App-wide `TileRepositoryManager`: region download/cancel/delete lifecycle, fully decoupled from Trail — no pause/resume (removed 2026-07-23; Dio's `deleteOnError` deletes `.part` files on any cancellation including a deliberate pause, so cancel-and-restart-from-0 replaced it) (v1.6 — Phase 23)
- [x] Region downloads with an independent per-region Mapterhorn DEM package (own list tile, own download/cancel/delete, gated on Vector being downloaded first), reusing the existing DEM pipeline keyed to regions (v1.6 — Phase 23/24)
- [x] Trail download guard: region coverage check before trail download; dialog names missing region(s) with per-region "Download region" CTA, supports partial-coverage subset downloads (v1.6 — Phase 26)
- [x] Settings → Offline Maps/Regions page: flat searchable region list, size breakdown, independent Vector/DEM download/cancel/delete + progress bars, total disk usage, non-blocking `updateAvailable` badge (v1.6 — Phase 24)
- [x] Map rendering reads offline tiles through a local loopback HTTP tile proxy (MapLibre Native's own viewport tracking selects regions, not hand-rolled incremental source diffing — Phase 25's reconcile approach had a reentrancy race, replaced in Phase 25.1); legacy trail-scoped tile download/cache code removed outright, no migration path (v1.6 — Phases 25, 25.1, 27)

- ✓ Seeded `regions` table (PocketBase), hierarchical group/leaf, CoMaps-sourced (v1.7 — Phase 28)
- ✓ `seed_regions.go` maintainer tool + auto-run migration for zero-admin-action seeding (v1.7 — Phase 28)
- ✓ Polygon-based `pmtiles extract --region` replacing bbox extraction (v1.7 — Phase 29)
- ✓ Custom PocketBase admin page: collapsible region tree + live map, toggle `enabled` (v1.7 — Phase 30)
- ✓ Archive cron reads `kind = 'leaf' AND enabled = true`, retiring `region_config.json` (v1.7 — Phase 29)
- ✓ Flutter Settings screen: flat region list → collapsible hierarchy matching admin tree (v1.7 — Phase 31, on-device pass outstanding)
- ✓ Geometry fetched on demand from CoMaps at a pinned commit; catalog 54.65 MB → ~315 KB (v1.7 — Phase 32)


- ✓ Four metric defects fixed in the shared GPX→trail computation: per-segment first-point skip, centroid divisor mismatch, `ele ?? 0` treating missing elevation as sea level, and elevation sampling gated behind the 5 m horizontal threshold (v1.8 — Phase 33)
- ✓ `cumulativeDistance` rebuilt as a raw, index-aligned array and the trail-edit crop slider rescaled to it (v1.8 — Phase 33). **Adjusted:** CONV-05's switch to the *smoothed* accumulator was superseded 2026-08-01 — FIT ground truth measured raw at +0.54% against the 5 m gate's −3.29%, so distance reports raw
- ✓ Recorded trails report moving time, excluding `pausedAccum`; imported files keep elapsed time (v1.8 — Phase 34)
- ✓ Dart port of the metrics computation, pinned against the TS by a ten-fixture on-disk corpus both languages read (v1.8 — Phase 34)
- ✓ `/api/v1/trail/convert` reduced to transcode-only; `.gpx` imports, recordings, and planner output all measured in Dart (v1.8 — Phase 34)
- ✓ Locally-captured trails persist with no server id and a sync state, account-scoped, visible in the hiker's own-trails list immediately (v1.8 — Phase 36). **Widened during execution** from "recorded" to any on-device capture, so an offline GPX import can be saved too
- ✓ Undrained captures upload automatically on foreground/connectivity/cold-start, with inline progress, manual retry, and resume-from-step so an interrupted upload cannot duplicate a trail (v1.8 — Phase 36)
- ✓ `trail_create_screen` works with no connection: offline basemap, non-throwing tag autocomplete, and a clear message when a non-GPX import needs a connection (v1.8 — Phase 35)
- ✓ *Remove download* and *Delete trail* separated, derived from library membership and authorship rather than `Trail.isLocal`; editing always operates on the server copy and refuses with a stated reason when it cannot be fetched (v1.8 — Phases 38, 38.1)
- ✓ Library edits stopped duplicating photos on the server; unsynced photo directories are account-scoped (v1.8 — Phases 38, 38.1)

### Active

(None — v1.8 shipped. Run `/gsd-new-milestone` to define v1.9's requirements.)

### Out of Scope

- Storing canonical polygon/bbox on leaf `regions` rows — shipped in v1.7 Phase 28, then deliberately superseded by Phase 32; geometry now lives in `region_geometry`, fetched on demand (CATALOG-02 retired by SLIM-01/02)

- Web frontend changes — `web/` already runs maplibre-gl-js; app-only apart from the v1.4 glyph/sprite endpoint. **Amended 2026-07-31 (v1.8):** shared GPX→trail conversion logic (`web/src/lib/models/gpx/*`, `web/src/lib/util/gpx_util.ts`, `/api/v1/trail/convert`) is explicitly in scope, because leaving its metric defects unfixed on one side would make a single GPX yield two different answers. The boundary still holds for web UI.
- Migrating already-saved trails to the corrected metrics — no backfill; `PUT /trail/form` stores client values and never recomputes, and the app is pre-production so there is no meaningful install base to migrate (v1.8)
- Porting the kml/kmz/tcx/fit transcoders to Dart — they depend on vendored `toGeoJSON`, JSZip, and `fit-parser`, none of which a recording can reach; those formats stay online-only via `/api/v1/trail/convert` (v1.8)
- Queuing a non-GPX file import for conversion on reconnect — a queued import has no trail to show until it transcodes, which breaks the local-first model the recordings use (v1.8)
- Moving time for imported GPX files — pause data only exists for trails recorded in the app; imports keep elapsed time (v1.8)
- Switching offline tile generation to OpenMapTiles schema — would invalidate every downloaded trail archive and every operator's tile cache
- Basemap picker in app settings (OpenTopoMap / CyclOSM / Carto, as web offers) — deferred; v1.4 shipped the Protomaps wanderer style only (FUT-02/03)
- Contributing `cluster` fields upstream to maplibre's `GeoJsonSource` — unnecessary once clustering is server-side
- Linux/Windows/macOS map support — `maplibre` has no Linux backend and the app is mobile-only
- 3D terrain, hillshade, globe projection, pitch/tilt gestures — newly possible on native GL, not yet scoped (FUT-04/05)
- Text-to-speech maneuver announcements — deferred to v2 (audio infra adds complexity)
- Routing from user's current position to the trailhead — assume user is already at the trail start
- Re-routing if user goes off-trail
- API token management (ACCT-F01) — mobile clients don't need API tokens; web-only feature
- Favourite sport picker, Export, Integrations (Strava/Komoot), Maintenance, Map settings — out of scope for mobile settings v1
- Drag-to-reorder for category priority (use ReorderableListView or up/down buttons instead of pointer-drag like web)
- Category/subcategory picker in trail create/edit form — deferred to a later milestone (form rework needed)
- Bulk-edit modal — web-only feature, out of scope for mobile
- Render `is_large` trails as full polylines on the map screen (FUT-01) — cluster endpoint already emits the flag; app ignores it
- Dark mode for the Flutter app — quick task 260612-gmg was planned but never executed; candidate for a future milestone or standalone task
- Car/driving costing profile for the Route Planner — `gpx_util.dart`'s `costingForCategory` only maps foot/bike today; v1.5 ships foot/bike only
- Editing an existing trail's route in the Route Planner — v1.5 is from-scratch planning only
- Per-segment travel profiles in the Route Planner — a single profile applies to the whole route
- Switching travel profile mid-session in the Route Planner (ROUTE-03 cut) — profile is fixed once set via the entry hike/bike dialog
- Offline route caching for in-progress route plans
- Legacy trail-cache migration — app is pre-production; old trail-scoped tile/DEM cache code and files are deleted outright, no conversion path (v1.6, shipped)
- One-time on-device sweep for orphaned legacy tile files (CLEAN-02) — descoped, not deferred: app is pre-production, no real install base to sweep (27-CONTEXT.md D-05)
- Admin UI/API for region catalog CRUD — region definition is a config file mounted via Docker volume, edited by redeploying; a live-editing settings/admin screen is deferred (v1.6)
- Polygon region geometries — v1.6 regions are bounding-box only; arbitrary polygon boundaries deferred (v1.6)
- 3D terrain/hillshade rendering redesign — v1.6 only relocates the existing DEM download/storage pipeline to be region-based; `offline_style_rewriter.dart`'s hillshade rendering is reused as-is (v1.6, shipped)
- Resumable downloads, of any kind — originally scoped as session-only pause/resume, amended 2026-07-23 to drop resume entirely (Dio's `deleteOnError` treats a deliberate pause identically to a genuine transfer error); cancel always restarts a download from byte 0 (v1.6, shipped)
- Hierarchical region tree navigation — manifest is tens of entries, not thousands; a flat searchable list is the right complexity (v1.6)
- Granular per-layer toggles beyond vector/DEM (roads/POIs/water) at the region-download level — no reviewed hiking app exposes this (v1.6)
- Map boundary highlight overlay showing downloaded region coverage on the map — nice differentiator, deferred (REGN-F01)
- Region entitlement/paywall model — no paywall exists in Wanderer; the guard dialog borrows Komoot's messaging pattern only, not its unlock/purchase logic (v1.6)

## Context

- Flutter mobile app with Riverpod state management, go_router navigation, `maplibre` (native GL) for maps, geolocator for GPS
- v1.4 retired both `flomp/*` forks and `flutter_map` + 4 plugins entirely; `maplibre` 0.3.5 is pinned exact. Map surface: `wanderer_map.dart`, `trail_layer.dart`, `search_map.dart`, `map_screen.dart`, `navigation_screen.dart`, `trail_detail_map_screen.dart`, `list_detail_map_screen.dart`, `list_detail_screen.dart`, `map_style_json_provider.dart`, plus `trail.dart`, `gpx_util.dart`, `polyline_util.dart`, `foreground_position_stream_provider.dart` now speaking `Geographic`/`LngLatBounds`
- Wanderer light/dark styles are checked-in MapLibre Style Spec v8 `.json` assets with `glyphs`/`sprite` keys; `map_style_json_provider.dart` injects operator URLs and swaps live on theme change
- `/api/v1/map/style-sources` (SvelteKit) resolves tile, glyph, and sprite URLs in one call, operator-overridable, defaulting to Protomaps' public `basemaps-assets` host
- App-wide glyph/sprite cache under `<app-docs>/map_cache`, warmed on both map-open and trail-download; `rewriteStyleForOffline` rewrites `glyphs`/`sprite`/tile sources to `file://`/`pmtiles://file://` for airplane-mode rendering, including multi-cell trails
- Offline `.pmtiles` cells are extracted from `build.protomaps.com` by `db/services/tiles/generator.go` → **Protomaps schema**. Web's `static/styles/ofm.json` is **OpenMapTiles schema** via openfreemap — the two remain non-interchangeable, so the app keeps its own Protomaps-schema style
- `POST /api/v1/search/trails/cluster` (server-side Supercluster) now drives the map screen's clustering, rendered as native circle/symbol layers via `cluster_layer.dart` — matches web's `cluster-layer.ts` rendering
- `maplibre` 0.3.5 quirks worth remembering: `file://` sprite resolution failed on-device in early spike testing (glyph `file://` resolution works fine) — trail arrow icon sidesteps this via `addImageFromIconData` self-registration; camera "instant" moves must use `Duration(milliseconds: 1)`, never `Duration.zero` (crashes Android native binding); `onStyleLoaded` can fire before `onMapCreated`, requiring a buffered-replay pattern
- Shipped v1.0: Navigation screen (SvelteKit Valhalla endpoint + Flutter screen + stats sheet)
- Shipped v1.1: Offline navigation (ObjectBox cache, DioException fallback, offline indicator)
- Shipped v1.2: Full settings suite (Language/Units, Privacy, Account/Profile, Notifications)
- Shipped v1.3: Category/subcategory model + preferences, subcategory-aware trail filters, Settings → Categories screen
- Shipped v1.4: Full `maplibre` migration — native GL rendering, self-hosted glyph/sprite serving, offline parity, server-side clustering, both `flomp/*` forks retired
- Shipped v1.7: Seeded CoMaps region catalog (1,306 rows) + polygon-based extraction + PocketBase admin picker + Flutter hierarchy; geometry moved on-demand, repo pack down 268 → 198 MiB
- Web PR #1059 merged: new category model with translations/icon/short_name, subcategories, user category/subcategory preferences, favourite sport replaced by priority-based category ordering
- Settings infrastructure: `Settings` freezed model, `settingsProvider` with `saveToServer()` — reused by all settings screens
- `localeProvider` and `unitProvider` derived from `settingsProvider` — live-switch locale and unit system app-wide
- 14 supported locales with ARB files; AppLocalizations regenerated in Phase 6
- Background navigation via tracelet package (from quick tasks)
- Along-track projection for waypoint advancement (from quick tasks)
- Open item: dark mode (quick task 260612-gmg) was planned but never executed — the app has no Appearance/theme-mode setting yet
- Shipped v1.5 (2026-07-17): Route Planner — from-scratch route building with waypoint editing, Valhalla auto-routing, elevation profile, and handoff to trail create/edit. See `.planning/milestones/v1.5-ROADMAP.md` / `v1.5-REQUIREMENTS.md`
- Shipped v1.6 (2026-07-24): Offline Region Tile Repository — app-wide, region-based offline tile system fully replaces the old trail-scoped one; admin-configured backend catalog, `TileRepositoryManager`, local loopback HTTP tile proxy for map rendering, trail-download coverage guard, and legacy tile code deleted outright. See `.planning/milestones/v1.6-ROADMAP.md` / `v1.6-REQUIREMENTS.md`
- Offline tiles are now **region-scoped, not trail-scoped**: `TileRepositoryManager` (`app/lib/services/tile_repository_manager.dart`) owns download lifecycle for `regions/<id>/{vector,dem}.pmtiles`, independent of any Trail. The old `trail_download_service.dart` tile methods, `TrailEntity.pmTiles`/`demPmTiles` fields, and `app/lib/models/map_cell.dart` no longer exist (removed Phase 27)
- Map rendering (`TrailMap`/`navigation_screen`) reads tiles through a local loopback `HttpServer` tile proxy (`app/lib/services/local_tile_proxy_service.dart` or similar, Phase 25.1) — MapLibre Native's own viewport tracking selects which downloaded regions to render, not hand-rolled Dart source diffing
- Mapterhorn DEM pipeline: `db/services/tiles/generator.go` extracts region-mosaicked DEM archives from `https://download.mapterhorn.com/planet.pmtiles` (const `mapterhornSource`) at `demMaxZoom = 12` (vs vector `maxZoom = 14`), pre-built by a cronjob per configured region (BACK-02/03), served via the region catalog API (BACK-04). DEM generation is best-effort — failures never block the vector basemap
- DEM tiles are raster hillshade only — no numeric elevation is sourced from them. Elevation gain/loss/profile comes exclusively from GPX `<ele>` track points (`gpx_util.dart`); v1.6 does not change that
- `app/lib/util/offline_style_rewriter.dart` special-cases `raster-dem` style sources (encoding `terrarium`, tileSize 512, max zoom locked to 12) and drops hillshade layers cleanly when no DEM was downloaded — reused as-is, now sourcing from region-based files
- Downloads are never resumable, at any level (amended 2026-07-23) — cancelling (deliberate or a genuine transfer error) always deletes the `.part` file via Dio's `deleteOnError: true`; a later attempt always restarts from byte 0
- Region catalog is admin-defined per instance via a config file mounted through Docker volume (`db/` backend), not a bundled Flutter asset — a fresh/default instance with no admin config returns an empty catalog
- Shipped v1.8 (2026-08-07): Offline Recording & Deferred Upload — corrected + Dart-ported GPX→trail metrics, transcode-only `/trail/convert`, local-first unsynced trails with automatic background upload, and downloaded-trail state separated from trail identity. See `.planning/milestones/v1.8-ROADMAP.md` / `v1.8-REQUIREMENTS.md`
- **Trail metrics have exactly one implementation per language, and they are pinned to each other.** `fixtures/gpx-corpus/` (10 on-disk fixtures + README contract) is read by both `web/src/lib/models/gpx/gpx-corpus.test.ts` and `app/test/util/gpx_corpus_test.dart`. `GpxMappingUtils.getTotals()`/`GpxStats` were deleted — do not reintroduce a second Dart metrics path
- Distance is the **raw** haversine accumulator, not the 5 m-gated smoothed total (amended 2026-08-01). `thresholdXY_m` and both `GpxMetricsComputation(5, 5)` call sites remain because the XY threshold still drives the elevation noise filter — elevation is unchanged and its 5 m Z-gate is known to undercount ~12% on barometric data
- `duration` means GPX-derived elapsed time everywhere; recorded moving time lives in the separate `moving_duration` field (PocketBase, OpenAPI, TS, Dart), surfaced via `trailDisplayDuration()` on both sides
- **"Unsynced" and "downloaded" are independent axes, not mutually exclusive** — a trail can be both. `Trail.isLocal` is cache provenance only and must never gate a destructive action, a badge, or a tab; use `syncState` / library membership / authorship instead (Phase 36's D-10 is retracted in place)
- Locally-captured trails live in ObjectBox via `local_trail_store.dart` (the single owner-scoped read/write layer), addressed by `localId` through `trailDetailLocation`/`trailMapLocation`, never by the blanked server id. Their photos live under `<app-docs>/unsynced/<accountId>/<localId>/`
- `TrailSync` (keepAlive) drains uploads on foreground, regained connectivity, and cold start, replaying `PUT /tag` → `PUT /trail/form` → `PUT /waypoint` and writing each server id back before the next step; it parks after 4 attempts and retires the local row the instant an upload completes

## Constraints

- **Tech Stack**: Flutter + Riverpod (riverpod_annotation codegen) + go_router + freezed — must follow existing patterns. v1.4 replaces `flutter_map` with `maplibre` as the map renderer
- **API**: SvelteKit app proxies Valhalla at POST /api/v1/valhalla/navigate; Flutter calls via Dio
- **Online-only navigation v1**: Navigation requires network; falls back to ObjectBox cache when offline
- **No breaking changes**: Existing trail detail screens, bottom nav, and routes must be unaffected
- **Offline parity is a hard gate**: a downloaded trail must render basemap *and labels* with no network. Any phase that regresses this is not done
- **Both map stacks coexist during v1.4**: `flutter_map` stays in `pubspec.yaml` until the last screen is migrated. Every phase must leave the app building and runnable
- **Tile source stays Protomaps**: offline cells are Protomaps-schema, so the app's style targets Protomaps. `TILE_SERVER_URL` / `PROTOMAPS_API_KEY` remain the operator's control surface
- **maplibre pinned**: pre-1.0 with breaking 0.x minors — pin an exact version, do not float

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| [v1.8] Audit the TS `gpx2trail` before porting it, rather than porting bug-for-bug | Surfaced four real defects (per-segment first-point skip, centroid divisor, `ele ?? 0`, elevation gated behind the XY threshold); a faithful port would have made them permanent and unfixable without diverging | ✓ Good — all four fixed in Phase 33 and pinned by the cross-language corpus in Phase 34 |
| [v1.8] Fix the metrics in web and app together, overriding the app-only boundary | The defects live in shared conversion logic; fixing one side means a single GPX yields two different answers | ✓ Good — the ten-fixture on-disk corpus is now the executable contract between the two implementations |
| [v1.8] Accept re-baselined metrics with no backfill | `PUT /trail/form` stores client values and never recomputes, so old trails keep old numbers; app is pre-production, so no meaningful install base to migrate | ✓ Good — no migration written; CONV-F01 tracked as future work |
| [v1.8] Locally-captured trails are local-first records that acquire a server id later, not a pending-upload queue | Matches how Komoot and AllTrails both model it — the capture is in the hiker's own-trails list immediately with an inline sync badge, not in a separate inbox | ✓ Good — but the model's edges cost six gap-closure plans (36-15…36-20) and a whole inserted phase (38.1); every one was about what happens to a row *between* capture and retirement |
| [v1.8] `/api/v1/trail/convert` becomes transcode-only rather than gaining an opt-in mode | The Flutter app is its only caller (web converts in-browser via `fromFile`/`gpx2trail`) and the endpoint is not deployed in production anywhere, so the breaking change is safe | ✓ Good — landed last within Phase 34, after every app call site had already moved to the Dart path |
| [v1.8] Don't port the kml/kmz/tcx/fit transcoders to Dart | They need vendored `toGeoJSON`, JSZip, and `fit-parser`; no recording can reach them, so the cost buys only offline import of rare formats | ✓ Good — OFFUI-04 explains the limit to the hiker instead; REC-F01 (queue-for-reconnect) tracked as future work |
| Extend SvelteKit Valhalla API rather than calling Valhalla directly from Flutter | Keeps credentials server-side, consistent with existing route endpoint pattern | ✓ Good — used in v1.0 through v1.2 |
| DraggableScrollableSheet for stats, not fixed bottom bar | Matches MapScreen pattern; user can expand for more detail without blocking map | ✓ Good |
| Button-driven PageView for stats (not horizontal swipe) | Locked during Phase 3 context; horizontal swipe conflicts with map pan gesture | ✓ Good |
| TTS deferred to v2 | Adds Flutter TTS dependency and audio session handling — not needed for navigation core | — Pending |
| Assume user is at trailhead | Simplifies v1 scope; off-trail routing is a separate problem | — Pending |
| Dio try-catch as sole offline gate (not connectivity packages) | Connectivity packages have false-positive/negative failure modes (captive portals, VPN) | ✓ Good |
| `String? navCacheJson` on TrailEntity (not typed List) | ObjectBox doesn't support List<List<double>> or List<NavigateManeuver> natively; follows `gpxData` precedent | ✓ Good |
| freezed 3.x: @JsonSerializable(explicitToJson: true) on factory constructor | Class-level placement breaks json_serializable codegen in freezed 3.x | ✓ Good — critical for future freezed models |
| `Settings` freezed model + `settingsProvider.saveToServer()` shared across all v1.2 screens | No new persistence layer needed; single source of truth | ✓ Good |
| RadioGroup<Language> with hardcoded native-name map | Only approved exception to the no-hardcoded-strings rule; localized names require the locale itself to render | ✓ Good |
| Imperial = on-position for unit switch (UI-SPEC D-10) | Toggle polarity chosen to match natural reading: "imperial is the non-default" | ✓ Good |
| `State.mounted` in ConsumerState, `context.mounted` in ConsumerWidget helpers | `mounted` refers to State object — only available in ConsumerState; ConsumerWidget uses context.mounted | ✓ Good — important Flutter/Riverpod gotcha |
| Colors.red.shade400 for destructive foreground (not colorScheme.error) | colorScheme.error maps to #FEF2F2 (background token) — illegible as foreground text | ✓ Good |
| [v1.4] Migrate to `maplibre`, not `maplibre_gl` | `maplibre` binds maplibre-native via FFI/JNI and reads our style JSON directly; `maplibre_gl` is the older package the author rewrote | ✓ Good — shipped v1.4, pinned 0.3.5 |
| [v1.4] Retiring both `flomp/*` forks is the primary payoff, not map performance | The forks exist only to parse and render a style JSON that maplibre consumes natively. Native GL rendering, rotation, and pitch are follow-on benefits | ✓ Good — both forks and all 6 packages removed |
| [v1.4] Clustering reuses `POST /search/trails/cluster` rather than `cluster: true` | Endpoint already exists and web already renders its output; keeps app/web parity, keeps the `is_large` polyline split, avoids maplibre's missing `GeoJsonSource` cluster fields entirely | ✓ Good |
| [v1.4] Keep the Protomaps wanderer style; do not adopt web's `ofm.json` | Offline `.pmtiles` cells are Protomaps schema (`earth`, `landcover`, `roads`); OpenMapTiles styles (`water`, `landuse`, `transportation`) would render them blank | ✓ Good |
| [v1.4] Style JSON lives as an app asset, not a server-hosted style URL | Offline rendering needs the style before any network call, and the `glyphs`/`sprite` keys get rewritten to `file://` at runtime anyway | ✓ Good |
| [v1.4] Self-host glyphs + sprite on the Wanderer server | Style references 4 fontstacks (incl. Devanagari) with no `glyphs` key; hosting mirrors the `TILE_SERVER_URL` operator-control model | ✓ Good |
| [v1.4] Glyphs cached once app-wide, not per-trail | Glyphs are style-global; per-trail download would re-fetch identical fontstacks and multiply storage | ✓ Good |
| [v1.4] Incremental screen-by-screen migration, forks deleted last | Keeps the app runnable at every phase boundary; the `LatLng`→`Geographic` churn touches GPX parsing, so a big-bang landing risks trail data, not just maps | ✓ Good — app stayed buildable at every phase boundary |
| [v1.4] `file://` sprite resolution unreliable on-device; self-register trail arrow instead of relying on sprite atlas | 15-01 spike found glyph `file://` works but sprite `file://` failed on a physical Android device despite valid cached files | ✓ Good — `addImageFromIconData` sidesteps the gap; other sprite-atlas icons (route shields) remain a pre-existing non-regression |
| [v1.4] OFFL-06 (`pm_tile_provider.dart` deletion) deferred from Phase 15 to Phase 17/18 | `navigation_screen` was the last flutter_map holdout still consuming `MultiPmTilesVectorTileProvider`; deleting the file in Phase 15 would have broken the screen the criterion required to keep building | ✓ Good — deleted in Phase 17 once navigation_screen migrated |
| [v1.5] Route anchor list + elevation profile as tabs of one docked sheet, not separate map-control-toggled views (PLANUI-01 scope change) | Simpler mental model — one persistent sheet instead of two mutually-exclusive overlay states | ✓ Good — shipped Phase 20 |
| [v1.5] Handoff synthesizes a GPX track only, no Waypoint records (HANDOFF-01 scope change) | Route anchors are planner-only; converting them to named Waypoints would conflate two different concepts on the receiving trail_create_screen | ✓ Good — shipped Phase 21 |
| [v1.5] Travel profile (hike/bike) fixed at planner entry, no mid-session switch (ROUTE-03 cut, tracked as PLANNER-07) | Avoids re-resolving every existing segment on profile change; simplifies the auto-routing engine | ✓ Good — deferred to v2 backlog |
| [v1.5] Dedicated `LocationSearchScreen` instead of reusing `GlobalSearchScreen` | `GlobalSearchScreen` also returns trails/lists/accounts; the planner only needs location results | ✓ Good — shipped Phase 20 |
| [v1.6] Region catalog is an admin-supplied Docker-volume-mounted config file + cron-pre-built archives + API endpoint, not a bundled `regions.json` app asset | For a self-hostable app, offline-downloadable regions are a per-instance admin decision, not fixed at Flutter build time; pre-building ahead of request makes downloads instant instead of orchestrating N per-cell requests client-side | ✓ Good — shipped Phase 21.5, corrected an earlier bundled-asset assumption made during Phase 22 planning |
| [v1.6] Pause/resume dropped entirely from region downloads, replaced with cancel-and-restart-from-0 | Dio's native `deleteOnError` handler deletes the `.part` file on ANY cancellation, including a deliberate pause — the very first pause of a fresh download silently destroyed its own resume progress | ✓ Good — amended 2026-07-23 (commit `4732d20e`), simpler than working around the footgun |
| [v1.6] DEM is its own independent list tile (not a toggle on the Vector row), download gated on Vector being downloaded first | Hillshading without a basemap underneath it is meaningless; two independent packages with clearer per-item progress/state beat one row with a hidden sub-toggle | ✓ Good — amended 2026-07-23 (commit `663f049a`) |
| [v1.6] Local loopback HTTP tile proxy for map rendering, not incremental `addSource`/`removeSource` region-swap reconciliation | Phase 25's hand-rolled Dart diffing had a reentrancy race (no in-flight guard, `MapEventCameraIdle` over-firing during GPS-follow) found in UAT; a proxy lets MapLibre Native's own viewport tracking pick regions, eliminating the race structurally instead of patching it | ✓ Good — shipped as urgent Phase 25.1 insertion |
| [v1.6] CLEAN-02 (one-time orphaned-legacy-tile sweep) cut outright, not deferred | App is pre-production — no real install base with orphaned legacy tile files exists to clean up; any dev/test device can be wiped/reinstalled manually | ✓ Good — descoped per 27-CONTEXT.md D-05, ROADMAP success criterion #2 annotated accordingly |
| [v1.6] Legacy trail-scoped tile code deleted outright in Phase 27, no dual-run or migration path | App is pre-production; `pmTiles`/`demPmTiles` had zero readers left once Phase 25 moved map rendering to the region pipeline | ✓ Good — zero remaining references confirmed by goal-backward verification |
| [v1.7] Region catalog is seeded from CoMaps' own extract hierarchy, not hand-drawn or admin-authored | A curated upstream catalog removes the research and bbox-arithmetic burden that freehand drawing reintroduces, and CoMaps' boundaries are real, maintained, and ODbL-compatible with the existing OSM-derived tile data | ✓ Good — 1,306 rows seeded, zero admin action on a fresh instance |
| [v1.7] Boundary geometry is fetched on demand from CoMaps at a pinned commit, not committed to the repo | Geometry was 99% of a 54.65 MB seed that only the build path and admin map ever read; a pinned SHA on a raw-file endpoint is content-addressed and immutable, so on-demand costs nothing in reproducibility. Offline boot is unaffected — only archive building, already network-gated, needs connectivity | ✓ Good — catalog to ~315 KB, seed run from ~1,153 requests to 1, clone pack 268 → 198 MiB |
| [v1.7] `bbox` moved out of the catalog into `region_geometry` alongside the polygon | Verified every consumer needs bbox only for *enabled* regions; keeping it in the catalog would have forced the generator to keep scraping all ~1,153 `.poly` files for data most regions never use | ✓ Good — what made the one-request seed run possible |
| [v1.7] Geometry persists on the `enabled` false→true transition via a server hook, not a client call | The original design made persistence depend on one specific UI call site doing one specific thing, and it silently never fired — enabling a region cached nothing. Keying off the record transition covers the picker, a REST PATCH, and the collection editor alike | ✓ Good — amended 2026-07-28 (`4b98c48b`) after the hole surfaced in manual use |
| [v1.7] Phase-level verification under-covered cross-phase wiring | Three integration bugs surfaced through manual use *after* Phase 32's verification passed (`4b98c48b`, `0149b83e`, `6069cb57`), all in the seam where a field moved between collections and its writers weren't all traced | ⚠️ Revisit — milestone audit's integration check caught what phase verification did not; consider making cross-phase checks routine rather than milestone-only |
| [v1.8] Report raw distance, not the 5 m-gated smoothed total (supersedes CONV-05's smoothed half) | The gate chord-shortcuts switchbacks at real GPS sampling density. FIT ground truth (`session.total_distance` = 10912.01 m) put raw at +0.54% against the gate's −3.29%, and the corpus's own `04-switchback-scramble` asserted 0.000 m for an 88 m climb | ✓ Good — `thresholdXY_m` deliberately left intact, since it is load-bearing for the elevation noise filter |
| [v1.8] `Trail.isLocal` (cache provenance) must never gate destructive actions, badges, or tabs | Phase 36 shipped on the premise that "unsynced" and "downloaded" were mutually exclusive. They are independent axes. Phase 38's review found three real defects following from that false premise, and Phase 38.1 was inserted to close them | ⚠️ Revisit — a wrong premise inherited across two phases cost an inserted phase to unwind. The premise was stated in comments, not encoded in a type or a test |
| [v1.8] Close the milestone without a `/gsd-audit-milestone` pass | Requirements were 25/25 with a full traceability table, and Phases 38/38.1 had already had code review + verification | — Pending — v1.7's audit caught integration gaps phase verification missed; skipping it here means Phase 36's `human_needed` verification is the only unclosed signal |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---

---
*Last updated: 2026-07-31 after starting milestone v1.8*

---
*Last updated: 2026-08-07 after v1.8 milestone*
