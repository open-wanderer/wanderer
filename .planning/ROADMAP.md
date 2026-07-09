# Roadmap: Wanderer Trail Navigation

## Milestones

- ✅ **v1.0 MVP** — Phases 1-3 (shipped 2026-06-13)
- ✅ **v1.1 Offline** — Phases 4-5 (shipped 2026-06-14)
- ✅ **v1.2 Settings Screens** — Phases 6-9 (shipped 2026-06-29)
- ✅ **v1.3 Category Redesign** — Phases 10-12 (shipped 2026-07-02)
- 🚧 **v1.4 MapLibre Migration** — Phases 13-18 (in progress)

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

### 🚧 v1.4 MapLibre Migration (In Progress)

**Milestone Goal:** Replace the Flutter app's `flutter_map` + forked `vector_map_tiles`/`vector_tile_renderer` stack with the `maplibre` package, retiring both `flomp/*` forks and moving map rendering onto native GL — without ever leaving the app unbuildable and without regressing offline trail rendering.

- [x] **Phase 13: Glyph & Sprite Endpoint** - A unified `/map/style-sources` SvelteKit endpoint (replacing `/map/tileurl`) resolves tile, glyph, and sprite URLs in one object, under operator override (completed 2026-07-08)
- [x] **Phase 14: Coordinate Type Migration** - `latlong2.LatLng` → `Geographic`, `LatLngBounds` → `LngLatBounds`, test-guarded, before any map code moves (completed 2026-07-08)
- [ ] **Phase 15: MapLibre Core, Trail Rendering & Offline Parity** - `WandererMap` on `MapLibreMap`; a downloaded trail renders basemap *and labels* in airplane mode
- [ ] **Phase 16: List & Map Screens on MapLibre** - Multi-trail list maps plus server-clustered map-screen search on native circle/symbol layers
- [ ] **Phase 17: Navigation on MapLibre** - Heading-up follow, compass reset, location puck; the last `flutter_map` plugin call sites disappear
- [ ] **Phase 18: Retire flutter_map and the flomp Forks** - Both forks and all five packages leave `pubspec.yaml`; `maplibre` pinned

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

## Phase Details

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
  5. `lib/vendor/vector_map_tiles/pm_tile_provider.dart` is deleted, and the app still builds and runs with `flutter_map` serving `list_detail_map_screen`, `list_detail_screen`, `map_screen`, and `navigation_screen`.

**Plans**: 6 plans

Plans:

- [ ] 15-01-PLAN.md — Throwaway `file://` glyph+sprite resolution spike (risk gate, physical-device verify)
- [ ] 15-02-PLAN.md — Style extraction to `.json` assets + `mapStyleJsonProvider` (STYLE-01..04)
- [ ] 15-03-PLAN.md — App-wide glyph/sprite cache + path-safety + download trigger (GLYPH-04, OFFL-01)
- [ ] 15-04-PLAN.md — `WandererMap` on `MapLibreMap`: camera, live theme swap, chrome, markers (CORE-01..04, TRAIL-05)
- [ ] 15-05-PLAN.md — Trail track/casing, static arrows, waypoint/pin markers (TRAIL-01..04)
- [ ] 15-06-PLAN.md — Offline rewrite (`pmtiles://file://` + `file://`), multi-cell decision, delete vendor provider (OFFL-02..06)
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

**Plans**: TBD
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

**Plans**: TBD
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

**Plans**: TBD

## Progress

**Execution Order:**
Phases 13 and 14 are independent and may execute in either order or in parallel. Everything after is strictly sequential:

```
13 ─┐
    ├─→ 15 → 16 → 17 → 18
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
| 15. MapLibre Core, Trail Rendering & Offline Parity | v1.4 | 0/TBD | Not started | - |
| 16. List & Map Screens on MapLibre | v1.4 | 0/TBD | Not started | - |
| 17. Navigation on MapLibre | v1.4 | 0/TBD | Not started | - |
| 18. Retire flutter_map and the flomp Forks | v1.4 | 0/TBD | Not started | - |
