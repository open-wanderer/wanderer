---
status: partial
phase: 23-tilerepositorymanager-download-engine
source: [23-VERIFICATION.md]
started: 2026-07-22T12:30:00Z
updated: 2026-07-22T13:00:00Z
---

## Current Test

[testing paused — 5 items outstanding, deferred until Phase 24 ships]

**Deferred:** User decided to defer this on-device human-check until after Phase 24 (Settings —
Offline Maps/Regions UI) ships. Rationale: the debug-only `tile_repository_manager_harness.dart`
is a workable stand-in, but all 5 behaviors are more naturally and more representatively exercised
through the real Settings UI once it exists, rather than the throwaway harness. Re-run
`/gsd-verify-work 23` once Phase 24 is built — either via the harness or the shipped UI, whichever
is available at that point.

## Tests

### 1. RESUME (TILE-02)
expected: Logged bytes continue from partial offset; resumed request is `206`; file validates and promotes to `downloaded`.
result: skipped
reason: Deferred until Phase 24 Settings UI ships — user's decision, not a technical blocker.

### 2. DISK REFUSAL (TILE-03)
expected: On a device/emulator with little free space, a download exceeding free space minus the 1.75x margin is refused; package marked `error`; no `.part` file (or only a zero-byte one) written.
result: skipped
reason: Deferred until Phase 24 Settings UI ships — user's decision, not a technical blocker.

### 3. BACKGROUNDING (TILE-04)
expected: Starting a download then backgrounding the app shows `paused` (not a silently dead transfer); returning to foreground resumes cleanly from the preserved `.part` file.
result: skipped
reason: Deferred until Phase 24 Settings UI ships — user's decision, not a technical blocker.

### 4. DEM INDEPENDENCE (DEM-01/02)
expected: Toggling DEM on/off independently of vector never affects the other package; deleting one leaves the other's status/file untouched.
result: skipped
reason: Deferred until Phase 24 Settings UI ships — user's decision, not a technical blocker.

### 5. QUERY (TILE-05)
expected: With a region fully downloaded, "Query inside bbox" prints the region's real local file path(s); "Query outside bbox" prints an empty result.
result: skipped
reason: Deferred until Phase 24 Settings UI ships — user's decision, not a technical blocker.

## Summary

total: 5
passed: 0
issues: 0
pending: 0
skipped: 5
blocked: 0

## Gaps
