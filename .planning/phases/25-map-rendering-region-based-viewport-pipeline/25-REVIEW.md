---
phase: 25-map-rendering-region-based-viewport-pipeline
reviewed: 2026-07-23T00:00:00Z
depth: standard
files_reviewed: 6
files_reviewed_list:
  - app/lib/components/base/trail_map.dart
  - app/lib/routes/navigation_screen.dart
  - app/lib/services/tile_repository_manager.dart
  - app/test/services/region_render_spike_harness.dart
  - app/test/services/tile_repository_manager_harness.dart
  - app/test/services/tile_repository_manager_test.dart
findings:
  critical: 3
  warning: 6
  info: 1
  total: 10
status: issues_found
---

# Phase 25: Code Review Report

**Reviewed:** 2026-07-23T00:00:00Z
**Depth:** standard
**Files Reviewed:** 6
**Status:** issues_found

## Summary

Reviewed the region-based viewport rendering pipeline: `TrailMap`'s add-only region reconcile, `NavigationScreen`'s add/remove viewport reconcile, `TileRepositoryManager`'s download lifecycle, and the accompanying unit test / on-device spike harnesses. `dart analyze` is clean and `tile_repository_manager_test.dart`'s pure-function coverage of `bboxOverlaps`/`splitRegionTilePaths` is solid, but tracing the actual runtime paths surfaces three real correctness bugs (a permanently unrecoverable download status on non-`DioException` failures, a missing de-dup guard that lets two concurrent downloads corrupt the same `.part` file, and a viewport-bounds fallback that can never resolve for an offline recording session) plus several concurrency/robustness gaps around the fire-and-forget incremental style reconcile and the disk-space safety check silently no-opping when a region's declared size is unknown.

## Critical Issues

### CR-01: Download status gets stuck at `downloading` forever on any non-`DioException` failure

**File:** `app/lib/services/tile_repository_manager.dart:153-196` (vector), `app/lib/services/tile_repository_manager.dart:241-276` (DEM)
**Issue:** `startVectorDownload`/`startDemDownload` wrap the download + validation + promote sequence in `try { ... } on DioException catch (e) { ... }`. `_isValidPmTiles` only catches `CorruptArchiveException`/`UnsupportedError` internally (line 454-464) and re-throws everything else; `partFile.renameSync(finalPath)` (line 167, 255) and `File(finalPath).lengthSync()` (line 172, 260) can also throw plain `FileSystemException`. None of those are `DioException`, so they are not caught by the surrounding `on DioException` clause — the exception propagates out of the `async` method as an unhandled `Future` error. The `finally` block only removes the cancel token; it never runs `_updatePackageStatus(..., status: PackageStatus.error)`, so the package's `DownloadedTilePackageEntity.status` is left at `downloading` permanently (surviving app restarts, since it's persisted), and the stray `.part` file is never deleted since the `if (partFile.existsSync()) partFile.deleteSync();` cleanup only runs inside the `DioException` branch.
**Fix:**
```dart
} on DioException catch (e) {
  if (partFile.existsSync()) partFile.deleteSync();
  _updatePackageStatus(
    package,
    status: CancelToken.isCancel(e)
        ? PackageStatus.notDownloaded
        : PackageStatus.error,
  );
} catch (e) {
  // Any non-Dio failure (corrupt/partial file I/O, unexpected
  // PmTilesArchive error, etc.) must still leave the package recoverable.
  if (partFile.existsSync()) partFile.deleteSync();
  _updatePackageStatus(package, status: PackageStatus.error);
  rethrow; // or swallow + log, per this file's own debugPrint convention
} finally {
  if (identical(_activeCancelTokens[tokenKey], token)) {
    _activeCancelTokens.remove(tokenKey);
  }
}
```
Apply the same fix to `startDemDownload`.

### CR-02: No de-duplication guard lets two concurrent downloads for the same region write the same `.part` file

**File:** `app/lib/services/tile_repository_manager.dart:115-196` (`startVectorDownload`), `app/lib/services/tile_repository_manager.dart:203-276` (`startDemDownload`)
**Issue:** Neither method checks whether a download is already in flight for `'<id>:vector'` / `'<id>:dem'` before starting a new one. `_activeCancelTokens[tokenKey] = token;` (line 149, 237) unconditionally overwrites any existing entry. If `startVectorDownload` is called twice concurrently for the same region (e.g. a double-tap on the download button, or two independent call sites), both Dio transfers write to the identical `partPath` simultaneously — a real data-corruption risk for the resulting archive — and the *first* download's original `CancelToken` is silently dropped from `_activeCancelTokens`, so it can no longer be reached by `cancelVectorDownload`/`cancelDemDownload`, nor by `dispose()`'s cleanup loop (which only iterates whatever tokens currently remain in the map). The first transfer keeps running, uncancellable, writing into a file the second transfer is also writing into.
**Fix:**
```dart
final tokenKey = '$id:vector';
if (_activeCancelTokens.containsKey(tokenKey)) {
  // Already downloading — ignore the duplicate request rather than
  // starting a second writer against the same .part file.
  return;
}
final token = CancelToken();
_activeCancelTokens[tokenKey] = token;
```
Apply symmetrically to `startDemDownload`. If overriding a previous package's request is the desired UX (e.g. "restart download"), cancel-and-await the previous token first instead of silently replacing it.

### CR-03: Offline recording sessions can never resolve initial viewport bounds — permanent loading-spinner deadlock

**File:** `app/lib/routes/navigation_screen.dart:1369-1377`
**Issue:**
```dart
final composed = _composeStyle(
  baseJson,
  cache,
  _controller?.getVisibleRegion() ?? trailAsync.value?.bounds,
);
```
`_controller` is only assigned once `MapLibreMap` has actually mounted, which itself requires `styleJson != null` — i.e. requires `_composeStyle` to already have returned non-null once. Before the controller exists, the only fallback is `trailAsync.value?.bounds`, where `trailAsync = ref.watch(trailProvider(widget.id))`. For a recording session (`widget.isRecording == true`), `widget.id` is `''` (see the `/record` route in `router_provider.dart`, which pushes `NavigationScreen(id: '', ...)`), and `trailProvider('')` resolves to `AsyncError` — confirmed by this same file's own comment in `_buildElevationPage` ("`trailProvider('')` resolves to `AsyncError`"). `AsyncError.value` is always `null`, and it never becomes non-null on a later rebuild, since the trail-less recording never has a backing `Trail`. So when `widget.isOffline && widget.isRecording` are both true, `viewportBounds` evaluates to `null` on every build, forever: `_composeStyle` returns `null` (line 896: `if (viewportBounds == null) return null;`), `styleJson` stays `null`, the `Scaffold.body` stays on the `CircularProgressIndicator()` branch (line 1392-1395), `MapLibreMap` never mounts, `_controller` never gets set, and the deadlock is permanent — the recording screen never renders a map. `widget.initialCenter` (documented specifically as "Initial map camera center for a fresh recording session", line 69-74) is never consulted here.
**Fix:** Fall back to a small bounds window derived from `widget.initialCenter` when both other sources are unavailable:
```dart
ml.LngLatBounds? _fallbackViewportBounds() {
  final center = widget.initialCenter;
  if (center == null) return null;
  const pad = 0.01; // ~1km, enough to resolve the first region query
  return ml.LngLatBounds(
    longitudeWest: center.lon - pad,
    longitudeEast: center.lon + pad,
    latitudeSouth: center.lat - pad,
    latitudeNorth: center.lat + pad,
  );
}
...
final composed = _composeStyle(
  baseJson,
  cache,
  _controller?.getVisibleRegion() ??
      trailAsync.value?.bounds ??
      _fallbackViewportBounds(),
);
```
Note this is currently unreachable in production because `/record`'s route builder never passes `isOffline: true` (see WR-05) — but it is a live bug in the file under review and will bite immediately the moment that plumbing gap is closed, with no test coverage to catch it.

## Warnings

### WR-01: Disk-space safety check silently bypassed when a region's declared size is unknown

**File:** `app/lib/services/tile_repository_manager.dart:132-140` (vector), `app/lib/services/tile_repository_manager.dart:220-228` (DEM)
**Issue:** `hasEnoughSpace(freeBytes: free, declaredSizeBytes: region.vectorSize ?? 0)` coalesces a `null` declared size to `0`. `hasEnoughSpace` (`app/lib/util/disk_space_util.dart:114-121`) computes `freeBytes > declaredSizeBytes * safetyMultiplier`, so with `declaredSizeBytes == 0` the check passes for any non-zero free space — the TILE-03 "fail-closed" disk-space refusal this helper is documented to provide is silently defeated whenever the catalog hasn't supplied a size. A download can proceed with zero space-safety margin and fail mid-transfer on a near-full device.
**Fix:** Treat an unknown declared size as "unknown, therefore unsafe" rather than "zero, therefore always safe":
```dart
final declaredSize = region.vectorSize;
if (declaredSize == null ||
    !hasEnoughSpace(freeBytes: free, declaredSizeBytes: declaredSize)) {
  final package = _getOrCreatePackage(region, region.vectorPackage);
  _updatePackageStatus(package, status: PackageStatus.error);
  return;
}
```

### WR-02: `deleteRegion`/`deleteDemPackage` don't wait for an in-flight download's own cancellation cleanup, risking resurrected rows

**File:** `app/lib/services/tile_repository_manager.dart:314-356` (`deleteRegion`), `app/lib/services/tile_repository_manager.dart:364-393` (`deleteDemPackage`), cross-referenced against `app/lib/services/tile_repository_manager.dart:180-196`
**Issue:** `deleteRegion` calls `entry.value.cancel('deleted')` on any active token (line 319) and then, without awaiting the cancellation's completion, immediately removes the `DownloadedTilePackageEntity` rows and clears `region.vectorPackage.target`/`region.demPackage.target` in a transaction (lines 329-338), then deletes the on-disk files. `CancelToken.cancel()` only *requests* cancellation — the in-flight `startVectorDownload`/`startDemDownload` call keeps running until its own `await` resolves with the resulting `DioException`, at which point its `on DioException catch` block still calls `_updatePackageStatus(package, ...)` (line 182-187) against the very `package`/`region` objects `deleteRegion` just removed/cleared, and `_updatePackageStatus` unconditionally `put()`s the package back into the box. Depending on ObjectBox's put-with-explicit-id semantics, this can re-insert a `DownloadedTilePackageEntity` row for a region that was just deleted, orphaned from `region.vectorPackage`/`demPackage` (already nulled), leaking a DB row that nothing references.
**Fix:** Either await the cancellation before mutating state (e.g. give `cancel()` a completer the in-flight method resolves in its `finally`), or have `_updatePackageStatus`/the `DioException` handler check that the package/region row still exists before writing back to it.

### WR-03: Incremental region reconcile/swap calls are fire-and-forget with no reentrancy guard

**File:** `app/lib/components/base/trail_map.dart:100-106, 317-393`, `app/lib/routes/navigation_screen.dart:1309-1318, 1076-1218, 922-935`
**Issue:** `_addRegionComposition()` (TrailMap) and `_reconcileRegionComposition()`/`_swapStyle()` (NavigationScreen) are all `async` but invoked without `await` from `ref.listen(regionListNotifierProvider, ...)` callbacks and, in `NavigationScreen`, also from the `MapEventCameraIdle` handler inside `onEvent` (`navigation_screen.dart:1455`). Nothing prevents two calls from overlapping: a region finishing download while a camera-idle event is also firing, or two camera-idle events firing close together, both mutate the same `_addedSourceIds`/`_addedLayerIds` sets across `await` boundaries (`await style.addSource(...)`, `await style.removeLayer(...)`). A `setStyle()` triggered by a concurrent theme toggle (`_swapStyle`) makes this worse — it invalidates the native style out from under an in-flight incremental reconcile, whose subsequent `addSource`/`removeSource` calls will target sources/layers that no longer exist post-`setStyle`, and `_addedSourceIds`/`_addedLayerIds` won't be correctly reseeded until `setStyle`'s own `onStyleLoaded` eventually fires. Failures are all caught and logged individually, so nothing crashes, but the tracked-vs-mounted style state can end up inconsistent (a region silently missing from the map, or a stale layer left rendered) until the next trigger happens to repair it.
**Fix:** Guard each of these methods with a simple in-flight flag (and optionally coalesce a trailing call rather than dropping it):
```dart
bool _reconciling = false;
Future<void> _reconcileRegionComposition() async {
  if (_reconciling) return;
  _reconciling = true;
  try {
    // ...existing body...
  } finally {
    _reconciling = false;
  }
}
```

### WR-04: `_sourceFromJson`'s `raster-dem` branch uses an unguarded cast inconsistent with the rest of the function

**File:** `app/lib/components/base/trail_map.dart:220-234`, `app/lib/routes/navigation_screen.dart:973-987`
**Issue:** Every other numeric field read in `_sourceFromJson`/`_layerFromJson` uses a null-safe cast with a default (`(source['maxzoom'] as num?)?.toDouble() ?? 14`), but the `raster-dem` branch does `maxZoom: (source['maxzoom'] as num).toDouble()` — a non-nullable cast that throws `TypeError` if `maxzoom` is missing or the wrong type. It happens to be safe today only because every call site wraps it in a broad `try { ... } catch (e) { debugPrint(...); }`, but that safety is incidental to the caller, not to this helper itself — a future call site (or a refactor that moves the cast earlier) reintroduces an uncaught crash.
**Fix:**
```dart
maxZoom: (source['maxzoom'] as num?)?.toDouble() ?? 14,
```

### WR-05: A resumed recording session silently loses its persisted `isOffline` flag

**File:** `app/lib/routes/navigation_screen.dart:653-659` (writer), `app/lib/provider/router_provider.dart:284-326` (reader gap, evidence)
**Issue:** `_persistNow()` faithfully writes `isOffline: widget.isOffline` onto the `ActiveNavigationEntity` for a recording session (line 658), and `ActiveNavigationEntity.isOffline` exists specifically to be restored on resume (it *is* read back for turn-by-turn resume — `main.dart:232` passes `row.isOffline ?? false` into the `/trail/:id/navigate` extra tuple). But the `/record` route's resume path (`main.dart:_maybeResumeRecording` → `router_provider.dart`'s `/record` builder) never reads `row.isOffline` back out — the builder only extracts `resume`, `center`, `seedPosition`, `recordingCosting` from `extra` and constructs `NavigationScreen` with no `isOffline:` argument, so it silently defaults to `false` regardless of what was persisted. This also means `widget.isOffline` is currently *unreachable as `true`* for every recording code path (fresh or resumed), so all of this file's offline-composition logic for recording (this file's `_composeStyle`/`_reconcileRegionComposition`/the `Icons.cloud_off` banner icon) is presently dead code for `isRecording == true`, masking CR-03 above.
**Fix:** Thread `row.isOffline` through the `/record` route builder the same way the `/navigate` route already does for turn-by-turn resume, e.g. `isOffline: resume?.isOffline ?? false` on the `NavigationScreen(...)` construction in `router_provider.dart`.

### WR-06: ~120 lines of region-composition helpers duplicated verbatim between `TrailMap` and `NavigationScreen`

**File:** `app/lib/components/base/trail_map.dart:184-307`, `app/lib/routes/navigation_screen.dart:937-1062`
**Issue:** `_seedRegionTracking`, `_sourceFromJson`, and `_layerFromJson` — including their doc comments — are byte-for-byte identical between the two files (only the `debugPrint` log-message prefix differs, `'TrailMap: ...'` vs `'NavigationScreen: ...'`). Any future fix to how a style-spec field is parsed (e.g. supporting a new layer type, or fixing the WR-04 cast) must be applied in both places; nothing enforces that, and the two implementations can silently drift apart.
**Fix:** Extract the three methods (plus their shared doc comments) into a standalone helper — e.g. a top-level function/class in `offline_style_rewriter.dart` or a new `region_style_reconciler.dart` — parameterized by a `String` log-tag, and have both widgets delegate to it.

## Info

### IN-01: Buffered `onStyleLoaded` style controller is captured then discarded without being used

**File:** `app/test/services/region_render_spike_harness.dart:334-345`
**Issue:**
```dart
onMapCreated: (controller) {
  _controller = controller;
  final pending = _pendingStyle;
  if (pending != null) {
    _pendingStyle = null;
  }
},
onStyleLoaded: (style) {
  if (_controller == null) {
    _pendingStyle = style;
  }
},
```
This mirrors the buffer-then-flush pattern used in `trail_map.dart`/`navigation_screen.dart` (where the buffered `pending` `StyleController` is passed into `_onStyleLoaded(pending)`), but here `pending` is read and then dropped — nothing is ever done with the buffered style. Since this is a throwaway on-device spike harness (per its own header comment) this is harmless, but it's dead code that looks like an incomplete copy of the production pattern.
**Fix:** Either remove the dead buffering entirely (nothing in this harness currently needs an `onStyleLoaded` callback) or, if intentional, drop the unused `pending` local.

---

_Reviewed: 2026-07-23T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
