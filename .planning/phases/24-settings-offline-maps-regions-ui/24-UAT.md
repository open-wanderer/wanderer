---
status: testing
phase: 24-settings-offline-maps-regions-ui
source: [24-VERIFICATION.md]
started: 2026-07-22T00:00:00Z
updated: 2026-07-22T00:00:00Z
---

## Current Test

number: 1
name: Open Settings -> Offline Maps/Regions on a physical device; confirm the list is A-Z and searchable by name.
expected: |
  Flat list, alphabetically sorted, filters live as you type in the search bar.
awaiting: user response

## Tests

### 1. Open Settings -> Offline Maps/Regions on a physical device; confirm the list is A-Z and searchable by name.
expected: Flat list, alphabetically sorted, filters live as you type in the search bar.
result: [pending]

### 2. Download a ready region: start a vector download, watch the progress bar, pause it, resume it, then delete it via the confirm dialog.
expected: A single combined progress bar advances during download; pause stops it; resume continues it; delete requires confirming in a dialog and then removes the region from disk.
result: [pending]

### 3. Toggle a region's DEM switch on, wait for completion, then toggle it off.
expected: Toggle-on starts a DEM download with visible in-flight feedback (spinner); toggle-off deletes only the DEM immediately with no dialog, and the vector package/file remain downloaded and usable.
result: [pending]

### 4. Watch the disk-usage total as packages are added/removed, including while a download is paused mid-transfer.
expected: The total updates after each mutation and includes partial (.part) bytes for a paused/in-progress download.
result: [pending]

### 5. With airplane mode on and at least one previously-downloaded region, open the screen.
expected: The previously-downloaded region still appears and is usable; the screen does not blank or show a full-screen error just because the catalog refresh failed offline.
result: [pending]

### 6. View a building/error catalog region row.
expected: Row renders dimmed/disabled with a caption ('Not yet available' / 'Build failed') and no Download action.
result: [pending]

## Summary

total: 6
passed: 0
issues: 0
pending: 6
skipped: 0
blocked: 0

## Gaps
