# Roadmap: Wanderer Trail Navigation

## Milestones

- ✅ **v1.0 MVP** — Phases 1-3 (shipped 2026-06-13)
- ✅ **v1.1 Offline** — Phases 4-5 (shipped 2026-06-14)
- ✅ **v1.2 Settings Screens** — Phases 6-9 (shipped 2026-06-29)
- ✅ **v1.3 Category Redesign** — Phases 10-12 (shipped 2026-07-02)
- ✅ **v1.4 MapLibre Migration** — Phases 13-18 (shipped 2026-07-10)
- ✅ **v1.5 Route Planner** — Phases 19-21 (shipped 2026-07-17)
- ✅ **v1.6 Offline Region Tile Repository** — Phases 21.5, 22-27 (shipped 2026-07-24)
- ✅ **v1.7 Admin Region Picker** — Phases 28-32 (shipped 2026-07-28)
- 🚧 **v1.8 Offline Recording & Deferred Upload** — Phases 33-36 (in progress)
- 📋 **Unscheduled** — Phase 37 (no milestone yet; must not start before Phase 36 completes)

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

<details>
<summary>✅ v1.7 Admin Region Picker (Phases 28-32) — SHIPPED 2026-07-28</summary>

- [x] Phase 28: Region Catalog Data Model & Seeding (4/4 plans) — completed 2026-07-26
- [x] Phase 29: Polygon-Based Extraction & Region API (4/4 plans) — completed 2026-07-26
- [x] Phase 30: Admin Region Picker UI (2/2 plans) — completed 2026-07-27
- [x] Phase 31: Flutter Settings Hierarchy (3/3 plans) — completed 2026-07-27
- [x] Phase 32: On-Demand Polygon Fetch & Seed Slimming (6/6 plans) — completed 2026-07-28

See `.planning/milestones/v1.7-ROADMAP.md` for full details.
Audit: `.planning/milestones/v1.7-MILESTONE-AUDIT.md` (status `gaps_found` — verification coverage, accepted at close).

</details>

### 🚧 v1.8 Offline Recording & Deferred Upload (Phases 33-36, in progress)

**Milestone Goal:** A hiker who records a trail with no signal can save it, review it, and fill in its details on the spot — and it uploads itself when the phone next has a connection, without the hiker doing anything.

- [x] **Phase 33: Conversion Correctness** - Corrected GPX→trail metrics in the shared TS computation (`gpx.ts`, `gpx-metrics-computation.ts`, `gpx_util.ts`), fixing four defects plus GPS-jitter-inflated distance before anything ports or builds on top of them (re-verification found 3 new regressions 2026-07-31 — see 33-VERIFICATION.md) (completed 2026-07-31)
- [x] **Phase 34: Dart Conversion Port** - The app computes trail metrics from a GPX entirely on-device (including moving time for recordings), pinned against the corrected TS by a shared fixture test; `/trail/convert` becomes transcode-only (7/7 plans; UAT gaps closed, security audit 0 threats open) (completed 2026-08-01)
- [x] **Phase 35: Offline Trail Creation** - `trail_create_screen` is fully usable with no connection: map, tags, GPX import, and a clear message for formats that need one (completed 2026-08-02)
- [ ] **Phase 36: Local-First Recording & Automatic Upload** - A recording saves instantly with no connection, stays in the hiker's own-trails list, and uploads itself once the phone is back online (8/8 plans executed; UAT returned 5 diagnosed gaps + 1 blocked test — 7 gap closure plans 36-09..36-15 planned)

#### Sequencing Rationale

Four phases, strictly sequential — each one's success criteria depend on groundwork the previous phase lays down:

1. **Fix the shared math before porting it.** Phase 33 touches only `web/src/lib/models/gpx/gpx.ts`, `gpx-metrics-computation.ts`, and `gpx_util.ts` — no app changes. Phase 34's PORT-02 pins the Dart port against these exact corrected outputs with a shared fixture test; porting first would have made the four defects (CONV-01..05) permanent and indistinguishable from intended behavior in Dart.

2. **Make the app self-sufficient before breaking the endpoint.** PORT-04 turns `/api/v1/trail/convert` into a transcode-only endpoint — a breaking change to its response shape. It lands last within Phase 34, after PORT-01 (on-device conversion) and PORT-03 (every app call site switched to the Dart path), so nothing in the app still depends on the old contract when it changes.

3. **Bundle all four Offline Create/Import UX requirements in one phase, even though two have no dependency.** OFFUI-01 (blank map) and OFFUI-02 (throwing tag autocomplete) are live bugs today, independent of the conversion work, and already tracked in `.planning/todos/pending/2026-07-31-trail-create-screen-offline-gaps.md` — plan-phase can schedule them first within Phase 35's plans. OFFUI-03 (offline GPX import) and OFFUI-04 (clear non-GPX offline message) need Phase 34's on-device conversion and transcode-only contract respectively. Together the four close out the milestone's "trail_create_screen usable with no connection" target feature as one coherent, user-observable capability.

4. **Recording before sync, and both after the screen works offline.** SYNC-* drains a queue that REC-* must create first — you cannot drain what doesn't exist. REC-05 (editing an unsynced recording while offline) reuses the exact map/tag fixes Phase 35 ships, so Phase 36 depends on Phase 35, not just Phase 34. REC-01 (saving a recording offline at all) needs Phase 34's on-device conversion to produce a trail with no network call.

### Phase 33: Conversion Correctness

**Goal**: Every GPX converted anywhere in Wanderer — a web upload or a server-side conversion — reports correct distance, elevation, and duration, fixing four real defects in the shared TS computation before the Dart port can be pinned against it.
**Depends on**: Nothing new this milestone (continues from Phase 32)
**Requirements**: CONV-01, CONV-02, CONV-03, CONV-04, CONV-05
**Success Criteria** (what must be TRUE):

  1. Converting a 2-point GPX track segment reports its real length instead of zero, and its centroid/bounding box sum and divide by the same point count instead of silently losing the first point.
  2. Converting a GPX with only some elevation-tagged points no longer reports a phantom drop to sea level, and a steep, low-horizontal-movement stretch (switchbacks, scrambles) is measured instead of skipped.
  3. A converted trail's distance comes from the smoothed accumulator instead of the raw, GPS-jitter-inflated haversine sum, and the dead, misaligned `cumulativeDistance` array is gone.
  4. A route planned in the web planner reports a distance that follows its anchors instead of cutting the corner at each one.

**Plans**: 5 plans (3 shipped + 2 gap closure)
Plans:
**Wave 1**

- [x] 33-01-PLAN.md — Fix the `getTotals()` off-by-one loop bound and the centroid divisor (CONV-01/02), plus the first GPX Vitest fixture suite

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 33-02-PLAN.md — Undefined-aware elevation via `parseElevation` (CONV-03) and threshold-independent elevation sampling (CONV-04), with fixtures

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 33-03-PLAN.md — Report the smoothed distance and rebuild `cumulativeDistance` index-aligned (CONV-05/D-01), rescale the crop slider (D-02)
  - The smoothed-distance half was **superseded 2026-08-01** by quick task `260801-opr` — distance is now the raw accumulator; see `.planning/REQUIREMENTS.md` CONV-05. The `cumulativeDistance` rebuild and crop-slider rescale stand.

**Wave 4** *(gap closure — from 33-VERIFICATION.md)*

- [x] 33-04-PLAN.md — Replace the removed horizontal gate with a commit-then-retract elevation noise filter so a stationary track stops fabricating 210 m of gain (CONV-04)
- [x] 33-05-PLAN.md — Make crop interpolation degenerate-safe in a testable module and stop `croppedGPX` resurrecting a discarded route (CONV-05 consumer)

**Scope note:** deliberately web-only — `web/src/lib/models/gpx/gpx.ts`, `gpx-metrics-computation.ts`, `gpx_util.ts`. CONV-06 (moving time) was originally mapped here and moved to Phase 34: pause data lives in the app's `navigation_stats_provider` (`pausedAccum`) and exists only for trails recorded in the app, so it cannot be satisfied or observed by a web-only change.

### Phase 34: Dart Conversion Port

**Goal**: The app computes a trail's name, waypoints, distance, elevation, duration, and bounding box from a GPX entirely on-device — for recordings, route-planner output, and file imports — proven identical to the corrected web implementation; the server's convert endpoint stops computing trails at all.
**Depends on**: Phase 33 (the port must be pinned against the corrected algorithm — porting the buggy TS first would make the defects permanent and unfixable without diverging)
**Requirements**: PORT-01, PORT-02, PORT-03, PORT-04, PORT-05, CONV-06
**Success Criteria** (what must be TRUE):

  1. The app derives a draft trail's name, description, waypoints, start coordinates, date, distance, elevation gain/loss, duration, and bounding box from a GPX with no network call.
  2. A shared fixture test proves the Dart and TypeScript implementations produce identical metrics for the same GPX inputs, explicitly covering the CONV-01..05 defect cases.
  3. Recordings, route-planner output, and `.gpx` file imports all produce their trail through the Dart path — `POST /trail/convert` is called for none of them.
  4. `POST /api/v1/trail/convert` transcodes kml/kmz/tcx/fit to GPX and returns it without computing a trail, and its published OpenAPI description matches the new behavior.
  5. Importing a kml/kmz/tcx/fit file while online still produces a correct trail, computed by the app from the server-transcoded GPX.
  6. A trail saved from an in-app recording reports moving time — elapsed minus the session's accumulated pause — while an imported file continues to report elapsed time.

**Plans**: 7 plans in 4 waves
Plans:
**Wave 1**

- [x] 34-01-PLAN.md — Dart GPX sanitize pass and the ported `GpxMetricsComputation` / `computeTrailMetrics`, with the CONV-01..05 defect suite
- [x] 34-02-PLAN.md — `moving_duration` end to end: PocketBase migration, OpenAPI, TS + Dart models, trail form body, web display rule
- [x] 34-03-PLAN.md — The shared on-disk `fixtures/gpx-corpus/` and the TypeScript parity suite

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 34-04-PLAN.md — `trailFromGpx` trail assembly, the Dart corpus parity suite, and retiring the app's second (buggy) metrics implementation

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 34-05-PLAN.md — All three capture paths onto the Dart path; `/trail/convert` reduced to a transcode-only helper; recording moving-time hand-off

**Wave 4** *(blocked on Wave 3 completion; 34-06 and 34-07 run in parallel)*

- [x] 34-06-PLAN.md — Online-gated `showTrackSaveOptionsSheet` for all three sources, fixing the route planner's offline dead end
- [x] 34-07-PLAN.md — `POST /api/v1/trail/convert` becomes transcode-only and returns raw GPX; OpenAPI regenerated

### Phase 35: Offline Trail Creation

**Goal**: A hiker can open `trail_create_screen` with no connection and complete every step — see the map, enter tags, import a GPX — with non-GPX formats clearly explained instead of failing generically.
**Depends on**: Phase 34 (OFFUI-03 needs the on-device Dart conversion path; OFFUI-04 needs the transcode-only endpoint contract to explain correctly). OFFUI-01/02 have no such dependency and are live bugs today (`.planning/todos/pending/2026-07-31-trail-create-screen-offline-gaps.md`) — plan-phase may sequence them first within this phase's plans.
**Requirements**: OFFUI-01, OFFUI-02, OFFUI-03, OFFUI-04
**Success Criteria** (what must be TRUE):

  1. The map on `trail_create_screen` renders from downloaded regions when there is no connection, instead of going blank.
  2. Typing a tag with no connection shows no suggestions instead of throwing, and a typed free-form tag still reaches the saved trail.
  3. Importing a `.gpx` file works with no connection, converted on-device via Phase 34's Dart path.
  4. Attempting to import a kml/kmz/tcx/fit file with no connection explains that format needs a connection and that GPX works offline, instead of a generic failure.

**Scope boundary:** criterion 3 ends at a populated `trail_create_screen` — import, convert,
draw. **Persisting that trail offline is Phase 36's job** (REC-01, widened 2026-08-01 to cover
imports as well as recordings). If Phase 36 slips behind this phase, Save must still refuse
offline *before* the hiker fills in details rather than after.

**Plans**: TBD
**UI hint**: yes

### Phase 36: Local-First Recording & Automatic Upload

**Goal**: A hiker who records a trail or uploads a GPX with no signal can save it, review it, and fill in its details on the spot — and it uploads itself the next time the phone has a connection, without the hiker doing anything.
**Depends on**: Phase 35 (REC-05's offline edit reuses the exact `trail_create_screen` map/tag fixes Phase 35 ships; REC-01 needs Phase 34's on-device conversion, carried forward)
**Requirements**: REC-01, REC-02, REC-03, REC-04, REC-05, REC-06, SYNC-01, SYNC-02, SYNC-03, SYNC-04, SYNC-05
**Success Criteria** (what must be TRUE):

  1. Capturing a trail with no connection saves it immediately into the hiker's own-trails list — whether it came from ending a recording or from importing a GPX file — with no save-failure ever shown for being offline, and the unsynced trail visibly distinguishable from both a synced trail and a trail downloaded for offline use.
  2. An unsynced trail survives app restart and stays tied to the account that captured it — a different account never sees or uploads it, and logging out never deletes it.
  3. A hiker can open, review, and edit an unsynced trail's title, description, category, and photos while still offline, on the same screen Phase 35 made offline-capable.
  4. Once the app is foregrounded with a working connection, an unsynced trail uploads on its own — with inline per-item progress visible on the trail itself (not a separate pending-uploads screen), and a manual retry when an upload fails or stalls.
  5. An interrupted upload never produces a duplicate trail when retried, and once uploaded the unsynced trail becomes an ordinary trail in place — keeping its identity in the own-trails list rather than appearing a second time.
  6. With no connection the own-trails list still renders, showing every not-yet-uploaded trail plus those downloaded trails the hiker authored themselves, and says plainly that it is showing only what is available offline.

**IA decision (2026-08-01) — unsynced trails live in the own-trails list, NOT the Library.**
The Library (`/library`) is exclusively trails the hiker *downloaded*: membership is defined
solely by `TrailEntity.savedByUserIds`, written only by `TrailDownloadService`, and removing an
item there means *un-downloading* it. The semantics run opposite to an unsynced trail (content
made here, not yet gone up), and the removal semantics collide dangerously — un-downloading is
harmless, deleting an unsynced trail is permanent data loss.

The home is **`/profile/<handle>/trails`**, already the only own-trails index in the app, made
local-first for the hiker's own handle. The trail then never moves surface when it uploads — it
just loses its badge — which is what SYNC-05 requires. Offline composition is REC-06: every
not-yet-uploaded trail, plus downloaded trails the hiker authored themselves, with the screen
stating plainly it is showing only what is available offline.

Three constraints found in the code, for plan-phase:

- **Ownership must not be expressed via `savedByUserIds`** — that field means "downloaded", and
  the two are orthogonal (a hiker can record a trail *and* later download it). Needs a separate
  owner field plus a sync-state field; `grep` for `pendingUpload|unsynced|isDraft|syncState`
  returns zero hits today.

- **All three `TrailEntity` readers filter on `savedByUserIds.containsElement(userId)`**
  (`trail_library_provider.dart:28`, `trail_provider.dart:74`, `navigation_launch_util.dart:40`),
  so an unsynced row would be silently invisible unless those gain an owner clause.

- **`TrailEntity.id` is the server id**, `@Unique(onConflict: replace)`, and `trail.id.isEmpty`
  is the create-vs-update discriminator (`trail_create_screen.dart:401`). A local trail has no
  id, so it needs a local identity plus an idempotency key — which SYNC-04 requires anyway.

Already free: `TrailEntity` is deliberately NOT purged on logout/account-switch
(`account_data_purge_util.dart:76-98`), so REC-04's "signing out never deletes it" holds.

**Scope note (2026-08-01):** REC-01…05 and SYNC-01…05 were originally worded "recording"-only,
which left this phase's own goal clause "or uploads a GPX" carried by no requirement. They are
now source-agnostic ("unsynced trail"). Phase 35's OFFUI-03 delivers offline GPX import only as
far as a populated `trail_create_screen`; **saving it is this phase's job**, and without the
widening, Save would have failed after the hiker filled in title, description, category and
photos — worse than refusing up front.

**Plans**: 15 plans in 10 waves (8 shipped + 7 gap closure)
Plans:
**Wave 1**

- [x] 36-01-PLAN.md — Local-first data model: `TrailSyncState`, collision-free local ids, and the owner/localId/syncState/localPhotos fields on both entities and their models

**Wave 2** *(blocked on Wave 1 completion; 36-02 and 36-03 run in parallel)*

- [x] 36-02-PLAN.md — App-owned unsynced-photo storage (copy, reconcile, delete, orphan sweep) plus the phase's ten English l10n strings
- [x] 36-03-PLAN.md — `local_trail_store.dart`, the single owner-scoped read/write layer for locally-captured rows, and the `TrailDownloadService` carry-forward

**Wave 3** *(blocked on Wave 2 completion; 36-04 and 36-05 run in parallel)*

- [x] 36-04-PLAN.md — The deferred-upload drain: resumable tag→trail→waypoint sequence, backoff and park, manual retry, foreground/connectivity/cold-start triggers, startup photo sweep
- [x] 36-05-PLAN.md — Sign-out warning naming the count of not-yet-uploaded trails

**Wave 4** *(blocked on Wave 3 completion; 36-06, 36-07 and 36-08 run in parallel)*

- [x] 36-06-PLAN.md — Local-first `_onSave`: three-way branch, photo copy with D-03 reporting, empty ids for every not-yet-uploaded waypoint
- [x] 36-07-PLAN.md — `/profile/<handle>/trails` goes local-first: local+network merge deduped by server id, offline banner and empty state, unsynced tap routing
- [x] 36-08-PLAN.md — `SyncStatusChip` on card and list item, and the `trail_dropdown` split (download hidden, delete confirmed as unrecoverable and blocked mid-drain)

**Wave 5** *(gap closure from 36-UAT.md; runs alone — it is wave 5's only codegen plan)*

- [x] 36-09-PLAN.md — Stop the offline reload storm: filter fallback whose slider bounds are computed from the trails on the device + bounded retry, list decoupled from filter churn, `skipLoadingOnReload`

**Wave 6** *(blocked on Wave 5; 36-10 and 36-11 run in parallel — only 36-11 runs codegen)*

- [x] 36-10-PLAN.md — Local save notifies the own-trails list: one canonical producer of the signed-in account's handle across all ten former inline sites, propagation targets extracted and proven in a live `ProviderContainer`, both save tails wired
- [x] 36-11-PLAN.md — Make an unsynced trail addressable: owner-scoped local read, `localTrailProvider`, route-location helper, `/trail/local/:localId`, dual-mode detail screen

**Wave 7** *(blocked on Wave 6)*

- [ ] 36-12-PLAN.md — Tapping an unsynced trail opens detail: divert removed, panel map pushes retargeted, local map route, navigation widget test

**Wave 8** *(blocked on Wave 7)*

- [ ] 36-13-PLAN.md — Behavioural coverage for the D-14/D-17 dropdown gating, replacing the source-grep-only signal that let the reachability gap ship; plus the whole-tree codegen reconciliation that first established the fixpoint

**Wave 9** *(blocked on Wave 8; second UAT round — the orphan blocker)*

- [ ] 36-14-PLAN.md — A local trail row is retired the moment its upload succeeds, so the post-delete orphan cannot exist: delete-or-demote inside the drain's success transaction, plus the save-routing fix a retired row forces

**Wave 10** *(blocked on Wave 9; runs alone — it holds the phase's final codegen run)*

- [ ] 36-15-PLAN.md — A permanent 404 is terminal, not retried ten times behind a chromeless spinner: bounded retry policy on `trailProvider`, a plain "no longer exists" message, and a back button in every non-data state

**User decision (2026-08-03) — the offline filter's slider bounds are COMPUTED from the
trails on the device, not hard-coded.**
36-09's offline fallback for `GET /trail/filter` originally returned a compile-time
`kOfflineTrailFilterValues` (100 km / 5000 m / 5000 m). The product owner asked whether the
bounds could come from on-device trails instead, and decided yes. Rationale: offline, the only
trails that can possibly match a search are the ones on this device, so bounds derived from
those rows fit the actual search space exactly — better than both the constant and a cached
server value, either of which would let a hiker aim a slider at 500 km when nothing on the
phone exceeds 40.

Cheap by inspection, and the expensive thing does not apply: `distance`, `elevationGain` and
`elevationLoss` are plain scalar `double?` columns on `TrailEntity` (`trail_entity.dart:28-30`),
so **no GPX parsing is involved**. The read is `Query.property(...).find()` per axis, which
returns a native double array without materialising a single `TrailEntity`.

Four points settled with the decision:

- **Predicate.** `owner == accountId OR savedByUserIds contains accountId` — the exact union of
  what this account can search offline (the library sheet's own read is the second clause; the
  own-trails read is this broad net narrowed further in Dart; the map has no offline search).
  Account-scoped because D-13 requires it: an unfiltered read would let one account's slider
  maximum disclose the length of another account's private downloaded trail.

- **Empty store.** `kOfflineTrailFilterValues` survives as the per-axis empty-store floor
  (fresh install, signed out, or an axis where every row is null), not as the primary path.

- **Rounding.** Each axis rounds strictly UP by one full step (5 km distance, 250 m elevation),
  so the longest trail is never pinned to the slider's extreme. The step doubles as the floor:
  the smallest bound the arithmetic can produce is one step, so a device holding one 800 m walk
  gets a 5 km slider. No separate floor constant — a floor above one step would put that trail
  at a SMALLER fraction of slider travel and make aiming worse.

- **`max == limit` is unchanged.** It is a property of `buildDefaultTrailFilter`, not of the
  numbers fed in, so the fallback still emits no upper-bound filter-text clause and still
  cannot exclude a trail. 36-09 pins it with computed values that differ from both the constant
  and a server fixture.

Verification honesty: the bound arithmetic is a pure function unit-tested against real numeric
input (empty, single short trail, exact-multiple rounding boundary, per-axis independence,
non-finite guards); the ObjectBox query is a thin shim left **deliberately uncovered** — this
repo has no ObjectBox test harness — and is explicitly NOT backed by a source-grep test, since
this phase already shipped a gap behind exactly that pattern. Its D-13 account scoping is
carried by 36-09's `key_links` and by an account-switch device check.

**Planner decision (2026-08-03) — the two second-round gaps are separate plans.**
UAT Test 5 produced two independent defects with different blast radii, and they were planned
apart rather than merged. 36-14 is the orphan itself, scoped (after the 2026-08-03 revision) to the upload path. 36-15 is the
retry storm, which fires for ANY trail deleted elsewhere — from the web UI, from another device
— and would have been worth shipping even if the orphan had never existed. They share no file,
so merging them would have coupled a narrow local-storage fix to a provider-wide retry policy
change with no benefit. They are sequential rather than parallel only because 36-14 edits a
`@riverpod` source without regenerating and 36-15 holds the final `build_runner` run.

**Planner decision (2026-08-03, revised same day) — no reconciliation sweep, and after the
row-retirement decision below, nothing left to reconcile.**
UAT gap 4 asked whether a synced local row should stay owner-scoped forever with no
server-state reconciliation. First answered "yes for now": a sweep would have to delete local
rows whose server record 404s, and the app lets a hiker change `serverUrl` — after which every
local row 404s against the new instance and the sweep silently destroys their trails. It needs
a server-origin marker on the row before it can be safe, which is its own design. That
reasoning still holds and no sweep is being built. The question itself has since dissolved: a
successful upload now RETIRES the local row, so the `owner != null && syncState == synced`
combination a sweep would have had to reconcile is never created. Orphans already sitting on a
pre-fix device are still not healed, which is accepted because this phase has not shipped and
the affected population is pre-fix test devices.

**Planner decision (2026-08-02, revision 1) — codegen is serialised across the gap-closure waves.**
`dart run build_runner build` takes an exclusive lock on `app/.dart_tool/build` and regenerates
every `.g.dart` in the package, not just the annotated file a plan declares — a blast radius
`files_modified` overlap analysis structurally cannot see. 36-09 and 36-11 were originally both
wave 5 and both ran it. 36-11 now depends on 36-09 for that reason alone (no semantic
dependency), 36-10 and 36-12 run no codegen at all, and 36-13 — alone in the last wave —
reconciles the whole tree once and proves a fixpoint by running `build_runner` twice. The
second UAT round extends the same rule: 36-14 edits `trail_sync_provider.dart` and runs no
codegen, and 36-15 — alone in wave 10 — inherits the fixpoint duty and re-proves it the same
way. 36-13 is no longer the phase's last plan; 36-15 is.

**Planner decision (2026-08-02) — save-time branch order.** RESEARCH.md left Open Question 1 (local-first-always vs network-first-with-offline-fallback) to plan time. Resolved as **local-first always**: both local `_onSave` branches write to ObjectBox and never touch the network, online or offline, followed by a fire-and-forget drain kick. One code path instead of two, matching the Komoot/AllTrails model the design record cites, and it makes REC-01's "no save failure caused by being offline" structurally true rather than a caught-exception behaviour. A network-first fallback was rejected because a `createTrail` that fails *after* `PUT /trail/form` succeeded would fall back to a local save and produce a duplicate on the next drain — a direct SYNC-04 violation.

**~~Planner decision (2026-08-02) — local rows survive promotion.~~ SUPERSEDED 2026-08-03.**
~~A trail keeps its `TrailEntity` row after a successful drain (`syncState` flips to `synced`,
`obxId` unchanged) rather than being deleted and re-fetched. That is what SYNC-05's "keeps its
identity in place" means concretely, and the merge in 36-07 dedupes the network hit by server id
so it can never render twice.~~ Kept on the record rather than deleted: it is what
`markTrailSynced`'s doc comment argued, what 36-01 through 36-08 were built against, and what
produced the row UAT Test 5 found had no removal affordance anywhere in the app.

**User decision (2026-08-03) — a local trail row is DELETED once it uploads successfully.**
Decided by the product owner, superseding the 2026-08-02 planner decision above. Their model,
verbatim: *"a non-synced trail appears in my own trails list. A synced one does as well because
it now lives on the server (as long as I'm online). A synced trail will not appear in my own
trails list once I'm going back offline but that is ok. Once a trail has been uploaded it needs
to be downloaded by the user to appear again in own trails."*

So the state machine is: **unsynced** → in own-trails via the local row; **synced** → in
own-trails via the network fetch, online only; **synced + offline** → absent, and that is
correct behaviour, not a regression. To have it offline again the hiker downloads it like any
other trail, gaining `savedByUserIds` through the normal path. The cost — "your just-recorded
trail vanishes offline" — was raised explicitly and accepted as the intended UX; it is not to be
re-litigated or mitigated.

This does not weaken SYNC-05. The requirement forbids a second entry and requires the trail to
become an ordinary trail: `writeServerTrailId` stamps the server id onto the row before it is
retired, so one identity is preserved throughout and exactly one entry remains — the server's.
36-07's dedupe-by-server-id stays load-bearing for the window between the server id being
stamped and the row being retired, and becomes a no-op after it. REC-06 independently supports
the change: its enumeration of the offline own-trails list is "every not-yet-uploaded trail plus
those downloaded trails the hiker authored themselves", which never included an
uploaded-and-not-downloaded trail — the retained row made the list a superset of the
requirement.

Implemented by 36-14 (rewritten for this decision): `retireUploadedLocalTrail` deletes the row
and its `WaypointEntity` children inside the drain's success transaction, or demotes it to an
ordinary downloaded row when some account holds it in its offline library. The delete-path
cleanup 36-14 previously planned is dropped — with no row surviving an upload, the orphan class
cannot exist, and `trail_dropdown.dart` needs no change at all.

**UI hint**: yes

**Open for discuss-phase:** four decisions are deliberately unresolved and belong to this phase's discuss-phase — photo file durability (`image_picker` returns paths into an OS-purgeable cache directory), partial-failure semantics of the `tag → trail → waypoint` upload sequence, whether logout with undrained unsynced trails needs a confirmation UX, and — from the 2026-08-01 scope note above — whether REC-03's "visibly distinguishable" should further distinguish a recorded trail from an imported one, or treat both simply as unsynced. Full context: `.planning/research/questions.md`, `.planning/notes/offline-recording-deferred-upload-design.md`.

---

## Unscheduled — after v1.8

Phases below are **not part of any milestone yet**. They are parked here rather than in the
backlog because their scope is already understood at file level, but they must not be picked up
while v1.8 is executing. When the next milestone is opened, `/gsd-new-milestone` should claim
them explicitly.

### Phase 37: Way Types & Surfaces Breakdown (mobile-first)

**Goal**: A hiker looking at any trail sees what they will actually be walking on — a stacked
breakdown of way types (path, footpath, track, road…) and surfaces (paved, gravel, dirt,
unpaved…) with distance per category, including off-road alpine paths that naive map-matching
silently drops.
**Milestone**: none — **explicitly NOT part of v1.8 (Offline Recording & Deferred Upload)**.
This is online-only trail enrichment and shares no requirement with REC-*/SYNC-*.
**Depends on**: Phase 36 complete — a hard sequencing constraint, not a preference. See the
file-conflict note below.
**Requirements**: TBD (derive from `.planning/todos/pending/2026-07-18-way-types-and-surfaces-breakdown.md`)
**Plans**: 0 plans

**⚠ File conflict with Phase 36 — do not execute concurrently.** Three surfaces are edited by
both:

- `app/lib/models/trail.dart` — 36-01 (done) and **36-07 (not yet executed)** both modify it.
  Phase 37 adds a `way_type_surface` field plus two new freezed classes to the same file; both
  sides regenerate `*.freezed.dart` / `*.g.dart`, so concurrent work collides in generated
  output, not just in source.

- `web/src/routes/api/v1/trail/+server.ts` (and `[id]/+server.ts`) — Phase 36's SYNC-04
  idempotency work reshapes the trail save path; Phase 37 wants to hook way-type computation
  into the same create/update handlers.

- `db/migrations/` — Phase 36 adds owner/sync-state fields to trail storage; Phase 37 adds a
  `way_type_surface` json field to the same `trails` collection (`e864strfxo14pm4`). Two
  migrations against one collection must land in a known order.

- `app/lib/components/trail/trail_panel.dart` — Phase 36's gap-closure plan 36-12 (Task 3)
  rewrites three `context.push` sites here to re-target the map route for unsynced trails.

Clean (no Phase 36 plan touches them): `app/lib/theme/colors.dart`, `web/src/lib/server/`.

**Source material:** the todo carries a complete file-level implementation plan, including the
verified root cause — Valhalla's default `pedestrian` costing caps `max_hiking_difficulty ≈ 1`,
excluding `sac_scale >= mountain_hiking` paths from the routable graph, which is why the earlier
POC dropped off-road segments (OSM way 39669166: 16 m matched by default vs 1.09 km with
`max_hiking_difficulty: 6`). Full research: `37-RESEARCH-SOURCE.md` in this phase's directory.

**Deferred follow-up:** the SvelteKit web rendering of the same persisted field is a separate,
smaller phase — not part of Phase 37.

Plans:

- [ ] TBD (run /gsd-plan-phase 37 to break down)

---

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

v1.8 continues from Phase 32. Phases 33-36 are strictly sequential — each phase's success criteria depend on groundwork the previous phase lays (see the v1.8 Sequencing Rationale above):

```
33 → 34 → 35 → 36
```

Phase 37 is unscheduled and sits outside v1.8, but it is **not parallelizable with Phase 36** —
both edit `app/lib/models/trail.dart`, the `api/v1/trail` handlers, and the `trails` collection
migrations. It starts only after 36 lands:

```
36 → (v1.8 ships) → 37
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
| 32. On-Demand Polygon Fetch & Seed Slimming | v1.7 | 6/6 | Complete   | 2026-07-28 |
| 33. Conversion Correctness | v1.8 | 5/5 | Complete    | 2026-07-31 |
| 34. Dart Conversion Port | v1.8 | 7/7 | Complete    | 2026-08-01 |
| 35. Offline Trail Creation | v1.8 | 1/0 | Complete    | 2026-08-02 |
| 36. Local-First Recording & Automatic Upload | v1.8 | 11/15 | In Progress|  |
| 37. Way Types & Surfaces Breakdown (mobile-first) | — (post-v1.8) | 0/0 | Not planned |  |
