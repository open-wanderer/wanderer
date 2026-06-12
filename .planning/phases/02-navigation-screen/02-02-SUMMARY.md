---
phase: 02-navigation-screen
plan: "02"
subsystem: flutter-mobile
tags: [dart, flutter, riverpod, flutter_map, gps, navigation, go_router, i18n]
dependency_graph:
  requires:
    - "Phase 02 Plan 01 — NavigateResponse freezed model, NavigationState, navigationProvider family notifier"
  provides:
    - "NavigationScreen (ConsumerStatefulWidget) — app/lib/routes/navigation_screen.dart"
    - "/trail/:id/navigate sub-route in router_provider.dart"
    - "5 nav i18n keys in app_en.arb + app_de.arb (navigate, you_have_arrived, reached_end_of_trail, couldnt_start_navigation, in_distance)"
    - "Regenerated app_localizations*.dart with new accessors"
  affects:
    - "Plan 02-03 (entry screens) — consumes the navigate sub-route path and i18n keys"
tech_stack:
  added: []
  patterns:
    - "ConsumerStatefulWidget + TickerProviderStateMixin for animated map controller"
    - "Single broadcast Geolocator.getPositionStream() shared by CurrentLocationLayer and navigation notifier (D-13)"
    - "AlignOnUpdate.always/never on CurrentLocationLayer for follow + heading-up (no camera-fight)"
    - "StreamController<double?>.broadcast() as one-shot recenter trigger for alignPositionStream"
    - "ScaleTransition AnimationController pattern (from map_screen.dart) for recenter button"
    - "context.pop() for exit with no confirmation (NAV-07)"
    - "formatDistance(maneuver.length * 1000) for km->m conversion in banner sub-label"
key_files:
  created:
    - app/lib/routes/navigation_screen.dart
  modified:
    - app/lib/i18n/app_en.arb
    - app/lib/i18n/app_de.arb
    - app/lib/i18n/app_localizations.dart
    - app/lib/i18n/app_localizations_en.dart
    - app/lib/i18n/app_localizations_de.dart
    - app/lib/provider/router_provider.dart
decisions:
  - "Used AlignOnUpdate.always for follow and AlignOnUpdate.never after drag to implement D-09 free-pan — avoids camera-fight vs. manual animateTo per GPS frame (Pitfall 2)"
  - "Only MapEventMoveStart with source==onDrag disables follow; pinch-zoom events do NOT disable follow (D-10)"
  - "mapStyle loaded via AsyncValue.when; returns CircularProgressIndicator during loading instead of empty box (consistent UX)"
  - "Recenter button is a FilledButton.icon with SizedBox.shrink() label to render icon-only (no label text needed per UI-SPEC)"
  - "formatDistance(maneuver.length * 1000) converts Valhalla km lengths to meters for the in_distance banner sub-label"
  - "trailProvider(id) from existing trail_provider.dart used for trail polyline (D-08 — no second API call inside NavigationScreen)"
metrics:
  duration_minutes: 45
  completed_date: "2026-06-12"
  tasks_completed: 2
  files_created: 1
  files_modified: 6
---

# Phase 02 Plan 02: NavigationScreen Summary

**One-liner:** Full-screen `ConsumerStatefulWidget` with a shared-stream follow map (VectorTileLayer → TrailLayer → crimson breadcrumb → CurrentLocationLayer), auto-advancing maneuver banner, compass toggle, exit, and recenter — all wired to the `/trail/:id/navigate` go_router sub-route and five new i18n keys.

## What Was Built

### Task 1: Navigation i18n keys + navigate sub-route

Added five English source keys to `app/lib/i18n/app_en.arb` and matching German translations to `app/lib/i18n/app_de.arb`:
- `"navigate": "Navigate"` / `"Navigieren"`
- `"you_have_arrived": "You've arrived"` / `"Angekommen"`
- `"reached_end_of_trail": "You've reached the end of the trail."` / `"Du hast das Ende des Wegs erreicht."`
- `"couldnt_start_navigation": "Couldn't start navigation. ..."` / German equivalent
- `"in_distance": "in {distance}"` with `@in_distance` placeholder metadata / German equivalent

Ran `flutter gen-l10n` which regenerated `app_localizations.dart`, `app_localizations_en.dart`, `app_localizations_de.dart` with the new accessors.

In `router_provider.dart`:
- Added `import 'package:wanderer/models/navigate_response.dart'`
- Added `import 'package:wanderer/routes/navigation_screen.dart'`
- Added `GoRoute(path: 'navigate', ...)` sibling sub-route inside `GoRoute(path: '/trail/:id', routes: [...])` whose builder casts `state.extra as NavigateResponse` and returns `NavigationScreen(id: trailId, response: response)` (D-04, D-08)

### Task 2: NavigationScreen — full-screen follow map, maneuver banner, controls, breadcrumb

Created `app/lib/routes/navigation_screen.dart` (405 lines) as `class NavigationScreen extends ConsumerStatefulWidget` with:

**State fields:**
- `late final _animatedMapController = AnimatedMapController(vsync: this)` (D-16, mirror map_screen.dart:35)
- `final StreamController<double?> _recenterTrigger` — one-shot stream for `alignPositionStream` (D-09)
- `late final Stream<Position> _positionStream` — single broadcast GPS stream (D-13)
- `StreamSubscription<Position>? _sub` — subscription to feed notifier
- `bool _followEnabled = true` and `bool _headingUp = false`
- `AnimationController _recenterButtonController` + `Animation<double> _recenterButtonScale`

**`initState`:**
- Builds ONE broadcast stream: `Geolocator.getPositionStream().asBroadcastStream()`
- Subscribes it to feed the navigation notifier: `_sub = _positionStream.listen((pos) => ref.read(...).onPosition(LatLng(pos.latitude, pos.longitude)))` (D-13, Pattern 2)
- Initializes recenter button AnimationController with ScaleTransition

**`dispose` (T-02-04 mitigation — Pitfall 6):**
- `_sub?.cancel()`, `_recenterTrigger.close()`, `_recenterButtonController.dispose()`, `_animatedMapController.dispose()`, `super.dispose()`

**`build`:**
- Watches `navigationProvider(widget.response)`, `trailProvider(widget.id)`, `mapStyleProvider`
- Returns `Scaffold(body: styleAsync.when(...))` wrapping a `Stack`

**FlutterMap children (bottom → top per UI-SPEC layering):**
1. `VectorTileLayer` from `mapStyleProvider` style (tiles)
2. `TrailLayer(trail: trail, showWaypoints: false)` — trail polyline #3549BB, 5px (guarded on `trail.expand?.gpx != null`)
3. `PolylineLayer` — crimson breadcrumb `Color(0xFFDC2626)`, `strokeWidth: 3.5` (D-18, D-20)
4. `CurrentLocationLayer` with:
   - `positionStream: const LocationMarkerDataStreamFactory().fromGeolocatorPositionStream(stream: _positionStream)`
   - `alignPositionStream: _recenterTrigger.stream`
   - `alignPositionOnUpdate: _followEnabled ? AlignOnUpdate.always : AlignOnUpdate.never` (D-09)
   - `alignDirectionOnUpdate: _headingUp ? AlignOnUpdate.always : AlignOnUpdate.never` (D-11)

**`MapOptions`:**
- `initialCenter: widget.response.shapeAsLatLng.first`
- `initialZoom: 15` — sensible hiking zoom, NO `minZoom` lock (D-10)
- `maxZoom: 22`
- `onMapEvent`: only `MapEventMoveStart` with `source == MapEventSource.onDrag` disables follow; pinch-zoom events pass through (D-09, D-10)

**Overlay chrome (above map):**
- **Maneuver banner** (top, SafeArea, 24px horizontal): `Card` (surface, radius 16, elevation 4); shows leading `FontAwesomeIcons.locationArrow` icon + `titleLarge` instruction + `bodyMedium` `in_distance(formatDistance(maneuver.length * 1000))`; when arrived, shows completion banner: `titleLarge` `you_have_arrived` + `bodyMedium` `reached_end_of_trail` (D-14, D-15)
- **Exit button** (top-left, 16px): `Material(CircleBorder, canvasColor)` + `InkWell(context.pop())` + `FaIcon(xmark)` (NAV-07)
- **Compass toggle** (top-right, 16px): `MapCompass(hideIfRotatedNorth: false, onPressed: ...)` cycles `_headingUp`; resets rotation to 0 when switching back to north-up (D-11, NAV-05)
- **Recenter button** (bottom-center): `ScaleTransition` wrapping `FilledButton.icon(locationCrosshairs)`; visible only when `!_followEnabled`; clicking calls `_onRecenter()` which sets `_followEnabled = true` and emits to `_recenterTrigger` (D-09)

## Verification Results

```
dart analyze lib/routes/navigation_screen.dart          → No issues found ✓
dart analyze lib/provider/router_provider.dart          → No issues found ✓
dart analyze lib/routes/navigation_screen.dart lib/provider/router_provider.dart → No issues found ✓
flutter gen-l10n                                        → Regenerated cleanly ✓
app_localizations.dart contains couldnt_start_navigation, in_distance, navigate,
  reached_end_of_trail, you_have_arrived                → Verified ✓
app_localizations_en.dart + app_localizations_de.dart  → Verified ✓
router_provider.dart contains path: 'navigate'         → Verified ✓
```

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — no placeholder values, hardcoded empty returns, or TODO markers in any created or modified file.

## Threat Flags

No new threat surface beyond the plan's threat register. All three threats have their mitigations implemented:
- T-02-04 (DoS/GPS drain): `dispose()` cancels `_sub`, closes `_recenterTrigger`, disposes `_animatedMapController` before `super.dispose()`
- T-02-05 (Info disclosure): Breadcrumb is in-memory only (`navState.breadcrumb`), never persisted, discarded on screen exit
- T-02-06 (Tampering): `state.extra as NavigateResponse` cast is safe — entry is gated on successful Plan-03 API response (D-05); this screen is only reachable with a valid `NavigateResponse`

## Self-Check: PASSED
