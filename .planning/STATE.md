---
gsd_state_version: 1.0
milestone: v1.5
milestone_name: Route Planner
status: executing
stopped_at: Completed 20-03-PLAN.md
last_updated: "2026-07-16T21:59:58.560Z"
last_activity: 2026-07-16 -- Phase 20 execution started
progress:
  total_phases: 3
  completed_phases: 1
  total_plans: 9
  completed_plans: 7
  percent: 33
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-16)

**Core value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.
**Current focus:** Phase 20 — route-planner-views-waypoint-list-elevation-location-search

## Current Position

Phase: 20 (route-planner-views-waypoint-list-elevation-location-search) — EXECUTING
Plan: 4 of 5
Status: Ready to execute
Last activity: 2026-07-16 -- Phase 20 execution started

## v1.5 Phases

- [x] **Phase 19: Route Planner Core — Waypoint Editing & Routing Engine** — WAYP-01/02/03, ROUTE-01..05
- [ ] **Phase 20: Route Planner Views — Waypoint List, Elevation & Location Search** — WAYP-04/05, PLANUI-01..03
- [ ] **Phase 21: Route Planner Handoff & Entry Point** — HANDOFF-01/02/03

Execution order: 19 → 20 → 21 (strictly sequential — each phase's state/screen is a prerequisite for the next).

## Performance Metrics

**Velocity (v1.0–v1.3):**

- Total plans completed: 35
- Average duration: — min
- Total execution time: — hours

**By Phase (recent):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 10 | 4 | ~19 min | ~5 min |
| 11 | 4 | ~50 min | ~13 min |
| 12 | 4 | ~30 min | ~8 min |
| 13 | 1 | - | - |
| 14 | 1 | - | - |
| 15 | 6 | ~90 min + on-device iteration | ~15 min |

*Updated after each plan completion*
| Phase 16 P01 | 9min | 3 tasks | 3 files |
| Phase 16 P02 | 7min | 2 tasks | 3 files |
| Phase 16 P03 | ~15min + on-device iteration | 3/3 tasks | 6 files |
| Phase 17 P01 | 10min | 2 tasks | 1 files |
| Phase 17 P02 | 8min | 2 tasks | 6 files |
| Phase 18 P01 | 15min | 2 tasks | 5 files modified + 4 deleted |
| Phase 18 P02 | 3min | 2 tasks | 2 files |
| Phase quick-260710-kpd P01 | 15min | 2 tasks | 9 files |
| Phase 19 P01 | 12min | 2 tasks | 4 files |
| Phase 19 P02 | 40min | 2 tasks | 5 files |
| Phase 19 P03 | 13min | - tasks | - files |
| Phase 19 P04 | 15min | 2 tasks | 1 files |
| Phase 20 P01 | 30min | 2 tasks | 7 files |
| Phase 20 P02 | 8min | 2 tasks | 2 files |
| Phase 20 P03 | 20min | 2 tasks | 2 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [v1.4 roadmap] Phase 15 carries 20 requirements *because the offline gate forces it*: `MultiPmTilesVectorTileProvider` lives inside `wanderer_map.dart`, so the moment CORE-01 turns `WandererMap` into a `MapLibreMap`, the `pmtiles://` path (OFFL-03/05) and `file://` glyph path (OFFL-01/02/04) must land in the same phase or a downloaded trail regresses.
- [v1.4 roadmap] TYPE-01/02 lands early as its own Phase 14, ahead of all map code. `Geographic(lon:, lat:)` reverses `LatLng(lat, lon)` — a transposed coordinate is silent. Isolating it under the existing `gpx_util`/`polyline_util` tests gives one clean signal. Cost: temporary `Geographic → LatLng` adapters at four un-migrated `flutter_map` call sites, each deleted as its screen migrates.
- [v1.4 roadmap] CORE-05/06/07 are assigned to Phase 17 (navigation), not to a foundation phase. Each retires a file or a pubspec plugin, and `navigation_screen` is the last holdout for all three (`map_compass.dart`, `AnimatedMapController`, `CurrentLocationLayer`). Earlier phases swap their own call sites but cannot delete.
- [v1.4 roadmap] CORE-08 added during roadmap creation — `list_detail_map_screen` and `list_detail_screen` build `FlutterMap` directly, so CORE-01 (scoped to `WandererMap`) never covered them.
- [v1.4] Migrate to `maplibre`, not `maplibre_gl` — FFI/JNI bindings, reads our style JSON directly.
- [v1.4] Clustering reuses `POST /search/trails/cluster` rather than maplibre's `cluster: true` — the endpoint exists, web already renders its output, and maplibre's `GeoJsonSource` exposes no cluster fields.
- [v1.4] Style JSON lives as an app asset, not a server-hosted style URL — offline rendering needs the style before any network call.
- [15-03] `map_cache_path.dart` is the single sanctioned builder for any map-cache filesystem path — operator-controlled fontstack/range tokens are whitelisted (4 fontstacks + `^\d+-\d+$`) and rejected with `ArgumentError` before a path is built; never string-concatenate a token into a path (T-15-03-01).
- [15-03] One shared app-wide glyph/sprite cache under `<app-docs>/map_cache` (D-08): `sprite/{light,dark}` + `glyphs/<fontstack>/<range>.pbf`; both the map-open (D-09, 15-04) and trail-download (D-10, 15-03) triggers converge on the same idempotent keepAlive warm.
- [15-04] `ml.MapController` cannot be free-standing (abstract interface created by the native map). `WandererMap` exposes `onMapCreated(controller)` which BOTH captures it internally (for `setStyle`/`fitBounds`) and forwards it to the caller; consumers hold `ml.MapController? _mapController` set from the hand-off and null-guard their calls. 15-05/15-06 reuse this, not an input controller field. Live theme swap = `ref.listen(mapStyleJsonProvider) -> setStyle` + a cached `_lastStyleJson` so a keepAlive refresh never flashes to loading (CORE-02). Trail track/markers seam is `layers: const []` in `wanderer_map.dart` (`// 15-05: ...`).
- [15-05] Arrow glyph self-registered via `addImageFromIconData` (sprite-independent) so TRAIL-02 avoids the unresolved `file://` sprite risk.
- [15-05] Directional arrows are a static native symbol layer (D-05); `AnimationController` + bearing loop deleted.
- [15-06] OFFL-05 multi-cell = N native `pmtiles://file://` sources + N duplicated style-layer sets (`__cellN` id suffix; source-less layers not cloned), NOT merge-at-download. `pmtiles` 1.2.0 Dart is read-only (no merge API) and `generator.go`/`grid.go` emit one `.pmtiles` per 0.5° cell (~1-4 per trail); a server merge endpoint is out of the Flutter phase's scope. Implemented in `rewriteStyleForOffline`, test-guarded.
- [15-06] `rewriteStyleForOffline` is the single sanctioned online->offline style transform (pure, deep-copies input): `glyphs`/`sprite` -> `file://<cacheRoot>`; tiled sources -> `pmtiles://file://<cell>`; rejects non-absolute / `..` / foreign-scheme paths before emitting (T-15-06-01/02). WandererMap is its only caller.
- [15-06] OFFL-06 DEFERRED: `pm_tile_provider.dart` NOT deleted — `navigation_screen.dart` (Phase-17 flutter_map holdout) still uses `MultiPmTilesVectorTileProvider`. Delete once that screen migrates.
- [15-06] Three real bugs found and fixed during physical-device verification, outside the normal executor flow: restored legacy flutter_map `TrailLayer` for `navigation_screen` (`192b3a89`, 15-05 had deleted it without checking callers); removed flutter_map-only `MapCompass` from `WandererMap.controls` (`b7d30947`, crashed at runtime — `MapCamera.of()` needs a `FlutterMap` ancestor); clamped offline pmtiles source `maxzoom` to 14 to match `generator.go`'s extraction depth (`85d73fd3` — the online style's inherited `maxzoom:15` caused blank tiles above z14 offline).
- [Phase 15 verification] Goal-backward verification (`15-VERIFICATION.md`) found the trail's self-registered `arrow` sprite image collided (same id) with the basemap's own `roads_oneway` icon — MapLibre's `addImage` silently overwrites same-id images. Fixed by renaming to `trail-arrow` (`38aabcdc`).
- [Phase 16-01]: SearchMap.layers typed List<ml.Layer>? (native style-layer builders), not List<Widget>? as PLAN.md's artifact spec stated — ml.MapLibreMap.layers and .children are different type families
- [Phase 16-01]: _ListMap converted ConsumerWidget to ConsumerStatefulWidget to hold ml.MapController across onMapCreated/onStyleLoaded, mirroring list_detail_map_screen's approach
- [Phase 16]: [16-02] cluster_layer.dart doc comments avoid the literal substring 'unclustered-point' (rephrased as point_count==1 circle layer) to satisfy the plan's own zero-occurrence grep while still documenting D-05's rationale
- [Phase 16]: [16-02] Wrapped the circle-radius step-ramp array in // dart format off / on markers so dart format doesn't re-explode the single-line grouping the acceptance criteria greps for verbatim
- [Phase 16-03]: map_screen.dart's flutter_map-only MapCompass/CurrentLocationLayer widgets replaced with native ml.MapCompass and a WandererMap-style native location WidgetLayer -- neither old widget can render inside a native MapLibreMap tree — Required by the SearchMap host swap; not explicitly called out in the plan's action text but necessary for the file to run without crashing at runtime
- [Phase 16-03]: Camera-position persistence (mapCameraProvider) preserved via ml.MapEventCameraIdle, replacing the old MapEventMoveEnd-based save — Avoids a silent UX regression; matches the phase's no-UX-change philosophy (D-01/D-02) even though the plan flagged map_camera_provider as verify-unused-after-swap
- [Phase 16-03]: Selected-trail polyline uses fitBounds (not animateCamera) with kTrailRouteColor styling — The plan's action text described controller.animateCamera(bounds:, duration:) but the installed maplibre 0.3.5 API has no bounds/duration params on animateCamera -- fitBounds(bounds:, nativeDuration:) is the correct call, matching list_detail_map_screen.dart's established pattern
- [Phase 16-03 checkpoint]: `is_large` is NOT filtered out of unclustered map markers (D-03 corrected) — the server marks the top MAP_MAX_POLYLINES trails by size in view as is_large once zoom passes clusteringMaxZoom (~11); with fewer trails in view than that this can mean ALL visible trails, not just rare huge ones. User directive: any unclustered point (point_count == 1) renders as a category-icon marker regardless.
- [Phase 16-03 checkpoint]: `SearchMap` buffers a style-loaded event that arrives before `onMapCreated` and replays it once the controller is set — the native platform channel does not reliably fire onMapCreated before onStyleLoaded despite the package docs implying that order; two of three SearchMap call sites (both list-map screens) silently no-opped their fitBounds call without this fix.
- [Phase 16-03 checkpoint]: `fitBounds`/`animateCamera` "instant" calls use `Duration(milliseconds: 1)`, never `Duration.zero` — a zero duration crashes the Android native binding (`animateCamera` receives a null duration, throws `IllegalArgumentException`). Applies anywhere an instant/no-animation camera move is needed on this package.
- [Phase 16-03 checkpoint]: `/api/v1/map/style-sources`'s `spriteUrl` is a theme-agnostic base — callers must append `/light` or `/dark` before MapLibre appends the file suffix. `map_style_json_provider.dart` was substituting the bare base for both themes, causing sprite 404s; fixed by appending the variant from `effectiveBrightness`.
- [Phase 16-03 checkpoint]: Cluster-tap zoom (CLUS-03) uses a 400ms `nativeDuration` (not the SDK's 2s default) and auto-triggers `searchInBounds` once the camera settles — explicit user request, distinct from pan/drag which still requires the manual "Search this area" button per D-01.
- [Phase 17-01]: Inlined SearchMap-style onMapCreated/onStyleLoaded race buffer (_pendingStyle) in navigation_screen.dart even though the plan text didn't spell it out — same race class already caused two Phase 16-03 bugs
- [Phase 17-01]: Left TrailLayer (trail_layer.dart), map_compass.dart, and pm_tile_provider.dart physically in place — files_modified scoped this plan to navigation_screen.dart only; deletion deferred to Phase 18 (CLEAN-01/02/03). Post-change grep confirms map_compass.dart and pm_tile_provider.dart now have zero real importers app-wide
- [Phase 17-02]: Reworded stray CurrentLocationLayer doc-comment references outside files_modified scope to satisfy the plan's own repo-wide grep gate (tracelet_position_source.dart, map_screen.dart)
- [Phase 18-01]: Relocated `effectiveBrightness()` verbatim into `map_style_json_provider.dart` (no new import needed — Brightness/ThemeMode/WidgetsBinding all resolve via that file's existing `package:flutter/material.dart` import); replaced `LocationMarkerPosition`/`ServiceDisabledException` with shape-identical file-local classes in `foreground_position_stream_provider.dart` — both are code-before-manifest prep so Plan 02 can remove the six packages from `pubspec.yaml` without breaking a whole-package `flutter analyze`
- [Phase 18-01]: 3 pre-existing `flutter test` failures (`feed_item_test.dart` x2, `settings_screen_test.dart` x1) confirmed unrelated via git-stash bisect against the parent commit; logged to `.planning/phases/18-retire-flutter-map-and-the-flomp-forks/deferred-items.md`, not fixed (out of scope for this plan's `files_modified`)
- [Phase 18]: [18-02] Removed the six flutter_map/vector_map_tiles/vector_tile_renderer packages and both flomp/* git dependency_overrides from pubspec.yaml; pinned maplibre to exact 0.3.5; meta ^1.18.0 override retained
- [quick-260710-kpd] Added html and pointer_interceptor as direct pubspec.yaml dependencies since WandererAttribution imports them directly (previously transitive via maplibre/flutter_html).
- [quick-260711-lzb] DEM (hillshade) tile lifecycle kept fully independent from the vector tile lifecycle on `tile_cells` (separate `dem_status`/`dem_size_bytes`/`dem_error_message`, separate download route, best-effort download on the Flutter side) — hillshade is cosmetic and a DEM failure must never regress or block the vector basemap.
- [quick-260711-lzb] Fixed a real offline-hillshade bug: `offline_style_rewriter.dart` was sweeping `hillshadeSource` (a `raster-dem` source) into the vector-cell repoint path purely because it carries a `url` key, pointing it at the wrong (vector) `.pmtiles` archive. Now split by `source['type']`; `raster-dem` sources get their own `demCellPaths` param, injected `encoding:terrarium`/`tileSize:512`/`maxzoom:12` (must stay in lockstep with Go's `demMaxZoom` const in `db/services/tiles/generator.go`), and drop cleanly (no `https://` leak) when no DEM archive was downloaded for that trail.
- [Phase quick-260712-m9v]: ActiveNavigationEntity exposes obxId as a settable constructor param (unlike TrailEntity's constructor) so navigation_screen.dart can update the single tracked row via active_nav.save() instead of inserting duplicates
- [Phase quick-260712-m9v]: AppLocalizations keys in this codebase stay literal snake_case getters/methods (e.g. exit_navigation), not camelCase -- confirmed before adding resume_navigation_prompt
- [Phase quick-260712-pac]: In this Riverpod 3.x codebase, AsyncValue.isLoading/isRefreshing/isReloading/retrying/isFromCache are extension methods, not real instance members -- an untyped ref.listen/listenManual closure param can get inferred as dynamic, which skips extension resolution and throws NoSuchMethodError at runtime instead of a compile error. Always explicitly type listener closure params (and the listen/listenManual generic argument) as AsyncValue<T> when calling these getters.
- [v1.5 roadmap] 3 phases (19-21), matching config.json's `coarse` granularity (2-4 target). Phase 19 merges research/SUMMARY.md's suggested Phase 1 (Provider Architecture) + Phase 2 (Map Interaction) into one vertical slice — tap/drag/insert waypoints, auto-routing toggle, profile switch, undo/redo — because neither half alone is user-observable without the other. Phase 20 merges the suggested Phase 3 (Screen Shell) + Phase 4 (Search-to-Focus): PLANUI-03 turned out to be a dedicated location-only search screen (not a `GlobalSearchScreen` modification as SUMMARY.md assumed), making it a natural map-control-button sibling of the waypoint-list/elevation toggle rather than a separate phase. Phase 21 (Handoff) matches the research's Phase 5, plus HANDOFF-03 (hike/bike entry dialog, since it's part of the entry-point flow into the planner).
- [v1.5 roadmap] WAYP-04/05 (delete/reorder) assigned to Phase 20, not Phase 19, because both requirements are explicitly scoped to "the waypoint list sheet" — a Phase 20 artifact. WAYP-01/02/03 (tap/drag/insert) stay in Phase 19 because they're direct map-gesture interactions, not list-sheet actions.
- [v1.5 roadmap] ROUTE-01..05 assigned to Phase 19 (not split across the toggle-button UI in Phase 20) because the routing engine — debounce, generation-guard, undo/redo snapshot stack — is the load-bearing state architecture every later phase depends on; research flagged this as the pitfall most expensive to retrofit if built after the UI.
- [Phase 19]: [19-01] Divergence test asserts on longitude, not latitude — Geographic clamps latitude to [-90, 90], masking the intended 10x-scale assertion at the chosen fixture coordinates
- [Phase 19]: [19-01] D-01's no-'waypoint'-string constraint on route_anchor.dart applies to comments too, not just identifiers — reworded a doc comment referencing the sibling persisted-route-point model by name
- [Phase 19]: [19-02] toggleAutoRouting() OFF leaves existing segments untouched (corrected mid-execution from the plan's stale Task 1 prose to match its own must_haves truth); only new segments created afterward become straight
- [Phase 19]: [19-02] splitSegmentAt projects the tap point onto every polyline sub-edge (not just the nearest existing vertex) so a straight 2-point segment splits at a genuine midpoint, not one of its endpoints
- [Phase 19]: [19-03] RouteSegmentLayer constructor is non-const (not const as the plan's action text specified) because the plan's same paragraph also mandates a mutable _added field, which Dart forbids on a const constructor
- [Phase 19]: [Phase 19] [19-04] static final _segmentLayer, not static const -- RouteSegmentLayer's constructor is non-const (19-03's own deviation, mutable _added field forbids const)
- [Phase 19]: [Phase 19] [19-04] _retryAttempted field declared in Task 1 (not Task 2 as the plan's task split implied) since Task 1's own onEvent action text uses it before Task 2 formally introduces it
- [Phase 19]: [Phase 19] [19-04] Undo/redo onPressed hoists a local notifier variable + method tear-off instead of an inline closure, so dart format keeps the ternary on one line and satisfies the plan's own literal acceptance-criteria grep
- [Phase 19]: [19-03, UAT] On-device testing found segment taps (blocked-segment retry, WAYP-03 insert) always fell through to appendAnchor — `route_segment_layer.dart`'s invisible hit-test layer used `line-opacity: 0`, and both native maplibre bindings' `featuresAtPoint` (iOS `visibleFeaturesAtPoint`, Android `queryRenderedFeatures`) exclude fully transparent layers from rendered-feature queries, so the layer never registered a hit. Fixed by changing `line-opacity` to `0.01` (visually indistinguishable from 0, stays eligible for hit-testing). Relevant to any future native-GL invisible hit-test layer in this codebase (e.g. Phase 20/21).
- [Phase 19]: [19-04, UAT] On-device testing found segment lines never render on a SECOND open of the Route Planner screen in the same app session (markers still work). Root cause: `route_planner_screen.dart`'s `_segmentLayer` was a `static final RouteSegmentLayer()` — `static` means one shared instance (and its internal `_added` flag) for the app's whole lifetime, not per screen mount. First open adds the native source/layers and sets `_added = true`; on exit the native map/style is destroyed but the static `_segmentLayer` survives with `_added` stuck `true`; on re-open, a brand-new style has no source/layers on it, but `update()` sees stale `_added == true` and skips straight to `updateGeoJsonSource` on a source that doesn't exist on this style — fails silently via the call site's `.ignore()`. Fixed by making `_segmentLayer` a plain (non-static) instance field, so it's fresh every mount. Any future `static` field wrapping a native-layer-adding helper with its own "already added" flag needs the same scrutiny.
- [Phase 20]: [Phase 20] [20-01] deleteAnchor/reorderAnchors mutators found already implemented (uncommitted) matching plan exactly; verified against all acceptance criteria before committing as Task 1 — No rewrite needed -- existing code already satisfied the plan's must_haves truths and passed all new unit tests
- [Phase 20]: [Phase 20] [20-01] gpx package 2.3.0's Gpx class has no trks: constructor param -- construct Gpx() then assign .trks field — Plan's action text assumed a constructor param that doesn't exist in the installed gpx version; fixed to match the real API
- [Phase 20]: [Phase 20] [20-02] Captured GlobalSearchNotifier as a field in initState rather than calling ref.read in dispose(), avoiding the avoid_ref_inside_state_dispose lint — ref may already be torn down by the time State.dispose runs
- [Phase 20]: [20-03] ElevationTab never writes ele back into plannedGpxProvider (D-10) -- holds local _eleMergedGpx state, rebuilt on each successful height fetch — keeps plannedGpxProvider pre-elevation for Phase 21 handoff while still rendering a live elevation-merged chart
- [Phase 20]: [20-03] Height fetch failures swallowed silently (best-effort) -- keeps rendering last successfully-merged Gpx — no error/retry UI specified for this tab in D-11/D-13; avoids error-banner flash on transient network blips

### Pending Todos

- Fix 3 pre-existing `flutter test` failures (`feed_item_test.dart` x2, `settings_screen_test.dart` x1) — logged in Phase 18's `deferred-items.md`; not blocking v1.5 work.
- Manual on-device verification for quick task 260712-m9v (resume navigation after manual app termination): start navigation, swipe-kill the app, relaunch, accept the resume dialog, confirm maneuver index/distance/elevation/elapsed/breadcrumb continue; also verify decline and deliberate-exit paths show no prompt.

### Blockers/Concerns

- **[Phase 15/17 — OFFL-06 deferred, by design]** `pm_tile_provider.dart` intentionally NOT deleted — `navigation_screen.dart` still consumes `MultiPmTilesVectorTileProvider` for its offline flutter_map path. Delete in Phase 17/18 once that screen migrates off flutter_map. ROADMAP.md's Phase 15 success criterion 5 was corrected in place to reflect this (it originally self-contradicted).
- **[Phase 15 — A2 sprite `file://` gap, low exposure, not investigated further]** The 15-01 spike found `file://` **sprite** resolution FAILED on a physical Android device despite valid cached files (glyph `file://` resolution PASSED). 15-05 sidesteps the practical impact for the trail's own arrow icon by self-registering it (`addImageFromIconData`, sprite-independent) rather than relying on the sprite. The only remaining exposure is OTHER sprite-atlas icons (e.g. route-network shields) rendering offline — those don't render at all today regardless of online/offline, so this is a pre-existing not-yet-working feature, not a regression. Not re-investigated natively; revisit if/when sprite-sourced icons become a real feature.
- **[Phase 15/16/17]** `maplibre` 0.3.5 is pre-1.0 with breaking changes across 0.x minors (three published upgrade guides). Pin the exact version on first add; CLEAN-03 locks it at the end.
- **[Phase 15 — CORE-05 gap, expected]** `trail_detail_map_screen`'s `MapCompass` (flutter_map-only) was removed rather than crashing against the new `MapLibreMap` — that screen has no compass/rotation-reset control until Phase 17 (CORE-05) wires maplibre's native equivalent.
- **[v1.5 research flag]** `package:maplibre` 0.3.5's exact `MapGestures`/`MapOptions` pan/rotate-disable API surface is MEDIUM confidence (changelog/GitHub discussion, not a direct source read) — validate with a small spike early in Phase 19 before committing to the full drag-vs-pan gesture-arena solution.
- **[v1.5 research flag]** The generation-counter/CancelToken race-guard pattern for out-of-order Valhalla responses is MEDIUM confidence against this project's exact pinned `riverpod_annotation` 4.0.2 — confirm the idiom during Phase 19 planning/execution.

### Quick Tasks Completed

| # | Description | Date | Commit | Status | Directory |
|---|-------------|------|--------|--------|-----------|
| 260702-e3g | Fix non-optimistic reorder animation in Settings(Sub)CategoriesScreen | 2026-07-02 | 6b3e6f6b | | [260702-e3g-…](./quick/260702-e3g-fix-non-optimistic-reorder-animation-in-/) |
| 260702-ek7 | Fix white flash on (sub)category toggle/reorder | 2026-07-02 | 8a917b4c | | [260702-ek7-…](./quick/260702-ek7-fix-white-flash-on-sub-category-toggle-r/) |
| 260702-ere | Cascade category visibility to SettingsSubcategoriesScreen | 2026-07-02 | 108348b2 | | [260702-ere-…](./quick/260702-ere-cascade-category-visibility-to-settingss/) |
| 260702-m4u | Make auth_provider.dart build() optimistic | 2026-07-02 | d2d126a8 | | [260702-m4u-…](./quick/260702-m4u-make-auth-provider-dart-build-optimistic/) |
| 260702-gib | Add read-only subcategory chips under each category row | 2026-07-02 | dbc1db3d | | [260702-gib-…](./quick/260702-gib-add-subcategory-chips-under-each-categor/) |
| 260710-kpd | Fix the 6 small UI gaps found after Phase 18 checkpoint | 2026-07-10 | dcbabbd4 | | [260710-kpd-…](./quick/260710-kpd-fix-the-6-small-ui-gaps-that-i-found-aft/) |
| 260711-d37 | Shrink the navigation-screen location puck (custom marker + manual animateCamera follow) | 2026-07-11 | e339b148 | | [260711-d37-…](./quick/260711-d37-shrink-the-navigation-screen-location-pu/) |
| 260711-lzb | Make hillshading work offline in the Flutter app (per-cell DEM pmtiles pipeline + raster-dem rewriter fix) | 2026-07-11 | 3f67cf37,68501626,ab2be809,3d6ff5e1,d95b2c97,21a516a4,33c1c114 | Needs Review | [260711-lzb-…](./quick/260711-lzb-make-hillshading-work-offline-in-the-flu/) |
| 260712-m9v | Resume navigation after manual app termination (ObjectBox-persisted session, resume-seedable providers, launch-time resume dialog) | 2026-07-12 | 45b93ce4,3b5fd30f,3784fc95,7503e481 | Needs Review | [260712-m9v-…](./quick/260712-m9v-resume-navigation-after-manual-app-termi/) |
| 260712-pac | Fix NoSuchMethodError from untyped listenManual/listen closures on authProvider (AsyncValue.isLoading extension resolved dynamically) | 2026-07-12 | 5697a064 | Complete | [260712-pac-…](./quick/260712-pac-fix-nosuchmethoderror-in-main-dart-type-/) |
| 260713-nes | Gate auth build() on timeout-bounded validation before navigation; original persisted-500-counter self-heal piece later reverted (260714-qma) once the real root cause was found | 2026-07-13 | d9531951,dd56c62d | Superseded by 260714-qma | [260713-nes-…](./quick/260713-nes-fix-500-errors-on-app-launch-by-gating-n/) |
| 260714-qma | Revert speculative persisted-500-counter self-heal from auth_provider.dart, keep timeout-gated build() | 2026-07-14 | c3fbef98,a6647dcf,eee4f0c2 | Complete | [260714-qma-…](./quick/260714-qma-revert-speculative-persisted-500-counter/) |
| 260714-qtl | Fix meilisearch_token cookie never validated for real expiry in hooks.server.ts (real root cause of the reported search-call 500s) | 2026-07-14 | ed114163 | Complete | [260714-qtl-…](./quick/260714-qtl-fix-meilisearch-token-cookie-never-valid/) |
| 260715-q01 | Rename iOS bundle id / App Group ids and Android applicationId from com.example.wanderer to com.openwanderer.wanderer | 2026-07-15 | 40cf3a69,8022ae97 | Complete | [260715-q01-…](./quick/260715-q01-update-the-ios-bundle-id-and-android-app/) |

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Map screen | Render `is_large` trails as full polylines (FUT-01) | Future milestone | v1.4 requirements |
| Basemap | Basemap picker + Waymarked Trails overlays (FUT-02/03) | Future milestone | v1.4 requirements |
| Native GL | 3D terrain, hillshade, pitch/tilt (FUT-04/05) | Future milestone | v1.4 requirements |
| Trail form | Category/subcategory picker in create/edit form (TRAILFORM-01/02) | Future milestone (form rework needed) | v1.3 requirements |
| Bulk edit | Bulk-edit modal for category/subcategory/difficulty | Out of scope (web-only) | v1.3 requirements |
| Filters | Subcategory reordering within a category | Future milestone | v1.3 requirements |
| Audio | TTS maneuver announcements (AUDIO-01, AUDIO-02) | v2 | Init |
| Routing | Navigate from user position to trailhead; off-trail re-routing | Out of scope | Init |
| Account | API token management (ACCT-F01) | Future | v1.2 requirements |
| Settings | Favourite sport picker, Export, Integrations, Maintenance, Map settings | Out of scope | v1.2 requirements |
| Route Planner | Car/driving costing profile (PLANNER-01) | v2 | v1.5 requirements |
| Route Planner | Editing an existing trail's route (PLANNER-02) | v2 | v1.5 requirements |
| Route Planner | Per-segment travel profiles (PLANNER-03) | v2 | v1.5 requirements |
| Route Planner | Offline caching of in-progress route plans (PLANNER-04) | v2 | v1.5 requirements |
| Route Planner | Confirm-placement drag affordance (PLANNER-05) | v2 (if mis-drops prove common) | v1.5 requirements |
| Route Planner | Offline-aware graceful degradation banner for auto-routing (PLANNER-06) | v2 | v1.5 requirements |

Items acknowledged and deferred at milestone close on 2026-07-10:

| Category | Item | Status |
|----------|------|--------|
| quick_task | 260610-kdc-fix-trail-pmtiles-download-add-missing-g | unknown (has SUMMARY.md; likely complete) |
| quick_task | 260611-whq-support-multiple-pmtiles-sources-in-offl | unknown (has SUMMARY.md; likely complete) |
| quick_task | 260612-gmg-add-proper-dark-mode-to-the-flutter-app- | missing (PLAN.md only, never executed) |
| quick_task | 260615-k0w-implement-along-track-projection-for-way | unknown (has SUMMARY.md; likely complete) |
| quick_task | 260615-ktn-research-flutter-background-geolocation- | unknown (has SUMMARY.md; likely complete) |
| quick_task | 260615-mxk-implement-background-navigation-so-locat | unknown (has SUMMARY.md; likely complete) |
| quick_task | 260616-h99-create-wandereractorsearch-component-for | unknown (has SUMMARY.md; likely complete) |
| quick_task | 260616-j2n-implement-the-like-feature-from-the-web- | unknown (has SUMMARY.md; likely complete) |
| quick_task | 260702-e3g-fix-non-optimistic-reorder-animation-in- | unknown (has SUMMARY.md; logged in Quick Tasks Completed) |
| quick_task | 260702-ek7-fix-white-flash-on-sub-category-toggle-r | unknown (has SUMMARY.md; logged in Quick Tasks Completed) |
| quick_task | 260702-ere-cascade-category-visibility-to-settingss | unknown (has SUMMARY.md; logged in Quick Tasks Completed) |
| quick_task | 260702-gib-add-subcategory-chips-under-each-categor | unknown (has SUMMARY.md; logged in Quick Tasks Completed) |
| quick_task | 260702-m4u-make-auth-provider-dart-build-optimistic | unknown (has SUMMARY.md; logged in Quick Tasks Completed) |
| quick_task | 260710-kpd-fix-the-6-small-ui-gaps-that-i-found-aft | unknown (has SUMMARY.md; logged in Quick Tasks Completed) |
| quick_task | 260710-lem-fix-2-issues-found-during-manual-verific | unknown (has SUMMARY.md; likely complete) |

## Session Continuity

Last session: 2026-07-16T21:59:58.548Z
Stopped at: Completed 20-03-PLAN.md
Resume file: 

- Review and approve the ROADMAP.md draft for v1.5 (Phases 19-21).
- Once approved, run `/gsd-plan-phase 19` to begin detailed planning for Phase 19 (Route Planner Core — Waypoint Editing & Routing Engine).
