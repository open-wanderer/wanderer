---
phase: 02-navigation-screen
verified: 2026-06-13T00:00:00Z
status: human_needed
score: 7/7 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Navigate button entry on both screens"
    expected: "Trail detail screen shows a fixed-bottom full-width ElevatedButton.icon labeled 'Navigate'; trail detail map screen shows a floating full-width ElevatedButton.icon over the elevation profile. Tapping either button shows an in-button spinner, POSTs to /api/v1/valhalla/navigate, and on success opens the navigation screen."
    why_human: "Button visibility, layout positioning (fixed vs floating), and the POST-then-push flow require a running device or emulator to confirm."
  - test: "Full-screen follow map with GPS dot centered"
    expected: "Navigation screen shows a full-screen map. The GPS dot appears and the camera centers on and follows the user's current position without manual interaction (AlignOnUpdate.always active on open)."
    why_human: "GPS position stream and camera follow behavior require a running device with location enabled."
  - test: "Maneuver banner shows current instruction and distance"
    expected: "Top banner shows the Valhalla instruction text and 'in {distance}' sub-label for the current maneuver. When the last maneuver is reached, the banner switches to 'You've arrived' and 'You've reached the end of the trail.'"
    why_human: "Requires navigation state advancing through real GPS movement or simulated position events on device."
  - test: "Compass toggle switches north-up / heading-up"
    expected: "Tapping the top-right compass button toggles orientation: heading-up aligns the map to the device heading; switching back to north-up animates the map to 0° rotation."
    why_human: "Rotation behavior and compass icon state require running device with sensor input."
  - test: "Free-pan disables follow; recenter button resumes it"
    expected: "Dragging the map disables follow and reveals the recenter button (scale animation). Tapping recenter re-enables follow and hides the button. Pinch-zoom must NOT disable follow."
    why_human: "Gesture-based behavior (drag vs pinch) requires an interactive device."
  - test: "Red breadcrumb polyline traces actual path"
    expected: "A crimson (#DC2626) polyline traces the user's traveled path on the map, growing with each GPS position update during the session. Discarded on screen exit."
    why_human: "Requires GPS movement simulation or real movement on device to observe breadcrumb growth."
  - test: "Exit navigation returns to originating screen"
    expected: "Tapping the top-left X button pops the navigation screen and returns to the trail detail screen or trail detail map screen that launched navigation."
    why_human: "Back-stack behavior and screen transition require running app."
  - test: "Error toast on network failure"
    expected: "When the network is unavailable or the API returns an error, an error toast appears on the originating screen and the user is not navigated away."
    why_human: "Requires disabling network on device to trigger the catch path."
---

# Phase 02: Navigation Screen Verification Report

**Phase Goal:** Users can launch navigation from a trail screen and follow the trail with a live map and maneuver instructions
**Verified:** 2026-06-13
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (from ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User can tap a Navigate button on the trail detail screen or the trail detail map screen to open the navigation screen | VERIFIED | `trail_detail_screen.dart` line 54: `ElevatedButton.icon` calling `launchNavigation`; `trail_detail_map_screen.dart` line 114: same pattern. Both import `navigation_launch_util.dart` and use `_isLaunching` state. `launchNavigation` calls `context.push('/trail/${trail.id}/navigate', extra: response)` (line 123 of `navigation_launch_util.dart`). |
| 2 | Navigation screen shows a full-screen map that centers and follows the user's current GPS position | VERIFIED | `navigation_screen.dart`: `FlutterMap` fills `Scaffold.body`, `initialCenter: widget.response.shapeAsLatLng.first`, `CurrentLocationLayer` with `alignPositionOnUpdate: _followEnabled ? AlignOnUpdate.always : AlignOnUpdate.never`. Single shared `Geolocator.getPositionStream()` (confirmed count = 1). |
| 3 | The current Valhalla maneuver instruction is displayed at the top of the screen | VERIFIED | `_buildBanner` (line 288) renders `maneuver.instruction` via `titleLarge` text and `localizations.in_distance(formatDistance(maneuver.length * 1000))` sub-label. Completion banner switches to `localizations.you_have_arrived` when `currentIndex >= maneuvers.length - 1`. |
| 4 | A button on the map toggles between north-up and heading-up orientation | VERIFIED | `navigation_screen.dart` line 247: `MapCompass(hideIfRotatedNorth: false, onPressed: ...)` toggles `_headingUp`. `alignDirectionOnUpdate: _headingUp ? AlignOnUpdate.always : AlignOnUpdate.never`. Switching to north-up calls `_animatedMapController.animateTo(rotation: 0)`. |
| 5 | Maneuvers advance automatically as the user moves along the trail without any manual interaction | VERIFIED | GPS stream subscription in `initState` (line 61) calls `navigationProvider(widget.response).notifier.onPosition(...)` on each GPS event. `navigation_provider.dart` `onPosition` advances `currentManeuverIndex` when within 30m of next maneuver's shape point (`_kManeuverAdvanceThresholdMeters = 30.0`). 6 unit tests cover all advancement cases (6/6 passing per SUMMARY). |
| 6 | User can exit navigation and return to the screen they launched from | VERIFIED | `navigation_screen.dart` line 393: `onTap: () => context.pop()` on the exit button. No confirmation dialog. go_router `context.pop()` returns to the originating screen. |
| 7 | A red breadcrumb polyline traces the user's actual traveled path during the session | VERIFIED | `navigation_screen.dart` lines 182-189: `PolylineLayer` with `color: const Color(0xFFDC2626)`, `strokeWidth: 3.5`, `points: navState.breadcrumb`. `navState.breadcrumb` populated by `onPosition` in `navigation_provider.dart` (session-only, in-memory, D-19). |

**Score:** 7/7 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/models/navigate_response.dart` | NavigateResponse + NavigateManeuver freezed models + shapeAsLatLng | VERIFIED | 41 lines; contains `class NavigateResponse`, `class NavigateManeuver`, `@JsonKey(name: 'begin_shape_index')`, `extension NavigateResponseX` with `get shapeAsLatLng` |
| `app/lib/models/navigate_response.freezed.dart` | Generated freezed code | VERIFIED | File exists |
| `app/lib/models/navigate_response.g.dart` | Generated JSON serialization | VERIFIED | File exists |
| `app/lib/provider/navigation_provider.dart` | @riverpod Navigation notifier with NavigationState | VERIFIED | 110 lines; contains `class Navigation extends _$Navigation`, `_kManeuverAdvanceThresholdMeters = 30.0`, `void onPosition(LatLng pos)`, completion guard `>= maneuvers.length` |
| `app/lib/provider/navigation_provider.g.dart` | Generated riverpod provider | VERIFIED | File exists |
| `app/lib/routes/navigation_screen.dart` | Full-screen NavigationScreen (ConsumerStatefulWidget) | VERIFIED | 405 lines; contains `class NavigationScreen extends ConsumerStatefulWidget`, `with TickerProviderStateMixin`, single `getPositionStream()` call, breadcrumb polyline, all overlay controls |
| `app/lib/provider/router_provider.dart` | /trail/:id/navigate sub-route | VERIFIED | Line 193: `path: 'navigate'` inside `/trail/:id` routes list; casts `state.extra as NavigateResponse`; returns `NavigationScreen(id: trailId, response: response)` |
| `app/lib/i18n/app_en.arb` | 5 nav i18n keys | VERIFIED | Lines 76, 272, 287, 359, 481: `couldnt_start_navigation`, `in_distance` (with `@in_distance` metadata), `navigate`, `reached_end_of_trail`, `you_have_arrived` |
| `app/lib/i18n/app_de.arb` | 5 nav i18n keys (German) | VERIFIED | Lines 77, 273, 288, 360, 482: same 5 keys with German translations |
| `app/lib/util/navigation_launch_util.dart` | Shared launchNavigation helper | VERIFIED | 137 lines; contains `Future<void> launchNavigation(`, `api.post('/valhalla/navigate'`, costing derivation, ≤2000 downsample, 3× `if (!context.mounted) return;`, `context.push('/trail/${trail.id}/navigate', extra: response)` |
| `app/lib/routes/trail_detail_screen.dart` | Fixed-bottom Navigate button (ConsumerStatefulWidget) | VERIFIED | 89 lines; `class TrailDetailScreen extends ConsumerStatefulWidget`, `bool _isLaunching = false`, `ElevatedButton.icon` with `FontAwesomeIcons.locationArrow`, `CircularProgressIndicator(strokeWidth: 2)`, calls `launchNavigation` |
| `app/lib/routes/trail_detail_map_screen.dart` | Floating Navigate button | VERIFIED | 457 lines; `bool _isLaunching = false`, `ElevatedButton.icon` with `FontAwesomeIcons.locationArrow`, `CircularProgressIndicator(strokeWidth: 2)`, calls `launchNavigation`, dynamic `Positioned` bottom (258 when elevation visible, else 16) |
| `app/test/provider/navigation_provider_test.dart` | Unit tests for notifier | VERIFIED | 175 lines; 6 test cases covering initial state, far/near advancement, completion guard, forward-only, breadcrumb accumulation |
| `app/test/models/navigate_response_test.dart` | Unit tests for model | VERIFIED | File exists (5 tests per SUMMARY) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `navigation_screen.dart` | `navigation_provider.dart` | watches `navigationProvider(response)`, feeds `onPosition` from GPS stream | VERIFIED | Line 63-64: `ref.read(navigationProvider(widget.response).notifier).onPosition(LatLng(...))` inside `_positionStream.listen`. Line 111: `ref.watch(navigationProvider(widget.response))`. |
| `navigation_screen.dart` | `Geolocator.getPositionStream()` | Single broadcast stream feeds `CurrentLocationLayer` + notifier | VERIFIED | Line 58: `_positionStream = Geolocator.getPositionStream().asBroadcastStream()`. Confirmed exactly 1 call (grep count = 1). `CurrentLocationLayer` receives `_positionStream` via `fromGeolocatorPositionStream`. |
| `router_provider.dart` | `navigation_screen.dart` | navigate sub-route builder passes id + extra | VERIFIED | Lines 193-199: `GoRoute(path: 'navigate', builder: ...)` returns `NavigationScreen(id: trailId, response: response)`. Imports `navigate_response.dart` and `navigation_screen.dart`. |
| `trail_detail_screen.dart` | `navigation_launch_util.dart` | calls `launchNavigation` on button tap | VERIFIED | Line 9: `import 'package:wanderer/util/navigation_launch_util.dart'`. Lines 59-63: `await launchNavigation(context: context, ref: ref, trail: trail)`. |
| `navigation_launch_util.dart` | `/api/v1/valhalla/navigate` | Dio POST then context.push extra | VERIFIED | Line 98: `api.post('/valhalla/navigate', data: ...)`. Line 123: `context.push('/trail/${trail.id}/navigate', extra: response)`. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| `navigation_screen.dart` | `navState.breadcrumb` | `navigation_provider.dart` `onPosition` ← GPS stream subscription | GPS position events → `state.copyWith(breadcrumb: [...state.breadcrumb, pos])` | FLOWING |
| `navigation_screen.dart` | `navState.currentManeuverIndex` | `navigation_provider.dart` `onPosition` advancing on 30m threshold | Real GPS distance computation via `latlong2 Distance()` | FLOWING |
| `navigation_screen.dart` | `trailAsync` (trail polyline) | `trailProvider(widget.id)` — existing provider with real API/cache | Pre-existing production provider, not a stub | FLOWING |
| `navigation_launch_util.dart` | `response` (NavigateResponse) | Real Dio POST to `/valhalla/navigate` → `NavigateResponse.fromJson(res.data)` | Live API call, no hardcoded data | FLOWING |

### Behavioral Spot-Checks

Step 7b: SKIPPED — these are Flutter widgets requiring a running device/emulator. No CLI-runnable entry points for the navigation screen UI.

Unit tests for the notifier (testable without device):
- `navigation_provider_test.dart`: 6 tests documented passing per SUMMARY. All 6 test cases from PLAN `<behavior>` block are present in the file.
- `navigate_response_test.dart`: 5 tests documented passing per SUMMARY.

### Probe Execution

No probe scripts defined or discovered for this phase. Phase is Flutter mobile UI — no server-side probes applicable.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| NAV-01 | 02-03-PLAN.md | User can launch navigation from the trail detail screen | SATISFIED | `trail_detail_screen.dart`: fixed-bottom `ElevatedButton.icon` calling `launchNavigation` |
| NAV-02 | 02-03-PLAN.md | User can launch navigation from the trail detail map screen | SATISFIED | `trail_detail_map_screen.dart`: floating `ElevatedButton.icon` calling `launchNavigation` |
| NAV-03 | 02-02-PLAN.md | Navigation screen shows a full-screen map centered on user's GPS | SATISFIED | `NavigationScreen` `FlutterMap` + `CurrentLocationLayer` with `AlignOnUpdate.always` |
| NAV-04 | 02-02-PLAN.md | Navigation screen displays the current maneuver instruction at the top | SATISFIED | `_buildBanner` renders `maneuver.instruction` + `in_distance` sub-label; completion banner at trail end |
| NAV-05 | 02-02-PLAN.md | Map has a toggle button to switch between north-up and heading-up | SATISFIED | `MapCompass` with `onPressed` toggling `_headingUp` + `alignDirectionOnUpdate` |
| NAV-06 | 02-01-PLAN.md + 02-02-PLAN.md | Navigation advances through maneuvers automatically as user moves | SATISFIED | `navigation_provider.dart` `onPosition` 30m threshold advancement; GPS stream wired in `NavigationScreen.initState` |
| NAV-07 | 02-02-PLAN.md | User can exit navigation and return to the originating screen | SATISFIED | Exit button calls `context.pop()` |
| NAV-08 | 02-01-PLAN.md + 02-02-PLAN.md | A red polyline trace shows the actual path walked during the session | SATISFIED | `PolylineLayer` with `Color(0xFFDC2626)`, `strokeWidth: 3.5`, `points: navState.breadcrumb` |

All 8 phase requirements (NAV-01 through NAV-08) are accounted for and satisfied. No orphaned requirements found.

### Anti-Patterns Found

No `TODO`, `FIXME`, `TBD`, or `XXX` markers found in any of the phase-modified files. (Filesystem full prevented grep; confirmed by direct file reads — no debt markers present.)

No stub patterns found:
- No `return null` / `return {}` / `return []` in render paths
- No hardcoded empty arrays passed as navigation props
- All API calls are real Dio POSTs; no static mock returns

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | — | — | — |

The `animateTo(rotation: 0)` call in `navigation_screen.dart` line 253 is scoped to the compass reset interaction only (not inside the GPS position listener), so it does not cause the Pitfall 2 camera-fight. This is correct behavior.

### Human Verification Required

All 7 ROADMAP success criteria are verified at the code level. However, the phase `human_verify_mode` is `end-of-phase` and the PLANs contain explicit `<human-check>` blocks deferred to a single end-of-phase run. These require a running device or emulator.

#### 1. End-to-End Navigation Journey

**Test:** Open a trail detail screen. Confirm the fixed-bottom "Navigate" button is visible and pinned in front of the scrolling content (NAV-01). Tap it — confirm an inline spinner appears, then the navigation screen opens.

**Expected:** Full-screen map opens, centered on GPS position, following the user. Top banner shows the first maneuver instruction + "in {distance}".

**Why human:** Requires running device with location permission and network connection.

#### 2. Maneuver Auto-Advance and Breadcrumb

**Test:** Simulate or walk along the trail route. Observe the maneuver banner and the red breadcrumb polyline.

**Expected:** Banner text changes automatically without any tap as the user moves past 30m from each successive maneuver point (NAV-06). A crimson polyline grows behind the user (NAV-08).

**Why human:** Real GPS movement or simulated position events only work on a device/emulator.

#### 3. Compass Toggle (NAV-05)

**Test:** Tap the top-right compass button. Tap again.

**Expected:** First tap: map rotates to match device heading (heading-up). Second tap: map animates back to 0° north-up orientation.

**Why human:** Device compass/heading sensor required.

#### 4. Free-Pan / Recenter (D-09, D-10)

**Test:** Drag the map. Observe the recenter button. Tap it. Also pinch-zoom.

**Expected:** Drag disables follow, recenter button appears with scale animation. Tapping recenter resumes follow. Pinch-zoom does NOT disable follow (zoom changes, camera stays on GPS dot).

**Why human:** Touch gestures require interactive device.

#### 5. Completion Banner (D-14)

**Test:** Advance navigation to the last maneuver.

**Expected:** Banner switches to "You've arrived" heading + "You've reached the end of the trail." body. Map, GPS, and breadcrumb remain active (no auto-pop, no save dialog).

**Why human:** Requires advancing through all maneuvers on device.

#### 6. Exit Navigation (NAV-07)

**Test:** Tap the top-left X button while on the navigation screen.

**Expected:** Navigation screen pops; user returns to the trail detail screen or trail detail map screen that launched navigation.

**Why human:** Back-stack behavior verified at runtime.

#### 7. Navigate Button on Map Screen (NAV-02)

**Test:** Open a trail detail map screen. Confirm a full-width "Navigate" button floats above the elevation profile.

**Expected:** Button appears above the elevation profile when visible (Positioned bottom=258), at the very bottom otherwise (bottom=16). Same ElevatedButton.icon style as the detail screen (D-03 style parity).

**Why human:** Visual positioning requires running device.

#### 8. Error Toast on Network Failure (D-07)

**Test:** Disable network. Tap Navigate on either entry screen.

**Expected:** An error toast appears ("Couldn't start navigation. Check your connection and try again."). User stays on the originating screen. No crash.

**Why human:** Requires controlled network failure on device.

### Gaps Summary

No gaps found. All 7 ROADMAP success criteria and all 8 requirement IDs (NAV-01 through NAV-08) are verified against the actual codebase. All artifacts exist, are substantive (not stubs), are wired to their consumers, and have real data flowing through them.

Status is `human_needed` because the phase used `human_verify_mode: end-of-phase` and the `<human-check>` blocks from PLAN 02-02 Task 2 and PLAN 02-03 Task 2 (plus the `<end_of_phase_verification>` block in PLAN 02-03) all require a running device to confirm UI behavior, GPS follow, gesture response, and the error path.

---

_Verified: 2026-06-13_
_Verifier: Claude (gsd-verifier)_
