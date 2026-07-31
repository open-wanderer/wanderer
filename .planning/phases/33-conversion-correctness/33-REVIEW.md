---
phase: 33-conversion-correctness
reviewed: 2026-07-31T12:55:00Z
depth: standard
files_reviewed: 7
files_reviewed_list:
  - web/src/lib/models/gpx/crop.ts
  - web/src/lib/models/gpx/crop.test.ts
  - web/src/lib/models/gpx/gpx-metrics-computation.ts
  - web/src/lib/models/gpx/gpx-metrics-computation.test.ts
  - web/src/lib/models/gpx/gpx.ts
  - web/src/lib/models/gpx/gpx.test.ts
  - web/src/routes/trail/edit/[id]/+page.svelte
findings:
  critical: 4
  warning: 12
  info: 7
  total: 23
status: issues_found
---

# Phase 33: Code Review Report

**Reviewed:** 2026-07-31T12:55:00Z
**Depth:** standard
**Files Reviewed:** 7
**Status:** issues_found

## Summary

This round re-reviews phase 33 after gap-closure plans 33-04 (commit-then-retract elevation
noise filter) and 33-05 (degenerate-safe crop interpolation extracted to `crop.ts` +
`croppedGPX` lifecycle fixes). All 34 tests in `web/src/lib/models/gpx/` pass
(`npx vitest run src/lib/models/gpx/`).

What genuinely landed:

- `getCoordinateAtDistance()` is now degenerate-safe (`span > 0`), extracted, and covered by
  tests. The `[NaN, NaN, 1]` → `Invalid LngLat object` crash from the prior round's CR-01 is
  closed.
- `croppedGPX` is now cleared in `resetTrail()`, `replaceRoute()`, `toggleCropMarkers(false)`
  and `updateTrailWithRouteData()`, and `confirmCrop()` hoists it locally. The prior round's
  CR-03 (stale crop restoring a discarded route) is closed on every path I traced except
  `handleFileSelection()` (WR-12).
- The stationary ±7 m oscillation case (prior CR-02) now reports 0/0 for the single-swing
  fixtures the tests use.

What is not closed, and what the gap-closure work newly broke:

1. The 33-05 guard's *mitigation* (`toggleCropMarkers(false)`) is undone one tick later by
   `route_editor.svelte`, so the prior round's WR-05 (crop pins stranded at 0°N 0°E) is still
   live on the exact path the new comment claims to fix (CR-02).
2. That same mitigation calls `updateTotals(valhallaStore.route)`, which overwrites
   user-entered distance/duration/elevation with the empty route's zeros — a new data-loss
   path introduced by 33-05 (CR-01).
3. 33-04 made `totalElevationGainSmoothed` / `totalElevationLossSmoothed` **non-monotonic**.
   An existing consumer (`trail_anchor_list.svelte`) derives per-segment metrics by
   subtracting consecutive snapshots of those fields and can now render negative elevation
   gain (CR-03).
4. The retract filter is single-level and horizontal-distance gated, so it neither removes
   multi-step stationary drift (WR-01) nor preserves genuine elevation on the very
   low-horizontal-movement geometry CONV-04 declared real (WR-02) — the in-file comment
   "a genuine climb is never under-reported" is provably false.

Findings below were verified by reading the call chain and, for the metrics filter, by
re-running the exact algorithm on hand-built elevation series.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01 (BLOCKER): The new degenerate-route guard silently zeroes user-entered distance / duration / elevation

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1483-1492` (guard), `:1428-1439`
(`toggleCropMarkers`), `:2182-2200` (the editable fields), `:269-306` (submit)

**Issue:** Before 33-05 the degenerate-route branch was a bare `return` — it touched no state.
It now calls `toggleCropMarkers(false)`, whose `else` branch runs
`updateTotals(valhallaStore.route)`. `updateTotals()` (`:1546-1555`) writes `distance`,
`duration`, `elevation_gain`, `elevation_loss` straight into `$formData`.

The guard fires precisely when the route is empty or degenerate, so
`valhallaStore.route.features` is `{distance: 0, duration: 0, elevationGain: 0, elevationLoss: 0}`
— the four fields are overwritten with zeros.

Reachable through plain UI, no route required: `RouteEditor` is rendered whenever
`drawingActive` (`:2413-2432`), not gated on `routeHasTrackPoints()`.

1. New trail → expand "basic info" → type distance / elevation by hand (these are real
   `TextField`s bound to `$formData`, `:2182-2200`).
2. Click "draw route" → click the crop icon.
3. `DoubleSlider` mounts → `onupdate([0, 100])` → `updateCropMarkers` → guard →
   `toggleCropMarkers(false)` → `updateTotals(empty route)` → all four fields become 0.
4. Save. `onSubmit` persists the felte `form` object, so the zeros are written to the DB.

**Fix:** the guard must not mutate form state. Hide the markers directly and leave totals
alone:

```ts
if (!hasCropInterpolationBasis(cumulativeRoute)) {
    croppedGPX = null;
    cropStartMarker?.setOpacity("0");
    cropEndMarker?.setOpacity("0");
    return;
}
```

Separately, `toggleCropMarkers(false)`'s unconditional `updateTotals(valhallaStore.route)`
should only run when a crop preview was actually applied (`if (croppedGPX) { ... }`),
otherwise every crop-panel close also clobbers hand-entered metrics on trails whose stored
numbers were never derived from the GPX.

---

### CR-02 (BLOCKER): The guard's marker-hiding is undone one tick later — pins are still stranded at 0°N 0°E

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1483-1492`;
`web/src/lib/components/trail/route_editor.svelte:124-131`, `:410`;
`web/src/lib/components/base/double_slider.svelte:26-45`

**Issue:** The new comment claims the guard prevents "stranding two visible pins at 0N 0E".
Call ordering makes that false.

`togglePanels()` in `route_editor.svelte` is:

```ts
crop = _crop;
editRoute = _edit;
await tick();      // <- DoubleSlider mounts HERE
onCropToggle(_crop);
```

`DoubleSlider.onMount` calls `noUiSlider.create(...)` then binds an `update` listener, and
noUiSlider fires `update` *immediately on bind*
(`node_modules/nouislider/dist/nouislider.js:1767-1769`, once per handle). So the sequence on
crop-panel open is:

1. markers are constructed and `setLngLat([0, 0]).addTo(map!)` (`:1468-1469`) — **before** the
   guard;
2. guard fires → `toggleCropMarkers(false)` → opacity `"0"`;
3. `tick()` resolves → `onCropToggle(true)` → `toggleCropMarkers(true)` → opacity `"1"`.

Net effect on an empty/degenerate route: two fully visible crop pins at Null Island, exactly
the prior round's WR-05, plus a crop panel whose slider does nothing. The MapLibre `NaN` throw
is fixed; the visual defect the same commit claims to fix is not.

**Fix:** do not rely on a sibling component to set the final visibility. Track whether a crop
basis exists and make `toggleCropMarkers(true)` respect it:

```ts
let cropBasisAvailable = $state(false);

function toggleCropMarkers(active: boolean) {
    if (active && cropBasisAvailable) {
        cropStartMarker?.setOpacity("1");
        cropEndMarker?.setOpacity("1");
        return;
    }
    cropStartMarker?.setOpacity("0");
    cropEndMarker?.setOpacity("0");
    ...
}
```

and set `cropBasisAvailable = hasCropInterpolationBasis(cumulativeRoute)` at the top of
`updateCropMarkers()`. Better still: only construct/add the markers *after* the guard passes,
so nothing is ever added to the map at `[0, 0]`.

---

### CR-03 (BLOCKER): 33-04 made the smoothed elevation totals non-monotonic; an existing consumer subtracts them and can now show negative gain

**File:** `web/src/lib/models/gpx/gpx-metrics-computation.ts:161-170` (retraction);
consumer: `web/src/lib/components/trail/trail_anchor_list.svelte:219-250`

**Issue:** Before 33-04, `totalElevationGainSmoothed` and `totalElevationLossSmoothed` were
accumulate-only. The retraction branch now *decreases* them:

```ts
if (this.retractableDelta > 0) this.totalElevationGainSmoothed -= this.retractableDelta;
else this.totalElevationLossSmoothed += this.retractableDelta;  // delta < 0
```

`trail_anchor_list.svelte` computes per-segment metrics by snapshotting the running instance
at each segment boundary and subtracting the previous snapshot:

```ts
elevationGain: metrics.elevationGain - previous.elevationGain,
```

That subtraction is only valid while the totals are non-decreasing. If a retraction happens on
the first committed point of segment *N* (retracting a commit made at the end of segment
*N-1*), that segment's rendered gain is negative. Planner segments share a duplicated anchor
point (identical coordinates → `retractDistance = 0 < thresholdXY_m`), so the
horizontal-stillness half of the retraction condition is trivially satisfied at every segment
boundary; only the ±5 m elevation return is needed.

Verified by replaying the algorithm: on `[1000, 1006, 1012, 1006, 1000]` the smoothed gain
sequence is `0 → 6 → 12 → 6`; any segment boundary between samples 3 and 4 yields a negative
per-segment gain.

The class comment at `:126-129` only argues the *totals* cannot go negative — it never states
(and the code no longer honours) the monotonicity contract the consumer depends on.

**Fix:** either (a) make retraction non-observable by consumers — buffer the pending commit
and only fold it into the public totals once it can no longer be retracted; or (b) expose an
explicit clamped snapshot and clamp in the consumer:

```ts
elevationGain: Math.max(0, metrics.elevationGain - previous.elevationGain),
```

Option (a) is preferable because (b) silently loses the retracted amount from the segment
where it was earned. Whichever is chosen, add a test asserting monotonic non-decrease (or the
documented replacement contract) — no current test covers a retraction across a snapshot.

---

### CR-04 (BLOCKER): Saving while a crop preview is pending persists cropped metrics with the uncropped GPX

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1511-1517`, `:269-306`

**Issue:** `updateCropMarkers()` writes the *preview* crop's totals into `$formData`
(`updateTotals(croppedGPX)`, `:1517`) while `valhallaStore.route` still holds the full route.
The Save button lives in the always-visible form (`:2404-2410`) and `onSubmit` writes
`form.expand!.gpx_data = valhallaStore.route.toString()` (`:298`) — the **uncropped** track —
while `form.distance` / `elevation_gain` / `elevation_loss` / `duration` are the cropped
preview values.

Repro: open crop, drag the slider to ~50 %, click Save without pressing Crop. The saved trail
reports half the distance/elevation of the GPX it stores. Nothing rolls the preview back on
submit; the discrepancy is silent and persisted (it also flows to Meilisearch and the stats
page, which sum `elevation_gain`).

This is pre-existing (not introduced by 33-05), but it sits squarely in the `croppedGPX`
lifecycle 33-05 set out to make correct, and it is the highest-impact remaining hole in it.

**Fix:** keep the crop preview out of `$formData` — render preview totals from separate state
used only by the UI — or discard the preview on submit:

```ts
onSubmit: async (form) => {
    if (croppedGPX) {                       // unconfirmed preview -> discard
        croppedGPX = null;
        updateTotals(valhallaStore.route);
        Object.assign(form, {
            distance: $formData.distance,
            duration: $formData.duration,
            elevation_gain: $formData.elevation_gain,
            elevation_loss: $formData.elevation_loss,
        });
    }
    ...
```

---

## Warnings

### WR-01: The retract filter is single-level — multi-step stationary drift still ratchets gain

**File:** `web/src/lib/models/gpx/gpx-metrics-computation.ts:118-182`

**Issue:** Only the *immediately preceding* commit is retractable (`retractableDelta` is
overwritten by each new commit and zeroed after a retraction). A stationary device whose
altimeter drifts in more than one step before returning is not filtered.

Verified by replaying the algorithm (identical lat/lon throughout, thresholds 5/5):

| elevation series (stationary) | reported gain | reported loss | truth |
|---|---|---|---|
| `1000, 1006, 1012, 1006, 1000` | 6 | 6 | 0 / 0 |
| `1000, 1006, 1012, 1018, 1012, 1006, 1000` | 12 | 12 | 0 / 0 |
| `1000, 1007, 1000, 1007, …` (61 samples, in tests) | 0 | 0 | 0 / 0 |

The test suite only exercises the single-swing shape, so this gap is invisible to CI. Real
altimeter noise on a paused device drifts in steps at least as often as it alternates.

Related, lower-impact: a retraction resets `lastFilteredZ` to `preRetractZ` rather than the
observed `elevation`, so up to `thresholdZ_m` of real elevation change is absorbed per
retraction (self-limiting — a replay of ten drifting cycles reported 10.5 m against a true
9 m — but it is a documented-nowhere bias).

**Fix:** keep a bounded stack (or a running excursion window anchored at the pre-excursion
elevation) so an excursion that returns to the anchor after *k* commits retracts all *k*:
track `excursionStartZ` + `committedSinceExcursionStart` and unwind the whole run when
`|elevation - excursionStartZ| < thresholdZ_m` and horizontal movement is below
`thresholdXY_m`. Add fixtures for the two rows above.

---

### WR-02: The comment's core claim is false — a genuine climb *is* under-reported in the CONV-04 regime

**File:** `web/src/lib/models/gpx/gpx-metrics-computation.ts:118-129`

**Issue:** The comment asserts "a genuine climb is never under-reported" and that horizontal
stillness distinguishes noise from terrain. Both fail in exactly the geometry CONV-04 declared
genuine (88 m of climb over 4.4 m of horizontal movement is an accepted fixture at
`gpx-metrics-computation.test.ts:88-116`).

Replayed results:

| series | horizontal spacing | reported gain / loss |
|---|---|---|
| `1000, 1010, 1000` | ~2 m per sample | **0 / 0** (all real elevation erased) |
| `1000, 1008, 1000, 1008, 1000, 1008` | ~4 m per sample | 8 / 0 |
| `1000, 1008, 1000, 1008, 1000, 1008` | ~100 m per sample | 24 / 16 |

The same elevation profile reports three times the gain purely because of sample spacing.
Anyone who climbs and down-climbs a steep step (via ferrata, tower, scramble to a viewpoint)
loses the whole excursion, and densely-sampled tracks are penalised relative to sparse ones.

**Fix:** at minimum correct the comment so it documents the real trade-off. Better: gate
retraction on elapsed *time* / sample count as well as horizontal distance (noise on a paused
device returns within seconds; a down-climb does not), or require the excursion to have zero
net horizontal progress rather than "less than 5 m from the last commit point".

---

### WR-03: Crop start is off by one point — the start pin is drawn at a point the crop excludes

**File:** `web/src/lib/models/gpx/crop.ts:44`, `:66`;
`web/src/routes/trail/edit/[id]/+page.svelte:1495-1515`

**Issue:** Carried over unfixed from the prior round (WR-04) and now baked into the extracted
API. `getCoordinateAtDistance()` returns the *later* endpoint of the bracketing segment
(`i = Math.max(1, …)`), so for `target = 0` it returns coordinate = `points[0]` but
`index = 1`. `updateCropMarkers` uses the coordinate for the pin and the index for
`cropGPX(flatRoute[startIndex], …)`.

Consequence: opening the crop panel and pressing Crop without moving the slider (range
`[0, 100]`) silently deletes the route's first track point — the pin sits on a point that is
not in the result, the distance shrinks by the first hop, and `$formData.lat/lon` (derived
from the first `trkpt` at `:914-923` / `:308-320`) moves.

**Fix:** return both bracket indices, or have the caller floor the start:

```ts
const startPointIndex = targetStartDistance <= cumulativeRoute[startIndex - 1]
    ? startIndex - 1
    : startIndex;
croppedGPX = cropGPX(flatRoute[startPointIndex], flatRoute[endIndex], valhallaStore.route);
```

and add a `crop.test.ts` case asserting that the index returned for `target = 0` maps to the
point whose coordinate is returned.

---

### WR-04: `startIndex === endIndex` makes `cropGPX` crop to the end of the track

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1511-1515`;
`web/src/lib/util/gpx_util.ts:379-403`

**Issue:** Nothing validates that the two interpolated indices differ. When both slider
handles land in the same segment (easy on a coarse route — noUiSlider is continuous, the index
space is not), `startIndex === endIndex`. In `cropGPX`, the branch that matches `start` does
`newPoints.push(pt); continue;` — the `pt === end` test is never evaluated for that same
point, so `done` stays `false` and **every remaining point of the track is appended**.

Result: the previewed totals (and the confirmed route, since `confirmCrop` trusts
`croppedGPX`) describe "from here to the end", while the two pins sit millimetres apart. The
user sees a tiny selection and gets most of the route.

**Fix:** guard in the caller and fail closed:

```ts
if (endIndex <= startIndex) {
    croppedGPX = null;
    updateTotals(valhallaStore.route);
    return;
}
```

and, separately, make `cropGPX` terminate immediately when `start === end`.

---

### WR-05: `updateCropMarkers` assumes `features.cumulativeDistance` is index-aligned with `flatten()`, and nothing enforces it

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1473-1506`

**Issue:** `flatRoute` is read live from `valhallaStore.route.flatten()`, while
`cumulativeRoute` is read from the cached `features` object. The alignment invariant holds
only if every mutation recomputes `features`. `deleteFromRoute()`
(`web/src/lib/stores/valhalla_store.svelte.ts:148-159`) assigns
`snapshot.features = valhallaStore.route.getTotals()` — totals computed from the route
*before* the splice — so after deleting a segment the cached `cumulativeDistance` is longer
than `flatten()` and `rawRouteTotal` is inflated.

`crop.ts`'s `Math.min(low, points.length - 1)` clamp keeps this from crashing (good), but the
pins are then placed at the wrong percentage of the route and the crop indices are wrong — a
silent wrong-data failure rather than a loud one.

**Fix:** cheap consistency check in the caller (and fix `deleteFromRoute` to compute totals
from the spliced snapshot):

```ts
if (cumulativeRoute.length !== flatRoute.length || !hasCropInterpolationBasis(cumulativeRoute)) {
    croppedGPX = null;
    ...
    return;
}
```

---

### WR-06: `getCoordinateAtDistance` uses a valid coordinate as its error sentinel, and its "never non-finite" doc is not enforced

**File:** `web/src/lib/models/gpx/crop.ts:25-31`, `:10-24`

**Issue:** Two problems in the extracted API:

1. `points.length === 0` returns `[0, 0, 0]` — a *legal* LngLat in the Gulf of Guinea. A
   caller that forgets the `hasCropInterpolationBasis` guard gets a plausible-looking pin
   instead of an error. That is the same Null Island failure mode the file was created to
   prevent, moved one layer down.
2. The doc block promises the function "never returns a non-finite number", but `target` is
   never validated: `getCoordinateAtDistance(points, cumulative, NaN)` returns `[NaN, NaN, i]`
   because `ratio = (NaN - prevDist) / span`. The promise is a comment, not an invariant.

**Fix:** return `null` for the empty case and force callers to handle it, and enforce the
documented invariant:

```ts
export function getCoordinateAtDistance(...): [number, number, number] | null {
  if (points.length === 0 || !Number.isFinite(target)) return null;
  ...
```

Add `crop.test.ts` cases for `target = NaN` / `target = Infinity` — today nothing covers them.

---

### WR-07: After the guard fires (or after any route edit) the crop UI stays open and the Crop button silently does nothing

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1483-1492`, `:1520-1532`, `:1534-1535`;
`web/src/lib/components/trail/route_editor.svelte:410-418`

**Issue:** Two silent no-op paths:

1. Guard path: `updateCropMarkers` hides the markers but `route_editor`'s own `crop` boolean
   stays `true`, so the panel remains open with a live slider that does nothing. Pressing
   "crop" calls `confirmCrop()` → `croppedGPX` is `null` → `return` with no feedback.
2. `updateTrailWithRouteData()` now sets `croppedGPX = null` (`:1535`). Any route edit made
   while the crop panel is open (undo/redo, anchor drag, split, recalculate elevation) throws
   the pending crop away; the pins stay where they were and the Crop button becomes a silent
   no-op until the slider is touched again.

Nulling the cache is the right call (a stale `croppedGPX` holds object identities from a
replaced route — see the prior round's CR-03), but the user gets no signal.

**Fix:** give `confirmCrop()` a user-visible failure and have the guard path close the panel
through the component (a `bind:crop` or an `onCropUnavailable` callback) rather than only
hiding the markers:

```ts
function confirmCrop() {
    const confirmedCrop = croppedGPX;
    if (!confirmedCrop) {
        show_toast({ type: "error", icon: "close", text: $_("crop-unavailable") });
        return;
    }
    ...
```

---

### WR-08: `any` typing throughout `GpxMetricsComputation` defeats strict mode in the file being hardened

**File:** `web/src/lib/models/gpx/gpx-metrics-computation.ts:29-37`, `:56`

**Issue:** `addAndFilter(point: any)`, `lastPointXY: any | null`,
`lastFilteredPointXY: any | null`, `lastFilteredZPointXY: any | null`. `any | null` collapses
to `any`, so `this.lastFilteredZPointXY !== null` is unchecked by the compiler, and a typo
such as `point.$.long` or `point.elevation` compiles cleanly and silently produces
`NaN`/`undefined` at runtime. This is the hottest correctness path in the phase and the only
one with no type safety.

**Fix:** `import type Waypoint from './waypoint'` and type the fields/parameters as
`Waypoint | null` / `Waypoint`. The class only touches `$.lat`, `$.lon` and `ele`, all present
on `Waypoint`.

---

### WR-09: The `elevation_loss` hidden input is named and valued `elevation_gain`

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:2240-2244`

**Issue:** Inside the elevation-*loss* block:

```svelte
<input type="hidden" name="elevation_gain" value={$formData.elevation_gain} />
```

The form emits `elevation_gain` twice and never emits `elevation_loss` in read-only mode.
Felte's `data-felte-keep-on-remove` on the fieldset (`:2179`) plus the store-based `onSubmit`
currently masks the consequence, but any change to how the payload is built (or a
`new FormData(htmlForm)` read — one already exists at `:275`) turns this into a wrong-value
submission.

**Fix:**

```svelte
<input type="hidden" name="elevation_loss" value={$formData.elevation_loss} />
```

---

### WR-10: `correctElevation` blanks trailing elevations when Valhalla returns a short `height` array

**File:** `web/src/lib/models/gpx/gpx.ts:203-224`

**Issue:** Carried over unfixed from the prior round (WR-08). The all-non-finite guard
(`:207-209`) covers total failure, but nothing checks
`heightResponse.height.length === pointCount`. The write loop indexes past the end, so every
point beyond the response length gets `pt.ele = undefined`, destroying elevation that was
already correct. Post-33-02 those points are treated as "no data" by `parseElevation`, so the
elevation profile silently shortens instead of erroring.

**Fix:**

```ts
const pointCount = this.flatten().length;
if (heights.length !== pointCount) {
    throw new APIError(502, "elevation-service-returned-incomplete-data");
}
```

or skip assignment when `heightResponse.height[heightIndex] === undefined`.

---

### WR-11: `GPX.parse`'s `if (error) throw error` is unreachable; malformed uploads throw a `TypeError` instead

**File:** `web/src/lib/models/gpx/gpx.ts:240-253`

**Issue:** Carried over unfixed from the prior round (WR-09). Inside the `parseString`
callback, `error = err` is assigned and then `new GPX({ $: xml.gpx.$, … })` runs
unconditionally. When parsing fails, `xml` is `undefined`, so the callback throws
`Cannot read properties of undefined (reading 'gpx')` before the `if (error) throw error` line
is ever reached. The same happens for well-formed XML with a non-`gpx` root (e.g. a
mis-detected KML). Callers that branch on `gpx instanceof Error` (`gpx_util.ts:26`, `:106`,
`+page.svelte:386`) are dead code — `parse` never returns an `Error`.

**Fix:**

```ts
xml2js.parseString(sanitizedGPX, opts, (err, xml) => {
    if (err) { error = err; return; }
    if (!xml?.gpx) { error = new Error("Not a GPX document"); return; }
    data = new GPX({ ... });
});
if (error) throw error;
```

and delete the dead `instanceof Error` branches at the call sites.

---

### WR-12: `handleFileSelection()` is the one route-replacing path that does not clear `croppedGPX`

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:440-544`

**Issue:** 33-05 cleared the crop cache in `resetTrail`, `replaceRoute`, `toggleCropMarkers`
and `updateTrailWithRouteData`, but `handleFileSelection()` calls `clearRoute()` / `setRoute()`
without going through any of them (it never calls `updateTrailWithRouteData()`). In the normal
"replace route" flow `replaceRoute()` runs first and clears the cache, so this is currently
defence-in-depth rather than a live bug — but it is the same class of stale-cache hazard the
prior round's CR-03 documented, and one refactor away from being live again.

**Fix:** add `croppedGPX = null;` alongside `clearRoute();` at `:455`, or route all
route-replacement paths through a single helper that owns the invalidation.

---

## Info

### IN-01: Redundant `croppedGPX = null` assignments

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1400`, `:1411`, `:1489`

`resetTrail()` and `replaceRoute()` both null the cache and then call
`updateTrailWithRouteData()`, which nulls it again (`:1535`); `:1489` nulls it immediately
before `toggleCropMarkers(false)`, which nulls it again (`:1436`). Harmless, but it obscures
who owns the invariant. Keep ownership in `updateTrailWithRouteData()` / `toggleCropMarkers()`
and drop the duplicates.

---

### IN-02: `crop.ts` is more review history than code

**File:** `web/src/lib/models/gpx/crop.ts:1-19`, `:42-53`, `:69-79`

Roughly half the file is prose about a defect that no longer exists, citing planning artifacts
(`.planning/phases/33-conversion-correctness/33-VERIFICATION.md`, "gap 2 / CR-01") and phrasing
like "the shipped code returns […]". Source comments should describe current behaviour; the
narrative belongs in the phase summary. The same pattern is heavy in
`gpx-metrics-computation.ts` ("D-01: …", "T-33-11") and in both test files.

---

### IN-03: `rawRouteTotal` is derived twice, in two places

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1481`; `web/src/lib/models/gpx/crop.ts:81`

The caller and the predicate each independently read `cumulative[cumulative.length - 1]`. If
the predicate's notion of "total" ever changes (e.g. `Math.max(...)` for a non-monotonic
array), the caller silently keeps the old one. Have `hasCropInterpolationBasis` return the
total (or `null`) and use it.

---

### IN-04: Dead accumulators and a dead hash

**File:** `web/src/lib/models/gpx/gpx-metrics-computation.ts:43-44`, `:101-116`;
`web/src/lib/models/gpx/gpx.ts:161`, `:179-182`

`totalElevationGain` / `totalElevationLoss` (raw) are maintained on every point and read by
nothing in production (`getTotals` uses the smoothed fields; no test asserts them either).
`features.hash` / `generateMinHash()` is likewise unread anywhere in `web/` or `db/` — and
despite the name it is not a MinHash: `hashes.sort().join('').slice(0, 10)` is determined by
the single lexicographically-smallest geohash, so unrelated trails starting in the same cell
collide. Delete both, or wire them up deliberately.

---

### IN-05: The crop range callback fires twice per panel open

**File:** `web/src/lib/components/base/double_slider.svelte:39-42`

noUiSlider fires `update` once per handle when a listener is bound, so `onupdate([0, 100])` —
and therefore `updateCropMarkers()`, `cropGPX()` over the whole track, and `updateTotals()` —
runs twice on every crop-panel open. Bind with a namespace and fire once, or debounce.

---

### IN-06: Commented-out summit-log block

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:491-506`

16 lines of commented-out code in `handleFileSelection()`. Delete it; git has it.

---

### IN-07: Crop markers are never removed, only made transparent

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1442-1470`, `:1428-1439`

The markers are constructed once, `addTo(map!)` (unchecked non-null assertion) and thereafter
only toggled via `setOpacity`. They remain attached to the map for the lifetime of the page,
and `updateCropMarkers()`'s success path never restores opacity — so if the route becomes
valid again while the panel is open (draw anchors with the crop panel showing), the pins stay
invisible even though a crop preview is being computed. Prefer `remove()` / `addTo()`, and set
opacity `"1"` explicitly on the success path.

---

_Reviewed: 2026-07-31T12:55:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
