# Roadmap: Wanderer Trail Navigation

## Milestones

- ✅ **v1.0 MVP** — Phases 1-3 (shipped 2026-06-13)
- ✅ **v1.1 Offline** — Phases 4-5 (shipped 2026-06-14)
- ✅ **v1.2 Settings Screens** — Phases 6-9 (shipped 2026-06-29)
- ✅ **v1.3 Category Redesign** — Phases 10-12 (shipped 2026-07-02)
- ✅ **v1.4 MapLibre Migration** — Phases 13-18 (shipped 2026-07-10)
- ✅ **v1.5 Route Planner** — Phases 19-21 (shipped 2026-07-17)
- ✅ **v1.6 Offline Region Tile Repository** — Phases 21.5, 22-27 (shipped 2026-07-24)
- 🚧 **v1.7 Admin Region Picker** — Phases 28-32 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-3) — SHIPPED 2026-06-13</summary>

- [x] Phase 1: Backend API (1/1 plans) — completed 2026-06-12
- [x] Phase 2: Navigation Screen (3/3 plans) — completed 2026-06-13
- [x] Phase 3: Stats Sheet (2/2 plans) — completed 2026-06-13

See `.planning/milestones/v1.0-ROADMAP.md` for full details.

</details>

<details>
<summary>✅ v1.1 Offline (Phases 4-5) — SHIPPED 2026-06-14</summary>

- [x] Phase 4: Serialization Fix + Entity Schema (2/2 plans) — completed 2026-06-14
- [x] Phase 5: Cache Write + Fallback + UI (4/4 plans) — completed 2026-06-14

See `.planning/milestones/v1.1-ROADMAP.md` for full details.

</details>

<details>
<summary>✅ v1.2 Settings Screens (Phases 6-9) — SHIPPED 2026-06-29</summary>

- [x] Phase 6: Settings Navigation + Language & Units (4/4 plans) — completed 2026-06-20
- [x] Phase 7: Privacy (1/1 plan) — completed 2026-06-20
- [x] Phase 8: Account & Profile (3/3 plans) — completed 2026-06-20
- [x] Phase 9: Notifications (1/1 plan) — completed 2026-06-21

See `.planning/milestones/v1.2-ROADMAP.md` for full details.

</details>

<details>
<summary>✅ v1.3 Category Redesign (Phases 10-12) — SHIPPED 2026-07-02</summary>

**Milestone Goal:** Bring the Flutter app's category system to parity with web PR #1059 — a translations/icon/short_name-aware Category model, a new Subcategory model + provider, subcategory-aware trail filters, and a Settings → Categories screen for per-category/subcategory visibility and priority preferences.

- [x] **Phase 10: Category & Subcategory Data Layer** (4/4 plans) — completed 2026-06-29
- [x] **Phase 11: Trail Filter Subcategory Support** (4/4 plans) — completed 2026-07-02
- [x] **Phase 12: Settings Categories Screen** (4/4 plans) — completed 2026-07-02

### Phase 10: Category & Subcategory Data Layer

**Goal**: The app's category data model matches web PR #1059 — categories expose locale-aware names, subcategories are fetched and cached, preference models and providers are in place, and the deprecated favourite-sport field is gone.
**Depends on**: Phase 9 (v1.2 settings infrastructure)
**Requirements**: CAT-01, CAT-02, CAT-03, CAT-04, CAT-05, SETCAT-03, SETCAT-04, SETCAT-05
**Success Criteria** (what must be TRUE):

  1. A category's display name renders in the active locale, falling back to English then the raw `name`, and exposes its `icon` and `short_name`.
  2. Subcategories load from `/subcategory` through a Riverpod provider and persist to ObjectBox with indexed `id` and `category` fields, surviving app restarts.
  3. Each subcategory carries its parent `category` id, `name`, `short_name`, `icon`, `badge_icon`, and `translations`.
  4. CategoryPreferenceNotifier and SubcategoryPreferenceNotifier providers fetch the user's preferences from their respective API endpoints.
  5. The app builds and runs with `Settings.category` removed — no remaining references to the old favourite-sport field.

**Plans**: 4 plans

  - [x] 10-01-PLAN.md — Category + Subcategory + CategoryTranslation models, locale displayName (CAT-01, CAT-02)
  - [x] 10-02-PLAN.md — CategoryEntity extension + SubcategoryEntity (indexed id/category, JSON-blob translations) (CAT-03)
  - [x] 10-03-PLAN.md — Preference models + CategoryPreferenceNotifier/SubcategoryPreferenceNotifier providers (SETCAT-03/04/05)
  - [x] 10-04-PLAN.md — CategoryNotifier ObjectBox write, cache-first SubcategoryNotifier, remove Settings.category (CAT-04, CAT-05)

### Phase 11: Trail Filter Subcategory Support

**Goal**: Users can narrow trail searches by subcategory in both the full filter screen and the quick filter bar, with category labels shown in their language and hidden categories/subcategories omitted from the picker.
**Depends on**: Phase 10
**Requirements**: FILTER-01, FILTER-02, FILTER-03, FILTER-04, FILTER-05, FILTER-06, FILTER-07
**Success Criteria** (what must be TRUE):

  1. With at least one category selected in TrailFilterScreen, the user sees subcategory chips limited to the subcategories of those selected categories.
  2. Tapping subcategory chips toggles them and changes which trails the search returns (the subcategory selection reaches the API filter payload).
  3. Category chips throughout the filter UI display locale-resolved names using the CAT-01 fallback chain.
  4. The quick filter bar's Category bottom sheet lets the user pick both categories and subcategories, and the chosen filter persists in the active trail filter.
  5. Categories and subcategories the user has marked hidden in Settings → Categories do not appear as selectable chips in the filter UI.

**Plans**: 4 plans
**UI hint**: yes

### Phase 12: Settings Categories Screen

**Goal**: A user can open Settings → Categories to control which categories and subcategories appear and in what priority order, with changes saved automatically.
**Depends on**: Phase 10 (preference providers come from Phase 10)
**Requirements**: SETCAT-01, SETCAT-02, SETCAT-06, SETCAT-07, SETCAT-08, SETCAT-09, SETCAT-10, SETCAT-11
**Success Criteria** (what must be TRUE):

  1. From SettingsScreen the user taps a "Categories" tile and lands on SettingsCategoriesScreen via the `/settings/categories` route.
  2. The screen lists categories sorted by priority (ascending, alphabetical for ties), each row showing the category icon, its locale-resolved name, a visibility switch, and a drag handle.
  3. Toggling a category's visibility switch auto-saves to `/user-category-preference`; tapping the row body navigates to `SettingsSubcategoriesScreen` for that category, which lists its subcategories with their own visibility switches saving to `/user-subcategory-preference`.
  4. Reordering categories via drag handle persists the new order via `POST /user-category-preference/reorder`; reordering subcategories persists via `POST /user-subcategory-preference/reorder`. Both reflect the saved order on reload, and a failed reorder reverts the list with an error toast.
  5. Turning off a category/subcategory that has the user's own trails shows a confirm dialog with the trail count and a link to view them before saving.

**Plans**: 4 plans

  - [x] 12-01-PLAN.md — Provider reorder methods + sort/visibility helpers + own-trail count helper + l10n keys
  - [x] 12-02-PLAN.md — SettingsCategoriesScreen: sorted list, visibility toggle, drag-handle reorder, own-trail confirm dialog
  - [x] 12-03-PLAN.md — SettingsSubcategoriesScreen: parent-scoped list + empty state, toggle, reorder, own-trail confirm dialog
  - [x] 12-04-PLAN.md — Settings "Categories" tile + go_router route wiring (SETCAT-01/02)

**UI hint**: yes

</details>

<details>
<summary>✅ v1.4 MapLibre Migration (Phases 13-18) — SHIPPED 2026-07-10</summary>

**Milestone Goal:** Replace the Flutter app's `flutter_map` + forked `vector_map_tiles`/`vector_tile_renderer` stack with the `maplibre` package, retiring both `flomp/*` forks and moving map rendering onto native GL — without ever leaving the app unbuildable and without regressing offline trail rendering.

- [x] **Phase 13: Glyph & Sprite Endpoint** - A unified `/map/style-sources` SvelteKit endpoint (replacing `/map/tileurl`) resolves tile, glyph, and sprite URLs in one object, under operator override (completed 2026-07-08)
- [x] **Phase 14: Coordinate Type Migration** - `latlong2.LatLng` → `Geographic`, `LatLngBounds` → `LngLatBounds`, test-guarded, before any map code moves (completed 2026-07-08)
- [x] **Phase 15: MapLibre Core, Trail Rendering & Offline Parity** - `WandererMap` on `MapLibreMap`; a downloaded trail renders basemap *and labels* in airplane mode (completed 2026-07-09)
- [x] **Phase 16: List & Map Screens on MapLibre** - Multi-trail list maps plus server-clustered map-screen search on native circle/symbol layers
- [x] **Phase 17: Navigation on MapLibre** - Heading-up follow, compass reset, location puck; the last `flutter_map` plugin call sites disappear (completed 2026-07-10)
- [x] **Phase 18: Retire flutter_map and the flomp Forks** - Both forks and all five packages leave `pubspec.yaml`; `maplibre` pinned (completed 2026-07-10)

#### Sequencing Rationale

Three hard constraints shape this order, and they are load-bearing rather than stylistic:

1. **The app builds and runs at every phase boundary.** `flutter_map` and `maplibre` coexist in `pubspec.yaml` from Phase 15 through Phase 17. Migration is screen-by-screen. Each phase's final success criterion asserts the un-migrated screens still render.

2. **Backend before the offline gate.** The glyph config endpoint (Phase 13) is a hard prerequisite for OFFL-04 — you cannot verify "a downloaded trail renders labels with no network" until the app has a stable URL (Wanderer-controlled, defaulting to Protomaps) to fetch and cache glyphs/sprites from. Phase 13 is SvelteKit config-route work and shares no code with Phase 14, so the two can execute in parallel; both gate Phase 15.

3. **Offline parity is a hard gate, and it forces Phase 15's size.** Today's offline path (`MultiPmTilesVectorTileProvider`) lives *inside* `wanderer_map.dart`. The moment `WandererMap` becomes a `MapLibreMap` (CORE-01), that path breaks — so `pmtiles://` (OFFL-03/05) must land in the same phase. And a downloaded trail that renders a basemap with no place names is a regression against today's bundled-font rendering, so `file://` glyphs (OFFL-01/02/04) must land there too. Phase 15 carries 20 requirements not by preference but because the offline gate cannot be deferred a phase without breaking it.

**Why the type migration is its own early phase (TYPE-01/02 → Phase 14).**
`Geographic(lon:, lat:)` reverses the argument order of `LatLng(lat, lon)`. A transposed coordinate is silent: no crash, no type error, just a track drawn in the wrong hemisphere. The affected files are GPX parsing and polyline decoding — trail *data*, not map rendering. Landing this change alone, guarded by the existing `gpx_util` / `polyline_util` tests and with `flutter_map` still rendering through boundary adapters, gives one unambiguous verification signal: *do the coordinates still come out identical?* Bundled into Phase 15, a transposed lat/lon would be indistinguishable from a maplibre camera bug.

The alternative — trailing the type change behind screen migration — was rejected: `Trail.gpxPoints` is consumed by every map screen, so flipping its type last would touch all remaining screens in one commit. That is precisely the big-bang cutover PROJECT.md rules out. The cost of going early is temporary `Geographic → LatLng` adapters at the four not-yet-migrated `flutter_map` call sites; each screen deletes its own adapter as it migrates.

**Watch for the late-closing "foundation" requirements.**
CORE-05 (`MapCompass`), CORE-06 (`animateCamera`/`fitBounds`), and CORE-07 (`enableLocation`/`trackLocation`) each read like foundation work but each *retires a file or a pubspec plugin*, and therefore cannot close until the last screen using it is migrated. `map_compass.dart` is imported by `map_screen`, `navigation_screen`, and `trail_detail_map_screen`; `AnimatedMapController` by `list_detail_map_screen`, `map_screen`, and `navigation_screen`; `CurrentLocationLayer` by `wanderer_map`, `map_screen`, and `navigation_screen`. In all three cases the last holdout is `navigation_screen`. All three are assigned to Phase 17. Earlier phases still swap their own call sites — they just cannot delete the file or drop the dependency.

### Phase 13: Glyph & Sprite Endpoint

**Goal**: The app resolves tile, glyph, and sprite URLs through a single Wanderer-controlled config endpoint, under the same operator override as tiles today — defaulting to Protomaps' public assets rather than self-hosting a copy.
**Depends on**: Nothing (Phase 12 complete; independent of Phase 14 — the two can run in parallel)
**Requirements**: GLYPH-01, GLYPH-02, GLYPH-03
**Success Criteria** (what must be TRUE):

  1. A single `/api/v1/map/style-sources` endpoint replaces `/api/v1/map/tileurl` and returns the tile URL, the glyph URL template (`{fontstack}/{range}.pbf`), and the sprite base URL in one JSON object, defaulting to Protomaps' public `basemaps-assets` host for glyphs/sprite (the same URLs the app's theme references today) when no override is set.
  2. The returned glyph template resolves valid SDF glyph PBFs for `Noto Sans Regular`, `Noto Sans Medium`, `Noto Sans Italic`, and `Noto Sans Devanagari Regular v1` (the 4th fontstack the style's data-driven `text-font` expression uses), for every `{range}` the style's 14 symbol layers request. The returned sprite base resolves `sprite.json`, `sprite.png`, and `sprite@2x.png` (light and dark variants), indexing the `arrow` icon plus the route-network shield icons the style names.
  3. An operator sets one environment variable and the config endpoint returns glyph/sprite URLs pointing at their own host instead; unset, it returns the Protomaps default. Tile URL override behavior is unchanged from today's `TILE_SERVER_URL` handling, just served from the merged endpoint.
  4. The app (`tile_url_provider.dart` and its consumer `map_style_provider.dart`) is updated to fetch and parse the unified `/map/style-sources` response instead of `/map/tileurl`; tile URL resolution behaves identically to today, and the app on today's `flutter_map` stack still builds and runs untouched. No new Go/PocketBase routes, asset vendoring, or Docker build changes — this phase is SvelteKit + a small Flutter provider change only.

**Plans**: 1 plan

Plans:

- [x] 13-01-PLAN.md — Unified `/api/v1/map/style-sources` endpoint (replaces `/map/tileurl`) + Flutter `MapConfig` provider fetch/parse

### Phase 14: Coordinate Type Migration

**Goal**: The app's trail, GPX, and position data speak maplibre's coordinate vocabulary, with the lat/lon argument-order swap isolated and test-guarded before any map code changes.
**Depends on**: Nothing (independent of Phase 13 — the two can run in parallel; both gate Phase 15)
**Requirements**: TYPE-01, TYPE-02
**Success Criteria** (what must be TRUE):

  1. Importing a GPX file produces coordinates identical to those produced before the migration — the existing `gpx_util` and `polyline_util` tests pass, extended with assertions that latitude and longitude are not transposed.
  2. A hiker opens any trail — detail, list, map, or navigation — and the track draws in exactly the same place it did before, still rendered by `flutter_map` through boundary adapters.
  3. `trail.dart`, `gpx_util.dart`, `polyline_util.dart`, and `foreground_position_stream_provider.dart` expose `Geographic` and `LngLatBounds`; no `latlong2.LatLng` survives in the data layer.
  4. The app builds and runs on the unchanged `flutter_map` stack.

**Plans**: TBD

### Phase 15: MapLibre Core, Trail Rendering & Offline Parity

**Goal**: `WandererMap` renders through `MapLibreMap`, and a hiker opening a trail — online, or downloaded with the device in airplane mode — sees basemap, place labels, icons, track, waypoints, and pins.
**Depends on**: Phase 13 (glyphs must exist to cache), Phase 14 (`Geographic` types)
**Requirements**: STYLE-01, STYLE-02, STYLE-03, STYLE-04, GLYPH-04, CORE-01, CORE-02, CORE-03, CORE-04, TRAIL-01, TRAIL-02, TRAIL-03, TRAIL-04, TRAIL-05, OFFL-01, OFFL-02, OFFL-03, OFFL-04, OFFL-05, OFFL-06
**Success Criteria** (what must be TRUE):

  1. A hiker opens a trail's map and the Protomaps basemap renders through native GL from the operator's `TILE_SERVER_URL`; place-name labels render in all four Noto Sans fontstacks (incl. Devanagari), and the `arrow` and route-shield icons appear — icons the app silently drops today.
  2. The GPX track draws as a 5px route-colored line over a 2px white casing, with directional arrows along it; waypoints are tappable and animate on selection; start and finish pins render, nudged apart when they fall within 36 screen pixels; the elevation-profile marker tracks the hiker's scrub position.
  3. Switching the app between light and dark theme swaps the map style live; the initial camera fits the trail's bounds with the caller's padding; and every map shows a scale bar plus the Protomaps/OpenStreetMap attribution the app owes under ODbL and does not display today.
  4. **The offline gate.** With the device in airplane mode, a downloaded trail renders its basemap from `.pmtiles` via native `pmtiles://` — every cell, when the trail spans several — *and renders its place-name labels* from `file://` glyphs cached at download time. Downloading a second trail reuses the cached glyphs and sprite instead of re-fetching them.
  5. ~~`lib/vendor/vector_map_tiles/pm_tile_provider.dart` is deleted~~ **CORRECTED during 15-06 execution:** this criterion contradicted itself — `navigation_screen`'s offline flutter_map path consumes `MultiPmTilesVectorTileProvider` from that exact file, so deleting it here would break the same screen this criterion requires to keep building. OFFL-06 (deletion) is deferred to Phase 17/18, when `navigation_screen` migrates off flutter_map and stops needing it. The actual criterion 5: the app still builds and runs with `flutter_map` serving `list_detail_map_screen`, `list_detail_screen`, `map_screen`, and `navigation_screen` — **met**.

**Plans**: 6 plans

Plans:

- [x] 15-01-PLAN.md — Throwaway `file://` glyph+sprite resolution spike (risk gate, physical-device verify)
- [x] 15-02-PLAN.md — Style extraction to `.json` assets + `mapStyleJsonProvider` (STYLE-01..04)
- [x] 15-03-PLAN.md — App-wide glyph/sprite cache + path-safety + download trigger (GLYPH-04, OFFL-01)
- [x] 15-04-PLAN.md — `WandererMap` on `MapLibreMap`: camera, live theme swap, chrome, markers (CORE-01..04, TRAIL-05)
- [x] 15-05-PLAN.md — Trail track/casing, static arrows, waypoint/pin markers (TRAIL-01..04)
- [x] 15-06-PLAN.md — Offline rewrite (`pmtiles://file://` + `file://`), multi-cell decision, delete vendor provider (OFFL-02..06)

**Risk gate**: This phase retires the milestone's highest-risk unknown — whether maplibre-native resolves `file://` glyph URLs for offline label rendering. Nothing downstream is safe until criterion 4 passes on a physical device in airplane mode. The first plan of this phase should be a throwaway spike proving `file://` glyph resolution against a hand-built style, *before* the phase invests in trail rendering or download-time caching. If maplibre-native rejects `file://`, the milestone needs a different offline-label strategy and this roadmap needs revision.
**UI hint**: yes

### Phase 16: List & Map Screens on MapLibre

**Goal**: The browse surfaces run on maplibre — a list's trails on one fitted map, and the map screen's trail search rendered from the server's cluster endpoint as native layers.
**Depends on**: Phase 15
**Requirements**: CORE-08, CLUS-01, CLUS-02, CLUS-03, CLUS-04, CLUS-05
**Success Criteria** (what must be TRUE):

  1. A hiker opens a list and sees every trail in it drawn on one `MapLibreMap`, with the camera animating to fit all of them; the list detail screen's inline map does the same.
  2. Panning or zooming the map screen re-queries `POST /search/trails/cluster` at the new bounds and zoom, debounced exactly as today, and the returned FeatureCollection renders as native circle layers sized by `point_count` and labelled from `point_count_abbreviated`, matching web's `ClusterLayer` step ramp.
  3. Tapping a cluster zooms the camera toward it; tapping an unclustered point selects that trail and fits the camera to its polyline.
  4. Hiding a category or subcategory in Settings → Categories changes which trails the map screen returns, because the endpoint applies the preference filters server-side.
  5. The app builds and runs; `navigation_screen` still renders on `flutter_map`.

**Plans**: 3 plans (2 waves)

- [x] 16-01-PLAN.md — Lightweight SearchMap host + list maps on MapLibre (CORE-08)
- [x] 16-02-PLAN.md — Cluster search provider + verbatim native cluster layers (CLUS-01/02/04/05)
- [x] 16-03-PLAN.md — Map screen wiring: cluster rendering, category-icon markers, native tap handling (CLUS-01..05)

**UI hint**: yes

### Phase 17: Navigation on MapLibre

**Goal**: Turn-by-turn navigation runs on maplibre with heading-up follow, compass reset, and a live location puck — and the last `flutter_map` plugin call sites disappear from `lib/`.
**Depends on**: Phase 16
**Requirements**: NAV-01, NAV-02, NAV-03, NAV-04, CORE-05, CORE-06, CORE-07
**Success Criteria** (what must be TRUE):

  1. A hiker taps Navigate and the route line, their location puck, and heading-up follow render on `MapLibreMap`; maneuver instructions advance as they move along the trail.
  2. Dragging the map during navigation breaks follow mode and the recenter control restores it, matching today's `MapEventMoveStart` / `dragStart` behavior; the compass control animates the bearing back to north.
  3. With the device offline, navigation still serves maneuvers from the ObjectBox cache and shows the offline indicator — v1.1 behavior, unregressed.
  4. The location puck and follow mode come from maplibre's `enableLocation` / `trackLocation`, camera moves from native `animateCamera` / `fitBounds`, and the compass from maplibre's built-in `MapCompass`. `lib/components/map/map_compass.dart` is deleted, and no `AnimatedMapController` or `CurrentLocationLayer` reference remains anywhere in `lib/`.

**Plans**: 3 plans (3 waves)

Plans:

- [x] 17-01-PLAN.md — Migrate navigation_screen to ml.MapLibreMap: native puck/follow, compass toggle, breadcrumb, offline pmtiles (NAV-01/02/03, CORE-06/07)
- [x] 17-02-PLAN.md — Delete map_compass.dart + pm_tile_provider.dart, strip legacy TrailLayer, restore trail-detail compass (CORE-05, OFFL-06)
- [x] 17-03-PLAN.md — On-device checkpoint: drag-break precision, follow, compass, offline navigation (NAV-01/02/03/04, CORE-07)

**UI hint**: yes

### Phase 18: Retire flutter_map and the flomp Forks

**Goal**: The forks and the old map stack leave the project — the milestone's primary payoff — and `maplibre` is pinned against its pre-1.0 breaking-change cadence.
**Depends on**: Phase 17 (every map surface migrated)
**Requirements**: CLEAN-01, CLEAN-02, CLEAN-03
**Success Criteria** (what must be TRUE):

  1. `flutter_map`, `flutter_map_animations`, `flutter_map_location_marker`, and `flutter_map_marker_cluster` are absent from `pubspec.yaml`, and `flutter pub deps` shows none of them anywhere in the dependency tree.
  2. `vector_map_tiles` and `vector_tile_renderer` are absent from `pubspec.yaml`, and `dependency_overrides` no longer names either `flomp/*` fork — the app builds from published packages only.
  3. `maplibre` is pinned to an exact version rather than a caret range, so `flutter pub upgrade` cannot pull a breaking 0.x minor.
  4. A hiker walks every map surface — trail detail, trail map, list, list map, map screen, navigation — online and in airplane mode, and each renders as it did before the migration began.

**Plans**: 3 plans (3 waves)

Plans:

- [x] 18-01-PLAN.md — Sever source deps: relocate effectiveBrightness, local LocationMarkerPosition/ServiceDisabledException classes, delete 4 dead fork/adapter files (CLEAN-01/02 prep)
- [x] 18-02-PLAN.md — Remove 6 packages + 2 flomp overrides from pubspec.yaml, pin maplibre 0.3.5 exact, whole-package analyze/deps/test gate (CLEAN-01/02/03)
- [x] 18-03-PLAN.md — On-device regression walk of all six map surfaces, online and airplane mode (CLEAN-01/02/03 verify)

See `.planning/milestones/v1.4-ROADMAP.md` for full details.

</details>

<details>
<summary>✅ v1.5 Route Planner (Phases 19-21) — SHIPPED 2026-07-17</summary>

- [x] Phase 19: Route Planner Core — Waypoint Editing & Routing Engine (4/4 plans) — completed 2026-07-16
- [x] Phase 20: Route Planner Views — Waypoint List, Elevation & Location Search (5/5 plans) — completed 2026-07-16
- [x] Phase 21: Route Planner Handoff & Entry Point (4/4 plans) — completed 2026-07-17

See `.planning/milestones/v1.5-ROADMAP.md` for full details.

</details>

<details>
<summary>✅ v1.6 Offline Region Tile Repository (Phases 21.5, 22-27) — SHIPPED 2026-07-24</summary>

- [x] Phase 21.5: Region Catalog & Archive Pre-Build (Backend) (3/3 plans) — completed 2026-07-21
- [x] Phase 22: Region & Package Data Model (3/3 plans) — completed 2026-07-22
- [x] Phase 23: TileRepositoryManager — Download Engine (6/6 plans) — completed 2026-07-22 (amended 2026-07-23 — pause/resume removed, cancel-and-restart-from-0)
- [x] Phase 24: Settings — Offline Maps/Regions UI (plans complete) — completed 2026-07-22 (amended 2026-07-23 — DEM toggle replaced by a gated DEM tile)
- [x] Phase 25: Map Rendering — Region-Based Viewport Pipeline (4/4 plans) — completed 2026-07-23
- [x] Phase 25.1: Local HTTP Tile Proxy (INSERTED) (plans complete) — completed 2026-07-23/24
- [x] Phase 26: Trail Download Guard (plans complete) — completed 2026-07-24
- [x] Phase 27: Legacy Cleanup (2/2 plans) — completed 2026-07-24 (CLEAN-02 descoped per D-05)

See `.planning/milestones/v1.6-ROADMAP.md` for full details.

</details>

### 🚧 v1.7 Admin Region Picker (Phases 28-32, in progress)

**Milestone Goal:** A server owner defines downloadable regions by toggling entries in a curated, seeded catalog — sourced from CoMaps' extract hierarchy, with real known boundaries — instead of hand-authoring `region_config.json`; the app's settings screen presents the same hierarchy.

Full design settled via `/gsd-explore` — see `.planning/notes/streamlined-region-definition.md`. Provider-source research resolved — see `.planning/research/questions.md`. Automated catalog refresh explicitly deferred — see seed `.planning/seeds/region-list-refresh-mechanism.md`.

- [x] **Phase 28: Region Catalog Data Model & Seeding** — the seeded `regions` table exists and a fresh instance boots with it fully populated (completed 2026-07-25)
- [x] **Phase 29: Polygon-Based Extraction & Region API** — the archive cron and the client-facing catalog endpoint both read from the new table (completed 2026-07-26)
- [x] **Phase 30: Admin Region Picker UI** — a server owner toggles regions on a collapsible tree with a live coverage map (completed 2026-07-27)
- [x] **Phase 31: Flutter Settings Hierarchy** — the app's region list becomes a hierarchy matching the admin tree (completed 2026-07-27)
- [ ] **Phase 32: On-Demand Polygon Fetch & Seed Slimming** — geometry leaves the repo entirely and becomes on-demand; the committed catalog drops from 54.65 MB gzipped to ~292 KB of plain hierarchy JSON

#### Sequencing Rationale

The milestone spans two backend subsystems and two independent UIs (a PocketBase admin page and a Flutter screen), and the real constraint is data dependency, not code-layer boundaries:

1. **The table must exist and be populated before anything else can read it.** Phase 28 delivers the `regions` schema, the maintainer-run seeding tool, and the auto-run migration that bulk-inserts the CoMaps hierarchy on every fresh instance startup. Nothing in Phases 29-31 is meaningful until this lands.

2. **Phase 29 (cron + API) and Phase 30 (admin UI) are siblings, not a chain.** Both need only the seeded table from Phase 28 — the cron reads `kind = 'leaf' AND enabled = true` to pick build targets, and the admin page reads/writes the same rows directly. Neither depends on the other's output, so they may be planned and executed in either order (or in parallel, per `config.json`'s `parallelization: true`).

3. **Phase 31 (Flutter) depends specifically on Phase 29, not Phase 30.** The app's Settings hierarchy is built from `GET /api/v1/regions`, which only gains `parent`/`path`/`depth` fields in Phase 29 (EXTRACT-03). The admin page is a separate PocketBase-only surface the app never talks to — Phase 30 can land before, after, or alongside Phase 31 without blocking it.

```
28 ─┬─→ 29 → 31
    └─→ 30
```

A pre-planning validation spike (not itself a phase deliverable) should confirm `pmtiles extract --region <polygon>` performs true polygon clipping against a real CoMaps `.poly`-derived boundary before Phase 29 planning commits to that extraction approach — see `.planning/todos/pending/2026-07-24-comaps-poly-region-extraction-spike.md`.

### Phase 28: Region Catalog Data Model & Seeding

**Goal**: A fresh, self-hosted Wanderer instance boots with a fully populated, hierarchical, toggleable region catalog — sourced from CoMaps' extract hierarchy — with zero admin action required.
**Depends on**: Phase 27 (the region download/archive system is fully shipped; this milestone only replaces the admin-facing region-*definition* mechanism, not the download pipeline)
**Requirements**: CATALOG-01, CATALOG-02, CATALOG-03, SEED-01, SEED-02
**Success Criteria** (what must be TRUE):

  1. The `regions` PocketBase collection exists with `comaps_id`, a self-referencing `parent`, materialized `path`, `depth`, `sort_order`, `name`, and `kind` (`group`|`leaf`) on every row; leaf rows additionally carry `polygon` (GeoJSON) and a derived `bbox`, while group rows carry neither.
  2. A maintainer can run `db/commands/seed_regions.go` against vendored CoMaps `hierarchy.txt` + `.poly` files and produce a flattened JSON seed file matching the `regions` schema.
  3. A fresh instance boots (auto-run migration, zero admin action) with the `regions` collection created and bulk-inserted from the committed JSON seed — querying the collection shows the full CoMaps group/leaf hierarchy with correct parent/path/depth relationships.
  4. Every leaf row's `enabled` defaults to `false` on first seed — no region is pre-selected for archive building on a fresh install.

**Plans**: 3 plans + 1 gap-closure plan

- [x] 28-01-PLAN.md — Hand-rolled Osmosis .poly -> GeoJSON parser + hierarchy.txt indentation-tree parser (pure, unit-tested)
- [x] 28-02-PLAN.md — seed_regions.go Cobra command: fetch CoMaps hierarchy/.poly from Codeberg, flatten to committed regions_seed.json
- [x] 28-03-PLAN.md — Auto-run migration: create the regions collection (Go SDK, self-relation) + idempotent two-pass bulk-insert from the seed
- [x] 28-04-PLAN.md — Gap closure: gzip-compress the committed regions_seed.json (730MB -> 57MB), closing SEED-02's GitHub 100MB push-limit gap found by 28-VERIFICATION.md

### Phase 29: Polygon-Based Extraction & Region API

**Goal**: The archive-generation cron builds precisely-clipped region archives directly from the seeded catalog, and the region API exposes the hierarchy so clients can render a tree.
**Depends on**: Phase 28
**Requirements**: EXTRACT-01, EXTRACT-02, EXTRACT-03
**Success Criteria** (what must be TRUE):

  1. The archive-generation cron determines its build targets strictly by querying `regions` where `kind = 'leaf' AND enabled = true` — no code path parses `region_config.json` anymore.
  2. Each enabled leaf region's PMTiles archive is produced via `pmtiles extract --region <polygon>` using that region's canonical polygon (not its bounding box) — the resulting archive's tile coverage follows the polygon boundary, not the bbox.
  3. `GET /api/v1/regions` returns each region's `parent`, `path`, and `depth` alongside its existing bbox/status/size fields, so a client can reconstruct the tree from a flat response.

**Plans**: 4 plans (3 waves)
Plans:
**Wave 1**

- [x] 29-01-PLAN.md — Table-driven cron + polygon-based `pmtiles extract --region` + relaxed region-id allow-list (EXTRACT-01, EXTRACT-02)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 29-02-PLAN.md — Hierarchy-aware `GET /api/v1/regions` (parent/path/depth/kind) + retire the `region_config.json` loader (EXTRACT-03, EXTRACT-02)
- [x] 29-03-PLAN.md — SvelteKit region-id regex lockstep + hierarchy OpenAPI docs (EXTRACT-03)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 29-04-PLAN.md — End-to-end human-verify checkpoint: real polygon-clipped build + hierarchy API + `.`-path download (EXTRACT-01/02/03)

### Phase 30: Admin Region Picker UI

**Goal**: A server owner manages the region catalog visually — toggling regions on a tree with a live coverage map — instead of hand-authoring a config file.
**Depends on**: Phase 28 (parallel-safe with Phase 29 — neither depends on the other)
**Requirements**: ADMINUI-01, ADMINUI-02, ADMINUI-03
**Success Criteria** (what must be TRUE):

  1. An admin navigates to a custom PocketBase admin page (AlpineJS bundle, reusing the `feature/ap-instance-actors` pattern) and sees the full CoMaps region catalog rendered as a collapsible tree, with group nodes expanding to reveal their child groups and leaf regions.
  2. The admin toggles a leaf region's `enabled` state directly on the tree, and the change persists immediately with no other admin action required — the cron's next run acts on it.
  3. A live map on the same page renders the boundary polygon of every currently-enabled leaf region, so the admin can see coverage before and after toggling with no separate save/publish step.

**Plans**: 2 plans (2 waves)

- [x] 30-01-PLAN.md — Custom PocketBase admin page shell + collapsible filterable region tree (ADMINUI-01)
- [x] 30-02-PLAN.md — Optimistic leaf toggle + live MapLibre coverage map of enabled polygons (ADMINUI-02, ADMINUI-03)

**UI hint**: yes

### Phase 31: Flutter Settings Hierarchy

**Goal**: The app's Settings → Offline Maps/Regions screen mirrors the admin-defined hierarchy, with every existing per-region action unregressed.
**Depends on**: Phase 29 (`GET /api/v1/regions` must expose hierarchy fields before the app can render a tree)
**Requirements**: APPUI-01, APPUI-02
**Success Criteria** (what must be TRUE):

  1. Settings → Offline Maps/Regions renders the region catalog as a collapsible hierarchy — tapping a group node expands/collapses to reveal its child groups and leaf regions — matching the shape of the admin-defined tree, replacing today's flat list.
  2. Each leaf region row still exposes its existing independent Vector and DEM download/cancel/delete controls exactly as before, now nested inside the hierarchy.
  3. The disk-usage summary (total space used, per-region breakdown) continues to work unchanged within the new hierarchical presentation.

**Plans**: 3 plans (3 waves)

- [x] 31-01-PLAN.md — sort_order backend field + RegionHierarchyRow/RegionTreeNode models + ported tree algorithm + provider hierarchy fetch (APPUI-01)
- [x] 31-02-PLAN.md — Collapsible hierarchy render on the Settings regions screen + ARB keys + widget tests (APPUI-01, APPUI-02)
- [x] 31-03-PLAN.md — Gap closure: prune the hierarchy to downloadable (enabled) regions plus ancestors (APPUI-01)

**UI hint**: yes

### Phase 32: On-Demand Polygon Fetch & Seed Slimming

**Goal**: Geometry stops being a distributed artifact and becomes on-demand. The committed catalog carries pure hierarchy — no polygon, no bbox; geometry is fetched from CoMaps at the moment intent is expressed and cached only for regions an admin actually enabled.
**Depends on**: Phase 28 (revises its seeding approach), Phase 29 (owns `buildRegion`), Phase 30 (owns the admin SPA's geometry flows)
**Requirements**: SLIM-01, SLIM-02, SLIM-03, SLIM-04, SLIM-05
**Supersedes**: CATALOG-02 entirely (leaf rows store neither `polygon` nor `bbox`), SEED-01 and SEED-02 (seed content and migration behavior both change)

**Rationale** (settled via `/gsd-explore`, 2026-07-28; revised the same day after a second pass):

The committed seed is 54.65 MB gzipped, and essentially all of it is full-precision leaf geometry — roughly 165 KB × ~1153 leaves. The hierarchy the catalog exists to serve is a rounding error by comparison. Stripping geometry from the seed takes it to 291.9 KB of plain JSON, a ~190× reduction, and solves all four original motivations at once: repo weight, GitHub's push limits, Docker image size, and a first-boot migration that parses ~216 MB of JSON.

Geometry has **two** consumers, not one. `buildRegion` reads it at archive-build time for enabled leaves; the admin SPA reads it over the PocketBase collection REST API for its coverage map, its toggle-on draw, and — critically — a hover preview that renders **any** leaf's boundary, including disabled regions that have never been built. An earlier draft of this phase proposed deleting the geometry collection outright; that was based on a `*.go`-only grep that missed the JavaScript consumer entirely. The collection survives, renamed `region_geometry` and holding both `bbox` and `polygon`, but it ships **empty** — which is sufficient, because the win comes from the seed carrying no geometry, not from the table being absent.

Geometry is therefore fetched on demand through a superuser-authenticated backend endpoint, which persists to `region_geometry` **only when the region is currently enabled**. That one rule serves both UI flows with no client cooperation: toggling a region on caches it; hovering a disabled region is a pass-through. Persisting on hover was rejected — the table has no eviction policy, so an admin idly scrolling ~1153 leaves would grow it toward the same ~55 MB just removed from git, relocating the weight into `pb_data` rather than shedding it. The endpoint must be server-side regardless of the persistence rule: CoMaps serves Osmosis-format `.poly`, and converting it is `ParsePoly` — Go, with the multi-ring hole/exclave support Phase 28 deliberately built.

`bbox` moves out of the catalog alongside `polygon` because every consumer needs it only for *enabled* regions — verified against `fitToEnabled`, `buildRegion`, and the Flutter guard's `?enabled=true` fetch. The payoff is not the 95 KB it saves but what it does to the generator: with no bbox to derive, `seed-regions` needs only `hierarchy.txt`, one HTTP request instead of ~1153. That deletes the Codeberg rate-limit problem, the retry machinery, and `ParsePoly` from the generator's path, and turns a multi-minute scrape into a one-second run — which is what finally makes CATALOG-F01's automated refresh tractable.

Fetching from CoMaps preserves reproducibility because `seed_regions.go` already pins a concrete commit SHA (`defaultCommitHash`) and never uses `main`. A pinned SHA on a raw-file endpoint is content-addressed and immutable, so this gets zero-infra hosting without giving up determinism.

Offline launch is unaffected: the catalog stays a committed in-repo artifact, so `migrate up` and the region-serving endpoints need no network. Both new network moments — `buildRegion` and the admin picker — are surfaces that cannot work offline today regardless.

**Success Criteria** (what must be TRUE):

  1. A maintainer runs `seed-regions` and gets a pure-hierarchy catalog — no `bbox`, no `polygon` — as plain, pretty-printed JSON with no gzip layer, around 292 KB, with the CoMaps commit SHA it fetched from recorded inside the artifact itself. The run issues a single HTTP request for `hierarchy.txt`.
  2. A fresh instance boots with no network, runs the migration, and serves the full 1306-row catalog through `GET /api/v1/regions` and the admin picker, with `region_geometry` present and empty.
  3. For a given region, the geometry `buildRegion` uses is value-equal to what the old seed held for that region — asserted on the parsed GeoJSON, following the `490a685f` precedent, not by comparing built archives.
  4. When the GitHub mirror is unreachable, the fetch transparently falls back to CoMaps' canonical Codeberg repository at the same commit and the build still succeeds; when both are unreachable the region is marked `status: "error"` naming which upstreams were tried, and the run continues to the next region.
  5. A region whose `region_geometry` row is deleted or corrupted rebuilds successfully on the next cron run without admin intervention — the build refetches and re-persists the geometry, and only a failed refetch marks the region errored.
  6. An admin toggles a region on and its boundary is cached in `region_geometry`; an admin hovers a disabled region, sees its true boundary outline, and no row is written.

**Design notes for planning**:

- **The gzip layer retires with the polygons.** The artifact becomes plain, pretty-printed JSON. Measured against the real 1306-row seed: stripping geometry gives compact 281.8 KB / pretty-printed 386.9 KB / one-object-per-line 283.0 KB; **also dropping bbox gives pretty-printed 291.9 KB, the shipping shape** — versus 54.65 MB gzipped today, a ~190× reduction. Gzip existed solely because the output was 55 MB (`seed_regions.go:64-69`, including its "do not drop below level 6" warning); that constraint is gone. Pretty-printed was chosen for readability. Note the tradeoff accepted: pretty-printing expands a single renamed region into ~10 changed diff lines, where one-object-per-line would show exactly one — revisit only if catalog-refresh diffs prove annoying to review in practice.
- **`bbox` moves to `region_geometry` and the generator collapses.** Verified: every bbox consumer needs it only for enabled regions (`fitToEnabled` unions over `enabledLeafRows`; `buildRegion:183` runs only for enabled leaves; the Flutter guard fetches with `?enabled=true` per commit `407b767c`). The lone exception is `RegionsList` without `?enabled=true`, whose own comment scopes it to the dev harness. Three consequences accepted: (1) unfiltered `/api/v1/regions` returns disabled leaves with no bbox, and the Flutter parser drops entries missing it — a real behavior change to a shipped endpoint, though disabled regions aren't downloadable anyway; (2) an enabled region whose fetch failed has no bbox at all, so the API needs a defined shape for `status: "error"` rows; (3) `bboxChanged` in `staleness.go` moves its comparison source from the catalog record to the fetch result.
- **Build-time geometry resolution is self-healing, and the malformed case is the trap.** `buildRegion` resolves in strict order: read `region_geometry` → on **absent *or* malformed** row, refetch and re-persist → only on refetch failure, `setError`. A literal "fetch on cache miss" implementation silently misses the second condition: a corrupt row exists, so there is no miss, and today's `builder.go:238-241` dead-ends it with `log` + `return` — leaving the region permanently broken. Geometry is derived state with an authoritative upstream, so a bad local copy is a cache fault, not data loss.
- **Geometry has two consumers, and the admin SPA is the one that constrains the design.** `db/routes/regions_ext/regions_ui.html` hits the collection REST API at lines 1110 (`loadEnabledPolygons`), 1157 (`addPolygonForRow`), and 1183 (`onLeafHoverStart`). Only the hover flow needs geometry for regions that are disabled and never built — that is what forces the pass-through endpoint. Lines 1110 and 1157 can keep reading the collection directly, since enabled regions always have rows under the persist-on-enable rule.
- **Two migration-side guards retire with it.** The token-by-token streaming decoder and the `io.LimitReader(gzReader, 512<<20)` decompression-bomb bound (`1785000000_create_regions_collection.go:142-157`) both exist only because `map[string]any` polygons could balloon peak heap past 512 MB on small hosts. At ~387 KB the migration collapses to a plain `ReadAll` + `Unmarshal` into `[]SeedRow`. Note this is a code-clarity win, not a speed one — the performance gain comes entirely from dropping the polygons, not from dropping gzip.
- **`ParsePoly` is the only bbox source** — it returns `(geometry, [4]float64, error)` and CoMaps publishes no separate bbox. That is *why* bbox travels with the polygon into `region_geometry` rather than staying in the catalog: deriving it at seed time would force the generator to keep scraping all ~1153 `.poly` files for data only enabled regions ever use. bbox remains load-bearing offline for enabled regions — `regions_get.go:99` serves it and `app/lib/util/trail_coverage_util.dart` runs the on-device download-guard overlap math against a local catalog snapshot — which is satisfied because a region cannot be enabled without its geometry having been fetched.
- **The pinned SHA travels with the catalog, not as a shared Go const.** A const goes stale the moment someone regenerates with `--commit X`, silently desyncing geometry from hierarchy. Writing the SHA that `seed-regions` actually used into the artifact makes that desync structurally impossible.
- **GitHub is primary, Codeberg is fallback — deliberately, and this is not a reversal.** `seed_regions.go:19-31` chose GitHub because Codeberg's raw endpoint enforces ~250 requests/600s, which a ~1150-file maintainer run routinely exhausts. That limit is irrelevant to a single on-demand leaf fetch, so Codeberg is a sound fallback here even though it is a poor primary there. Note Codeberg is CoMaps' canonical home and GitHub is the mirror. Codeberg's Forgejo raw URL form (`/{owner}/{repo}/raw/commit/{sha}/{path}`) should be verified against a real request during planning.
- **The disputed-territory special case dissolves.** Five leaves share a `comaps_id` across two paths (Jerusalem, Crimea, Abkhazia, South Ossetia, Campo de Hielo Sur). Fetching keyed on `comaps_id` serves both paths from the same `.poly`, so the "path is the safe join key" constraint that shaped the Phase 28 migration no longer applies to geometry.
- **Existing instances need the collection dropped**, not just newly-created ones skipping it.
- **Tests that relied on seeded polygons** need network access or local fixtures; prefer fixtures.
- **Availability becomes a product of three services** (Mapterhorn, Protomaps, and now CoMaps' host). Reuse the existing `fetch` 429/`Retry-After` backoff, and make failures name the upstream that failed.

**Plans**: 6 plans (5 waves)

**Wave 1**

- [x] 32-01-PLAN.md — Slim `seed-regions` to one `hierarchy.txt` request, relocate `ParsePoly` into `services/regions`, regenerate the 292 KB plain-JSON catalog (SLIM-01)

**Wave 2** *(blocked on Wave 1)*

- [x] 32-02-PLAN.md — Edit the migration in place: hierarchy-only `regions` with `catalog_commit`, empty `region_geometry` holding bbox + polygon, hook rebind (SLIM-02, SLIM-04)
- [x] 32-03-PLAN.md — Two-host on-demand `.poly` fetcher (GitHub primary, Codeberg fallback) + pure unit tests + D-07 value-equality proof (SLIM-03)

**Wave 3** *(blocked on Wave 2)*

- [ ] 32-04-PLAN.md — `ResolveGeometry` self-heal store, `buildRegion` restructure, `RegionsList` bbox join from `region_geometry` (SLIM-03, SLIM-04)

**Wave 4** *(blocked on Wave 3)*

- [ ] 32-05-PLAN.md — Superuser-gated `GET /regions/{id}/geometry` + admin picker hover repoint + `fitToEnabled` bbox fix (SLIM-05, SLIM-04)

**Wave 5** *(blocked on Wave 4)*

- [ ] 32-06-PLAN.md — Retire the gzip artifact and purge the ~55 MB blob from `feature/app` history (SLIM-01, SLIM-04) — contains a blocking force-push decision checkpoint

**UI hint**: no

## Progress

**Execution Order:**
Phases 13 and 14 are independent and may execute in either order or in parallel; 15-27 are strictly sequential:

```
13 ─┐
    ├─→ 15 → 16 → 17 → 18 → 19 → 20 → 21 → 22 → 23 → 24 → 25 → 26 → 27
14 ─┘
```

v1.7 continues from Phase 27. Phase 29 and Phase 30 both depend only on Phase 28 and may execute in parallel; Phase 31 depends specifically on Phase 29. Phase 32 revises Phase 28's seeding approach and changes Phase 29's `buildRegion`, so it follows both:

```
28 ─┬─→ 29 ─┬─→ 31
    │       └─→ 32
    └─→ 30
```

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Backend API | v1.0 | 1/1 | Complete | 2026-06-12 |
| 2. Navigation Screen | v1.0 | 3/3 | Complete | 2026-06-13 |
| 3. Stats Sheet | v1.0 | 2/2 | Complete | 2026-06-13 |
| 4. Serialization Fix + Entity Schema | v1.1 | 2/2 | Complete | 2026-06-14 |
| 5. Cache Write + Fallback + UI | v1.1 | 4/4 | Complete | 2026-06-14 |
| 6. Settings Navigation + Language & Units | v1.2 | 4/4 | Complete | 2026-06-20 |
| 7. Privacy | v1.2 | 1/1 | Complete | 2026-06-20 |
| 8. Account & Profile | v1.2 | 3/3 | Complete | 2026-06-20 |
| 9. Notifications | v1.2 | 1/1 | Complete | 2026-06-21 |
| 10. Category & Subcategory Data Layer | v1.3 | 4/4 | Complete | 2026-06-29 |
| 11. Trail Filter Subcategory Support | v1.3 | 4/4 | Complete | 2026-07-02 |
| 12. Settings Categories Screen | v1.3 | 4/4 | Complete | 2026-07-02 |
| 13. Glyph & Sprite Endpoint | v1.4 | 1/1 | Complete    | 2026-07-08 |
| 14. Coordinate Type Migration | v1.4 | 1/0 | Complete    | 2026-07-08 |
| 15. MapLibre Core, Trail Rendering & Offline Parity | v1.4 | 6/6 | Complete   | 2026-07-09 |
| 16. List & Map Screens on MapLibre | v1.4 | 3/3 | Complete   | 2026-07-09 |
| 17. Navigation on MapLibre | v1.4 | 3/3 | Complete   | 2026-07-10 |
| 18. Retire flutter_map and the flomp Forks | v1.4 | 3/3 | Complete   | 2026-07-10 |
| 19. Route Planner Core — Waypoint Editing & Routing Engine | v1.5 | 4/4 | Complete   | 2026-07-16 |
| 20. Route Planner Views — Waypoint List, Elevation & Location Search | v1.5 | 5/5 | Complete   | 2026-07-16 |
| 21. Route Planner Handoff & Entry Point | v1.5 | 4/4 | Complete   | 2026-07-17 |
| 22. Region & Package Data Model | v1.6 | 2/2 | Complete   | 2026-07-22 |
| 23. TileRepositoryManager — Download Engine | v1.6 | 6/6 | Complete   | 2026-07-22 |
| 24. Settings — Offline Maps/Regions UI | v1.6 | 4/4 | Complete   | 2026-07-23 |
| 25. Map Rendering — Region-Based Viewport Pipeline | v1.6 | 4/4 | Complete   | 2026-07-23 |
| 26. Trail Download Guard | v1.6 | 5/5 | Complete   | 2026-07-24 |
| 27. Legacy Cleanup | v1.6 | 2/2 | Complete    | 2026-07-24 |
| 28. Region Catalog Data Model & Seeding | v1.7 | 4/4 | Complete    | 2026-07-26 |
| 29. Polygon-Based Extraction & Region API | v1.7 | 4/4 | Complete   | 2026-07-26 |
| 30. Admin Region Picker UI | v1.7 | 2/2 | Complete   | 2026-07-27 |
| 31. Flutter Settings Hierarchy | v1.7 | 3/3 | Complete   | 2026-07-27 |
| 32. On-Demand Polygon Fetch & Seed Slimming | v1.7 | 3/6 | In Progress|  |
