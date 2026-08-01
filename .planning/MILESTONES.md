# Milestones

## v1.7 Admin Region Picker (Shipped: 2026-07-28)

**Phases completed:** 5 phases, 19 plans, 46 tasks
**Timeline:** 2026-07-25 → 2026-07-28 (4 days) · 166 files, +30,818/−711 · `58ae4d20..e41fa3be`

**Delivered:** A server owner defines downloadable regions by toggling entries in a seeded CoMaps
catalog with real boundaries, instead of hand-authoring `region_config.json` — and the app's
settings screen presents the same hierarchy.

**Key accomplishments:**

- Seeded the full 1,306-row CoMaps region catalog (153 groups, 1,153 leaves) into a new `regions`
  PocketBase collection via a maintainer-run Cobra command plus an auto-run migration, so a fresh
  self-hosted instance boots with a populated, toggleable catalog and zero admin action.
- Retired `region_config.json` entirely: the archive cron now reads build targets from
  `kind='leaf' AND enabled=true` and clips both vector and DEM PMTiles to each leaf's canonical
  GeoJSON polygon via `pmtiles extract --region`, replacing bbox-based extraction.
- Shipped a standalone PocketBase admin page rendering the catalog as a collapsible, filterable
  tree with optimistic enable/disable toggles and a live MapLibre map showing every enabled
  region's true boundary.
- Brought the Flutter Settings → Offline Maps/Regions screen to the same hierarchy, pruned to
  admin-enabled regions plus their ancestors, with every existing download/cancel/delete action
  and the disk-usage summary preserved.
- Moved boundary geometry off-repo entirely: the committed catalog went from **54.65 MB gzipped
  to ~315 KB of plain hierarchy JSON** (~190×), geometry is now fetched on demand from CoMaps at a
  pinned commit and cached only for regions an admin actually enabled, and the maintainer seed run
  collapsed from ~1,153 HTTP requests to one.
- Purged the retired 57 MB seed blob from 133 commits of published history with
  `--force-with-lease`, shrinking a fresh clone's pack from 268.00 MiB to 198.14 MiB while still
  migrating up to the identical 1,306-row catalog.

### Known Gaps

Accepted at close; see `.planning/milestones/v1.7-MILESTONE-AUDIT.md` (status `gaps_found`).
Both are verification-coverage gaps, not defects — integration checking found 8/8 cross-phase
seams wired and the full E2E flow unbroken.

- **EXTRACT-01/02/03 (Phase 29)** — the phase has no VERIFICATION.md. Wiring is code-verified
  correct today, but nothing phase-owned would catch it regressing. The `region_id` → `path`
  rename already broke this area once, silently.
- **APPUI-01/02 (Phase 31)** — `status: human_needed`. The on-device pass its VERIFICATION.md
  explicitly requires after Phase 32's server rewrite has never been performed.

**Known deferred items at close:** 40 (see STATE.md Deferred Items) — 39 predate v1.7.

### Notes

- CATALOG-02 was satisfied in Phase 28 and then deliberately superseded by Phase 32; leaf rows
  store neither `polygon` nor `bbox`. Recorded as superseded-by-design, not a gap.
- Three integration bugs surfaced through manual use *after* Phase 32's verification passed
  (`4b98c48b`, `0149b83e`, `6069cb57`), all fixed. Phase-level verification under-covered
  cross-phase wiring in this milestone.
- No holed `Polygon` exists anywhere in the real 1,306-row CoMaps catalog — Phase 28's multi-ring
  hole support has never been exercised by actual data.

## v1.6 Offline Region Tile Repository (Shipped: 2026-07-24)

**Phases completed:** 8 phases, 30 plans, 63 tasks

**Key accomplishments:**

- JSON region-catalog loader (tolerant of missing/empty config) plus path-traversal-safe id validation and the `region_archives` PocketBase collection tracking per-region vector/DEM build state.
- `BuildAll(app)` cron entrypoint pre-builds one mosaicked vector PMTiles archive (Protomaps, z14) and one DEM archive (Mapterhorn, z12) per configured region, with atomic-rename availability, date-gated vector rebuilds, build-once DEM, and a region-id in-flight guard — a parallel, independent build path that never touches the existing per-cell `generator.go`.
- `GET /api/v1/regions` (auth-gated, reachable at the app's public origin via a SvelteKit proxy) returns the merged config-plus-build-state region catalog; archive downloads are auth-gated and id-validated; the daily `region-archive-build` cron now runs `regions.BuildAll`; docker-compose wires the admin config path in all three deployment files.
- Typed freezed parse model for GET /api/v1/regions plus ObjectBox RegionEntity/DownloadedTilePackageEntity, all status enums persisted via explicit `.code` int constants (never `.index`)
- RegionRepository.refreshCatalog() GETs /api/v1/regions through the cookie-authenticated Dio client, parses the bare JSON array while dropping malformed elements, and upserts by business id inside a single write transaction -- preserving every region's ToOne package links and local download status, with orphaned regions flipped inCatalog=false rather than deleted
- Both region-archive SvelteKit proxy routes (vector and DEM) now forward the client's Range header inbound and the backend's actual status (200/206) plus Content-Range/Accept-Ranges outbound, proven by 4 new vitest assertions — unblocking Plan 04's Flutter resumable-download work (TILE-02).
- Appended paused/error to PackageStatus and RegionStatus with stable new codes, wired RegionEntity.status to map them, and shipped region_file_path.dart validating catalog region ids against the backend's exact allow-list before building vector/dem archive paths.
- Added `disk_space_2` (post-legitimacy-review) and wrapped it in `disk_space_util.dart`, whose pure `hasEnoughSpace` margin decision is unit-tested independently of the plugin.
- TileRepositoryManager downloads a region's vector and DEM `.pmtiles` archives to a `.part` file, resuming from the existing byte offset via HTTP Range + `FileAccessMode.append`, refusing to write when disk space is tight, validating with `PmTilesArchive.fromFile` before promoting `.part` to its final path, and treating app backgrounding as a deliberate `AppLifecycleListener`-driven pause.
- Added `localTilePathsForBounds`/`bboxOverlaps`/`deleteRegion` to `TileRepositoryManager` and wired it into Riverpod via a construction-only `tileRepositoryManagerProvider` seam plus a `keepAlive` `TileRepositoryStatus` notifier exposing per-region `RegionDownloadState`.
- Standalone `flutter run -t`-launched debug driver exercising every public `TileRepositoryManager` method against a real region, plus the recorded 5-behavior end-of-phase human-check for TILE-02/03/04, DEM-01/02, and TILE-05.
- DEM-only cascade delete, an A-Z synchronous region-list provider, real-on-disk byte formatting/aggregation utilities (incl. `.part` partial files), and all 18 Phase 24 English l10n keys — the full symbol contract Plan 02's screen is written against.
- SettingsOfflineRegionsScreen — a searchable A-Z region list with a live disk-usage summary, six-state download rows (download/pause/resume/delete/retry/update), an independent DEM toggle, and a Settings entry wired via `/settings/regions` — the single user-facing deliverable of the v1.6 milestone's UI phase.
- Implemented the device-wide disk-space fallback `freeDiskSpaceBytes`'s own doc comment already promised but never delivered, fixing the Phase 24 UAT blocker where every first-ever region download (vector or DEM) was refused on a real device because the path-specific disk-space query throws on a `regions/<id>/` directory that doesn't exist yet.
- Fixed the stale ObjectBox ToOne-cached `region.status` render bug by adding a pure `resolveRowStatus` resolver that prefers the live ephemeral download state during an in-flight vector download, restoring the progress bar/pause-button UI and unblocking the paused mid-transfer disk-usage check.
- Built a throwaway on-device spike harness that materializes 10-20 duplicated region vector/DEM sources via the production `rewriteStyleForOffline` helper, and settled RENDER-03 in favor of incremental `addSource`/`removeSource` over full-style-reload after physical-device testing surfaced two implementation gaps (missing repaint-on-remove, wrong hillshade z-order) that Wave 2 must now design around.
- `TileRepositoryManager.localTilePathsForBounds` now returns `({List<String> vectorPaths, List<String> demPaths})` via a unit-tested `@visibleForTesting splitRegionTilePaths` pure helper, closing the RENDER-01 data-shape gap before any Wave 2 rendering code can conflate a DEM archive with a vector cell.
- TrailMap's offline basemap/hillshade now sources vector+DEM tile paths from the app-wide region registry (`localTilePathsForBounds(trail.bounds)`) instead of the trail's own `pmTiles`/`demPmTiles` fields, and applies a mid-session region download incrementally via `addSource`/`addLayer` (hillshade below the first vector layer) instead of a full `setStyle` reload.
- navigation_screen's offline basemap now sources tiles from the region registry via the live camera viewport, incrementally swapping region sources/layers in and out on camera-idle (never a full setStyle reload) with a hillshade z-order fix and a post-removal repaint nudge.
- Android network_security_config.xml scoped to 127.0.0.1 + iOS Info.plist NSAppTransportSecurity loopback exception, both narrowly bounded with negative greps proving no blanket cleartext/insecure-loads allowance was introduced
- Loopback-only `dart:io HttpServer` serving vector/DEM map tiles from region `.pmtiles` archives via a per-request smallest-bbox winner resolver, paired with a static-XYZ offline style rewriter that replaces per-cell `pmtiles://` source duplication with one fixed proxy URL template.
- PROXY-03 confirmed PROCEED on a physical Pixel 6: the loopback HTTP tile proxy reliably serves vector tiles to MapLibre Native in full airplane mode, clearing the phase's risk gate for Plan 04's TrailMap/navigation_screen rewiring.
- Both `TrailMap` and `navigation_screen` now compose their offline style through one static loopback-proxy XYZ source (`rewriteStyleForProxy`), with the entire incremental `addSource`/`removeSource` region-reconcile machinery — and the `MapEventCameraIdle` trigger that raced it — deleted from both files, closing Phase 25's UAT Test 4 gap structurally.
- Pure bboxesOverlap/overlappingRegions/missingCoverageRegions functions deciding which trail-overlapping regions still need downloading, with updateAvailable treated as covered (GUARD-04)
- Bottom modal sheet listing missing regions with Vector/DEM checkboxes (Vector-on/DEM-off default) and an always-enabled Download button, plus a sibling DownloadNotificationService method for D-10's unified progress copy
- Coverage guard + parallel region-package downloads + unified aggregate notification wired into `DownloadingTrailIds.download()`, the single shared entry point both trail-download call sites already use
- Closed the one blocking gap (missing `regionListNotifierProvider` invalidation) and three robustness findings from Phase 26 verification/code-review in `DownloadingTrailIds.download()`, without changing any already-verified guard behavior or button-unlock timing.
- Monotonic per-package progress latch stops the id-42 aggregate bar from resetting on each package completion, and a fresh-row read-modify-write in TileRepositoryManager stops a concurrent Vector download from clobbering a concurrently-linked DEM package relation.
- Deleted the three trail-scoped tile-download methods, all generating-state wiring, the showGenerating() spinner, and the orphaned map_cell.dart model from the Flutter app, leaving downloadTrail() to handle only photos/waypoint-photos/nav-cache with a fixed up-front progress total.
- Deleted the persisted `pmTiles`/`demPmTiles` fields from `TrailEntity` (ObjectBox) and `Trail` (freezed) together, then ran a single `build_runner build --delete-conflicting-outputs` pass to regenerate `trail.freezed.dart`, `trail.g.dart`, `objectbox-model.json`, and `objectbox.g.dart` with both property UIDs cleanly retired.

---

## v1.5 Route Planner (Shipped: 2026-07-24)

**Phases completed:** 3 phases, 13 plans, 23 tasks

**Key accomplishments:**

- Precision-parameterized Google-encoded-polyline codec (default 5, Valhalla decodes at 6) plus a new freezed `RouteAnchor`/`RouteSegment`/`SegmentState`/`RouteAnchorsSnapshot` in-memory route model that never reuses the persisted `Waypoint` type
- Class-based `RouteAnchors` `@riverpod` family notifier: per-segment Valhalla routing engine with a CancelToken + generation-counter race-guard, append/drag/insert anchor mutations, geometric segment-split for plain taps, and an immutable-snapshot undo/redo stack
- Native-map rendering surfaces for the route planner: `RouteAnchorLayer` (numbered, draggable `WidgetLayer` markers) and `RouteSegmentLayer` (GeoJSON-backed, state-filtered `LineStyleLayer` segment renderer with an invisible wide hit-test layer), plus a unit-tested `buildSegmentsGeoJson` builder
- `RoutePlannerScreen`: the screen that hosts the native map, disambiguates marker/segment/empty-map taps into the correct 19-02 mutation, exposes the auto-routing toggle, and puts undo/redo + blocked-segment/retry toast copy in the app bar — closing the goal-backward reachability chain for all of Phase 19's requirements
- Added deleteAnchor/reorderAnchors mutators to RouteAnchors (segment-collapse-on-delete, adjacency-diff reuse-on-reorder) plus a buildGpxFromPoints helper and plannedGpxProvider that derives a live pre-elevation Gpx by walking the anchor-id chain.
- LocationSearchScreen — a locations-only mirror of GlobalSearchScreen that reuses the existing debounced globalSearchProvider and pops its result back to the caller via `/location-search`, a new pushable go_router route.
- ElevationProfile now accepts a null Trail with gpx.getTotals()-derived stats and no anchor icons; new ElevationTab fetches /valhalla/height only while visible, debounced 500ms, with index-aligned ele-merging and a <2-anchor empty state.
- RouteAnchorListTab — a ReorderableListView.builder of route anchors with a numbered accent badge, coordinate subtitle, immediate no-confirmation delete, and long-press-drag reorder wired directly to the sheet's ScrollController.
- RouteAnchorSheet — a docked, tabbed DraggableScrollableSheet (Route Anchors + Elevation) wired into route_planner_screen.dart alongside a magnifying-glass search control that pans the map to a searched location at zoom 13.
- `finishPlanning` orchestration util that turns the in-progress planner route into a draft Trail (GPX track only, no Waypoint records) with a silent one-time elevation merge and hike/bike category pre-fill, handed off via the existing `pendingImportedTrail` mechanism.
- Ported the `Behavior` nested type (`allowAutoGeolocate`, `mapClusteringMaxZoom`, `showTrailStartMarker`) from web's `Settings.behavior` onto the Flutter `Settings` freezed model, with `SettingsEntity.behaviorJson` persisting it via the same JSON-blob strategy as `privacyJson` — closing the D-03 gap so Plan 03's GPS gate at planner entry has a real field to read.
- Real HANDOFF-02/03 entry point: the "Plan a route" card now opens a dismissible hike/bike bottom sheet, resolves a GPS-gated (or fallback) initial map center via `Settings.behavior?.allowAutoGeolocate`, and pushes `/route-planner` with the chosen travel profile — replacing both TEMPORARY Phase-19 stubs (the hardcoded route registration and the card's direct push).
- App-bar "Finish" action on `RoutePlannerScreen`, gated on >=2 route anchors, wired to Plan 01's `finishPlanning`; undo/redo relocated into the top-right map controls Column to free the app-bar slot Finish now occupies.

---

## v1.4 MapLibre Migration (Shipped: 2026-07-10)

**Phases completed:** 6 phases, 17 plans, 40 requirements

**Key accomplishments:**

- Native GL map rendering — `WandererMap` and all 6 map screens now run on `maplibre` (`MapLibreMap`) instead of `flutter_map`, with live light/dark style swapping, scale bar, and ODbL attribution.
- Self-hosted glyph & sprite serving — new unified `/api/v1/map/style-sources` endpoint resolves tile, glyph, and sprite URLs under operator override, fixing missing place-name labels and route-shield icons that silently failed to render before.
- Offline parity preserved — downloaded trails render basemap via native `pmtiles://` and place labels via cached `file://` glyphs/sprites in airplane mode, including multi-cell trails.
- Server-side clustering — the map screen now renders `POST /search/trails/cluster` results as native circle/symbol layers matching web's `ClusterLayer`, replacing client-side rendering.
- Turn-by-turn navigation migrated — heading-up follow, compass reset, and live location puck all run on maplibre-native APIs; offline navigation from the ObjectBox cache is unregressed.
- Both `flomp/*` forks retired — `flutter_map` + 4 plugins, `vector_map_tiles`, and `vector_tile_renderer` are gone from `pubspec.yaml`; `maplibre` is pinned to an exact version (0.3.5).

**Known deferred items at close:** 15 (see STATE.md Deferred Items — 14 are completed quick tasks the audit tool couldn't classify; 1, dark mode for the Flutter app, was planned but never executed and remains open for a future milestone).

---

## v1.2 Settings Screens (Shipped: 2026-06-29)

**Phases completed:** 4 phases, 9 plans, 12 tasks

**Key accomplishments:**

- Five-row settings list (Account/Privacy/Language/Notifications/Appearance) wired to /settings sub-routes, with a full 14-locale RadioGroup<Language> + metric/imperial switch screen that auto-saves to the server, plus two themed stub screens.
- Every format_util call site (14 files, ~50 calls) now reads the live metric/imperial preference from unitProvider, so toggling units in settings re-renders distances, elevations, and speeds across trail cards, lists, navigation, and the elevation profile — backed by new imperial conversion tests.
- 1. [Rule 3 - Blocking] Plural ARB getters require positional, not named, arguments
- Added image_picker ^1.2.2 dependency, iOS photo-library plist key, `AppLocalizations.account` l10n getter, and `Auth.refresh()` Riverpod notifier method as prerequisites for Plans 02 and 03.
- Two ConsumerStatefulWidget bottom-sheet forms for credential changes: EmailChangeSheet posts {email, currentPassword} to /user/$id/email and refreshes auth; PasswordChangeSheet posts {oldPassword, password, passwordConfirm} to /user/$id.
- Filled SettingsAccountScreen stub with all five ACCT sections: CircleAvatar gallery upload with multipart POST, change-aware bio TextField, email/password modal sheets, and AlertDialog-gated account deletion with logout.
- Filled the stub SettingsNotificationsScreen with 9 sections of independent Web/Email SwitchListTile toggles that auto-save to the server via the D-09 map-copy pattern, plus a new l10n.web ARB key and a widget test.

---
