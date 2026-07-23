# Phase 25: Map Rendering — Region-Based Viewport Pipeline - Research

**Researched:** 2026-07-23
**Domain:** Flutter native-GL map rendering (`maplibre` 0.3.5 pinned) — swapping trail-scoped offline tile sourcing for an app-wide region registry, with viewport-scoped style composition
**Confidence:** HIGH (RENDER-01/RENDER-03's API question resolved via direct pub-cache source read; RENDER-02's debounce precedent corrected against actual `map_screen.dart` source; one CRITICAL data-shape mismatch discovered between Phase 23's shipped code and this phase's assumed contract)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Uncovered-Viewport Behavior**
- **D-01:** When the map viewport (TrailMap's fixed trail-bounds query, or navigation_screen's live GPS position) falls in an area with no downloaded region, render a blank/empty basemap — the same failure mode as today's offline trail-map when `cellPaths` is empty. No new "no offline data" banner or online-fallback attempt; this stays a pure offline-rendering pipeline swap with no new UI surface.
- **D-02:** This blank-basemap behavior is uniform across both `TrailMap` and `navigation_screen` — no screen-specific "no map data here" indicator. Navigation continues live GPS/maneuver tracking regardless of whether basemap tiles are rendering.

**Region-Swap Visual Feel (navigation_screen)**
- **D-03:** A brief flicker/camera-reset when crossing a region boundary during live navigation is an acceptable v1.6 tradeoff if the RENDER-03 spike finds maplibre 0.3.5 only supports full style reload (not incremental `addSource`/`removeSource`). Seamless swapping is NOT a hard requirement gating the composition strategy — ship whichever approach the spike confirms works.
- **D-04:** Region-swap recomposition is debounced on camera-idle/moveend, not recomputed on every camera-move event — mirrors the existing cluster-search bbox-query debounce pattern in `map_screen.dart`, minimizes exposure to Pitfall 5's known `maplibre` 0.3.5 style-reload quirks.

**Trail Spanning Two Downloaded Regions (TrailMap)**
- **D-05:** If a trail's bounds straddle two downloaded regions, both regions' vector/DEM sources render together in the same style composition — not swapped. `TrailMap`'s viewport is fixed for the screen's lifetime (queried once from `trail.bounds`), so `localTilePathsForBounds(trail.bounds)` naturally returns every overlapping region's paths in one query.

**Viewport Scope: Which Screen Gets Swap Logic**
- **D-06:** RENDER-02's viewport-scoped swap-in/accumulate logic (debounced camera-idle recompute) runs on `navigation_screen` only. `TrailMap` does a one-time bounds query at build/composition time.

### Claude's Discretion
- The exact composition strategy (incremental `addSource`/`removeSource` vs. full style reload) is Claude's/the spike's call, per D-03 — resolve via RENDER-03's direct package-API verification against the installed `maplibre` 0.3.5, not a user preference.
- Whether the RENDER-03 spike ships as a standalone throwaway harness (mirroring 23-06's `tile_repository_manager_harness.dart` precedent) or is folded directly into the first implementation plan's Task 1 is a planning-level call.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed entirely within phase scope. No new capabilities were proposed; all four discussed areas were implementation-mechanics questions about the already-scoped rendering pipeline swap.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| RENDER-01 | `TrailMap` and `navigation_screen` read offline tiles from the region registry via `TileRepositoryManager`, replacing trail-bound cache reads | §Standard Stack, §Code Examples, §Common Pitfalls (Pitfall 1: flat-list/vector-DEM split gap) |
| RENDER-02 | Style composition is viewport-scoped — only regions intersecting the current viewport contribute style sources, not every downloaded region unconditionally | §Architecture Patterns (Pattern 2), §Code Examples (camera-idle listener) |
| RENDER-03 | Verify maplibre 0.3.5's incremental source add/remove behavior (vs. full style reload) and layer-count scaling with a spike against the pinned package version | §RENDER-03 Spike Findings (resolved to HIGH confidence via direct source read), §Open Questions (on-device performance spike still required) |
</phase_requirements>

## Summary

This phase is a pure rewiring + one architecture-verification task, not new feature-building: two existing call sites (`TrailMap._composeStyle`, `navigation_screen`'s inline `_composeStyle`) swap their tile-path source from `Trail.pmTiles`/`demPmTiles` to `TileRepositoryManager.localTilePathsForBounds`, and `navigation_screen` gains a new camera-idle-debounced recompute so panning across a region boundary swaps sources instead of accumulating every downloaded region unconditionally. `offline_style_rewriter.dart` is untouched.

**The single most important finding in this research is not RENDER-03 — it's a data-shape mismatch in RENDER-01.** `TileRepositoryManager.localTilePathsForBounds` (built in Phase 23, tested only via an on-device harness, not unit tests) returns **one flat, untyped `List<String>`** mixing vector and DEM archive paths together — not the `({List<String> vectorPaths, List<String> demPaths})` record shape that `.planning/research/ARCHITECTURE.md`'s Q1 speculated before Phase 23 was actually built. `rewriteStyleForOffline` requires these as two **separate** parameters (`cellPaths` for vector, `demCellPaths` for DEM) — feeding it a merged list would silently misroute a DEM `.pmtiles` archive into the vector-source cell-duplication path, reproducing the exact bug class already fixed once in commit `3f67cf37` (quick-260711-lzb, "hillshadeSource swept into the vector-cell repoint path"). This must be fixed at the `TileRepositoryManager` layer (a small, additive signature change to a Phase-23-owned function whose own doc comment says it exists specifically to feed this phase) before RENDER-01's callers can be wired correctly.

On RENDER-03: direct inspection of the installed `maplibre` 0.3.5 platform-interface and both native binding packages (Android JNI, iOS Obj-C/FFI) in the local pub-cache — not docs, not changelog — confirms `StyleController.addSource`/`removeSource`/`addLayer`/`removeLayer` are **real, implemented, non-stub native calls** on both platforms, and `navigation_screen.dart` **already calls this exact incremental API today** (`style.addSource(GeoJsonSource(...))` / `style.addLayer(LineStyleLayer(...))` for its breadcrumb layer, lines ~961-978) — proven working in production, not theoretical. This resolves PITFALLS.md's LOW/MEDIUM-confidence "unconfirmed at the API level" flag to HIGH confidence: **incremental composition is available.** However, `offline_style_rewriter.dart`'s contract stays JSON-document-based (D-06, unchanged) and the typed `Source`/`StyleLayer` incremental API requires building a **second, parallel** typed-object translation of `_pointSourceAtCell`/`_pointDemSourceAtCell` — extra surface for a seamlessness requirement (D-03) that is explicitly optional. The empirical 10-20-duplicated-source-set on-device performance spike ROADMAP's success criterion 1 calls for **cannot be resolved by static source-reading** and must still run as a real on-device Task 1.

**Primary recommendation:** Fix `TileRepositoryManager.localTilePathsForBounds` to return vector/DEM paths separately (small, additive change) before wiring RENDER-01. For RENDER-02/03, default to the full-style-reload strategy (`rewriteStyleForOffline` + `controller.setStyle(json)`, reusing the existing `_swapStyle`/`_onStyleLoaded` re-add-track machinery both call sites already have) gated behind a `ml.MapEventCameraIdle` listener on `navigation_screen` (mirroring the exact event `map_screen.dart` already uses for camera-position persistence) — this is the lowest-risk path that satisfies D-03/D-06 without new typed-object composition code. Run the on-device layer-count spike as Task 1, and only reach for incremental `addSource`/`removeSource` if the spike shows full-reload flicker/perf is unacceptable.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Offline tile path resolution (bbox → local paths) | Client — Service layer (`TileRepositoryManager`) | — | Already built in Phase 23; pure ObjectBox query, no network/server involvement |
| Style JSON composition (online base + offline rewrite) | Client — Provider/compose-helper layer (`TrailMap`/`navigation_screen` `_composeStyle`) | Client — Pure util (`offline_style_rewriter.dart`) | Compose helpers own *when* to recompute; the pure function owns *how* to transform JSON — this phase only changes the compose helpers' data source, not the transform |
| Viewport/camera state (bounds, idle detection) | Client — Widget/native-GL binding (`ml.MapController`, `onEvent` callback) | — | Native GL SDK owns camera state; Dart only reacts to `MapEventCameraIdle` |
| Native style source/layer mutation (add/remove/setStyle) | Client — Native platform binding (`maplibre_android`/`maplibre_ios` via `StyleController`) | — | JNI/FFI calls directly into the native MapLibre GL SDK; no server or Dart-only equivalent exists |
| Region download state / disk paths | Client — Service layer (`TileRepositoryManager`, `ObjectBox`) | — | Unchanged from Phase 23; this phase only *reads* the state, never writes it |

No capability in this phase touches the backend (Go) or SvelteKit web tiers — confirmed consistent with `.planning/research/ARCHITECTURE.md`'s "no backend changes" finding for the whole v1.6 milestone.

## RENDER-03 Spike Findings (resolve before planning composition strategy)

### Direct API verification (HIGH confidence — source read, not docs)

Inspected `~/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/style_controller.dart` (the abstract contract) and both native implementations:

```dart
// maplibre_platform_interface-0.3.5/lib/src/style_controller.dart
abstract class StyleController {
  Future<void> addSource(Source source);
  Future<void> addLayer(StyleLayer layer, {String? belowLayerId, String? aboveLayerId, int? atIndex});
  Future<void> updateGeoJsonSource({required String id, required String data});
  Future<void> removeLayer(String id);
  Future<void> removeSource(String id);
  List<String> getLayerIds(); // @visibleForTesting today
  // ... addImage*, dispose, setProjection
}
```

- **Android** (`maplibre_android-0.3.5/lib/src/style_controller.dart`, `StyleControllerAndroid`): `addSource`/`addLayer`/`removeSource`/`removeLayer` call real JNI bindings into `org.maplibre.android.style` (`jni.Style.addSource`/`addLayer`/`removeSource`/`removeLayer`, `jni.VectorSource`, `jni.RasterDemSource`, etc.) — not stubs.
- **iOS** (`maplibre_ios-0.3.5/lib/src/style_controller.dart`, `StyleControllerIos`): same, via Obj-C FFI bindings into `MLNStyle`/`MLNVectorTileSource`/`MLNRasterDEMSource`.
- **`navigation_screen.dart` already uses this exact incremental API in production** (lines ~961-978): `await style.addSource(ml.GeoJsonSource(id: 'breadcrumb', data: ...))` then `await style.addLayer(const ml.LineStyleLayer(id: 'breadcrumb-route', sourceId: 'breadcrumb', ...))`. This is called on every `_onStyleLoaded` — i.e., every style reload — and is a proven, working, non-experimental pattern in this exact codebase today.

**Verdict: incremental `addSource`/`removeSource`/`addLayer`/`removeLayer` ARE supported by the installed `maplibre` 0.3.5 package on both mobile platforms.** This resolves `.planning/research/PITFALLS.md` Pitfall 5's LOW/MEDIUM-confidence "unconfirmed at the API level" flag to HIGH confidence — the API exists and is exercised in production by this exact codebase already.

### What this does NOT resolve

1. **The empirical 10-20-duplicated-source-set performance spike** (ROADMAP's Phase 25 success criterion 1, and PITFALLS.md Pitfall 4) is a real-device rendering-behavior/perf measurement, not an API-availability question — it cannot be answered by reading source. This must still be a genuine on-device Task 1 (mid-tier Android device) before committing to how many regions can be simultaneously materialized as style sources.
2. **Typed `Source`/`StyleLayer` objects vs. the JSON style document `offline_style_rewriter.dart` produces are two different representations.** `VectorSource`/`RasterDemSource` (Dart classes in `maplibre_platform_interface`) expose every field `_pointSourceAtCell`/`_pointDemSourceAtCell` need (`url`, `maxZoom` for vector; `url`, `tileSize`, `encoding: RasterDemTerrariumEncoding()` for DEM) — so incremental composition IS structurally feasible — but nothing today converts a `rewriteStyleForOffline`-produced JSON style document into typed `Source`/`StyleLayer` objects for incremental add/remove. Building that conversion is new code, not a reuse of the existing pure function.
3. **`addSource`/`addLayer` throw if the id already exists** (both Android and iOS implementations check-and-throw). Any incremental-swap implementation MUST track which source/layer ids are currently active and either skip re-adding an already-present id or `removeSource`/`removeLayer` first.
4. **`removeLayer`/`removeSource` are NOT symmetric across platforms on a missing id**: iOS null-checks and silently no-ops; Android's `StyleControllerAndroid.removeLayer`/`removeSource` call the native method directly with no existence check first — behavior on a missing id is unverified from source alone (native SDK behavior, not this binding's Dart code) and should be wrapped defensively (try/catch) rather than assumed safe.
5. **Native GL layer-before-source removal ordering** is a standard constraint of the underlying MapLibre/Mapbox GL native SDKs (well-established platform behavior, not verified against this specific binding version): every layer referencing a source must be removed before the source itself, or the native call is liable to fail. Any incremental-swap implementation must remove layers first, then sources, in that order.

### Recommendation for the composition strategy decision

Given D-06 ("`offline_style_rewriter.dart` reused unchanged — only its callers' data source changes") and D-03 ("brief flicker... acceptable... if spike shows only full-reload is possible... not a hard blocker"), the **lowest-risk default is full-style-reload**: recompute `rewriteStyleForOffline(...)` with the newly-resolved region paths and call `controller.setStyle(json)` on camera-idle — reusing the exact `_swapStyle`/`_onStyleLoaded` re-add-track/breadcrumb machinery both call sites already have for theme-toggle swaps. This requires zero new typed-object composition code and stays inside `offline_style_rewriter.dart`'s existing, tested contract.

**Only reach for incremental `addSource`/`removeSource`** if the on-device spike (Task 1) shows the full-reload flicker/perf is unacceptable in practice — and if so, budget it as materially more implementation work (new typed-object source/layer builder duplicating the JSON rewriter's cell-pointing logic, plus the id-collision/removal-ordering handling above), not a drop-in swap.

## Critical Data-Shape Gap (RENDER-01 blocker — must fix before wiring)

`app/lib/services/tile_repository_manager.dart` (Phase 23, lines 251-275):

```dart
/// Returns the local vector/DEM archive file paths for every downloaded
/// region whose bbox overlaps [query] (TILE-05) — feeds Phase 25's
/// viewport-based tile-reading pipeline. ...
List<String> localTilePathsForBounds(LngLatBounds query) {
  final paths = <String>[];
  for (final region in _store.box<RegionEntity>().getAll()) {
    if (!bboxOverlaps(...)) continue;
    final vectorPath = region.vectorPackage.target?.localFilePath;
    final demPath = region.demPackage.target?.localFilePath;
    if (vectorPath != null) paths.add(vectorPath);
    if (demPath != null) paths.add(demPath);
  }
  return paths;
}
```

This returns **one untyped, merged `List<String>`** — vector and DEM archive paths interleaved with no distinguishing tag. Compare to what the caller actually needs:

```dart
// offline_style_rewriter.dart's signature (unchanged, per D-06)
Map<String, dynamic> rewriteStyleForOffline(
  Map<String, dynamic> style, {
  required String cacheRoot,
  required List<String> cellPaths,       // vector ONLY
  List<String> demCellPaths = const [],  // DEM ONLY, separately typed
  bool dark = false,
});
```

`.planning/research/ARCHITECTURE.md`'s Q1 (written before Phase 23 was built) assumed `localTilePathsForBounds` would return `({List<String> vectorPaths, List<String> demPaths})` — a record splitting the two. **The actual Phase-23 implementation does not do this.** No unit test currently exercises this method's return shape either — `app/test/services/tile_repository_manager_test.dart` explicitly excludes it ("`localTilePathsForBounds` require a live [ObjectBox store]"), and the only exerciser is the on-device `tile_repository_manager_harness.dart`, which just `debugPrint`s the merged list without asserting its composition.

**Consequence if left unfixed:** naively passing `localTilePathsForBounds(bounds)`'s result as `cellPaths` would feed a DEM `.pmtiles` archive into `_pointSourceAtCell` (the vector-source cell-duplication path) instead of `_pointDemSourceAtCell` (which injects the required `encoding: terrarium`/`tileSize: 512`/DEM-specific `maxzoom`). This reproduces the exact bug class already found and fixed once in this codebase (commit `3f67cf37`, quick-260711-lzb: "`offline_style_rewriter.dart` was sweeping `hillshadeSource`... into the vector-cell repoint path purely because it carries a `url` key").

**Recommended fix (small, additive — touches Phase-23-owned code but is exactly the seam its own doc comment names for this phase):** Change `localTilePathsForBounds`'s return type to separate vector and DEM paths, e.g.:

```dart
({List<String> vectorPaths, List<String> demPaths}) localTilePathsForBounds(LngLatBounds query) {
  final vectorPaths = <String>[];
  final demPaths = <String>[];
  for (final region in _store.box<RegionEntity>().getAll()) {
    if (!bboxOverlaps(...)) continue;
    final vectorPath = region.vectorPackage.target?.localFilePath;
    final demPath = region.demPackage.target?.localFilePath;
    if (vectorPath != null) vectorPaths.add(vectorPath);
    if (demPath != null) demPaths.add(demPath);
  }
  return (vectorPaths: vectorPaths, demPaths: demPaths);
}
```

This is a signature change to an already-shipped, in-use (by the Settings harness) method — the plan must account for updating `tile_repository_manager_harness.dart`'s one call site alongside it, and should add a real unit test for the split (currently absent) since ObjectBox-store-dependent tests are excluded from the existing suite but this specific shape (which list a path lands in) can be tested with a fake/in-memory harness matching the existing test file's own patterns.

## Standard Stack

### Core (all pre-existing — no new dependencies)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `maplibre` | 0.3.5 (exact-pinned) | Native GL map rendering, style composition | Already the app's only map binding since v1.4; pinning is deliberate (pre-1.0, breaking changes across minors) |
| `objectbox`/`objectbox_flutter_libs` | ^5.3.1 | `RegionEntity`/`DownloadedTilePackageEntity` storage this phase queries | Already the app's only local DB |
| `pmtiles` | ^1.2.0 | Read-only `.pmtiles` archive access (validation only in this phase, via `TileRepositoryManager`) | Already used; this phase doesn't touch it directly |
| `flutter_riverpod`/`riverpod_annotation` | existing pinned versions | `ref.watch`/`ref.listen` composition of style inputs | Established pattern this phase extends, not replaces |

**No new packages are introduced by this phase.** Package Legitimacy Audit is not applicable — skip.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Full-style-reload (`setStyle(json)`) for region swap | Incremental `addSource`/`removeSource`/`addLayer`/`removeLayer` on the existing `StyleController` | Incremental avoids the flicker D-03 already accepts as optional; costs a new typed-object translation layer + id-collision/removal-ordering handling not currently needed anywhere else in the codebase |
| Modifying `localTilePathsForBounds`'s return shape | Post-processing the merged list client-side by checking each path's basename (`vector.pmtiles` vs `dem.pmtiles`, per `region_file_path.dart`'s fixed naming) | Fragile — couples call sites to `TileRepositoryManager`'s internal file-naming convention instead of an explicit typed contract; breaks silently if that naming ever changes |

## Architecture Patterns

### System Overview — Data Flow Through the Rewired Pipeline

```
┌─────────────────────────────────────────────────────────────────────┐
│ TrailMap.build() / navigation_screen.build()                        │
│   bounds = trail.bounds (TrailMap, once)                             │
│           | controller.getVisibleRegion() (navigation_screen, on     │
│             MapEventCameraIdle, debounced-by-event-nature)           │
└───────────────────────────┬────────────────────────────────────────┘
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│ ref.read(tileRepositoryManagerProvider)                              │
│   .localTilePathsForBounds(bounds)  [FIX NEEDED: split vector/DEM]   │
│   -> (vectorPaths: [...], demPaths: [...])                           │
└───────────────────────────┬────────────────────────────────────────┘
                             ▼  (parallel input, same as today)
┌─────────────────────────────────────────────────────────────────────┐
│ ref.watch(mapStyleJsonProvider)   [unchanged — online base style]    │
│ ref.watch(glyphSpriteCacheProvider) [unchanged — file:// cache root] │
│ ref.watch(regionListNotifierProvider) [NEW — triggers recompute on   │
│   region download/delete completing, mirrors glyph-cache-warm swap]  │
└───────────────────────────┬────────────────────────────────────────┘
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│ _composeStyle(baseJson, cache, vectorPaths, demPaths)                 │
│   rewriteStyleForOffline(decoded, cacheRoot:, cellPaths: vectorPaths,│
│     demCellPaths: demPaths, dark:)  — UNCHANGED function (D-06)      │
└───────────────────────────┬────────────────────────────────────────┘
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│ controller.setStyle(json)  — full reload (default strategy)          │
│   -> onStyleLoaded fires -> re-add trail track / breadcrumb layer    │
│      (existing _onStyleLoaded machinery, unchanged)                  │
└─────────────────────────────────────────────────────────────────────┘
```

### Recommended Edit Points (no new files)
```
app/lib/services/tile_repository_manager.dart   # MODIFIED — split return shape (see gap above)
app/test/services/tile_repository_manager_harness.dart  # MODIFIED — one call-site update
app/lib/components/base/trail_map.dart          # MODIFIED — _composeStyle data source swap (one-time bounds query)
app/lib/routes/navigation_screen.dart           # MODIFIED — _composeStyle data source swap + NEW camera-idle listener
app/lib/util/offline_style_rewriter.dart        # UNCHANGED (D-06)
```

### Pattern 1: Provider composition mirrors the existing dual-listen pattern

**What:** Both call sites already do `ref.listen(mapStyleJsonProvider, ...)` + (`offline` branch) `ref.listen(glyphSpriteCacheProvider, ...)`, both triggering `_swapStyle()`. Add a third: `ref.listen(regionListNotifierProvider, (_, _) => _swapStyle())`.

**Why `regionListNotifierProvider`, not a new provider:** It already exists (`app/lib/provider/region/region_provider.dart`, `@Riverpod(name: 'regionListNotifierProvider')`), returns a synchronous `List<RegionEntity>` snapshot, and — critically — is already invalidated by `settings_offline_regions_screen.dart` after every download/cancel/delete action (`ref.invalidate(regionListNotifierProvider)` at 4 call sites). This means a region finishing download in Settings and the user navigating back to a map screen will see fresh data on next build without any new plumbing — satisfying `.planning/research/ARCHITECTURE.md`'s "region finishing download mid-session live-swaps the style" goal using an already-shipped invalidation seam, not a new one.

**Trade-off:** `regionListNotifierProvider`'s `build()` re-reads every `RegionEntity` synchronously on invalidation — cheap at the "handful to dozens of regions" scale this milestone targets (per `.planning/research/ARCHITECTURE.md`'s Scaling Considerations), consistent with the existing linear-scan design of `localTilePathsForBounds` itself.

**Example (grounded in the actual `TrailMap._composeStyle` structure):**
```dart
// Source: app/lib/components/base/trail_map.dart (existing structure, RENDER-01 delta shown)
ref.listen(mapStyleJsonProvider, (_, _) => _swapStyle());
if (widget.offline) {
  ref.listen(glyphSpriteCacheProvider, (_, _) => _swapStyle());
  ref.listen(regionListNotifierProvider, (_, _) => _swapStyle()); // NEW
}
```

### Pattern 2: navigation_screen's camera-idle recompute is the event, not a Timer

**Correction to CONTEXT.md's D-04 framing:** D-04 says the debounce "mirrors the existing cluster-search bbox-query debounce pattern in `map_screen.dart`." Direct inspection of `map_screen.dart` found **no `Timer`-based debounce anywhere in the file** — the actual pattern is:
- `MapEventStartMoveCamera` (pan begins) → shows a "Search this area" button (`_searchAreaController.forward()`), does NOT search.
- The bbox search only fires on an **explicit user tap** of that button, or after a cluster-tap zoom completes (`animateCamera(...).then(...)`) — never automatically on pan/drag settling.
- `MapEventCameraIdle` IS listened to, but only to persist camera position (`mapCameraProvider.notifier.save(...)`) — not to trigger a search.
- This is corroborated by `.planning/STATE.md`'s own Phase 16-03 checkpoint note: *"Cluster-tap zoom... auto-triggers `searchInBounds`... explicit user request, distinct from pan/drag which still requires the manual 'Search this area' button."*

**What this means for RENDER-02:** there is no reusable `Timer`-debounce code to copy. What D-04 actually needs — "recompute only once movement settles, not per-frame" — maps directly onto listening for `ml.MapEventCameraIdle` in `onEvent`, which **only fires once per gesture/pan settling**, not per rendered frame. This is a naturally-debounced trigger by construction (no `Timer` required) and is the exact same event `map_screen.dart` already uses for camera-position persistence — reuse that event type, not a manual timer.

**Example:**
```dart
// Source: pattern derived from app/lib/routes/map_screen.dart line 429
// (MapEventCameraIdle already used there for camera persistence — same
// event type, applied here to region-swap recompute instead)
onEvent: (event) {
  // ...existing event handling...
  if (event is ml.MapEventCameraIdle) {
    final controller = _controller;
    if (controller == null) return;
    final bounds = controller.getVisibleRegion();
    _recomputeRegionSources(bounds); // triggers _swapStyle() with new paths
  }
},
```

### Anti-Patterns to Avoid
- **Recomputing region sources on `MapEventMoveCamera` (fires every frame) or `MapEventStartMoveCamera`:** would defeat the entire purpose of D-04's debounce — only `MapEventCameraIdle` settles once per gesture.
- **Feeding `localTilePathsForBounds`'s current merged list straight into `cellPaths`:** see Critical Data-Shape Gap above — this is a real, not hypothetical, correctness bug waiting to happen.
- **Building a `Timer`-based debounce to satisfy D-04's wording literally:** unnecessary — `MapEventCameraIdle` is already the correct, idiomatic, already-precedented trigger in this codebase; a `Timer` would be redundant complexity solving a problem the native SDK's own event model already solves.
- **Coupling the incremental-source-swap decision to `offline_style_rewriter.dart`'s contract:** per D-06, that function stays JSON-in/JSON-out. If the spike ultimately favors incremental composition, build that as an entirely separate typed-object path — do not bend the pure function's signature to serve both strategies at once.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Bbox → local tile paths | A new query method | `TileRepositoryManager.localTilePathsForBounds` (fixed per the gap above) | Already exists, already tested via harness, explicitly built (per its own doc comment) to feed this exact phase |
| Style JSON offline rewrite | A parallel rewriter for region-sourced paths | `offline_style_rewriter.dart`'s `rewriteStyleForOffline` (unchanged, D-06) | Contract is already path-list-agnostic (`cellPaths`/`demCellPaths` are opaque strings) — no `Trail` coupling to remove |
| Debounced viewport recompute | A `Timer`/`Debouncer` utility class | `ml.MapEventCameraIdle` in the existing `onEvent` callback | Native SDK already fires this exactly once per settled gesture; a manual timer would duplicate what the platform already guarantees |
| Region-download-completion → live style swap | A new stream/callback wired through `TileRepositoryManager` | `ref.listen(regionListNotifierProvider, ...)` | Already invalidated by every mutation in `settings_offline_regions_screen.dart`; reuse the existing seam |

**Key insight:** every piece of infrastructure RENDER-01/02/03 need already exists from Phases 22-24 except one signature fix. This phase is almost entirely wiring, which raises the cost of the one real gap (the vector/DEM merged-list mismatch) disproportionately — it's easy to miss precisely because everything *around* it looks done.

## Common Pitfalls

### Pitfall 1: Merged vector/DEM path list silently mis-sources a DEM archive as a vector cell
**What goes wrong:** See Critical Data-Shape Gap section above. `localTilePathsForBounds` returns one `List<String>`; naively assigning it to `cellPaths` (vector) would route a DEM `.pmtiles` file through `_pointSourceAtCell` instead of `_pointDemSourceAtCell`, dropping the required `encoding: terrarium`/`tileSize: 512` injection.
**Why it happens:** The method's name and doc comment ("local vector/DEM archive file paths") don't make the merged-vs-split shape obvious at a glance; ARCHITECTURE.md (written before Phase 23 shipped) assumed a split shape, creating a documentation/reality mismatch a planner could easily inherit.
**How to avoid:** Fix `localTilePathsForBounds`'s return type before wiring either call site (see recommended fix above). Add a unit test asserting a DEM-only region's path lands in the DEM list, not the vector list.
**Warning signs:** Hillshade rendering with garbage/misaligned relief offline (same symptom as the already-fixed quick-260711-lzb bug), or a native crash from `_pointSourceAtCell` treating a DEM archive's tile scheme as vector.

### Pitfall 2: `addSource`/`addLayer` throw on a duplicate id if the incremental strategy is chosen
**What goes wrong:** Both Android and iOS `StyleController` implementations explicitly check for an existing id and throw `Exception('A Source/Layer with the id "..." already exists...')`.
**Why it happens:** Easy to overlook when a region re-enters the viewport (pan away and back) — the incremental-swap code must track currently-active ids and skip/remove-then-add rather than blindly calling `addSource` again.
**How to avoid:** Maintain an explicit `Set<String>` (or similar) of currently-materialized source/layer ids in the compose helper's state; only call `addSource`/`addLayer` for ids not already present, and always `removeLayer` (all layers referencing a source) before `removeSource`.
**Warning signs:** Uncaught exceptions when re-panning across a previously-visited region boundary — only reachable if the incremental strategy is chosen over the recommended full-reload default.

### Pitfall 3: `removeLayer`/`removeSource` on Android have no existence guard
**What goes wrong:** `StyleControllerAndroid.removeLayer`/`removeSource` call the native JNI method directly (`_jStyle.removeLayer(id)`) with no null-check first, unlike iOS which no-ops safely on a missing id.
**Why it happens:** An asymmetry between the two platform implementations that isn't visible from the shared Dart interface — code that "works on iOS" during dev could throw on Android.
**How to avoid:** Wrap incremental remove calls in try/catch (or check `getLayerIds()`/an equivalent existence check first) if the incremental strategy is ever adopted; not a concern for the recommended full-reload default.
**Warning signs:** Android-only crashes when removing a source/layer that (due to a state-tracking bug) was already gone.

### Pitfall 4: D-04's "debounce" language could lead to an unnecessary `Timer` implementation
**What goes wrong:** Building a manual `Timer`-based debounce because CONTEXT.md says "debounced," when `map_screen.dart` — the pattern D-04 says to mirror — doesn't actually use one anywhere (see Pattern 2 above).
**Why it happens:** CONTEXT.md's own description of the "existing... debounce pattern" doesn't precisely match what's in the codebase; a planner trusting that description literally without checking the source would build unneeded complexity.
**How to avoid:** Use `ml.MapEventCameraIdle` directly — it is the debounce, no `Timer` needed.
**Warning signs:** A `Timer.periodic`/`Timer(Duration(...))` appearing in `navigation_screen.dart` for this feature — treat as a signal the wrong precedent was followed.

### Pitfall 5: `setStyle`/incremental mutation both drop previously-added layers, requiring re-add discipline
**What goes wrong:** `TrailMap`'s own doc comment on `_trailLayer.add`/`_onStyleLoaded` already notes "Re-add track + arrows after every style load — `setStyle` drops them." The same is true for `navigation_screen`'s breadcrumb source/layer and trail track. Any new region-swap recompute path that calls `setStyle` (full-reload strategy) must re-run the exact same "re-add everything native-GL-layer-based" step `_onStyleLoaded` already performs — it is easy to build the region-swap path as a standalone code path that forgets this and silently drops the trail track/breadcrumb on the first region-boundary crossing.
**Why it happens:** The existing `_onStyleLoaded` re-add logic is triggered from the `onStyleLoaded` callback specifically for the mount-time/theme-toggle style loads; a new debounced camera-idle-triggered `setStyle` call will also fire `onStyleLoaded` and SHOULD reuse the exact same handler — but only if the implementation routes through `controller.setStyle(json)` (which does trigger `onStyleLoaded`) rather than some other path.
**How to avoid:** Reuse `controller.setStyle(json)` (not a lower-level bypass) for the full-reload strategy so the existing `onStyleLoaded` → `_onStyleLoaded()` → re-add-track/breadcrumb chain fires automatically, exactly as it already does for theme-toggle swaps.
**Warning signs:** Trail track or breadcrumb line disappearing specifically after panning across a region boundary (but present on initial load and on theme toggle) — a sign the region-swap path isn't routing through the same `setStyle`/`onStyleLoaded` chain.

### Pitfall 6: `TrailMap`'s one-time bounds query means a region download completing after the screen mounts won't recompute the *bounds query itself* — only the provider watch does
**What goes wrong:** D-05 fixes `TrailMap`'s bounds query as `trail.bounds`, queried once. If `regionListNotifierProvider` is watched (Pattern 1 above) but the *bounds* passed into `localTilePathsForBounds` are cached from first build rather than recomputed on each provider-change, a stale bounds value could be reused (not a bug for `TrailMap` since `trail.bounds` never changes for a mounted trail, but worth being explicit about in the plan so it isn't confused with `navigation_screen`'s live-viewport case).
**Why it happens:** The two call sites have genuinely different "what varies" semantics (D-06) — `TrailMap`'s bounds are fixed, only the *paths available for that fixed bbox* can change; `navigation_screen`'s bounds AND paths both change.
**How to avoid:** In `TrailMap._composeStyle`, re-call `localTilePathsForBounds(widget.trail.bounds)` fresh on every `_swapStyle()` invocation (cheap — it's a linear ObjectBox scan) rather than caching the path lists — the *query* stays fixed, but its *result* must be re-evaluated each time `regionListNotifierProvider` changes.
**Warning signs:** A trail's map not updating to show a region's basemap after that region finishes downloading while the trail detail screen is already open.

## Code Examples

### Bounds source for each call site
```dart
// TrailMap — D-05, one-time, from Trail's own stored bbox fields
final bounds = widget.trail.bounds; // ml.LngLatBounds, exact type match with
                                     // TileRepositoryManager.localTilePathsForBounds(LngLatBounds)

// navigation_screen — D-06, live viewport, resolved on MapEventCameraIdle
final bounds = controller.getVisibleRegion(); // ml.MapController.getVisibleRegion() -> LngLatBounds
```
Both are directly type-compatible with `TileRepositoryManager.localTilePathsForBounds(LngLatBounds query)` — no conversion layer needed (confirmed via `app/lib/models/trail.dart:108` and `maplibre_platform_interface-0.3.5/lib/src/map_controller.dart:98`).

### Existing precedent for the "re-add after setStyle" pattern to preserve
```dart
// Source: app/lib/components/base/trail_map.dart, existing code (unchanged by this phase)
void _onStyleLoaded(ml.StyleController style) {
  _fitInitialCamera().ignore();
  // Re-add track + arrows after every style load — setStyle drops them.
  if (widget.showTrail && widget.trail.expand?.gpx != null) {
    _trailLayer.add(style, widget.trail).ignore();
  }
}
```
A region-swap `setStyle` call routes through this exact same handler automatically — no new re-add logic needed if the full-reload strategy calls `controller.setStyle(json)` as today's theme-toggle swap already does.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `Trail.pmTiles`/`demPmTiles` — trail-scoped, per-cell arrays populated at trail-download time | `TileRepositoryManager.localTilePathsForBounds(bounds)` — app-wide region registry, resolved per-render | This phase (RENDER-01) | Trail download no longer needs its own tile step (Phase 26/27 will formalize this); rendering becomes decoupled from any specific trail's download history |
| Style composed once at trail-open, static for the screen's lifetime | Style recomposed on region-list change (`regionListNotifierProvider`) and, for `navigation_screen`, on viewport change (`MapEventCameraIdle`) | This phase (RENDER-02) | More frequent style reload than the trail-scoped design was validated for — see Pitfall 5 above for the re-add-track discipline this requires |

**Deprecated/outdated:** `.planning/research/ARCHITECTURE.md`'s Q1 assumption that `localTilePathsForBounds` returns a pre-split `({vectorPaths, demPaths})` record — superseded by direct inspection of the Phase-23-shipped implementation (see Critical Data-Shape Gap).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Android's `removeLayer`/`removeSource` throw (not silently no-op) on a missing id, based on the absence of a null-check in the JNI wrapper's Dart code | Pitfall 3 | If wrong (native side actually no-ops gracefully), the recommended try/catch defensiveness is merely unnecessary, not harmful — low risk either way |
| A2 | The native GL layer-must-be-removed-before-its-source constraint applies to this specific `maplibre` 0.3.5 binding (based on general MapLibre/Mapbox GL native SDK behavior, not verified against this binding's exact enforcement) | RENDER-03 Spike Findings, point 5 | Only relevant if the incremental strategy is chosen (not the recommended default); if unenforced, removal-ordering discipline is extra caution rather than a requirement — low risk |

**Both entries are LOW risk and only apply if the (non-default, spike-gated) incremental composition strategy is chosen over the recommended full-reload default** — neither blocks planning the recommended path.

## Open Questions

1. **Does `pmtiles://` protocol resolution work identically whether a source arrives via a full `setStyle(json)` document or via `StyleController.addSource(VectorSource(url: 'pmtiles://file://...'))`?**
   - What we know: The full-reload path is already proven (current production code uses `pmtiles://file://` inside JSON style documents applied via `setStyle`). The incremental path's `VectorSource`/`RasterDemSource` classes accept a plain `url` string field, forwarded to the same native `Source` constructors either way.
   - What's unclear: Whether the `pmtiles://` custom URL-scheme handler is registered at a point in the native lifecycle (e.g., map/style creation) that also covers sources added incrementally after initial load — not verified from source inspection alone.
   - Recommendation: Only relevant if the incremental strategy is chosen; the on-device spike (Task 1) should explicitly test `addSource(VectorSource(url: 'pmtiles://file://...'))` if incremental composition is being evaluated, not just the full-reload path.

2. **What is the actual layer-count/performance ceiling on a mid-tier Android device with 10-20 duplicated region source/layer sets?**
   - What we know: PITFALLS.md's Pitfall 4 flags this as MEDIUM confidence from JS-binding (maplibre-gl-js) community reports, not native-binding-specific.
   - What's unclear: The exact threshold for this native binding and this app's typical style complexity (basemap layers × N regions).
   - Recommendation: This is precisely ROADMAP's success criterion 1 — must be a genuine on-device Task 1, cannot be resolved by further static research.

## Security Domain

`security_enforcement: true` per `.planning/config.json`, but this phase has no new external input surface, no auth, no network calls, and no new user-facing form/data-entry. The only "input" is region-catalog-derived local file paths (already validated end-to-end by Phase 22/23's `region_file_path.dart` `assertValidRegionId` and `offline_style_rewriter.dart`'s own `_assertSafePath` path-traversal/scheme guard, both unchanged by this phase).

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|----------------|---------|-------------------|
| V5 Input Validation | Indirect — no new input, but existing path-safety controls must remain intact | `offline_style_rewriter.dart`'s `_assertSafePath` (unchanged, D-06) continues to reject non-absolute/`..`/foreign-scheme paths for every path this phase's new data source (`localTilePathsForBounds`) feeds in |
| V6 Cryptography | No | Not applicable — no crypto in this phase |
| V2/V3/V4 Auth/Session/Access Control | No | Not applicable — local, offline-only rendering pipeline; no auth boundary crossed |

### Known Threat Patterns for this stack
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| A corrupted/malicious `RegionEntity.localFilePath` value escaping the app sandbox via a `..`-traversal or foreign-scheme string | Tampering | Already covered — `offline_style_rewriter.dart`'s `_assertSafePath` runs on every `cellPath`/`demCellPath` before any path enters the style JSON, unchanged by this phase. No new validation needed; just confirm the split-return fix (Critical Data-Shape Gap) still routes both lists through the same rewriter call. |

## Sources

### Primary (HIGH confidence — direct source inspection)
- `~/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/style_controller.dart` — abstract `StyleController` contract (`addSource`/`removeSource`/`addLayer`/`removeLayer`/`updateGeoJsonSource`)
- `~/.pub-cache/hosted/pub.dev/maplibre_android-0.3.5/lib/src/style_controller.dart` — Android JNI implementation, confirmed real (not stub), confirmed duplicate-id throw behavior
- `~/.pub-cache/hosted/pub.dev/maplibre_ios-0.3.5/lib/src/style_controller.dart` — iOS Obj-C/FFI implementation, confirmed real, confirmed missing-id no-op on remove (asymmetric with Android)
- `~/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/style/sources/vector_source.dart`, `raster_dem_source.dart` — confirmed typed `Source` classes cover every field `offline_style_rewriter.dart` injects (`maxZoom`, `tileSize`, `encoding: RasterDemTerrariumEncoding()`)
- `~/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/map_controller.dart` — `getVisibleRegion() -> LngLatBounds`, `setStyle(String)`
- `~/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/map_events.dart` — `MapEventCameraIdle`, `MapEventStartMoveCamera`, `MapEventMoveCamera` class hierarchy
- `app/lib/components/base/trail_map.dart` — existing `_composeStyle`/`_swapStyle`/`_onStyleLoaded`/`_fitInitialCamera` (lines 66-303)
- `app/lib/routes/navigation_screen.dart` — existing `_composeStyle` (lines 863-909), `_onStyleLoaded` (lines 940-982), confirmed live `style.addSource`/`addLayer` usage for the breadcrumb layer
- `app/lib/util/offline_style_rewriter.dart` — full read, confirmed `cellPaths`/`demCellPaths` separate-parameter contract, confirmed `_assertSafePath` path-safety guard
- `app/lib/services/tile_repository_manager.dart` — full read, confirmed `localTilePathsForBounds`'s actual (merged-list) return shape
- `app/lib/routes/map_screen.dart` — full read of the cluster-search/camera-idle/pan-start event handling (lines 1-560), confirmed no `Timer`-based debounce exists; confirmed `MapEventCameraIdle` used only for camera-position persistence
- `app/lib/provider/region/region_provider.dart`, `app/lib/provider/region/tile_repository_provider.dart` — confirmed `regionListNotifierProvider`/`tileRepositoryManagerProvider` names and invalidation seam
- `app/lib/routes/settings_offline_regions_screen.dart` — confirmed 4 `ref.invalidate(regionListNotifierProvider)` call sites after region mutations
- `app/lib/models/trail.dart` — confirmed `Trail.bounds` returns `ml.LngLatBounds` directly, type-compatible with `localTilePathsForBounds`
- `app/lib/util/region_file_path.dart` — confirmed fixed `vector.pmtiles`/`dem.pmtiles` naming convention
- `app/test/services/tile_repository_manager_test.dart`, `tile_repository_manager_harness.dart` — confirmed no unit test currently exercises `localTilePathsForBounds`'s return shape

### Secondary (MEDIUM confidence)
- `.planning/research/ARCHITECTURE.md`, `.planning/research/PITFALLS.md`, `.planning/research/SUMMARY.md` — milestone-level research, largely confirmed by this phase's direct code inspection, with one correction (the `localTilePathsForBounds` return-shape assumption) and one confidence upgrade (RENDER-03's API-availability question, LOW/MEDIUM → HIGH)

### Tertiary (LOW confidence, flagged in Assumptions Log)
- Android `removeLayer`/`removeSource` missing-id behavior (A1) — inferred from absence of a null-check, not tested against the native SDK directly
- Native GL layer-before-source removal ordering constraint for this specific binding (A2) — general native-SDK-behavior inference, not verified against this binding's exact enforcement

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies, all pre-existing and already proven in this codebase
- Architecture: HIGH — provider composition pattern directly grounded in existing `TrailMap`/`navigation_screen`/`region_provider.dart` code, not speculative
- RENDER-03 API question: HIGH — resolved via direct pub-cache source read across all three platform packages, cross-confirmed by existing production usage in `navigation_screen.dart`
- RENDER-03 performance/layer-count threshold: LOW — genuinely unresolvable without an on-device spike; flagged, not assumed
- Pitfalls: HIGH for the data-shape gap (directly confirmed via source read of the actual Phase 23 implementation); MEDIUM for the platform-asymmetry pitfalls (inferred from binding source, not native-SDK-tested)

**Research date:** 2026-07-23
**Valid until:** Until `maplibre` is unpinned from 0.3.5 (no other freshness concern — all findings are direct source reads of already-installed, version-locked code)
