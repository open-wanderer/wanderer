# Phase 20: Route Planner Views — Pattern Map

**Mapped:** 2026-07-16
**Files analyzed:** 8 (2 modify existing, 5 new, 1 modify existing utility)
**Analogs found:** 8 / 8 (all have strong in-repo analogs; RESEARCH.md already did most of the legwork)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `app/lib/provider/route_anchor_provider.dart` (MODIFY: add `deleteAnchor`, `reorderAnchors`) | store/provider | CRUD (in-memory) | itself — `appendAnchor`/`dragAnchor`/`insertAnchorOnSegment` (same file) | exact (extend existing mutator conventions) |
| `app/lib/provider/planned_gpx_provider.dart` (NEW) | store/provider (derived) | transform | `app/lib/provider/trail/trail_polyline_provider.dart` | role-match (derived `@riverpod` function provider) |
| `app/lib/util/gpx_util.dart` (MODIFY: add Gpx-from-points helper) | utility | transform | itself — `buildNavShape`, `GpxMappingUtils.getTotals()` (same file) | exact |
| `app/lib/components/trail/elevation_profile.dart` (MODIFY: `trail` → `Trail?`) | component | transform/render | itself (in-place adaptation) | exact |
| `app/lib/components/route_planner/route_anchor_sheet.dart` (NEW) | component | event-driven (drag/tab) | `app/lib/components/trail/waypoint_sheet.dart` | role-match (sheet chrome) + `app/lib/routes/trail_detail_map_screen.dart` (Stack composition) |
| `app/lib/components/route_planner/route_anchor_list_tab.dart` (NEW) | component | CRUD (delete/reorder) | `app/lib/routes/settings_categories_screen.dart` (lines 228-327) | exact (ReorderableListView + optimistic reorder) |
| `app/lib/components/route_planner/elevation_tab.dart` (NEW) | component | request-response (debounced fetch) | `app/lib/provider/search/global_search_provider.dart` (debounce idiom) + `elevation_profile.dart` (render) | role-match |
| `app/lib/routes/location_search_screen.dart` (NEW) | component/route | request-response | `app/lib/routes/global_search_screen.dart` | exact (near-verbatim, filtered) |
| `app/lib/routes/route_planner_screen.dart` (MODIFY: search button + tabbed sheet) | component/route | event-driven | itself (existing controls Column) + `trail_detail_map_screen.dart` (Stack + conditional sheet) | exact |

## Pattern Assignments

### `app/lib/provider/route_anchor_provider.dart` (store, CRUD) — add `deleteAnchor`/`reorderAnchors`

**Analog:** same file's existing mutators (`appendAnchor`, `dragAnchor`, `insertAnchorOnSegment`)

**Imports** (lines 1-8, already present, no changes needed):
```dart
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart' show UniqueKey;
import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/util/polyline_util.dart';
import 'package:wanderer/util/route_segment_util.dart';
```

**Mutator pattern to follow** (lines 217-260, `_pushUndo` + `appendAnchor`):
```dart
void _pushUndo() {
  state = state.copyWith(
    undoStack: [
      ...state.undoStack,
      RouteAnchorsSnapshot(anchors: state.anchors, segments: state.segments),
    ],
    redoStack: const [],
  );
}

void appendAnchor(Geographic point) {
  _pushUndo();
  final newAnchor = RouteAnchor(id: UniqueKey().toString(), lat: point.lat, lon: point.lon);
  final previousLast = state.anchors.isNotEmpty ? state.anchors.last : null;
  state = state.copyWith(anchors: [...state.anchors, newAnchor]);
  if (previousLast != null) {
    final newSegment = RouteSegment(
      beforeAnchorId: previousLast.id,
      afterAnchorId: newAnchor.id,
      polyline: [previousLast.point, newAnchor.point],
      state: SegmentState.straight,
    );
    state = state.copyWith(segments: [...state.segments, newSegment]);
    if (state.autoRoutingEnabled) {
      _resolveSegment(previousLast.id, newAnchor.id, previousLast, newAnchor).ignore();
    }
  }
}
```
Every mutator follows: `_pushUndo()` → build new `anchors`/`segments` lists via `copyWith` → conditionally call `_resolveSegment(...).ignore()` when `state.autoRoutingEnabled`, else `_applySegment(...)` with `SegmentState.straight` (see `dragAnchor`, lines 265-300, for the else-branch pattern). `deleteAnchor`/`reorderAnchors` must follow this exact shape — RESEARCH.md's Pattern 2/Pattern 3 give concrete proposed bodies modeled directly on this.

**Data-flow gotcha (carried from Phase 19, still applies):** segment identity is `(beforeAnchorId, afterAnchorId)` pairs, not array index — never iterate `state.segments` by index to reconstruct path order (see `plannedGpxProvider` below, which must walk the anchor-id chain).

---

### `app/lib/provider/planned_gpx_provider.dart` (NEW; derived provider, transform)

**Analog:** `app/lib/provider/trail/trail_polyline_provider.dart` (function-style `@riverpod` derived provider convention) + `route_anchor_provider.dart` (family key convention, `travelProfile` string param)

**Core pattern** (from RESEARCH.md Pattern 4, ready to copy near-verbatim):
```dart
@riverpod
Gpx plannedGpx(Ref ref, String travelProfile) {
  final state = ref.watch(routeAnchorsProvider(travelProfile));
  if (state.anchors.isEmpty) return Gpx();

  final segByBefore = {for (final s in state.segments) s.beforeAnchorId: s};
  final points = <Geographic>[state.anchors.first.point];
  var currentId = state.anchors.first.id;
  while (segByBefore.containsKey(currentId)) {
    final seg = segByBefore[currentId]!;
    points.addAll(seg.polyline.skip(1)); // skip(1): avoid duplicating shared boundary point
    currentId = seg.afterAnchorId;
  }

  return Gpx(
    trks: [Trk(trksegs: [Trkseg(trkpts: [for (final p in points) Wpt(lat: p.lat, lon: p.lon)])])],
  );
}
```
Must walk the anchor-id chain (`segByBefore[currentId]`), never `state.segments[i]` by array index (Pitfall 3).

---

### `app/lib/util/gpx_util.dart` (MODIFY: add Gpx-from-points helper)

**Analog:** itself — `buildNavShape()` (lines ~45-69) and `GpxMappingUtils.getTotals()` (lines ~106-145) already establish the file's helper-function conventions (extension methods on `Gpx`, plain top-level functions for construction helpers). No existing "build Gpx from raw points" helper — this is genuinely new logic (Claude's Discretion per CONTEXT.md); place the new helper alongside `buildNavShape` following the same top-level-function style.

---

### `app/lib/components/trail/elevation_profile.dart` (MODIFY in place: `trail` → `Trail?`)

**Analog:** itself (in-place adaptation, not a copy-from-elsewhere case)

**Imports** (lines 1-13, unchanged):
```dart
import 'dart:math';

import 'package:duration/duration.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:gpx/gpx.dart';
import 'package:latlong2/latlong.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/util/format_util.dart';
import 'package:wanderer/util/gpx_util.dart';
```

**Required diff (from RESEARCH.md Pattern 6, verified against actual source):**
1. `final Trail trail;` (line 17) → `final Trail? trail;`; constructor `required this.trail` → `this.trail`.
2. Stats header (lines ~176-187): currently reads `widget.trail.elevationGain`/`elevationLoss` **without** a null-safe accessor — this **will** crash once `trail` is nullable. Replace with a value sourced from `widget.gpx.getTotals()` (`GpxStats.totalElevationGain`/`totalElevationloss` — note existing lowercase-`l` typo, must match exactly) when `widget.trail == null`.
3. `_buildChart`'s waypoint-overlay (line ~230) already uses `widget.trail.expand?.waypointsViaTrail ?? []` — becomes safe automatically once `trail` is nullable (`widget.trail?.expand?.waypointsViaTrail ?? []`); re-verify at implementation time rather than assume.
4. `_EmptyState` (lines ~670-698) is already trail-independent — reuse verbatim; only override its caption text for D-13's copy.

---

### `app/lib/components/route_planner/route_anchor_sheet.dart` (NEW component)

**Analog:** `app/lib/components/trail/waypoint_sheet.dart` (sheet chrome) + `app/lib/routes/trail_detail_map_screen.dart` (Stack composition, lines ~76-209)

**Imports pattern** (from `waypoint_sheet.dart` lines 1-9, adapt names):
```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
```
`WaypointSheet` takes a `DraggableScrollableController controller` and `VoidCallback onClose` (lines 11-14) — the new sheet follows the controller-injection convention but must NOT include an `onClose`/full-dismiss callback (D-03: peek is the floor, never zero).

**Sheet chrome to copy verbatim** (`waypoint_sheet.dart` lines 43-56):
```dart
Container(
  decoration: BoxDecoration(
    color: theme.canvasColor,
    borderRadius: const BorderRadius.vertical(top: Radius.circular(16)),
    boxShadow: [
      BoxShadow(
        color: Colors.black.withValues(alpha: 0.15),
        blurRadius: 10,
        offset: const Offset(0, -2),
      ),
    ],
  ),
  ...
)
```

**Composition pattern (Stack + conditional trailing sheet child)** proven in `trail_detail_map_screen.dart:76-209` (`TrailMap` as `Positioned.fill`, `WaypointSheet` as the last unconstrained `Stack` child, conditionally rendered). Apply the same shape in `route_planner_screen.dart`'s `Scaffold.body`, conditioned on `state.anchors.isNotEmpty` (D-03).

**Core tabbed-sheet + drag pattern** — see RESEARCH.md Pattern 1 in full (this is the phase's hardest problem: do NOT share the `DraggableScrollableSheet.builder`'s `scrollController` across both `TabBarView` children — attach it only to the Route Anchors tab; drive expand/collapse via a separate `DraggableScrollableController` + `GestureDetector.onVerticalDragUpdate` on the handle-bar row, outside the `TabBarView`).

---

### `app/lib/components/route_planner/route_anchor_list_tab.dart` (NEW; WAYP-04/05)

**Analog:** `app/lib/routes/settings_categories_screen.dart` (lines 228-327) — read in full, exact pattern below.

**Imports pattern** (lines 1-3, 20 of `settings_categories_screen.dart`, adapt names):
```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
```

**Core reorder pattern to copy near-verbatim** (`settings_categories_screen.dart` lines 235-327):
```dart
Widget _buildList(...) {
  final byId = {for (final c in sorted) c.id: c};
  return ReorderableListView.builder(
    itemCount: _orderedIds.length,
    itemBuilder: (context, index) {
      final item = byId[_orderedIds[index]];
      if (item == null) return const SizedBox.shrink();
      return _buildRow(item, index, ...);
    },
    onReorder: (oldIndex, newIndex) => _onReorder(oldIndex, newIndex, ...),
    onReorderStart: (_) => setState(() => _reordering = true),
  );
}

Future<void> _onReorder(int oldIndex, int newIndex, ...) async {
  if (newIndex > oldIndex) newIndex -= 1; // canonical index-shift adjustment
  final snapshot = List<String>.from(_orderedIds);
  final reordered = List<String>.from(_orderedIds);
  final moved = reordered.removeAt(oldIndex);
  reordered.insert(newIndex, moved);
  setState(() => _orderedIds = reordered);
  try {
    await ref.read(routeAnchorsProvider(travelProfile).notifier).reorderAnchors(_orderedIds);
    if (!mounted) return;
    setState(() => _reordering = false);
  } catch (_) {
    if (!mounted) return;
    setState(() {
      _orderedIds = snapshot;
      _reordering = false;
    });
    // surface error toast, per D-06 no confirmation dialog needed for delete,
    // but reorder failure still needs a revert+toast per this analog
  }
}
```
State field declarations to mirror: `List<String> _orderedIds = const [];` and `bool _reordering = false;` (lines 44, 58) — `build()` must not reseed `_orderedIds` from provider state while `_reordering` is true (guards mid-drag race, see comment block lines 27-58).

**Delete pattern (D-05/D-06):** trailing icon button per row (not swipe), calling `ref.read(routeAnchorsProvider(travelProfile).notifier).deleteAnchor(id)` directly — no confirmation, no undo-snackbar (Undo/Redo app-bar buttons from Phase 19 are the safety net).

---

### `app/lib/components/route_planner/elevation_tab.dart` (NEW; PLANUI-02)

**Analog:** `app/lib/provider/search/global_search_provider.dart` (debounce idiom) + `app/lib/components/trail/elevation_profile.dart` (render, adapted per above)

**Debounce pattern to copy** (`global_search_provider.dart` lines 109-117 idiom):
```dart
Timer? _debounce;

void _onRouteChanged() {
  _debounce?.cancel();
  _debounce = Timer(const Duration(milliseconds: 500), _fetchHeights);
}
```

**Tab-visibility gating (D-11, Pitfall 2)** — must NOT rely on widget build/mount lifecycle (TabBarView keeps both tabs built simultaneously). Listen to `DefaultTabController.of(context)` directly:
```dart
late final TabController _tabController;

@override
void initState() {
  super.initState();
  _tabController = DefaultTabController.of(context);
  _tabController.addListener(_onTabChanged);
}

void _onTabChanged() {
  if (_tabController.index == 1 && !_tabController.indexIsChanging) {
    _fetchHeights();
  }
}
```

**Height fetch call** (verified against `web/src/routes/api/v1/valhalla/height/+server.ts`, pure pass-through proxy):
```dart
final response = await api.post('/valhalla/height', data: {
  'shape': buildNavShape(gpxPoints), // reuse existing helper from gpx_util.dart
});
final heights = (response.data['height'] as List).cast<num>();
```
Merge `ele` into a **local copy** of `plannedGpxProvider`'s `Gpx` (never write back into the derived provider — D-10 requires it stay pre-elevation).

**Empty state (D-13):** reuse `ElevationProfile`'s existing `_EmptyState` widget/pattern when `< 2` anchors — do not build a new one or call `/valhalla/height` with insufficient points.

---

### `app/lib/routes/location_search_screen.dart` (NEW; PLANUI-03)

**Analog:** `app/lib/routes/global_search_screen.dart` — read in full, mirror almost verbatim, dropping category-chip row and non-location branches.

**Imports pattern** (lines 1-18):
```dart
import 'dart:math' as math;

import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/search/global_search_provider.dart';
import 'package:wanderer/util/format_util.dart';
import 'package:wanderer/util/polyline_util.dart';
```
Drop `category.dart`/`subcategory.dart` model imports and `category_provider.dart`/`subcategory_provider.dart` provider imports (not needed once trails/lists/actors branches are dropped).

**Result model** (`app/lib/models/global_search_models.dart:169-178`, reuse as-is):
```dart
@freezed
abstract class LocationSearchResult with _$LocationSearchResult {
  const factory LocationSearchResult({
    required String name,
    required String description,
    required double lat,
    required double lon,
    required String category,
    required String type,
  }) = _LocationSearchResult;
}
```

**Selection handler — the one required behavior change** (D-14/D-15). Current (`global_search_screen.dart` lines 326-330):
```dart
onTap: () => context.go(
  '/map',
  extra: {'lat': location.lat, 'lon': location.lon, 'zoom': 13.0},
),
```
New: `onTap: () => context.pop(result)` — pop back to `RoutePlannerScreen` (already-open route, not `context.go`) with the `LocationSearchResult`; the planner screen then calls its own `MapController.animateCamera(center: result, zoom: 13)` directly (D-15: zoom 13, matching the existing `/map` handoff behavior).

---

### `app/lib/routes/route_planner_screen.dart` (MODIFY: search button + tabbed sheet host)

**Analog:** itself — existing controls `Column` and `body:` structure (lines 169, 250-262) + `trail_detail_map_screen.dart` Stack composition (see `route_anchor_sheet.dart` section above).

**Current controls Column (exact insertion point, D-04):**
```dart
Positioned(
  top: 128,
  right: 0,
  child: Column(
    crossAxisAlignment: CrossAxisAlignment.end,
    children: [_buildAutoRoutingToggle(state.autoRoutingEnabled)],
    // D-04: Search button goes ABOVE this — prepend to the list, not append.
  ),
),
```
Add `_buildSearchButton()` (mirroring `_buildAutoRoutingToggle`'s existing button styling) as the first child.

**`body:` restructure:** currently `_buildMap(context, state, styleJson)` directly (line 169) — must become a `Stack` with the map as `Positioned.fill` and the new `RouteAnchorSheet` as a conditional (`if (state.anchors.isNotEmpty)`) trailing `Stack` child, matching `trail_detail_map_screen.dart`'s proven composition.

**On location-search result:** `await context.push<LocationSearchResult>(...)`, then on non-null result call `_mapController.animateCamera(center: ..., zoom: 13)` directly (D-14/D-15) — no `context.go('/map')`.

---

## Shared Patterns

### Undo/redo mutator shape
**Source:** `app/lib/provider/route_anchor_provider.dart:217-260` (`_pushUndo` + `appendAnchor`)
**Apply to:** `deleteAnchor`, `reorderAnchors` (new mutators) — every mutation must call `_pushUndo()` first, then rebuild `anchors`/`segments` via `copyWith`, then conditionally re-resolve via `_resolveSegment(...).ignore()` (auto-routing on) or `_applySegment(...)` (auto-routing off).

### Optimistic reorder + revert-on-error
**Source:** `app/lib/routes/settings_categories_screen.dart:228-327`
**Apply to:** `route_anchor_list_tab.dart`'s `ReorderableListView.builder` — local `_orderedIds` working copy, `_reordering` guard, canonical `if (newIndex > oldIndex) newIndex -= 1` index-shift, snapshot-and-revert on failure.

### Debounced search-as-you-type
**Source:** `app/lib/provider/search/global_search_provider.dart:109-117`
**Apply to:** `location_search_screen.dart` (reuse as-is) and `elevation_tab.dart`'s height-refetch debounce (same idiom, different trigger).

### DraggableScrollableSheet chrome
**Source:** `app/lib/components/trail/waypoint_sheet.dart:43-56`
**Apply to:** `route_anchor_sheet.dart` — rounded-top container, `canvasColor` background, drop shadow. Critical divergence: do NOT share the builder's `scrollController` across both `TabBarView` children (see Pattern 1 detail above under `route_anchor_sheet.dart`).

### Stack + conditional trailing sheet child
**Source:** `app/lib/routes/trail_detail_map_screen.dart:76-209`
**Apply to:** `route_planner_screen.dart`'s `Scaffold.body` restructure (map `Positioned.fill` + conditional sheet).

## No Analog Found

None — every file in this phase has at least a role-match analog. The two genuinely novel logic surfaces (`deleteAnchor`/`reorderAnchors` segment-recompute algorithms, and the Gpx-from-points helper) have no verbatim precedent to copy but RESEARCH.md provides concrete proposed implementations modeled on the closest existing inverse/sibling operations (`insertAnchorOnSegment`, `buildNavShape`) — flagged as MEDIUM confidence in RESEARCH.md's Assumptions Log (A3) and should be treated as the phase's primary design-review checkpoint, not as risk-free copy-paste.

## Metadata

**Analog search scope:** `app/lib/provider/`, `app/lib/components/trail/`, `app/lib/routes/`, `app/lib/util/` (all directly informed by RESEARCH.md's exhaustive in-repo source reads, cross-verified against actual file contents in this pass)
**Files scanned:** 8 target files + 5 analog files read directly (`route_anchor_provider.dart`, `waypoint_sheet.dart`, `global_search_screen.dart`, `settings_categories_screen.dart`, `elevation_profile.dart`)
**Pattern extraction date:** 2026-07-16
