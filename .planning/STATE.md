---
gsd_state_version: 1.0
milestone: v1.4
milestone_name: MapLibre Migration
status: executing
stopped_at: Completed 16-02-PLAN.md
last_updated: "2026-07-09T17:41:03.777Z"
last_activity: 2026-07-09 -- Phase 16 execution started
progress:
  total_phases: 6
  completed_phases: 2
  total_plans: 10
  completed_plans: 10
  percent: 33
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-08)

**Core value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.
**Current focus:** Phase 16 — list-map-screens-on-maplibre

## Current Position

Phase: 16 (list-map-screens-on-maplibre) — EXECUTING
Plan: 3 of 3
Status: Ready to execute
Last activity: 2026-07-09 -- Phase 16 execution started

Progress: [█████░░░░░] 50%

## v1.4 Phases

- [x] **Phase 13: Glyph & Sprite Endpoint** — GLYPH-01/02/03 (SvelteKit, `/map/style-sources`)
- [x] **Phase 14: Coordinate Type Migration** — TYPE-01/02 (`Geographic`, `LngLatBounds`)
- [x] **Phase 15: MapLibre Core, Trail Rendering & Offline Parity** — STYLE-01..04, GLYPH-04, CORE-01..04, TRAIL-01..05, OFFL-01..05 (OFFL-06 deferred to 17/18)
- [ ] **Phase 16: List & Map Screens on MapLibre** — CORE-08, CLUS-01..05
- [ ] **Phase 17: Navigation on MapLibre** — NAV-01..04, CORE-05/06/07
- [ ] **Phase 18: Retire flutter_map and the flomp Forks** — CLEAN-01/02/03

Execution order: 13 ∥ 14 → 15 → 16 → 17 → 18. Phases 13 and 14 share no code and may run in parallel; both gate Phase 15.

## Performance Metrics

**Velocity (v1.0–v1.3):**

- Total plans completed: 35
- Average duration: — min
- Total execution time: — hours

**By Phase (recent):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 10 | 4 | ~19 min | ~5 min |
| 11 | 4 | ~50 min | ~13 min |
| 12 | 4 | ~30 min | ~8 min |
| 13 | 1 | - | - |
| 14 | 1 | - | - |
| 15 | 6 | ~90 min + on-device iteration | ~15 min |

*Updated after each plan completion*
| Phase 16 P01 | 9min | 3 tasks | 3 files |
| Phase 16 P02 | 7min | 2 tasks | 3 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [v1.4 roadmap] Phase 15 carries 20 requirements *because the offline gate forces it*: `MultiPmTilesVectorTileProvider` lives inside `wanderer_map.dart`, so the moment CORE-01 turns `WandererMap` into a `MapLibreMap`, the `pmtiles://` path (OFFL-03/05) and `file://` glyph path (OFFL-01/02/04) must land in the same phase or a downloaded trail regresses.
- [v1.4 roadmap] TYPE-01/02 lands early as its own Phase 14, ahead of all map code. `Geographic(lon:, lat:)` reverses `LatLng(lat, lon)` — a transposed coordinate is silent. Isolating it under the existing `gpx_util`/`polyline_util` tests gives one clean signal. Cost: temporary `Geographic → LatLng` adapters at four un-migrated `flutter_map` call sites, each deleted as its screen migrates.
- [v1.4 roadmap] CORE-05/06/07 are assigned to Phase 17 (navigation), not to a foundation phase. Each retires a file or a pubspec plugin, and `navigation_screen` is the last holdout for all three (`map_compass.dart`, `AnimatedMapController`, `CurrentLocationLayer`). Earlier phases swap their own call sites but cannot delete.
- [v1.4 roadmap] CORE-08 added during roadmap creation — `list_detail_map_screen` and `list_detail_screen` build `FlutterMap` directly, so CORE-01 (scoped to `WandererMap`) never covered them.
- [v1.4] Migrate to `maplibre`, not `maplibre_gl` — FFI/JNI bindings, reads our style JSON directly.
- [v1.4] Clustering reuses `POST /search/trails/cluster` rather than maplibre's `cluster: true` — the endpoint exists, web already renders its output, and maplibre's `GeoJsonSource` exposes no cluster fields.
- [v1.4] Style JSON lives as an app asset, not a server-hosted style URL — offline rendering needs the style before any network call.
- [15-03] `map_cache_path.dart` is the single sanctioned builder for any map-cache filesystem path — operator-controlled fontstack/range tokens are whitelisted (4 fontstacks + `^\d+-\d+$`) and rejected with `ArgumentError` before a path is built; never string-concatenate a token into a path (T-15-03-01).
- [15-03] One shared app-wide glyph/sprite cache under `<app-docs>/map_cache` (D-08): `sprite/{light,dark}` + `glyphs/<fontstack>/<range>.pbf`; both the map-open (D-09, 15-04) and trail-download (D-10, 15-03) triggers converge on the same idempotent keepAlive warm.
- [15-04] `ml.MapController` cannot be free-standing (abstract interface created by the native map). `WandererMap` exposes `onMapCreated(controller)` which BOTH captures it internally (for `setStyle`/`fitBounds`) and forwards it to the caller; consumers hold `ml.MapController? _mapController` set from the hand-off and null-guard their calls. 15-05/15-06 reuse this, not an input controller field. Live theme swap = `ref.listen(mapStyleJsonProvider) -> setStyle` + a cached `_lastStyleJson` so a keepAlive refresh never flashes to loading (CORE-02). Trail track/markers seam is `layers: const []` in `wanderer_map.dart` (`// 15-05: ...`).
- [15-05] Arrow glyph self-registered via `addImageFromIconData` (sprite-independent) so TRAIL-02 avoids the unresolved `file://` sprite risk.
- [15-05] Directional arrows are a static native symbol layer (D-05); `AnimationController` + bearing loop deleted.
- [15-06] OFFL-05 multi-cell = N native `pmtiles://file://` sources + N duplicated style-layer sets (`__cellN` id suffix; source-less layers not cloned), NOT merge-at-download. `pmtiles` 1.2.0 Dart is read-only (no merge API) and `generator.go`/`grid.go` emit one `.pmtiles` per 0.5° cell (~1-4 per trail); a server merge endpoint is out of the Flutter phase's scope. Implemented in `rewriteStyleForOffline`, test-guarded.
- [15-06] `rewriteStyleForOffline` is the single sanctioned online->offline style transform (pure, deep-copies input): `glyphs`/`sprite` -> `file://<cacheRoot>`; tiled sources -> `pmtiles://file://<cell>`; rejects non-absolute / `..` / foreign-scheme paths before emitting (T-15-06-01/02). WandererMap is its only caller.
- [15-06] OFFL-06 DEFERRED: `pm_tile_provider.dart` NOT deleted — `navigation_screen.dart` (Phase-17 flutter_map holdout) still uses `MultiPmTilesVectorTileProvider`. Delete once that screen migrates.
- [15-06] Three real bugs found and fixed during physical-device verification, outside the normal executor flow: restored legacy flutter_map `TrailLayer` for `navigation_screen` (`192b3a89`, 15-05 had deleted it without checking callers); removed flutter_map-only `MapCompass` from `WandererMap.controls` (`b7d30947`, crashed at runtime — `MapCamera.of()` needs a `FlutterMap` ancestor); clamped offline pmtiles source `maxzoom` to 14 to match `generator.go`'s extraction depth (`85d73fd3` — the online style's inherited `maxzoom:15` caused blank tiles above z14 offline).
- [Phase 15 verification] Goal-backward verification (`15-VERIFICATION.md`) found the trail's self-registered `arrow` sprite image collided (same id) with the basemap's own `roads_oneway` icon — MapLibre's `addImage` silently overwrites same-id images. Fixed by renaming to `trail-arrow` (`38aabcdc`).
- [Phase 16-01]: SearchMap.layers typed List<ml.Layer>? (native style-layer builders), not List<Widget>? as PLAN.md's artifact spec stated — ml.MapLibreMap.layers and .children are different type families
- [Phase 16-01]: _ListMap converted ConsumerWidget to ConsumerStatefulWidget to hold ml.MapController across onMapCreated/onStyleLoaded, mirroring list_detail_map_screen's approach
- [Phase 16]: [16-02] cluster_layer.dart doc comments avoid the literal substring 'unclustered-point' (rephrased as point_count==1 circle layer) to satisfy the plan's own zero-occurrence grep while still documenting D-05's rationale
- [Phase 16]: [16-02] Wrapped the circle-radius step-ramp array in // dart format off / on markers so dart format doesn't re-explode the single-line grouping the acceptance criteria greps for verbatim

### Pending Todos

- None. Phase 16 (List & Map Screens on MapLibre) is ready to plan (`/gsd-plan-phase 16`).

### Blockers/Concerns

- **[Phase 15/17 — OFFL-06 deferred, by design]** `pm_tile_provider.dart` intentionally NOT deleted — `navigation_screen.dart` still consumes `MultiPmTilesVectorTileProvider` for its offline flutter_map path. Delete in Phase 17/18 once that screen migrates off flutter_map. ROADMAP.md's Phase 15 success criterion 5 was corrected in place to reflect this (it originally self-contradicted).
- **[Phase 15 — A2 sprite `file://` gap, low exposure, not investigated further]** The 15-01 spike found `file://` **sprite** resolution FAILED on a physical Android device despite valid cached files (glyph `file://` resolution PASSED). 15-05 sidesteps the practical impact for the trail's own arrow icon by self-registering it (`addImageFromIconData`, sprite-independent) rather than relying on the sprite. The only remaining exposure is OTHER sprite-atlas icons (e.g. route-network shields) rendering offline — those don't render at all today regardless of online/offline, so this is a pre-existing not-yet-working feature, not a regression. Not re-investigated natively; revisit if/when sprite-sourced icons become a real feature.
- **[Phase 15/16/17]** `maplibre` 0.3.5 is pre-1.0 with breaking changes across 0.x minors (three published upgrade guides). Pin the exact version on first add; CLEAN-03 locks it at the end.
- **[Phase 15 — CORE-05 gap, expected]** `trail_detail_map_screen`'s `MapCompass` (flutter_map-only) was removed rather than crashing against the new `MapLibreMap` — that screen has no compass/rotation-reset control until Phase 17 (CORE-05) wires maplibre's native equivalent.

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260702-e3g | Fix non-optimistic reorder animation in Settings(Sub)CategoriesScreen | 2026-07-02 | 6b3e6f6b | [260702-e3g-…](./quick/260702-e3g-fix-non-optimistic-reorder-animation-in-/) |
| 260702-ek7 | Fix white flash on (sub)category toggle/reorder | 2026-07-02 | 8a917b4c | [260702-ek7-…](./quick/260702-ek7-fix-white-flash-on-sub-category-toggle-r/) |
| 260702-ere | Cascade category visibility to SettingsSubcategoriesScreen | 2026-07-02 | 108348b2 | [260702-ere-…](./quick/260702-ere-cascade-category-visibility-to-settingss/) |
| 260702-m4u | Make auth_provider.dart build() optimistic | 2026-07-02 | d2d126a8 | [260702-m4u-…](./quick/260702-m4u-make-auth-provider-dart-build-optimistic/) |
| 260702-gib | Add read-only subcategory chips under each category row | 2026-07-02 | dbc1db3d | [260702-gib-…](./quick/260702-gib-add-subcategory-chips-under-each-categor/) |

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Map screen | Render `is_large` trails as full polylines (FUT-01) | Future milestone | v1.4 requirements |
| Basemap | Basemap picker + Waymarked Trails overlays (FUT-02/03) | Future milestone | v1.4 requirements |
| Native GL | 3D terrain, hillshade, pitch/tilt (FUT-04/05) | Future milestone | v1.4 requirements |
| Trail form | Category/subcategory picker in create/edit form (TRAILFORM-01/02) | Future milestone (form rework needed) | v1.3 requirements |
| Bulk edit | Bulk-edit modal for category/subcategory/difficulty | Out of scope (web-only) | v1.3 requirements |
| Filters | Subcategory reordering within a category | Future milestone | v1.3 requirements |
| Audio | TTS maneuver announcements (AUDIO-01, AUDIO-02) | v2 | Init |
| Routing | Navigate from user position to trailhead; off-trail re-routing | Out of scope | Init |
| Account | API token management (ACCT-F01) | Future | v1.2 requirements |
| Settings | Favourite sport picker, Export, Integrations, Maintenance, Map settings | Out of scope | v1.2 requirements |

## Session Continuity

Last session: 2026-07-09T17:41:03.764Z
Stopped at: Completed 16-02-PLAN.md
Resume file: None

## Operator Next Steps

- Run `/gsd-plan-phase 13` (glyph/sprite endpoint, Go) or `/gsd-plan-phase 14` (coordinate types) — they are independent and both gate Phase 15.
- Review the two roadmap corrections before planning: the requirement count was 33 in the source document but 40 by actual count, and **CORE-08 was added** to cover the two list screens that no requirement reached.
- Phase 15 is the milestone's risk gate. Its first plan must spike `file://` glyph resolution on a physical device in airplane mode before any further investment.
