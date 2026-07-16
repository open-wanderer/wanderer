# Architecture Research

**Domain:** Route Planner screen integration — Flutter/Riverpod/maplibre/go_router mobile app (Wanderer v1.5)
**Researched:** 2026-07-16
**Confidence:** HIGH — every claim below is backed by a specific file read in this repo (app/lib, web/src), not by general Flutter/Riverpod conventions. No external web research was needed; the codebase already contains a near-complete precedent (the web app's own route-planner-equivalent) and the exact map-interaction pattern the new screen needs.

## Standard Architecture

### System Overview

```
┌───────────────────────────────────────────────────────────────────────────┐
│  trail_source_select_screen.dart          (MODIFIED — entry point)        │
│    "Planner" card.onTap → context.push('/trail/create/plan')              │
└───────────────────────────────┬───────────────────────────────────────────┘
                                 ▼
┌───────────────────────────────────────────────────────────────────────────┐
│  router_provider.dart                      (MODIFIED — +1 top-level route)│
│    GoRoute('/trail/create/plan') → RoutePlannerScreen()                   │
└───────────────────────────────┬───────────────────────────────────────────┘
                                 ▼
┌───────────────────────────────────────────────────────────────────────────┐
│  route_planner_screen.dart                 (NEW — ConsumerStatefulWidget) │
│  ┌─────────────────────────────┐  ┌──────────────────────────────────┐   │
│  │ RoutePlannerMap (NEW)        │  │ DraggableScrollableSheet          │   │
│  │  wraps RoutePlannerMarkerLayer│  │  (waypoint list XOR elevation)   │   │
│  │  tap-to-add / drag / insert   │  │  toggled by _buildButtonRow-style │   │
│  └──────────────┬───────────────┘  └──────────────┬────────────────────┘  │
│                 │  ref.watch/read                  │ ref.watch            │
│                 ▼                                  ▼                      │
│         routePlannerProvider  (NEW — app/lib/provider/route_planner_      │
│         provider.dart, @riverpod class RoutePlanner)                      │
│           state: waypoints, autoRouting, profile, undo/redo stacks,       │
│                  synthesized in-memory Gpx                                │
└───────────────────────────────┬───────────────────────────────────────────┘
                                 │ imperative methods call out to:
                                 ▼
┌───────────────────────────────────────────────────────────────────────────┐
│  apiProvider (Dio, EXISTING) → SvelteKit (EXISTING, zero backend changes) │
│    POST /api/v1/valhalla/route   — per-waypoint-pair routed shape         │
│    POST /api/v1/valhalla/height  — elevation for the composed shape       │
└───────────────────────────────┬───────────────────────────────────────────┘
                                 ▼ on "Use this route"
┌───────────────────────────────────────────────────────────────────────────┐
│  trail_import_util.dart            (MODIFIED — +1 function, same         │
│    pendingImportedTrail global + handoff to)                             │
│  router_provider.dart '/trail/create/edit' (UNCHANGED — reused as-is)    │
│  trail_create_screen.dart                  (UNCHANGED — reused as-is)   │
└───────────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | File status |
|-----------|----------------|-------------|
| `RoutePlanner` provider | Owns in-progress waypoints, undo/redo, auto-routing flag, profile, synthesized `Gpx`; imperative add/move/insert/delete/reorder/undo/redo/resolveRouting methods | NEW — `app/lib/provider/route_planner_provider.dart` |
| `buildGpxFromPoints` (util) | Synthesizes a `package:gpx` `Gpx` from an in-memory point/elevation list — the reverse of `buildNavShape` | MODIFIED — added to `app/lib/util/gpx_util.dart` |
| `RoutePlannerMap` | Native GL map host for the planner: tap-to-add, per-marker drag, tap-on-segment insert | NEW — `app/lib/components/base/route_planner_map.dart` |
| `RoutePlannerMarkerLayer` | `ml.WidgetLayer` of draggable/deletable/reorderable waypoint markers + route polyline | NEW — `app/lib/components/map/route_planner_marker_layer.dart` |
| `RoutePlannerScreen` | Screen shell: map + waypoint-list/elevation sheet + control buttons + handoff button | NEW — `app/lib/routes/route_planner_screen.dart` |
| `handoffPlannedRoute()` | Builds a draft `Trail` from the planner's `Gpx`+waypoints, sets `pendingImportedTrail`, pushes `/trail/create/edit` | MODIFIED — added to `app/lib/util/trail_import_util.dart` |
| `router_provider.dart` | Registers `/trail/create/plan` | MODIFIED — +1 `GoRoute`, +1 import |
| `trail_source_select_screen.dart` | Wires the existing "Planner" card to the new route | MODIFIED — 1-line `onTap` change |
| `global_search_screen.dart` | Location-tile currently hardcodes `context.go('/map', ...)` | MODIFIED (small) — see Pitfall below |
| Valhalla `/route`, `/height` (SvelteKit) | Point-to-point routing and elevation lookup | UNCHANGED — already exist, unused by Flutter today |

## Recommended Project Structure

```
app/lib/
├── provider/
│   └── route_planner_provider.dart       # NEW — @riverpod class RoutePlanner, top-level
│                                          #   (sibling to navigation_provider.dart, NOT under
│                                          #   provider/trail/ — see rationale below)
├── components/
│   ├── base/
│   │   └── route_planner_map.dart        # NEW — modeled on trail_map.dart's shell
│   └── map/
│       └── route_planner_marker_layer.dart # NEW — modeled on trail_layer.dart's
│                                          #   TrailMarkerLayer, adds insert/delete/reorder
├── routes/
│   └── route_planner_screen.dart         # NEW — the screen itself
└── util/
    ├── gpx_util.dart                     # MODIFIED — + buildGpxFromPoints()
    └── trail_import_util.dart            # MODIFIED — + handoffPlannedRoute()
```

### Structure Rationale

- **`provider/route_planner_provider.dart` (top-level, not `provider/trail/`):** `provider/trail/*` providers are all backed by a persisted PocketBase `Trail` record (fetch/save/filter against `/trail`). The Route Planner's state has no backing record until handoff — it is ephemeral session state scoped to one screen's lifetime, the same shape as `navigation_provider.dart` (also top-level, also ephemeral, also holds an in-memory mutable path + progress). Following that precedent keeps `provider/trail/` reserved for record-backed providers.
- **New map host instead of reusing `TrailMap` directly:** `trail_create_screen.dart` already proves the *pattern* (synthesize a stub `Trail`, feed it to `TrailMap`, wire `onTap`/`onWaypointDragEnd`) — but `TrailMarkerLayer` only supports tap + drag, not the Route Planner's insert-mid-route / delete / reorder requirements. Extending `TrailMarkerLayer` in place would touch a component shared by `trail_detail_map_screen.dart`, `list_detail_map_screen.dart`, `navigation_screen.dart`, and `trail_create_screen.dart` — high blast radius for a v1.5-only need. A new sibling file isolates all planner-only interaction logic and matches the milestone's "addition, not rework" framing.
- **`gpx_util.dart` gets the new function, not a new file:** it is already the single home for every `package:gpx` ↔ app-type bridge (`buildNavShape`, `GpxMappingUtils`, `sanitizeGpxEmail`). `buildGpxFromPoints` is the structural inverse of `buildNavShape` and has zero Riverpod/UI dependencies — same profile as the existing functions in that file. A new file would fragment Gpx-construction logic across two places for no benefit.
- **`trail_import_util.dart` gets the handoff function, not a new file:** `pendingImportedTrail` is a single module-level global with a subtle race-condition safety net (documented in its own comment — go_router's `RouteMatchListCodec` can drop non-JSON `extra` on a same-process refresh). A second global in a new file would either duplicate that fragile logic or risk two competing "pending trail" globals. The Route Planner is conceptually the same case as GPX import — an unsaved `Trail` built client-side that must survive to `/trail/create/edit` — so it belongs next to the existing precedent.

## Architectural Patterns

### Pattern 1: Synthesized-stub-Trail map host (established, reused as-is)

**What:** A screen that doesn't yet have a persisted `Trail` builds a throwaway `Trail.empty().copyWith(expand: TrailExpand(gpx: ..., waypointsViaTrail: ...))` purely so it can be handed to a `Trail`-shaped map/form component.
**When to use:** Any screen that edits trail geometry/waypoints before a save exists.
**Trade-offs:** Avoids a parallel "unsaved trail" type, but requires placeholder `id: ''`/timestamps on nested `Waypoint`s (see `trail_import_util.dart` lines 76-89) so `Waypoint.fromJson`/`copyWith` don't choke on required non-nullable fields.

**Example (already shipping, `trail_create_screen.dart:417-438`):**
```dart
TrailMap(
  trail: trail, // stub or real Trail, both work identically
  onTap: (point) => _onCreateWaypoint(context, at: point),
  onWaypointTap: (wp) => _onEditWaypoint(context, wp),
  onWaypointDragEnd: _onWaypointMoved,
)
```
The Route Planner reuses this exact idea but via a new `RoutePlannerMap`/`RoutePlannerMarkerLayer` pair (see Pattern 2) rather than `TrailMap` itself, because it needs interactions `TrailMarkerLayer` doesn't have.

### Pattern 2: Widget-space GestureDetector markers, NOT a map-wide GestureDetector

**What:** Tap-to-add uses the **native** map click callback (`ml.MapEventClick` via `onEvent`, forwarded as `TrailMap.onTap`). Drag uses a **per-marker** `GestureDetector` (`onPanStart`/`onPanUpdate`/`onPanEnd`) inside each `ml.Marker`'s child within a single `ml.WidgetLayer`, converting screen-space `Offset` deltas back to `Geographic` via `controller.toLngLat`/`toScreenLocation`. There is no map-wide `GestureDetector` wrapping the whole map, and `maplibre` 0.3.5 does **not** expose a native marker-drag callback — drag is 100% Flutter-side.
**When to use:** Any interactive-marker map screen in this app (confirmed identical in `trail_map.dart`'s `TrailMarkerLayer`, reused verbatim by `trail_create_screen.dart` and `navigation_screen.dart`).
**Trade-offs:** Precise, no native marker-drag API to fight; the marker layer must read `ml.MapController.maybeOf(context)`/`ml.MapCamera.maybeOf(context)` to convert screen↔geo, and must rebuild on camera move (already established).

**Example (`app/lib/components/map/trail_layer.dart:314-341`):**
```dart
ml.Marker(
  point: point,
  child: GestureDetector(
    onTap: () => widget.onWaypointTap?.call(wp),
    onPanStart: (d) => setState(() { _draggingWaypointId = wp.id; ... }),
    onPanUpdate: (d) => setState(() => _dragOffset = _dragOffset! + d.delta),
    onPanEnd: (d) => widget.onWaypointDragEnd?.call(wp, c.toLngLat(offset)),
    child: _buildCircularMarker(...),
  ),
)
```
**New for the Route Planner:** insert-mid-route (tap on the route line, not a marker) has no existing precedent. Recommended approach: give the route polyline its own native GL layer id (as `TrailLayer` already does for `trail-route`), then on `ml.MapEventClick` call `controller.featuresAtPoint(event.screenPoint, layerIds: ['planner-route'])` (same API `map_screen.dart` already uses for cluster hit-testing) to detect a tap near the line and compute the nearest-segment insertion index.

### Pattern 3: Class-with-imperative-methods Riverpod provider (established, reused as-is)

**What:** `@riverpod class X extends _$X` holding mutable session state, exposing `Future<...>`/`void` methods rather than only computed getters — the same idiom as `TrailSave`, `Navigation`, `NavigationStats`.
**When to use:** Any provider driving user-initiated mutations with async side effects (network calls) interleaved with local state updates.
**Trade-offs:** State (`waypoints`, `autoRouting`, `profile`, `undoStack`, `redoStack`, synthesized `Gpx`) lives in one immutable state object rebuilt via `copyWith`, same as `NavigationState`.

**Example (recommended shape for `route_planner_provider.dart`):**
```dart
class RoutePlannerState {
  final List<Waypoint> waypoints;
  final bool autoRouting;
  final String profile; // 'pedestrian' | 'bicycle' — reuses gpx_util's values
  final Gpx routeGpx;    // rebuilt after every mutation
  // ...copyWith
}

@riverpod
class RoutePlanner extends _$RoutePlanner {
  final List<List<Waypoint>> _undoStack = [];
  final List<List<Waypoint>> _redoStack = [];

  @override
  RoutePlannerState build() => RoutePlannerState(
    waypoints: const [], autoRouting: false, profile: 'pedestrian',
    routeGpx: Gpx(),
  );

  Future<void> addWaypoint(Geographic point) async { /* snapshot, mutate, resolve */ }
  Future<void> moveWaypoint(int index, Geographic point) async { ... }
  Future<void> insertWaypoint(int afterIndex, Geographic point) async { ... }
  void deleteWaypoint(int index) { ... }
  void reorder(int oldIndex, int newIndex) { ... }
  void undo() { ... }
  void redo() { ... }
  void setAutoRouting(bool value) { /* re-resolve all segments */ }
  void setProfile(String profile) { /* re-resolve all segments */ }
}
```
**Undo/redo implementation choice:** the web app's equivalent (`valhalla_store.svelte.ts`) uses `json-diff-ts` changesets. There is no Dart port of that library in this project and waypoint counts for a planned route are small (tens, not thousands), so snapshotting the whole `List<Waypoint>` per edit onto `_undoStack`/`_redoStack` is simpler, dependency-free, and sufficiently cheap — recommended over porting a diff library.

## Data Flow

### Waypoint edit → route resolution

```
User taps/drags/inserts on RoutePlannerMap
    ↓
RoutePlannerMarkerLayer callback (onTap / onWaypointDragEnd / onSegmentInsert)
    ↓
ref.read(routePlannerProvider.notifier).addWaypoint(point)  // or move/insert/delete
    ↓
1. Snapshot current waypoints onto _undoStack, clear _redoStack
2. Mutate waypoints list, set state immediately (instant visual feedback, straight lines)
3. If autoRouting: for each consecutive waypoint PAIR, POST /api/v1/valhalla/route
   (locations: [start, end], costing: profile) → decode trip.legs[0].shape
4. Concatenate all per-pair shapes into one point list
5. POST /api/v1/valhalla/height (encoded_polyline: <that shape>) → height[] array
6. buildGpxFromPoints(points, elevations: height) → Gpx, set into state.routeGpx
    ↓
ref.watch(routePlannerProvider) in RoutePlannerScreen rebuilds:
  - RoutePlannerMap's route polyline + markers
  - Waypoint list sheet (ReorderableListView)
  - ElevationProfile(gpx: state.routeGpx) when elevation view is toggled on
```

### Handoff → existing create/edit flow (unchanged downstream)

```
User taps "Use this route"
    ↓
handoffPlannedRoute(ref, navContext, gpx: state.routeGpx, waypoints: state.waypoints)
    ↓
Trail draft = Trail.empty().copyWith(expand: TrailExpand(gpx: gpx, waypointsViaTrail: waypoints))
pendingImportedTrail = draft   // same global trail_import_util.dart already defines
navContext.push('/trail/create/edit', extra: draft)
    ↓
router_provider.dart's existing '/trail/create/edit' route (UNCHANGED)
    ↓
TrailCreateScreen(trail: draft) — UNCHANGED, same TrailForm/TrailMap/TrailSave flow
```

## Integration Points — Direct Answers

### (1) Where does the new Riverpod provider fit?

**New file:** `app/lib/provider/route_planner_provider.dart`, **top-level** (not `provider/trail/`), `@riverpod class RoutePlanner extends _$RoutePlanner` (codegen, `part 'route_planner_provider.g.dart'`). Rationale: it mirrors `navigation_provider.dart`'s placement — ephemeral, non-record-backed session state — not `trail_save_provider.dart`'s placement, which is record-backed CRUD. No `family` parameter is needed (unlike `navigationProvider(response, ...)`, which is keyed by the trail being navigated) since there is exactly one planner session per screen instance; default `autoDispose` is correct (no `@Riverpod(keepAlive: true)`) since the state should not outlive the screen.

### (2) Reusing the `TrailMap`/`onMapCreated` handoff for tap-to-add + drag

**Confirmed: no map-wide `GestureDetector` is needed, and native `onMapClick`/marker-drag callbacks are NOT how this app does it today.** The established, already-shipping pattern (`trail_map.dart` + `trail_layer.dart`'s `TrailMarkerLayer`, consumed by `trail_create_screen.dart`) is:
- **Tap-to-add:** the native `ml.MapEventClick` event, forwarded through `TrailMap.onTap` (see `trail_map.dart:215-219`). Reuse this exact forwarding shape in the new `RoutePlannerMap`.
- **Drag:** a per-`ml.Marker` `GestureDetector` (`onPanStart`/`onPanUpdate`/`onPanEnd`) living inside a single `ml.WidgetLayer`, not a native drag callback — `maplibre` 0.3.5 has no such API. Reuse `TrailMarkerLayer`'s exact drag-offset/`toLngLat` conversion logic in the new `RoutePlannerMarkerLayer`.
- **New capability needed (no precedent exists):** insert-mid-route (tap on the route line itself). Recommended: give the route polyline a dedicated native layer id and use `controller.featuresAtPoint(screenPoint, layerIds: [...])` on `MapEventClick` — the same API `map_screen.dart` already uses for cluster-circle hit-testing (`map_screen.dart:394-398`) — then compute nearest-segment insertion index client-side.
- **Recommendation:** build `route_planner_map.dart` + `route_planner_marker_layer.dart` as new files modeled on this pattern rather than extending `TrailMap`/`TrailMarkerLayer` in place, to avoid touching a component shared by 3 other screens.

### (3) Is a new backend endpoint required for multi-waypoint routing?

**No new backend endpoint is required. Definitive recommendation: call `POST /api/v1/valhalla/route` once per consecutive waypoint pair — NOT `/api/v1/valhalla/navigate`.**

Investigation found the SvelteKit backend already exposes **three** Valhalla proxy endpoints, not one:
- `POST /api/v1/valhalla/navigate` (`web/src/routes/api/v1/valhalla/navigate/+server.ts`) — wraps Valhalla's **map-matching** action (`shape_match: "map_snap"`, `directions_type: "instructions"`). It is designed to take an *already-known* dense trail track (2-500 points) and snap it to the road network for turn-by-turn narration — this is what `navigation_launch_util.dart` uses today for turn-by-turn nav along a saved trail's full GPX.
- `POST /api/v1/valhalla/route` (`web/src/routes/api/v1/valhalla/route/+server.ts`) — a thin passthrough to Valhalla's actual **routing** action (`locations: [{lat,lon}, {lat,lon}]`, `costing`). This is Valhalla's real point-to-point path-finding engine.
- `POST /api/v1/valhalla/height` (`web/src/routes/api/v1/valhalla/height/+server.ts`) — passthrough to Valhalla's elevation/height service (`encoded_polyline` in, `height: number[]` out).

Critically, **the web app already ships this exact feature** (`web/src/lib/stores/valhalla_store.svelte.ts`, function `calculateRouteBetween`) — a route-planner-equivalent that:
1. Calls `/api/v1/valhalla/route` once per pair of anchor points (`locations: [start, end]`, `costing` derived from the transport mode), not a single multi-waypoint call.
2. Decodes `trip.legs[0].shape` (encoded polyline — same encoding `PolylineUtil.decode` in the Flutter app already handles).
3. Calls `/api/v1/valhalla/height` with that segment's `encoded_polyline` to get per-point elevation.
4. Zips `height[i]` onto each decoded point as `ele`.

This is the exact per-pair pattern the milestone asked about, and it is proven, shipped code, not a hypothesis. `/navigate` is the wrong semantic fit here (it snaps an already-fixed shape rather than route between two arbitrary tapped points, and its `directions_type: "instructions"` response carries maneuver data the planner doesn't need). Recommendation: the Flutter Route Planner should call `/api/v1/valhalla/route` per waypoint pair (via the existing `apiProvider`, whose `baseUrl` already includes `/api/v1` — see `navigation_launch_util.dart:186-191` for the exact call-shape precedent to copy), then `/api/v1/valhalla/height` on the composed shape. **Zero Go or SvelteKit changes are needed** — both endpoints already exist and work; they are simply not yet called from the Flutter app.

### (4) New utility to synthesize a `Gpx` from a live waypoint list

**New function in the existing `app/lib/util/gpx_util.dart`** (not a new file — see Structure Rationale above), e.g.:
```dart
Gpx buildGpxFromPoints(List<Geographic> points, {List<double?>? elevations}) {
  final trkpts = [
    for (var i = 0; i < points.length; i++)
      Wpt(lat: points[i].lat, lon: points[i].lon,
          ele: elevations != null && i < elevations.length ? elevations[i] : null),
  ];
  return Gpx()..trks = [Trk(trksegs: [Trkseg(trkpts: trkpts)])];
}
```
Confirmed via the `gpx` package (v2.3.0, already a pinned dependency) source: `Gpx`, `Trk`, `Trkseg` all have simple mutable-field/named constructors with no required arguments, so this is a straightforward, dependency-free addition. No app code currently constructs a `Gpx` object (only `GpxReader().fromString(...)` is used elsewhere) — this is genuinely new territory, but small and isolated. This is the direct structural inverse of the existing `buildNavShape(List<Geographic>) → shape` in the same file, keeping the file as the single Gpx-conversion home.

### (5) Does the Go backend have a separate elevation-lookup service?

**No — confirmed by search of the entire `db/` Go backend: there is no elevation/height service or endpoint there.** Every Go-side "elevation" hit is the `trails.elevation_gain`/`elevation_loss` schema fields (post-hoc computed stats on a saved trail), not a live lookup service. **However, elevation does NOT need to be omitted at planning time** — the elevation source already exists, just one layer up: SvelteKit's `POST /api/v1/valhalla/height` (`web/src/routes/api/v1/valhalla/height/+server.ts`), which proxies Valhalla's own height/elevation service directly (independent of `/navigate`, independent of Go/`db/`). It is unused by the Flutter app today (only `/navigate` is called, which carries no elevation), but is fully wired, environment-configured (`VALHALLA_HEIGHT_URL`), and already proven in production by the web app's `calculateRouteBetween` (see Q3). Recommendation: call it directly from Flutter with an `encoded_polyline` (reuse `PolylineUtil.encode`) exactly as the web store does, and feed the returned `height[]` array into `buildGpxFromPoints`'s `elevations` parameter. Zero new backend work.

## Anti-Patterns

### Anti-Pattern 1: Wrapping the whole `RoutePlannerMap` in a Flutter `GestureDetector` for tap/drag

**What people do:** Reach for a screen-level `GestureDetector` to capture taps/drags "above" the map widget.
**Why it's wrong:** It would fight `ml.MapGestures.all()`'s own pan/pinch/rotate recognizers and break normal map panning; this app never does this. Every existing interactive-marker screen puts the `GestureDetector` *inside* each `ml.Marker`'s child within a `ml.WidgetLayer`, and uses the map's own `onEvent`/`MapEventClick` for background taps.
**Do this instead:** Follow `TrailMarkerLayer`'s per-marker `GestureDetector` + `TrailMap.onTap`'s native-click-forwarding split exactly (see Pattern 2).

### Anti-Pattern 2: Calling `/api/v1/valhalla/navigate` for point-to-point auto-routing

**What people do:** Reuse the one Valhalla endpoint the mobile app already calls, assuming "it does routing."
**Why it's wrong:** `/navigate` performs map-matching (`shape_match: map_snap`) of an *already-known* path, not point-to-point route-finding between two arbitrary taps — semantically wrong for what auto-routing needs, and it returns maneuver instructions the planner has no use for.
**Instead:** Call `/api/v1/valhalla/route` (Valhalla's actual routing action) per waypoint pair — see Q3 above. This is proven by the web app's own equivalent feature.

### Anti-Pattern 3: Introducing a second "pending trail" global for the planner handoff

**What people do:** Add a new module-level `pendingPlannedTrail` variable in a new file to mirror `trail_import_util.dart`'s pattern.
**Why it's wrong:** `pendingImportedTrail`'s existence is specifically to survive a documented go_router `extra`-loss race (`RouteMatchListCodec` dropping non-JSON `extra` on same-process refresh). A second, independent global creates two competing safety nets and doubles the surface area for that race to resurface.
**Instead:** Reuse the existing `pendingImportedTrail` global via a new function added to `trail_import_util.dart` (see Q4/Structure Rationale).

## Pitfall Found During Research: `GlobalSearchScreen`'s location tile is hardcoded to `/map`

The milestone context assumes "Search-to-focus map panning via existing GlobalSearchScreen flow" is a drop-in reusable pattern. Investigation of `app/lib/routes/global_search_screen.dart` (`_LocationTile.onTap`, line 327-330) found it does:
```dart
onTap: () => context.go('/map', extra: {'lat': ..., 'lon': ..., 'zoom': 13.0}),
```
This is hardcoded to navigate (`context.go`, replacing the stack) straight to `/map` — it has no concept of "return to whichever screen opened me." For the Route Planner to reuse this flow (pan its *own* map, not navigate away to `/map`), `global_search_screen.dart` needs a small, additive modification: either (a) an optional callback/route-target parameter on `GlobalSearchScreen` defaulting to today's `/map` behavior, or (b) switch the location tile to `context.pop(location)` when the search screen was pushed for a "pick a location" purpose (distinguishable via a constructor flag), with the Route Planner using `context.push<LocationSearchResult>('/search')` and awaiting the popped result to re-center its own map. This is a real, small, necessary modification — flag it explicitly in phase planning rather than assuming zero-touch reuse.

## Build Order Recommendation (dependency-ordered)

1. **Provider + pure utilities first** (`route_planner_provider.dart`, `gpx_util.dart`'s `buildGpxFromPoints`, the `/valhalla/route` + `/valhalla/height` call sequence as an imperative method) — testable without any UI, and every other piece depends on this state shape existing.
2. **Map interaction layer** (`route_planner_map.dart`, `route_planner_marker_layer.dart`) — wire tap-to-add/drag/insert against the provider from step 1; this is the highest-uncertainty new code (insert-mid-route hit-testing has no precedent) so de-risk it early, before sheet/elevation UI is built on top.
3. **Screen shell + sheet/elevation UI** (`route_planner_screen.dart`): waypoint list `DraggableScrollableSheet` (reuse the established snapSizes/`ValueNotifier<double>` pattern from `map_screen.dart`/`trail_create_screen.dart`), `_buildButtonRow`-style control buttons for auto-routing toggle + profile switch + sheet↔elevation toggle, `ElevationProfile(gpx: state.routeGpx, trail: <stub>)` reused as-is.
4. **Search-to-focus wiring**: requires the `global_search_screen.dart` modification identified above; do this after the screen shell exists so there's a concrete "re-center my map" target to wire the popped result into.
5. **Handoff last**: `trail_import_util.dart`'s new `handoffPlannedRoute()`, the `router_provider.dart` route registration, and the `trail_source_select_screen.dart` one-line entry-point wire-up. Ordering this last means the entry point only goes live once the full screen behind it actually works, avoiding a dead/broken route being reachable mid-phase-plan.

## Sources

All findings are first-party, from direct reads of this repository (no external documentation needed):
- `app/lib/provider/router_provider.dart` — existing route table, `/trail/create` vs `/trail/create/edit` nesting
- `app/lib/routes/map_screen.dart` — DraggableScrollableSheet/snapSizes pattern, `featuresAtPoint` hit-testing precedent
- `app/lib/routes/navigation_screen.dart`, `app/lib/provider/navigation_provider.dart` — ephemeral top-level provider placement precedent, `_buildButtonRow`/elevation-toggle sheet pattern
- `app/lib/routes/trail_create_screen.dart` — synthesized-stub-`Trail` + `TrailMap` interaction precedent (exact tap/drag wiring)
- `app/lib/components/base/trail_map.dart`, `app/lib/components/base/trail_collection_map.dart` — `onMapCreated`/`onStyleLoaded` handoff pattern
- `app/lib/components/map/trail_layer.dart` (`TrailMarkerLayer`) — per-marker `GestureDetector` drag implementation
- `app/lib/util/trail_import_util.dart` — `pendingImportedTrail` global + import-to-draft-`Trail` precedent
- `app/lib/util/gpx_util.dart` — `buildNavShape`, `costingForCategory`, `GpxMappingUtils` (existing Gpx-bridging home)
- `app/lib/util/navigation_launch_util.dart` — exact `/valhalla/navigate` call shape via `apiProvider`
- `app/lib/util/polyline_util.dart` — polyline encode/decode already available for `/valhalla/height`'s `encoded_polyline`
- `app/lib/models/navigate_response.dart`, `app/lib/models/waypoint.dart`, `app/lib/models/trail.dart` — model shapes for handoff
- `app/lib/routes/trail_source_select_screen.dart` — existing "Planner" card entry point (`_comingSoon` placeholder)
- `app/lib/routes/global_search_screen.dart` — location-tile hardcoded `/map` navigation (pitfall)
- `web/src/routes/api/v1/valhalla/navigate/+server.ts`, `.../route/+server.ts`, `.../height/+server.ts` — the three distinct existing Valhalla proxy endpoints and their exact contracts
- `web/src/lib/stores/valhalla_store.svelte.ts` — the web app's shipped route-planner-equivalent (`calculateRouteBetween`, undo/redo via `json-diff-ts`), proving the per-pair `/route` + `/height` pattern in production
- `db/` (grep across all `.go` files) — confirmed no elevation/height service exists Go-side
- `.pub-cache/hosted/pub.dev/gpx-2.3.0/lib/src/model/{gpx,trk,trkseg,wpt}.dart` — confirmed `Gpx`/`Trk`/`Trkseg`/`Wpt` constructor shapes for `buildGpxFromPoints`
- `.planning/PROJECT.md` — v1.5 milestone scope, constraints, out-of-scope list (car costing, editing existing trails, per-segment profiles, offline route caching)

---
*Architecture research for: Wanderer Route Planner screen (v1.5 milestone)*
*Researched: 2026-07-16*
