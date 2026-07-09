---
phase: 15-maplibre-core-trail-rendering-offline-parity
plan: 05
subsystem: ui
tags: [flutter, maplibre, trail-rendering, style-layers, widget-layer, markers, arrows]

# Dependency graph
requires:
  - phase: 15-04
    provides: "WandererMap on MapLibreMap — onStyleLoaded StyleController seam + onMapCreated controller hand-off; empty layers:[] seam marked for the trail track/arrows/markers"
provides:
  - "addTrailTrackLayers(StyleController, Trail, {routeColor}) — GeoJsonSource 'trail' + trail-casing/trail-route LineStyleLayers + trail-arrows SymbolStyleLayer (static), re-added in onStyleLoaded so they survive the theme swap"
  - "TrailMarkerLayer widget — a single WidgetLayer(allowInteraction:true) hosting tappable animated waypoint markers + start/finish pins with the 36px proximity nudge"
  - "self-registered 'arrow' map image (addImageFromIconData) — sprite-independent directional glyph"
affects: [15-06, TRAIL-01, TRAIL-02, TRAIL-03, TRAIL-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Track/arrow rendering as native GL style layers added imperatively via StyleController.addSource/addLayer inside onStyleLoaded (re-fires after setStyle, so layers survive CORE-02 theme swap)"
    - "Self-register the arrow icon via addImageFromIconData rather than depending on the style sprite (15-03 found file:// sprite resolution unreliable on device) — deterministic arrows"
    - "Interactive markers as a WidgetLayer widget in MapLibreMap.children reading MapController/MapCamera from context, recomputing the toScreenLocations nudge on every camera move"

key-files:
  created: []
  modified:
    - app/lib/components/map/trail_layer.dart
    - app/lib/components/base/wanderer_map.dart

key-decisions:
  - "Registered our own 'arrow' image (Icons.navigation, 32px, white) via addImageFromIconData instead of relying on the Protomaps sprite — 15-03 flagged file:// sprite resolution as unreliable on device, so shipping the glyph makes TRAIL-02 deterministic on both online and offline styles"
  - "symbol-spacing is a zoom-interpolated PIXEL expression [interpolate linear zoom 8->250, 12->160, 16->90] — denser at higher zoom per D-05, replicating the old meter-table density intent without the AnimationController motion"
  - "Track layers live in onStyleLoaded (not a one-shot after first load) so a CORE-02 setStyle theme swap re-adds source+layers+arrow image, which setStyle drops"

patterns-established:
  - "trail_layer.dart is now a stateless helper module (a builder fn + a StatelessWidget), not a StatefulWidget — no animation lifecycle to manage"

requirements-completed: [TRAIL-01, TRAIL-02, TRAIL-03, TRAIL-04]

# Metrics
duration: ~15min
completed: 2026-07-09
---

# Phase 15 Plan 05: Trail Track, Arrows & Markers Summary

**Rewrote `trail_layer.dart` from a `flutter_map` `StatefulWidget` (with a dead 12s `AnimationController` + ~60-line bearing loop) into two stateless pieces — `addTrailTrackLayers()` that draws the GPX track as a native white-casing/colored-route `LineStyleLayer` pair plus a static `trail-arrows` `SymbolStyleLayer`, and `TrailMarkerLayer` that renders tappable animated waypoints and nudged start/finish pins via a single `WidgetLayer` — then wired both into the 15-04 `WandererMap` shell (track in `onStyleLoaded`, markers in `children`).**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-07-09
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- **Track line (TRAIL-01):** `trail-casing` `LineStyleLayer` (`#ffffff`, width 9, round cap/join) is added before `trail-route` (routeColor `#3549bb` default, width 5, round cap/join), so 2px of white reads as an outline around the 5px route. Geometry is a single `GeoJsonSource(id:'trail')` LineString built from `gpx.allPoints` via `jsonEncode` (threat T-15-05-01 — no string concatenation).
- **Static arrows (TRAIL-02, D-04/D-05):** a single `trail-arrows` `SymbolStyleLayer` with `symbol-placement:line`, `icon-rotation-alignment:map`, `icon-allow-overlap`/`icon-ignore-placement`, `minZoom:8`, and a zoom-interpolated pixel `symbol-spacing`. No `AnimationController` anywhere — the old motion loop is deleted, only the density-by-zoom intent survives.
- **Waypoints (TRAIL-03):** 32px tappable `WidgetLayer` markers firing `onWaypointTap`, with the `AnimatedScale` 0.875→1.0 / 200ms / `easeOutBack` selection animation preserved verbatim.
- **Start/finish pins (TRAIL-04):** 28px green-bullseye / red-flag pins at `gpx.allPoints.first`/`.last`, nudged to `Alignment(1,0)`/`Alignment(-1,0)` when within 36 screen pixels via `MapController.toScreenLocations`, else centered. Recomputes on camera move (reads `MapCamera` from context).
- **Theme-swap survival (CORE-02):** track layers + the arrow image are (re)added inside `onStyleLoaded`, which re-fires after every `setStyle`, so a light/dark toggle never loses the track.
- **Deletions:** `AnimationController` + `SingleTickerProviderStateMixin`, `_addArrowsAlongPath`, `SphericalGreatCircle`/`initialBearingTo` bearing math, `PolylineLayer`/`MarkerLayer`/`Stack` scaffolding, and the `flutter_map` + `map_coordinate_adapter` imports are gone from `trail_layer.dart`.

## Requested Output Details

- **`symbol-spacing` chosen (pixels, zoom-interpolated):** `['interpolate', ['linear'], ['zoom'], 8, 250, 12, 160, 16, 90]` — denser (smaller spacing) at higher zoom. Declared in `_kArrowSpacing`. Marked for on-device tuning (D-05 / RESEARCH Pitfall 5: meter→pixel is approximate).
- **Arrow icon source:** **NOT** resolved from the style sprite. The plan allowed a sprite `arrow` with an `addImageFromIconData` fallback "if on-device verification shows the sprite does not provide it." Since 15-03 already found `file://` sprite resolution unreliable on device and I cannot verify on a physical device, I registered the arrow deterministically up front: `addImageFromIconData(id:'arrow', iconData: Icons.navigation, size: 32, color: Colors.white)`, scaled by `icon-size: 0.5`. This guarantees arrows render on both the online Protomaps style and the offline style regardless of sprite availability.

## Task Commits

1. **Task 1: Track casing+route + static arrows as native style layers** — `785bc925` (feat)
2. **Task 2: Waypoint + start/finish markers via WidgetLayer** — `9f51989e` (feat)

## Files Modified

- `app/lib/components/map/trail_layer.dart` — full rewrite: `addTrailTrackLayers()` builder + `TrailMarkerLayer` widget + ported `_buildCircularMarker`; all animation/bearing/flutter_map code deleted.
- `app/lib/components/base/wanderer_map.dart` — `onStyleLoaded` now takes the `StyleController` and calls `addTrailTrackLayers` (gated `showTrail && gpx != null`); `TrailMarkerLayer` added to `children` under the same gate.

## Decisions Made

See frontmatter `key-decisions`. Load-bearing: the arrow glyph is self-registered (sprite-independent) so TRAIL-02 does not inherit the unresolved 15-03/15-06 `file://` sprite risk.

## Deviations from Plan

### Interpreted (within plan latitude)

**1. [Rule 2 - Correctness] Registered the arrow image unconditionally instead of sprite-first**
- **Found during:** Task 1
- **Issue:** The plan's primary path was the sprite `arrow` with `addImageFromIconData` only as an on-device fallback. On-device verification is not available to this executor, and 15-03 already documented `file://` sprite resolution failing on device — a sprite-first path would risk shipping arrow-less tracks.
- **Fix:** Always register the `arrow` image before adding the arrows layer. No behavior lost; arrows are guaranteed on every style. Documented above and in code comments.
- **Files modified:** `trail_layer.dart`
- **Commit:** `785bc925`

**Total deviations:** 1 (correctness, within the plan's stated fallback latitude). No scope creep, no new packages.

## Known Stubs

None. Track, arrows, waypoints, and start/finish pins are all wired to live trail data (`gpx.allPoints`, `trail.expand.waypointsViaTrail`).

## Verification

- `flutter analyze` on both modified files: **No issues found**.
- Whole-app `flutter analyze`: **40 pre-existing issues**, all in unrelated files (`icon_util.dart` Font Awesome deprecations ×39, `test/models/feed_item_test.dart` unused import ×1) — none in this plan's files (SCOPE BOUNDARY; matches the 15-04 baseline of ~39).
- Grep gates pass: `trail-casing`, `symbol-placement`, `toScreenLocations`, `easeOutBack`, `flagCheckered`, `WidgetLayer` present; `AnimationController`, `_addArrowsAlongPath`, `PolylineLayer`, `MarkerLayer(` (flutter_map), and `flutter_map`/`map_coordinate_adapter` imports absent from `trail_layer.dart`.
- Static-arrow requirement (D-04/D-05): confirmed the `AnimationController`, `SingleTickerProviderStateMixin`, and the `_addArrowsAlongPath` bearing loop are physically deleted, not merely unused.
- Preserved: `AnimatedScale` (0.875→1.0 / 200ms / `easeOutBack`) and `onWaypointTap` both present in `TrailMarkerLayer`.

## Next Phase Readiness

- **15-06 (offline)** still owns the `pmtiles://file://` basemap + `file://` glyph/sprite rewrite and the unresolved on-device `file://` sprite risk. This plan's arrow glyph is sprite-independent, so it is unaffected by that risk.
- On-device tuning items for 15-06/verification: the `symbol-spacing` pixel values and the `icon-size: 0.5` arrow scale (D-05 explicitly accepts device tuning).
- No blockers introduced.

## Self-Check: PASSED

- Both modified files exist on disk with the described changes.
- Both task commits (`785bc925`, `9f51989e`) present in git history.
- No new packages; no architectural changes.

---
*Phase: 15-maplibre-core-trail-rendering-offline-parity*
*Completed: 2026-07-09*
