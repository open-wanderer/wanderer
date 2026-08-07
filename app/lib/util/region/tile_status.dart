import 'package:wanderer/models/region_status.dart';

/// Pure resolvers for the status each of the Settings/Regions row's two
/// tiles (Vector, Elevation data) should render.
///
/// `persisted` (`RegionEntity.status` / `DownloadedTilePackageEntity.status`
/// read through the `demPackage` `ToOne`) is computed from an ObjectBox
/// `ToOne` relation that lazily resolves ONCE per Dart object instance and
/// then caches that value PERMANENTLY for the lifetime of that instance
/// (confirmed by reading the installed `objectbox` 5.3.1 package source,
/// `~/.pub-cache/hosted/pub.dev/objectbox-5.3.1/lib/src/relations/to_one.dart`
/// — there is no TTL, invalidation hook, or listener). The Settings/Regions
/// screen holds its `RegionEntity` instances across the whole duration of a
/// download, so `persisted` is stale for that entire duration and only
/// becomes fresh again the instant the screen's `_save` wrapper's terminal
/// `ref.invalidate(regionListNotifierProvider)` fires.
///
/// `liveProgress` is the ephemeral per-package progress fraction from
/// `tileRepositoryStatusProvider`'s `RegionDownloadState` — an in-memory
/// Riverpod value immune to the ObjectBox `ToOne` cache, present exactly
/// while that package's download is in flight. It is the SINGLE source of
/// truth for the `downloading` state, since a download is inherently a live,
/// in-memory activity — vector and DEM downloads are fully independent (each
/// tile owns its own `CancelToken` and progress stream).
///
/// A persisted `downloading` with NO live progress is therefore treated as
/// stale, not as a live download: it means the package was mid-download when
/// the in-memory tracker was lost (a cancel whose ObjectBox `notDownloaded`
/// write hasn't landed yet, or a download killed by an app kill/crash). The
/// resolver presents it as not-downloaded so the tile shows a Download
/// button rather than a frozen, never-advancing progress bar. Relying on the
/// persisted `downloading` value here is exactly what left the tile stuck
/// after Cancel.

/// Resolves the Vector tile's status.
RegionStatus resolveVectorTileStatus(
  RegionStatus persisted,
  double? liveVectorProgress,
) {
  if (liveVectorProgress != null) return RegionStatus.downloading;
  if (persisted == RegionStatus.downloading) return RegionStatus.notDownloaded;
  return persisted;
}

/// Resolves the Elevation data (DEM) tile's status.
PackageStatus resolveDemTileStatus(
  PackageStatus persisted,
  double? liveDemProgress,
) {
  if (liveDemProgress != null) return PackageStatus.downloading;
  if (persisted == PackageStatus.downloading) {
    return PackageStatus.notDownloaded;
  }
  return persisted;
}
