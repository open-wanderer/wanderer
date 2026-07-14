---
phase: quick-260714-qma
plan: 01
subsystem: auth
tags: [flutter, riverpod, objectbox, auth-provider, dio]

# Dependency graph
requires:
  - phase: quick-260712-pac
    provides: typed AsyncValue listener closures on authProvider (unrelated fix, same file area)
provides:
  - auth_provider.dart build() catch block simplified back to two branches (auth-error logout, everything-else offline-preserve)
  - LocalSettingsEntity schema reverted to themeMode-only (no persisted failure counter)
affects: [auth-provider, local-settings, objectbox-schema]

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - app/lib/provider/auth_provider.dart
    - app/lib/entities/local_settings_entity.dart
    - app/lib/objectbox.g.dart
    - app/lib/objectbox-model.json
    - app/lib/provider/auth_provider.g.dart

key-decisions:
  - "Removed the persisted consecutive-500-counter self-heal logic entirely rather than gating it off, since the root cause it targeted (thundering-herd/SQLite contention) was unconfirmed and has since been superseded by a real, separately-fixed root cause (Meilisearch token expiry)"
  - "Kept the timeout-gated build() await and 401/403/404 deferred-logout branches untouched — that logic addresses an independent, legitimate race condition"
  - "Regenerated only auth_provider.g.dart among the four provider .g.dart files build_runner touched — the other three (router_provider.g.dart, glyph_sprite_cache_provider.g.dart, map_style_json_provider.g.dart) reflect pre-existing unrelated dirty source files and were left uncommitted as out-of-scope per the scope boundary rule"

patterns-established: []

requirements-completed: [QUICK-260714-qma]

# Metrics
duration: ~10min
completed: 2026-07-14
---

# Quick Task 260714-qma: Revert Speculative Persisted 500-Counter Summary

**Reverted the persisted consecutive-500-counter self-heal logic in `auth_provider.dart`/`LocalSettingsEntity` (commits `d9531951`, `dd56c62d`), while preserving the independent timeout-gated `build()` race-condition fix.**

## Performance

- **Duration:** ~10 min
- **Completed:** 2026-07-14T17:15:28Z
- **Tasks:** 2 completed (+ 1 necessary generated-code sync commit)
- **Files modified:** 5

## Accomplishments
- `Auth.build()`'s catch block now has exactly two branches: `_isAuthError` (401/403/404 → deferred logout) and a catch-all that preserves the cached session (covers 5xx and connection errors alike)
- Removed `_maxConsecutiveValidationFailures`, `_localSettingsBox`, `_isServerError`, `_bumpConsecutiveValidationFailures`, `_resetConsecutiveValidationFailures` and their call sites from `auth_provider.dart`
- `LocalSettingsEntity` reverted to a single `themeMode` field; ObjectBox generated bindings (`objectbox.g.dart`, `objectbox-model.json`) regenerated via `build_runner` and confirmed clean of `consecutiveAuthValidationFailures`
- `build()`'s timeout-gated await (`_updateUserEntity(...).timeout(const Duration(seconds: 3))`), the `TimeoutException` offline-fallback branch, and the 401/403/404 deferred-logout branch are all unchanged

## Task Commits

Each task was committed atomically:

1. **Task 1: Strip the 500-counter self-heal logic from auth_provider.dart** - `c3fbef98` (fix)
2. **Task 2: Revert the persisted counter field and regenerate ObjectBox bindings** - `a6647dcf` (fix)

Additional deviation commit:
3. **Regenerate stale auth_provider.g.dart hash** - `eee4f0c2` (chore) — see Deviations below

**Plan metadata:** committed separately by the orchestrator after this summary.

## Files Created/Modified
- `app/lib/provider/auth_provider.dart` - Removed the persisted-counter self-heal branch from `build()`'s catch block; kept auth-error and timeout handling
- `app/lib/entities/local_settings_entity.dart` - Removed `consecutiveAuthValidationFailures` field and constructor param; `themeMode` untouched
- `app/lib/objectbox.g.dart` - Regenerated; no longer references the removed property
- `app/lib/objectbox-model.json` - Regenerated; ObjectBox model no longer declares the removed property (build_runner logged "removing from the model")
- `app/lib/provider/auth_provider.g.dart` - Regenerated Riverpod provider hash to match the Task 1 source edit

## Decisions Made
- Full removal (not a feature flag/gate) — the theory behind the counter is unconfirmed and now superseded by a separately fixed real root cause, per the plan's stated rationale.
- Left three other build_runner-regenerated `.g.dart` files (`router_provider.g.dart`, `glyph_sprite_cache_provider.g.dart`, `map_style_json_provider.g.dart`) uncommitted — their hash changes stem from pre-existing, unrelated dirty source files already in the working tree before this task started, out of this task's scope.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Regenerated stale auth_provider.g.dart hash**
- **Found during:** Task 2 (build_runner run for ObjectBox regeneration also regenerates all Riverpod `.g.dart` outputs)
- **Issue:** Task 1's edit to `auth_provider.dart` changed the source that `_$authHash()` is derived from; the stale hash in `auth_provider.g.dart` was left un-regenerated by the Task 1 commit, which would surface as generated-code drift
- **Fix:** Committed the regenerated `auth_provider.g.dart` (single-line hash change) produced by the same `build_runner` invocation Task 2 required anyway
- **Files modified:** app/lib/provider/auth_provider.g.dart
- **Verification:** `flutter analyze lib/provider/auth_provider.dart` reports no issues; diff confirms only the hash literal changed
- **Committed in:** `eee4f0c2`

---

**Total deviations:** 1 auto-fixed (1 bug — stale generated code)
**Impact on plan:** Necessary for generated-code consistency; no scope creep. Three other regenerated `.g.dart` files (reflecting pre-existing unrelated dirty sources) were deliberately left uncommitted.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- No follow-up work required; this quick task is self-contained.
- Pre-existing unrelated working-tree changes (`router_provider.dart`, i18n files, `trail.dart`, `pubspec.yaml`/`.lock`, web `trail.ts`/`gpx_util.ts`, new `trail_create_screen.dart`/`trail_source_select_screen.dart` routes, `web/src/routes/api/v1/trail/convert/`) remain untouched in the working tree — out of scope for this task, left for the user's separate in-progress work.

---
*Phase: quick-260714-qma*
*Completed: 2026-07-14*

## Self-Check: PASSED

- FOUND: app/lib/provider/auth_provider.dart
- FOUND: app/lib/entities/local_settings_entity.dart
- FOUND: app/lib/objectbox.g.dart
- FOUND: app/lib/objectbox-model.json
- FOUND: .planning/quick/260714-qma-revert-speculative-persisted-500-counter/260714-qma-SUMMARY.md
- FOUND commit: c3fbef98
- FOUND commit: a6647dcf
- FOUND commit: eee4f0c2
