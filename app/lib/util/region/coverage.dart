/// Pure, synchronous trail/region coverage logic.
///
/// Decides — from a trail's bbox and a local region catalog snapshot —
/// which overlapping regions are NOT yet usable offline. `overlappingRegions`
/// and `missingCoverageRegions` are kept as two distinct functions so
/// a caller can tell "fully covered" (overlapping non-empty, missing empty)
/// apart from "no region overlaps at all" (both empty).
///
/// No Riverpod, ObjectBox I/O, or network access here — these are pure
/// functions over already-fetched inputs; the call site (Plan 03) passes a
/// `regionListNotifierProvider` snapshot in.
library;

import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/models/trail.dart';

/// True iff the two axis-aligned rectangles (each described by four
/// `min`/`max` doubles) share any area — edges touching counts as overlap.
///
/// A degenerate box where `minLon > maxLon` or `minLat > maxLat` (e.g. from
/// a corrupted local row) naturally fails one of the four
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

/// Whether [region] can actually be downloaded from the server right now: it
/// is still present in the latest catalog fetch ([RegionEntity.inCatalog])
/// AND the server currently advertises a ready vector archive for it
/// ([RegionEntity.vectorUrl] non-null — `RegionsList` sets it only when the
/// region's status is "ready" AND the archive file exists on disk).
///
/// A region that is still building, errored, or has been removed from the
/// server (orphaned, `inCatalog == false`, keeping only a stale `vectorUrl`)
/// is NOT downloadable and must never be offered — starting its download would
/// only fail with "no vector archive available yet".
bool isRegionDownloadable(RegionEntity region) =>
    region.inCatalog && region.vectorUrl != null;

/// Whether [region] already satisfies offline coverage: its tiles are on the
/// device ([RegionStatus.downloaded]) or on the device but merely stale
/// ([RegionStatus.updateAvailable]) — stale counts as covered so a
/// merely-out-of-date region never re-fires the guard. Coverage here is
/// independent of the server: a downloaded region still covers even if it was
/// later removed from the catalog.
bool _coversOffline(RegionEntity region) =>
    region.status == RegionStatus.downloaded ||
    region.status == RegionStatus.updateAvailable;

/// Overlapping regions that are usable as offline coverage for [trail] —
/// either already covering offline ([_coversOffline]) or downloadable from the
/// server right now ([isRegionDownloadable]). Regions that overlap but can't
/// be used (still building/errored, or removed from the server) are excluded,
/// so a trail covered ONLY by such regions is reported as having no offered
/// coverage (empty result) rather than being silently treated as covered.
List<RegionEntity> usableCoverageRegions(
  Trail trail,
  List<RegionEntity> catalog,
) {
  return overlappingRegions(trail, catalog)
      .where((region) => _coversOffline(region) || isRegionDownloadable(region))
      .toList();
}

/// The overlapping regions the missing-coverage sheet may actually offer:
/// downloadable now ([isRegionDownloadable]) but not yet covering offline
/// ([_coversOffline]). A region that overlaps but isn't downloadable (still
/// building/errored on the server, or removed from it entirely) is NEVER
/// offered — the sheet would otherwise present a Download action that can only
/// fail.
List<RegionEntity> missingCoverageRegions(
  Trail trail,
  List<RegionEntity> catalog,
) {
  return overlappingRegions(trail, catalog)
      .where(
        (region) => isRegionDownloadable(region) && !_coversOffline(region),
      )
      .toList();
}
