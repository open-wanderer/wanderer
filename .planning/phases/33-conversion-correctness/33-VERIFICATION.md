---
phase: 33-conversion-correctness
verified: 2026-07-31T13:05:00Z
status: gaps_found
score: 4/7 must-haves verified
overrides_applied: 0
gaps:
  - truth: "CONV-04: a steep, low-horizontal-movement stretch is measured instead of skipped, without fabricating gain/loss from GPS/altitude noise"
    status: failed
    reason: >
      The horizontal-movement gate was removed entirely instead of being replaced with a
      noise-aware filter. Elevation is now diffed against lastFilteredZ on every sample with
      only a flat >= thresholdZ_m (5 m) commit rule and no direction/hysteresis check.
      Independently reproduced: a 60-sample fully-stationary track (identical lat/lon,
      altitude alternating 1000/1007 m — inside normal consumer GPS/barometric noise)
      reports totalElevationGainSmoothed = 210 and totalElevationLossSmoothed = 203 over 0 m
      of travel. Pre-phase this fixture returned 0/0. These values flow unmodified into
      GPX.getTotals() -> features.elevationGain/elevationLoss -> updateTotals() ->
      trail.elevation_gain/elevation_loss, and are persisted permanently (CONV-F01 migration
      is explicitly deferred). No fixture in the shipped suite exercises an
      oscillating/stationary track — only a monotonic 12-point climb, which cannot expose
      this.
    artifacts:
      - path: "web/src/lib/models/gpx/gpx-metrics-computation.ts"
        issue: "Lines 107-123 (smoothed elevation block) commit any |diff| >= thresholdZ_m with no reversal/hysteresis check, so bidirectional noise ratchets gain and loss simultaneously."
    missing:
      - "A noise-tolerant elevation filter (e.g. hysteresis on direction of travel / hold-and-reverse ZigZag filter) that still measures genuine low-horizontal climbs but does not ratchet on oscillating noise."
      - "A stationary-noise regression fixture (60 samples, fixed lat/lon, +/-7 m alternation) asserting elevationGain === 0 / elevationLoss === 0, alongside the existing monotonic-climb fixture."
  - truth: "D-02 / 33-03 must-have: the route-crop feature still works — getCoordinateAtDistance() never produces a NaN coordinate"
    status: failed
    reason: >
      The guard added in updateCropMarkers() — `cumulativeRoute.length < 2 ||
      !Number.isFinite(rawRouteTotal)` — does not catch rawRouteTotal === 0. Independently
      reproduced by extracting the shipped getCoordinateAtDistance()/guard logic into a probe:
      for cumulativeRoute = [0, 0, 0] (a fully degenerate/coincident-point route) the guard
      evaluates to "passes" (0 is finite) and getCoordinateAtDistance(..., target=0) returns
      [NaN, NaN, 1] because prevDist === nextDist === 0 makes the interpolation ratio 0/0.
      The same NaN occurs for any route whose first two points are merely coincident (a
      routine GPS artefact) when the crop range is queried at 0%, which DoubleSlider's
      onMount "update" event fires unconditionally on every crop-panel open (range defaults
      to [0,100]). cropStartMarker.setLngLat([NaN, NaN]) reaches MapLibre's LngLat.convert,
      which throws on NaN — an uncaught exception on crop-panel open for these routes. This is
      the exact scenario 33-03's plan deferred to end-of-phase human verification
      ("including 0%/100%, with no NaN totals") and it fails.
    artifacts:
      - path: "web/src/routes/trail/edit/[id]/+page.svelte"
        issue: "Lines 1480-1485 guard checks the wrong two conditions (array length, non-finite total) and misses the actual failure mode (a usable-but-zero interpolation basis / zero-length adjacent span)."
    missing:
      - "Guard on a usable interpolation basis: reject when rawRouteTotal is not > 0, not merely non-finite."
      - "Make getCoordinateAtDistance() itself degenerate-safe: when nextDist - prevDist === 0, do not divide (return ratio 0 instead of NaN)."
      - "Regression tests for a leading-duplicate-point route and an all-identical-points route."
  - truth: "D-02 / 33-03 must-have: crop panel state does not resurrect a route the user already discarded"
    status: failed
    reason: >
      croppedGPX is declared once (`let croppedGPX: GPX | null = null;` at line 166) and
      assigned only inside updateCropMarkers() at line 1504; nothing resets it to null —
      confirmed by grepping every reference to croppedGPX in the file (declaration,
      assignment, the two confirmCrop() reads). The early return added by this phase at line
      1484 (see the CR-01 gap above) exits before reassigning croppedGPX, so a stale value
      from a previously-cropped route survives. confirmCrop() only checks `if (!croppedGPX)`
      before calling setRoute(croppedGPX, true) — it does not re-validate that croppedGPX
      corresponds to the currently-drawn route. Reachable sequence: crop route A (croppedGPX
      = crop-of-A) -> replace the route with new/short route B -> reopen the crop panel (the
      guard above returns early for a degenerate B, croppedGPX unchanged) -> click confirm ->
      route B is silently destroyed and replaced with the discarded crop of A.
    artifacts:
      - path: "web/src/routes/trail/edit/[id]/+page.svelte"
        issue: "croppedGPX has no reset path (resetRoute(), replaceRoute(), toggleCropMarkers(false) all leave it untouched) and confirmCrop() trusts the cached value unconditionally."
    missing:
      - "Reset croppedGPX = null on the new early-return path, and in resetRoute()/replaceRoute()/toggleCropMarkers(false), so the cache cannot outlive the route it was derived from."
deferred: []
human_verification:
  - test: "Open a trail with a route that has near-coincident start points (e.g. a route drawn/paused at the trailhead) or a very short/degenerate route in the trail-edit crop panel and observe whether the app throws or silently misbehaves."
    expected: "No uncaught exception; crop pins render at a sane position or the crop control is disabled/hidden for a degenerate route."
    why_human: "Requires a live MapLibre instance to observe the uncaught throw and its user-facing effect; grep/static analysis can prove the NaN is produced (done above) but not the exact on-screen failure mode."
  - test: "Crop route A, replace/discard the route, draw a new short route B, reopen the crop panel, and confirm the crop without dragging the slider."
    expected: "Route B (the current route) is cropped/kept — not silently replaced by a stale crop of route A."
    why_human: "Requires exercising the multi-step UI sequence end-to-end; the code-level trace is conclusive as a data-loss possibility but the actual save/discard UX outcome should be confirmed by hand."
---

# Phase 33: Conversion Correctness Verification Report

**Phase Goal:** Every GPX converted anywhere in Wanderer — a web upload or a server-side conversion — reports correct distance, elevation, and duration, fixing four real defects in the shared TS computation before the Dart port can be pinned against it.

**Verified:** 2026-07-31T13:05:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A 2-point GPX track segment reports its real length instead of zero; centroid/bbox sum and divide by the same point count | VERIFIED | `gpx.ts:126` loop starts at `i = 0`; `summedPointCount` drives the centroid divisor (`gpx.ts:104,130-132,147`); `allPoints.length` no longer appears (`grep -c "allPoints.length"` = 0). `gpx.test.ts` asserts a 2-point ~134.6 m segment reports ~134.59 m (was 0) and the extreme-first-point bbox/centroid case. Whole suite green (31/31). |
| 2 | A GPX with partial elevation tags no longer reports a phantom drop to sea level, **and** a steep, low-horizontal-movement stretch is measured instead of skipped, without fabricating gain/loss | ✗ FAILED | The missing-`<ele>` half is genuinely fixed: `parseElevation()` (`gpx-metrics-computation.ts:15-24`) returns `undefined` for absent/empty/non-numeric input, tested through the real XML parse path (`gpx-metrics-computation.test.ts`). But the low-horizontal-movement half overshot: the horizontal gate was removed entirely rather than replaced with a noise-tolerant filter. Independently reproduced via an executed probe (see Anti-Patterns / CR-02 below): a 60-sample **stationary** track with realistic ±7 m altitude noise reports **210 m gain / 203 m loss over 0 m of travel** (was 0/0 pre-phase). This is not "measuring a real climb" — it is fabricating one, directly contradicting "reports correct elevation." |
| 3 | A converted trail's distance comes from the smoothed accumulator instead of the raw, GPS-jitter-inflated haversine sum; the dead, misaligned `cumulativeDistance` array is gone | VERIFIED (per locked context decision D-01) | `gpx.ts:144` reads `metrics.totalDistanceSmoothed` (`grep` confirms `totalDistance = metrics.totalDistance;` no longer present). `cumulativeDistance` is not deleted but repaired in place as an index-aligned, raw, per-point array — an explicit, locked product decision in `33-CONTEXT.md` (D-01) because RESEARCH.md found it is a live, wired consumer (the crop slider) whose actual defect was misalignment, not deadness. Jitter fixture reports ~100.075 m (smoothed) vs. ~110.083 m (raw array total) — confirmed in `gpx.test.ts` and independently by `npx vitest run` (31/31 green). See Gaps below: the repaired array's *consumer* (the crop slider) has its own new defects. |
| 4 | A route planned in the web planner reports a distance that follows its anchors instead of cutting the corner at each one | VERIFIED | Root cause was the same `i = 1` loop bound (D-03); fixed for the whole file, not planner-specific. `gpx.test.ts`'s two-leg planner fixture (shared anchor point repeated as the next leg's first point) reports ~444.78 m instead of the pre-fix 333.585 m (a whole 111 m hop restored). Cross-segment continuity preserved: single shared `GpxMetricsComputation` instance, no anchor reset introduced (`grep -nE "lastPointXY|lastFilteredPointXY" gpx.ts` = no matches outside the class). |

**Score:** 2/4 ROADMAP truths fully verified, 2/4 have confirmed regressions (see below); merged with PLAN-level must-haves the phase score is 4/7 (see frontmatter).

### Additional Must-Haves from PLAN Frontmatter (33-03)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 5 | `cumulativeDistance` is index-aligned with `flatten()`, first entry 0, raw and non-decreasing | VERIFIED | `gpx-metrics-computation.ts:62,86` pushes exactly once per `addAndFilter()` call including the first-call branch. `gpx.test.ts` asserts length equality and monotonicity across 4 fixtures (2-point, two-leg planner, empty, jitter). Confirmed via `npx vitest run` (31/31). |
| 6 | The route-crop feature still works: `getCoordinateAtDistance()` never produces a NaN coordinate | ✗ FAILED | Independently reproduced (see CR-01 below): the shipped guard does not cover `rawRouteTotal === 0` or a leading-duplicate-point route; `getCoordinateAtDistance` returns `[NaN, NaN, 1]`, which throws inside MapLibre's `LngLat.convert`. |
| 7 | Crop panel state (`croppedGPX`) does not silently resurrect a discarded route | ✗ FAILED | Confirmed by reading every reference to `croppedGPX` in the file: no reset path exists on the new early-return branch or on `resetRoute()`/`replaceRoute()`/`toggleCropMarkers(false)`. `confirmCrop()` trusts the cache unconditionally. |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `web/src/lib/models/gpx/gpx.ts` | Corrected `getTotals()` loop + centroid divisor; distance sourced from smoothed accumulator | VERIFIED | `for (let i = 0; ...)`, `summedPointCount`, `metrics.totalDistanceSmoothed` all present and wired; `allPoints.length` divisor removed. |
| `web/src/lib/models/gpx/gpx-metrics-computation.ts` | `parseElevation`, undefined-aware elevation, threshold-independent elevation sampling, raw index-aligned `cumulativeDistance` | VERIFIED (with a correctness regression) | `parseElevation` exported and used at every former `?? 0` site. Horizontal gate correctly still governs `totalDistanceSmoothed` only. But the elevation block it decoupled has no noise rejection — see CR-02. |
| `web/src/lib/models/gpx/gpx.test.ts` | Vitest suite covering CONV-01/02/05, D-01 | VERIFIED | 12 tests, all passing; no stub patterns found. |
| `web/src/lib/models/gpx/gpx-metrics-computation.test.ts` | Vitest suite covering CONV-03/04 | VERIFIED but incomplete coverage | 9 tests, all passing; the CONV-04 fixture is a monotonic 12-point climb only — no oscillating/stationary fixture exists, which is exactly why CR-02 was not caught by the shipped suite. |
| `web/src/routes/trail/edit/[id]/+page.svelte` | Crop slider rescaled to raw cumulative total; no NaN coordinates | ⚠️ PARTIALLY WIRED / DEFECTIVE | `rawRouteTotal` is used consistently for both target-distance computations (D-02's rescale intent is correctly implemented) — the guard is present but incomplete (CR-01), and `croppedGPX` staleness is a newly introduced data-loss path (CR-03). |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `gpx.ts` `getTotals()` loop | `gpx-metrics-computation.ts` `addAndFilter()` | called once per point from index 0 | WIRED | Confirmed by grep and by test assertions restoring the dropped first point. |
| `gpx-metrics-computation.ts` `point.ele` | `parseElevation()` | coercion guard replacing `?? 0` | WIRED | All three former sites replaced; zero `point.ele ?? 0` occurrences remain. |
| `gpx.ts` reported `distance` | `metrics.totalDistanceSmoothed` | direct assignment | WIRED | `grep -c "totalDistance = metrics.totalDistanceSmoothed"` = 1. |
| `+page.svelte` `updateCropMarkers()` | `valhallaStore.route.features.cumulativeDistance` | `rawRouteTotal` as percentage basis | WIRED but produces NaN on a real input class | See CR-01 — the wiring is correct, but the value it computes is not always finite/sane for realistic route shapes. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `trail.elevation_gain` / `elevation_loss` (persisted) | `totals.elevationGain/Loss` | `GPX.getTotals()` -> `metrics.totalElevationGain/LossSmoothed` | Real data for genuine climbs, **fabricated data for stationary/noisy tracks** | ⚠️ HOLLOW/INCORRECT — flows end-to-end, but the value is not trustworthy for a class of common real-world inputs (paused/stationary recordings). Since CONV-F01 migration is deferred, any such value saved becomes permanent. |
| `cropStartMarker` / `cropEndMarker` map markers | `[startLon, startLat]` / `[endLon, endLat]` | `getCoordinateAtDistance(flatRoute, cumulativeRoute, targetDistance)` | NaN for a real, reachable input class | ✗ DISCONNECTED for that input class — see CR-01. |

### Behavioral Spot-Checks (executed, not inferred)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full unit suite green (as SUMMARY.md claims) | `cd web && npx vitest run` | `Test Files 5 passed (5)` / `Tests 31 passed (31)` | ✓ PASS |
| CR-02 repro: stationary ±7 m altitude noise, 60 samples, `new GpxMetricsComputation(5,5)` | ad-hoc Vitest probe (built, run, deleted) | `gain 210 loss 203 dist 0` | ✗ FAIL (confirms blocker) |
| CR-01 repro: shipped `getCoordinateAtDistance` + shipped guard logic, `cumulativeRoute = [0,0,0]`, target = 0 | ad-hoc Vitest probe (built, run, deleted) | guard evaluates "passes" = `true`; returned coordinate `[NaN, NaN]` | ✗ FAIL (confirms blocker) |
| CR-01 repro variant: leading duplicate point only, non-zero total (`[0,0,111.19,222.39]`), target = 0 | ad-hoc Vitest probe (built, run, deleted) | guard "passes" = `true`; returned coordinate `[NaN, NaN]` | ✗ FAIL (confirms blocker, matches reviewer's exact fixture) |
| CR-03 repro: grep every `croppedGPX` reference in `+page.svelte` | `grep -n "croppedGPX" "src/routes/trail/edit/[id]/+page.svelte"` | Only one assignment site (line 1504), no reset anywhere | ✗ FAIL (confirms blocker) |
| Commits referenced in SUMMARY.md exist and match claimed content | `git log --oneline` / `git show --stat` for all 9 hashes | All 9 commits found, messages match phase/task | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| CONV-01 | 33-01 | First track point included in distance/bbox/centroid; planner stops dropping opening hop | ✓ SATISFIED | Loop fix + fixtures, independently confirmed. |
| CONV-02 | 33-01 | Centroid divides by the same point count it summed | ✓ SATISFIED | `summedPointCount`, independently confirmed. |
| CONV-03 | 33-02 | Points with no elevation excluded from gain/loss, not treated as 0 m | ✓ SATISFIED | `parseElevation`, independently confirmed via XML-parse-path tests. |
| CONV-04 | 33-02 | Elevation gain/loss sampled independently of the horizontal threshold | ⚠️ PARTIALLY SATISFIED / REGRESSION | The literal requirement (decouple from horizontal threshold) is implemented, but the implementation is unsafe: it fabricates gain/loss from ordinary GPS/altimeter noise on stationary tracks (independently reproduced: 210/203 on a 0 m-travel track). This is a new defect introduced by the phase, not present before. REQUIREMENTS.md marks this `[x]` Complete — that checkbox is not supported by the evidence above. |
| CONV-05 | 33-03 | Distance from smoothed accumulator; misaligned `cumulativeDistance` array replaced | ✓ SATISFIED for the distance-source swap; the array's *consumer* (crop slider) regressed | `totalDistanceSmoothed` correctly wired. `cumulativeDistance` correctly repaired per the locked D-01 decision. However, the crop-slider consumer of this array now has two independently-confirmed defects (CR-01, CR-03) introduced by this same phase's Task 2. |

No orphaned requirements: all five REQUIREMENTS.md IDs mapped to Phase 33 (CONV-01..05) appear in a plan's `requirements` frontmatter (33-01: CONV-01/02; 33-02: CONV-03/04; 33-03: CONV-05).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `web/src/lib/models/gpx/gpx-metrics-computation.ts` | 107-123 | No hysteresis/reversal check on smoothed elevation commit | 🛑 Blocker (CR-02) | Fabricates elevation gain/loss on stationary noise; persisted permanently (CONV-F01 deferred). |
| `web/src/routes/trail/edit/[id]/+page.svelte` | 1480-1485 | Guard checks wrong conditions (`!Number.isFinite` instead of `> 0`) | 🛑 Blocker (CR-01) | NaN coordinates reach MapLibre, uncaught throw on a routine input (coincident leading points). |
| `web/src/routes/trail/edit/[id]/+page.svelte` | 166, 1504, 1513-1521 | `croppedGPX` has no reset path; `confirmCrop()` trusts cache | 🛑 Blocker (CR-03) | Can silently replace a user's current route with a stale crop of a discarded one. |
| `web/src/lib/models/gpx/gpx-metrics-computation.ts` | 125-128 | Trailing sub-threshold residual never flushed into `totalDistanceSmoothed` | ⚠️ Warning (WR-01, code review) | A route shorter than 5 m reports exactly 0; every route under-reports by up to 5 m. Not independently re-verified in this pass (accepted reviewer's static analysis — matches code read). |
| `web/src/lib/models/gpx/gpx-metrics-computation.ts` | 73-78 | `totalDistanceSmoothed` accumulates chord length, not path length, between anchors | ⚠️ Warning (WR-02, code review) | Systematic under-reporting on curvy/switchback geometry; not caught by the shipped collinear-only fixtures. |
| `web/src/lib/models/gpx/gpx.ts` | 106, 111-140 | Single `GpxMetricsComputation` spans `<trk>` boundaries | ⚠️ Warning (WR-03, code review) | Pre-existing, but now locked in by new tests; disjoint tracks accrue a phantom connecting leg. |
| `web/src/lib/components/trail/trail_anchor_list.svelte` | ~221, ~242 | Independent, unfixed copy of the `i = 1` loop bug reading raw (not smoothed) distance | ⚠️ Warning (WR-07, code review; self-flagged in 33-03-SUMMARY.md) | Same screen shows two different, non-summing distances for the same route. Explicitly out of this phase's locked scope per `33-CONTEXT.md`/`33-01-PLAN.md`; carried as a known, disclosed risk — not counted as a gap here. |
| No TBD/FIXME/XXX markers | — | — | — | `grep` across all five modified/created files: none found. |

No debt-marker blocker triggered (Step 7 gate) — clean.

### Human Verification Required

See `human_verification` in frontmatter. Both items stem directly from CR-01/CR-03 and were already flagged by the plan itself as deferred end-of-phase human checks (`33-03-PLAN.md` Task 2 `<human-check>`); this verification pass supplies the code-level proof that they will fail, so they are listed here as required confirmation of the user-facing failure mode rather than open-ended exploration.

### Gaps Summary

Three of the four ROADMAP success criteria's constituent claims hold up under independent, executed verification (CONV-01, CONV-02, CONV-05's distance-source swap, the planner corner-cutting fix). The fourth (elevation correctness) and a directly-adjacent must-have from Plan 33-03 (the crop slider "still works") do not:

1. **CONV-04 overshot its fix.** Removing the horizontal-movement gate without replacing it with a noise-aware filter turns ordinary GPS/altimeter jitter into fabricated elevation gain/loss. Reproduced: 210 m gain / 203 m loss on a 0 m-travel, ±7 m-noise fixture. This directly contradicts the phase goal's "reports correct... elevation" and is not caught by the shipped test suite (only a monotonic-climb fixture exists).
2. **The crop slider's new NaN guard checks the wrong condition.** `!Number.isFinite(rawRouteTotal)` does not reject `rawRouteTotal === 0`, so a route with coincident leading points — a routine GPS artefact, and true by construction for any fully degenerate route — produces `[NaN, NaN]` coordinates that throw inside MapLibre. This is precisely the case Plan 33-03 deferred to end-of-phase human verification, and it fails.
3. **`croppedGPX` staleness is a new data-loss path.** The early return added alongside the (broken) NaN guard exits before `croppedGPX` is ever cleared, and nothing else clears it either. `confirmCrop()` can silently apply a crop of a route the user already discarded.

All three were independently reproduced against the shipped code in this verification pass (not accepted on the reviewer's word alone) via disposable Vitest probes built and deleted during this session, plus direct grep/read confirmation of the `croppedGPX` lifecycle. The code-review's CR-01/CR-02/CR-03 findings are corroborated, not just trusted.

**This does not look like an acceptable intentional deviation** — no override is suggested. All three are regressions introduced by this phase's own changes (not pre-existing, not out-of-scope), directly touch requirements this phase claims complete (CONV-04, CONV-05's crop-slider consumer), and are classified BLOCKER by both the independent code review and this verification pass.

---

_Verified: 2026-07-31T13:05:00Z_
_Verifier: Claude (gsd-verifier)_
