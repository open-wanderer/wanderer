---
phase: quick-260619-okw
status: complete
---

# Quick Task 260619-okw: Use tracelet for background location tracking

## What was done

Replaced `Geolocator.getPositionStream` in the navigation screen with tracelet 3.5.0.

**Task 1 (pre-approved):** User confirmed tracelet legitimacy by referencing pub.dev URL directly.

**Task 2:** Added `tracelet: ^3.5.0` to `app/pubspec.yaml` with `dependency_overrides: meta: ^1.18.0` to resolve the flutter_test SDK pin conflict. Created `app/lib/util/tracelet_position_source.dart` — a `TraceletPositionSource` adapter that bridges `Tracelet.onLocation` into a broadcast `Stream<geo.Position>` with high accuracy and 5 m distance filter.

**Task 3:** Rewired `navigation_screen.dart` to construct `TraceletPositionSource`, start it in `initState` via `unawaited(_positionSource.start())`, source `_positionStream` from the adapter, and dispose it in `dispose()`. Removed `_buildLocationSettings()`, direct `Geolocator.getPositionStream` call, and unused `dart:io` import. All three consumers — `navigationProvider.onPosition`, `navigationStatsProvider.onPosition`, and `CurrentLocationLayer` — consume the same broadcast stream unchanged.

## Commits

- `4d70bd85` — feat(quick-260619-okw): use tracelet for background location tracking in navigation

## Files changed

- `app/pubspec.yaml` — tracelet ^3.5.0 added; meta override ^1.18.0
- `app/pubspec.lock` — resolved tracelet 3.5.0, meta 1.18.3
- `app/lib/util/tracelet_position_source.dart` — new adapter (created)
- `app/lib/routes/navigation_screen.dart` — rewired to TraceletPositionSource
