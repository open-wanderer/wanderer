---
phase: 14-coordinate-type-migration
plan: 01
subsystem: map-data-layer
tags: [maplibre, geobase, geographic, coordinate-migration, flutter]
provides:
  - Trail.bounds, GpxMappingUtils.allPoints/getBounds, and PolylineUtil encode/decode speak Geographic/LngLatBounds instead of latlong2.LatLng/flutter_map.LatLngBounds
  - Temporary Geographic<->LatLng adapter layer (map_coordinate_adapter.dart) isolating the type change from the 7 not-yet-migrated flutter_map call sites
tech-stack:
  added: []
  patterns: ["Boundary adapter functions at each un-migrated flutter_map call site, deleted individually as each screen migrates to MapLibre widgets (Phase 15+)", "Aliased import (`as ml`) instead of removing maplibre imports, to avoid ambiguous_import collisions with flutter_map's colliding symbol names (MapController, MapEvent, MapOptions, MarkerLayer)"]
key-files:
  created:
    - app/lib/util/map_coordinate_adapter.dart
  modified:
    - app/lib/models/trail.dart
    - app/lib/util/gpx_util.dart
    - app/lib/util/polyline_util.dart
    - app/lib/components/base/wanderer_map.dart
    - app/lib/components/map/trail_layer.dart
    - app/lib/routes/list_detail_map_screen.dart
    - app/lib/routes/list_detail_screen.dart
    - app/lib/routes/map_screen.dart
    - app/lib/routes/navigation_screen.dart
    - app/lib/routes/trail_detail_map_screen.dart
    - app/lib/provider/map_camera_provider.dart
    - app/lib/provider/navigation_provider.dart
    - app/lib/provider/trail/trail_polyline_provider.dart
    - app/lib/provider/trail/map_trail_search_provider.dart
    - app/test/util/gpx_util_test.dart
key-decisions:
  - "Executed manually outside /gsd-execute-phase — user renamed the data-layer types directly (Trail.bounds, GpxMappingUtils, PolylineUtil), then asked for a plan to fix the resulting ambiguous_import/type errors across the 7 flutter_map call sites. No CONTEXT.md/RESEARCH.md/PLAN.md were produced for this phase."
  - "Chosen fix (user-directed): alias maplibre imports (`as ml`) rather than delete them, and add a temporary bidirectional adapter file (map_coordinate_adapter.dart: toLatLng, toGeographic, toLatLngBounds, toLngLatBounds, toLatLngList) at every flutter_map boundary — per ROADMAP.md's own Phase 14 rationale (isolate the type change behind adapters; each screen deletes its own adapter as it migrates in Phase 15+)."
  - "foreground_position_stream_provider.dart required no changes — it only handles geolocator's Position and flutter_map_location_marker's LocationMarkerPosition, never LatLng/Geographic directly, so TYPE-01/02's 'no latlong2.LatLng survives in the data layer' criterion holds there trivially."
  - "Fixed 3 latent bugs surfaced by the type change, unrelated to import ambiguity: stale PolylineTools reference (renamed to PolylineUtil) in list_detail_map_screen.dart; invalid positional Geographic(0, 0) call and a List<Geographic>-vs-Polyline type mismatch in map_screen.dart."
  - "gpx_util_test.dart's buildNavShape fixtures used out-of-range synthetic lat/lon (e.g. 999.0) that geobase's Geographic constructor silently clamps/wraps (lat to [-90,90], lon to (-180,180]) — unlike the old LatLng, which stored raw doubles verbatim. Rescaled fixtures (index * 0.0001) to stay in valid range while remaining uniquely identifiable per index; algorithm itself was never buggy."
duration: n/a (interactive session, not timed agent execution)
completed: 2026-07-08
---

# Phase 14: Coordinate Type Migration Summary

**The data layer (`Trail.bounds`, GPX parsing, polyline encode/decode) now speaks MapLibre's `Geographic`/`LngLatBounds` instead of `latlong2`'s `LatLng`/`LatLngBounds`, with every still-`flutter_map`-rendered screen bridged through a temporary, self-documenting adapter layer.**

## Accomplishments
- `Trail.bounds`, `GpxMappingUtils.allPoints`/`getBounds()`, `buildNavShape`, and `PolylineUtil.encode`/`decode` all operate on `Geographic`/`LngLatBounds` — TYPE-01/02 satisfied in the data layer.
- Created `app/lib/util/map_coordinate_adapter.dart` — the single conversion surface (`toLatLng`, `toGeographic`, `toLatLngBounds`, `toLngLatBounds`, `toLatLngList`) every not-yet-migrated `flutter_map` call site uses, so each Phase 15+ screen migration deletes its own usage rather than reintroducing ad-hoc conversions.
- Resolved `ambiguous_import` across all 7 affected files (`wanderer_map.dart`, `trail_layer.dart`, `list_detail_map_screen.dart`, `list_detail_screen.dart`, `map_screen.dart`, `navigation_screen.dart`, `trail_detail_map_screen.dart`) by aliasing `maplibre` as `ml` rather than removing it — keeping both packages' types available without symbol collisions.
- `flutter analyze` and `flutter test` (gpx_util_test.dart) are clean; the app still builds and runs on today's `flutter_map` stack (success criterion 4).

## Decisions & Deviations
- Ran outside the standard GSD plan → execute → verify loop: the user made the data-layer renames directly, hit compile errors project-wide, and asked for a diagnosis + fix plan mid-session rather than through `/gsd-plan-phase 14`.
- No CONTEXT.md/RESEARCH.md exist for this phase; this SUMMARY.md is the only phase-level record.
- 3 additional pre-existing/latent bugs (unrelated to the import-ambiguity fix itself) were found and fixed as part of restoring a clean build — see key-decisions.

## Next Phase Readiness
Phase 15 can begin porting `wanderer_map.dart` and its screen callers to `MapLibreMap`. Each screen's Phase-15 migration should delete its own `map_coordinate_adapter.dart` usages as it moves off `flutter_map` — the adapter file itself should be deleted once Phase 17/18 removes the last `flutter_map` call site.
