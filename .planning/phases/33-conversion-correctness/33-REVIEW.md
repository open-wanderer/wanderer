---
phase: 33-conversion-correctness
reviewed: 2026-07-31T16:40:00Z
depth: standard
files_reviewed: 11
files_reviewed_list:
  - web/src/lib/models/gpx/crop.ts
  - web/src/lib/models/gpx/crop.test.ts
  - web/src/lib/models/gpx/gpx-metrics-computation.ts
  - web/src/lib/models/gpx/gpx-metrics-computation.test.ts
  - web/src/lib/models/gpx/gpx.ts
  - web/src/lib/models/gpx/gpx.test.ts
  - web/src/lib/stores/valhalla_store.svelte.ts
  - web/src/lib/stores/valhalla_store.test.ts
  - web/src/lib/components/trail/route_editor.svelte
  - web/src/lib/components/trail/trail_anchor_list.svelte
  - web/src/routes/trail/edit/[id]/+page.svelte
findings:
  critical: 2
  warning: 11
  info: 0
  total: 13
status: issues_found
---

# Phase 33: Code Review Report (round 3)

**Reviewed:** 2026-07-31T16:40:00Z
**Depth:** standard
**Files Reviewed:** 11
**Status:** issues_found

## Summary

Three fixes were submitted against the previous round's four criticals. I re-derived each
one from the source rather than from the commit messages, and ran targeted throwaway
Vitest fixtures against the real modules to confirm behaviour (scratch tests removed
afterwards; the working tree is clean). All 43 existing tests in the four touched test
files pass.

**Verdict on the three claimed fixes:**

- **CR-03 (monotonic smoothed totals) — holds.** `totalElevationGainSmoothed` /
  `LossSmoothed` are now written only by `publishPending()`
  (`gpx-metrics-computation.ts:87-95`), which is `+=` / `-=`-of-a-negative only. No
  decrement path remains. `finalElevationGain/Loss` correctly surface the pending slot,
  and `gpx.ts:146-147` is the only consumer that reads them. The pending slot is never
  lost: a track that ends mid-swing still reports its last excursion through the getters.
  I re-verified the first-point path (no `<ele>` → both anchors stay `null`), the
  no-elevation-at-all path (gain/loss `0`, no NaN), the sign-flip publish path, and the
  cancel path's anchor rewind.
- **CR-01 / CR-04 (preview must not reach `$formData`) — holds.** `updateTotals()` has
  exactly one call site (`+page.svelte:1597`) and it is fed only by
  `valhallaStore.route`. `cropPreview.gpx` is a class instance, so Svelte 5 does not
  deep-proxy it and `confirmCrop()` gets the real `GPX` back. The "restore" call that
  zeroed hand-entered metrics is gone. I found no path that reaches `$formData` with
  preview data.
- **CR-02 (derived marker visibility) — holds for every ordering I could construct, but
  the mechanism has two independent soft spots** (WR-03, WR-04). The non-reactive plain
  `let` markers are safe *only* because `updateCropMarkers()` always assigns the markers
  strictly before it assigns `cropPreview`, so the effect's re-run always observes
  non-null markers. That coupling is undocumented and one `throw` away from breaking —
  see WR-03.

Also confirmed fixed: `deleteFromRoute` now recomputes from the spliced snapshot
(`cumulativeDistance.length === flatten().length` after a delete, verified), the
`elevation_loss` hidden input carries `$formData.elevation_loss`, and the basis-length
check in `updateCropMarkers` is present and correct.

Beyond re-verification, this round found two crash-on-real-input defects (CR-01, CR-02
below), a metric in the *other* consumer of `GpxMetricsComputation` that directly
contradicts this phase's own CONV-01/CONV-05 fixes, and a cluster of robustness gaps
around the crop feature.

Out of scope per instructions and deliberately not reported: single-level pending-buffer
staircase ratcheting, and the anchor list's one-pending-excursion under-attribution at the
end of a track.

No security issues found — these files carry no injection, auth, or deserialization
surface.

## Critical Issues

### CR-01: `correctElevation()` crashes on a Valhalla response without a `height` array, and the caller swallows it silently

**File:** `web/src/lib/models/gpx/gpx.ts:207-227`, caller `web/src/routes/trail/edit/[id]/+page.svelte:1442-1446`

**Issue:** Line 208 explicitly treats `heightResponse.height` as possibly absent
(`heightResponse.height ?? []`), but the guard on line 211 only fires when
`heights.length > 0`. When `height` is missing entirely, `heights` is `[]`, the guard is
skipped, and line 222 dereferences `heightResponse.height[heightIndex]` on `undefined`.

Reproduced against the real module (`f` stubbed to return `{ ok: true, json: () => ({}) }`):

```
TypeError: Cannot read properties of undefined (reading '0')
 ❯ src/lib/models/gpx/gpx.ts:222:35
```

The caller compounds it: `recalculateElevationData()` is the only route mutation in
`+page.svelte` with no `try/catch` and no `show_toast` — every sibling
(`addAnchorAndRecalculate`, `recalculateRoute`, `handleSegmentDragEnd`, `moveAnchor`)
wraps its await. So both this `TypeError` and the `APIError` thrown on line 204 become
unhandled promise rejections: the user clicks "recalculate elevation data", nothing
happens, and no error is shown.

**Fix:**

```ts
// gpx.ts
const heights = heightResponse.height ?? [];
const finiteHeights = heights.filter((h) => Number.isFinite(h)).length;
if (heights.length === 0 || finiteHeights === 0) {
  return;
}
// ...and read from the local `heights`, never heightResponse.height:
segment.trkpt.forEach((pt) => {
  pt.ele = heights[heightIndex];
  heightIndex++;
});
```

```svelte
// +page.svelte
async function recalculateElevationData() {
    try {
        await recalculateHeight();
        updateTrailWithRouteData();
    } catch (e) {
        console.error(e);
        show_toast({ text: routeCalculationErrorText(e), icon: "close", type: "error" });
    }
}
```

### CR-02: `initRouteAnchors()` dereferences the last point of a segment without the length guard it uses two lines earlier

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:576-596` (unguarded deref at 588-595)

**Issue:**

```ts
if (points.length > 0) {           // guarded
    addAnchor(points[0].$.lat!, ...);
}
if (i == segments.length - 1) {    // NOT guarded
    addAnchor(points[points.length - 1].$.lat!, ...);
}
```

If the *last* `<trkseg>` carries no `<trkpt>`, `points` is `[]`, `points[-1]` is
`undefined`, and `.$.lat` throws. That input is real: `xml2js` with
`explicitArray: false` parses `<trkseg></trkseg>` to the empty string, and `Track`'s
constructor maps it through `new TrackSegment("")` with no `typeof === 'object'` filter
(unlike `GPX`'s own `trk`/`rte`/`wpt` handling), yielding a segment with
`trkpt === undefined`. I confirmed the parse output directly: `segments.length === 2`,
`trkpt` lengths `[2, undefined]`.

Reachability matters here: the guarded first `addAnchor` proves the author knew `points`
can be empty. The `handleFileSelection` call site (line 541) is inside a `try` and would
surface an "error reading file" toast, but the **`onMount` call site (line 421) is not** —
opening any saved trail whose stored GPX ends in an empty `trkseg` throws during mount,
so the route is never set, drawing never starts, and the edit page is unusable.

**Fix:**

```ts
for (let i = 0; i < segments.length; i++) {
    const points = segments[i].trkpt ?? [];
    if (points.length === 0) {
        continue;
    }
    addAnchor(points[0].$.lat!, points[0].$.lon!, valhallaStore.anchors.length, addToMap);
    if (i === segments.length - 1) {
        const last = points[points.length - 1];
        addAnchor(last.$.lat!, last.$.lon!, valhallaStore.anchors.length, addToMap);
    }
}
```

Also worth hardening `Track`'s constructor with the same
`.filter(seg => typeof seg === 'object')` that `GPX` already applies to `trk`/`rte`/`wpt`.

## Warnings

### WR-01: the anchor list reports raw distance while the trail form reports smoothed distance — the segments never sum to the total

**File:** `web/src/lib/components/trail/trail_anchor_list.svelte:219-225`

**Issue:** `snapshotMetrics()` reads `metrics.totalDistance` (the raw haversine sum),
while `gpx.ts:148` reports `metrics.totalDistanceSmoothed` into `features.distance` and
thence into `$formData.distance`. Phase 33-03 deliberately decoupled these two
(`gpx.test.ts` asserts `distance < rawTotal` as an executable invariant), so the anchor
list now displays a different quantity from the form it sits next to. The project's own
jitter fixture puts the gap at ~110 m raw vs ~100 m smoothed — a visible ~10%
disagreement on any GPS-recorded track opened for editing.

**Fix:** use the same basis as the reported total:

```ts
function snapshotMetrics(metrics: GpxMetricsComputation): SegmentMetrics {
    return {
        distance: metrics.totalDistanceSmoothed,
        elevationGain: metrics.totalElevationGainSmoothed,
        elevationLoss: metrics.totalElevationLossSmoothed,
    };
}
```

### WR-02: the anchor list still carries the CONV-01 `i = 1` loop bug that `gpx.ts` fixed

**File:** `web/src/lib/components/trail/trail_anchor_list.svelte:241-251`

**Issue:** `for (let i = 1; i < points.length; i++)` is correct for segments 2..n (their
point 0 duplicates the previous segment's endpoint) but wrong for **segment 1**, whose
point 0 is a genuine route start. This is exactly the defect `gpx.test.ts` pins as
"Pre-fix value was exactly 0 — the loop started at i = 1". Two consequences: the first
segment's opening hop is dropped from its distance, and — because that skipped point
would have been the metrics instance's very first `addAndFilter()` call — the elevation
anchor is established one point late, so any climb across the opening hop is invisible.

Measured on a two-leg fixture (`1000 m → 1020 m`, then `1020 m → 1040 m`):
`gpx.features` reports a 40 m gain over 222 m, while the anchor-list algorithm reports
segment gains of `0` and `0` over a total of 111 m.

**Fix:**

```ts
for (const [segmentIndex, segment] of segments.entries()) {
    const points = segment.trkpt ?? [];
    // segment 0's first point is a real route start; later segments duplicate
    // the previous segment's endpoint, so skip only those.
    for (let i = segmentIndex === 0 ? 0 : 1; i < points.length; i++) {
        metrics.addAndFilter(points[i]);
    }
    ...
}
```

### WR-03: a failed `addTo(map!)` permanently disables the crop pins for the rest of the session

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1520-1548`

**Issue:** The `if (!cropStartMarker || !cropEndMarker)` block does two things: it
constructs the markers *and* adds them to the map. The construction assignments happen
first; `addTo(map!)` is last. If `map` is `undefined` at that moment the assertion
throws — but `cropStartMarker`/`cropEndMarker` are already non-null, so every later call
takes the `else` path, calls `setLngLat`, sets `cropPreview`, and the `$effect` faithfully
sets opacity `"1"` on two markers that were never added to any map. The crop pins are then
silently invisible forever, behind a valid-looking preview panel.

`map` being undefined is not hypothetical: `startDrawing()` (line 912) explicitly guards
`if (!map) return;`, and `drawingActive` is set to `true` *before* that guard, so
`RouteEditor` — and therefore the crop panel — can render while `map` is still unbound.

**Fix:** guard the map, and only latch the fields once the markers are actually attached.

```ts
if (!map) {
    cropPreview = null;
    return;
}
if (!cropStartMarker || !cropEndMarker) {
    const start = new FontawesomeMarker({ /* ... */ }, {});
    const end = new FontawesomeMarker({ /* ... */ }, {});
    start.setOpacity("0").setLngLat([startLon, startLat]).addTo(map);
    end.setOpacity("0").setLngLat([endLon, endLat]).addTo(map);
    cropStartMarker = start;   // only after addTo succeeded
    cropEndMarker = end;
}
```

### WR-04: `setOpacity("0")` does not reliably hide a marker when 3D terrain is enabled

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1468-1472`, `1546-1547`

**Issue:** MapLibre's `Marker.setOpacity(opacity)` sets `_opacity` but leaves
`_opacityWhenCovered` at its `"0.2"` default — verified in
`node_modules/maplibre-gl/dist/maplibre-gl.js`:
`setOpacity(e,t){...void 0!==e&&(this._opacity=String(e))...}`. `_updateOpacity()` then
applies `_opacityWhenCovered`, not `_opacity`, whenever the map has a terrain and the
marker is occluded by it. This map is created with `showTerrain={true}` and adds a
`TerrainLayer` when `settings.terrain.terrain` is set
(`map_with_elevation_maplibre.svelte:1002-1020`), so a "hidden" crop pin sitting behind a
ridge renders at 20% opacity. The markers are also never `.remove()`d and never reset to
`null`, so they stay in the map's DOM (and hit-testable) after the crop panel closes,
after `replaceRoute()`, and after `resetTrail()`.

**Fix:** pass both arguments — `cropStartMarker?.setOpacity(opacity, opacity)` — or,
better, make the `$effect` add/remove rather than fade:

```ts
$effect(() => {
    const visible = cropPanelOpen && cropPreview !== null;
    if (!visible) {
        cropStartMarker?.remove();
        cropEndMarker?.remove();
        return;
    }
    if (map) {
        cropStartMarker?.addTo(map);
        cropEndMarker?.addTo(map);
    }
});
```

### WR-05: cropping at 0% silently drops the route's first point, and the start pin lies about it

**File:** `web/src/lib/models/gpx/crop.ts:42-53`, consumer
`web/src/routes/trail/edit/[id]/+page.svelte:1553-1557`

**Issue:** `const i = Math.max(1, Math.min(low, points.length - 1))` forces the returned
index to at least `1`. For `target === 0` the binary search converges on `low === 0`, so
the returned *coordinate* is correct (`ratio === 0` → `points[0]`'s position) but the
returned *index* is `1`. `updateCropMarkers()` passes that index straight into
`cropGPX(flatRoute[startIndex], ...)`, so the crop actually begins at `points[1]`.

Net effect with the sliders untouched at their `[0, 100]` mount default: the start pin is
drawn on point 0, the preview totals exclude the point 0 → point 1 hop, and confirming
the crop permanently deletes the route's first point. On a Valhalla polyline that hop can
be tens of metres. The mirror case exists at the end — a route with trailing coincident
points resolves 100% to index `n-2`, dropping the final point.

The extracted module preserved the original `Math.max(1, low)` semantics, so this is not a
regression — but the module is now the documented, tested home of this logic and
`crop.test.ts` only asserts `index >= 0 && index < points.length`, which never pins the
behaviour.

**Fix:** return the snapped endpoint rather than always the later one — e.g. use
`ratio === 0 ? i - 1 : i` as the returned index — and add a `crop.test.ts` case asserting
`getCoordinateAtDistance(points, cumulative, 0)[2] === 0`.

### WR-06: the crop panel goes silently inert whenever a route mutation clears the preview

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1494-1502`, `1593-1594`

**Issue:** `cropPreview` is written in exactly one place — `updateCropMarkers()` — and
that runs only from `DoubleSlider`'s `onupdate`, i.e. on mount and on user drag.
`updateTrailWithRouteData()` nulls it on *every* route mutation. So if the crop panel is
open and anything else changes the route (clicking the map adds an anchor while drawing,
undo/redo, recalculate elevation, segment drag), the pins vanish, the totals `dl`
disappears, and the Crop button becomes `disabled` — with no indication why and no
recovery until the user drags a slider handle. The same shape occurs when the panel is
opened before a route exists and one is drawn afterwards.

**Fix:** retain the last requested range and re-derive after invalidation:

```ts
let cropRange: [number, number] | null = null;

function updateCropMarkers(range: [number, number]) {
    cropRange = range;
    // ...
}

$effect(() => {
    valhallaStore.route;              // re-derive when the route changes
    if (cropPanelOpen && cropRange) {
        untrack(() => updateCropMarkers(cropRange!));
    }
});
```

### WR-07: `cropPanelOpen` is not cleared on every path that unmounts `RouteEditor`

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:469-479`

**Issue:** `toggleCropMarkers(false)` is the documented "sole entry point" for panel
state, but `handleFileSelection()` sets `drawingActive = false` directly (line 477),
destroying `RouteEditor` without ever reporting the panel closed. `cropPanelOpen` then
stays `true` while no panel exists. Today this is benign only by accident — nothing else
sets `cropPreview` non-null while the panel is closed, so the `$effect`'s conjunction
still evaluates to `"0"`. That is a coincidence, not an invariant, and it undercuts the
"single source of truth" claim in the comment at lines 1448-1457.

**Fix:** call `toggleCropMarkers(false)` alongside the `drawingActive = false` assignment
in `handleFileSelection()` (and anywhere else `drawingActive` is cleared outside
`stopDrawing()`), or derive `cropPanelOpen` as `drawingActive && panelOpen`.

### WR-08: `insertIntoRoute(waypoints, 0)` appends instead of inserting at the head

**File:** `web/src/lib/stores/valhalla_store.svelte.ts:116-120`

**Issue:** `if (index)` is a truthiness test on a `number | undefined`, so index `0` takes
the `push` branch and the segment lands at the *end* of the track. No current caller
passes `0` (`splitSegment` passes `index + 1`, `handleSegmentDragEnd` passes
`data.segment + 1`, `addAnchorAndRecalculate` passes nothing), so this is latent — but it
is a silent, geometry-corrupting failure the moment anyone inserts a leading leg, and the
resulting bad route is what gets persisted.

**Fix:**

```ts
if (index !== undefined) {
    snapshot.trk?.at(0)?.trkseg?.splice(index, 0, segment);
} else {
    snapshot.trk?.at(0)?.trkseg?.push(segment);
}
```

### WR-09: `setRoute()` is the only mutator that does not refresh `features`, and `updateCropMarkers()` now hard-depends on that cache

**File:** `web/src/lib/stores/valhalla_store.svelte.ts:48-56`

**Issue:** `insertIntoRoute`, `editRoute`, `reverseRoute`, `undo`, `redo`, and
`revertRouteChange` all end with
`valhallaStore.route.features = valhallaStore.route.getTotals()`. `setRoute()` does not —
it relies on `json-diff-ts` faithfully transporting the whole `features` object, including
the `cumulativeDistance` number array, through `diff`/`applyChangeset`. I verified
empirically that it currently does (shrinking 10 → 3 points and 8 → 0 points both leave
`cumulativeDistance.length === flatten().length`), so this is not a live bug.

It is still worth closing, because `updateCropMarkers()` now treats
`cumulativeDistance.length !== flatten().length` as a hard stop
(`+page.svelte:1492-1502`). Any future change to the diff library, or to `GPXFeature`'s
shape, silently disables the entire crop feature with no error surfaced anywhere — the
exact "plausible-looking but wrong" outcome the guard's own comment warns about.
`setRoute()` is on the `confirmCrop()`, `recalculateRouteFromAnchors()`, and `onMount()`
paths, so it is the one that matters most.

**Fix:** make `setRoute()` symmetric with its siblings —
`valhallaStore.route.features = valhallaStore.route.getTotals();` before returning — and
extend `valhalla_store.test.ts` with the same length-alignment assertion for `setRoute`
that it already has for `deleteFromRoute`.

### WR-10: dead `undefined` branches in `CropTotals`, justified by a comment describing behaviour the code cannot produce

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:174-182`, `1562-1573`

**Issue:** The comment claims the optional fields exist so "the format helpers render `-`
for a missing value, which is the honest display for a GPX carrying no elevation data".
That case is unreachable: `GPXFeature.duration` is `Math.abs(totalDuration)` and
`elevationGain`/`elevationLoss` are `finalElevationGain`/`finalElevationLoss` — all always
`number` — and `removeEmpty()` strips only `null`/`undefined`, never `0`. A GPX with no
elevation data therefore renders `0 m`, not `-`. The
`previewGPX.features.duration === undefined ? undefined : .../1000` ternary is a dead
branch, and the comment misdirects the next reader about what `-` means.

**Fix:** either drop the optionals and the ternary, or make `getTotals()` genuinely emit
`undefined` when no point in the track carried elevation — which would also require
auditing `updateTotals()`'s write into `$formData.elevation_gain`.

### WR-11: dead commented-out code and an unused binding in the trail-edit page

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:511-526`, `627`, `635`, `654`, `832`, `897`

**Issue:** The 16-line commented-out `SummitLog` block at 511-526 is actively misleading:
it encodes `duration: $formData.duration * 60`, implying `$formData.duration` is in
minutes, while the live `updateTotals()` (line 1610) writes `totals.duration / 1000` —
seconds — and `formatTimeHHMM()` reads seconds. A future reader restoring that block
would introduce a 60× unit error. Line 627 assigns `const wp = ...splice(...)` and never
uses `wp`. Lines 635 / 654 / 832 / 897 are commented-out `updateTrailOnMap()` /
`lists_index()` calls with no note on whether their removal was deliberate.

**Fix:** delete the commented blocks (git history preserves them) and drop the unused
`wp` binding.

---

_Reviewed: 2026-07-31T16:40:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
