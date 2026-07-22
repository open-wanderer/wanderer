---
status: issues_found
phase: 24-settings-offline-maps-regions-ui
depth: standard
files_reviewed: 13
findings:
  critical: 0
  warning: 4
  info: 1
  total: 5
---

## Summary

The core Phase 24 design decisions hold up: every mutating region action routes through `assertValidRegionId` before touching the filesystem (no raw path construction from catalog/user-influenced ids), `regionListNotifierProvider` is invalidated unconditionally in `_save`'s `finally` block after every screen-triggered mutation, and `region_disk_usage_util.dart` correctly stats real on-disk bytes (final file, else `.part`, else 0) rather than trusting the persisted `sizeBytesOnDisk` field. The asymmetric confirm-before-destroy UX (full delete confirms, DEM toggle-off does not) is intentional and correctly implemented, not a bug. Four warning-level correctness/UX bugs and one info-level polish gap were found, concentrated in the download-resume progress plumbing and error-recovery paths.

## Findings

### WR-1: Resumed DEM downloads report progress into the wrong field
**File:** `app/lib/provider/region/tile_repository_provider.dart:107-134` (`TileRepositoryStatus.resume`), root cause in `app/lib/services/tile_repository_manager.dart:275-291` (`TileRepositoryManager.resumeRegion`)

`resumeRegion` fans out to both `startVectorDownload` and `startDemDownload` for whichever package(s) are `paused`, passing the **same** `onProgress` callback to both:

```dart
if (region.vectorPackage.target?.status == PackageStatus.paused) {
  futures.add(startVectorDownload(id, onProgress: onProgress));
}
if (region.demPackage.target?.status == PackageStatus.paused) {
  futures.add(startDemDownload(id, onProgress: onProgress));
}
```

`TileRepositoryStatus.resume`'s callback unconditionally writes to `vectorProgress`:

```dart
onProgress: (received, total) {
  if (total <= 0) return;
  state = {
    ...state,
    regionId: (state[regionId] ?? const RegionDownloadState())
        .copyWith(vectorProgress: received / total),
  };
},
```

Contrast with `downloadDem`, which correctly targets `demProgress`. When only the DEM package is paused and gets resumed (a realistic case — vector already downloaded, DEM paused separately), its progress ticks are written into `vectorProgress` and `demProgress` stays `null` for the whole resume. Two user-visible effects: the DEM row's `demDownloading` spinner (gated on `downloadState?.demProgress != null`) never appears during a DEM-only resume, and if vector and DEM are resumed concurrently, `_combinedProgress`'s average is corrupted by both progress streams fighting over the same `vectorProgress` field.

**Fix:** give `resumeRegion` two distinct progress callbacks (or a `{required bool dem}` parameter threaded through) so the vector and DEM resume paths update their respective state fields, mirroring how `downloadVector`/`downloadDem` already do it.

### WR-2: "Pause" button's tooltip reads "Retry"
**File:** `app/lib/routes/settings_offline_regions_screen.dart:432-436`

```dart
case RegionStatus.downloading:
  return IconButton(
    icon: const FaIcon(FontAwesomeIcons.pause),
    tooltip: l10n.regions_retry,
    onPressed: () => _onPause(region),
  );
```

The pause icon (shown while a region is downloading) is labeled with `l10n.regions_retry` ("Retry") instead of the existing `l10n.pause` ("Pause") key (`app/lib/i18n/app_en.arb:173`). This is user-facing incorrect text — both for the visible long-press tooltip and for screen-reader accessibility, which announces the tooltip as the button's semantic label. Likely a copy-paste artifact from the adjacent `RegionStatus.error` case, which legitimately uses `l10n.regions_retry` for its retry `TextButton`.

**Fix:** change `tooltip: l10n.regions_retry` to `tooltip: l10n.pause`.

### WR-3: A single corrupt region id silently zeroes the entire disk-usage summary
**File:** `app/lib/util/region_disk_usage_util.dart:45-52`, consumed by `app/lib/routes/settings_offline_regions_screen.dart:180-204`

`totalRegionDiskUsageBytes` sums `regionDiskUsageBytes` in a loop; `regionDiskUsageBytes` throws `ArgumentError` via `assertValidRegionId` for any region whose `id` fails the allow-list regex (test-covered in `region_disk_usage_util_test.dart`'s "an invalid region id throws ArgumentError" case). If any single persisted `RegionEntity` row has an invalid id — e.g. corrupted local ObjectBox state, or a future non-catalog writer — the whole `Future<int>` rejects, and the screen's `FutureBuilder` does:

```dart
final totalBytes = snapshot.data ?? 0;
```

`snapshot.hasError` is never checked, so the summary silently renders "0 B used across N region(s)" instead of surfacing the failure or at least degrading gracefully (e.g. skipping the bad row and summing the rest). Since catalog ids are currently always server-validated before reaching this code path, likelihood is low today, but the failure mode is a silent, total (not partial) loss of a supposedly-live disk-usage figure with no diagnostic trail.

**Fix:** either make `totalRegionDiskUsageBytes` skip-and-continue on a bad id (consistent with how `parseRegionCatalog`/`upsertCatalog` already treat malformed elements as skippable, not fatal), or have the `FutureBuilder` fall back visibly (log / toast) instead of masking the error as a valid zero.

### WR-4: A non-network failure mid-download leaves the package permanently stuck in "downloading"
**File:** `app/lib/services/tile_repository_manager.dart:138-177` (`startVectorDownload`), `184-257` (`startDemDownload`); caller `app/lib/provider/region/tile_repository_provider.dart:34-64`, `69-97`

Both download methods only catch `on DioException`:

```dart
} on DioException catch (e) {
  if (CancelToken.isCancel(e)) {
    _updatePackageStatus(package, status: PackageStatus.paused);
    return;
  }
  if (!wasResuming && partFile.existsSync()) partFile.deleteSync();
  _updatePackageStatus(package, status: PackageStatus.error);
}
```

Any other exception thrown between `_updatePackageStatus(package, status: PackageStatus.downloading)` and this catch — e.g. a `FileSystemException`/`OSError` from `partFile.renameSync(finalPath)`, `File(finalPath).lengthSync()`, or `_isValidPmTiles`'s file open — is not caught here, and the caller (`TileRepositoryStatus.downloadVector`/`downloadDem`) only wraps the call in `try { } finally { }` with no `catch`, so the package's persisted `PackageStatus` is never flipped out of `downloading`. The exception does eventually surface as a toast via the screen's `_save` wrapper, but `RegionEntity.status` (D-12's computed getter) will keep reporting `RegionStatus.downloading` on every future app open — and the `downloading` case's trailing action is only a pause button (`_onPause`), which cancels a `CancelToken` that no longer exists (already removed in the manager's own `finally`), so it's a no-op. There is no UI path back to `notDownloaded`/`error` for a region stuck this way short of deleting and re-adding it.

**Fix:** widen the catch to `catch (e)` (not just `on DioException`) in both download methods so any failure — network or otherwise — reliably transitions the package to `PackageStatus.error`, which does have a working retry affordance.

### IN-1: Paused region's resume button has no tooltip
**File:** `app/lib/routes/settings_offline_regions_screen.dart:437-442`

```dart
case RegionStatus.paused:
  return IconButton(
    icon: const FaIcon(FontAwesomeIcons.play),
    color: accentColor,
    onPressed: () => _onResume(region),
  );
```

Every other `IconButton` in `_buildTrailingActions` (`notDownloaded`, `downloading`, `downloaded`'s delete, `error`'s delete) sets a `tooltip`; this one doesn't. Minor accessibility/consistency gap — screen readers fall back to no label for this action. Low priority; bundle with the WR-2 tooltip fix if touching this switch statement.
