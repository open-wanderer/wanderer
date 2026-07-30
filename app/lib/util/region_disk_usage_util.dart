/// Real on-disk byte usage for the region tile repository's disk-usage
/// summary (SETUI-05, D-06).
///
/// `DownloadedTilePackageEntity.sizeBytesOnDisk` is only ever populated on a
/// successful download completion (RESEARCH.md Pitfall 1) -- it stays null
/// for a `downloading`/`paused` package, silently under-reporting usage if
/// naively summed. This util instead stats the filesystem directly: a
/// region's usage is its final vector/DEM file size if present, else its
/// `.part` partial file size if present, else 0 -- so an in-progress or
/// paused download's partial bytes are still counted.
///
/// Every path is built via `region_file_path.dart`'s sanctioned builders
/// (never string-concatenated) so an invalid region path fails closed via
/// [assertValidRegionPath]'s `ArgumentError` (T-24-01).
library;

import 'dart:io';

import 'package:path_provider/path_provider.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/util/region_file_path.dart';

/// Sums the real on-disk bytes for [region]'s vector and DEM archives under
/// [root] -- final file bytes if present, else `.part` partial bytes if
/// present, else 0 for that package. Throws [ArgumentError] if
/// `region.path` is not a valid region path.
int regionDiskUsageBytes(String root, RegionEntity region) {
  final path = assertValidRegionPath(region.path);
  return _fileOrPartBytes(regionVectorPath(root, path)) +
      _fileOrPartBytes(regionDemPath(root, path));
}

int _fileOrPartBytes(String finalPath) {
  final finalFile = File(finalPath);
  if (finalFile.existsSync()) return finalFile.lengthSync();

  final partFile = File('$finalPath.part');
  if (partFile.existsSync()) return partFile.lengthSync();

  return 0;
}

/// Sums [regionDiskUsageBytes] across every region in [regions], resolving
/// the on-disk root once via `getApplicationDocumentsDirectory()`.
Future<int> totalRegionDiskUsageBytes(Iterable<RegionEntity> regions) async {
  final root = (await getApplicationDocumentsDirectory()).path;
  var total = 0;
  for (final region in regions) {
    total += regionDiskUsageBytes(root, region);
  }
  return total;
}
