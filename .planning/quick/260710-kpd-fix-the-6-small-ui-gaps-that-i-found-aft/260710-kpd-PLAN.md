---
phase: quick-260710-kpd
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/lib/components/base/wanderer_attribution.dart
  - app/lib/components/map/trail_layer.dart
  - app/lib/components/base/wanderer_map.dart
  - app/lib/components/base/search_map.dart
  - app/lib/routes/list_detail_screen.dart
  - app/lib/routes/list_detail_map_screen.dart
  - app/lib/routes/map_screen.dart
  - app/lib/routes/navigation_screen.dart
autonomous: true
requirements: [KPD-01, KPD-02, KPD-03, KPD-04, KPD-05, KPD-06]
must_haves:
  truths:
    - "Directional arrows render along the trail polylines on list_detail_screen's preview map and list_detail_map_screen, matching trail_detail_map_screen"
    - "list_detail_map_screen shows a compass control that appears when the map is rotated off-north"
    - "list_detail_screen's preview map shows a scale bar and an attribution control"
    - "The attribution control starts collapsed (info button only, no expanded text) on every map surface on first load"
    - "trail_detail_map_screen and trail_detail_screen fit the full trail bounds on open instead of centering on the trail's start point"
    - "On trail_detail_map_screen the map controls stay right-aligned when the compass appears/disappears during rotation"
  artifacts:
    - path: "app/lib/components/base/wanderer_attribution.dart"
      provides: "WandererAttribution — a collapsed-by-default attribution control, drop-in for ml.SourceAttribution"
      contains: "class WandererAttribution"
    - path: "app/lib/components/map/trail_layer.dart"
      provides: "addPolylineArrowLayer helper that adds a native directional-arrow symbol layer over a set of polylines"
      contains: "addPolylineArrowLayer"
    - path: "app/lib/components/base/wanderer_map.dart"
      provides: "GPX-derived bounds fit + right-aligned controls + collapsed attribution"
      contains: "getBounds()"
  key_links:
    - from: "app/lib/routes/list_detail_map_screen.dart"
      to: "addPolylineArrowLayer"
      via: "onStyleLoaded call after fitBounds"
      pattern: "addPolylineArrowLayer"
    - from: "app/lib/components/base/wanderer_map.dart"
      to: "gpx.getBounds()"
      via: "_fitInitialCamera prefers GPX bounds over trail.bounds"
      pattern: "getBounds\\(\\)"
---

<objective>
Close the six on-device UI gaps recorded in Phase 18's verification walk (`18-03-SUMMARY.md` "Gap Candidates"). All six are pre-existing MapLibre polish issues on the shared `WandererMap` / `SearchMap` widgets and the four trail/list screens that host them — none is a regression from the v1.4 migration, they were simply never wired.

Purpose: Bring the list and single-trail map surfaces to parity with `trail_detail_map_screen` / `navigation_screen`, which already get arrows, compass, scale/attribution, and bounds-fit right.
Output: One new shared widget (`WandererAttribution`), one new helper in `trail_layer.dart` (`addPolylineArrowLayer`), and edits across the two base map widgets and four route screens.

Gap → fix map:
- Gap 1 (arrows on list polylines) → new `addPolylineArrowLayer`, called from both list screens' `onStyleLoaded`.
- Gap 2 (compass on `list_detail_map_screen`) → add `ml.MapCompass(hideIfRotatedNorth: true)` to its `SearchMap` children.
- Gap 3 (scale/attribution on `list_detail_screen`) → add scale bar + attribution to `_ListMap`'s children (its non-null `children` currently suppresses `SearchMap`'s defaults).
- Gap 4 (attribution starts collapsed everywhere) → `ml.SourceAttribution` has no "start collapsed" option (it hardcodes `_expanded = true`, only auto-collapsing on the first camera move). Replace every usage with a collapsed-by-default `WandererAttribution`.
- Gap 5 (detail views fit start point, not full bounds) → `WandererMap._fitInitialCamera` uses `trail.bounds`, whose `min/max_lat/lon` are `@Default(0)` and unpopulated on the single-trail `GET /trail/:id` record, so `hasExtent` is false and it falls back to `moveCamera(center, zoom:18)`. Prefer bounds derived from the GPX track (`gpx.getBounds()`), which is always present on detail records.
- Gap 6 (control shift on rotate) → `WandererMap`'s controls `Column` uses `CrossAxisAlignment.center`; when the wider compass appears the narrower control stack re-centers. Switch to `CrossAxisAlignment.end`.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/phases/18-retire-flutter-map-and-the-flomp-forks/18-03-SUMMARY.md

# Files to modify
@app/lib/components/map/trail_layer.dart
@app/lib/components/base/wanderer_map.dart
@app/lib/components/base/search_map.dart
@app/lib/routes/list_detail_screen.dart
@app/lib/routes/list_detail_map_screen.dart
@app/lib/routes/map_screen.dart
@app/lib/routes/navigation_screen.dart

# Read-only references (the already-correct patterns to mirror)
# - addTrailTrackLayers in trail_layer.dart: the single-trail arrow symbol layer + _kTrailArrowImageId + _kArrowSpacing this plan reuses
# - trail_detail_map_screen.dart: correct MapCompass-in-controls usage
# - gpx_util.dart: GpxMappingUtils.getBounds() -> LngLatBounds? (LngLatBounds.fromPoints)
# - maplibre 0.3.5 source_attribution.dart: model WandererAttribution's pill + link rendering on it, but start collapsed
@app/lib/routes/trail_detail_map_screen.dart
@app/lib/util/gpx_util.dart
</context>

<tasks>

<task type="auto">
  <name>Task 1: Shared map-widget fixes — collapsed attribution, arrow helper, GPX bounds fit, right-aligned controls</name>
  <files>app/lib/components/base/wanderer_attribution.dart, app/lib/components/map/trail_layer.dart, app/lib/components/base/wanderer_map.dart, app/lib/components/base/search_map.dart</files>
  <action>
Create `app/lib/components/base/wanderer_attribution.dart` — a collapsed-by-default attribution control (Gap 4). Model it closely on maplibre 0.3.5's `SourceAttribution` (`~/.pub-cache/hosted/pub.dev/maplibre-0.3.5/lib/src/ui/source_attribution.dart`), reusing its pill layout (a rounded `Container` at `Alignment.bottomRight` with an `Icons.info` `IconButton` that toggles an expanded row) and its `_HtmlWidget`-style `RichText` rendering of attribution HTML (keep attribution links tappable via `TapGestureRecognizer` + `url_launcher`). Two deliberate differences from the package widget: (1) initialize the expanded flag to `false` so it always starts collapsed; (2) delete the `_initMapCamera` / camera-change auto-collapse logic entirely — it is unnecessary once the control starts collapsed. Read the style via `ml.MapController.maybeOf(context)?.style`; if null, return `const SizedBox.shrink()`. Build the attribution list as `['<a href="https://pub.dev/packages/maplibre">MapLibre</a>', ...style.getAttributionsSync()]`. Expose a `const WandererAttribution({super.key})` constructor so it is a drop-in for `const ml.SourceAttribution()`. Wrap in `SafeArea` + `PointerInterceptor` exactly as the package does.

In `app/lib/components/map/trail_layer.dart`, add a public helper `Future<void> addPolylineArrowLayer(ml.StyleController style, List<List<ml.Geographic>> lines, {String sourceId = 'list-trail-arrows', String layerId = 'list-trail-arrows-symbols'})` (Gap 1 mechanism). It mirrors the arrow half of the existing `addTrailTrackLayers`: return early if `lines.isEmpty`; register the arrow image via `style.addImageFromIconData(id: _kTrailArrowImageId, iconData: Icons.arrow_left, size: 32, color: Colors.white)` (reuse the existing file-private `_kTrailArrowImageId`); build a GeoJSON FeatureCollection whose features are one LineString per entry in `lines` (`jsonEncode`, never string concatenation — same T-15-05-01 hygiene as `addTrailTrackLayers`, coordinates as `[lon, lat]`); `await style.addSource(ml.GeoJsonSource(id: sourceId, data: <encoded>))`; then `await style.addLayer(ml.SymbolStyleLayer(id: layerId, sourceId: sourceId, minZoom: 8, layout: {...}))` with the SAME layout map the existing `trail-arrows` layer uses (`symbol-placement: line`, `icon-image: _kTrailArrowImageId`, `icon-rotation-alignment: map`, `icon-allow-overlap: true`, `icon-ignore-placement: true`, `icon-size: 0.5`, `symbol-spacing: _kArrowSpacing`), reusing the existing file-private `_kArrowSpacing` constant. Do NOT add casing/route line layers here — the list screens keep drawing their line via the declarative `ml.PolylineLayer`; this helper adds arrows only, on top.

In `app/lib/components/base/wanderer_map.dart`:
- Gap 5: in `_fitInitialCamera`, prefer bounds derived from the actual GPX track over the record's `min/max`-based `trail.bounds`. Compute `final gpxBounds = widget.trail.expand?.gpx?.getBounds();` and use `final bounds = gpxBounds ?? widget.trail.bounds;` before the existing `hasExtent` check. This fixes the single-trail detail views where `min/max_lat/lon` are unpopulated (`@Default(0)`) so `hasExtent` was false and the camera fell back to `moveCamera(center, zoom: 18)`. Add `import 'package:wanderer/util/gpx_util.dart';` for the `GpxMappingUtils.getBounds()` extension. Leave the `moveCamera` else-branch as the final fallback for the (degenerate) no-extent case.
- Gap 6: change the controls `Column`'s `crossAxisAlignment: CrossAxisAlignment.center` (inside the `Align(alignment: Alignment.topRight, ...)` at the bottom of `_buildMap`) to `CrossAxisAlignment.end`, so the control stack stays right-aligned regardless of whether the wider `MapCompass` is currently visible.
- Gap 4: replace `const ml.SourceAttribution()` in the map children with `const WandererAttribution()`; add `import 'package:wanderer/components/base/wanderer_attribution.dart';`. Keep `const ml.MapScalebar()` as-is.

In `app/lib/components/base/search_map.dart`, Gap 4: in the default `children` fallback (`const [ml.MapScalebar(), ml.SourceAttribution()]`), replace `ml.SourceAttribution()` with `WandererAttribution()` and add the import. Note the list must stop being `const` if `WandererAttribution`'s constructor cannot be const at that position — keep it const if possible, otherwise drop `const` on that literal only.
  </action>
  <verify>
    <automated>cd app && test $(flutter analyze 2>&1 | grep -c "error •") -eq 0 && grep -q "class WandererAttribution" lib/components/base/wanderer_attribution.dart && grep -q "addPolylineArrowLayer" lib/components/map/trail_layer.dart && grep -q "getBounds()" lib/components/base/wanderer_map.dart && grep -q "CrossAxisAlignment.end" lib/components/base/wanderer_map.dart</automated>
  </verify>
  <done>`flutter analyze` reports zero `error •` lines; `WandererAttribution` and `addPolylineArrowLayer` exist; `wanderer_map.dart` fits GPX bounds and right-aligns its controls Column; both base widgets use `WandererAttribution` instead of `ml.SourceAttribution`.</done>
</task>

<task type="auto">
  <name>Task 2: Route-screen wiring — arrows, compass, scale/attribution on list & detail surfaces, collapsed attribution everywhere</name>
  <files>app/lib/routes/list_detail_screen.dart, app/lib/routes/list_detail_map_screen.dart, app/lib/routes/map_screen.dart, app/lib/routes/navigation_screen.dart</files>
  <action>
Depends on Task 1 (`WandererAttribution`, `addPolylineArrowLayer` must exist).

In `app/lib/routes/list_detail_screen.dart` (`_ListMap`):
- Gap 1: build `final lines = widget.trails.where((t) => t.polyline != null && t.polyline!.isNotEmpty).map((t) => PolylineUtil.decode(t.polyline!)).toList();` (a `List<List<ml.Geographic>>`, reusing the same `PolylineUtil.decode` already used to build `polylines`). In the `SearchMap` `onStyleLoaded` callback, after the existing `fitBounds`, call `addPolylineArrowLayer(style, lines).ignore();` (the callback's `style` parameter is the `ml.StyleController`). Add `addPolylineArrowLayer` to the existing `import '.../trail_layer.dart' show kTrailRouteColor;` show-clause.
- Gap 3: add `const ml.MapScalebar()` and `const WandererAttribution()` to the `_ListMap` `SearchMap` `children` list (currently only `[if (markers.isNotEmpty) ml.WidgetLayer(markers: markers)]`). Its non-null `children` is exactly why `SearchMap`'s default scale/attribution never showed. Add the `WandererAttribution` import.

In `app/lib/routes/list_detail_map_screen.dart`:
- Gap 1: build the same `lines` list from `trails` and call `addPolylineArrowLayer(style, lines).ignore();` after the existing `fitBounds` inside `onStyleLoaded`. Add `addPolylineArrowLayer` to the `trail_layer.dart` show-clause.
- Gap 2: add `const ml.MapCompass(hideIfRotatedNorth: true)` to the `SearchMap` `children` (mirroring `trail_detail_map_screen`'s compass; this screen has gestures enabled so rotation is possible and a compass is meaningful). `MapCompass` self-positions at `Alignment.topRight` by default.
- Gap 4: replace `const ml.SourceAttribution()` with `const WandererAttribution()`; keep `const ml.MapScalebar()`. Add the import.

In `app/lib/routes/map_screen.dart` and `app/lib/routes/navigation_screen.dart`:
- Gap 4 ("across all map screens"): replace their `const ml.SourceAttribution()` usages with `const WandererAttribution()` and add the `import 'package:wanderer/components/base/wanderer_attribution.dart';`. Leave `ml.MapScalebar` and the existing `MapCompass` usages untouched.

After these edits, there must be no remaining `ml.SourceAttribution(` widget construction anywhere under `app/lib` (all six call sites — 2 base widgets from Task 1 + these 4 route files — now use `WandererAttribution`). The `getAttributionsSync()` call inside `WandererAttribution` is the only place attribution data is read.
  </action>
  <verify>
    <automated>cd app && test $(flutter analyze 2>&1 | grep -c "error •") -eq 0 && test $(grep -rc "ml.SourceAttribution(" lib | grep -v ':0' | wc -l | tr -d ' ') -eq 0 && grep -q "addPolylineArrowLayer" lib/routes/list_detail_screen.dart && grep -q "addPolylineArrowLayer" lib/routes/list_detail_map_screen.dart && grep -q "MapCompass" lib/routes/list_detail_map_screen.dart && grep -q "WandererAttribution" lib/routes/map_screen.dart && grep -q "WandererAttribution" lib/routes/navigation_screen.dart</automated>
  </verify>
  <done>`flutter analyze` reports zero `error •` lines; no `ml.SourceAttribution(` construction remains under `app/lib`; both list screens call `addPolylineArrowLayer`; `list_detail_map_screen` has a `MapCompass`; `list_detail_screen`'s preview has scale + attribution; `map_screen` and `navigation_screen` use `WandererAttribution`.</done>
</task>

</tasks>

<verification>
Automated (both tasks): `cd app && flutter analyze` — zero `error •` lines (the ~36 pre-existing info/warning issues from Phase 18 are unchanged and acceptable; only new *errors* fail the gate).

On-device confirmation (the six gaps are visual/behavioral — confirm on a physical Android device, mirroring the Phase 18 walk):
1. Open a list detail screen and its full-screen map: trail polylines now show direction-of-travel arrows.
2. On the full-screen list map, rotate the map: a compass appears (and hides when rotated back to north).
3. On the list detail screen preview map: a scale bar and attribution info button are visible.
4. On every map surface (trail map, list map, list preview, map screen, navigation): the attribution starts as a collapsed info button, not expanded text.
5. Open a trail detail screen and the trail detail map: the camera fits the whole trail, not just its start point.
6. On the trail detail map, rotate off-north and back: the right-hand control stack stays right-aligned the whole time (no horizontal jump when the compass appears/disappears).
</verification>

<success_criteria>
- All 6 gaps from `18-03-SUMMARY.md` are addressed by the changes described above.
- `flutter analyze` introduces zero new errors.
- No `ml.SourceAttribution(` widget construction remains under `app/lib`; attribution everywhere goes through the collapsed-by-default `WandererAttribution`.
- Arrow rendering on the list surfaces reuses the exact `trail-arrow` image id and `_kArrowSpacing` the single-trail arrows already use (visual parity, no id collision).
- The GPX-bounds fit and right-aligned controls changes live in the shared `WandererMap`, so both `trail_detail_map_screen` and `trail_detail_screen` (via `trail_panel`) benefit without per-screen edits.
</success_criteria>

<output>
Create `.planning/quick/260710-kpd-fix-the-6-small-ui-gaps-that-i-found-aft/260710-kpd-SUMMARY.md` when done.
</output>
