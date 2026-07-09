# Requirements: Wanderer Trail Navigation

**Defined:** 2026-07-08
**Core Value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.

## v1.4 Requirements

Requirements for milestone v1.4 — MapLibre Migration. Each maps to exactly one roadmap phase.

### Style & Glyph Serving

- [x] **STYLE-01**: The wanderer light and dark styles exist as plain `.json` assets in the app, equivalent in rendered output to today's `wandererLightTheme` / `wandererDarkTheme`
- [x] **STYLE-02**: The app injects the operator's `TILE_SERVER_URL` (via the unified `/map/style-sources` endpoint) into the style's `protomaps` source at runtime, preserving today's operator control
- [x] **STYLE-03**: The style carries a `glyphs` key, and its 14 symbol layers render place-name labels in the 4 referenced fontstacks (Noto Sans Regular, Medium, Italic, and Devanagari Regular for the data-driven Devanagari-script case)
- [x] **STYLE-04**: The style carries a `sprite` key, and the `arrow` and route-network shield icons render — icons silently dropped today
- [x] **GLYPH-01**: The unified `/map/style-sources` endpoint resolves the glyph URL template (`{fontstack}/{range}.pbf`) for the 4 fontstacks the style references, defaulting to Protomaps' public `basemaps-assets` host
- [x] **GLYPH-02**: The unified `/map/style-sources` endpoint resolves the sprite base URL (`sprite.json`, `sprite.png`, `sprite@2x.png`, light and dark), defaulting to Protomaps' public `basemaps-assets` host
- [x] **GLYPH-03**: An operator can override the glyph and sprite origin by environment variable, mirroring how `TILE_SERVER_URL` overrides tiles; unset, `/map/style-sources` falls back to Protomaps' public assets rather than a self-hosted copy
- [x] **GLYPH-04**: The app resolves glyph and sprite URLs from the server on first use and caches them app-wide, not per-trail

### Offline

- [x] **OFFL-01**: Downloading a trail also fetches the glyph ranges and sprite sheet once into the app documents directory; a second trail download reuses the cache without re-fetching
- [x] **OFFL-02**: When rendering a downloaded trail, the app rewrites the style's `glyphs` and `sprite` keys to `file://` paths before handing the style to the map
- [x] **OFFL-03**: A downloaded trail renders its basemap from `.pmtiles` archives via native `pmtiles://` source URLs, with no network
- [x] **OFFL-04**: A downloaded trail renders place-name labels with no network — the offline parity gate
- [x] **OFFL-05**: A trail whose offline tiles span multiple `.pmtiles` cells renders every cell, replacing `MultiPmTilesVectorTileProvider`'s request fan-out
- [ ] **OFFL-06**: `lib/vendor/vector_map_tiles/pm_tile_provider.dart` is deleted

### Map Core

- [x] **CORE-01**: `WandererMap` renders via `MapLibreMap`, accepting the same widget contract (trail, controls, disabled, offline, callbacks) its call sites use today
- [x] **CORE-02**: The map honors the app's light/dark theme, switching styles live when the user changes theme
- [x] **CORE-03**: The map's initial camera fits the trail bounds with the caller's padding, matching today's `CameraFit.bounds` behavior
- [x] **CORE-04**: A user sees a scale bar and the Protomaps/OpenStreetMap source attribution on every map — an ODbL obligation the app does not meet today
- [ ] **CORE-05**: The compass uses maplibre's built-in `MapCompass`; `lib/components/map/map_compass.dart` is deleted
- [ ] **CORE-06**: Camera animations use maplibre's native `animateCamera` / `fitBounds`; `flutter_map_animations` and every `AnimatedMapController` reference are gone
- [ ] **CORE-07**: The user's location puck and heading-up follow use maplibre's `enableLocation` / `trackLocation`; `flutter_map_location_marker` is gone
- [x] **CORE-08**: `list_detail_map_screen` and `list_detail_screen` render their multi-trail polylines via `MapLibreMap`, with the camera animating to fit every trail in the list

### Trail Rendering

- [x] **TRAIL-01**: A trail's GPX track renders as a native line layer with a casing — 5px route color over a 2px white border, visually equivalent to today's `Polyline`
- [x] **TRAIL-02**: Directional arrows render along the trail line via a native symbol layer, replacing the ~60 lines of bearing math currently dead behind `showArrows = false`
- [x] **TRAIL-03**: Waypoints render as tappable widget markers, preserving today's `AnimatedScale` selection animation and `onWaypointTap` callback
- [x] **TRAIL-04**: Start and finish pins render as widget markers, including the alignment nudge applied when the two points fall within 36 screen pixels
- [x] **TRAIL-05**: The elevation-profile position marker tracks the user's scrub position along the trail

### Clustering & Map Screen

- [x] **CLUS-01**: The map screen's bbox search calls `POST /search/trails/cluster` and renders the returned FeatureCollection
- [x] **CLUS-02**: Clusters render as native circle layers sized by `point_count`, labelled from `point_count_abbreviated`, matching web's `ClusterLayer` step ramp
- [ ] **CLUS-03**: Tapping a cluster zooms the camera toward it; tapping an unclustered point selects that trail and fits its polyline
- [x] **CLUS-04**: Panning or zooming re-queries the cluster endpoint at the new bounds and zoom, debounced as today
- [x] **CLUS-05**: Active category and subcategory filters continue to constrain map results (the endpoint applies preference filters server-side)

### Navigation Screen

- [ ] **NAV-01**: Navigation renders the route line, the user's position, and heading-up follow on maplibre
- [ ] **NAV-02**: Dragging the map during navigation breaks follow mode, and the recenter control restores it — matching today's `MapEventMoveStart` / `dragStart` behavior
- [ ] **NAV-03**: The compass control resets bearing to north with an animated camera transition
- [ ] **NAV-04**: Offline navigation continues to serve maneuvers from the ObjectBox cache (v1.1 behavior, unregressed)

### Type Migration & Cleanup

- [x] **TYPE-01**: `latlong2.LatLng` is replaced by `Geographic` across `trail.dart`, `gpx_util.dart`, `polyline_util.dart`, and `foreground_position_stream_provider.dart`, with GPX parsing and polyline decoding verified against existing tests
- [x] **TYPE-02**: `LatLngBounds` is replaced by `LngLatBounds` at every bounds call site
- [ ] **CLEAN-01**: `flutter_map`, `flutter_map_animations`, `flutter_map_location_marker`, and `flutter_map_marker_cluster` are removed from `pubspec.yaml`
- [ ] **CLEAN-02**: `vector_map_tiles` and `vector_tile_renderer` are removed from `pubspec.yaml`, and `dependency_overrides` no longer references either `flomp/*` fork
- [ ] **CLEAN-03**: `maplibre` is pinned to an exact version rather than a caret range, given its pre-1.0 breaking-change cadence

## Future Requirements

Acknowledged but deferred. Not in this roadmap.

### Map Screen Parity

- **FUT-01**: Render `is_large` trails as full polylines on the map screen — the cluster endpoint already emits the flag and web consumes it; the app ignores it

### Basemap Choice

- **FUT-02**: Basemap picker in app settings (OpenTopoMap, OpenHikingMap, CyclOSM, Carto Light/Dark) matching web's `baseMapStyles`
- **FUT-03**: Waymarked Trails hiking/cycling raster overlays, matching web's `overlays`

### Native GL Capabilities

- **FUT-04**: 3D terrain and hillshade layers
- **FUT-05**: Map pitch/tilt gestures during navigation

## Out of Scope

| Feature | Reason |
|---------|--------|
| Web frontend changes | `web/` already runs maplibre-gl-js; v1.4 is app-only apart from the new glyph/sprite endpoint |
| Switching offline tile generation to OpenMapTiles schema | Would invalidate every downloaded trail archive and every operator's tile cache |
| Adopting web's `ofm.json` as the app style | OpenMapTiles schema cannot render the Protomaps-schema `.pmtiles` cells the server generates |
| Upstream PR adding `cluster` fields to maplibre's `GeoJsonSource` | Unnecessary once clustering is server-side |
| Client-side clustering (supercluster on device) | Re-clusters on the UI isolate — the exact cost the migration exists to shed |
| Native symbol layers for waypoint markers | Loses per-marker tap handling and selection animation, for scale the app does not need |
| Linux / Windows / macOS map support | `maplibre` has no Linux backend; the app is mobile-only |
| Big-bang cutover | `LatLng`→`Geographic` touches GPX parsing; a bad landing risks trail data, not just maps |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| GLYPH-01 | Phase 13 | Complete |
| GLYPH-02 | Phase 13 | Complete |
| GLYPH-03 | Phase 13 | Complete |
| TYPE-01 | Phase 14 | Complete |
| TYPE-02 | Phase 14 | Complete |
| STYLE-01 | Phase 15 | Complete |
| STYLE-02 | Phase 15 | Complete |
| STYLE-03 | Phase 15 | Complete |
| STYLE-04 | Phase 15 | Complete |
| GLYPH-04 | Phase 15 | Complete |
| CORE-01 | Phase 15 | Complete |
| CORE-02 | Phase 15 | Complete |
| CORE-03 | Phase 15 | Complete |
| CORE-04 | Phase 15 | Complete |
| TRAIL-01 | Phase 15 | Complete |
| TRAIL-02 | Phase 15 | Complete |
| TRAIL-03 | Phase 15 | Complete |
| TRAIL-04 | Phase 15 | Complete |
| TRAIL-05 | Phase 15 | Complete |
| OFFL-01 | Phase 15 | Complete |
| OFFL-02 | Phase 15 | Complete |
| OFFL-03 | Phase 15 | Complete |
| OFFL-04 | Phase 15 | Complete |
| OFFL-05 | Phase 15 | Complete |
| OFFL-06 | Phase 15 | Deferred to Phase 17/18 |
| CORE-08 | Phase 16 | Complete |
| CLUS-01 | Phase 16 | Complete |
| CLUS-02 | Phase 16 | Complete |
| CLUS-03 | Phase 16 | Pending |
| CLUS-04 | Phase 16 | Complete |
| CLUS-05 | Phase 16 | Complete |
| NAV-01 | Phase 17 | Pending |
| NAV-02 | Phase 17 | Pending |
| NAV-03 | Phase 17 | Pending |
| NAV-04 | Phase 17 | Pending |
| CORE-05 | Phase 17 | Pending |
| CORE-06 | Phase 17 | Pending |
| CORE-07 | Phase 17 | Pending |
| CLEAN-01 | Phase 18 | Pending |
| CLEAN-02 | Phase 18 | Pending |
| CLEAN-03 | Phase 18 | Pending |

**Coverage:**

- v1.4 requirements: 41 total
- Mapped to phases: 41 ✓
- Unmapped: 0

**Per-phase counts:** Phase 13 → 3 · Phase 14 → 2 · Phase 15 → 20 · Phase 16 → 6 · Phase 17 → 7 · Phase 18 → 3

**Corrections made during roadmap creation:**

- The pre-roadmap count of "33 total" was wrong — the document listed 40 v1.4 requirements (STYLE 4, GLYPH 4, OFFL 6, CORE 7, TRAIL 5, CLUS 5, NAV 4, TYPE 2, CLEAN 3). Corrected.
- **CORE-08 added.** `list_detail_map_screen.dart` (244 lines) and `list_detail_screen.dart` (429 lines) build `FlutterMap` directly rather than through `WandererMap`, so CORE-01 did not cover them and no other requirement did. Migrating them was implied only transitively, by CLEAN-01's removal of `flutter_map` from `pubspec.yaml` — a terminal-phase requirement. Without CORE-08, Phase 16's list-map success criterion had no supporting requirement. Count is therefore 41, not 40.

---
*Requirements defined: 2026-07-08*
*Last updated: 2026-07-08 — traceability mapped to Phases 13–18; count corrected 33 → 41; CORE-08 added*
