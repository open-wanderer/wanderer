# Phase 33: Conversion Correctness - Pattern Map

**Mapped:** 2026-07-31
**Files analyzed:** 5 (3 modify targets, 1 bounded-edit target, 1 new test class covering 3 files)
**Analogs found:** 5 / 5 (self-analog for modify-in-place files; cross-codebase analog for tests)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|---------------|
| `web/src/lib/models/gpx/gpx.ts` (MODIFY, `getTotals()` loop bound + distance source) | model / transform | batch (parse→accumulate) | itself (in-place fix) — structurally mirrors `web/src/lib/util/gpx_util.ts`'s numeric-guard idiom | exact (self) |
| `web/src/lib/models/gpx/gpx-metrics-computation.ts` (MODIFY, elevation "no data" + threshold gating) | service / transform | streaming (point-by-point accumulator) | itself (in-place fix) | exact (self) |
| `web/src/lib/util/gpx_util.ts` (touch only if `cumulativeDistance` rebuild changes its shape used downstream — confirm no direct references; currently none found) | utility / transform | batch | itself | exact (self) |
| `web/src/routes/trail/edit/[id]/+page.svelte` (bounded edit: `updateCropMarkers()` / `getCoordinateAtDistance()`, ~lines 1438-1537) | component (Svelte, crop-slider logic) | request-response (UI event → recompute) | itself (in-place fix, D-02 rescale) | exact (self) |
| `web/src/lib/models/gpx/gpx.test.ts` (NEW) | test | CRUD/batch (unit, pure function) | `web/src/lib/models/trail.test.ts` | exact — same directory tier (co-located model test), same Vitest idiom |
| `web/src/lib/models/gpx/gpx-metrics-computation.test.ts` (NEW) | test | streaming (unit, pure function) | `web/src/lib/models/trail.test.ts` | exact — same idiom, smaller unit under test |
| `web/src/lib/util/*.gpx` fixtures (NEW, 5+ files per D-04) | fixture (static test data) | file-I/O (read-from-disk in test) | none in-repo — see "No Analog Found" | no analog — recommend inline string fixtures instead (see below) |

## Pattern Assignments

### `web/src/lib/models/gpx/gpx.ts` (model, batch) — MODIFY in place

**This is a defect-repair, not new-file work.** The executor should treat the *existing* file
as its own pattern source. Concrete conventions already present, to preserve:

**Imports pattern** (lines 1-13):
```typescript
import * as xml2js from 'isomorphic-xml2js';
import Metadata from './metadata';
import Route from './route';
import Track from './track';
import { allDatesToISOString, haversineDistance, removeEmpty } from './utils';
import Waypoint from './waypoint';
import GpxMetricsComputation from './gpx-metrics-computation';
//@ts-ignore
import geohash from "ngeohash"
import { encodePolyline } from '$lib/util/polyline_util';
import { APIError } from '$lib/util/api_util';
import type { ValhallaHeightResponse } from '../valhalla';
import { bbox } from '$lib/util/geojson_util';
```
Default-export classes, relative `./` imports for sibling gpx models, `$lib/` alias only for
cross-directory imports. No barrel file.

**Core accumulation pattern to fix** (lines 105-157) — the exact site of D-03/CONV-01/02/05:
```typescript
const metrics = new GpxMetricsComputation(5, 5);
...
const allPoints: Waypoint[] = []
for (const track of this.trk ?? []) {
  for (const segment of track.trkseg ?? []) {
    const points = segment.trkpt ?? [];
    allPoints.push(...points);
    ...
    const pointLength = points.length
    for (let i = 1; i < pointLength; i++) {        // ← D-03: change to i = 0
      const point = points[i];
      metrics.addAndFilter(point)
      totalLat += point.$.lat ?? 0;
      totalLon += point.$.lon ?? 0;
      minLat = Math.min(minLat, point.$.lat ?? Infinity);
      ...
    }
  }
}
totalDistance = metrics.totalDistance;               // ← D-05 (before): should read totalDistanceSmoothed
...
const centroid = { lat: totalLat / allPoints.length, lon: totalLon / allPoints.length };
return {
  ...
  distance: totalDistance,
  cumulativeDistance: metrics.cumulativeDistance,     // ← D-01: rebuild as raw, index-aligned
  ...
};
```
**Null-handling idiom in this file:** `point.$.lat ?? 0` / `?? Infinity` — nullish coalescing
throughout, never `||`. `Number.isFinite(bbox.minLat)` guard pattern (mirrored in
`gpx_util.ts:82`) is the established way to defend against non-finite propagation — reuse this
exact guard shape for any new elevation/distance branch per the RESEARCH.md V5 security note.

**Units convention:** latitude/longitude in decimal degrees (`$.lat`/`$.lon`), distances in
metres (`haversineDistance` returns metres, matches `thresholdXY_m`/`thresholdZ_m` naming
convention — suffix `_m` on metre-denominated fields), duration in milliseconds internally
(`totalDuration`, divided by 1000 only at the `gpx_util.ts:73` trail-assembly boundary).

**Waypoint/TrackSegment/GPXFeature shape** (for constructing fixtures):
- `Waypoint` constructor requires `{ $: { lat, lon }, ele?, time?, ... }` — note `$.lat`/`$.lon`
  default missing/falsy input to `-1` (`waypoint.ts:53-54`), NOT `undefined`. `ele` has no
  fallback — genuinely `undefined` when omitted (`waypoint.ts:55`). Fixture points needing "no
  elevation" must omit `ele` entirely, not pass `ele: undefined` explicitly (both work, but
  omission matches real XML-absence semantics).
- `TrackSegment` constructor: `{ trkpt?: Waypoint[] }` — accepts single or array, always
  normalizes to array internally (`track-segment.ts:8-13`).
- `Track` constructor: `{ trkseg?: TrackSegment[] }`, same array-normalization idiom.
- `GPXFeature` type (lines 25-34) is the return shape of `getTotals()` — `cumulativeDistance:
  number[]` is currently declared as this shape; D-01's rebuilt array keeps the same type.

---

### `web/src/lib/models/gpx/gpx-metrics-computation.ts` (service, streaming) — MODIFY in place

**Core streaming-accumulator pattern** (lines 23-83) — single method `addAndFilter(point)`
called once per point, mutating instance fields (`totalDistance`, `totalElevationGain*`,
`cumulativeDistance`). First-call special case at lines 24-31 initializes anchors and returns
early (no distance/elevation delta on the very first point — Pitfall 2 in RESEARCH.md: this
must NOT be reset per-segment).

```typescript
addAndFilter(point: any) {
  if (!this.lastPointXY || !this.lastFilteredPointXY) {
    this.lastPointXY = point;
    this.lastFilteredPointXY = point;
    this.lastFilteredZ = point.ele ?? 0;   // ← D-03 site: change to undefined-aware guard
    this.lastZ = point.ele ?? 0;           // ← same
    return;
  }
  ...
  const elevation = point.ele ?? 0;        // ← D-03 site
  ...
  if (smoothedDistance < this.thresholdXY_m) {
    return;                                 // ← D-04 site: this early-return also skips elevation-diff eval
  }
  ...
}
```
**Naming convention:** paired raw/smoothed fields (`totalElevationGain` vs
`totalElevationGainSmoothed`, `totalDistance` vs `totalDistanceSmoothed`), `last*` prefix for
accumulator anchor state, `threshold{Axis}_m` for constructor params. Any new field added for
the D-01 raw-array rebuild should follow this pattern (e.g. keep `cumulativeDistance` as the
raw/unsmoothed name, matching its existing per-point-per-call `push` site at line 48).

---

### `web/src/routes/trail/edit/[id]/+page.svelte` — bounded edit (D-02)

**Analog:** itself — `updateCropMarkers()` / `getCoordinateAtDistance()`, lines 1438-1537.

**Current percentage-basis pattern to rescale** (lines 1472-1486):
```typescript
const targetStartDistance =
  valhallaStore.route.features.distance * (start / 100);   // ← D-02: must rescale to
const [startLon, startLat, startIndex] = getCoordinateAtDistance(  //   raw cumulative array's
  flatRoute,                                                        //   own total, not
  valhallaStore.route.features.cumulativeDistance,                  //   features.distance
  targetStartDistance,
);
```
D-02's fix: replace `valhallaStore.route.features.distance` (which becomes the smoothed total
once CONV-05 lands) with the raw cumulative array's own last entry
(`cumulativeDistance[cumulativeDistance.length - 1]`) as the 100% reference for both start and
end target-distance calculations. `getCoordinateAtDistance()` itself (binary search + linear
interpolation, lines 1510-1537) is unaffected — it already consumes the raw array positionally
and needs no change.

---

### `web/src/lib/models/gpx/gpx.test.ts` (NEW test)

**Analog:** `web/src/lib/models/trail.test.ts` — closest and only real Vitest unit-test analog
in the codebase for a co-located model test.

**Full structural pattern to copy** (from `trail.test.ts`, entire file, 154 lines):
```typescript
import { describe, expect, it } from "vitest";
import { SummitLog } from "./summit_log";
import { Trail, ... } from "./trail";
import { Waypoint } from "./waypoint";

describe("Trail.from", () => {
    it("duplicates route and metadata without photos or summit logs", () => {
        const original = originalTrail();
        const duplicate = Trail.from(original, duplicateActor);
        expect(duplicate.name).toBe(original.name);
        ...
    });
    ...
});

function originalTrail() {
    // builder function constructing a realistic domain object via its real constructor,
    // not a mock — Waypoint/SummitLog/Trail are all instantiated for real
    ...
    return original;
}
```
Key idioms to replicate for `gpx.test.ts` / `gpx-metrics-computation.test.ts`:
- `import { describe, expect, it } from "vitest"` — no `vi` needed for pure-function tests
  (only the `server.test.ts` files under `routes/api/` use `vi.fn()` for request/fetch mocks,
  not applicable here).
- Relative imports (`./gpx`, `./waypoint`, `./track`, `./track-segment`,
  `./gpx-metrics-computation`) since the test is co-located in `web/src/lib/models/gpx/`,
  mirroring `trail.test.ts`'s co-location in `web/src/lib/models/`.
- One top-level `describe` per function/behavior under test, `it(...)` sentences in plain
  English describing the assertion, no nested `beforeEach`/`afterEach` — each `it` builds its
  own fixture inline or via a small local builder function (see `originalTrail()` above).
- Builder-function fixtures construct real class instances via their real constructors rather
  than plain object literals or mocks — apply the same idiom to GPX fixtures: build `new GPX({
  trk: [new Track({ trkseg: [new TrackSegment({ trkpt: [new Waypoint({ $: { lat, lon }, ele
  })] })] })] })` inline per RESEARCH.md's own Code Examples section (already drafted there,
  confirmed consistent with this analog).
- Assertions use `toBe`, `toBeCloseTo` (for float comparisons — RESEARCH.md's fixture examples
  already use `toBeCloseTo` for centroid), `toHaveLength`, `toMatchObject`, `toEqual` — no
  custom matchers or snapshot testing anywhere in the codebase.

**Location placement:** co-locate `gpx.test.ts` and `gpx-metrics-computation.test.ts` directly
in `web/src/lib/models/gpx/` next to `gpx.ts` and `gpx-metrics-computation.ts` — matches
`trail.test.ts` sitting beside `trail.ts` in `web/src/lib/models/`. This is the dominant
pattern (2 of 3 existing test files are co-located; the third pair
`server.test.ts`/`+server.ts` co-locates route tests the same way under `routes/api/v1/...`).

---

## Shared Patterns

### Vitest include glob (governs where new tests must live)
**Source:** `web/vite.config.ts:14`
```typescript
test: { include: ['src/**/*.{test,spec}.{js,ts}'] },
```
Any `*.test.ts` anywhere under `web/src/` is picked up automatically — no special directory
requirement beyond staying under `src/`. Co-location with the file under test (per the analog
above) satisfies this trivially.

### Numeric-guard idiom (Number.isFinite)
**Source:** `web/src/lib/util/gpx_util.ts:82`, mirrored need in `gpx.ts`'s `getTotals()`
```typescript
if (Number.isFinite(bbox.minLat) && Number.isFinite(bbox.maxLat)) { ... }
```
**Apply to:** any new branch touching elevation-`undefined` handling (CONV-03) or the rebuilt
`cumulativeDistance` array (D-01), per RESEARCH.md's V5/ASVS note — do not let `NaN`/`Infinity`
reach `trail.distance`/`elevation_gain`/`elevation_loss`.

### Nullish-coalescing, never `||`, for numeric point fields
**Source:** `gpx.ts:129-135` (`point.$.lat ?? 0`, `?? Infinity`)
**Apply to:** the CONV-03 `point.ele ?? 0` → `undefined`-aware rewrite must use `===
undefined` checks (per RESEARCH.md Pitfall 3), not `!point.ele` and not switch to `||`.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `*.gpx` fixture files (e.g. `web/src/lib/models/gpx/fixtures/*.gpx`) | fixture | file-I/O | No existing fixture-loading-from-disk pattern exists anywhere in the repo — no test reads a file via `fs.readFileSync` or an import-as-asset. Both existing non-trivial test files (`trail.test.ts`, the two `server.test.ts` files) build all fixture data as **inline JS/TS objects/builder functions**, never from an external file. **Recommendation:** follow the established inline idiom — construct GPX fixtures as inline `new GPX({...})` object graphs directly in the test file (exactly as RESEARCH.md's own Code Examples section already does), NOT as standalone `.gpx` XML files read from disk. This keeps the new tests consistent with the codebase's only established fixture convention and avoids introducing a first-of-its-kind disk-read pattern for a defect-fix phase. If D-04's "fixture suite" is interpreted literally as requiring `.gpx` text fixtures (e.g. to also exercise `GPX.parse()`'s XML path), a small `web/src/lib/models/gpx/fixtures/*.gpx` directory plus `fs.readFileSync(new URL('./fixtures/x.gpx', import.meta.url))` would be the natural next step, but this is genuinely new territory the planner should call out explicitly rather than assume. |

## Metadata

**Analog search scope:** `web/src/lib/models/gpx/`, `web/src/lib/models/trail.test.ts`,
`web/src/routes/api/v1/regions/[id]/download/server.test.ts`,
`web/src/routes/api/v1/regions/[id]/download-dem/server.test.ts`, `web/vite.config.ts`,
`web/src/routes/trail/edit/[id]/+page.svelte` (targeted read, lines 1420-1560).
**Files scanned:** 11 (gpx.ts, gpx-metrics-computation.ts, gpx_util.ts, waypoint.ts, track.ts,
track-segment.ts, trail.test.ts, both server.test.ts files, vite.config.ts, +page.svelte)
**Pattern extraction date:** 2026-07-31
