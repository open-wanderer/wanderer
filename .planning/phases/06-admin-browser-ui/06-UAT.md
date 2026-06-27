---
status: testing
phase: 06-admin-browser-ui
source: [06-VERIFICATION.md]
started: 2026-06-27T20:00:00Z
updated: 2026-06-27T20:00:00Z
---

## Current Test

number: 1
name: Auth gate visibility - login prompt shown to unauthenticated users
expected: |
  Visiting /federation/ without a valid PocketBase superuser session shows
  the "Admin login required" prompt with a link to /_/. The dashboard content
  is hidden. After logging in at /_/ and returning to /federation/, the
  dashboard loads and shows the peer table.
awaiting: user response

## Tests

### 1. Auth gate visibility
expected: Login prompt shown when no valid superuser session in localStorage; dashboard shown after login
result: [pending]

### 2. End-to-end peer workflow
expected: Discovery → Connect → Approve/Reject/Disconnect workflow completes successfully against live Phase 5 API endpoints
result: [pending]

### 3. Dark/light theme flash prevention
expected: No visible flash when loading the page in dark mode; theme is applied before body renders
result: [pending]

## Summary

total: 3
passed: 0
issues: 0
pending: 3
skipped: 0
blocked: 0

## Gaps
