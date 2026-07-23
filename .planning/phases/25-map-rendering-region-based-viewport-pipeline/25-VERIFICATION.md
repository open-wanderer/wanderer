---
phase: 25-map-rendering-region-based-viewport-pipeline
verified: 2026-07-23T13:11:23Z
status: human_needed
score: 11/11 must-haves verified (code-level); 2 items require on-device human confirmation
overrides_applied: 0
human_verification:
  - test: "Trail detail map offline render + hillshade z-order (physical device, airplane mode)"
    expected: "With a region (ideally with DEM) downloaded, opening that trail's detail map renders the basemap offline from the region's .pmtiles, with hillshade rendering UNDERNEATH the vector basemap, not on top."
    why_human: "Visual rendering/z-order correctness on a real GPU compositor cannot be confirmed by static analysis or grep — requires eyes on a physical device."
  - test: "Trail detail map uncovered viewport (physical device, airplane mode)"
    expected: "Opening a trail whose bounds fall outside every downloaded region shows a blank basemap with NO 'no offline data' banner/toast."
    why_human: "Absence-of-UI-element and visual blank-state confirmation requires on-device observation."
  - test: "Trail detail map mid-session incremental region add (physical device, airplane mode)"
    expected: "With the trail detail screen already open, finishing a region download from Settings causes the basemap to appear incrementally without remounting or a full-style flash; hillshade still underneath."
    why_human: "Flicker/flash absence and incremental-vs-full-reload visual behavior can only be judged by watching the live render."
  - test: "Navigation screen region-boundary pan swap (physical device, airplane mode, two adjacent downloaded regions)"
    expected: "Starting navigation renders the region basemap + hillshade underneath offline; panning across the region boundary swaps the newly-entered region's sources in and the departed region's out once the pan settles, with no full-style flash; a removed region visually disappears immediately (no stale tiles lingering until the next tap, confirming the repaint-nudge actually repaints on real hardware)."
    why_human: "Real-time visual swap behavior, flicker absence, and confirmation that the 1ms camera-nudge repaint workaround actually forces a redraw on device GPU/driver combinations cannot be verified from source."
  - test: "Navigation screen within-region pan (no re-flicker) and uncovered-viewport blank state (physical device, airplane mode)"
    expected: "Panning within a single region does not re-flicker on every camera-idle (empty diff -> no-op); panning into an area with no downloaded region removes region sources -> blank basemap while navigation keeps tracking GPS/maneuvers, with no banner/toast; the swap happens on gesture settle, not a fixed delay."
    why_human: "Requires observing live pan gestures and confirming absence of visual glitching/flicker on real hardware timing, which no static check can substitute for."
---

# Phase 25: Map Rendering — Region-Based Viewport Pipeline Verification Report

**Phase Goal:** Trail detail maps and the navigation screen render offline tiles from the region registry instead of trail-bound caches, with style composition limited to what the current viewport actually needs.
**Verified:** 2026-07-23T13:11:23Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | (Roadmap SC1/RENDER-03) A spike against pinned maplibre 0.3.5 confirms incremental source add/remove support and measures 10-20 duplicated source/layer sets on a mid-tier Android device, settling the composition strategy | ✓ VERIFIED | `app/test/services/region_render_spike_harness.dart` exists (452 lines), analyzer-clean, reuses production `rewriteStyleForOffline` (5 occurrences), exercises both `addSource`/`setStyle` (13 occurrences), N slider spans `min: 10, max: 20` (lines 418-419), not imported anywhere under `app/lib` (confirmed via grep). Commit `f202e2de`. 25-01-SUMMARY.md records a human ran the harness on a physical Android device at N=10/N=20 and selected incremental (option-b) — a `checkpoint:decision gate="blocking"` task, which by design requires and records a human verdict; this is the correct artifact for this truth, not a gap. |
| 2 | (Roadmap SC2/RENDER-01) `TrailMap` reads offline vector/DEM tiles via `TileRepositoryManager.localTilePathsForBounds` instead of `Trail.pmTiles`/`demPmTiles` | ✓ VERIFIED | `grep -c "localTilePathsForBounds(widget.trail.bounds)" trail_map.dart` = 2 (bake + reconcile); `grep -c "widget.trail.pmTiles\|widget.trail.demPmTiles" trail_map.dart` = 0. Code read directly at lines 153-155, 321-323. |
| 3 | (Roadmap SC2/RENDER-01) `navigation_screen` reads offline vector/DEM tiles via `TileRepositoryManager.localTilePathsForBounds(live viewport)` instead of `Trail.pmTiles`/`demPmTiles` | ✓ VERIFIED | `grep -c "localTilePathsForBounds" navigation_screen.dart` = 3; `grep -c "\.pmTiles\|\.demPmTiles" navigation_screen.dart` = 0; `_composeStyle` takes `viewportBounds` param (5 occurrences), `_swapStyle`/build-time seed both resolve `controller.getVisibleRegion()` (4 occurrences). Code read directly at lines 888-916, 922-935, 1373-1377. |
| 4 | (RENDER-01) `localTilePathsForBounds` returns vector/DEM paths as two SEPARATE typed lists — a DEM path can never be conflated into vector `cellPaths` | ✓ VERIFIED | `tile_repository_manager.dart` returns `({List<String> vectorPaths, List<String> demPaths})` via `@visibleForTesting splitRegionTilePaths`; real ObjectBox query `_store.box<RegionEntity>().getAll()` (line 306, not a static stub — Level 4 data-flow confirmed FLOWING). `flutter test test/services/tile_repository_manager_test.dart` — all 9 tests pass, including the DEM-only anti-conflation case. |
| 5 | (Roadmap SC3/RENDER-02) Only regions intersecting the current viewport contribute style sources — panning to a different region swaps sources in/out rather than accumulating every downloaded region | ✓ VERIFIED | `navigation_screen.dart._reconcileRegionComposition` computes `desiredSourceIds`/`desiredLayerIds` fresh from `controller.getVisibleRegion()` each call, diffs against `_addedSourceIds`/`_addedLayerIds`, removes stale ids (`removeLayer`/`removeSource`, 7 occurrences) BEFORE adding new ones — genuine swap, not accumulation. `TrailMap` (fixed trail-bounds viewport, D-05) correctly has 0 `removeSource`/`removeLayer` occurrences since its single viewport never changes — add-only is the correct behavior for a screen whose "viewport" is fixed for the widget's lifetime. |
| 6 | (25-01 finding 1) After every incremental removal in `navigation_screen`, the map is force-repainted via a 1ms no-op camera nudge (maplibre 0.3.5 does not auto-repaint after removeSource/removeLayer) | ✓ VERIFIED (code-level) | `nativeDuration: const Duration(milliseconds: 1)` present (1 occurrence) inside `if (removedAny) { ... controller.animateCamera(...) }` (lines 1208-1217), gated correctly on the `removedAny` flag set only inside the two removal loops. Whether this workaround actually forces a GPU repaint on real hardware is a human on-device concern (see Human Verification). |
| 7 | (25-01 finding 2) Incrementally-added hillshade/DEM layers are inserted below the first vector layer in both `TrailMap` and `navigation_screen` | ✓ VERIFIED | Both files: `belowLayerId` present (1 occurrence each), computed from `firstVectorLayerId` = first layer whose type is fill/line/symbol, passed only on the `layer['type'] == 'hillshade'` branch of `addLayer`. |
| 8 | (RESEARCH.md Pattern 1) A region finishing download mid-session is applied incrementally via a `regionListNotifierProvider` listen, not a `setStyle` reload, in both screens | ✓ VERIFIED | `trail_map.dart`: `ref.listen(regionListNotifierProvider, (_, _) => _addRegionComposition())` (1 occurrence, inside `if (widget.offline)`). `navigation_screen.dart`: `ref.listen(regionListNotifierProvider, (_, _) => _reconcileRegionComposition())` (1 occurrence, inside `if (widget.isOffline)`). Neither routes through `_swapStyle`/`setStyle`. |
| 9 | (D-01/D-02) An uncovered viewport (no downloaded region overlaps) renders a blank basemap with no banner/indicator in both screens; navigation keeps tracking GPS/maneuvers regardless | ✓ VERIFIED (code-level) | Both `_composeStyle` methods short-circuit to `return null` on `tiles.vectorPaths.isEmpty` before `rewriteStyleForOffline` would throw (2 occurrences each). `grep -n "SnackBar\|MaterialBanner\|no offline data\|No offline"` across both files returns zero matches — no new user-facing indicator was added. Breadcrumb/location-marker layers are separate from the region reconcile and untouched by it, so live tracking continues. Visual "renders blank, no flash" confirmation is a human on-device concern. |
| 10 | (D-04, Pitfall 4) The region-boundary swap in `navigation_screen` is reconciled once per settled gesture via `ml.MapEventCameraIdle`, not a manual `Timer`/debounce | ✓ VERIFIED | `grep -c "MapEventCameraIdle"` = 1, routed to `if (widget.isOffline) _reconcileRegionComposition()` (line 1455). `grep -Ec "Timer\(\|Timer\.periodic\|Debouncer"` = 1, and that one match is the pre-existing `_persistTimer` (navigation-state persistence, unrelated) — no new Timer was introduced for the region swap. |
| 11 | `offline_style_rewriter.dart` is reused unchanged by both screens (roadmap SC4 constraint) | ✓ VERIFIED | Neither plan's `files_modified` nor the actual diffs touch `offline_style_rewriter.dart`; both `_composeStyle`/reconcile methods call the existing `rewriteStyleForOffline` with `cellPaths`/`demCellPaths` unchanged in signature. |

**Score:** 11/11 truths pass code-level verification. Two of these (6 and 9, plus the general "renders correctly offline" roadmap SC4 claim) additionally require on-device human confirmation of actual visual/GPU behavior that cannot be settled by static analysis — see Human Verification Required below. This is why overall status is `human_needed`, not `passed`.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/test/services/region_render_spike_harness.dart` | RENDER-03 spike harness, >=120 lines, contains `rewriteStyleForOffline` | ✓ VERIFIED | 452 lines; 5 `rewriteStyleForOffline` occurrences; 13 `addSource\|setStyle` occurrences; `flutter analyze` clean; not imported under `app/lib` |
| `app/lib/services/tile_repository_manager.dart` | `localTilePathsForBounds` returning `({vectorPaths, demPaths})` + `@visibleForTesting splitRegionTilePaths` | ✓ VERIFIED | Read directly (lines 53-77, 294-307); real ObjectBox-backed query, not a stub |
| `app/test/services/tile_repository_manager_test.dart` | `splitRegionTilePaths` unit tests incl. DEM-only anti-conflation case | ✓ VERIFIED | `flutter test` — 9/9 tests pass |
| `app/lib/components/base/trail_map.dart` | `_composeStyle` region-sourced; `_addRegionComposition` incremental add-only reconcile; tracking sets | ✓ VERIFIED | All plan grep-gates pass with exact/exceeding counts (see Key Link Verification); `flutter analyze` clean |
| `app/lib/routes/navigation_screen.dart` | Viewport-scoped composition; `_reconcileRegionComposition` incremental add/remove swap; camera-idle recompute; repaint nudge; tracking sets | ✓ VERIFIED | All plan grep-gates pass with exact/exceeding counts; `flutter analyze` clean |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `trail_map.dart` | `TileRepositoryManager.localTilePathsForBounds` | `ref.read(tileRepositoryManagerProvider).localTilePathsForBounds(widget.trail.bounds)` | WIRED | 2 occurrences (bake + reconcile), matches plan's required count exactly |
| `trail_map.dart` | `regionListNotifierProvider` | `ref.listen(...) => _addRegionComposition()` | WIRED | 1 occurrence, inside `if (widget.offline)` guard |
| `trail_map.dart` | `ml.StyleController.addLayer belowLayerId` | hillshade z-order insertion | WIRED | 1 occurrence, gated on `layer['type'] == 'hillshade'` |
| `navigation_screen.dart` | `TileRepositoryManager.localTilePathsForBounds` | `ref.read(tileRepositoryManagerProvider).localTilePathsForBounds(controller.getVisibleRegion())` | WIRED | 3 occurrences (build-time via `_composeStyle`, `_swapStyle`, `_reconcileRegionComposition`) |
| `navigation_screen.dart` | `ml.MapEventCameraIdle` | `onEvent` branch → `_reconcileRegionComposition()` | WIRED | 1 occurrence, gated on `widget.isOffline`, no manual Timer introduced |
| `navigation_screen.dart` | `ml.StyleController.removeSource/removeLayer` + repaint nudge | remove stale ids (layers before sources) then 1ms `animateCamera` nudge | WIRED | 7 `removeSource\|removeLayer` occurrences; repaint nudge correctly gated on `removedAny` flag |
| `navigation_screen.dart` | `regionListNotifierProvider` | `ref.listen(...) => _reconcileRegionComposition()` | WIRED | 1 occurrence, inside `if (widget.isOffline)` guard |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `localTilePathsForBounds` (both call sites) | `tiles.vectorPaths` / `tiles.demPaths` | `TileRepositoryManager._store.box<RegionEntity>().getAll()` → `splitRegionTilePaths` | Yes — real ObjectBox query against the persisted region/package entities, not a static/empty return | ✓ FLOWING |
| `trail_map.dart` / `navigation_screen.dart` composed style | `composed['sources']` / `composed['layers']` | `rewriteStyleForOffline(decoded, cellPaths: tiles.vectorPaths, demCellPaths: tiles.demPaths, ...)` — the unchanged production composer | Yes — real style JSON built from real archive paths, not hardcoded | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `flutter analyze` on all 3 modified production files | `cd app && flutter analyze lib/routes/navigation_screen.dart lib/components/base/trail_map.dart lib/services/tile_repository_manager.dart` | "No issues found!" | ✓ PASS |
| `flutter analyze` on the RENDER-03 spike harness | `cd app && flutter analyze test/services/region_render_spike_harness.dart` | "No issues found!" | ✓ PASS |
| `splitRegionTilePaths`/`bboxOverlaps` unit test suite | `cd app && flutter test test/services/tile_repository_manager_test.dart` | 9/9 tests passed, incl. DEM-only anti-conflation case | ✓ PASS |
| No new user-facing uncovered-viewport indicator | `grep -n "SnackBar\|MaterialBanner\|no offline data\|No offline" lib/routes/navigation_screen.dart lib/components/base/trail_map.dart` | zero matches | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` files exist in this repository and neither the PLAN files nor SUMMARY files reference a probe-based verification convention (this is a Flutter/Dart mobile project, not a script-driven migration/tooling phase). Step 7c: SKIPPED — no probes declared or discovered.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| RENDER-01 | 25-02, 25-03, 25-04 | `TrailMap`/`navigation_screen` read offline tiles from the region registry via `TileRepositoryManager`, replacing trail-bound cache reads | ✓ SATISFIED | Data-shape split (25-02, unit-tested) + both call sites migrated off `pmTiles`/`demPmTiles` onto `localTilePathsForBounds` (25-03, 25-04) — verified directly in code |
| RENDER-02 | 25-04 | Style composition is viewport-scoped — only regions intersecting the current viewport contribute style sources, not every downloaded region unconditionally | ✓ SATISFIED | `navigation_screen._reconcileRegionComposition` diffs desired-vs-tracked ids and removes stale sources before adding new ones on every settled camera gesture — verified directly in code |
| RENDER-03 | 25-01 | Before finalizing the rendering approach, verify maplibre 0.3.5's incremental source add/remove behavior (vs. full style reload) and layer-count scaling with a spike against the pinned package version | ✓ SATISFIED | Spike harness built and analyzer-clean, exercises both strategies, human ran it on-device at N=10/N=20 per `checkpoint:decision gate="blocking"` and selected incremental — recorded in 25-01-SUMMARY.md |

No orphaned requirements: `REQUIREMENTS.md`'s Phase 25 mapping (RENDER-01/02/03, all marked "Complete") matches exactly the set claimed across the four plans' `requirements:` frontmatter.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers found in any of the 6 files this phase touched (`trail_map.dart`, `navigation_screen.dart`, `tile_repository_manager.dart`, `region_render_spike_harness.dart`, `tile_repository_manager_test.dart`, `tile_repository_manager_harness.dart`) | — | — |
| `app/lib/components/base/trail_map.dart:317-393`, `app/lib/routes/navigation_screen.dart:1076-1218, 922-935` | — | Fire-and-forget `async` reconcile methods (`_addRegionComposition`, `_reconcileRegionComposition`, `_swapStyle`) invoked without `await` from `ref.listen`/`onEvent` callbacks, with no reentrancy guard — two overlapping calls (e.g. a region download completing during a camera-idle event) can interleave writes to the same `_addedSourceIds`/`_addedLayerIds` tracking sets across `await` boundaries | ⚠️ WARNING | Pre-existing finding from `25-REVIEW.md` (WR-03). Does not prevent the phase's core goal (region-registry sourcing, viewport-scoping) from working under normal single-gesture usage, but is a real robustness gap under rapid concurrent triggers. Not a blocker for this phase's success criteria; recommend tracking as a follow-up (e.g. a `_reconciling` in-flight guard, per the review's suggested fix). |
| `app/lib/routes/navigation_screen.dart:1369-1377` (CR-03 in `25-REVIEW.md`) | — | `viewportBounds` fallback chain (`_controller?.getVisibleRegion() ?? trailAsync.value?.bounds`) never resolves for an offline recording session (`trailProvider('')` → `AsyncError`, `.value` stays permanently `null`), which would deadlock the map on a permanent loading spinner IF that code path were ever reached with `isOffline: true` | ℹ️ INFO | Independently confirmed via direct code read: `app/lib/provider/router_provider.dart`'s `/record` route (lines 316-324) constructs `NavigationScreen` with no `isOffline:` argument, and `NavigationScreen.isOffline` defaults to `false` (line 96) — so this path is UNREACHABLE in the current codebase. This is a latent bug guarded by an unrelated plumbing gap (`WR-05` in the review, `/record`'s resume path never threading `row.isOffline` back in), not a regression introduced by this phase's RENDER-01/02/03 work, and does not block Phase 25's own goal. Recommend a follow-up ticket so it doesn't resurface silently once WR-05 is fixed. |
| `app/lib/services/tile_repository_manager.dart:153-196, 241-276` (CR-01/CR-02 in `25-REVIEW.md`) | — | Download-status-stuck-on-non-DioException-failure and missing concurrent-download de-dup guard | ℹ️ INFO | Both findings are in Phase 23's pre-existing download-lifecycle code (`startVectorDownload`/`startDemDownload`), not code this phase's plans (`25-01..04`) added or changed — only `localTilePathsForBounds`/`splitRegionTilePaths` in this file are Phase 25's own additions. Out of scope for this phase's goal-backward verification; flagged here for completeness per the review, not counted as a Phase 25 gap. |

### Human Verification Required

Per this project's `human_verify_mode=end-of-phase` convention, 25-03 and 25-04 both deliberately deferred their `<verify><human-check>` on-device (airplane-mode) checks from execution time to end-of-phase. These were not run during execution and are consolidated here as required human verification before the phase can be marked fully `passed`.

### 1. Trail detail map offline render + hillshade z-order

**Test:** On a physical device in airplane mode, download a region (ideally with DEM) via Settings → Offline Maps/Regions, then open a trail whose bounds fall inside that region and view its detail map.
**Expected:** The basemap renders offline from the region's `.pmtiles`; hillshade (if DEM was downloaded) renders UNDERNEATH the vector basemap, not on top.
**Why human:** Actual GPU-compositor z-order and offline-tile-resolution correctness cannot be confirmed by static analysis — `belowLayerId` being present in the source proves intent, not rendered outcome.

### 2. Trail detail map uncovered viewport

**Test:** On the same device/session, open a trail whose bounds fall OUTSIDE every downloaded region.
**Expected:** A blank basemap renders with NO "no offline data" banner/toast.
**Why human:** Confirms a UI absence and blank-render visual state, not just the absence of banner-widget code.

### 3. Trail detail map mid-session incremental region add

**Test:** With the trail detail screen already open (bounds not yet covered), finish downloading its covering region from Settings, then return to the map.
**Expected:** The basemap appears incrementally without remounting or a full-style flash; hillshade still renders underneath.
**Why human:** Flicker/flash absence during a live style mutation is a visual-timing property no grep or unit test can substitute for.

### 4. Navigation screen region-boundary pan swap

**Test:** With two adjacent regions downloaded and a trail spanning both, start navigation and pan across the region boundary.
**Expected:** The newly-entered region's sources swap in and the departed region's sources swap out once the pan settles, with no full-style flash; a removed region visually disappears immediately (no stale tiles lingering until the next tap or pan) — confirming the 1ms repaint-nudge workaround actually forces a redraw on real hardware.
**Why human:** This directly tests the 25-01 finding-1 workaround's real-world effectiveness (maplibre 0.3.5's lack of a native invalidate/redraw call) on actual GPU/driver behavior, which cannot be verified from source code alone.

### 5. Navigation screen within-region pan and uncovered-viewport blank state

**Test:** Pan within a single downloaded region repeatedly, then pan into an area with no downloaded region coverage.
**Expected:** Panning within a single region does not re-flicker on every camera-idle (empty diff → no-op); panning into uncovered territory removes region sources and shows a blank basemap while navigation keeps tracking GPS/maneuvers, with no banner/toast; the swap happens on gesture settle, not a fixed delay.
**Why human:** Confirms absence of re-flicker under normal panning and correct fallback behavior under real GPS/gesture timing — a purely runtime, device-dependent property.

### Gaps Summary

No code-level gaps were found. All 11 derived observable truths (merged from ROADMAP.md's 4 success criteria and the four plans' `must_haves` frontmatter) verified against the actual codebase: `grep`-based acceptance-criteria gates from all four plans pass with exact or exceeding counts, `flutter analyze` is clean on every file this phase touched, and the `tile_repository_manager_test.dart` unit suite (9 tests, including the RENDER-01 anti-conflation guarantee) passes. Commits referenced in every SUMMARY.md (`f202e2de`, `4b3368bf`, `5d639a5c`, `92bd26af`, `a03d6eec`, `65407f3b`, `f4e5ecaa`) were independently confirmed to exist via `git log`.

The phase is withheld from `passed` status solely because its own plans (25-03, 25-04) deliberately deferred on-device human verification of visual rendering correctness to end-of-phase, per this project's `human_verify_mode=end-of-phase` convention — these are the 5 items listed above, not defects. Three findings from the independent `25-REVIEW.md` code review are carried forward as INFO/WARNING context (not gaps): two (CR-01, CR-02) are in Phase 23's pre-existing download code untouched by this phase's plans, and the third (CR-03, navigation-screen viewport-bounds deadlock for recording sessions) was independently re-confirmed via direct code read to be unreachable in the current codebase because `/record`'s route builder never passes `isOffline: true` — a latent bug masked by an unrelated plumbing gap (WR-05), not a regression from this phase's RENDER-01/02/03 work.

---

*Verified: 2026-07-23T13:11:23Z*
*Verifier: Claude (gsd-verifier)*
