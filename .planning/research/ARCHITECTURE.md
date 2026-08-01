# Architecture Research

**Domain:** Region-based offline map/tile repository — Flutter + Riverpod + ObjectBox + MapLibre-native-GL, replacing a trail-scoped download model
**Researched:** 2026-07-21
**Confidence:** HIGH (all findings verified against current repo source, not training-data assumptions)

## Naming correction (read this first)

The milestone brief's file list (`wanderer_map.dart`, `search_map.dart`) doesn't match the actual repo. The real map-host components are:

| Brief's name | Actual file | Role |
|---|---|---|
| `wanderer_map.dart` (single-trail) | `app/lib/components/base/trail_map.dart` (`TrailMap`) | Single-`Trail` host: offline style rewrite, bounds-fit, trail track layer. Used by `trail_detail_map_screen.dart`. |
| `wanderer_map.dart` (collection) | `app/lib/components/base/trail_collection_map.dart` (`TrailCollectionMap`) | Trail-agnostic host, no offline branch today. Used by `map_screen.dart`, `list_detail_map_screen.dart`, `list_detail_screen.dart`. |
| `search_map.dart` | Doesn't exist as a separate file — `map_screen.dart` and `list_detail_screen.dart` build directly on `TrailCollectionMap`. |
| `trail_layer.dart` | `app/lib/components/map/trail_layer.dart` (`TrailLayer`) — confirmed, draws the GPX track + arrows via `MapController`/`StyleController`, not style-JSON rewriting. |

`navigation_screen.dart` and `route_planner_screen.dart` build `ml.MapLibreMap` directly (not via `TrailMap`/`TrailCollectionMap`) and duplicate their own copy of the offline-style-compose logic — `navigation_screen.dart` calls `rewriteStyleForOffline` inline at line ~878, keyed off `trailProvider(widget.id).pmTiles`/`.demPmTiles`. This is the second of two current call sites of `rewriteStyleForOffline` (the first is `TrailMap._composeStyle`). Both must move to the region-based lookup.

`map_style_json_provider.dart` (`mapStyleJsonProvider`) is untouched by this milestone in principle — it only resolves the *online base style* (theme + operator tile/glyph/sprite URLs from `mapStyleSourcesProvider`). The offline rewrite is a separate, downstream step layered on top of its output by each consuming widget. This separation is exactly why the region work is additive to it rather than a replacement.

## Answering the four questions

### Q1 — Decoupling the style rewriter and map screens from `TrailEntity`

**Yes: a single `TileRepositoryManager`-backed Riverpod provider, but the interface it exposes to callers should be "coverage for a bbox," not "coverage for a viewport," and it must stay a thin second layer on top of the existing `mapStyleJsonProvider`, not a replacement for it.**

Today both offline call sites (`TrailMap._composeStyle`, `navigation_screen`'s inline compose) do the same three things: (1) watch `mapStyleJsonProvider` for the base online style, (2) watch `glyphSpriteCacheProvider` for the local glyph/sprite root, (3) pull `cellPaths`/`demCellPaths` from the `Trail` model (itself sourced from `TrailEntity.pmTiles`/`demPmTiles`). Step (3) is the only trail-scoped part — steps (1) and (2) are already app-wide and reusable as-is.

Introduce:
- `regionTileRepositoryProvider` (or similar) — a `@Riverpod(keepAlive: true)` provider wrapping a `TileRepositoryManager` instance (owns ObjectBox `Region`/`DownloadedTilePackage` boxes + `Dio`, mirrors the existing `objectBoxProvider`/`apiProvider` composition pattern used by `TrailDownloadServiceNotifier`).
- A query method/provider such as `localTilePathsForBounds(LngLatBounds bounds) -> ({List<String> vectorPaths, List<String> demPaths})`, resolved by intersecting the bbox against every `downloaded`-status `Region` in ObjectBox (bbox-only regions per this milestone's scope — no polygon geometry, no spatial index needed; a linear scan over what will realistically be a few dozen regions is fine).
- Callers (`TrailMap`, `navigation_screen`, and — newly — `TrailCollectionMap` if it grows an offline mode) replace `widget.trail.pmTiles`/`demPmTiles` with a call to this query, using the same bounds source they already compute (`trail.bounds`, or the current viewport/camera bounds for a general map screen). `rewriteStyleForOffline` itself needs **no signature change** — it already takes `cellPaths`/`demCellPaths` as opaque path lists; only the caller-side source of those lists changes.
- Keep the compose helper pattern (`_composeStyle`) each widget already has; just swap its data source. This avoids a state-management redesign — the `ref.listen(mapStyleJsonProvider, ...)` + `ref.listen(<offline source>, ...)` dual-listen pattern `TrailMap` already established for `glyphSpriteCacheProvider` extends directly to a new `ref.listen(regionTileRepositoryProvider, ...)` so a region finishing download mid-session live-swaps the style, exactly like a glyph-cache-warm completion does today.

For a viewport-scale host like `TrailCollectionMap` (`map_screen.dart`), "what covers this viewport" should be resolved on **camera idle/moveend**, not continuously — mirror the debounce pattern likely already used for the cluster-search bbox query in `map_screen.dart`, not a synchronous per-frame recompute.

**Do not** make `TileRepositoryManager` reach into `mapStyleJsonProvider` or vice versa — keep them composable, independently testable layers, matching this repo's existing separation of "resolve online style" (`map_style_json_provider.dart`) vs. "resolve offline glyph/sprite cache" (`glyph_sprite_cache_provider.dart`) vs. "rewrite style for offline" (`offline_style_rewriter.dart`, a pure function). The region provider slots in as a fourth, parallel data source feeding the same pure `rewriteStyleForOffline` function — it doesn't change that function's contract.

### Q2 — Does the backend grid-cell system map onto "region," or does this need new backend concepts?

**Partial reuse — cells stay the atomic backend unit, but "region" is a new, purely additive backend/manifest concept layered on top, not a renaming of `GridCell`.**

Evidence for reuse:
- `db/services/tiles/grid.go`'s `GridCell` (0.5° cells, `CacheKey()` = `"%.2f_%.2f_%.2f_%.2f"`) and `db/services/tiles/generator.go`'s `EnsureCell`/`CellPath`/`DemCellPath` are bbox-driven and trail-agnostic already — `EnsureCell` takes a `GridCell`, not a `TrailEntity` or `Trail`. Nothing here special-cases trails; the trail-scoping lives entirely on the **client** (`trail_download_service.dart` converts a trail's bbox → cell list via `GET /map/cells?bbox=...`). This means the backend requires **zero changes** to serve region downloads — a region is just a (typically larger) bbox that maps to more grid cells via the same `BboxToGridCells` math and the same `/map/cells?bbox=`, `/map/cells/{cellKey}/download[-dem]` endpoints already exposed by `db/routes/map_cells_id.go`.
- A "region" in this milestone is explicitly **bbox-only** (per PROJECT.md's Out of Scope: "Polygon region geometries... deferred"), which is exactly the shape `BboxToGridCells` already consumes.

Evidence a new concept is still needed:
- There is no backend notion of a *named, curated area* (e.g. "Yosemite Valley," with a stable id, display name, and a size estimate the app can show before downloading). `tile_cells` records are anonymous, keyed only by bbox rounding — they have no name, no grouping, no pre-computed total size for a multi-cell area.
- PROJECT.md is explicit: **"Bundled `regions.json` manifest... Remote/server-fetched region manifest — v1.6 ships a bundled `regions.json` app asset only."** This settles the question directly: the region *catalog* (id, name, bbox, vector/DEM URL+size) is a **client-side, checked-in app asset** (`assets/map/regions.json`, fits the already-declared `assets/map/` pubspec glob), not a new backend collection/endpoint. No new PocketBase collection, no new SvelteKit route, no new Go route is required for v1.6.
- The "vector PMTiles URL/size, optional DEM URL/size" fields in `regions.json` should point at the **existing** `/map/cells?bbox=<region bbox>` → per-cell `download_url`/`dem_download_url` flow (i.e., the manifest doesn't need a pre-baked download URL per region; it needs a bbox, and the client re-derives the cell list + per-cell URLs from the existing endpoint exactly as `trail_download_service.dart` does today, just called with the region's bbox instead of a trail's bbox). Size can be precomputed once (server-side script or manual entry) and shipped as a static number in the manifest — it doesn't need to be live-queried, since `tile_cells.size_bytes` is only known after a cell has actually been generated once, which the manifest curator would do out-of-band before publishing.

**Net conclusion:** No backend/Go changes are required by this milestone. The multi-cell download orchestration currently inside `TrailDownloadService._downloadMapTiles` (fetch cell list for bbox → poll pending → download vector + best-effort DEM per cell) is the piece to **extract and reuse**, not reinvent — it should move into `TileRepositoryManager` verbatim (parameterized by region bbox instead of trail bbox), since it already implements exactly the multi-cell, best-effort-DEM, progress-tracked download this milestone needs region-side.

### Q3 — Trail-download-guard integration point

**Single choke point already exists: `DownloadingTrailIds.download(Trail trail)` in `app/lib/provider/trail/trail_download_state_provider.dart`.**

Every trail-download UI entry point (detail-screen button, dropdown menu item — per that file's own doc comment: "shared across every download entry point") already funnels through this one Riverpod notifier method before calling `trailDownloadService.downloadTrail(trail, ...)`. This is the correct and only integration point needed:

1. At the top of `DownloadingTrailIds.download()`, before the existing `if (state.contains(trail.id)) return;` dedup check (or immediately after it), compute `trail.bounds` and query the region provider's coverage check (e.g. `regionTileRepositoryProvider.isBboxFullyCovered(trail.bounds)` or reuse the same `localTilePathsForBounds` query from Q1 and check it returned a non-empty, bounds-spanning result).
2. If uncovered, **do not** silently proceed with the old per-trail cell download — PROJECT.md's requirement is "prompt to download the covering region if missing." This means the guard returns/throws a distinguishable "needs region" signal that the calling widget (detail screen button) surfaces as a dialog ("Download the {region name} region to enable offline use of this trail" with a CTA into the new Settings → Offline Maps page), rather than the notifier silently downloading tiles itself.
3. Once a covering region is confirmed downloaded (either already present, or the user completes the region download and returns), `trailDownloadService.downloadTrail` no longer needs its own `_downloadMapTiles` step at all — per PROJECT.md, trail-scoped tile download is deleted outright. The trail download becomes photo-only + Valhalla-nav-cache-only; map rendering for that trail leans entirely on the covering region's already-downloaded tiles via the Q1 query.

This has a real behavioral consequence worth flagging for roadmap phasing: once trail-scoped tile download is removed, **`downloadTrail` can no longer succeed offline-renderable without a region present**. That's an intentional product change (the whole point of the milestone), but it means the guard is not just a UX nicety — it's now load-bearing for the "offline parity" hard gate PROJECT.md's Constraints section still lists ("a downloaded trail must render basemap and labels with no network"). The guard must block, not just warn, or that gate silently breaks.

### Q4 — Safest build order (keep the app buildable at every step)

This project has one directly-precedented incremental-migration pattern to reuse: the v1.4 MapLibre migration kept **both stacks coexisting** until the last screen was migrated, and explicitly deferred file deletion (`pm_tile_provider.dart`) to the phase where the last consumer was gone (see PROJECT.md Key Decisions: "Incremental screen-by-screen migration, forks deleted last" and "`pm_tile_provider.dart` deletion deferred... deleting the file in Phase 15 would have broken the screen the criterion required to keep building"). Apply the same discipline here: **new region code lands and becomes the active path before any trail-scoped code is deleted**, and deletion is its own final phase.

Recommended order:

1. **Data model phase** — `Region` + `DownloadedTilePackage` ObjectBox `@Entity()` classes (mirror `TrailEntity`'s field/`ToOne`/`ToMany` conventions; no backlink to `TrailEntity` needed — this is the decoupling point), plus the bundled `assets/map/regions.json` manifest and its freezed parse model (mirror `MapCellInfo`'s `@freezed` + `fromJson` pattern in `app/lib/models/map_cell.dart`). Nothing consumes these yet — app builds unchanged, this phase is purely additive.
2. **`TileRepositoryManager` phase** — new service class (mirror `TrailDownloadService`'s constructor-injected `Store _store, Dio _api` pattern) implementing region download (lift `_downloadMapTiles`'s cell-list/poll/download-with-progress logic out of `TrailDownloadService` into a shared/parameterized form, or duplicate it initially and dedupe later — duplicating first is lower-risk and matches this codebase's general preference for working code over premature abstraction) + the `localTilePathsForBounds`-style query provider from Q1. Still nothing in the UI calls this yet — still buildable, still a no-op addition.
3. **UI phase** — Settings → Offline Maps/Regions screen (list, download/pause/resume/delete, DEM toggle, disk usage) driven by the Phase 2 manager. This is now user-visible and independently testable/demoable without touching any existing map screen or trail flow.
4. **Map-screen rewiring phase** — swap `TrailMap._composeStyle` and `navigation_screen`'s inline compose from `trail.pmTiles`/`demPmTiles` to the Phase 2 region query, **while leaving `TrailEntity.pmTiles`/`demPmTiles` and `trail_download_service.dart`'s tile-download step physically in place** (now unused by rendering, but still present so nothing breaks if a trail happens to have old-format cached tiles from before this migration — moot in practice since PROJECT.md says pre-production/no-migration-needed, but keeping deletion as a separate step is what kept v1.4 buildable at every boundary and should be repeated here). At the end of this phase, region-based rendering is live and is the only thing actually exercised by manual testing, even though the old fields/code are technically still compiled in.
5. **Guard phase** — wire `DownloadingTrailIds.download()`'s pre-check (Q3) now that Phase 4 proves region coverage is reliably queryable and Phase 3's UI gives the guard's "prompt to download" dialog somewhere to send the user.
6. **Ripout phase, last** — delete `trail_download_service.dart`'s `_downloadMapTiles`/`_fetchCellList`/`_pollUntilReady` and the cell-download portion of `downloadTrail`, delete `TrailEntity.pmTiles`/`demPmTiles` fields (ObjectBox migration note: removing fields from an `@Entity()` is safe/non-breaking in ObjectBox — old data for removed fields is simply ignored, no explicit migration step needed, unlike a rename), delete `MapCellInfo`/`MapCellStatusResponse`-consuming code paths that were trail-specific (the models themselves likely stay if `TileRepositoryManager` reuses the same `/map/cells` response shape — check before deleting `app/lib/models/map_cell.dart` wholesale, since Phase 2's manager should be consuming those same DTOs for region cell fetches).

This order's key property: **every phase boundary leaves exactly one thing user-visibly different, and nothing is deleted before its replacement is proven live** — matching the "app must stay buildable/runnable at every phase boundary" constraint this project has already validated once in a comparably-sized migration (v1.4, 6 screens across ~4 phases).

## Standard Architecture

### System Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│                          Settings UI (new)                            │
│  Offline Maps/Regions screen — list, download/pause/resume/delete,    │
│  DEM toggle, disk usage                                               │
└───────────────────────────────┬───────────────────────────────────────┘
                                 │ reads/commands
┌───────────────────────────────▼───────────────────────────────────────┐
│                    TileRepositoryManager (new)                        │
│  owns: Region + DownloadedTilePackage ObjectBox boxes, Dio             │
│  does: region download orchestration (reused cell-download logic),    │
│        bbox → local-path resolution (localTilePathsForBounds)         │
└───────┬───────────────────────────────────────────────────┬───────────┘
        │ query (bbox → paths)                               │ downloads via
┌───────▼─────────────────┐  ┌─────────────────────┐  ┌──────▼───────────┐
│ TrailMap (existing)      │  │ navigation_screen     │  │ /map/cells...     │
│ _composeStyle: base style│  │ (existing, inline     │  │ (existing, no     │
│ + glyph/sprite cache +   │  │ compose): same swap    │  │ backend changes)  │
│ [Q1 swap] region paths   │  │                        │  │                   │
└───────┬───────────────────┘  └────────────────────────┘  └───────────────────┘
        │ pure transform
┌───────▼───────────────────────────────────────────────────────────────┐
│              offline_style_rewriter.dart (unchanged contract)          │
│  cellPaths/demCellPaths in → pmtiles://file:// style JSON out          │
└──────────────────────────────────────────────────────────────────────┘

        Trail download flow (guard inserted, Q3):
┌──────────────────────────┐   pre-check    ┌───────────────────────────┐
│ DownloadingTrailIds       │───────────────▶│ region coverage query      │
│ .download(trail)          │  bbox covered? │ (same manager as above)    │
└──────────┬─────────────────┘  no → prompt   └───────────────────────────┘
           │ yes / covered
┌──────────▼─────────────────┐
│ trailDownloadService        │  (post-ripout: photos + Valhalla cache
│ .downloadTrail(trail)       │   only — tile download step removed)
└──────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | File (new/modified) |
|-----------|----------------|------------------------|
| `Region` entity | Manifest-sourced region metadata + live download status | **new** `app/lib/entities/region_entity.dart` |
| `DownloadedTilePackage` entity | Per-region local file paths, sizes, timestamps (vector + optional DEM) | **new** `app/lib/entities/downloaded_tile_package_entity.dart` |
| `regions.json` + parse model | Bundled catalog of downloadable regions (id, name, bbox, URLs/sizes) | **new** `assets/map/regions.json` + `app/lib/models/region_manifest.dart` |
| `TileRepositoryManager` | Region download orchestration, bbox→local-paths resolution, disk usage accounting | **new** `app/lib/services/tile_repository_manager.dart` |
| Region Riverpod providers | Expose manager + reactive region list/coverage queries to widgets | **new** `app/lib/provider/tile_repository_provider.dart` (mirrors `trail_download_provider.dart`'s notifier-wrapping-a-service pattern) |
| `TrailMap._composeStyle` | Swap trail-bbox tile source for region-query tile source | **modified** `app/lib/components/base/trail_map.dart` |
| `navigation_screen` inline compose | Same swap | **modified** `app/lib/routes/navigation_screen.dart` |
| `DownloadingTrailIds.download` | Insert region-coverage guard before trail download proceeds | **modified** `app/lib/provider/trail/trail_download_state_provider.dart` |
| `offline_style_rewriter.dart` | Pure style transform — **no changes**, contract already path-list-based | unchanged `app/lib/util/offline_style_rewriter.dart` |
| `map_style_json_provider.dart` | Online base style resolution — **no changes**, orthogonal layer | unchanged `app/lib/provider/map_style_json_provider.dart` |
| `db/services/tiles/*.go`, `db/routes/map_cells_id.go` | Grid-cell generation/serving — **no changes**, region reuses bbox→cells as-is | unchanged |
| `trail_download_service.dart` | Photo + nav-cache download only, post-ripout | **modified then trimmed** (tile-download methods deleted in final phase) |
| `TrailEntity.pmTiles`/`demPmTiles` | Deleted once region rendering is proven live | **deleted** (final phase) |

## Recommended Project Structure

```
app/lib/
├── entities/
│   ├── region_entity.dart              # new — ObjectBox Region
│   └── downloaded_tile_package_entity.dart  # new — ObjectBox DownloadedTilePackage
├── models/
│   └── region_manifest.dart            # new — freezed parse model for regions.json
├── services/
│   └── tile_repository_manager.dart    # new — download orchestration + query
├── provider/
│   └── tile_repository_provider.dart   # new — Riverpod wrapping the manager
├── routes/
│   └── settings_offline_maps_screen.dart  # new — Settings UI
├── components/base/
│   ├── trail_map.dart                  # modified — swap tile source
│   └── trail_collection_map.dart       # modified only if viewport-mode offline is added
└── util/
    └── offline_style_rewriter.dart     # unchanged
assets/map/
└── regions.json                        # new — bundled manifest (assets/map/ glob already declared)
```

### Structure Rationale

- New region code is siblings of the existing trail-download code (`entities/`, `services/`, `provider/`), not nested under a `trail/` subfolder — this is the literal expression of "decoupled from Trail" and matches how `provider/trail/` is already its own namespace distinct from top-level providers.
- `region_manifest.dart` is deliberately a separate model from the ObjectBox `Region` entity (mirrors the existing `Trail` model vs. `TrailEntity` split) — the manifest is the bundled catalog (static, ships with the app binary), the entity is the live, mutable download-state record. Conflating them would make an app update's new `regions.json` unable to cleanly reconcile against already-downloaded state.

## Architectural Patterns

### Pattern 1: Provider-wraps-service, mirroring `trail_download_provider.dart`

**What:** A thin `@riverpod` (or `@Riverpod(keepAlive: true)`) class whose `build()` composes existing infra providers (`objectBoxProvider`, `apiProvider`) into a plain Dart service class; the service itself has no Riverpod dependency and is independently testable.
**When to use:** Exactly this milestone's `TileRepositoryManager` — same shape as `TrailDownloadServiceNotifier`.
**Trade-offs:** Slightly more indirection than a bare `Provider`, but keeps the service unit-testable without a `ProviderContainer` and matches every existing service in this codebase.

### Pattern 2: Pure-function style rewrite, data-source-agnostic

**What:** `offline_style_rewriter.dart` takes only primitive `List<String>` path lists — it has never known about `Trail` and shouldn't learn about `Region` either.
**When to use:** Keep this boundary. The only thing that changes with this milestone is *what populates* `cellPaths`/`demCellPaths` before calling it.
**Trade-offs:** None — this is already correctly decoupled; the milestone's job is to stop being the one exception (trail-bbox-sourced paths) to an otherwise generic function.

### Pattern 3: Guard-at-the-choke-point

**What:** Insert cross-cutting checks (region coverage) at the single shared notifier method (`DownloadingTrailIds.download`) rather than at each UI call site.
**When to use:** Whenever multiple UI entry points funnel through one state-owning method — this codebase already does this for the download-dedup check; extend it, don't duplicate the coverage check per button.
**Trade-offs:** The guard's "prompt to download" UX (opening a dialog/navigating to Settings) has to be signaled back out of a notifier method rather than handled inline — use a return value or a dedicated toast/dialog provider (the codebase already has `toast_provider.dart` used inside this exact method for error/success messaging; extend that pattern for the "needs region" case rather than inventing a new signaling mechanism).

## Data Flow

### Region download flow

```
Settings UI → TileRepositoryManager.downloadRegion(region)
    ↓
regions.json bbox → GET /map/cells?bbox=... (existing endpoint, unchanged)
    ↓
per cell: poll status → GET .../download (+ best-effort .../download-dem)
    ↓
write to <app-docs>/regions/<regionId>/tiles/<cellKey>[.pmtiles|_dem.pmtiles]
    ↓
ObjectBox: upsert DownloadedTilePackage (paths, sizes, timestamp) + Region.status = downloaded
```

### Map render flow (post-migration)

```
Map widget build() → bounds (trail.bounds or viewport bounds)
    ↓
TileRepositoryManager.localTilePathsForBounds(bounds) → (vectorPaths, demPaths)
    ↓
mapStyleJsonProvider (online base style, unchanged) + glyphSpriteCacheProvider (unchanged)
    ↓
rewriteStyleForOffline(base, cacheRoot, vectorPaths, demPaths) — unchanged contract
    ↓
MapController.setStyle(json) / initStyle
```

### Trail download flow (guarded, post-migration)

```
User taps "Download" → DownloadingTrailIds.download(trail)
    ↓
[NEW] TileRepositoryManager coverage check on trail.bounds
    ↓ covered                              ↓ not covered
trailDownloadService.downloadTrail    show "download region" prompt
(photos + nav cache only, no tiles)   → Settings → Offline Maps
```

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| A handful of curated regions (v1.6 launch) | Linear scan over ObjectBox `Region` box for bbox-intersection is fine — no spatial index needed. |
| Dozens of regions, large countries | Still fine for bbox-only rectangles; if this grows to hundreds, add a simple ObjectBox index on region min/max lat/lon rather than a full spatial engine — premature otherwise. |
| Remote/updatable manifest (explicitly deferred) | Not this milestone — noted only so the `Region`/manifest split (Pattern above) doesn't need rework when that lands later; the reconciliation logic (manifest region vs. already-downloaded entity) is exactly what that split anticipates. |

## Anti-Patterns

### Anti-Pattern 1: Making `TileRepositoryManager` a superset of `TrailDownloadService`

**What people do:** Extend the existing `TrailDownloadService` in place to also handle regions, keeping one class doing both.
**Why it's wrong:** Directly contradicts PROJECT.md's explicit goal ("decoupled from Trail") and the "legacy code deleted outright" requirement — a merged class can't be cleanly deleted down to its trail-only remainder later.
**Do this instead:** New, separate `TileRepositoryManager`; `TrailDownloadService` shrinks to photos + nav cache once the guard/ripout phases land.

### Anti-Pattern 2: Coupling the region query to `mapStyleJsonProvider`

**What people do:** Fold the region path lookup directly into `mapStyleJsonProvider` so there's "one provider that returns the final style."
**Why it's wrong:** `mapStyleJsonProvider` is `keepAlive` and theme-driven only; conflating it with viewport/trail-bounds-driven region data would force it to re-evaluate on every camera move and lose its clean single-responsibility (and its useful independence — e.g. it's reused for both the online and about-to-be-offline-rewritten case).
**Do this instead:** Keep them as parallel inputs composed by the calling widget's own `_composeStyle`, exactly as `glyphSpriteCacheProvider` already is today.

### Anti-Pattern 3: Deleting trail-scoped tile code before region rendering is proven live

**What people do:** Since PROJECT.md says "no migration, delete outright," it's tempting to delete `pmTiles`/`demPmTiles`/`_downloadMapTiles` in the same phase that introduces `TileRepositoryManager`.
**Why it's wrong:** Violates this project's own established and explicitly-praised migration discipline (v1.4's "forks deleted last," "deletion deferred to the phase where the last consumer was gone") — a big-bang swap risks a build break or a silent offline-rendering regression with no fallback to diagnose against.
**Do this instead:** Follow the Q4 order — data model → manager → UI → rewiring → guard → ripout, each phase independently buildable and demoable.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| `build.protomaps.com` / `download.mapterhorn.com` (via Go backend) | Unchanged — `db/services/tiles/generator.go` already abstracts this behind `/map/cells`; region downloads call the same endpoint with a larger bbox. | No new external integration. |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `TileRepositoryManager` ↔ ObjectBox | Direct `Store.box<Region>()`/`box<DownloadedTilePackage>()`, mirrors `TrailDownloadService`'s `_store.box<TrailEntity>()` | Same `Store` instance via `objectBoxProvider`. |
| `TileRepositoryManager` ↔ `/map/cells` API | `Dio` via `apiProvider`, identical request/response DTOs (`MapCellInfoList`, `MapCellStatusResponse`) to what `trail_download_service.dart` already uses | Reuse `app/lib/models/map_cell.dart` as-is; don't fork new DTOs. |
| `TrailMap`/`navigation_screen` ↔ region provider | `ref.watch`/`ref.read` on a new bbox-query provider, replacing direct `trail.pmTiles` field reads | Contract: `(List<String>, List<String>)` in, matching `rewriteStyleForOffline`'s existing parameter shapes — zero change to that function. |
| `DownloadingTrailIds` ↔ region provider | `ref.read` coverage check inside the existing notifier method | Single choke point; no new UI-side duplication needed. |
| Flutter app ↔ Go backend | **No new boundary** — region concept is client-only (bundled manifest); backend continues to only know about grid cells | Confirmed by PROJECT.md's explicit "bundled manifest, no remote manifest" scoping. |

## Sources

- `.planning/PROJECT.md` (v1.6 milestone scope, constraints, Out of Scope, Key Decisions — especially the v1.4 migration-discipline entries this order re-applies)
- `app/lib/services/trail_download_service.dart` (multi-cell download orchestration to be lifted into `TileRepositoryManager`)
- `app/lib/util/offline_style_rewriter.dart` (confirmed path-list-only contract, no `Trail` coupling)
- `app/lib/entities/trail_entity.dart` (fields to be deleted: `pmTiles`, `demPmTiles`)
- `app/lib/components/base/trail_map.dart`, `trail_collection_map.dart` (actual map-host components; corrected against the milestone brief's stale file names)
- `app/lib/routes/navigation_screen.dart` (second, independent offline-rewrite call site — grep-confirmed at line ~874-889)
- `app/lib/provider/trail/trail_download_state_provider.dart` (confirmed single choke point for all trail-download UI entry points — doc comment explicitly states this)
- `app/lib/provider/trail/trail_download_provider.dart`, `app/lib/provider/objectbox_store_provider.dart` (provider-wraps-service pattern to replicate)
- `app/lib/models/map_cell.dart` (DTOs to reuse for region cell fetches, not fork)
- `db/services/tiles/generator.go`, `db/services/tiles/grid.go`, `db/routes/map_cells_id.go` (confirmed bbox/cell-based, trail-agnostic already — no backend changes required)
- `app/pubspec.yaml` (confirmed `assets/map/` glob already covers a new `regions.json` bundled asset with no pubspec change)

---
*Architecture research for: region-based offline map/tile repository (Wanderer Flutter app, v1.6 milestone)*
*Researched: 2026-07-21*
