---
phase: 15-maplibre-core-trail-rendering-offline-parity
plan: 06
subsystem: ui
tags: [maplibre, flutter, offline, pmtiles, glyphs, sprite, file-uri, style-rewrite, path-safety]

# Dependency graph
requires:
  - phase: 15-03
    provides: "glyphSpriteCacheProvider -> GlyphSpriteCachePaths (app-wide file:// cache root under <app-docs>/map_cache); map_cache_path.dart path-safety helpers (spriteCacheBasePath)"
  - phase: 15-04
    provides: "WandererMap on MapLibreMap — mapStyleJsonProvider base JSON feeds initStyle/setStyle; onStyleLoaded seam; live theme swap via cached _lastStyleJson"
  - phase: 15-05
    provides: "addTrailTrackLayers re-added in onStyleLoaded (survives offline style too); self-registered sprite-independent 'arrow' icon"
provides:
  - "rewriteStyleForOffline(style, {cacheRoot, cellPaths, dark}) — pure transform: online style -> offline style with file:// glyphs/sprite + pmtiles://file:// sources (OFFL-02/03), N-source + N-layer-clone multi-cell strategy (OFFL-05), path-safety + scheme allowlist (T-15-06-01/02)"
  - "WandererMap offline branch: composes the rewritten style from mapStyleJsonProvider base + glyphSpriteCacheProvider root + trail.pmTiles before initStyle/setStyle"
  - "15-01 throwaway file:// glyph spike deleted"
affects: [16, 17, 18, OFFL-02, OFFL-03, OFFL-04, OFFL-05, OFFL-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure, I/O-free style transform (mirrors map_coordinate_adapter shape): deep-copy input via jsonEncode/jsonDecode so the shared keepAlive base style is never mutated"
    - "Offline multi-cell basemap (OFFL-05): N native pmtiles://file:// sources + per-extra-cell __cellN layer clones (source-less layers not cloned), because pmtiles 1.2.0 Dart is read-only and server merge is out of Flutter-phase scope"
    - "Path-safety at the trust boundary: reject non-absolute paths, .. traversal segments, and foreign URL schemes before any path enters the style — only file:// / pmtiles://file:// emitted"

key-files:
  created:
    - app/lib/util/offline_style_rewriter.dart
    - app/test/util/offline_style_rewriter_test.dart
  modified:
    - app/lib/components/base/wanderer_map.dart
    - app/lib/main.dart
    - app/lib/components/map/trail_layer.dart
  deleted:
    - app/lib/routes/spike_glyph_file_screen.dart
    - app/lib/util/spike_glyph_seed.dart

key-decisions:
  - "OFFL-05 multi-cell: N pmtiles://file:// sources + N duplicated layer sets (NOT merge-at-download). pmtiles 1.2.0 Dart is read-only (PmTilesArchive exposes only from/fromFile/fromReadAt — no write/merge) and db/services/tiles/generator.go runs `pmtiles extract` per 0.5° grid cell (grid.go GridSize=0.5, ~1-4 cells per realistic trail); a server merge endpoint is out of this Flutter phase's scope, so client/download-time merge is infeasible."
  - "OFFL-06 DEFERRED (not completed): pm_tile_provider.dart NOT deleted because navigation_screen.dart (Phase-17 flutter_map holdout) still consumes MultiPmTilesVectorTileProvider. Deleting it would break a screen the plan itself requires to keep building."
  - "Sprite variant (light/dark) selected from effectiveBrightness(themeMode) at rewrite time; arrow icon is sprite-independent (15-05 self-registers it), so offline arrow rendering does NOT depend on the unverified file:// sprite path."
  - "Offline pmtiles source maxzoom clamped to 14 (matching generator.go's pmtiles extract --maxzoom=14), overriding the online style's inherited maxzoom:15 — found via on-device testing (blank basemap at very high zoom, fine when zoomed out) and fixed in _pointSourceAtCell, test-guarded."

patterns-established:
  - "offline_style_rewriter.dart is the single sanctioned online->offline style transform; WandererMap is its only caller"

requirements-completed: [OFFL-02, OFFL-03, OFFL-04, OFFL-05]  # OFFL-06 deferred to Phase 17/18 — see key-decisions

# Metrics
duration: ~10min (resume; Task 1 restarted after a prior session cut-off) + orchestrator build-blocker fixes + on-device iteration
completed: 2026-07-09
---

# Phase 15 Plan 06: Offline Style Rewriter & WandererMap Offline Branch Summary

**COMPLETE — PHYSICAL-DEVICE OFFLINE PARITY GATE PASSED.** Tasks 1-2 shipped, three real bugs found and fixed during on-device verification (see below), Task 3 (OFFL-04) confirmed PASS by Christian.

Added `rewriteStyleForOffline` — a pure, test-guarded transform that rewrites a downloaded trail's MapLibre style so `glyphs`/`sprite` resolve from the 15-03 `file://` cache and the `protomaps` tiles resolve from the trail's local `.pmtiles` via native `pmtiles://file://` — then wired the offline branch into `WandererMap` and deleted the 15-01 spike.

## Status

- **Task 1 (offline style rewriter, OFFL-02/03/05):** DONE — RED `b1539bf7`, GREEN `2c44b5ab`. All 11 tests pass; `flutter analyze` clean.
- **Task 2 (WandererMap offline wiring + spike removal + OFFL-06):** PARTIAL — wiring `a4e593c4` + spike removal `5d7ae3df` done; **OFFL-06 (delete `pm_tile_provider.dart`) DEFERRED** (see Deviations).
- **Build blocker fix (orchestrator, commit `192b3a89`):** restored the flutter_map `TrailLayer` widget (deleted by 15-05) into `trail_layer.dart` for `navigation_screen.dart`'s continued use, explicitly scoped as legacy/delete-at-Phase-17. Whole-app `flutter analyze`: 0 errors.
- **Runtime crash fix (orchestrator, commit `b7d30947`):** `MapCompass` (flutter_map-only, calls `MapCamera.of(context)`) was still in `trail_detail_map_screen.dart`'s `WandererMap.controls` list — crashed on first real render since `WandererMap` became `MapLibreMap` in 15-04. Removed; a MapLibre-native compass is CORE-05 (Phase 17).
- **Offline maxzoom fix (orchestrator, commit `85d73fd3`):** found via on-device testing — basemap rendered blank at very high zoom offline, fine when zoomed out. Root cause: the offline pmtiles source kept the online style's `maxzoom: 15`, but local archives are extracted server-side at `--maxzoom=14` (`generator.go`), so MapLibre requested nonexistent z15+ tiles instead of overzooming z14. Fixed by clamping `maxzoom` to 14 in `_pointSourceAtCell`, test-guarded.
- **Task 3 (physical-device airplane-mode offline gate, OFFL-04):** **PASS**, confirmed by Christian after the three fixes above.

## Performance

- **Duration:** ~10 min (resume run; a prior session was cut off mid-Task-1 before any commit)
- **Completed:** 2026-07-09
- **Tasks:** 2 of 3 (Task 3 is a human-only gate, not run)
- **Files:** 2 created, 2 modified, 2 deleted

## Accomplishments

- **`rewriteStyleForOffline` (OFFL-02/03/05):** pure transform, no I/O. Deep-copies the input (via `jsonEncode`/`jsonDecode`) so the shared `keepAlive` base style is never mutated. Sets `glyphs` -> `file://<cacheRoot>/glyphs/{fontstack}/{range}.pbf` (literal tokens preserved for native substitution), `sprite` -> `file://<cacheRoot>/sprite/<light|dark>` (reuses `spriteCacheBasePath` from `map_cache_path.dart`), and repoints every remote (tiled) source to `pmtiles://file://<cell>`.
- **Multi-cell (OFFL-05):** first cell keeps the original source key; each extra cell `i` gets a `<source>-cell-<i>` source and a `<layerId>__cell<i>` clone of every layer that referenced that source. Source-less layers (e.g. `background`) are never cloned. No cell is dropped.
- **Path safety (T-15-06-01/02):** rejects non-absolute paths, any `..` traversal segment (in `cacheRoot` or a `cellPath`), foreign URL schemes, and an empty `cellPaths` list — with `ArgumentError`, before any path enters the style. The transform emits only `file://` and `pmtiles://file://` URL fields; no `https://` is ever produced for a downloaded trail. Test-guarded.
- **WandererMap offline branch:** when `widget.offline`, `_composeStyle` `jsonDecode`s the base style, calls `rewriteStyleForOffline(cacheRoot: glyphSpriteCacheProvider.root, cellPaths: trail.pmTiles, dark: <effectiveBrightness>)`, re-encodes, and feeds it to `initStyle`/`setStyle`. Online trails use the base JSON unchanged. The CORE-02 live swap recomposes the rewrite on theme toggle / cache warm. Existing loading (`ColoredBox`) + error passthrough preserved. Track/marker layers (15-05) still re-attach in `onStyleLoaded` for the offline style.
- **Spike removed:** `spike_glyph_file_screen.dart`, `spike_glyph_seed.dart`, and the debug `kDebugMode` FAB + imports in `main.dart` are gone.

## OFFL-05 Multi-Cell Decision (recorded)

**Investigation findings:**
- `pmtiles` 1.2.0 (Dart) is **read-only** — `PmTilesArchive` exposes only `from` / `fromFile` / `fromReadAt` and per-tile reads; there is no write/merge API. (Consistent with the vendor `pm_tile_provider.dart`, which only ever calls `PmTilesArchive.from(source)` + `.tile(...)`.)
- The server `db/services/tiles/generator.go` produces one `.pmtiles` per 0.5° grid cell via `pmtiles extract` (`grid.go` `GridSize = 0.5` ≈ 55 km), so `trail_download_service.dart` writes one file per cell into `trail.pmTiles`. A realistic trail spans ~1-4 cells.

**Decision:** N native `pmtiles://file://` sources + N duplicated style-layer sets — **not** merge-at-download. RESEARCH's initial recommendation (merge into one archive) is infeasible with the installed read-only package, and a server merge endpoint is out of this Flutter phase's scope. The duplication is bounded by `cellPaths.length` (T-15-06-03 DoS: accepted — small realistic cell counts; `is_large` full-polyline trails deferred, FUT-01).

## A2 Sprite `file://` Reality (carried forward from 15-01, honest exposure)

- The 15-01 spike found `file://` **sprite** resolution FAILED on a physical Android device (A2 FAIL) despite valid cached files, while `file://` **glyph** resolution PASSED (A1). This plan's rewrite implements the sprite `file://` rewrite because the test suite requires it.
- **Practical exposure is low and did not block the gate:** the directional `arrow` icon does **not** depend on sprite `file://` resolution — 15-05 self-registers it via `addImageFromIconData` (sprite-independent), and Task 3's PASS confirms the track/arrows/waypoints/pins all render offline. The only thing still exposed to the A2 gap is any *other* sprite-atlas icon (e.g. route-network shields, if the theme renders any via the sprite) — those do not render at all today via the current renderer regardless of online/offline, so this is a pre-existing not-yet-working feature, not a regression introduced by this plan.
- **No native-sprite re-investigation was performed.** Tracked as a known, separately-scoped gap — not a blocker for OFFL-02/03/04/05.

## Task Commits

1. **Task 1 RED — failing test for offline style rewriter** — `b1539bf7` (test)
2. **Task 1 GREEN — implement `rewriteStyleForOffline`** — `2c44b5ab` (feat)
3. **Task 2 — wire WandererMap offline branch** — `a4e593c4` (feat)
4. **Task 2 — remove 15-01 throwaway spike** — `5d7ae3df` (chore)
5. **Fix — restore flutter_map `TrailLayer` for `navigation_screen`** — `192b3a89` (fix, orchestrator, build blocker)
6. **Fix — remove flutter_map-only `MapCompass` from `WandererMap.controls`** — `b7d30947` (fix, orchestrator, runtime crash)
7. **Fix — clamp offline pmtiles source `maxzoom` to 14** — `85d73fd3` (fix, orchestrator, on-device finding)

**Plan metadata:** `commit_docs: false` in config — the SDK skips committing `.planning/` docs (SUMMARY/STATE/ROADMAP updated on disk directly, not committed).

## Files Created/Modified/Deleted

- `app/lib/util/offline_style_rewriter.dart` (created) — `rewriteStyleForOffline` pure transform.
- `app/test/util/offline_style_rewriter_test.dart` (created, was pre-written) — 12 tests: single/multi-cell, maxzoom clamp, deep-copy non-mutation, path-safety, scheme allowlist.
- `app/lib/components/base/wanderer_map.dart` (modified) — offline branch (`_composeStyle`/`_swapStyle`), offline-aware live theme swap.
- `app/lib/main.dart` (modified) — removed the debug spike FAB + `foundation`/spike imports.
- `app/lib/components/map/trail_layer.dart` (modified) — restored legacy flutter_map `TrailLayer` for `navigation_screen`.
- `app/lib/routes/trail_detail_map_screen.dart` (modified) — removed flutter_map-only `MapCompass` from `WandererMap.controls`.
- `app/lib/routes/spike_glyph_file_screen.dart`, `app/lib/util/spike_glyph_seed.dart` (deleted) — 15-01 throwaway spike.

## Decisions Made

See frontmatter `key-decisions`. Load-bearing: OFFL-05 = N-source/N-layer (read-only pmtiles); OFFL-06 deferred; offline arrow is sprite-independent.

## Deviations from Plan

### 1. [Rule 4 - Architectural] OFFL-06 deferred — `pm_tile_provider.dart` NOT deleted

- **Found during:** Task 2 (whole-app grep for remaining references).
- **Issue:** The plan assumed 15-04 removed all consumers of the vendor provider, so Task 2 would delete `lib/vendor/vector_map_tiles/pm_tile_provider.dart`. In fact `navigation_screen.dart` still consumes `MultiPmTilesVectorTileProvider.fromSources(...)` for its offline flutter_map vector tiles (lines 31, 71, 111, 160). STATE.md confirms `navigation_screen` is the Phase-17 flutter_map holdout (CORE-05/06/07).
- **Resolution:** Kept the provider. Deleting it would break `navigation_screen` and violate the plan's own "the 4 flutter_map screens still build" criterion. OFFL-06 is deferred to Phase 17 (or Phase 18 CLEAN), when `navigation_screen` migrates off flutter_map. Logged in `deferred-items.md`.
- **Files modified:** none (deletion intentionally not performed).

### 2. Task 2 automated verify partially fails by design

- The plan's Task 2 `<automated>` check asserts `test ! -f .../pm_tile_provider.dart` and no `pm_tile_provider` refs in `lib`. Both fail because of Deviation 1 (the file and its `navigation_screen` consumer legitimately remain). All other Task 2 gates pass: spike files removed, `rewriteStyleForOffline` referenced in `wanderer_map.dart`, and no NEW analyze issues in any file this plan touched.

## Issues Encountered / Blockers

### RESOLVED — the app did not build (fixed by orchestrator, commit `192b3a89`)

- `flutter analyze` reported **1 error**: `lib/routes/navigation_screen.dart:250` — `The method 'TrailLayer' isn't defined` (plus an unused `trail_layer.dart` import at line 15).
- **Origin:** PRE-EXISTING, not caused by 15-06. 15-05 (`785bc925`, `9f51989e`) deleted the old flutter_map `TrailLayer` widget from `trail_layer.dart` (replacing it with maplibre-native `addTrailTrackLayers` + `TrailMarkerLayer`), but `navigation_screen.dart` (last touched by `8f9705cb`, before 15-05) still calls `TrailLayer(...)`.
- **Fix applied:** restored the old flutter_map `TrailLayer` widget verbatim into `trail_layer.dart` (dropping only the already-dead `showArrows` animation branch, which was hardcoded `false` and never rendered anything), reusing the file's existing `ml.` alias + `map_coordinate_adapter.dart` helpers. Doc-commented as legacy — delete when `navigation_screen` migrates to MapLibre in Phase 17. Whole-app `flutter analyze` now reports **0 errors**.
- Logged in `deferred-items.md` as RESOLVED.

### Analyze baseline (after fix)

- Whole-app `flutter analyze`: **38 issues, 0 errors** = 3 warnings + 34 info + 1 pre-existing unrelated warning, all in files untouched by Phase 15 (`icon_util.dart` FontAwesome deprecations, `feed_item_test.dart` unused import, `trail_dropdown.dart` dead code from 15-03). Every file 15-06 touched is individually clean (`No issues found`): `offline_style_rewriter.dart`, `wanderer_map.dart`, `main.dart`, `trail_layer.dart`.

## Known Stubs

None. The offline sprite rewrite is real (not a stub); its A2 gap is a known, documented limitation, not a stub.

## User Setup Required

None.

## Next Phase Readiness

- **Phase 15's hard offline gate (OFFL-04, success criterion #4) is CLOSED.** A downloaded trail renders basemap (every cell), place-name labels, track, arrows, waypoints, and pins with the device offline — confirmed by Christian on a physical Android device.
- **OFFL-06 remains open** — delete `pm_tile_provider.dart` once `navigation_screen` migrates (Phase 17 / Phase 18 CLEAN). This is the only Phase 15 requirement not closed, and it was never in scope to close early (the plan's own "4 flutter_map screens still build" criterion requires keeping it).
- **Phase 16 (List & Map Screens on MapLibre) can now build on a working `WandererMap`.**

## Self-Check: PASSED

- FOUND: app/lib/util/offline_style_rewriter.dart
- FOUND: app/test/util/offline_style_rewriter_test.dart
- FOUND: app/lib/components/base/wanderer_map.dart (contains rewriteStyleForOffline)
- CONFIRMED DELETED: app/lib/routes/spike_glyph_file_screen.dart, app/lib/util/spike_glyph_seed.dart
- FOUND: commits b1539bf7 (RED), 2c44b5ab (GREEN), a4e593c4 (wiring), 5d7ae3df (spike removal), 192b3a89, b7d30947, 85d73fd3 (three on-device fixes)
- TESTS: 12/12 pass (`flutter test test/util/offline_style_rewriter_test.dart`)
- PHYSICAL DEVICE: OFFL-04 PASS, confirmed by Christian

---
*Phase: 15-maplibre-core-trail-rendering-offline-parity*
*Status: COMPLETE — offline parity gate PASSED; OFFL-06 deferred to Phase 17/18 (by design)*
