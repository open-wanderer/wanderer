import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:maplibre/maplibre.dart';
import 'package:objectbox/objectbox.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/map_cell.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/util/gpx_util.dart';
import 'package:wanderer/util/valhalla_util.dart';

class TrailDownloadService {
  final Store _store;
  final Dio _api;

  static const _pollInterval = Duration(seconds: 3);
  static const _pollTimeout = Duration(minutes: 3);

  TrailDownloadService(this._store, this._api);

  Future<void> downloadTrail(
    Trail trail, {
    CancelToken? cancelToken,
    void Function(int done, int total)? onProgress,
  }) async {
    final box = _store.box<TrailEntity>();
    final trailId = trail.id;
    final appDir = await getApplicationDocumentsDirectory();
    final trailDir = Directory('${appDir.path}/library/$trailId');
    final baseUrl = Uri.parse(_api.options.baseUrl).origin;
    if (!await trailDir.exists()) {
      await trailDir.create(recursive: true);
    }

    final trailPhotoUrls = trail.photos
        .map((p) => trail.getFileUrl(baseUrl, p))
        .whereType<String>()
        .toList();

    // Build the per-waypoint photo jobs up front so the progress total below
    // matches exactly the number of photos `_downloadPhotos` will process
    // (after the same null-URL filtering).
    final waypoints = trail.expand?.waypointsViaTrail ?? [];
    final waypointPhotoJobs = <(String, Directory, List<String>)>[];
    for (final waypoint in waypoints) {
      final urls = waypoint.photos
          .map((p) => waypoint.getFileUrl(baseUrl, p))
          .whereType<String>()
          .toList();
      if (urls.isEmpty) continue;
      waypointPhotoJobs.add((
        waypoint.id,
        Directory('${trailDir.path}/waypoints/${waypoint.id}'),
        urls,
      ));
    }

    // Progress counts photos and tile cells together. The photo total is known
    // now; the cell total only after `_fetchCellList` returns. `report()`
    // no-ops until the cell total is known (that brief window shows the seeded
    // indeterminate "Preparing download…"), then emits a combined count so the
    // bar only reaches 100% right before the entity is persisted — never a
    // misleading "100% but tiles/photos still running" state.
    final photoTotal = trailPhotoUrls.length +
        waypointPhotoJobs.fold<int>(0, (sum, job) => sum + job.$3.length);
    var done = 0;
    int? cellTotal;
    void report() {
      if (cellTotal == null) return;
      onProgress?.call(done, cellTotal! + photoTotal);
    }

    void onUnit() {
      done++;
      report();
    }

    // Kick photos and tiles off together: starting the tile download alongside
    // the photos overlaps server-side tile generation with photo downloading.
    // Default Future.wait (eagerError: false) lets in-flight photo downloads
    // settle before cleanup runs on a tile failure, so we never delete the
    // directory while a photo is still being written.
    List<String> localPaths = [];
    var waypointLocalPhotos = <String, List<String>>{};
    (List<String>, List<String>)? tileResult;

    final futures = <Future<void>>[
      () async {
        tileResult = await _downloadMapTiles(
          trail,
          trailDir,
          cancelToken: cancelToken,
          onCellTotal: (total) {
            cellTotal = total;
            report();
          },
          onCellDone: onUnit,
        );
      }(),
      () async {
        localPaths = await _downloadPhotos(
          trailPhotoUrls,
          trailDir,
          cancelToken: cancelToken,
          onPhotoDone: onUnit,
        );
      }(),
      () async {
        waypointLocalPhotos = await _downloadWaypointPhotos(
          waypointPhotoJobs,
          cancelToken: cancelToken,
          onPhotoDone: onUnit,
        );
      }(),
    ];

    try {
      await Future.wait(futures);
    } catch (e) {
      if (await trailDir.exists()) {
        await trailDir.delete(recursive: true);
      }
      rethrow;
    }

    final (cellPaths, demCellPaths) = tileResult!;

    final entity = TrailEntity.fromModel(trail);
    entity.photos = localPaths;
    entity.pmTiles = cellPaths;
    entity.demPmTiles = demCellPaths;
    for (final waypointEntity in entity.waypoints) {
      final paths = waypointLocalPhotos[waypointEntity.id];
      if (paths != null) waypointEntity.localPhotos = paths;
    }

    // Best-effort Valhalla cache write; any failure is swallowed so tile
    // download and entity persistence are never blocked.
    try {
      final gpx = trail.expand?.gpx;
      if (gpx != null) {
        final points = gpx.allPoints;
        if (points.length >= 2) {
          // Shared with the online path so cache and live requests send
          // identical payloads to Valhalla.
          final shape = buildNavShape(points);
          final costing = costingForCategory(trail.expand?.category?.name);

          final res = await _api.post(
            '/valhalla/navigate',
            data: {'shape': shape, 'costing': costing},
            cancelToken: cancelToken,
          );

          final response = NavigateResponse.fromJson(
            res.data as Map<String, dynamic>,
          );

          // Only persist a non-empty, valid response (no broken cache).
          if (response.maneuvers.isNotEmpty && response.shape.isNotEmpty) {
            entity.navCacheJson = jsonEncode(response.toJson());
          }
        }
      }
    } catch (e) {
      // Re-throw on cancellation — must not write a partial entity.
      if (e is DioException && CancelToken.isCancel(e)) rethrow;
    }

    _store.runInTransaction(TxMode.write, () {
      box.put(entity);
    });
  }

  /// Downloads the vector `.pmtiles` cell (fatal on failure) and, best-effort,
  /// its companion DEM cell (hillshade is cosmetic, so DEM failures never
  /// fail the download). `dem paths` may be shorter than `vector paths`.
  Future<(List<String>, List<String>)> _downloadMapTiles(
    Trail trail,
    Directory trailDir, {
    CancelToken? cancelToken,
    void Function(int total)? onCellTotal,
    void Function()? onCellDone,
  }) async {
    final LngLatBounds bounds = trail.bounds;

    final bbox =
        '${bounds.longitudeWest},${bounds.latitudeSouth},${bounds.longitudeEast},${bounds.latitudeNorth}';

    final infoList = await _fetchCellList(bbox, cancelToken: cancelToken);
    if (infoList.cells.isEmpty) {
      onCellTotal?.call(0);
      return (<String>[], <String>[]);
    }

    final tilesDir = Directory('${trailDir.path}/tiles');
    if (!await tilesDir.exists()) {
      await tilesDir.create(recursive: true);
    }

    onCellTotal?.call(infoList.cells.length);

    // Downloads run concurrently; each task reports as it finishes. A plain
    // counter incremented from these async tails is race-free (single-threaded
    // event loop) and strictly monotonic.
    final downloadTasks = infoList.cells.map((cell) async {
      final key = cell.key;
      final localPath = '${tilesDir.path}/$key.pmtiles';
      final demLocalPath = '${tilesDir.path}/${key}_dem.pmtiles';

      final vectorCached = await File(localPath).exists();
      final demCached = await File(demLocalPath).exists();

      if (vectorCached && demCached) {
        onCellDone?.call();
        return (localPath, demLocalPath);
      }

      try {
        MapCellStatusResponse? readyCell;

        // A cache-hit on both vector and DEM skips the network entirely.
        if (!vectorCached || !demCached) {
          final requestRes = await _api.get(
            cell.url,
            cancelToken: cancelToken,
          );
          readyCell = MapCellStatusResponse.fromJson(requestRes.data!);

          if (readyCell.status != MapCellStatus.ready) {
            readyCell = await _pollUntilReady(
              readyCell.statusUrl!,
              key,
              cancelToken,
            );
          }
        }

        if (!vectorCached) {
          await _api.download(
            readyCell!.downloadUrl!,
            localPath,
            cancelToken: cancelToken,
          );
        }

        String? demPath = demCached ? demLocalPath : null;
        if (!demCached && readyCell?.demDownloadUrl != null) {
          try {
            await _api.download(
              readyCell!.demDownloadUrl!,
              demLocalPath,
              cancelToken: cancelToken,
            );
            demPath = demLocalPath;
          } catch (e) {
            if (e is DioException && CancelToken.isCancel(e)) rethrow;
            if (await File(demLocalPath).exists()) {
              await File(demLocalPath).delete();
            }
            demPath = null;
          }
        }

        onCellDone?.call();
        return (localPath, demPath);
      } on DioException {
        if (await File(localPath).exists()) {
          await File(localPath).delete();
        }
        rethrow;
      }
    }).toList();

    final results = await Future.wait(downloadTasks);
    return (
      results.map((r) => r.$1).toList(),
      results.map((r) => r.$2).whereType<String>().toList(),
    );
  }

  Future<MapCellInfoList> _fetchCellList(
    String bbox, {
    CancelToken? cancelToken,
  }) async {
    final res = await _api.get(
      '/map/cells',
      queryParameters: {'bbox': bbox},
      cancelToken: cancelToken,
    );
    return MapCellInfoList.fromJson(res.data!);
  }

  Future<MapCellStatusResponse> _pollUntilReady(
    String statusUrl,
    String cellKey, [
    CancelToken? cancelToken,
  ]) async {
    final deadline = DateTime.now().add(_pollTimeout);

    while (true) {
      final res = await _api.get(statusUrl, cancelToken: cancelToken);
      final data = MapCellStatusResponse.fromJson(res.data!);

      switch (data.status) {
        case MapCellStatus.ready:
          return data;
        case MapCellStatus.error:
          throw Exception(
            'Map cell $cellKey generation failed: ${data.error ?? 'unknown error'}',
          );
        case MapCellStatus.pending:
        case MapCellStatus.isNew:
          if (DateTime.now().isAfter(deadline)) {
            throw Exception('Timed out waiting for map cell $cellKey');
          }
          await Future.delayed(_pollInterval);
          continue;
      }
    }
  }

  Future<List<String>> _downloadPhotos(
    List<String> urls,
    Directory trailDir, {
    CancelToken? cancelToken,
    void Function()? onPhotoDone,
  }) async {
    final photoDir = Directory('${trailDir.path}/photos');
    if (!await photoDir.exists()) {
      await photoDir.create(recursive: true);
    }

    final downloadTasks = urls.map((url) async {
      try {
        final fileName = p.basename(Uri.parse(url).path);
        final savePath = '${photoDir.path}/$fileName';
        await _api.download(
          url,
          savePath,
          cancelToken: cancelToken,
          onReceiveProgress: (received, total) {
            if (total != -1) {}
          },
        );
        return savePath;
      } on DioException catch (e) {
        if (CancelToken.isCancel(e)) rethrow;
        debugPrint('Failed to download photo $url: $e');
        return null;
      } catch (e) {
        debugPrint('Failed to download photo $url: $e');
        return null;
      } finally {
        // Count as done on success or best-effort failure, so the shared
        // progress counter can't stall on a single dropped photo.
        onPhotoDone?.call();
      }
    }).toList();

    final results = await Future.wait(downloadTasks);
    return results.whereType<String>().toList();
  }

  /// Downloads all waypoint photos concurrently (across waypoints, not just
  /// within one), keyed by waypoint id so the caller can attach local paths
  /// back onto the corresponding `TrailEntity.waypoints` entry.
  Future<Map<String, List<String>>> _downloadWaypointPhotos(
    List<(String, Directory, List<String>)> jobs, {
    CancelToken? cancelToken,
    void Function()? onPhotoDone,
  }) async {
    final entries = await Future.wait(
      jobs.map((job) async {
        final (waypointId, waypointDir, urls) = job;
        final paths = await _downloadPhotos(
          urls,
          waypointDir,
          cancelToken: cancelToken,
          onPhotoDone: onPhotoDone,
        );
        return MapEntry(waypointId, paths);
      }),
    );
    return Map.fromEntries(entries);
  }
}
