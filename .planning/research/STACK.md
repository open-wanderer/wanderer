# Stack Research

**Domain:** App-wide, region-based offline map tile repository (Flutter + ObjectBox + maplibre native GL)
**Researched:** 2026-07-21
**Confidence:** HIGH

> Note: This supersedes the previous `STACK.md` (v1.5 Route Planner research, dated 2026-07-16). That research is preserved in git history; this file now covers the v1.6 Offline Region Tile Repository milestone.

## Recommended Stack

### Core Technologies

No new core technologies are needed. v1.6 is a refactor/extension of already-validated infra (`dio` ^5.9.2, `objectbox`/`objectbox_flutter_libs` ^5.3.1, `maplibre` 0.3.5 pinned, `pmtiles` ^1.2.0, `path_provider` ^2.1.5), plus one small new pattern (byte-range resume) built directly on Dio's existing API surface.

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `dio` (existing) | ^5.9.2 | Resumable region-package downloads | `Dio.download()` gained a `fileAccessMode` parameter (`FileAccessMode.append`) in 5.8.0 — already satisfied by the pinned 5.9.2. Combined with a manual `Range: bytes=<offset>-` request header, this gives byte-range resume without adding a dependency. `_downloadTracked` in `trail_download_service.dart` is the direct precedent to extend, not replace. |
| `objectbox` / `objectbox_flutter_libs` / `objectbox_generator` (existing) | ^5.3.1 | `Region` + `DownloadedTilePackage` persistence | Same store (`Store _store`) the app already uses for `TrailEntity`, `WaypointEntity`, `ActiveNavigationEntity`, etc. No new database — v1.6 only adds two more `@Entity()` classes and a codegen re-run (`build_runner`). |
| `maplibre` (existing, pinned) | 0.3.5 (exact) | Multi-region tile rendering | Style JSON is a plain map of named sources; nothing in the MapLibre Style Spec limits a style to one PMTiles source. `offline_style_rewriter.dart` already proves N-source, N-layer-clone rendering across multiple `.pmtiles` cells for a single trail — the same mechanism generalizes to per-region sources. Do not bump this version as part of v1.6 (pre-1.0, breaking 0.x minors, per existing constraint). |
| `pmtiles` (existing) | ^1.2.0 | Reading each downloaded `.pmtiles` archive | Confirmed read-only (`PmTilesArchive.from`/`fromFile`/`fromReadAt`, no write/merge API) — this was already discovered for the trail-cell case and holds identically for regions. Each region's vector/DEM archive stays a separate file/source; there is no in-app merge step. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `dart:io` (`Directory`, `File`) | SDK-bundled | Disk usage calculation | Recursively walk the region-storage root (`Directory.list(recursive: true, followLinks: false)`) and sum `FileSystemEntity.stat().size` (or `File.length()`) per file. This is the same low-level API the app already uses for all download I/O (`trailDir`, `photoDir`, `tilesDir` in `trail_download_service.dart`) — no platform channel, no plugin, works identically on iOS/Android because Dart's `dart:io` file APIs are already OS-abstracted. Do not add a disk-usage plugin (see "What NOT to Use"). |
| `path_provider` (existing) | ^2.1.5 | Locating the app-wide region storage root | Reuse `getApplicationDocumentsDirectory()` exactly as `trail_download_service.dart` and `map_cache_path.dart` already do; put regions under a sibling directory (e.g. `<app-docs>/regions/<regionId>/`) instead of `<app-docs>/library/<trailId>/tiles/`. |
| `path` (existing) | ^1.9.1 | Safe path joining/validation | Already used in `offline_style_rewriter.dart`'s `_assertSafePath` (rejects `..`, relative paths, foreign URL schemes) and in `map_cache_path.dart`. Reuse the same guard for region file paths — the path-safety invariant ("no `https://` or traversal path ever reaches the style JSON") must hold for region sources exactly as it does for trail cells today. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `build_runner` (existing) | Regenerate ObjectBox bindings (`objectbox.g.dart`) and Riverpod codegen after adding `RegionEntity`/`DownloadedTilePackageEntity` and a `TileRepositoryManager` provider | No config changes needed; same `dart run build_runner build --delete-conflicting-outputs` flow already used for `TrailEntity`. |

## Installation

```bash
# No new dependencies required for v1.6 — the stack additions below are all
# patterns built on packages already in app/pubspec.yaml.

# If a byte-formatting helper is wanted for the disk-usage display, it is
# trivial enough (divide by 1024^n, pick a unit) to hand-roll in
# format_util.dart rather than adding a package — see "What NOT to Use".
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|--------------------------|
| Manual `Range` header + `FileAccessMode.append` on the existing `Dio` instance | `resumable_downloader` package (Dio-based, updated Jan 2026) | Only if the app later needs queueing, retry policies, or cross-session/background resume as first-class features beyond single-session pause/resume — which v1.6 explicitly excludes ("Background/resumable downloads across app restarts... out of scope"). For in-session resume only, the manual pattern is ~40 lines that stays inside the already-audited `_downloadTracked` code path and adds zero new supply-chain surface. |
| `dart:io` recursive directory walk for disk usage | `disk_space_2` / `storage_space` / `disk_space_plus` plugins | These report **device-wide** free/total storage (a platform-channel call), not "bytes consumed by files under `<app-docs>/regions/`". They solve a different problem (e.g. "warn before downloading a region if device is nearly full") and could be added later as a *device capacity* check, but are not a substitute for summing downloaded-region file sizes. |
| Region+package tracked as flat fields on one entity, mirrored via `ToMany<DownloadedTilePackageEntity>` relation from `RegionEntity` (like `TrailEntity.waypoints`) | Single `RegionEntity` with `vectorPath`/`demPath`/`vectorSizeBytes`/`demSizeBytes` flat fields (no second entity) | The milestone's own scope (PROJECT.md) names two entities explicitly, and a real second entity is justified: vector and DEM packages have independently pausable/resumable/deletable lifecycles and independent status (`notDownloaded`/`downloading`/`downloaded`/`updateAvailable`) — cramming both into one row means two parallel sets of status/progress/path fields with `_vector`/`_dem` prefixes, which is worse than the existing `ToMany` relation pattern the codebase already uses for exactly this "one parent, N independently-lifecycled children" shape. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|--------------|
| A new HTTP/download package (e.g. `resumable_downloader`, `background_downloader`) | Constraint explicitly forbids replacing Dio; the server (`e.FileFS` in `db/routes/map_cells_id.go`) already serves via Go's `http.ServeContent`, which handles `Range`/`If-Range` transparently — no server change needed either. Adding a second HTTP client fragments error handling, cancellation, and cookie/auth config (`dio_cookie_manager` is already wired to the shared `Dio` instance). | Extend `_downloadTracked`-style code with a `Range: bytes=<n>-` header + `Options.headers` + `fileAccessMode: FileAccessMode.append` on the existing `_api` `Dio` instance. |
| A second local database (SQLite via `sqflite`, Hive, Isar, etc.) for region/package state | Constraint explicitly forbids introducing a new database. ObjectBox already models exactly this shape (`TrailEntity` + `ToMany<WaypointEntity>` via `@Backlink`). | `RegionEntity` + `DownloadedTilePackageEntity` in the same ObjectBox `Store`. |
| Raw `int` fields for status (`int status = 0;`) instead of an enum-backed property | Loses type safety and self-documentation at every call site; not what the codebase does anywhere status-like data is stored. | The established `@Transient()` enum field + `int get dbX / set dbX` bridge property pattern — already used twice (`TrailEntity.dbDifficulty` / `TrailDifficulty`, `ActiveNavigationEntity.dbSessionType` / `ActiveSessionType`). Apply the same shape for a `RegionStatus`/`PackageStatus` enum (`notDownloaded`, `downloading`, `downloaded`, `updateAvailable`, `error`). |
| A custom multi-file `TileProvider`/`MultiPmTilesVectorTileProvider`-style abstraction for reading across regions | That class existed only because `flutter_map`'s tile-provider interface expected a single virtual tile source per layer, forcing a hand-rolled merge shim — it was deleted in v1.4 precisely because native GL doesn't need it. A MapLibre style is just a JSON map of named sources; MapLibre-native has no such single-source-per-layer constraint. | Extend `offline_style_rewriter.dart`'s existing N-source/N-layer-clone strategy (currently keyed by trail cell index) to be keyed by region id instead — same `_rewriteSourceGroup` mechanism, same `pmtiles://file://` scheme, one (or two, vector+DEM) source pair per downloaded region. |
| A disk-usage/free-space plugin as the "total space consumed by downloaded regions" source of truth | Answers a different question (device-wide free/total bytes via a platform channel), not "bytes under our regions directory." Adds a native-code dependency (Android/iOS platform implementations) for a value `dart:io` already computes. | Recursive `Directory.list(recursive: true)` + `File.length()` summation scoped to the regions root, exactly as the app already walks `trailDir`/`photoDir`/`tilesDir`. |
| Bumping `maplibre` past 0.3.5, or introducing `maplibre_gl`, as part of this milestone | Out of scope per PROJECT.md constraint ("maplibre pinned: pre-1.0 with breaking 0.x minors — pin an exact version, do not float") and per the milestone's own scope (no 3D/hillshade rendering redesign). Multi-region source support doesn't require a version bump — it's a style-JSON authoring concern, not a renderer capability gap. | Keep `maplibre: 0.3.5` exact; solve multi-region rendering entirely in `offline_style_rewriter.dart`. |

## Stack Patterns by Variant

**If a region has a downloaded vector package but the DEM toggle is off (or DEM download failed):**
- Store only a `DownloadedTilePackageEntity` row of kind `vector` under that region; no `dem` row.
- Reuse `offline_style_rewriter.dart`'s existing "drop raster-dem source and referencing layers when `demCellPaths` is empty" branch unchanged — it already does exactly this per-trail, and per-region needs no new logic, only a new source of `demCellPaths` (region DEM package paths instead of trail DEM cell paths).

**If a trail's bounding box isn't covered by any downloaded region (the "trail download guard"):**
- Compute region coverage via the same bbox-intersection style already used server-side for cell lookup (`bbox` query param format `west,south,east,north` in `_fetchCellList`), applied client-side against `RegionEntity.minLat/maxLat/minLon/maxLon` rows in ObjectBox — no new geospatial package needed for simple bbox-vs-bbox/point-in-bbox checks.

**If a download is paused mid-transfer and the app is later killed (not just backgrounded within-session):**
- Out of scope for v1.6 ("Background/resumable downloads across app restarts... pause/resume applies within a single app session only"). Do not build persistence for partial-byte offsets across process restarts; a killed-mid-download package should be treated as `notDownloaded` on next launch (delete the partial file, same "fatal → cleanup" pattern `_downloadMapTiles` already applies to trail cells).

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|------------------|-------|
| `dio` ^5.9.2 | `FileAccessMode.append` on `Dio.download()`/`downloadUri()` | Feature landed in dio 5.8.0; the pinned `^5.9.2` in `app/pubspec.yaml` already satisfies this — no version bump needed. Not available on Dio's web platform target, but this app is iOS/Android-only (per PROJECT.md: "Linux/Windows/macOS map support... app is mobile-only"), so this is not a constraint here. |
| PocketBase `e.FileFS` (Go backend, `db/routes/map_cells_id.go`) | HTTP `Range` request headers | `FileFS` serves via Go's standard-library `http.ServeContent`, which handles `Range`/`If-Range`/`If-Match` transparently — confirmed no server-side change is required to support client-driven byte-range resume against `MapCellsDownload`/`MapCellsDownloadDem` (and their future region-keyed equivalents). |
| `objectbox` / `objectbox_flutter_libs` / `objectbox_generator` ^5.3.1 | `@Backlink('region') final packages = ToMany<DownloadedTilePackageEntity>()` relation pattern | Identical relation shape to the already-working `TrailEntity.waypoints` (`@Backlink('trail')`) — no version-specific concern, this is the same ObjectBox release already generating that binding. |
| `maplibre` 0.3.5 (pinned) | Multiple named PMTiles sources in one style JSON | Style-spec-level capability, not renderer-version-gated; the existing multi-cell-per-trail rendering in `offline_style_rewriter.dart` is the in-repo proof this already works on 0.3.5. |

## Sources

- `app/lib/services/trail_download_service.dart` — existing `_downloadTracked`/Dio download pattern to extend for resume (read directly, HIGH confidence)
- `app/lib/util/offline_style_rewriter.dart` — existing multi-cell source/layer duplication strategy and `pmtiles` read-only-archive finding (read directly, HIGH confidence)
- `app/lib/entities/trail_entity.dart`, `app/lib/entities/active_navigation_entity.dart` — confirmed idiomatic ObjectBox enum-as-int (`@Transient()` + `dbX` getter/setter) and `ToMany`/`@Backlink` relation patterns already in use twice in this codebase (read directly, HIGH confidence)
- `db/routes/map_cells_id.go` — confirms server serves tiles via `e.FileFS` (read directly, HIGH confidence)
- pub.dev `dio` changelog — `FileAccessMode` support added in 5.8.0 (WebFetch, MEDIUM confidence, cross-checked against pinned `^5.9.2`)
- https://dev.to/rlazom/resume-downloads-in-flutter-with-dio-abc — Range-header + `FileAccessMode.append` resumable-download pattern on Dio (WebFetch, MEDIUM confidence — community source, but pattern is straightforward and consistent with Dio's own documented API)
- PocketBase docs/community confirmation that `FileFS`/`fileFS()` wraps `http.ServeContent`, so Range requests are handled transparently (WebSearch, MEDIUM confidence — corroborated by Go's well-documented standard-library behavior)
- pub.dev search for disk-usage packages (`disk_space_2`, `storage_space`, `disk_space_plus`, `universal_disk_space`) — confirmed these report device-wide free/total space via platform channels, not directory-scoped file-size totals (WebSearch, MEDIUM confidence)
- pub.dev `maplibre` package page / changelog — PMTiles support and source-type overview (WebSearch, MEDIUM confidence; multi-source claim itself verified HIGH confidence directly from in-repo `offline_style_rewriter.dart` behavior, not from this source)

---
*Stack research for: app-wide region-based offline map/tile management in a Flutter app*
*Researched: 2026-07-21*
