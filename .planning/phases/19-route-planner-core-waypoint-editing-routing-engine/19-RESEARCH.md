# Phase 19: Route Planner Core — Waypoint Editing & Routing Engine - Research

**Researched:** 2026-07-16
**Domain:** Flutter mobile map interaction (native MapLibre GL), Riverpod state architecture, Valhalla routing integration
**Confidence:** MEDIUM-HIGH (both v1.5 STATE.md research flags resolved to HIGH via direct source reads; routing/undo-redo architecture is MEDIUM — novel in this codebase but strongly informed by an equivalent, already-shipped web implementation)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Terminology (locked — applies to all downstream artifacts)**
- **D-01:** The in-progress route's tap points are called **"route anchors"**, never "waypoints," anywhere in code, UI copy, or docs for this phase. This is a deliberate distinction from the existing `Waypoint` model (`app/lib/models/waypoint.dart`), which belongs to a persisted `Trail` and is a different data type/concept. A new type (e.g. `RouteAnchor`) should back the in-progress route rather than reusing `Waypoint`.
- **D-02:** Route anchors are numbered in ascending order as the user adds them. Inserting a new anchor mid-route (WAYP-03) renumbers every anchor after the insertion point so the sequence stays contiguous and ascending.

**Waypoint (route-anchor) gesture disambiguation**
- **D-03:** A tap on empty map space always adds a new route anchor (appended to the end of the route) — no explicit "add mode" toggle.
- **D-04:** Tap-on-marker vs. tap-on-segment vs. tap-on-empty-map is disambiguated by giving markers a larger invisible hit-radius (matching the existing 32px marker + 36px proximity-nudge pattern in `trail_layer.dart`), checked *before* segment hit-testing. Marker wins on overlap.
- **D-05:** Route-anchor drag reuses the existing `GestureDetector.onPanStart/Update/End` pattern from `TrailMarkerLayer` (`trail_layer.dart`) — proven to coexist with native map pan/zoom gestures. No live route preview during drag; the anchor shows at a straight temporary screen position while dragging, and connected segments re-resolve to the current routing mode only once the drag ends.

**Auto-routing toggle**
- **D-06:** The auto-routing on/off toggle lives in the top-right map control buttons, matching the existing `TrailMap.controls` Column pattern (top-right corner) and consistent with Phase 20's planned waypoint-list/elevation toggle buttons in the same spot.
- **D-07 (SCOPE CHANGE):** There is **no in-planner travel-profile switch**. The foot/bike profile is set once via the Phase 21 entry hike/bike dialog (HANDOFF-03) and is fixed for the entire planning session. **ROUTE-03 was cut**. Phase 19's requirement list is WAYP-01/02/03, ROUTE-01/02/04/05 (ROUTE-03 removed).

**Blocked segment & retry**
- **D-08:** A blocked segment (ROUTE-05 — failed to auto-route) renders as a dashed, red/warning-colored line, visually distinct from both a normal Valhalla-routed segment and a straight-line (auto-routing-off) segment.
- **D-09:** Retry lives on the blocked segment itself — tapping a blocked segment retries auto-routing for that segment. This reuses the same tap-on-segment gesture as WAYP-03's insert-anchor interaction; on a *blocked* segment specifically, tap means retry instead of insert (mutually exclusive by segment state, not competing gestures).

**Undo/redo**
- **D-10:** Undo/redo (ROUTE-04) are icon buttons in the screen's app bar — not grouped with the top-right map controls.
- **D-11:** Undo/redo buttons are disabled (grayed out, non-interactive) when their respective history direction is empty — standard undo/redo UX, not always-tappable no-ops.

### Claude's Discretion
None flagged — all gray areas resolved to explicit user decisions in the discussion session (mostly the recommended option, except the profile-switch scope cut).

### Deferred Ideas (OUT OF SCOPE)
- **PLANNER-07** (v2/deferred): mid-session travel-profile switching, cut from ROUTE-03 during this discussion. Profile is fixed at entry via HANDOFF-03 instead.

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| WAYP-01 | User can tap the map to add a route anchor to the in-progress route | D-03/D-04 hit-testing order; `MapEventClick`/`onEvent` pattern already proven in `trail_map.dart`; see Architecture Patterns → Map Tap Routing |
| WAYP-02 | User can drag an existing route anchor to reposition it, with connected segments re-resolving | `TrailMarkerLayer`'s `onPanStart/Update/End` pattern (VERIFIED coexists with full native map gestures via `trail_create_screen.dart`'s shipped waypoint-drag feature); web precedent `recalculateRoute()` for the ≤2-adjacent-segment recompute; see Code Examples |
| WAYP-03 | User can tap an existing route segment to insert a new route anchor between its endpoints | `MapController.featuresAtPoint(Offset, {layerIds})` VERIFIED via source for segment hit-testing; web precedent `splitSegment` (geometric split, no re-route) vs `handleSegmentDragEnd` (full Valhalla recompute) — see Architecture Patterns → Segment Insert Strategy (flagged as Open Question) |
| ROUTE-01 | Toggle auto-routing on (Valhalla via `/api/v1/valhalla/route`, foot/bike only) or off (straight-line) | Exact request/response shape VERIFIED from `web/src/lib/stores/valhalla_store.svelte.ts`'s `calculateRouteBetween`; proxy endpoint confirmed unchanged; see Code Examples → Valhalla Route Call |
| ROUTE-02 | Toggling auto-routing re-resolves all existing segments to the new mode | Same call pattern, batched with `Future.wait` over all segment pairs (mirrors web's `Promise.all` in `recalculateRouteFromAnchors`) |
| ROUTE-04 | Undo/redo route-anchor actions (in-memory only) | Web precedent uses a `json-diff-ts` delta/reverseDelta stack — **not portable to Dart** (npm-only lib); recommend a simpler immutable-snapshot stack instead, see Architecture Patterns → Undo/Redo |
| ROUTE-05 | Blocked segment + retry, never silent straight-line fallback while auto-routing is on | Novel UX vs. the web precedent (which reverts/toasts on failure rather than persisting a blocked state) — see Common Pitfalls → "Don't copy the web app's revert-on-failure pattern" |

</phase_requirements>

## Project Constraints (from CLAUDE.md)

- Flutter/Dart mobile app under `app/lib/`; Riverpod 3.x + `riverpod_annotation` codegen (`part 'x.g.dart'`, `@riverpod`) is the established state-management convention — do not introduce a different state library.
- Naming conventions: snake_case files (`route_anchor_provider.dart`), PascalCase classes/types, camelCase functions/variables, `is`-prefixed booleans.
- Strict TypeScript on the web side is irrelevant to this phase (no SvelteKit changes); Dart strict-mode conventions (typed everything, explicit return types) apply instead, matching the rest of `app/lib/`.
- No backend (Go/PocketBase) or SvelteKit changes for this phase — `/api/v1/valhalla/route` already exists and needs no modification (re-confirmed by this research).
- GSD workflow enforcement note: file-changing work must go through a GSD command (`/gsd-execute-phase` etc.), not ad hoc edits.

## Summary

Phase 19 has an unusually strong precedent: **the web app already ships a from-scratch route planner** (`web/src/routes/trail/edit/[id]/+page.svelte` + `web/src/lib/stores/valhalla_store.svelte.ts` + `web/src/lib/models/valhalla.ts` + `web/src/lib/util/valhalla_anchor_util.ts`). It uses the exact same vocabulary the user chose in CONTEXT.md ("anchor", not "waypoint") and calls the exact same `/api/v1/valhalla/route` endpoint this phase targets. This is high-value prior art: it fixes the Valhalla request/response shape, the segment-recompute math for a drag (only the ≤2 touched segments are recalculated, not the whole route), and the geometric-split approach for a plain segment-tap insert. It also has one gap directly relevant to ROUTE-05: on a failed route calculation it shows a toast and reverts, rather than persisting a "blocked" segment state — so ROUTE-05's UX must be built fresh, not ported.

Both of STATE.md's open v1.5 research flags are now resolved by direct source reads of the installed `maplibre`/`maplibre_platform_interface` 0.3.5 packages (not changelog/GitHub-discussion secondhand knowledge as previously rated):

1. **`MapGestures` API surface** — confirmed: a plain `class MapGestures({required rotate, pan, zoom, pitch})` with `.all({...})`/`.none({...})` named constructors that accept per-gesture overrides (e.g. `MapGestures.all(rotate: false)`). More importantly, `app/lib/routes/trail_create_screen.dart` **already ships** `TrailMarkerLayer`'s draggable-marker `GestureDetector.onPanStart/Update/End` pattern on a `TrailMap` running with full, un-disabled gestures (`disabled` defaults to `false` → `MapGestures.all()`). This is not a hypothetical spike target — it is a live, shipped feature proving the coexistence D-04/D-05 assume. Confidence raised from MEDIUM to HIGH.
2. **Generation-counter/CancelToken race guard** — no existing precedent for this pattern exists anywhere in `app/lib` (verified via repo-wide grep). Riverpod's own official docs (`riverpod.dev/docs/how_to/cancel`) show a `ref.onDispose` + local-flag idiom, but that pattern is scoped to `autoDispose` *family* providers that get torn down per-request — it does not directly solve overlapping in-flight requests against the *same* long-lived notifier instance (our case: one anchor dragged twice in quick succession, or auto-routing toggled mid-flight). The correct idiom for this phase is a **per-segment `CancelToken` + generation counter combo**, detailed in Architecture Patterns → Race-Guard Pattern below. This remains a genuinely novel pattern for this codebase (MEDIUM confidence — the mechanism is standard Dio/Riverpod practice, but untested in this specific class-based-Notifier shape until built).

A separate, concrete finding not flagged in STATE.md but discovered during this research: the app's existing `PolylineUtil.decode()` (`app/lib/util/polyline_util.dart`) hardcodes polyline **precision 5** — correct for the app's own GPX-derived trail polylines, but **wrong** for Valhalla's `/route` response, which encodes `trip.legs[0].shape` at **precision 6** (confirmed against the web app's own `decodePolyline(str, precision = 6)` default, itself commented "default 6 for Valhalla"). This phase is the *first* Flutter consumer of an encoded Valhalla polyline (the existing `/valhalla/navigate` path already returns decoded numeric arrays, sidestepping this issue entirely) — reusing `PolylineUtil.decode` unmodified against a Valhalla `/route` response will silently produce wrong coordinates. See Common Pitfalls.

**Primary recommendation:** Build a new `RouteAnchor` freezed model + a class-based `@riverpod` notifier (`route_anchor_provider.dart`) holding an immutable `RouteAnchorsState` (anchors, segments, auto-routing flag, fixed travel profile, undo/redo snapshot stacks). Render anchors as `ml.WidgetLayer` markers (reusing `TrailMarkerLayer`'s drag/scale/hit-radius pattern) and segments as native GL `LineStyleLayer`s over a single `GeoJsonSource`, filtered by a `state` property (`routed`/`straight`/`blocked`) into three paint variants. Use `MapController.featuresAtPoint(event.screenPoint, layerIds: [...])` for segment hit-testing and a per-segment `CancelToken` + generation counter to guard against out-of-order Valhalla responses.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Route anchor add/drag/insert gestures (WAYP-01/02/03) | Mobile Client (Flutter gesture layer: `GestureDetector`, `MapEventClick`) | State/Provider (Riverpod notifier) | Gesture recognition must live in the widget/native-event layer; the notifier is the single source of truth the gesture handlers mutate |
| Ordered anchor list + ascending renumbering (D-02) | State/Provider (Riverpod notifier) | — | Pure derived client state; number is computed from list index, never stored independently |
| Auto-routing toggle + segment resolution (ROUTE-01/02) | State/Provider (Riverpod notifier) | External Service (Valhalla, via existing SvelteKit proxy) | Notifier decides *when* to call the proxy; Valhalla performs the actual routing computation — no logic duplicated client-side |
| Blocked segment state + retry (ROUTE-05/D-08/D-09) | State/Provider (Riverpod notifier) | Mobile Client (segment-tap gesture triggers retry) | Failure state and its guard (never silently fall back) belongs in state; the user-facing retry trigger is a map gesture |
| Undo/redo history (ROUTE-04) | State/Provider (Riverpod notifier) | — | In-memory snapshot stack; explicitly no persistence layer per requirement text |
| Native segment/marker rendering | Native Map Renderer (MapLibre GL native, via `StyleController`) | Mobile Client (`ml.WidgetLayer` for interactive markers) | GL style layers draw segment polylines; Flutter widgets draw interactive (tappable/draggable) markers layered on top, matching the existing `TrailMap`/`TrailMarkerLayer` split |
| Valhalla routing computation | External Service (Valhalla instance) | Backend Proxy (`/api/v1/valhalla/route`, SvelteKit, unchanged) | Existing infra; this phase is a pure consumer, confirmed no backend changes needed |

## Standard Stack

No new packages are introduced by this phase — everything needed already exists in `app/pubspec.yaml` (confirmed by reading the file directly).

### Core (existing, reused)
| Library | Version (pubspec.lock) | Purpose | Why Standard |
|---------|-----|---------|--------------|
| `maplibre` | 0.3.5 (exact-pinned) | Native map host, `MapGestures`, `WidgetLayer`, `StyleController`, `MapController.featuresAtPoint` | Already the sole map stack post-Phase-18 migration; pinned per CLEAN-03 |
| `flutter_riverpod` / `riverpod_annotation` | `^3.3.1` / `4.0.2` (locked exact) | State management for the new route-anchor provider | Sole state pattern used throughout `app/lib/provider/` |
| `dio` | `^5.9.2` | HTTP client for the Valhalla `/route` proxy call, incl. `CancelToken` | Already the app's HTTP client (`apiProvider`) |
| `gpx` | `^2.3.0` | Not directly needed for in-progress route state (Phase 21 synthesizes the final GPX at handoff), but the app's existing `GpxMappingUtils` extensions are precedent for any along-track math needed | Existing dependency |
| `font_awesome_flutter` | `^11.0.0` | Icons: `FontAwesomeIcons.route` (auto-routing toggle, VERIFIED present at `font_awesome_flutter.dart:20861`), `FontAwesomeIcons.arrowRotateLeft`/`arrowRotateRight` (undo/redo, VERIFIED present at `font_awesome_flutter.dart:1110`/`1138`) | Matches UI-SPEC's icon contract exactly |
| `collection` | `^1.19.1` | Deep list/map equality helpers if manual snapshot-diffing is needed | Already a direct dependency |

### Supporting (existing helpers to reuse directly)
| Library/File | Purpose | When to Use |
|---------|---------|-------------|
| `app/lib/util/gpx_util.dart` — `costingForCategory` | Foot/bike costing string ('pedestrian'/'bicycle') | Not strictly needed in-planner (profile comes from the Phase-21 entry dialog directly as a costing string, not a trail category), but keep the same two costing string literals for consistency |
| `app/lib/util/gpx_util.dart` — `buildNavShape` | Downsampling ≥500-point shapes for Valhalla requests | Per-segment calls (2 anchors) are always small; not needed for individual segment routing, but relevant if Phase 20's `/valhalla/height` call needs the full synthesized route |
| `app/lib/util/polyline_util.dart` — `PolylineUtil.decode/encode` | Encoded-polyline codec | **Needs a precision parameter added** (defaults to 5; Valhalla `/route` responses are precision 6) — see Common Pitfalls |
| `app/lib/components/map/trail_layer.dart` — `TrailMarkerLayer`, `_buildCircularMarker`, `kTrailRouteColor` | Marker widget shape, drag pattern, route line color constant | Direct visual/interaction precedent per D-04/D-05 and UI-SPEC |
| `app/lib/components/base/trail_map.dart` — `TrailMap` | Native map host: `onMapCreated`/`onStyleLoaded` race-buffer pattern, `onEvent`/`MapEventClick` routing, `controls` top-right slot | Host-widget shape precedent (a new `RoutePlannerMap` or similar should follow this shape, not extend `TrailMap` directly since it isn't bound to a persisted `Trail`) |
| `app/lib/provider/toast_provider.dart` | Toast/snackbar messaging | For the blocked-segment/retry copy strings from UI-SPEC |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Immutable snapshot-stack undo/redo | Porting `json-diff-ts`-style delta/reverseDelta (the web app's approach) | No Dart equivalent exists; hand-rolling a generic object-diff library is far more complex than snapshotting a small list (a route-planner session realistically has a few dozen anchors at most) — snapshotting is strictly simpler and equally correct here |
| Geometric segment-split on tap-insert (no Valhalla call) | Always re-call Valhalla for both new segments on every insert | Web's own `handleSegmentClick`/`splitSegment` uses the geometric approach for a plain tap (no network round-trip, no risk of the new sub-segments drifting from the already-rendered path); Valhalla-recompute-both is only used by the web app for a *different* gesture (dropping an anchor from a list onto a segment). See Architecture Patterns → Segment Insert Strategy (flagged as Open Question since CONTEXT.md doesn't explicitly resolve it) |
| Per-segment `CancelToken` + generation counter | Riverpod's official `ref.onDispose`-based debounce pattern verbatim | The official pattern is designed for `autoDispose` *family* providers recreated per keystroke/request; our notifier is a single long-lived instance mutated by many gesture events, so the guard must be scoped per-segment inside the notifier, not at the provider-disposal level |

## Package Legitimacy Audit

Not applicable — this phase introduces zero new third-party packages. All libraries referenced above are pre-existing, already-audited direct dependencies in `app/pubspec.yaml`/`pubspec.lock`.

## Architecture Patterns

### System Architecture Diagram

```text
User gesture (tap / drag / segment-tap)
        │
        ▼
┌───────────────────────────────────────────┐
│ Native MapLibreMap (ml.MapLibreMap)        │
│  onEvent → MapEventClick(point,screenPoint)│
│  children: [ WidgetLayer (anchor markers) ]│
└───────────────┬─────────────────────────────┘
                │  (a) marker hit? (36px radius, D-04) → handled inside
                │      WidgetLayer's own GestureDetector, short-circuits (b)
                │  (b) else: controller.featuresAtPoint(screenPoint,
                │      layerIds:['route-segments-hit']) → segment tap?
                │  (c) else: empty-map tap → append new anchor (D-03)
                ▼
┌───────────────────────────────────────────┐
│ RouteAnchors notifier (Riverpod)           │
│  - mutates ordered anchor list             │
│  - recomputes 1-2 affected segments        │
│  - pushes an immutable snapshot to undo    │
└───────────────┬─────────────────────────────┘
                │  if autoRoutingEnabled: per-segment routing call
                ▼
┌───────────────────────────────────────────┐
│ Dio → POST /api/v1/valhalla/route          │
│  (existing SvelteKit proxy, unchanged)     │
│  → Valhalla instance                       │
└───────────────┬─────────────────────────────┘
                │  trip.legs[0].shape (polyline6) + summary.time
                ▼
┌───────────────────────────────────────────┐
│ RouteAnchors notifier                      │
│  - decode polyline (precision 6!)          │
│  - mark segment routed / blocked on error  │
│  - discard if generation counter stale     │
└───────────────┬─────────────────────────────┘
                │  state change
                ▼
┌───────────────────────────────────────────┐
│ StyleController.updateGeoJsonSource(       │
│   id: 'route-segments', data: geojson)     │
│  3 filtered LineStyleLayers                │
│  (routed / straight / blocked)             │
└───────────────────────────────────────────┘
```

### Recommended Project Structure
```
app/lib/
├── models/
│   └── route_anchor.dart          # freezed RouteAnchor model (D-01: never named Waypoint)
├── provider/
│   └── route_anchor_provider.dart # @riverpod class RouteAnchors extends _$RouteAnchors
├── components/
│   └── map/
│       └── route_anchor_layer.dart # WidgetLayer of anchor markers, mirrors TrailMarkerLayer
├── util/
│   ├── route_segment_util.dart    # geometric split, generation/cancel-guard helpers
│   └── polyline_util.dart         # MODIFY: add precision param (default 5, pass 6 for Valhalla)
└── routes/
    └── route_planner_screen.dart  # hosts the native map + app bar undo/redo (D-10)
```

### Map Tap Routing (WAYP-01/03, D-03/D-04)
**What:** Disambiguate marker-tap vs. segment-tap vs. empty-map-tap inside the single `onEvent`/`MapEventClick` callback `TrailMap` already establishes.
**When to use:** Every tap on the route-planner map.
**Example:**
```dart
// Source: app/lib/components/base/trail_map.dart (existing onEvent pattern),
// combined with maplibre_platform_interface-0.3.5/lib/src/map_controller.dart
// (featuresAtPoint — VERIFIED present, takes screenPoint directly, no
// Geographic→Offset conversion needed since MapEventClick already carries both).
onEvent: (event) {
  if (event is ml.MapEventClick) {
    // (a) Marker hits are handled inside RouteAnchorLayer's own
    // WidgetLayer/GestureDetector — a marker tap never reaches here because
    // the marker widget consumes the tap first (matches D-04's "marker wins
    // on overlap": the marker's own GestureDetector.onTap fires before the
    // map's onEvent for taps within the marker's hit area).

    // (b) Segment hit-test: query only the invisible wide hit-test layer.
    final hits = controller.featuresAtPoint(
      event.screenPoint,
      layerIds: ['route-segments-hit'],
    );
    if (hits.isNotEmpty) {
      final segmentIndex = hits.first.properties['segmentIndex'] as int;
      final state = hits.first.properties['state'] as String; // routed/straight/blocked
      if (state == 'blocked') {
        ref.read(routeAnchorsProvider.notifier).retrySegment(segmentIndex); // D-09
      } else {
        ref.read(routeAnchorsProvider.notifier).insertAnchorOnSegment(
          segmentIndex, event.point,
        ); // WAYP-03
      }
      return;
    }

    // (c) Empty map: append (D-03).
    ref.read(routeAnchorsProvider.notifier).appendAnchor(event.point);
  }
},
```
**Pitfall this avoids:** A thin 3-5px rendered line (per UI-SPEC's Segment Rendering Contract) is a tiny, hard-to-hit target with pixel-accurate `featuresAtPoint` queries. Register a **second, invisible, wide-`line-width` layer** (e.g. 24px, `'line-opacity': 0`) sharing the same source, and query *that* layer id for hit-testing — mirrors the existing 36px invisible marker hit-radius precedent (D-04) applied to segments instead of markers.

### Segment Rendering (D-08, UI-SPEC's Segment Rendering Contract)
**What:** One `GeoJsonSource` of all segments (as `LineString` features with a `state` + `segmentIndex` property each), three `LineStyleLayer`s filtered by `state`, plus one invisible wide hit-test layer.
**Example:**
```dart
// Source: maplibre_platform_interface-0.3.5/lib/src/style/layers/line_style_layer.dart
// (filter param VERIFIED present on StyleLayerWithSource) + style_controller.dart
// (updateGeoJsonSource VERIFIED present — efficient in-place update, no
// remove/re-add flicker on every anchor mutation).
await style.updateGeoJsonSource(id: 'route-segments', data: geojson);

// Casing/route pair for the "routed" state only (matches TrailLayer.add's
// 9px-white-casing-under-5px-colored-line pattern, kTrailRouteColor).
await style.addLayer(ml.LineStyleLayer(
  id: 'route-segments-routed-casing',
  sourceId: 'route-segments',
  filter: const ['==', ['get', 'state'], 'routed'],
  paint: const {'line-color': '#ffffff', 'line-width': 9},
));
await style.addLayer(ml.LineStyleLayer(
  id: 'route-segments-routed',
  sourceId: 'route-segments',
  filter: const ['==', ['get', 'state'], 'routed'],
  paint: const {'line-color': '#3549bb', 'line-width': 5},
));
// Straight-line: thinner, reduced-opacity, no casing.
await style.addLayer(ml.LineStyleLayer(
  id: 'route-segments-straight',
  sourceId: 'route-segments',
  filter: const ['==', ['get', 'state'], 'straight'],
  paint: const {'line-color': '#3549bb', 'line-width': 3, 'line-opacity': 0.55},
));
// Blocked: dashed red, no casing.
await style.addLayer(ml.LineStyleLayer(
  id: 'route-segments-blocked',
  sourceId: 'route-segments',
  filter: const ['==', ['get', 'state'], 'blocked'],
  paint: const {
    'line-color': '#EF5350', // Colors.red.shade400
    'line-width': 3,
    'line-dasharray': [2, 2],
  },
));
// Invisible wide hit-test layer, ALL states, for WAYP-03/D-09 tap-detection.
await style.addLayer(ml.LineStyleLayer(
  id: 'route-segments-hit',
  sourceId: 'route-segments',
  paint: const {'line-color': '#000000', 'line-width': 24, 'line-opacity': 0},
));
```
**Note on `line-dasharray`:** this is a MapLibre style-spec **paint** property (not layout); the `LineStyleLayer` constructor accepts arbitrary paint maps with no compile-time property allow-list (`paint` is typed `Map<String, Object>`), so this is a runtime-only contract — verify visually on-device, standard practice already established by this codebase's other native style layers.

### Undo/Redo (ROUTE-04)
**What:** An immutable snapshot stack, not a diff/patch stack.
**When to use:** Every anchor mutation that successfully resolves (add/drag/insert/delete/reorder) pushes the *previous* full `(anchors, segments)` pair onto `undoStack` and clears `redoStack`.
**Rationale:** The web app's `valhalla_store.svelte.ts` uses `json-diff-ts` deltas (`applyChangeset`/`diff`) — there is no Dart equivalent, and porting a generic object-diff library is disproportionate complexity for what is realistically a small, bounded list (a mobile route plan is not going to have thousands of anchors). A `List<RouteAnchorsSnapshot>` (each snapshot a deep, immutable copy of anchors+segments) is simpler, trivially correct, and avoids needing a network round-trip on undo (the segment's already-resolved polyline is stored in the snapshot, not recomputed).
```dart
@freezed
abstract class RouteAnchorsSnapshot with _$RouteAnchorsSnapshot {
  const factory RouteAnchorsSnapshot({
    required List<RouteAnchor> anchors,
    required List<RouteSegment> segments,
  }) = _RouteAnchorsSnapshot;
}

// Inside the notifier:
void _pushUndo() {
  state = state.copyWith(
    undoStack: [...state.undoStack, RouteAnchorsSnapshot(anchors: state.anchors, segments: state.segments)],
    redoStack: const [], // any new action clears redo, matches D-11 + standard UX
  );
}

void undo() {
  final stack = state.undoStack;
  if (stack.isEmpty) return; // D-11: button is disabled anyway, but guard defensively
  final previous = stack.last;
  final redoSnapshot = RouteAnchorsSnapshot(anchors: state.anchors, segments: state.segments);
  state = state.copyWith(
    anchors: previous.anchors,
    segments: previous.segments,
    undoStack: stack.sublist(0, stack.length - 1),
    redoStack: [...state.redoStack, redoSnapshot],
  );
}
```

### Race-Guard Pattern for Out-of-Order Valhalla Responses (STATE.md flag #2)
**What:** A per-segment `CancelToken` + monotonically increasing generation counter, keyed by a stable segment key (e.g. the pair of anchor ids, not array index — indices shift on insert/delete).
**Why both mechanisms:** Cancelling the Dio request (`CancelToken.cancel()`) stops the network call/parsing work early, but a response that is already in-flight when cancel fires can still resolve the `Future` with a `DioException(type: cancel)` — catching and swallowing that specific exception type is sufficient on its own, but pairing it with a generation counter also protects against a *different* narrow race: two calls dispatched back-to-back where the first's cancel token is replaced (not cancelled) before the network layer has actually cancelled it, letting an old, non-error response slip through and be applied out of order.
```dart
// Inside RouteAnchors notifier — per-segment guards, not provider-level.
final Map<String, CancelToken> _inFlight = {};
final Map<String, int> _generation = {};

Future<void> _resolveSegment(String segmentKey, RouteAnchor a, RouteAnchor b) async {
  _inFlight[segmentKey]?.cancel();
  final token = CancelToken();
  _inFlight[segmentKey] = token;
  final myGeneration = (_generation[segmentKey] ?? 0) + 1;
  _generation[segmentKey] = myGeneration;

  try {
    final api = ref.read(apiProvider);
    final res = await api.post('/valhalla/route', data: {
      'directions_type': 'none',
      'locations': [
        {'lat': a.lat, 'lon': a.lon},
        {'lat': b.lat, 'lon': b.lon},
      ],
      'costing': state.travelProfile, // fixed at entry, D-07 — 'pedestrian' | 'bicycle'
    }, cancelToken: token);

    // Stale-response guard: a newer request may have started (and even
    // completed) while this one was in flight.
    if (_generation[segmentKey] != myGeneration) return;

    final shape = res.data['trip']['legs'][0]['shape'] as String;
    final points = PolylineUtil.decode(shape, precision: 6); // NOT the default 5!
    _applySegment(segmentKey, points, blocked: false);
  } on DioException catch (e) {
    if (e.type == DioExceptionType.cancel) return; // superseded, not a real failure
    if (_generation[segmentKey] != myGeneration) return;
    _applySegment(segmentKey, const [], blocked: true); // ROUTE-05
  }
}
```
**Confidence:** MEDIUM — the individual mechanisms (Dio `CancelToken`, a local generation counter) are standard, well-documented practice, and Dio 5.9.2 is already a direct dependency, but this exact combined idiom has no precedent in this codebase and should be spiked/unit-tested during Phase 19 execution rather than assumed correct on first write.

### Segment Insert Strategy (WAYP-03) — Open Question, not fully resolved by CONTEXT.md
The web app uses **two different strategies** depending on which gesture triggers an insert:
- `handleSegmentClick` (a plain tap on a segment — the direct analog of Flutter's WAYP-03) calls `splitSegment`, which does **not** call Valhalla again. It finds the nearest existing vertex on the already-rendered polyline to the tapped point and slices the point array in two there — correct by construction because the split point lies on the existing routed path.
- `handleSegmentDragEnd` (dropping a *marker* onto a segment — a different gesture, not present in Flutter's D-03/D-04 model) *does* call Valhalla twice, once for each new sub-segment.

**Recommendation:** For Flutter's WAYP-03 (a plain tap-to-insert), follow the geometric-split approach for segments in the `routed` or `straight` state (no extra network round-trip, no risk of the new sub-segments drifting from what's already on screen). This should be confirmed with the user/planner since CONTEXT.md's D-09 only speaks to *blocked*-segment tap behavior, not the routed/straight case explicitly.

```dart
// Source: web/src/lib/stores/valhalla_store.svelte.ts's splitSegment (JS),
// translated to the equivalent Dart shape.
(RouteSegment, RouteSegment) splitSegmentAt(RouteSegment segment, Geographic tapPoint) {
  final points = segment.polyline; // List<Geographic>
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

### Anti-Patterns to Avoid
- **Reusing `Waypoint`/`TrailMarkerLayer` directly:** D-01 explicitly requires a new type; `TrailMarkerLayer` is bound to a persisted `Trail`/`Waypoint` — copy its *shape* (drag pattern, marker styling), not the class itself.
- **Falling back to a straight line silently when a Valhalla call fails while auto-routing is ON:** this is explicitly forbidden by ROUTE-05; the web app's toast-and-revert pattern is the wrong model to copy here — persist the `blocked` state instead.
- **Recomputing the entire route on every single anchor mutation:** both this phase's own design and the web precedent (`recalculateRouteFromAnchors`) only recompute the 1-3 segments actually touched by a given mutation; full-route recomputation is unnecessary network load and slower UX.
- **Decoding a Valhalla `/route` shape with the app's default `PolylineUtil.decode` precision:** see Common Pitfalls.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Unique per-anchor identity for stable list operations across reorders | A custom random-string generator (the web app's `cryptoRandomString({length:15})` has no Flutter equivalent dependency in this project) | `UniqueKey().toString()` (`package:flutter/foundation.dart`, already available, zero new dependency) | Flutter ships exactly this primitive for generating unique, collision-free tokens; no need for a random-string package |
| Segment vs. marker vs. empty-map tap disambiguation | Manual point-in-polygon/distance-to-line math against Dart-side coordinate lists | `MapController.featuresAtPoint`/`queryLayers` (native GL hit-testing, VERIFIED present) | The native renderer already does pixel-accurate hit-testing against actual rendered geometry — reimplementing this in Dart would be slower and diverge from what's visually on screen (e.g. ignoring current zoom/projection) |
| Polyline encode/decode | A new encoder/decoder | `app/lib/util/polyline_util.dart`'s existing `PolylineUtil` (extended with a precision parameter) | Already implemented, tested indirectly via existing trail-polyline usage; just needs a precision fix, not a rewrite |
| Undo/redo object diffing | A ported/hand-written generic object-diff engine (mirroring `json-diff-ts`) | Immutable full-snapshot stack (see Architecture Patterns → Undo/Redo) | For the realistic scale of a mobile route plan, snapshotting is simpler, has zero third-party dependency risk, and needs no network round-trip to undo |

**Key insight:** Every non-trivial piece of this phase (routing call shape, undo/redo architecture skeleton, segment-recompute scoping, geometric split) already has a working reference implementation in this exact repository (the web trail editor) — the main engineering work is translating that reference from Svelte/TypeScript idioms to Dart/Riverpod idioms and closing the one deliberate UX gap (ROUTE-05's persistent blocked state) the web app doesn't have.

## Common Pitfalls

### Pitfall 1: Valhalla polyline precision mismatch (HIGH confidence — directly verified via source comparison)
**What goes wrong:** Decoding a Valhalla `/route` response's `trip.legs[0].shape` with the app's existing `PolylineUtil.decode()` (hardcoded to precision 5, i.e. `/1E5`) produces coordinates that are wrong by roughly an order of magnitude, because Valhalla encodes at precision 6.
**Why it happens:** The app's only prior polyline consumer (`trailPolyline` provider, reading `trail.polyline` from the trail record) and the turn-by-turn `/valhalla/navigate` path (which returns pre-decoded numeric shape arrays, not an encoded string) never needed precision-6 decoding, so the mismatch was never hit before. This phase is the first to decode an actual Valhalla-encoded polyline string in Dart.
**How to avoid:** Add a `precision` parameter to `PolylineUtil.decode`/`encode` (default 5, pass 6 explicitly for any Valhalla `/route` response), mirroring `web/src/lib/util/polyline_util.ts`'s `decodePolyline(str, precision = 6)`.
**Warning signs:** Routed segments rendering as a tiny cluster of points near (0,0) or wildly off-map; segment length wildly disagreeing with the straight-line distance between its anchors.

### Pitfall 2: Reusing the web app's error-toast-and-revert pattern for ROUTE-05
**What goes wrong:** Copying `addAnchorAndRecalculate`'s catch block (show toast, don't persist a failure state) would violate ROUTE-05's explicit requirement that a failed segment must remain visibly blocked and retryable, never silently reverting or falling back to a straight line while auto-routing is on.
**Why it happens:** The web app's route editor predates ROUTE-05's blocked-segment UX; it was designed around "retry the whole gesture" rather than "leave a persistent, taggable failure state."
**How to avoid:** On a Valhalla call failure (non-cancel), set that segment's `state` to `blocked` in the notifier and leave it rendered (dashed, red) rather than reverting the anchor list or removing the segment.
**Warning signs:** A blocked segment that disappears or silently becomes a straight line on the next unrelated state update.

### Pitfall 3: Segment index drift after insert/delete/reorder
**What goes wrong:** If segments are keyed by array position (`index`) rather than a stable anchor-pair key, an in-flight routing request for "segment 2" can silently apply to the wrong segment after the user inserts/deletes/reorders anchors mid-flight.
**Why it happens:** The web app itself has to work around this (see `recalculateRouteFromAnchors`'s careful `fromIndex`/`toIndex` remapping logic when an anchor moves) — it's an inherent hazard of index-based segment identity, not a Flutter-specific issue.
**How to avoid:** Key in-flight requests and generation counters by a stable pair of anchor ids (e.g. `'${anchorA.id}_${anchorB.id}'`), not array index; recompute the displayed segment list from current anchor order every render, but keep the request/generation bookkeeping keyed by anchor identity.
**Warning signs:** A drag/insert/delete performed in quick succession produces a segment rendered in the wrong position, or an old blocked-state indicator lingering on a segment that has since been resolved.

### Pitfall 4: Thin rendered line as the hit-test target
**What goes wrong:** Querying `featuresAtPoint` against the *visible* 3-5px line layer directly makes WAYP-03/D-09 taps unreliable — users will frequently miss.
**Why it happens:** Native GL hit-testing is essentially pixel-exact against rendered geometry; a thin line has a tiny screen footprint.
**How to avoid:** Add a separate, invisible, wide (e.g. 24px) `line-opacity: 0` layer sharing the same source and query *that* layer id for hit-testing, exactly mirroring D-04's existing 36px invisible marker hit-radius precedent.
**Warning signs:** Manual testing shows segment taps "not registering" near the line but registering when tapped exactly on the pixel center.

### Pitfall 5: `onStyleLoaded`/`onMapCreated` ordering race (pre-existing, project-wide)
**What goes wrong:** The native platform channel does not reliably fire `onMapCreated` before `onStyleLoaded`; a route-planner screen that adds its segment/marker layers directly inside `onStyleLoaded` without buffering will silently no-op if the controller isn't set yet.
**Why it happens:** Documented, previously-hit bug class (STATE.md: hit twice in Phase 16/17).
**How to avoid:** Reuse `TrailMap`'s `_pendingStyle` buffer-and-replay pattern verbatim in the new route-planner map host.
**Warning signs:** Segments/markers occasionally fail to render on cold app start but work fine on a subsequent hot reload/style swap (e.g. theme toggle).

## Code Examples

### Valhalla Route Call (ROUTE-01) — exact verified request/response shape
```dart
// Source: web/src/lib/stores/valhalla_store.svelte.ts calculateRouteBetween
// (VERIFIED — this is the actual shape the existing, unmodified
// /api/v1/valhalla/route proxy expects and returns).
final response = await api.post('/valhalla/route', data: {
  'directions_type': 'none',
  'locations': [
    {'lat': anchorA.lat, 'lon': anchorA.lon},
    {'lat': anchorB.lat, 'lon': anchorB.lon},
  ],
  'costing': travelProfile, // 'pedestrian' | 'bicycle' — fixed at entry, D-07
});

final trip = response.data['trip'] as Map<String, dynamic>;
final leg = (trip['legs'] as List)[0] as Map<String, dynamic>;
final encodedShape = leg['shape'] as String;
final durationSeconds = (trip['summary'] as Map<String, dynamic>)['time'] as num;

final points = PolylineUtil.decode(encodedShape, precision: 6); // NOT default 5
```

### Freezed `RouteAnchor` model (D-01)
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

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| flutter_map + custom draggable-marker plugins for route editing | Native `maplibre` 0.3.5 `WidgetLayer`/`GestureDetector` pattern | Phase 15-18 (v1.4 migration, already complete) | This phase builds entirely on the post-migration native stack; no flutter_map-era pattern is relevant here |
| Web route editor's `json-diff-ts` delta-based undo/redo | Recommend: immutable full-snapshot stack for the Dart port | This research session | Simpler for Dart, no equivalent library needed |

**Deprecated/outdated:** N/A — no deprecated APIs identified in the reviewed package versions for this phase's needs.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Insert-on-a-plain-segment-tap (WAYP-03) should use the geometric-split strategy (no new Valhalla call) rather than always re-routing both new sub-segments | Architecture Patterns → Segment Insert Strategy | If wrong, WAYP-03 tasks would need to add a Valhalla call + loading/blocked-state handling identical to WAYP-02's drag-resolve path instead of the simpler split; low risk either way is buildable, but affects task count/shape — flagged as an explicit open question for the planner/user rather than asserted as locked |
| A2 | A per-segment `CancelToken` + generation-counter combo (not just one or the other) is the right race-guard shape for this specific long-lived-notifier architecture | Architecture Patterns → Race-Guard Pattern | If the simpler generation-counter-only (or CancelToken-only) approach turns out sufficient, the combo is extra code with no functional cost — low risk, but should be validated with a quick unit test during execution rather than assumed |
| A3 | `Geographic` (from `geobase`) provides correct value equality for freezed-model structural comparisons (undo/redo snapshot equality checks, if used) | Standard Stack, Undo/Redo | Low risk — verified an `operator ==` override exists on `Geographic`/`Position`, but the exact field-by-field semantics (e.g. floating-point epsilon handling) weren't unit-tested in this research pass |

**If this table is empty:** N/A — see rows above; both are inherited *design* choices this research recommends but the CONTEXT.md discussion did not explicitly lock, not unverified factual claims about external libraries.

## Open Questions

1. **Should WAYP-03 (plain tap-to-insert on a routed/straight segment) call Valhalla again, or use a geometric split?**
   - What we know: the web app's equivalent gesture (`handleSegmentClick`) uses a geometric split (no Valhalla call); a *different* web gesture (`handleSegmentDragEnd`) does call Valhalla.
   - What's unclear: CONTEXT.md's D-09 only resolves this question for the *blocked* segment case (tap = retry). It's silent on whether a normal insert should re-route or just split.
   - Recommendation: default to the geometric-split approach (simpler, faster, no extra network call) unless the planner/user specifically wants insert-triggered re-routing; confirm during planning.

2. **Exact segment-key stability strategy across renumbering (D-02)**
   - What we know: anchor *display numbers* are derived from list order and must never be stored independently (per CONTEXT.md's Specific Ideas). Segment identity for the routing/undo bookkeeping should be anchor-id-pair based, not index-based (Pitfall 3).
   - What's unclear: whether the planner wants `RouteSegment` to store `beforeAnchorId`/`afterAnchorId` explicitly (recommended) vs. deriving segment identity implicitly from adjacent-anchor-in-list-order at read time only.
   - Recommendation: store explicit `beforeAnchorId`/`afterAnchorId` on `RouteSegment` for stability across concurrent async operations; this is a small modeling decision for the planner to make explicit in the plan.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `maplibre` native map plugin | All map rendering in this phase | ✓ (already integrated, pinned exact) | 0.3.5 | — |
| Valhalla routing service (via `/api/v1/valhalla/route` proxy) | ROUTE-01/02/05 | Not directly checkable from this environment (server-side, network-dependent at runtime) | — | ROUTE-05's blocked-segment/retry UX *is* the fallback — an unreachable Valhalla instance is treated as a per-segment failure, not a hard blocker for the phase |
| Dio `CancelToken` | Race-guard pattern | ✓ (bundled with `dio` 5.9.2, no separate package) | 5.9.2 | — |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** Valhalla reachability — already has an explicit, in-scope fallback UX (ROUTE-05).

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No new auth surface — the phase reuses the existing authenticated `apiProvider`/cookie-jar Dio client unchanged |
| V3 Session Management | No | No session changes |
| V4 Access Control | No | No new authorization boundary; `/api/v1/valhalla/route` has no additional access control today and this phase does not change that (confirmed: no `locals.user` check in the existing `+server.ts` handler) |
| V5 Input Validation | Yes | Client-side validation before POSTing to `/valhalla/route`: reject/clamp anchor lat/lon to valid ranges (±90/±180) before constructing the request body; cap the number of route anchors accepted per session at a sane upper bound (the existing `buildNavShape`'s 500-point downsample precedent shows this codebase already treats unbounded point counts as a real concern) to avoid an unbounded-size payload being relayed through the SvelteKit proxy |
| V6 Cryptography | No | No new cryptographic material handled by this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Unbounded/malicious anchor count driving an oversized POST body through the existing `/valhalla/route` proxy (which has no request-size validation of its own, confirmed by reading `+server.ts`) | Denial of Service | Client-side cap on anchor count / per-segment call (each call is only 2 points, so this is a low-severity concern already largely mitigated by the per-segment call shape recommended in this research, unlike a single whole-route request) |
| Malformed/out-of-range coordinates reaching Valhalla | Tampering (of routing intent, low severity) | Standard lat/lon range validation before constructing any request body — no new library needed, simple bounds check |

## Sources

### Primary (HIGH confidence — direct source reads of installed packages/repo code)
- `app/pubspec.yaml` / `app/pubspec.lock` — exact installed/locked versions (`maplibre` 0.3.5, `riverpod_annotation` 4.0.2, `dio` 5.9.2, `font_awesome_flutter` 11.0.0)
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/options/map_gestures.dart` — `MapGestures` API surface
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/map_controller.dart` — `featuresAtPoint`, `queryLayers`
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/style_controller.dart` — `updateGeoJsonSource`
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/style/layers/line_style_layer.dart` and `style_layer.dart` — `filter` support
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/map_events.dart` — `MapEventClick(point, screenPoint)`
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/geobase-1.5.0/lib/src/coordinates/geographic/geographic.dart` — `Geographic` value equality
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/font_awesome_flutter-11.0.0/lib/font_awesome_flutter.dart` — icon existence checks
- `app/lib/components/map/trail_layer.dart`, `app/lib/components/base/trail_map.dart` — existing marker/map host patterns
- `app/lib/routes/trail_create_screen.dart` — live proof of drag-on-full-gestures coexistence
- `app/lib/util/gpx_util.dart`, `app/lib/util/polyline_util.dart` — reusable helpers and the precision bug
- `web/src/routes/api/v1/valhalla/route/+server.ts`, `web/src/lib/server/valhalla.ts` — proxy confirmed unchanged/no-auth
- `web/src/lib/stores/valhalla_store.svelte.ts`, `web/src/lib/models/valhalla.ts`, `web/src/lib/util/valhalla_anchor_util.ts`, `web/src/routes/trail/edit/[id]/+page.svelte` — full prior-art reference implementation
- `web/src/lib/util/polyline_util.ts` — precision-6 default for Valhalla, confirming the Flutter-side bug

### Secondary (MEDIUM confidence — official docs)
- [riverpod.dev/docs/how_to/cancel](https://riverpod.dev/docs/how_to/cancel) — official cancellation/debounce idiom (`ref.onDispose` + local flag), used to justify why a *different*, per-segment guard is needed for this phase's long-lived-notifier shape

### Tertiary (LOW confidence — WebSearch only, not directly verified against this project's exact code)
- General Dio `CancelToken` usage articles (Medium/codewithandrea) — corroborate that `CancelToken` is standard practice for this kind of race, but the exact combined generation-counter idiom recommended here is this research's own synthesis, not copied from any single source

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages, all versions read directly from `pubspec.lock`
- Architecture (map interaction / gesture coexistence): HIGH — both STATE.md flags resolved via direct source verification, one backed by a live shipped feature (`trail_create_screen.dart`)
- Architecture (routing/undo-redo): MEDIUM — strongly informed by an equivalent, working web implementation in this same repo, but the Dart translation (esp. the race-guard combo and snapshot-stack undo/redo) is novel and untested in this codebase
- Pitfalls: HIGH — the polyline-precision pitfall and the "don't copy the web revert pattern" pitfall are both grounded in direct source comparison, not speculation

**Research date:** 2026-07-16
**Valid until:** 2026-08-15 (30 days — stable, pinned dependency set; re-verify if `maplibre` or `riverpod_annotation` versions change before execution)
