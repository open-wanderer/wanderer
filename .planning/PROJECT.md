# Wanderer Trail Navigation

## Current Milestone: v1.4 MapLibre Migration

**Goal:** Replace the Flutter app's `flutter_map` + forked `vector_map_tiles`/`vector_tile_renderer` stack with the `maplibre` package, retiring both forks and moving map rendering onto native GL.

**Target features:**
- `maplibre` renders all 8 map screens; `flutter_map` and its 4 plugins removed from `pubspec.yaml`
- Both `flomp/*` forks retired; `wandererLight/DarkTheme` extracted from Dart into plain `.json` style assets in the app
- Self-hosted glyph + sprite endpoint on the Wanderer server (Noto Sans Regular/Medium/Italic), operator-controlled like `TILE_SERVER_URL`
- Glyphs downloaded once, cached to the app documents dir, and rewired to `file://` so offline trails keep their labels
- Map clustering reads `POST /search/trails/cluster` and renders as native circle/symbol layers, matching web's `ClusterLayer`
- Offline `.pmtiles` archives load via native `pmtiles://` — vendored `pm_tile_provider.dart` deleted
- `latlong2.LatLng` → `Geographic` across `trail.dart`, `gpx_util.dart`, `polyline_util.dart`

## What This Is

Turn-by-turn trail navigation, full settings management, and category-aware trail discovery for the Wanderer Flutter mobile app. Users launch navigation from a trail's detail or map screen and get Valhalla-powered maneuver instructions, a live map centered on their position, and a stats sheet tracking distance, elevation, and speed. The app includes a complete settings suite (language, units, privacy, account, notifications, categories) and subcategory-aware trail filters.

## Core Value

A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.

## Requirements

### Validated

- [x] Navigate button on trail_detail_screen and trail_detail_map_screen (v1.0 — Phase 2)
- [x] Full-screen navigation screen with map centered on current GPS position (v1.0 — Phase 2)
- [x] Valhalla turn-by-turn maneuver instructions shown at the top of the screen (v1.0 — Phase 2)
- [x] North-up / heading-up map orientation toggle button (v1.0 — Phase 2)
- [x] DraggableScrollableSheet with live stats: distance, elevation gain/loss, speed (v1.0 — Phase 3)
- [x] New SvelteKit API endpoint POST /api/v1/valhalla/navigate (v1.0 — Phase 1)
- [x] Progress tracking: advance through maneuvers as user moves along the trail (v1.0 — Phase 2)
- [x] Offline navigation fallback: serve maneuvers from ObjectBox cache when network is unavailable (v1.1 — Phase 5)
- [x] Silent cache write at trail download time and re-cache after online sessions (v1.1 — Phase 5)
- [x] SettingsScreen: Privacy, Language, Notifications list entries with sub-routes (v1.2 — Phase 6)
- [x] SettingsLanguageScreen: 14-locale language picker + metric/imperial unit toggle, persists and auto-saves (v1.2 — Phase 6)
- [x] Unit toggle propagates to all format_util call sites app-wide (v1.2 — Phase 6)
- [x] SettingsPrivacyScreen: account, trails, and lists visibility radio groups, auto-save (v1.2 — Phase 7)
- [x] SettingsAccountScreen: avatar upload, bio editor, email change, password change, account deletion with confirmation (v1.2 — Phase 8)
- [x] SettingsNotificationsScreen: 9 notification types × web + email toggles, auto-save (v1.2 — Phase 9)
- [x] Category model updated with `icon`, `short_name`, `translations`; locale-aware display name resolution (v1.3 — Phase 10)
- [x] Subcategory freezed model, ObjectBox entity, and provider fetching from `/subcategory` (v1.3 — Phase 10)
- [x] Category and subcategory preference Riverpod providers backed by `/user-category-preference` and `/user-subcategory-preference` (v1.3 — Phase 10)
- [x] `Settings.category` field removed (replaced by priority-based category preferences) (v1.3 — Phase 10)
- [x] `TrailFilter` gains `subcategory` list; TrailFilterScreen and quick filter bar show subcategory chips (v1.3 — Phase 11)
- [x] New SettingsCategoriesScreen: per-category and per-subcategory visibility toggles + priority reordering (v1.3 — Phase 12)

### Active

- [ ] Wanderer style JSONs (light + dark) live as plain app assets; both `flomp/*` forks removed from `dependency_overrides`
- [ ] Wanderer server serves glyph PBFs and a sprite sheet; app resolves them like it resolves `TILE_SERVER_URL`
- [ ] Trail download fetches glyphs + sprite once, caches them, and rewrites the style to `file://` for offline label rendering
- [ ] All 8 map screens render via `MapLibreMap`; `flutter_map` + 4 plugins gone from `pubspec.yaml`
- [ ] Map screen clustering reads `POST /search/trails/cluster` and renders native circle/symbol layers
- [ ] Offline trails render from `.pmtiles` via native `pmtiles://`; `pm_tile_provider.dart` deleted
- [ ] Navigation screen keeps heading-up follow mode, compass reset, and live location puck on maplibre

### Out of Scope

- Web frontend changes — `web/` already runs maplibre-gl-js; this milestone is app-only apart from the new glyph/sprite endpoint
- Switching offline tile generation to OpenMapTiles schema — would invalidate every downloaded trail archive and every operator's tile cache
- Basemap picker in app settings (OpenTopoMap / CyclOSM / Carto, as web offers) — deferred; v1.4 ships the Protomaps wanderer style only
- Contributing `cluster` fields upstream to maplibre's `GeoJsonSource` — unnecessary once clustering is server-side
- Linux/Windows/macOS map support — `maplibre` has no Linux backend and the app is mobile-only
- 3D terrain, hillshade, globe projection — newly possible on native GL, but not this milestone
- Text-to-speech maneuver announcements — deferred to v2 (audio infra adds complexity)
- Routing from user's current position to the trailhead — assume user is already at the trail start
- Re-routing if user goes off-trail
- API token management (ACCT-F01) — mobile clients don't need API tokens; web-only feature
- Favourite sport picker, Export, Integrations (Strava/Komoot), Maintenance, Map settings — out of scope for mobile settings v1
- Drag-to-reorder for category priority (use ReorderableListView or up/down buttons instead of pointer-drag like web)
- Category/subcategory picker in trail create/edit form — deferred to a later milestone (form rework needed)
- Bulk-edit modal — web-only feature, out of scope for mobile

## Context

- Flutter mobile app with Riverpod state management, go_router navigation, flutter_map for maps, geolocator for GPS
- v1.4 map surface: 14 files / ~3,850 lines touch maps. `wanderer_map.dart`, `trail_layer.dart`, `map_compass.dart`, `map_screen.dart`, `navigation_screen.dart`, `trail_detail_map_screen.dart`, `list_detail_map_screen.dart`, `list_detail_screen.dart`, `map_style_provider.dart`, `vendor/vector_map_tiles/pm_tile_provider.dart`, plus type spill into `trail.dart`, `gpx_util.dart`, `polyline_util.dart`, `foreground_position_stream_provider.dart`
- The `flomp/dart-vector-tile-renderer` fork carries 11 commits over upstream: the two wanderer themes, plus `format`/`zoom`/`in` expression-parser support needed only to *parse* those themes. `flomp/flutter-vector-map-tiles` is the second fork. Both exist solely to render a style JSON that maplibre reads natively
- `wandererDarkTheme(tileUrl)` / `wandererLightTheme(tileUrl)` are already valid MapLibre Style Spec v8 documents (7,677 lines each, Protomaps vector source, 14 symbol layers) — expressed as Dart `Map<String, dynamic>` rather than `.json`. They carry no `glyphs` and no `sprite` key, because `vector_tile_renderer` draws text from the bundled `assets/fonts/NotoSans` asset instead
- Fontstacks referenced by the style: `Noto Sans Regular`, `Noto Sans Medium`, `Noto Sans Italic`. Sprite icons referenced: `arrow` plus route-network shield icons — these do not render today (no `sprite` key), so wiring a sprite endpoint is a fix, not a regression
- Offline `.pmtiles` cells are extracted from `build.protomaps.com` by `db/services/tiles/generator.go` → **Protomaps schema**. Web's `static/styles/ofm.json` is **OpenMapTiles schema** via openfreemap. The two are not interchangeable; this is why the app keeps its own style rather than adopting web's
- `POST /api/v1/search/trails/cluster` already runs Supercluster server-side, honors category-preference filters, splits `is_large` trails out for polyline rendering, and returns `point_count` / `point_count_abbreviated` on each feature. Web's `cluster-layer.ts` renders exactly this and sets no `cluster: true`. The app reaches the same base URL, so this endpoint is callable today
- `maplibre` 0.3.5 is pre-1.0 with breaking changes across 0.x minors (three published upgrade guides). `GeoJsonSource` exposes no cluster fields (`// TODO add more fields`), but `initStyle` accepts raw JSON and `updateGeoJsonSource(id:, data:)` resolves sources by id regardless of how they were declared
- `Geographic(lon:, lat:)` reverses the argument order relative to `LatLng(lat, lon)` — a silent-swap footgun during the type migration
- Shipped v1.0: Navigation screen (SvelteKit Valhalla endpoint + Flutter screen + stats sheet)
- Shipped v1.1: Offline navigation (ObjectBox cache, DioException fallback, offline indicator)
- Shipped v1.2: Full settings suite (Language/Units, Privacy, Account/Profile, Notifications)
- Web PR #1059 merged: new category model with translations/icon/short_name, subcategories, user category/subcategory preferences, favourite sport replaced by priority-based category ordering
- Settings infrastructure: `Settings` freezed model, `settingsProvider` with `saveToServer()` — reused by all four screens
- `localeProvider` and `unitProvider` derived from `settingsProvider` — live-switch locale and unit system app-wide
- 14 supported locales with ARB files; AppLocalizations regenerated in Phase 6
- Background navigation via tracelet package (from quick tasks)
- Along-track projection for waypoint advancement (from quick tasks)

## Constraints

- **Tech Stack**: Flutter + Riverpod (riverpod_annotation codegen) + go_router + freezed — must follow existing patterns. v1.4 replaces `flutter_map` with `maplibre` as the map renderer
- **API**: SvelteKit app proxies Valhalla at POST /api/v1/valhalla/navigate; Flutter calls via Dio
- **Online-only navigation v1**: Navigation requires network; falls back to ObjectBox cache when offline
- **No breaking changes**: Existing trail detail screens, bottom nav, and routes must be unaffected
- **Offline parity is a hard gate**: a downloaded trail must render basemap *and labels* with no network. Any phase that regresses this is not done
- **Both map stacks coexist during v1.4**: `flutter_map` stays in `pubspec.yaml` until the last screen is migrated. Every phase must leave the app building and runnable
- **Tile source stays Protomaps**: offline cells are Protomaps-schema, so the app's style targets Protomaps. `TILE_SERVER_URL` / `PROTOMAPS_API_KEY` remain the operator's control surface
- **maplibre pinned**: pre-1.0 with breaking 0.x minors — pin an exact version, do not float

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Extend SvelteKit Valhalla API rather than calling Valhalla directly from Flutter | Keeps credentials server-side, consistent with existing route endpoint pattern | ✓ Good — used in v1.0 through v1.2 |
| DraggableScrollableSheet for stats, not fixed bottom bar | Matches MapScreen pattern; user can expand for more detail without blocking map | ✓ Good |
| Button-driven PageView for stats (not horizontal swipe) | Locked during Phase 3 context; horizontal swipe conflicts with map pan gesture | ✓ Good |
| TTS deferred to v2 | Adds Flutter TTS dependency and audio session handling — not needed for navigation core | — Pending |
| Assume user is at trailhead | Simplifies v1 scope; off-trail routing is a separate problem | — Pending |
| Dio try-catch as sole offline gate (not connectivity packages) | Connectivity packages have false-positive/negative failure modes (captive portals, VPN) | ✓ Good |
| `String? navCacheJson` on TrailEntity (not typed List) | ObjectBox doesn't support List<List<double>> or List<NavigateManeuver> natively; follows `gpxData` precedent | ✓ Good |
| freezed 3.x: @JsonSerializable(explicitToJson: true) on factory constructor | Class-level placement breaks json_serializable codegen in freezed 3.x | ✓ Good — critical for future freezed models |
| `Settings` freezed model + `settingsProvider.saveToServer()` shared across all v1.2 screens | No new persistence layer needed; single source of truth | ✓ Good |
| RadioGroup<Language> with hardcoded native-name map | Only approved exception to the no-hardcoded-strings rule; localized names require the locale itself to render | ✓ Good |
| Imperial = on-position for unit switch (UI-SPEC D-10) | Toggle polarity chosen to match natural reading: "imperial is the non-default" | ✓ Good |
| `State.mounted` in ConsumerState, `context.mounted` in ConsumerWidget helpers | `mounted` refers to State object — only available in ConsumerState; ConsumerWidget uses context.mounted | ✓ Good — important Flutter/Riverpod gotcha |
| Colors.red.shade400 for destructive foreground (not colorScheme.error) | colorScheme.error maps to #FEF2F2 (background token) — illegible as foreground text | ✓ Good |
| [v1.4] Migrate to `maplibre`, not `maplibre_gl` | `maplibre` binds maplibre-native via FFI/JNI and reads our style JSON directly; `maplibre_gl` is the older package the author rewrote | — Pending |
| [v1.4] Retiring both `flomp/*` forks is the primary payoff, not map performance | The forks exist only to parse and render a style JSON that maplibre consumes natively. Native GL rendering, rotation, and pitch are follow-on benefits | — Pending |
| [v1.4] Clustering reuses `POST /search/trails/cluster` rather than `cluster: true` | Endpoint already exists and web already renders its output; keeps app/web parity, keeps the `is_large` polyline split, avoids maplibre's missing `GeoJsonSource` cluster fields entirely | — Pending |
| [v1.4] Keep the Protomaps wanderer style; do not adopt web's `ofm.json` | Offline `.pmtiles` cells are Protomaps schema (`earth`, `landcover`, `roads`); OpenMapTiles styles (`water`, `landuse`, `transportation`) would render them blank | — Pending |
| [v1.4] Style JSON lives as an app asset, not a server-hosted style URL | Offline rendering needs the style before any network call, and the `glyphs`/`sprite` keys get rewritten to `file://` at runtime anyway | — Pending |
| [v1.4] Self-host glyphs + sprite on the Wanderer server | Style references 3 fontstacks with no `glyphs` key; hosting mirrors the `TILE_SERVER_URL` operator-control model | — Pending |
| [v1.4] Glyphs cached once app-wide, not per-trail | Glyphs are style-global; per-trail download would re-fetch identical fontstacks and multiply storage | — Pending |
| [v1.4] Incremental screen-by-screen migration, forks deleted last | Keeps the app runnable at every phase boundary; the `LatLng`→`Geographic` churn touches GPX parsing, so a big-bang landing risks trail data, not just maps | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-07-08 — v1.4 MapLibre Migration started; v1.3 requirements (Phases 10–12) moved to Validated.*
