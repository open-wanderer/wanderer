---
status: complete
phase: 31-flutter-settings-hierarchy
source: [31-VERIFICATION.md]
started: 2026-07-27T00:00:00Z
updated: 2026-07-27T00:00:00Z
---

## Current Test

[testing complete]

## Tests

### 1. On-device collapsible hierarchy walkthrough
expected: Group nodes expand/collapse to reveal child groups and leaf regions matching the admin tree; per-region Vector/DEM controls and disk-usage summary work unchanged inside the new hierarchical layout.
result: issue
reported: "It works but is unnecessary. Only show the regions (and their parents) that are actually downloadable."
severity: minor

## Summary

total: 1
passed: 0
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- truth: "Settings → Offline Maps/Regions shows the collapsible hierarchy with group and leaf rows matching the admin-defined tree"
  status: failed
  reason: "User reported: It works but is unnecessary. Only show the regions (and their parents) that are actually downloadable."
  severity: minor
  test: 1
  root_cause: ""
  artifacts: []
  missing: []
  debug_session: ""
