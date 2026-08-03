---
phase: 36-local-first-recording-automatic-upload
plan: 09
subsystem: ui
tags: [riverpod, dio, objectbox, trail-filter, flutter]

# Dependency graph
requires:
  - phase: 36-07
    provides: profile_trails_provider.dart's local-first own-trails merge (offline flag, isOwnHandle)
provides:
  - Device-derived offline fallback bounds for the trail filter (kOfflineTrailFilterValues + computeOfflineTrailFilterValues + readLocalTrailMetrics)
  - A bounded retry policy on trailFilterProvider (2 attempts, not 10)
  - A render path (profile_trails_provider.dart + profile_trail_screen.dart) that no longer discards an already-rendered own-trails list on filter-driven reload or search
affects: [profile, trail-filter, offline-ux]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ref.watch(provider.select((async) => async.value)) to decouple a dependent provider from a watched AsyncNotifier's error/retry churn"
    - "AsyncLoading<T>().copyWithPrevious(state) to keep a rendered list visible during a state-driven reload (search bursts)"
    - "Pure bounds-computation function + thin ObjectBox-reading shim, with the shim explicitly marked deliberately-uncovered by flutter test (no ObjectBox harness in this repo)"

key-files:
  created:
    - app/lib/util/offline_trail_filter_bounds.dart
    - app/test/util/offline_trail_filter_bounds_test.dart
    - app/test/provider/trail/trail_filter_fallback_test.dart
  modified:
    - app/lib/provider/trail/trail_filter_provider.dart
    - app/lib/provider/trail/trail_filter_provider.g.dart
    - app/lib/provider/profile/profile_trails_provider.dart
    - app/lib/routes/profile_trail_screen.dart

key-decisions:
  - "Offline filter fallback bounds are computed from the signed-in account's own on-device trails (nextBoundAbove rounding, 5km/250m steps), not from a fixed constant -- the constant survives only as the empty-store floor (2026-08-03 user decision)"
  - "trailFilterProvider degrades to a device-derived fallback on DioException connection failure instead of throwing, so there is nothing left for Riverpod's defaultRetry to retry; every other error still throws"
  - "trailFilterRetry caps automatic retries at 2 (400ms, 800ms), replacing defaultRetry's 10 attempts over ~45s"

patterns-established:
  - "buildDefaultTrailFilter(values) is the single sanctioned constructor for the default TrailFilter, used identically by the online success path and the offline fallback path, guaranteeing max == limit on every bounded axis regardless of where the values came from"

requirements-completed: [REC-06]

# Metrics
duration: ~20min
completed: 2026-08-03
---

# Phase 36 Plan 09: Offline trail filter bounds and the own-trails spinner-flicker fix Summary

**Own-trails list stops flickering to a spinner offline: `trailFilterProvider` degrades to a device-derived fallback filter on connection failure instead of throwing into a 10-attempt retry storm, and the render path (`.select`, `copyWithPrevious`, `skipLoadingOnReload`) no longer discards an already-rendered list on any of the three churn sources that used to blank it.**

## Performance

- **Duration:** ~20 min
- **Tasks:** 3 completed
- **Files modified:** 7 (3 created, 4 modified)

## Accomplishments
- `offline_trail_filter_bounds.dart`: pure `nextBoundAbove`/`computeOfflineTrailFilterValues` bound arithmetic plus an account-scoped ObjectBox shim (`readLocalTrailMetrics`), unit-tested against real numeric input (empty store, single short trail, exact-multiple rounding boundary, per-axis independence, non-finite guards).
- `trail_filter_provider.dart`: extracted `buildDefaultTrailFilter(values)`, added a `DioException`/`isConnectionFailure` branch that returns a device-derived fallback filter instead of throwing, and wired a 2-attempt `trailFilterRetry` policy via `@Riverpod(retry: trailFilterRetry)` — replacing Riverpod's 10-attempt `defaultRetry` that fed the ~20-flash spinner storm.
- `profile_trails_provider.dart` / `profile_trail_screen.dart`: `build()` now watches only the filter's `.value` via `.select`, `search()` preserves the previous list via `AsyncLoading.copyWithPrevious`, and the screen passes `skipLoadingOnReload: true` plus a 2px `LinearProgressIndicator` so a reload stays perceptible without blanking the list.

## Task Commits

Each task was committed atomically:

1. **Task 1: Derive the offline filter bounds from the trails on this device** - `74b01eda` (test — combines the pure implementation with its unit tests, matching this phase's prior tdd task convention of a single commit when source and its test suite are authored together)
2. **Task 2: Give the trail filter a device-derived offline fallback and a bounded retry policy** - `936adf06` (feat)
3. **Task 3: Stop the render path from discarding data it already holds** - `e03fd743` (fix)

_Note: TDD tasks may have multiple commits (test -> feat -> refactor); Task 1 here landed as one commit — see Deviations._

## Files Created/Modified
- `app/lib/util/offline_trail_filter_bounds.dart` - `kOfflineTrailFilterValues`, `nextBoundAbove`, `computeOfflineTrailFilterValues` (pure) and `readLocalTrailMetrics` (Store shim)
- `app/test/util/offline_trail_filter_bounds_test.dart` - Unit coverage of the bound arithmetic
- `app/lib/provider/trail/trail_filter_provider.dart` - `buildDefaultTrailFilter`, offline fallback branch, `trailFilterRetry`
- `app/lib/provider/trail/trail_filter_provider.g.dart` - Regenerated (`retry: trailFilterRetry` x2, replacing `retry: null`)
- `app/test/provider/trail/trail_filter_fallback_test.dart` - Coverage for the fallback filter's max==limit invariant (computed values), empty-store slider bounds, retry policy
- `app/lib/provider/profile/profile_trails_provider.dart` - `.select`-based filter watch, `copyWithPrevious` in `search()`
- `app/lib/routes/profile_trail_screen.dart` - `skipLoadingOnReload: true`, inline `LinearProgressIndicator`

## Decisions Made
- Offline fallback bounds are device-derived (per the 2026-08-03 user decision recorded in the plan), not a fixed constant — the constant (`kOfflineTrailFilterValues`) now only applies as the per-axis empty-store floor.
- `_offlineFilterValues()` uses `ref.read(objectBoxProvider)`, not `ref.watch` — `objectBoxProvider` is a `$SyncValueProvider` that never re-emits after `main.dart`'s one-time `overrideWithValue`, and `ref.read` cannot create a dependency edge at all, so this cannot reintroduce the rebuild storm this plan removes.
- A null signed-in account degrades straight to the constant (never an unfiltered on-device read) — a D-13 obligation, since an unfiltered read would let one account's slider disclose another account's downloaded trail lengths.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added `flutter_riverpod` import to `profile_trails_provider.dart` for `.select`**
- **Found during:** Task 3
- **Issue:** `ProviderListenableSelect.select` is defined in `package:riverpod`/`package:flutter_riverpod`, not re-exported by `package:riverpod_annotation` (the file's only pre-existing Riverpod import). Without it, `trailFilterProvider(...).select(...)` failed to resolve (`select` isn't defined for `TrailFilterNotifierProvider`).
- **Fix:** Added `import 'package:flutter_riverpod/flutter_riverpod.dart';`, matching existing precedent in `download_notification_provider.dart`, `trail_download_state_provider.dart`, and `foreground_position_stream_provider.dart`.
- **Files modified:** `app/lib/provider/profile/profile_trails_provider.dart`
- **Verification:** `flutter analyze` clean; `.select(` and `async.value` grep gates hit exactly 1 each.
- **Committed in:** `e03fd743` (Task 3 commit)

**2. [Rule 3 - Blocking] Suppressed `invalid_use_of_internal_member` on the plan-mandated `copyWithPrevious` call**
- **Found during:** Task 3
- **Issue:** `AsyncValue.copyWithPrevious` is `@internal` to the `riverpod` package. Calling it from app code (exactly as the plan specifies) is a documented Riverpod idiom for seamless reload-with-previous-data, but the analyzer flags it as a warning, which `flutter analyze`'s nonzero exit would have failed this task's own `<verify>` gate.
- **Fix:** Added a scoped `// ignore: invalid_use_of_internal_member` with a comment explaining the usage is deliberate, not a misuse.
- **Files modified:** `app/lib/provider/profile/profile_trails_provider.dart`
- **Verification:** `flutter analyze --no-pub lib/provider/profile/profile_trails_provider.dart lib/routes/profile_trail_screen.dart` -> "No issues found!"
- **Committed in:** `e03fd743` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 3 - blocking, both required to make the plan's own literal instructions compile/analyze cleanly)
**Impact on plan:** No scope creep — both fixes are mechanical prerequisites for code the plan explicitly specified. No behavior changed beyond what the plan described.

## Issues Encountered
- Task 1 is tagged `tdd="true"`, but its `<action>` describes creating the pure-function source file and its comprehensive unit test together as one unit (not a red-test-first cycle), matching how this phase's prior `tdd="true"` tasks (36-01 through 36-08) were committed — a single commit per task rather than a split `test(...)` → `feat(...)` pair. Task 1 here landed as one commit (`74b01eda`, typed `test` since the commit is dominated by test-file content and existing phase precedent used a `test(36-06)` commit for a similar source-plus-test unit). No RED-phase failure was separately verified before the implementation existed, since the implementation was authored in the same edit as the test. Functionally the task's full behavior list passes as written; this is a process note rather than a correctness gap.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- REC-06's offline own-trails spinner/retry-storm gap is closed at the source (filter provider no longer throws offline, no longer retries 10x), at the dependency edge (`.select` decouples the list provider from filter churn), and at the render path (`skipLoadingOnReload` + preserved previous list).
- Device re-test (UAT Test 2's remaining manual half) still needed: airplane-mode 60s watch, filter-sheet slider-scale check against on-device trail lengths, and the D-13 account-switch slider-isolation check — none of which `flutter test`/`flutter analyze` can perform in this repo (no ObjectBox test harness).
- No blockers for subsequent Phase 36 plans.

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-03*

## Self-Check: PASSED

All created/modified files verified present on disk; all three task commit hashes (`74b01eda`, `936adf06`, `e03fd743`) verified present in git history.
