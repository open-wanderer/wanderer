# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

---

## Milestone: v1.2 — Settings Screens

**Shipped:** 2026-06-29
**Phases:** 4 (Phases 6-9) | **Plans:** 9 | **Timeline:** 2026-06-19 → 2026-06-21 (3 days active)
**Dart files touched:** ~36 | **Phase commits:** ~60

### What Was Built
- **Phase 6:** Five-row settings list wired to sub-routes; 14-locale language picker + metric/imperial unit toggle that live-switches app-wide via `localeProvider` / `unitProvider`; ~50 `format_util` call sites ported to read unitProvider
- **Phase 7:** `SettingsPrivacyScreen` — three `RadioGroup<String>` sections (account/trails/lists visibility) with auto-save via `settingsProvider`
- **Phase 8:** Full `SettingsAccountScreen` — avatar upload (image_picker multipart POST), change-aware bio editor, email/password bottom-sheet forms with proper credential handling, AlertDialog-gated account deletion
- **Phase 9:** `SettingsNotificationsScreen` — 9 notification types × web/email SwitchListTile toggles, map-copy pattern auto-save, widget test, new `l10n.web` ARB key

### What Worked
- **Reusing `settingsProvider`**: All four screens read/write from the same `Settings` freezed model via `settingsProvider.saveToServer()` — zero new persistence infrastructure needed
- **Wave parallelism in Phase 6**: Plans 06-02, 06-03, and 06-04 ran concurrently with no file overlap — pure parallel execution
- **Auto-save pattern (D-09 map-copy)**: Copying the notifications map, mutating the copy, and saving — same pattern reused across privacy and notifications screens cleanly
- **Stub screens from Phase 6**: Stubs for Privacy/Notifications created upfront; Phases 7 and 9 were pure fills with no scaffolding work
- **Widget tests as acceptance**: Tall-viewport widget tests (1080×4000) for lazy ListViews caught real rendering issues early

### What Was Inefficient
- **`--force` flag missing from CLI wrapper**: `gsd milestone complete --force` silently failed because `gsd-tools.cjs` didn't forward the flag to `cmdMilestoneComplete`. Required a one-line patch at close time.
- **Phase 7 one-liner extraction**: The 07-01-SUMMARY.md one_liner field captured a rule annotation rather than a human description — MILESTONES.md has a low-quality entry for that phase
- **Human-needed verification items**: Phases 6-8 all have `human_needed` verification entries that were acknowledged-and-deferred rather than tested on device. Device testing should be scheduled as a quick task before close, not discovered at close

### Patterns Established
- **`State.mounted` in ConsumerState, `context.mounted` in ConsumerWidget helpers** — `mounted` is a State property only; ConsumerWidget doesn't inherit State. Verify before every async BuildContext guard.
- **`@JsonSerializable(explicitToJson: true)` on freezed factory constructor** — class-level placement breaks codegen in freezed 3.x; always on the factory.
- **`Colors.red.shade400` for destructive foreground text** — `colorScheme.error` is a background token (#FEF2F2), not a foreground color
- **Hardcoded native-name map for language labels** — the only approved hardcoded-string exception; localized names require the locale to be active, which creates a bootstrap problem
- **`findWidgets` not `findsOneWidget` when hintText appears in header + field** — duplicate widget keys in lazy lists require `findsWidgets`

### Key Lessons
1. **Device testing is a milestone blocker, not a nice-to-have**: Acknowledging `human_needed` verification items at close is a smell — these should be scheduled as a quick task (`/gsd-quick "device test phases 6-8"`) before milestone close
2. **Stub screens reduce planning cost**: Creating stubs in the foundational phase (Phase 6) meant downstream phases (7, 9) had zero route/scaffold work — just filling the screen body
3. **Wave parallelism requires explicit no-overlap verification in the plan**: Phase 6 Wave 2 succeeded because plans were written with explicit non-overlapping file lists; this needs to be a planning checklist item
4. **CLI `--force` flags need end-to-end testing**: The missing `--force` passthrough is a category of bug that only surfaces at milestone close — worth a smoke test after any gsd-tools update

---

## Milestone: v1.4 — MapLibre Migration

**Shipped:** 2026-07-10
**Phases:** 6 (Phases 13-18) | **Plans:** 17 | **Timeline:** 2026-07-08 → 2026-07-10 (3 days active)
**Requirements:** 40/40 v1.4 requirements complete

### What Was Built
- **Phase 13:** Unified `/api/v1/map/style-sources` SvelteKit endpoint replacing `/map/tileurl`, resolving tile + glyph + sprite URLs under one operator override
- **Phase 14:** `latlong2.LatLng`/`LatLngBounds` → `Geographic`/`LngLatBounds` across the trail data layer, isolated and test-guarded before any map rendering code changed
- **Phase 15:** `WandererMap` rewritten onto native `MapLibreMap` — style JSON assets, app-wide glyph/sprite cache, trail track/waypoint/pin rendering, and the offline parity gate (`.pmtiles` + `file://` glyphs rendering with no network)
- **Phase 16:** List and map-screen browse surfaces ported to `SearchMap`, with server-side `/search/trails/cluster` rendered as native circle/symbol layers
- **Phase 17:** Navigation screen migrated to native puck/follow/compass; last `flutter_map` plugin call sites removed from `lib/`
- **Phase 18:** Both `flomp/*` forks and all 6 legacy map packages removed from `pubspec.yaml`; `maplibre` pinned to exact 0.3.5

### What Worked
- **Risk gate first**: Phase 15 opened with a throwaway spike (15-01) proving `file://` glyph resolution before investing in trail rendering — caught the sprite `file://` gap early enough to design around it (self-registered arrow icon) rather than discover it late
- **Screen-by-screen migration with both stacks coexisting**: `flutter_map` and `maplibre` lived in `pubspec.yaml` from Phase 15 through 17; every phase boundary left the app buildable, so no phase was ever blocked waiting on another
- **Isolating the coordinate-order footgun**: Landing `Geographic`/`LngLatBounds` as its own early phase (14), guarded by existing GPX/polyline tests, gave one unambiguous signal (identical coordinates) instead of conflating it with camera/rendering bugs later
- **Deferring file deletion until the last consumer migrates**: `pm_tile_provider.dart` (OFFL-06) and `map_compass.dart` (CORE-05) stayed in place until `navigation_screen` — their last holdout — migrated in Phase 17, avoiding a broken intermediate state

### What Was Inefficient
- **Two real on-device bugs only surfaced during physical-device verification, not planning**: a flutter_map-only `MapCompass` widget crashed at runtime (`MapCamera.of()` needs a `FlutterMap` ancestor) and offline pmtiles `maxzoom` mismatched the server's extraction depth, causing blank tiles above z14 — both were physical-device-only failure modes invisible to `flutter analyze`/unit tests
- **Six UI polish gaps surfaced at the Phase 18 on-device checkpoint** were pre-existing, not regressions, but required a follow-up quick task (260710-kpd) rather than being caught during earlier phases' own device passes
- **Milestone close found 15 quick tasks the audit tool couldn't classify** (missing/unrecognized status field) — 14 were actually complete with SUMMARY.md on disk; only 1 (dark mode, 260612-gmg) was a genuine gap. Worth tightening the audit tool's status detection so real gaps aren't buried in noise

### Patterns Established
- **`rewriteStyleForOffline` as the single sanctioned online→offline style transform** — pure, deep-copies input, rejects non-absolute/`..`/foreign-scheme paths before emitting
- **`map_cache_path.dart` as the single sanctioned builder for map-cache filesystem paths** — whitelists fontstack/range tokens, rejects unknown ones with `ArgumentError` before any path is built
- **`onMapCreated`/`onStyleLoaded` race buffering** — the native platform channel doesn't reliably fire `onMapCreated` before `onStyleLoaded`; any screen using both must buffer a style-loaded event that arrives first and replay it once the controller is set
- **`Duration(milliseconds: 1)`, never `Duration.zero`, for instant camera moves** — a zero duration crashes the Android native `animateCamera` binding
- **Physical-device checkpoints as explicit plan tasks, not implicit assumptions** — Phases 15, 17, and 18 each ended with a dedicated on-device verification plan; this caught all real regressions this milestone

### Key Lessons
1. **A throwaway spike for the riskiest unknown pays for itself** — 15-01's `file://` glyph spike de-risked the entire offline parity gate before Phase 15's other 5 plans were written
2. **Both-stacks-coexist migrations need an explicit "who deletes what, when" map** — CORE-05/06/07 and OFFL-06 all "retire a file the last screen still uses"; tracking this explicitly in ROADMAP.md's sequencing rationale avoided a phase silently breaking another
3. **Native GL packages fail differently than pure-Dart ones** — crashes and blank-tile bugs here were FFI/platform-channel-level, invisible to `flutter analyze` and unit tests; on-device checkpoints are not optional for this class of dependency
4. **Audit tooling needs richer status detection before milestone close** — a binary "has SUMMARY.md" check would have cut this milestone's audit noise from 15 items to 1

---

## Milestone: v1.5 — Route Planner

**Shipped:** 2026-07-17
**Phases:** 3 (Phases 19-21) | **Plans:** 13 | **Timeline:** 2026-07-16 → 2026-07-17 (2 days active)
**Requirements:** 15/15 v1.5 requirements complete

### What Was Built
- **Phase 19:** From-scratch route building directly on the map — tap/drag/insert waypoints, an auto-routing toggle (Valhalla-routed vs straight-line, fixed foot/bike profile), undo/redo — backed by a class-based `RouteAnchors` `@riverpod` notifier with a CancelToken + generation-counter race guard and an immutable-snapshot undo/redo stack
- **Phase 20:** Route anchor list (delete/reorder) and a live elevation profile as two tabs of one docked, draggable sheet; a dedicated locations-only `LocationSearchScreen` for search-to-focus
- **Phase 21:** Real hike/bike entry-point dialog wired into the trail-source-select flow, an app-bar "Finish" action, and `finishPlanning` handoff to `trail_create_screen` as a draft Trail (GPX track only, one-time elevation merge at handoff)

### What Worked
- **Two deliberate scope changes caught during discuss-phase, not after building**: PLANUI-01 (tabs-of-one-sheet instead of two toggled views) and HANDOFF-01 (GPX-only handoff, no synthesized Waypoint records) were both resolved in CONTEXT.md before planning started, avoiding a rebuild
- **A single fetch-at-handoff for elevation, not a continuous background fetch** — `plannedGpxProvider` deliberately stayed pre-elevation through Phase 20; Phase 21 added one `/api/v1/valhalla/height` call at the moment of handoff instead of re-firing Valhalla on every anchor edit
- **Reusing existing infrastructure aggressively**: `GlobalSearchScreen`'s debounced search provider, the `pendingImportedTrail` handoff mechanism, and `ElevationProfile`'s existing null-Trail path were all extended rather than rebuilt

### What Was Inefficient
- **Milestone was never formally closed at the time** — v1.6 requirements-gathering silently overwrote `REQUIREMENTS.md` and phase directories 19-21 were deleted from disk (commit `fcd63d17`, "update planning") before `/gsd-complete-milestone` ever ran. This archive was reconstructed from git history a week later, during v1.6's own close. Always run `/gsd-complete-milestone` immediately after a milestone's last phase lands, before starting the next milestone's requirements pass

### Patterns Established
- **Segment-split-on-tap + adjacency-diff-on-reorder** for route topology mutations — avoids full route reconstruction on every edit
- **`RouteAnchorLayer`/`RouteSegmentLayer`** as the native-map rendering pair for editable, numbered, draggable route geometry (vs. the read-only `TrailLayer` pattern used elsewhere)

### Key Lessons
1. **Close milestones before starting the next one's requirements pass** — skipping `/gsd-complete-milestone` let the next milestone's fresh `REQUIREMENTS.md` silently destroy the previous one's traceability table, with no error or warning
2. **Deleting phase directories outside the GSD archival flow loses recoverable-but-inconvenient history** — the 46-file bulk deletion (`fcd63d17`) wasn't malicious, just informal cleanup; formal archival (`milestones/vX.Y-phases/`) exists precisely so this kind of cleanup doesn't require git archaeology later

---

## Milestone: v1.6 — Offline Region Tile Repository

**Shipped:** 2026-07-24
**Phases:** 8 (Phases 21.5, 22-27, incl. inserted 25.1) | **Plans:** 30 | **Timeline:** 2026-07-21 → 2026-07-24 (4 days active)
**Requirements:** 40/41 v1.6 requirements complete (CLEAN-02 explicitly descoped, not failed)

### What Was Built
- **Phase 21.5:** Go backend region catalog loaded from an admin-supplied, Docker-volume-mounted config file; a cronjob pre-builds one mosaicked vector + one DEM PMTiles archive per region; an auth-gated API endpoint (proxied through SvelteKit) serves the catalog
- **Phase 22:** App-side region manifest fetched at runtime (no bundled asset) plus ObjectBox `Region`/`DownloadedTilePackage` entities with explicit-int status persistence
- **Phase 23:** `TileRepositoryManager` — disk-safe, resumable region downloads (later amended to cancel-and-restart), bbox-to-local-paths query, fully decoupled from Trail
- **Phase 24:** Settings → Offline Maps/Regions: searchable region list, independent Vector/DEM download rows, total disk usage (amended mid-milestone: DEM toggle → gated DEM tile)
- **Phase 25:** `TrailMap`/`navigation_screen` read region tiles through a viewport-scoped style pipeline, settled by an on-device maplibre 0.3.5 spike
- **Phase 25.1 (inserted):** Local loopback HTTP tile proxy replacing incremental `addSource`/`removeSource` region-swap reconciliation, closing a UAT-diagnosed reentrancy race structurally
- **Phase 26:** Trail download guard checks region coverage first, naming missing regions with an inline per-region download CTA, supporting partial-coverage subset downloads
- **Phase 27:** Legacy trail-scoped tile system (3 service methods, `map_cell.dart`, `TrailEntity.pmTiles`/`demPmTiles`) deleted outright; CLEAN-02's cleanup sweep descoped as unnecessary for a pre-production app

### What Worked
- **A backend-first phase (21.5) inserted after Phase 22 was already planned, before any client work landed** — discussion surfaced that "region" couldn't be a purely client-side bundled-manifest concept (self-hostable app, per-instance admin decision) before real work compounded on the wrong assumption
- **Mid-milestone amendments landed as ROADMAP/REQUIREMENTS notes, not silent drift** — both the pause/resume removal (Phase 23) and the DEM-toggle→DEM-tile change (Phase 24) were explicitly documented with dates and commit hashes at the point of change, so later phases and this retrospective could cite them precisely
- **An urgent phase insertion (25.1) replaced a broken mechanism structurally instead of patching it** — Phase 25's reentrancy race in hand-rolled Dart source diffing was eliminated by moving region selection to MapLibre Native's own viewport tracking via a tile proxy, not by adding more guards around the old reconcile loop
- **Sequential dependency chain with explicit sequencing rationale in ROADMAP.md** — each phase's "why this order" was written down before execution (data model → engine → UI → rendering spike → guard → ripout), so a phase 27 ripout only happened once every upstream consumer was proven

### What Was Inefficient
- **A background planner agent stalled twice during Phase 27 planning** (laptop standby interrupted the agent stream) before a third attempt completed — cost extra wall-clock time but no data loss, since each retry re-read the same RESEARCH.md/PATTERNS.md rather than diverging
- **The milestone was executed to completion (all 8 phases) before `/gsd-complete-milestone` was ever run** — same pattern as v1.5: formal closure lagged actual completion by the length of the whole milestone, only triggered when the *next* milestone close was requested

### Patterns Established
- **Local loopback `dart:io HttpServer` as a tile-proxy pattern** for handing region selection off to the native map engine's own viewport tracking, applicable to any future "which of N downloaded assets covers this viewport" problem
- **Explicit-int status enum persistence** (`.code` constants, never `.index`) for ObjectBox — avoids silent status corruption if enum ordering ever changes, now the house style for all new status enums
- **Amendment notes with commit hashes directly in ROADMAP.md/REQUIREMENTS.md**, not just in phase CONTEXT.md — keeps the top-level planning docs honest without requiring readers to dig into per-phase archives for "does this still match what shipped"

### Key Lessons
1. **Formally close a milestone the moment its last phase verifies passed** — don't let the next milestone's planning start first; both v1.5 and v1.6 in this project sat "done but unclosed" for their entire successor milestone's duration
2. **A structural fix (change the mechanism) beats a defensive fix (guard the old mechanism) for race conditions found in UAT** — Phase 25.1's tile-proxy insertion cost one extra phase but eliminated the reentrancy bug class entirely, versus patching `MapEventCameraIdle` handling indefinitely
3. **Descoping a requirement (CLEAN-02) is not the same as failing it** — recording the descope rationale (D-05, pre-production app) in CONTEXT.md, REQUIREMENTS.md, ROADMAP.md, and PROJECT.md all at once meant no downstream verification or audit step mistook "cut deliberately" for "missed"

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Phases | Plans | Key Change |
|-----------|--------|-------|------------|
| v1.0 MVP | 3 | 6 | Established navigation screen pattern (freezed models, Riverpod notifiers, flutter_map) |
| v1.1 Offline | 2 | 6 | Added ObjectBox caching; established DioException-only offline gate pattern |
| v1.2 Settings | 4 | 9 | Shared `settingsProvider` pattern; live locale/unit switching; wave parallelism |
| v1.3 Category Redesign | 3 | 12 | Category/subcategory model + preference providers; subcategory-aware filters |
| v1.4 MapLibre Migration | 6 | 17 | Full native-GL map migration; risk-gate spike pattern; both-stacks-coexist screen-by-screen cutover |
| v1.5 Route Planner | 3 | 13 | Editable route-anchor map layers; fetch-at-handoff pattern for derived data; scope changes resolved pre-plan via discuss-phase |
| v1.6 Offline Region Tile Repository | 8 | 30 | Backend-first phase insertion when a client-only assumption broke; structural fix (tile proxy) over defensive patching for a UAT-found race; documented mid-milestone amendments with commit hashes |

### Cumulative Quality

| Milestone | Widget Tests Added | Notable |
|-----------|--------------------|---------|
| v1.0 | ~6 (navigation + stats) | TDD approach for navigation notifier |
| v1.1 | ~4 (serialization roundtrip, offline fallback) | ObjectBox integration tests via unit tests |
| v1.2 | ~5 (one per settings screen) | Tall-viewport pattern for lazy ListViews |
| v1.4 | On-device checkpoints per phase (15, 17, 18) rather than widget tests | Native GL / platform-channel bugs are invisible to `flutter analyze` and unit tests |

### Top Lessons (Verified Across Milestones)

1. **Stub screens in foundational phases pay forward** — confirmed in v1.2 (Phase 6 stubs → fast Phases 7+9)
2. **Wave parallelism requires explicit file-overlap analysis at plan time** — confirmed valuable in v1.2 Phase 6
3. **Human/device testing needs a dedicated quick task before milestone close** — first surfaced v1.2; confirmed again in v1.4 (Phase 18's on-device walk found 6 real UI gaps only a physical device could surface)
4. **Risk gates (throwaway spikes) for the riskiest unknown de-risk everything downstream** — v1.4's 15-01 glyph spike validated the offline parity approach before 5 more plans were built on top of it
