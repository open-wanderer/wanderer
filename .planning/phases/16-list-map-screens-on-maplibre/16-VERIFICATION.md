---
phase: 16-list-map-screens-on-maplibre
verified: 2026-07-09T20:12:40Z
status: passed
score: 5/5 must-haves verified (roadmap success criteria); 17/17 plan-level must-have truths verified
overrides_applied: 0
---

# Phase 16: List & Map Screens on MapLibre Verification Report

**Phase Goal:** The browse surfaces run on maplibre — a list's trails on one fitted map, and the map screen's trail search rendered from the server's cluster endpoint as native layers.
**Verified:** 2026-07-09T20:12:40Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A hiker opens a list and sees every trail in it drawn on one `MapLibreMap`, camera animates to fit all of them; list detail screen's inline map does the same | ✓ VERIFIED | `list_detail_map_screen.dart:151-183` builds `ml.PolylineLayer(polylines: polylines, color: kTrailRouteColor, width: 5)` + `WidgetLayer` markers over `SearchMap`, fits `combinedBounds` in `onStyleLoaded`. `list_detail_screen.dart:385-416`'s `_ListMap` does the identical thing with `SearchMap(disabled: true)`. Both `flutter analyze` clean. |
| 2 | Panning/zooming re-queries `POST /search/trails/cluster` at new bounds/zoom, debounced as today; FeatureCollection renders as native circle layers sized by `point_count`, labelled `point_count_abbreviated`, matching web's `ClusterLayer` step ramp | ✓ VERIFIED | `map_cluster_search_provider.dart:43-55` — 400ms `Timer` debounce, POSTs `southWest/northEast/zoom/filterText/q`. `cluster_layer.dart:30-88` — `circle-radius` step ramp (`10,5,12,10,15,50,18,100,22,500,25`) and `cluster-count` symbol layer are byte-identical to `web/src/lib/vendor/maplibre-layer-manager/cluster-layer.ts:47-96` (diffed line-by-line). |
| 3 | Tapping a cluster zooms the camera toward it; tapping an unclustered point selects that trail and fits the camera to its polyline | ✓ VERIFIED | `map_screen.dart:315-346` — `featuresAtPoint(layerIds: ['clusters'])` → `animateCamera(center: event.point, zoom: currentZoom + 2, ...)`. `map_screen.dart:142-160` `_selectTrail` — fetch-then-fit via `trailPolylineProvider` + `controller.fitBounds(bounds: LngLatBounds.fromPoints(polyline), ...)`. |
| 4 | Hiding a category/subcategory in Settings → Categories changes which trails the map screen returns, because the endpoint applies preference filters server-side | ✓ VERIFIED | `web/src/routes/api/v1/search/trails/cluster/+server.ts:37-40` calls `withTrailPreferenceMeiliFilter(event, [geoFilter, filterText])`, which reads `user_category_preferences`/`user_subcategory_preferences` collections server-side (`web/src/lib/server/category_preference_filter.ts`) — independent of any client-sent filter text. |
| 5 | The app builds and runs; `navigation_screen` still renders on `flutter_map` | ✓ VERIFIED | `flutter analyze` (whole app): 0 issues in any Phase 16 file, only the same 36 pre-existing unrelated issues logged in `deferred-items.md`. `navigation_screen.dart:5-13` still imports `flutter_map`, `flutter_map_animations`, `flutter_map_location_marker`, `vector_map_tiles`, and builds a `FlutterMap` widget at line 213. |

**Score:** 5/5 ROADMAP success criteria verified.

### Plan-Level Must-Haves (frontmatter truths, all 3 plans)

All 17 `must_haves.truths` entries across 16-01/16-02/16-03-PLAN.md frontmatter were independently checked against the current code (not the SUMMARY narration) — all 17 VERIFIED. Full detail in Artifacts/Key-Links tables below; no plan-level truth failed or was left partial.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/components/base/search_map.dart` | Trail-agnostic MapLibreMap host (style load + theme swap) | ✓ VERIFIED | `class SearchMap` present, substantive (132 lines, real style-swap + buffered-callback logic), wired into 3 call sites (`grep` confirms). |
| `app/lib/routes/list_detail_map_screen.dart` | Full list map, fit-to-all-trails + per-marker tap | ✓ VERIFIED | Real `SearchMap` usage, polylines/markers built from `listProvider` data, `flutter analyze` clean. |
| `app/lib/routes/list_detail_screen.dart` | Inline non-interactive list map (`_ListMap`) | ✓ VERIFIED | `_ListMap` is `ConsumerStatefulWidget`, `SearchMap(disabled: true)`, tap-through via `MapEventClick`. |
| `app/lib/provider/trail/map_cluster_search_provider.dart` (+`.g.dart`) | Debounced bounds+zoom-keyed cluster search provider | ✓ VERIFIED | Real POST to `/search/trails/cluster`, 400ms debounce, `ref.listen(trailFilterProvider('map'))` re-query, imported and used in `map_screen.dart`. |
| `app/lib/components/map/cluster_layer.dart` | Native `clusters`/`cluster-count` layer builder (verbatim web port) | ✓ VERIFIED | `addClusterLayers`/`updateClusterSource` present; filter/paint/layout values diffed byte-for-byte against `cluster-layer.ts`; imported and called from `map_screen.dart`. |
| `app/lib/routes/map_screen.dart` | Map screen on `SearchMap` + cluster layers + category-icon markers + native tap handling | ✓ VERIFIED | No `flutter_map`/`vector_map_tiles` imports remain; hosts on `SearchMap`; wires both `mapClusterSearchProvider` and `mapTrailSearchProvider`; native `featuresAtPoint`/`WidgetLayer.onTap` handling present. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `search_map.dart` | `mapStyleJsonProvider` | `ref.watch` + `ref.listen` live theme swap | ✓ WIRED | `_swapStyle()` calls `controller.setStyle(json)` on provider change. |
| `list_detail_map_screen.dart` | `ml.MapController.fitBounds` | `onStyleLoaded` imperative bounds fit | ✓ WIRED | Confirmed at lines 153-163. |
| `list_detail_screen.dart` | `/list/{id}/map` | `MapEventClick` tap-through navigation | ✓ WIRED | `context.push('/list/${widget.list.id}/map')` on tap. |
| `map_cluster_search_provider.dart` | `POST /search/trails/cluster` | `api.post` with bounds/zoom/filterText/q body | ✓ WIRED | Confirmed against server route `web/src/routes/api/v1/search/trails/cluster/+server.ts`, which parses exactly this body shape. |
| `map_cluster_search_provider.dart` | `TrailFilter.toFilterText` | filter serialization (CLUS-05) | ✓ WIRED | `filter.toFilterText(actor:, includeGeo: false)` called before POST. |
| `cluster_layer.dart` | `ml.StyleController.updateGeoJsonSource` | re-query data swap (CLUS-04) | ✓ WIRED | `updateClusterSource` calls `style.updateGeoJsonSource(id: kClusterSourceId, data: geojson)`, no remove/re-add churn. |
| `map_screen.dart` | `mapClusterSearchProvider` + `cluster_layer.updateClusterSource` | `ref.listen`, `updateGeoJsonSource` on new data | ✓ WIRED | `ref.listen(mapClusterSearchProvider, ...)` calls `updateClusterSource(style, jsonEncode(data))`. |
| `map_screen.dart` | `mapTrailSearchProvider` (parallel, retained) | bottom-sheet list + tapped-trail metadata | ✓ WIRED | `ref.watch(mapTrailSearchProvider)` drives `searchResultAsync`/`trails`, used for both the sheet list and `firstWhereOrNull` marker metadata lookup. |
| `map_screen.dart` | `ml.MapController.featuresAtPoint(layerIds: ['clusters'])` | cluster-tap detection | ✓ WIRED | Confirmed at lines 322-325. |
| `map_screen.dart` | `trailPolylineProvider` | unclustered-tap fetch-then-fit (D-02) | ✓ WIRED | `_selectTrail` reads `trailPolylineProvider(trailId).future`. |

## Targeted Double-Checks (requested)

### 1. Does the `is_large` filter fix (`b3949935`) reintroduce visual duplication with the native `clusters` circle layer?

**No duplication — verified false alarm.** The native `clusters` and `cluster-count` layers in `cluster_layer.dart` (and their web source-of-truth in `cluster-layer.ts`) both filter on `[">", ["get", "point_count"], 1]`. Every `is_large` feature the server emits always carries `point_count: 1` (confirmed in `web/src/routes/api/v1/search/trails/cluster/+server.ts:112-113`), so the native circle layer's `point_count > 1` clause excludes `is_large` features unconditionally — independent of whatever the `is_large` clause in the same filter does. `map_screen.dart`'s Dart-side `WidgetLayer` marker loop only iterates `point_count == 1` features (line 208), a disjoint set from what the native `clusters` layer ever renders. The two rendering paths (native circle layer for clusters, Dart `WidgetLayer` for individual points) are mutually exclusive by `point_count` value, not by `is_large`, so removing the `is_large` guard from the Dart loop cannot cause a point to be drawn twice. `cluster_layer.dart`'s own comment (lines 24-29) documents this reasoning; independently re-derived here from the raw filter/query logic rather than trusted from the comment.

### 2. Does the `SearchMap` buffering fix leave a dangling `_pendingStyle` on early disposal (e.g., before `onMapCreated` fires)?

**No leak or crash risk found.** Traced the actual widget-disposal order: Flutter unmounts children before parents, so the native `ml.MapLibreMap`'s own `State` (e.g. `MapLibreMapStateAndroid` in `maplibre_android-0.3.5/lib/src/map_state.dart`) is disposed — releasing its JNI map/listener handles — *before* `_SearchMapState.dispose()` would ever run. `_SearchMapState` itself has no `dispose()` override and holds no subscriptions, timers, or native handles of its own; `_pendingStyle` is a plain nullable field that is simply garbage-collected with the rest of the State object if the widget unmounts before `onMapCreated` fires. There is no explicit `mounted` guard around the two callback-forwarding call sites (`widget.onMapCreated?.call(...)`, `widget.onStyleLoaded?.call(...)`), but this matches the pre-existing pattern used by the underlying `maplibre_android` package itself (`widget.onMapCreated?.call(this)` at `map_state.dart:256` has no `mounted` check either) — not a regression introduced by this fix. One residual, pre-existing (not new) risk class: if a native callback arrives asynchronously after the *parent* caller's own `State` (e.g. `_ListDetailMapScreenState`, `_MapScreenState`) has been disposed, calling back into `ref.read(...)` inside `onStyleLoaded` (as `map_screen.dart` does) could throw. This is unrelated to the `_pendingStyle` buffering itself — it exists identically whether the callback fires immediately or is replayed from the buffer — and none of the three call sites' `onStyleLoaded` handlers currently guard with `mounted`. Flagged as a minor, non-blocking code-quality note, not a Phase 16 goal failure (same risk class pre-dates this phase throughout the app's other `ConsumerState` + `ref.read`-in-async-callback code).

### 3. Does `navigation_screen.dart` genuinely still render on `flutter_map` (Success Criterion 5)?

**Confirmed genuine, not just an unused import left behind.** `navigation_screen.dart` imports `package:flutter_map/flutter_map.dart`, `package:flutter_map_animations/flutter_map_animations.dart`, `package:flutter_map_location_marker/flutter_map_location_marker.dart`, and `package:vector_map_tiles/vector_map_tiles.dart`, and constructs a `FlutterMap(...)` widget directly at `navigation_screen.dart:213`. It also still depends on `map_style_provider.dart` (the legacy `vtr.Style`-returning provider, distinct from the new `mapStyleJsonProvider` Phase 16 uses) and `map_compass.dart` (the flutter_map-only compass widget Phase 16's `map_screen.dart` explicitly replaced with `ml.MapCompass`). This matches ROADMAP's stated sequencing — Phase 17 is where `navigation_screen` migrates and these shared files (`map_compass.dart`, `map_style_provider.dart`) are retired.

## Anti-Patterns Found

None blocking. `flutter analyze` on all 8 Phase-16-touched files: "No issues found!" Whole-app `flutter analyze`: 36 issues, all pre-existing and unrelated to Phase 16 (documented in `deferred-items.md`, independently re-confirmed here) — the one Phase-16-relevant pre-existing issue noted in `deferred-items.md` (`map_screen.dart:27` unused `subcategory.dart` import) is confirmed fixed (no longer present).

**Informational, non-blocking:**
- `flutter test test/` has 3 pre-existing failing tests (`feed_item_test.dart` ×2, `settings_screen_test.dart` ×1) — none touch Phase 16 files (feed items, settings screen), unrelated to this phase's scope, not introduced by it.
- `.planning/ROADMAP.md`'s **Progress summary table** (the `| Phase | Milestone | Plans Complete | Status | Completed |` table, Phase 16 row) still reads `2/3 | In Progress` with no completion date, while the phase-level checkbox (`- [x] **Phase 16...**`) and all three plan checkboxes (`16-01`/`16-02`/`16-03-PLAN.md`) were correctly flipped to `[x]` by commit `16d6ab9c`. This is a stale documentation row, not a code defect — the authoritative "Phase Details" section and REQUIREMENTS.md traceability table are both internally consistent and correctly marked complete. Recommend a follow-up doc-only edit to Phase 16's Progress table row (`3/3 | Complete | 2026-07-09`) before/alongside starting Phase 17, so the summary table doesn't mislead a future reader.

## Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| CORE-08 | 16-01 | `list_detail_map_screen`/`list_detail_screen` render multi-trail polylines via `MapLibreMap`, camera fits list | ✓ SATISFIED | See Truth #1, Artifacts table |
| CLUS-01 | 16-02/16-03 | Map screen's bbox search calls `POST /search/trails/cluster`, renders FeatureCollection | ✓ SATISFIED | See Truth #2, Key Links |
| CLUS-02 | 16-02 | Clusters render as native circle layers sized/labelled matching web's step ramp | ✓ SATISFIED | Byte-for-byte diff against `cluster-layer.ts` |
| CLUS-03 | 16-03 | Cluster tap zooms; unclustered tap selects + fits polyline | ✓ SATISFIED | See Truth #3 |
| CLUS-04 | 16-02/16-03 | Pan/zoom re-queries endpoint, debounced | ✓ SATISFIED | 400ms `Timer` debounce confirmed |
| CLUS-05 | 16-02 | Category/subcategory filters constrain map results server-side | ✓ SATISFIED | See Truth #4, `withTrailPreferenceMeiliFilter` |

No orphaned requirements — REQUIREMENTS.md's Phase 16 mapping (CORE-08, CLUS-01..05) matches exactly what all three plans' `requirements:` frontmatter fields declare in aggregate.

## Human Verification Required

None outstanding. The phase's `checkpoint:human-verify` gate (16-03 Task 3, `autonomous: false`) was already executed on a physical Android device by Christian, per `16-03-SUMMARY.md`'s documented 8-step checklist (all 8 steps passed) and 5 specific bugs found/fixed live (`b3949935`, `16d6ab9c`). This verifier independently re-derived and confirmed the code-level correctness of all 5 checkpoint fixes (diffed against the actual commits, cross-checked against the installed `maplibre` package source and the SvelteKit-side sprite/style-sources contract) rather than trusting the SUMMARY's narration — see Targeted Double-Checks above and the fix-by-fix diff verification performed against `16d6ab9c`. No new UI/visual behavior was introduced by this verification pass that would require a fresh device pass.

## Gaps Summary

No blocking gaps. One informational documentation inconsistency (stale ROADMAP.md Progress table row) noted above — does not affect code correctness or the phase's actual goal achievement, but should be cleaned up.

---

*Verified: 2026-07-09T20:12:40Z*
*Verifier: Claude (gsd-verifier)*
