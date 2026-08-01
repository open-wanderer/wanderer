---
phase: 33-conversion-correctness
plan: 04
subsystem: gpx-conversion
tags: [gpx, elevation, noise-filter, gap-closure]
dependency_graph:
  requires:
    - "web/src/lib/models/gpx/gpx-metrics-computation.ts (33-02's decoupled elevation sampling)"
  provides:
    - "Noise-tolerant smoothed elevation filter (commit-then-retract) in GpxMetricsComputation"
    - "Stationary-noise regression fixtures pinning the CONV-04 gap closure"
  affects:
    - "trail.elevation_gain / elevation_loss (PocketBase persistence, via gpx.ts getTotals())"
tech_stack:
  added: []
  patterns:
    - "Commit-then-retract streaming filter: credit an excursion immediately, retract it only when the signal reverses AND the horizontal anchor hasn't moved"
key_files:
  created: []
  modified:
    - "web/src/lib/models/gpx/gpx-metrics-computation.ts"
    - "web/src/lib/models/gpx/gpx-metrics-computation.test.ts"
decisions:
  - "60-sample ends-mid-swing fixture asserts 7/0, not 0/0 -- a causal streaming filter that credits a genuine single-step climb cannot simultaneously discard a track's real net displacement; the 61-sample fixture is the one that delivers the gap's literal 0/0"
  - "Horizontal stillness (haversine < thresholdXY_m) is checked only on the retraction path, never on the commit path -- keeps CONV-04's decoupling intact (a monotonic low-horizontal climb never reaches the retraction branch)"
metrics:
  duration: "~8 min"
  completed: "2026-07-31"
---

# Phase 33 Plan 04: Commit-then-retract elevation noise filter Summary

Replaced the flat `|diff| >= thresholdZ_m` smoothed-elevation commit rule with a
commit-then-retract filter that distinguishes stationary GPS/altimeter noise from
genuine low-horizontal climbs using horizontal stillness as the sole discriminator,
closing VERIFICATION gap 1 (CR-02, BLOCKER).

## What Was Built

**Task 1 (RED):** Added four regression fixtures to
`gpx-metrics-computation.test.ts`, driven through the real XML parse path
(`GPX.parse` -> `gpx.features.elevationGain/Loss`):

- 61-sample fully-stationary +/-7 m altitude oscillation returning to its
  starting elevation — asserts `0`/`0`.
- 60-sample truncation of the same generator, ending mid-swing — asserts `7`/`0`
  (the track's genuine net displacement).
- Stationary out-and-back bump (1000, 1008, 1000, 1008) followed by a genuine
  16 m climb (1016, 1024) — asserts `24`/`0`.
- Rolling terrain with ~100 m horizontal spacing between each elevation sample
  — asserts `24`/`16` (guard: proves noise rejection never eats real terrain
  that came with real horizontal movement).

Observed RED evidence (verbatim from the executed run, before any implementation
change):

```
✗ 61-sample fixture:  expected 0,  received 210  (gain)
✗ 60-sample fixture:  expected 7,  received 210  (gain)
✗ bump+climb fixture: expected 24, received 32   (gain)
✓ rolling-terrain guard fixture: passed unchanged (24/16)
```

These match the pre-fix numbers documented in `33-VERIFICATION.md` exactly
(210/210, 210/203, 32/8).

**Task 2 (GREEN):** Replaced the smoothed elevation block in
`gpx-metrics-computation.ts` with the specified commit-then-retract decision
order:

1. First real elevation reading anchors `lastFilteredZ` / `lastFilteredZPointXY`.
2. Below `thresholdZ_m`: no change.
3. An excursion that reverses direction AND returns within `thresholdZ_m` of
   the pre-excursion elevation AND has not moved `>= thresholdXY_m`
   horizontally since the elevation anchor is **retracted**: the exact amount
   the immediately preceding commit added is undone.
4. Otherwise the excursion is **committed** immediately, and becomes the new
   retractable delta.

Three new private fields were added: `lastFilteredZPointXY` (the elevation
anchor's own point, distinct from `lastFilteredPointXY` which belongs to
distance smoothing), `retractableDelta`, and `preRetractZ`. No early `return`
was introduced, and the block remains fully independent of `smoothedDistance`
— CONV-04's decoupling (elevation sampled on every point, not gated behind the
horizontal threshold) stays intact.

## Verification

- `cd web && npx vitest run` — 35/35 tests pass (up from the pre-plan 31),
  across all 5 test files.
- `cd web && npx vitest run src/lib/models/gpx/gpx-metrics-computation.test.ts`
  — 13/13 pass, including the CONV-04 88 m scramble fixture still reporting
  `elevationGain === 88` / `elevationLoss === 0`.
- `grep -c "retractableDelta" gpx-metrics-computation.ts` → 9 (>= 4 required).
- `grep -c "lastFilteredZPointXY" gpx-metrics-computation.ts` → 8 (>= 4 required).
- `grep -n "lastFilteredPointXY"` confirms it is used only by the first-call
  anchor branch and the (unchanged) distance-smoothing block — the two
  anchors stayed separate.
- `grep -n "smoothedDistance"` confirms matches only in the distance
  computation and distance-smoothing block, not inside the elevation logic.
- `cd web && npx svelte-check --output human --threshold error` → `0 errors and 0 warnings`.
- `git diff --stat` (against the plan's starting commit) touches exactly the
  two files in `files_modified` — no change to `gpx.ts`, `gpx_util.ts`, or any
  `.svelte` file.

Manually traced all four new fixtures plus the pre-existing CONV-03/CONV-04
fixtures through the new decision order by hand before running the suite;
every trace matched the executed result.

## Deviations from Plan

None — plan executed exactly as written. Both tasks matched the specified
algorithm, field names, and decision order verbatim.

## Known Stubs

None.

## Threat Flags

None — this plan changes only the arithmetic inside an existing pure class
(`GpxMetricsComputation`); no new network boundary, endpoint, persistence
write path, or dependency was introduced. Matches the plan's own threat
model, which assessed the change as "no new attack surface."

## Self-Check: PASSED

- `web/src/lib/models/gpx/gpx-metrics-computation.ts` — FOUND
- `web/src/lib/models/gpx/gpx-metrics-computation.test.ts` — FOUND
- Commit `15e56570` (test(33-04): add failing stationary-noise regression fixtures for CONV-04) — FOUND
- Commit `f981ce15` (feat(33-04): replace flat threshold elevation commit with commit-then-retract noise filter) — FOUND
