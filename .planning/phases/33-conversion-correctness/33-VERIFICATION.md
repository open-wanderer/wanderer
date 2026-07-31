---
phase: 33-conversion-correctness
verified: 2026-07-31T13:10:00Z
status: gaps_found
score: 6/9 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 4/7
  gaps_closed:
    - "CONV-04: a steep, low-horizontal-movement stretch is measured instead of skipped, without fabricating gain/loss from GPS/altitude noise"
    - "D-02 / 33-03 must-have: the route-crop feature still works — getCoordinateAtDistance() never produces a NaN coordinate"
    - "D-02 / 33-03 must-have: crop panel state does not resurrect a route the user already discarded"
  gaps_remaining: []
  regressions:
    - "CR-01 (new, 33-05): the degenerate-route crop guard's own mitigation (toggleCropMarkers(false) -> updateTotals(valhallaStore.route)) silently zeroes hand-entered distance/duration/elevation_gain/elevation_loss form fields, and those zeros are persisted on Save."
    - "CR-02 (new, 33-05): the guard's marker-hiding (opacity '0') is unconditionally undone one tick later by route_editor.svelte's togglePanels() calling onCropToggle(true), so crop pins are still stranded visibly at [0,0] on a degenerate route — the exact WR-05 defect the guard's own comment claims to prevent."
    - "CR-03 (new, 33-04): the commit-then-retract filter makes totalElevationGainSmoothed/totalElevationLossSmoothed non-monotonic (a retraction decreases a total that a downstream consumer assumes only increases). trail_anchor_list.svelte's per-segment display subtracts consecutive snapshots of these totals and can render negative elevation gain/loss when a retraction straddles a segment boundary — independently reproduced with an executed probe (segment gain = -6)."
gaps:
  - truth: "The crop-panel degenerate-route guard's mitigation must not mutate the trail's saved distance/duration/elevation form fields"
    status: failed
    reason: >
      33-05's fix for the NaN-coordinate defect (CR-01/CR-03 from the prior round) added
      `toggleCropMarkers(false)` to the guard's early-return branch. `toggleCropMarkers`'s
      inactive branch unconditionally calls `updateTotals(valhallaStore.route)`, which writes
      `distance`, `duration`, `elevation_gain`, `elevation_loss` straight into `$formData` —
      all four are real, hand-editable `TextField`s (`+page.svelte:2182-2200`) that the user may
      have typed in manually. The guard fires whenever the route has no positive interpolation
      basis (empty or near-empty), so `valhallaStore.route.features` is
      `{distance: 0, duration: 0, elevationGain: 0, elevationLoss: 0}` at that moment — the
      guard's own "fix" overwrites the user's real numbers with zeros. `RouteEditor` (and
      therefore the crop panel) renders whenever `drawingActive` is true
      (`+page.svelte:2413-2432`), independent of `routeHasTrackPoints()`, so this is reachable
      through plain UI on a brand-new trail (type metrics by hand -> click "draw route" ->
      click the crop icon) with no malformed data required. `onSubmit` persists the felte
      `form` object directly (`+page.svelte:269-306`), so the zeroed values are written to the
      database on Save. This is not a corner case of the original gap — it is a new data-loss
      path introduced by this round's own fix, and it directly contradicts 33-05's stated
      purpose ("no silent data loss of a user's route").
    artifacts:
      - path: "web/src/routes/trail/edit/[id]/+page.svelte"
        issue: "toggleCropMarkers()'s inactive branch (lines 1428-1439) unconditionally calls updateTotals(valhallaStore.route); updateCropMarkers()'s degenerate-route guard (lines 1483-1492) calls toggleCropMarkers(false), so opening the crop panel on an empty/near-empty route zeroes $formData.distance/duration/elevation_gain/elevation_loss."
    missing:
      - "The guard must not mutate form state: hide the markers directly (setOpacity) instead of routing through toggleCropMarkers(false)."
      - "toggleCropMarkers(false)'s updateTotals(valhallaStore.route) call should be conditional on a crop preview having actually been applied (e.g. only run when croppedGPX was non-null before this call), so closing/discarding a crop never clobbers hand-entered metrics that were never derived from the GPX."
      - "A regression test/fixture: opening the crop panel on a route with no interpolation basis must leave $formData's distance/duration/elevation fields untouched."
  - truth: "The crop-panel degenerate-route guard actually keeps the crop pins hidden — it must not strand them visibly at [0,0]"
    status: failed
    reason: >
      The guard's own comment (`+page.svelte:1483-1492`) claims it prevents "stranding two
      visible pins at 0N 0E", closing the prior round's WR-05. It does not: `route_editor.svelte`'s
      `togglePanels()` sets `crop = true` (mounting `DoubleSlider`), awaits `tick()`, and only then
      calls `onCropToggle(_crop)` — i.e. `toggleCropMarkers(true)` — unconditionally, regardless of
      what happened during the tick. `DoubleSlider`'s `onMount` binds noUiSlider's `update` event,
      and per noUiSlider's own `bindEvent()` ("If the event bound is 'update,' fire it immediately
      for all handles" — `node_modules/nouislider/dist/nouislider.js`), that fires
      `updateCropMarkers([0, 100])` synchronously during the DOM flush the `tick()` is awaiting.
      Sequence on every crop-panel open: (1) `updateCropMarkers` constructs the markers at [0,0]
      and adds them to the map (`:1468-1469`, before the guard); (2) the guard fires for a
      degenerate route and calls `toggleCropMarkers(false)`, setting opacity "0"; (3) `tick()`
      resolves back in `route_editor.svelte`; (4) `onCropToggle(true)` fires unconditionally
      (since `_crop` was `true`) and calls `toggleCropMarkers(true)`, setting opacity back to "1".
      Net effect: two fully visible pins at Null Island on every degenerate-route crop-panel open
      — the exact defect the new comment claims is fixed.
    artifacts:
      - path: "web/src/routes/trail/edit/[id]/+page.svelte"
        issue: "toggleCropMarkers(true) unconditionally re-shows markers with no memory of the guard's decision (lines 1428-1439)."
      - path: "web/src/lib/components/trail/route_editor.svelte"
        issue: "togglePanels() (lines 125-131) calls onCropToggle(_crop) unconditionally after tick(), overriding whatever the guard already decided during that same tick."
    missing:
      - "Track crop-basis availability in +page.svelte state (e.g. cropBasisAvailable, set at the top of updateCropMarkers()) and make toggleCropMarkers(true) respect it instead of unconditionally setting opacity '1'."
      - "Alternatively: construct/add the crop markers to the map only after the guard passes, so nothing is ever added at [0,0] in the first place."
  - truth: "totalElevationGainSmoothed / totalElevationLossSmoothed remain monotonically non-decreasing so a downstream per-segment consumer never renders negative elevation gain or loss"
    status: failed
    reason: >
      33-04's commit-then-retract filter (`gpx-metrics-computation.ts:161-170`) can *decrease*
      `totalElevationGainSmoothed`/`totalElevationLossSmoothed` when a retraction fires (by
      design, to undo a prior commit). The class's own comment only claims the totals "cannot go
      negative" — true — but the class was already non-monotonic before this defect: an existing
      consumer, `trail_anchor_list.svelte`, computes each anchor segment's displayed
      elevation gain/loss by snapshotting the running instance at each segment boundary and
      subtracting the previous snapshot (`snapshotMetrics`/`subtractMetrics`,
      `trail_anchor_list.svelte:219-233`, consumed at `:262-270` and rendered at `:481-486`/
      `:497-501`). That subtraction is only valid while the totals are non-decreasing across the
      whole route. Independently reproduced with an executed Vitest probe against the shipped
      `GpxMetricsComputation` class (built and deleted during this verification pass): committing
      +6 then +6 (running total 12, the "previous" segment-boundary snapshot) followed by a
      point that retracts the second commit (running total 6, the "cumulative" snapshot at the
      next boundary) yields `cumulativeSnapshotGain - previousSnapshotGain = -6` — a negative
      displayed segment gain. Planner segments share a duplicated anchor point (identical
      coordinates -> horizontal distance 0 < thresholdXY_m), so the horizontal-stillness half of
      the retraction condition is trivially satisfied at every segment boundary; only the ±5 m
      elevation-return condition gates whether a given boundary is affected. This is a new
      defect: before 33-04, the smoothed elevation totals were accumulate-only and could not
      decrease, so no consumer needed a monotonicity guard.
    artifacts:
      - path: "web/src/lib/models/gpx/gpx-metrics-computation.ts"
        issue: "Lines 161-170: the retraction branch decreases totalElevationGainSmoothed/totalElevationLossSmoothed with no monotonicity contract documented or enforced for callers that snapshot the running instance."
      - path: "web/src/lib/components/trail/trail_anchor_list.svelte"
        issue: "Lines 219-233: subtractMetrics() assumes metrics.elevationGain/elevationLoss are non-decreasing across segment boundaries; no clamp, no test."
    missing:
      - "Either (a) make retraction non-observable to snapshot consumers — buffer the pending commit and only fold it into the public totals once it can no longer be retracted — or (b) clamp the consumer's subtraction to >= 0 and document that the class does not guarantee monotonicity, accepting that the retracted amount is then lost from the segment it was earned in."
      - "A regression test (in gpx-metrics-computation.test.ts or a new trail_anchor_list.svelte test) asserting monotonic non-decrease across a retraction, or asserting the documented replacement contract if (b) is chosen."
deferred: []
human_verification:
  - test: "Open a new (empty or near-empty) trail, type distance/duration/elevation values by hand under 'basic info', click 'draw route', then click the crop icon to open the crop panel without drawing anything."
    expected: "The hand-typed distance/duration/elevation values remain unchanged after the crop panel opens."
    why_human: "The code-level trace (guard -> toggleCropMarkers(false) -> updateTotals(valhallaStore.route) -> $formData zeroed) is conclusive as a data-loss mechanism, but the exact on-screen sequence and whether the zeroed values are visible before Save should be confirmed by hand."
  - test: "Repeat the same sequence and observe the map after the crop panel opens on a degenerate/empty route."
    expected: "No crop pins are visible at [0,0] / off the coast of West Africa."
    why_human: "The nouislider/tick() sequencing is confirmed by reading nouislider's own bindEvent() source and route_editor.svelte's togglePanels(), but the actual on-screen marker visibility should be confirmed in a live browser."
  - test: "In the trail-edit anchor list, create a route with at least two segments where an anchor's elevation profile rises then returns to (or near) its prior value across a segment boundary (e.g. draw a route that climbs 6-12 m, pauses/reverses briefly at the anchor, then continues), and check the per-segment elevation gain/loss shown next to each anchor."
    expected: "No segment ever displays a negative elevation gain or loss."
    why_human: "Reproduced programmatically against the shipped GpxMetricsComputation class with a hand-built probe (segment gain = -6); confirming the exact UI rendering (formatElevation of a negative number) should be checked by hand since it depends on real route-drawing interaction, not just the class in isolation."
---

# Phase 33: Conversion Correctness Verification Report

**Phase Goal:** Every GPX converted anywhere in Wanderer — a web upload or a server-side conversion — reports correct distance, elevation, and duration, fixing four real defects in the shared TS computation before the Dart port can be pinned against it.

**Verified:** 2026-07-31T13:10:00Z
**Status:** gaps_found
**Re-verification:** Yes — after gap closure (33-04, 33-05)

## Goal Achievement

This is the second verification round for Phase 33. The first round (2026-07-31T13:05:00Z)
found 3 BLOCKER gaps. Plans 33-04 and 33-05 were written and executed to close them. All three
are now genuinely closed — confirmed independently below, not accepted on SUMMARY.md's word.
However, a fresh code review (`33-REVIEW.md`, run after 33-04/33-05 landed) found that the
closure work introduced 3 new BLOCKER-level regressions. This verification pass independently
re-derived and confirmed all three against the actual shipped code (one via an executed Vitest
probe, two via call-chain tracing including reading `nouislider`'s own event-binding source) —
they are not accepted on the reviewer's word alone.

### Previously-Failed Truths — Closure Confirmed

| # | Truth (from prior VERIFICATION.md) | Status | Evidence |
|---|-------|--------|----------|
| 1 | CONV-04: a steep, low-horizontal-movement stretch is measured instead of skipped, without fabricating gain/loss from GPS/altitude noise | ✓ VERIFIED (closed) | `gpx-metrics-computation.ts:130-184` replaces the flat threshold with a commit-then-retract filter (`retractableDelta`, `preRetractZ`, `lastFilteredZPointXY`). `npx vitest run` (44/44 green) includes the 61-sample stationary fixture (0/0), the 60-sample ends-mid-swing fixture (7/0), the stationary-bump-then-climb fixture (24/0), the rolling-terrain guard (24/16), and the pre-existing 88 m scramble (88/0) all passing in the same run. Read the full decision order in the source; matches the plan exactly. |
| 2 | D-02/33-03: `getCoordinateAtDistance()` never produces a NaN coordinate | ✓ VERIFIED (closed) | `web/src/lib/models/gpx/crop.ts:44-53` computes `span = nextDist - prevDist` and only divides when `span > 0`, else `ratio = 0`; explicit early returns for `points.length === 0` and `< 2` inputs. `crop.test.ts`'s leading-duplicate (`[0,0,111.19,222.39]`) and all-identical (`[0,0,0]`) fixtures both pass. `+page.svelte`'s guard now uses `hasCropInterpolationBasis(cumulativeRoute)` (`total > 0`, not merely finite) — `grep -c "Number.isFinite(rawRouteTotal)"` = 0. |
| 3 | D-02/33-03: crop panel state does not resurrect a route the user already discarded | ✓ VERIFIED (closed, with one non-blocking gap — see WR-12 below) | `croppedGPX` is reset to `null` at 5 sites: the degenerate-route guard (`:1489`), `toggleCropMarkers(false)` (`:1436`), `updateTrailWithRouteData()` (`:1535`, the choke point all 16 route-mutation call sites reach), `resetTrail()` (`:1400`), `replaceRoute()` (`:1411`). `confirmCrop()` hoists `confirmedCrop = croppedGPX` before the choke point can null it mid-apply (`:1524-1531`). One path, `handleFileSelection()` (`:440-544`), does not clear it directly (WR-12) — confirmed by reading the function — but it is currently defence-in-depth-only since `replaceRoute()` clears it first on the normal replace flow; classified Warning, not Blocker, matching the code review. |

### New Regressions Introduced By The Closure Work

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 4 | The crop-panel degenerate-route guard's mitigation must not mutate the trail's saved distance/duration/elevation | ✗ FAILED | Independently traced: `updateCropMarkers()`'s guard (`+page.svelte:1483-1492`) calls `toggleCropMarkers(false)`, whose inactive branch unconditionally calls `updateTotals(valhallaStore.route)` (`:1428-1439`, `:1546-1555`), writing `distance`/`duration`/`elevation_gain`/`elevation_loss` into `$formData` from the empty/degenerate route's zero-valued `features`. These are real, hand-editable `TextField`s (`:2182-2200`), and `RouteEditor`/the crop panel render whenever `drawingActive` is true (`:2413-2432`), not gated on `routeHasTrackPoints()`. `onSubmit` persists `$formData` directly (`:269-306`). Reachable through plain UI with no route drawn. |
| 5 | The crop-panel degenerate-route guard must not strand visible pins at [0,0] | ✗ FAILED | Independently traced through 3 files: markers are added to the map at `[0,0]` before the guard runs (`+page.svelte:1468-1469`); the guard hides them via `toggleCropMarkers(false)` (opacity "0"); but `route_editor.svelte`'s `togglePanels()` (`:125-131`) unconditionally calls `onCropToggle(true)` after `await tick()`, re-showing them (opacity "1"). Confirmed `DoubleSlider`'s `onMount` (`double_slider.svelte:26-47`) binds noUiSlider's `update` listener during that same tick, and noUiSlider's own `bindEvent()` fires the `update` callback immediately upon binding (read directly from `node_modules/nouislider/dist/nouislider.js`: "If the event bound is 'update,' fire it immediately for all handles"). This is the exact prior-round WR-05 defect the guard's new comment claims to close. |
| 6 | `totalElevationGainSmoothed`/`totalElevationLossSmoothed` remain monotonic so a downstream per-segment consumer never shows negative elevation | ✗ FAILED | Independently reproduced with an executed, disposable Vitest probe against the shipped `GpxMetricsComputation` class (built and deleted during this pass): two commits of +6 each (running total 12) followed by a retracting point (running total 6) yields a segment subtraction of `6 - 12 = -6`. `trail_anchor_list.svelte`'s `subtractMetrics()` (`:227-233`) performs exactly this subtraction on live route data, with no clamp. Planner segments share a duplicated anchor coordinate, trivially satisfying the retraction's horizontal-stillness condition at every segment boundary. |

**Score:** 6/9 must-haves verified (3 original gaps closed + 3 new regressions found and confirmed). The phase goal is not yet achieved — the gap-closure work traded 3 known defects for 3 new ones in the same subsystem.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `web/src/lib/models/gpx/gpx-metrics-computation.ts` | Commit-then-retract elevation filter; totals never fabricate on stationary noise | ⚠️ VERIFIED for its own must-haves, but a correctness regression (CR-03) sits on top of it | `retractableDelta`, `preRetractZ`, `lastFilteredZPointXY` present and wired exactly as specced. Totals themselves never go negative (confirmed) but are not monotonic, which a downstream consumer relies on. |
| `web/src/lib/models/gpx/gpx-metrics-computation.test.ts` | Stationary-noise regression fixtures | ✓ VERIFIED | 4 new fixtures (61-sample 0/0, 60-sample 7/0, bump+climb 24/0, rolling-terrain guard 24/16) all pass; `grep -c "stationary"` >= 3; no disk fixtures. |
| `web/src/lib/models/gpx/crop.ts` | Testable, degenerate-safe crop interpolation | ✓ VERIFIED | `getCoordinateAtDistance` and `hasCropInterpolationBasis` exported, `span > 0` guard present, no surviving `nextDist - prevDist)` inline division. |
| `web/src/lib/models/gpx/crop.test.ts` | Regression tests for leading-duplicate and all-identical-point routes | ✓ VERIFIED | Both degenerate fixtures plus 2 normal-route fixtures plus a 5-case `hasCropInterpolationBasis` predicate block, all passing. |
| `web/src/routes/trail/edit/[id]/+page.svelte` | Corrected crop guard, imported interpolation, croppedGPX lifecycle that cannot go stale | ⚠️ PARTIALLY WIRED / NEW DEFECTS | The literal acceptance criteria (grep gates for `hasCropInterpolationBasis(cumulativeRoute)`, `croppedGPX = null` count >= 5, `const confirmedCrop`, no local `getCoordinateAtDistance`) all pass — but the guard's own mitigation introduces CR-01 (form-data zeroing) and CR-02 (marker stranding), both confirmed above. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `+page.svelte updateCropMarkers()` | `crop.ts getCoordinateAtDistance`/`hasCropInterpolationBasis` | named import | WIRED | `grep -c "from \"$lib/models/gpx/crop\""` = 1; local duplicate deleted. |
| `+page.svelte updateCropMarkers()` guard | `+page.svelte toggleCropMarkers()` | direct call on the degenerate path | WIRED, but the callee's side effects are wrong | This is the wiring that produces CR-01 and CR-02: the guard correctly calls `toggleCropMarkers(false)`, but that function's own body (unconditional `updateTotals`) and the caller's later unconditional re-invocation with `true` (from `route_editor.svelte`) are both defective. |
| `gpx-metrics-computation.ts` `totalElevationGainSmoothed`/`Loss` | `trail_anchor_list.svelte` `snapshotMetrics`/`subtractMetrics` | direct field reads, two consecutive snapshots subtracted | WIRED, but the source's contract (monotonic) is violated | See CR-03 above. |
| `+page.svelte confirmCrop()` | `initRouteAnchors` | hoisted `confirmedCrop` local | WIRED | `grep -c "const confirmedCrop"` = 1, `grep -c "initRouteAnchors(confirmedCrop"` = 1, `grep -c "setRoute(croppedGPX"` = 0. Reading `croppedGPX` after the `updateTrailWithRouteData()` choke point would have broken this; the hoist correctly avoids it. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `trail.elevation_gain`/`elevation_loss` (whole-route persistence) | `totals.elevationGain/Loss` | `GPX.getTotals()` -> `metrics.totalElevationGain/LossSmoothed` | Correct for stationary/noisy input now (0/0 or the genuine net displacement) | ✓ FLOWING — the original CR-02 fabrication defect is closed. |
| `$formData.distance/duration/elevation_gain/elevation_loss` (trail-edit form, on crop-panel guard fire) | Zero-valued `valhallaStore.route.features` | `updateCropMarkers()` guard -> `toggleCropMarkers(false)` -> `updateTotals()` | Overwrites real (possibly hand-entered) data with zeros | ✗ HOLLOW/DESTRUCTIVE — see CR-01. |
| `trail_anchor_list.svelte` per-segment `elevationGain`/`elevationLoss` display | `subtractMetrics(cumulative, previous)` | Two `GpxMetricsComputation` snapshots | Can be negative for genuinely-climbed terrain | ✗ INCORRECT for the class of routes where a retraction straddles a segment boundary — see CR-03. |

### Behavioral Spot-Checks (executed, not inferred)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full unit suite green | `cd web && npx vitest run` | `Test Files 6 passed (6)` / `Tests 44 passed (44)` | ✓ PASS |
| CONV-04 closure: gpx-metrics-computation + crop suites in isolation | `cd web && npx vitest run src/lib/models/gpx/gpx-metrics-computation.test.ts src/lib/models/gpx/crop.test.ts` | `Test Files 2 passed (2)` / `Tests 22 passed (22)` | ✓ PASS |
| CR-03 repro: commit +6, commit +6 (snapshot A = 12), retract (snapshot B = 6) | ad-hoc Vitest probe against the shipped `GpxMetricsComputation` class (built, run, deleted) | `segmentGain (trail_anchor_list.svelte style subtraction): -6` | ✗ FAIL (confirms new blocker) |
| CR-01/CR-02 repro: `RouteEditor` render condition, `toggleCropMarkers`/`updateTotals`/`togglePanels`/noUiSlider `bindEvent` call chain | `grep`/`Read` across `+page.svelte`, `route_editor.svelte`, `double_slider.svelte`, `node_modules/nouislider/dist/nouislider.js` | Guard reachable with no drawn route (`drawingActive`-gated only); `updateTotals(valhallaStore.route)` unconditional in the inactive branch; noUiSlider fires `update` synchronously on bind (source comment: "fire it immediately for all handles") | ✗ FAIL (confirms both new blockers) |
| Commits referenced in 33-04-SUMMARY.md / 33-05-SUMMARY.md exist and match claimed content | `git log --oneline` | All 6 commits present (`15e56570`, `f981ce15`, `3860706e`, `ef698579`, `f1108d71`, plus the two `docs(33-0x)` commits) | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| CONV-01 | 33-01 | First track point included in distance/bbox/centroid | ✓ SATISFIED | Unaffected by this round; re-confirmed passing in the full suite. |
| CONV-02 | 33-01 | Centroid divides by the same point count it summed | ✓ SATISFIED | Unaffected by this round; re-confirmed passing. |
| CONV-03 | 33-02 | Points with no elevation excluded from gain/loss | ✓ SATISFIED | Unaffected by this round; re-confirmed passing. |
| CONV-04 | 33-02, 33-04 | Elevation gain/loss sampled independently of the horizontal threshold, without fabricating on noise | ⚠️ PARTIALLY SATISFIED / NEW REGRESSION | The literal defect (fabricated 210/203 on stationary noise) is genuinely closed. But the fix makes the underlying totals non-monotonic, breaking a downstream consumer (`trail_anchor_list.svelte`) that was correct before this phase touched the file — CR-03. REQUIREMENTS.md marks this `[x]` Complete; that checkbox is not fully supported by this evidence. |
| CONV-05 | 33-03, 33-05 | Distance from smoothed accumulator; crop-slider consumer of the repaired `cumulativeDistance` array works | ⚠️ PARTIALLY SATISFIED / NEW REGRESSIONS | The distance-source swap and the original NaN/staleness defects are genuinely closed. But the closure work introduced CR-01 (silent form-data zeroing) and CR-02 (stranded pins), both squarely inside "the route-crop feature still works." REQUIREMENTS.md marks this `[x]` Complete; not fully supported by this evidence either. |

No orphaned requirements: all five REQUIREMENTS.md IDs mapped to Phase 33 (CONV-01..05) appear in a plan's `requirements` frontmatter (33-01: CONV-01/02; 33-02: CONV-03/04; 33-03: CONV-05; 33-04: CONV-04 gap closure; 33-05: CONV-05 gap closure).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `web/src/routes/trail/edit/[id]/+page.svelte` | 1428-1439, 1483-1492 | `toggleCropMarkers(false)`'s unconditional `updateTotals(valhallaStore.route)` reached from the new degenerate-route guard | 🛑 Blocker (CR-01, new this round) | Silently zeroes hand-entered distance/duration/elevation on Save. |
| `web/src/routes/trail/edit/[id]/+page.svelte` + `route_editor.svelte` | 1428-1439; 125-131 | `togglePanels()` calls `onCropToggle(true)` unconditionally after `tick()`, undoing the guard's `toggleCropMarkers(false)` | 🛑 Blocker (CR-02, new this round) | Crop pins visibly stranded at [0,0] — the exact defect the new guard comment claims to prevent. |
| `web/src/lib/models/gpx/gpx-metrics-computation.ts` | 161-170 | Retraction decreases previously-committed totals with no monotonicity contract for snapshot-based consumers | 🛑 Blocker (CR-03, new this round) | `trail_anchor_list.svelte` can display negative per-segment elevation gain/loss. |
| `web/src/routes/trail/edit/[id]/+page.svelte` | 1511-1517, 269-306 | Crop preview totals (`updateTotals(croppedGPX)`) are written into `$formData` while `valhallaStore.route` (uncropped) is what gets saved on Submit if the user clicks Save without confirming the crop | ⚠️ Warning (CR-04, pre-existing per code review — not introduced by 33-04/33-05, confirmed by reading the same code paths) | Save-without-confirm persists mismatched cropped totals against the uncropped GPX. Not counted as a new regression of this round, but directly relevant to "reports correct distance/elevation/duration" and worth a follow-up gap-closure plan. |
| `web/src/routes/trail/edit/[id]/+page.svelte` | 440-544 | `handleFileSelection()` is the one route-replacing path that does not itself clear `croppedGPX` | ⚠️ Warning (WR-12, carried from code review) | Currently defence-in-depth only (the normal replace flow clears it via `replaceRoute()` first); one refactor away from live. |
| `web/src/lib/models/gpx/gpx-metrics-computation.ts` | 118-129 | Retraction is single-level (only the immediately preceding commit is retractable) and gated on a fixed 5 m horizontal threshold from the elevation anchor, not time/sample-count | ⚠️ Warning (WR-01/WR-02, carried from code review, independently plausible from reading the algorithm) | Multi-step stationary drift is not fully filtered; a genuine down-climb with the same low-horizontal profile CONV-04 declared real can be partially erased depending on sample spacing. Not independently re-verified with a probe in this pass; matches the code read. |
| `web/src/routes/trail/edit/[id]/+page.svelte` | 2240-2244 | The `elevation_loss` hidden input is named and valued `elevation_gain` (read-only form mode) | ⚠️ Warning, confirmed by direct read | Not introduced by 33-04/33-05; pre-existing in the file this phase repeatedly touches. |
| No TBD/FIXME/XXX markers | — | — | — | `grep` across all modified/created files for this round: none found. |

No debt-marker blocker triggered (Step 7 gate) — clean.

### Human Verification Required

See `human_verification` in frontmatter. All three items have conclusive code-level or executed-probe evidence already (this is not open-ended exploration) — they are listed to confirm the exact user-facing symptom in a live browser/app session.

### Gaps Summary

The prior round's 3 BLOCKER gaps are genuinely closed: the stationary-noise elevation
fabrication (CR-02 old), the crop-panel NaN crash (CR-01 old), and the `croppedGPX` staleness
data-loss path (CR-03 old) are all fixed and independently re-confirmed against the shipped
code in this pass (not accepted on SUMMARY.md's word).

However, the two gap-closure plans introduced 3 new BLOCKER-level defects in the same
subsystem, all independently confirmed in this pass (one via an executed, disposable Vitest
probe against the shipped `GpxMetricsComputation` class; two via reading the actual call chain
across `+page.svelte`, `route_editor.svelte`, `double_slider.svelte`, and — to settle the
timing question definitively — noUiSlider's own `bindEvent()` source):

1. **CR-01 (new):** The degenerate-route crop guard's own fix (`toggleCropMarkers(false)`)
   silently zeroes hand-entered distance/duration/elevation form fields and persists the zeros
   on Save — a new, more severe form of data loss than the one 33-05 set out to close, and a
   direct contradiction of 33-05's own stated purpose ("no silent data loss of a user's route").
2. **CR-02 (new):** The same guard's marker-hiding is unconditionally undone one tick later by
   `route_editor.svelte`'s `togglePanels()`, so crop pins are still stranded visibly at [0,0] on
   a degenerate route — the exact WR-05 defect the guard's new comment claims to have fixed.
3. **CR-03 (new):** 33-04's commit-then-retract elevation filter makes the smoothed elevation
   totals non-monotonic. `trail_anchor_list.svelte`'s per-segment display, which subtracts
   consecutive snapshots of those totals, can now render a negative elevation gain or loss — a
   direct violation of the phase goal's "reports correct... elevation" for a class of routes
   this phase's own changes made possible (they were not affected before 33-04).

Net effect: 3 known defects were traded for 3 new ones in the same two files this phase set out
to harden. The phase goal — "every GPX converted anywhere in Wanderer... reports correct
distance, elevation, and duration" — is not yet achieved. CR-01 in particular is a regression on
the load-bearing safety property (no data loss) that both this phase and its own gap-closure
plan explicitly promised.

**This does not look like an acceptable intentional deviation** — no override is suggested. All
three are regressions introduced by this phase's own gap-closure work (not pre-existing, not
out-of-scope), sit inside files this phase already owns and was actively fixing, and are
classified BLOCKER by both the independent code review and this independent verification pass.

A fourth finding from the code review (CR-04: saving with a pending unconfirmed crop preview
persists mismatched cropped totals against the uncropped GPX) was independently confirmed by
reading the same call chain, but is genuinely pre-existing — not introduced by 33-04 or 33-05 —
so it is listed under Anti-Patterns as a Warning rather than a new gap in this frontmatter. It
remains directly relevant to the phase's correctness goal and is a strong candidate for the next
gap-closure plan alongside CR-01/CR-02/CR-03.

---

_Verified: 2026-07-31T13:10:00Z_
_Verifier: Claude (gsd-verifier)_
