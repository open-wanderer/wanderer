import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:objectbox/objectbox.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/map_cell.dart';
import 'package:wanderer/models/trail.dart';

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

    if (!await trailDir.exists()) {
      await trailDir.create(recursive: true);
    }

    final List<String> localPaths = await _downloadPhotos(
      trail.photos
          .map((p) => trail.getFileUrl(_api.options.baseUrl, p)!)
          .toList(),
      trailDir,
      cancelToken: cancelToken,
    );

    final cellPaths = await _downloadMapTiles(
      trail,
      trailDir,
      cancelToken: cancelToken,
      onProgress: onProgress,
    );

    final entity = TrailEntity.fromModel(trail);
    entity.photos = localPaths;
    entity.pmTiles = cellPaths;
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
    final LatLngBounds bounds = trail.bounds;

    final bbox =
        '${bounds.west},${bounds.south},${bounds.east},${bounds.north}';

    final infoList = await _fetchCellList(bbox, cancelToken: cancelToken);
    if (infoList.cells.isEmpty) return [];

    final tilesDir = Directory('${trailDir.path}/tiles');
    if (!await tilesDir.exists()) {
      await tilesDir.create(recursive: true);
    }

    var completed = 0;
    final total = infoList.cells.length;

    final downloadTasks = infoList.cells.map((cell) async {
      final key = cell.key;
      final localPath = '${tilesDir.path}/$key.pmtiles';

      if (await File(localPath).exists()) {
        onProgress?.call(++completed, total);
        return localPath;
      }

      try {
        final requestRes = await _api.get(
          cell.url,
          cancelToken: cancelToken,
        );
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
        onProgress?.call(++completed, total);
        return localPath;
      } on DioException catch (e) {
        if (CancelToken.isCancel(e)) {
          if (await File(localPath).exists()) {
            await File(localPath).delete();
          }
          rethrow;
        }
        print('Failed to download map cell $key: $e');
        return null;
      } catch (e) {
        print('Failed to download map cell $key: $e');
        return null;
      }
    }).toList();

    final results = await Future.wait(downloadTasks);
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
            if (total != -1) {
              // print("Download $fileName: ${(received / total * 100).toStringAsFixed(0)}%");
            }
          },
        );
        return savePath;
      } on DioException catch (e) {
        if (CancelToken.isCancel(e)) rethrow;
        print('Failed to download photo $url: $e');
        return null;
      } catch (e) {
        print('Failed to download photo $url: $e');
        return null;
      }
    }).toList();

    final results = await Future.wait(downloadTasks);
    return results.whereType<String>().toList();
  }
}
