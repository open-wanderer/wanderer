---
gsd_state_version: 1.0
milestone: v1.4
milestone_name: MapLibre Migration
status: "Phase 15 plan 15-02 complete (online style-JSON path); 15-01 Task 2 still blocked on Christian's physical-device airplane-mode PASS/FAIL for the file:// glyph spike (OFFL-04 risk gate)"
stopped_at: "Phase 15 plan 15-02 complete: JSON style assets + mapStyleJsonProvider shipped"
last_updated: "2026-07-09T11:00:00.000Z"
last_activity: 2026-07-09 -- Phase 15 Plan 02 complete (16a6b9a1, fc343987): style assets + mapStyleJsonProvider
progress:
  total_phases: 6
  completed_phases: 1
  total_plans: 7
  completed_plans: 4
  percent: 57
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-08)

**Core value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.
**Current focus:** Phase 15 — maplibre-core-trail-rendering-offline-parity

## Current Position

Phase: 15 (maplibre-core-trail-rendering-offline-parity) — EXECUTING
Plan: 2 of 6 complete — STYLE-01..04 shipped. 15-01 Task 2 (physical-device file:// glyph gate) still PENDING human verification.
Status: 15-02 complete (online style-JSON path); 15-01 offline glyph gate remains blocked on Christian's physical-device airplane-mode PASS/FAIL (OFFL-04 risk gate)
Last activity: 2026-07-09 -- Phase 15 Plan 02 complete (16a6b9a1, fc343987): JSON style assets + mapStyleJsonProvider

Progress: [██░░░░░░░░] 17%

## v1.4 Phases

- [ ] **Phase 13: Glyph & Sprite Endpoint** — GLYPH-01/02/03 (Go, `db/routes/`)
- [ ] **Phase 14: Coordinate Type Migration** — TYPE-01/02 (`Geographic`, `LngLatBounds`)
- [ ] **Phase 15: MapLibre Core, Trail Rendering & Offline Parity** — STYLE-01..04, GLYPH-04, CORE-01..04, TRAIL-01..05, OFFL-01..06
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

*Updated after each plan completion*

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

### Pending Todos

- None. Phase 13 and Phase 14 are both ready to plan (`/gsd-plan-phase 13` or `/gsd-plan-phase 14`).

### Blockers/Concerns

- **[Phase 15 — open unknown, blocks the milestone — SPIKE BUILT, AWAITING VERDICT]** It is unverified whether maplibre-native resolves `file://` glyph URLs for offline label rendering. OFFL-04 (a downloaded trail renders labels with no network) is the hard offline parity gate and nothing downstream is safe until it passes on a physical device in airplane mode. The throwaway `file://` spike (Plan 01 Task 1) is now BUILT and committed (`d713456b`): `SpikeGlyphFileScreen` + `seedSpikeGlyphCache()`, reachable via a debug-only FAB. **Awaiting Christian's physical-device airplane-mode PASS/FAIL** (resume signal: "PASS" or "FAIL: <native log>"). On FAIL, per D-03 the failure mode is returned for a fresh offline-label decision and the roadmap is revised — do NOT run plans 15-02..15-06 until PASS.
- **[Phase 15/16/17]** `maplibre` 0.3.5 is pre-1.0 with breaking changes across 0.x minors (three published upgrade guides). Pin the exact version on first add; CLEAN-03 locks it at the end.
- **[Phase 13]** The style references 3 fontstacks (`Noto Sans Regular`/`Medium`/`Italic`) but carries no `glyphs` key today, because `vector_tile_renderer` draws text from the bundled `assets/fonts/NotoSans`. Sprite icons (`arrow`, route shields) do not render at all today — wiring the sprite endpoint is a fix, not a regression.

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

Last session: 2026-07-09T08:51:55.029Z
Stopped at: Phase 15 plan 15-01 spike complete: A1 glyph PASS, A2 sprite FAIL (tracked for 15-06)
Resume file: .planning/phases/15-maplibre-core-trail-rendering-offline-parity/15-02-PLAN.md

## Operator Next Steps

- Run `/gsd-plan-phase 13` (glyph/sprite endpoint, Go) or `/gsd-plan-phase 14` (coordinate types) — they are independent and both gate Phase 15.
- Review the two roadmap corrections before planning: the requirement count was 33 in the source document but 40 by actual count, and **CORE-08 was added** to cover the two list screens that no requirement reached.
- Phase 15 is the milestone's risk gate. Its first plan must spike `file://` glyph resolution on a physical device in airplane mode before any further investment.
