---
status: testing
phase: 31-flutter-settings-hierarchy
source: [31-VERIFICATION.md]
started: 2026-07-27T00:00:00Z
updated: 2026-07-27T00:00:00Z
---

## Current Test

number: 1
name: On-device collapsible hierarchy walkthrough
expected: |
  Open Settings → Offline Maps/Regions on a real device/simulator. Group nodes
  expand/collapse on tap, revealing nested child groups and leaf regions matching
  the admin-defined tree shape. Each leaf region still exposes its existing
  independent Vector and DEM download/cancel/delete controls, now nested inside
  the hierarchy. The disk-usage summary (total + per-region breakdown) still
  works unchanged.
awaiting: user response

## Tests

### 1. On-device collapsible hierarchy walkthrough
expected: Group nodes expand/collapse to reveal child groups and leaf regions matching the admin tree; per-region Vector/DEM controls and disk-usage summary work unchanged inside the new hierarchical layout.
result: [pending]

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps
