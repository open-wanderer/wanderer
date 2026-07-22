---
status: testing
phase: 23-tilerepositorymanager-download-engine
source: [23-VERIFICATION.md]
started: 2026-07-22T12:30:00Z
updated: 2026-07-22T12:30:00Z
---

## Current Test

number: 1
name: RESUME (TILE-02)
expected: |
  On a physical device against a backend with a `ready` region, start the region's vector
  download via the harness (`flutter run -t test/services/tile_repository_manager_harness.dart`);
  partway through, toggle airplane mode so the transfer fails; restore connectivity and resume.
  The logged received/total byte count continues from the partial offset (not restarting at 0),
  the resumed HTTP request hits the server as 206 Partial Content (not a full 200 re-download),
  the final file passes PmTilesArchive validation, and the package promotes to `downloaded`.
awaiting: user response

## Tests

### 1. RESUME (TILE-02)
expected: Logged bytes continue from partial offset; resumed request is `206`; file validates and promotes to `downloaded`.
result: [pending]

### 2. DISK REFUSAL (TILE-03)
expected: On a device/emulator with little free space, a download exceeding free space minus the 1.75x margin is refused; package marked `error`; no `.part` file (or only a zero-byte one) written.
result: [pending]

### 3. BACKGROUNDING (TILE-04)
expected: Starting a download then backgrounding the app shows `paused` (not a silently dead transfer); returning to foreground resumes cleanly from the preserved `.part` file.
result: [pending]

### 4. DEM INDEPENDENCE (DEM-01/02)
expected: Toggling DEM on/off independently of vector never affects the other package; deleting one leaves the other's status/file untouched.
result: [pending]

### 5. QUERY (TILE-05)
expected: With a region fully downloaded, "Query inside bbox" prints the region's real local file path(s); "Query outside bbox" prints an empty result.
result: [pending]

## Summary

total: 5
passed: 0
issues: 0
pending: 5
skipped: 0
blocked: 0

## Gaps
