import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/models/region_download_state.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/download_notification_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/region/region_provider.dart';
import 'package:wanderer/services/tile_repository_manager.dart';
import 'package:wanderer/util/region/notification_id.dart';

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

/// Per-region download state, keyed by region PATH (never the server record
/// id, which the backend re-mints) -- the Settings/Regions screen subscribes
/// to this. `keepAlive` so in-progress state survives
/// whichever widget happens to rebuild or unmount mid-download (mirrors
/// `DownloadingTrailIds`). Vector and DEM downloads are fully independent:
/// each has its own start/cancel method and its own progress field on
/// [RegionDownloadState].
@Riverpod(keepAlive: true)
class TileRepositoryStatus extends _$TileRepositoryStatus {
  @override
  Map<String, RegionDownloadState> build() => {};

  /// The in-flight [startVectorDownload]/[startDemDownload] `Future`s,
  /// keyed by region path. Used only to identity-guard each download's
  /// terminal progress-clear so a rapid cancel-then-redownload can't let the
  /// OLD download's `finally` wipe the NEW download's freshly-set progress.
  final Map<String, Future<void>> _activeVectorDownloads = {};
  final Map<String, Future<void>> _activeDemDownloads = {};

  /// Last whole percent pushed to each notification id — the throttle state
  /// for [_showRegionProgress]. Entries are removed at each download's
  /// terminal so a later re-download of the same package starts clean and its
  /// opening 0% is never swallowed as a duplicate.
  final Map<int, int> _lastNotifiedPercent = {};

  /// Starts a fresh vector download for [regionPath]. Idempotent re-entry
  /// guard: a second call while already downloading is a no-op, matching
  /// `DownloadingTrailIds.download`'s guard.
  ///
  /// [showNotification] drives this package's own ongoing progress
  /// notification. `DownloadingTrailIds.download` passes `false`: that path
  /// selects region packages alongside a trail and already renders ONE
  /// aggregate id-42 notification spanning the trail plus every selected
  /// package, so per-package notifications there would double-report the same
  /// transfer. Every other caller (the Settings/Regions screen) wants one.
  Future<void> downloadVector(
    String regionPath, {
    bool showNotification = true,
  }) async {
    if (state[regionPath]?.vectorProgress != null) return;

    state = {
      ...state,
      regionPath: (state[regionPath] ?? const RegionDownloadState()).copyWith(
        vectorProgress: 0,
      ),
    };

    // Resolved ONCE, before the transfer starts: the notification title must
    // stay stable for the whole download, and a per-tick lookup would rebuild
    // the region snapshot on every progress chunk.
    final notification = showNotification
        ? _RegionNotification(
            id: regionNotificationId(regionPath, dem: false),
            title: _regionDisplayName(regionPath),
            label: 'map',
          )
        : null;
    _showRegionProgress(notification, 0);

    final future = ref
        .read(tileRepositoryManagerProvider)
        .startVectorDownload(
          regionPath,
          onProgress: (received, total) {
            if (total <= 0) return;
            // A cancel clears the entry synchronously; never let a late
            // in-flight progress callback resurrect it into `downloading`.
            if (state[regionPath]?.vectorProgress == null) return;
            state = {
              ...state,
              regionPath: (state[regionPath] ?? const RegionDownloadState())
                  .copyWith(vectorProgress: received / total),
            };
            _showRegionProgress(notification, received / total);
          },
        );
    _activeVectorDownloads[regionPath] = future;

    try {
      await future;
    } finally {
      // Only THIS download owns the clear — a superseding download (started
      // after a cancel) has already replaced the map entry and its own
      // progress, which must survive. The notification terminal is inside the
      // same guard for the same reason: a superseded download must not report
      // an outcome over its successor's live progress notification.
      if (identical(_activeVectorDownloads[regionPath], future)) {
        _activeVectorDownloads.remove(regionPath);
        _clearVectorProgress(regionPath);
        _refreshRegionSnapshot();
        _finishRegionNotification(
          notification,
          _freshRegionByPath(regionPath)?.vectorPackage.target?.status,
        );
      }
    }
  }

  /// Starts a fresh DEM download for [regionPath]. Fully independent from
  /// [downloadVector] -- a DEM failure never touches the vector entry, and
  /// vice versa.
  Future<void> downloadDem(
    String regionPath, {
    bool showNotification = true,
  }) async {
    if (state[regionPath]?.demProgress != null) return;

    state = {
      ...state,
      regionPath: (state[regionPath] ?? const RegionDownloadState()).copyWith(
        demProgress: 0,
      ),
    };

    final notification = showNotification
        ? _RegionNotification(
            id: regionNotificationId(regionPath, dem: true),
            title: _regionDisplayName(regionPath),
            label: 'elevation data',
          )
        : null;
    _showRegionProgress(notification, 0);

    final future = ref
        .read(tileRepositoryManagerProvider)
        .startDemDownload(
          regionPath,
          onProgress: (received, total) {
            if (total <= 0) return;
            if (state[regionPath]?.demProgress == null) return;
            state = {
              ...state,
              regionPath: (state[regionPath] ?? const RegionDownloadState())
                  .copyWith(demProgress: received / total),
            };
            _showRegionProgress(notification, received / total);
          },
        );
    _activeDemDownloads[regionPath] = future;

    try {
      await future;
    } finally {
      if (identical(_activeDemDownloads[regionPath], future)) {
        _activeDemDownloads.remove(regionPath);
        _clearDemProgress(regionPath);
        _refreshRegionSnapshot();
        _finishRegionNotification(
          notification,
          _freshRegionByPath(regionPath)?.demPackage.target?.status,
        );
      }
    }
  }

  /// Cancels [regionPath]'s in-flight vector download and resets the UI
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
  void cancelVector(String regionPath) {
    ref.read(tileRepositoryManagerProvider).cancelVectorDownload(regionPath);
    _clearVectorProgress(regionPath);
    _refreshRegionSnapshot();
    // Dismissed HERE, synchronously, rather than left to the download's own
    // terminal handler: cancel deliberately does not await the Dio unwind, so
    // that handler may be seconds away on a stalled transfer, and the
    // notification must not keep claiming a download that the user just
    // stopped. The terminal handler's later dismiss of the same id is a no-op.
    _dismissRegionNotification(regionNotificationId(regionPath, dem: false));
  }

  /// Cancels [regionPath]'s in-flight DEM download. See [cancelVector].
  void cancelDem(String regionPath) {
    ref.read(tileRepositoryManagerProvider).cancelDemDownload(regionPath);
    _clearDemProgress(regionPath);
    _refreshRegionSnapshot();
    _dismissRegionNotification(regionNotificationId(regionPath, dem: true));
  }

  /// Deletes [regionPath]'s downloaded packages (vector AND DEM) + on-disk
  /// files and clears its tracked state.
  Future<void> delete(String regionPath) async {
    try {
      await ref.read(tileRepositoryManagerProvider).deleteRegion(regionPath);
      state = {...state}..remove(regionPath);
    } finally {
      _refreshRegionSnapshot();
    }
  }

  /// Removes [regionPath]'s DEM package only -- the vector package and
  /// its lifecycle are untouched. Mirrors [delete]'s ephemeral-state
  /// clearing but performs no progress tracking (pure removal, not a status
  /// transition).
  Future<void> deleteDemPackage(String regionPath) async {
    try {
      await ref
          .read(tileRepositoryManagerProvider)
          .deleteDemPackage(regionPath);
    } finally {
      _clearDemProgress(regionPath);
      _refreshRegionSnapshot();
    }
  }

  /// Drops the persisted region snapshot so the next read rebuilds it from
  /// ObjectBox with FRESH `RegionEntity` instances.
  ///
  /// This MUST live here rather than in the calling widget: ObjectBox
  /// `ToOne.target` caches permanently per Dart object instance (see
  /// `util/region/tile_status.dart`), so a snapshot taken mid-download keeps
  /// reporting `downloading` forever. The Settings/Regions screen used to own
  /// this invalidation behind a `mounted` guard, which silently skipped it
  /// whenever the user navigated away before the download finished — the
  /// completed region then rendered as `notDownloaded` on return. This
  /// notifier is `keepAlive`, so it outlives any screen and always fires.
  void _refreshRegionSnapshot() {
    ref.invalidate(regionListNotifierProvider);
  }

  /// The freshest persisted row for [regionPath], or null if it is gone.
  /// Always called AFTER [_refreshRegionSnapshot], so the returned entity's
  /// `ToOne` targets resolve against current data rather than a cached
  /// mid-download value (see `util/region/tile_status.dart`).
  RegionEntity? _freshRegionByPath(String regionPath) {
    for (final region in ref.read(regionListNotifierProvider)) {
      if (region.path == regionPath) return region;
    }
    return null;
  }

  /// Display title for a region's notifications. Falls back to the path when
  /// the catalog row is missing — an unnamed notification is far better than
  /// no notification, and this must never throw into a download's start path.
  String _regionDisplayName(String regionPath) =>
      _freshRegionByPath(regionPath)?.name ?? regionPath;

  /// Pushes a progress update, throttled to whole-percent changes. The
  /// download's `onProgress` fires per received chunk — pushing every one
  /// across the platform channel would be thousands of no-op notification
  /// rebuilds for a single large archive.
  void _showRegionProgress(_RegionNotification? notification, double fraction) {
    if (notification == null) return;
    final percent = (fraction.clamp(0.0, 1.0) * 100).round();
    if (_lastNotifiedPercent[notification.id] == percent) return;
    _lastNotifiedPercent[notification.id] = percent;

    ref
        .read(downloadNotificationServiceProvider)
        .showRegionProgress(
          notification.id,
          notification.title,
          'Downloading ${notification.label}… $percent%',
          percent,
        )
        // Best-effort throughout: a denied notification permission or a
        // platform-channel failure must never fail the download itself.
        .catchError((_) {});
  }

  /// Terminal state for one package's notification, chosen from the PERSISTED
  /// package status rather than from whether the future threw — the manager
  /// swallows Dio failures and records them as [PackageStatus.error], so a
  /// completed future says nothing about the outcome on its own.
  void _finishRegionNotification(
    _RegionNotification? notification,
    PackageStatus? status,
  ) {
    if (notification == null) return;
    _lastNotifiedPercent.remove(notification.id);

    final service = ref.read(downloadNotificationServiceProvider);
    switch (status) {
      case PackageStatus.downloaded:
        service
            .showRegionResult(
              notification.id,
              notification.title,
              'Saved for offline use',
            )
            .catchError((_) {});
      case PackageStatus.error:
        service
            .showRegionResult(
              notification.id,
              notification.title,
              'Download failed',
            )
            .catchError((_) {});
      // notDownloaded (a cancel), downloading (a write that hasn't landed),
      // or a vanished row: no outcome worth reporting, so clear the ongoing
      // notification rather than leaving it stuck at its last percentage.
      case PackageStatus.notDownloaded:
      case PackageStatus.downloading:
      case null:
        _dismissRegionNotification(notification.id);
    }
  }

  void _dismissRegionNotification(int id) {
    _lastNotifiedPercent.remove(id);
    ref
        .read(downloadNotificationServiceProvider)
        .dismissRegion(id)
        .catchError((_) {});
  }

  /// Clears only the vector progress field, preserving a concurrently
  /// in-flight DEM download's entry -- removing the whole [regionPath] key
  /// outright would wipe `demProgress` out from under it.
  void _clearVectorProgress(String regionPath) {
    final current = state[regionPath];
    if (current == null) return;
    if (current.demProgress == null) {
      state = {...state}..remove(regionPath);
    } else {
      state = {...state, regionPath: current.copyWith(vectorProgress: null)};
    }
  }

  /// See [_clearVectorProgress] -- the DEM-side mirror.
  void _clearDemProgress(String regionPath) {
    final current = state[regionPath];
    if (current == null) return;
    if (current.vectorProgress == null) {
      state = {...state}..remove(regionPath);
    } else {
      state = {...state, regionPath: current.copyWith(demProgress: null)};
    }
  }
}

/// The immutable identity of one package download's notification, resolved
/// once at download start. Null wherever a caller opted out of notifications
/// (see [TileRepositoryStatus.downloadVector]'s `showNotification`), which is
/// what every notification helper treats as "do nothing".
class _RegionNotification {
  const _RegionNotification({
    required this.id,
    required this.title,
    required this.label,
  });

  final int id;

  /// The region's display name — the notification title.
  final String title;

  /// Which package this is, as it appears mid-sentence in the body copy
  /// ("Downloading map… 40%"). Matches the hardcoded-English convention the
  /// rest of `DownloadNotificationService` already uses; these strings never
  /// reach a `BuildContext`, so there is no `AppLocalizations` to read.
  final String label;
}
