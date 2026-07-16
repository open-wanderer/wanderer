# Project Research Summary

**Project:** Wanderer v1.5 Route Planner
**Domain:** Interactive multi-waypoint route-planning UI on native-GL map (Flutter/Riverpod mobile app)
**Researched:** 2026-07-16
**Confidence:** HIGH

## Executive Summary

This is a mobile route-planning feature (tap/drag/insert/delete/reorder waypoints, auto-routing toggle, undo/redo, live elevation/distance stats) added to Wanderer's existing Flutter app, in the same category as Komoot, Strava Route Builder, and gpx.studio. The research is unusually strong: the web app already ships an equivalent feature (`valhalla_store.svelte.ts`'s `calculateRouteBetween`), the exact SvelteKit/Valhalla backend endpoints already exist and require zero changes (`/api/v1/valhalla/route`, `/api/v1/valhalla/height`), and the Flutter side already has near-identical map-interaction precedent (`TrailMarkerLayer`'s per-marker `GestureDetector` + `MapController.toLngLat`). No new pubspec dependencies are required.

The recommended approach: build a new ephemeral, top-level `@riverpod class RoutePlanner` provider (mirroring `navigation_provider.dart`, not the record-backed `provider/trail/*` providers) that holds waypoints, undo/redo stacks, auto-routing flag/profile, and a synthesized `Gpx`. Map interaction is a new `RoutePlannerMap`/`RoutePlannerMarkerLayer` pair modeled on (not extending) `TrailMap`/`TrailMarkerLayer`, to avoid touching a component shared by 3+ other screens. Routing calls Valhalla's `/route` endpoint per consecutive waypoint pair (never `/navigate`, which is map-matching, not routing) plus `/height` for elevation — exactly mirroring the web implementation. Handoff to the existing `trail_create_screen` reuses the `pendingImportedTrail` global pattern from GPX import, requiring one new `Trail`/`Gpx` synthesis function.

The primary risks are all state/interaction-timing bugs, not technology gaps: gesture-arena conflicts between marker-drag and map-pan, un-debounced Valhalla calls causing request storms during drag, out-of-order async responses overwriting newer edits with stale ones, undo/redo capturing derived (not source-of-truth) state, and mixing fast local drag state with async network state in one Riverpod notifier. All five are well-understood with concrete prevention patterns (generation counters, debounce-in-provider, split sync/async providers, gesture-disable-during-drag) — none require new packages, but all require deliberate architecture decisions made early rather than retrofitted.

## Key Findings

### Recommended Stack

No new pubspec dependencies are required. The existing pinned stack (`package:maplibre` 0.3.5, `flutter_riverpod`/`riverpod_annotation` 3.3.1/4.0.2, `freezed` 3.2.5, `package:gpx` 2.3.0) covers every requirement, confirmed by reading the actual installed package sources rather than relying on training data.

**Core technologies:**
- `package:maplibre` (`ml.WidgetLayer`/`ml.Marker`) — map rendering and waypoint pins; has no native drag support, so drag is a ~40-line custom `GestureDetector` + `MapController.toLngLat(Offset)` synchronous conversion, not a library gap.
- `flutter_riverpod` (`@riverpod class RoutePlanner extends _$RoutePlanner`) — route-plan state container with imperative add/move/insert/delete/reorder/undo/redo methods, matching the existing `TrailSave`/`Navigation` idiom.
- `freezed` — immutable `RoutePlanState`/`RoutePlanWaypoint` models; remember the project's known 3.x gotcha (`@JsonSerializable(explicitToJson: true)` must sit on the factory constructor, not the class).
- `package:gpx` — synthesizing a `Gpx` object (`Gpx`/`Trk`/`Trkseg`/`Wpt`) from the in-memory route for `ElevationProfile` and GPX handoff; a new `buildGpxFromPoints` helper is the structural inverse of the existing `buildNavShape`.
- Hand-rolled `List<RoutePlanState>` snapshot-stack undo/redo — simpler than the optional `undo` package (rodydavis, 1.6.0) given the route plan's small state size; prefer this unless action variety grows large.

### Expected Features

Verified against Komoot, Strava Route Builder, gpx.studio, and OsmAnd documentation, cross-checked against Wanderer's own PROJECT.md scope.

**Must have (table stakes):**
- Tap-to-add, drag-to-reposition, insert-mid-segment, delete, and reorder waypoints — the entire editing surface; missing any one makes the tool feel half-built.
- Undo/redo — mobile drag precision is worse than desktop; users will mis-place waypoints and need a cheap recovery path.
- Auto-routing toggle (Valhalla, foot/bike) vs. straight-line, re-resolved on toggle/profile change.
- Live distance/elevation stats + elevation profile view (reusing existing `GpxMappingUtils.getTotals()` and `ElevationProfile` widget).
- Search-to-focus panning (reuse `GlobalSearchScreen`, though it needs a small modification — see Architecture).
- Handoff to `trail_create_screen` as a draft `Trail`.

**Should have (competitive, already in scope):**
- Foot/bike profile switch mid-plan with instant re-routing (already in Active scope).
- Trail-category-aware waypoint icons reusing the existing icon system.

**Defer (v2+):**
- Offline-aware graceful degradation banner for auto-routing (add after real-world connectivity issues surface).
- Confirm-placement drag affordance (tap-drop-adjust-confirm) if mis-drops prove common in testing.
- Per-segment travel profiles, automatic loop generation, multi-day touring, editing an existing trail's route — all explicitly out of scope per PROJECT.md.
- **Do not build:** route optimization/auto-reorder (conflicts with user-intended waypoint ordering).

### Architecture Approach

The new screen is additive: a new top-level ephemeral Riverpod provider (`route_planner_provider.dart`), a new map-interaction pair (`route_planner_map.dart` + `route_planner_marker_layer.dart`) modeled on but not extending `TrailMap`/`TrailMarkerLayer`, and small additive changes to three existing files (`gpx_util.dart` gets `buildGpxFromPoints`, `trail_import_util.dart` gets `handoffPlannedRoute()`, `router_provider.dart` gets one new route). Zero backend changes — `/api/v1/valhalla/route` and `/api/v1/valhalla/height` already exist and are already proven by the web app's own shipped equivalent feature.

**Major components:**
1. `RoutePlanner` provider — owns waypoints, undo/redo stacks, auto-routing flag/profile, synthesized `Gpx`; imperative mutation methods with async routing resolution.
2. `RoutePlannerMap` / `RoutePlannerMarkerLayer` — native-GL map host: tap-to-add via `MapEventClick`, per-marker drag via `GestureDetector` + `toLngLat`, insert-mid-segment via `featuresAtPoint` hit-testing on a dedicated route-line layer (genuinely new, no precedent).
3. `RoutePlannerScreen` — screen shell: map + mutually-exclusive waypoint-list/elevation-profile bottom sheet + control buttons.
4. `handoffPlannedRoute()` — builds a draft `Trail` from planner state, reuses the existing `pendingImportedTrail` global, pushes to `/trail/create/edit` (unchanged downstream).

**Known gotcha:** `GlobalSearchScreen`'s location tile is hardcoded to `context.go('/map', ...)` — it cannot be reused as a drop-in "pan my own map" flow without a small additive modification (optional callback or pop-with-result pattern). Flag this explicitly during phase planning.

### Critical Pitfalls

1. **Marker-drag vs. map-pan gesture conflict** — both recognizers can claim the same pointer. Avoid by disabling map pan/rotate gestures for the drag's duration (restore on end/cancel/dispose) and using padded `HitTestBehavior.opaque` hit regions.
2. **Un-debounced Valhalla re-routing on every drag frame** — update straight-line preview locally/instantly, but debounce the actual routing network call (provider-layer `Timer`, not per-`onPanUpdate`), triggering primarily on drag-end.
3. **Out-of-order async responses overwrite newer edits** — this exact race exists latently today in `map_cluster_search_provider.dart` (debounce-only, no in-flight guard). Add a monotonically increasing request-generation counter (or `CancelToken`) in the routing notifier; discard stale responses. This is the single highest-value pitfall to test explicitly (mock Dio with reversed resolution order).
4. **Undo/redo capturing derived route geometry instead of source-of-truth waypoints** — never snapshot the Valhalla-derived polyline; always re-derive it (debounced, generation-guarded) after undo/redo restores a waypoint list. Cap undo depth and coalesce per-gesture, not per-frame.
5. **Combining fast local drag state with async network state in one Riverpod notifier** — split into a synchronous waypoint-list `Notifier` and a separate `AsyncNotifier`/`FutureProvider` for Valhalla-derived geometry; never route per-frame drag updates through `AsyncValue.guard`/`AsyncLoading`.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Provider Architecture & Pure Utilities
**Rationale:** Every other piece (map interaction, screen UI, handoff) depends on this state shape existing; also the phase most exposed to Pitfall 5 (combined sync/async provider), which is expensive to retrofit later. Testable without any UI.
**Delivers:** `RoutePlanner`/waypoint-list split providers, `buildGpxFromPoints()` in `gpx_util.dart`, the Valhalla `/route`+`/height` per-pair call sequence with debounce and generation-guard, hand-rolled undo/redo snapshot stack (source-of-truth waypoints only).
**Addresses:** Auto-routing toggle, undo/redo, live distance/elevation stats (FEATURES.md table stakes).
**Avoids:** Pitfall 2 (un-debounced routing), Pitfall 3 (out-of-order responses), Pitfall 4 (undo storing derived state), Pitfall 5 (combined sync/async provider).

### Phase 2: Map Interaction Layer
**Rationale:** Highest-uncertainty new code (insert-mid-route hit-testing has zero precedent in the codebase) — de-risk it early, before UI is built on top. Depends on Phase 1's provider existing to wire callbacks into.
**Delivers:** `route_planner_map.dart` + `route_planner_marker_layer.dart` — tap-to-add (native `MapEventClick`), per-marker drag (`GestureDetector`+`toLngLat`), insert-mid-segment (`featuresAtPoint` on a dedicated route layer).
**Uses:** `package:maplibre` `WidgetLayer`/`Marker`/`MapController.toLngLat`, existing `TrailMarkerLayer` drag pattern as reference.
**Avoids:** Pitfall 1 (gesture-arena conflict between drag and map-pan).

### Phase 3: Screen Shell & Elevation/List UI
**Rationale:** Builds the visible screen once the state and map-interaction foundations are proven; reuses established `DraggableScrollableSheet`/`ElevationProfile` patterns almost unmodified.
**Delivers:** `route_planner_screen.dart` — mutually-exclusive waypoint-list (`ReorderableListView`) / elevation-profile bottom sheet, control buttons (auto-routing toggle, profile switch, undo/redo).
**Addresses:** Waypoint list view, live elevation profile chart, delete/reorder waypoint UI (FEATURES.md table stakes).

### Phase 4: Search-to-Focus Wiring
**Rationale:** Requires the screen shell to exist as a concrete "re-center my map" target; also requires the small `GlobalSearchScreen` modification identified in ARCHITECTURE.md (currently hardcoded to `context.go('/map', ...)`, not reusable as-is).
**Delivers:** Modified `global_search_screen.dart` (optional callback or pop-with-result pattern) wired into `RoutePlannerScreen`.
**Addresses:** Search-to-focus panning (FEATURES.md table stakes).

### Phase 5: Handoff to trail_create_screen
**Rationale:** Ordered last so the entry point (`/trail/create/plan`) only goes live once the full screen behind it actually works, avoiding a dead/broken route reachable mid-build.
**Delivers:** `handoffPlannedRoute()` in `trail_import_util.dart`, `router_provider.dart` route registration, `trail_source_select_screen.dart` one-line entry-point wire-up.
**Implements:** Synthesized-stub-Trail pattern (reusing `pendingImportedTrail` global), Trail/Gpx synthesis from planner state.

### Phase Ordering Rationale

- Provider/state-shape decisions (Phase 1) come first because Pitfall 5 (combined sync/async provider) and Pitfall 3 (race conditions) are foundational and expensive to retrofit once UI/undo/routing all depend on a single provider shape.
- Map interaction (Phase 2) is sequenced before screen UI because it's the highest-uncertainty new code (no precedent for insert-mid-route hit-testing) and should be de-risked early rather than discovered late.
- Handoff is deliberately last, matching ARCHITECTURE.md's explicit build-order recommendation, so the entry point never exposes a half-working screen.
- This ordering directly avoids Pitfalls 1-5 by construction: debounce/generation-guard exist before undo/redo is layered on (Pitfall 4's stated hard dependency), and gesture handling is proven before it's wrapped in screen chrome.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 1 (Provider Architecture):** the generation-counter/CancelToken race-guard pattern is MEDIUM confidence (standard Riverpod community pattern, not verified against this project's exact pinned Riverpod version) — worth a research-phase pass to confirm the exact idiom against `riverpod_annotation` 4.0.2.
- **Phase 2 (Map Interaction Layer):** `package:maplibre` 0.3.5's exact `MapGestures`/`MapOptions` pan/rotate-disable API surface is MEDIUM confidence (cross-referenced from GitHub discussions and changelog, not confirmed against the exact pinned API by direct source read) — validate with a small implementation spike before committing to the full interaction layer.

Phases with standard patterns (skip research-phase):
- **Phase 3 (Screen Shell):** reuses already-shipping `DraggableScrollableSheet`/`ElevationProfile`/`ReorderableListView` patterns verbatim — HIGH confidence, no new research needed.
- **Phase 5 (Handoff):** reuses the exact, already-shipping `pendingImportedTrail` + `trail_create_screen` pattern from GPX import — HIGH confidence.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Verified against actual installed package source (`maplibre_platform_interface`), actual existing SvelteKit endpoints, and actual existing Flutter code — not training data. No new dependencies needed. |
| Features | MEDIUM-HIGH | Verified against official Komoot/Strava/gpx.studio/OsmAnd documentation plus Wanderer's own existing infrastructure; competitor docs are official support pages (MEDIUM), cross-checked against a working in-repo reference implementation (web app) for HIGH-confidence internal claims. |
| Architecture | HIGH | Every claim backed by a specific file read in this repo; the web app already ships a near-identical feature, providing a working reference implementation rather than a hypothesis. |
| Pitfalls | MEDIUM | Grounded in verified existing codebase patterns (HIGH) plus MEDIUM-confidence external verification of `package:maplibre` gesture/coordinate APIs (pub.dev changelog + GitHub discussions; official docs site is thin on gesture internals). |

**Overall confidence:** HIGH

### Gaps to Address

- `package:maplibre` 0.3.5's exact `MapGestures`/`MapOptions` gesture-disable API (pan/rotate booleans) was verified via changelog/GitHub discussion, not direct source read of the exact toggle surface — validate with a quick spike at the start of Phase 2 before committing to the full drag-disable implementation.
- The Valhalla `/route` and `/height` endpoints currently have no auth check (`event.locals.user`), unlike `/navigate`. Not blocking, but flagged in STACK.md for a one-line confirmation during implementation that this is accepted behavior rather than an oversight, since the Flutter app newly calling these endpoints slightly increases exposure.
- `GlobalSearchScreen`'s hardcoded `/map` navigation needs a concrete design decision (optional callback vs. pop-with-result) before Phase 4 — flagged, not yet resolved to a single approach.

## Sources

### Primary (HIGH confidence)
- Direct reads of this repository: `app/lib/util/gpx_util.dart`, `navigation_launch_util.dart`, `trail_import_util.dart`, `components/trail/elevation_profile.dart`, `provider/trail/trail_save_provider.dart`, `provider/router_provider.dart`, `routes/map_screen.dart`, `routes/trail_create_screen.dart`, `components/base/trail_map.dart`, `components/map/trail_layer.dart`, `provider/trail/map_cluster_search_provider.dart`, `routes/global_search_screen.dart`, `routes/trail_source_select_screen.dart`
- `web/src/lib/stores/valhalla_store.svelte.ts` and `web/src/lib/util/valhalla_anchor_util.ts` — the already-shipped web version of this exact feature (route planner with drag anchors, undo/redo, auto-routing toggle, pairwise Valhalla calls, height lookups).
- `web/src/routes/api/v1/valhalla/route/+server.ts`, `.../height/+server.ts`, `.../navigate/+server.ts` — confirmed exact existing endpoint contracts.
- `.pub-cache/hosted/pub.dev/maplibre-0.3.5/lib/src/widget_layer.dart`, `maplibre_platform_interface-0.3.5/lib/src/map_controller.dart` — confirmed `Marker`/`WidgetLayer`/`MapController.toLngLat` API surface directly from installed package source.
- `.pub-cache/hosted/pub.dev/gpx-2.3.0/lib/src/model/{gpx,trk,trkseg,wpt}.dart` — confirmed constructor shapes for `buildGpxFromPoints`.
- `.planning/PROJECT.md` — v1.5 milestone scope, constraints, out-of-scope list.

### Secondary (MEDIUM confidence)
- Official support docs: Komoot (route planning, Android/iOS), Strava (Creating Routes on Mobile), gpx.studio (routing/edit toolbar), OsmAnd (Plan a Route, Map Markers).
- [maplibre changelog (pub.dev)](https://pub.dev/packages/maplibre/changelog) and [GitHub Discussion #683](https://github.com/maplibre/flutter-maplibre-gl/discussions/683) — gesture-toggle API cross-referenced but not confirmed against exact pinned 0.3.5 surface.
- Valhalla routing documentation (Stadia Maps guide, Valhalla API reference) — cross-checked against the working web implementation.
- `undo` package (pub.dev, rodydavis 1.6.0) — presented as optional, not required.

### Tertiary (LOW confidence)
- General UX blog sources (Eleken, Upslide Design Studio) on touch-target sizing and one-handed mobile UX — used only for general guidance, not product-specific claims.
- [Marker - MapLibre Flutter docs](https://flutter-maplibre.pages.dev/docs/annotations/markers/) — confirms marker/layer distinction but does not document drag gestures explicitly (verified gap).

---
*Research completed: 2026-07-16*
*Ready for roadmap: yes*
