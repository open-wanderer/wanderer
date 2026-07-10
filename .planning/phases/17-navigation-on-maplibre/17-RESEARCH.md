# Phase 17: Navigation on MapLibre - Research

**Researched:** 2026-07-09
**Domain:** Flutter native MapLibre GL (`maplibre` 0.3.5) — location tracking, camera control, gesture-reason events, native compass, applied to a turn-by-turn navigation screen migrating off `flutter_map`.
**Confidence:** MEDIUM-HIGH (all core API signatures verified directly against the installed `maplibre`/`maplibre_android`/`maplibre_ios` package source in the pub cache; one significant native-SDK behavioral gap found and must be designed around; a few native-default behaviors remain training-knowledge-level and are flagged `[ASSUMED]`)

## Project Constraints (from CLAUDE.md)

**Note on scope mismatch:** `./CLAUDE.md`'s `## Project` section header describes "Wanderer Instance Federation" (Go/PocketBase ActivityPub federation, "no SvelteKit or Flutter changes required for v1") — this is a *different, unrelated* milestone/feature from the one this phase belongs to (v1.4 MapLibre Migration, tracked in `.planning/PROJECT.md`/`ROADMAP.md`). CLAUDE.md's federation-specific constraints do not apply to this Flutter navigation phase and are not enforced here. The remaining CLAUDE.md sections (Technology Stack, Conventions, Architecture) are repo-wide and DO apply:

- **Tech stack confirmation:** `maplibre` 4.7.1 web / `maplibre` 0.3.5 Dart, `Flutter Riverpod` 3.3.1, `Go Router` 17.2.1, `objectbox` 5.3.1, `geolocator` 13.0.2 are all listed as the app's established stack — this phase introduces no new dependency, consistent with CLAUDE.md's documented stack.
- **Naming conventions (Dart, by analogy from the documented TS conventions — no Dart-specific linter config exists in CLAUDE.md):** camelCase functions/variables, PascalCase classes/widgets, boolean fields prefixed `is`/`show` (already followed by `isOffline`, `showTrail`, `showLocation` in `WandererMap`) — the plan should keep new state fields (`_followEnabled`, `_headingUp`, `_activePointers`) consistent with this existing style, already the case in `navigation_screen.dart` today.
- **Error handling:** "Components: Guard against missing data" — applies directly to null-guarding `_controller`/`style` before every native call (already the established pattern throughout `WandererMap`/`map_screen.dart`, e.g. `final controller = _controller; if (controller == null) return;`).
- **GSD Workflow Enforcement:** file-changing work must go through a GSD command (`/gsd-execute-phase` for this planned phase work) — not a research-time constraint, but the planner/executor must respect it.
- **No explicit ESLint/Prettier/Dart-lint config detected** in CLAUDE.md — `flutter_lints: ^6.0.0` is present in `pubspec.yaml`'s `dev_dependencies` (confirmed via this session's read), so standard Flutter lint rules apply even though CLAUDE.md doesn't separately document them.

No CLAUDE.md directive conflicts with any recommendation in this research.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Compass button behavior (NAV-03)**
- **D-01:** Keep today's toggle behavior, not maplibre's native reset-only default. The compass button continues to toggle between north-up and heading-up map rotation (tap once to start following GPS heading, tap again to snap back to north) — this is the validated "north-up/heading-up map orientation toggle" requirement from PROJECT.md and must not regress. Implement via `ml.MapCompass`'s `onPressed` override (it explicitly supports overriding the default tap behavior) calling `controller.trackLocation(trackLocation: true, trackBearing: _headingUp ? BearingTrackMode.gps : BearingTrackMode.none)`, plus an explicit `animateCamera(bearing: 0, ...)` when turning heading-up off (native `MapCompass`'s own default reset-to-north only fires on tap when `onPressed` is unset, so the explicit reset call must be preserved in the override).
- **D-02:** Compass stays always visible during navigation (`hideIfRotatedNorth: false`) — contrast with Phase 16's map screen which used `hideIfRotatedNorth: true`. Rationale: during turn-by-turn the compass is a primary orientation control the hiker actively watches/uses, not an occasional map-browsing aid (map screen's rationale for hiding it doesn't apply here).

**Recenter + follow-mode state coupling (NAV-02)**
- **D-03:** Recenter restores the exact prior state, including heading-up if it was active before the drag broke follow — `trackLocation(trackLocation: true, trackBearing: previousBearingMode)`, mirroring today's independent `_followEnabled`/`_headingUp` booleans (dragging away doesn't reset `_headingUp`; recenter re-engages follow using whatever `_headingUp` was already set to). Do not simplify to "recenter always resets to north" — this would be a real behavior regression on repeated pan-then-recenter cycles during heading-up navigation.
- Drag-only breaks follow (not pinch/zoom) — already locked by ROADMAP.md's success criterion 2 ("matching today's `MapEventMoveStart`/`dragStart` behavior"), not re-litigated here. Maps to maplibre's `MapEventStartMoveCamera` with the appropriate `CameraChangeReason` (confirm exact reason value during research/planning — Phase 16's `map_screen.dart` already distinguishes `CameraChangeReason.apiGesture` for its own drag-detection use, establishing the pattern to follow).

**Location puck visuals (CORE-07)**
- **D-04:** Accept maplibre's native `enableLocation()` puck with its default visual options (`pulseFade: true`, `accuracyAnimation: true`, `pulse: true`, `compassAnimation: true`) — do not attempt to visually match today's custom 18px blue dot (`WandererMap`'s puck, reused by `map_screen.dart`). CORE-07 requires the puck itself to come from `enableLocation`/`trackLocation`, which has no custom color/size API — the native puck's appearance (GPS accuracy pulse/circle) is a deliberate, accepted visual change, not a regression to fix.
- `bearingRenderMode`/`trackBearing` values: use `BearingRenderMode.gps` / `BearingTrackMode.gps` (GPS-derived heading), matching today's `LocationMarkerDataStreamFactory.fromGeolocatorPositionStream` — no compass-sensor mode discussed or needed, GPS heading is what the app already uses via `TraceletPositionSource`.

### Claude's Discretion
- Exact camera animation durations for recenter/heading-up transitions (short, native, non-`Duration.zero` per the Phase 16 checkpoint lesson — `Duration.zero` crashes the Android native binding's `animateCamera`/`fitBounds`; use a short non-zero duration like Phase 16's established 400ms–750ms range).
- Whether the legacy flutter_map `TrailLayer` widget (restored in `trail_layer.dart` during Phase 15's checkpoint specifically for this screen, doc-commented "Phase-17 deletion") gets physically deleted in this phase or left dead-but-unused until Phase 18's cleanup pass — functionally must stop being *called* by `navigation_screen.dart` (switch to the native `addTrailTrackLayers`/`TrailMarkerLayer` functions already used by `WandererMap`), deletion timing is not user-facing.
- Exact widget/provider structure for wiring `enableLocation`/`trackLocation` into a `ConsumerStatefulWidget` lifecycle (mirrors `WandererMap`'s established `onMapCreated` hand-off pattern from Phase 15 — see canonical refs).
- Offline vector tile source wiring for the navigation screen's `MapLibreMap` (replacing `VectorTileLayer`/`MultiPmTilesVectorTileProvider` with the `pmtiles://file://` + `rewriteStyleForOffline` pattern `WandererMap` already established in Phase 15) — mechanical port, no user decision needed.

### Deferred Ideas (OUT OF SCOPE)
- Compass icon redesign to match `ml.MapCompass`'s native look-and-feel more closely, or vice versa — not raised as a concern; native default accepted implicitly by not being discussed further.
- Physically deleting the legacy flutter_map `TrailLayer` widget and `pm_tile_provider.dart` this phase vs. leaving them dead until Phase 18 — left to Claude's discretion (not user-facing either way).

**Phase boundary (from CONTEXT.md `<domain>`):** Out of scope for this phase: deleting `flutter_map`/plugins from `pubspec.yaml` (CLEAN-01/02/03 — Phase 18); any change to maneuver logic, Valhalla integration, offline ObjectBox caching behavior, or the stats sheet's content (all v1.0/v1.1 behavior, unregressed per ROADMAP success criterion 3).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| NAV-01 | Navigation renders the route line, the user's position, and heading-up follow on maplibre | `addTrailTrackLayers` (route line, existing pattern) + `enableLocation`/`trackLocation` (puck + follow) — see Pattern 1, Code Examples |
| NAV-02 | Dragging the map during navigation breaks follow mode, and the recenter control restores it — matching today's `MapEventMoveStart`/`dragStart` behavior | Pitfall 1 + Pattern 2 (pointer-count heuristic, since `CameraChangeReason` alone cannot distinguish drag from pinch/rotate) + D-03 recenter-restores-prior-state code example |
| NAV-03 | The compass control resets bearing to north with an animated camera transition | Pattern 3 (`ml.MapCompass.onPressed` override, verified `_onTap` source semantics) |
| NAV-04 | Offline navigation continues to serve maneuvers from the ObjectBox cache (v1.1 behavior, unregressed) | Untouched by this phase (confirmed: `navigationProvider`/ObjectBox cache logic is orthogonal to the map-rendering swap) — see Open Question 2 for the one new cross-cutting check (native puck rendering during the existing offline test) |
| CORE-05 | The compass uses maplibre's built-in `MapCompass`; `lib/components/map/map_compass.dart` is deleted | Pattern 3; `map_compass.dart` has exactly one remaining importer (`navigation_screen.dart`) per repo grep |
| CORE-06 | Camera animations use maplibre's native `animateCamera`/`fitBounds`; `flutter_map_animations` and every `AnimatedMapController` reference are gone | Verified native signatures (`animateCamera`, `fitBounds`) in Standard Stack/Code Examples; `AnimatedMapController` has exactly one remaining call site (`navigation_screen.dart`) |
| CORE-07 | The user's location puck and heading-up follow use maplibre's `enableLocation`/`trackLocation`; `flutter_map_location_marker` is gone | Full API verification in Summary/Pattern 1/Pitfalls 2-5 — the phase's core new-ground work |

</phase_requirements>

## Summary

`navigation_screen.dart` is the last `flutter_map` screen in the app. Its four `flutter_map`-only dependencies — `AnimatedMapController`, `CurrentLocationLayer`/`LocationMarkerDataStreamFactory`, the app-local `MapCompass` widget, and `MultiPmTilesVectorTileProvider` — all have direct native-`maplibre` replacements that Phase 15/16 already exercised for other screens (`WandererMap`, `SearchMap`, `map_screen.dart`). The mechanical parts of this migration (offline style rewrite, trail track/casing/arrows, `ml.MapCompass`, `fitBounds`/`animateCamera`, `onMapCreated`/`onStyleLoaded` hand-off, `Duration.zero` crash avoidance) are low-risk — they are the same patterns already proven on-device in Phase 15/16.

The one genuinely new, non-mechanical piece of this phase is `enableLocation`/`trackLocation` (CORE-07) — **no prior phase has called these APIs**; `map_screen.dart`'s `_LocationLayer` is a hand-rolled static `WidgetLayer` puck, not the native follow-tracking component. Direct inspection of the `maplibre_android`/`maplibre_ios` package source turned up three findings the CONTEXT.md's stated assumptions got wrong or left unresolved, all now answered with source-level confidence:

1. **`enableLocation` silently no-ops if called before the style has loaded** (Android: `if (style == null) return;`). It must be called from `onStyleLoaded`, and — because a `setStyle` theme swap replaces the native `Style` object the `LocationComponent` was activated against — it likely must be **re-called on every `onStyleLoaded`**, mirroring the already-established `addTrailTrackLayers` re-add pattern.
2. **`enableLocation`/`trackLocation` do NOT take a position-stream parameter.** They activate the native platform's own GPS engine (Android's `LocationComponent` + `LocationEngineRequest`, iOS's `MLNMapView.showsUserLocation`/`userTrackingMode`) directly — completely independent of the app's `TraceletPositionSource`/`_positionStream`. CONTEXT.md's phrasing ("feeding `enableLocation`/`trackLocation` instead of `CurrentLocationLayer`") implies a stream hookup that does not exist in this API. `TraceletPositionSource` keeps feeding `navigationProvider`/`navigationStatsProvider` unchanged; the map puck runs on a second, independent native location subscription. This is a real architectural correction the planner needs, not a nuance.
3. **`CameraChangeReason` has exactly three values** (`developerAnimation`, `apiAnimation`, `apiGesture`) and the Android JNI binding confirms `apiGesture` fires for `REASON_API_GESTURE` — the underlying MapLibre/Mapbox Android SDK constant that covers *every* user touch gesture (pan, pinch-zoom, rotate, fling) with no sub-classification. There is **no drag-vs-pinch distinction available at the `MapEventStartMoveCamera` level**, unlike `flutter_map`'s `MapEventSource` enum which has a dedicated `dragStart` distinct from `multiFingerGestureStart`. Success criterion 2 ("matching today's `MapEventMoveStart`/`dragStart` behavior") cannot be satisfied by reading `event.reason` alone — a supplementary heuristic (raw pointer-count tracking, detailed below) is required and must be flagged for on-device verification, continuing this project's established pattern of native-only bugs surfacing on-device (Phase 15/16 checkpoints).

**Primary recommendation:** Reuse `WandererMap`'s `onMapCreated`/`onStyleLoaded` hand-off and `SearchMap`'s style-loaded-buffering fix as the template for a new nav-specific host (or inline `ml.MapLibreMap` in `navigation_screen.dart` directly, given its bespoke breadcrumb/maneuver/waypoint-sheet state doesn't fit `SearchMap`'s or `WandererMap`'s existing contracts). Call `enableLocation` once per `onStyleLoaded` (idempotent, cheap). Drive follow/heading-up purely through `trackLocation(trackLocation:, trackBearing:)` — no manual `_recenterTrigger` stream is needed since the native camera auto-follows once tracking is engaged. Detect drag-only follow-break with a `Listener`-based pointer-count wrapper around the map (see Pitfall 1), not `CameraChangeReason` alone. Add the breadcrumb as its own `GeoJsonSource`/`LineStyleLayer` pair (imperative, `updateGeoJsonSource` on every position update), matching the existing `cluster_layer.dart` "swap source data in place" pattern — not a declarative `MapLibreMap.layers` `ml.PolylineLayer`, which would remove+re-add the layer on every single GPS fix.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Route line / breadcrumb rendering | Client (Flutter, native GL layer) | — | Native `LineStyleLayer`/`GeoJsonSource`, same tier as Phase 15's `addTrailTrackLayers` |
| Location puck + heading-up follow | Client (Flutter, native platform location engine) | OS location services (GPS/fused provider) | `enableLocation`/`trackLocation` delegate entirely to the native Android `LocationComponent` / iOS `MLNMapView` — no app-tier stream involved |
| Maneuver progress / stats tracking | Client (Flutter, app-tier providers) | OS location services (via `TraceletPositionSource`) | Untouched by this phase — `navigationProvider`/`navigationStatsProvider` keep consuming `TraceletPositionSource`, a *separate* GPS subscription from the map's native puck |
| Compass control (bearing reset) | Client (Flutter, native GL widget) | — | `ml.MapCompass` reads `MapController.getCamera().bearing` directly from the native map; no app-tier state needed except the `_headingUp` toggle |
| Offline maneuver cache | Client (ObjectBox, on-device) | — | v1.1 behavior, unregressed (NAV-04) — this phase does not touch it |
| Offline basemap tiles | Client (native `pmtiles://file://` source) | Filesystem (downloaded `.pmtiles` cells) | Reuses `rewriteStyleForOffline`, WandererMap's established OFFL-02/03/05 pattern |
| Camera animation (recenter, follow) | Client (Flutter, native GL controller) | — | `animateCamera`/`fitBounds`/`trackLocation` are native-SDK calls; no server round-trip |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `maplibre` | 0.3.5 (resolved; pubspec pins `^0.3.3+2`) [VERIFIED: pub cache `.dart_tool/package_config.json`] | Native GL map, location, compass, camera control | Already the app's sole map engine since Phase 15; no alternative under consideration |
| `maplibre_android` / `maplibre_ios` / `maplibre_platform_interface` | 0.3.5 each | Platform bindings backing `maplibre` | Transitive deps of `maplibre`; already installed |

No new packages are introduced by this phase — CORE-05/06/07 and NAV-01..04 are entirely a matter of **removing** `flutter_map`/`flutter_map_animations`/`flutter_map_location_marker` call sites from one file and replacing them with `maplibre` APIs already present in `pubspec.yaml`. `flutter_map`/plugin *removal from pubspec.yaml* itself is out of scope (CLEAN-01, Phase 18) — this phase only needs the last import statements gone from `navigation_screen.dart`.

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `tracelet` (via `TraceletPositionSource`) | already installed | Background-capable GPS stream feeding maneuver/stats logic | Keep as-is — do not attempt to bridge it into `enableLocation` (no stream parameter exists, see Pitfall 2) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Native `enableLocation`/`trackLocation` puck | Reuse `WandererMap._buildLocationLayer`'s custom `WidgetLayer` static dot fed from `TraceletPositionSource` | Rejected by CORE-07's explicit requirement and D-04's acceptance of the native puck's visual — the custom dot has no built-in follow/heading-up camera behavior, which NAV-01/NAV-02 require |
| Imperative `GeoJsonSource`/`LineStyleLayer` for breadcrumb | Declarative `MapLibreMap.layers: [ml.PolylineLayer(...)]` (used once already in `map_screen.dart` for the single selected-trail polyline) | Declarative `layers:` diffs by index and calls `removeLayer`+`addLayer` whenever `Layer.list` changes (verified in `layer_manager.dart`) — fine for an occasional selection change, wasteful for a breadcrumb that grows on every GPS fix. Imperative `updateGeoJsonSource` (already used by `cluster_layer.dart`/`map_screen.dart` for CLUS-04) only patches the source data, no layer churn |

**Installation:** none required — no `pubspec.yaml` changes for this phase.

## Package Legitimacy Audit

Not applicable — this phase installs no new external packages. All APIs used (`enableLocation`, `trackLocation`, `ml.MapCompass`, `animateCamera`, `fitBounds`, `MapEventStartMoveCamera`) are part of the already-installed, already-audited `maplibre` 0.3.5 dependency tree.

## Architecture Patterns

### System Architecture Diagram

```
TraceletPositionSource (background GPS)              Native platform LocationComponent
        │                                             (Android LocationEngine / iOS CoreLocation)
        ▼                                                      │  (enableLocation, own GPS subscription)
navigationProvider.onPosition()                                ▼
navigationStatsProvider.onPosition()                   MapLibreMap native puck
        │                                              (pulse/accuracy circle, GPS-heading arrow)
        ▼                                                      │
 maneuver index / breadcrumb list                    trackLocation(trackLocation, trackBearing)
        │                                                      │  (camera auto-follows while engaged)
        ▼                                                      ▼
 ┌─────────────────────────── navigation_screen.dart (ConsumerStatefulWidget) ───────────────────────────┐
 │                                                                                                         │
 │   onMapCreated(controller) ──► captures ml.MapController                                                │
 │   onStyleLoaded(style)     ──► addTrailTrackLayers(style, trail)   [route: casing+route+arrows]         │
 │                             ──► style.addSource('breadcrumb') + addLayer('breadcrumb-route') (once)     │
 │                             ──► controller.enableLocation(...)     [re-armed every style load]          │
 │                             ──► controller.trackLocation(trackLocation:true, trackBearing:_headingUp?)  │
 │                                                                                                          │
 │   onEvent(MapEventStartMoveCamera(reason: apiGesture))                                                  │
 │        + pointer-count wrapper (Listener) ──► if singleFingerDrag: trackLocation(trackLocation:false)   │
 │                                                                                                          │
 │   navState.breadcrumb changes ──► controller.style?.updateGeoJsonSource(id:'breadcrumb', data: ...)     │
 │                                                                                                          │
 │   MapCompass.onPressed ──► toggle _headingUp ──► trackLocation(trackBearing: gps|none)                  │
 │                         ──► if turning off: animateCamera(bearing: 0)                                   │
 │                                                                                                          │
 │   Recenter button ──► trackLocation(trackLocation:true, trackBearing:_headingUp?gps:none)  [D-03]       │
 └──────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure
No new files are structurally required. Options, in order of preference:

```
lib/routes/navigation_screen.dart      # inline ml.MapLibreMap (simplest — bespoke state doesn't fit SearchMap/WandererMap contracts)
lib/components/map/trail_layer.dart    # addTrailTrackLayers/TrailMarkerLayer already here — delete legacy TrailLayer widget once nav switches off it
lib/components/map/map_compass.dart    # DELETED (CORE-05)
lib/vendor/vector_map_tiles/pm_tile_provider.dart  # DELETED (OFFL-06) — this phase is its last caller
lib/util/map_coordinate_adapter.dart   # last caller removed here; leave the file itself (Phase 18/CLEAN scope) unless it becomes fully dead — grep other callers before deleting (see Pitfall 6)
```

### Pattern 1: onStyleLoaded re-arm for anything that binds to the native `Style` object
**What:** `addTrailTrackLayers`, the breadcrumb source/layer, AND `enableLocation` all must be (re-)invoked inside `onStyleLoaded`, not just once at `onMapCreated`.
**When to use:** Any native GL construct that references the current `Style` instance — `setStyle` (CORE-02 theme swap) replaces that instance, and both layers (documented, `15-05` comment) and the location component (verified via source, `activateLocationComponent(activationOptions)` takes `style._jStyle` at build time) are almost certainly invalidated when the underlying style is swapped.
**Example:**
```dart
// Source: app/lib/components/base/wanderer_map.dart (15-05 established pattern), extended here
onStyleLoaded: (style) async {
  _fitInitialCamera().ignore();
  if (widget.trail.expand?.gpx != null) {
    await addTrailTrackLayers(style, widget.trail); // re-added every style load
  }
  await style.addSource(ml.GeoJsonSource(id: 'breadcrumb', data: _breadcrumbGeoJson()));
  await style.addLayer(const ml.LineStyleLayer(
    id: 'breadcrumb-route',
    sourceId: 'breadcrumb',
    paint: {'line-color': '#DC2626', 'line-width': 3.5},
  ));
  final controller = _controller;
  if (controller != null) {
    await controller.enableLocation(bearingRenderMode: ml.BearingRenderMode.gps);
    await controller.trackLocation(
      trackLocation: _followEnabled,
      trackBearing: _headingUp ? ml.BearingTrackMode.gps : ml.BearingTrackMode.none,
    );
  }
},
```

### Pattern 2: Drag-only follow-break via pointer-count wrapper (no native distinction exists)
**What:** Since `CameraChangeReason.apiGesture` fires identically for drag, pinch, and rotate (verified: Android `REASON_API_GESTURE`, JNI binding `map_state.dart` line ~100), wrap the map in a `Listener` that tracks active pointer count. Treat `apiGesture` as a drag-break trigger **only if exactly one pointer was down** when the gesture started.
**When to use:** NAV-02's drag-only follow-break requirement.
**Example:**
```dart
// Pattern derived from source-code analysis (no direct precedent in this codebase yet —
// flag as new, needs on-device verification per this project's established pattern of
// native-gesture-arena surprises, Phase 16-03 checkpoint).
int _activePointers = 0;

Widget build(BuildContext context) {
  return Listener(
    onPointerDown: (_) => _activePointers++,
    onPointerUp: (_) => _activePointers = (_activePointers - 1).clamp(0, 10),
    onPointerCancel: (_) => _activePointers = (_activePointers - 1).clamp(0, 10),
    child: ml.MapLibreMap(
      // ...
      onEvent: (event) {
        if (event is ml.MapEventStartMoveCamera &&
            event.reason == ml.CameraChangeReason.apiGesture &&
            _activePointers <= 1 &&
            _followEnabled) {
          _controller?.trackLocation(trackLocation: false);
          setState(() => _followEnabled = false);
        }
      },
    ),
  );
}
```
**Caveat:** This is a heuristic, not a guarantee — a user could rest a second finger on the screen without it registering as a scale gesture, or `MapEventStartMoveCamera` could fire a frame before/after the second `onPointerDown`. Flag as `checkpoint:human-verify` in the plan; test explicitly: (a) one-finger pan breaks follow, (b) two-finger pinch-zoom does NOT break follow, (c) two-finger rotate does NOT break follow.

### Pattern 3: Compass toggle with explicit north-reset override (D-01)
**What:** `ml.MapCompass.onPressed` fully replaces the default tap behavior (verified: `_onTap` calls `controller.animateCamera(bearing:...)` only `if (rotateNorthOnPressed || removePinchOnPressed)`, THEN calls `onPressed?.call()` — both fire, so setting `rotateNorthOnPressed: false` and supplying `onPressed` is the correct way to fully take over, not layer on top of the default).
**Example:**
```dart
// Source: maplibre-0.3.5/lib/src/ui/map_compass.dart (verified _onTap logic)
ml.MapCompass(
  hideIfRotatedNorth: false, // D-02
  rotateNorthOnPressed: false, // prevent double-handling; onPressed does the reset explicitly (D-01)
  onPressed: () {
    setState(() => _headingUp = !_headingUp);
    final controller = _controller;
    if (controller == null) return;
    controller.trackLocation(
      trackLocation: _followEnabled,
      trackBearing: _headingUp ? ml.BearingTrackMode.gps : ml.BearingTrackMode.none,
    );
    if (!_headingUp) {
      controller.animateCamera(bearing: 0, nativeDuration: const Duration(milliseconds: 400));
    }
  },
)
```

### Anti-Patterns to Avoid
- **Feeding `_positionStream` into `enableLocation`:** There is no stream parameter on `enableLocation`/`trackLocation` (verified against `maplibre_platform_interface`'s `MapController` interface). Don't try to construct one — the native engine manages its own GPS subscription.
- **Relying on `CameraChangeReason` alone for drag detection:** Confirmed only 3 enum values exist and Android's `apiGesture` maps 1:1 to the native SDK's single "any gesture" reason — it cannot distinguish pan from pinch.
- **Calling `enableLocation` from `onMapCreated`:** Android silently no-ops (`if (style == null) return;`) since `style` is only set inside `_onStyleLoaded`. iOS doesn't have this restriction, but calling it consistently from `onStyleLoaded` on both platforms avoids an Android-only "call it and nothing happens" bug.
- **Declarative `ml.PolylineLayer` in `MapLibreMap.layers` for the breadcrumb:** Causes a remove+re-add of the style layer on every GPS-driven `setState`, per `LayerManager.updateLayers`'s equality check (`layer != oldLayer` is true whenever `Layer.list` changes, since `list` is part of `==`). Use `addSource` once + `updateGeoJsonSource` per update instead.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Location puck rendering, pulse/accuracy animation | A custom `WidgetLayer` marker driven by `TraceletPositionSource` (as `WandererMap`/`map_screen.dart` currently do for their non-navigation use cases) | `controller.enableLocation(...)` | CORE-07 explicitly requires the native API; D-04 already accepted its default visual |
| Heading-up camera follow | Manual `animateCamera(bearing: currentHeading)` on every GPS fix | `controller.trackLocation(trackBearing: BearingTrackMode.gps)` | The native `LocationComponent`/`MLNMapView.userTrackingMode` already re-centers/re-bears the camera per fix — reimplementing this in Dart would fight the native component and likely stutter |
| North-bearing reset animation | Custom `Tween`/`AnimationController` rotating the map | `controller.animateCamera(bearing: 0, nativeDuration: ...)` | Native camera animation, already the established pattern (`WandererMap._fitInitialCamera`, `map_screen.dart`'s cluster-tap zoom) |

**Key insight:** Everything CORE-05/06/07 ask for already has a 1:1 native API — the only actual engineering work in this phase is (a) the pointer-count drag heuristic (no native equivalent exists) and (b) correctly sequencing `enableLocation`/`trackLocation`/layer calls relative to `onStyleLoaded`.

## Common Pitfalls

### Pitfall 1: `CameraChangeReason.apiGesture` cannot distinguish drag from pinch/rotate
**What goes wrong:** A plan that reads `event.reason == ml.CameraChangeReason.apiGesture` to detect "user dragged the map" will also break follow mode on pinch-zoom or two-finger rotate — a real regression versus today's `MapEventSource.dragStart`-gated behavior.
**Why it happens:** The `maplibre` package's `CameraChangeReason` enum is a thin wrapper over the underlying native MapLibre/Mapbox Android SDK's `OnCameraMoveStartedListener` reason codes (`REASON_API_GESTURE` / `REASON_API_ANIMATION` / `REASON_DEVELOPER_ANIMATION`), which itself doesn't sub-classify gesture types — this is an upstream native-SDK limitation, not a bug in the Dart wrapper.
**How to avoid:** Wrap the map in a `Listener` tracking `_activePointers`; only treat `apiGesture` as a drag-break if `_activePointers <= 1` at the moment the event fires (Pattern 2 above).
**Warning signs:** On-device testing shows follow mode breaking during pinch-zoom-in on the trail.

### Pitfall 2: `enableLocation`/`trackLocation` are not fed by `TraceletPositionSource`
**What goes wrong:** A plan/implementation that tries to pass `_positionStream` (or any position stream) into `enableLocation` will fail to compile — no such parameter exists. Worse, if a developer assumes the two systems are unified, they might remove `TraceletPositionSource`'s stream subscription "since the map now handles location" — which would break `navigationProvider.onPosition()`/`navigationStatsProvider.onPosition()` (maneuver advancement and stats), since those are NOT wired to the native puck at all.
**Why it happens:** CONTEXT.md's phrasing ("only the consumption point... changes") assumed a stream hand-off that the API doesn't support.
**How to avoid:** Keep `TraceletPositionSource`/`_sub`/`initState`/`dispose` exactly as they are today. `enableLocation`/`trackLocation` are additive, operating on a second, independent native GPS subscription purely for map rendering.
**Warning signs:** Maneuver banner stops advancing, or stats sheet freezes, while the map puck still moves — a sign the two systems got conflated during refactor.

### Pitfall 3: `enableLocation` silently no-ops before the style loads
**What goes wrong:** Calling `controller.enableLocation(...)` from `onMapCreated` (matching WandererMap's captured-controller pattern for other calls) does nothing on Android — no error, no puck, no exception.
**Why it happens:** Android's implementation does `final style = this.style; if (style == null) return;` — `style` is only non-null after the first `onStyleLoaded` fires.
**How to avoid:** Call `enableLocation` (and `trackLocation`) from inside `onStyleLoaded`, not `onMapCreated`.
**Warning signs:** No location puck ever appears on Android but works fine on iOS during dev testing (iOS's `enableLocation` only requires `_mapView != null`, set at map-creation time — an asymmetry that will hide this bug on iOS-only testing).

### Pitfall 4: theme-swap (`setStyle`) likely drops the active location component
**What goes wrong:** If the app is toggled light/dark (or the offline glyph cache warms and triggers `_swapStyle`, per `WandererMap`'s CORE-02 pattern) mid-navigation, the location puck may silently disappear, since `_onStyleLoaded` creates a *new* `jni.Style` object and the `LocationComponent` was `activateLocationComponent`'d against the old one.
**Why it happens:** `activateLocationComponent(activationOptions)` is built from `style._jStyle` at call time (verified in `maplibre_android`'s `map_state.dart`) — there's no evidence the component migrates across a style reload.
**How to avoid:** Re-call `enableLocation`+`trackLocation` on every `onStyleLoaded`, matching the already-established `addTrailTrackLayers` re-add pattern (documented in `wanderer_map.dart`: "(re)add ... after every style load, so they survive the CORE-02 theme swap").
**Warning signs:** Puck vanishes after a manual theme toggle during a live navigation session (not caught by static analysis — must be tested on-device).

### Pitfall 5: gesture-based auto-cancel of camera tracking may not be enabled by default
**What goes wrong:** A plan might assume the native `LocationComponent` automatically disengages tracking mode when the user drags (a common native-SDK convenience, exposed as `LocationComponentOptions.trackingGesturesManagement`). The `maplibre_android` Dart wrapper's `enableLocation` builds `LocationComponentOptions` WITHOUT setting `trackingGesturesManagement` (only `pulseFadeEnabled`/`accuracyAnimationEnabled`/`compassAnimationEnabled`/`pulseEnabled` are wired — verified against `map_state.dart`), even though the native JNI binding for `trackingGesturesManagement` exists at a lower level.
**Why it happens:** The Dart package doesn't expose this option; native SDK default (commonly `false` in Mapbox/MapLibre Android — `[ASSUMED]`, not independently re-verified this session) likely means tracking does NOT auto-cancel.
**How to avoid:** Rely entirely on the explicit `MapEventStartMoveCamera`+pointer-count heuristic (Pattern 2) to call `trackLocation(trackLocation: false)` — don't assume the native component handles this itself.
**Warning signs:** Dragging the map does nothing to the camera (it keeps snapping back to the puck) because tracking never got disengaged.

### Pitfall 6: `map_coordinate_adapter.dart` may become fully dead but has other callers
**What goes wrong:** Deleting `map_coordinate_adapter.dart` outright once `navigation_screen.dart` stops importing it could break other files still using it.
**Why it happens:** `grep -rl flutter_map` shows `map_coordinate_adapter.dart` itself still exists and is imported by exactly `navigation_screen.dart` today among route files, but the file's own implementation likely still depends on `latlong2`/`flutter_map` types (it converts `Geographic`↔`LatLng`) — confirm zero remaining importers before deleting; this phase's CONTEXT.md explicitly scopes deletion of `pm_tile_provider.dart` and `map_compass.dart` but is silent on this adapter file, which is arguably now unused too. **Recommend leaving it in place** (Phase 18/CLEAN-01 cleanup scope) unless a grep after this phase's edits confirms zero remaining callers.
**How to avoid:** `grep -rl "map_coordinate_adapter" app/lib/` after implementation; only delete if the result is empty.
**Warning signs:** A stray unused-import lint warning if left with zero callers — harmless, but note it for Phase 18.

## Code Examples

### Full `enableLocation`/`trackLocation` initial wiring
```dart
// Source: maplibre_platform_interface-0.3.5/lib/src/map_controller.dart (verified interface)
onStyleLoaded: (style) async {
  final controller = _controller;
  if (controller == null) return;
  await controller.enableLocation(
    bearingRenderMode: ml.BearingRenderMode.gps, // D-04: GPS heading, not compass sensor
  );
  await controller.trackLocation(
    trackLocation: _followEnabled,
    trackBearing: _headingUp ? ml.BearingTrackMode.gps : ml.BearingTrackMode.none,
  );
},
```

### Recenter restoring prior heading-up state (D-03)
```dart
void _onRecenter() {
  setState(() => _followEnabled = true);
  _controller?.trackLocation(
    trackLocation: true,
    trackBearing: _headingUp ? ml.BearingTrackMode.gps : ml.BearingTrackMode.none,
  );
}
```

### Breadcrumb as an imperative, in-place-updated source (avoids layer churn)
```dart
// Pattern precedent: app/lib/components/map/cluster_layer.dart + map_screen.dart's
// `updateClusterSource` call in `ref.listen(mapClusterSearchProvider, ...)` (CLUS-04).
ref.listen(navigationProvider(widget.response), (prev, next) {
  final controller = _controller;
  final style = controller?.style;
  if (style == null) return;
  if (prev?.breadcrumb == next.breadcrumb) return;
  style.updateGeoJsonSource(
    id: 'breadcrumb',
    data: jsonEncode({
      'type': 'Feature',
      'properties': <String, Object?>{},
      'geometry': {
        'type': 'LineString',
        'coordinates': [for (final p in next.breadcrumb) [p.lon, p.lat]],
      },
    }),
  );
});
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `flutter_map` + `flutter_map_location_marker`'s `CurrentLocationLayer` fed by a `Stream<LocationMarkerPosition>` | `maplibre`'s `enableLocation`/`trackLocation`, self-managed native GPS subscription | This phase (CORE-07) | Location rendering decouples entirely from app-tier position streams; simpler code, but two independent GPS subscriptions now run concurrently (native puck + `TraceletPositionSource`) |
| `AnimatedMapController.animateTo(rotation: 0)` | `controller.animateCamera(bearing: 0, nativeDuration: ...)` | This phase (CORE-06) | Same visual effect, native-driven animation instead of Flutter-tween-driven |
| App-local `MapCompass` (flutter_map-only, `MapCamera.of(context)`) | `ml.MapCompass` (native-map-aware, `MapController.maybeOf(context)`) | This phase (CORE-05) | Icon changes (native default, not app's "N + carets"); accepted per deferred item, not discussed further |
| `MapEventSource.dragStart` sub-classified gesture detection | `CameraChangeReason.apiGesture` (undifferentiated) + manual pointer-count heuristic | This phase (NAV-02) | Real capability loss versus `flutter_map`; must be compensated in application code, not assumed away |

**Deprecated/outdated:**
- `flutter_map_animations`' `AnimatedMapController`: gone from this file (CORE-06); package removal from `pubspec.yaml` deferred to Phase 18 (CLEAN-01).
- `flutter_map_location_marker`'s `LocationMarkerDataStreamFactory.fromGeolocatorPositionStream`: gone from this file (CORE-07); package removal deferred to Phase 18.
- `MultiPmTilesVectorTileProvider` (`pm_tile_provider.dart`): this phase is its last caller — OFFL-06 unblocks; file deletion is Claude's discretion this phase per CONTEXT.md (functionally must stop being called; physical deletion timing flexible).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Native `LocationComponentOptions.trackingGesturesManagement` defaults to `false` (no auto-cancel of tracking on user gesture) in the version of the MapLibre Android SDK bundled by `maplibre_android` 0.3.5 | Pitfall 5 | If actually `true` by default, the manual pointer-count drag-break (Pattern 2) becomes redundant-but-harmless — low risk either way, since the manual handler is a superset behavior, not a conflicting one |
| A2 | Deactivating a `LocationComponent` bound to a stale `Style` after `setStyle` behaves as "silently stops rendering" rather than throwing — inferred from the same-shaped pattern already documented for style-bound layers (`addTrailTrackLayers`'s re-add requirement), not independently exercised for `LocationComponent` this session | Pitfall 4 | If it instead throws or crashes (rather than silently no-op), the plan's `checkpoint:human-verify` for on-device testing will surface it before ship — no undetected-regression risk, just possibly a slightly different error mode than described |

**Note:** All other claims in this research (`enableLocation`/`trackLocation` signatures, `CameraChangeReason` enum values, `MapCompass.onPressed` override semantics, `LayerManager` id-collision-avoidance, `StyleController.updateGeoJsonSource` existence, Android's `style == null` early-return) are `[VERIFIED: maplibre/maplibre_android/maplibre_ios/maplibre_platform_interface 0.3.5 package source, ~/.pub-cache]` — read directly from the installed package source this session, not training-data recall.

## Open Questions

1. **Does the pointer-count `Listener` wrapper reliably observe pointer events through the platform view's hybrid-composition gesture arena?**
   - What we know: Android's `AndroidViewSurface` is configured with `hitTestBehavior: PlatformViewHitTestBehavior.opaque` and its own `gestureRecognizers` set (verified in `map_state.dart`); Flutter's pointer-event dispatch generally still notifies ancestor `Listener`s of `PointerDownEvent`/`PointerUpEvent` for hit-testable regions even when a platform view claims the gesture, since pointer routing happens before gesture-arena resolution.
   - What's unclear: whether `opaque` hit-test behavior on this specific platform view configuration suppresses ancestor `Listener` pointer events entirely — this is package/Flutter-engine-version-specific behavior not verifiable from static source alone.
   - Recommendation: The plan should mark the pointer-count heuristic implementation as `checkpoint:human-verify` and test explicitly on a physical device (not the simulator, consistent with this project's established need for physical-device GPS/gesture testing) before considering NAV-02 done.

2. **Does `enableLocation`'s native puck render correctly with airplane-mode-but-GPS-enabled (the offline navigation scenario, NAV-04)?**
   - What we know: `enableLocation` uses `useDefaultLocationEngine(true)` with a `LocationEngineRequest` set to `PRIORITY_HIGH_ACCURACY` (Android) / `CoreLocation` via `showsUserLocation` (iOS) — both are OS-level GPS/GNSS subsystems that function without a network connection (assuming the device's location mode isn't restricted to network-only, which is a device-setting concern, not an app concern).
   - What's unclear: whether MapLibre Android's default `LocationEngineProvider.getBestLocationEngine()` selection could, on some device/OS combos, prefer a network-based fused provider that silently degrades in airplane mode rather than falling back to raw GPS — not verifiable from package source alone (this is Android OS/vendor-dependent behavior).
   - Recommendation: Reuse the same on-device offline test already required for NAV-04's ObjectBox-cached-maneuver verification (existing v1.1 test procedure) and additionally confirm the puck renders during that same offline device test — no new test infrastructure needed, just an added assertion to an existing checkpoint.

3. **Should navigation_screen build a lightweight dedicated host widget (third sibling to `WandererMap`/`SearchMap`) or inline `ml.MapLibreMap` directly?**
   - What we know: CONTEXT.md's "Established Patterns" section explicitly leaves this to planning/research, noting navigation's extra state (breadcrumb, maneuver banner, waypoint sheet) doesn't fit `SearchMap`'s trail-agnostic contract or `WandererMap`'s single-trail-display contract.
   - What's unclear: whether extracting a `NavigationMap` host widget (mirroring `SearchMap`) would meaningfully reduce duplication given `navigation_screen.dart` is already a single 835-line file with no other consumer of a nav-specific map host.
   - Recommendation: Inline `ml.MapLibreMap` directly in `navigation_screen.dart`'s existing `_NavigationScreenState`, applying the `SearchMap` buffering fix (style-loaded-before-map-created race) inline rather than extracting a new host class — there is no second call site to justify the abstraction, and CONTEXT.md's "Claude's Discretion" section explicitly leaves this open.

## Environment Availability

Skipped — this phase has no external tool/service dependencies beyond the already-installed `maplibre` package and OS-level GPS, both already verified present and working by Phase 15/16's on-device checkpoints.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V4 Access Control | No | No new endpoints or permission boundaries introduced |
| V5 Input Validation | Partial — reused, not new | `rewriteStyleForOffline`'s existing `_assertSafePath` guard (T-15-06-01/02) already rejects non-absolute/`..`/foreign-scheme paths; this phase's offline branch reuses that function unchanged, not a new validation surface |
| V8 Data Protection | Yes | GPS position data (breadcrumb, maneuver progress) stays on-device, rendered locally; no new network transmission of location data introduced by this phase |
| V6 Cryptography | No | Not applicable — no new crypto surface |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Malformed/oversized breadcrumb GeoJSON crashing the map render (analogous to T-16-02's cluster-response fail-soft) | Denial of Service | Wrap `updateGeoJsonSource` calls in try/catch, matching `map_screen.dart`'s established `T-16-02` fail-soft pattern for malformed cluster data — never let a bad breadcrumb payload crash live navigation |
| Style-injection via untrusted path components in the offline branch | Tampering | Already mitigated — `rewriteStyleForOffline`'s `_assertSafePath` (T-15-06-01/02) is reused unchanged, not reimplemented, for this phase's offline navigation branch |

## Sources

### Primary (HIGH confidence)
- `~/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/map_controller.dart` — `enableLocation`/`trackLocation` signatures, `BearingTrackMode`/`BearingRenderMode` enums
- `~/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/map_events.dart` — `MapEventStartMoveCamera`, `CameraChangeReason` (3 values), `MapEventCameraIdle`
- `~/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/layer/layer.dart` + `layer_manager.dart` — declarative-layer id scheme and update/diff semantics
- `~/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/style_controller.dart` — `addSource`/`addLayer`/`updateGeoJsonSource`/`removeLayer`/`removeSource` signatures
- `~/.pub-cache/hosted/pub.dev/maplibre-0.3.5/lib/src/ui/map_compass.dart` — `MapCompass` full source, `_onTap` override semantics
- `~/.pub-cache/hosted/pub.dev/maplibre-0.3.5/lib/src/layer/polyline_layer.dart` — declarative `PolylineLayer` id generation
- `~/.pub-cache/hosted/pub.dev/maplibre_android-0.3.5/lib/src/map_state.dart` — `enableLocation`/`trackLocation` Android impl (style-null guard, `LocationComponentOptions` builder, `CameraChangeReason` JNI mapping, `_onStyleLoaded` style replacement), `trackingGesturesManagement` absence
- `~/.pub-cache/hosted/pub.dev/maplibre_ios-0.3.5/lib/src/map_state.dart` — `enableLocation`/`trackLocation` iOS impl (`showsUserLocation`, `userTrackingMode`, no style dependency)
- This repo: `app/lib/routes/navigation_screen.dart`, `app/lib/components/base/wanderer_map.dart`, `app/lib/components/map/trail_layer.dart`, `app/lib/routes/map_screen.dart`, `app/lib/util/offline_style_rewriter.dart`, `app/lib/util/tracelet_position_source.dart`, `app/lib/components/map/cluster_layer.dart`, `app/pubspec.yaml` — existing patterns and current-state grep

### Secondary (MEDIUM confidence)
- `.planning/phases/15-*/15-06-SUMMARY.md`-referenced decisions (via STATE.md's Accumulated Context) — `Duration.zero` Android crash, style-swap layer re-add requirement, cited but not re-read in full this session

### Tertiary (LOW confidence)
- None — no unverified WebSearch-only claims were used in this research; all API-level claims trace to package source read directly this session.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages, all APIs read from installed source
- Architecture: HIGH — hand-off patterns directly extend Phase 15/16's proven, on-device-verified `WandererMap`/`SearchMap` patterns
- Pitfalls: MEDIUM-HIGH — API-level pitfalls (1-4, 6) are source-verified; Pitfall 5's native default and Open Question 1/2's platform-view gesture-routing behavior are flagged for on-device confirmation, consistent with this project's established need for physical-device checkpoints on gesture/GPS behavior

**Research date:** 2026-07-09
**Valid until:** 30 days (stable — `maplibre` is pre-1.0 but this phase pins to the already-resolved 0.3.5; re-verify if `pubspec.lock` changes the resolved version before this phase executes)
