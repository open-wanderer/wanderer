import 'dart:async';
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

  // Progress is tracked in fractional "points" per unit (photo or cell) so a
  // download can report smooth incremental movement instead of jumping in
  // whole-unit steps. Reserved purely for the *downloading* phase — the
  // (opaque, server-side) tile-generation wait is reported separately via
  // `onGeneratingChanged`, since the server exposes no generation percentage,
  // only new/pending/ready/error.
  static const _pointsPerUnit = 1000;

  // When a download's response has no `Content-Length` header, Dio can't
  // report real byte progress. In that case we fake a plausible climb (capped
  // below 100%) over this assumed duration so the bar keeps moving instead of
  // freezing, then jump to 100% the instant the download actually completes.
  static const _fakeDownloadDuration = Duration(seconds: 5);

  TrailDownloadService(this._store, this._api);

  Future<void> downloadTrail(
    Trail trail, {
    CancelToken? cancelToken,
    void Function(bool isGenerating)? onGeneratingChanged,
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

    // Download progress counts photos and tile-file downloads together, in
    // fractional points per unit, so the bar animates smoothly rather than in
    // big whole-unit jumps. It's suppressed (via `isGenerating`) for the
    // entire time any cell is still being generated on the server — that
    // phase has no measurable progress, so it's reported as a separate
    // indeterminate signal instead of a fake percentage. Once generation
    // finishes (or was never needed), accumulated progress is flushed and the
    // bar takes over for the remaining, genuinely trackable download work.
    final photoTotal = trailPhotoUrls.length +
        waypointPhotoJobs.fold<int>(0, (sum, job) => sum + job.$3.length);
    var currentPoints = 0;
    int? totalPoints;
    var isGenerating = false;
    void report() {
      if (totalPoints == null || isGenerating) return;
      onProgress?.call(currentPoints, totalPoints!);
    }

    void handleGeneratingChanged(bool generating) {
      isGenerating = generating;
      if (!generating) report();
      onGeneratingChanged?.call(generating);
    }

    void onPhotoPointsDelta(int delta) {
      currentPoints += delta;
      report();
    }

    void onCellPointsDelta(int delta) {
      currentPoints += delta;
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
          onCellTotal: (cellCount) {
            totalPoints = (cellCount + photoTotal) * _pointsPerUnit;
            report();
          },
          onGeneratingChanged: handleGeneratingChanged,
          onCellPointsDelta: onCellPointsDelta,
        );
      }(),
      () async {
        localPaths = await _downloadPhotos(
          trailPhotoUrls,
          trailDir,
          cancelToken: cancelToken,
          onPhotoPointsDelta: onPhotoPointsDelta,
        );
      }(),
      () async {
        waypointLocalPhotos = await _downloadWaypointPhotos(
          waypointPhotoJobs,
          cancelToken: cancelToken,
          onPhotoPointsDelta: onPhotoPointsDelta,
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
    void Function(bool isGenerating)? onGeneratingChanged,
    void Function(int delta)? onCellPointsDelta,
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

    // Aggregate "is any cell still waiting on server-side generation" across
    // all cells running concurrently below, so the caller can show a single
    // indeterminate "Generating map tiles" signal for as long as at least one
    // cell isn't ready yet, and switch to real download progress once none
    // are (including the common case where nothing needed generating at all).
    var waitingForGeneration = 0;
    void enterWaiting() {
      waitingForGeneration++;
      onGeneratingChanged?.call(true);
    }

    void exitWaiting() {
      waitingForGeneration--;
      if (waitingForGeneration == 0) onGeneratingChanged?.call(false);
    }

    // Downloads run concurrently; each task reports its own cumulative points
    // as they increase. `reported` tracks this cell's own last-reported value
    // so only the delta (never negative) is forwarded — safe to sum across
    // concurrently running cells since each cell only ever reports its own
    // monotonic progress.
    final downloadTasks = infoList.cells.map((cell) async {
      final key = cell.key;
      final localPath = '${tilesDir.path}/$key.pmtiles';
      final demLocalPath = '${tilesDir.path}/${key}_dem.pmtiles';

      var reported = 0;
      void reportPoints(int points) {
        final clamped = points.clamp(0, _pointsPerUnit);
        if (clamped <= reported) return;
        onCellPointsDelta?.call(clamped - reported);
        reported = clamped;
      }

      final vectorCached = await File(localPath).exists();
      final demCached = await File(demLocalPath).exists();

      if (vectorCached && demCached) {
        reportPoints(_pointsPerUnit);
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
            enterWaiting();
            try {
              readyCell = await _pollUntilReady(
                readyCell.statusUrl!,
                key,
                cancelToken,
              );
            } finally {
              exitWaiting();
            }
          }
        }

        // Ready (whether confirmed instantly or via poll): only local file
        // transfer remains, which is genuinely trackable — split the cell's
        // full point budget between the vector and (optional) DEM download.
        final wantsDem = !demCached && readyCell?.demDownloadUrl != null;
        final vectorSlice = wantsDem
            ? (_pointsPerUnit * 0.8).round()
            : _pointsPerUnit;
        final demSlice = _pointsPerUnit - vectorSlice;

        if (!vectorCached) {
          await _downloadTracked(
            readyCell!.downloadUrl!,
            localPath,
            cancelToken: cancelToken,
            onFraction: (f) => reportPoints((f * vectorSlice).round()),
          );
        }
        // Full credit for the vector slice regardless of path taken (already
        // cached, or downloaded without a Content-Length header to track).
        reportPoints(vectorSlice);

        String? demPath = demCached ? demLocalPath : null;
        if (wantsDem) {
          try {
            await _downloadTracked(
              readyCell!.demDownloadUrl!,
              demLocalPath,
              cancelToken: cancelToken,
              onFraction: (f) =>
                  reportPoints(vectorSlice + (f * demSlice).round()),
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

        reportPoints(_pointsPerUnit);

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
    String cellKey,
    CancelToken? cancelToken,
  ) async {
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

  /// Wraps a Dio `.download` call so progress is reported continuously even
  /// when the response has no `Content-Length` header: real byte progress is
  /// used when available, otherwise a timer fakes a plausible climb (capped
  /// below 100%, see `_fakeDownloadDuration`) so the bar keeps moving. Either
  /// way `onFraction(1.0)` fires exactly once, right when the download
  /// actually completes.
  Future<void> _downloadTracked(
    String url,
    String savePath, {
    required CancelToken? cancelToken,
    required void Function(double fraction) onFraction,
  }) async {
    var sawRealProgress = false;
    final start = DateTime.now();
    final fakeProgressTimer = Timer.periodic(
      const Duration(milliseconds: 250),
      (_) {
        if (sawRealProgress) return;
        final elapsed = DateTime.now().difference(start).inMilliseconds;
        final fraction = elapsed / _fakeDownloadDuration.inMilliseconds;
        onFraction(fraction.clamp(0.0, 0.9));
      },
    );

    try {
      await _api.download(
        url,
        savePath,
        cancelToken: cancelToken,
        onReceiveProgress: (received, total) {
          if (total <= 0) return;
          sawRealProgress = true;
          onFraction((received / total).clamp(0.0, 1.0));
        },
      );
      onFraction(1.0);
    } finally {
      fakeProgressTimer.cancel();
    }
  }

  Future<List<String>> _downloadPhotos(
    List<String> urls,
    Directory trailDir, {
    CancelToken? cancelToken,
    void Function(int delta)? onPhotoPointsDelta,
  }) async {
    final photoDir = Directory('${trailDir.path}/photos');
    if (!await photoDir.exists()) {
      await photoDir.create(recursive: true);
    }

    final downloadTasks = urls.map((url) async {
      var reported = 0;
      void reportPoints(int points) {
        final clamped = points.clamp(0, _pointsPerUnit);
        if (clamped <= reported) return;
        onPhotoPointsDelta?.call(clamped - reported);
        reported = clamped;
      }

      try {
        final fileName = p.basename(Uri.parse(url).path);
        final savePath = '${photoDir.path}/$fileName';
        await _downloadTracked(
          url,
          savePath,
          cancelToken: cancelToken,
          onFraction: (f) => reportPoints((f * _pointsPerUnit).round()),
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
        // Count as fully done on success or best-effort failure, so the
        // shared progress counter can't stall on a single dropped photo.
        reportPoints(_pointsPerUnit);
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
    void Function(int delta)? onPhotoPointsDelta,
  }) async {
    final entries = await Future.wait(
      jobs.map((job) async {
        final (waypointId, waypointDir, urls) = job;
        final paths = await _downloadPhotos(
          urls,
          waypointDir,
          cancelToken: cancelToken,
          onPhotoPointsDelta: onPhotoPointsDelta,
        );
        return MapEntry(waypointId, paths);
      }),
    );
    return Map.fromEntries(entries);
  }
}
