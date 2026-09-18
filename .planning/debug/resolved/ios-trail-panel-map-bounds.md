---
status: resolved
trigger: "On iOS the trail map does not focus correctly on the trail bounds in this screen. On Android it works"
created: 2026-09-18
updated: 2026-09-18
---

## Current Focus

reasoning_checkpoint:
  hypothesis: "On iOS, `MapController.fitBounds()` (maplibre_ios 0.3.5's `mapView.setVisibleCoordinateBounds$1(...)`, wrapping the deprecated `-[MLNMapView setVisibleCoordinateBounds:edgePadding:animated:]`) silently no-ops the first time it is called right after a map's style loads, because it is invoked synchronously from `onStyleLoaded`/`onMapCreated` — which can fire before Flutter's iOS platform view has been laid out to its final on-screen frame. With no valid viewport to fit against, the native call leaves the camera untouched at whatever `initCenter`/`initZoom` the map was created with. Android does not hit this because its platform view is already sized when `onMapCreated` fires, so `CameraUpdateFactory.newLatLngBounds` + `animateCamera` always operates on a valid viewport."
  confirming_evidence:
    - "app/lib/components/base/trail_map.dart: TrailMap's `initCenter` is the trail's own lat/lon and `initZoom: 18` — exactly the 'zoomed in on the trail's centre' behaviour reported."
    - "app/lib/components/base/trail_collection_map.dart: TrailCollectionMap defaults `initCenter` to `Geographic(lat: 0, lon: 0)` and `initZoom` to 3 when the caller doesn't override them; list_detail_map_screen.dart never passes `initCenter` — exactly the '0,0' behaviour reported for list maps. Both symptoms are explained by the same mechanism: fitBounds is a no-op and the map is simply left at its init values, which differ per widget."
    - "~/.pub-cache/hosted/pub.dev/maplibre_ios-0.3.5/lib/src/map_state.dart:182-202 — iOS `fitBounds()` only forwards `bounds` and `padding` to `setVisibleCoordinateBounds:edgePadding:animated:`; it silently drops `nativeDuration`/`bearing`/`pitch`/`offset`, consistent with a thin, best-effort wrapper around a method whose docs mark it 'Deprecated'."
    - "~/.pub-cache/hosted/pub.dev/maplibre_ios-0.3.5/lib/src/map_state.dart:116-123 — vendor code explicitly acknowledges style loading can complete synchronously ('may have already finished loading before FlutterApi is registered, e.g. if the style is provided via JSON directly') and manually re-invokes the style-loaded callback in that case. TrailMap/TrailCollectionMap both pass `initStyle` as an inline JSON string (via `rewriteStyleForProxy`), not a URL — exactly the fast/synchronous path the vendor comment describes, which is what lets `onStyleLoaded` (and our `fitBounds` call from inside it) fire before the platform view has necessarily been given its final frame."
    - "Every current fitBounds call site in app/lib is invoked from onStyleLoaded/onMapCreated (grep across 9 files) — matches the documented general Flutter-iOS platform-view race where the first camera-fit command after view creation can silently fail because the native view isn't laid out yet (same class of bug as flutter/flutter#59502 for google_maps_flutter's newLatLngBounds)."
    - "profile_trail_map_screen.dart:277-288 chains `cameraFuture.then()` to read `controller.getVisibleRegion()`/`getCamera().zoom` for the initial trail search — if fitBounds silently no-ops on iOS, this reads the WRONG (un-fitted) viewport too, meaning the bug likely also causes wrong initial search results on iOS, not just a visual zoom issue. This is a functional consequence consistent with treating fitBounds as unreliable until verified."
  falsification_test: "If this hypothesis is wrong, then after fixing fitBounds calls to retry against `controller.getCamera()` until the camera value actually changes (or a bounded retry budget is exhausted), the map would STILL open un-fitted on iOS. The user confirming (on-device) that trail panel / trail_detail_map_screen / list_detail_map_screen now fit correctly on iOS is the falsification/confirmation test — this cannot be verified without running on a real iOS device/simulator, which this environment cannot do (build/run is user-only per project convention)."
  fix_rationale: "The bug is inside a third-party plugin's (maplibre_ios 0.3.5) deprecated native call, not in our bounds data (bounds are correct — Android renders them fine) or our widget logic. Patching the vendor package directly is fragile (gets wiped by `pub cache repair`/`flutter pub get`, already a known pain point for this project per the maplibre_ffi.g.dart stret patch). The robust, in-repo fix is a shared `fitBoundsReliably()` helper that calls `MapController.fitBounds()`, waits briefly, and checks `controller.getCamera()` against the pre-fit camera — retrying up to a few times if the camera didn't actually move. This directly targets the OBSERVED symptom (camera doesn't change) regardless of the exact internal native timing cause, is a no-op-cost change on Android (succeeds on attempt 1), and replaces the ad-hoc, duplicated fitBounds calls at every initial-load call site with one shared, testable function — matching the debug file's own note to 'look for a shared bounds-fitting helper' as the common point of failure."
  blind_spots: "Cannot run the app on an iOS simulator/device in this environment to directly observe `setVisibleCoordinateBounds` being called with a zero/stale frame, or to confirm the retry actually resolves it — this is inferred from vendor source comments, the exact-match symptom/init-value correlation, and a well-documented general Flutter-iOS platform-view race in other map plugins, not from a captured native stack/frame dump. If the real cause were instead a hard iOS SDK bug where the deprecated selector NEVER applies regardless of timing (e.g. permanently broken in MapLibre Native 6.25), retries would exhaust and the map would remain un-fitted — the human-verify checkpoint after this fix is required to confirm on-device."
next_action: none — resolved, user confirmed on iOS device

## Symptoms

DATA_START
expected: When the trail panel in the trail detail screen opens, the map camera fits the trail's bounds (whole trail visible, sensibly zoomed). This is the behaviour observed on Android.
actual: On iOS the map is fully zoomed in on the centre of the trail — presumably the trail's lat/lon point — instead of being fitted to the trail bounds.
errors: None seen in console / Xcode log.
timeline: Never worked on iOS. Android works.
reproduction: Open any trail in trail_detail_screen or trail_detail_map_screen on iOS. Consistent, every time.
additional: The same happens with list maps — list_detail_map_screen opens at lat/lon 0,0 on iOS. So this appears to be a general problem with fitting the map camera to bounds on iOS, not specific to the trail panel. Note the two symptoms differ: trail panel = zoomed in on the trail's centre point; list map = 0,0.
scope: Flutter app under app/. Relevant files: app/lib/components/trail/trail_panel.dart, app/lib/routes/list_detail_map_screen.dart, app/lib/components/base/trail_map.dart, app/lib/components/base/trail_collection_map.dart, app/lib/components/map/trail_layer.dart, app/lib/routes/trail_detail_screen.dart, app/lib/routes/trail_detail_map_screen.dart. Map library is MapLibre (see memory: iOS-specific MapLibre quirks exist, e.g. maplibre_ffi stret patch, setConnected). Look for a shared bounds-fitting helper used by all these screens — that's the likely common point of failure.
DATA_END

## Evidence

- timestamp: 2026-09-18T00:00:00Z
  checked: app/lib/components/base/trail_map.dart, app/lib/components/base/trail_collection_map.dart
  found: TrailMap._fitInitialCamera() calls `controller.fitBounds(bounds: widget.trail.bounds, ...)` on `hasExtent`, else falls back to `moveCamera(center: trail.lat/lon, zoom: 18)`. TrailCollectionMap has no bounds-fit of its own — callers (list_detail_map_screen.dart etc.) call `fitBounds` themselves from `onStyleLoaded`. TrailCollectionMap's default `initCenter` is `Geographic(lat: 0, lon: 0)`, default `initZoom` is 3.
  implication: Both symptom screens (trail panel via TrailMap, list map via TrailCollectionMap) funnel through the same `MapController.fitBounds()` call, exactly as the debug file's scope note predicted. The "actual" behaviour in both cases exactly matches each widget's own init defaults, which is the signature of fitBounds being a complete no-op rather than a miscalculation.
- timestamp: 2026-09-18T00:05:00Z
  checked: ~/.pub-cache/hosted/pub.dev/maplibre_ios-0.3.5/lib/src/map_state.dart (fitBounds, _onPlatformViewCreated, _didFinishLoadingStyle)
  found: iOS `fitBounds()` forwards only `bounds`+`padding` to `mapView.setVisibleCoordinateBounds$1(...)`, silently dropping nativeDuration/bearing/pitch/offset. That native selector is documented "Deprecated" in the generated FFI bindings. `_onPlatformViewCreated` explicitly handles the case where style-loading already finished "before FlutterApi is registered, e.g. if the style is provided via JSON directly" by manually invoking `_didFinishLoadingStyle` inline, synchronously, during view creation.
  implication: Our maps pass `initStyle` as inline JSON (not a URL) via `rewriteStyleForProxy`, hitting exactly this fast/synchronous path — `onStyleLoaded` (and therefore our `fitBounds` call) can fire essentially at platform-view-creation time, before the native view is guaranteed to have its final laid-out frame. This is a known general class of Flutter-iOS platform-view bug (matches flutter/flutter#59502, the equivalent bug for google_maps_flutter's newLatLngBounds).
- timestamp: 2026-09-18T00:10:00Z
  checked: maplibre_android-0.3.5/lib/src/map_state.dart fitBounds vs maplibre_ios-0.3.5's
  found: Android's fitBounds uses `CameraUpdateFactory.newLatLngBounds` + `animateCamera` with a completer awaiting native completion/cancellation callbacks — a fundamentally different, non-deprecated code path from iOS's `setVisibleCoordinateBounds`.
  implication: Explains the platform asymmetry directly — Android's camera-fit mechanism and platform-view creation timing don't share iOS's early/synchronous style-load race.
- timestamp: 2026-09-18T00:15:00Z
  checked: grep -rn "fitBounds(" app/lib/ (11 call sites across 9 files)
  found: Every fitBounds call site is invoked from onStyleLoaded/onMapCreated (initial load) or from a later user-interaction callback (marker tap, expand button, search result selection). Initial-load sites: trail_map.dart, list_detail_map_screen.dart (onStyleLoaded), list_detail_screen.dart (onStyleLoaded), route_planner_screen.dart (_fitInitialCamera, called from onMapCreated), settings_offline_regions_map_screen.dart (_fitToBbox, called from onStyleLoaded), profile_trail_map_screen.dart (_maybeFitAndSearch). User-interaction sites (map_screen.dart _selectTrail, list_detail_map_screen.dart _onMarkerTap/_deselect, profile_trail_map_screen.dart _selectTrail, trail_detail_map_screen.dart's expand IconButton) run against an already-visible, already-sized map and are not subject to the same race.
  implication: The fix should target the initial-load call sites specifically via one shared, reliable helper, per the debug file's own suspicion of "a shared bounds-fitting helper" as the common point of failure — no such helper currently exists; every screen duplicates its own inline fitBounds call.
- timestamp: 2026-09-18T00:20:00Z
  checked: app/lib/routes/profile_trail_map_screen.dart:277-297
  found: After the initial `fitBounds`/`animateCamera` future resolves, the code reads `controller.getVisibleRegion()` and `controller.getCamera().zoom` to run the screen's initial trail/cluster search.
  implication: If fitBounds silently no-ops on iOS, this reads the wrong (un-fitted) viewport too — the bug likely also causes wrong initial search results on profile_trail_map_screen on iOS, not just an incorrect zoom. Strengthens the case for a verified (not fire-and-forget) fit helper.

## Eliminated

- hypothesis: "Bounds values themselves are wrong/degenerate (e.g. trail.bounds or the list's combined bounds compute to a zero-size box)."
  evidence: "Android renders the exact same bounds data correctly (per symptom report: 'On Android it works'), and the observed iOS camera state in both symptom sites exactly matches each widget's *init* center/zoom defaults rather than a plausible-but-wrong fit of the real bounds — the signature of a no-op, not a bad calculation."
  timestamp: 2026-09-18T00:12:00Z

## Resolution

root_cause: "On iOS, `MapController.fitBounds()` (maplibre 0.3.5's iOS binding, `mapView.setVisibleCoordinateBounds$1(...)` wrapping the deprecated native `-[MLNMapView setVisibleCoordinateBounds:edgePadding:animated:]`) is called synchronously from `onStyleLoaded`/`onMapCreated`, which — because our maps load their style from an inline JSON string rather than a URL — can fire before Flutter's iOS platform view has been laid out to its final on-screen frame. The native call then has no valid viewport to fit against and silently leaves the camera at its `initCenter`/`initZoom`, which is why the trail panel shows the trail's own centre point at initZoom 18, and the list map shows lat/lon 0,0 at initZoom 3 (TrailCollectionMap's un-overridden defaults)."
fix: "Added `app/lib/util/map/reliable_fit_bounds.dart` exporting `fitBoundsReliably()`, a drop-in wrapper around `MapController.fitBounds()` that reads `controller.getCamera()` before fitting and retries (up to 5 attempts, 100ms apart) until the camera actually changes or the retry budget is exhausted. Replaced the initial-load `fitBounds()` calls with it in: trail_map.dart (_fitInitialCamera), list_detail_map_screen.dart (onStyleLoaded), list_detail_screen.dart (onStyleLoaded), route_planner_screen.dart (_fitInitialCamera), settings_offline_regions_map_screen.dart (_fitToBbox), profile_trail_map_screen.dart (_maybeFitAndSearch). User-interaction fitBounds call sites (already-visible, already-sized maps) were left unchanged."
verification: "flutter analyze and flutter test run clean after the change (see verifying phase). Cannot be verified end-to-end without an iOS device/simulator — user must confirm on-device per project convention (user builds/installs)."
files_changed:
  - app/lib/util/map/reliable_fit_bounds.dart
  - app/lib/components/base/trail_map.dart
  - app/lib/routes/list_detail_map_screen.dart
  - app/lib/routes/list_detail_screen.dart
  - app/lib/routes/route_planner_screen.dart
  - app/lib/routes/settings_offline_regions_map_screen.dart
  - app/lib/routes/profile_trail_map_screen.dart
