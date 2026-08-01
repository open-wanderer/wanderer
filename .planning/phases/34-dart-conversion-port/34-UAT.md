---
status: complete
phase: 34-dart-conversion-port
source: [34-VERIFICATION.md, 34-06-PLAN.md, 34-07-PLAN.md]
started: 2026-08-01T13:20:00Z
updated: 2026-08-01T14:05:00Z
---

## Current Test

[testing complete]

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
result: issue
reported: |
  a) pass
  b) partial. Why does the save options sheet appear in route planner when
  online? It has no purpose. The route is following roads and using valhalla
  elevation anyways
  c) partial. The save options sheet has the wrong title ("Save recordings")
  and needs more bottom padding because of the bottom navigation bar
severity: minor
note: |
  Offline behaviour (the D-15/D-16 fix under test) passed for all three flows.
  Both reported problems are about the ONLINE save-options sheet.

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
result: skipped
reason: |
  User: "See my answer in Test 1. This is unnecessary." The test's setup step
  (enable both toggles on the route planner's save-options sheet) is exactly
  the sheet the user says should not appear for the route planner at all —
  the route already follows roads and already carries Valhalla elevation.
  If gap 1 is closed by removing the sheet from the planner flow, this
  round trip becomes unreachable and the leg-boundary anchor re-pin becomes
  dead code on that path. Diagnosis must confirm whether the re-pin is still
  reachable from any other flow before it is dropped or left untested.
severity_note: not a code defect — superseded by gap 1

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
result: issue
reported: |
  a) pass. However: the same trail uploaded on the web vs. the app produces
  different lengths:
  ~/Downloads/19440058502_ACTIVITY.fit
  On web: 10.51km | On app: 10.97km
  Elevation matches: 344m up, 351 down
  b) pass. Check if the json content type is still needed. If not remove it
  alongside the associated tests
  c) pass
severity: major
note: |
  All three sub-checks passed as written. The mismatch is a cross-client
  disagreement the test did not cover: same .fit file, same transcoded GPX,
  same elevation totals (344 up / 351 down) but 10.51 km on web vs 10.97 km
  on the app — a ~4.4% distance-only divergence.

## Summary

total: 3
passed: 0
issues: 2
pending: 0
skipped: 1
blocked: 0

## Gaps

- truth: "Online route planner Finish should not offer save options that cannot change the result"
  status: failed
  reason: "User reported: b) partial. Why does the save options sheet appear in route planner when online? It has no purpose. The route is following roads and using valhalla elevation anyways"
  severity: minor
  test: 1
  root_cause: ""
  artifacts: []
  missing: []
  debug_session: ""
  supersedes_test: 2

- truth: "The save-options sheet is correctly titled and laid out for the flow that opened it"
  status: failed
  reason: "User reported: c) partial. The save options sheet has the wrong title (\"Save recordings\") and needs more bottom padding because of the bottom navigation bar"
  severity: cosmetic
  test: 1
  root_cause: ""
  artifacts: []
  missing: []
  debug_session: ""

- truth: "The same track file yields the same distance whether imported on web or in the app"
  status: failed
  reason: "User reported: the same trail uploaded on the web vs. the app produces different lengths for ~/Downloads/19440058502_ACTIVITY.fit — web 10.51km, app 10.97km. Elevation matches exactly (344m up, 351m down)."
  severity: major
  test: 3
  evidence:
    file: "~/Downloads/19440058502_ACTIVITY.fit"
    web_distance_km: 10.51
    app_distance_km: 10.97
    elevation_gain_m: 344
    elevation_loss_m: 351
    note: "Elevation agreeing while distance diverges points at the distance formula/filtering, not at the transcode or the point set."
  root_cause: ""
  artifacts: []
  missing: []
  debug_session: ""

- truth: "The convert endpoint carries no content type it no longer serves"
  status: failed
  reason: "User reported: Check if the json content type is still needed. If not remove it alongside the associated tests"
  severity: minor
  test: 3
  root_cause: ""
  artifacts: []
  missing: []
  debug_session: ""
