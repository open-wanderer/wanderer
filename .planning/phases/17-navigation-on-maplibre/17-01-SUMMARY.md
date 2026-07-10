---
phase: 17-navigation-on-maplibre
plan: 01
subsystem: mobile-map
tags: [flutter, maplibre, riverpod, navigation, gps, location-puck]

# Dependency graph
requires:
  - phase: 15-maplibre-core-trail-rendering-offline-parity
    provides: "WandererMap onMapCreated/onStyleLoaded/style-swap lifecycle, addTrailTrackLayers/TrailMarkerLayer, rewriteStyleForOffline, GlyphSpriteCachePaths/glyphSpriteCacheProvider"
  - phase: 16-list-map-screens-on-maplibre
    provides: "SearchMap onMapCreated/onStyleLoaded race-buffering fix, ml.MapCompass usage precedent, cluster_layer.dart addSource/updateGeoJsonSource in-place pattern, Duration.zero crash avoidance"
provides:
  - "navigation_screen.dart map-rendering layer fully on ml.MapLibreMap (native GL) — zero flutter_map/vector_map_tiles imports"
  - "Native location puck + heading-up follow via controller.enableLocation/trackLocation (CORE-07)"
  - "Native ml.MapCompass toggle (D-01 explicit north-reset override, D-02 always-visible) replacing app-local MapCompass (CORE-05)"
  - "Native controller.animateCamera for compass north-reset (CORE-06)"
  - "Imperative breadcrumb GeoJsonSource/LineStyleLayer, updated in place per GPS fix, fail-soft on malformed data (T-17-01)"
  - "Offline basemap composed via rewriteStyleForOffline + trailProvider.pmTiles (retires this screen's use of MultiPmTilesVectorTileProvider, unblocking OFFL-06 cleanup in a later phase)"
  - "Pointer-count Listener heuristic narrowing follow-break to single-finger drag (NAV-02) — pinch/rotate leave follow engaged"
affects: [17-02-navigation-on-maplibre, 17-03-navigation-on-maplibre, 18-retire-flutter-map-and-flomp-forks]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "onMapCreated/onStyleLoaded pending-style buffer inlined in navigation_screen.dart (mirrors SearchMap's Phase 16-03 race fix) — applied even though not spelled out in the plan's literal action text, since the same race class was already found twice in this project"
    - "enableLocation/trackLocation re-armed on every onStyleLoaded (native Style object is replaced by setStyle, dropping the location component and style-bound layers)"
    - "Breadcrumb as an imperative GeoJsonSource + LineStyleLayer pair, updated via style.updateGeoJsonSource in a ref.listen(navigationProvider) callback — never removed/re-added"
    - "Pointer-count Listener wrapper compensates for CameraChangeReason.apiGesture's lack of drag-vs-pinch distinction"

key-files:
  created: []
  modified:
    - app/lib/routes/navigation_screen.dart

key-decisions:
  - "Applied the SearchMap-style onMapCreated/onStyleLoaded race buffer (ml.StyleController? _pendingStyle) even though Task 1's action text did not explicitly call it out — RESEARCH.md/PATTERNS.md flag this exact race as already having caused two prior on-device bugs (Phase 16-03), so it was added as Rule 2 (missing critical functionality) rather than waiting for a checkpoint to surface it."
  - "TrailLayer (legacy flutter_map widget in trail_layer.dart), map_compass.dart, and pm_tile_provider.dart were left in place, unmodified — files_modified in this plan's frontmatter scopes to navigation_screen.dart only; their physical deletion is explicitly Phase 18 (CLEAN-01/02/03) per 17-CONTEXT.md's domain boundary. Post-change grep confirms map_compass.dart and pm_tile_provider.dart now have zero real importers app-wide, ready for that later deletion."
  - "TickerProviderStateMixin removed from _NavigationScreenState — no remaining member needs a vsync once AnimatedMapController is gone (sheet controllers are DraggableScrollableController, not AnimationController). Confirmed via flutter analyze reporting no issues."

patterns-established:
  - "Native onStyleLoaded re-arm bundles addTrailTrackLayers + breadcrumb source/layer + enableLocation/trackLocation together in one _onStyleLoaded helper, wrapped in a single try/catch (fail-soft, matches map_screen.dart's T-16-02 precedent)."

requirements-completed: [NAV-01, NAV-02, NAV-03, NAV-04, CORE-06, CORE-07]

# Metrics
duration: 10min
completed: 2026-07-10
---

# Phase 17 Plan 01: Navigation Map Host on MapLibre Summary

**navigation_screen.dart's map layer moved from FlutterMap to native ml.MapLibreMap — native location puck/follow via enableLocation/trackLocation, native ml.MapCompass toggle, imperative breadcrumb GeoJsonSource, offline pmtiles compose via rewriteStyleForOffline, and a pointer-count heuristic narrowing follow-break to single-finger drag.**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-07-10T07:52:00Z (approx, session start)
- **Completed:** 2026-07-10T08:01:14Z
- **Tasks:** 2/2 completed
- **Files modified:** 1 (`app/lib/routes/navigation_screen.dart`)

## Accomplishments

- Replaced the `FlutterMap` map host, `AnimatedMapController`, `CurrentLocationLayer`/`LocationMarkerDataStreamFactory`, app-local `MapCompass`, and `MultiPmTilesVectorTileProvider` offline branch with a single `ml.MapLibreMap` widget driven by native `enableLocation`/`trackLocation`/`animateCamera`/`ml.MapCompass` APIs.
- Route line + waypoints now render via the already-established `addTrailTrackLayers`/`TrailMarkerLayer` native GL functions instead of the legacy `TrailLayer` flutter_map widget.
- Breadcrumb (traveled-path) rendering became an imperative `GeoJsonSource`/`LineStyleLayer` pair, updated in place via `style.updateGeoJsonSource` inside a `ref.listen(navigationProvider)` callback with `.catchError` fail-soft handling (T-17-01).
- Compass toggle preserves the validated north-up/heading-up UX (D-01: explicit `animateCamera(bearing: 0, ...)` reset when turning heading-up off, `rotateNorthOnPressed: false` to prevent double-handling; D-02: `hideIfRotatedNorth: false`, always visible during navigation).
- Recenter restores the prior heading-up state rather than forcing north (D-03).
- Follow-break is now gated on a pointer-count heuristic (`_activePointers <= 1`) so a two-finger pinch-zoom or rotate no longer disengages follow — only a single-finger drag does (NAV-02).
- Offline navigation composes its basemap from the trail's `.pmtiles` cells via the same `rewriteStyleForOffline` transform `WandererMap` already uses, resolved from `trailProvider(widget.id).value?.pmTiles` (navigation resolves its trail asynchronously).
- Maneuver banner, stats sheet, button row, waypoint sheet, `_confirmExit`, `_iconForManeuverType`, and the `TraceletPositionSource`/GPS-stream wiring feeding `navigationProvider`/`navigationStatsProvider` are byte-for-byte unchanged.

## Task Commits

1. **Task 1: Swap navigation_screen's map host to ml.MapLibreMap with offline compose, trail track, breadcrumb, and native location puck** - `3f71eb5f` (feat)
2. **Task 2: Narrow follow-break to single-finger drag via a pointer-count Listener (NAV-02)** - `b9f78d6f` (feat)

**Plan metadata:** commit skipped — `commit_docs: false` in `.planning/config.json` (see Final Commit note below)

## Files Created/Modified

- `app/lib/routes/navigation_screen.dart` - Map-rendering layer rewritten onto `ml.MapLibreMap`; maneuver/stats/button-row UI and GPS-stream wiring unchanged.

## Decisions Made

- Inlined the `SearchMap`-style `onMapCreated`/`onStyleLoaded` race buffer (`_pendingStyle`) directly in `navigation_screen.dart` rather than extracting a third host widget — no second call site exists to justify the abstraction (per 17-RESEARCH.md Open Question 3's recommendation), and the buffer itself prevents a bug class already hit twice in Phase 16.
- Kept `_composeStyle`/`_swapStyle`/`_lastStyleJson`/`_cacheWarmed` field names and shapes matching `WandererMap`'s established CORE-02 pattern for consistency across the three map hosts (`WandererMap`, `SearchMap`, `navigation_screen`).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added onMapCreated/onStyleLoaded race buffering**
- **Found during:** Task 1 (map host rewrite)
- **Issue:** The plan's action text for Task 1 specifies `onMapCreated: (controller) => _controller = controller` with no buffering, but 17-RESEARCH.md and 17-PATTERNS.md both document that the native platform channel does not reliably fire `onMapCreated` before `onStyleLoaded` — `SearchMap` already had to add this exact buffer in Phase 16-03 after two list-map screens silently no-opped their first `onStyleLoaded` work. Without it, `_onStyleLoaded`'s trail-track/breadcrumb/location calls could silently no-op on first paint if the race lands the same way here.
- **Fix:** Added `ml.StyleController? _pendingStyle` field; `onMapCreated` replays a buffered pending style via `_onStyleLoaded`, and `onStyleLoaded` buffers into `_pendingStyle` if `_controller` is still null.
- **Files modified:** `app/lib/routes/navigation_screen.dart`
- **Verification:** `flutter analyze` clean; `flutter test` on the three required suites passes. On-device confirmation of the race itself is deferred to the 17-03 checkpoint, consistent with how Phase 16-03 originally surfaced it.
- **Committed in:** `3f71eb5f` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Preventative fix for a bug class already observed twice in this project (Phase 16-03); no scope creep — same file, same task, no new files or dependencies.

## Issues Encountered

None — `flutter analyze lib/routes/navigation_screen.dart` reported zero issues after each task, and `flutter test` on `navigation_provider_test.dart`, `navigation_stats_provider_test.dart`, and `offline_style_rewriter_test.dart` passed after Task 1 (all three, per plan) and `navigation_provider_test.dart` passed again after Task 2 (per plan).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `navigation_screen.dart` builds clean (`flutter analyze`) with zero `flutter_map`/`vector_map_tiles` imports and the existing unit suite green — ready for 17-02/17-03.
- Native behaviors (puck render, follow, single-finger-drag-vs-pinch precision, offline basemap render) are **not yet verified on a physical device** — this is explicitly deferred to a later on-device checkpoint (RESEARCH Open Questions 1 & 2), consistent with this project's established pattern of catching native-gesture/GPS surprises only on real hardware.
- `map_compass.dart` and `pm_tile_provider.dart` now have zero real importers app-wide (confirmed via grep) — both are ready for physical deletion whenever Phase 18 (CLEAN-01/02/03) runs; not deleted in this plan per its `files_modified` scope (navigation_screen.dart only).

## Self-Check: PASSED

- FOUND: `app/lib/routes/navigation_screen.dart`
- FOUND: `3f71eb5f` (Task 1 commit)
- FOUND: `b9f78d6f` (Task 2 commit)

---
*Phase: 17-navigation-on-maplibre*
*Completed: 2026-07-10*
