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
  /// keyed by region id. Used only to identity-guard each download's
  /// terminal progress-clear so a rapid cancel-then-redownload can't let the
  /// OLD download's `finally` wipe the NEW download's freshly-set progress.
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
            // A cancel clears the entry synchronously; never let a late
            // in-flight progress callback resurrect it into `downloading`.
            if (state[regionId]?.vectorProgress == null) return;
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
      // Only THIS download owns the clear — a superseding download (started
      // after a cancel) has already replaced the map entry and its own
      // progress, which must survive.
      if (identical(_activeVectorDownloads[regionId], future)) {
        _activeVectorDownloads.remove(regionId);
        _clearVectorProgress(regionId);
      }
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
            if (state[regionId]?.demProgress == null) return;
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
      if (identical(_activeDemDownloads[regionId], future)) {
        _activeDemDownloads.remove(regionId);
        _clearDemProgress(regionId);
      }
    }
  }

  /// Cancels [regionId]'s in-flight vector download and resets the UI
  /// SYNCHRONOUSLY by clearing the ephemeral progress entry — deliberately
  /// NOT awaiting the download future.
  ///
  /// `downloading` is derived purely from this ephemeral progress
  /// (`resolveVectorTileStatus`), so clearing it here flips the tile back to
  /// a Download button immediately, no matter how long Dio takes to unwind
  /// the cancelled request or when the manager's `notDownloaded` write lands.
  /// An earlier version awaited the download future so the manager's write
  /// would precede `_save`'s `regionListNotifierProvider` invalidate — but a
  /// stalled/hung transfer meant that await never returned, so Cancel did
  /// nothing and the tile stayed frozen. No pause/resume: the manager's
  /// `deleteOnError: true` deletes the `.part` file, so a later
  /// [downloadVector] always restarts from byte 0.
  void cancelVector(String regionId) {
    ref.read(tileRepositoryManagerProvider).cancelVectorDownload(regionId);
    _clearVectorProgress(regionId);
  }

  /// Cancels [regionId]'s in-flight DEM download. See [cancelVector].
  void cancelDem(String regionId) {
    ref.read(tileRepositoryManagerProvider).cancelDemDownload(regionId);
    _clearDemProgress(regionId);
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
