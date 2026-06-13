---
phase: 02-navigation-screen
plan: "03"
subsystem: flutter-mobile
tags: [dart, flutter, riverpod, go_router, navigation, entry-screens, ux]
dependency_graph:
  requires:
    - "Phase 02 Plan 01 — NavigateResponse freezed model, apiProvider (Dio)"
    - "Phase 02 Plan 02 — NavigationScreen, /trail/:id/navigate route, navigate i18n key, couldnt_start_navigation i18n key"
  provides:
    - "launchNavigation top-level helper — app/lib/util/navigation_launch_util.dart"
    - "_costingFor private helper — same file"
    - "Navigate button (fixed bottom, ConsumerStatefulWidget) — app/lib/routes/trail_detail_screen.dart"
    - "Navigate button (floating over elevation profile) — app/lib/routes/trail_detail_map_screen.dart"
  affects:
    - "End-to-end navigation journey: entry → POST → NavigationScreen → follow → advance → arrive → exit"
tech_stack:
  added: []
  patterns:
    - "Top-level async helper owns costing derivation, waypoint downsampling, Dio POST, parse, push-with-extra, toast-on-failure"
    - "context.mounted guards before every context use after await (Pitfall 5 from 02-RESEARCH.md)"
    - "Caller-owned loading state: caller sets _isLaunching true/false around await launchNavigation()"
    - "ConsumerStatefulWidget for TrailDetailScreen to hold _isLaunching bool"
    - "Positioned.fill + bottom padding (72px) on TrailPanel so scrollable content does not hide behind fixed button (Pitfall 4)"
    - "Dynamic Positioned bottom for map screen button: 258 when elevation visible, 16 otherwise"
key_files:
  created:
    - app/lib/util/navigation_launch_util.dart
  modified:
    - app/lib/routes/trail_detail_screen.dart
    - app/lib/routes/trail_detail_map_screen.dart
decisions:
  - "launchNavigation does not manage loading state — the caller wraps the await and owns the setState, keeping the helper stateless and reusable"
  - "context.mounted guards added at all post-await context accesses (line 108, 120, 127) — not just before context.push — to satisfy use_build_context_synchronously lint"
  - "Floating button in TrailDetailMapScreen uses dynamic Positioned bottom (258 when elevation profile visible, 16 otherwise) so it remains tappable when the profile is hidden"
  - "Pre-existing unnecessary_null_comparison warning on trail.bounds in trail_detail_map_screen.dart left unchanged (out of scope per deviation rules)"
  - "TrailDetailScreen converted from ConsumerWidget to ConsumerStatefulWidget solely to hold the _isLaunching bool — minimal conversion, all existing logic preserved"
metrics:
  duration_minutes: 30
  completed_date: "2026-06-13"
  tasks_completed: 2
  files_created: 1
  files_modified: 2
---

# Phase 02 Plan 03: Navigation Entry Points Summary

**One-liner:** Shared `launchNavigation` helper (costing derivation, ≤2000-point downsampling, Dio POST, NavigateResponse parse, push-with-extra, toast-on-failure) wired into a fixed-bottom `ElevatedButton.icon` on `TrailDetailScreen` and a floating one on `TrailDetailMapScreen`, completing the end-to-end navigation user story.

## What Was Built

### Task 1: Shared launchNavigation helper

Created `app/lib/util/navigation_launch_util.dart` with:

**`_costingFor(String? category) → String`** (private):
- Lowercases the category name and returns `'bicycle'` if it contains `'bike'`, `'cycling'`, or `'bicycle'`; else `'pedestrian'` (Pattern 4 from 02-RESEARCH.md, T-02-09)

**`launchNavigation({BuildContext, WidgetRef, Trail}) → Future<void>`** (top-level export):
1. Guards: `trail.expand?.gpx == null` or `points.length < 2` → error toast + return (V5, T-02-08)
2. Derives costing via `_costingFor(trail.expand?.category?.name)`
3. Builds waypoint list `[{'lat': p.latitude, 'lon': p.longitude}]`; downsamples to ≤2000 by taking every Nth point while always preserving first and last (A4, T-02-07)
4. POSTs to `/valhalla/navigate` via `ref.read(apiProvider)` (baseUrl already includes `/api/v1`)
5. Parses `NavigateResponse.fromJson(res.data)`; guards `maneuvers.isEmpty || shape.isEmpty` → toast + return with mounted guard
6. `if (!context.mounted) return;` before `context.push` (Pitfall 5)
7. `context.push('/trail/${trail.id}/navigate', extra: response)` (D-05)
8. `catch (_)` wraps entire Dio + parse block → mounted guard → error toast (D-07)

`dart analyze lib/util/navigation_launch_util.dart` → No issues found ✓

### Task 2: Navigate buttons on both entry screens

**`trail_detail_screen.dart`** — converted from `ConsumerWidget` to `ConsumerStatefulWidget`:
- Added `bool _isLaunching = false` to `_TrailDetailScreenState`
- Wrapped the `data:` branch in a `Stack` with:
  - `Positioned.fill` + `Padding(bottom: 72)` around `TrailPanel` (Pitfall 4 — prevents last panel content hiding behind button)
  - `Positioned(left:0, right:0, bottom:0)` fixed Navigate button at bottom of SafeArea
- Button: `ElevatedButton.icon`, `SizedBox(width: double.infinity)`, leading `FaIcon(FontAwesomeIcons.locationArrow)` (or `CircularProgressIndicator(strokeWidth: 2)` during loading), `label: Text(localizations.navigate)` (D-01, D-03, NAV-01)
- Tap: `setState(_isLaunching=true) → await launchNavigation(...) → if (mounted) setState(_isLaunching=false)` (D-06)

**`trail_detail_map_screen.dart`** — `ConsumerStatefulWidget` (already was):
- Added `bool _isLaunching = false` to `_TrailDetailMapScreenState`
- Added `navigation_launch_util.dart` import
- Added `Positioned(left:16, right:16, bottom: elevationVisible ? 258 : 16)` floating button sibling in the `Stack`, same D-03 `ElevatedButton.icon` style and loading pattern (D-02, NAV-02)

`dart analyze lib/routes/trail_detail_screen.dart lib/routes/trail_detail_map_screen.dart` → 2 pre-existing warnings (unnecessary_null_comparison on trail.bounds), no new errors ✓

## Verification Results

```
dart analyze lib/util/navigation_launch_util.dart                         → No issues found ✓
dart analyze lib/routes/trail_detail_screen.dart                          → No issues found ✓
dart analyze lib/routes/trail_detail_map_screen.dart                      → 2 pre-existing warnings (out of scope) ✓
dart analyze lib/util/... lib/routes/trail_detail_screen.dart \
  lib/routes/trail_detail_map_screen.dart                                  → 2 pre-existing warnings only ✓

Acceptance criteria:
  trail_detail_screen.dart: ConsumerStatefulWidget                         ✓
  trail_detail_screen.dart: launchNavigation( present                      ✓
  trail_detail_screen.dart: _isLaunching state field                       ✓
  trail_detail_map_screen.dart: launchNavigation( present                  ✓
  trail_detail_map_screen.dart: _isLaunching present                       ✓
  Both files: ElevatedButton.icon with navigate key + locationArrow icon   ✓
  Both files: CircularProgressIndicator(strokeWidth: 2)                    ✓
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical functionality] Added context.mounted guards before all post-await context usages**
- **Found during:** Task 1 — `dart analyze` reported `use_build_context_synchronously` lint at line 111 (empty-response toast after `await api.post`) and would have also flagged the catch block toast
- **Fix:** Added `if (!context.mounted) return;` before the empty-response guard toast (line 108) and before the catch-block toast (line 127), in addition to the planned guard before `context.push` (line 120)
- **Files modified:** `app/lib/util/navigation_launch_util.dart`
- **Commit:** bcc401db

## Known Stubs

None — all logic is fully wired. The helper makes a real Dio POST to `/valhalla/navigate`, parses the real `NavigateResponse`, and calls `context.push` to the real `NavigationScreen`.

## Threat Flags

No new threat surface beyond the plan's threat register. All three planned mitigations are implemented:
- T-02-07 (DoS/oversized payload): `points.length > 2000` downsample guard with every-Nth + first/last preservation
- T-02-08 (Input validation): `gpx == null`, `points.length < 2`, and `maneuvers.isEmpty || shape.isEmpty` guards
- T-02-09 (Tampering/costing): `_costingFor` returns only `'pedestrian'` or `'bicycle'`; server re-validates via Zod enum

## Self-Check: PASSED
