---
phase: 03-stats-sheet
plan: 01
subsystem: ui
tags: [flutter, riverpod, freezed, geolocator, latlong2, gps-stats, navigation]

# Dependency graph
requires:
  - phase: 02-navigation
    provides: "NavigationScreen broadcast GPS stream, navigationProvider pattern, NavigateResponse model"
provides:
  - "navigationStatsProvider — family-keyed @riverpod notifier computing live distance/elevation/speed/elapsed from GPS, fed only via onPosition(Position) (D-13)"
  - "NavigationStats freezed state (elapsed, distanceMeters, elevationGain/LossMeters, current/averageSpeedKmh, isPaused)"
  - "formatSpeed() and formatElapsed() in format_util.dart for the Wave 2 sheet UI"
affects: [03-stats-sheet Wave 2 NavigationScreen sheet UI]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Method-fed Riverpod notifier (onPosition) mirroring navigationProvider — no self-subscribed GPS stream (D-13)"
    - "Provider-owned 1s Timer.periodic for the elapsed clock, cancelled via ref.onDispose (Pitfall 5)"
    - "Altitude noise-floor threshold constant for elevation gain/loss smoothing (Pitfall 2)"
    - "Pause/resume re-anchor of distance & altitude references to avoid jumps (Pitfall 6)"

key-files:
  created:
    - app/lib/provider/navigation_stats_provider.dart
    - app/lib/provider/navigation_stats_provider.g.dart
    - app/lib/provider/navigation_stats_provider.freezed.dart
    - app/test/provider/navigation_stats_provider_test.dart
    - app/test/util/format_util_test.dart
  modified:
    - app/lib/util/format_util.dart
    - app/lib/provider/router_provider.g.dart

key-decisions:
  - "NavigationStats implemented as @freezed (A3) for free copyWith/equality"
  - "Altitude noise floor = 2.0 m (A1) as a tunable named constant"
  - "Average speed derived from accumulated distance / elapsed × 3.6 in the 1s tick"
  - "Pause early-returns in onPosition; resume nulls _lastPoint/_lastAltitude so the paused interval contributes no distance/elevation"

patterns-established:
  - "Pattern 1: stats notifier fed by onPosition(Position) call, never opening a second GPS stream"
  - "Pattern: noise-floor-gated elevation accumulation with reference updated only when threshold crossed"

requirements-completed: [STATS-02, STATS-04, STATS-05]

# Metrics
duration: 8min
completed: 2026-06-13
---

# Phase 3 Plan 01: Stats Computation Layer Summary

**navigationStatsProvider — a family-keyed @riverpod notifier that accumulates Haversine distance, noise-floor-smoothed elevation gain/loss, NaN-guarded current/average speed, and a pause-aware elapsed clock from the existing GPS stream, plus formatSpeed/formatElapsed helpers.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-06-13T13:41:11Z
- **Completed:** 2026-06-13T13:50:01Z
- **Tasks:** 2 (both TDD)
- **Files modified:** 7

## Accomplishments
- `formatSpeed(double? kmh, {unit})` and `formatElapsed(Duration)` added to `format_util.dart` (existing helpers untouched)
- `NavigationStats` freezed state and `NavigationStatsNotifier` (@riverpod, family-keyed on `NavigateResponse`) implemented
- Distance via latlong2 Haversine; elevation gain/loss with a 2 m noise floor; current speed m/s→km/h with NaN/negative guard; average speed from distance/elapsed
- Pause freezes accumulation and forces current speed to 0; resume re-anchors references so no distance/elevation jump occurs
- Provider-owned 1s ticker cancelled on dispose; D-13 honored (zero `getPositionStream` references)
- 21 unit tests pass (10 formatter + 11 provider); `flutter analyze` clean on both new source files

## Task Commits

Each task was committed atomically following the TDD RED→GREEN cycle:

1. **Task 1: formatSpeed/formatElapsed (RED)** - `edabc23a` (test)
2. **Task 1: formatSpeed/formatElapsed (GREEN)** - `99e09502` (feat)
3. **Task 2: navigationStatsProvider + tests (RED)** - `edf2ee8f` (test)
4. **Task 2: navigationStatsProvider codegen (GREEN)** - `14870f91` (feat)

_No refactor commits needed — both implementations passed analyze cleanly._

## Files Created/Modified
- `app/lib/provider/navigation_stats_provider.dart` - NavigationStats freezed state + NavigationStatsNotifier
- `app/lib/provider/navigation_stats_provider.g.dart` / `.freezed.dart` - build_runner generated parts
- `app/lib/util/format_util.dart` - added formatSpeed and formatElapsed
- `app/test/provider/navigation_stats_provider_test.dart` - 11 provider unit tests
- `app/test/util/format_util_test.dart` - 10 formatter unit tests
- `app/lib/provider/router_provider.g.dart` - regenerated hash (codegen sync only, no behavior change)

## Public API for Wave 2

```dart
// Provider symbol (generator strips the "Notifier" suffix):
final navigationStatsProvider = NavigationStatsNotifierFamily._();
// Watch state:    ref.watch(navigationStatsProvider(response))            → NavigationStats
// Read notifier:  ref.read(navigationStatsProvider(response).notifier)    → NavigationStatsNotifier

// NavigationStats fields:
//   Duration elapsed              (default Duration.zero)
//   double   distanceMeters       (default 0)
//   double   elevationGainMeters  (default 0)
//   double   elevationLossMeters  (default 0, stored positive)
//   double   currentSpeedKmh      (default 0)
//   double   averageSpeedKmh      (default 0)
//   bool     isPaused             (default false)

// Methods:
void onPosition(Position pos);   // feed from the existing GPS listener (NOT in build)
void togglePause();              // pause freezes accumulation; resume re-anchors

// Noise-floor constant: static const _kAltitudeNoiseFloorMeters = 2.0;
// togglePause re-anchor: on resume, _pausedAccum += paused interval, then
//   _lastPoint = null; _lastAltitude = null; so the next fix re-anchors.
```

## Decisions Made
- `NavigationStats` as `@freezed` (A3) — `copyWith`/equality for free, matching `navigate_response.dart`.
- Altitude noise floor fixed at `2.0` m (A1) as a single tunable named constant.
- Elapsed clock started on the first `onPosition` fix (Open Question #2 resolution) and driven by an independent 1s `Timer.periodic`, not GPS cadence.
- Average speed computed in `_tick()` from `distanceMeters / elapsed.inSeconds × 3.6`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Corrected provider symbol name in test**
- **Found during:** Task 2 (GREEN)
- **Issue:** Plan/test referenced `navigationStatsNotifierProvider`, but riverpod_generator strips the `Notifier` suffix and emits `navigationStatsProvider`.
- **Fix:** Updated the two test helper references to `navigationStatsProvider`.
- **Files modified:** app/test/provider/navigation_stats_provider_test.dart
- **Verification:** All 11 provider tests pass.
- **Committed in:** `14870f91` (Task 2 GREEN commit)

**2. [Rule 3 - Blocking] Committed regenerated router_provider.g.dart hash**
- **Found during:** Task 2 (build_runner run)
- **Issue:** The required `build_runner build` step regenerated a stale provider hash in `router_provider.g.dart` (single-line `_$routerHash()` change, no behavior change).
- **Fix:** Staged the regenerated file to keep generated artifacts in sync rather than leave a dirty/stale generated file in the tree.
- **Files modified:** app/lib/provider/router_provider.g.dart
- **Verification:** `flutter analyze` clean; unrelated to stats logic.
- **Committed in:** `14870f91`

---

**Total deviations:** 2 auto-fixed (both Rule 3 — blocking/codegen sync).
**Impact on plan:** Both necessary to compile/keep codegen consistent. No scope creep; the planned behavior is unchanged.

## Issues Encountered
- **Pre-existing, out-of-scope test failures:** A full `flutter test` regression run surfaced 2 failing tests in `test/models/feed_item_test.dart` originating in `trail.g.dart`/`list.g.dart` — files this plan never touched. Logged to `deferred-items.md`; not fixed per the scope boundary. The two new test files for this plan pass 21/21.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Stats logic layer is complete and unit-tested; Wave 2 can wire `navigationStatsProvider(response).onPosition(pos)` into the existing GPS listener and `togglePause()` to the Pause button.
- `formatSpeed`/`formatElapsed` ready for the sheet's stat rows.
- No blockers for the Wave 2 sheet UI.

---
*Phase: 03-stats-sheet*
*Completed: 2026-06-13*
