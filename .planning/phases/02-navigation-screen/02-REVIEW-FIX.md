---
phase: 02-navigation-screen
fixed_at: 2026-06-13T00:00:00Z
status: all_fixed
fix_scope: critical_warning
findings_in_scope: 7
fixed: 7
skipped: 0
iteration: 1
---

# Phase 02 Code Review Fix Report

All 7 in-scope findings (3 Critical, 4 Warning) fixed and committed atomically.

## Fixed Findings

### CR-01 — Unsafe router cast guarded (commit 7cbf8559)
**File:** `app/lib/provider/router_provider.dart:197`
Replaced bare `state.extra as NavigateResponse` with `is! NavigateResponse` guard. Returns `TrailDetailScreen` as fallback on deep-link or state restoration when `extra` is null or wrong type.

### CR-02 — Auth `.requireValue!` replaced with `.valueOrNull` (commit 6b018a38)
**File:** `app/lib/routes/trail_detail_map_screen.dart:65`
Changed `.requireValue!` to `.valueOrNull`. Added `user != null` guard around `PhotoCollage` usage so auth loading/error states no longer crash the screen.

### CR-03 — `shapeAsLatLng` guarded against short entries (commit aab9e9a3)
**File:** `app/lib/models/navigate_response.dart:39`
Added `.where((p) => p.length >= 2)` before mapping shape list to `LatLng`. Malformed shape entries from upstream no longer cause a `RangeError`.

### WR-01 — Downsample step fixed to guarantee ≤ 2000 points (commit 7f57e675)
**File:** `app/lib/util/navigation_launch_util.dart`
Changed divisor from `2000` to `1999`, moved first/last point handling outside the loop, and added post-loop last-point deduplication to prevent the off-by-one that could emit 2001 points.

### WR-02 — GPS stream `onError` handler added (commit 7dc48b0b)
**File:** `app/lib/routes/navigation_screen.dart`
Added `onError` callback to the GPS stream subscription in `NavigationScreen`. Errors now show an error toast and pop the screen rather than silently killing position updates.

### WR-03 — `AnimationController` moved to field declaration (commit 7c242175)
**File:** `app/lib/routes/navigation_screen.dart`
Moved `_recenterButtonController` and `_recenterButtonScale` initialization from `initState` to inline field declarations. Eliminates the `LateInitializationError` risk if `initState` throws before assignment.

### WR-04 — `ScrollController` moved to state field with dispose (commit c9f5562b)
**File:** `app/lib/routes/trail_detail_screen.dart`
Added `late final _scrollController = ScrollController()` as a state field. Added `dispose()` override calling `_scrollController.dispose()`. Removed the inline `ScrollController()` allocation from `build()` that leaked a new controller on every rebuild.

## Skipped Findings

None — all 7 in-scope findings fixed.

## Verification

- `flutter test test/models/navigate_response_test.dart test/provider/navigation_provider_test.dart` → 11/11 passed ✓
