---
status: diagnosed
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
  root_cause: "_reconcileRegionComposition() has no reentrancy guard. ml.MapEventCameraIdle was wired assuming it fires once per settled user gesture (the map_screen.dart precedent), but navigation_screen.dart also drives the camera continuously and programmatically via _pushCamera() (called from the 200ms GPS-fix position tween in _applyAnimatedFrame and from _onBearingFollowTick while heading-up follow is active), each of which re-fires MapEventCameraIdle and re-triggers the reconcile fire-and-forget with no coalescing. The REMOVE loop unconditionally advances _addedSourceIds/_addedLayerIds even when removeLayer/removeSource throws, while the ADD loop only advances tracking state on success -- this asymmetry lets overlapping reconciles desync the Dart-side tracking sets from the real native style, producing either a silently-failed/never-retried add (map does not load) or a race against a concurrent _swapStyle() setStyle call that wipes and re-adds trail/breadcrumb layers (trail layer disappears). Independently predicted by 25-REVIEW.md's WR-03 finding. A region/trail layer-id naming collision was investigated and ruled out."
  artifacts:
    - path: "app/lib/routes/navigation_screen.dart"
      issue: "_reconcileRegionComposition has no in-flight/reentrancy guard; onEvent's MapEventCameraIdle branch over-triggers due to continuous _pushCamera calls during active navigation follow; REMOVE loop mutates tracking sets unconditionally while ADD loop only mutates on success (asymmetric, self-desyncing error handling)"
  missing:
    - "In-flight guard so overlapping _reconcileRegionComposition calls coalesce/no-op instead of interleaving"
    - "Symmetric tracking-set mutation (only advance on verified success, or re-derive from native truth via style.getLayerIds() instead of an optimistic diff)"
    - "Gate/suppress camera-idle-triggered reconcile while navigation's own continuous camera-follow (_followEnabled) is active, since raw MapEventCameraIdle is not a valid debounce signal on this screen"
  debug_session: ".planning/debug/navigation-screen-region-swap-broken.md"
