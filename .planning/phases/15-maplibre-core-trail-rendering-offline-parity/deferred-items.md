# Deferred Items — Phase 15

Out-of-scope discoveries logged during execution (not fixed — see SCOPE BOUNDARY).

## 15-03

- **Pre-existing dead code in `app/lib/components/trail/trail_dropdown.dart:124-126`**
  — `_allowDelete()` unconditionally `return false;` before its real body
  (`final user = ref.watch(authProvider).value; return user != null && ...`),
  so `flutter analyze` reports one `dead_code` warning at line 126. This predates
  15-03 (the download-delete permission gate is intentionally hard-disabled) and
  is untouched by this plan's glyph-cache wiring. Left as-is; whoever re-enables
  trail deletion should remove the early `return false;`.

## 15-06

### RESOLVED — app does not build: `navigation_screen.dart` references deleted `TrailLayer`

- **Status: FIXED** (commit `192b3a89`, orchestrator-applied) — restored the old flutter_map
  `TrailLayer` widget verbatim into `trail_layer.dart` (minus the already-dead `showArrows`
  animation branch), explicitly scoped as "legacy, delete when navigation_screen migrates to
  MapLibre in Phase 17" via a doc comment on the class. Whole-app `flutter analyze`: 0 errors.
  This is option (a) from the recommendation below — cheapest, no Phase-17 pull-forward.
- **Discovered:** 15-06 Task 2 (whole-app `flutter analyze`).
- **Error:** `lib/routes/navigation_screen.dart:250` — `The method 'TrailLayer' isn't
  defined` (`undefined_method`). Plus an `unused_import` of `trail_layer.dart` at line 15.
- **Origin:** PRE-EXISTING, not caused by 15-06. 15-05 (`785bc925`, `9f51989e`) rewrote
  `trail_layer.dart`, deleting the old flutter_map `TrailLayer` `StatefulWidget` and
  replacing it with the maplibre-native `addTrailTrackLayers()` + `TrailMarkerLayer`.
  `navigation_screen.dart` (a flutter_map screen, last touched by `8f9705cb`) still calls
  the old `TrailLayer(...)`. The error survived at the end of 15-05.
- **Impact:** The app will not `flutter build` / `flutter run` — so the 15-06 physical-device
  offline gate (Task 3) cannot be exercised until this compiles.
- **Why not fixed here:** Out of 15-06's scope (`navigation_screen.dart` is not in
  `files_modified`) and architecturally entangled — the new `addTrailTrackLayers`/
  `TrailMarkerLayer` API is maplibre-native and cannot render inside `navigation_screen`'s
  flutter_map. A proper fix either (a) restores a flutter_map trail-render path for
  `navigation_screen` (reverses part of 15-05) or (b) migrates `navigation_screen` to
  maplibre (Phase 17, NAV-01..04 / CORE-05/06/07). Both are Rule 4 architectural decisions.
- **Recommendation:** Resolve before the offline gate. Fastest unblock for testing:
  temporarily restore/guard the flutter_map trail rendering in `navigation_screen`, or
  fold in the Phase-17 navigation migration.

### RESOLVED — runtime crash: `MapCompass` requires a `FlutterMap` ancestor

- **Status: FIXED** (orchestrator-applied, post-Task-3-attempt) — removed `MapCompass` from
  `trail_detail_map_screen.dart`'s `WandererMap.controls` list. `MapCompass` (`map_compass.dart`)
  calls `MapCamera.of(context)`, a `flutter_map` API requiring a `FlutterMap` ancestor.
  `WandererMap` became `MapLibreMap` in 15-04, so this crashed at runtime with "Bad state:
  `MapCamera.of()` should not be called outside a `FlutterMap`" the first time the screen
  actually rendered on device (not caught by `flutter analyze`, since it's a runtime-only
  failure, not a static type error).
- **Impact of the fix:** `trail_detail_map_screen` temporarily has no rotation-reset compass
  control. A MapLibre-native compass is CORE-05 (Phase 17) — restore with maplibre's built-in
  compass widget then, rather than building an interim one now.
- **Unused import cleanup:** `map_compass.dart` import removed from `trail_detail_map_screen.dart`.
- Whole-app `flutter analyze`: still 0 errors after this fix.

### OFFL-06 deferred — `pm_tile_provider.dart` NOT deleted

- **Discovered:** 15-06 Task 2.
- **Planned:** delete `lib/vendor/vector_map_tiles/pm_tile_provider.dart` and confirm no
  `MultiPmTilesVectorTileProvider` reference remains (the plan assumed 15-04 removed all
  consumers).
- **Reality:** `navigation_screen.dart` still consumes `MultiPmTilesVectorTileProvider`
  for its offline flutter_map vector tiles (lines 31, 71, 111, 160). STATE.md confirms
  `navigation_screen` is the Phase-17 flutter_map holdout for CORE-05/06/07.
- **Why not deleted:** Deleting the provider would break `navigation_screen`, violating the
  plan's own "the 4 flutter_map screens still build" acceptance criterion.
- **Recommendation:** Delete `pm_tile_provider.dart` in Phase 17 (or Phase 18 CLEAN) once
  `navigation_screen` migrates off flutter_map. OFFL-06 closes then.
