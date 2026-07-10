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

    final List<String> localPaths = await _downloadPhotos(
      trail.photos
          .map((p) => trail.getFileUrl(baseUrl, p))
          .whereType<String>()
          .toList(),
      trailDir,
      cancelToken: cancelToken,
    );

    final waypoints = trail.expand?.waypointsViaTrail ?? [];
    final Map<String, List<String>> waypointLocalPhotos = {};
    for (final waypoint in waypoints) {
      if (waypoint.photos.isEmpty) continue;
      final waypointDir = Directory(
        '${trailDir.path}/waypoints/${waypoint.id}',
      );
      waypointLocalPhotos[waypoint.id] = await _downloadPhotos(
        waypoint.photos
            .map((p) => waypoint.getFileUrl(baseUrl, p))
            .whereType<String>()
            .toList(),
        waypointDir,
        cancelToken: cancelToken,
      );
    }

    final List<String> cellPaths;
    try {
      cellPaths = await _downloadMapTiles(
        trail,
        trailDir,
        cancelToken: cancelToken,
        onProgress: onProgress,
      );
    } catch (e) {
      await trailDir.delete(recursive: true);
      rethrow;
    }

    final entity = TrailEntity.fromModel(trail);
    entity.photos = localPaths;
    entity.pmTiles = cellPaths;
    for (final waypointEntity in entity.waypoints) {
      final paths = waypointLocalPhotos[waypointEntity.id];
      if (paths != null) waypointEntity.localPhotos = paths;
    }

    // Best-effort Valhalla cache write. Any failure (Valhalla outage, null GPX,
    // parse error) is silently swallowed so the tile download and entity
    // persistence are never blocked.
    try {
      final gpx = trail.expand?.gpx;
      if (gpx != null) {
        final points = gpx.allPoints;
        if (points.length >= 2) {
          // Same shape helper as the online path: ensures cache and live
          // requests send byte-identical shape payloads to Valhalla.
          final shape = buildNavShape(points);

          // Derive costing from category via shared helper. Shared with
          // launchNavigation so cache and live requests use the same
          // costing string.
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
      // Re-throw if the download was cancelled — must not write a partial entity.
      if (e is DioException && CancelToken.isCancel(e)) rethrow;
      // Best-effort: Valhalla outage must not block download.
    }

    _store.runInTransaction(TxMode.write, () {
      box.put(entity);
    });
  }

  Future<List<String>> _downloadMapTiles(
    Trail trail,
    Directory trailDir, {
    CancelToken? cancelToken,
    void Function(int done, int total)? onProgress,
  }) async {
    final LngLatBounds bounds = trail.bounds;

    final bbox =
        '${bounds.longitudeWest},${bounds.latitudeSouth},${bounds.longitudeEast},${bounds.latitudeNorth}';

    final infoList = await _fetchCellList(bbox, cancelToken: cancelToken);
    if (infoList.cells.isEmpty) return [];

    final tilesDir = Directory('${trailDir.path}/tiles');
    if (!await tilesDir.exists()) {
      await tilesDir.create(recursive: true);
    }

    final total = infoList.cells.length;

    // Tile downloads run concurrently via Future.wait. Progress is reported
    // monotonically after all tasks finish to avoid non-monotonic counter
    // updates caused by interleaving at await points.
    final downloadTasks = infoList.cells.map((cell) async {
      final key = cell.key;
      final localPath = '${tilesDir.path}/$key.pmtiles';

      if (await File(localPath).exists()) {
        return localPath;
      }

      try {
        final requestRes = await _api.get(cell.url, cancelToken: cancelToken);
        var readyCell = MapCellStatusResponse.fromJson(requestRes.data!);

        if (readyCell.status != MapCellStatus.ready) {
          readyCell = await _pollUntilReady(
            readyCell.statusUrl!,
            key,
            cancelToken,
          );
        }

        await _api.download(
          readyCell.downloadUrl!,
          localPath,
          cancelToken: cancelToken,
        );
        return localPath;
      } on DioException {
        if (await File(localPath).exists()) {
          await File(localPath).delete();
        }
        rethrow;
      }
    }).toList();

    final results = await Future.wait(downloadTasks);
    // Report final progress after all concurrent tasks complete.
    if (onProgress != null) {
      final done = results.whereType<String>().length;
      onProgress(done, total);
    }
    return results.whereType<String>().toList();
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

    while (DateTime.now().isBefore(deadline)) {
      await Future.delayed(_pollInterval);

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
          continue;
      }
    }
    throw Exception('Timed out waiting for map cell $cellKey');
  }

  Future<List<String>> _downloadPhotos(
    List<String> urls,
    Directory trailDir, {
    CancelToken? cancelToken,
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
      }
    }).toList();

    final results = await Future.wait(downloadTasks);
    return results.whereType<String>().toList();
  }
}
