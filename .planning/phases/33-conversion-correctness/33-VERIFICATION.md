---
phase: 33-conversion-correctness
verified: 2026-07-31T16:55:00Z
status: passed
score: 9/9 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 6/9
  gaps_closed:
    - "CR-01 (round 2, 33-05 fix attempt): crop-panel guard mitigation no longer zeroes hand-entered distance/duration/elevation form fields"
    - "CR-02 (round 2, 33-05 fix attempt): crop pins are never stranded visibly at [0,0] on a degenerate route"
    - "CR-03 (round 2, 33-04 fix attempt): totalElevationGainSmoothed/totalElevationLossSmoothed are monotonically non-decreasing by construction, so trail_anchor_list.svelte's per-segment differencing never renders negative elevation"
  gaps_remaining: []
  regressions: []
deferred: []
human_verification_result: passed  # all 3 items confirmed in a live browser, 2026-07-31 — see 33-UAT.md
human_verification:
  - test: "Open a new (empty or near-empty) trail, type distance/duration/elevation values by hand under 'basic info', click 'draw route', then click the crop icon to open the crop panel without drawing anything."
    expected: "The hand-typed distance/duration/elevation values remain unchanged after the crop panel opens (crop preview totals render in the crop panel itself, not in the form)."
    why_human: "Code-level trace and static analysis are conclusive (updateTotals() now has exactly one call site, fed only by valhallaStore.route; cropPreview never touches $formData) — but the exact on-screen sequence in a live browser has not been exercised in this pass."
  - test: "Repeat the same sequence and observe the map after the crop panel opens on a degenerate/empty route; then open the crop panel on a normal route with a real track and drag the slider."
    expected: "No crop pins are visible at [0,0]/off the coast of West Africa on the degenerate route; pins render and track the slider correctly on a normal route."
    why_human: "Marker visibility is now purely derived from state (`cropPanelOpen && cropPreview !== null`) and markers are only constructed once a real coordinate exists, which closes the previous tick-ordering race by construction — but live map rendering (including WR-04's terrain-occlusion opacity caveat) should be confirmed by hand."
  - test: "In the trail-edit anchor list, create a route with at least two segments where the elevation profile rises then returns close to its prior value across a segment boundary (e.g. climb, pause/reverse briefly at the anchor, then continue), and check the per-segment elevation gain/loss shown next to each anchor."
    expected: "No segment ever displays a negative elevation gain or loss."
    why_human: "Reproduced and closed at the unit level: the exact fixture that previously yielded segment gain = -6/-8 now passes (`gpx-metrics-computation.test.ts`, 'per-segment differencing never goes negative'), and re-running that same fixture against the pre-fix class still fails as expected. Confirming the on-screen render (`formatElevation`) in the actual anchor list UI has not been exercised in this pass."
---

# Phase 33: Conversion Correctness Verification Report

**Phase Goal:** Every GPX converted anywhere in Wanderer — a web upload or a server-side conversion — reports correct distance, elevation, and duration, fixing four real defects in the shared TS computation before the Dart port can be pinned against it.

**Verified:** 2026-07-31T16:55:00Z
**Status:** passed (human verification cleared via 33-UAT.md)
**Re-verification:** Yes — third round, after root-cause fixes for round 2's three regressions

## Goal Achievement

Round 1 found 3 BLOCKER gaps. Plans 33-04/33-05 closed them but introduced 3 new BLOCKER
regressions (CR-01/CR-02/CR-03, round 2). This round evaluates three root-cause fixes shipped
as commits `f864add3`, `8da605f5`, `32747e53` (no new PLAN/SUMMARY was produced for this round;
verified directly against the shipped code, not against commit messages or the prior round's
narrative).

**None of the three round-2 regressions are accepted on the executor's or reviewer's word.**
Each was independently re-derived from the current source, and for CR-03 additionally confirmed
by building the pre-fix class from git history and running the new regression suite against it
to see it fail exactly as claimed.

### Previously-Failed Truths — Closure Confirmed (Round 2 → Round 3)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | CR-01: the crop-panel guard's mitigation must not mutate the trail's saved distance/duration/elevation form fields | ✓ VERIFIED (closed) | `+page.svelte:1458-1463`: `toggleCropMarkers()` only sets `cropPanelOpen` and nulls `cropPreview` — it no longer calls `updateTotals()`. `grep -n "updateTotals("` across the file returns exactly one call site (`:1597`, inside `updateTrailWithRouteData()`), fed only by `valhallaStore.route`. `cropPreview`'s totals are rendered via a separate `cropPreviewTotals` prop on `route_editor.svelte` (`:434-459`), never written into `$formData`. |
| 2 | CR-02: the crop-panel guard must not strand visible pins at [0,0] | ✓ VERIFIED (closed) | Marker visibility is now a single derived `$effect` (`+page.svelte:1465-1472`) reading `cropPanelOpen && cropPreview !== null` — no imperative opacity writer remains outside it. `route_editor.svelte`'s `togglePanels()` now calls `onCropToggle(_crop)` synchronously, before `crop`'s reactive re-render (`:142-154`), removing the `tick()`-deferred double-write that caused the round-2 race. Markers (`cropStartMarker`/`cropEndMarker`) are constructed only inside the `if (!basisMatchesRoute \|\| !hasCropInterpolationBasis(...))` early-return's else-path, i.e. only once a real coordinate exists (`:1518-1547`) — so on a degenerate route no marker is ever added to the map for the `$effect` to have to hide. |
| 3 | CR-03: totalElevationGainSmoothed/totalElevationLossSmoothed remain monotonic so a downstream per-segment consumer never renders negative elevation | ✓ VERIFIED (closed) | `gpx-metrics-computation.ts:87-95`: `totalElevationGainSmoothed`/`LossSmoothed` are written only inside `publishPending()`, which is `+=`/`-= (negative pendingDelta)` — both strictly additive. The prior commit-then-retract branch that decremented these fields is gone; retraction now only ever discards an *unpublished* `pendingDelta`. Independently re-verified: checked out the pre-fix class from git history (`f864add3^`), ran the new test suite against it — 16/20 tests fail, including the exact `-8` segment-gain reproduction (`AssertionError: expected -8 to be greater than or equal to 0`); ran the same suite against the shipped class — all pass. `trail_anchor_list.svelte:219-225` reads `totalElevationGainSmoothed`/`totalElevationLossSmoothed` (the monotonic fields), not the non-monotonic `finalElevationGain`/`finalElevationLoss` getters that `gpx.ts` uses for the completed-track total. |

### Round-1 Gaps — Confirmed Still Closed (No Reopening)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 4 | CONV-04: a steep, low-horizontal-movement stretch is measured instead of skipped, without fabricating gain/loss from GPS/altitude noise | ✓ VERIFIED | Elevation is still evaluated on every sample with no `smoothedDistance`-gated early return (`gpx-metrics-computation.ts:178-229`); the 88 m scramble fixture and the stationary-noise fixtures both pass in the current suite. |
| 5 | D-02/33-03: `getCoordinateAtDistance()` never produces a NaN coordinate | ✓ VERIFIED | `crop.ts:44-53` unchanged since round 2's verification: `span > 0` guard, explicit `points.length === 0`/`< 2` early returns. No diff to this file since round 2. |
| 6 | D-02/33-03: crop panel state does not resurrect a route the user already discarded | ✓ VERIFIED | `cropPreview` (successor to `croppedGPX`) is nulled at 3 sites (`toggleCropMarkers(false)`, the degenerate-route guard, `updateTrailWithRouteData()`'s choke point) and `confirmCrop()` hoists a local `confirmedCrop` before the choke point can null it mid-apply (`:1579-1591`) — same pattern round 2 verified, now built on the redesigned `cropPreview` state object rather than `croppedGPX`. |

### New Findings From This Round's Code Review — Scope Determination

`33-REVIEW.md` (this round) reports 2 new criticals and 11 warnings. Per the orchestrator's
guidance, both criticals were independently re-confirmed as pre-existing and outside this
phase's diff, not phase-33 regressions:

| # | Finding | Status | Evidence |
|---|---------|--------|----------|
| — | Review CR-01: `correctElevation()`/`recalculateElevationData()` crashes when a Valhalla height response omits `height` entirely | Pre-existing, out of phase-33 scope | `git diff 9933f364..HEAD -- web/src/lib/models/gpx/gpx.ts` shows the `correctElevation()` method (lines 188-228) byte-identical to the phase's base commit; the only change in this file across the whole phase is the `getTotals()` elevation-source swap to `finalElevationGain/Loss`. Confirmed independently, not accepted on the review's or the prompt's word alone. |
| — | Review CR-02: `initRouteAnchors()` dereferences the last point of an empty final `<trkseg>` without a length guard | Pre-existing, out of phase-33 scope | The offending code (`+page.svelte:576-596`, unguarded deref at 588) sits outside every diff hunk this phase produced against `+page.svelte` (hunks cluster at lines 19-30, 65-95, 160-190, 1398-1594, 2256-2298, 2442-2484 — none overlap 573-596). `git log` traces the code to `b3ed2b10` ("more route drawing"), which sits far outside phase 33's commit range. Confirmed independently via `git diff`, not accepted on the prompt's claim alone. |

Both are real defects and are recorded as follow-up debt below (Anti-Patterns), but they do not
block this phase — they were not introduced or touched by any 33-0X plan or this round's fixes.

**Score:** 9/9 must-haves verified (3 round-1 truths, confirmed still closed + 3 round-2
regressions, now genuinely closed + verification of no reopening). The phase goal, as scoped by
REQUIREMENTS.md to `web/src/lib/models/gpx/gpx.ts`, `gpx-metrics-computation.ts`, and
`web/src/lib/util/gpx_util.ts`, is achieved.

### WR-01 / WR-02 Scope Judgment (trail_anchor_list.svelte)

The review's WR-01 (segment list reads raw `totalDistance` while the form reads
`totalDistanceSmoothed`) and WR-02 (`for (let i = 1; ...)` drops each segment's opening hop,
the same defect shape CONV-01 fixed in `gpx.ts`) both concern `trail_anchor_list.svelte`. Judged
**out of phase 33's scope**, for three independently-checked reasons, not merely the review's own
classification:

1. **Pre-existing, untouched by this phase.** `git log --follow` shows `trail_anchor_list.svelte`'s only commit is `cbff6472` (2026-06-06), and `git diff 9933f364..HEAD -- web/src/lib/components/trail/trail_anchor_list.svelte` is empty — no 33-0X plan or this round's fixes ever modified this file.
2. **REQUIREMENTS.md explicitly scopes CONV-01..05 to three files** ("Fixes to the shared GPX→trail metrics computation, applied to `web/src/lib/models/gpx/gpx.ts`, `web/src/lib/models/gpx/gpx-metrics-computation.ts`, and `web/src/lib/util/gpx_util.ts`") — `trail_anchor_list.svelte` is not one of them.
3. **It is not a "GPX conversion" path.** `gpx2trail()` (`gpx_util.ts:74-76`), the function the upload and server-side conversion routes call, reads `totals.elevationGain/Loss/distance` from the corrected `gpx.getTotals()` — confirming the phase's actual "GPX converted anywhere" claim holds for the conversion pipeline. `trail_anchor_list.svelte` is a route-planner display widget that independently re-implements its own `GpxMetricsComputation` loop over an already-drawn route; it never converts a GPX.

This is real, user-visible drift (the anchor list's segment total will not sum to the form's
route total, and a route's first leg will under-report by exactly the CONV-01 amount) and is
recommended as a dedicated follow-up item — but it does not gate this phase's pass, and is not
counted as a phase-33 gap.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `web/src/lib/models/gpx/gpx-metrics-computation.ts` | Monotonic smoothed elevation totals via defer-then-publish | ✓ VERIFIED | `publishPending()` is the sole writer of `totalElevationGainSmoothed`/`LossSmoothed`, strictly additive. `finalElevationGain`/`finalElevationLoss` getters surface the pending excursion for completed-track totals. |
| `web/src/lib/models/gpx/gpx-metrics-computation.test.ts` | Monotonicity + per-segment differencing regression fixtures | ✓ VERIFIED | New `describe` blocks "smoothed elevation totals are monotonic" (6 fixtures) and "per-segment differencing never goes negative" (1 fixture); independently confirmed to fail against the pre-fix class (16/20 failures reproduced, including the `-8` segment gain) and pass against the shipped class. |
| `web/src/routes/trail/edit/[id]/+page.svelte` | Crop preview isolated from `$formData`; marker visibility derived, not imperative | ✓ VERIFIED | Single `updateTotals()` call site; `cropPreview` state object separate from `$formData`; markers constructed lazily; visibility is a pure `$effect`. |
| `web/src/lib/components/trail/route_editor.svelte` | `cropPreviewTotals` prop renders preview without touching the form; `togglePanels()` reports state synchronously | ✓ VERIFIED | `cropPreviewTotals?.distance/duration/elevationGain/elevationLoss` rendered directly in the crop panel (`:434-459`); `onCropToggle(_crop)` called synchronously, not deferred behind `tick()`. |
| `web/src/lib/stores/valhalla_store.svelte.ts` | `deleteFromRoute()` recomputes `features` from the mutated (spliced) snapshot | ✓ VERIFIED | `snapshot.features = snapshot.getTotals()` runs after the splice, on `snapshot` (the mutated object), not on the stale pre-splice `valhallaStore.route`. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `gpx-metrics-computation.ts addAndFilter()` | `publishPending()` | direct call on confirm path | WIRED | Only additive writer of the monotonic fields; confirmed by reading and by executed before/after test runs. |
| `gpx.ts getTotals()` | `metrics.finalElevationGain/Loss` | direct field read | WIRED | Completed-track totals correctly include a still-pending excursion; `totalElevationGain/LossSmoothed` deliberately not used here (those are for snapshot-differencing consumers). |
| `trail_anchor_list.svelte subtractMetrics()` | `metrics.totalElevationGainSmoothed/LossSmoothed` | two consecutive snapshots subtracted | WIRED, monotonicity contract now upheld by the source | Verified the consumer reads the monotonic fields, and the source now guarantees monotonicity — the round-2 defect is closed at the contract boundary, not patched at the consumer. |
| `+page.svelte updateCropMarkers()` | `+page.svelte $effect` (marker opacity) | shared `cropPreview`/`cropPanelOpen` state, no direct call | WIRED | Confirmed no remaining direct `setOpacity` call outside the `$effect` and the lazy-creation block; state-derived visibility removes the ordering dependency that caused CR-02. |
| `web/src/lib/util/gpx_util.ts gpx2trail()` | `GPX.getTotals()` | `totals.elevationGain/Loss/distance` | WIRED | Confirms the corrected shared computation reaches the upload/server-side conversion path, not just the trail-edit page. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `trail.elevation_gain`/`elevation_loss` (persisted, any conversion path) | `totals.elevationGain/Loss` | `GPX.getTotals()` → `finalElevationGain/Loss` | Correct for stationary/noisy input, monotonic-by-construction internally | ✓ FLOWING |
| `$formData.distance/duration/elevation_gain/elevation_loss` on crop-panel guard fire | N/A — no longer written by the guard path | `toggleCropMarkers()` no longer calls `updateTotals()` | N/A | ✓ CLOSED — the round-2 destructive write no longer exists |
| `trail_anchor_list.svelte` per-segment `elevationGain`/`elevationLoss` display | `subtractMetrics(cumulative, previous)` on `totalElevationGainSmoothed`/`LossSmoothed` | Two `GpxMetricsComputation` snapshots | Non-negative by construction now (source is monotonic) | ✓ FLOWING for elevation. Distance basis (`totalDistance`, raw) still diverges from the form's `totalDistanceSmoothed` — see WR-01, judged out of phase-33 scope above. |

### Behavioral Spot-Checks (executed, not inferred)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full unit suite green | `cd web && npx vitest run` | `Test Files 7 passed (7)` / `Tests 53 passed (53)` | ✓ PASS |
| gpx model suite in isolation | `cd web && npx vitest run src/lib/models/gpx` | `Test Files 3 passed (3)` / `Tests 41 passed (41)` | ✓ PASS |
| valhalla_store + crop suites in isolation | `cd web && npx vitest run src/lib/stores/valhalla_store.test.ts src/lib/models/gpx/crop.test.ts` | `Test Files 2 passed (2)` / `Tests 11 passed (11)` | ✓ PASS |
| `svelte-check` | `cd web && npx svelte-check --tsconfig ./tsconfig.json` | `2462 FILES 0 ERRORS 0 WARNINGS` | ✓ PASS |
| CR-03 regression proof: new monotonicity/segment-differencing tests against the **pre-fix** class (`f864add3^`) | swapped `gpx-metrics-computation.ts` for the pre-fix version, ran the new test file, restored the fixed version, confirmed `git status` clean afterward | `16 failed \| 4 passed (20)`, including `expected -8 to be greater than or equal to 0` | ✓ FAIL as expected (proves the new tests are not vacuous) |
| Same tests against the **shipped** class | `npx vitest run src/lib/models/gpx/gpx-metrics-computation.test.ts` | `20/20 passed` | ✓ PASS |
| `gpx.ts correctElevation()` byte-identical to phase base | `git diff 9933f364..HEAD -- web/src/lib/models/gpx/gpx.ts` | Only the `getTotals()` elevation-source hunk shown; `correctElevation()` untouched | ✓ CONFIRMS review CR-01 is pre-existing |
| `+page.svelte` diff hunks vs. `initRouteAnchors()` line range | `git diff 9933f364..HEAD -- "web/src/routes/trail/edit/[id]/+page.svelte"` (hunk headers) | No hunk overlaps lines 573-596 | ✓ CONFIRMS review CR-02 is pre-existing |
| `trail_anchor_list.svelte` untouched by phase 33 | `git diff 9933f364..HEAD -- web/src/lib/components/trail/trail_anchor_list.svelte` | Empty diff | ✓ CONFIRMS WR-01/WR-02 predate and sit outside this phase |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| CONV-01 | 33-01 | First track point included in distance/bbox/centroid | ✓ SATISFIED | Unaffected by rounds 2-3; `gpx.ts:126` loop starts at `i = 0`; re-confirmed passing in the full suite. |
| CONV-02 | 33-01 | Centroid divides by the same point count it summed | ✓ SATISFIED | Unaffected; `summedPointCount` present and used as the sole divisor. |
| CONV-03 | 33-02 | Points with no elevation excluded from gain/loss | ✓ SATISFIED | Unaffected; `parseElevation()` returns `undefined` for missing/empty `<ele>`. |
| CONV-04 | 33-02, 33-04 | Elevation gain/loss sampled independently of the horizontal threshold, without fabricating on noise | ✓ SATISFIED | The round-2 regression (non-monotonic totals breaking `trail_anchor_list.svelte`) is closed; defer-then-publish keeps `totalElevationGainSmoothed/LossSmoothed` monotonic while still crediting genuine low-horizontal climbs. All fixtures (stationary 0/0, mid-swing 7/0, 88 m scramble, rolling terrain) pass. |
| CONV-05 | 33-03, 33-05 | Distance from smoothed accumulator; crop-slider consumer of the repaired `cumulativeDistance` array works | ✓ SATISFIED | The round-2 regressions (form-data zeroing, stranded pins) are closed; `totalDistanceSmoothed` is the reported distance; crop preview isolated from `$formData`; markers derived and lazily created. |

No orphaned requirements: all five REQUIREMENTS.md IDs mapped to Phase 33 (CONV-01..05) appear in
a plan's `requirements` frontmatter (33-01: CONV-01/02; 33-02: CONV-03/04; 33-03: CONV-05; 33-04:
CONV-04 gap closure; 33-05: CONV-05 gap closure). CONV-06 and the PORT-0x requirements remain
correctly unaddressed (out of Phase 33's scope, tracked for Phase 34+).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `web/src/lib/models/gpx/gpx.ts` | 207-227 | `correctElevation()` dereferences `heightResponse.height[heightIndex]` (not the locally-guarded `heights` array) and its length guard only checks `finiteHeights === 0`, not `heights.length === 0` | ⚠️ Warning — pre-existing, confirmed byte-identical to phase-33's base commit, not a phase-33 regression | Crashes `recalculateElevationData()` if a Valhalla height response omits `height`; the caller has no `try/catch`, so the error is silently swallowed as an unhandled rejection. Recommend a follow-up fix; out of this phase's scope. |
| `web/src/routes/trail/edit/[id]/+page.svelte` | 576-596 | `initRouteAnchors()`'s last-point branch dereferences `points[points.length - 1]` without the length guard used two lines earlier | ⚠️ Warning — pre-existing (traces to `b3ed2b10`, "more route drawing"), confirmed via diff hunks, not a phase-33 regression | Opening a saved trail whose GPX ends in an empty `<trkseg>` throws during `onMount`, making the edit page unusable for that trail. Recommend a follow-up fix; out of this phase's scope. |
| `web/src/lib/components/trail/trail_anchor_list.svelte` | 219-225 | `snapshotMetrics()` reads `metrics.totalDistance` (raw) while `gpx.ts` reports `totalDistanceSmoothed` for the same route | ⚠️ Warning — pre-existing, file untouched by any 33-0X plan, out of REQUIREMENTS.md's stated CONV-01..05 file scope | Segment distances in the anchor list will not sum to the form's reported total distance; visible drift on any GPS-jittery track. See "WR-01/WR-02 Scope Judgment" above. Recommend a dedicated follow-up plan. |
| `web/src/lib/components/trail/trail_anchor_list.svelte` | 241-251 | `for (let i = 1; i < points.length; i++)` drops the opening hop of segment 0, the exact defect shape CONV-01 fixed in `gpx.ts` | ⚠️ Warning — pre-existing, same scope reasoning as above | First route segment's distance/elevation under-reports in the anchor list. Same defect class as CONV-01, different (out-of-scope) file. Strongly recommend folding into a near-term follow-up plan given the direct conceptual overlap with this phase's own fix. |
| `web/src/routes/trail/edit/[id]/+page.svelte` | various (WR-03, WR-04, WR-06 through WR-11 per `33-REVIEW.md`) | Assorted robustness gaps: unguarded `addTo(map!)`, terrain-occlusion opacity edge case, crop panel going inert on route mutation, `cropPanelOpen` not cleared on every unmount path, dead code/comments | ℹ️ Info / low-severity Warning, not independently re-verified line-by-line in this pass beyond spot checks | None of these affect the phase's stated distance/elevation/duration correctness goal; recorded for a future hardening pass, not phase-33 blocking. |
| No TBD/FIXME/XXX markers | — | — | — | `grep` across all files touched or read in this pass: none found. |

No debt-marker blocker triggered (Step 7 gate) — clean.

### Human Verification Required

See `human_verification` in frontmatter. All three items now have strong automated/executed
evidence (unit-level reproduction of the exact pre-fix failure, now passing) — they are listed to
confirm the on-screen behavior in a live browser/map session, since none of this phase's changes
are covered by browser/e2e tests and this is the third round of fixes to the same
data-loss-sensitive area.

### Gaps Summary

No gaps remain. The three round-2 regressions (CR-01: form-data zeroing, CR-02: stranded crop
pins, CR-03: non-monotonic elevation totals) are all genuinely closed, confirmed independently in
this pass — not accepted on commit messages, SUMMARY.md, or the prior round's narrative. CR-03's
closure was additionally confirmed by resurrecting the pre-fix class from git history and watching
the new regression suite fail against it exactly as claimed (`-8` segment gain reproduced), then
pass against the shipped class.

This round's fresh code review surfaced 2 new criticals and a cluster of warnings. Both criticals,
and the trail_anchor_list.svelte-related warnings (WR-01, WR-02), were independently confirmed via
`git diff`/`git log` to be pre-existing defects outside every file this phase's plans (33-01
through 33-05) or this round's three fix commits ever touched — not new regressions introduced by
phase 33's own work, unlike the round-2 regressions were. They are recorded as follow-up debt, with
WR-01/WR-02 flagged for priority given their direct conceptual overlap with CONV-01/CONV-05's own
defect classes.

The phase goal — "every GPX converted anywhere in Wanderer... reports correct distance,
elevation, and duration" — is achieved for the shared TS computation and its actual conversion
consumers (trail-edit, upload, server-side conversion via `gpx2trail()`), which is what
REQUIREMENTS.md scopes CONV-01..05 to. Status is `human_needed` rather than `passed` only because
the crop-panel visual behavior and the anchor-list's on-screen elevation rendering — both areas
with two prior rounds of regressions — have not been exercised in an actual browser in this pass.

---

_Verified: 2026-07-31T16:55:00Z_
_Verifier: Claude (gsd-verifier)_
