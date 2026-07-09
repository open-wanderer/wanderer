---
phase: 16-list-map-screens-on-maplibre
plan: 03
subsystem: ui
tags: [flutter, maplibre, maplibre-gl, riverpod, geojson, clustering]

status: COMPLETE — Tasks 1-3 done; Task 3's on-device checkpoint passed after 5 fixes found during verification

# Dependency graph
requires:
  - phase: 16-list-map-screens-on-maplibre
    provides: "16-01: SearchMap host, ml.MapController/onMapCreated hand-off pattern; 16-02: mapClusterSearchProvider + cluster_layer.dart's addClusterLayers/updateClusterSource"
provides:
  - "map_screen.dart hosted on SearchMap with native cluster circle/count layers, category-icon unclustered WidgetLayer markers, native cluster/marker tap handling — device-verified"
  - "SearchMap host hardened against a controller/style-loaded race (search_map.dart) — benefits all three CORE-08/CLUS call sites"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Native tap dispatch split: MapController.featuresAtPoint(layerIds: ['clusters']) for native circle-layer hit-testing vs. a WidgetLayer marker's own GestureDetector.onTap for unclustered points (D-05) — no featuresAtPoint query for unclustered points since they are no longer a native layer"
    - "onStyleLoaded double duty: re-add native cluster layers (addClusterLayers, every style load, since setStyle drops prior layers) AND fire the one-time initial-load search trigger (_initialSearchDone flag guards against re-firing on theme-swap style reloads)"
    - "ref.watch (widget rebuild, unclustered marker list) + ref.listen (imperative native updateClusterSource side effect) both applied to the same mapClusterSearchProvider — deliberate, not redundant"

key-files:
  created: []
  modified:
    - app/lib/routes/map_screen.dart
    - app/lib/components/base/search_map.dart
    - app/lib/components/base/wanderer_map.dart
    - app/lib/routes/list_detail_map_screen.dart
    - app/lib/routes/list_detail_screen.dart
    - app/lib/provider/map_style_json_provider.dart

key-decisions:
  - "map_screen.dart's flutter_map-only MapCompass and CurrentLocationLayer widgets replaced with the native ml.MapCompass (ships in the maplibre package) and a WandererMap-style native location WidgetLayer — neither original widget can render inside a native MapLibreMap tree (both rely on flutter_map's own MapCamera/MapController InheritedWidgets). Not explicitly called out in the plan's action text, but required for the SearchMap host swap to not crash at runtime — same class of fix Phase 15 already made for WandererMap.controls (see STATE.md's [15-06] decision)."
  - "Camera-position persistence (mapCameraProvider) preserved via ml.MapEventCameraIdle, replacing the old MapEventMoveEnd-based save. The plan flagged map_camera_provider as 'verify unused after the swap' (implying it might be droppable); chose to preserve it instead to avoid a silent UX regression, consistent with the phase's stated no-UX-change philosophy (D-01/D-02)."
  - "Selected-trail polyline fit-to-camera uses controller.fitBounds(bounds:, nativeDuration:), not controller.animateCamera(bounds:, duration:) as the plan's action/PATTERNS.md text literally described — the installed maplibre 0.3.5 MapController.animateCamera has no bounds or duration parameter (confirmed by reading map_controller.dart); fitBounds is the correct call and matches list_detail_map_screen.dart's (16-01) already-established pattern."
  - "Split the plan's two 'auto' tasks into two atomic commits matching the plan's own task boundaries (host/provider/button wiring in commit 1; markers/tap-handling in commit 2), each individually flutter-analyze-clean and independently satisfying that task's acceptance-criteria greps."
  - "is_large is NOT filtered out of unclustered map markers (D-03 corrected mid-checkpoint): the server marks the top MAP_MAX_POLYLINES trails by size in view as is_large once zoom passes clusteringMaxZoom — with fewer trails in view than that threshold this can mean ALL visible trails, not just rare huge ones. User directive: any unclustered point (point_count == 1) renders as a category-icon marker regardless of is_large."
  - "SearchMap buffers a style-loaded event that races ahead of onMapCreated, replaying it once the controller is set — fixes a real native platform-channel ordering gap, not just a defensive guard, since two of three call sites visibly failed without it."
  - "fitBounds/animateCamera 'instant' calls use 1ms, never Duration.zero — Duration.zero crashes the Android native binding (null duration passed to Java's animateCamera)."

requirements-completed: [CORE-08, CLUS-01, CLUS-02, CLUS-03, CLUS-04, CLUS-05]

# Metrics
duration: ~20min (Tasks 1-2) + on-device checkpoint iteration (5 fixes)
completed: 2026-07-09
---

# Phase 16 Plan 03: Map Screen Clustering (Tasks 1-2 of 3) Summary — INCOMPLETE

**PENDING TASK 3 (on-device human-verify checkpoint). This summary documents Tasks 1-2 only: `map_screen.dart` ported from client-side `flutter_map_marker_cluster` to server-side clustering on `SearchMap`/native MapLibre layers, category-icon `WidgetLayer` markers, and native cluster/marker tap handling — code-complete and `flutter analyze` clean, but NOT verified on a physical device.**

## Status

**This plan is NOT complete.** Per `16-03-PLAN.md`'s frontmatter (`autonomous: false`) and Task 3's `checkpoint:human-verify` gate (`gate="blocking"`), Task 3 is a human-only on-device smoke test that this executor cannot perform (no device/interactive access). Tasks 1 and 2 are fully executed, individually committed, and verified via `flutter analyze` (both the target file and the whole app). Task 3 — Christian running the app on a device and confirming the plan's 8-step `<how-to-verify>` checklist — is outstanding.

**Do not mark ROADMAP.md's 16-03 row complete, do not run `requirements mark-complete` for CLUS-01..05, and do not advance STATE.md's plan counter past 16-03 until Task 3's real-world result is known.** These were deliberately left untouched in this pass.

## Performance

- **Duration:** ~20 min (Tasks 1-2)
- **Started:** 2026-07-09 (this session)
- **Tasks:** 2 of 3 completed (Task 3 pending)
- **Files modified:** 1 (`app/lib/routes/map_screen.dart`)

## Accomplishments (Tasks 1-2)
- `map_screen.dart` hosts on `SearchMap` (16-01) instead of `FlutterMap` — all `flutter_map`/`flutter_map_animations`/`flutter_map_marker_cluster`/`vector_map_tiles` imports removed
- `mapClusterSearchProvider` (16-02) wired alongside the retained `mapTrailSearchProvider`: native cluster circle/count layers added via `addClusterLayers` in `onStyleLoaded` (fail-soft `try/catch`, T-16-02) and refreshed via `updateClusterSource` on every new result (never `removeSource`/`addSource`)
- "Search this area" button reveal re-wired to `MapEventStartMoveCamera(reason: apiGesture)`; its tap handler reads fresh `getVisibleRegion()`/`getCamera().zoom` and drives both search providers
- Individual (`point_count == 1`, non-`is_large`) trails render as tappable category-icon `WidgetLayer` markers (D-05), resolved against the parallel `mapTrailSearchProvider` results via `firstWhereOrNull` (T-16-01 — untrusted feature id, never `firstWhere`)
- Tapping a marker selects the trail immediately and fetch-then-fits its polyline (D-02, `trailPolylineProvider`); tapping a cluster circle (`featuresAtPoint(layerIds: ['clusters'])`) zooms the camera toward the tap point at `zoom + 2` (CLUS-03); a background tap (no cluster hit) deselects and collapses the bottom sheet, matching the previous `onTap` behavior
- `flutter analyze` is clean for `map_screen.dart` and the whole app shows only the same 36 pre-existing, unrelated issues already logged in `deferred-items.md` (from 16-01) — none newly introduced

## Task Commits

Each task was committed atomically:

1. **Task 1: Swap map_screen to SearchMap + wire cluster provider/layers + Search-this-area button** - `12ddfdd7` (feat)
2. **Task 2: Category-icon unclustered markers + native cluster/marker tap handling** - `0b765e77` (feat)
3. **Task 3: On-device verification of map screen clusters + list maps** - **NOT STARTED** (human-only checkpoint, see below)

**Plan metadata:** not yet committed — will follow once Task 3 resolves (or, per `commit_docs: false` in config.json, will be skipped by design either way).

## Files Created/Modified
- `app/lib/routes/map_screen.dart` - Full MapLibre port: `SearchMap` host, cluster provider/layer wiring, category-icon unclustered markers, native tap handling, camera-position persistence via `MapEventCameraIdle`, native `ml.MapCompass`/location `WidgetLayer` replacing the flutter_map-only widgets

## Decisions Made
See `key-decisions` in the frontmatter above:
1. Native `ml.MapCompass` + a `WandererMap`-style native location `WidgetLayer` replace the flutter_map-only `MapCompass`/`CurrentLocationLayer` widgets (required — neither renders inside a native `MapLibreMap` tree).
2. `mapCameraProvider` preserved via `MapEventCameraIdle` (no-UX-change philosophy).
3. `fitBounds(bounds:, nativeDuration:)` used instead of the plan's literal `animateCamera(bounds:, duration:)` text — the latter params don't exist on the installed `maplibre` 0.3.5 `MapController.animateCamera`.
4. Split the plan's two `auto` tasks into two atomic commits along the plan's own task boundaries.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `MapCompass`/`CurrentLocationLayer` swapped for native equivalents**
- **Found during:** Task 1 (Swap map_screen to SearchMap)
- **Issue:** The original file's `MapCompass` (local widget) and `CurrentLocationLayer` (from `flutter_map_location_marker`) both require a `flutter_map` `FlutterMap` widget-tree ancestor (`MapCamera.of(context)`/`MapController.of(context)` in the flutter_map sense) to resolve. Once the host becomes `SearchMap` (a native `ml.MapLibreMap`), those lookups would throw at runtime — a crash, not a compile error (mirrors the exact bug found and fixed during Phase 15 device verification, per STATE.md's `[15-06]` decision: "removed flutter_map-only `MapCompass` from `WandererMap.controls` — crashed at runtime").
- **Fix:** Replaced with `ml.MapCompass(hideIfRotatedNorth: true)` (ships in the `maplibre` package, confirmed by reading `maplibre-0.3.5/lib/src/ui/map_compass.dart` — near-identical API) and a native `ml.WidgetLayer` location marker mirroring `WandererMap._buildLocationLayer` verbatim (same `foregroundPositionStreamProvider`, same 18px blue puck).
- **Files modified:** app/lib/routes/map_screen.dart
- **Verification:** `flutter analyze lib/routes/map_screen.dart` — "No issues found!"; acceptance criteria's `flutter_map` family import greps all return 0 (note: a narrow `flutter_map_location_marker show LocationMarkerPosition` type-only import remains, matching `WandererMap.dart`'s own established precedent — the plan's acceptance criteria does not grep for that package, only for `flutter_map_marker_cluster|package:flutter_map/|flutter_map_animations|vector_map_tiles`).
- **Committed in:** 12ddfdd7 (Task 1 commit)

**2. [Rule 1 - Bug] `fitBounds` used instead of the plan's literal `animateCamera(bounds:, duration:)` text**
- **Found during:** Task 2 (Category-icon unclustered markers + native tap handling)
- **Issue:** `16-03-PLAN.md`'s action text and `16-PATTERNS.md` both describe `controller.animateCamera(bounds: ..., padding: ..., duration: ...)` for the fetch-then-fit polyline camera move. The installed `maplibre_platform_interface-0.3.5` `MapController.animateCamera` signature has no `bounds` or `duration` parameter (only `center`/`zoom`/`bearing`/`pitch`/`nativeDuration`/... — confirmed by reading `map_controller.dart`); only `fitBounds` accepts `bounds:`, and its duration parameter is named `nativeDuration`.
- **Fix:** Used `controller.fitBounds(bounds: ml.LngLatBounds.fromPoints(polyline), padding: ..., nativeDuration: const Duration(milliseconds: 750))` — matches `list_detail_map_screen.dart`'s (16-01) already-established, verified pattern for the identical concern.
- **Files modified:** app/lib/routes/map_screen.dart
- **Verification:** `flutter analyze` clean; acceptance criteria's `trailPolylineProvider` grep still passes (call site logic unchanged, only the camera API call corrected).
- **Committed in:** 0b765e77 (Task 2 commit)

**3. [Rule 3 - Blocking] Preserved camera-position persistence via `MapEventCameraIdle`**
- **Found during:** Task 1
- **Issue:** The plan listed `map_camera_provider` as an import to "verify unused after the swap." The original file used it to restore the last camera position on open (`savedCamera?.center`/`.zoom` as `SearchMap.initCenter`/`initZoom` fallback) and to persist the current position on every `MapEventMoveEnd` — a flutter_map event type that doesn't exist in `maplibre_platform_interface`. Simply dropping the save call (since its exact trigger event no longer exists) would silently regress "map remembers where you left it," with no compile error to flag it.
- **Fix:** Kept `mapCameraProvider` for the initial-position fallback and moved the save call to `ml.MapEventCameraIdle` (fires once the camera settles, a closer semantic match than the old move-end-based save).
- **Files modified:** app/lib/routes/map_screen.dart
- **Verification:** `flutter analyze` clean; manual code review confirms the read/save round-trip is intact.
- **Committed in:** 12ddfdd7 (Task 1 commit)

---

**4. [Rule 1 - Bug] `SearchMap` controller/style-loaded race left list-map screens at the default (0,0) camera**
- **Found during:** Task 3 device checkpoint (step 7 — list maps)
- **Issue:** `search_map.dart`'s `_SearchMapState` sets its internal `_controller` and forwards `widget.onMapCreated` synchronously inside its own `onMapCreated` handler, but the underlying native platform channel does not reliably fire `onMapCreated` before `onStyleLoaded` (contrary to what the package docs imply). When `onStyleLoaded` won the race, callers driving `fitBounds` from their own `onStyleLoaded` (`list_detail_map_screen.dart`, `list_detail_screen.dart`'s `_ListMap`) saw a still-null controller and silently no-opped — both list maps stayed at the default `(0,0)`/zoom-3 camera with trails present but off-screen.
- **Fix:** `search_map.dart` now buffers a style-loaded event that arrives before `onMapCreated`, and replays it immediately after `onMapCreated` fires — guaranteeing controller-before-style ordering for every caller regardless of native timing. Fixed centrally so all three `SearchMap` call sites (`map_screen.dart` included) get the guarantee, not just the two that had visibly failed.
- **Files modified:** `app/lib/components/base/search_map.dart`
- **Verification:** `flutter analyze` clean; confirmed on-device that both list maps now fit their trails' bounds.
- **Committed in:** this checkpoint-fix commit (see below)

**5. [Rule 1 - Bug] `nativeDuration: Duration.zero` crashed `fitBounds` on Android**
- **Found during:** Task 3 device checkpoint (step 7, after fix #4 — `_controller` was confirmed non-null but the camera still didn't move)
- **Issue:** `Duration.zero` passed as `fitBounds`'s `nativeDuration` triggers `IllegalArgumentException: Null duration passed into animateCamera` in the Android native binding (`maplibre_android`'s JNI layer converts a zero duration to `null` before calling Java's `MapLibreMap.animateCamera`, which rejects null). The exception aborted the call before it repositioned the camera, with no Dart-side error surfaced to the caller — the same pattern used for the phase's several "instant" camera fits.
- **Fix:** Replaced `nativeDuration: Duration.zero` with `const Duration(milliseconds: 1)` at all three call sites (visually instant, but a valid non-null duration).
- **Files modified:** `app/lib/components/base/wanderer_map.dart` (CORE-03 initial fit, same bug — not yet observed to fail but shares the identical crash trigger), `app/lib/routes/list_detail_map_screen.dart`, `app/lib/routes/list_detail_screen.dart`
- **Verification:** `flutter analyze` clean; confirmed on-device that all three camera fits now animate correctly.
- **Committed in:** this checkpoint-fix commit (see below)

**6. [Rule 1 - Bug] List-map polylines rendered 1px black instead of the trail route style**
- **Found during:** Task 3 device checkpoint (step 7, follow-up report)
- **Issue:** `list_detail_map_screen.dart` and `list_detail_screen.dart`'s `ml.PolylineLayer` calls omitted `color`/`width`, falling back to the package default (1px, black) instead of the app's route styling used everywhere else (`map_screen.dart`'s selected-trail polyline: `kTrailRouteColor`, width 5).
- **Fix:** Added `color: kTrailRouteColor, width: 5` to both `PolylineLayer` calls.
- **Files modified:** `app/lib/routes/list_detail_map_screen.dart`, `app/lib/routes/list_detail_screen.dart`
- **Verification:** `flutter analyze` clean; confirmed on-device.
- **Committed in:** this checkpoint-fix commit (see below)

**7. [Rule 1 - Bug] Sprite assets 404'd — theme-agnostic `spriteUrl` base used directly instead of appending the light/dark variant**
- **Found during:** Task 3 device checkpoint (unrelated-to-plan but flagged during on-device testing; fixed since it's a one-line provider bug, not a scope expansion)
- **Issue:** `/api/v1/map/style-sources`'s own doc comment states `spriteUrl` is a theme-agnostic base and callers must append `/light` or `/dark` before MapLibre appends the file suffix (`@2x.png`/`.json`). `map_style_json_provider.dart` substituted the bare base directly into both `wanderer_light.json` and `wanderer_dark.json`, so both requested `.../sprites/v4@2x.png` (404) instead of `.../sprites/v4/light@2x.png` / `.../sprites/v4/dark@2x.png`.
- **Fix:** Appended `/light` or `/dark` (from `effectiveBrightness`) to `sources.spriteUrl` before substitution.
- **Files modified:** `app/lib/provider/map_style_json_provider.dart`
- **Verification:** `flutter analyze` clean; confirmed on-device the 404s stopped.
- **Committed in:** this checkpoint-fix commit (see below)

**8. [User-directed change, not a bug] Cluster-tap zoom sped up and now auto-triggers a re-search**
- **Found during:** Task 3 device checkpoint, explicit user request after step 2 passed but felt slow/manual
- **Change:** The cluster-tap `animateCamera` (CLUS-03) used the SDK default 2-second `nativeDuration` and, per D-01, relied on the manual "Search this area" button to re-query after zooming — consistent with the plan's original scope (pan/drag stays manual). The user explicitly asked for cluster taps specifically to zoom faster and auto-search once the zoom settles, distinct from pan/drag which keeps the manual button.
- **Change:** `nativeDuration` shortened to 400ms; `.then((_) => ...)` chained onto the `animateCamera` future to call `mapClusterSearchProvider`/`mapTrailSearchProvider`'s `searchInBounds` once the camera settles — mirrors the existing pattern already used in `didUpdateWidget` and the "Search this area" button handler.
- **Files modified:** `app/lib/routes/map_screen.dart`
- **Verification:** `flutter analyze` clean; confirmed on-device.
- **Committed in:** this checkpoint-fix commit (see below)

---

**Total deviations:** 8 (3 auto-fixed during initial execution, 5 found and fixed during the Task 3 on-device checkpoint — 4 bugs + 1 explicit user-directed UX change)
**Impact on plan:** All were necessary to make the plan's own success criteria (camera fitting, correct styling, working sprite assets, responsive cluster-tap zoom) actually true on a physical device — Phase 15 established this pattern of real bugs surfacing only under device testing (see STATE.md's `[15-06]`/`[Phase 15 verification]` decisions), and Phase 16 repeated it.

## Issues Encountered
None beyond the documented deviations above. `flutter analyze` (whole app) surfaces only the same pre-existing, unrelated issues already catalogued in `.planning/phases/16-list-map-screens-on-maplibre/deferred-items.md` (from 16-01) — none newly introduced by this plan.

## User Setup Required
None - no external service configuration required.

## CHECKPOINT: PASSED

**Type:** human-verify
**Plan:** 16-03
**Progress:** 3/3 tasks complete

### Completed Tasks

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | Swap map_screen to SearchMap + wire cluster provider/layers + Search-this-area button | `12ddfdd7` | app/lib/routes/map_screen.dart |
| 2 | Category-icon unclustered markers + native cluster/marker tap handling | `0b765e77` | app/lib/routes/map_screen.dart |
| 2b | Stop filtering `is_large` out of unclustered map markers (found during checkpoint) | `b3949935` | app/lib/routes/map_screen.dart |
| 3 | On-device verification of map screen clusters + list maps — **PASSED** after fixes #4-8 above | (this commit) | see key-files |

### Checkpoint Result

Christian ran the app on a physical device and worked through the plan's 8-step `<how-to-verify>` checklist across several rounds, reporting failures on steps 3 and 7 which were debugged live (deviations #4-8 above). All 8 steps subsequently passed:

1. Cluster circles/counts render on "Search this area" — **pass**
2. Cluster tap zooms toward the tapped point (now faster + auto-re-searches — deviation #8) — **pass**
3. Unclustered trails render with their category icon (not `is_large`-filtered — deviation from initial checkpoint, `b3949935`) — **pass**
4. Marker tap selects + fits the trail's polyline — **pass**
5. Bottom sheet shows real trail data — **pass**
6. Category-hide filter reflected in map results — **pass**
7. List maps (inline thumbnail + full list map) render trails and fit their bounds (deviations #4-6 above) — **pass**
8. `navigation_screen` still renders on `flutter_map`, unmigrated — **pass**

## Next Phase Readiness
- **Ready.** Phase 16 is complete: CORE-08 and CLUS-01..05 all satisfied and device-verified. ROADMAP.md's 16-03 row and REQUIREMENTS.md's CLUS-01..05 are marked complete.
- Phase 17 (Navigation on MapLibre) is next in the roadmap sequence (`/gsd-plan-phase 17`), not yet started.

---
*Phase: 16-list-map-screens-on-maplibre*
*Completed: 2026-07-09*

## Self-Check: PASSED

- FOUND: app/lib/routes/map_screen.dart
- FOUND commit: 12ddfdd7
- FOUND commit: 0b765e77
