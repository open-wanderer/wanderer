# Phase 20: Route Planner Views — Waypoint List, Elevation & Location Search - Research

**Researched:** 2026-07-16
**Domain:** Flutter mobile UI (Riverpod state, `DraggableScrollableSheet` + `TabBarView` composition, `fl_chart` elevation rendering, GPX synthesis, Valhalla elevation API integration)
**Confidence:** HIGH (all claims verified against in-repo source or official docs; one architectural composition problem independently confirmed via a Flutter framework GitHub issue)

## Summary

This phase is pure Flutter/Dart work inside `app/lib/`. No SvelteKit or Go backend changes are required — both consumed endpoints (`/api/v1/valhalla/height`, `/api/v1/geocoding/search`) already exist as pass-through proxies and were re-verified in this session. The phase's hardest technical problem is **not** any of the individual widgets (delete/reorder list, elevation chart, search screen) — each of those has a proven in-repo analog to copy almost verbatim. The hardest problem is the **composition of a `TabBarView` inside a `DraggableScrollableSheet`**, which has a well-documented Flutter framework pitfall: passing the sheet's single `scrollController` to more than one simultaneously-built `TabBarView` child throws `"ScrollController attached to multiple scroll views"` (confirmed via [flutter/flutter#55388](https://github.com/flutter/flutter/issues/55388)). With only 2 tabs, `TabBarView` keeps both pages built/mounted concurrently (it preloads adjacent pages), so this is not a theoretical edge case — it will reproduce immediately if the sheet's `scrollController` is wired into both tabs' content.

The verified fix: attach the sheet's `scrollController` to **only** the Route Anchors tab's `ReorderableListView` (the one genuinely-scrollable tab), give the Elevation tab no shared controller at all (its content is short, fixed-height, non-scrolling), and drive expand/collapse **explicitly** via a `DraggableScrollableController` (already an established pattern in this codebase via `WaypointSheet`) wired to a `GestureDetector.onVerticalDragUpdate` on the sheet's handle-bar row, which sits in the fixed header outside the `TabBarView`. This mirrors Flutter's own official example (`examples/api/lib/widgets/draggable_scrollable_sheet/draggable_scrollable_sheet.0.dart`), which uses exactly this decoupled "manual grabber" strategy for non-touch/non-scrolling drag surfaces.

Every other sub-problem in this phase (delete/reorder UX, GPX synthesis, elevation adaptation, location search) is a direct, low-risk adaptation of an existing, working in-repo pattern — `settings_categories_screen.dart`'s `ReorderableListView`, `global_search_screen.dart`'s search shell, and `elevation_profile.dart`'s chart. The `route_anchor_provider.dart`'s two new mutators (`deleteAnchor`, `reorderAnchors`) need a concrete segment-recompute algorithm that isn't specified anywhere yet (20-CONTEXT.md defers this to "recompute affected segments" without an algorithm) — this research proposes one below, modeled directly on the existing `insertAnchorOnSegment`/`appendAnchor` mutators.

**Primary recommendation:** Reuse every named component per 20-CONTEXT.md's Existing Code Insights verbatim; solve the sheet/tab composition with the handle-bar-drag + single-tab-scrollController pattern described above; implement `deleteAnchor`/`reorderAnchors` using the adjacency-diff algorithm in Architecture Patterns; call Valhalla's `/height` endpoint **without** `range: true` (simpler `{shape, height}` response, matches existing assumption, now verified against Valhalla's own docs).

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| WAYP-04 | User can delete a route anchor from the route anchor list tab | Row layout + `deleteAnchor(id)` algorithm below; D-05/D-06/D-07 constraints from CONTEXT.md |
| WAYP-05 | User can reorder route anchors via the route anchor list tab | `ReorderableListView.builder` pattern (verified from `settings_categories_screen.dart:244-327`) + `reorderAnchors(newOrder)` algorithm below |
| PLANUI-01 | Route anchor list + elevation profile as two tabs of one persistent bottom sheet | Sheet/Tab composition pattern below (verified fix for the `TabBarView`/`DraggableScrollableSheet` scrollController conflict) |
| PLANUI-02 | Elevation profile built from incrementally-synthesized `Gpx`, fetched only while tab visible | `plannedGpxProvider` derived-provider pattern + Valhalla `/height` verified request/response shape + `TabController`-gated debounced fetch |
| PLANUI-03 | Location-search screen (locations only), pans/zooms planner map to result | `GlobalSearchScreen`/`globalSearchProvider` adaptation, verified exact current source |

</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Route anchor list display/delete/reorder | Browser/Client (Flutter widget tree) | API/Backend (none — in-memory only) | All state lives in `RouteAnchorsState` (Riverpod, in-memory, undo/redo); no persistence until Phase 21 handoff |
| Elevation profile chart + data | Browser/Client (Flutter widget + derived provider) | API/Backend (`/api/v1/valhalla/height` proxy) | Chart rendering and GPX synthesis are 100% client-side; only the `ele` values require a network round-trip through the existing SvelteKit proxy to Valhalla |
| Location search | Browser/Client (Flutter search screen) | API/Backend (`/api/v1/geocoding/search` proxy) | Search UI/debounce is client-side; result data comes from the existing Nominatim proxy, already used by `GlobalSearchScreen` |
| Map camera pan/zoom on result select | Browser/Client (MapLibre `MapController`) | — | Camera animation is a native map SDK call, no server involvement |

No capability in this phase touches the Go/PocketBase backend or SvelteKit web frontend — confirmed consistent with 20-CONTEXT.md and `.planning/REQUIREMENTS.md`'s Out of Scope table ("Any Go backend or SvelteKit changes").

## Package Legitimacy Audit

Not applicable — this phase adds **no new external packages**. All required packages (`gpx: ^2.3.0`, `fl_chart: ^1.2.0`, `font_awesome_flutter: ^11.0.0`, `flutter_riverpod: ^3.3.1`, `go_router: ^17.2.1`) are already declared in `app/pubspec.yaml` and already used by the exact files this phase modifies/reuses. `[VERIFIED: app/pubspec.yaml, read directly]`.

## Standard Stack

### Core (all pre-existing, no new installs)

| Package | Version (verified from `app/pubspec.yaml`) | Purpose in this phase | Why Standard |
|---------|------|---------|--------------|
| `flutter_riverpod` | ^3.3.1 | `plannedGpxProvider`, `RouteAnchors.deleteAnchor`/`reorderAnchors` | Already the app's sole state-management library |
| `riverpod_annotation` / `riverpod_generator` | ^4.0.2 / ^4.0.3 | `@riverpod` codegen for the new derived provider | Matches `route_anchor_provider.dart`'s existing pattern |
| `gpx` | ^2.3.0 | `Gpx`/`Trk`/`Trkseg`/`Wpt` construction for the synthesized route GPX | Already the app's GPX model library (`gpx_util.dart`) |
| `fl_chart` | ^1.2.0 | Elevation chart, via the existing `ElevationProfile` widget | Already wired into `elevation_profile.dart` |
| `font_awesome_flutter` | ^11.0.0 | All new icons (`listOl`, `mountain`, `trash`, `magnifyingGlass`) | UI-SPEC mandates FA icons exclusively for new elements |
| `go_router` | ^17.2.1 | `context.push`/`context.pop(result)` for the location-search screen | Already the app's router |
| `dio` | ^5.9.2 | `/valhalla/height` POST call via existing `apiProvider` | Already the app's HTTP client |

**Installation:** none required — every package is already a pubspec dependency.

**Version verification:** Confirmed directly from `app/pubspec.yaml` (read 2026-07-16), not from a registry lookup — these are pinned project dependencies, not new installs, so `npm view`/`pip index`-equivalent registry checks are not applicable here.

## Architecture Patterns

### System Architecture Diagram

```text
User taps map (empty)                      User opens Elevation tab
        │                                            │
        ▼                                            ▼
RouteAnchorsState.appendAnchor()        TabController.index == 1
        │                                            │
        ▼                                            ▼
state.anchors / state.segments ────► plannedGpxProvider (derived)
        │  (watched)                     │  recomputes Gpx skeleton
        │                                │  (points, no `ele`) on every
        │                                │  anchors/segments change
        ▼                                ▼
RouteAnchorLayer / RouteSegmentLayer   _ElevationTab widget state
  (map markers/lines, unchanged)         │  debounces 500ms, then
                                         │  POST /api/v1/valhalla/height
                                         │  (only while tab visible)
                                         ▼
                                   merges `ele` into Gpx copy
                                         │
                                         ▼
                                   ElevationProfile(trail: null, gpx: ...)
                                         │
                                         ▼
                                   fl_chart LineChart + stats header


User taps trailing delete icon           User long-presses + drags a row
        │                                            │
        ▼                                            ▼
RouteAnchors.deleteAnchor(id)            RouteAnchors.reorderAnchors(newOrder)
        │ _pushUndo() → remove anchor,   │ _pushUndo() → reassign anchors,
        │ splice ≤2 touching segments    │ diff old vs new adjacency pairs,
        │ into ≤1 new segment            │ keep/reuse unchanged segments,
        ▼                                │ rebuild only newly-adjacent ones
state.segments (new identity)            ▼
        │                          state.anchors / state.segments (new identity)
        ▼                                │
ref.listen in route_planner_screen ──────┘──► RouteSegmentLayer.update() (native map)


User taps magnifying-glass control
        │
        ▼
context.push(LocationSearchScreen)  ──► GET /api/v1/geocoding/search (debounced 500ms)
        │
        │ user taps a result row
        ▼
context.pop(LocationSearchResult)
        │
        ▼
RoutePlannerScreen awaits pushed route
        │
        ▼
_mapController.animateCamera(center: result, zoom: 13)
```

### Pattern 1: Tabbed `DraggableScrollableSheet` composition (the phase's hardest problem)

**What:** A single `DraggableScrollableSheet` hosting a `DefaultTabController(length: 2)` wrapping `TabBar` + `TabBarView`, docked at a non-zero peek height, never fully dismissible.

**The verified pitfall:** `TabBarView` does not lazily unmount pages the way an `IndexedStack` with `Offstage` would suggest — with only 2 children it keeps **both** built simultaneously so it can animate the horizontal swipe. If the `ScrollController` supplied by `DraggableScrollableSheet.builder` is attached to a `Scrollable` (e.g., a `ListView`/`ReorderableListView`) inside **each** tab, Flutter throws `"ScrollController attached to multiple scroll views"` at runtime. This exact failure mode — sharing the sheet's `scrollController` across a `TabBarView`'s children — is a filed, unresolved Flutter framework issue: [flutter/flutter#55388](https://github.com/flutter/flutter/issues/55388) `[CITED: github.com/flutter/flutter/issues/55388]`. A related, still-open issue for the `NestedScrollView` variant of the same underlying conflict is [flutter/flutter#118713](https://github.com/flutter/flutter/issues/118713) and [flutter/flutter#64157](https://github.com/flutter/flutter/issues/64157) `[CITED]` — confirming `NestedScrollView` is not a clean escape hatch either (it forbids assigning your own `ScrollController` to descendant scrollables at all, expecting `PrimaryScrollController` wiring instead, which conflicts with `DraggableScrollableSheet`'s own controller-passing contract).

**Verified fix (mirrors Flutter's own official example):** Flutter's official cookbook example for `DraggableScrollableSheet` (`examples/api/lib/widgets/draggable_scrollable_sheet/draggable_scrollable_sheet.0.dart`, referenced from the [DraggableScrollableSheet class docs](https://api.flutter.dev/flutter/widgets/DraggableScrollableSheet-class.html)) solves an analogous problem (desktop mouse-drag support where the descendant isn't naturally scroll-draggable) by adding a **`Grabber`** widget — a plain `GestureDetector.onVerticalDragUpdate` that computes a new fractional size and applies it directly, entirely decoupled from the `scrollController` plumbing. `[CITED: github.com/flutter/flutter examples/api]`.

Applied to this phase:
1. Do **not** attach the builder's `scrollController` to more than one `TabBarView` child.
2. Attach it only to the Route Anchors tab's `ReorderableListView.builder(controller: scrollController, ...)` — the one tab with content that can outgrow the sheet's expanded height.
3. Give the Elevation tab **no** shared controller — its content (`ElevationProfile`, stats header + fixed 150px chart) is short and doesn't need to scroll; if overflow protection is wanted, wrap it in its own unshared `SingleChildScrollView()` (default, unattached controller).
4. Implement expand/collapse via a `DraggableScrollableController` (already an established pattern in this codebase — `WaypointSheet` accepts and uses one, see `app/lib/components/trail/waypoint_sheet.dart:14,36-41`), wired to `onVerticalDragUpdate` on the handle-bar `Row` (which lives in the `Column`, above `Expanded(TabBarView(...))`, i.e., **outside** the tab content entirely — always present regardless of active tab).
5. `DraggableScrollableController` exposes exactly the methods needed: `size` (current fractional size, getter), `jumpTo(double size)` (immediate), `animateTo(double size, {duration, curve})` (animated), `isAttached` (bool) — all confirmed from the [official API docs](https://api.flutter.dev/flutter/widgets/DraggableScrollableController-class.html) `[CITED]`.

```dart
// Source: pattern derived from WaypointSheet (app/lib/components/trail/waypoint_sheet.dart)
// + Flutter's official draggable_scrollable_sheet.0.dart example
// + github.com/flutter/flutter#55388 (verified scrollController conflict)
final _sheetController = DraggableScrollableController();

DraggableScrollableSheet(
  controller: _sheetController,
  initialChildSize: 0.14,
  minChildSize: 0.14,   // no full-dismiss (D-03) — never 0.0
  maxChildSize: 0.6,
  snap: true,
  snapSizes: const [0.14, 0.6],
  builder: (context, scrollController) {
    return DefaultTabController(
      length: 2,
      child: Container(
        decoration: /* same rounded-16 canvasColor chrome as WaypointSheet */,
        child: Column(
          children: [
            GestureDetector(
              onVerticalDragUpdate: (details) {
                final target = (_sheetController.size -
                        details.delta.dy / MediaQuery.sizeOf(context).height)
                    .clamp(0.14, 0.6);
                _sheetController.jumpTo(target);
              },
              child: /* handle bar Row — no close button per D-03 */,
            ),
            const TabBar(tabs: [
              Tab(icon: FaIcon(FontAwesomeIcons.listOl), text: 'Route Anchors'),
              Tab(icon: FaIcon(FontAwesomeIcons.mountain), text: 'Elevation'),
            ]),
            Expanded(
              child: TabBarView(children: [
                _RouteAnchorListTab(scrollController: scrollController),
                const _ElevationTab(), // NOT given scrollController
              ]),
            ),
          ],
        ),
      ),
    );
  },
)
```

**Gating the height fetch on tab visibility (D-11):** `DefaultTabController`'s `TabController` is available via `DefaultTabController.of(context)`. The Elevation tab widget must listen to it directly (`TabController.addListener` in `initState`, checking `controller.index == 1` and `!controller.indexIsChanging`) rather than relying on `TabBarView` merely building the widget — per the UI-SPEC's own note, `TabBarView` does not lazily unmount, so "is this widget currently built" is not equivalent to "is this tab currently visible." This is consistent with, and reinforces, D-11's explicit requirement.

### Pattern 2: `deleteAnchor(id)` — proposed segment-recompute algorithm

**What:** 20-CONTEXT.md's D-08 specifies the mutator must exist and follow "`_pushUndo()` → mutate `anchors`/`segments` → recompute affected segments" but does not specify the recompute algorithm. No prior mutator deletes an anchor; the closest existing precedent is `insertAnchorOnSegment` (which does the geometric *inverse*: split one segment into two around a new anchor). This research proposes the following, deleting an anchor collapses ≤2 segments into ≤1 new segment — mirroring `insertAnchorOnSegment`'s pattern in reverse:

```dart
// Source: proposed by this research — no existing method to copy verbatim;
// modeled directly on insertAnchorOnSegment's inverse operation and
// appendAnchor's "create straight, then auto-resolve if enabled" pattern.
void deleteAnchor(String anchorId) {
  _pushUndo();

  final anchors = state.anchors.where((a) => a.id != anchorId).toList();

  final before = state.segments
      .where((s) => s.afterAnchorId == anchorId)
      .firstOrNull; // null if the deleted anchor was first
  final after = state.segments
      .where((s) => s.beforeAnchorId == anchorId)
      .firstOrNull; // null if the deleted anchor was last

  final segments = [
    for (final s in state.segments)
      if (s.beforeAnchorId != anchorId && s.afterAnchorId != anchorId) s,
    if (before != null && after != null)
      RouteSegment(
        beforeAnchorId: before.beforeAnchorId,
        afterAnchorId: after.afterAnchorId,
        polyline: [
          anchors.firstWhere((a) => a.id == before.beforeAnchorId).point,
          anchors.firstWhere((a) => a.id == after.afterAnchorId).point,
        ],
        state: SegmentState.straight,
      ),
  ];

  state = state.copyWith(anchors: anchors, segments: segments);

  if (before != null && after != null && state.autoRoutingEnabled) {
    final a = anchors.firstWhere((x) => x.id == before.beforeAnchorId);
    final b = anchors.firstWhere((x) => x.id == after.afterAnchorId);
    _resolveSegment(before.beforeAnchorId, after.afterAnchorId, a, b).ignore();
  }
}
```

**When the deleted anchor was first, last, or the only anchor:** at most 1 segment is removed and no new segment is created (`before`/`after` guard above naturally handles all three cases — first anchor has no `before` segment, last has no `after` segment, sole anchor has neither).

### Pattern 3: `reorderAnchors(newOrder)` — proposed adjacency-diff algorithm

**What:** Reordering changes which anchors are adjacent. A naive "rebuild every segment from scratch" would discard `SegmentState.routed` polylines (and refire Valhalla) for pairs that are still adjacent after the reorder — wasteful and visually jarring (a resolved segment would flash to a straight line and back). The proposed algorithm diffs old vs. new adjacency and only rebuilds what actually changed:

```dart
// Source: proposed by this research — no existing method to copy verbatim.
void reorderAnchors(List<String> newOrder) {
  _pushUndo();

  final anchorsById = {for (final a in state.anchors) a.id: a};
  final reordered = newOrder.map((id) => anchorsById[id]!).toList();

  // Old adjacency, keyed by segmentKey, so an unchanged pair's existing
  // segment (with its resolved polyline/state) can be reused as-is.
  final oldByKey = {
    for (final s in state.segments)
      segmentKey(s.beforeAnchorId, s.afterAnchorId): s,
  };

  final segments = <RouteSegment>[];
  final toResolve = <(String, String, RouteAnchor, RouteAnchor)>[];

  for (var i = 0; i < reordered.length - 1; i++) {
    final a = reordered[i];
    final b = reordered[i + 1];
    final key = segmentKey(a.id, b.id);
    final existing = oldByKey[key];
    if (existing != null) {
      segments.add(existing); // unchanged adjacency — reuse verbatim
    } else {
      segments.add(RouteSegment(
        beforeAnchorId: a.id,
        afterAnchorId: b.id,
        polyline: [a.point, b.point],
        state: SegmentState.straight,
      ));
      if (state.autoRoutingEnabled) toResolve.add((a.id, b.id, a, b));
    }
  }

  state = state.copyWith(anchors: reordered, segments: segments);

  for (final (beforeId, afterId, a, b) in toResolve) {
    _resolveSegment(beforeId, afterId, a, b).ignore();
  }
}
```

**Why this matters for planning:** without an explicit algorithm, a plan task like "implement `reorderAnchors`" is underspecified enough that an implementer could reasonably (and wrongly) rebuild every segment from scratch on every reorder, causing every already-routed segment to flash straight and re-fetch from Valhalla on an unrelated drag elsewhere in the list. Flag this pattern for the plan so the corresponding task's must-haves/verification explicitly check "unaffected adjacent pairs keep their existing `SegmentState.routed` polyline across a reorder."

### Pattern 4: `plannedGpxProvider` — derived Gpx synthesis (pre-elevation)

**What:** A `@riverpod` function-style provider (matching the established derived-provider convention in `app/lib/provider/trail/trail_polyline_provider.dart:9-21`), keyed by `travelProfile` (same family key as `routeAnchorsProvider`), watching `routeAnchorsProvider(travelProfile)` and rebuilding a `Gpx` skeleton (points only, no `ele`) whenever anchors/segments change.

**Building the ordered point list:** must walk `state.segments` in anchor order (via `state.anchors`' sequence and each segment's own polyline), not just concatenate `state.segments` in array order — segment array order is not guaranteed to match anchor traversal order after `insertAnchorOnSegment`/future `reorderAnchors` mutations (Pitfall 3 from Phase 19's research, still applicable: segment identity is anchor-id-pair based, not index-based).

```dart
// Source: proposed by this research — no existing "Gpx from points" helper
// exists yet (confirmed absent from gpx_util.dart); Claude's Discretion
// per 20-CONTEXT.md for exact placement, this is the recommended shape.
@riverpod
Gpx plannedGpx(Ref ref, String travelProfile) {
  final state = ref.watch(routeAnchorsProvider(travelProfile));
  if (state.anchors.isEmpty) return Gpx();

  final segByBefore = {for (final s in state.segments) s.beforeAnchorId: s};
  final points = <Geographic>[state.anchors.first.point];
  var currentId = state.anchors.first.id;
  while (segByBefore.containsKey(currentId)) {
    final seg = segByBefore[currentId]!;
    points.addAll(seg.polyline.skip(1)); // skip(1): avoid duplicating the shared boundary point
    currentId = seg.afterAnchorId;
  }

  return Gpx(
    trks: [
      Trk(trksegs: [
        Trkseg(trkpts: [
          for (final p in points) Wpt(lat: p.lat, lon: p.lon),
        ]),
      ]),
    ],
  );
}
```

`Gpx`/`Trk`/`Trkseg`/`Wpt` constructors confirmed directly from the installed `gpx-2.3.0` package source (`~/.pub-cache/hosted/pub.dev/gpx-2.3.0/lib/src/model/{gpx,trk,trkseg,wpt}.dart`) `[VERIFIED: installed package source]` — `Trk({List<Trkseg>? trksegs, ...})`, `Trkseg({List<Wpt>? trkpts, ...})`, `Wpt({double? lat, double? lon, double? ele, ...})` are all optional-named-parameter constructors, so the shape above compiles as written.

### Pattern 5: Height fetch, debounced, gated on tab visibility

**What:** While the Elevation tab is the active `TabBarView` page, POST the synthesized `Gpx`'s points to `/api/v1/valhalla/height`, debounced ~500ms on further route edits (D-11), matching `GlobalSearchNotifier`'s existing debounce idiom (`app/lib/provider/search/global_search_provider.dart:109-117`, `Timer(const Duration(milliseconds: 500), _search)`).

**Verified Valhalla request/response shape** (official Valhalla docs, `valhalla-docs/elevation/api-reference.md`) `[CITED: github.com/valhalla/valhalla-docs]`:
- Request (no `range`, simplest form — sufficient here since distance-along-track is already computed client-side by `GpxMappingUtils.getTotals()`/`ElevationProfile`'s own haversine logic): `{"shape": [{"lat": ..., "lon": ...}, ...]}`
- Response (default, `range` omitted or `false`): `{"shape": [...], "height": [...]}` — a flat array of elevation values, one per input shape point, in order.
- The alternate `range: true` response shape is `{"shape": [...], "range_height": [[cumDist, ele], ...]}` — **not needed** for this phase (recommend the simpler default form).
- No documented hard cap on shape-point count was found for the elevation endpoint specifically (unlike `/route`'s well-known 500-point-ish practical limits). **Recommendation (not verified against an authoritative limit):** reuse the existing `buildNavShape()` helper (`app/lib/util/gpx_util.dart:45-69`, downsamples to ≤500 points, always preserving first/last) defensively before sending to `/height`, since it is already proven safe for the sibling `/route` call and no evidence contradicts reusing it here.

```dart
// Source: web/src/routes/api/v1/valhalla/height/+server.ts (read directly,
// confirmed pure pass-through proxy — POST body forwarded verbatim to
// Valhalla's own /height endpoint, no shape/field transformation server-side)
final response = await api.post('/valhalla/height', data: {
  'shape': buildNavShape(gpxPoints), // reuse existing helper, see above
});
final heights = (response.data['height'] as List).cast<num>();
```

**Merging elevation back into the Gpx:** build a **copy** of `plannedGpxProvider`'s `Gpx` with `ele` populated per-point (matching heights array by index — same order as the request shape), held in local widget state (or a separate non-family provider scoped to the Elevation tab widget's lifetime), not written back into `plannedGpxProvider` itself — D-10 is explicit that the derived provider stays "pre-elevation," and the elevation-tab-only widget owns the elevation-merged copy plus its own debounce `Timer`.

### Pattern 6: Adapting `ElevationProfile` for `trail: null` (D-12)

**What:** `app/lib/components/trail/elevation_profile.dart` currently requires `final Trail trail;` (non-nullable, line 17) and reads `widget.trail.elevationGain`/`widget.trail.elevationLoss` for the non-scrub stats header (lines 176-187), and `widget.trail.expand?.waypointsViaTrail` for the chart's vertical-line/icon overlay (line 230). Both must be guarded.

**Verified minimal diff:**
1. `final Trail trail;` → `final Trail? trail;` (constructor's `required this.trail` → `this.trail`).
2. Stats header (non-scrub branch, lines 176-187): replace `widget.trail.elevationGain`/`elevationLoss` with a value sourced from `widget.gpx.getTotals()` (already an extension method on `Gpx`, confirmed in `gpx_util.dart:106-145`, returns `GpxStats` with `totalElevationGain`/`totalElevationloss` — note existing lowercase-`l` typo `totalElevationloss`, must match exactly) when `widget.trail == null`.
3. `_buildChart`'s `waypoints` local (line 230): guard with `widget.trail?.expand?.waypointsViaTrail ?? []` — already effectively null-safe via the existing `?? []`, so this line requires **no change** once `trail` is nullable; the `.expand?.` chain already short-circuits cleanly to an empty list. Confirm this at implementation time (re-read after making `trail` nullable) rather than assuming — the surrounding duration/distance stats (`maxDur`, `maxDist`) are already computed from `_points` (parsed from `gpx`), not from `trail`, so they need no change at all.
4. `_EmptyState` (lines 670-698) is already trail-independent — reuse verbatim, only its caption text needs an optional override per D-13's copy ("Add at least 2 anchors to see the elevation profile.").

### Recommended Project Structure (new/modified files only)

```
app/lib/
├── components/
│   └── trail/
│       └── elevation_profile.dart        # MODIFY: trail → Trail?, D-12 guards
├── provider/
│   ├── route_anchor_provider.dart        # MODIFY: add deleteAnchor, reorderAnchors
│   └── planned_gpx_provider.dart         # NEW: plannedGpxProvider (Pattern 4)
├── routes/
│   ├── route_planner_screen.dart         # MODIFY: search button, tabbed sheet
│   └── location_search_screen.dart       # NEW: PLANUI-03, mirrors global_search_screen.dart
├── components/
│   └── route_planner/
│       ├── route_anchor_sheet.dart       # NEW: the tabbed DraggableScrollableSheet (Pattern 1)
│       ├── route_anchor_list_tab.dart    # NEW: WAYP-04/05, ReorderableListView.builder
│       └── elevation_tab.dart            # NEW: PLANUI-02, wraps adapted ElevationProfile
└── util/
    └── gpx_util.dart                     # MODIFY: add Gpx-from-points helper (used by Pattern 4)
```

Exact file/folder naming above (`components/route_planner/`) is a reasonable proposal only — no existing convention dictates a `route_planner/` subfolder; the planner should confirm placement against how `app/lib/components/map/` (route_anchor_layer.dart, route_segment_layer.dart) was organized in Phase 19 for consistency.

### Anti-Patterns to Avoid

- **Sharing `DraggableScrollableSheet`'s `scrollController` across both `TabBarView` children** — throws `"ScrollController attached to multiple scroll views"` at runtime (verified, see Pattern 1). Attach it to exactly one tab's scrollable, or none.
- **Gating the `/valhalla/height` fetch on `plannedGpxProvider` re-emitting alone** — since `TabBarView` keeps both tab widgets built simultaneously, "the Elevation tab widget exists in the tree" is not equivalent to "the user is looking at it." Gate on `TabController.index`/`indexIsChanging`, not on build/mount lifecycle.
- **Rebuilding every segment from scratch on `reorderAnchors`** — discards resolved `SegmentState.routed` polylines for pairs that remain adjacent, causing visible flicker and unnecessary Valhalla calls. Diff old vs. new adjacency (Pattern 3).
- **Writing `ele` values back into `plannedGpxProvider`'s own `Gpx`** — D-10 requires the derived provider to stay pre-elevation (it's also the Phase 21 handoff artifact for HANDOFF-01, which should carry a clean, GPS-only skeleton, not stale cached elevation from whichever moment the Elevation tab happened to last be open).
- **Reusing `GlobalSearchScreen` directly with a location filter** — already explicitly ruled out in `.planning/REQUIREMENTS.md`'s Out of Scope table ("it also returns trails/lists/accounts; a dedicated location-only search screen is required instead").

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Draggable bottom sheet chrome (rounded corners, shadow, canvasColor background) | A new custom sheet container | Copy `WaypointSheet`'s exact `BoxDecoration`/`boxShadow` values (`app/lib/components/trail/waypoint_sheet.dart:44-56`) | Already the app's one other `DraggableScrollableSheet`; visual consistency is UI-SPEC-mandated |
| Optimistic reorder + revert-on-error | A new reorder controller/state machine | Copy `settings_categories_screen.dart`'s `_orderedIds`/`_reordering` pattern (lines 244-327) verbatim, including the canonical `if (newIndex > oldIndex) newIndex -= 1` index-shift | Proven, already handles the exact Pitfall 1/2 class of bugs (index-shift, mid-drag reseed race) |
| Debounced search-as-you-type | A new `Timer`/debounce wrapper | Copy `GlobalSearchNotifier`'s `Timer(const Duration(milliseconds: 500), _search)` idiom (`global_search_provider.dart:109-117`) | Already proven in this exact app for this exact API family (geocoding search) |
| Elevation chart rendering, smoothing, gradient coloring | A new lightweight chart widget | Adapt `ElevationProfile` in place (D-12, Pattern 6) | ~500 lines of proven `fl_chart` configuration (smoothing window, gradient-by-gradient coloring, scrub tooltips) — a purpose-built rewrite would re-introduce every edge case already solved here |
| GPX distance/elevation totals | A new stats calculator | `GpxMappingUtils.getTotals()` extension (`gpx_util.dart:106-145`) | Already computes exactly the `GpxStats` shape `ElevationProfile`'s stats header needs |

**Key insight:** every widget/logic surface this phase touches has an exact in-repo analog from Phase 19 or the existing trail-detail/settings screens. The only genuinely novel logic is the two new `route_anchor_provider.dart` mutators (no existing delete/reorder precedent) and the `plannedGpxProvider` Gpx-from-points synthesis (no existing "build Gpx from raw points" helper) — both are specified concretely in Architecture Patterns above precisely because they are the phase's only truly new logic.

## Common Pitfalls

### Pitfall 1: `TabBarView` + `DraggableScrollableSheet` scrollController conflict
**What goes wrong:** App crashes with `"ScrollController attached to multiple scroll views"` the moment both tabs are built (immediately, since `TabBarView` preloads adjacent pages).
**Why it happens:** `DraggableScrollableSheet.builder` supplies one `ScrollController`; a `ScrollController` can only be attached to one active `ScrollPosition` at a time; `TabBarView` keeps neighbor pages alive.
**How to avoid:** Attach the builder's `scrollController` to exactly one tab (Route Anchors); use a separate `DraggableScrollableController` + manual `GestureDetector` drag on the handle bar for expand/collapse (Pattern 1).
**Warning signs:** Any code that passes the same `scrollController` variable into more than one widget inside `TabBarView.children`.

### Pitfall 2: Gating the height fetch on widget lifecycle instead of `TabController`
**What goes wrong:** `/valhalla/height` fires on every route edit regardless of which tab is showing, violating D-11 and wasting Valhalla calls.
**Why it happens:** Assuming `TabBarView` lazily builds/unmounts inactive tabs (it doesn't, for the reason in Pitfall 1).
**How to avoid:** Listen to `DefaultTabController.of(context).index`/`indexIsChanging` explicitly inside the Elevation tab's `initState`/listener, gate the fetch there.
**Warning signs:** A height-fetch call inside `build()` or `initState()` with no `TabController` check at all.

### Pitfall 3: Segment identity is anchor-id-pair based, not array-index based (carried over from Phase 19 research, still applicable)
**What goes wrong:** Iterating `state.segments` in array order to build the ordered GPX point list produces an out-of-sequence track after any insert/delete/reorder, since segment array order is not guaranteed to match anchor traversal order.
**Why it happens:** `RouteSegment` identity is `(beforeAnchorId, afterAnchorId)`, and segments are appended to the array in mutation order, not maintained in path order.
**How to avoid:** Walk from `state.anchors.first` following `segByBefore[currentId]` chains (Pattern 4), never `state.segments[i]` by index.
**Warning signs:** Any code doing `for (final s in state.segments) points.addAll(s.polyline)` without following the anchor-id chain.

### Pitfall 4: Rebuilding all segments on reorder discards resolved routing
**What goes wrong:** Every drag-to-reorder action (even reordering two anchors that don't change any other anchors' adjacency) re-resolves every segment via Valhalla, causing visible straight-line flicker and needless network calls.
**Why it happens:** The naive implementation of "recompute affected segments" is "recompute all segments."
**How to avoid:** Diff old vs. new adjacency by `segmentKey`, reuse unchanged segments verbatim (Pattern 3).
**Warning signs:** A `reorderAnchors` implementation with no reference to the pre-reorder `state.segments`.

### Pitfall 5: `ElevationProfile`'s `_buildChart` waypoint-overlay reading `widget.trail!` unguarded
**What goes wrong:** A naive nullable-conversion of `trail` (`Trail` → `Trail?`) without checking every existing usage site crashes with a null-check operator error the first time `trail` is actually null.
**Why it happens:** `_buildChart` (line 230) already uses `widget.trail.expand?.waypointsViaTrail ?? []` — safe once `trail` itself is nullable via `widget.trail?.expand?.`. The stats header (lines 176-187), however, reads `widget.trail.elevationGain`/`elevationLoss` **without** a null-safe accessor, since `trail` was non-nullable at write time — this line **will** crash if not updated.
**How to avoid:** Grep every `widget.trail` usage after the nullable change (2 sites: line 230's chart waypoints, lines 176-187's stats header) before considering the adaptation complete.
**Warning signs:** `dart analyze` will actually catch this at compile time (a non-null-safe access on a nullable type is a compile error, not a runtime crash) — so this is lower-risk than it sounds, but still worth an explicit verification step in the plan.

## Code Examples

### Verified current `RouteAnchorsState` shape (for planning the new mutators' signatures)
```dart
// Source: app/lib/provider/route_anchor_provider.dart:12-49 (read directly, 2026-07-16)
class RouteAnchorsState {
  final List<RouteAnchor> anchors;
  final List<RouteSegment> segments;
  final bool autoRoutingEnabled;
  final String travelProfile;
  final List<RouteAnchorsSnapshot> undoStack;
  final List<RouteAnchorsSnapshot> redoStack;
  // copyWith(...) — standard pattern, all fields nullable-override
}
```
Existing mutators (confirmed, none named `delete`/`reorder` yet): `appendAnchor`, `dragAnchor`, `insertAnchorOnSegment`, `retrySegment`, `toggleAutoRouting`, `undo`, `redo`.

### Verified `WaypointSheet` chrome constants to reuse for the new sheet
```dart
// Source: app/lib/components/trail/waypoint_sheet.dart:43-56 (read directly)
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

### Verified `route_planner_screen.dart`'s current top-right controls Column (exact insertion point for D-04's search button)
```dart
// Source: app/lib/routes/route_planner_screen.dart:250-262 (read directly)
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
`body:` is currently `_buildMap(context, state, styleJson)` directly (line 169) — to host the new bottom-docked sheet as a `Stack` sibling (not a `Positioned` child of the controls column, per D-03), `route_planner_screen.dart`'s `Scaffold.body` must become a `Stack` with the map as `Positioned.fill` and the new sheet as a conditional (`if (state.anchors.isNotEmpty)`) final child — this exact composition (map `Positioned.fill` + conditional sheet as trailing `Stack` child) is already proven working in `trail_detail_map_screen.dart:76-209` (`TrailMap` as `Positioned.fill`, `WaypointSheet` as the last unconstrained `Stack` child, conditionally rendered via `if (selectedWaypoint != null)`).

### Verified `LocationSearchResult` model (for the location-search screen's result type)
```dart
// Source: app/lib/models/global_search_models.dart:169-178 (read directly)
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
Existing `_LocationTile`'s current `onTap` (to be changed per D-14, from `context.go('/map', extra: {...})` to `context.pop(result)`):
```dart
// Source: app/lib/routes/global_search_screen.dart:326-330 (read directly)
onTap: () => context.go(
  '/map',
  extra: {'lat': location.lat, 'lon': location.lon, 'zoom': 13.0},
),
```

## State of the Art

Not applicable in the "old vs. new library version" sense — this phase makes no library upgrades. The one relevant "old vs. new approach" is architectural, within this same codebase:

| Old Approach (original ROADMAP wording) | Current Approach (D-02, this phase) | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Two map-control buttons toggling separate list/elevation views | Single persistent `DraggableScrollableSheet` with 2 tabs | 2026-07-16, during `/gsd-discuss-phase` | Simpler mechanism, same user-visible capability; `.planning/ROADMAP.md`/`REQUIREMENTS.md` amended accordingly |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | No documented hard point-count limit on Valhalla's `/height` endpoint specifically (unlike `/route`) | Pattern 5 | If Valhalla silently truncates or errors past some undocumented limit, elevation fetches for very long/dense routes could fail silently; mitigated by reusing the existing 500-point `buildNavShape()` downsampler defensively |
| A2 | Proposed file locations under `components/route_planner/` (new subfolder) | Recommended Project Structure | Low risk — purely organizational; planner/implementer can place files differently without functional impact |
| A3 | `deleteAnchor`/`reorderAnchors` algorithms (Patterns 2/3) are this research's own design, not sourced from any spec or existing code | Architecture Patterns | If the planner adopts a different (e.g., naive full-rebuild) approach instead, the specific "reuse unchanged segments" behavior described in Pitfall 4 won't hold — worth an explicit plan verification step either way |

**If this table is empty:** N/A — see entries above. All three are either external-library edge cases with no authoritative documentation, or original architectural proposals filling a gap 20-CONTEXT.md explicitly left as "recompute affected segments" without an algorithm.

## Open Questions

1. **Should `plannedGpxProvider` include `time` values on synthesized `Wpt`s?**
   - What we know: `ElevationProfile._parseGpx` reads `wpt.time` to compute cumulative duration (falls back to `Duration.zero` per point if absent, confirmed at `elevation_profile.dart:72-77` — `if (prevTime != null && currTime != null)`).
   - What's unclear: The route planner has no timestamp concept at all (anchors are just `lat`/`lon`); is a duration stat (shown in the stats header) meaningful/expected for a not-yet-hiked planned route, or should it always read as 0/omitted?
   - Recommendation: Leave `time` unset (null) on synthesized `Wpt`s — the existing fallback already handles this gracefully (duration reads as 0), and Phase 21's HANDOFF-01 (named waypoints on a draft Trail) is a more natural place to decide whether timing estimates matter, not this phase.

2. **Exact peek-height `initialChildSize`/`snapSizes` tuning.**
   - What we know: UI-SPEC proposes `0.14`/`0.14`/`0.6` as a starting point, explicitly marked "Claude's Discretion... tune against `WaypointSheet`'s existing values... if peek/expand feel wrong on-device."
   - What's unclear: Whether 0.14 (roughly 90-110px) comfortably fits the handle bar + `TabBar` without clipping on smaller devices — this is an on-device/visual judgment call, not something resolvable from source alone.
   - Recommendation: Treat as a checkpoint for on-device visual verification during execution, not a blocking planning question.

## Environment Availability

Skipped — this phase has no external tool/service/runtime dependencies beyond what's already installed (all packages pre-existing in `pubspec.yaml`; both consumed HTTP endpoints already exist and require no new server-side setup).

## Validation Architecture

Skipped — `.planning/config.json`'s `workflow.nyquist_validation` is explicitly `false`.

## Security Domain

`security_enforcement` is enabled (`security_asvs_level: 1`, absent-defaults-to-enabled confirmed via `.planning/config.json`). This phase's ASVS surface is minimal — it is a UI phase over already-existing, already-reviewed proxy endpoints; no new endpoints, no new auth/session logic, no new persisted data.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | No new auth surface — reuses existing `apiProvider`/cookie-jar session, unchanged |
| V3 Session Management | No | Same as above |
| V4 Access Control | No | No new server-side authorization logic; both consumed endpoints are unauthenticated pass-through proxies (confirmed reading both `+server.ts` files — no `event.locals.pb.authStore` check in either) |
| V5 Input Validation | Yes (client-side only) | The synthesized GPX points sent to `/valhalla/height` are derived entirely from user map-tap coordinates already validated by Phase 19's `_isValidCoordinate` guard (`route_anchor_provider.dart:78-83`, checks `lat`/`lon` range before ever calling Valhalla) — no new validation surface needed, the existing guard already covers every anchor these features read from |
| V6 Cryptography | No | Not applicable — no new crypto/secrets handling |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Unbounded query fanout via search-as-you-type (location search) | Denial of Service (resource exhaustion on the Nominatim upstream) | Already mitigated by the existing 500ms debounce idiom this phase reuses verbatim (`GlobalSearchNotifier`'s pattern) — no new mitigation needed, just don't remove the debounce when stripping the category-chip logic |
| Oversized `/valhalla/height` payload from a very long synthesized route | Denial of Service (upstream Valhalla load) / potential request failure | Mitigated by reusing `buildNavShape()`'s existing 500-point downsampler (A1 above) before sending |

No new authentication, authorization, or data-persistence surface is introduced by this phase — the security review for this phase should focus on confirming the two mitigations above are actually carried over, not on discovering new threats.

## Sources

### Primary (HIGH confidence)
- `app/lib/provider/route_anchor_provider.dart` — read in full, current mutators/state confirmed
- `app/lib/components/trail/elevation_profile.dart` — read in full, exact prop signature and `trail`-dependent lines identified
- `app/lib/util/gpx_util.dart` — read in full, `GpxStats`/`getTotals()`/`buildNavShape` confirmed
- `app/lib/models/route_anchor.dart`, `app/lib/util/route_segment_util.dart` — read in full, `RouteSegment`/`segmentKey`/`splitSegmentAt` confirmed
- `app/lib/routes/global_search_screen.dart`, `app/lib/provider/search/global_search_provider.dart` — read in full, exact current `GlobalSearchScreen`/`_LocationTile`/debounce idiom confirmed
- `app/lib/components/trail/waypoint_sheet.dart` — read in full, sheet chrome constants confirmed
- `app/lib/routes/trail_detail_map_screen.dart` — read in full, confirmed the `Stack`(map `Positioned.fill` + conditional trailing sheet child) composition pattern already works in this codebase
- `app/lib/routes/settings_categories_screen.dart` (lines 200-340) — read in full, exact `ReorderableListView.builder`/optimistic-reorder pattern confirmed
- `app/lib/routes/route_planner_screen.dart` — read in full, exact current build tree, controls Column, and `body:` structure confirmed
- `web/src/routes/api/v1/valhalla/height/+server.ts`, `web/src/routes/api/v1/geocoding/search/+server.ts` — read in full, confirmed pure pass-through proxies, no auth check, no request/response transformation
- `~/.pub-cache/hosted/pub.dev/gpx-2.3.0/lib/src/model/{gpx,trk,trkseg,wpt}.dart` — read directly, exact constructor signatures confirmed
- `app/pubspec.yaml` — read directly, all package versions confirmed
- [DraggableScrollableController class docs](https://api.flutter.dev/flutter/widgets/DraggableScrollableController-class.html) — `jumpTo`/`animateTo`/`size`/`isAttached` signatures confirmed
- [Valhalla elevation API reference](https://github.com/valhalla/valhalla-docs/blob/master/elevation/api-reference.md) — exact request/response JSON shapes confirmed

### Secondary (MEDIUM confidence)
- [flutter/flutter#55388](https://github.com/flutter/flutter/issues/55388) — "ScrollController attached to multiple scroll views" on `DraggableScrollableSheet` + `TabBarView`, cross-referenced against Flutter's own official example's workaround strategy
- [flutter/flutter#118713](https://github.com/flutter/flutter/issues/118713), [flutter/flutter#64157](https://github.com/flutter/flutter/issues/64157) — related `NestedScrollView` variant of the same conflict, confirming `NestedScrollView` is not a clean alternative
- Flutter official example `examples/api/lib/widgets/draggable_scrollable_sheet/draggable_scrollable_sheet.0.dart` (fetched via raw GitHub content) — `Grabber`/`onVerticalDragUpdate` pattern confirmed as the framework team's own recommended decoupling strategy

### Tertiary (LOW confidence)
- None — every claim above was either read directly from this repo's source, the installed package's source, or an official Flutter/Valhalla documentation source.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages, all versions read directly from `pubspec.yaml`
- Architecture (sheet/tab composition): HIGH — the core risk (scrollController conflict) is a filed, confirmed Flutter framework issue, and the fix mirrors Flutter's own official example
- Architecture (delete/reorder algorithms): MEDIUM — these are this research's own proposed designs (no existing precedent to verify against), flagged explicitly in the Assumptions Log
- Pitfalls: HIGH — each pitfall is either a verified framework issue or a directly-observed gap in existing source (e.g., `ElevationProfile`'s unguarded `widget.trail` stats-header reads)

**Research date:** 2026-07-16
**Valid until:** 30 days (stable, in-repo-source-driven research; no fast-moving external dependency)
