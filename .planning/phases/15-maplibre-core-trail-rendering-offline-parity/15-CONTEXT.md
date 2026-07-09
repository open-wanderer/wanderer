# Phase 15: MapLibre Core, Trail Rendering & Offline Parity - Context

**Gathered:** 2026-07-08
**Status:** Ready for planning

<domain>
## Phase Boundary

`WandererMap` renders through `MapLibreMap` instead of `FlutterMap`. A hiker opening a trail — online, or downloaded with the device in airplane mode — sees basemap, place labels (all 4 fontstacks), icons (arrow + route shields via sprite), track, waypoints, and start/finish pins. This phase also carries the milestone's single highest-risk unknown (whether `file://` glyph URLs resolve on maplibre-native) as an explicit risk-gate spike that must pass before further phase investment.

Out of scope for this phase (deferred to Phase 16/17 per ROADMAP.md): `list_detail_map_screen`, `list_detail_screen`, `map_screen`, and `navigation_screen` keep rendering via `flutter_map` — those 4 screens are consumers of `WandererMap`/direct `FlutterMap` builders migrated in later phases, and continue relying on the `map_coordinate_adapter.dart` bridge from Phase 14 until their own migration.

</domain>

<decisions>
## Implementation Decisions

### Risk-gate spike (blocks all other Phase 15 work)
- **D-01:** The `file://` glyph resolution spike must be validated on a **physical device**, not just a simulator/emulator — matches ROADMAP.md's risk-gate wording; simulators can have different filesystem sandboxing behavior than real devices, which is exactly the false-positive this gate exists to prevent.
- **D-02:** The user (Christian) will run the physical-device spike build himself and report pass/fail plus any error output back — Claude cannot access a physical iOS/Android device directly.
- **D-03:** No fallback strategy is pre-decided for a spike failure. If maplibre-native rejects `file://` glyph URLs, stop and bring the actual failure mode back to the user for a fresh decision (e.g., a local loopback HTTP server serving cached files, or re-scoping OFFL-04) rather than assuming a fallback direction now.
- The first plan of this phase MUST be this throwaway spike, proving `file://` resolution against a hand-built minimal style, before any trail-rendering or download-caching work is planned (per ROADMAP.md's "Risk gate" note).

### Directional arrows (TRAIL-02)
- **D-04:** Re-enable the directional-arrow feature for real on a native MapLibre symbol layer — it is currently 100% dead code (`showArrows = false` in `trail_layer.dart`, no user has ever seen it), but TRAIL-02 is in scope and this phase should actually ship it rather than leave it dark again.
- **D-05:** Simplify the animation: render **static** arrows at fixed intervals along the line — no crawling/pulsing animation loop. This is a deliberate simplification versus the old dead implementation's continuous `AnimationController`-driven animation; the spacing-by-zoom logic (denser at higher zoom) should still be replicated, just without motion.

### Attribution & scale bar (CORE-04)
- **D-06:** Use maplibre's default/built-in `AttributionButton` control as-is — no custom-styled attribution UI. This is the first time the app has shown ANY attribution (ODbL obligation not met today).
- **D-07:** Scale bar bottom-left, attribution bottom-right (or the maplibre-default equivalent corner placement) — standard map-app convention, kept out of the way of existing overlays (compass top-right on some screens, bottom sheets, elevation profile panel).

### Glyph/sprite caching scope & timing (GLYPH-04, OFFL-01/02)
- **D-08:** One shared **app-wide** cache — not a per-trail pruned subset. Cache the full set (all 4 fontstacks, complete 256-range set per fontstack, both light/dark sprite themes) once; every trail download and every map open after that reads from the same cache. Matches GLYPH-04's "app-wide, not per-trail" wording literally; second-trail-download re-fetch avoidance (OFFL-01) falls out of this for free.
- **D-09:** The fetch is lazy — first triggered on **first map open** (mirrors how `mapStyleSourcesProvider`/the old `tileUrlProvider` already work: no work at app startup for users who never open a map).
- **D-10:** Trail download is a **second, independent trigger** for the same cache-warm fetch — if a hiker downloads a trail for offline use before ever opening a map screen, the download must still populate the shared cache (otherwise OFFL-01 would have a gap for that ordering). Whichever of {first map open, first trail download} happens first performs the fetch; the other becomes a no-op against the already-warm cache.

### Claude's Discretion
- Exact style-JSON extraction mechanism for STYLE-01 (dumping the 7,677-line `wandererLightTheme`/`DarkTheme` Dart `Map` literals to `.json` assets) — not discussed; left to the planner/researcher to determine the safest extraction approach (programmatic dump vs. manual port) given the forks aren't deleted until Phase 18.
- Exact storage location/format for the app-wide glyph/sprite cache on disk (ObjectBox vs. plain files in the app documents directory) — not discussed.
- `CORE-01`'s exact widget-contract preservation details for `WandererMap` (which existing params/callbacks survive verbatim vs. need signature changes) — left to planning, informed by the Phase-14 `map_coordinate_adapter.dart` boundary already in place at the 4 not-yet-migrated screens.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap & requirements
- `.planning/ROADMAP.md` §"Phase 15: MapLibre Core, Trail Rendering & Offline Parity" — goal, success criteria, and the explicit risk-gate note mandating the file:// spike as the first plan
- `.planning/REQUIREMENTS.md` §"Style & Glyph Serving", §"Offline", §"Map Core", §"Trail Rendering" — STYLE-01..04, GLYPH-04, CORE-01..04, TRAIL-01..05, OFFL-01..06
- `.planning/PROJECT.md` — milestone goal, target features, and the "Context" section's map-surface file inventory (14 files / ~3,850 lines touching maps)

### Prior phase work this phase builds on
- `.planning/phases/13-glyph-sprite-endpoint/13-01-SUMMARY.md` — the actual shipped `/api/v1/map/style-sources` endpoint (not `/api/v1/map/config` as originally planned) returning `{tileUrl, glyphUrl, spriteUrl}`; `MapStyleSources` model + `mapStyleSourcesProvider` in `app/lib/provider/map_style_sources_provider.dart` is the existing consumption point for GLYPH-04 to build on
- `.planning/phases/14-coordinate-type-migration/14-01-SUMMARY.md` — `Geographic`/`LngLatBounds` now used in the data layer; `app/lib/util/map_coordinate_adapter.dart` is the temporary bridge at every still-`flutter_map` call site — this phase's `WandererMap` migration should DELETE its own adapter usages as it moves off `flutter_map`, per that summary's "Next Phase Readiness" note

### Source of truth for what to port
- `wandererLightTheme(tileUrl)` / `wandererDarkTheme(tileUrl)` in the pinned `flomp/dart-vector-tile-renderer` fork (referenced via `vector_tile_renderer` package import in `app/lib/provider/map_style_provider.dart`) — the 7,677-line Style Spec v8 documents STYLE-01 extracts to `.json`
- `app/lib/components/base/wanderer_map.dart`, `app/lib/components/map/trail_layer.dart` — current `flutter_map`-based implementations being ported
- `app/lib/routes/trail_detail_map_screen.dart` — the sole current consumer of `WandererMap` (per Phase 14's exploration, `list_detail_map_screen`/`list_detail_screen`/`map_screen`/`navigation_screen` build their own `FlutterMap` directly and are NOT `WandererMap` consumers — they stay on `flutter_map` until Phase 16/17)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `app/lib/util/map_coordinate_adapter.dart` (Phase 14) — `toLatLng`/`toGeographic`/`toLatLngBounds`/`toLngLatBounds`/`toLatLngList` conversions; no longer needed at any call site this phase fully migrates to `MapLibreMap`, but still needed at the 4 screens staying on `flutter_map`.
- `app/lib/provider/map_style_sources_provider.dart` (Phase 13) — `mapStyleSourcesProvider` already fetches `{tileUrl, glyphUrl, spriteUrl}`; GLYPH-04's app-wide caching layer builds on top of this rather than re-fetching config independently.

### Established Patterns
- `@Riverpod(keepAlive: true)` providers for server-config-derived, app-wide-cached values (see `map_style_sources_provider.dart`, `map_camera_provider.dart`) — the glyph/sprite cache provider should likely follow this same shape.
- Existing offline trail-download flow already writes to the app documents directory and ObjectBox for `.pmtiles` cells (Phase 4/5 v1.1 work) — the glyph/sprite cache should follow the same storage convention rather than inventing a new one.

### Integration Points
- `WandererMap`'s only current consumer is `trail_detail_map_screen.dart` — the migration's blast radius for CORE-01 is contained to this one screen plus `TrailLayer`.
- `foreground_position_stream_provider.dart` (used for `CurrentLocationLayer` today) will need a maplibre-native equivalent wiring in `WandererMap`, though the location-puck-specific `enableLocation`/`trackLocation` API (CORE-07) is explicitly Phase 17 scope, not this phase's — confirm during planning whether Phase 15's `WandererMap` needs an interim location-display approach or can defer it.

</code_context>

<specifics>
## Specific Ideas

None beyond the decisions above.

</specifics>

<deferred>
## Deferred Ideas

- Per-trail pruned glyph caching (considered and rejected in favor of one shared app-wide cache — D-08).
- Custom-styled attribution control (considered and rejected in favor of maplibre's default — D-06).
- Continuous/animated directional-arrow motion (considered and rejected in favor of static arrows — D-05).
- Pre-deciding a `file://` spike-failure fallback (explicitly deferred to whenever/if the spike actually fails — D-03).

</deferred>

---

*Phase: 15-maplibre-core-trail-rendering-offline-parity*
*Context gathered: 2026-07-08*
