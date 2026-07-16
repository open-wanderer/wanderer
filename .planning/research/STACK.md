# Stack Research

**Domain:** Interactive multi-waypoint route-planning UI on top of `package:maplibre` in a Flutter/Riverpod app (Wanderer v1.5 Route Planner)
**Researched:** 2026-07-16
**Confidence:** HIGH (all four questions verified against the actual installed package source, the actual existing SvelteKit endpoints, and the actual existing Flutter code in this repo — not just training data)

> Note: This supersedes the previous `STACK.md` (v1.1 Offline Navigation research, dated 2026-06-14). That research is preserved in git history; this file now covers the v1.5 Route Planner milestone.

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `package:maplibre` | 0.3.5 (already pinned, no change) | Map rendering, `ml.WidgetLayer`/`ml.Marker` for waypoint pins | Already the app's only map stack. Confirmed by reading `widget_layer.dart` in the installed package: `Marker` has **no drag/gesture support at all** — it is a pure data holder (`point`, `size`, `child`, `alignment`, `rotate`, `flat`). Dragging must be built as a custom `GestureDetector` wrapped around the marker's `child`, using the existing `MapController.toLngLat(Offset)` / `toLngLats(List<Offset>)` synchronous conversion methods (`maplibre_platform_interface-0.3.5/lib/src/map_controller.dart`) to turn drag-frame screen offsets back into `Geographic` coordinates every `onPanUpdate` tick. No new package is needed for this — it is a ~40-line custom widget, not a library gap. |
| `flutter_riverpod` + `riverpod_annotation` | 3.3.1 / 4.0.2 (already pinned, no change) | Route-plan state container with undo/redo | Follow the exact `@riverpod class X extends _$X` + imperative-method idiom already used by `TrailSave` (`app/lib/provider/trail/trail_save_provider.dart`) and `WaypointSave`. A `RoutePlan` provider (`@Riverpod(keepAlive: true) class RoutePlan extends _$RoutePlan`) holding a `RoutePlanState` (freezed) with methods `addWaypoint`, `moveWaypoint`, `insertWaypoint`, `deleteWaypoint`, `reorderWaypoint`, `toggleAutoRouting`, `undo`, `redo` is the idiomatic fit — no new state-management package needed. |
| `freezed` + `freezed_annotation` | 3.2.5 / 3.1.0 (already pinned, no change) | Immutable `RoutePlanState` / `RoutePlanWaypoint` models | Matches existing model conventions (`Trail`, `Waypoint`, `NavigateResponse`). Remember the codebase's known freezed 3.x gotcha: `@JsonSerializable(explicitToJson: true)` must sit on the **factory constructor**, not the class, or json_serializable codegen breaks (see Key Decisions in PROJECT.md). |
| `package:gpx` | 2.3.0 (already pinned, no change) | Synthesizing a `Gpx` object from the in-memory route for `ElevationProfile` and for GPX-string handoff to `trail_create_screen` | `ElevationProfile` (`app/lib/components/trail/elevation_profile.dart`) requires a real `Gpx` with `trk`/`trkseg`/`trkpt`. `gpx_util.dart`'s `buildNavShape` only goes `Geographic list → Valhalla shape array`; there is no existing `List<Geographic> (+ele) → Gpx` helper. Add one (e.g. `buildGpxFromRoute`) using `package:gpx`'s existing `Gpx`, `Trk`, `Trkseg`, `Wpt` classes directly — all already public, already a dependency, no new package required. `Wpt` already has nullable `lat`, `lon`, `ele`, `time` fields, so per-point elevation can be attached directly when synthesizing. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `undo` (pub.dev, rodydavis) | 1.6.0 | Optional off-the-shelf command-pattern undo/redo | Only if the hand-rolled snapshot-stack approach (see Stack Patterns below) starts feeling repetitive across many action types. Zero dependencies, framework-agnostic (works fine alongside Riverpod), Command-Pattern based (`Change` = old state + execute + undo, with `Change.group()` for compound actions). **Not required** — the route plan's state is small (a handful to a few dozen waypoints), so a plain `List<RoutePlanState>` snapshot stack inside the `RoutePlan` notifier is simpler, has zero new dependencies, and mirrors what the codebase already does elsewhere (freezed `copyWith` immutability). Prefer the hand-rolled approach unless action variety grows large enough to justify the abstraction. |
| None (no polyline/geodesy package addition needed) | — | Great-circle distance for straight-line segment length/interpolation | `gpx_util.dart`'s `GpxMappingUtils.getTotals()` already uses `SphericalGreatCircle` (from `package:maplibre`) for distance calculations — reuse the same class for straight-line segment lengths and any split/insert math, exactly as `gpx_util.dart` already does. Do not add `latlong2`'s distance helpers or a new geodesy package for this; `latlong2` is still a dependency (legacy `flutter_map`-era code) but `maplibre`-native `Geographic`/`SphericalGreatCircle` types are the established pattern post-v1.4 migration. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `build_runner` + `riverpod_generator` + `json_serializable` + `freezed` | Codegen for the new `RoutePlan` provider and `RoutePlanState`/`RoutePlanWaypoint` freezed models | Already configured; run `dart run build_runner build --delete-conflicting-outputs` after adding the new provider/model files, same as every other phase in this app. |

## Installation

No new pubspec dependencies are required for the core feature. Optional:

```yaml
dependencies:
  undo: ^1.6.0 # OPTIONAL — only if hand-rolled snapshot undo/redo proves insufficient
```

```bash
# If added:
flutter pub add undo

# Regenerate codegen for new provider/model files (always required for this feature regardless):
dart run build_runner build --delete-conflicting-outputs
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|--------------------------|
| Custom `GestureDetector` + `MapController.toLngLat` for marker dragging | A dedicated "draggable marker" Flutter package (none exists for `package:maplibre`; some exist for `flutter_map`, e.g. community drag plugins) | Never for this project — `flutter_map` plugins are incompatible with `package:maplibre` (different rendering engine, different `Marker`/`WidgetLayer` API), and the v1.4 migration explicitly retired all `flutter_map` plugins. There is no equivalent drag plugin for `package:maplibre` at 0.3.5; the custom `GestureDetector` approach is the only option and is small enough not to need one. |
| Hand-rolled snapshot-stack undo/redo inside a `@riverpod` class | `undo` package (Command Pattern) | Use the package if the team wants named, composable `Change` objects per action type (e.g. for a future "action log" UI) or if action grouping (`Change.group()`) becomes necessary for compound edits. For v1.5's scope (flat list of waypoint edits), a snapshot stack of `RoutePlanState` is simpler and avoids introducing a new abstraction for a short-lived, in-memory-only feature. |
| Pairwise (2-location) `/api/v1/valhalla/route` calls per segment (N-1 calls for N waypoints) | Valhalla's native N-location single `/route` call (verified: Valhalla's `/route` endpoint accepts an ordered list of ≥2 locations and returns one `trip.legs[]` entry per consecutive pair in a single response) | Use a single N-location call only if the route planner ever needed a fully-routed, non-editable, single "solve entire trip at once" button with no per-segment straight-line/routed mixing and no independent segment undo. That is explicitly not this feature: v1.5 requires a per-segment auto-routing toggle (routed vs straight-line coexisting on the same route) and insert/delete/reorder at waypoint granularity, which the existing web implementation (`web/src/lib/stores/valhalla_store.svelte.ts`'s `calculateRouteBetween`) already solves by calling `/api/v1/valhalla/route` with exactly `locations: [start, end]` per segment, then stitching per-segment `Waypoint` arrays into `trk.trkseg[]`. **Mirror this exact pattern in Flutter** — do not attempt a single N-waypoint request shape for this feature. |
| Reuse `/api/v1/valhalla/route` + `/api/v1/valhalla/height` (already-existing SvelteKit endpoints, currently only consumed by the web frontend) | Extending `/api/v1/valhalla/navigate` to accept multiple discrete segments | Do not extend `/navigate`: it is purpose-built for the "download a saved trail's full shape, get maneuvers, cache offline" flow (requires `event.locals.user`, uses `directions_type: "instructions"` + `shape_match: "map_snap"`, downsamples to ≤500 points). `/route` and `/height` are the correct existing endpoints for planning-time segment routing and elevation — they already exist in this exact Go/SvelteKit backend, and the Flutter app's `apiProvider` (Dio + cookie jar) can call them with zero new networking code. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|--------------|
| `flutter_map` drag-marker plugins / any `flutter_map`-ecosystem package | Incompatible with `package:maplibre`'s rendering pipeline; v1.4 fully retired `flutter_map` and its 4 plugins from `pubspec.yaml` — reintroducing any `flutter_map`-dependent package would resurrect a second map stack | Custom `GestureDetector` + `MapController.toLngLat`/`toLngLats` on top of `ml.WidgetLayer`/`ml.Marker`, as this app's own `map_screen.dart`, `trail_layer.dart`, and `list_detail_map_screen.dart` already do for tap gestures (just extend the pattern to pan gestures) |
| A generic JS-style delta/diff library port (there is no Dart equivalent of `json-diff-ts`, and none is needed) | The web app's `valhalla_store.svelte.ts` uses `json-diff-ts` changesets for undo/redo because Svelte 5 runes + a large mutable GPX object benefit from minimal-diff patches; Dart/Riverpod's `copyWith`-based immutable freezed models make full-state snapshots cheap and simple for a route plan of realistic size (tens of waypoints, not thousands) | Plain `List<RoutePlanState>` snapshot stack (push full immutable state on each undoable action, pop on undo/redo) |
| Building a custom TSP/route-optimization solver, or calling Valhalla's `/optimized_route` service | Out of scope — v1.5 requirements are ordered tap/drag/insert/reorder, not automatic waypoint-order optimization; `/optimized_route` reorders intermediate waypoints, which would fight the explicit "reorder waypoints" UI requirement | Ordered pairwise `/route` calls in the user-specified waypoint order (matches existing web behavior) |
| Client-side elevation approximation via SRTM tile parsing, a bundled DEM, or a third-party elevation API/package | Valhalla's `/height` service (Valhalla's dedicated elevation-lookup action, distinct from `/route`) is **already proxied** in this exact backend at `POST /api/v1/valhalla/height` (`web/src/routes/api/v1/valhalla/height/+server.ts` → `getValhallaHeightUrl()` → `VALHALLA_HEIGHT_URL`), already consumed by the web route planner (`calculateRouteBetween` posts `{ encoded_polyline: shape }` and receives `{ height: number[] }`), and is trivially reachable from Flutter via the existing `apiProvider` Dio instance | `POST /api/v1/valhalla/height` with an encoded polyline built from the current (routed-or-straight) segment shape; decode the height array 1:1 against the decoded polyline points, exactly as `valhalla_store.svelte.ts` does |

## Stack Patterns by Variant

**If a waypoint is dragged (existing point moved):**
- Track drag with `GestureDetector(onPanStart/onPanUpdate/onPanEnd)` wrapping the marker's `child` inside `ml.WidgetLayer` (`allowInteraction: true` required — confirmed this flag exists and gates gesture pass-through on iOS/Android per `widget_layer.dart`'s `TranslucentPointer` fallback when `false`).
- On each `onPanUpdate`, convert the global/local `Offset` to `Geographic` via `MapController.toLngLat(offset)` (synchronous, no async gap) and update a **transient, non-undoable** local position for live visual feedback.
- On `onPanEnd`, commit the final position into the `RoutePlan` provider as a single undoable action (one snapshot push), then re-resolve the (at most two) adjacent segments if auto-routing is on.
- Because this pattern doesn't exist anywhere yet in the codebase (only tap `GestureDetector`s exist today, e.g. `map_screen.dart:316`), write it once as a small reusable `DraggableMarker` widget under `app/lib/components/map/` rather than inlining it in the route planner screen.

**If auto-routing is ON for a segment:**
- Call `POST /api/v1/valhalla/route` with `{ locations: [{lat, lon}, {lat, lon}], costing: 'pedestrian' | 'bicycle', costing_options: {...}, directions_type: 'none' }` (mirrors `calculateRouteBetween`'s `costingBody` construction) for exactly that one segment.
- Take `trip.legs[0].shape` (encoded polyline) and `trip.summary.time`.
- Then call `POST /api/v1/valhalla/height` with `{ encoded_polyline: shape }` to get per-point elevation for that segment's decoded points.
- Store both the decoded `[lat, lon]` list and the elevation array as the segment's resolved points (this becomes one `Trkseg` when synthesizing the final `Gpx`).

**If auto-routing is OFF for a segment (straight-line):**
- Build the 2-point (or interpolated N-point, for elevation-profile smoothness) shape client-side, encoded the same way `calculateRouteBetween`'s `else` branch does (`encodePolyline([[startLat, startLon], [endLat, endLon]])`). Check whether the Flutter side's `polyline_util.dart` already has an `encodePolyline` counterpart to its `decodePolyline` before porting the web encoding scheme — if missing, add one so segment shapes stay round-trippable.
- Still call `/api/v1/valhalla/height` for that straight segment's shape so the elevation profile stays populated for both routed and unrouted segments — Valhalla's height service works on any polyline, routed or not.

**If a segment's auto-routing toggle or transport profile changes:**
- Re-resolve only that segment (re-run the two branches above) — do not re-fetch or invalidate other segments. This is the direct reason the web implementation (and this recommendation) uses per-segment pairwise calls instead of one N-location call: partial re-resolution isn't possible against a single monolithic multi-leg trip response without re-requesting the whole thing.

**At handoff to `trail_create_screen`:**
- Synthesize one `Gpx` (either one `Trk` with one `Trkseg` per segment, or a single flattened `Trkseg` — `gpx_util.dart`'s `GpxMappingUtils.allWaypoints` already flattens across all `trks`/`trksegs`, so either shape is consumable) using `package:gpx`'s `Gpx`/`Trk`/`Trkseg`/`Wpt` constructors directly.
- Follow `trail_import_util.dart`'s exact pattern: build a stub `Trail` via `Trail.empty()`-based construction with `expand.gpx` set to the synthesized `Gpx` (and, if a GPX string is also needed server-side later, serialize via `package:gpx`'s `GpxWriter().asString(gpx)` into `expand.gpxData`), set the global `pendingImportedTrail` safety net, then `context.push('/trail/create/edit', extra: trail)`.

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|------------------|-------|
| `maplibre@0.3.5` | `maplibre_platform_interface@0.3.5`, `maplibre_android@*`, `maplibre_ios@*`, `maplibre_web@*` | All four are locked together in `pubspec.lock` at matching `0.3.5`-family versions; do not bump only one. `MapController.toLngLat`/`toLngLats` (needed for drag) live in `maplibre_platform_interface`, confirmed present at 0.3.5 — no version bump needed for the drag feature. |
| `gpx@2.3.0` | `freezed@3.2.5`/`json_serializable@6.13.0` (unrelated, no interaction) | `gpx`'s `Wpt`/`Trk`/`Trkseg`/`Gpx` are plain mutable classes (not freezed) — safe to construct imperatively when synthesizing the draft route's `Gpx`; no serialization conflict with the app's freezed model conventions since `Gpx` is intentionally kept out of `Trail`'s JSON serialization (`expand.gpx` is a client-only, non-serialized field per `trail_import_util.dart`'s comment: "The Gpx object isn't serializable, so this step stays client-side"). |
| `undo@1.6.0` (if adopted) | `flutter_riverpod@3.3.1` | Framework-agnostic, zero dependencies — no compatibility risk with Riverpod 3.x. |

## Pitfall Flagged for Roadmap

`/api/v1/valhalla/route` and `/api/v1/valhalla/height` currently have **no** `event.locals.user` auth check (unlike `/api/v1/valhalla/navigate`, which requires a logged-in user). This is pre-existing behavior (the web route planner already calls them unauthenticated-gated), not something v1.5 introduces — but since the Flutter app will now also call these two endpoints, it's worth a one-line confirmation during implementation that this is accepted behavior rather than an oversight, since a route-planning proxy to Valhalla is a potential abuse/cost vector if left fully open. Not blocking; flagged for awareness only.

## Sources

- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre-0.3.5/lib/src/widget_layer.dart` — read directly; confirmed `Marker`/`WidgetLayer` have no drag support, `allowInteraction` gating behavior. HIGH confidence (primary source, not training data).
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/map_controller.dart` — read directly; confirmed `toLngLat`/`toLngLats`/`toScreenLocation(s)` synchronous conversion methods exist. HIGH confidence.
- `/Users/christianbeutel/Documents/svelte/wanderer/web/src/lib/stores/valhalla_store.svelte.ts` and `web/src/lib/util/valhalla_anchor_util.ts` — read directly; this is the **already-shipped web version of this exact feature** (route planner with drag anchors, undo/redo, auto-routing toggle, pairwise Valhalla calls, height lookups). HIGH confidence — this is the single strongest source for this research since it's a working reference implementation of the same feature in the same monorepo.
- `/Users/christianbeutel/Documents/svelte/wanderer/web/src/routes/api/v1/valhalla/route/+server.ts`, `.../height/+server.ts`, `.../navigate/+server.ts` — read directly; confirmed both `/route` and `/height` already exist and proxy Valhalla, unauthenticated (no `event.locals.user` check, unlike `/navigate`). HIGH confidence.
- `/Users/christianbeutel/Documents/svelte/wanderer/app/lib/util/gpx_util.dart`, `navigation_launch_util.dart`, `trail_import_util.dart`, `components/trail/elevation_profile.dart`, `provider/trail/trail_save_provider.dart`, `provider/api_provider.dart` — read directly; confirmed existing helper shapes, the `@riverpod` class idiom, and the `Gpx`-required `ElevationProfile` contract. HIGH confidence.
- Valhalla routing service documentation (Stadia Maps guide, Valhalla docs, GitHub issue #5183) — WebSearch, MEDIUM confidence, cross-checked against the working web implementation which independently confirms Valhalla's `/route` accepts >2 ordered locations returning multi-leg trips (verified via actual behavior, not just docs). [Stadia Maps Valhalla guide](https://docs.stadiamaps.com/guides/getting-the-best-routes-with-valhalla-turn-by-turn-directions-apis/), [Valhalla turn-by-turn API reference](https://valhalla.github.io/valhalla/api/turn-by-turn/api-reference/), [Valhalla optimized route API](https://valhalla.github.io/valhalla/api/optimized/api-reference/)
- `undo` package pub.dev page — WebFetch, MEDIUM confidence (version 1.6.0, zero dependencies, Command Pattern) — presented as optional, not required. [pub.dev/packages/undo](https://pub.dev/packages/undo)

---
*Stack research for: Flutter Route Planner screen (Wanderer v1.5)*
*Researched: 2026-07-16*
