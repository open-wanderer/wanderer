---
phase: 02-navigation-screen
plan: "01"
subsystem: flutter-mobile
tags: [dart, flutter, riverpod, freezed, navigation, model, notifier, tdd]
dependency_graph:
  requires:
    - "Phase 01 (backend API) — POST /api/v1/valhalla/navigate endpoint and NavigateResponse JSON contract"
  provides:
    - "NavigateResponse freezed model — app/lib/models/navigate_response.dart"
    - "NavigateManeuver freezed model — same file"
    - "NavigateResponseX.shapeAsLatLng extension — same file"
    - "NavigationState immutable class — app/lib/provider/navigation_provider.dart"
    - "Navigation @riverpod notifier — same file"
    - "Navigation.onPosition(LatLng) — advancement + breadcrumb method"
    - "navigationProvider(response) — generated family provider"
  affects:
    - "Plan 02-02 (NavigationScreen) — consumes NavigateResponse model and navigationProvider"
    - "Plan 02-03 (entry screens) — passes NavigateResponse as go_router extra"
tech_stack:
  added: []
  patterns:
    - "freezed v3 abstract class with @JsonKey(name: ...) for snake_case deserialization"
    - "Extension on freezed class (NavigateResponseX) for shapeAsLatLng — freezed v3 forbids methods in main body"
    - "@riverpod class notifier family-keyed on NavigateResponse (D-17 testability)"
    - "Plain immutable NavigationState with manual copyWith (no freezed in provider file — avoids double-codegen per plan)"
    - "latlong2 const Distance().as(LengthUnit.Meter, a, b) for 30m threshold geometry"
key_files:
  created:
    - app/lib/models/navigate_response.dart
    - app/lib/models/navigate_response.freezed.dart
    - app/lib/models/navigate_response.g.dart
    - app/lib/provider/navigation_provider.dart
    - app/lib/provider/navigation_provider.g.dart
    - app/test/models/navigate_response_test.dart
    - app/test/provider/navigation_provider_test.dart
  modified: []
decisions:
  - "Used plain immutable NavigationState class (not freezed) to keep provider file self-contained without a second build_runner annotation target"
  - "Used NavigateResponseX extension for shapeAsLatLng to comply with freezed v3 no-methods-in-body constraint"
  - "Coordinate order [lat, lon] confirmed from Phase-1 endpoint +server.ts lines 120-122 and baked into shapeAsLatLng with JSDoc"
  - "NavigationState breadcrumb is const [] at build time, satisfying D-19 session-only guarantee"
metrics:
  duration_minutes: 25
  completed_date: "2026-06-12"
  tasks_completed: 2
  files_created: 7
  files_modified: 0
---

# Phase 02 Plan 01: NavigateResponse Model + Navigation Notifier Summary

**One-liner:** Freezed `NavigateResponse`/`NavigateManeuver` models mirroring the Phase-1 Valhalla API contract, plus a `@riverpod` `Navigation` family notifier holding `currentManeuverIndex` + session-only breadcrumb with 30m threshold advancement and full unit test coverage.

## What Was Built

### Task 1: NavigateResponse + NavigateManeuver freezed model (TDD)

Created `app/lib/models/navigate_response.dart` with:
- `@freezed abstract class NavigateManeuver` with `instruction`, `length`, `beginShapeIndex` (`@JsonKey(name: 'begin_shape_index')`), `bearing` (default 0.0)
- `@freezed abstract class NavigateResponse` with `maneuvers: List<NavigateManeuver>`, `shape: List<List<double>>`
- `extension NavigateResponseX on NavigateResponse` providing `get shapeAsLatLng` converting `shape` to `List<LatLng>` in confirmed `[lat, lon]` order
- Generated `navigate_response.freezed.dart` and `navigate_response.g.dart` via build_runner
- 5 unit tests in `app/test/models/navigate_response_test.dart` (all passing)

### Task 2: Navigation notifier + unit tests (TDD)

Created `app/lib/provider/navigation_provider.dart` with:
- `NavigationState` plain-immutable class with `response`, `currentManeuverIndex`, `breadcrumb` fields and manual `copyWith`
- `@riverpod class Navigation extends _$Navigation` — family-keyed on `NavigateResponse`
- `static const _kManeuverAdvanceThresholdMeters = 30.0` (D-12, NAV-06)
- `void onPosition(LatLng pos)` implementing:
  - Breadcrumb append via `state.copyWith(breadcrumb: [...state.breadcrumb, pos])` (D-18, NAV-08)
  - Completion guard: `if (next >= state.response.maneuvers.length) return;` (D-14, T-02-01)
  - `beginShapeIndex` clamped to `shape.length - 1` (T-02-01, Pitfall 3)
  - Distance check via `_distance.as(LengthUnit.Meter, pos, targetLatLng)` (latlong2)
  - Forward-only advancement: never decrements (T-02-02)
- Generated `navigation_provider.g.dart` via build_runner
- 6 unit tests in `app/test/provider/navigation_provider_test.dart` (all passing)

## Verification Results

```
dart run build_runner build --delete-conflicting-outputs  → Built; 0 errors ✓
flutter test test/provider/navigation_provider_test.dart  → 6/6 passed ✓
flutter test test/models/navigate_response_test.dart      → 5/5 passed ✓
dart analyze lib/models/navigate_response.dart lib/provider/navigation_provider.dart → No issues found ✓
```

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — no placeholder values, hardcoded empty returns, or TODO markers in any created file.

## Threat Flags

No new threat surface beyond the plan's threat register. All three threats have their mitigations implemented:
- T-02-01 (DoS/RangeError): `beginShapeIndex` clamped + `next >= maneuvers.length` guard
- T-02-02 (Tampering/rewind): Forward-only advancement in `onPosition`
- T-02-03 (Info disclosure): `breadcrumb` is in-memory only, session-scoped, never persisted

## TDD Gate Compliance

Both tasks used `tdd="true"`. Tests were written first (RED), implementation written to pass (GREEN), no REFACTOR needed. All test cases from `<behavior>` sections are covered.

## Self-Check: PASSED
