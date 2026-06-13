---
status: testing
phase: 02-navigation-screen
source: [02-VERIFICATION.md]
started: 2026-06-13T00:00:00Z
updated: 2026-06-13T00:00:00Z
---

## Current Test

number: 1
name: End-to-end navigation launch
expected: |
  Tapping Navigate on a trail opens a spinner in the button, then transitions to NavigationScreen full-screen with map centered on GPS
awaiting: user response

## Tests

### 1. End-to-end navigation launch
expected: Tapping Navigate on a trail opens a spinner in the button, then transitions to NavigationScreen full-screen with map centered on GPS
result: [pending]

### 2. Maneuver auto-advance and breadcrumb growth
expected: Moving near a maneuver shape point (within 30m) advances the instruction banner; a red polyline traces the traveled path
result: [pending]

### 3. Compass toggle north-up / heading-up
expected: Tapping the compass button switches between north-up and heading-up map orientation
result: [pending]

### 4. Free-pan and recenter behavior
expected: Free-panning the map disables follow; tapping the recenter button resumes GPS follow; pinch-zoom does NOT disable follow
result: [pending]

### 5. Completion banner at last maneuver
expected: When the last maneuver is reached, a completion banner appears; map and GPS stay active, no auto-pop
result: [pending]

### 6. Exit button
expected: Tapping exit returns to the screen that launched navigation (trail detail or trail detail map)
result: [pending]

### 7. Navigate button on trail detail map screen
expected: A full-width Navigate button floats above the elevation profile on the trail detail map screen and launches navigation
result: [pending]

### 8. Error toast on network failure
expected: When the navigate API call fails, an error toast appears and the user stays on the originating screen (no crash)
result: [pending]

## Summary

total: 8
passed: 0
issues: 0
pending: 8
skipped: 0
blocked: 0

## Gaps
