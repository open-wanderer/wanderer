# Phase 23: TileRepositoryManager — Download Engine - Pattern Map

**Mapped:** 2026-07-22
**Files analyzed:** 7
**Analogs found:** 7 / 7

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `app/lib/services/tile_repository_manager.dart` (new) | service | file-I/O + streaming (resumable HTTP download) | `app/lib/services/trail_download_service.dart` | role-match (same download-service shape; different resumability requirement) |
| `app/lib/provider/region/tile_repository_provider.dart` (new) | provider/store | event-driven (per-region status/progress state) | `app/lib/provider/trail/trail_download_state_provider.dart` | exact (keepAlive Riverpod notifier wrapping a download service) |
| `app/lib/util/region_file_path.dart` (new) | utility | transform (path-safety) | `app/lib/util/map_cache_path.dart` (and `_assertSafePath` in `app/lib/util/offline_style_rewriter.dart`) | exact (whitelist-regex / traversal-guard pattern) |
| `app/lib/models/package_status.dart` / `app/lib/models/region_status.dart` (append new enum values) | model | CRUD (persisted enum shadow) | `app/lib/entities/downloaded_tile_package_entity.dart` (`PackageStatus` `.code` shadow) | exact |
| `app/lib/entities/region_entity.dart` (modify — `localTilePathsForBounds` query support) | model | CRUD + transform (bbox overlap query) | itself (existing bbox fields) — pattern precedent from `offline_style_rewriter.dart`'s multi-cell reasoning | role-match |
| `web/src/routes/api/v1/regions/[id]/download/+server.ts` (modify — Range forwarding) | route (SvelteKit server proxy) | streaming (request-response, byte-range passthrough) | itself (existing file, in-place fix) — structurally identical to `download-dem/+server.ts` | exact (near-duplicate sibling route) |
| `web/src/routes/api/v1/regions/[id]/download-dem/+server.ts` (modify — Range forwarding) | route (SvelteKit server proxy) | streaming (request-response, byte-range passthrough) | `web/src/routes/api/v1/regions/[id]/download/+server.ts` (sibling, same fix applied identically) | exact |

## Pattern Assignments

### `app/lib/services/tile_repository_manager.dart` (service, file-I/O/streaming)

**Analog:** `app/lib/services/trail_download_service.dart`

**Imports pattern** (lines 1-16):
```dart
import 'dart:async';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:objectbox/objectbox.dart';
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/entities/downloaded_tile_package_entity.dart';
```
Constructor takes `Store` + `Dio` exactly like `TrailDownloadService(this._store, this._api)` (line 39) — mirror this for `TileRepositoryManager(this._store, this._api)`.

**Cancellation pattern** (throughout `trail_download_service.dart`): every long-running call accepts an optional `CancelToken? cancelToken` param and threads it through every `_api.get`/`_api.download` call, and treats `DioException` + `CancelToken.isCancel(e)` as a distinct "must rethrow, never treat as a swallow-and-continue failure" case (lines 200-204, 339-345, 478-480). `TileRepositoryManager` should keep a `Map<String, CancelToken>` keyed by region id (per Architecture Patterns' "per-region CancelToken map") rather than a single token, since multiple regions can download concurrently.

**Core resumable-download pattern** (new — no direct in-repo analog; use RESEARCH.md Pattern 1 verbatim, source: pub.dev `Dio.download`/`FileAccessMode` official docs):
```dart
final partFile = File(partPath);
final alreadyDownloaded = await partFile.exists() ? await partFile.length() : 0;

await dio.download(
  url,
  partPath,
  cancelToken: cancelToken,
  fileAccessMode: alreadyDownloaded > 0 ? FileAccessMode.append : FileAccessMode.write,
  deleteOnError: alreadyDownloaded == 0, // MUST be false when resuming — see Pitfall 2
  options: alreadyDownloaded > 0
      ? Options(headers: {'range': 'bytes=$alreadyDownloaded-'})
      : null,
  onReceiveProgress: (received, total) {
    onProgress(alreadyDownloaded + received, total < 0 ? -1 : alreadyDownloaded + total);
  },
);
```
Contrast with the existing, non-resumable `_downloadTracked` in `trail_download_service.dart` (lines 413-446) — that method always does a fresh `_api.download` with no `Range`/`FileAccessMode`, and a fake-progress timer for missing `Content-Length`; TileRepositoryManager should keep the "graceful missing-Content-Length" idea (`total < 0` guard) but does NOT need the fake-progress-timer trick, since region archives always report `Content-Length` from the Go backend's `e.FileFS`.

**Cleanup-on-failure pattern** (lines 351-356, 340-345 of `trail_download_service.dart`):
```dart
} on DioException {
  if (await File(localPath).exists()) {
    await File(localPath).delete();
  }
  rethrow;
}
```
Adapt for the new engine: on a **fresh** (non-resumed) failure, delete the `.part` file; on a **resumed** failure, explicitly do NOT delete it (that's the entire point of `deleteOnError: false` — see Pitfall 2). Only delete `.part` on cancellation-that-is-a-true-user-cancel (not an app-backgrounding pause).

**Post-download validation pattern** (new — no in-repo analog; from RESEARCH.md Pattern 2, source: direct inspection of installed `pmtiles` 1.2.0):
```dart
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
Only rename `.part` → final path (and only then write `DownloadedTilePackageEntity`) after this returns `true` — mirrors the "never trust file-exists as complete" anti-pattern warning in RESEARCH.md.

**Batched ObjectBox status write pattern** (mirrors `trail_download_service.dart` lines 206-208):
```dart
_store.runInTransaction(TxMode.write, () {
  box.put(entity);
});
```
Applies per RESEARCH.md's Pitfall 5 guidance: batch progress writes (every N% or every second), always updating `status` + byte counters inside one `runInTransaction`, never once per raw progress callback.

**App-lifecycle pause pattern** (new capability, no non-widget precedent in-repo; closest analog is `navigation_screen.dart`'s widget-based `didChangeAppLifecycleState`):
```dart
// app/lib/routes/navigation_screen.dart lines 588-604 (widget-based, for contrast)
@override
void didChangeAppLifecycleState(AppLifecycleState state) {
  switch (state) {
    case AppLifecycleState.resumed:
      unawaited(_positionSource.setForeground(true));
      _startHeadingSub();
    case AppLifecycleState.paused:
    case AppLifecycleState.inactive:
    case AppLifecycleState.detached:
    case AppLifecycleState.hidden:
      unawaited(_positionSource.setForeground(false));
  }
}

@override
void dispose() {
  WidgetsBinding.instance.removeObserver(this);
  // ... cancel every subscription/controller ...
  super.dispose();
}
```
`TileRepositoryManager` is a plain Dart class (not a `State`), so it must use `AppLifecycleListener` (SDK, no `WidgetsBindingObserver` mixin needed) instead — see RESEARCH.md Pattern 3 for the exact API. The **dispose discipline** shown above (unregister the observer/listener, cancel every timer/subscription) is the pattern to copy regardless of API: `TileRepositoryManager` needs its own `dispose()` that calls `_lifecycleListener.dispose()` and cancels every in-flight `CancelToken`.

**Region id path-safety pattern** — reuse `_assertSafePath`-style validation before ANY path is built from `RegionEntity.id`:
```dart
// app/lib/util/offline_style_rewriter.dart lines 271-281 (adapt: also add the
// backend's ^[a-z0-9][a-z0-9_-]*$ allow-list check for a bare id, since a
// region id is not itself a path yet — mirror map_cache_path.dart's
// _rangePattern regex-reject style, lines 22-29 and 44-50)
void _assertSafePath(String path, String label) {
  if (path.contains('://')) {
    throw ArgumentError.value(path, label, 'must not carry a URL scheme');
  }
  if (!p.isAbsolute(path)) {
    throw ArgumentError.value(path, label, 'must be an absolute path');
  }
  if (p.split(path).contains('..')) {
    throw ArgumentError.value(path, label, 'must not contain a ".." segment');
  }
}
```

---

### `app/lib/provider/region/tile_repository_provider.dart` (provider, event-driven)

**Analog:** `app/lib/provider/trail/trail_download_state_provider.dart` (full file, 88 lines — small enough to copy the whole shape)

**Core pattern** (lines 17-20, 22-26 adapted):
```dart
@Riverpod(keepAlive: true)
class TileRepositoryStatus extends _$TileRepositoryStatus {
  @override
  Map<String, RegionDownloadState> build() => {};
  // per-region status/progress map, mirrors DownloadingTrailIds' Set<String>
  // but needs a richer per-id value (status + progress), not just membership

  Future<void> download(String regionId) async {
    if (state[regionId]?.isActive == true) return;
    // ... update state map, call tileRepositoryManagerProvider, catch/finally
    // to always clear the in-flight marker, exactly like lines 22-75 of the
    // analog ...
  }
}
```
Key behaviors to copy verbatim from the analog:
- Idempotent re-entry guard (`if (state.contains(...)) return;`, lines 23-24) — prevents duplicate concurrent downloads of the same id from two UI entry points.
- `try { ... } catch (e) { ... } finally { state = {...state}..remove(id); }` (lines 46-75) — guarantees the "in progress" flag always clears, success or failure.
- `keepAlive: true` on the provider annotation (line 17) — state must survive widget rebuild/unmount mid-download.
- Best-effort side actions (there: glyph cache warm, lines 35, 80-85) are fired-and-awaited **separately** from the main try/catch so their failure never taints the primary operation's success/failure path — the same isolation discipline should apply to progress-notification / toast side effects here.

---

### `app/lib/util/region_file_path.dart` (utility, transform)

**Analog:** `app/lib/util/map_cache_path.dart` (full file, 63 lines)

**Whitelist-regex path-safety pattern** (lines 22-29, 36-52):
```dart
final RegExp _rangePattern = RegExp(r'^\d+-\d+$');

bool isAllowedFontstack(String fontstack) => allowedFontstacks.contains(fontstack);

String glyphCacheFilePath(String root, String fontstack, String range) {
  if (!isAllowedFontstack(fontstack)) {
    throw ArgumentError.value(fontstack, 'fontstack', 'not a whitelisted fontstack');
  }
  if (!_rangePattern.hasMatch(range)) {
    throw ArgumentError.value(range, 'range', r'range must match ^\d+-\d+$');
  }
  return p.join(root, 'glyphs', fontstack, '$range.pbf');
}
```
Adapt directly for region ids (Pitfall 4): replace `_rangePattern` with the backend's own `^[a-z0-9][a-z0-9_-]*$` regex (matches `IsValidRegionID`/SvelteKit's `RegionIdSchema` above), reject with `ArgumentError` before building `<appDocs>/regions/<id>/vector.pmtiles` or `.../dem.pmtiles`, and always join via `package:path` (`p.join`), never string concatenation.

---

### `app/lib/entities/region_entity.dart` (model, CRUD + transform) — `localTilePathsForBounds`

**Analog:** the entity's own existing bbox fields (`minLon`/`minLat`/`maxLon`/`maxLat`, lines 30-33) plus the multi-cell reasoning already proven in `offline_style_rewriter.dart`.

**Core bbox-overlap pattern** (from RESEARCH.md Pattern 4, direct inspection of `RegionEntity` + `maplibre_platform_interface` confirming no `intersects()` helper exists on `LngLatBounds`):
```dart
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
This function belongs on `TileRepositoryManager` (or a small standalone query helper it owns), reading `RegionEntity.vectorPackage.target?.localFilePath` / `.demPackage.target?.localFilePath` (fields already defined in `region_entity.dart` lines 86-87, `downloaded_tile_package_entity.dart` line 32) — no schema change needed.

---

### `app/lib/models/package_status.dart` / `region_status.dart` — new enum values (`paused`, error/corrupt)

**Analog:** `app/lib/entities/downloaded_tile_package_entity.dart` lines 16-30 (the existing `PackageStatus` `.code` shadow pattern)

```dart
@Transient()
PackageStatus status = PackageStatus.notDownloaded;

int get dbStatus => status.code;

set dbStatus(int value) {
  status = PackageStatus.values.firstWhere(
    (s) => s.code == value,
    orElse: () => PackageStatus.notDownloaded,
  );
}
```
**Hard rule (Pitfall 5 / Common Pitfalls #3 in RESEARCH.md):** any new status (`paused`, `error`/`corrupted`) must be **appended** with a brand-new `.code` int — never renumber or insert into the existing `{notDownloaded, downloading, downloaded}` sequence. The `orElse` fallback pattern above must be preserved so an unrecognized future code degrades safely rather than throwing.

---

### `web/src/routes/api/v1/regions/[id]/download/+server.ts` and `download-dem/+server.ts` (route, streaming request-response)

**Analog:** each file is the other's near-identical sibling; the same fix must be applied to both, byte-for-byte identical except the URL path segment and `Content-Disposition` filename.

**Current gap (both files, identical shape)** — lines 46-79 of `download/+server.ts` (`download-dem/+server.ts` is structurally identical at the same line numbers):
```typescript
export async function GET(event: RequestEvent) {
  try {
    const { id } = RegionIdSchema.parse(event.params);

    const response = await event.fetch(
      `${event.locals.pb.baseURL}/regions/${id}/download`,
      {
        headers: {
          Authorization: event.locals.pb.authStore.token
            ? `Bearer ${event.locals.pb.authStore.token}`
            : '',
        },
      }
    );

    if (!response.ok) {
      return new Response(response.body, { status: response.status });
    }

    return new Response(response.body, {
      status: 200, // BUG: hardcoded, drops upstream 206 Partial Content
      headers: {
        'Content-Type': 'application/octet-stream',
        'Content-Disposition': `attachment; filename="${id}-vector.pmtiles"`,
        ...(response.headers.get('Content-Length')
          ? { 'Content-Length': response.headers.get('Content-Length')! }
          : {}),
      },
    });
  } catch (e) {
    return handleError(e);
  }
}
```

**Required fix pattern** (per RESEARCH.md Pitfall 1 — apply identically to both `download/+server.ts` and `download-dem/+server.ts`):
```typescript
export async function GET(event: RequestEvent) {
  try {
    const { id } = RegionIdSchema.parse(event.params);
    const range = event.request.headers.get('Range');

    const response = await event.fetch(
      `${event.locals.pb.baseURL}/regions/${id}/download`,
      {
        headers: {
          Authorization: event.locals.pb.authStore.token
            ? `Bearer ${event.locals.pb.authStore.token}`
            : '',
          ...(range ? { Range: range } : {}),
        },
      }
    );

    if (!response.ok) {
      return new Response(response.body, { status: response.status });
    }

    return new Response(response.body, {
      status: response.status, // forward 200 or 206 verbatim
      headers: {
        'Content-Type': 'application/octet-stream',
        'Content-Disposition': `attachment; filename="${id}-vector.pmtiles"`,
        ...(response.headers.get('Content-Length')
          ? { 'Content-Length': response.headers.get('Content-Length')! }
          : {}),
        ...(response.headers.get('Content-Range')
          ? { 'Content-Range': response.headers.get('Content-Range')! }
          : {}),
        ...(response.headers.get('Accept-Ranges')
          ? { 'Accept-Ranges': response.headers.get('Accept-Ranges')! }
          : {}),
      },
    });
  } catch (e) {
    return handleError(e);
  }
}
```
Error handling (`handleError(e)` from `$lib/util/api_util`, line 1 import) and the existing `RegionIdSchema` Zod validation (lines 5-7) are unchanged — do not touch either; only the `Range` forward-in / status+headers forward-out lines need editing, in both files identically.

## Shared Patterns

### Construction-only service injection (Store + Dio)
**Source:** `app/lib/services/trail_download_service.dart` line 39 (`TrailDownloadService(this._store, this._api)`), `app/lib/provider/region/region_provider.dart` lines 86-90, 152-155 (`RegionRepository(this._api, this._store)` + `@Riverpod(keepAlive: true) RegionRepository regionRepository(Ref ref) => RegionRepository(ref.watch(apiProvider), ref.watch(objectBoxProvider));`)
**Apply to:** `TileRepositoryManager` + its `tileRepositoryManagerProvider` — identical constructor-injection + `keepAlive` provider seam, no fetch/side-effect performed at construction time.

### Cancellation via `CancelToken` + `DioException`/`CancelToken.isCancel` rethrow discipline
**Source:** `app/lib/services/trail_download_service.dart` (throughout, e.g. lines 200-204, 339-345, 478-480)
**Apply to:** All download methods on `TileRepositoryManager` — cancellation must always propagate (rethrow), never be silently swallowed as a generic failure, so callers (lifecycle pause vs. explicit user cancel vs. real network failure) can distinguish outcomes.

### Batched ObjectBox writes inside `runInTransaction(TxMode.write, ...)`
**Source:** `app/lib/services/trail_download_service.dart` lines 206-208; `app/lib/provider/region/region_provider.dart` lines 104-138
**Apply to:** Every `DownloadedTilePackageEntity`/`RegionEntity` status/progress write in `TileRepositoryManager` — never write inside a tight per-byte progress callback.

### Path-safety allow-list-before-path-construction
**Source:** `app/lib/util/map_cache_path.dart` lines 22-52; `app/lib/util/offline_style_rewriter.dart` lines 266-281 (`_assertSafePath`)
**Apply to:** `region_file_path.dart` (new) — validate `RegionEntity.id` against `^[a-z0-9][a-z0-9_-]*$` before any directory/file path is built from it (Pitfall 4).

### Cascade delete: entity row + on-disk file as one unit
**Source:** No exact analog exists in-repo (`trail_download_service.dart`'s cleanup, lines 351-356, only deletes files during in-progress-download failure, not full-entity delete) — treat as a new pattern: `TileRepositoryManager.deleteRegion(regionId)` must delete both the `DownloadedTilePackageEntity` row(s) and the on-disk file(s) inside one operation, since ObjectBox does not cascade (RESEARCH.md Pitfall 5 / "No cascade delete").

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| Disk-space pre-check helper (new, location TBD by planner — likely `app/lib/util/disk_space_util.dart` or inline in `tile_repository_manager.dart`) | utility | file-I/O guard | No in-repo precedent — first disk-space check in the codebase; depends on an `[ASSUMED]` third-party package (`disk_space_2`) gated behind `checkpoint:human-verify` per RESEARCH.md. Use the RESEARCH.md Code Examples "Disk-space pre-check" snippet as the starting shape once the package is verified. |
| `AppLifecycleListener`-in-a-plain-class (non-widget lifecycle observation) | service (lifecycle hook) | event-driven | Only in-repo lifecycle-observation precedent (`navigation_screen.dart`) is widget-based (`WidgetsBindingObserver` mixin on a `State`); `TileRepositoryManager` needs the SDK's standalone `AppLifecycleListener` API instead — no in-repo class currently uses it. Use RESEARCH.md Pattern 3 (official Flutter API docs) as the template; borrow only the *dispose discipline* from `navigation_screen.dart`'s `dispose()`. |

## Metadata

**Analog search scope:** `app/lib/services/`, `app/lib/provider/region/`, `app/lib/provider/trail/`, `app/lib/util/`, `app/lib/entities/`, `app/lib/routes/navigation_screen.dart`, `web/src/routes/api/v1/regions/`
**Files scanned:** 9 read in full (trail_download_service.dart, map_cache_path.dart, trail_download_state_provider.dart, offline_style_rewriter.dart, region_provider.dart, region_entity.dart, downloaded_tile_package_entity.dart, download/+server.ts, download-dem/+server.ts) + 1 targeted grep+read (navigation_screen.dart lifecycle section)
**Pattern extraction date:** 2026-07-22
