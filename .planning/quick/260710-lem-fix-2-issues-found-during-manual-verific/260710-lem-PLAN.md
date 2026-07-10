---
phase: quick-260710-lem
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/lib/components/base/wanderer_map.dart
  - app/lib/routes/list_detail_map_screen.dart
autonomous: true
requirements: [LEM-01, LEM-02]
must_haves:
  truths:
    - "Opening trail_detail_map_screen (and trail_detail_screen via trail_panel) fits the full trail bounds on first load — the camera is no longer stuck zoomed-in on the trailhead/start point, even when the native map fires onStyleLoaded before onMapCreated"
    - "The compass on list_detail_map_screen sits at the same vertical height as the AppBar back button (top-right), not jammed against the top edge of the screen"
  artifacts:
    - path: "app/lib/components/base/wanderer_map.dart"
      provides: "onMapCreated/onStyleLoaded _pendingStyle race buffer so the initial fitBounds always runs; initial fit reads the populated trail.bounds (min/max_lat/lon), not gpx.getBounds()"
      contains: "_pendingStyle"
    - path: "app/lib/routes/list_detail_map_screen.dart"
      provides: "SafeArea-wrapped, back-button-aligned MapCompass placement"
      contains: "SafeArea"
  key_links:
    - from: "app/lib/components/base/wanderer_map.dart onStyleLoaded"
      to: "_fitInitialCamera via _onStyleLoaded"
      via: "buffered replay once _controller is non-null"
      pattern: "_pendingStyle"
---

<objective>
Fix the 2 issues the user found while manually verifying the 260710-kpd UI-gap fixes on a physical device:

1. **Compass too high on `list_detail_map_screen`** (kpd gap 2, PASS-but-misplaced): the native `ml.MapCompass` self-positions at `Alignment.topRight` with a 10px pad and NO `SafeArea`, so on this `extendBodyBehindAppBar: true` screen it lands up in the status-bar area — above the AppBar back button. Move it down to align with the back button's height, mirroring how `navigation_screen.dart` wraps its compass in a `SafeArea`.

2. **Camera fits the trailhead, not the trail bounds, on `trail_detail_map_screen`** (kpd gap 5, FAIL): the prior fix wrongly assumed `trail.bounds` (`min/max_lat/lon`) was unpopulated on `GET /trail/:id` and switched `_fitInitialCamera` to prefer `gpx.getBounds()`. The user has corrected this: `min/max_lat/lon` ARE populated on all trails. Changing the bounds *source* never mattered — the REAL root cause is that `WandererMap` is the only map host in the codebase missing the documented `onMapCreated`/`onStyleLoaded` race buffer. When the native channel fires `onStyleLoaded` before `onMapCreated` (STATE.md lines 107 & 111 — "does not reliably fire onMapCreated before onStyleLoaded"; this exact race already silently no-opped `fitBounds` in `SearchMap` twice and forced a `_pendingStyle` buffer into both `SearchMap` and `navigation_screen`), `_controller` is still null, `_fitInitialCamera()` returns early, and the camera is left at `initCenter` (start point) + `initZoom: 18`. That is precisely the "focuses on the trailhead" symptom.

Purpose: Make the single-trail detail map reliably fit the whole trail on open, and put the list-map compass where the user expects it.
Output: Two focused edits — a race buffer + bounds-source revert in the shared `WandererMap`, and a compass reposition on `list_detail_map_screen`.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/quick/260710-kpd-fix-the-6-small-ui-gaps-that-i-found-aft/260710-kpd-SUMMARY.md

# Files to modify
@app/lib/components/base/wanderer_map.dart
@app/lib/routes/list_detail_map_screen.dart

# Read-only reference patterns (already-correct, mirror these — do NOT edit)
# - app/lib/components/base/search_map.dart: the canonical _pendingStyle race buffer
#   (onMapCreated replays a buffered onStyleLoaded; onStyleLoaded buffers if _controller == null)
# - app/lib/routes/navigation_screen.dart: same _pendingStyle buffer inlined into a MapLibreMap host,
#   AND the SafeArea-wrapped compass placement (the compass positioning reference the user cited)
# - app/lib/routes/trail_detail_map_screen.dart: _buildMapControls' expand button already
#   calls fitBounds(bounds: trail.bounds) successfully — proof trail.bounds is populated and the
#   ONLY reason the initial fit fails is the null-controller race, not the bounds source
</context>

<tasks>

<task type="auto">
  <name>Task 1 (LEM-02): Fix trail-detail initial camera fit — add the onMapCreated/onStyleLoaded race buffer and read the populated trail.bounds</name>
  <files>app/lib/components/base/wanderer_map.dart</files>
  <action>
Fix the real root cause of the trailhead-focus bug: `WandererMap` runs its `onStyleLoaded` work (`_fitInitialCamera` + `addTrailTrackLayers`) immediately, but `_fitInitialCamera` early-returns when `_controller` is null. Because the native platform channel does not reliably fire `onMapCreated` before `onStyleLoaded` (STATE.md decisions from Phase 16-03 and 17-01 — this race already forced a `_pendingStyle` buffer into `SearchMap` and `navigation_screen`), the fit silently no-ops and the camera is left at `initCenter` (start point) + `initZoom: 18`. Apply the exact same buffer this codebase already uses in `search_map.dart` and `navigation_screen.dart`.

Add a nullable `ml.StyleController? _pendingStyle;` field on `_WandererMapState`, alongside the existing `_controller`, with a doc comment matching `search_map.dart`'s (native channel may fire onStyleLoaded before onMapCreated).

Extract the current inline `onStyleLoaded` body into a private method `void _onStyleLoaded(ml.StyleController style)` that does exactly what the inline callback does today: call `_fitInitialCamera().ignore();` then, when `widget.showTrail && widget.trail.expand?.gpx != null`, call `addTrailTrackLayers(style, widget.trail).ignore();`. Preserve the existing `// 15-05:` comment.

Rewire the two `MapLibreMap` callbacks to buffer/replay, mirroring `search_map.dart`:
- `onMapCreated`: after `_controller = controller;` and the existing `widget.onMapCreated?.call(controller);`, read `final pending = _pendingStyle;` and if non-null, clear it (`_pendingStyle = null;`) and call `_onStyleLoaded(pending);`.
- `onStyleLoaded`: if `_controller == null`, set `_pendingStyle = style;` and `return;`; otherwise call `_onStyleLoaded(style);`.

Then revert the bounds source in `_fitInitialCamera` per the user's correction — `min/max_lat/lon` ARE populated on every trail, so do NOT rely on `gpx.getBounds()`:
- Replace `final gpxBounds = widget.trail.expand?.gpx?.getBounds();` and `final bounds = gpxBounds ?? widget.trail.bounds;` with a single `final bounds = widget.trail.bounds;`. Leave the existing `hasExtent` check, the `fitBounds(... nativeDuration: Duration(milliseconds: 1))` branch, and the `moveCamera` else-branch (degenerate no-extent fallback) untouched. Update/trim the now-stale gpx-bounds comment so it explains reading the record's `min/max`-based bounds instead.
- Remove the now-unused `import 'package:wanderer/util/gpx_util.dart';` (line 15) — `getBounds()` on line 246 is its only use in this file (`addTrailTrackLayers` pulls its own gpx extensions inside `trail_layer.dart`); leaving the import triggers an `unused_import` analyzer warning.

Do NOT change `initCenter`, `initZoom: 18`, the theme-swap `setStyle` path, or any marker/layer wiring — the fix is the race buffer plus the bounds-source revert only.
  </action>
  <verify>
    <automated>cd app && test $(flutter analyze 2>&1 | grep -c "error •") -eq 0 && grep -q "_pendingStyle" lib/components/base/wanderer_map.dart && grep -q "final bounds = widget.trail.bounds" lib/components/base/wanderer_map.dart && test $(grep -c "getBounds" lib/components/base/wanderer_map.dart) -eq 0 && test $(grep -c "util/gpx_util.dart" lib/components/base/wanderer_map.dart) -eq 0</automated>
  </verify>
  <done>`flutter analyze` reports zero `error •` lines; `wanderer_map.dart` has a `_pendingStyle` buffer replayed from `onMapCreated` and buffered in `onStyleLoaded`; `_fitInitialCamera` reads `widget.trail.bounds` (no `getBounds()` call remains); the `gpx_util.dart` import is gone.</done>
</task>

<task type="auto">
  <name>Task 2 (LEM-01): Reposition the list-map compass to align with the AppBar back button</name>
  <files>app/lib/routes/list_detail_map_screen.dart</files>
  <action>
Move the compass down so it sits at the same height as the AppBar back button instead of in the status-bar zone. The screen uses `extendBodyBehindAppBar: true`, and `ml.MapCompass` (per maplibre 0.3.5) renders a full-bleed `Container(alignment: Alignment.topRight, padding: EdgeInsets.all(10))` with NO `SafeArea` — so its default placement is ~10px from the very top of the screen, above the toolbar. `navigation_screen.dart` (the reference the user cited) wraps its compass in a `SafeArea`; do the same here.

In the `SearchMap` `children` list, replace the current `const ml.MapCompass(hideIfRotatedNorth: true),` entry with a `SafeArea`-wrapped compass that also overrides the compass padding so it lands at the back button's vertical center: wrap `ml.MapCompass(hideIfRotatedNorth: true, padding: const EdgeInsets.only(top: 6, right: 8))` in a `const SafeArea(child: ...)`. The `SafeArea` clears the status bar (matching where the AppBar toolbar begins), and `top: 6` centers the 44px compass (radius 22) on the 56px `kToolbarHeight` toolbar so it lines up with the `leading` back button; `right: 8` matches `navigation_screen`'s right inset. Keep `hideIfRotatedNorth: true` so it still hides when the map is north-up. Keep this entry `const` (SafeArea, MapCompass, and EdgeInsets.only all have const constructors). Leave the `MapScalebar`, `WandererAttribution`, markers, and every other child untouched.

No import changes are needed (`SafeArea` and `EdgeInsets` come from the already-imported `package:flutter/material.dart`).
  </action>
  <verify>
    <automated>cd app && test $(flutter analyze 2>&1 | grep -c "error •") -eq 0 && grep -q "SafeArea" lib/routes/list_detail_map_screen.dart && grep -q "MapCompass" lib/routes/list_detail_map_screen.dart</automated>
  </verify>
  <done>`flutter analyze` reports zero `error •` lines; `list_detail_map_screen.dart`'s compass is wrapped in a `SafeArea` with a top-offset padding, so it renders at back-button height rather than the top edge.</done>
</task>

</tasks>

<verification>
Automated (both tasks): `cd app && flutter analyze` — zero `error •` lines (the ~36 pre-existing info/warning issues carried since Phase 18 are unchanged and acceptable; only new *errors* fail the gate).

On-device confirmation (both fixes are visual/behavioral — verify on a physical Android device):
1. Open a trail detail screen and its full-screen map (`trail_detail_map_screen`): on first load the camera fits the WHOLE trail (with the elevation-profile bottom padding), not a tight zoom on the start point. Repeat a few times / on several trails — the fix removes an intermittent race, so it must be correct every open, not just sometimes.
2. Open `list_detail_map_screen`, rotate the map off-north: the compass appears at the top-right at the SAME height as the back button (not up near the status bar), matching `navigation_screen`'s compass placement.
</verification>

<success_criteria>
- `trail_detail_map_screen` (and `trail_detail_screen` via `trail_panel`, which share `WandererMap`) reliably fit the full trail bounds on open — the null-`_controller` race no longer skips the initial `fitBounds`.
- The initial fit reads the populated `trail.bounds` (`min/max_lat/lon`); no `gpx.getBounds()` dependency remains in `wanderer_map.dart`.
- `WandererMap` now uses the same `_pendingStyle` race buffer already present in `search_map.dart` and `navigation_screen.dart`.
- The `list_detail_map_screen` compass is `SafeArea`-wrapped and vertically aligned with the AppBar back button.
- `flutter analyze` introduces zero new errors (unused `gpx_util` import removed).
</success_criteria>

<output>
Create `.planning/quick/260710-lem-fix-2-issues-found-during-manual-verific/260710-lem-SUMMARY.md` when done.
</output>
