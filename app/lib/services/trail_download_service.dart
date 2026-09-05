import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/util/gpx/gpx.dart';
import 'package:wanderer/util/local/library_membership.dart';
import 'package:wanderer/util/local/id.dart';
import 'package:wanderer/util/route/valhalla.dart';

class TrailDownloadService {
  final Store _store;
  final Dio _api;

  // Progress is tracked in fractional "points" per unit (photo) so a
  // download can report smooth incremental movement instead of jumping in
  // whole-unit steps.
  static const _pointsPerUnit = 1000;

  // When a download's response has no `Content-Length` header, Dio can't
  // report real byte progress. In that case we fake a plausible climb (capped
  // below 100%) over this assumed duration so the bar keeps moving instead of
  // freezing, then jump to 100% the instant the download actually completes.
  static const _fakeDownloadDuration = Duration(seconds: 5);

  TrailDownloadService(this._store, this._api);

  /// Downloads [trail] for offline use.
  ///
  /// [subcategories] is passed in by the caller because this service holds no
  /// `Ref` — it feeds [costingForTrail] so the cached Valhalla payload uses
  /// the same settings-driven costing as the online path.
  ///
  /// [savedByUserId] is the id of the account downloading the trail, passed in
  /// for the same reason. It is added to `TrailEntity.savedByUserIds`, which is
  /// what scopes the offline library per account. A null id (no signed-in user)
  /// adds no membership, so the trail stays invisible to every library rather
  /// than leaking into all of them.
  Future<void> downloadTrail(
    Trail trail, {
    CancelToken? cancelToken,
    void Function(int done, int total)? onProgress,
    List<Subcategory> subcategories = const [],
    String? savedByUserId,
  }) async {
    final box = _store.box<TrailEntity>();
    final trailId = trail.id;
    final appDir = await getApplicationDocumentsDirectory();
    // p.join + a whitelisted segment, not interpolation. `trailId` and the
    // waypoint ids below arrive over the network; a federated or compromised
    // instance returning an id containing `..` wrote outside `library/`.
    // `local_photo_store.dart` names this file's old interpolation style
    // as the one it refuses to reuse -- this brings it onto that standard.
    final trailDir = Directory(
      p.join(appDir.path, 'library', recordIdDirSegment(trailId)),
    );
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
        Directory(
          p.join(trailDir.path, 'waypoints', recordIdDirSegment(waypoint.id)),
        ),
        urls,
      ));
    }

    // Download progress counts photos in fractional points per unit, so the
    // bar animates smoothly rather than in big whole-unit jumps.
    final photoTotal =
        trailPhotoUrls.length +
        waypointPhotoJobs.fold<int>(0, (sum, job) => sum + job.$3.length);
    var currentPoints = 0;
    // Initialized up front (not derived from a tile-generation callback,
    // since there's no tile download anymore) so `report()` can fire from
    // the very first photo progress delta.
    final totalPoints = photoTotal * _pointsPerUnit;
    void report() {
      onProgress?.call(currentPoints, totalPoints);
    }

    void onPhotoPointsDelta(int delta) {
      currentPoints += delta;
      report();
    }

    // Default Future.wait (eagerError: false) lets in-flight photo downloads
    // settle before cleanup runs on a failure, so we never delete the
    // directory while a photo is still being written.
    List<String> localPaths = [];
    var waypointLocalPhotos = <String, List<String>>{};

    final futures = <Future<void>>[
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

    final entity = TrailEntity.fromModel(trail, store: _store);
    entity.photos = localPaths;
    for (final waypointEntity in entity.waypoints) {
      final paths = waypointLocalPhotos[waypointEntity.id];
      if (paths != null) waypointEntity.localPhotos = paths;
    }

    // Best-effort Valhalla cache write; any failure is swallowed so photo
    // download and entity persistence are never blocked.
    try {
      final gpx = trail.expand?.gpx;
      if (gpx != null) {
        final points = gpx.allPoints;
        if (points.length >= 2) {
          // Shared with the online path so cache and live requests send
          // identical payloads to Valhalla.
          final shape = buildNavShape(points);
          final costing = costingForTrail(trail, subcategories: subcategories);

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
      // Carry the existing library membership across a re-download. `id` is
      // `@Unique(onConflict: replace)` and `entity` is a FRESH row built by
      // `fromModel`, so putting it blind would wipe `savedByUserIds` -- and
      // with it every other account's claim on this trail. Re-read inside the
      // transaction rather than trusting an entry-time snapshot.
      //
      // The same blind-put risk applies to a trail's LOCAL bookkeeping
      // (owner/localId/syncState/syncAttempts/syncNextAttemptAt/localPhotos):
      // a hiker who captures a trail here, lets it upload, and later
      // downloads it would otherwise lose `owner` -- silently vanishing from
      // the offline own-trails list with no error anywhere -- or, if
      // caught mid-drain, lose the resume state a failed upload needs to
      // retry correctly. Carry all six forward from the existing row for the
      // same reason `savedByUserIds` is carried forward above.
      final query = box.query(TrailEntity_.id.equals(trailId)).build();
      final existing = query.findFirst();
      query.close();

      entity.savedByUserIds = libraryMembersAfterSave(
        existing?.savedByUserIds ?? const [],
        savedByUserId,
      );
      entity.owner = existing?.owner;
      entity.localId = existing?.localId;
      entity.syncState = existing?.syncState ?? entity.syncState;
      entity.syncAttempts = existing?.syncAttempts ?? 0;
      entity.syncNextAttemptAt = existing?.syncNextAttemptAt;
      entity.localPhotos = existing?.localPhotos ?? const [];

      box.put(entity);
    });
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
    final photoDir = Directory(p.join(trailDir.path, 'photos'));
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
        // `p.basename` is not sufficient on its own: `Uri.path` percent-
        // decodes, so a crafted URL delivers `..` here, and `p.basename('..')`
        // is `'..'`. Validated and p.join-ed, so a hostile filename can only
        // ever fail this one photo (the general catch below) rather than
        // writing outside `photos/`.
        final fileName = fileNameSegment(p.basename(Uri.parse(url).path));
        final savePath = p.join(photoDir.path, fileName);
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
