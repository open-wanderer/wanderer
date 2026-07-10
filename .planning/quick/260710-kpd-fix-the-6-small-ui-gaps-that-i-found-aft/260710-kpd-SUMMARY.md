---
phase: quick-260710-kpd
plan: 01
subsystem: mobile-map
tags: [flutter, maplibre, ui-polish]
dependency-graph:
  requires: []
  provides:
    - WandererAttribution (collapsed-by-default attribution control)
    - addPolylineArrowLayer (native directional-arrow symbol layer helper)
  affects:
    - app/lib/components/base/wanderer_map.dart
    - app/lib/components/base/search_map.dart
    - app/lib/routes/list_detail_screen.dart
    - app/lib/routes/list_detail_map_screen.dart
    - app/lib/routes/map_screen.dart
    - app/lib/routes/navigation_screen.dart
tech-stack:
  added:
    - html (^0.15.6, direct pubspec dependency — was transitive via maplibre/flutter_html)
    - pointer_interceptor (^0.10.1, direct pubspec dependency — was transitive via maplibre)
  patterns:
    - Collapsed-by-default StatefulWidget attribution control, modeled on maplibre 0.3.5's SourceAttribution internals
    - GeoJSON FeatureCollection (jsonEncode, never string concatenation) for multi-line arrow layers
key-files:
  created:
    - app/lib/components/base/wanderer_attribution.dart
  modified:
    - app/lib/components/map/trail_layer.dart
    - app/lib/components/base/wanderer_map.dart
    - app/lib/components/base/search_map.dart
    - app/lib/routes/list_detail_screen.dart
    - app/lib/routes/list_detail_map_screen.dart
    - app/lib/routes/map_screen.dart
    - app/lib/routes/navigation_screen.dart
    - app/pubspec.yaml
    - app/pubspec.lock
decisions:
  - "Added html and pointer_interceptor as direct pubspec.yaml dependencies (previously transitive-only) since WandererAttribution imports them directly — avoids the depend_on_referenced_packages lint info and keeps the dependency graph honest."
  - "addPolylineArrowLayer filters lines to length >= 2 before building the FeatureCollection (mirrors addTrailTrackLayers' own `points.length < 2` guard) to avoid degenerate LineString geometry."
metrics:
  duration: ~15 min
  completed: 2026-07-10
---

# Quick Task 260710-kpd: Fix the 6 small UI gaps found after Phase 18 checkpoint Summary

Closed the six pre-existing MapLibre polish gaps recorded in Phase 18's on-device verification walk (`18-03-SUMMARY.md` "Gap Candidates") — none were regressions from the v1.4 migration, they were simply never wired on the list/single-trail surfaces.

## What Was Built

**Task 1 — Shared map-widget fixes** (`79260357`):
- `app/lib/components/base/wanderer_attribution.dart` (new): `WandererAttribution`, a collapsed-by-default drop-in for `ml.SourceAttribution`. Modeled on maplibre 0.3.5's `SourceAttribution` internals (pill layout, tappable HTML-link rendering via `TapGestureRecognizer` + `url_launcher`), with the expanded flag starting `false` and all camera-change auto-collapse logic removed (unnecessary once already collapsed).
- `app/lib/components/map/trail_layer.dart`: added `addPolylineArrowLayer(style, lines, {sourceId, layerId})` — adds a native directional-arrow symbol layer over already-drawn polylines, reusing the exact same `_kTrailArrowImageId` and `_kArrowSpacing` constants the single-trail `addTrailTrackLayers` already uses (visual parity, no id collision). Draws arrows only — the list screens keep drawing the route line itself via `ml.PolylineLayer`.
- `app/lib/components/base/wanderer_map.dart`: `_fitInitialCamera` now prefers `widget.trail.expand?.gpx?.getBounds()` over the record's `min/max`-based `trail.bounds` (fixes single-trail detail views where `min/max_lat/lon` are unpopulated `@Default(0)` on `GET /trail/:id`); controls `Column` switched from `CrossAxisAlignment.center` to `.end` so the right-aligned control stack doesn't shift when the compass appears/disappears; `ml.SourceAttribution()` replaced with `WandererAttribution()`.
- `app/lib/components/base/search_map.dart`: default `children` fallback now uses `WandererAttribution()` instead of `ml.SourceAttribution()`.

**Task 2 — Route-screen wiring** (`dcbabbd4`):
- `app/lib/routes/list_detail_screen.dart` (`_ListMap`): builds a `lines` list via `PolylineUtil.decode`, calls `addPolylineArrowLayer(style, lines)` in `onStyleLoaded` after the existing `fitBounds`; adds `const ml.MapScalebar()` and `const WandererAttribution()` to the `SearchMap` `children` (previously only markers, which suppressed `SearchMap`'s defaults).
- `app/lib/routes/list_detail_map_screen.dart`: same `lines` + `addPolylineArrowLayer` wiring; adds `const ml.MapCompass(hideIfRotatedNorth: true)`; replaces `ml.SourceAttribution()` with `WandererAttribution()`.
- `app/lib/routes/map_screen.dart` and `app/lib/routes/navigation_screen.dart`: replaced `ml.SourceAttribution()` with `WandererAttribution()`.

After these edits, `grep -rn "SourceAttribution(" lib` returns nothing — all six former call sites (2 base widgets + 4 route screens) now go through `WandererAttribution`.

**Gap 5/6 propagation (no extra edits needed):** `trail_detail_map_screen.dart` and `trail_panel.dart` (used by `trail_detail_screen`) both consume the shared `WandererMap`, so the GPX-bounds fit and right-aligned controls fixes apply to both automatically.

## Verification

- `flutter analyze`: 0 `error •` lines, 36 pre-existing info/warning issues unchanged (matches the plan's documented baseline).
- `grep -rn "ml.SourceAttribution(" app/lib`: no matches.
- `grep -q "addPolylineArrowLayer"` passes on both `list_detail_screen.dart` and `list_detail_map_screen.dart`.
- `grep -q "MapCompass"` passes on `list_detail_map_screen.dart`.
- `grep -q "WandererAttribution"` passes on `map_screen.dart` and `navigation_screen.dart`.
- `app/pubspec.yaml`/`pubspec.lock` updated cleanly via `flutter pub get` (2 dependencies changed: `html`, `pointer_interceptor` moved from transitive to direct).

On-device confirmation of the six visual/behavioral gaps (arrows on list polylines, compass on rotate, scale+attribution on list preview, collapsed-by-default attribution everywhere, full-bounds fit on trail detail, stable control alignment on rotate) was **not** performed as part of this quick task — it is a manual physical-device check per the plan's `<verification>` section, left to the user.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - missing critical functionality] Added `html` and `pointer_interceptor` as direct pubspec.yaml dependencies**
- **Found during:** Task 1, immediately after creating `wanderer_attribution.dart`
- **Issue:** The new file imports `package:html/dom.dart`, `package:html/parser.dart`, and `package:pointer_interceptor/pointer_interceptor.dart` directly, but both packages were only transitive dependencies (pulled in via `maplibre`/`flutter_html`), triggering `depend_on_referenced_packages` lint info diagnostics.
- **Fix:** Added `html: ^0.15.6` and `pointer_interceptor: ^0.10.1` to `app/pubspec.yaml` `dependencies`, ran `flutter pub get` to update `pubspec.lock`.
- **Files modified:** `app/pubspec.yaml`, `app/pubspec.lock`
- **Commit:** `79260357`

No other deviations — the remaining implementation followed the plan's action text exactly.

## Known Stubs

None — all six gaps are wired to real data (GPX-derived polylines/bounds, live style attributions); nothing renders placeholder/empty content as a result of this plan.

## Self-Check: PASSED

- FOUND: `app/lib/components/base/wanderer_attribution.dart`
- FOUND: commit `79260357`
- FOUND: commit `dcbabbd4`
- `flutter analyze`: 0 errors, 36 pre-existing issues (baseline match)
