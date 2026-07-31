---
status: complete
phase: 33-conversion-correctness
source: [33-VERIFICATION.md]
started: 2026-07-31T17:05:00Z
updated: 2026-07-31T15:06:27Z
---

## Current Test

[testing complete]

## Tests

### 1. Crop panel must not overwrite hand-entered trail metrics
steps: Open a new (empty or near-empty) trail. Type distance/duration/elevation values by hand under "basic info". Click "draw route", then click the crop icon to open the crop panel without drawing anything.
expected: The hand-typed distance/duration/elevation values remain unchanged after the crop panel opens (crop preview totals render in the crop panel itself, not in the form). Save and reload to confirm the typed values persisted.
result: pass

### 2. No crop pins stranded at 0N 0E
steps: Repeat the sequence above and watch the map after the crop panel opens on a degenerate/empty route. Then open the crop panel on a normal route with a real track and drag the slider.
expected: No crop pins appear at [0,0] (off the coast of West Africa) on the degenerate route, before or after the panel's open animation settles. Pins render and track the slider correctly on a normal route.
result: pass

### 3. No negative per-segment elevation in the anchor list
steps: In the trail-edit anchor list, create a route with at least two segments where the elevation rises then returns close to its prior value across a segment boundary (climb, pause/reverse briefly at the anchor, then continue). Check the per-segment elevation gain/loss shown next to each anchor.
expected: No segment ever displays a negative elevation gain or loss.
result: pass

## Summary

total: 3
passed: 3
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps
