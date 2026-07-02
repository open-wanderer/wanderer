---
phase: quick-260702-m4u
plan: 01
subsystem: auth
tags: [flutter, dart, riverpod, dio, objectbox, optimistic-ui]

requires: []
provides:
  - Optimistic Auth.build() that returns the cached user immediately and refreshes in the background
affects: [app startup, auth restoration, session invalidation]

tech-stack:
  added: []
  patterns:
    - "Optimistic provider build(): return cached entity, fire refresh via unawaited(...) with catchError->logout"

key-files:
  created: []
  modified:
    - app/lib/provider/auth_provider.dart

key-decisions:
  - "Used a single .catchError((Object err) => logout()) on the unawaited refresh instead of an on-DioException catch, so the 404->logout intent is preserved while any other session-invalidating error also triggers logout"
  - "Swapped the dio import for dart:async since DioException is no longer referenced and unawaited() is needed"

patterns-established:
  - "Optimistic build(): return cached state immediately, refresh in the background via unawaited() and log out on failure"

requirements-completed: [QUICK-m4u]

duration: 3min
completed: 2026-07-02
---

# Quick Task 260702-m4u: Optimistic Auth.build() Summary

**`Auth.build()` now returns the cached user immediately on startup and refreshes user data in the background via `unawaited(...)`, logging out if the refresh reveals an invalid session.**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-07-02T11:21:04Z
- **Completed:** 2026-07-02
- **Tasks:** 1 of 2 (Task 2 is a human-verify checkpoint)
- **Files modified:** 1

## Accomplishments
- Startup no longer blocks on the `_updateUserEntity` network round-trip when a saved user + `pb_auth` cookie exist — the cached `savedUserEntity` is returned immediately.
- `_updateUserEntity(savedUserEntity.id)` now runs in the background via `unawaited(...)`, keeping the analyzer clean on the discarded future.
- Background refresh failure (DioException 404 or any other error) calls `logout()`, preserving the original 404 -> logout intent while defensively handling any session-invalidating error.
- `login()` and `refresh()` call sites of `_updateUserEntity` remain awaited and unchanged; `_updateUserEntity` body unchanged.

## Task Commits

1. **Task 1: Make Auth.build() return the cached user immediately and refresh in the background** - `d2d126a8` (feat)

## Files Created/Modified
- `app/lib/provider/auth_provider.dart` - Replaced the awaited `try/on DioException` block in `build()` with an immediate `return savedUserEntity;` plus a background `unawaited(_updateUserEntity(...).catchError(...))` that logs out on failure. Swapped `package:dio/dio.dart` import for `dart:async`.

## Decisions Made
- Used a single `.catchError((Object err) => logout())` on the unawaited future rather than an `on DioException` clause. This preserves the 404 -> logout behavior while also logging out on any other session-invalidating error, and removes the now-unused `dio` import (replaced by `dart:async` for `unawaited`).

## Deviations from Plan
None - plan executed exactly as written. (The plan explicitly permitted a `.catchError`/`then` chain and swapping to `dart:async`; the `dio` import became unused as a direct consequence and was removed.)

## Issues Encountered
- Removing the `dio` import initially left the `on DioException` catch clause dangling (analyzer error), which was resolved by the same edit that rewrote the `build()` branch to use `.catchError`. Final `dart analyze lib/provider/auth_provider.dart` reports "No issues found!".

## Verification
- `cd app && dart analyze lib/provider/auth_provider.dart` -> No issues found.
- `build()` returns `savedUserEntity` before the background refresh completes; refresh call wrapped in `unawaited(...)`, not awaited.
- `login()` (still `await _updateUserEntity(...)`) and `refresh()` (`AsyncValue.guard(() => _updateUserEntity(id))`) unchanged.

## Pending Checkpoint (Task 2 — human-verify)

On-device verification required (cannot be automated in this environment):
1. Launch the app while already logged in — confirm the authenticated UI appears immediately with no startup network stall.
2. Confirm user data still refreshes shortly after startup (background `_updateUserEntity` ran).
3. (Optional) With a stale/invalid session where `/user/{id}` returns 404, confirm the app logs out shortly after startup.

## Next Phase Readiness
- Code change complete and analysis-clean. Awaiting human on-device verification (Task 2 checkpoint) before final sign-off.

## Self-Check: PASSED

- FOUND: `.planning/quick/260702-m4u-make-auth-provider-dart-build-optimistic/260702-m4u-SUMMARY.md`
- FOUND: commit `d2d126a8`
- FOUND: `unawaited(` in `app/lib/provider/auth_provider.dart` `build()`

---
*Phase: quick-260702-m4u*
*Completed: 2026-07-02*
