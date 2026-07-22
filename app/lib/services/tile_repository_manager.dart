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

/// Pure decision for how a `.part` download should be (re)started, given the
/// number of bytes already present on disk.
///
/// - `0` bytes: a fresh download — write from scratch, no `Range` header,
///   and it's safe to let Dio delete the (empty/partial) file on error.
/// - `>0` bytes: a resumed download — append starting at the existing byte
///   offset via `Range: bytes=<offset>-`, and `deleteOnError` MUST be
///   `false`, or a second failure on the resumed request would destroy the
///   resume progress Dio just appended to (RESEARCH.md Pitfall 2).
@visibleForTesting
({int offset, FileAccessMode mode, bool sendRange, bool deleteOnError})
resumePlanFor(int existingPartBytes) {
  if (existingPartBytes <= 0) {
    return (
      offset: 0,
      mode: FileAccessMode.write,
      sendRange: false,
      deleteOnError: true,
    );
  }
  return (
    offset: existingPartBytes,
    mode: FileAccessMode.append,
    sendRange: true,
    deleteOnError: false,
  );
}

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

/// Owns the region tile-repository download lifecycle (TILE-01..05,
/// DEM-01/02): resumable `.part` downloads (Range + `FileAccessMode.append`)
/// for a region's vector and (independently) DEM archives, a disk-space
/// pre-check before every file write, `PmTilesArchive` validation before a
/// `.part` is promoted to its final path, an `AppLifecycleListener`-driven
/// deliberate pause of every in-flight download on backgrounding, a
/// bbox-overlap viewport query (`localTilePathsForBounds`), and a cascade
/// delete (`deleteRegion`) since ObjectBox does not cascade a `ToOne`
/// target's removal.
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

  late final AppLifecycleListener _lifecycleListener;

  TileRepositoryManager(this._store, this._api) {
    _lifecycleListener = AppLifecycleListener(onPause: _pauseAllActiveDownloads);
  }

  /// Starts (or resumes, if a `.part` file already exists on disk) the
  /// region's vector archive download. Refuses to write any bytes — marking
  /// the vector package `error` instead — when disk space is tight
  /// (TILE-03).
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
    final wasResuming = partFile.existsSync() && partFile.lengthSync() > 0;

    try {
      await _downloadResumable(
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
      if (CancelToken.isCancel(e)) {
        _updatePackageStatus(package, status: PackageStatus.paused);
        return;
      }
      if (!wasResuming && partFile.existsSync()) {
        partFile.deleteSync();
      }
      _updatePackageStatus(package, status: PackageStatus.error);
    } finally {
      _activeCancelTokens.remove(tokenKey);
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
    final wasResuming = partFile.existsSync() && partFile.lengthSync() > 0;

    try {
      await _downloadResumable(
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
      if (CancelToken.isCancel(e)) {
        _updatePackageStatus(package, status: PackageStatus.paused);
        return;
      }
      if (!wasResuming && partFile.existsSync()) {
        partFile.deleteSync();
      }
      _updatePackageStatus(package, status: PackageStatus.error);
    } finally {
      _activeCancelTokens.remove(tokenKey);
    }
  }

  /// Cancels every active vector/DEM download for [regionId] with a
  /// deliberate `'paused'` reason — the `CancelToken.isCancel` branch in
  /// [startVectorDownload]/[startDemDownload] preserves the `.part` file and
  /// marks the package `paused` rather than `error`.
  Future<void> pauseRegion(String regionId) async {
    final id = assertValidRegionId(regionId);
    for (final entry in _activeCancelTokens.entries.toList()) {
      if (entry.key == '$id:vector' || entry.key == '$id:dem') {
        entry.value.cancel('paused');
      }
    }
  }

  /// Re-invokes [startVectorDownload]/[startDemDownload] for whichever
  /// package(s) of [regionId] are currently `paused` — the resumable
  /// primitive's `.part`-length detection resumes automatically via Range.
  Future<void> resumeRegion(
    String regionId, {
    void Function(int received, int total)? onProgress,
  }) async {
    final id = assertValidRegionId(regionId);
    final region = _regionById(id);
    if (region == null) return;

    final futures = <Future<void>>[];
    if (region.vectorPackage.target?.status == PackageStatus.paused) {
      futures.add(startVectorDownload(id, onProgress: onProgress));
    }
    if (region.demPackage.target?.status == PackageStatus.paused) {
      futures.add(startDemDownload(id, onProgress: onProgress));
    }
    await Future.wait(futures);
  }

  /// Returns the local vector/DEM archive file paths for every downloaded
  /// region whose bbox overlaps [query] (TILE-05) — feeds Phase 25's
  /// viewport-based tile-reading pipeline. Regions whose vector/DEM package
  /// target is null (not downloaded) or whose bbox doesn't overlap [query]
  /// contribute nothing to the result.
  List<String> localTilePathsForBounds(LngLatBounds query) {
    final paths = <String>[];
    for (final region in _store.box<RegionEntity>().getAll()) {
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
      if (vectorPath != null) paths.add(vectorPath);
      if (demPath != null) paths.add(demPath);
    }
    return paths;
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

  /// Releases the lifecycle listener and cancels every remaining in-flight
  /// download (mirrors `navigation_screen.dart`'s dispose discipline).
  void dispose() {
    _lifecycleListener.dispose();
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

  /// Downloads [requestPath] to [partPath], resuming from the `.part`
  /// file's existing byte length via `resumePlanFor` (RESEARCH.md Pattern
  /// 1). Never trusts the response's `Content-Length` header for anything
  /// beyond progress reporting — completion is validated separately by
  /// [_isValidPmTiles] before the caller promotes `.part` to its final path.
  Future<void> _downloadResumable({
    required String requestPath,
    required String partPath,
    required CancelToken cancelToken,
    void Function(int received, int total)? onProgress,
  }) async {
    final partFile = File(partPath);
    final alreadyDownloaded = partFile.existsSync() ? partFile.lengthSync() : 0;
    final plan = resumePlanFor(alreadyDownloaded);

    await _api.download(
      requestPath,
      partPath,
      cancelToken: cancelToken,
      fileAccessMode: plan.mode,
      deleteOnError: plan.deleteOnError,
      options: plan.sendRange
          ? Options(headers: {'range': 'bytes=${plan.offset}-'})
          : null,
      onReceiveProgress: (received, total) => onProgress?.call(
        plan.offset + received,
        total < 0 ? -1 : plan.offset + total,
      ),
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

  /// TILE-04: cancels every active download's `CancelToken` with a
  /// deliberate `'app-backgrounded'` reason — distinct from a user-
  /// initiated `pauseRegion`'s `'paused'` reason, in case status writes
  /// ever need to distinguish the two. The `.part` file is preserved (see
  /// the `CancelToken.isCancel` branch in `startVectorDownload`/
  /// `startDemDownload`).
  void _pauseAllActiveDownloads() {
    for (final token in _activeCancelTokens.values) {
      token.cancel('app-backgrounded');
    }
  }
}
