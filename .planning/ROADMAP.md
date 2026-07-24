# Roadmap: Wanderer Trail Navigation

## Milestones

- ✅ **v1.0 MVP** — Phases 1-3 (shipped 2026-06-13)
- ✅ **v1.1 Offline** — Phases 4-5 (shipped 2026-06-14)
- ✅ **v1.2 Settings Screens** — Phases 6-9 (shipped 2026-06-29)
- ✅ **v1.3 Category Redesign** — Phases 10-12 (shipped 2026-07-02)
- ✅ **v1.4 MapLibre Migration** — Phases 13-18 (shipped 2026-07-10)
- 🚧 **v1.5 Route Planner** — Phases 19-21 (in progress)
- 🚧 **v1.6 Offline Region Tile Repository** — Phases 21.5, 22-27 (in progress)

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

### v1.5 Route Planner (In Progress)

**Milestone Goal:** A user can build a route from scratch on the map (tap/drag waypoints, optional auto-routing via Valhalla) and hand it off as a draft trail to the existing create/edit screen.

- [x] **Phase 19: Route Planner Core — Waypoint Editing & Routing Engine** - Tap/drag/insert waypoints on the map with auto-routing toggle (fixed foot/bike profile) and undo/redo (completed 2026-07-16)
- [x] **Phase 20: Route Planner Views — Waypoint List, Elevation & Location Search** - Route anchor list (delete/reorder) and live elevation profile as tabs of one docked sheet, plus location-search-to-focus (completed 2026-07-16)
- [x] **Phase 21: Route Planner Handoff & Entry Point** - New planner entry point with hike/bike dialog, handoff to trail create/edit as a draft Trail (completed 2026-07-17)

### Phase 19: Route Planner Core — Waypoint Editing & Routing Engine

**Goal**: A user can build a route from scratch directly on the map — tapping to add waypoints, dragging to reposition them, inserting mid-segment — with an auto-routing toggle (Valhalla, fixed foot/bike profile set at entry) and undo/redo, all backed by a dedicated route-planner state provider.
**Depends on**: Phase 18 (maplibre-native map stack; first phase of v1.5)
**Requirements**: WAYP-01, WAYP-02, WAYP-03, ROUTE-01, ROUTE-02, ROUTE-04, ROUTE-05
**Success Criteria** (what must be TRUE):

  1. A user can tap anywhere on the Route Planner map to add a waypoint, which appears as a marker connected to the previous waypoint by a route segment.
  2. A user can drag an existing waypoint to a new position; the segments connecting it to its neighbors re-resolve automatically to the current routing mode.
  3. A user can tap an existing route segment to insert a new waypoint between its two endpoints.
  4. A user can toggle auto-routing on (Valhalla-routed segments, using the fixed foot/bike profile set at entry) or off (straight-line segments); toggling on re-resolves every existing segment via Valhalla, toggling off leaves existing segments untouched and only affects segments created afterward.
  5. A user can undo and redo their waypoint edits; when a segment fails to auto-route (unreachable backend, no route found), it is shown as blocked with a retry action rather than silently falling back to a straight line.

**Plans**: TBD
**UI hint**: yes

### Phase 20: Route Planner Views — Waypoint List, Elevation & Location Search

**Goal**: A user can inspect and manage the in-progress route through a persistent bottom sheet with two tabs — a route anchor list (delete, reorder) and a live elevation profile — and can pan the planner map to a searched location.
**Depends on**: Phase 19
**Requirements**: WAYP-04, WAYP-05, PLANUI-01, PLANUI-02, PLANUI-03
**Success Criteria** (what must be TRUE):

  1. Once the route has at least one anchor, a docked bottom sheet is visible at peek height (draggable to expand), showing a "Route Anchors" tab with every anchor listed in route order.
  2. From the route anchor list tab, a user can delete an anchor (immediate, no confirmation — undo/redo is the safety net) or drag to reorder anchors, and the map and route update to match.
  3. A second tab in the same sheet ("Elevation") shows a live elevation profile — built from a `Gpx` synthesized incrementally from the in-progress route and fetched from `/api/v1/valhalla/height` only while that tab is visible — that updates as the route changes; with fewer than 2 anchors it shows an empty-state message instead.
  4. A user can tap a magnifying-glass map control button (top-right, above auto-routing toggle) to open a dedicated location-search screen that searches locations only (not trails, lists, or accounts); selecting a result pans/zooms the planner map to it (zoom 13).

**SCOPE CHANGE (from original PRD wording, resolved in 20-CONTEXT.md):** the list and elevation profile were originally specified as two separate views "toggled via map control buttons." Discussion converged on a simpler mechanism — one persistent tabbed sheet, no separate toggle buttons for these two views — that still satisfies the same user-visible capability (inspect route as a list or as an elevation profile, mutually exclusive at a time).

**Plans**: 5 plans (3 waves)

Plans:

- [x] 20-01-PLAN.md — Route anchor deleteAnchor/reorderAnchors mutators + buildGpxFromPoints + plannedGpxProvider (WAYP-04/05, PLANUI-02)
- [x] 20-02-PLAN.md — LocationSearchScreen (locations-only) + /location-search route (PLANUI-03)
- [x] 20-03-PLAN.md — ElevationProfile trail→Trail? + ElevationTab (tab-gated debounced height fetch) (PLANUI-02)
- [x] 20-04-PLAN.md — RouteAnchorListTab (numbered rows, immediate delete, drag reorder) (WAYP-04/05)
- [x] 20-05-PLAN.md — RouteAnchorSheet tabbed DraggableScrollableSheet + planner screen integration (search button, sheet host, camera hand-off) (PLANUI-01/03)

**UI hint**: yes

### Phase 21: Route Planner Handoff & Entry Point

**Goal**: A user reaches the Route Planner from the trail-source-select flow, chooses an initial travel profile up front, and hands off a finished plan as a draft Trail to the existing create/edit screen.
**Depends on**: Phase 20
**Requirements**: HANDOFF-01, HANDOFF-02, HANDOFF-03
**Success Criteria** (what must be TRUE):

  1. From the trail-source-select flow, a user sees a new "Plan a route" entry point alongside the existing "Import trail file" option.
  2. Tapping the new entry point shows a hike/bike selection dialog before the Route Planner screen opens; the choice sets the planner's initial travel profile, fixed for the rest of the planning session (no in-planner profile switch).
  3. From the Route Planner, a user can finish planning and hand off the route as a draft Trail (synthesized GPX track only — no Waypoint records; route anchors stay planner-only, **with elevation populated regardless of whether the Elevation tab was ever opened**) that opens directly in the existing trail create/edit screen, pre-filled with the planned route, reusing the existing `pendingImportedTrail` safety net.

**SCOPE CHANGE (resolved in 21-CONTEXT.md):** HANDOFF-01 was originally worded "synthesized GPX + named waypoints." Discussion clarified route anchors never become `Waypoint` records — the draft Trail carries only the synthesized GPX track (with elevation).

**Implementation note (carried over from Phase 20):** `plannedGpxProvider` (Phase 20) intentionally stays pre-elevation — the Elevation tab's `/api/v1/valhalla/height` fetch is gated on tab visibility (D-11) and its ele-merged result lives only in that tab's local widget state, never written back to the shared provider. This phase must add a **one-time elevation fetch at handoff time**: before constructing the draft Trail, fetch `/api/v1/valhalla/height` once against the final `plannedGpxProvider` route and merge `ele` into the handed-off GPX. Deliberately not a continuous background fetch in `plannedGpxProvider` — that would re-fire Valhalla on every anchor edit regardless of tab visibility, which is exactly what D-11 avoided. A single fetch at the moment of handoff is sufficient and cheaper.

**Plans**: 4 plans (3 waves)

Plans:

- [x] 21-01-PLAN.md — Handoff logic: categoryForTravelProfile + route_planner_handoff_util (one-time elevation merge, GPX-track-only draft Trail) (HANDOFF-01)
- [x] 21-02-PLAN.md — Settings Behavior port: allowAutoGeolocate field + SettingsEntity behaviorJson (HANDOFF-03/D-03)
- [x] 21-03-PLAN.md — Entry point: hike/bike modal sheet + trail-source card wiring + real /route-planner registration (HANDOFF-02/03)
- [x] 21-04-PLAN.md — App-bar Finish action + undo/redo relocation to map controls, wired to finishPlanning (HANDOFF-01/D-04/D-05)

**UI hint**: yes

### v1.6 Offline Region Tile Repository (In Progress)

**Milestone Goal:** Replace trail-scoped PMTiles downloads with an app-wide, region-based offline tile repository (vector + optional Mapterhorn DEM), managed in Settings, so map rendering and offline trail recording work anywhere within a downloaded region instead of only within a specific trail's cached cells.

- [x] **Phase 21.5: Region Catalog & Archive Pre-Build (Backend)** - Go backend reads an admin-supplied, Docker-volume-mounted config file defining this instance's regions, a cronjob pre-builds one mosaicked vector PMTiles + one DEM archive per region, and an API endpoint serves the resulting catalog (completed 2026-07-21)
- [x] **Phase 22: Region & Package Data Model** - App-side region manifest (fetched from Phase 21.5's API, not bundled) and ObjectBox `Region`/`DownloadedTilePackage` entities with an explicit-int status enum (completed 2026-07-22)
- [x] **Phase 23: TileRepositoryManager — Download Engine** - Disk-safe region downloads plus a bbox-to-local-paths query, fully decoupled from Trail (completed 2026-07-22; amended 2026-07-23 — pause/resume/backgrounding-pause replaced by cancel-and-restart-from-0, see Phase 23 note)
- [x] **Phase 24: Settings — Offline Maps/Regions UI** - Flat searchable region list, independent Vector/DEM tiles with download/cancel/delete and total disk usage (completed 2026-07-22; amended 2026-07-23 — DEM toggle replaced by a gated DEM tile, see Phase 24 note)
- [x] **Phase 25: Map Rendering — Region-Based Viewport Pipeline** - `TrailMap`/`navigation_screen` read region tiles through a viewport-scoped style pipeline, settled by a maplibre 0.3.5 spike (completed 2026-07-23)
- [x] **Phase 26: Trail Download Guard** - Trail downloads check region coverage first, naming missing regions with an inline download CTA (completed 2026-07-24)
- [ ] **Phase 27: Legacy Cleanup** - Trail-scoped tile code deleted outright, orphaned legacy files swept on first launch

#### Sequencing Rationale

This milestone follows research/SUMMARY.md's recommended build order almost verbatim — data model → download engine → Settings UI → map-screen rewiring (with a spike) → trail guard → legacy ripout — because each step is either purely additive to the running app or swaps exactly one thing while the old trail-scoped path stays physically present until proven redundant, mirroring the discipline validated in v1.4 ("forks deleted last").

0. **Backend region catalog and archive pre-build come first (Phase 21.5), ahead of any Flutter work.** Every later phase's data model, download engine, and UI assume a region resolves to one real, already-built downloadable file — that file has to exist, generated by an admin-configured, cron-driven backend, before the app-side manifest model (Phase 22) can even be shaped correctly. This reverses an earlier assumption (see note below) that the existing per-cell backend needed no changes.

1. **Data model next (Phase 22), with zero UI.** `Region`/`DownloadedTilePackage` are new ObjectBox entities and an app-side manifest fetched from Phase 21.5's catalog API — nothing downstream (download engine, UI, rendering) can exist without this schema, and getting the status-enum persistence contract right here (explicit int constants, not `Enum.values[index]`) avoids a data-corruption class of bug research flagged as expensive to retrofit once real downloads exist on-device.

2. **Download engine before any UI depends on it (Phase 23).** Resume, disk-space pre-check, and backgrounding-aware pause are download-primitive concerns specific to region-sized files (10s-100s of MB) that the old trail-cell code never needed. Building and proving these against `TileRepositoryManager` directly — before Settings UI exists to mask failures behind a spinner — is cheaper than discovering a resume bug after the UI ships.

3. **Settings UI next (Phase 24), independently demoable.** The region list, download/pause/resume/delete, DEM toggle, and disk-usage summary are user-visible and testable without touching any existing map or trail flow — this validates Phase 23 end-to-end before anything else depends on it.

4. **Map rendering is pulled forward, ahead of the guard (Phase 25).** This is the architectural crux — "app-wide, region-based" — and PITFALLS.md flags it as the piece most likely to need rework if discovered late (unconfirmed maplibre 0.3.5 incremental-source API, unproven layer-count scaling past a handful of trail cells). An explicit spike settles the composition strategy before the guard (Phase 26) or ripout (Phase 27) build on top of it.

5. **The trail guard (Phase 26) comes after the region system it points users into is real.** Its dialog offers an in-dialog "Download region" CTA — that only makes sense once Phase 23's download engine and Phase 24's UI patterns are proven, not while the download lifecycle is still being designed.

6. **Legacy ripout is always last (Phase 27).** `trail_download_service.dart`'s tile methods and `TrailEntity.pmTiles`/`demPmTiles` are deleted only once the region path has replaced them end-to-end (rendering + guard both live) — deleting the fallback earlier would leave no working offline path if something upstream needed rework. The one-time orphaned-file cleanup sweep ships in this same phase, not as a follow-up, so the new disk-usage figure is trustworthy from the moment the old code is gone.

**Correction (2026-07-21):** This milestone was originally planned assuming backend needed no changes — the grid-cell/`bbox` endpoints (`db/routes/map_cells_id.go`, `db/services/tiles/generator.go`) are trail-agnostic, so "region" was treated as a purely client-side, bundled-manifest concept. Discussion after Phase 22 was planned (but before execution) surfaced that this doesn't hold: a region download should be one resumable file, not N per-cell requests, and for a self-hostable app the region list itself is an admin decision per instance, not a fixed bundled asset. Phase 21.5 was inserted to address both. See `.planning/notes/region-catalog-backend-decision-trail.md` for the full reasoning and `.planning/todos/pending/replan-phase-22-region-manifest.md` for the resulting Phase 22 replan requirement.

### Phase 21.5: Region Catalog & Archive Pre-Build (Backend)

**Goal**: A Wanderer instance admin can define the regions their instance offers for offline download in a config file; the backend pre-builds each region's vector + DEM archive ahead of time via a cronjob and serves the resulting catalog through an API endpoint — so a client-side download is always a single, already-ready file.
**Depends on**: Phase 21 (v1.5 complete; first phase of v1.6)
**Requirements**: BACK-01, BACK-02, BACK-03, BACK-04, BACK-05
**Success Criteria** (what must be TRUE):

  1. On startup, the Go backend reads a region catalog (id, name, bbox per region) from a config file mounted via Docker volume; an instance with no config file or an empty one serves an empty catalog rather than erroring.
  2. A cronjob pre-builds a single mosaicked vector PMTiles archive per configured region, merging the grid cells covering that region's bbox, without waiting for any user request. **SCOPE CHANGE (resolved in 21.5-CONTEXT.md, D-01/D-02):** implemented as a direct `pmtiles extract --bbox=<region bbox>` against the Protomaps/Mapterhorn sources per region, not a merge of existing per-cell grid archives — same end result (one archive per region, built ahead of request), simpler build path.
  3. The same cronjob pre-builds a single DEM archive per configured region on the same basis, reusing the existing Mapterhorn extraction pipeline as its data source.
  4. `GET /api/v1/regions` (auth-gated, any logged-in user — served internally by the Go backend and proxied at the same public path by a thin SvelteKit route, since the Go backend's port is not published in the real deployment compose files) returns this instance's region catalog — id, name, bbox, status, version, vector archive URL + size, DEM archive URL + size — reflecting only what pre-built successfully.
  5. Cron regeneration only rebuilds a region's archive when its underlying source tiles changed since the last build; exact cadence and staleness-detection mechanics are open questions for this phase's discuss-phase step, not locked here.

**Plans**: 3 plans (3 waves)

Plans:

- [x] 21.5-01-PLAN.md — Region config loader (id/bbox validation, path-safety) + `region_archives` collection migration (BACK-01)
- [x] 21.5-02-PLAN.md — Archive builder: vector + DEM `pmtiles extract` with atomic rename, date-gated vector rebuild, build-once DEM, in-flight guard (BACK-02/03/05)
- [x] 21.5-03-PLAN.md — Auth-gated `GET /api/v1/regions` catalog + archive download routes, daily build cron, docker-compose config wiring (BACK-04/01/05)

**Note:** This phase reverses an assumption made when Phase 22 was originally planned (that no backend change was needed). See the milestone-level "Correction" note above `Phase 22` for context, and the linked note/todo for what this means for Phase 22's already-drafted plans.

### Phase 22: Region & Package Data Model

**Goal**: The app has a region manifest — fetched from Phase 21.5's catalog API, not bundled — and an ObjectBox schema for regions and their downloadable tile packages — the foundation every later phase in this milestone builds on.
**Depends on**: Phase 21.5 (region catalog API must exist for the app to fetch from)
**Requirements**: REGN-01, REGN-02, REGN-03
**Success Criteria** (what must be TRUE):

  1. The app fetches its region catalog from this instance's backend API at runtime and parses it into a typed manifest model exposing, per region, an id, name, bbox, vector archive URL + size, and optional DEM archive URL + size.
  2. An ObjectBox `Region` entity persists every fetched-catalog field plus a live status (notDownloaded/downloading/downloaded/updateAvailable) backed by explicit stable int constants — never `Enum.values[index]` — and the status survives an app restart.
  3. An ObjectBox `DownloadedTilePackage` entity tracks the vector and DEM packages for a region independently — separate local file path, timestamp, size on disk, and status per package — so a region can show its vector package downloaded while its DEM package is not.
  4. The app builds and runs unchanged; nothing yet reads from the new entities.

**Plans**: 2 plans (2 waves) — replanned 2026-07-22 against the Phase 21.5 backend-fetched catalog design (supersedes the earlier bundled-`regions.json` drafts).

Plans:

- [x] 22-01-PLAN.md — `RegionCatalogEntry` @freezed parse model for `GET /api/v1/regions` + `CatalogStatus`/`RegionStatus`/`PackageStatus` explicit-`.code` enums + ObjectBox `RegionEntity` (computed status getter, two `ToOne` package links, `fromCatalogEntry`/`applyCatalogEntry`) + `DownloadedTilePackageEntity` (REGN-01/02/03)
- [x] 22-02-PLAN.md — `RegionRepository` fetch-and-upsert (`fetchRegionCatalog`, upsert-by-id preserving local state, orphan `inCatalog` flip, typed `RegionCatalogException`) + construction-only `regionRepository` provider seam (REGN-01)

### Phase 23: TileRepositoryManager — Download Engine

> **Amended 2026-07-23** (post-completion, no new phase): pause/resume was removed entirely — Dio's `deleteOnError` deletes the `.part` file on ANY cancellation (deliberate pause included, not just genuine errors), so the first pause of a fresh download destroyed its own resume progress. Replaced with cancel-deletes-and-restarts-from-0 (commits `3adeb11c`, `4732d20e`). Criteria 1/2/4 below are historical (what Phase 23 originally shipped) — see TILE-01/02/04 in REQUIREMENTS.md for current behavior. Do not treat this as an unresolved gap in a later phase.

**Goal**: A region's vector and DEM tile packages can be downloaded and deleted through one app-wide manager, safely and independent of any Trail. ~~paused, resumed,~~ (superseded, see amendment above)
**Depends on**: Phase 22
**Requirements**: TILE-01, TILE-02, TILE-03, TILE-04, TILE-05, DEM-01, DEM-02
**Success Criteria** (what must be TRUE):

  1. ~~`TileRepositoryManager` starts, pauses, resumes, and deletes a region's vector download~~ — superseded: starts, cancels, and deletes. The DEM download fetches that region's pre-built DEM archive from the same catalog, independently of the vector package.
  2. ~~Interrupting an in-progress region download and resuming it continues from a partial file via HTTP Range + `FileAccessMode.append`, not from byte 0, within the same app session (no cross-restart resume).~~ — superseded: there is no resume, at any level. Cancelling always deletes the `.part` file; a later download always restarts from byte 0.
  3. Before each file write, available disk space is checked with a safety margin; a download that would exceed it is refused with a specific state rather than partially writing a corrupt file.
  4. ~~The app backgrounding mid-download... leaves the download in a deliberate paused state that resumes cleanly on foreground~~ — superseded: there is no pause state. Downloads keep running in the background as long as the OS allows; if the OS kills the transfer, it ends in `error`, retryable by the user.
  5. `localTilePathsForBounds(bbox)` returns the local vector/DEM file paths for every downloaded region intersecting a given bounding box, ready for map rendering to consume.

**Plans**: 6 plans (4 waves)

Plans:

- [x] 23-01-PLAN.md — SvelteKit regions download/DEM proxy: forward Range in, forward 206/Content-Range out (TILE-02 prerequisite)
- [x] 23-02-PLAN.md — Append paused/error to PackageStatus/RegionStatus + RegionEntity.status getter + region_file_path.dart id allow-list (TILE-01/04)
- [x] 23-03-PLAN.md — disk_space_2 legitimacy checkpoint + disk_space_util fail-closed margin check (TILE-03)
- [x] 23-04-PLAN.md — TileRepositoryManager download engine: resumable .part+Range+append vector/DEM, disk pre-check, PMTiles validation, backgrounding pause (TILE-01/02/03/04, DEM-01/02)
- [x] 23-05-PLAN.md — localTilePathsForBounds + bboxOverlaps + deleteRegion cascade + tile_repository_provider/RegionDownloadState (TILE-05/01)
- [x] 23-06-PLAN.md — On-device checkpoint: resume-from-partial, disk refusal, backgrounding pause, DEM independence, bbox query

### Phase 24: Settings — Offline Maps/Regions UI

> **Amended 2026-07-23** (post-completion, no new phase): the single-row-with-DEM-toggle design was replaced by two independent list tiles per region — Vector and Elevation data — each with its own download/cancel/delete action and progress bar (commit `4732d20e`). The DEM tile is additionally gated on Vector being `downloaded`/`updateAvailable` — hillshading without a basemap underneath it doesn't make sense — showing a disabled download button + explanatory subtitle until then (commit `663f049a`). Criteria 3/4 below are historical; see SETUI-03/04 in REQUIREMENTS.md for current behavior. Do not treat this as an unresolved gap in a later phase.

**Goal**: A user can discover, download, manage, and monitor offline regions entirely from Settings.
**Depends on**: Phase 23
**Requirements**: SETUI-01, SETUI-02, SETUI-03, SETUI-04, SETUI-05, SETUI-06
**Success Criteria** (what must be TRUE):

  1. From Settings, a user opens "Offline Maps/Regions" and sees a flat, searchable list of every bundled region (no hierarchical tree).
  2. Each region row shows its name, current 4-state status, and a size breakdown (vector vs DEM) visible before any download starts.
  3. ~~A user can download, pause, resume, or delete a region directly from its row, with visible progress while downloading.~~ — superseded: a user can download, cancel, or delete Vector and DEM independently, each with its own visible progress bar. No pause/resume (see amendment above).
  4. ~~Each region row has its own DEM toggle, presented as the optional/adds-size choice, independent of the vector download.~~ — superseded: DEM is its own tile with its own download/cancel/delete action, disabled until Vector is downloaded (see amendment above). Deleting Vector still cascades to delete DEM.
  5. The screen shows a total disk usage summary across all downloaded regions.
  6. A region with `updateAvailable` status shows a non-blocking badge with an optional user-triggered "update" action, and continues to appear/behave as downloaded while the badge is shown.

**Plans**: 4 plans

Plans:

- [x] 24-01-PLAN.md — DEM-only delete engine method (D-01) + regionListNotifier snapshot provider + byte-format/disk-usage utilities + English l10n keys (SETUI-01/02/04/05)
- [x] 24-02-PLAN.md — Offline Maps/Regions screen (6-state rows, combined progress, DEM toggle, delete/retry/update, disk-usage summary) + Settings entry + /settings/regions route (SETUI-01..06)
- [x] 24-03-PLAN.md — Gap closure (UAT tests 2/3): implement the promised device-wide fallback in freeDiskSpaceBytes so a never-downloaded region's disk-space check no longer fails closed on a real device (SETUI-03/04)
- [x] 24-04-PLAN.md — Gap closure (UAT re-run tests 1/3): _buildActiveRow renders row status from the live ephemeral downloadState (resolveRowStatus util) instead of the ToOne-stale region.status, so a vector download shows downloading/progress/pause and pause-mid-transfer becomes observable (SETUI-03/04)

**UI hint**: yes

### Phase 25: Map Rendering — Region-Based Viewport Pipeline

**Goal**: Trail detail maps and the navigation screen render offline tiles from the region registry instead of trail-bound caches, with style composition limited to what the current viewport actually needs.
**Depends on**: Phase 24
**Requirements**: RENDER-01, RENDER-02, RENDER-03
**Success Criteria** (what must be TRUE):

  1. A spike against the pinned maplibre 0.3.5 confirms whether incremental style source/layer add/remove is supported (vs. only full style reload) and measures rendering behavior with 10-20 duplicated source/layer sets on a mid-tier Android device, settling which composition strategy the phase ships.
  2. `TrailMap` and `navigation_screen` read offline vector/DEM tile paths via `TileRepositoryManager.localTilePathsForBounds` instead of `Trail.pmTiles`/`demPmTiles`.
  3. Only regions intersecting the current map viewport contribute style sources — panning to an area covered by a different downloaded region swaps sources in rather than accumulating every downloaded region's sources unconditionally.
  4. A downloaded region's basemap (and hillshade, when its DEM was downloaded) renders correctly offline on both the trail detail map and the navigation screen, reusing `offline_style_rewriter.dart` unchanged.

**Plans**: 4 plans (2 waves)
Plans:
**Wave 1**

- [x] 25-01-PLAN.md — RENDER-03 risk-gate spike: standalone on-device harness measuring full-reload vs incremental composition with 10-20 duplicated region source/layer sets; checkpoint:decision settles the strategy (RENDER-03)
- [x] 25-02-PLAN.md — Fix the Critical Data-Shape Gap: split `localTilePathsForBounds` into a typed `({vectorPaths, demPaths})` record via a `@visibleForTesting splitRegionTilePaths` helper + unit test + harness call-site update (RENDER-01)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 25-03-PLAN.md — TrailMap `_composeStyle` sourced from `localTilePathsForBounds(trail.bounds)` (one-time query, both regions when straddling) + regionListNotifier live-swap listen + uncovered-viewport blank basemap (RENDER-01, D-01/D-02/D-05/D-06)
- [x] 25-04-PLAN.md — navigation_screen `_composeStyle` sourced from the live viewport query + camera-idle region-swap recompute (no Timer) + regionListNotifier live-swap listen + uncovered-viewport blank basemap (RENDER-01/RENDER-02, D-01/D-02/D-04/D-06)

**UI hint**: yes

**Risk gate**: RENDER-03's spike is this milestone's highest-risk unknown — research flags maplibre 0.3.5's incremental source/layer API availability and layer-count scaling as LOW/MEDIUM confidence, unconfirmed from docs. Resolve this first, before investing in the rest of the phase's rewiring.

### Phase 25.1: Local HTTP tile proxy for region-based offline map rendering (INSERTED)

**Goal:** Replace `navigation_screen.dart`'s incremental `addSource`/`removeSource` region-swap reconcile (Phase 25's RENDER-02/03 delivery) with a loopback HTTP tile proxy — a static XYZ vector/DEM source served by an in-app `HttpServer` that resolves each tile request against the downloaded region archives via `pmtiles`' per-tile random access — so MapLibre Native's own viewport tracking handles which regions contribute tiles, instead of hand-rolled Dart source/layer diffing.
**Why:** Phase 25's on-device UAT (Test 4) found the incremental reconcile unreliable — root cause is a reentrancy race (`_reconcileRegionComposition` has no in-flight guard, and `ml.MapEventCameraIdle` fires far more often on this screen than assumed because `navigation_screen.dart` continuously drives the camera itself during GPS-follow). Full diagnosis: `.planning/debug/navigation-screen-region-swap-broken.md`. Patching the reconcile in place was considered and rejected in favor of this structural fix, which eliminates the entire bug class (no reconcile call, no race) and also removes the `_sourceFromJson`/`_layerFromJson` duplication between `trail_map.dart` and `navigation_screen.dart`.
**Requirements**: PROXY-01, PROXY-02, PROXY-03
**Depends on:** Phase 25 (specifically 25-02's `TileRepositoryManager.localTilePathsForBounds` split; consumes it as the per-tile archive lookup source)
**Risk gate:** Whether MapLibre Native reliably resolves loopback-HTTP tile sources while the device is offline (airplane mode) is unconfirmed and needs an on-device spike before committing to the full build — mirrors RENDER-03's spike in Phase 25. Regions can also overlap (bbox-only, no dedup today), so a per-tile "which archive wins" resolution rule needs to be designed; none exists yet.
**Plans:** 4/4 plans complete
**Status:** ✅ VERIFIED (2026-07-24) — on-device UAT passed on Android (basemap, hillshade, glyphs, sprite icons render offline via the loopback proxy on both TrailMap and navigation_screen). Required a post-UAT fix (commit 9e164aab): `MapLibre.setConnected(true)` defeats MapLibre Native's airplane-mode connectivity gate, plus a network-free offline style path and the sprite `@2x.json` cache fix. **iOS deferred** — same gate applies, needs a `maplibre_ios` fork (see 25.1 deferred-items.md).

Plans:
**Wave 1**

- [x] 25.1-01-PLAN.md — Android cleartext + iOS ATS loopback network exceptions (proxy/spike prerequisite) (PROXY-03)
- [x] 25.1-02-PLAN.md — Proxy core: xyz_tile_bounds + resolveRegionForTile (smallest-bbox/most-recent, D-02/D-03) + loopback TileProxyServer + base-URL provider + rewriteStyleForProxy (PROXY-01/02)

**Wave 2** *(blocked on Wave 1)*

- [x] 25.1-03-PLAN.md — On-device airplane-mode spike harness + blocking checkpoint:decision risk gate; no pre-scripted fallback per D-04 (PROXY-03) — settled PROCEED on a physical Pixel 6

**Wave 3** *(blocked on the spike passing)*

- [x] 25.1-04-PLAN.md — Start proxy in main.dart + rewire TrailMap & navigation_screen onto the static proxy source, deleting reconcile/tracking machinery + the camera-idle trigger from both (PROXY-01)

### Phase 26: Trail Download Guard

**Goal**: Before a trail downloads, the app makes sure its area is actually covered by a downloaded region, and makes it easy to fix when it isn't.
**Depends on**: Phase 25 (region-based rendering live; guard sends users into a proven download flow)
**Requirements**: GUARD-01, GUARD-02, GUARD-03, GUARD-04
**Success Criteria** (what must be TRUE):

  1. Tapping a trail's download action checks the trail's bbox against downloaded (or updateAvailable) regions before proceeding; when fully covered, the download starts immediately as before.
  2. When coverage is missing, a dialog names the specific missing region(s) and their size, with a direct in-dialog "Download region" action per region — never a silent block or generic message.
  3. A trail spanning multiple regions lists every missing region with individual and combined size, lets the user download any subset, and never forces full coverage before letting the trail download proceed.
  4. A region with `updateAvailable` status satisfies the coverage check the same as `downloaded` — the guard never re-fires for a region that is merely stale.

**Plans**: 5 plans (4 waves)

Plans:
**Wave 1**

- [x] 26-01-PLAN.md — Pure trail-coverage util (bboxesOverlap/overlappingRegions/missingCoverageRegions) + table-driven tests (GUARD-01/04)
- [x] 26-02-PLAN.md — Missing-coverage bottom sheet (Vector/DEM checkboxes, sizes, always-on Download) + MissingCoverageSelection contract + showAggregateProgress notification method (GUARD-02/03)

**Wave 2** *(blocked on Wave 1)*

- [x] 26-03-PLAN.md — Guard wiring in DownloadingTrailIds.download: local coverage check, conditional sheet via navigatorKey, D-04 no-region warning, parallel region+trail downloads, unified aggregate notification (GUARD-01/02/03/04)

**Wave 3** *(gap closure — from 26-VERIFICATION.md / 26-REVIEW.md)*

- [x] 26-04-PLAN.md — Close CR-02 (invalidate regionListNotifierProvider after guard-triggered region downloads) + CR-01 (always-clear trail.id) + WR-01/WR-02 (notification robustness) in trail_download_state_provider.dart (GUARD-01/03/04)

**Wave 4** *(gap closure — from 26-UAT.md)*

- [x] 26-05-PLAN.md — Close 2 UAT gaps: aggregate progress bar resets (monotonic per-package latch in updateAggregate + region-futures-gated id-42 success) in trail_download_state_provider.dart, and DEM-shows-not-downloaded concurrency race (fresh-row read-modify-write for every region-row put) in tile_repository_manager.dart (GUARD-02/03)

### Phase 27: Legacy Cleanup

**Goal**: The old trail-scoped tile system is gone and any files it left behind are cleaned up, so the region system is the only tile path left and its disk-usage figure is trustworthy.
**Depends on**: Phase 26 (region system fully proven end-to-end before removing the fallback)
**Requirements**: CLEAN-01, CLEAN-02
**Success Criteria** (what must be TRUE):

  1. `trail_download_service.dart`'s tile-download methods, `TrailEntity.pmTiles`/`demPmTiles` fields, and any trail-scoped tile-download UI are deleted outright — no dual-run, no migration path; the app builds and runs with zero remaining references.
  2. On first launch after the update, a one-time sweep deletes orphaned legacy tile files from existing dev/test installs, and the Settings disk-usage total reflects only region-based storage afterward. **[DESCOPED for Phase 27 per CONTEXT.md D-05 — pre-production app, no real install base to sweep; CLEAN-02 cut, not deferred. Requirements/roadmap to be reconciled in a future editing pass.]**
  3. A hiker can still download and use a trail fully offline (basemap + navigation) end-to-end purely through the region system, with no functional regression from removing the legacy path.

**Plans**: 2 plans (2 waves)

- [x] 27-01-PLAN.md — Remove trail-tile logic from the guard, notification service, and download service; delete map_cell.dart (Wave 1)
- [ ] 27-02-PLAN.md — Remove pmTiles/demPmTiles fields from TrailEntity/Trail and regenerate generated code (Wave 2)

### Phase 28: Admin Region Picker — Curated Catalog + Hierarchy

**Goal**: A server owner defines downloadable regions by toggling entries in a curated, seeded catalog — picking from real, known-size extracts on a nested tree with a live bbox map — instead of hand-authoring `region_config.json`; the app's settings screen presents the same hierarchy.
**Depends on**: Phase 27 (region system is the only tile path; safe to replace the admin-facing definition mechanism)
**Requirements**: TBD (new REGN-* requirements to be added; un-defers the admin region UI parked as Out of Scope in the v1.6 config-file decision)
**Design**: `.planning/notes/streamlined-region-definition.md`
**Open research**: `.planning/research/questions.md` (region catalog source) — blocks the seed migration
**Success Criteria** (what must be TRUE):

  1. A new seeded `regions` table (nested parent/child, canonical bbox per row, `enabled` flag) exists; the archive-generation cron reads `enabled = true` and no longer parses `region_config.json`.
  2. A custom PocketBase admin page (AlpineJS bundle, reusing the `feature/ap-instance-actors` pattern) lets the admin toggle regions on a collapsible tree while a live map renders the bboxes of enabled regions.
  3. Enabling/disabling a region is the only admin action required — no bbox authoring, no config file edit; the toggle takes effect on the cron's next run.
  4. The Flutter settings screen presents downloadable regions as the same hierarchy (collapsible tree), not a flat list, with no download-UX regression.

**Plans**: TBD

## Progress

**Execution Order:**
Phases 13 and 14 are independent and may execute in either order or in parallel; 15-27 are strictly sequential:

```
13 ─┐
    ├─→ 15 → 16 → 17 → 18 → 19 → 20 → 21 → 22 → 23 → 24 → 25 → 26 → 27
14 ─┘
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
| 27. Legacy Cleanup | v1.6 | 1/2 | In Progress|  |
