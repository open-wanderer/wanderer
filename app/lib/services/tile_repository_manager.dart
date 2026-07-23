import 'dart:async';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter/widgets.dart';
import 'package:maplibre/maplibre.dart' show LngLatBounds;
import 'package:path_provider/path_provider.dart';
import 'package:pmtiles/pmtiles.dart';
import 'package:wanderer/entities/downloaded_tile_package_entity.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/util/disk_space_util.dart';
import 'package:wanderer/util/region_file_path.dart';

/// Pure axis-aligned rectangle-overlap test between a region's bbox and a
/// query [LngLatBounds] (TILE-05). There is no `intersects()` helper on this
/// package's `LngLatBounds`, so this is hand-rolled against `RegionEntity`'s
/// four bbox doubles as the negated-disjoint form: two rectangles overlap
/// unless one is entirely to one side of the other. Uses strict `</`/`>`
/// (not `<=`/`>=`) so an edge-touching pair still counts as overlapping.
@visibleForTesting
bool bboxOverlaps({
  required double minLon,
  required double minLat,
  required double maxLon,
  required double maxLat,
  required LngLatBounds query,
}) {
  return !(maxLon < query.longitudeWest ||
      minLon > query.longitudeEast ||
      maxLat < query.latitudeSouth ||
      minLat > query.latitudeNorth);
}

/// Pure per-region split of local vector/DEM archive paths for every
/// [regions] entry whose bbox overlaps [query] (RENDER-01) — the fix for
/// `localTilePathsForBounds` previously returning ONE flat, untyped
/// `List<String>` with vector and DEM paths interleaved and no
/// discriminator. Returning two SEPARATE lists makes it structurally
/// impossible for a DEM `.pmtiles` archive to be mis-fed into
/// `rewriteStyleForOffline`'s vector `cellPaths` param (which would route it
/// through the vector cell-duplication path instead of the DEM one —
/// reproducing the bug class already found and fixed once in commit
/// `3f67cf37`, quick-260711-lzb: "hillshadeSource swept into the vector-cell
/// repoint path").
///
/// Takes an `Iterable<RegionEntity>` (not `_store`) so this "which list does
/// a path land in" logic is unit-testable without a live ObjectBox store —
/// mirrors the [bboxOverlaps] precedent. Regions whose vector/DEM package
/// target is null (not downloaded) or whose bbox doesn't overlap [query]
/// contribute nothing to either list.
@visibleForTesting
({List<String> vectorPaths, List<String> demPaths}) splitRegionTilePaths(
  Iterable<RegionEntity> regions,
  LngLatBounds query,
) {
  final vectorPaths = <String>[];
  final demPaths = <String>[];
  for (final region in regions) {
    if (!bboxOverlaps(
      minLon: region.minLon,
      minLat: region.minLat,
      maxLon: region.maxLon,
      maxLat: region.maxLat,
      query: query,
    )) {
      continue;
    }

    final vectorPath = region.vectorPackage.target?.localFilePath;
    final demPath = region.demPackage.target?.localFilePath;
    if (vectorPath != null) vectorPaths.add(vectorPath);
    if (demPath != null) demPaths.add(demPath);
  }
  return (vectorPaths: vectorPaths, demPaths: demPaths);
}

/// Pure per-tile winner resolution among overlapping downloaded regions
/// (PROXY-02, D-02/D-03) — sibling to [bboxOverlaps]/[splitRegionTilePaths],
/// called once per incoming tile-proxy HTTP request (`tile_proxy_server.dart`)
/// rather than per viewport, so it must stay cheap.
///
/// Filters [regions] to those whose requested-kind package ([dem] selects
/// [RegionEntity.demPackage] vs [RegionEntity.vectorPackage]) has a non-null
/// `localFilePath` AND whose bbox overlaps [tileBounds] (via [bboxOverlaps]).
/// A region overlapping the tile but whose requested package target is null
/// (not downloaded) is NOT a candidate. Returns `null` when no candidate
/// remains.
///
/// Among overlapping candidates, the winner is the SMALLEST planar bbox area
/// (`(maxLon-minLon)*(maxLat-minLat)`) — the most specific/local region wins
/// over a broader one that happens to also cover the same spot (D-02). Plain
/// planar degree² area is deliberate, not a great-circle/projected area:
/// regions are hand-curated, admin-defined bboxes, not user-drawn shapes, so
/// latitude-dependent distortion is irrelevant to the "which region feels
/// more local" UX intent (RESEARCH.md Assumption A1). Ties in area are broken
/// by the requested package's `downloadedAtUtc` descending — most-recently-
/// downloaded wins (D-03); a candidate whose package `downloadedAtUtc` is
/// null sorts as epoch-zero (loses to any dated one).
@visibleForTesting
RegionEntity? resolveRegionForTile(
  Iterable<RegionEntity> regions,
  LngLatBounds tileBounds, {
  required bool dem,
}) {
  final candidates = regions.where((region) {
    final target = dem ? region.demPackage.target : region.vectorPackage.target;
    if (target?.localFilePath == null) return false;
    return bboxOverlaps(
      minLon: region.minLon,
      minLat: region.minLat,
      maxLon: region.maxLon,
      maxLat: region.maxLat,
      query: tileBounds,
    );
  }).toList();

  if (candidates.isEmpty) return null;

  candidates.sort((a, b) {
    final areaA = (a.maxLon - a.minLon) * (a.maxLat - a.minLat);
    final areaB = (b.maxLon - b.minLon) * (b.maxLat - b.minLat);
    final areaCompare = areaA.compareTo(areaB);
    if (areaCompare != 0) return areaCompare;

    // Tie-break: most-recently-downloaded wins (D-03).
    final aTarget = dem ? a.demPackage.target : a.vectorPackage.target;
    final bTarget = dem ? b.demPackage.target : b.vectorPackage.target;
    final aTime =
        aTarget?.downloadedAtUtc ?? DateTime.fromMillisecondsSinceEpoch(0);
    final bTime =
        bTarget?.downloadedAtUtc ?? DateTime.fromMillisecondsSinceEpoch(0);
    return bTime.compareTo(aTime); // descending — most recent first
  });

  return candidates.first;
}

/// Owns the region tile-repository download lifecycle (TILE-01..05,
/// DEM-01/02): a fresh (never resumed) `.part` download for a region's
/// vector and (independently) DEM archives, a disk-space pre-check before
/// every file write, `PmTilesArchive` validation before a `.part` is
/// promoted to its final path, a bbox-overlap viewport query
/// (`localTilePathsForBounds`), and a cascade delete (`deleteRegion`) since
/// ObjectBox does not cascade a `ToOne` target's removal.
///
/// There is no pause/resume: cancelling an in-flight download (deliberately,
/// via [cancelVectorDownload]/[cancelDemDownload], or the transfer erroring
/// out) always deletes the `.part` file and returns the package to
/// `notDownloaded`/`error`. A later download attempt always starts from
/// byte 0. Downloads keep running while the app is backgrounded — there is
/// no lifecycle-driven auto-cancel — since cancelling would now mean losing
/// progress rather than merely pausing it.
///
/// Mirrors `TrailDownloadService`'s construction-injection
/// (`Store` + `Dio`) and `CancelToken`-based cancellation shape, but adds
/// the region-archive-scale concerns that phase's small per-cell files
/// never needed. The Riverpod wiring lives in `tile_repository_provider.dart`
/// (Plan 05).
class TileRepositoryManager {
  final Store _store;
  final Dio _api;

  /// One `CancelToken` per active (region, package-kind) download, keyed
  /// `'<regionId>:vector'` / `'<regionId>:dem'` so vector and DEM downloads
  /// for the same region can run concurrently without clobbering each
  /// other's token.
  final Map<String, CancelToken> _activeCancelTokens = {};

  TileRepositoryManager(this._store, this._api);

  /// Starts a fresh vector archive download for [regionId]. Refuses to
  /// write any bytes — marking the vector package `error` instead — when
  /// disk space is tight (TILE-03).
  Future<void> startVectorDownload(
    String regionId, {
    void Function(int received, int total)? onProgress,
  }) async {
    final id = assertValidRegionId(regionId);
    final region = _regionById(id);
    if (region == null) {
      throw StateError('Unknown region: $id');
    }
    if (region.vectorUrl == null) {
      throw StateError('Region $id has no vector archive available yet');
    }

    final root = (await getApplicationDocumentsDirectory()).path;
    final finalPath = regionVectorPath(root, id);
    final partPath = '$finalPath.part';

    final free = await freeDiskSpaceBytes(regionStorageDir(root, id));
    if (!hasEnoughSpace(
      freeBytes: free,
      declaredSizeBytes: region.vectorSize ?? 0,
    )) {
      final package = _getOrCreatePackage(region, region.vectorPackage);
      _updatePackageStatus(package, status: PackageStatus.error);
      return;
    }

    final package = _getOrCreatePackage(region, region.vectorPackage);
    _updatePackageStatus(package, status: PackageStatus.downloading);

    Directory(regionStorageDir(root, id)).createSync(recursive: true);

    final tokenKey = '$id:vector';
    final token = CancelToken();
    _activeCancelTokens[tokenKey] = token;

    final partFile = File(partPath);

    try {
      await _download(
        requestPath: _requestPathFor(id, dem: false),
        partPath: partPath,
        cancelToken: token,
        onProgress: onProgress,
      );

      if (!await _isValidPmTiles(partPath)) {
        if (partFile.existsSync()) partFile.deleteSync();
        _updatePackageStatus(package, status: PackageStatus.error);
        return;
      }

      partFile.renameSync(finalPath);
      _updatePackageStatus(
        package,
        status: PackageStatus.downloaded,
        localFilePath: finalPath,
        sizeBytesOnDisk: File(finalPath).lengthSync(),
        downloadedAtUtc: DateTime.now().toUtc(),
      );

      region.lastDownloadedVersion = region.version;
      _store.runInTransaction(TxMode.write, () {
        _store.box<RegionEntity>().put(region);
      });
    } on DioException catch (e) {
      if (partFile.existsSync()) partFile.deleteSync();
      _updatePackageStatus(
        package,
        status: CancelToken.isCancel(e)
            ? PackageStatus.notDownloaded
            : PackageStatus.error,
      );
    } finally {
      // Only remove OUR token — a cancel-then-redownload may have already
      // replaced this key with the new download's token, which must stay
      // cancellable.
      if (identical(_activeCancelTokens[tokenKey], token)) {
        _activeCancelTokens.remove(tokenKey);
      }
    }
  }

  /// Identical to [startVectorDownload] but keyed off the region's DEM
  /// archive (`demUrl`/`demSize`/`demPackage`). Never touches
  /// `lastDownloadedVersion` — DEM has no staleness concept (matches
  /// `RegionEntity`'s own doc comment). A DEM failure never marks the
  /// vector package, keeping the two lifecycles fully independent.
  Future<void> startDemDownload(
    String regionId, {
    void Function(int received, int total)? onProgress,
  }) async {
    final id = assertValidRegionId(regionId);
    final region = _regionById(id);
    if (region == null) {
      throw StateError('Unknown region: $id');
    }
    if (region.demUrl == null) {
      throw StateError('Region $id has no DEM archive available yet');
    }

    final root = (await getApplicationDocumentsDirectory()).path;
    final finalPath = regionDemPath(root, id);
    final partPath = '$finalPath.part';

    final free = await freeDiskSpaceBytes(regionStorageDir(root, id));
    if (!hasEnoughSpace(
      freeBytes: free,
      declaredSizeBytes: region.demSize ?? 0,
    )) {
      final package = _getOrCreatePackage(region, region.demPackage);
      _updatePackageStatus(package, status: PackageStatus.error);
      return;
    }

    final package = _getOrCreatePackage(region, region.demPackage);
    _updatePackageStatus(package, status: PackageStatus.downloading);

    Directory(regionStorageDir(root, id)).createSync(recursive: true);

    final tokenKey = '$id:dem';
    final token = CancelToken();
    _activeCancelTokens[tokenKey] = token;

    final partFile = File(partPath);

    try {
      await _download(
        requestPath: _requestPathFor(id, dem: true),
        partPath: partPath,
        cancelToken: token,
        onProgress: onProgress,
      );

      if (!await _isValidPmTiles(partPath)) {
        if (partFile.existsSync()) partFile.deleteSync();
        _updatePackageStatus(package, status: PackageStatus.error);
        return;
      }

      partFile.renameSync(finalPath);
      _updatePackageStatus(
        package,
        status: PackageStatus.downloaded,
        localFilePath: finalPath,
        sizeBytesOnDisk: File(finalPath).lengthSync(),
        downloadedAtUtc: DateTime.now().toUtc(),
      );
    } on DioException catch (e) {
      if (partFile.existsSync()) partFile.deleteSync();
      _updatePackageStatus(
        package,
        status: CancelToken.isCancel(e)
            ? PackageStatus.notDownloaded
            : PackageStatus.error,
      );
    } finally {
      if (identical(_activeCancelTokens[tokenKey], token)) {
        _activeCancelTokens.remove(tokenKey);
      }
    }
  }

  /// Cancels [regionId]'s in-flight vector download, if any — a no-op when
  /// nothing is currently downloading. The `.part` file is deleted (no
  /// resume support): a later download tap always starts fresh from byte 0.
  /// Independent of [cancelDemDownload] (separate `CancelToken`).
  void cancelVectorDownload(String regionId) {
    final id = assertValidRegionId(regionId);
    _activeCancelTokens['$id:vector']?.cancel('cancelled');
  }

  /// Cancels [regionId]'s in-flight DEM download, if any. See
  /// [cancelVectorDownload].
  void cancelDemDownload(String regionId) {
    final id = assertValidRegionId(regionId);
    _activeCancelTokens['$id:dem']?.cancel('cancelled');
  }

  /// Returns the local vector/DEM archive file paths for every downloaded
  /// region whose bbox overlaps [query] (TILE-05, RENDER-01) — feeds Phase
  /// 25's viewport-based tile-reading pipeline. Vector and DEM paths are
  /// returned as two SEPARATE lists (never merged into one), so they can
  /// never be conflated when fed to `rewriteStyleForOffline`'s
  /// `cellPaths`/`demCellPaths` params — see [splitRegionTilePaths]'s doc
  /// comment for the exact bug class this prevents. Regions whose vector/DEM
  /// package target is null (not downloaded) or whose bbox doesn't overlap
  /// [query] contribute nothing to the result.
  ({List<String> vectorPaths, List<String> demPaths}) localTilePathsForBounds(
    LngLatBounds query,
  ) {
    return splitRegionTilePaths(_store.box<RegionEntity>().getAll(), query);
  }

  /// Cancels any in-flight vector/DEM download for [regionId], then removes
  /// both `DownloadedTilePackageEntity` rows and their on-disk files
  /// (vector, DEM, and any `.part` siblings) as one logical unit — ObjectBox
  /// does not cascade a `ToOne` target's deletion (RESEARCH.md Pitfall 5 /
  /// T-23-07). Deleting an unknown/never-downloaded region is a no-op.
  Future<void> deleteRegion(String regionId) async {
    final id = assertValidRegionId(regionId);

    for (final entry in _activeCancelTokens.entries.toList()) {
      if (entry.key == '$id:vector' || entry.key == '$id:dem') {
        entry.value.cancel('deleted');
      }
    }

    final region = _regionById(id);
    if (region == null) return;

    final vectorPackage = region.vectorPackage.target;
    final demPackage = region.demPackage.target;

    _store.runInTransaction(TxMode.write, () {
      final packageBox = _store.box<DownloadedTilePackageEntity>();
      if (vectorPackage != null) packageBox.remove(vectorPackage.obxId);
      if (demPackage != null) packageBox.remove(demPackage.obxId);

      region.vectorPackage.target = null;
      region.demPackage.target = null;
      region.lastDownloadedVersion = null;
      _store.box<RegionEntity>().put(region);
    });

    // Best-effort, outside the transaction: a missing file is never fatal.
    final root = (await getApplicationDocumentsDirectory()).path;
    for (final finalPath in [
      regionVectorPath(root, id),
      regionDemPath(root, id),
    ]) {
      for (final candidate in [finalPath, '$finalPath.part']) {
        final file = File(candidate);
        if (file.existsSync()) file.deleteSync();
      }
    }

    final dir = Directory(regionStorageDir(root, id));
    if (dir.existsSync() && dir.listSync().isEmpty) {
      dir.deleteSync();
    }
  }

  /// DEM-only cascade delete (D-01, T-24-02): removes ONLY [regionId]'s DEM
  /// package row and its on-disk file(s) — the sibling (non-DEM) package
  /// row, its file, and [RegionEntity.lastDownloadedVersion] are provably
  /// untouched, unlike [deleteRegion], which removes both packages together.
  /// Deleting an unknown/never-downloaded region is a no-op (matches
  /// [deleteRegion]).
  Future<void> deleteDemPackage(String regionId) async {
    final id = assertValidRegionId(regionId);

    for (final entry in _activeCancelTokens.entries.toList()) {
      if (entry.key == '$id:dem') {
        entry.value.cancel('deleted');
      }
    }

    final region = _regionById(id);
    if (region == null) return;

    final demPackage = region.demPackage.target;

    _store.runInTransaction(TxMode.write, () {
      if (demPackage != null) {
        _store.box<DownloadedTilePackageEntity>().remove(demPackage.obxId);
      }
      region.demPackage.target = null;
      _store.box<RegionEntity>().put(region);
    });

    // Best-effort, outside the transaction: a missing file is never fatal.
    final root = (await getApplicationDocumentsDirectory()).path;
    final finalPath = regionDemPath(root, id);
    for (final candidate in [finalPath, '$finalPath.part']) {
      final file = File(candidate);
      if (file.existsSync()) file.deleteSync();
    }
  }

  /// Cancels every remaining in-flight download (mirrors
  /// `navigation_screen.dart`'s dispose discipline).
  void dispose() {
    for (final token in _activeCancelTokens.values) {
      token.cancel('disposed');
    }
    _activeCancelTokens.clear();
  }

  RegionEntity? _regionById(String id) {
    final box = _store.box<RegionEntity>();
    final query = box.query(RegionEntity_.id.equals(id)).build();
    final region = query.findFirst();
    query.close();
    return region;
  }

  /// Builds `/regions/<validatedId>/download[-dem]` from the validated
  /// region id — NEVER from `region.vectorUrl`/`region.demUrl`. Those
  /// catalog-declared strings are `/api/v1/regions/<id>/download` (the
  /// backend prefixes them), but the Dio client's `baseUrl` already ends in
  /// `/api/v1`, so passing them verbatim would double-prefix to
  /// `/api/v1/api/v1/...` and 404. Building from the validated id also keeps
  /// the request path off any catalog-sourced string (defense in depth,
  /// T-23-01) — `vectorUrl`/`demUrl` are only used as an "archive is ready"
  /// availability gate, never as the request URL.
  String _requestPathFor(String validatedId, {required bool dem}) {
    return dem
        ? '/regions/$validatedId/download-dem'
        : '/regions/$validatedId/download';
  }

  /// Downloads [requestPath] to [partPath] from scratch, always overwriting
  /// any existing bytes at [partPath] (no resume). `deleteOnError: true`
  /// means Dio deletes the `.part` file on ANY termination that isn't a
  /// clean completion — including a deliberate [cancelVectorDownload]/
  /// [cancelDemDownload] cancellation — which is exactly the desired
  /// cancel-means-delete semantics now that there is no resume path. Never
  /// trusts the response's `Content-Length` header for anything beyond
  /// progress reporting — completion is validated separately by
  /// [_isValidPmTiles] before the caller promotes `.part` to its final path.
  Future<void> _download({
    required String requestPath,
    required String partPath,
    required CancelToken cancelToken,
    void Function(int received, int total)? onProgress,
  }) async {
    await _api.download(
      requestPath,
      partPath,
      cancelToken: cancelToken,
      deleteOnError: true,
      onReceiveProgress: onProgress,
    );
  }

  /// Never trust "file exists" as "download complete" — opens [path] with
  /// the `pmtiles` package's own header/magic-byte validator before it is
  /// ever promoted from `.part` to a final archive path.
  Future<bool> _isValidPmTiles(String path) async {
    try {
      final archive = await PmTilesArchive.fromFile(File(path));
      await archive.close();
      return true;
    } on CorruptArchiveException {
      return false;
    } on UnsupportedError {
      return false;
    }
  }

  /// Returns the existing package target on [toOne] if present, otherwise
  /// creates one and persists it (via `RegionEntity`'s own box — a `ToOne`
  /// with a new, unstored target is cascaded on `put`, which also stores
  /// the relation's foreign key on `region`).
  DownloadedTilePackageEntity _getOrCreatePackage(
    RegionEntity region,
    ToOne<DownloadedTilePackageEntity> toOne,
  ) {
    final existing = toOne.target;
    if (existing != null) return existing;

    final created = DownloadedTilePackageEntity();
    toOne.target = created;
    _store.runInTransaction(TxMode.write, () {
      _store.box<RegionEntity>().put(region);
    });
    return created;
  }

  /// Batches every status/byte-counter field write inside one
  /// `runInTransaction` (RESEARCH.md Pitfall 5 — never write per-byte).
  void _updatePackageStatus(
    DownloadedTilePackageEntity package, {
    required PackageStatus status,
    String? localFilePath,
    int? sizeBytesOnDisk,
    DateTime? downloadedAtUtc,
  }) {
    _store.runInTransaction(TxMode.write, () {
      package.status = status;
      if (localFilePath != null) package.localFilePath = localFilePath;
      if (sizeBytesOnDisk != null) package.sizeBytesOnDisk = sizeBytesOnDisk;
      if (downloadedAtUtc != null) package.downloadedAtUtc = downloadedAtUtc;
      _store.box<DownloadedTilePackageEntity>().put(package);
    });
  }
}
