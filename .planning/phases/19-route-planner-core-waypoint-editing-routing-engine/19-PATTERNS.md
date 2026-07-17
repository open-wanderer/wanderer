# Phase 19: Route Planner Core — Waypoint Editing & Routing Engine - Pattern Map

**Mapped:** 2026-07-16
**Files analyzed:** 6 (new) + 1 (modified)
**Analogs found:** 7 / 7

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `app/lib/models/route_anchor.dart` | model | CRUD (in-memory) | `app/lib/models/waypoint.dart` (shape only, per D-01 do-not-reuse) / RESEARCH.md's own freezed example | role-match |
| `app/lib/provider/route_anchor_provider.dart` | store/provider | event-driven + request-response (Valhalla calls) | `app/lib/provider/navigation_provider.dart` (class-based `@riverpod` notifier, mutable-field + immutable-state split) | role-match |
| `app/lib/components/map/route_anchor_layer.dart` | component (map overlay) | event-driven (gestures) | `app/lib/components/map/trail_layer.dart` (`TrailMarkerLayer`) | exact |
| `app/lib/routes/route_planner_screen.dart` | route/screen | request-response + event-driven | `app/lib/components/base/trail_map.dart` (`TrailMap`) for map host shape; `app/lib/routes/trail_create_screen.dart` for screen/AppBar shape | role-match |
| `app/lib/util/route_segment_util.dart` | utility | transform (geometric split) | `app/lib/util/gpx_util.dart` (`buildNavShape`, along-track helpers) | role-match |
| `app/lib/util/polyline_util.dart` (MODIFY) | utility | transform | itself — existing file, add `precision` param | exact (self) |
| Valhalla routing call sites (inside provider) | service call | request-response | `web/src/lib/stores/valhalla_store.svelte.ts` (`calculateRouteBetween`) — cross-stack reference only, translate to Dart/Dio | role-match (cross-language) |

## Pattern Assignments

### `app/lib/models/route_anchor.dart` (model, CRUD)

**Analog:** RESEARCH.md's verified freezed shape (no direct in-repo freezed model with an identical field set, but the project's freezed convention is established elsewhere, e.g. `navigation_stats_provider.freezed.dart`'s source class).

**Pattern to copy** (already fully specified in RESEARCH.md → Code Examples):
```dart
import 'package:flutter/foundation.dart' show UniqueKey;
import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:maplibre/maplibre.dart' show Geographic;

part 'route_anchor.freezed.dart';

@freezed
abstract class RouteAnchor with _$RouteAnchor {
  const factory RouteAnchor({
    required String id, // UniqueKey().toString() at creation
    required double lat,
    required double lon,
  }) = _RouteAnchor;

  const RouteAnchor._();

  Geographic get point => Geographic(lat: lat, lon: lon);
}
```
**Naming constraint (D-01):** never name this type or its file with "Waypoint" — `RouteAnchor` / `route_anchor.dart` only, even though `app/lib/models/waypoint.dart` is the nearest existing sibling model by role.

A parallel `RouteSegment` model is needed (not present in RESEARCH.md's code block but described throughout Architecture Patterns) — same freezed shape, fields: `beforeAnchorId`, `afterAnchorId`, `polyline` (`List<Geographic>`), `state` (`routed`/`straight`/`blocked` — use an enum, matching `ToastType` enum convention in `app/lib/provider/toast_provider.dart:6`).

---

### `app/lib/provider/route_anchor_provider.dart` (provider, event-driven + request-response)

**Analog:** `app/lib/provider/navigation_provider.dart`

**Imports pattern** (lines 1-7):
```dart
import 'dart:math' as math;

import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/navigate_response.dart';

part 'navigation_provider.g.dart';
```
For the route-anchor provider: swap `navigate_response.dart` for `route_anchor.dart`/`route_segment.dart`, add `package:dio/dio.dart` (for `CancelToken`), and `package:wanderer/provider/api_provider.dart` (existing `apiProvider` — confirmed reused per RESEARCH.md's Valhalla call code).

**State class pattern** (lines 9-35): plain class with `copyWith`, not freezed, holding both the reactive fields (`anchors`, `segments`, `autoRoutingEnabled`) and bookkeeping (`undoStack`, `redoStack`). Mirrors `NavigationState`'s shape exactly — copy this structure directly, extending with the two stacks from RESEARCH.md's Undo/Redo section.

**Class-based notifier pattern** (lines 37-54):
```dart
@riverpod
class Navigation extends _$Navigation {
  static const _kManeuverAdvanceThresholdMeters = 30.0;

  // Non-reactive bookkeeping lives as class fields, NOT in state —
  // the UI doesn't need to know about this internal detail.
  int _currentShapeIndex = 0;

  @override
  NavigationState build(
    NavigateResponse response, {
    int? resumeManeuverIndex,
    List<Geographic>? resumeBreadcrumb,
  }) {
    ...
    return NavigationState(...);
  }
}
```
For `RouteAnchors`: the per-segment `_inFlight`/`_generation` maps (RESEARCH.md's Race-Guard Pattern) are exactly this kind of non-reactive class field — same rationale as `_currentShapeIndex`/`_shapeCumulativeMeters` here (UI-irrelevant bookkeeping that must survive across mutation calls but not trigger rebuilds itself).

**Mutation method pattern** (lines 182-217, `onPosition`):
```dart
void onPosition(Geographic pos) {
  state = state.copyWith(breadcrumb: [...state.breadcrumb, pos]);
  ...
  if (newIndex > current) {
    state = state.copyWith(currentManeuverIndex: newIndex);
  }
}
```
Copy this shape for `appendAnchor`/`dragAnchor`/`insertAnchorOnSegment`/`retrySegment`/`undo`/`redo`: read `state`, compute, reassign `state = state.copyWith(...)` — never mutate lists in place.

**Simple `@Riverpod(keepAlive: true)` list-notifier alternative** — if the planner decides undo/redo stacks warrant a *separate* lighter provider, `app/lib/provider/toast_provider.dart` shows the minimal shape (`build() => []`, `state = [...state, x]` to append, filtered list comprehension to remove). Not recommended as the primary shape here (undo/redo needs to be tightly coupled to anchor/segment state, per RESEARCH.md), but usable as a fallback pattern for any standalone queue-like sub-state.

**Valhalla request pattern** (translate from `web/src/lib/stores/valhalla_store.svelte.ts`'s `calculateRouteBetween`, already Dart-translated in RESEARCH.md's Code Examples → "Valhalla Route Call" and Architecture Patterns → "Race-Guard Pattern" — reuse those blocks verbatim, they are already correct against this project's `apiProvider`/Dio conventions).

---

### `app/lib/components/map/route_anchor_layer.dart` (component, event-driven)

**Analog:** `app/lib/components/map/trail_layer.dart` → `TrailMarkerLayer` (lines 262-404)

**Imports pattern** (lines 1-9):
```dart
import 'dart:convert';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/util/gpx_util.dart';
```
For the new layer: drop `trail.dart`/`waypoint.dart`, add `route_anchor.dart`, keep `ml`/`font_awesome_flutter` imports, add `flutter_riverpod` (this new layer should be a `ConsumerStatefulWidget` reading `routeAnchorsProvider` directly, since it's not bound to a passed-in `Trail` object the way `TrailMarkerLayer` is).

**Drag gesture pattern** (lines 314-343, exact D-05 precedent):
```dart
child: GestureDetector(
  onTap: () => widget.onWaypointTap?.call(wp),
  onPanStart: (details) {
    final c = ml.MapController.maybeOf(context);
    if (c == null) return;
    setState(() {
      _draggingWaypointId = wp.id;
      _dragOffset = c.toScreenLocation(
        ml.Geographic(lon: wp.lon, lat: wp.lat),
      );
    });
  },
  onPanUpdate: (details) {
    if (_draggingWaypointId != wp.id || _dragOffset == null) return;
    setState(() => _dragOffset = _dragOffset! + details.delta);
  },
  onPanEnd: (details) {
    if (_draggingWaypointId != wp.id) return;
    final c = ml.MapController.maybeOf(context);
    final offset = _dragOffset;
    _clearDrag();
    if (c != null && offset != null) {
      widget.onWaypointDragEnd?.call(wp, c.toLngLat(offset));
    }
  },
  onPanCancel: () {
    if (_draggingWaypointId == wp.id) _clearDrag();
  },
  child: AnimatedScale(...),
),
```
Copy this verbatim, renaming `_draggingWaypointId` → `_draggingAnchorId` and `wp` → `anchor`. Per D-05, no live route preview during drag — the temporary straight-position render during drag (`_dragOffset`/`toLngLat`) is exactly this existing mechanism; segments only re-resolve `onPanEnd` (call into `ref.read(routeAnchorsProvider.notifier).dragAnchor(...)` there, not in `onPanUpdate`).

**Marker widget wrapper pattern** (lines 300-313, `ml.Marker` + `size: const Size(32, 32)`): reuse the 32px marker size and `_buildCircularMarker` styling helper (lines 409-435) directly — D-04's "32px marker + 36px proximity-nudge" language refers to this exact constant.

**Return shape** (line 402): `return ml.WidgetLayer(allowInteraction: true, markers: markers);` — copy directly.

---

### `app/lib/routes/route_planner_screen.dart` (route/screen, request-response + event-driven)

**Analog A (map host shape):** `app/lib/components/base/trail_map.dart`

**Style-loaded race-buffer pattern** (lines 78-89, 199-214) — copy verbatim per Pitfall 5:
```dart
ml.StyleController? _pendingStyle;
...
onMapCreated: (controller) {
  _controller = controller;
  widget.onMapCreated?.call(controller);
  final pending = _pendingStyle;
  if (pending != null) {
    _pendingStyle = null;
    _onStyleLoaded(pending);
  }
},
onStyleLoaded: (style) {
  if (_controller == null) {
    _pendingStyle = style;
    return;
  }
  _onStyleLoaded(style);
},
```

**Tap routing pattern** (lines 215-220):
```dart
onEvent: (event) {
  widget.onMapEvent?.call(event);
  if (event is ml.MapEventClick) {
    widget.onTap?.call(event.point);
  }
},
```
Extend this per RESEARCH.md's Map Tap Routing section (marker-hit short-circuit is implicit via the WidgetLayer's own GestureDetector consuming the tap first; add the `featuresAtPoint(event.screenPoint, layerIds: ['route-segments-hit'])` segment check before falling through to append).

**Top-right controls slot** (lines 245-251, D-06 precedent):
```dart
Align(
  alignment: Alignment.topRight,
  child: Column(
    crossAxisAlignment: CrossAxisAlignment.end,
    children: widget.controls ?? const [],
  ),
),
```
The auto-routing toggle button is passed into this same `controls` slot shape.

**Gestures-enabled-by-default pattern** (lines 194-196):
```dart
gestures: widget.disabled
    ? const ml.MapGestures.none()
    : const ml.MapGestures.all(),
```
Confirms D-04/D-05's assumption — the route planner map should run with full, un-disabled `MapGestures.all()` (matching the live-shipped `trail_create_screen.dart` precedent RESEARCH.md cites), not a restricted gesture set.

**Analog B (screen scaffold + app bar icon buttons, D-10):** `app/lib/routes/trail_create_screen.dart` (lines 378-410):
```dart
return Scaffold(
  extendBodyBehindAppBar: true,
  appBar: AppBar(
    leading: IconButton(
      icon: const FaIcon(FontAwesomeIcons.arrowLeft, size: 18),
      onPressed: () => context.canPop()
          ? context.pop()
          : context.pushReplacement('/map'),
      style: IconButton.styleFrom(
        backgroundColor: Theme.of(context).colorScheme.surface,
      ),
    ),
    actions: [
      IconButton(
        icon: _saving
            ? const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2))
            : const FaIcon(FontAwesomeIcons.floppyDisk, size: 18),
        onPressed: _saving ? null : () => _onSave(context),
        style: IconButton.styleFrom(
          backgroundColor: Theme.of(context).colorScheme.surface,
        ),
      ),
    ],
    backgroundColor: Colors.transparent,
    shadowColor: Colors.transparent,
    elevation: 0,
    scrolledUnderElevation: 0,
  ),
  ...
);
```
For undo/redo (D-10, D-11): add two `IconButton`s to `actions` (`FontAwesomeIcons.arrowRotateLeft`/`arrowRotateRight`, VERIFIED present per RESEARCH.md), each with `onPressed: canUndo ? () => notifier.undo() : null` — the existing `_saving ? null : ...` conditional-disable idiom on line 399 is the exact precedent for D-11's "disabled when stack empty" requirement (a null `onPressed` renders IconButton in its built-in disabled/grayed style automatically, no extra styling needed).

---

### `app/lib/util/route_segment_util.dart` (utility, transform)

**Analog:** `app/lib/util/gpx_util.dart` (`buildNavShape`, `costingForCategory`, along-track helpers) — read for the general "small pure functions over `Geographic`/point lists, no state" shape this project's util files follow. The concrete algorithm to implement (`splitSegmentAt`) is already fully specified in RESEARCH.md → Architecture Patterns → Segment Insert Strategy:
```dart
(RouteSegment, RouteSegment) splitSegmentAt(RouteSegment segment, Geographic tapPoint) {
  final points = segment.polyline;
  var bestIndex = 0;
  var minDistance = double.infinity;
  for (var i = 1; i < points.length; i++) {
    final calculator = SphericalGreatCircle(points[i]);
    final dist = calculator.distanceTo(tapPoint);
    if (dist < minDistance) {
      minDistance = dist;
      bestIndex = i;
    }
  }
  final splitPoint = points[bestIndex];
  final first = RouteSegment(polyline: [...points.sublist(0, bestIndex), splitPoint], state: segment.state);
  final second = RouteSegment(polyline: [splitPoint, ...points.sublist(bestIndex)], state: segment.state);
  return (first, second);
}
```
Note `SphericalGreatCircle` is the same distance-calculator class already used in `app/lib/provider/navigation_provider.dart` (line 69, 172) — consistent cross-file usage, no new geo-math dependency introduced.

---

### `app/lib/util/polyline_util.dart` (MODIFY — utility, transform)

**Self-analog** — existing file, full contents already read (84 lines). Required change per Pitfall 1: add a `precision` parameter (default `5`, preserving all existing callers) to both `encode`/`decode`:

Current `decode` signature (line 32): `static List<Geographic> decode(String polyline)` — hardcodes `1E5` at line 66 (`points.add(Geographic(lat: lat / 1E5, lon: lng / 1E5))`).

**Required pattern:**
```dart
static List<Geographic> decode(String polyline, {int precision = 5}) {
  final factor = math.pow(10, precision);
  ...
  points.add(Geographic(lat: lat / factor, lon: lng / factor));
  ...
}
static String encode(List<Geographic> points, {int precision = 5}) {
  final factor = math.pow(10, precision);
  ...
  final int lat = (point.lat * factor).round();
  final int lng = (point.lon * factor).round();
  ...
}
```
Mirrors `web/src/lib/util/polyline_util.ts`'s `decodePolyline(str, precision = 6)` default-vs-explicit-override shape exactly (web defaults differently per its own callers, but the parameter-with-default idiom is the pattern to copy). All existing call sites (trail GPX-derived polylines) keep working unmodified since they omit the new parameter; only the new Valhalla `/route` call site passes `precision: 6` explicitly.

---

## Shared Patterns

### Style-loaded race-buffer (project-wide, Pitfall 5)
**Source:** `app/lib/components/base/trail_map.dart` lines 78-89, 199-214
**Apply to:** `route_planner_screen.dart`'s map host — copy the `_pendingStyle` buffer-and-replay verbatim.

### Drag gesture coexistence with native map gestures (D-04/D-05)
**Source:** `app/lib/components/map/trail_layer.dart` lines 314-343 (`TrailMarkerLayer`'s `GestureDetector.onPanStart/Update/End`), proven live in `app/lib/routes/trail_create_screen.dart` against `MapGestures.all()`.
**Apply to:** `route_anchor_layer.dart`'s anchor drag handling.

### Class-based `@riverpod` notifier with non-reactive class-field bookkeeping
**Source:** `app/lib/provider/navigation_provider.dart` (whole file shape: precomputed `late final` fields in `build()`, plain mutable class fields for per-call bookkeeping like `_currentShapeIndex`, `state = state.copyWith(...)` for every reactive mutation).
**Apply to:** `route_anchor_provider.dart`'s `RouteAnchors` notifier, including its `_inFlight`/`_generation` per-segment race-guard maps.

### Icon-button disabled-when-empty pattern (D-11)
**Source:** `app/lib/routes/trail_create_screen.dart` line 399 (`onPressed: _saving ? null : () => _onSave(context)`)
**Apply to:** Undo/redo app-bar buttons — `onPressed: canUndo ? notifier.undo : null` / `onPressed: canRedo ? notifier.redo : null`.

### Top-right map controls slot (D-06)
**Source:** `app/lib/components/base/trail_map.dart` lines 245-251 (`Align(alignment: Alignment.topRight, child: Column(children: widget.controls ?? const []))`)
**Apply to:** The route-planner map host's `controls` parameter, carrying the auto-routing toggle button (and, per RESEARCH.md, anticipating Phase 20's list/elevation toggles in the same slot).

### GeoJSON source + filtered LineStyleLayers for multi-state line rendering (D-08)
**Source:** `app/lib/components/map/trail_layer.dart` lines 78-156 (`TrailLayer.add`'s casing+route layer pairing, `jsonEncode` GeoJSON construction pattern) — extend with `filter` per RESEARCH.md's Segment Rendering Contract code block (routed/straight/blocked + invisible hit-test layer), which is not present verbatim in `trail_layer.dart` (that file has no `filter` usage) but follows its exact `addSource`/`addLayer`/`jsonEncode` idiom.
**Apply to:** `route_planner_screen.dart`'s segment rendering (or a new `route_segment_layer.dart` if the planner splits marker/segment rendering into separate files — RESEARCH.md's structure combines both concerns under one screen, but a split mirroring `trail_layer.dart`'s own `TrailLayer`/`TrailMarkerLayer` split is equally valid).

## No Analog Found

None — every file in RESEARCH.md's Recommended Project Structure has a strong in-repo analog (the routing/undo-redo *logic* itself is genuinely novel per RESEARCH.md's own confidence rating, but the *code shapes* to copy — notifier structure, gesture handling, screen scaffold, GeoJSON layer construction — all have direct precedent).

## Metadata

**Analog search scope:** `app/lib/components/map/`, `app/lib/components/base/`, `app/lib/provider/`, `app/lib/routes/`, `app/lib/util/`
**Files scanned:** `trail_layer.dart`, `trail_map.dart`, `polyline_util.dart`, `navigation_provider.dart`, `toast_provider.dart`, `trail_create_screen.dart`, `gpx_util.dart` (directly read); provider directory listing scanned for additional class-based notifier candidates
**Pattern extraction date:** 2026-07-16
