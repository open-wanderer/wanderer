# Phase 17: Navigation on MapLibre - Context

**Gathered:** 2026-07-09
**Status:** Ready for planning

<domain>
## Phase Boundary

`navigation_screen.dart` moves off `flutter_map` onto `MapLibreMap` — the last screen still building `FlutterMap` directly. Turn-by-turn navigation keeps its route line, breadcrumb, waypoints, maneuver banner, and stats sheet, but the location puck, heading-up follow, camera animations, and compass now come from maplibre's native APIs (`enableLocation`/`trackLocation`, `animateCamera`/`fitBounds`, `ml.MapCompass`) instead of `flutter_map_location_marker`, `flutter_map_animations`, and the app-local `map_compass.dart`. `MultiPmTilesVectorTileProvider`/`pm_tile_provider.dart` (OFFL-06, deferred from Phase 15) is also retired here since this is its only remaining caller.

Out of scope: deleting `flutter_map`/plugins from `pubspec.yaml` (CLEAN-01/02/03 — Phase 18); any change to maneuver logic, Valhalla integration, offline ObjectBox caching behavior, or the stats sheet's content (all v1.0/v1.1 behavior, unregressed per ROADMAP success criterion 3).

</domain>

<decisions>
## Implementation Decisions

### Compass button behavior (NAV-03)
- **D-01:** Keep today's toggle behavior, not maplibre's native reset-only default. The compass button continues to toggle between north-up and heading-up map rotation (tap once to start following GPS heading, tap again to snap back to north) — this is the validated "north-up/heading-up map orientation toggle" requirement from PROJECT.md and must not regress. Implement via `ml.MapCompass`'s `onPressed` override (it explicitly supports overriding the default tap behavior) calling `controller.trackLocation(trackLocation: true, trackBearing: _headingUp ? BearingTrackMode.gps : BearingTrackMode.none)`, plus an explicit `animateCamera(bearing: 0, ...)` when turning heading-up off (native `MapCompass`'s own default reset-to-north only fires on tap when `onPressed` is unset, so the explicit reset call must be preserved in the override).
- **D-02:** Compass stays always visible during navigation (`hideIfRotatedNorth: false`) — contrast with Phase 16's map screen which used `hideIfRotatedNorth: true`. Rationale: during turn-by-turn the compass is a primary orientation control the hiker actively watches/uses, not an occasional map-browsing aid (map screen's rationale for hiding it doesn't apply here).

### Recenter + follow-mode state coupling (NAV-02)
- **D-03:** Recenter restores the exact prior state, including heading-up if it was active before the drag broke follow — `trackLocation(trackLocation: true, trackBearing: previousBearingMode)`, mirroring today's independent `_followEnabled`/`_headingUp` booleans (dragging away doesn't reset `_headingUp`; recenter re-engages follow using whatever `_headingUp` was already set to). Do not simplify to "recenter always resets to north" — this would be a real behavior regression on repeated pan-then-recenter cycles during heading-up navigation.
- Drag-only breaks follow (not pinch/zoom) — already locked by ROADMAP.md's success criterion 2 ("matching today's `MapEventMoveStart`/`dragStart` behavior"), not re-litigated here. Maps to maplibre's `MapEventStartMoveCamera` with the appropriate `CameraChangeReason` (confirm exact reason value during research/planning — Phase 16's `map_screen.dart` already distinguishes `CameraChangeReason.apiGesture` for its own drag-detection use, establishing the pattern to follow).

### Location puck visuals (CORE-07)
- **D-04:** Accept maplibre's native `enableLocation()` puck with its default visual options (`pulseFade: true`, `accuracyAnimation: true`, `pulse: true`, `compassAnimation: true`) — do not attempt to visually match today's custom 18px blue dot (`WandererMap`'s puck, reused by `map_screen.dart`). CORE-07 requires the puck itself to come from `enableLocation`/`trackLocation`, which has no custom color/size API — the native puck's appearance (GPS accuracy pulse/circle) is a deliberate, accepted visual change, not a regression to fix.
- `bearingRenderMode`/`trackBearing` values: use `BearingRenderMode.gps` / `BearingTrackMode.gps` (GPS-derived heading), matching today's `LocationMarkerDataStreamFactory.fromGeolocatorPositionStream` — no compass-sensor mode discussed or needed, GPS heading is what the app already uses via `TraceletPositionSource`.

### Claude's Discretion
- Exact camera animation durations for recenter/heading-up transitions (short, native, non-`Duration.zero` per the Phase 16 checkpoint lesson — `Duration.zero` crashes the Android native binding's `animateCamera`/`fitBounds`; use a short non-zero duration like Phase 16's established 400ms–750ms range).
- Whether the legacy flutter_map `TrailLayer` widget (restored in `trail_layer.dart` during Phase 15's checkpoint specifically for this screen, doc-commented "Phase-17 deletion") gets physically deleted in this phase or left dead-but-unused until Phase 18's cleanup pass — functionally must stop being *called* by `navigation_screen.dart` (switch to the native `addTrailTrackLayers`/`TrailMarkerLayer` functions already used by `WandererMap`), deletion timing is not user-facing.
- Exact widget/provider structure for wiring `enableLocation`/`trackLocation` into a `ConsumerStatefulWidget` lifecycle (mirrors `WandererMap`'s established `onMapCreated` hand-off pattern from Phase 15 — see canonical refs).
- Offline vector tile source wiring for the navigation screen's `MapLibreMap` (replacing `VectorTileLayer`/`MultiPmTilesVectorTileProvider` with the `pmtiles://file://` + `rewriteStyleForOffline` pattern `WandererMap` already established in Phase 15) — mechanical port, no user decision needed.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap & requirements
- `.planning/ROADMAP.md` §"Phase 17: Navigation on MapLibre" — goal, 4 success criteria, requirements NAV-01..04/CORE-05/06/07
- `.planning/REQUIREMENTS.md` §"Navigation Screen", §"Map Core" — NAV-01..04, CORE-05/06/07 full text
- `.planning/PROJECT.md` — validated requirement "North-up / heading-up map orientation toggle button (v1.0 — Phase 2)" that D-01/D-02 must not regress

### Prior phase work this phase builds on
- `.planning/phases/15-maplibre-core-trail-rendering-offline-parity/15-06-SUMMARY.md` and related Phase 15 SUMMARYs — `WandererMap`'s `onMapCreated`/`onStyleLoaded` hand-off pattern, `rewriteStyleForOffline` (OFFL-02/03/05), the location `WidgetLayer` puck this phase explicitly does NOT reuse (D-04), and the restored legacy `TrailLayer` widget in `trail_layer.dart` doc-commented for Phase-17 deletion
- `.planning/phases/16-list-map-screens-on-maplibre/16-03-SUMMARY.md` — `SearchMap`'s controller/style-loaded race fix (buffer `onStyleLoaded` until `onMapCreated` fires — same class of bug likely to recur here if navigation_screen builds its own host rather than reusing `SearchMap`/`WandererMap`), the `Duration.zero` Android crash lesson (never use `Duration.zero` for "instant" `fitBounds`/`animateCamera` — use ~1ms or a short explicit duration), `ml.MapCompass` usage precedent (`hideIfRotatedNorth: true` there vs. `false` here per D-02)
- `app/lib/components/map/trail_layer.dart` — already contains both the native `addTrailTrackLayers`/`TrailMarkerLayer` functions (used by `WandererMap`) AND the restored legacy flutter_map `TrailLayer` widget (navigation_screen's current import, to be dropped)

### Source of truth for what's being replaced
- `app/lib/routes/navigation_screen.dart` (835 lines) — current `flutter_map`-based implementation: `AnimatedMapController`, `CurrentLocationLayer`/`LocationMarkerDataStreamFactory`, app-local `MapCompass` (`map_compass.dart`), `MultiPmTilesVectorTileProvider` offline branch, `_followEnabled`/`_headingUp` state, `_recenterTrigger` stream, drag-only follow-break via `MapEventMoveStart`/`MapEventSource.dragStart`
- `app/lib/components/map/map_compass.dart` (138 lines) — the app-local flutter_map-only compass widget CORE-05 deletes; its "N + carets" custom icon is NOT preserved (native `ml.MapCompass`'s own icon is used instead — visual change not discussed/flagged as a concern)
- `app/lib/vendor/vector_map_tiles/pm_tile_provider.dart` — `MultiPmTilesVectorTileProvider`, OFFL-06's deferred deletion target, now unblocked since this is its last caller

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `app/lib/util/map_coordinate_adapter.dart` (Phase 14) — `toLatLng`/`toLatLngList` etc. still imported by `navigation_screen.dart` today; every usage should be deleted as this screen migrates (it's the last `flutter_map`-side consumer per Phase 15's CONTEXT.md note).
- `app/lib/components/base/wanderer_map.dart` — not directly reusable (navigation needs its own host: breadcrumb polyline, maneuver banner, waypoint sheet, offline indicator are all nav-specific), but its `onMapCreated`/`onStyleLoaded`/live theme-swap pattern is the template to replicate for a nav-specific host, same as `SearchMap` did for the list/cluster screens in Phase 16.
- `app/lib/util/tracelet_position_source.dart` — `TraceletPositionSource`/background GPS stream stays unchanged; only the consumption point (feeding `enableLocation`/`trackLocation` instead of `CurrentLocationLayer`) changes.

### Established Patterns
- Phase 15/16 both used a dedicated lightweight host widget per screen family (`WandererMap` for single-trail, `SearchMap` for multi-trail/search) rather than one shared god-widget — navigation's host will likely be a third, nav-specific host or a `WandererMap` extension point; left to planning/research to decide given navigation's extra state (breadcrumb, maneuver banner, waypoint sheet) doesn't fit `SearchMap`'s "trail-agnostic" contract.
- `SearchMap`'s controller/style-loaded race (Phase 16-03 checkpoint fix) — any new host built for this phase must apply the same buffering fix from day one rather than rediscovering it on a physical device.

### Integration Points
- `navigation_provider.dart`/`navigation_stats_provider.dart` — maneuver progress and stats tracking are untouched; only the map rendering layer changes.
- `local_settings_provider.dart`'s `unitProvider` — untouched, stats sheet formatting unaffected.

</code_context>

<specifics>
## Specific Ideas

None beyond the decisions above.

</specifics>

<deferred>
## Deferred Ideas

- Compass icon redesign to match `ml.MapCompass`'s native look-and-feel more closely, or vice versa — not raised as a concern; native default accepted implicitly by not being discussed further.
- Physically deleting the legacy flutter_map `TrailLayer` widget and `pm_tile_provider.dart` this phase vs. leaving them dead until Phase 18 — left to Claude's discretion (not user-facing either way).

</deferred>

---

*Phase: 17-navigation-on-maplibre*
*Context gathered: 2026-07-09*
