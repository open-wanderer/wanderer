import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/region_download_state.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/services/tile_repository_manager.dart';

part 'tile_repository_provider.g.dart';

/// Construction-only provider seam (mirrors `regionRepositoryProvider`) --
/// builds a [TileRepositoryManager] from the existing
/// [objectBoxProvider]/[apiProvider] without performing any I/O at build
/// time.
@Riverpod(keepAlive: true)
TileRepositoryManager tileRepositoryManager(Ref ref) {
  return TileRepositoryManager(
    ref.watch(objectBoxProvider),
    ref.watch(apiProvider),
  );
}

/// Per-region download state, keyed by region id -- the Settings/Regions
/// screen subscribes to this. `keepAlive` so in-progress state survives
/// whichever widget happens to rebuild or unmount mid-download (mirrors
/// `DownloadingTrailIds`). Vector and DEM downloads are fully independent:
/// each has its own start/cancel method and its own progress field on
/// [RegionDownloadState].
@Riverpod(keepAlive: true)
class TileRepositoryStatus extends _$TileRepositoryStatus {
  @override
  Map<String, RegionDownloadState> build() => {};

  /// The in-flight [startVectorDownload]/[startDemDownload] `Future`s,
  /// keyed by region id -- lets [cancelVector]/[cancelDem] await the
  /// cancelled download's own catch/finally handling (ObjectBox status
  /// write + ephemeral progress clear) before returning, instead of racing
  /// ahead of it. See [cancelVector]'s doc comment for why that ordering
  /// matters.
  final Map<String, Future<void>> _activeVectorDownloads = {};
  final Map<String, Future<void>> _activeDemDownloads = {};

  /// Starts a fresh vector download for [regionId]. Idempotent re-entry
  /// guard: a second call while already downloading is a no-op, matching
  /// `DownloadingTrailIds.download`'s guard.
  Future<void> downloadVector(String regionId) async {
    if (state[regionId]?.vectorProgress != null) return;

    state = {
      ...state,
      regionId: (state[regionId] ?? const RegionDownloadState()).copyWith(
        vectorProgress: 0,
      ),
    };

    final future = ref
        .read(tileRepositoryManagerProvider)
        .startVectorDownload(
          regionId,
          onProgress: (received, total) {
            if (total <= 0) return;
            state = {
              ...state,
              regionId: (state[regionId] ?? const RegionDownloadState())
                  .copyWith(vectorProgress: received / total),
            };
          },
        );
    _activeVectorDownloads[regionId] = future;

    try {
      await future;
    } finally {
      _activeVectorDownloads.remove(regionId);
      _clearVectorProgress(regionId);
    }
  }

  /// Starts a fresh DEM download for [regionId]. Fully independent from
  /// [downloadVector] -- a DEM failure never touches the vector entry, and
  /// vice versa.
  Future<void> downloadDem(String regionId) async {
    if (state[regionId]?.demProgress != null) return;

    state = {
      ...state,
      regionId: (state[regionId] ?? const RegionDownloadState()).copyWith(
        demProgress: 0,
      ),
    };

    final future = ref
        .read(tileRepositoryManagerProvider)
        .startDemDownload(
          regionId,
          onProgress: (received, total) {
            if (total <= 0) return;
            state = {
              ...state,
              regionId: (state[regionId] ?? const RegionDownloadState())
                  .copyWith(demProgress: received / total),
            };
          },
        );
    _activeDemDownloads[regionId] = future;

    try {
      await future;
    } finally {
      _activeDemDownloads.remove(regionId);
      _clearDemProgress(regionId);
    }
  }

  /// Cancels [regionId]'s in-flight vector download, if any, and awaits its
  /// resulting `DioException` unwinding all the way through
  /// [downloadVector]'s catch/finally -- ObjectBox status write to
  /// `notDownloaded` AND the ephemeral progress-clear -- before returning.
  /// Without this, the screen's `_save` wrapper would invalidate
  /// `regionListNotifierProvider` (forcing a fresh read of the region's
  /// `ToOne`) before the manager finished writing `notDownloaded`, so the
  /// newly-fetched `RegionEntity` would still read the pre-cancel
  /// `downloading` status and the tile would appear stuck. No pause/resume:
  /// the `.part` file is deleted (`TileRepositoryManager`'s
  /// `deleteOnError: true`), so a later [downloadVector] call always starts
  /// from byte 0.
  Future<void> cancelVector(String regionId) async {
    ref.read(tileRepositoryManagerProvider).cancelVectorDownload(regionId);
    await _activeVectorDownloads[regionId];
  }

  /// Cancels [regionId]'s in-flight DEM download, if any. See
  /// [cancelVector].
  Future<void> cancelDem(String regionId) async {
    ref.read(tileRepositoryManagerProvider).cancelDemDownload(regionId);
    await _activeDemDownloads[regionId];
  }

  /// Deletes [regionId]'s downloaded packages (vector AND DEM) + on-disk
  /// files and clears its tracked state.
  Future<void> delete(String regionId) async {
    await ref.read(tileRepositoryManagerProvider).deleteRegion(regionId);
    state = {...state}..remove(regionId);
  }

  /// Removes [regionId]'s DEM package only (D-01) -- the vector package and
  /// its lifecycle are untouched. Mirrors [delete]'s ephemeral-state
  /// clearing but performs no progress tracking (pure removal, not a status
  /// transition).
  Future<void> deleteDemPackage(String regionId) async {
    try {
      await ref.read(tileRepositoryManagerProvider).deleteDemPackage(regionId);
    } finally {
      _clearDemProgress(regionId);
    }
  }

  /// Clears only the vector progress field, preserving a concurrently
  /// in-flight DEM download's entry -- removing the whole [regionId] key
  /// outright would wipe `demProgress` out from under it.
  void _clearVectorProgress(String regionId) {
    final current = state[regionId];
    if (current == null) return;
    if (current.demProgress == null) {
      state = {...state}..remove(regionId);
    } else {
      state = {...state, regionId: current.copyWith(vectorProgress: null)};
    }
  }

  /// See [_clearVectorProgress] -- the DEM-side mirror.
  void _clearDemProgress(String regionId) {
    final current = state[regionId];
    if (current == null) return;
    if (current.vectorProgress == null) {
      state = {...state}..remove(regionId);
    } else {
      state = {...state, regionId: current.copyWith(demProgress: null)};
    }
  }
}
