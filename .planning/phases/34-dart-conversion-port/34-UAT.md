---
status: testing
phase: 34-dart-conversion-port
source: [34-VERIFICATION.md, 34-06-PLAN.md, 34-07-PLAN.md]
started: 2026-08-01T13:20:00Z
updated: 2026-08-01T13:20:00Z
---

## Current Test

number: 1
name: Offline flows complete with no dead end
expected: |
  All three offline capture flows complete with no network call and no dead
  end — the D-15/D-16 fix for route_planner_screen.dart's pre-phase offline
  stranding.
awaiting: user response

## Tests

### 1. Offline flows complete with no dead end
steps: |
  Put the device in airplane mode, then:
  (a) record a short track and save it;
  (b) plan a 3-anchor route and tap Finish;
  (c) import a .gpx file from the share sheet.
expected: |
  (a) the save-options sheet does NOT appear; the app lands on
  trail_create_screen with the track drawn.
  (b) no error toast, no stall; the app lands on trail_create_screen with all
  three anchors intact. (Before this phase the planner showed an error toast
  and stranded you here — you could not finish a route offline at all.)
  (c) no options sheet; a trail is created with correct distance/elevation.
result: [pending]

### 2. Anchor structure survives the online round trip
steps: |
  Online: plan a 3-anchor route, tap Finish, enable BOTH toggles (recalculate
  heights + follow roads), confirm. Then re-open the resulting trail in the
  route planner.
expected: |
  It still shows three anchors, not a collapsed start/end pair. This exercises
  the leg-boundary anchor re-pin, which the planner flagged as the phase's
  highest-risk new code — snapped legs must be forced back to the original
  anchor coordinates, since anchors are located by exact coordinate match.
result: [pending]

### 3. Transcode round trip and published API docs
steps: |
  With the app pointed at a server running this change:
  (a) import a .kml and a .fit file while online;
  (b) open /docs/api and find the POST /api/v1/trail/convert entry;
  (c) on the web trail-edit page, import a file with the client-side picker.
expected: |
  (a) each produces a trail whose distance/elevation match what the same track
  reports after saving — proving the app measured the server-transcoded GPX
  itself rather than trusting a server-computed value (PORT-05).
  (b) the entry describes a GPX response, not a Trail.
  (c) still works unchanged — the web computes client-side and only uses the
  endpoint for transcoding (D-08).
result: [pending]

## Summary

total: 3
passed: 0
issues: 0
pending: 3
skipped: 0
blocked: 0

## Gaps
