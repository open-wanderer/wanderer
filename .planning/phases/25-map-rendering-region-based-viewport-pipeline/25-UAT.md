---
status: complete
phase: 25-map-rendering-region-based-viewport-pipeline
source: [25-VERIFICATION.md]
started: 2026-07-23T13:15:00Z
updated: 2026-07-23T13:40:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Trail detail map offline render + hillshade z-order
expected: The basemap renders offline from the region's `.pmtiles`; hillshade (if DEM was downloaded) renders UNDERNEATH the vector basemap, not on top.
result: pass

### 2. Trail detail map uncovered viewport
expected: Opening a trail whose bounds fall OUTSIDE every downloaded region shows a blank basemap with NO "no offline data" banner/toast.
result: pass

### 3. Trail detail map mid-session incremental region add
expected: With the trail detail screen already open, finishing a region download from Settings causes the basemap to appear incrementally without remounting or a full-style flash; hillshade still underneath.
result: pass

### 3. Trail detail map mid-session incremental region add
expected: With the trail detail screen already open, finishing a region download from Settings causes the basemap to appear incrementally without remounting or a full-style flash; hillshade still underneath.
result: [pending]

### 4. Navigation screen region-boundary pan swap
expected: Starting navigation renders the region basemap + hillshade underneath offline; panning across the region boundary (two adjacent downloaded regions) swaps the newly-entered region's sources in and the departed region's out once the pan settles, with no full-style flash; a removed region visually disappears immediately (no stale tiles lingering until the next tap, confirming the repaint-nudge actually repaints on real hardware).
result: issue
reported: "The hot swapping does not work in navigation screen. Sometimes the map does not load at all, sometimes the trail layer disappears."
severity: major

### 5. Navigation screen within-region pan (no re-flicker) and uncovered-viewport blank state
expected: Panning within a single region does not re-flicker on every camera-idle (empty diff -> no-op); panning into an area with no downloaded region removes region sources -> blank basemap while navigation keeps tracking GPS/maneuvers, with no banner/toast; the swap happens on gesture settle, not a fixed delay.
result: pass

## Summary

total: 5
passed: 4
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- truth: "Panning across the region boundary swaps the newly-entered region's sources in and the departed region's out once the pan settles, with no full-style flash; a removed region visually disappears immediately."
  status: failed
  reason: "User reported: The hot swapping does not work in navigation screen. Sometimes the map does not load at all, sometimes the trail layer disappears."
  severity: major
  test: 4
  root_cause: ""
  artifacts: []
  missing: []
  debug_session: ""
