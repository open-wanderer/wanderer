---
phase: 03-stats-sheet
verified: 2026-06-13T00:00:00Z
status: human_needed
score: 5/5 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Run the app on a device/emulator with GPS enabled (or simulated). Open a trail, tap Navigate to reach NavigationScreen. Verify the bottom sheet is visible collapsed (showing Time, Distance, Elevation gain). Pan/zoom the map and confirm the bottom-right compass + recenter buttons are not covered by the collapsed sheet. Drag the sheet up to the expanded snap and confirm Elevation loss, Current speed, and Average speed appear. Drag back down and confirm it snaps to collapsed."
    expected: "Sheet snaps between 0.18 (collapsed, 3 stats) and 0.45 (expanded, 6 stats); map + bottom-right controls remain reachable at both snaps."
    why_human: "Layout coverage of map controls by sheet bottom band is a visual/spatial check that cannot be verified by static analysis."
  - test: "While in NavigationScreen, tap the left Elevation-profile button. Confirm the sheet content switches to the elevation chart. Tap the back arrow. Confirm it returns to the stats page. Attempt to swipe the PageView horizontally — confirm nothing happens (button-only)."
    expected: "Button-driven page switch works; horizontal swipe is suppressed (NeverScrollableScrollPhysics)."
    why_human: "PageView gesture behavior (swipe suppression) and visual correctness of the elevation chart require live execution."
  - test: "Tap the center Pause button. Confirm label toggles to Resume, the Time clock freezes, and Distance/Elevation stop increasing with simulated movement. Tap Resume. Confirm Time resumes and Distance/Elevation do not jump."
    expected: "Pause freezes elapsed + accumulation; resume re-anchors references so no jump occurs (Pitfall 6)."
    why_human: "Pause/resume correctness under real GPS motion requires device-level observation."
  - test: "Tap the right Exit button. Confirm navigation closes and you return to the originating trail screen. Confirm there is NO X button in the top-left corner of the NavigationScreen."
    expected: "Exit via sheet button works; old top-left overlay is completely gone."
    why_human: "Visual absence of the removed widget and route-pop behavior require running the app."
  - test: "With simulated or real GPS movement, confirm Distance and Elevation gain/loss increase plausibly (not inflated while stationary). Confirm Current/Average speed show sane km/h values (no NaN/negative displayed as dashes)."
    expected: "Stats display real values: speed shows km/h, no '-' for valid speeds, elevation does not inflate at rest."
    why_human: "Live GPS data quality and UI rendering correctness require observation on a running device."
---

# Phase 3: Stats Sheet Verification Report

**Phase Goal:** Users can see live navigation statistics — distance, elevation, and speed — in a draggable bottom sheet during navigation
**Verified:** 2026-06-13
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A DraggableScrollableSheet is visible at the bottom of NavigationScreen and can be dragged between a collapsed and expanded snap point (STATS-01) | ✓ VERIFIED | `DraggableScrollableSheet` with `initialChildSize: 0.18`, `maxChildSize: 0.45`, `snap: true`, `snapSizes: const [0.18, 0.45]` at navigation_screen.dart:397-403; grep returns 3 (declaration + controller + builder). |
| 2 | Collapsed sheet shows Time elapsed, Distance covered, and Elevation gain (STATS-02, STATS-03, STATS-04) | ✓ VERIFIED | `_buildStatsPage` renders a compact Row with `formatElapsed(stats.elapsed)` (line 476), `formatDistance(stats.distanceMeters)` (line 481), `formatElevation(stats.elevationGainMeters)` (line 486). Labels use `localizations.time`, `localizations.distance`, `localizations.elevation_gain`. |
| 3 | Expanded sheet adds Elevation loss, Current speed, and Average speed (STATS-04, STATS-05) | ✓ VERIFIED | Second Row below a Divider renders `formatElevation(stats.elevationLossMeters)` (line 498), `formatSpeed(stats.currentSpeedKmh)` (line 503), `formatSpeed(stats.averageSpeedKmh)` (line 508). Always present in the layout, revealed naturally by sheet expansion. |
| 4 | A button-driven PageView switches to the reused trail elevation-profile chart and back (STATS-01) | ✓ VERIFIED | `PageView(physics: NeverScrollableScrollPhysics())` with two children: `_buildStatsPage` (page 0) and `_buildElevationPage` (page 1). `_buildElevationPage` renders `ElevationProfile(trail:, gpx:, enableLineTouch:false)` (line 575). Left button calls `_pageController.animateToPage(1)` (line 606-609); back arrow in page 1 calls `_pageController.animateToPage(0)` (line 566-570). |
| 5 | Button row provides Elevation profile, Pause/Resume, and Exit; old top-left exit overlay is removed | ✓ VERIFIED | Button row at `_buildButtonRow`: left = `FontAwesomeIcons.chartArea` → animateToPage(1); center = `FilledButton.icon` with pause/play icon + pause/resume label bound to `navigationStatsProvider.togglePause()`; right = `FontAwesomeIcons.xmark` → `context.pop()`. `grep -c '_buildExitButton'` in navigation_screen.dart returns 0 (completely removed). |

**Score:** 5/5 truths verified

### Provider and Computation Layer (STATS-02/04/05)

| Truth | Status | Evidence |
|-------|--------|----------|
| navigationStatsProvider accumulates cumulative distance | ✓ VERIFIED | `_distance.as(LengthUnit.Meter, _lastPoint!, here)` in `onPosition()` (line 127); Haversine via latlong2 `Distance()`. |
| Elevation gain/loss with 2 m noise-floor threshold | ✓ VERIFIED | `static const _kAltitudeNoiseFloorMeters = 2.0` (declared + used = 2 occurrences); delta.abs() >= threshold guard (line 137). |
| Current speed m/s→km/h with NaN/negative guard | ✓ VERIFIED | `(rawSpeed.isNaN \|\| rawSpeed < 0) ? 0.0 : rawSpeed * 3.6` (line 151). |
| Pause freezes elapsed + accumulation; resume re-anchors | ✓ VERIFIED | `togglePause()` sets `_lastPoint = null; _lastAltitude = null` on resume (lines 187-188); `onPosition()` early-returns when paused (line 110). |
| Timer cancelled via ref.onDispose | ✓ VERIFIED | `ref.onDispose(() => _ticker?.cancel())` (line 99). |
| D-13 honored: no self-subscribed GPS stream | ✓ VERIFIED | `grep -c 'getPositionStream' navigation_stats_provider.dart` = 0. |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/provider/navigation_stats_provider.dart` | NavigationStats freezed state + NavigationStatsNotifier with onPosition/togglePause | ✓ VERIFIED | 196 lines; `class NavigationStats` (2 occurrences — abstract + factory); `void onPosition(Position pos)` (1); `void togglePause()` (1). |
| `app/lib/provider/navigation_stats_provider.g.dart` | build_runner generated provider | ✓ VERIFIED | 5819 bytes, present on disk. |
| `app/lib/provider/navigation_stats_provider.freezed.dart` | build_runner generated freezed | ✓ VERIFIED | 13714 bytes, present on disk. |
| `app/lib/util/format_util.dart` | formatSpeed + formatElapsed added, existing helpers untouched | ✓ VERIFIED | `String formatSpeed` (1), `String formatElapsed` (1), `String formatDistance` (1), `String formatElevation` preserved. |
| `app/lib/routes/navigation_screen.dart` | DraggableScrollableSheet + PageView + button row + GPS-fed stats + old exit removed | ✓ VERIFIED | All greps pass (see Key Link Verification). |
| `app/lib/i18n/app_en.arb` | time, pause, resume, exit_navigation keys | ✓ VERIFIED | All 4 keys present; `speed` key reused for current speed (existing). |
| `app/lib/i18n/app_de.arb` | German equivalents for all new keys | ✓ VERIFIED | pause(1), resume(1), exit_navigation(1), time(1). |
| `app/test/provider/navigation_stats_provider_test.dart` | 11 provider unit tests | ✓ VERIFIED | 190 lines; covers initial state, first-fix anchor, distance (111 m), noise-floor below/above, speed conversion, NaN/negative guard, pause freeze, resume re-anchor (distance + elevation). |
| `app/test/util/format_util_test.dart` | 10 formatter unit tests | ✓ VERIFIED | 63 lines; covers formatSpeed (null/NaN/negative/metric/zero/imperial) and formatElapsed (seconds/minutes/hours/zero). |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `navigation_screen.dart` GPS listener (`_sub`) | `navigationStatsProvider(widget.response).notifier.onPosition` | Method call inside existing `_positionStream.listen` callback | ✓ WIRED | Lines 76-77: `ref.read(navigationStatsProvider(widget.response).notifier).onPosition(pos)` — raw `Position` (not LatLng) passed so altitude+speed are available. |
| `navigation_screen.dart` PageView page 1 | `ElevationProfile` widget | `trailAsync.when data → ElevationProfile(trail:, gpx:)` | ✓ WIRED | Line 575: `ElevationProfile(trail: trail, gpx: gpx, enableLineTouch: false)`. `grep -c 'ElevationProfile('` = 1. |
| `navigation_screen.dart` | `navigationStatsProvider` state | `ref.watch(navigationStatsProvider(widget.response))` | ✓ WIRED | Line 119: `final stats = ref.watch(navigationStatsProvider(widget.response));`. `grep -c 'navigationStatsProvider'` = 3 (watch + two reads). |
| D-13 single GPS stream | Both navigationProvider and navigationStatsProvider | Single `_positionStream.listen` callback feeds both | ✓ WIRED | `grep -c 'getPositionStream' navigation_screen.dart` = 1. Both providers fed from the same callback. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|-------------------|--------|
| `navigation_screen.dart` stats rendering | `stats` (NavigationStats) | `ref.watch(navigationStatsProvider(widget.response))` → `NavigationStatsNotifier` | Yes — accumulated from live GPS `Position` objects via `onPosition(pos)` calls in `_positionStream.listen` | ✓ FLOWING |
| `NavigationStatsNotifier.onPosition` | `distanceMeters`, `elevationGainMeters/Loss`, `currentSpeedKmh` | Real geolocator `Position` (latitude, longitude, altitude, speed) | Yes — Haversine accumulation + noise-floor-gated altitude | ✓ FLOWING |
| `_tick()` method | `elapsed`, `averageSpeedKmh` | `DateTime.now().difference(_start!) - _pausedAccum` | Yes — wall-clock time, not hardcoded | ✓ FLOWING |

### Behavioral Spot-Checks

Step 7b: SKIPPED — no runnable entry point without a device emulator. Behavioral verification delegated to the human verification section.

### Probe Execution

Step 7c: No probe files found in `scripts/*/tests/probe-*.sh`. SKIPPED.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| STATS-01 | 03-01-PLAN, 03-02-PLAN | Bottom DraggableScrollableSheet with collapsed/expanded state and button-driven PageView | ✓ SATISFIED | `DraggableScrollableSheet` (snapSizes [0.18, 0.45]) + `PageView(NeverScrollableScrollPhysics)` in navigation_screen.dart. |
| STATS-02 | 03-01-PLAN, 03-02-PLAN | Distance covered since navigation start | ✓ SATISFIED | `navigationStatsProvider.distanceMeters` accumulated via Haversine in `onPosition`; displayed via `formatDistance(stats.distanceMeters)`. |
| STATS-03 | 03-02-PLAN | Distance covered and ETA — met via Time elapsed + distance covered (ETA deferred per CONTEXT design note) | ✓ SATISFIED | `formatElapsed(stats.elapsed)` + `formatDistance(stats.distanceMeters)` in compact row. ROADMAP note: CONTEXT overrides original page-per-stat wording; elapsed time satisfies STATS-03 intent. |
| STATS-04 | 03-01-PLAN, 03-02-PLAN | Cumulative elevation gain and loss | ✓ SATISFIED | `elevationGainMeters` and `elevationLossMeters` with 2 m noise-floor in provider; displayed via `formatElevation` in both rows. |
| STATS-05 | 03-01-PLAN, 03-02-PLAN | Current GPS speed and average speed | ✓ SATISFIED | `currentSpeedKmh` (m/s→km/h with NaN guard) and `averageSpeedKmh` (distance/elapsed×3.6); displayed via `formatSpeed`. |

**Note on STATS-01 vs REQUIREMENTS.md wording:** The original REQUIREMENTS.md says "horizontal swipe between pages." The ROADMAP.md Phase 3 section contains an explicit design note stating the CONTEXT.md locked design overrides this wording — the realized design uses a DraggableScrollableSheet with button-driven PageView (no horizontal swipe). This is an intentional, documented design decision. The ROADMAP success criteria (SC 1-5) reflect the final design and are all met.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| None detected | — | — | — | — |

All checked files (`navigation_stats_provider.dart`, `format_util.dart`, `navigation_screen.dart`) were scanned for TBD/FIXME/XXX/TODO/HACK/placeholder markers. None found. No stub `return null` / `return []` / `return {}` detected in the stats computation path. No hardcoded-empty props passed to `ElevationProfile` or stat cells.

### Human Verification Required

The automated portion of the verification is complete (5/5 must-haves verified; all key links wired; data flows confirmed statically). The following items require running the app on a device or emulator — they are documented in the PLAN as a blocking `checkpoint:human-verify` gate (Task 3 of 03-02-PLAN.md).

The SUMMARY states Task 3 was "APPROVED" by the user on-device with all 9 verification steps passed. However, this verifier cannot confirm that outcome programmatically. The items below represent the standing human-verification requirement for a fresh confirmation or reliance on the documented SUMMARY approval.

#### 1. Stats Sheet Layout (Map Controls Not Covered)

**Test:** Run the app. Launch navigation. Confirm the collapsed sheet (0.18) is visible with Time / Distance / Elevation gain. Pan and zoom the map; confirm bottom-right compass + recenter buttons remain reachable.
**Expected:** Map fully interactive; bottom-right controls not obscured by the collapsed sheet.
**Why human:** Visual/spatial layout coverage cannot be verified by static analysis.

#### 2. PageView Button Control and Swipe Suppression

**Test:** Tap the left Elevation-profile button. Confirm the sheet switches to the elevation chart. Tap the back arrow. Attempt horizontal swipe — confirm nothing changes.
**Expected:** Button-driven navigation works; swipe does nothing (NeverScrollableScrollPhysics).
**Why human:** Gesture behavior and visual chart rendering require live execution.

#### 3. Pause/Resume Correctness Under Movement

**Test:** Tap Pause. Observe the Time clock freezes and Distance/Elevation stop increasing. Tap Resume. Confirm no distance/elevation jump.
**Expected:** Pause freezes; resume re-anchors cleanly.
**Why human:** GPS-motion-under-pause behavior requires device-level observation.

#### 4. Exit Navigation and Old Button Absence

**Test:** Tap the right Exit button. Confirm the route pops and the originating screen is shown. Confirm there is no X button in the top-left of NavigationScreen.
**Expected:** Route pops; no legacy exit overlay present.
**Why human:** Route-pop behavior and visual absence of the removed widget require runtime.

#### 5. Live Stats Plausibility

**Test:** With real or simulated GPS movement, observe that Distance and Elevation gain/loss increase plausibly. Observe Current/Average speed show positive km/h values (not "-") during movement.
**Expected:** Stats respond to GPS input with sensible values.
**Why human:** Live GPS data quality and live-update rendering require device-level observation.

### Gaps Summary

No automated gaps found. All 5 must-have truths are verified in the codebase. The single outstanding item is the human verification gate (Task 3 of 03-02-PLAN.md), which the SUMMARY reports was approved on-device during execution. Whether the verifier accepts that claim or requires a fresh human confirmation is the only open question.

---

_Verified: 2026-06-13_
_Verifier: Claude (gsd-verifier)_
