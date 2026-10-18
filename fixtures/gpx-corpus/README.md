# GPX Corpus

## Purpose

This directory is the single source of truth for the cross-language metrics contract: it pins
the Dart port to the corrected TypeScript GPX→trail metrics computation. Two independent test suites read
it:

- `web/src/lib/models/gpx/gpx-corpus.test.ts` — Vitest, run with CWD `web/`, reads the corpus at
  a relative path resolved from `process.cwd()`.
- `app/test/util/gpx_corpus_test.dart` — `flutter test`, run with CWD `app/`, reads
  the corpus at a relative path via `dart:io`'s `File`.

Neither suite copies, symlinks, or bundles this directory. Both read it directly from disk at
test time via a relative path. There is no `pubspec.yaml` `assets:` entry — the asset-bundle
restriction that governs `rootBundle.load()` inside a *running app* does not apply to
`flutter test`/`dart test`, which run with CWD = the package root and full filesystem access
(verified empirically).

The corpus lives at the repository root — sibling to `app/`, `web/`, `db/`, `docs/` — deliberately
outside both package roots, so that neither language owns it. This amends the web suite's own
precedent (fixtures inline in the TS test file) for exactly this cross-language case: a single
on-disk source of truth is the whole point. A duplicated per-language fixture set
could drift silently while both suites stayed green — the corpus exists specifically to make
that impossible.

## Layout

```
fixtures/gpx-corpus/
├── README.md                       (this file)
├── 01-two-point-segment/
│   ├── input.gpx
│   ├── expected.json
│   └── DERIVATION.md
├── 02-first-point-extremes/
│   └── ... (same three files)
├── ...
├── 10-realistic-track/
│   ├── input.gpx
│   └── expected.json               (no DERIVATION.md — this fixture is seeded, see below)
├── 11-malformed-time/
│   └── ... (same three files)
└── 12-dense-switchback/
    └── ... (same three files)
```

One directory per fixture, named `NN-slug`. Every fixture directory contains `input.gpx` and
`expected.json`. Every **hand-derived** fixture (`derivation: "hand"` in its `expected.json`)
additionally contains a `DERIVATION.md` explaining how every expected value was produced. The
one **seeded** fixture (`10-realistic-track`, `derivation: "seeded"`) has no `DERIVATION.md` —
see "Derivation methods" below.

## `expected.json` schema

Every `expected.json` has exactly this top-level shape:

```jsonc
{
  "id": "01-two-point-segment",       // string, equal to the directory name
  "covers": ["segment-first-point"],  // array of defect slugs this fixture pins
  "derivation": "hand",               // the string "hand" or the string "seeded"
  "notes": "...",                     // free-text explanation
  "metrics": { /* GPX.getTotals()-shaped */ },
  "trail": { /* gpx2trail()-shaped */ }
}
```

`metrics` keys (mirrors `GPX.getTotals()`'s public output, `gpx.ts:97-167`):

| Key | Type | Meaning |
|-----|------|---------|
| `distance` | number | raw (unsmoothed) total distance, metres — the sum of every consecutive-pair haversine hop |
| `elevationGain` | number | `finalElevationGain` — **not** `totalElevationGainSmoothed` |
| `elevationLoss` | number | `finalElevationLoss` — **not** `totalElevationLossSmoothed` |
| `durationMs` | number | last trkpt time minus first trkpt time, milliseconds |
| `pointCount` | number | total `<trkpt>` count across every `<trkseg>` |
| `boundingBox` | object | `{ minLat, maxLat, minLon, maxLon }` |
| `centroid` | object | `{ lat, lon }` — plain mean of every summed point |

`trail` keys (mirrors `gpx2trail()`'s output, `gpx_util.ts:21-90`):

| Key | Type | Meaning |
|-----|------|---------|
| `name` | string | from `metadata.name` / `trk.name` / `rte.name` fallback chain |
| `description` | string or null | from `metadata.desc` |
| `lat`, `lon` | number | first track (or route) point's coordinates |
| `date` | string (`YYYY-MM-DD`) or null | from the first trkpt's time, only if both start and end time exist |
| `distance` | number | same value as `metrics.distance` |
| `elevationGain`, `elevationLoss` | number | same values as `metrics.elevationGain`/`elevationLoss` |
| `duration` | number | `metrics.durationMs / 1000`, i.e. seconds |
| `minLat`, `maxLat`, `minLon`, `maxLon` | number | same as `metrics.boundingBox`, only present when finite |
| `waypoints` | array | one object per `<wpt>`, each `{ lat, lon, name, description, icon }` |

Both `metrics` and `trail` use **camelCase** keys throughout (e.g. `minLat`, not `min_lat`) —
the on-disk schema is language-neutral. Each language's own test helper maps these camelCase
keys onto its own model's naming convention (the TS `Trail` model uses snake_case fields like
`elevation_gain`/`min_lat`; the mapping happens inside the test helper, not in the fixture).

## Explicit exclusions

The corpus asserts **public metrics only**. It deliberately does **not** include:

- `cumulativeDistance` — the raw, index-aligned per-point distance array. Its only consumer is
  the web trail-edit crop slider, which has no Dart/app equivalent. Porting
  it would be dead Dart code.
- `hash` — the MinHash/Geohash track-shape fingerprint. Not part of the port's contract.
- `totalElevationGainSmoothed` / `totalElevationLossSmoothed` — the **monotonic** running
  elevation totals. These exist in the TS class solely for a per-segment-differencing consumer
  (`trail_anchor_list.svelte`, out of scope for this phase) and are deliberately **not** what
  `metrics.elevationGain`/`elevationLoss` assert. The corpus's `elevationGain`/`elevationLoss`
  fields are always the **`finalElevationGain`**/**`finalElevationLoss`** values — the ones that
  include a still-pending, unconfirmed excursion when a track ends mid-climb. Porting the
  monotonic smoothed pair instead of the `final*` pair is the single most likely way to fail this
  corpus (see fixture `04-switchback-scramble`'s `DERIVATION.md` for a worked example: `88` vs
  `80`).

## Tolerances

One authoritative table, encoded exactly once in each language's own shared assertion helper —
never per-fixture, never inline in an individual test body:

| Field(s) | Comparison | Tolerance |
|----------|------------|-----------|
| `distance`, `elevationGain`, `elevationLoss`, `trail.distance`, `trail.elevationGain`, `trail.elevationLoss` | absolute difference | **1e-6** metres |
| `centroid.lat`, `centroid.lon` | absolute difference | **1e-9** degrees |
| `boundingBox.*`, `trail.lat`, `trail.lon`, `trail.minLat`, `trail.maxLat`, `trail.minLon`, `trail.maxLon` | exact | — |
| `durationMs`, `trail.duration`, `pointCount`, `trail.name`, `trail.description`, `trail.date`, waypoint count, every waypoint field | exact | — |

**Empirical justification for 1e-6 m:** `dart:math` and V8's trig functions are IEEE-754
conformant but are not required to agree bit-for-bit. A 5000-point haversine accumulation loop
was run independently in Node/V8 and the Dart VM (see
the empirical float-agreement measurements ("Empirical float-agreement
measurement") and produced **bit-identical output to 17 significant digits**
(`36221.778933403148` in both runtimes). `1e-6` m is therefore not a fudge factor tuned to make
tests pass — it is a safety margin many orders of magnitude wider than any floating-point drift
actually observed between these two platforms. Any test failure at this tolerance is an
algorithmic divergence, not floating-point noise.

## Derivation methods

Every `expected.json` records how its values were produced, in its `derivation` field:

- **`"hand"`** (fixtures `01` through `09`, plus `11`): every expected value was derived **from first
  principles**, independently of the implementation under test, and the derivation is written
  down in that fixture's own `DERIVATION.md`. Distance figures come from a haversine formula
  transcribed fresh in a scratch `node -e` one-liner (`R = 6371000`), never from calling
  `GPX.getTotals()`, `gpx2trail()`, or `GpxMetricsComputation`. Elevation gain/loss figures come
  from hand-tracing the documented defer-then-publish state machine's rules (threshold 5 m,
  discard only on a horizontal-stillness return) against each fixture's own point sequence.
  Bounding box and centroid come from plain arithmetic over the fixture's own coordinates. The
  TypeScript implementation is explicitly **not treated as the oracle** for these nine fixtures —
  doing so would risk baking a surviving TS bug into the corpus as "expected."
- **`"seeded"`** (fixture `10` only): a bulk, realistic-shaped fixture, populated from the
  corrected TS implementation's actual output and then read back and sanity-checked (plausible
  distance for the coordinate span, duration matching the timestamp span exactly, point count
  matching the trkpt count, bounding box matching the coordinate extremes, both waypoints present
  with icons correctly mapped). The sanity check is recorded in that fixture's `notes` field.

## Fixture-to-defect coverage

| Fixture | Covers | What it pins |
|---------|--------|----------------|
| `01-two-point-segment` | segment-first-point | A 2-point segment reports its real hop distance, not 0 |
| `02-first-point-extremes` | segment-first-point, centroid-divisor | Bounding box includes the segment's own first point; centroid divides by the count it summed |
| `03-missing-vs-empty-ele` | missing-vs-empty-elevation | A literal `<ele></ele>` is treated identically to an omitted tag, never coerced to sea level 0 — the parser landmine canary |
| `04-switchback-scramble` | elevation-noise-filter | Elevation gain is sampled independently of the horizontal-movement threshold; the corpus asserts `finalElevationGain` (88), not `totalElevationGainSmoothed` (80) |
| `05-stationary-noise-returns` | elevation-noise-filter | A fully-stationary altitude oscillation returning to its start elevation reports 0/0, not a ratcheted total |
| `06-stationary-ends-mid-swing` | elevation-noise-filter, final-vs-running-elevation | A track ending mid-excursion reports the genuine un-cancelled net displacement |
| `07-rolling-terrain` | elevation-noise-filter | Noise rejection never eats real terrain when genuine horizontal movement accompanies every swing |
| `08-jittery-track` | raw-vs-smoothed-distance | Reported distance is the raw accumulator; the smoothed accumulator, ~9% smaller, is recorded as the counterfactual |
| `09-multi-segment-planner-route` | segment-first-point | No per-segment metrics-anchor reset — a multi-leg planner route measures continuously through its shared anchor points |
| `10-realistic-track` | end-to-end-pipeline | A plausible, realistic multi-point hike with metadata, waypoints, and timestamps exercises the whole pipeline end to end |
| `11-malformed-time` | unparseable-time | A non-empty but unparseable `<time>` body is "no time" in BOTH languages — the TS side used to build an `Invalid Date`, which is truthy, and so reported a `NaN` duration where Dart reported `0` |
| `12-dense-switchback` | raw-vs-smoothed-distance | Real-watch-density guard (~3.852 m mean hop, 41 points): pins the raw ~154 m total against the retired 5 m gate's ~77 m counterfactual, so re-introducing the gate fails loudly |

## How to add a fixture

1. Author `input.gpx` as a GPX 1.1 document (`version="1.1" creator="wanderer-corpus"
   xmlns="http://www.topografix.com/GPX/1/1"`) with a `<metadata><name>` element (see the rule
   below — never rely on the time-dependent fallback name).
2. Derive every expected value from first principles: haversine formula for distance, hand-traced
   state-machine rules for elevation gain/loss, plain arithmetic for bounding box/centroid. Do
   **not** obtain values by running `GPX.getTotals()`, `gpx2trail()`, or `GpxMetricsComputation`
   (unless the fixture is explicitly a bulk/realistic one, in which case mark it `"seeded"` and
   sanity-check the output before committing it).
3. Write `DERIVATION.md` (for hand-derived fixtures) recording the formula, inputs, and result
   for every field, plus an explicit statement that no expected value came from executing the
   implementation under test.
4. Write `expected.json` following the schema above.
5. Run both suites: `cd web && npx vitest run src/lib/models/gpx/gpx-corpus.test.ts` and (once
   34-04 lands) `cd app && flutter test test/util/gpx_corpus_test.dart`.

## No fixture may rely on the time-dependent name fallback

`gpx2trail()`'s name resolution chain is `metadata.name || trk.name || rte.name || fallbackName
|| \`trail-${new Date().toISOString()}\``. The final fallback is time-dependent and therefore
non-deterministic. Every fixture in this corpus **must** supply either a `<metadata><name>`
element or a `<trk><name>` element, so `trail.name` is a fixed, reproducible value in every
`expected.json`.
