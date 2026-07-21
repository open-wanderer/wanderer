# Project Research Summary

**Project:** Wanderer v1.6 — App-wide, Region-Based Offline Map Tile Repository
**Domain:** Flutter mobile app; offline map/tile management (ObjectBox + MapLibre-native-GL + Dio, replacing a trail-scoped tile cache)
**Researched:** 2026-07-21
**Confidence:** HIGH

## Executive Summary

This milestone replaces Wanderer's trail-scoped offline tile cache with an app-wide, region-based repository: users download named, predefined regions (bundled `regions.json`, bbox-only) from a new Settings screen, and every map surface (trail detail, navigation, route planner, general map) renders from whatever regions are downloaded rather than from a per-trail tile set. Experts building this kind of feature (OsmAnd, Organic Maps, Gaia GPS, Komoot) converge on a small, consistent pattern: a flat/searchable region list, size shown before download, a 4-state status model, and — critically — a trail/route "needs region X" guard that names the missing region and offers a direct download CTA rather than a silent block. Nothing about this milestone requires new core dependencies or backend changes: the existing `dio`, `objectbox`, `maplibre` 0.3.5 (pinned), and `pmtiles` stack, plus the already-generic backend grid-cell/`bbox` endpoints, are sufficient. The only genuinely new stack pattern is byte-range download resume on the existing `Dio` instance.

The recommended approach is a strictly additive, six-phase build: (1) data model (`Region`/`DownloadedTilePackage` ObjectBox entities + bundled manifest), (2) `TileRepositoryManager` service (download orchestration + bbox→local-path query, decoupled from `TrailEntity`), (3) Settings UI, (4) map-screen rewiring (`TrailMap`, `navigation_screen`) to consume region-based tile paths instead of trail-based ones, (5) the trail-download guard wired into the single existing choke point (`DownloadingTrailIds.download`), and (6) ripout of the legacy trail-scoped tile code — deletion always last, mirroring this project's own validated v1.4 MapLibre migration discipline ("forks deleted last").

The main risks are technical, not product-shaped: region-sized files (10s–100s of MB, vs. small trail cells) expose gaps the trail-scoped code never had to handle — no download resume, no disk-space pre-check, no backgrounding-aware pause, and a naive reuse of the existing multi-cell "N duplicate sources/layers" style-rewrite technique that only scales to a handful of trail cells, not to many simultaneously-relevant downloaded regions. ObjectBox-specific traps (index-backed enum persistence, no cascade delete, cross-isolate writes) and a mundane-but-real "orphaned legacy files never get cleaned up" gap round out the critical list. All are addressable pre-emptively in the phases where they're introduced (see Roadmap Implications) rather than retrofitted later.

## Key Findings

### Recommended Stack

No new core dependencies. v1.6 extends already-validated infrastructure: `dio` ^5.9.2 (add `Range`-header + `FileAccessMode.append` resume — feature already available in the pinned version), `objectbox`/`objectbox_flutter_libs` ^5.3.1 (two new `@Entity()` classes, same `Store`), `maplibre` 0.3.5 exact-pinned (style JSON is just a map of named sources — no version bump needed, multi-source rendering already proven by `offline_style_rewriter.dart`), and `pmtiles` ^1.2.0 (confirmed read-only; no merge API — each region stays a separate file/source). Disk-usage totals should be hand-rolled via `dart:io` recursive directory walk, not a disk-space plugin (those report device-wide free space, a different question).

**Core technologies:**
- `dio` (existing, ^5.9.2) — resumable region downloads via manual `Range` header + `FileAccessMode.append`, no new HTTP package
- `objectbox` (existing, ^5.3.1) — `Region` + `DownloadedTilePackage` entities in the same `Store`, `ToMany`/`@Backlink` relation pattern already used for `TrailEntity.waypoints`
- `maplibre` (existing, pinned 0.3.5) — multi-region rendering via the existing N-source-JSON style-composition mechanism in `offline_style_rewriter.dart`, generalized from per-trail-cell to per-region keying
- `pmtiles` (existing, ^1.2.0) — one archive per region (vector + optional DEM), read-only, no in-app merge step

### Expected Features

Research (OsmAnd, Organic Maps, Gaia GPS, Komoot, mapuipatterns.com) confirms the scope already decided in PROJECT.md and adds concrete UX guidance.

**Must have (table stakes):**
- Flat, searchable region list (no hierarchical tree — manifest is tens of entries, not thousands)
- Per-region download/pause/resume/delete with inline progress, size shown *before* download (vector + DEM breakdown)
- Downloaded-region visual distinction (4-state status: notDownloaded/downloading/downloaded/updateAvailable)
- Total disk usage summary
- Trail-download guard: names the specific missing region(s), with a direct in-dialog "Download region" CTA — never a silent block or generic message; partial-coverage (trail spans two regions) must list both and allow proceeding once minimum coverage exists

**Should have (competitive, not v1 blockers):**
- Non-blocking `updateAvailable` nudge (critical lesson from OsmAnd's bug history: stale-but-present regions must keep rendering/routing — never gate functionality on this flag)
- Map boundary highlight overlay for downloaded regions

**Defer (v2+):**
- Bulk download/delete actions (only useful once manifest grows past ~5+ regions)
- Auto-download region on first launch (depends on permission-flow sequencing)
- User-drawn custom areas, remote manifest, granular per-layer toggles, offline search — all explicitly and correctly excluded already

### Architecture Approach

A single new `TileRepositoryManager` (Riverpod-provider-wrapped, mirroring the existing `trail_download_provider.dart` pattern) owns `Region`/`DownloadedTilePackage` ObjectBox boxes and exposes a bbox→local-tile-paths query. This becomes a fourth, parallel data source feeding the existing pure `rewriteStyleForOffline` function (unchanged contract) alongside the existing online-style and glyph/sprite-cache providers — no redesign of that composition. Backend requires zero changes: the grid-cell/`bbox` endpoints (`db/routes/map_cells_id.go`, `db/services/tiles/generator.go`) are already trail-agnostic; "region" is a purely client-side, bundled-manifest concept. The trail-download guard integrates at the single existing choke point, `DownloadingTrailIds.download()` in `trail_download_state_provider.dart`, which already funnels every trail-download UI entry point.

**Major components:**
1. `Region` + `DownloadedTilePackage` ObjectBox entities — manifest metadata + live download/package state
2. `TileRepositoryManager` — download orchestration (lifted from `TrailDownloadService._downloadMapTiles`) + `localTilePathsForBounds` query
3. Settings → Offline Maps/Regions screen — list, download/pause/resume/delete, DEM toggle, disk usage
4. `TrailMap`/`navigation_screen` rewiring — swap `trail.pmTiles`/`demPmTiles` reads for region-query reads, `offline_style_rewriter.dart` itself untouched
5. Guard at `DownloadingTrailIds.download()` — pre-check region coverage before trail download proceeds

### Critical Pitfalls

1. **Full-file-only region downloads with no resume** — a dropped connection on a 100s-of-MB region restarts from byte 0. Fix: `.part` file + `Range`-header resume + post-download PMTiles header/magic-byte validation before promoting to final path, addressed in the `TileRepositoryManager` download-engine phase.
2. **No disk-space pre-check + platform-inconsistent free-space APIs** (iOS `volumeAvailableCapacityForImportantUsage` vs. Android `getFreeSpace()` semantics differ) — check space with a safety margin before each file in a multi-file region download; same phase as #1.
3. **Backgrounding mid-download** (iOS suspension / Android Doze) — a multi-hundred-MB region download is far more likely to be backgrounded than a small trail cell ever was; treat `AppLifecycleState.paused` as a first-class expected event with deliberate pause, not a silent frozen socket.
4. **Layer-clone explosion at region scale** — the existing N-source/N-layer-clone style technique was validated for 1-4 trail cells, not for potentially 10+ simultaneously downloaded regions; must scope style sources to viewport intersection, not "every downloaded region always." Spike required against pinned `maplibre` 0.3.5 before committing to the viewport pipeline.
5. **ObjectBox `.index`-backed enum for download status** — silently reinterprets persisted values if a future phase inserts (not appends) a new status; use explicit stable int constants from the first schema, never `Enum.values[index]`.

## Implications for Roadmap

Based on research (architecture Q4 build order + pitfalls-to-phase mapping), suggested phase structure:

### Phase 1: Data Model — Region & Package Entities
**Rationale:** Nothing downstream can exist without the schema; purely additive, app builds unchanged.
**Delivers:** `Region`/`DownloadedTilePackage` ObjectBox `@Entity()` classes (explicit-int status enum, not `.index`-backed), bundled `assets/map/regions.json` manifest + freezed parse model.
**Addresses:** Foundation for all FEATURES.md table-stakes items.
**Avoids:** Pitfall 6 (index-backed enum) — get the persistence contract right before any value ships to a device.

### Phase 2: TileRepositoryManager (Download Engine)
**Rationale:** Core orchestration must exist and be provably correct before any UI depends on it; still a no-op addition to the running app.
**Delivers:** Region download orchestration (lifted from `TrailDownloadService._downloadMapTiles`), `localTilePathsForBounds` bbox query, resumable downloads, disk-space pre-check, backgrounding-aware pause.
**Uses:** `dio` Range-resume pattern, `objectbox` `ToMany`/`@Backlink`, existing `/map/cells?bbox=` backend endpoint (no backend changes).
**Avoids:** Pitfalls 1, 2, 3, 7, 8 (resume, disk space, backgrounding, cascade-delete cleanup, progress-write races) — these are download-primitive concerns, cheapest to get right here, hardest to retrofit once UI depends on the status contract.

### Phase 3: Settings — Offline Maps/Regions UI
**Rationale:** User-visible, independently testable/demoable without touching any existing map or trail flow — validates Phase 2 end-to-end.
**Delivers:** Region list (flat, searchable), per-region download/pause/resume/delete, DEM toggle, size shown pre-download, total disk usage summary.
**Addresses:** FEATURES.md P1 table stakes — status affordances, storage communication guidance.

### Phase 4: Map-Screen Rewiring + Viewport Tile Pipeline
**Rationale:** This is the architectural crux ("app-wide, region-based") and must not be deferred to polish — requires a spike before committing to a rendering strategy.
**Delivers:** `TrailMap._composeStyle` and `navigation_screen`'s inline compose swapped from trail-bbox to region-query tile sources; viewport-scoped (not "all downloaded regions always") style composition; legacy `TrailEntity.pmTiles`/`demPmTiles` fields left in place but unused.
**Implements:** Architecture Q1 provider layering (`regionTileRepositoryProvider` as a parallel input to the unchanged `offline_style_rewriter.dart`/`mapStyleJsonProvider`).
**Avoids:** Pitfalls 4 and 5 (layer-clone explosion, frequent style-reload lifecycle quirks) — recommend an explicit spike task (10-20 duplicated source/layer sets on a mid-tier Android device) before finalizing the pipeline.

### Phase 5: Trail-Download Guard
**Rationale:** Requires Phase 2's coverage query and Phase 3's UI (to send the user to) to already be reliable — sequencing after core region functionality avoids building UI for a lifecycle that doesn't exist yet.
**Delivers:** Coverage pre-check inserted into `DownloadingTrailIds.download()`; informative dialog naming missing region(s) + size, direct in-dialog download CTA, partial-coverage (multi-region) handling.
**Addresses:** FEATURES.md P1 "trail guard dialog" + "partial-coverage handling."
**Implements:** Architecture Q3 — single choke-point integration, no per-button duplication.

### Phase 6: Legacy Ripout & Cleanup
**Rationale:** Deletion always last, mirroring this project's own v1.4 migration discipline — nothing is deleted before its replacement is proven live.
**Delivers:** Deletion of `trail_download_service.dart`'s tile-download methods, `TrailEntity.pmTiles`/`demPmTiles` fields, trail-scoped download UI — plus a mandatory one-time on-device cleanup sweep of orphaned `<app-docs>/library/*/tiles/` directories.
**Avoids:** Pitfall 9 (orphaned legacy files silently inflating "other" storage and undermining the new disk-usage feature's accuracy) — cleanup ships in the same phase as code removal, not as a follow-up.

### Phase Ordering Rationale

- Data model → engine → UI → rendering → guard → ripout is the only order that keeps the app buildable and demoable at every boundary (each phase is either purely additive or swaps exactly one thing while leaving the old path physically present until proven redundant).
- The guard is sequenced after core region UI specifically so its CTA can trigger a real download inline, not just deep-link to Settings — a materially better UX validated by the Komoot pattern in FEATURES.md.
- The viewport/rendering phase is pulled forward (Phase 4, before the guard) because it's flagged in PITFALLS.md as the architectural crux most likely to need rework if discovered late — better to spike and settle it before the guard and ripout phases build on top of it.
- Backend requires no phase of its own: confirmed zero-change reuse of existing grid-cell/bbox endpoints throughout.

### Research Flags

Needs research/spike during planning:
- **Phase 4 (map-screen rewiring / viewport pipeline):** Direct verification needed against installed `maplibre` 0.3.5 API — does it support incremental `addSource`/`removeSource`, or only full style replacement? Also needs an empirical layer-count/performance spike (10-20 duplicated sources) before committing to the viewport-scoping strategy. LOW/MEDIUM confidence in PITFALLS.md on both points.
- **Phase 2 (download engine):** iOS `volumeAvailableCapacityForImportantUsage` vs. Android `getFreeSpace()` reserve-margin behavior — needs concrete testing on real devices, not just documentation.

Phases with standard, well-documented patterns (skip research-phase):
- **Phase 1 (data model):** Directly mirrors existing `TrailEntity`/`ActiveNavigationEntity` ObjectBox conventions already twice-proven in this codebase.
- **Phase 3 (Settings UI):** Standard list/detail Settings pattern already established elsewhere in the app.
- **Phase 5 (guard):** Single choke-point insertion into an already-documented, single-purpose notifier method.
- **Phase 6 (ripout):** Mechanical deletion + cleanup sweep, no new patterns.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All findings verified directly against in-repo source (`trail_download_service.dart`, `offline_style_rewriter.dart`, entity files); no new dependencies introduced, so version-compatibility risk is minimal. |
| Features | MEDIUM | WebSearch-derived competitor UX conventions (OsmAnd, Organic Maps, Gaia GPS, Komoot), cross-checked across multiple apps but not official API docs — product/UX judgment, not library facts. |
| Architecture | HIGH | All findings verified against current repo source directly, including a naming correction against the milestone brief's stale file references. |
| Pitfalls | MEDIUM | Mix of HIGH-confidence platform/OS behavior (iOS backup exclusion, ObjectBox no-cascade-delete) and MEDIUM/LOW-confidence maplibre-native-0.3.5-specific claims (layer-count degradation threshold, incremental source API availability) that are explicitly flagged as needing an in-repo spike. |

**Overall confidence:** HIGH — the plan rests on solid, directly-verified in-repo architecture and stack decisions; the softer areas (competitor UX conventions, native-binding-specific performance thresholds) are exactly the ones already flagged for spikes/validation rather than assumed.

### Gaps to Address

- **maplibre 0.3.5 incremental source/layer API availability:** unconfirmed from docs; resolve via direct package inspection at the start of Phase 4.
- **Layer-clone performance threshold on the native binding:** JS-binding community reports are directionally relevant but not native-specific; resolve via the recommended Phase 4 spike (10-20 duplicated sources on a mid-tier Android device).
- **Wi-Fi-only download default/toggle:** flagged in FEATURES.md/PITFALLS.md as a reasonable UX addition but not explicitly scoped in PROJECT.md — surface as an open roadmap question for Phase 3, not assumed either way.
- **DEM toggle default (on/off):** PROJECT.md doesn't specify; STACK.md notes this should be decided based on typical vector-vs-DEM size ratio during Phase 3 planning.

## Sources

### Primary (HIGH confidence)
- In-repo: `app/lib/services/trail_download_service.dart`, `app/lib/util/offline_style_rewriter.dart`, `app/lib/entities/trail_entity.dart`, `app/lib/entities/active_navigation_entity.dart`, `app/lib/provider/trail/trail_download_state_provider.dart`, `db/routes/map_cells_id.go`, `db/services/tiles/generator.go`, `db/services/tiles/grid.go`, `.planning/PROJECT.md`
- [ObjectBox Relations docs](https://docs.objectbox.io/relations), [ObjectBox Data Model Updates docs](https://docs.objectbox.io/advanced/data-model-updates)
- [isExcludedFromBackupKey — Apple Developer Documentation](https://developer.apple.com/documentation/foundation/nsurlisexcludedfrombackupkey)
- [background_downloader | Flutter package](https://pub.dev/packages/background_downloader)

### Secondary (MEDIUM confidence)
- OsmAnd, Organic Maps, Gaia GPS, Komoot help docs and GitHub issue trackers (region UX conventions, real-world resume/outdated-map bug reports)
- pub.dev `dio` changelog (`FileAccessMode` support), community Dio resume-pattern write-up
- [Supporting multiple offline pmtile maps — maplibre/maplibre-native Discussion #3764](https://github.com/maplibre/maplibre-native/discussions/3764)
- [Performance bottleneck with 8000+ PMTiles layers — maplibre-gl-js Discussion #5988](https://github.com/maplibre/maplibre-gl-js/discussions/5988) (JS binding, directionally relevant only)
- [Removing an entity with ToOne relation keeps related entity — objectbox-dart Issue #547](https://github.com/objectbox/objectbox-dart/issues/547)
- Android Doze/App Standby background execution constraints (multiple community sources)

### Tertiary (LOW confidence)
- maplibre-native 0.3.5-specific layer-count performance threshold and incremental source/layer API availability — both explicitly flagged as needing direct in-repo verification/spike before Phase 4.

---
*Research completed: 2026-07-21*
*Ready for roadmap: yes*
