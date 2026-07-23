import 'package:wanderer/models/region_status.dart';

/// Pure resolvers for the status each of the Settings/Regions row's two
/// tiles (Vector, Elevation data) should render (SETUI gap closure — see
/// `.planning/debug/region-download-stale-toone.md`).
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
/// while that package's download is in flight. Non-null always overrides
/// `persisted` with `downloading`, since vector and DEM downloads are fully
/// independent (each tile owns its own `CancelToken` and progress stream).

/// Resolves the Vector tile's status.
RegionStatus resolveVectorTileStatus(
  RegionStatus persisted,
  double? liveVectorProgress,
) {
  return liveVectorProgress != null ? RegionStatus.downloading : persisted;
}

/// Resolves the Elevation data (DEM) tile's status.
PackageStatus resolveDemTileStatus(
  PackageStatus persisted,
  double? liveDemProgress,
) {
  return liveDemProgress != null ? PackageStatus.downloading : persisted;
}
