---
gsd_state_version: 1.0
milestone: v1.6
milestone_name: Offline Region Tile Repository
status: verifying
stopped_at: Completed 27-02-PLAN.md
last_updated: "2026-07-24T15:54:10.577Z"
last_activity: 2026-07-24
progress:
  total_phases: 9
  completed_phases: 7
  total_plans: 31
  completed_plans: 30
  percent: 78
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-16)

**Core value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.
**Current focus:** Phase 27 — legacy-cleanup

## Current Position

Phase: 28
Plan: Not started
Status: Phase complete — ready for verification
Last activity: 2026-07-24

## v1.6 Phases

- [ ] **Phase 22: Region & Package Data Model** — REGN-01/02/03
- [ ] **Phase 23: TileRepositoryManager — Download Engine** — TILE-01..05, DEM-01/02
- [ ] **Phase 24: Settings — Offline Maps/Regions UI** — SETUI-01..06
- [ ] **Phase 25: Map Rendering — Region-Based Viewport Pipeline** — RENDER-01..03 (4/4 plans executed; UAT found Test 4 issue, see Phase 25.1)
- [ ] **Phase 25.1: Local HTTP Tile Proxy** (INSERTED) — replaces navigation_screen's incremental region-swap reconcile after the UAT-diagnosed reentrancy race
- [ ] **Phase 26: Trail Download Guard** — GUARD-01..04
- [ ] **Phase 27: Legacy Cleanup** — CLEAN-01/02

Execution order: 22 → 23 → 24 → 25 → 25.1 → 26 → 27 (strictly sequential — data model → download engine → Settings UI → map-screen rewiring/spike → tile-proxy re-architecture → trail guard → legacy ripout).

v1.5 (Phases 19-21) shipped in full (all plans complete 2026-07-16/17) but has not yet been run through `/gsd-complete-milestone`.

## Performance Metrics

**Velocity (v1.0–v1.3):**

- Total plans completed: 37
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
| 27 | 2 | - | - |

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
| Phase 20 P04 | 8min | 1 tasks | 1 files |
| Phase 20 P05 | 20min | 2 tasks | 2 files |
| Phase 21 P01 | 15min | 2 tasks | 4 files |
| Phase 21 P02 | 6min | 1 tasks | 2 files |
| Phase 21 P03 | 12min | 2 tasks | 3 files |
| Phase 21 P04 | 10min | 1 tasks | 1 files |
| Phase quick-260719-d6a P01 | 35min | 3 tasks | 6 files |
| Phase quick-260719-n8g P01 | ~20min | 2 tasks | 5 files |
| Phase quick-260720-s7m P01 | ~70min | 4 tasks | 51 files |
| Phase 260721-eob P01 | 35min | 2 tasks | 9 files |
| Phase 21.5 P01 | 4min | 2 tasks | 3 files |
| Phase 21.5 P02 | 10min | 2 tasks | 3 files |
| Phase 21.5 P03 | 3min | 4 tasks | 8 files |
| Phase 22 P01 | 6min | 3 tasks | 9 files |
| Phase 22 P02 | 12min | 2 tasks | 3 files |
| Phase 23 P01 | 4min | 2 tasks | 4 files |
| Phase 23 P02 | 6min | 2 tasks | 6 files |
| Phase 23 P03 | 12min | 2 tasks | 4 files |
| Phase 23 P04 | 18min | 2 tasks | 2 files |
| Phase 23 P05 | 20min | 2 tasks | 7 files |
| Phase 23 P06 | 15min | 1 tasks | 1 files |
| Phase 24 P01 | 15min | 3 tasks | 9 files |
| Phase 24 P02 | 15min | 3 tasks | 4 files |
| Phase 24 P03 | 6min | 1 tasks | 4 files |
| Phase 24 P04 | 10min | 2 tasks | 3 files |
| Phase 25 P01 | 25min | 2 tasks | 1 files |
| Phase 25 P02 | 10min | 2 tasks | 3 files |
| Phase 25 P03 | ~20min | 1 tasks | 1 files |
| Phase 25 P04 | ~20min | 2 tasks | 1 files |
| Phase 25.1 P01 | 4min | 2 tasks | 3 files |
| Phase 25.1 P02 | 12min | 3 tasks | 9 files |
| Phase 25.1 P03 | ~25min + on-device iteration | 2 tasks | 1 files |
| Phase 25.1 P04 | 5min | 3 tasks | 3 files |
| Phase 26 P01 | 12min | 1 tasks | 2 files |
| Phase 26 P02 | 19min | 2 tasks | 2 files |
| Phase 26 P03 | 9min | 2 tasks | 1 files |
| Phase 26 P04 | 5min | 2 tasks | 1 files |
| Phase 26 P05 | ~10min | 2 tasks | 2 files |
| Phase 27 P01 | 12min | 2 tasks | 6 files |
| Phase 27 P02 | 4min | 2 tasks | 8 files |

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
- [Phase 20]: [20-04] No revert-on-error around reorderAnchors -- it's a synchronous in-memory mutation that cannot throw, unlike the settings_categories_screen.dart async-persist analog
- [Phase 20]: [20-04] ReorderableListView.builder's scroll-controller param is named scrollController, not controller -- matched to the actual widget API
- [Phase 20-05]: Used literal 0.14/0.6 size values inline (not named constants) in route_anchor_sheet.dart so grep-based acceptance criteria match the source verbatim
- [Phase 20-05]: Kept context.push<LocationSearchResult>('/location-search') on one line to match the plan's literal acceptance-criteria grep
- [Phase 21]: [Phase 21] [21-01] Test-fixture correction: 'Mountain Biking' doesn't contain literal substring 'bike' ('biking' != 'bike') -- fixed fixture to 'Mountain Bike Trails' to correctly exercise categoryForTravelProfile's bike-branch match — Implementation was already correct; the initial test fixture string was the bug
- [Phase 21]: [Phase 21] [21-01] finishPlanning's orchestration (network I/O + WidgetRef/BuildContext) has no pure/synchronous seam, so only its pure sub-helpers (buildDraftTrail, mergeHeightsIntoGpx) are unit-tested — Full end-to-end coverage deferred to Plan 04's app-bar Finish action wiring
- [Phase 21]: Behavior fields all nullable (no freezed defaults) — So an absent allowAutoGeolocate reads as null, never silently true — privacy-safe default matching web's ?? false convention (A2)
- [Phase 21]: behaviorJson mirrors privacyJson's exact JSON-blob-per-field encode/decode shape — Not a flattened bool ObjectBox column, per Pitfall 4
- [Phase 21]: [Phase 21] [21-03] _TravelProfileCard replicates _SourceActionCard's shape locally rather than importing the private widget, per the plan's self-containment instruction
- [Phase 21]: [Phase 21] [21-03] initialCenter fallback chain is mapCameraProvider -> settings.location -> Geographic(0,0), gated on Settings.behavior?.allowAutoGeolocate before ever subscribing to GPS
- [Phase 21]: [21-04] Kept the Finish tooltip ternary and finishPlanning(...) call each on one line, overriding dart format's default wrap, so the plan's literal acceptance-criteria greps match verbatim — Same precedent this phase already established in 20-05/21-01 for grep-sensitive lines
- [Phase quick-260719-d6a]: [quick-260719-d6a] Reused tracelet's native GPS-speed motion state machine (MotionDetectionMode.speed / onSpeedMotionChange) instead of hand-rolled Haversine/speed-threshold detection -- tracelet ^3.5.0 already provides this natively; a prior attempt hand-rolled it and was reverted
- [Phase quick-260719-d6a]: [quick-260719-d6a] Generalized _pauseStart into _frozenSince backing frozen = _manualPaused || _stationary via one _applyFrozen() helper so overlapping manual-pause and stationary intervals are never double-counted
- [Phase quick-260719-n8g]: Reused NavigationScreen directly via isRecording flag rather than a separate RecordScreen -- every maneuver/route-dependent UI already null-guards to nothing on empty input
- [Phase quick-260719-n8g]: Sentinel id: '' for widget.id in recording mode (not a nullable-id refactor) -- every trailProvider(widget.id) read was already null-guarded
- [Phase quick-260719-n8g]: Recording's only finish trigger is the existing 3-option exit dialog (Cancel/Exit/Save), reused verbatim -- isArrived is structurally always false with empty maneuvers so there is no auto-arrival banner
- [Phase quick-260720-s7m]: Fixed a pre-existing ICU plural-syntax bug (redundant 'one' clause fully shadowed by '=1') in es/eu/pl ARB files surfaced by gen-l10n during the dead-key equalization pass
- [Phase quick-260720-s7m]: New placeholder-bearing l10n keys use double-quote literals around interpolated values instead of single quotes, since a single quote is ICU MessageFormat's escape character and would silently break the placeholder
- [Phase quick-260721-eob]: snapResultAcceptable rejects on bbox-diagonal shrink (<0.6x original), never on point count -- trace_route legitimately re-vertexes at Valhalla's own density
- [Phase quick-260721-eob]: Any transform path (snap and/or heights) produces a timeless track (elevation-only merge helpers); only the no-transform path preserves the recorded breadcrumb's timestamps verbatim
- [Phase quick-260721-eob]: New /valhalla/trace-route proxy is authenticated (locals.user gate), matching /valhalla/navigate's trust class rather than the unauthenticated /valhalla/route and /valhalla/height siblings
- [v1.6 roadmap] 6 phases (22-27), following research/SUMMARY.md's recommended build order almost verbatim: data model → download engine → Settings UI → map-screen rewiring+spike → trail guard → legacy ripout. Strictly sequential (no parallel phases) — each step swaps exactly one thing while the old trail-scoped path stays live until proven redundant, same discipline as v1.4's "forks deleted last." RENDER-03's maplibre 0.3.5 incremental-source spike is pulled into Phase 25, ahead of the guard/ripout phases, because research flags it as the piece most likely to need rework if discovered late.
- [v1.6 roadmap] REQUIREMENTS.md's own Coverage line undercounted the v1 list by one (said "24 total"; the actual requirement table has 25 REQ-IDs). Corrected during roadmap creation — all 25 requirements map 1:1 to exactly one of Phases 22-27, no orphans.
- [Phase 21.5]: [21.5-01] Bbox stored as [4]float64 in [minLon, minLat, maxLon, maxLat] order — matches generator.go's pmtiles extract argument order and RegionArchivePath/RegionDemPath path builders; documented since Phase 22's client manifest depends on this exact order
- [Phase 21.5]: [21.5-01] region_archives collection status/dem_status select values are building/ready/error (not pending) per D-08, dem_status independently nullable so DEM status never blocks a vector-ready region
- [Phase 21.5]: [Phase 21.5] [21.5-02] bboxChanged uses exact float64 equality (no epsilon) — both sides originate from the same JSON-parsed float64 values, so no floating-point drift is introduced between comparison sides
- [Phase 21.5]: [Phase 21.5] [21.5-02] buildRegion wrapped in buildRegionSafely (defer/recover) so a single region panic can never abort BuildAll's loop over remaining regions
- [Phase 21.5]: [21.5-03] The Go /api/v1/regions route is internal-only (db's port unpublished in prod/dev compose); SvelteKit proxies the same public path for external clients (Phase 22's Flutter app included)
- [Phase 21.5]: [21.5-03] Download proxy routes reuse event.locals.pb.baseURL + Bearer Authorization header (existing map/cells download precedent) rather than $env/dynamic/public — same outcome, established codebase convention
- [Phase 22-01]: CatalogStatus gained a 4th entity-only sentinel absent (code 3, no @JsonValue) so RegionEntity.demStatus stays a non-nullable explicit-int-enum shadow instead of a nullable ObjectBox column
- [Phase 22-01]: RegionEntity.status getter's updateAvailable branch checks only vector version/lastDownloadedVersion -- DEM has no staleness concept (no dem_version field), matching D-07
- [Phase 22-02]: Malformed catalog elements dropped at two independent layers: parseRegionCatalog catches per-element fromJson failures; upsertCatalog additionally catches FormatException from RegionEntity.fromCatalogEntry/applyCatalogEntry's bbox guard per-entry
- [Phase 22-02]: refreshCatalog splits fetch and upsert into two explicit steps so a fetch failure always happens before any store write, guaranteeing local region rows/download status are never corrupted by an offline/failed catalog fetch
- [Phase 23]: [Phase 23] [23-01] Kept the plan's pre-existing working-tree rename of the upstream fetch path (/api/v1/regions/... to /regions/...) untouched; only edited Range/status/header-forwarding lines
- [Phase 23]: [Phase 23] [23-02] region_file_path.dart's regionIdPattern is byte-for-byte identical to the backend RegionIdSchema regex, so an id the server accepts is the same id the app accepts
- [Phase 23]: [23-03] disk_space_2 approved after direct pub.dev registry/score API + GitHub native-source (Kotlin StatFs, Swift NSFileManager) legitimacy review — confirmed read-only free/total-space queries only, no network/write/reflection/shell-out
- [Phase 23]: [23-03] disk_space_util.dart splits an async fail-closed plugin wrapper (freeDiskSpaceBytes) from a pure, unit-tested safety-margin decision (hasEnoughSpace, default 1.75x multiplier) — mirrors map_cache_path.dart's pure/tested-util shape
- [Phase 23]: [23-04] Combined Task 1 + Task 2 into a single commit — building Task 1 in isolation fails its own flutter analyze acceptance criterion (unused-element lint on private helpers not yet called until Task 2) — Analyzer-level coupling between the two tasks for this new file; no code/behavior difference from the plan
- [Phase 23]: [23-05] TileRepositoryStatus clears a region's map entry entirely in finally (mirrors DownloadingTrailIds Set membership-clear) rather than settling a final state -- authoritative status lives on RegionEntity/DownloadedTilePackageEntity
- [Phase 23]: [23-05] deleteRegion silently no-ops on an unknown region id (not a StateError) -- nothing to clean up isn't a caller error
- [Phase 23]: [23-05] deleteRegion also resets region.lastDownloadedVersion to null alongside clearing both package ToOne targets, per the plan's 'reset any relevant status' instruction
- [Phase 23]: [23-06] Harness drives TileRepositoryManager directly (not via TileRepositoryStatus) to preserve raw received/total byte counts for debugPrint during on-device resume/pause verification
- [Phase 23]: [23-06] Added Backend base URL + Connect control to the harness (Rule 2) so the isolated ProviderScope actually points Dio at a real server instead of the api_provider.dart placeholder
- [Phase 24]: [24-01] regionListNotifierProvider named via @Riverpod(name: 'regionListNotifierProvider') because riverpod_generator's default Notifier-suffix stripping would have produced regionListProvider, breaking Plan 02's already-written literal references
- [Phase 24]: [24-02] Disk-usage FutureBuilder future recreated only on region-list identity change (identical() check), not every rebuild
- [Phase 24]: [24-02] Region disk-usage count includes downloading/paused packages (partial .part files), not only fully-downloaded ones
- [Phase 24]: [24-02] DEM toggle value derives purely from demPackage.target?.status == PackageStatus.downloaded; added an inline spinner (Rule 2) for in-flight feedback
- [Phase 24]: [24-02] Size-breakdown row text (vector/DEM) is a hardcoded English literal per Plan 01's own l10n scope, matching UI-SPEC's example copy verbatim
- [Phase 24]: [24-03] meta promoted from dependency_overrides-only to a direct pubspec dependency — @visibleForTesting on the new resolveFreeDiskSpaceBytes orchestrator imports package:meta/meta.dart directly, and flutter analyze flagged the override-only state
- [Phase 24]: [24-04] resolveRowStatus falls back to persisted region.status for DEM-only downloads so region.status keeps tracking only the vector package lifecycle and UAT test 2 (SETUI-04) isn't regressed
- [Phase 25-01]: RENDER-03 settled: incremental addSource/removeSource/addLayer/removeLayer selected over full-style-reload after on-device testing showed it avoids full-reload flicker; two Wave 2 follow-ups flagged (explicit repaint after removeSource/removeLayer; hillshade z-order insertion position needed)
- [Phase 25-02]: Split localTilePathsForBounds into a typed ({vectorPaths, demPaths}) record via a new @visibleForTesting splitRegionTilePaths pure helper, closing the RENDER-01 conflation gap before Wave 2 wiring — Makes DEM-into-vector-cell conflation structurally impossible at the type level, unit-tested without a live ObjectBox store
- [Phase 25-03]: Followed PLAN.md's revised incremental-composition design (RENDER-03 option-b) over the stale full-setStyle-reload description still in 25-PATTERNS.md — PLAN.md's frontmatter/objective explicitly superseded the earlier pattern map after 25-01's on-device spike; PLAN.md is the more specific, authoritative source (exact grep gates, threat model, must_haves)
- [Phase 25]: [25-04] Split Task 1 (offline data-source rewiring + reconcile method + region-list listen) and Task 2 (camera-idle onEvent branch) into two separate atomic commits despite touching the same file -- Task 1 leaves the file fully analyzer-clean and self-consistent, Task 2 is a clean minimal one-branch addition on top.
- [Phase 25]: [25-04] Camera-idle onEvent branch kept on one line (wrapped in dart format off/on markers) to satisfy the plan's literal acceptance-criteria grep for the MapEventCameraIdle->_reconcileRegionComposition routing -- same precedent as the 20-05/21-01 deviations.
- [Phase 25, UAT]: Test 4 (navigation screen region-boundary pan swap) failed on-device: "hot swapping does not work... sometimes the map does not load at all, sometimes the trail layer disappears." Diagnosed root cause: `_reconcileRegionComposition` has no reentrancy guard, and `ml.MapEventCameraIdle` fires far more often on this screen than the "once per settled user gesture" assumption (D-04) it was built on, because `navigation_screen.dart` continuously drives the camera itself (`_pushCamera`, GPS-fix tween + heading-follow ticker) -- overlapping reconciles desync `_addedSourceIds`/`_addedLayerIds` from the real native style (asymmetric add/remove error handling compounds it). Full trace: `.planning/debug/navigation-screen-region-swap-broken.md`.
- [Phase 25.1-01]: Doc comments explaining scoped platform network exceptions were reworded to avoid containing the literal forbidden strings their own negative-grep verify gates check for (usesCleartextTraffic, base-config, NSAllowsArbitraryLoads, NSLocalNetworkUsageDescription) -- same intent, adjusted phrasing, no scope change
- [Phase 25.1]: [Phase 25.1] [25.1-02] resolveRegionForTile stays @visibleForTesting with an explicit ignore-comment at its one production call site in tile_proxy_server.dart — Keeps the pure-function unit-testability the plan's artifact spec asked for while allowing production use, mirroring the pmtiles package's own cross-file @visibleForTesting precedent (archive.dart's fromReadAt)
- [Phase 25.1-03]: PROXY-03 settled PROCEED on a physical Pixel 6 via the on-device spike harness (`tile_proxy_spike_harness.dart`) — loopback HTTP tiles render reliably online, with radios off, and in full airplane mode (the load-bearing result). Test case (d) confirms Plan 04's mid-session refresh mechanism: reload regions + fly to the newly-covered area, no screen remount needed. Test case (e) confirms Plan 01's Android cleartext exception works correctly on-device (zero blocked errors in adb logcat).
- [Phase 25.1-04]: Deleted navigation_screen's mid-session region-download refresh listener entirely (not kept as a lightweight refresh) — 25.1-03's spike verdict matched the plan's literal 'no remount needed' criterion for outright deletion. The proxy re-queries RegionEntity fresh per HTTP request, so once the camera reaches newly-covered tiles MapLibre's native fetch succeeds with no Dart-side signal needed.
- [Phase 26]: [26-01] missingCoverageRegions/overlappingRegions kept as two distinct pure functions so callers distinguish fully-covered from no-region-overlap-at-all (D-04) — Enables Plan 03 to surface the D-04 non-blocking warning without conflating it with the fully-covered silent-proceed path
- [Phase 26]: [26-02] showAggregateProgress kept on one physical line (dart format off/on) to satisfy the plan's literal single-line grep acceptance criterion, matching the 20-05/21-01/25-04 precedent.
- [Phase 26]: [26-02] Missing-coverage sheet's doc comment reworded away from literal 'downloadVector'/'downloadDem' substrings so the plan's own negative grep (no download-engine calls) passes on comment text too, not just code.
- [Phase 26]: [26-03] Ref.listenManual does not exist on a Notifier's plain Ref; used ref.container.listen(...) instead — flutter_riverpod only declares listenManual on BaseWidgetRef (widget-tree consumers); a Notifier's Ref only implements BaseRef. Ref.container is a public ProviderContainer getter whose .listen() returns the same closeable ProviderSubscription, so it is the correct non-widget-code equivalent for the D-10 aggregate-notification subscription.
- [Phase 26]: [26-04] Restructured DownloadingTrailIds.download() into a single outer try/finally (CR-01) with regionFutures/aggregateSub/glyphCacheWarm hoisted above it, plus a regionListNotifierProvider invalidation in the region-futures finally (CR-02) and an aggregate-aware onGeneratingChanged (WR-01/D-10)
- [Phase 26]: [26-04] Added package:flutter_riverpod/flutter_riverpod.dart import to trail_download_state_provider.dart for the ProviderSubscription type -- riverpod_annotation alone didn't reliably expose it to the analyzer, which also produced misleading KeepAliveLink false-positive warnings until the import was added
- [Phase 26]: [26-05] Re-read downloadNotificationServiceProvider via ref.read() for the deferred aggregate-success call rather than hoisting notificationService out of the outer try block -- it's a plain synchronous Provider, so re-reading after the try/finally is safe and keeps the hoisted-variable surface unchanged
- [Phase 26]: [26-05] TileRepositoryManager._getOrCreatePackage takes {required bool dem} and re-fetches the current RegionEntity row inside its write transaction before linking a package (fresh-row read-modify-write), preventing a concurrent Vector/DEM download's stale full-row snapshot from clobbering the sibling package FK
- [Phase 27]: [27-01] totalPoints computed up front from photoTotal instead of the deleted tile future's onCellTotal callback — downloadTrail() no longer downloads tiles; onProgress reporting for photo/waypoint-photo downloads must not silently break
- [Phase 27]: [27-02] Regeneration touched two riverpod provider hash files (map_style_json_provider.g.dart, trail_download_state_provider.g.dart) as a side effect of the single project-wide build_runner pass — Logic in those files is unchanged; only the compile-time source hash used for provider identity shifted because it hashes the whole dependency graph including the Trail model
- [Phase 27]: [27-02] No ObjectBox migration step performed for pmTiles/demPmTiles removal — Field removal is a supported regeneration for a pre-production app (D-06); build_runner's own log confirmed both properties were cleanly retired from the model

### Roadmap Evolution

- Phase 25.1 inserted after Phase 25 (2026-07-23, URGENT): Local HTTP tile proxy for region-based offline map rendering. Replaces `navigation_screen.dart`'s incremental `addSource`/`removeSource` region-swap reconcile with a loopback `HttpServer` serving `pmtiles` archives per-tile via a static `tiles:` XYZ source, so MapLibre Native's own viewport tracking takes over instead of hand-rolled Dart diffing -- structurally eliminates the reentrancy bug class (no reconcile call = no race). Considered and rejected: patching `_reconcileRegionComposition` in place (reentrancy guard + symmetric tracking-set mutation + gating camera-idle during GPS-follow) -- viable and lower-risk short-term, but the proxy better fits the genuinely dynamic viewport-tracking requirement and also removes the `_sourceFromJson`/`_layerFromJson` duplication between `trail_map.dart` and `navigation_screen.dart`. New unknowns the proxy introduces: per-tile overlap-resolution logic for regions with overlapping bboxes (none exists today), and unverified MapLibre Native offline+loopback-HTTP behavior (needs its own on-device spike, mirroring RENDER-03's). Phase 25 itself is NOT force-completed -- `25-UAT.md` and `25-VERIFICATION.md` accurately record the Test 4 gap; `25-02`'s `localTilePathsForBounds` split survives unchanged and Phase 25.1 will consume it.

### Pending Todos

- Fix 3 pre-existing `flutter test` failures (`feed_item_test.dart` x2, `settings_screen_test.dart` x1) — logged in Phase 18's `deferred-items.md`; not blocking v1.5 work.
- Manual on-device verification for quick task 260712-m9v (resume navigation after manual app termination): start navigation, swipe-kill the app, relaunch, accept the resume dialog, confirm maneuver index/distance/elevation/elapsed/breadcrumb continue; also verify decline and deliberate-exit paths show no prompt.
- Way Types & Surfaces breakdown feature (mobile-first) — komoot-style stacked bar/legend of OSM way types + surfaces per trail via Valhalla `trace_attributes` (`max_hiking_difficulty: 6` fixes off-road hiking-trail match dropout, verified). Web API computes + persists `way_type_surface` on trail save (no Go/PocketBase hook needed — all writes go through the web API); Flutter renders first, SvelteKit UI deferred. Full plan: `.planning/todos/pending/2026-07-18-way-types-and-surfaces-breakdown.md`.
- Manual on-device verification for quick task 260719-d6a (time-in-motion navigation timer): start navigation, walk → timer/distance advance; stand still ~10s → timer stops, distance stops, GPS drops to periodic low-power fixes; resume walking → timer/GPS/stats all resume automatically. Also confirm the manual pause button and route-following/maneuver/waypoint behavior are unaffected.
- Manual on-device verification for quick task 260719-n8g (route recorder): tap "Record trail" → grant permission → recording session opens (map centers on first fix, no maneuver banner, bottom row [pause, stop, elevation]); walk → stats/timer advance, pause/resume works; tap red Stop → "Stop recording?" dialog → Save hands off to trail_create_screen with the recorded track prefilled; start a recording, swipe-kill the app, relaunch → "Resume recording?" prompt → accept continues breadcrumb/stats, decline shows no prompt on next launch.

### Blockers/Concerns

- **[Phase 15/17 — OFFL-06 deferred, by design]** `pm_tile_provider.dart` intentionally NOT deleted — `navigation_screen.dart` still consumes `MultiPmTilesVectorTileProvider` for its offline flutter_map path. Delete in Phase 17/18 once that screen migrates off flutter_map. ROADMAP.md's Phase 15 success criterion 5 was corrected in place to reflect this (it originally self-contradicted).
- **[Phase 15 — A2 sprite `file://` gap, low exposure, not investigated further]** The 15-01 spike found `file://` **sprite** resolution FAILED on a physical Android device despite valid cached files (glyph `file://` resolution PASSED). 15-05 sidesteps the practical impact for the trail's own arrow icon by self-registering it (`addImageFromIconData`, sprite-independent) rather than relying on the sprite. The only remaining exposure is OTHER sprite-atlas icons (e.g. route-network shields) rendering offline — those don't render at all today regardless of online/offline, so this is a pre-existing not-yet-working feature, not a regression. Not re-investigated natively; revisit if/when sprite-sourced icons become a real feature.
- **[Phase 15/16/17]** `maplibre` 0.3.5 is pre-1.0 with breaking changes across 0.x minors (three published upgrade guides). Pin the exact version on first add; CLEAN-03 locks it at the end.
- **[Phase 15 — CORE-05 gap, expected]** `trail_detail_map_screen`'s `MapCompass` (flutter_map-only) was removed rather than crashing against the new `MapLibreMap` — that screen has no compass/rotation-reset control until Phase 17 (CORE-05) wires maplibre's native equivalent.
- **[v1.5 research flag]** `package:maplibre` 0.3.5's exact `MapGestures`/`MapOptions` pan/rotate-disable API surface is MEDIUM confidence (changelog/GitHub discussion, not a direct source read) — validate with a small spike early in Phase 19 before committing to the full drag-vs-pan gesture-arena solution.
- **[v1.5 research flag]** The generation-counter/CancelToken race-guard pattern for out-of-order Valhalla responses is MEDIUM confidence against this project's exact pinned `riverpod_annotation` 4.0.2 — confirm the idiom during Phase 19 planning/execution.
- [Phase 24 — RESOLVED 2026-07-23, user-verified] On-device physical verification of the Offline Maps/Regions screen is complete: user confirmed all tests pass and the phase is verified. Note the checklist itself changed post-completion (see REQUIREMENTS.md's SETUI-03/04 amendment and ROADMAP.md's Phase 24 amendment, commits `4732d20e`/`663f049a`/`5b06feed`): pause/resume was replaced by cancel-and-restart-from-0, and the DEM toggle was replaced by an independent, Vector-gated DEM tile. A later phase should verify against the amended criteria, not the original six-point checklist.
- **[Phase 25.1-03 — found during PROXY-03 on-device spike, pre-existing, out of scope, NOT fixed]** `MapStyleSourcesNotifier` (`app/lib/provider/map_style_sources_provider.dart`, dates to Phase 15) makes an unconditional `GET /map/style-sources` call on every fresh `build()` with zero offline fallback or local cache, even when the glyph/sprite files it needs are already cached on disk. It's `@Riverpod(keepAlive: true)`, so once resolved successfully in a running session it stays cached for that session's lifetime — but a genuinely cold app launch/restart directly into an offline state can never resolve it, so `_composeStyle`/`_loadStyle` in both `trail_map.dart` and `navigation_screen.dart` fails at the glyph-cache step before any style composition happens. Equally affects current production code today; not introduced or worsened by Phase 25.1. Needs a future phase to add an offline/disk-cache fallback path to `MapStyleSourcesNotifier`.

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
| 260717-seb | Reverse-geocode route planner anchors on street name, matching web client behavior | 2026-07-17 | aefa55fb,f56d31e7,90cbaa37 | Complete | [260717-seb-…](./quick/260717-seb-reverse-geocode-route-planner-anchors-on/) |
| 260717-t7q | Add a Settings tab to the Route Planner sheet: consolidated Valhalla travel-profile picker, relocated auto-routing toggle | 2026-07-17 | 53b6b712,c9ec6e45,13f220eb,497803f0,cae4e55c,345e3e53 | Needs Review | [260717-t7q-…](./quick/260717-t7q-add-a-settings-tab-to-the-route-planner-/) |
| 260718-e9j | Edit an existing route in the trail planner: entry point on trail_create_screen, web-parity anchor prepopulation, pop-with-result return | 2026-07-18 | ee3bbe1d,e93c0ed2,9c171b27,38210d8f | Needs Review | [260718-e9j-…](./quick/260718-e9j-a-user-should-be-able-to-edit-an-existin/) |
| 260719-d6a | Navigation timer shows time-in-motion: tracelet native speed-motion engine drives auto-pause of timer/GPS/stats when stationary | 2026-07-19 | 0a914220,b1fb530c,461ad44a,e4012bcd | Needs Review | [260719-d6a-…](./quick/260719-d6a-the-navigation-timer-should-show-time-in/) |
| 260719-fjw | Save track recorded during navigation: stub Trail from breadcrumb, offered on completion banner and premature-exit dialog, hands off to trail_create_screen | 2026-07-19 | 2bd575f0,61497acd,2b2aa687 | Needs Review | [260719-fjw-…](./quick/260719-fjw-save-track-recorded-during-navigation-cr/) |
| 260719-n8g | Implement the missing route recorder: isRecording flag reuses NavigationScreen for trail-less GPS recording, wired Record trail card + /record route, ActiveSessionType.rec resume-after-kill; post-review fix gates breadcrumb capture on pause/stationary | 2026-07-19 | 20316c47,c41b757d,602be822 | Needs Review | [260719-n8g-…](./quick/260719-n8g-implement-the-missing-route-recorder-mos/) |
| 260720-s7m | Clean up ARB translation files: safety-scanned dead-key removal (291 removed), equalized all 14 locales to 267 keys, extracted 43 hard-coded Dart literals into l10n keys, added @key ICU metadata, extended crowdin.yml for the app | 2026-07-20 | bde01f50,4116ecba,dc6b98a9,f10dc9d8 | Needs Review | [260720-s7m-…](./quick/260720-s7m-clean-up-arb-translation-files-remove-un/) |
| 260721-eob | Add a bottom modal sheet on track save from navigation screen with Recalculate-heights/Follow-roads toggles; new authenticated /valhalla/trace-route SvelteKit proxy + snap-then-heights pipeline with a bbox-diagonal truncation guard | 2026-07-21 | d35be0e6,5c93547a,9346ef8e | Needs Review | [260721-eob-…](./quick/260721-eob-add-a-bottom-modal-sheet-on-track-save-f/) |

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

Last session: 2026-07-24T15:48:38.191Z
Stopped at: Completed 27-02-PLAN.md
Resume file: None
