/// Region DB-to-disk self-heal (T-h2p-05/T-h2p-06).
///
/// The disk-usage summary (`region_disk_usage_util.dart`) stats the
/// filesystem directly, while a region's reported [RegionStatus] and the
/// paths handed to the offline map derive from `RegionEntity.vectorPackage`/
/// `demPackage` (`region_entity.dart`, `splitRegionTilePaths` in
/// `tile_repository_manager.dart`). ObjectBox does not cascade or repair a
/// lost `ToOne` link, so once those two views drift apart a region can be
/// counted in MB by the summary, reported not-downloaded by its row, and
/// skipped entirely by the offline map — all at once. This util reconciles
/// the persisted package status against what is actually on disk at
/// startup, and sweeps region directories that match no persisted row.
///
/// Every filesystem path in this file goes through `region_file_path.dart`'s
/// builders — never string concatenation.
library;

import 'dart:io';

import 'package:path/path.dart' as p;
import 'package:wanderer/entities/downloaded_tile_package_entity.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/util/region_file_path.dart';

/// The repair a single package row needs, resolved from whether its archive
/// exists on disk versus what the row currently claims.
enum PackageRepair { none, markDownloaded, markNotDownloaded }

/// Pure resolver: what repair (if any) a package row needs given whether its
/// archive file exists on disk and what status is currently persisted.
///
/// A final-named archive needs no re-validation here — the download path
/// (`TileRepositoryManager`) only renames a `.part` file to its final name
/// after `_isValidPmTiles` has already passed, so the mere presence of a
/// final-named archive is already a validity guarantee.
///
/// A `downloading` persisted status is safely repaired by this resolver:
/// this pass runs once at startup, before any download can possibly be
/// in flight, so a row still claiming `downloading` is necessarily stale
/// (an interrupted app termination mid-download), not a real active
/// transfer.
PackageRepair resolvePackageRepair({
  required bool archiveExists,
  required PackageStatus? persisted,
}) {
  if (archiveExists && persisted != PackageStatus.downloaded) {
    return PackageRepair.markDownloaded;
  }
  if (!archiveExists && persisted == PackageStatus.downloaded) {
    return PackageRepair.markNotDownloaded;
  }
  return PackageRepair.none;
}

/// The subset of [dirNames] that are unreachable disk: they pass
/// [isValidRegionId] (so they're a plausible region directory, not `..` or
/// some unrelated file) but are absent from [knownRegionIds].
///
/// Returns empty whenever [knownRegionIds] is empty — the important guard:
/// on a fresh install, or before the first catalog fetch, there are zero
/// region rows, and without this guard the sweep would delete every region
/// archive on the device.
Set<String> orphanedRegionDirNames(
  Iterable<String> dirNames,
  Iterable<String> knownRegionIds,
) {
  final known = knownRegionIds.toSet();
  if (known.isEmpty) return {};

  return dirNames
      .where((name) => isValidRegionId(name) && !known.contains(name))
      .toSet();
}

/// Result of a [reconcileRegionPackagesWithDisk] pass.
typedef RegionDiskReconcileResult = ({
  int repairedDownloaded,
  int repairedNotDownloaded,
  int orphanDirsDeleted,
});

const _zeroResult = (
  repairedDownloaded: 0,
  repairedNotDownloaded: 0,
  orphanDirsDeleted: 0,
);

/// Reconciles every [RegionEntity] row's persisted vector/DEM package status
/// against the archives actually present under [root], then sweeps any
/// region directory matching no persisted row.
///
/// Never throws — the whole body is wrapped in try/catch and returns zeroed
/// counts on any failure, so a reconcile fault can never block app startup
/// (T-h2p-06).
Future<RegionDiskReconcileResult> reconcileRegionPackagesWithDisk(
  Store store,
  String root,
) async {
  try {
    var repairedDownloaded = 0;
    var repairedNotDownloaded = 0;

    final regionBox = store.box<RegionEntity>();
    final regions = regionBox.getAll();

    for (final region in regions) {
      // Cheap gate: skip regions with no storage dir at all before paying
      // any per-package stat calls, keeping a large catalog affordable.
      if (!Directory(regionStorageDir(root, region.id)).existsSync()) {
        continue;
      }

      final vectorRepair = resolvePackageRepair(
        archiveExists: File(regionVectorPath(root, region.id)).existsSync(),
        persisted: region.vectorPackage.target?.status,
      );
      final demRepair = resolvePackageRepair(
        archiveExists: File(regionDemPath(root, region.id)).existsSync(),
        persisted: region.demPackage.target?.status,
      );

      if (vectorRepair != PackageRepair.none) {
        _applyRepair(
          store: store,
          regionId: region.id,
          dem: false,
          repair: vectorRepair,
          finalPath: regionVectorPath(root, region.id),
        );
        if (vectorRepair == PackageRepair.markDownloaded) {
          repairedDownloaded++;
        } else {
          repairedNotDownloaded++;
        }
      }

      if (demRepair != PackageRepair.none) {
        _applyRepair(
          store: store,
          regionId: region.id,
          dem: true,
          repair: demRepair,
          finalPath: regionDemPath(root, region.id),
        );
        if (demRepair == PackageRepair.markDownloaded) {
          repairedDownloaded++;
        } else {
          repairedNotDownloaded++;
        }
      }
    }

    final orphanDirsDeleted = _sweepOrphanDirs(root, regionBox.getAll());

    return (
      repairedDownloaded: repairedDownloaded,
      repairedNotDownloaded: repairedNotDownloaded,
      orphanDirsDeleted: orphanDirsDeleted,
    );
  } catch (_) {
    return _zeroResult;
  }
}

/// Applies a single [PackageRepair] to [regionId]'s vector ([dem] false) or
/// DEM ([dem] true) package inside one write transaction, re-reading the
/// row by business id first (mirrors `TileRepositoryManager`'s fresh-row
/// read-modify-write shape) so a concurrently-linked sibling package FK is
/// never clobbered.
///
/// Every branch that mutates a [DownloadedTilePackageEntity] MUST also
/// `put` it on its OWN box — do not "simplify" these calls away. Putting
/// the owning `RegionEntity` is NOT enough: ObjectBox's `ToOne.applyToDb`
/// writes the target only `if (targetId == 0)`, i.e. only when the target
/// is brand new, so on an already-persisted package row every field change
/// (`status`, `localFilePath`, `sizeBytesOnDisk`, `downloadedAtUtc`) is
/// silently discarded. That is exactly how this util originally shipped
/// computing correct repairs that never reached the database — a region
/// whose archive was on disk kept reporting `notDownloaded` while its bytes
/// were still counted by `region_disk_usage_util.dart`. This mirrors
/// `TileRepositoryManager._updatePackageStatus`, the same pattern for the
/// same reason.
void _applyRepair({
  required Store store,
  required String regionId,
  required bool dem,
  required PackageRepair repair,
  required String finalPath,
}) {
  store.runInTransaction(TxMode.write, () {
    final query = store
        .box<RegionEntity>()
        .query(RegionEntity_.id.equals(regionId))
        .build();
    final freshRegion = query.findFirst();
    query.close();
    if (freshRegion == null) return;

    final toOne = dem ? freshRegion.demPackage : freshRegion.vectorPackage;

    switch (repair) {
      case PackageRepair.markDownloaded:
        final file = File(finalPath);
        var package = toOne.target;
        package ??= DownloadedTilePackageEntity();
        package.status = PackageStatus.downloaded;
        package.localFilePath = finalPath;
        package.sizeBytesOnDisk = file.lengthSync();
        package.downloadedAtUtc = file.statSync().modified.toUtc();
        toOne.target = package;

        // Vector-only: seed lastDownloadedVersion so this repaired row does
        // not immediately read as RegionStatus.updateAvailable with no
        // evidence a newer version exists. A genuine later server version
        // bump still flips it to updateAvailable because `version` itself
        // changes on the next catalog fetch.
        if (!dem) {
          freshRegion.lastDownloadedVersion ??= freshRegion.version;
        }
        // Region first: persists the relation id (assigning `toOne.target`
        // above only inserts a BRAND-NEW package, giving it its id) plus the
        // `lastDownloadedVersion` seed. Then the package on its own box, so
        // the four field writes above land for an EXISTING package row too.
        store.box<RegionEntity>().put(freshRegion);
        store.box<DownloadedTilePackageEntity>().put(package);
      case PackageRepair.markNotDownloaded:
        final package = toOne.target;
        if (package != null) {
          package.status = PackageStatus.notDownloaded;
          // The region row itself is unchanged in this branch — only the
          // package box put matters. `package` is necessarily already
          // persisted here (it came from `toOne.target`), so an owner put
          // would have been a guaranteed no-op.
          store.box<DownloadedTilePackageEntity>().put(package);
        }
      case PackageRepair.none:
        break;
    }
  });
}

/// Deletes every directory under `<root>/regions` matching no persisted
/// region row (T-h2p-05). The tile proxy resolves archives strictly from
/// `RegionEntity` rows (`tile_proxy_server.dart` reads
/// `box<RegionEntity>().getAll()`), so a directory matching no row can never
/// be served, shown, or deleted through the UI — it is unreachable disk.
int _sweepOrphanDirs(String root, Iterable<RegionEntity> regions) {
  final regionsRoot = Directory(p.join(root, 'regions'));
  if (!regionsRoot.existsSync()) return 0;

  final dirNames = regionsRoot.listSync().whereType<Directory>().map(
    (d) => p.basename(d.path),
  );
  final knownIds = regions.map((r) => r.id);

  final orphans = orphanedRegionDirNames(dirNames, knownIds);

  var deleted = 0;
  for (final name in orphans) {
    try {
      Directory(p.join(regionsRoot.path, name)).deleteSync(recursive: true);
      deleted++;
    } catch (_) {
      // Best-effort — one failed deletion must never block the others.
    }
  }
  return deleted;
}
