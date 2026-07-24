# Wanderer Trail Navigation

## Current State

**Shipped:** v1.5 Route Planner (2026-07-17) — from-scratch route building in a dedicated `RoutePlannerScreen`: tap/drag/insert/delete/reorder waypoints with undo/redo, an auto-routing toggle (Valhalla-routed vs straight-line segments, fixed foot/bike profile per session), a docked tabbed sheet (route anchor list + live elevation profile), location-search-to-focus, and handoff to `trail_create_screen` as a draft Trail with a one-time elevation merge.

**Also complete (not yet formally closed):** v1.6 Offline Region Tile Repository — all phases (21.5, 22-27) executed, including Phase 27's removal of the legacy trail-scoped tile system. Not yet run through `/gsd-complete-milestone`.

## Current Milestone: v1.6 Offline Region Tile Repository

**Goal:** Replace trail-scoped PMTiles downloads with an app-wide, region-based offline tile repository (vector + optional Mapterhorn DEM), managed in Settings, so map rendering and offline trail recording work anywhere within a downloaded region instead of only within a specific trail's cached cells.

**Target features:**
- Bundled `regions.json` manifest (id, name, bbox, vector PMTiles URL/size, optional DEM URL/size)
- ObjectBox `Region` + `DownloadedTilePackage` entities tracking status (notDownloaded/downloading/downloaded/updateAvailable), local paths, timestamps, disk usage
- `TileRepositoryManager`: app-wide tile lifecycle manager and global viewport-based tile-reading pipeline, fully decoupled from `Trail`
- Region downloads reuse the existing Mapterhorn DEM pipeline (`generator.go` / download-dem endpoint), now keyed to regions instead of trail cells, with an optional per-region DEM toggle
- Trail download guard: checks region coverage before a trail download, prompts to download the covering region if missing
- Settings → Offline Maps/Regions page: list regions, size breakdown (vector vs DEM), download/pause/resume/delete, DEM toggle, total disk usage
- Refactor map rendering to read from the global region registry; delete legacy trail-scoped tile download/cache code outright (no migration — app is pre-production)

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

### Active

- [ ] Bundled region manifest (id, name, bbox, vector PMTiles URL/size, optional DEM URL/size) (v1.6)
- [ ] ObjectBox `Region` + `DownloadedTilePackage` entities tracking download status, paths, timestamps, disk usage (v1.6)
- [ ] App-wide `TileRepositoryManager`: tile lifecycle + global viewport-based tile-reading pipeline, decoupled from Trail (v1.6)
- [ ] Region downloads with optional per-region Mapterhorn DEM toggle, reusing the existing DEM pipeline keyed to regions (v1.6)
- [ ] Trail download guard: region coverage check before trail download, prompt to download covering region if missing (v1.6)
- [ ] Settings → Offline Maps/Regions page: region list, size breakdown, download/pause/resume/delete, DEM toggle, disk usage total (v1.6)
- [ ] Map rendering reads from global region registry; legacy trail-scoped tile download/cache code removed (v1.6)

### Out of Scope

- Web frontend changes — `web/` already runs maplibre-gl-js; app-only apart from the v1.4 glyph/sprite endpoint
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
- Legacy trail-cache migration — app is pre-production; old trail-scoped tile/DEM cache code and files are deleted outright, no conversion path (v1.6)
- Remote/server-fetched region manifest — v1.6 ships a bundled `regions.json` app asset only; remote manifest updates deferred (v1.6)
- Polygon region geometries — v1.6 regions are bounding-box only; arbitrary polygon boundaries deferred (v1.6)
- 3D terrain/hillshade rendering changes — v1.6 only relocates the existing DEM download/storage pipeline to be region-based; the offline hillshade rendering path (`offline_style_rewriter.dart`) is reused as-is, not redesigned (v1.6)
- Background/resumable downloads across app restarts — pause/resume applies within a single app session only (v1.6)

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
- Web PR #1059 merged: new category model with translations/icon/short_name, subcategories, user category/subcategory preferences, favourite sport replaced by priority-based category ordering
- Settings infrastructure: `Settings` freezed model, `settingsProvider` with `saveToServer()` — reused by all settings screens
- `localeProvider` and `unitProvider` derived from `settingsProvider` — live-switch locale and unit system app-wide
- 14 supported locales with ARB files; AppLocalizations regenerated in Phase 6
- Background navigation via tracelet package (from quick tasks)
- Along-track projection for waypoint advancement (from quick tasks)
- Open item: dark mode (quick task 260612-gmg) was planned but never executed — the app has no Appearance/theme-mode setting yet
- Shipped v1.5 (2026-07-17): Route Planner — from-scratch route building with waypoint editing, Valhalla auto-routing, elevation profile, and handoff to trail create/edit. See `.planning/milestones/v1.5-ROADMAP.md` / `v1.5-REQUIREMENTS.md`
- Offline tiles today are **trail-scoped, not region-scoped**: `app/lib/services/trail_download_service.dart` downloads vector + optional DEM `.pmtiles` per grid cell keyed to a specific trail's bounding box; `app/lib/entities/trail_entity.dart` stores `demPmTiles` per trail. v1.6 replaces this with an app-wide region model
- Mapterhorn DEM pipeline already exists server-side: `db/services/tiles/generator.go` extracts per-cell DEM archives from `https://download.mapterhorn.com/planet.pmtiles` (const `mapterhornSource`) at `demMaxZoom = 12` (vs vector `maxZoom = 14`), stored at `pb_data/pmtiles_cache/<cellKey>_dem.pmtiles`; `db/routes/map_cells_id.go` exposes `dem_download_url`/`MapCellsDownloadDem`; `web/.../map/cells/[cellKey]/download-dem/+server.ts` proxies it. DEM generation is best-effort — failures never block the vector basemap. v1.6 reuses this pipeline keyed to regions instead of trail cells
- DEM tiles are raster hillshade only — no numeric elevation is sourced from them. Elevation gain/loss/profile comes exclusively from GPX `<ele>` track points (`gpx_util.dart`); this milestone does not change that
- `app/lib/util/offline_style_rewriter.dart` already special-cases `raster-dem` style sources (encoding `terrarium`, tileSize 512, max zoom locked to 12) and drops hillshade layers cleanly when no DEM was downloaded — this rewriting logic is reused, just needs to source from region-based files instead of trail-based ones

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
*Last updated: 2026-07-24 — Milestone v1.5 Route Planner archived.*
