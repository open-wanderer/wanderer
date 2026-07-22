# Phase 23: TileRepositoryManager — Download Engine - Research

**Researched:** 2026-07-22
**Domain:** Flutter/Dart mobile download engine (Dio resumable downloads, ObjectBox-backed state, app-lifecycle-aware pause/resume, disk-space guarding), consuming Phase 21.5's backend region-archive API and Phase 22's data model
**Confidence:** HIGH (stack/architecture — directly verified against in-repo source and official package docs); MEDIUM (platform-specific disk-space/backgrounding behavior — verified against docs but not on-device)

## Summary

`TileRepositoryManager` is the first genuinely new download-primitive work this milestone requires: unlike `trail_download_service.dart`'s small per-cell `.pmtiles` files (which tolerate "just retry the whole file"), region archives are 10s-100s of MB and must resume from a byte offset, refuse to start when disk space is tight, and treat app backgrounding as an expected pause rather than a frozen socket. None of this needs a new HTTP or storage dependency — `dio` ^5.9.2 (already pinned) supports `Range` header + `FileAccessMode.append` resume today, `pmtiles` ^1.2.0 (already pinned) has a ready-made `PmTilesArchive.fromFile()` header/magic-byte validator that this phase should reuse rather than hand-roll, and Flutter's SDK-provided `AppLifecycleListener` (no package) can observe app lifecycle from a plain Riverpod-owned service class without a widget. The one thing genuinely missing from the stack is a disk-free-space check — no in-repo utility exists, and this requires a small, unverified third-party plugin (flagged `[ASSUMED]` below, gated behind human verification).

The most consequential finding from this research is **not** in the Flutter code: the SvelteKit proxy routes the app actually calls (`web/src/routes/api/v1/regions/[id]/download{,-dem}/+server.ts`) currently drop the incoming `Range` request header and hardcode a `200` response status, even though the Go backend they proxy to (`db/routes/regions_get.go`, using PocketBase's `e.FileFS`) fully supports HTTP Range requests already. As written today, a resumed download from the Flutter app would silently receive the *entire* file again with `status: 200`, defeating TILE-02 end-to-end even though the Flutter-side Dio code would be implemented correctly. This is a "SvelteKit — regions proxy" prerequisite the plan must include as an explicit task, even though the phase's primary surface is `app/lib/`.

**Primary recommendation:** Build `TileRepositoryManager` as a Riverpod-provider-owned Dart class mirroring `TrailDownloadService`'s existing shape (constructor-injected `Store` + `Dio`, `CancelToken`-based cancellation), but add: (1) `.part` + `Range`-resume + `FileAccessMode.append` for both vector and DEM downloads, (2) a disk-space pre-check before each file write, (3) an `AppLifecycleListener`-driven deliberate-pause path (not native background download), (4) `PmTilesArchive.fromFile()` post-download validation before promoting `.part` → final path and writing `DownloadedTilePackageEntity`, and (5) a `localTilePathsForBounds(bbox)` query implemented as a hand-rolled axis-aligned rectangle-overlap test against `RegionEntity`'s four bbox fields (there is no `intersects()` helper on the app's `LngLatBounds` type — confirmed by direct source inspection).

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TILE-01 | `TileRepositoryManager` owns region download lifecycle (start/pause/resume/delete), decoupled from `Trail` | See Architecture Patterns — mirrors `TrailDownloadService`'s existing shape; state exposure mirrors `DownloadingTrailIds` |
| TILE-02 | Resumable downloads within a session via HTTP Range + `FileAccessMode.append` | See Code Examples — Dio API confirmed via official pub.dev docs; **blocked today by the SvelteKit proxy gap**, see Common Pitfalls #1 |
| TILE-03 | Disk-space checked with safety margin before each file write | See Standard Stack (disk-space plugin, `[ASSUMED]`) and Common Pitfalls #2 |
| TILE-04 | App backgrounding mid-download treated as deliberate pause, not silent failure | See Architecture Patterns — `AppLifecycleListener` (SDK built-in, no widget needed) |
| TILE-05 | `localTilePathsForBounds(bbox)` returns local vector/DEM paths for regions intersecting a bbox | See Code Examples — hand-rolled bbox-overlap helper; `RegionEntity_` query fields confirmed via `objectbox.g.dart` |
| DEM-01 | Per-region DEM toggle downloads pre-built region-scoped DEM archive from BACK-04's catalog, independent of vector | See Architecture Patterns — same download primitive as vector, keyed off `RegionEntity.demUrl`/`demPackage` |
| DEM-02 | DEM download/deletion tracked as its own `DownloadedTilePackage`, independent of vector | Already modeled in Phase 22's `RegionEntity.demPackage` `ToOne` — this phase only needs to populate/mutate it |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Region archive file streaming (Range support) | API / Backend (Go, done in Phase 21.5) | Frontend Server (SSR proxy — **needs a fix this phase depends on**) | `db/routes/regions_get.go`'s `e.FileFS` already serves Range requests; the SvelteKit passthrough route is the actual client-facing endpoint and currently does not forward Range in or 206/Content-Range out |
| Download orchestration (start/pause/resume/delete, resume-from-offset) | Mobile client (`app/lib/services/tile_repository_manager.dart`, new) | — | Pure client-side download-primitive concern; no server state needed beyond the existing static-file endpoint |
| Disk-space pre-check | Mobile client (Browser-equivalent: on-device OS) | — | Must run immediately before each file write, entirely local |
| App-lifecycle-aware pause | Mobile client | — | `AppLifecycleState`/`AppLifecycleListener` is an OS-relayed signal consumed entirely on-device |
| Persisted download/package status | Mobile client (ObjectBox, Phase 22 schema) | — | `DownloadedTilePackageEntity`/`RegionEntity` already modeled; this phase only writes to it |
| bbox → local path query (TILE-05) | Mobile client | — | Pure local ObjectBox query + arithmetic; no network involved |
| Riverpod state exposure for Phase 24's UI | Mobile client (state layer) | — | Mirrors `DownloadingTrailIds` — a `keepAlive` provider exposing per-region status/progress that Phase 24 subscribes to |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|---------------|
| `dio` | ^5.9.2 (pinned, existing) [VERIFIED: npm registry — n/a, confirmed via in-repo `pubspec.yaml` + official pub.dev API docs] | HTTP client; `Range` header + `FileAccessMode.append` resumable download | Already the app's only HTTP client (`trail_download_service.dart`, `region_provider.dart`); `FileAccessMode` and manual `Range`-header resume are documented, current pub.dev API surface — no version bump needed |
| `pmtiles` | ^1.2.0 (pinned, existing) [VERIFIED: npm registry — n/a, confirmed via direct source inspection of the installed package] | Post-download integrity validation via `PmTilesArchive.fromFile()` / `Header.validate()` (magic bytes + version + zoom sanity) | Already a pinned dependency (offline rendering); its header parser is the correct tool for "is this .pmtiles file truncated/corrupt," not a hand-rolled byte check |
| `objectbox` / `objectbox_flutter_libs` | ^5.3.1 (pinned, existing) | Persisted `RegionEntity`/`DownloadedTilePackageEntity` reads/writes | Phase 22 already built the schema this phase writes to |
| `path_provider` | ^2.1.5 (pinned, existing) | Resolves `getApplicationDocumentsDirectory()` for the region storage root | Same directory root already used by `trail_download_service.dart` |
| `flutter` SDK (`AppLifecycleListener`) | Flutter SDK, no package | Detects app backgrounding/foregrounding from outside a widget tree | Built into the Flutter framework (`package:flutter/widgets.dart`); confirmed via official Flutter API docs to work standalone (accepts an optional `binding` param, `dispose()` when done) — no third-party plugin needed for TILE-04 |
| `riverpod`/`riverpod_annotation` | ^3.3.1 (pinned, existing) | Provider wrapping `TileRepositoryManager` + per-region status/progress state | Matches every other service in this codebase (`trailDownloadServiceProvider`, `regionRepositoryProvider`) |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `disk_space_2` | 1.0.12 (latest on pub.dev as of this research) `[ASSUMED]` | Pre-download available-storage check (TILE-03) | Only genuinely new dependency this phase needs; see Package Legitimacy Audit below — gate behind `checkpoint:human-verify` before adding to `pubspec.yaml` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `disk_space_2` | `storage_space` (1.2.0, verified publisher) | No path-specific free-space query documented (device-wide only) — less precise for a specific storage volume, but from a "verified publisher" account on pub.dev, which is a modest trust signal `disk_space_2` lacks |
| `disk_space_2`/`storage_space` (both) | Hand-rolled platform channel calling `NSFileManager.attributesOfFileSystem`/`StatFs` directly | Removes third-party dependency risk entirely, but duplicates work a maintained plugin already does; only worth it if both candidate packages fail a manual code-quality read during the `checkpoint:human-verify` gate |
| Hand-rolled resumable-download wrapper | `background_downloader` package (mentioned in milestone-level STACK.md research as an alternative) | Adds true OS-level background download sessions — explicitly out of scope per PROJECT.md ("Background/resumable downloads across app restarts" is out of scope; this milestone is session-scoped only), so the added complexity buys nothing here |
| Hand-rolled PMTiles magic-byte check (read first N bytes, compare to `"PMTiles"` literal) | `PmTilesArchive.fromFile()` (already in `pmtiles` package) | The hand-rolled version reinvents what the pinned dependency already validates (magic + version + zoom sanity) — no reason to duplicate it |

**Installation:**
```bash
# Only one new package candidate — gate behind human verification per the
# Package Legitimacy Audit below before running this.
flutter pub add disk_space_2
```

**Version verification:** `pub.dev` registry confirms `disk_space_2` latest is `1.0.12`, published `2026-01-05` (per `https://pub.dev/api/packages/disk_space_2`, `[VERIFIED: npm registry]`-equivalent for pub — the *existence and version* is registry-confirmed, but the *choice of this specific package* was sourced from WebSearch, so its recommendation itself stays `[ASSUMED]` per the package-name-provenance rule — registry existence alone does not upgrade a WebSearch-discovered name to VERIFIED).

## Package Legitimacy Audit

> `slopcheck` does not support the Dart/pub.dev ecosystem (`slopcheck install --help` lists only `pypi, npm, crates.io, go, rubygems, maven, packagist`). It could not be run against `disk_space_2`. Per the graceful-degradation rule, this package is tagged `[ASSUMED]` and the planner must gate its install behind a `checkpoint:human-verify` task.

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|--------------|-----------|-------------|
| `disk_space_2` | pub.dev | latest `1.0.12` published 2026-01-05 (package itself first published 2024-06-16) | ~19k (per pub.dev popularity data) | `github.com/tom-ludwig/disk_space_2` (per pubspec `repository` field) — a fork lineage from `github.com/activcoding/Disk-Space` | N/A — ecosystem unsupported by slopcheck | `[ASSUMED]` — planner must insert `checkpoint:human-verify` before `flutter pub add` |

**Packages removed due to slopcheck `[SLOP]` verdict:** none (slopcheck did not run against this ecosystem)
**Packages flagged as suspicious `[SUS]`:** none flagged by tooling, but `disk_space_2` is a community fork of an unmaintained original (`disk_space`) with a low-effort-looking homepage (`activecoding.de.cool`) — worth a human glance at the plugin's native (Kotlin/Swift) source before adding, given it will run with filesystem-adjacent native code on both platforms.

*All packages in this audit are tagged `[ASSUMED]` — the planner must gate the install behind a `checkpoint:human-verify` task.*

## Architecture Patterns

### System Architecture Diagram

```
Settings UI (Phase 24, not this phase)
        │  start(regionId) / pause(regionId) / resume(regionId) / delete(regionId)
        ▼
┌──────────────────────────────┐
│  TileRepositoryManager        │◄──── AppLifecycleListener.onPause/onResume
│  (Riverpod keepAlive provider)│         (deliberate pause/resume, TILE-04)
│                                │
│  - per-region CancelToken map │
│  - disk-space pre-check ──────┼──► disk_space_2 (or platform channel)
│  - .part + Range resume ──────┼──► Dio.download(fileAccessMode: append,
│                                │       options: Options(headers: {Range}))
│  - post-download validate ────┼──► pmtiles.PmTilesArchive.fromFile()
│  - status writes ─────────────┼──► ObjectBox: DownloadedTilePackageEntity
│                                │       (batched, single runInTransaction)
│  - localTilePathsForBounds() ─┼──► ObjectBox: RegionEntity query +
│                                │       hand-rolled bbox-overlap test
└──────────────┬────────────────┘
               │  GET /api/v1/regions/{id}/download[-dem]
               ▼
   SvelteKit proxy route (web/src/routes/api/v1/regions/[id]/download{,-dem}/+server.ts)
        ⚠ currently drops incoming Range header, hardcodes status:200 —
          MUST be patched to forward Range in and 206/Content-Range out
          before TILE-02 can work end-to-end (see Common Pitfalls #1)
               │
               ▼
   Go backend (db/routes/regions_get.go) — e.FileFS, already Range-capable
```

### Recommended Project Structure

```
app/lib/
├── services/
│   └── tile_repository_manager.dart   # new — TILE-01..05, DEM-01/02 core logic
├── provider/
│   └── region/
│       ├── region_provider.dart        # existing (Phase 22) — catalog fetch/upsert
│       └── tile_repository_provider.dart  # new — Riverpod wrapper + per-region
│                                          #   status/progress state, mirrors
│                                          #   trail_download_state_provider.dart
├── util/
│   └── region_file_path.dart           # new — path-safety helper, mirrors
│                                          #   map_cache_path.dart's pattern
├── entities/
│   ├── region_entity.dart              # existing (Phase 22) — read/write target
│   └── downloaded_tile_package_entity.dart  # existing (Phase 22)
```

### Pattern 1: `.part` file + Range-header resume

**What:** Download to a sibling `.part` path; on resume, `File(partPath).length()` gives the byte offset to request via `Range: bytes=<offset>-`; `FileAccessMode.append` tells Dio to append rather than overwrite. Only rename `.part` → final path after post-download validation succeeds.
**When to use:** Every vector/DEM region archive download (TILE-02).
**Example:**
```dart
// Source: pub.dev official API docs (dio/Dio/download.html, dio/FileAccessMode.html)
Future<void> _downloadResumable({
  required Dio dio,
  required String url,
  required String partPath,
  required CancelToken cancelToken,
  required void Function(int received, int total) onProgress,
}) async {
  final partFile = File(partPath);
  final alreadyDownloaded =
      await partFile.exists() ? await partFile.length() : 0;

  await dio.download(
    url,
    partPath,
    cancelToken: cancelToken,
    fileAccessMode:
        alreadyDownloaded > 0 ? FileAccessMode.append : FileAccessMode.write,
    // deleteOnError defaults to true — MUST override to false when resuming,
    // otherwise Dio deletes the .part file (and all resume progress) the
    // next time a resumed request itself fails (e.g. network drop again).
    deleteOnError: alreadyDownloaded == 0,
    options: alreadyDownloaded > 0
        ? Options(headers: {'range': 'bytes=$alreadyDownloaded-'})
        : null,
    onReceiveProgress: (received, total) {
      // `received` is bytes received THIS request only when resuming —
      // add `alreadyDownloaded` to get the true total for progress UI.
      onProgress(alreadyDownloaded + received, total < 0 ? -1 : alreadyDownloaded + total);
    },
  );
}
```

### Pattern 2: Post-download validation before promotion

**What:** Never trust "file exists" as "download complete" — open the `.part` file with `pmtiles`'s own archive loader before renaming to the final path or writing `DownloadedTilePackageEntity`.
**When to use:** Immediately after every completed vector/DEM download, before the file is considered `downloaded`.
**Example:**
```dart
// Source: direct inspection of installed pmtiles 1.2.0
// (lib/src/archive.dart PmTilesArchive.fromFile, lib/src/header.dart Header.validate)
Future<bool> _isValidPmTiles(String path) async {
  try {
    final archive = await PmTilesArchive.fromFile(File(path));
    await archive.close();
    return true;
  } on CorruptArchiveException {
    return false;
  } on UnsupportedError {
    return false;
  }
}
```

### Pattern 3: App-lifecycle-aware deliberate pause (no widget required)

**What:** `AppLifecycleListener` (Flutter SDK, `package:flutter/widgets.dart`) can be instantiated directly inside `TileRepositoryManager` — a plain Dart class, not a `State` — because it accepts an optional `binding` and exposes `onPause`/`onResume` callbacks without requiring a `WidgetsBindingObserver` mixin on a widget.
**When to use:** TILE-04 — cancel (not silently freeze) any in-flight download's `CancelToken` on backgrounding, mark the affected region `paused` (a new status value, appended — never inserted — to `PackageStatus`, per Phase 22's explicit-int enum contract), and offer resume on `onResume`.
**Example:**
```dart
// Source: official Flutter API docs (api.flutter.dev/flutter/widgets/AppLifecycleListener-class.html)
class TileRepositoryManager {
  TileRepositoryManager(this._store, this._api) {
    _lifecycleListener = AppLifecycleListener(
      onPause: _pauseAllActiveDownloads,
      onResume: _offerResumeForPausedDownloads,
    );
  }

  late final AppLifecycleListener _lifecycleListener;

  void dispose() => _lifecycleListener.dispose();

  void _pauseAllActiveDownloads() {
    for (final cancelToken in _activeCancelTokens.values) {
      // Deliberate cancel — distinct from a user-initiated cancel — so the
      // status write can distinguish "paused by backgrounding" from
      // "cancelled by user" (Pitfall 3's status-semantics warning).
      cancelToken.cancel('app-backgrounded');
    }
  }
}
```

### Pattern 4: `localTilePathsForBounds` — hand-rolled bbox overlap

**What:** `LngLatBounds` (re-exported by `maplibre` from `maplibre_platform_interface`, `lib/src/lng_lat_bounds.dart`) is a plain 4-field data class with **no** `intersects()`/overlap method — confirmed by direct source inspection. `geobase`'s `GeoBox` (also re-exported by `maplibre`) *does* have `intersects(Box other)`, but `RegionEntity` stores bbox as four raw `double` fields, not a `GeoBox`/`LngLatBounds` object, so the simplest correct implementation is a direct arithmetic overlap test against those four fields — no conversion needed.
**When to use:** TILE-05.
**Example:**
```dart
// Source: direct inspection of RegionEntity (Phase 22) + maplibre_platform_interface
// 0.3.5's LngLatBounds (no intersects() method exists on this type)
bool _bboxOverlaps(RegionEntity region, LngLatBounds query) {
  return !(region.maxLon < query.longitudeWest ||
      region.minLon > query.longitudeEast ||
      region.maxLat < query.latitudeSouth ||
      region.minLat > query.latitudeNorth);
}

List<String> localTilePathsForBounds(LngLatBounds bbox) {
  final box = _store.box<RegionEntity>();
  final paths = <String>[];
  for (final region in box.getAll()) {
    if (!_bboxOverlaps(region, bbox)) continue;
    final vectorPath = region.vectorPackage.target?.localFilePath;
    final demPath = region.demPackage.target?.localFilePath;
    if (vectorPath != null) paths.add(vectorPath);
    if (demPath != null) paths.add(demPath);
  }
  return paths;
}
```

### Anti-Patterns to Avoid

- **Treating `File(path).exists()` as "download complete"** — the existing trail-scoped code does this and it's fine at trail-cell scale; at region scale a `.part`-in-progress file existing under the *final* path name would be silently treated as cached. Always write to `.part`, only rename after validation (Pitfall 1).
- **Letting Dio's `deleteOnError: true` default apply on a resumed download** — it will delete the very `.part` file the resume is trying to preserve, the first time a resumed request itself fails. Explicitly set `deleteOnError: false` whenever `fileAccessMode: FileAccessMode.append` is used.
- **Persisting download status via `.index`** — Phase 22 already established the explicit-`.code` pattern (`PackageStatus`); any new status (e.g. `paused`) this phase adds must be appended with a new int, never inserted.
- **Passing ObjectBox objects/queries across an isolate boundary** — if a future iteration moves the download loop to a background isolate, `Store.attach()` a new `Store` inside that isolate rather than sharing entities directly (documented ObjectBox constraint, not spiked in this phase since the trail-scoped precedent runs downloads on the main isolate without visible jank).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| PMTiles file corruption/truncation detection | A custom byte-offset magic-number check | `pmtiles` package's `PmTilesArchive.fromFile()` / `Header.validate()` | Already a pinned dependency; validates magic bytes, spec version, and zoom-range sanity — more thorough than a hand-rolled check and zero new code to maintain |
| Resumable HTTP download | A custom byte-range HTTP client wrapper from scratch | `dio`'s existing `Range` header + `FileAccessMode.append` support | Already the app's HTTP client; no new dependency, officially documented API |
| App-lifecycle observation from a non-widget service | A custom platform channel or polling loop | `AppLifecycleListener` (Flutter SDK) | Built into the framework since it was designed for exactly this — non-widget lifecycle observation — confirmed usable standalone via official docs |
| Disk free-space query | A custom platform channel to `NSFileManager`/`StatFs` | `disk_space_2` (or `storage_space`) `[ASSUMED]`, pending `checkpoint:human-verify` | A maintained (if small) plugin already wraps the correct native APIs on both platforms; only fall back to a hand-rolled channel if the human-verify gate rejects both candidates |

**Key insight:** This phase's only genuinely new capability (disk-space checking) is also the only piece without an existing in-repo pattern to mirror — everything else (resumable download, corruption check, lifecycle awareness, ObjectBox status writes) either already exists as a pinned dependency capability or has a directly analogous pattern already proven elsewhere in this codebase (`trail_download_service.dart`, `map_cache_path.dart`, `navigation_screen.dart`'s `didChangeAppLifecycleState`).

## Common Pitfalls

### Pitfall 1: The SvelteKit regions-download proxy drops Range support end-to-end (blocks TILE-02)

**What goes wrong:** `web/src/routes/api/v1/regions/[id]/download/+server.ts` and its `download-dem` sibling call `event.fetch()` against the internal Go backend forwarding only an `Authorization` header — never the client's incoming `Range` header — and then construct the client-facing `Response` with a hardcoded `status: 200`, never inspecting or forwarding the upstream response's actual status (which would be `206 Partial Content` when Range is honored) or its `Content-Range` header. The Go backend (`db/routes/regions_get.go`) is already Range-capable via PocketBase's `e.FileFS` (confirmed by its own doc comment: "e.FileFS serves via the stdlib file server, which supports HTTP Range requests — Phase 23's resumable download engine (TILE-02) relies on this"), so the gap is entirely in the SvelteKit passthrough layer the app actually talks to.

**Why it happens:** These proxy routes were written in Phase 21.5 purely to stream a complete file with auth forwarding — Range forwarding wasn't needed until this phase's resumable-download requirement existed, so it was reasonably out of scope at the time.

**How to avoid:**
- Forward the incoming request's `Range` header (if present) into the `event.fetch()` call's headers.
- Construct the client-facing `Response` using the upstream response's actual `status` (200 or 206), and forward `Content-Range`/`Accept-Ranges`/`Content-Length` headers verbatim when present, not just `Content-Length`.
- Add this as an explicit task in the plan even though it touches `web/src/routes/`, not `app/lib/` — TILE-02's success criterion ("resuming continues from a partial file via HTTP Range... not from byte 0") is unverifiable without it.

**Warning signs:** A resumed download's byte count doesn't match `alreadyDownloaded + newBytes`; the server always returns the full `Content-Length` regardless of the sent `Range` header; `FileAccessMode.append` silently produces a file *larger* than the archive's real size (duplicated bytes from appending a full re-download onto existing partial bytes).

**Phase to address:** This phase (23) — it is a hard prerequisite for TILE-02, not a follow-up.

---

### Pitfall 2: `deleteOnError` defaults to `true` and will destroy resume progress

**What goes wrong:** Dio's `download()` defaults `deleteOnError: true` — if this default is left in place while also passing `fileAccessMode: FileAccessMode.append`, the very first time a *resumed* download itself fails (e.g. the user's connection drops again), Dio deletes the `.part` file it just appended to, discarding all prior progress and forcing a full restart from byte 0 anyway — silently defeating the entire point of TILE-02.

**Why it happens:** `deleteOnError` and `fileAccessMode` are independent parameters with no built-in interaction warning; the "delete broken files" default makes sense for `FileAccessMode.write` but is actively harmful for `FileAccessMode.append`.

**How to avoid:** Explicitly set `deleteOnError: false` whenever resuming (`fileAccessMode: FileAccessMode.append`); only allow `deleteOnError: true` on a fresh (`FileAccessMode.write`) download attempt.

**Warning signs:** A `.part` file that shrinks back to 0 bytes after a second network failure; users reporting a download "restarts from 0% every time it fails," even after a resume fix was supposedly shipped.

**Phase to address:** This phase — same download-primitive code path as Pattern 1.

---

### Pitfall 3: Disk-space check API choice differs meaningfully between iOS and Android (inherited from milestone-level PITFALLS.md, re-verified here)

**What goes wrong:** iOS's "available" capacity accounting differs from Android's raw `getFreeSpace()` — a naive `freeBytes > downloadSize` check using the wrong semantics on either platform under- or over-estimates real availability. See milestone `PITFALLS.md` Pitfall 2 for full detail; re-confirmed here that no in-repo utility currently exists for either platform, so this phase introduces the first disk-space check in the codebase.

**How to avoid:** Require a safety margin (milestone research suggests 1.5-2x the declared manifest size, accounting for the coexisting `.part` file during resume plus OS reserve) rather than a bare `free > size` comparison; re-check before each file in a region's vector+DEM pair, not just once at the start of the whole region download.

**Phase to address:** This phase — `TileRepositoryManager`'s pre-write check, per TILE-03.

---

### Pitfall 4: Region id must be defensively validated before it reaches a file path, even though the catalog is currently trusted

**What goes wrong:** `RegionEntity.id` flows from the fetched catalog (Phase 22) directly into this phase's on-disk directory naming (e.g. `<appDocs>/regions/<id>/vector.pmtiles`). The backend already constrains ids server-side (`^[a-z0-9][a-z0-9_-]*$` in both the Go `IsValidRegionID` and the SvelteKit `RegionIdSchema`), and Phase 22's `RegionEntity.fromCatalogEntry`/`applyCatalogEntry` already reject malformed bbox shapes — but neither of those validates the `id` string itself against path-traversal characters before this phase turns it into a directory name.

**How to avoid:** Reuse (or closely mirror) the existing `_assertSafePath` pattern from `offline_style_rewriter.dart` / the whitelist-regex pattern from `map_cache_path.dart` — validate `id` against the same `^[a-z0-9][a-z0-9_-]*$` regex the backend already enforces before building any path from it, rejecting with `ArgumentError` rather than silently trusting server-declared strings.

**Phase to address:** This phase — at region-storage-path construction time, mirroring milestone `PITFALLS.md`'s Security Mistakes table entry on this exact topic.

---

### Pitfall 5 (inherited, re-scoped to this phase): Backgrounding mid-download, disk-space pre-check, and progress-write batching

See milestone `PITFALLS.md` Pitfalls 1, 2, 3, 7, 8 for full detail — all five are explicitly scoped to "the phase that builds `TileRepositoryManager`'s download path," i.e. this phase. Summarized here for planning:
- **No resume (Pitfall 1):** addressed by Pattern 1/2 above.
- **No disk-space check (Pitfall 2):** addressed by TILE-03 + the `disk_space_2` candidate above.
- **Backgrounding (Pitfall 3):** addressed by Pattern 3 above (`AppLifecycleListener`).
- **No cascade delete (Pitfall 7):** region deletion in this phase must explicitly delete the `DownloadedTilePackageEntity` row(s) *and* the on-disk file(s) as one unit — ObjectBox does not cascade.
- **Progress-write races (Pitfall 8):** batch ObjectBox status writes (e.g. every N% or every second, not every byte-progress callback), always updating `status` + byte counters inside one `runInTransaction`, mirroring the pattern already visible in `trail_download_service.dart`'s `_store.runInTransaction(TxMode.write, () { box.put(entity); })`.

## Code Examples

### Disk-space pre-check (pattern, exact API pending human-verify of the chosen package)

```dart
// Source: WebSearch-derived pub.dev listing for disk_space_2 1.0.12 — [ASSUMED],
// exact static method availability (path-specific vs device-wide) not confirmed
// against the package's actual source, only its documented example.
Future<bool> _hasEnoughSpace(String targetDirPath, int declaredSizeBytes) async {
  final freeMb = await DiskSpace.getFreeDiskSpace; // documented example returns MB
  if (freeMb == null) return false; // fail closed: refuse rather than risk a partial write
  final freeBytes = freeMb * 1024 * 1024;
  const safetyMultiplier = 1.75; // accounts for coexisting .part file + OS reserve
  return freeBytes > declaredSizeBytes * safetyMultiplier;
}
```

### Batched ObjectBox status write (mirrors existing trail-entity pattern)

```dart
// Source: pattern mirrored from trail_download_service.dart's existing
// _store.runInTransaction(TxMode.write, () { box.put(entity); }) call.
void _updatePackageStatus(
  DownloadedTilePackageEntity package, {
  required PackageStatus status,
  int? sizeBytesOnDisk,
}) {
  _store.runInTransaction(TxMode.write, () {
    package.status = status;
    if (sizeBytesOnDisk != null) package.sizeBytesOnDisk = sizeBytesOnDisk;
    _store.box<DownloadedTilePackageEntity>().put(package);
  });
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `trail_download_service.dart`'s full-file-only `_downloadTracked` (no Range, no resume) | `.part` + Range + `FileAccessMode.append`, this phase | This phase (23) | Region-sized failures no longer cost a full re-download; first resumable download path in this codebase |
| Trail downloads: no disk-space check, no lifecycle awareness (fine at cell scale) | Region downloads: mandatory pre-check + lifecycle-aware pause | This phase (23) | New failure modes (full disk, backgrounding) that were statistically irrelevant at trail-cell scale become first-class at region scale |

**Deprecated/outdated:** None — this phase does not remove or replace any existing code; `trail_download_service.dart` is untouched (its ripout is Phase 27, per the milestone roadmap).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `disk_space_2` (or `storage_space`) is the right/only reasonable disk-space-check package for this use case | Standard Stack, Package Legitimacy Audit | Low functional risk (either package's exact API can be adapter-wrapped), but a low-maintenance/low-scrutiny native plugin running filesystem-adjacent code on both platforms deserves a human look before shipping — hence the mandatory `checkpoint:human-verify` gate |
| A2 | `disk_space_2`'s exact API includes a path-specific `getFreeDiskSpaceForPath` (only confirmed via a WebSearch-synthesized description of the pub.dev listing, not the package's actual source) | Code Examples | If the path-specific variant doesn't exist as described, the plan must fall back to the device-wide `getFreeDiskSpace` call shown in the example, which is less precise about which volume is being measured but still directionally usable |
| A3 | A safety margin of 1.75x the declared archive size is an adequate heuristic for TILE-03 | Common Pitfalls #3, Code Examples | Untested on real devices; too low a margin risks a mid-write `ENOSPC`, too high needlessly blocks downloads on constrained devices — recommend treating this constant as tunable, not fixed, and validating on a real low-storage device during phase execution |

## Open Questions

1. **Exact `disk_space_2` API surface (path-specific vs. device-wide free space)**
   - What we know: The documented example shows `DiskSpace.getFreeDiskSpace`/`getTotalDiskSpace` returning device-wide MB values as `Future<double>`; the package's pub.dev description additionally claims a `getFreeDiskSpaceForPath` method.
   - What's unclear: Whether `getFreeDiskSpaceForPath` is verified to exist in the current `1.0.12` API (not confirmed against the package's actual Dart/native source in this research pass).
   - Recommendation: During plan execution, `flutter pub add disk_space_2` (after the human-verify gate) and inspect the installed package's actual public API directly before writing the disk-check call, rather than trusting the WebSearch-derived description.

2. **Whether the SvelteKit Range-forwarding fix (Pitfall 1) belongs in this phase's plan or needs its own tiny prerequisite phase/patch**
   - What we know: It's a small, mechanical change to two existing `+server.ts` files (forward one header in, forward status + two headers out).
   - What's unclear: Whether the milestone's phase-scoping convention (this phase is nominally "Flutter/Dart mobile app code") tolerates a two-file SvelteKit change inline, or whether GSD process prefers it as a separate task/commit within the same phase.
   - Recommendation: Include it as an explicit task within this phase's plan (not a separate phase) — it is a hard, load-bearing prerequisite for TILE-02's success criterion, and deferring it would let the phase's Flutter code ship un-verifiable against a real resumed download.

3. **The exact `PackageStatus`/`RegionStatus` enum additions this phase needs (e.g. `paused`, `error`/`corrupted`)**
   - What we know: Phase 22 shipped `PackageStatus.{notDownloaded, downloading, downloaded}` and `RegionStatus.{notDownloaded, downloading, downloaded, updateAvailable}` with explicit `.code` values, append-only by contract.
   - What's unclear: This phase's pause/error/corrupt-on-validation-failure states aren't yet represented — exact new value names and their `.code` numbers need to be decided during planning, not assumed here, but must follow the append-only rule (new codes only, never renumber existing ones).
   - Recommendation: Planner should explicitly enumerate the new status values needed (at minimum: paused, and a distinct error/corrupt state per Pitfall 5's status-semantics guidance) as a dedicated early task, before any download-loop code depends on them.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|-------------------|
| V5 Input Validation | yes | Region `id` validated against the existing `^[a-z0-9][a-z0-9_-]*$` allow-list (mirrors `IsValidRegionID`/`RegionIdSchema`) before any use in a file path (Pitfall 4); `RegionCatalogEntry.bbox` length already validated in Phase 22 |
| V12 Files and Resources (OWASP ASVS terminology; path traversal / file handling) | yes | Reuse the `_assertSafePath`-style pattern (`offline_style_rewriter.dart`) for all region-derived file paths; never string-concatenate a catalog-sourced value into a path |
| V6 Cryptography | no | No new cryptographic operations in this phase (downloads run over the existing cookie-authenticated `Dio` client; TLS is the transport's existing responsibility, unchanged) |
| V2 Authentication / V3 Session Management | no (indirect only) | Downloads reuse the existing authenticated `apiProvider` `Dio` instance (`dio_cookie_manager`) — no new auth surface introduced by this phase |
| V4 Access Control | no (indirect only) | Region download authorization is enforced server-side (`apis.RequireAuth()` per `db/routes/regions_get.go`'s own doc comment) — unchanged by this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| Path traversal via a hostile/misconfigured catalog `id` reaching a file path | Tampering | Allow-list regex validation before path construction (Pitfall 4) — never trust catalog-sourced strings unvalidated, even though the current catalog is admin-controlled and server-validated |
| Truncated/corrupt archive silently accepted as "downloaded" | Tampering / Denial of Service (bad map data breaks offline rendering) | Mandatory `PmTilesArchive.fromFile()` validation before promoting `.part` → final path (Pattern 2) |
| Manifest-declared size mismatched from actual bytes served (oversized response exhausting device storage) | Denial of Service | Disk-space pre-check with safety margin (TILE-03) + post-download size sanity as part of the existing PMTiles header validation |
| Resumed download re-requesting from a stale/attacker-controlled offset (not realistic here — offset is locally computed from `.part` file length, not server-supplied) | — | N/A — the `Range` offset is always derived from the app's own local `.part` file size, never from server input, so this specific threat vector doesn't apply |

## Sources

### Primary (HIGH confidence)
- In-repo: `app/lib/services/trail_download_service.dart`, `app/lib/provider/region/region_provider.dart`, `app/lib/entities/region_entity.dart`, `app/lib/entities/downloaded_tile_package_entity.dart`, `app/lib/models/region_status.dart`, `app/lib/models/region_catalog_entry.dart`, `app/lib/objectbox.g.dart`, `app/lib/util/map_cache_path.dart`, `app/lib/util/offline_style_rewriter.dart`, `app/lib/provider/trail/trail_download_state_provider.dart`, `app/lib/routes/navigation_screen.dart`, `app/pubspec.yaml`
- In-repo backend: `db/routes/regions_get.go`, `db/services/regions/config.go`, `web/src/routes/api/v1/regions/+server.ts`, `web/src/routes/api/v1/regions/[id]/download/+server.ts`, `web/src/routes/api/v1/regions/[id]/download-dem/+server.ts`
- Installed package source (direct inspection): `~/.pub-cache/hosted/pub.dev/pmtiles-1.2.0/lib/src/{archive,header,exceptions}.dart`, `~/.pub-cache/hosted/pub.dev/maplibre_platform_interface-0.3.5/lib/src/lng_lat_bounds.dart`, `~/.pub-cache/hosted/pub.dev/geobase-1.5.0/lib/src/coordinates/geographic/geobox.dart`
- [Dio `Dio.download()` API docs — pub.dev](https://pub.dev/documentation/dio/latest/dio/Dio/download.html)
- [Dio `FileAccessMode` enum — pub.dev](https://pub.dev/documentation/dio/latest/dio/FileAccessMode.html)
- [Flutter `AppLifecycleListener` class — api.flutter.dev](https://api.flutter.dev/flutter/widgets/AppLifecycleListener-class.html)
- `.planning/research/SUMMARY.md`, `.planning/research/PITFALLS.md` (milestone-level research, this phase's direct upstream input)

### Secondary (MEDIUM confidence)
- [`disk_space_2` — pub.dev](https://pub.dev/packages/disk_space_2) and its [registry API](https://pub.dev/api/packages/disk_space_2) — version/publish-date registry-confirmed; the package's *suitability* is WebSearch-derived, `[ASSUMED]`
- [`storage_space` — pub.dev](https://pub.dev/packages/storage_space) and its [registry API](https://pub.dev/api/packages/storage_space) — same caveat

### Tertiary (LOW confidence)
- None beyond what's already logged in the Assumptions Log above.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH for `dio`/`pmtiles`/`objectbox`/`AppLifecycleListener` (all verified via official docs or direct source inspection); MEDIUM for the disk-space plugin (registry-confirmed existence, WebSearch-derived suitability, explicitly gated for human verification)
- Architecture: HIGH — directly mirrors existing, already-shipped patterns in this exact codebase (`trail_download_service.dart`, `map_cache_path.dart`, `navigation_screen.dart`'s lifecycle handling)
- Pitfalls: HIGH for the SvelteKit Range-forwarding gap (directly verified by reading both the Go and SvelteKit route source) and the `deleteOnError`/`FileAccessMode.append` interaction (directly verified via official Dio docs); MEDIUM for platform-specific disk-space semantics (carried over from milestone-level PITFALLS.md, not re-verified on-device in this pass)

**Research date:** 2026-07-22
**Valid until:** 30 days (stable dependency versions; the one open item — SvelteKit proxy fix — should be resolved within this phase itself, not left to go stale)
