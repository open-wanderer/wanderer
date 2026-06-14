# Phase 2: Navigation Screen - Research

**Researched:** 2026-06-12
**Domain:** Flutter mobile — live GPS navigation screen (flutter_map + Riverpod + go_router)
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Navigate Button Entry**
- **D-01:** `TrailDetailScreen` — fixed full-width `ElevatedButton.icon` pinned at the bottom, in front of (above) the scrollable content. Filled primary color.
- **D-02:** `TrailDetailMapScreen` — large full-width `ElevatedButton.icon` floating over the elevation profile when open. Same style as D-01.
- **D-03:** Button label "Navigate" + navigation icon. `ElevatedButton.icon`, filled primary, full-width.
- **D-04:** go_router path `/trail/:id/navigate` — sub-route of trail detail path (not top-level).

**API Call Timing**
- **D-05:** `POST /api/v1/valhalla/navigate` happens **on the trail detail screen** (before route transition), not inside `NavigationScreen`. Tap Navigate → button loading → Dio call → on success navigate; on failure toast + stay.
- **D-06:** While API in flight, Navigate button shows `CircularProgressIndicator` in the icon slot; rest of screen stays interactive.
- **D-07:** On API failure: existing `toast_provider` generic error message, cancel navigation, do not change screen.
- **D-08:** `NavigationScreen` receives only the trail ID via route param; watches existing `trailProvider(id)` for trail data. The fetched `NavigateResponse` is passed as go_router `extra` to avoid a second API call.

**Map Camera Follow**
- **D-09:** Auto-follow with free-pan. Auto-centers on GPS by default; if user pans, auto-follow pauses and a recenter button appears (reuse existing `MapScreen` recenter pattern). Recenter resumes auto-follow.
- **D-10:** Zoom inherit / user-adjustable. Start ~15–16, respect pinch-zoom, do not lock.
- **D-11:** Orientation toggle: compass icon button top-right. Reuse existing `MapCompass` widget pattern. Cycles north-up ↔ heading-up; in heading-up the map rotates to device bearing.

**Maneuver Advancement**
- **D-12:** Distance threshold ~30 m to the `begin_shape_index` point of the next maneuver. Within 30 m → advance `currentManeuverIndex` by 1.
- **D-13:** GPS stream source: `flutter_map_location_marker` stream (same `LocationMarkerDataStreamFactory` used by `CurrentLocationLayer`). No second Geolocator stream.
- **D-14:** Trail end: completion banner in the maneuver area ("You've arrived!"). Navigation stays active. User exits manually.
- **D-15:** Save-as-summit-log is deferred. Completion banner is display-only.

**Breadcrumb Trace**
- **D-18:** Red polyline of the actual traveled path (NAV-08). Append GPS positions to in-memory `List<LatLng>`, render as `PolylineLayer` on top of trail polyline.
- **D-19:** Session-only, not persisted. Discarded on exit.
- **D-20:** Red/crimson, slightly thinner stroke than the trail polyline.

**Screen Architecture**
- **D-16:** `NavigationScreen` is a `ConsumerStatefulWidget` (needs `initState`/`dispose`). Use `@riverpod` codegen for the navigate API provider.
- **D-17:** Navigation provider (holds `NavigateResponse` + current maneuver index) lives in a notifier, not inline widget state — survives hot-reload and is testable.

### Claude's Discretion
- The 30 m threshold is a starting heuristic — make it a named constant (`_kManeuverAdvanceThresholdMeters = 30.0`).
- Whether `NavigationScreen` extends or composes `WandererMap` (see Architecture below — research recommends composing a fresh `FlutterMap`, not `WandererMap`).

### Deferred Ideas (OUT OF SCOPE)
- Save completed trail as summit log (future iteration).
- Adaptive / speed-weighted threshold for maneuver advancement.
- Off-trail detection / re-routing (out of scope per REQUIREMENTS.md).
- Stats `DraggableScrollableSheet` layer (STATS-01..05 — that is **Phase 3**).
- TTS audio (AUDIO-01/02 — v2). Offline nav (OFFLINE-01/02 — v2).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| NAV-01 | Launch navigation from trail detail screen | `trail_detail_screen.dart` uses `SingleChildScrollView` inside `TrailPanel` → add fixed button via `Stack`/`bottomNavigationBar` overlay (D-01). API call + `context.push('/trail/$id/navigate', extra: response)`. |
| NAV-02 | Launch from trail detail map screen | `trail_detail_map_screen.dart` already a `Stack` with elevation profile overlay → add floating button (D-02). Same call path. |
| NAV-03 | Full-screen map centered + following GPS | Reuse `FlutterMap` + `AnimatedMapController` + `CurrentLocationLayer` pattern from `map_screen.dart`. `CurrentLocationLayer.alignPositionOnUpdate: AlignOnUpdate.always` gives built-in follow. |
| NAV-04 | Display current Valhalla maneuver at top | `NavigateResponse.maneuvers[currentManeuverIndex].instruction` in a `surface` card banner (UI-SPEC typography). |
| NAV-05 | North-up / heading-up toggle | Reuse `MapCompass`; heading-up via `AnimatedMapController.animateTo(rotation: -bearing)` or `CurrentLocationLayer.alignDirectionOnUpdate`. |
| NAV-06 | Auto-advance maneuvers as user moves | Subscribe to the shared GPS stream; on each position compute distance to `shape[nextManeuver.begin_shape_index]`; advance when < 30 m (D-12). |
| NAV-07 | Exit and return to originating screen | go_router sub-route → `context.pop()` returns to the detail/map screen that pushed it (D-04). Exit button top-left. |
| NAV-08 | Red breadcrumb trace of actual path | In-memory `List<LatLng>` appended from the GPS stream, rendered in a `PolylineLayer` above `TrailLayer` (D-18, D-20). |
</phase_requirements>

## Summary

This is a **pure Flutter phase** that builds one new screen (`navigation_screen.dart`), one new Riverpod notifier (`navigation_provider.dart`), and modifies two entry screens plus the router. **No new pub.dev packages are required** — every capability is already in `pubspec.yaml`: `flutter_map 8.3.0`, `flutter_map_animations 0.10.0`, `flutter_map_location_marker 10.0.2`, `geolocator 13.0.2`, `latlong2 0.9.1`, `gpx 2.3.0`, `go_router 17.2.1`, `flutter_riverpod 3.3.1`, `riverpod_annotation/generator`.

The single highest-leverage finding: **`CurrentLocationLayer` already implements auto-follow and heading-up natively.** Its `alignPositionOnUpdate` (`AlignOnUpdate.always`) keeps the camera centered on the GPS dot, `alignPositionStream` triggers a one-shot recenter (the D-09 recenter button), and `alignDirectionOnUpdate` rotates the map to device heading (D-11 heading-up). This means the follow/recenter/heading machinery does **not** need to be hand-rolled on top of a raw position stream — it is configuration on an existing widget.

The second key finding concerns the **shared GPS stream (D-13)**. `LocationMarkerDataStreamFactory().fromGeolocatorPositionStream()` produces a `Stream<LocationMarkerPosition?>`, but `LocationMarkerPosition` carries only `latitude/longitude/accuracy` — **it drops heading and speed.** For maneuver advancement (D-12) and the breadcrumb (D-18) that is sufficient (lat/lon only). But heading-up orientation and Phase-3 speed stats need the raw `geolocator` `Position` (which has `.heading` and `.speed`). The clean pattern: build one `Geolocator.getPositionStream()` as a broadcast source, feed it both to `CurrentLocationLayer.positionStream` (via the factory) and to the navigation notifier (raw). See Architecture Pattern 2.

**Primary recommendation:** Compose a fresh `FlutterMap` in `NavigationScreen` (modeled on `map_screen.dart`, NOT `WandererMap` — see Don't Hand-Roll), drive it with `AnimatedMapController`, configure `CurrentLocationLayer` for follow + heading, hold `NavigateResponse` + `currentManeuverIndex` + breadcrumb `List<LatLng>` in an `@riverpod` notifier, and advance maneuvers from a single shared `geolocator` position stream using `latlong2`'s `Distance` for the 30 m check.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Valhalla maneuver fetch | API / Backend (Phase 1, done) | Mobile Frontend (Dio call) | Endpoint exists; Flutter only consumes it (D-05). |
| Maneuver list + current index state | Mobile Providers (`@riverpod` notifier) | — | Survives hot-reload, testable (D-17). |
| GPS acquisition | Mobile Frontend (geolocator/location_marker) | — | Device sensor; one stream shared (D-13). |
| Camera follow / recenter | Mobile Frontend (`CurrentLocationLayer` + `AnimatedMapController`) | — | Built-in `AlignOnUpdate`; no custom follow loop. |
| Heading-up rotation | Mobile Frontend (map controller rotation) | — | Driven by GPS bearing or device heading. |
| Maneuver advancement logic | Mobile Providers (notifier) | Mobile Frontend (stream sub) | Pure geometry on position + shape; belongs off the widget. |
| Breadcrumb accumulation | Mobile Frontend / notifier (in-memory list) | — | Session-only, discarded on exit (D-19). |
| Route param + extra passing | Mobile Frontend (go_router) | — | `/trail/:id/navigate` sub-route, `extra: NavigateResponse` (D-04, D-08). |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| flutter_map | ^8.3.0 | Map widget + `PolylineLayer`/`MarkerLayer` | Already the app's map engine (`map_screen.dart`, `wanderer_map.dart`). |
| flutter_map_animations | ^0.10.0 | `AnimatedMapController` (`animateTo` center/zoom/rotation) | Used in `map_screen.dart` for follow/fit animations. |
| flutter_map_location_marker | ^10.0.2 | `CurrentLocationLayer` (GPS dot, follow, heading, position stream) | Already renders the GPS dot; provides the shared stream (D-13). |
| geolocator | ^13.0.2 | Raw `Position` stream (lat/lon + heading + speed) | Underlying source; needed for heading-up + Phase-3 speed. |
| latlong2 | ^0.9.1 | `LatLng`, `Distance.as(...)`, `Distance.bearing(...)` | Threshold geometry for D-12; same `Distance` used in `gpx_util.dart`/`trail_layer.dart`. |
| gpx | ^2.3.0 | Parse trail GPX track → waypoints | Already parsed in `trail_provider.dart`; `GpxMappingUtils.allPoints` extension exists. |
| go_router | ^17.2.1 | `/trail/:id/navigate` sub-route + `extra` | App router is `router_provider.dart`. |
| flutter_riverpod + riverpod_annotation | ^3.3.1 / ^4.0.2 | `@riverpod` notifier for nav state | App-wide state pattern. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| font_awesome_flutter | ^11.0.0 | `FaIcon` for buttons (`locationArrow`, `xmark`, `locationCrosshairs`) | Per UI-SPEC icon contract. |
| intl / format_util | — | `formatDistance` for "in {distance}" sub-label (NAV-04) | Existing `lib/util/format_util.dart`. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Sharing the location_marker stream | A second `geolocator` stream just for advancement | D-13 explicitly forbids; double GPS subscriptions waste battery and risk divergent positions. |
| `WandererMap` composition | Fresh `FlutterMap` in `NavigationScreen` | `WandererMap` hardcodes `initialZoom: 18`, `initialCameraFit` to trail bounds, and a fixed control layout — it fights the follow/zoom-15 requirement. Compose a new map. |
| `flutter_compass` for heading | geolocator `Position.heading` (GPS course) or location_marker's rotation-sensor heading stream | No new package needed; GPS course is adequate for heading-up while moving. |

**Installation:**
```bash
# None — all packages already in app/pubspec.yaml.
# After adding the @riverpod notifier, regenerate:
cd app && dart run build_runner build --delete-conflicting-outputs
```

**Version verification:** All versions read directly from `app/pubspec.yaml` and confirmed present in `~/.pub-cache/hosted/pub.dev/` (the resolved local copies were inspected for API signatures). [VERIFIED: pubspec.yaml + local pub-cache]

## Package Legitimacy Audit

> This phase installs **no new packages**. All dependencies are already resolved in `app/pubspec.yaml` and present in the local pub-cache. `slopcheck` targets npm/PyPI and does not cover pub.dev, so it is not applicable here.

| Package | Registry | Already in pubspec | Disposition |
|---------|----------|--------------------|-------------|
| flutter_map 8.3.0 | pub.dev | yes | Approved (no install) |
| flutter_map_animations 0.10.0 | pub.dev | yes | Approved (no install) |
| flutter_map_location_marker 10.0.2 | pub.dev | yes | Approved (no install) |
| geolocator 13.0.2 | pub.dev | yes | Approved (no install) |
| latlong2 0.9.1 | pub.dev | yes | Approved (no install) |
| gpx 2.3.0 | pub.dev | yes | Approved (no install) |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
[TrailDetailScreen / TrailDetailMapScreen]
        |  tap "Navigate"  (button enters loading state, D-06)
        v
  derive costing from trail.category ("bike"/"cycling"/"bicycle" -> bicycle, else pedestrian)
  extract waypoints from trail.expand.gpx.allPoints  (lat/lon only)
        |
        v
  Dio POST /api/v1/valhalla/navigate  { waypoints, costing }   (D-05)
        |                                   |
   success                              failure
        |                                   |
        v                                   v
  context.push('/trail/:id/navigate',   toast(error) + stay (D-07)
     extra: NavigateResponse)
        |
        v
[NavigationScreen  (ConsumerStatefulWidget, D-16)]
   reads NavigateResponse from GoRouterState.extra
   watches trailProvider(id) for trail polyline (D-08)
   |
   +--> navigationProvider notifier (D-17): { maneuvers, shape, currentManeuverIndex, breadcrumb[] }
   |
   +--> ONE geolocator position stream (broadcast)
   |        |                                   |
   |   -> CurrentLocationLayer.positionStream   -> notifier.onPosition(pos)
   |      (GPS dot, follow, heading)                 |
   |                                                 +-- append pos to breadcrumb (D-18)
   |                                                 +-- dist to shape[next.begin_shape_index]
   |                                                 |     < 30m -> currentManeuverIndex++ (D-12)
   |                                                 +-- last maneuver -> completion banner (D-14)
   v
  FlutterMap layers (bottom->top):
   tiles -> TrailLayer (trail polyline) -> breadcrumb PolylineLayer (crimson)
   -> CurrentLocationLayer (GPS dot) -> MapCompass -> floating controls -> maneuver banner
```

### Recommended Project Structure
```
app/lib/
├── routes/
│   ├── navigation_screen.dart          # NEW — ConsumerStatefulWidget (D-16)
│   ├── trail_detail_screen.dart        # MODIFY — add fixed bottom Navigate button (D-01)
│   └── trail_detail_map_screen.dart    # MODIFY — add floating Navigate button (D-02)
├── provider/
│   ├── navigation_provider.dart        # NEW — @riverpod notifier (D-17) + .g.dart
│   └── router_provider.dart            # MODIFY — add 'navigate' sub-route under /trail/:id (D-04)
├── models/
│   └── navigate_response.dart          # NEW — Dart model mirroring Phase-1 NavigateResponse (freezed)
└── i18n/
    ├── app_en.arb / app_de.arb         # MODIFY — add nav copy keys (UI-SPEC copywriting contract)
```

### Pattern 1: go_router sub-route with `extra` payload (D-04, D-08)
**What:** Add `navigate` as a child route of `/trail/:id` so the back stack returns to the originator (NAV-07), passing the pre-fetched response via `extra`.
**When to use:** Always — entry is gated on a successful API response (D-05).
```dart
// router_provider.dart — inside the existing GoRoute(path: '/trail/:id', routes: [...])
// Source: existing pattern at app/lib/provider/router_provider.dart:177-191
GoRoute(
  path: 'navigate',
  builder: (context, state) {
    final trailId = state.pathParameters['id']!;
    final response = state.extra as NavigateResponse; // gated: never null (D-05)
    return NavigationScreen(id: trailId, response: response);
  },
),
```
```dart
// Entry screen, after successful Dio POST:
if (!mounted) return;
context.push('/trail/${trail.id}/navigate', extra: navigateResponse);
```

### Pattern 2: Single shared GPS stream (D-13)
**What:** One broadcast `geolocator` stream feeds both the map's `CurrentLocationLayer` and the navigation notifier. Avoids a second subscription while still exposing heading/speed (which `LocationMarkerPosition` drops).
**When to use:** In `NavigationScreen.initState`.
```dart
// Source: flutter_map_location_marker 10.0.2 data_stream_factory.dart
final _positionStream = Geolocator.getPositionStream().asBroadcastStream();

// 1) Feed the location marker (GPS dot + follow + heading):
CurrentLocationLayer(
  positionStream: const LocationMarkerDataStreamFactory()
      .fromGeolocatorPositionStream(stream: _positionStream),
  alignPositionStream: _recenterTrigger.stream,        // recenter button (D-09)
  alignPositionOnUpdate: _followEnabled ? AlignOnUpdate.always : AlignOnUpdate.never,
  alignDirectionOnUpdate:
      _headingUp ? AlignOnUpdate.always : AlignOnUpdate.never, // heading-up (D-11)
)

// 2) Feed the notifier (raw Position has .heading + .speed; lat/lon used for D-12/D-18):
_sub = _positionStream.listen((pos) {
  ref.read(navigationProvider.notifier).onPosition(LatLng(pos.latitude, pos.longitude));
});
```
> Note: `CurrentLocationLayer.alignPositionOnUpdate` and `alignPositionStream` are the **built-in** follow + recenter mechanism — verified in `current_location_layer.dart`. Prefer this over manually calling `animateTo` on every position event. If finer control of zoom is needed, you may instead drive `AnimatedMapController.animateTo(dest: latLng, rotation: headingUp ? -bearing : 0)` yourself and set `alignPositionOnUpdate: never` — but do not do both at once (camera fight).

### Pattern 3: Maneuver advancement (D-12)
**What:** On each position, measure distance to the next maneuver's shape point; advance when within threshold.
```dart
// In the navigation notifier. Source: latlong2 0.9.1 Distance.as(...)
static const _kManeuverAdvanceThresholdMeters = 30.0; // D-12, tunable
final _distance = const Distance();

void onPosition(LatLng pos) {
  breadcrumb = [...breadcrumb, pos];                    // D-18
  final next = currentManeuverIndex + 1;
  if (next >= maneuvers.length) return;                 // already at last (D-14)
  final target = shape[maneuvers[next].beginShapeIndex];
  final meters = _distance.as(LengthUnit.Meter, pos, target);
  if (meters <= _kManeuverAdvanceThresholdMeters) {
    state = state.copyWith(currentManeuverIndex: next);
  }
  // emit new state (breadcrumb + index)
}
```

### Pattern 4: Costing derivation on the entry screen
**What:** Derive Valhalla costing from the trail category before the Dio call (CONTEXT "specifics").
```dart
String _costingFor(String? category) {
  final c = (category ?? '').toLowerCase();
  return (c.contains('bike') || c.contains('cycling') || c.contains('bicycle'))
      ? 'bicycle'
      : 'pedestrian';
}
// waypoints: trail.expand!.gpx!.allPoints.map((p) => {'lat': p.latitude, 'lon': p.longitude})
//   — uses the existing GpxMappingUtils.allPoints extension (gpx_util.dart).
```

### Anti-Patterns to Avoid
- **Composing `WandererMap` for the nav screen:** it forces `initialZoom: 18` and a trail-bounds camera fit, fighting follow + zoom-15. Compose a fresh `FlutterMap` like `map_screen.dart`.
- **Two GPS streams:** subscribing geolocator separately from the location marker (violates D-13; battery + position divergence).
- **Maneuver index in widget `setState`:** loses state on hot-reload and is untestable (D-17 requires a notifier).
- **Re-fetching the trail or the route inside `NavigationScreen`:** trail comes from `trailProvider(id)`, route comes from `extra` (D-08).
- **Driving both `alignPositionOnUpdate: always` AND manual `animateTo` per frame:** camera tug-of-war / jitter.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Camera auto-follow | Manual `animateTo` on every GPS event | `CurrentLocationLayer.alignPositionOnUpdate: always` | Built-in, animated, debounced. Verified in `current_location_layer.dart`. |
| Recenter trigger | Custom "is panned" detection + re-center button wiring | `alignPositionStream` one-shot event + reuse `map_screen.dart` `ScaleTransition` button pattern | One-shot stream re-aligns once; existing scale-in pattern at `map_screen.dart:490`. |
| Heading-up rotation | Manual rotation math per frame | `alignDirectionOnUpdate` OR `AnimatedMapController.animateTo(rotation: ...)` | Library/controller handle the tween. |
| Distance to maneuver point | Haversine implementation | `latlong2` `Distance().as(LengthUnit.Meter, a, b)` | Already the app's distance primitive (`gpx_util.dart`). |
| GPX → waypoint list | Manual XML walk | `gpx_util.dart` `GpxMappingUtils.allPoints` | Extension already exists and is used by `TrailLayer`. |
| Compass / north-up | New compass widget | `MapCompass` (`components/map/map_compass.dart`) | Already animates back to north; reused in `map_screen.dart`. |
| Toasts | New snackbar system | `toast_provider.dart` `Toast.add(ToastMessage(...))` | Existing app toast (D-07). |
| GPS permission flow | Manual permission prompts | `LocationMarkerDataStreamFactory.defaultPositionStreamSource` request callback | Already handled by the location marker stream. |

**Key insight:** `flutter_map_location_marker` is not just "a dot on the map" — it is a full follow/heading/permission stack. The navigation screen is mostly *configuration* of existing widgets plus a thin advancement notifier, not new map machinery.

## Runtime State Inventory

> Not a rename/refactor/migration phase — greenfield screen addition. The closest analog is router and state wiring:

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — breadcrumb is in-memory and discarded on exit (D-19); no persistence. | None — verified by D-19. |
| Live service config | None — Valhalla endpoint already exists from Phase 1; no infra change. | None. |
| OS-registered state | Location permission (iOS `NSLocationWhenInUseUsageDescription`, Android `ACCESS_FINE_LOCATION`) — already configured because `CurrentLocationLayer`/`showLocation` is in use on `TrailDetailMapScreen`. | Verify entitlements present (see Environment Availability); no new permission. |
| Secrets/env vars | None — call goes through existing Dio base URL + cookie auth. | None. |
| Build artifacts | New `navigation_provider.g.dart` and (if freezed) `navigate_response.freezed.dart`/`.g.dart` must be generated. | Run `dart run build_runner build` after adding annotations. |

## Common Pitfalls

### Pitfall 1: `LocationMarkerPosition` silently drops heading and speed
**What goes wrong:** Code reads heading/speed off the location-marker stream and gets nothing (the model only has lat/lon/accuracy).
**Why it happens:** `fromGeolocatorPositionStream` maps `Position` → `LocationMarkerPosition` discarding heading/speed (verified in `data_stream_factory.dart` lines 30-37).
**How to avoid:** Subscribe the notifier to the **raw `geolocator` `Position`** stream (has `.heading`, `.speed`); feed the location marker a *derived* stream. One broadcast source, two consumers (Pattern 2).
**Warning signs:** Heading-up doesn't rotate; Phase-3 speed stats read 0.

### Pitfall 2: Camera fight between follow and manual animate
**What goes wrong:** Map jitters or snaps unexpectedly.
**Why it happens:** `alignPositionOnUpdate: always` AND a manual `animateTo` per position both move the camera.
**How to avoid:** Pick one. For zoom-15 + heading-up, either (a) `alignPositionOnUpdate: always` + `alignDirectionOnUpdate: always` and let the layer own the camera, or (b) `alignPositionOnUpdate: never` + drive `AnimatedMapController.animateTo(dest, rotation)` yourself.
**Warning signs:** Visible stutter on each GPS tick.

### Pitfall 3: `begin_shape_index` out of bounds / non-monotonic
**What goes wrong:** `shape[maneuvers[next].beginShapeIndex]` throws or advancement skips/sticks.
**Why it happens:** Trusting Valhalla indices without guarding; last maneuver has no "next".
**How to avoid:** Guard `next >= maneuvers.length` (completion, D-14) and clamp `beginShapeIndex` to `shape.length - 1`. Only ever advance forward (never decrement).
**Warning signs:** RangeError on arrival; banner flickers between maneuvers.

### Pitfall 4: Fixed bottom button over a scroll view (D-01)
**What goes wrong:** Button scrolls away or overlaps content.
**Why it happens:** `TrailDetailScreen` body is `TrailPanel` → `SingleChildScrollView` (verified `trail_panel.dart:48`).
**How to avoid:** Wrap in a `Stack` with the button in a bottom-aligned `Positioned`/`SafeArea`, or use `Scaffold.bottomNavigationBar`/`persistentFooterButtons`. Add bottom padding to the scroll content so the last item isn't hidden.
**Warning signs:** Button disappears on scroll; CTA covers trail actions.

### Pitfall 5: `context.push` after async gap without `mounted` check
**What goes wrong:** Navigating after the widget is disposed throws.
**Why it happens:** The Dio call (D-05) is awaited; the user may pop meanwhile.
**How to avoid:** `if (!mounted) return;` before `context.push` / toast (the codebase already follows this — see `map_screen.dart:81`).
**Warning signs:** "Looking up a deactivated widget's ancestor" errors.

### Pitfall 6: Stream/controller leaks
**What goes wrong:** GPS keeps streaming after exit; battery drain; setState-after-dispose.
**Why it happens:** `StreamSubscription`, `AnimatedMapController`, recenter `StreamController` not disposed.
**How to avoid:** Cancel/dispose all in `NavigationScreen.dispose()` (mirror `map_screen.dart:103-111`).
**Warning signs:** Location indicator stays active after leaving the screen.

## Code Examples

### Map layer order in NavigationScreen (NAV-03/08, UI-SPEC layering)
```dart
// Source: composed from map_screen.dart:165-291 + wanderer_map.dart:127-167
FlutterMap(
  mapController: _animatedMapController.mapController,
  options: MapOptions(
    initialCenter: trail.expand!.gpx!.allPoints.first,
    initialZoom: 15,          // D-10 sensible hiking start; pinch not locked
    maxZoom: 22,
    onMapEvent: (e) {
      if (e is MapEventMoveStart && e.source == MapEventSource.onDrag) {
        setState(() => _followEnabled = false); // user panned -> pause follow (D-09)
      }
    },
  ),
  children: [
    _tileLayer(style),                                   // map_style_provider
    if (trail.expand?.gpx != null) TrailLayer(trail: trail, showWaypoints: false),
    PolylineLayer(polylines: [                           // breadcrumb (D-18/D-20)
      Polyline(points: breadcrumb, color: const Color(0xFFDC2626), strokeWidth: 3.5),
    ]),
    CurrentLocationLayer(/* Pattern 2 config */),
    // MapCompass + floating controls + maneuver banner stacked above
  ],
)
```

### Riverpod nav notifier skeleton (D-17)
```dart
// app/lib/provider/navigation_provider.dart
@riverpod
class Navigation extends _$Navigation {
  @override
  NavigationState build(NavigateResponse response) =>
      NavigationState(response: response, currentManeuverIndex: 0, breadcrumb: const []);

  void onPosition(LatLng pos) { /* Pattern 3 + breadcrumb append */ }
}
// Family keyed on the response keeps the screen's state isolated and testable.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual camera centering loop | `CurrentLocationLayer.alignPositionOnUpdate` / `alignPositionStream` | location_marker 8+ | Follow + recenter are config, not code. |
| `StateNotifierProvider` (CONTEXT mentions as option) | `@riverpod` notifier (riverpod 3 / generator 4) | riverpod_generator 4.x | App standard is codegen notifiers; prefer it over legacy `StateNotifier`. |

**Deprecated/outdated:**
- Legacy `flutter_map` `MapController.move` per-frame for follow — superseded by location_marker alignment + `flutter_map_animations`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Valhalla `shape` is `[lat, lon]` pairs (not `[lon, lat]`) when consumed in Flutter, matching `LatLng(shape[i][0], shape[i][1])`. | Pattern 3 | Maneuver targets land at wrong coordinates → advancement never triggers. **Planner should verify against Phase 1 endpoint output / `01-RESEARCH` coordinate notes (commit d8dbf1ee mentions encode/decode coordinate asymmetry).** |
| A2 | A Dart `NavigateResponse` model must be hand-created mirroring the TS type (the TS schema is web-only; no Dart equivalent found). | Standard Stack | Minor — just an extra model file. |
| A3 | GPS course (`Position.heading`) is acceptable for heading-up while moving (vs. magnetometer). | Pattern 2 / D-11 | Heading-up jitter when stationary; acceptable per "while moving" navigation use. |
| A4 | `trail.expand.gpx.allPoints` has ≤ 2000 points (Valhalla request `.max(2000)`). | Pattern 4 | Long trails exceed the cap → API 400. Planner may need to downsample waypoints before the POST. |

## Open Questions

1. **Coordinate order of `shape` (A1)**
   - What we know: Phase 1 schema types `shape` as `[number, number][]`; commit `d8dbf1ee` documents an encode/decode coordinate asymmetry in the Valhalla code.
   - What's unclear: Whether the array is `[lat, lon]` or `[lon, lat]` as delivered to Flutter.
   - Recommendation: Planner adds a verification task to confirm order against a live response before wiring D-12; make the `LatLng(shape[i][?], shape[i][?])` mapping a single helper so it's a one-line fix.

2. **Waypoint count cap for long trails (A4)**
   - What we know: request schema caps waypoints at 2000.
   - What's unclear: Whether any target trails exceed 2000 GPX points.
   - Recommendation: Downsample `allPoints` (e.g., every Nth point or distance-based) before the POST if length > 2000; otherwise pass through.

3. **Heading-up source**
   - What we know: GPS `Position.heading` and a rotation-sensor heading stream are both available.
   - What's unclear: Which gives a smoother result for hikers (slow movement).
   - Recommendation: Start with GPS course; expose the rotation-sensor option behind the same toggle if jitter is observed. Not a blocker.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| flutter_map et al. (pub) | Whole screen | ✓ (in pubspec + pub-cache) | per table above | — |
| Location permission (iOS/Android manifest) | GPS follow (NAV-03/06/08) | ✓ (already used by `TrailDetailMapScreen` `showLocation: true`) | — | If absent, location stream errors — planner adds a verify step |
| Valhalla `/api/v1/valhalla/navigate` | Maneuver fetch (D-05) | ✓ (Phase 1 complete per REQUIREMENTS API-01/02) | — | — |
| build_runner (dev) | Codegen for `.g.dart` | ✓ (in pubspec dev deps) | 2.13.1 | — |

**Missing dependencies with no fallback:** none identified.
**Missing dependencies with fallback:** Verify location entitlement strings exist in `ios/Runner/Info.plist` and `android/app/src/main/AndroidManifest.xml` (highly likely present given existing location usage).

## Validation Architecture

> `workflow.nyquist_validation` is `false` in `.planning/config.json` — section omitted per instructions.

## Security Domain

> `security_enforcement: true`, ASVS level 1. This phase adds no auth, no new endpoint, no new input surface on the backend.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Reuses existing Dio cookie auth; no change. |
| V3 Session Management | no | No change. |
| V4 Access Control | no | No new endpoint (Phase 1 endpoint already enforces). |
| V5 Input Validation | yes (light) | Outbound request body validated server-side by Phase-1 Zod schema; client should still guard waypoint count (A4) and non-empty shape/maneuvers before entering the screen. |
| V6 Cryptography | no | None hand-rolled. |

### Known Threat Patterns for Flutter mobile nav

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Trusting unbounded Valhalla indices (`begin_shape_index`) | Tampering/DoS (RangeError crash) | Bounds-guard + monotonic forward-only advancement (Pitfall 3). |
| Location data leakage | Information disclosure | Breadcrumb is in-memory only, discarded on exit (D-19); nothing persisted/transmitted. |
| Oversized waypoint payload | DoS on Valhalla | Server caps at 2000; client downsamples (A4). |

## Sources

### Primary (HIGH confidence)
- Local pub-cache source (inspected directly):
  - `flutter_map_location_marker-10.0.2/lib/src/data/data_stream_factory.dart` — stream factory, permission flow
  - `flutter_map_location_marker-10.0.2/lib/src/data/data.dart` — `LocationMarkerPosition` (no heading/speed), `LocationMarkerHeading`
  - `flutter_map_location_marker-10.0.2/lib/src/widgets/current_location_layer.dart` — `alignPositionOnUpdate`, `alignPositionStream`, `alignDirectionOnUpdate`
  - `flutter_map_animations-0.10.0/lib/src/animated_map_controller.dart` — `animateTo(dest, zoom, rotation, ...)`
  - `latlong2-0.9.1/lib/latlong/Distance.dart` — `as(LengthUnit, p1, p2)`, `bearing(p1, p2)`
  - `geolocator_platform_interface .../position.dart` — `Position.heading`, `Position.speed`
- Codebase (grep + read):
  - `app/lib/routes/map_screen.dart` — follow/recenter/compass reference implementation
  - `app/lib/components/base/wanderer_map.dart`, `components/map/map_compass.dart`, `components/map/trail_layer.dart`
  - `app/lib/provider/trail/trail_provider.dart`, `api_provider.dart`, `toast_provider.dart`, `router_provider.dart`
  - `app/lib/util/gpx_util.dart` (`GpxMappingUtils.allPoints`)
  - `web/src/lib/models/api/valhalla_navigate_schema.ts` — Phase 1 API contract
- `.planning/phases/02-navigation-screen/02-CONTEXT.md`, `02-UI-SPEC.md`, `.planning/REQUIREMENTS.md`

### Secondary (MEDIUM confidence)
- None required — all claims grounded in local source.

### Tertiary (LOW confidence)
- A1 coordinate order (flagged for verification against live Phase-1 output).

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all packages present and APIs read from local source.
- Architecture: HIGH — follow/recenter/heading verified as built-in widget config; entry/router patterns mirror existing screens.
- Pitfalls: HIGH — each derived from inspected source (stream model, scroll view, dispose patterns).
- Coordinate order (A1): LOW — needs one live-response check before wiring D-12.

**Research date:** 2026-06-12
**Valid until:** 2026-07-12 (stable; pinned package versions in pubspec)
