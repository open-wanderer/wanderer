import 'package:wanderer/models/region_download_state.dart';
import 'package:wanderer/models/region_status.dart';

/// Pure resolver for the `RegionStatus` a Settings/Regions row should
/// render (24-04 gap closure — see
/// `.planning/debug/region-download-stale-toone.md`).
///
/// `persisted` (`RegionEntity.status`) is computed from an ObjectBox `ToOne`
/// relation that lazily resolves ONCE per Dart object instance and then
/// caches that value PERMANENTLY for the lifetime of that instance
/// (confirmed by reading the installed `objectbox` 5.3.1 package source,
/// `~/.pub-cache/hosted/pub.dev/objectbox-5.3.1/lib/src/relations/to_one.dart`
/// — there is no TTL, invalidation hook, or listener). The Settings/Regions
/// screen holds its `RegionEntity` instances across the whole duration of a
/// download, so `persisted` is stale for that entire duration and only
/// becomes fresh again the instant `_save`'s terminal
/// `ref.invalidate(regionListNotifierProvider)` fires.
///
/// `live` is the ephemeral `RegionDownloadState` from
/// `tileRepositoryStatusProvider`, an in-memory Riverpod map that is immune
/// to the ObjectBox `ToOne` cache and updates on every progress callback —
/// so it is preferred over `persisted` whenever a VECTOR mutation is in
/// flight.
///
/// A DEM-only download (`live.demProgress != null` and
/// `live.vectorProgress == null`) is deliberately EXCLUDED from the
/// override: `region.status` intentionally tracks only the VECTOR package
/// lifecycle (see `RegionEntity.status`'s own doc comment), and the DEM
/// download surfaces its own inline spinner — letting it flip the whole row
/// to `downloading` would hide a downloaded region's checkmark/delete
/// action and regress UAT test 2 (SETUI-04).
RegionStatus resolveRowStatus(
  RegionStatus persisted,
  RegionDownloadState? live,
) {
  if (live == null) return persisted;
  final isDemOnly = live.demProgress != null && live.vectorProgress == null;
  if (isDemOnly) return persisted;
  return live.status;
}
