---
status: testing
phase: 24-settings-offline-maps-regions-ui
source: [24-VERIFICATION.md]
started: 2026-07-22T14:25:00Z
updated: 2026-07-22T14:25:00Z
---

## Current Test

number: 1
name: Re-run 24-UAT.md test 2 on a never-before-downloaded region (vector download)
expected: |
  Package transitions to downloading (not error); pause/resume/delete all function as originally specified; the device log line "freeDiskSpaceBytes gives: Specified path does not exist" no longer appears.
awaiting: user response

## Tests

### 1. Re-run 24-UAT.md test 2: on a physical Android device, pick a region that has NEVER been downloaded before, tap 'Download vector', watch it transition to `downloading` (progress bar advances, not `error`). Then pause it, resume it, and delete it via the confirm dialog.
expected: Package transitions to downloading (not error); pause/resume/delete all function as originally specified; the device log line 'freeDiskSpaceBytes gives: Specified path does not exist' no longer appears.
result: [pending]

### 2. Re-run 24-UAT.md test 3: toggle DEM on for that same never-downloaded region.
expected: DEM download starts (spinner, in-flight feedback), not error.
result: [pending]

### 3. Re-run 24-UAT.md test 4 (previously blocked): watch the disk-usage total as packages are added/removed, including while a download is paused mid-transfer.
expected: Total updates after each mutation and includes partial (.part) bytes for a paused/in-progress download.
result: [pending]

### 4. Re-run 24-UAT.md test 5 (previously blocked): with airplane mode on and at least one previously-downloaded region, open the screen.
expected: The previously-downloaded region still appears and is usable; the screen does not blank or show a full-screen error just because the catalog refresh failed offline.
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
