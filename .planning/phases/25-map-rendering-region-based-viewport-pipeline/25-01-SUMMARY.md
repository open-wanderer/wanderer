---
phase: 25-map-rendering-region-based-viewport-pipeline
plan: 01
subsystem: map-rendering
tags: [maplibre, flutter, region-swap, offline-tiles, spike]

# Dependency graph
requires:
  - phase: 23-tile-repository-manager-download-engine
    provides: TileRepositoryManager, RegionEntity local vector/DEM archive paths
  - phase: 15
    provides: rewriteStyleForOffline (offline_style_rewriter.dart), the production style-composition helper the spike reuses
provides:
  - RENDER-03 settled — incremental addSource/removeSource/addLayer/removeLayer selected as Phase 25's region-swap composition strategy over full-style-reload
  - Two on-device findings that Wave 2 (25-03/25-04) must design around before implementation
affects: [25-03, 25-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Composition-strategy spikes belong under app/test/services/ as a standalone flutter run -t harness (not flutter test), mirroring tile_repository_manager_harness.dart — throwaway, never imported from app/lib"

key-files:
  created:
    - app/test/services/region_render_spike_harness.dart
  modified: []

key-decisions:
  - "[25-01] RENDER-03 settled: Phase 25 ships incremental addSource/removeSource/addLayer/removeLayer (option-b), not full-style-reload (option-a) — on-device testing on a real Android phone showed incremental avoided the full-reload flicker/blank-frame risk, prompting the user's verdict 'Incremental is definitely the way to go.'"
  - "[25-01] Follow-up required in Wave 2 (finding 1): removeSource/removeLayer do not visually repaint the map on their own in this app's maplibre 0.3.5 usage — after tapping incremental remove, removed layers stayed visible until the user tapped or panned the map. 25-03/25-04's incremental-swap implementation must trigger an explicit repaint/invalidate immediately after removal, not rely on removeSource/removeLayer alone."
  - "[25-01] Follow-up required in Wave 2 (finding 2): incrementally-added DEM/hillshade layers render on TOP of the vector layers instead of underneath, because addLayer without an explicit insertion position appends to the top of the style's layer stack (unlike the full-JSON rewriteStyleForOffline path, which preserves the style document's original layer order). 25-03/25-04 must insert the hillshade layer at the correct position (e.g. a belowLayerId-style insertion before the first vector layer), not a bare append."

requirements-completed: [RENDER-03]

# Metrics
duration: ~25min (Task 1 execution + on-device measurement + Task 2 decision recording)
completed: 2026-07-23
---

# Phase 25 Plan 01: RENDER-03 Composition-Strategy Spike Summary

**Built a throwaway on-device spike harness that materializes 10-20 duplicated region vector/DEM sources via the production `rewriteStyleForOffline` helper, and settled RENDER-03 in favor of incremental `addSource`/`removeSource` over full-style-reload after physical-device testing surfaced two implementation gaps (missing repaint-on-remove, wrong hillshade z-order) that Wave 2 must now design around.**

## Performance

- **Duration:** ~25 min
- **Tasks:** 2 (1 auto, 1 checkpoint:decision)
- **Files modified:** 1 created (`app/test/services/region_render_spike_harness.dart`)

## Accomplishments

- Built `region_render_spike_harness.dart`, a standalone `flutter run -t` harness cloning `tile_repository_manager_harness.dart`'s scaffolding, that materializes N (10-20) duplicated region vector/DEM style sources+layers via the real production `rewriteStyleForOffline` composition path and exposes timed full-reload (`setStyle`) and incremental (`addSource`/`removeSource`/`addLayer`/`removeLayer`) swap actions.
- Ran the harness on a physical mid-tier Android device at N=10 and N=20; observed the incremental path avoided full-reload's flicker/blank-frame risk.
- Settled RENDER-03: Phase 25 ships the **incremental** `addSource`/`removeSource`/`addLayer`/`removeLayer` composition strategy (option-b), not the RESEARCH.md-recommended full-style-reload default (option-a).
- Surfaced two concrete on-device findings that materially affect Wave 2's implementation (see Decisions Made below) — both logged here as explicit follow-ups since 25-03/25-04 were written assuming full-reload.

## Task Commits

1. **Task 1: Build the RENDER-03 duplicated-source spike harness** - `f202e2de` (feat)
2. **Task 2: checkpoint:decision — composition strategy** - decision-recording only, no code commit (per plan: "no code is written in this task")

**Plan metadata:** (this commit) - `docs: complete 25-01 plan`

## Files Created/Modified

- `app/test/services/region_render_spike_harness.dart` - Throwaway on-device spike harness; materializes N duplicated region vector/DEM sources via `rewriteStyleForOffline` and exposes timed full-reload vs incremental swap actions with a running ms readout.

## Decisions Made

- **RENDER-03 settled: incremental composition strategy selected (option-b).** On a physical mid-tier Android device, the user ran the harness at N=10 and N=20, tapping "Full reload" and "Incremental add"/"Incremental remove" repeatedly while panning/zooming. The user's verdict: "Incremental is definitely the way to go" — incremental swaps avoided the flicker/blank-frame characteristic of `setStyle`-based full reloads that would have accompanied the RESEARCH.md-recommended default. No precise millisecond figures were captured; the decision was made on qualitative visual smoothness grounds, which the checkpoint's `<action>` explicitly allows ("select option-a or option-b").
- **Follow-up required in Wave 2 — Finding 1 (repaint-on-remove gap):** Tapping "Incremental remove" in the harness did not cause the removed sources/layers to visually disappear immediately; the user had to tap or pan the map before the removal actually rendered. This indicates maplibre-native 0.3.5 does not auto-repaint after `removeSource`/`removeLayer` in this app's usage. **25-03/25-04's incremental-swap implementation must add an explicit repaint/invalidate trigger (or equivalent forced redraw) immediately after removal** — `removeSource`/`removeLayer` calls alone are not sufficient.
- **Follow-up required in Wave 2 — Finding 2 (hillshade z-order gap):** The DEM/hillshade layer rendered on TOP of the vector fill/line/symbol layers when added incrementally, instead of underneath them as it does via the full-JSON `rewriteStyleForOffline` composition path. This is very likely because `addLayer` without an explicit insertion position appends to the top of the style's layer stack, whereas the full-style-document path preserves the style JSON's original layer ordering. **25-03/25-04 must insert the hillshade layer at the correct position** (e.g. a `belowLayerId`-style insertion point, placed before/below the first vector layer) rather than a bare `addLayer` append.

## Deviations from Plan

None - plan executed exactly as written. Task 2 was a pure human decision gate (`checkpoint:decision`, `gate="blocking"`); per its own `<action>` text, "no code is written in this task." The decision landed on option-b (incremental) rather than the plan's stated recommended default (option-a, full-reload) — this is the checkpoint functioning as designed, not a deviation from the plan's process.

## Issues Encountered

None during Task 1/Task 2 execution itself. The two on-device findings recorded above are not "issues encountered" during this plan's own execution — they are empirical results of the measurement this plan exists to produce, and are explicitly scoped as follow-up work for Wave 2, not for this plan.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- RENDER-03 is settled. Wave 2 (25-02 first — the map-screen rewiring baseline — followed by 25-03/25-04) can proceed, but **25-03 and 25-04 were written assuming the full-style-reload strategy and now need revision** before execution to (a) implement incremental `addSource`/`removeSource`/`addLayer`/`removeLayer` as the shipping composition strategy instead of `setStyle`, (b) add an explicit repaint/invalidate step after every incremental removal, and (c) insert the hillshade layer at a correct below-vector-layers position instead of a bare top-of-stack append.
- Per the orchestrator's own scoping instruction for this continuation, revising 25-03/25-04 is explicitly **not** done in this plan — it is a separate follow-up step the orchestrator will handle after this plan (25-01) and 25-02 both complete.
- The spike harness (`app/test/services/region_render_spike_harness.dart`) remains in the tree as a throwaway artifact per its own header note; it is out of production routes (`grep -rl "region_render_spike_harness" app/lib` returns nothing) and can be deleted once RENDER-03's follow-on work in 25-03/25-04 is complete, or kept as a reusable perf-regression check for future maplibre upgrades.

---
*Phase: 25-map-rendering-region-based-viewport-pipeline*
*Completed: 2026-07-23*

## Self-Check: PASSED
