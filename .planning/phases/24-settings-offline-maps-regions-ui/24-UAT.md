---
status: complete
phase: 24-settings-offline-maps-regions-ui
source: [24-VERIFICATION.md]
started: 2026-07-22T14:25:00Z
updated: 2026-07-23T00:05:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Re-run 24-UAT.md test 2: on a physical Android device, pick a region that has NEVER been downloaded before, tap 'Download vector', watch it transition to `downloading` (progress bar advances, not `error`). Then pause it, resume it, and delete it via the confirm dialog.
expected: Package transitions to downloading (not error); pause/resume/delete all function as originally specified; the device log line 'freeDiskSpaceBytes gives: Specified path does not exist' no longer appears.
result: issue
reported: "Package does not transition to downloading. It immediately goes into downloaded after clicking the download button."
severity: major

### 2. Re-run 24-UAT.md test 3: toggle DEM on for that same never-downloaded region.
expected: DEM download starts (spinner, in-flight feedback), not error.
result: pass

### 3. Re-run 24-UAT.md test 4 (previously blocked): watch the disk-usage total as packages are added/removed, including while a download is paused mid-transfer.
expected: Total updates after each mutation and includes partial (.part) bytes for a paused/in-progress download.
result: issue
reported: "pass. Paused can not be tested (see test 1)"
severity: minor

### 4. Re-run 24-UAT.md test 5 (previously blocked): with airplane mode on and at least one previously-downloaded region, open the screen.
expected: The previously-downloaded region still appears and is usable; the screen does not blank or show a full-screen error just because the catalog refresh failed offline.
result: pass

## Summary

total: 4
passed: 2
issues: 2
pending: 0
skipped: 0
blocked: 0

## Gaps

- truth: "Package transitions to downloading (not error); pause/resume/delete all function as originally specified; the device log line 'freeDiskSpaceBytes gives: Specified path does not exist' no longer appears."
  status: failed
  reason: "User reported: Package does not transition to downloading. It immediately goes into downloaded after clicking the download button."
  severity: major
  test: 1
  root_cause: ""
  artifacts: []
  missing: []
  debug_session: ""

- truth: "Total updates after each mutation and includes partial (.part) bytes for a paused/in-progress download."
  status: failed
  reason: "User reported: pass. Paused can not be tested (see test 1) — the paused/in-progress .part-byte accounting could not be exercised because the download never reaches a downloading/paused state (same underlying defect as test 1)."
  severity: minor
  test: 3
  root_cause: ""
  artifacts: []
  missing: []
  debug_session: ""
