# Phase 2: Navigation Screen - Pattern Map

**Mapped:** 2026-06-12
**Files analyzed:** 7 (3 new, 4 modified)
**Analogs found:** 7 / 7

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `app/lib/routes/navigation_screen.dart` (NEW) | screen (ConsumerStatefulWidget) | streaming (GPS) + request-response | `app/lib/routes/map_screen.dart` | exact (full-screen FlutterMap, ConsumerStatefulWidget, AnimatedMapController, CurrentLocationLayer, MapCompass, dispose) |
| `app/lib/provider/navigation_provider.dart` (NEW) | provider (notifier) | event-driven (per-position) + transform | `app/lib/provider/toast_provider.dart` (notifier shape) + `app/lib/provider/trail/trail_provider.dart` (`@riverpod` family) | role-match |
| `app/lib/models/navigate_response.dart` (NEW) | model (freezed) | transform (JSON deser) | `app/lib/models/trail.dart` (freezed + JsonKey) | role-match |
| `app/lib/routes/trail_detail_screen.dart` (MODIFY) | screen (ConsumerWidget) | request-response (Dio POST) | self + `app/lib/routes/login_screen.dart` (Toast + loading) | exact |
| `app/lib/routes/trail_detail_map_screen.dart` (MODIFY) | screen (ConsumerStatefulWidget) | request-response (Dio POST) | self (already Stack + overlay) | exact |
| `app/lib/provider/router_provider.dart` (MODIFY) | provider (router config) | routing | self (`/trail/:id` sub-route block, lines 176-191) | exact |
| `app/lib/i18n/app_en.arb` / `app_de.arb` (MODIFY) | config (i18n) | static | existing arb keys (referenced via `AppLocalizations.of(context)!`) | exact |

## Pattern Assignments

### `app/lib/routes/navigation_screen.dart` (screen, streaming + request-response)

**Analog:** `app/lib/routes/map_screen.dart`

**Imports pattern** (`map_screen.dart:1-21`): copy the flutter_map cluster — `flutter_map`, `flutter_map_animations`, `flutter_map_location_marker`, `flutter_riverpod`, `font_awesome_flutter`, `go_router`, `latlong2`, plus `package:wanderer/components/map/map_compass.dart` and `package:wanderer/i18n/app_localizations.dart`.

**ConsumerStatefulWidget + AnimatedMapController + dispose** (`map_screen.dart:23-35, 51-66, 103-111`): the class shape required by D-16. Note `with TickerProviderStateMixin` and `late final _animatedMapController = AnimatedMapController(vsync: this)`. Dispose pattern at lines 103-111 disposes the controller — extend it to also cancel the GPS `StreamSubscription` and the recenter `StreamController` (Pitfall 6). Constructor takes `final String id` + `final NavigateResponse response` (passed via `extra`, D-08).

**FlutterMap + MapOptions + onMapEvent for free-pan detection** (`map_screen.dart:167-205`): use `mapController: _animatedMapController.mapController`. For D-09, mirror the `onMapEvent` handler shape — RESEARCH specifies detecting `MapEventMoveStart` with `source == MapEventSource.onDrag` to set `_followEnabled = false`. Set `initialZoom: 15` (D-10) and `maxZoom: 22` (matches line 179).

**CurrentLocationLayer** (`map_screen.dart:217`): the analog uses bare `const CurrentLocationLayer()`. NavigationScreen must configure it per RESEARCH Pattern 2 — `positionStream` from `LocationMarkerDataStreamFactory().fromGeolocatorPositionStream(...)`, `alignPositionOnUpdate`, `alignPositionStream` (recenter), `alignDirectionOnUpdate` (heading-up).

**MapCompass placement** (`map_screen.dart:221-228`): `Positioned(top: 124, right: 8, child: Column(children: [MapCompass(hideIfRotatedNorth: true)]))`. Reuse for the orientation toggle (D-11). `MapCompass` auto-animates back to north (`map_compass.dart:107-131`).

**Breadcrumb PolylineLayer** (model on `map_screen.dart:219-220`, RESEARCH Code Example): render above `TrailLayer`, below `CurrentLocationLayer` — `PolylineLayer(polylines: [Polyline(points: breadcrumb, color: Color(0xFFDC2626), strokeWidth: 3.5)])` (D-18/D-20).

**Loading-style indicator / floating control button** (`map_screen.dart:467-513`): the recenter button (D-09) reuses the `ScaleTransition` + `FilledButton.icon` / `Positioned` overlay pattern at lines 490-513. Spinner card pattern at lines 467-488.

---

### `app/lib/provider/navigation_provider.dart` (notifier, event-driven)

**Analog:** `app/lib/provider/trail/trail_provider.dart` (`@riverpod` family shape) + `app/lib/provider/toast_provider.dart` (immutable state-replacement notifier)

**`@riverpod` class notifier with family arg** (`trail_provider.dart:1-14`): copy `part 'navigation_provider.g.dart';` declaration, `@riverpod class Navigation extends _$Navigation`, and `build(NavigateResponse response)` returning the initial state (RESEARCH notifier skeleton, D-17). Family keyed on the response keeps state isolated/testable.

**Immutable state replacement** (`toast_provider.dart:27-40`): the `state = [...state, x]` / `state = state.copyWith(...)` mutation idiom. Apply in `onPosition`: append breadcrumb and bump `currentManeuverIndex` (RESEARCH Pattern 3). Guard `next >= maneuvers.length` (completion D-14) and clamp `beginShapeIndex` (Pitfall 3).

**Distance primitive** (`gpx_util.dart:46-55`): reuse `const Distance()` and `distanceCalc(a, b)` / `.as(LengthUnit.Meter, a, b)` from latlong2 — same primitive already used in `gpx_util.dart`. Threshold constant `_kManeuverAdvanceThresholdMeters = 30.0` (D-12).

---

### `app/lib/models/navigate_response.dart` (model, freezed)

**Analog:** `app/lib/models/trail.dart`

**Freezed + JsonKey pattern** (`trail.dart:17-18, 29-40`): `part 'navigate_response.freezed.dart'; part 'navigate_response.g.dart';`, `@freezed abstract class NavigateResponse with _$NavigateResponse`, `const factory NavigateResponse({...}) = _NavigateResponse;`, `factory NavigateResponse.fromJson(Map<String,dynamic> json)`. Use `@JsonKey(name: 'begin_shape_index')` for the snake_case API fields (mirrors `@JsonKey(name: 'gpx_data')` at line 37). Mirror Phase-1 TS type: `maneuvers: NavigateManeuver[]`, `shape: [number,number][]`. **A1: verify shape coordinate order (lat,lon vs lon,lat) against a live response — make `LatLng` mapping a single helper.**

---

### `app/lib/routes/trail_detail_screen.dart` (MODIFY — Navigate button + Dio POST)

**Analog:** self + `app/lib/routes/login_screen.dart`

**Current structure** (`trail_detail_screen.dart:8-30`): plain `ConsumerWidget` with `trailAsync.when(data/loading/error)`. To add the fixed bottom button (D-01) over the `TrailPanel` `SingleChildScrollView`, wrap the body in a `Stack` with a bottom-aligned `SafeArea`/`Positioned`, or use `Scaffold.bottomNavigationBar` (Pitfall 4 — add bottom scroll padding). May need conversion to `ConsumerStatefulWidget` to hold the in-flight loading bool (D-06).

**Dio POST + costing derivation + extra push** (RESEARCH Patterns 1 & 4): get the client via `ref.read(apiProvider)` (see `trail_provider.dart:15` and `api_provider.dart:8-20`; baseUrl already `/api/v1`, so call `api.post('/valhalla/navigate', ...)`). Extract waypoints from `trail.expand!.gpx!.allPoints` (`gpx_util.dart:28-30`). Derive costing from `trail.expand?.category?.name`. On success `if (!mounted) return; context.push('/trail/${trail.id}/navigate', extra: response)`.

**Loading button** (D-06): swap the icon slot for `CircularProgressIndicator` — spinner sizing pattern at `map_screen.dart:476-487`.

**Toast on failure** (D-07) — copy `login_screen.dart:45-53`:
```dart
ref.read(toastProvider.notifier).add(
  ToastMessage(
    type: ToastType.error,
    icon: FontAwesomeIcons.circleExclamation,
    text: displayMessage,
  ),
);
```

---

### `app/lib/routes/trail_detail_map_screen.dart` (MODIFY — floating Navigate button)

**Analog:** self (already a `ConsumerStatefulWidget` with a `Stack` + elevation overlay)

**Existing Stack + overlay** (`trail_detail_map_screen.dart:77-159`): the elevation profile is an `Align(alignment: bottomCenter, ...)` overlay (lines 101-159). Add the full-width floating `ElevatedButton.icon` (D-02) as a sibling in the same `Stack`, positioned over the elevation profile. Same Dio call path, costing derivation, loading state, toast, and `context.push(..., extra:)` as the detail screen above. `mounted` check already idiomatic in this file's async flows.

---

### `app/lib/provider/router_provider.dart` (MODIFY — add `navigate` sub-route)

**Analog:** self — the `/trail/:id` block with nested `map` route (`router_provider.dart:176-191`)

Add a sibling sub-route to the existing `map` child inside `GoRoute(path: '/trail/:id', routes: [...])`:
```dart
GoRoute(
  path: 'navigate',
  builder: (context, state) {
    final trailId = state.pathParameters['id']!;
    final response = state.extra as NavigateResponse;
    return NavigationScreen(id: trailId, response: response);
  },
),
```
Add the import for `navigation_screen.dart` alongside the existing route imports (lines 7-24). The `extra`-reading pattern mirrors the `/map` route's `state.extra` handling at lines 113-122.

---

## Shared Patterns

### Dio HTTP client access
**Source:** `app/lib/provider/api_provider.dart:8-20`, used at `trail_provider.dart:15-25`
**Apply to:** both entry screens (the navigate POST)
`ref.watch(apiProvider)` / `ref.read(apiProvider)` returns a `Dio` with baseUrl `…/api/v1` and cookie auth already wired. Call relative paths: `api.post('/valhalla/navigate', data: {...})`. No auth headers to add (cookie jar interceptor handles it).

### Toast error feedback
**Source:** `app/lib/provider/toast_provider.dart:22-40`, usage `login_screen.dart:45-53`
**Apply to:** both entry screens on API failure (D-07)
`ref.read(toastProvider.notifier).add(ToastMessage(type: ToastType.error, icon: FontAwesomeIcons.circleExclamation, text: ...))`. Auto-dismisses after 4s.

### AsyncValue rendering + trailProvider
**Source:** `trail_detail_screen.dart:14-27`, `trail_detail_map_screen.dart:62-353`
**Apply to:** NavigationScreen (watch `trailProvider(id)` for the polyline) and both entry screens
`final trailAsync = ref.watch(trailProvider(id)); trailAsync.when(data:, loading: () => CircularProgressIndicator(), error: (e,s) => WandererError(err: e, stack: s))`.

### `@riverpod` codegen notifier
**Source:** `trail_provider.dart:1-14`, `toast_provider.dart:22-26`
**Apply to:** `navigation_provider.dart`
`part '…g.dart';`, `@riverpod class X extends _$X { @override T build(...) }`. Regenerate with `cd app && dart run build_runner build --delete-conflicting-outputs`.

### Mounted guard after async gap
**Source:** `map_screen.dart:81, 250`
**Apply to:** both entry screens before `context.push` / toast after the awaited Dio call (Pitfall 5)
`if (!mounted) return;`

### GPX → waypoint extraction
**Source:** `app/lib/util/gpx_util.dart:19-36`
**Apply to:** entry screens (build POST body) and NavigationScreen (initial center)
`trail.expand!.gpx!.allPoints` yields `List<LatLng>`. Map to `{'lat':, 'lon':}` for the request. **A4: downsample if `> 2000` points before POST.**

## No Analog Found

None. Every new file has a same-role or same-data-flow analog in the codebase. The only genuinely novel logic (maneuver-advancement geometry, shared-broadcast GPS stream wiring) is composed from existing primitives (`latlong2 Distance`, `LocationMarkerDataStreamFactory`) documented in RESEARCH Patterns 2-3 rather than copied from an existing file.

## Metadata

**Analog search scope:** `app/lib/routes/`, `app/lib/provider/`, `app/lib/components/map/`, `app/lib/models/`, `app/lib/util/`
**Files scanned:** 13 candidate analogs located; 9 read in full
**Pattern extraction date:** 2026-06-12
