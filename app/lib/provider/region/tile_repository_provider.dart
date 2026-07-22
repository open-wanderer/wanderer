import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/region_download_state.dart';
import 'package:wanderer/models/region_status.dart';
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

/// Per-region download state, keyed by region id -- Phase 24's Settings UI
/// subscribes to this. `keepAlive` so in-progress state survives whichever
/// widget happens to rebuild or unmount mid-download (mirrors
/// `DownloadingTrailIds`).
@Riverpod(keepAlive: true)
class TileRepositoryStatus extends _$TileRepositoryStatus {
  @override
  Map<String, RegionDownloadState> build() => {};

  /// Starts (or resumes an interrupted) vector download for [regionId].
  /// Idempotent re-entry guard: a second call while already downloading is a
  /// no-op, matching `DownloadingTrailIds.download`'s guard.
  Future<void> downloadVector(String regionId) async {
    if (state[regionId]?.status == RegionStatus.downloading) return;

    state = {
      ...state,
      regionId: (state[regionId] ?? const RegionDownloadState()).copyWith(
        status: RegionStatus.downloading,
        vectorProgress: 0,
      ),
    };

    try {
      await ref
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
    } finally {
      // Authoritative status lives on RegionEntity/DownloadedTilePackageEntity;
      // clear this ephemeral in-flight marker regardless of outcome.
      state = {...state}..remove(regionId);
    }
  }

  /// Starts (or resumes an interrupted) DEM download for [regionId]. Fully
  /// independent from [downloadVector] -- a DEM failure never touches the
  /// vector entry.
  Future<void> downloadDem(String regionId) async {
    if (state[regionId]?.status == RegionStatus.downloading) return;

    state = {
      ...state,
      regionId: (state[regionId] ?? const RegionDownloadState()).copyWith(
        status: RegionStatus.downloading,
        demProgress: 0,
      ),
    };

    try {
      await ref
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
    } finally {
      state = {...state}..remove(regionId);
    }
  }

  /// Cancels every active vector/DEM download for [regionId] with a
  /// deliberate pause (preserves `.part` files for a later [resume]).
  Future<void> pause(String regionId) async {
    await ref.read(tileRepositoryManagerProvider).pauseRegion(regionId);
    state = {...state}..remove(regionId);
  }

  /// Re-invokes whichever paused package(s) of [regionId] remain.
  Future<void> resume(String regionId) async {
    if (state[regionId]?.status == RegionStatus.downloading) return;

    state = {
      ...state,
      regionId: (state[regionId] ?? const RegionDownloadState()).copyWith(
        status: RegionStatus.downloading,
      ),
    };

    try {
      await ref
          .read(tileRepositoryManagerProvider)
          .resumeRegion(
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
    } finally {
      state = {...state}..remove(regionId);
    }
  }

  /// Deletes [regionId]'s downloaded packages + on-disk files and clears its
  /// tracked state.
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
      await ref
          .read(tileRepositoryManagerProvider)
          .deleteDemPackage(regionId);
    } finally {
      state = {...state}..remove(regionId);
    }
  }
}
