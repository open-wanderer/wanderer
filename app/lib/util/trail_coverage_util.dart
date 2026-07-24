/// Pure, synchronous trail/region coverage logic (GUARD-01/GUARD-04).
///
/// Decides — from a trail's bbox and a local region catalog snapshot —
/// which overlapping regions are NOT yet usable offline. `overlappingRegions`
/// and `missingCoverageRegions` are kept as two distinct functions (D-04) so
/// a caller can tell "fully covered" (overlapping non-empty, missing empty)
/// apart from "no region overlaps at all" (both empty).
///
/// No Riverpod, ObjectBox I/O, or network access here — these are pure
/// functions over already-fetched inputs; the call site (Plan 03) passes a
/// `regionListNotifierProvider` snapshot in (D-11).
library;

import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/models/trail.dart';

/// True iff the two axis-aligned rectangles (each described by four
/// `min`/`max` doubles) share any area — edges touching counts as overlap.
///
/// A degenerate box where `minLon > maxLon` or `minLat > maxLat` (e.g. from
/// a corrupted local row, T-26-01) naturally fails one of the four
/// comparisons below and returns `false` rather than throwing.
bool bboxesOverlap({
  required double aMinLon,
  required double aMinLat,
  required double aMaxLon,
  required double aMaxLat,
  required double bMinLon,
  required double bMinLat,
  required double bMaxLon,
  required double bMaxLat,
}) {
  return aMinLon <= bMaxLon &&
      aMaxLon >= bMinLon &&
      aMinLat <= bMaxLat &&
      aMaxLat >= bMinLat;
}

/// Every region in [catalog] whose bbox overlaps [trail]'s bbox, regardless
/// of that region's download status.
List<RegionEntity> overlappingRegions(Trail trail, List<RegionEntity> catalog) {
  return catalog.where((region) {
    return bboxesOverlap(
      aMinLon: trail.minLon,
      aMinLat: trail.minLat,
      aMaxLon: trail.maxLon,
      aMaxLat: trail.maxLat,
      bMinLon: region.minLon,
      bMinLat: region.minLat,
      bMaxLon: region.maxLon,
      bMaxLat: region.maxLat,
    );
  }).toList();
}

/// The subset of `overlappingRegions(trail, catalog)` whose `status` is
/// neither [RegionStatus.downloaded] nor [RegionStatus.updateAvailable]
/// (GUARD-04: `updateAvailable` satisfies coverage identically to
/// `downloaded`, so a merely-stale region never re-fires the guard).
List<RegionEntity> missingCoverageRegions(
  Trail trail,
  List<RegionEntity> catalog,
) {
  return overlappingRegions(trail, catalog)
      .where(
        (region) =>
            region.status != RegionStatus.downloaded &&
            region.status != RegionStatus.updateAvailable,
      )
      .toList();
}
