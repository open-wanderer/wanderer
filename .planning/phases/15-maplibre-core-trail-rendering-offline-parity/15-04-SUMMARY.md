---
phase: 15-maplibre-core-trail-rendering-offline-parity
plan: 04
subsystem: ui
tags: [flutter, maplibre, riverpod, map-core, camera, theme-swap, attribution, markers]

# Dependency graph
requires:
  - phase: 15-02
    provides: "mapStyleJsonProvider — keepAlive Future<String> raw style JSON, re-runs on theme change (feeds initStyle + setStyle)"
  - phase: 15-03
    provides: "glyphSpriteCacheProvider — app-wide glyph/sprite cache warmer (D-09 map-open trigger wired here)"
provides:
  - "WandererMap on MapLibreMap — native GL host: style injection, live theme swap (setStyle), camera fit (fitBounds), scalebar + ODbL attribution, elevation + interim-location WidgetLayer markers, fixed onEvent click routing"
  - "ml.MapController acquisition pattern: WandererMap.onMapCreated hands the native controller to callers (nullable field), replacing the old free-standing flutter_map MapController"
  - "15-05 seam: empty layers: [] with a documented comment where the trail track + arrows + waypoint/pin markers attach"
affects: [15-05, 15-06, CORE-01, CORE-02, CORE-03, CORE-04, TRAIL-05]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Live style swap without remount: ref.listen(mapStyleJsonProvider) -> controller.setStyle(json) on change; initStyle used only at creation; cache last JSON so a provider refresh never drops to the loading state (no flash)"
    - "Native controller hand-off: MapLibreMap.onMapCreated captures ml.MapController internally (for setStyle/fitBounds) AND forwards it to the caller via WandererMap.onMapCreated"
    - "Camera fit in onStyleLoaded: fitBounds(nativeDuration: Duration.zero) for an instant initial fit, moveCamera fallback for degenerate (zero-extent) bounds"

key-files:
  created: []
  modified:
    - app/lib/components/base/wanderer_map.dart
    - app/lib/routes/trail_detail_map_screen.dart
    - app/lib/components/trail/trail_panel.dart

key-decisions:
  - "Replaced the WandererMap `mapController` INPUT field with an `onMapCreated` callback: ml.MapController is an abstract interface created by the native map and cannot be free-standing, so callers now hold it as a nullable field set from the hand-off"
  - "Live theme swap uses ref.listen + setStyle plus a cached _lastStyleJson, so a keepAlive provider refresh on theme toggle never rebuilds into the loading branch (CORE-02 no-flash)"
  - "Bounds-present test is a non-degenerate check (N!=S || E!=W) because Trail.bounds is a non-null getter that returns a zero box when the trail has no geo extent; degenerate -> moveCamera fallback"
  - "Interim location marker is a simple static blue puck via foregroundPositionStreamProvider + StreamBuilder — explicitly NOT the Phase-17 native follow/heading puck (A5/CORE-07)"

patterns-established:
  - "WandererMap.onMapCreated is the sanctioned way for a screen to obtain the ml.MapController for its own camera calls (reused by 15-05/15-06)"

requirements-completed: [CORE-01, CORE-02, CORE-03, CORE-04, TRAIL-05]

# Metrics
duration: ~20min
completed: 2026-07-09
---

# Phase 15 Plan 04: WandererMap on MapLibreMap Summary

**Rewrote `WandererMap` from `FlutterMap` to native `MapLibreMap` — Protomaps basemap from `mapStyleJsonProvider`, live light/dark `setStyle` swap with no flash, `fitBounds` camera in `onStyleLoaded`, built-in scale bar + first-ever ODbL attribution, and elevation/interim-location `WidgetLayer` markers — then migrated both consumers to the `ml.MapController` hand-off contract while leaving the 4 flutter_map screens untouched.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-07-09
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- **WandererMap is now a `MapLibreMap` host (CORE-01):** constructor param names/defaults preserved verbatim except `mapController` (input field) → `onMapCreated` (controller hand-off). Loading (`ColoredBox` surface) / error (`Center(Text)`) passthrough ported verbatim per the UI-SPEC Copywriting Contract.
- **Live theme swap (CORE-02):** `ref.listen(mapStyleJsonProvider)` calls `controller.setStyle(json)` in place — no `ObjectKey` remount. A cached `_lastStyleJson` keeps the map mounted through the keepAlive provider's refresh so the toggle never flashes back to loading.
- **Camera fit (CORE-03):** `onStyleLoaded` → `fitBounds(bounds, padding, nativeDuration: Duration.zero)` for an instant initial fit, with a `moveCamera(center, zoom: 18)` fallback for zero-extent bounds.
- **Scale bar + attribution (CORE-04):** `const MapScalebar()` (bottom-left) and `const SourceAttribution()` (bottom-right) added to `children` — the app's first ODbL attribution surface.
- **Markers:** elevation scrub marker (12px white/black dot, TRAIL-05) and the interim location puck (A5) render via `ml.WidgetLayer`/`ml.Marker`.
- **Bug fixed:** the old `onMapEvent: (e) => widget.onMapEvent` (returned the field, never invoked) is replaced by an `onEvent` that calls `widget.onMapEvent` and routes `MapEventClick` to `widget.onTap`.
- **D-09 glyph cache warm:** `ref.read(glyphSpriteCacheProvider.future)` fires once on first map open, idempotent against the 15-03 download trigger.
- **Deletions:** `MultiPmTilesVectorTileProvider` / `_initOffline` / `_buildTileLayer` and the `flutter_map` / `vector_map_tiles` / `pm_tile_provider` / `map_coordinate_adapter` / `trail_layer` imports are gone from `wanderer_map.dart` (offline path re-added declaratively in 15-06; trail track in 15-05).
- **Consumers migrated:** `trail_detail_map_screen` holds `ml.MapController? _mapController` set via `onMapCreated`, and its expand button uses `_mapController?.fitBounds(bounds, padding)` (dropped `toLatLngBounds`/`CameraFit`/`fitCamera` and the `flutter_map`+`map_coordinate_adapter` imports). `trail_panel`'s `onTap` went from 2-arg to 1-arg; `disabled: true` routes to `MapGestures.none()`.

## ml.MapController acquisition approach (for 15-05 / 15-06)

`ml.MapController` is an abstract interface instantiated by the native map, so it **cannot** be constructed free-standing the way `flutter_map`'s `MapController()` was. The sanctioned pattern established here:

- `WandererMap` exposes `final void Function(ml.MapController controller)? onMapCreated;`.
- Inside `WandererMap`, `MapLibreMap.onMapCreated` both captures the controller in `_controller` (for internal `setStyle`/`fitBounds`) **and** forwards it via `widget.onMapCreated?.call(controller)`.
- A consumer that needs the controller (e.g. `trail_detail_map_screen` for its expand-to-bounds button) declares `ml.MapController? _mapController;` and sets it from `onMapCreated: (c) => _mapController = c`, then null-guards its calls (`_mapController?.fitBounds(...)`).

15-05/15-06 should reuse this hand-off rather than reintroducing an input controller field.

## 15-05 seam location

In `wanderer_map.dart`, `_buildMap`'s `MapLibreMap` call has an empty `layers: const []` carrying the comment `// 15-05: trail track + arrows + waypoint/start-finish markers wired here`. 15-05 attaches the `GeoJsonSource` + `LineStyleLayer`s + `SymbolStyleLayer` (arrows) there, and adds the waypoint/pin `WidgetLayer`s to `children` (the `showTrail` / `selectedWaypoint` / `onWaypointTap` fields are already preserved on the widget for that wiring).

## Task Commits

1. **Task 1: Rewrite WandererMap onto MapLibreMap** — `1fd21cf9` (feat)
2. **Task 2: Update the two WandererMap consumers** — `4cfc7dbc` (feat)

## Files Modified

- `app/lib/components/base/wanderer_map.dart` — full rewrite onto `MapLibreMap`.
- `app/lib/routes/trail_detail_map_screen.dart` — `ml.MapController?` field + `onMapCreated` hand-off + `fitBounds`.
- `app/lib/components/trail/trail_panel.dart` — `onTap` 1-arg signature.

## Decisions Made

See frontmatter `key-decisions`. The load-bearing one: **`onMapCreated` replaces the `mapController` input** because the native controller cannot be pre-constructed by the caller.

## Deviations from Plan

### Auto-fixed / Interpreted

**1. [Rule 3 - Blocking] `mapController` input field could not be preserved as an input**
- **Found during:** Task 1 / Task 2
- **Issue:** The plan's Component-Responsibilities table said "retype `mapController` to `ml.MapController`", but `ml.MapController` is an abstract interface created by the native map — a caller cannot construct one to pass in, unlike the old `flutter_map` `MapController()`.
- **Fix:** Replaced the input field with an `onMapCreated` callback (the plan's Task 2 explicitly anticipated this: "if it must be created by the map, hold it as a nullable field set from the map"). Both the internal capture and the caller hand-off go through it.
- **Files modified:** wanderer_map.dart, trail_detail_map_screen.dart
- **Committed in:** `1fd21cf9`, `4cfc7dbc`

**Total deviations:** 1 (blocking, anticipated by the plan). No scope creep.

## Known Stubs

- **`layers: const []` in `wanderer_map.dart`** — the trail track / arrows / waypoint / start-finish layers are intentionally deferred to **15-05** (this plan is the shell only, per its objective: "This plan does NOT yet draw the trail track/arrows/waypoints/pins"). The `showTrail`, `selectedWaypoint`, and `onWaypointTap` fields are preserved on the widget so 15-05 wires them without another contract change. This is a planned seam, not an unresolved stub.

## Verification

- `flutter analyze` reports **zero issues in all three modified files** and **zero errors** app-wide.
- Whole-app analyze shows 39 pre-existing warnings/infos in unrelated files (deprecated icon_util members, unused imports in list_detail/map screens, the `trail_dropdown.dart:126` dead_code already logged to deferred-items in 15-03) — all out of scope (SCOPE BOUNDARY), none in this plan's files.
- The 4 not-yet-migrated flutter_map screens (`list_detail_map_screen`, `list_detail_screen`, `map_screen`, `navigation_screen`) show no git changes (phase success criterion #5).
- Verify greps pass: `MapLibreMap`, `fitBounds`, `SourceAttribution`, `setStyle` present in `wanderer_map.dart`; `MultiPmTilesVectorTileProvider` absent; `ml.MapController` + `fitBounds` present and `toLatLngBounds` absent in `trail_detail_map_screen.dart`.

## Next Phase Readiness

- **15-05 (trail rendering)** attaches track/arrow/waypoint layers at the documented `layers:` seam and reuses the `onMapCreated` controller hand-off; the preserved `showTrail`/`selectedWaypoint`/`onWaypointTap` fields are ready.
- **15-06 (offline)** re-introduces the `pmtiles://file://` + `file://` glyph/sprite rewrite; the `offline` field is preserved on `WandererMap` for that branch. Note the 15-03 finding that `file://` sprite resolution failed on device — 15-06 still owns that.
- No blockers introduced.

## Self-Check: PASSED

- All 3 modified files exist on disk with the described changes.
- Both task commits (`1fd21cf9`, `4cfc7dbc`) present in git history.

---
*Phase: 15-maplibre-core-trail-rendering-offline-parity*
*Completed: 2026-07-09*
