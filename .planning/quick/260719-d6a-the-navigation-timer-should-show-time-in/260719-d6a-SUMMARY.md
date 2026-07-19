---
phase: quick-260719-d6a
plan: 01
subsystem: mobile-navigation
tags: [flutter, tracelet, gps, navigation, stats]
dependency-graph:
  requires: []
  provides:
    - "TraceletPositionSource.isMovingStream (native speed-motion transitions)"
    - "NavigationStatsNotifier.setStationary(bool)"
    - "NavigationStats.isStationary"
  affects:
    - app/lib/util/tracelet_position_source.dart
    - app/lib/provider/navigation_stats_provider.dart
    - app/lib/routes/navigation_screen.dart
tech-stack:
  added: []
  patterns:
    - "tracelet's native MotionDetectionMode.speed state machine as the sole motion detector, exposed as a Stream<bool>"
    - "Single derived 'frozen' condition (manual OR stationary) routed through one _applyFrozen() transition helper to avoid double-counting overlapping freeze intervals"
key-files:
  created: []
  modified:
    - app/lib/util/tracelet_position_source.dart
    - app/lib/provider/navigation_stats_provider.dart
    - app/lib/provider/navigation_stats_provider.freezed.dart
    - app/lib/provider/navigation_stats_provider.g.dart
    - app/test/provider/navigation_stats_provider_test.dart
    - app/lib/routes/navigation_screen.dart
decisions:
  - "Reused tracelet's native GPS-speed motion state machine (MotionDetectionMode.speed / onSpeedMotionChange) instead of hand-rolling Haversine/speed-threshold detection — a prior attempt at this same plan built a bespoke detector and was reverted for duplicating functionality the already-installed tracelet ^3.5.0 dependency provides natively."
  - "elapsed becomes time-in-motion for free by freezing the existing pause machinery on stationary — no separate 'motion clock' was introduced."
  - "Generalized _pauseStart into _frozenSince backing a single derived frozen = _manualPaused || _stationary condition (via _applyFrozen), so overlapping manual-pause and stationary intervals are counted once, not twice."
  - "tl.DesiredAccuracy.medium confirmed to exist in the installed tracelet_platform_interface 3.5.7 enum (high/medium/low/veryLow/passive) before use, per the plan's own unverified-value flag."
  - "tl.Tracelet.onSpeedMotionChange confirmed as the real, current API name in tracelet 3.5.7's source (despite a stale doc comment elsewhere in the package referencing onMotionModeChange)."
metrics:
  duration: ~35 min
  completed: 2026-07-19
---

# Phase quick-260719-d6a Plan 01: Time-in-motion navigation timer with tracelet native motion engine Summary

Navigation timer now reflects time spent moving (not total wall-clock elapsed), auto-freezing the timer, GPS tracking, and distance/elevation/speed accumulation when the hiker stops — all driven by tracelet's built-in GPS-speed motion state machine rather than any hand-rolled detector.

## What Was Built

**Task 1 — `app/lib/util/tracelet_position_source.dart`:** Reconfigured `_foregroundConfig()`'s `MotionConfig` from `MotionDetectionMode.accelerometer` (minute-grained `stopTimeout`) to `MotionDetectionMode.speed`, tuned for walking pace rather than tracelet's vehicle-oriented defaults (`speedMovingThreshold: 0.4` m/s, `speedStationaryDelay: 10`s, `stationaryTrackingMode: periodic`, `stationaryPeriodicInterval: 20`s, `stationaryPeriodicAccuracy: DesiredAccuracy.medium`). Added `Stream<bool> get isMovingStream`, sourced from `tl.Tracelet.onSpeedMotionChange`, mapping `SpeedMotionState.moving`/`slowing` to `true` and only a confirmed `stationary` transition to `false`. Subscription lifecycle mirrors the existing `_locationSub` pattern (created in `start()`, cancelled in `dispose()`); a new broadcast `_movingController` is closed alongside the existing `_controller`.

**Task 2 — `app/lib/provider/navigation_stats_provider.dart` (TDD):** Added `isStationary` to the `NavigationStats` freezed class (regenerated via `build_runner`). Generalized the old `_pauseStart` into `_frozenSince`, backing a single derived frozen condition (`_manualPaused || _stationary`) routed through a new `_applyFrozen(bool nowFrozen)` helper shared by `togglePause()` and the new `setStationary(bool)`. This guarantees overlapping manual-pause and stationary intervals are counted once — the accumulator only starts/stops on a real frozen↔unfrozen edge, not on every individual toggle. `onPosition`/`_tick` now check `state.isPaused || state.isStationary` instead of `state.isPaused` alone. Because `elapsed` already subtracts all frozen time, this alone turns `elapsed` into time-in-motion with no separate clock.

Four new tests cover: `setStationary(true)` freezes accumulation and forces speed to 0; `setStationary(false)` re-anchors cleanly with no distance jump; manual `togglePause()` and `setStationary()` compose without one overriding the other's frozen state; and the double-counting-prevention case (manual pause + stationary overlap, single re-anchor on final resume).

**Task 3 — `app/lib/routes/navigation_screen.dart`:** Subscribed to `_positionSource.isMovingStream` in `initState()`, calling `setStationary(!moving)` on the stats notifier for every transition. Subscription is cancelled in `dispose()` alongside the existing GPS subscription. The pause FAB's tooltip/icon now also reflect `stats.isStationary` (shows "Resume"/play icon when auto-paused by the motion engine, even if the user never pressed the button) — `onPressed` still only ever calls `togglePause()`, unchanged.

## Verification

- `flutter analyze lib/util/tracelet_position_source.dart` — clean.
- `dart run build_runner build --delete-conflicting-outputs` — regenerated `navigation_stats_provider.freezed.dart` and `.g.dart` cleanly.
- `flutter test test/provider/navigation_stats_provider_test.dart` — all 15 tests pass (11 pre-existing + 4 new).
- `flutter analyze lib/provider/navigation_stats_provider.dart lib/routes/navigation_screen.dart` — clean except one pre-existing `unnecessary_import` info-level lint on `navigation_stats_provider.dart:3` (unrelated to this plan's changes — confirmed via `git diff` that the import line itself was untouched).
- Manual on-device sanity (per plan, not automated by this executor): start navigation, walk → timer/distance advance; stand still ~10s → timer stops, distance stops, GPS drops to periodic low-power fixes; resume walking → everything resumes. **Not yet performed** — recommend on-device verification before considering this plan fully done.

## Deviations from Plan

None — plan executed exactly as written, including its own flagged unverified value (`tl.DesiredAccuracy.medium`), which was confirmed to exist in the installed `tracelet_platform_interface` 3.5.7 package before use.

## Key API findings confirmed against installed package source (`~/.pub-cache/hosted/pub.dev/tracelet-3.5.7`)

- `tl.DesiredAccuracy` enum: `high, medium, low, veryLow, passive` — `medium` exists exactly as the plan proposed.
- `tl.Tracelet.onSpeedMotionChange(void Function(SpeedMotionEvent) callback)` is the real, current subscribe method (`lib/src/tracelet.dart:2130`), despite a stale doc comment in `speed_motion_event.dart` referencing a nonexistent `Tracelet.onMotionModeChange`/`motionModeChangeStream`.
- `MotionConfig` (`lib/src/models/config.dart:1371`) has all fields the plan referenced: `motionDetectionMode`, `speedMovingThreshold`, `speedStationaryDelay`, `stationaryTrackingMode`, `stationaryPeriodicInterval`, `stationaryPeriodicAccuracy`, `speedWakeConfirmCount` — all present with matching defaults/types.

## TDD Gate Compliance

Task 2 followed the RED → GREEN cycle:
- RED: `b1fb530c` `test(260719-d6a): add failing tests for setStationary + freeze composition` — confirmed compile-failure (undefined `setStationary`/`isStationary`) before any implementation.
- GREEN: `461ad44a` `feat(260719-d6a): auto-pause on stationary + time-in-motion in stats provider` — all 15 tests pass after implementation.
- No REFACTOR commit was needed (no post-GREEN cleanup required).

## Self-Check: PASSED

All modified/created files exist on disk; all four commits are present in `git log`.
