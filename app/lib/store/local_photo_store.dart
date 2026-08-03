/// App-owned storage for photos attached to a trail that has not yet
/// uploaded (synced) to the server.
///
/// `image_picker` (and OS share sheets) hand back paths into an OS-purgeable
/// cache -- the OS is free to reclaim that space at any time, and an
/// unsynced trail can sit on-device for days on a multi-day hike. Without a
/// copy into app-owned storage here, a picked photo can be gone by the time
/// the sync drain (36-05/36-06) runs (D-01).
///
/// Root: `<app-docs>/unsynced`, deliberately named distinctly from
/// `library/` (downloaded trails, see `trail_download_service.dart`) so
/// [deleteUnsyncedPhotoDir] and [sweepOrphanedUnsyncedPhotos] can target it
/// without ever touching a downloaded trail's files --
/// `account_data_purge_util.dart`'s `accountScopedDirNames` doc comment
/// reasons about `library`/`regions`/`map_cache` the same way (T-36-02-03).
///
/// Every path below is built with `p.join` -- never string interpolation.
/// RESEARCH.md's Security Domain names `p.join` as the required pattern
/// here; `trail_download_service.dart`'s `'${appDir.path}/library/$trailId'`
/// interpolation style is deliberately NOT reused.
import 'dart:io';

import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/util/local/id.dart';

/// Root of all unsynced-trail photo storage: `<appDocsPath>/unsynced`.
String unsyncedPhotoRoot(String appDocsPath) {
  return p.join(appDocsPath, 'unsynced');
}

/// Directory holding [localId]'s trail-level photo copies.
///
/// [localId] is routed through [localIdDirSegment] before any path is
/// built, so a tampered or malformed id throws [ArgumentError] before any
/// filesystem call happens (T-36-02-01).
String unsyncedTrailPhotoDir(String appDocsPath, String localId) {
  return p.join(unsyncedPhotoRoot(appDocsPath), localIdDirSegment(localId));
}

/// Directory holding a single waypoint's photo copies, nested under
/// [localId]'s trail directory.
///
/// Both [localId] and [waypointLocalKey] are routed through
/// [localIdDirSegment] before any path is built (T-36-02-01).
String unsyncedWaypointPhotoDir(
  String appDocsPath,
  String localId,
  String waypointLocalKey,
) {
  return p.join(
    unsyncedTrailPhotoDir(appDocsPath, localId),
    'waypoints',
    localIdDirSegment(waypointLocalKey),
  );
}

/// Result of [reconcileLocalPhotos]: the kept photo paths, in the same
/// order as the caller's `desiredPaths`, and a count of photos that failed
/// to copy and were dropped.
class LocalPhotoCopyResult {
  const LocalPhotoCopyResult({required this.paths, required this.failedCount});

  final List<String> paths;
  final int failedCount;
}

/// Copies every entry of [desiredPaths] into [dir] (creating it recursively
/// if absent), removes any file left in [dir] that is no longer wanted, and
/// returns the kept paths in [desiredPaths]'s order.
///
/// A path already inside [dir] (tested with `p.isWithin`) is kept verbatim,
/// never re-copied. A destination filename for a new copy is derived from
/// `p.basename` of the source, joined into [dir] with `p.join` -- so a
/// crafted source basename can never climb out of [dir] (T-36-02-02) -- and
/// prefixed with an incrementing index when a file of that name already
/// exists, so two photos that happen to share an OS-picker basename never
/// collide.
///
/// **Per-photo failures never abort the batch.** A missing source, a full
/// disk, or a permission error is caught, counted in
/// [LocalPhotoCopyResult.failedCount], and the photo is dropped -- never
/// raised back to the caller, and this function never falls back to the
/// original picker path. This deliberately INVERTS `trail_download_service.dart`'s
/// abort-and-delete-everything polarity on purpose: a trail's recorded
/// track is irreplaceable, a photo is not, and disk pressure -- the most
/// likely cause of a copy failure -- is exactly when the OS cache purge
/// that motivated this whole util (D-01) is most likely to have already
/// happened (D-03).
Future<LocalPhotoCopyResult> reconcileLocalPhotos({
  required String dir,
  required List<String> desiredPaths,
}) async {
  final directory = Directory(dir);
  if (!await directory.exists()) {
    await directory.create(recursive: true);
  }

  // Filenames already claimed in `dir`, refreshed as new copies land so a
  // collision-free destination never overwrites a pre-existing file or a
  // photo copied earlier in this same call.
  final reservedNames = <String>{
    for (final entry in directory.listSync())
      if (entry is File) p.basename(entry.path),
  };

  final keptPaths = <String>[];
  var failedCount = 0;

  for (final sourcePath in desiredPaths) {
    if (p.isWithin(dir, sourcePath)) {
      keptPaths.add(sourcePath);
      continue;
    }
    try {
      final destName = _collisionFreeName(
        reservedNames,
        p.basename(sourcePath),
      );
      final destPath = p.join(dir, destName);
      await File(sourcePath).copy(destPath);
      reservedNames.add(destName);
      keptPaths.add(destPath);
    } catch (_) {
      // Never raise this failure back to the caller, never fall back to
      // the picker path -- see doc comment above (D-03).
      failedCount++;
    }
  }

  // Delete any file already inside `dir` that is not in the kept set, so a
  // re-save that removed a photo also removes its copy.
  //
  // Canonicalized on BOTH sides, not compared raw. A desired path is kept
  // verbatim when `p.isWithin(dir, sourcePath)` is true, and `isWithin`
  // normalizes -- `dir/./photo.jpg`, `dir//photo.jpg`, a trailing-separator
  // `dir` all pass -- while `listSync().path` does not. Any non-canonical
  // spelling of an in-dir path was therefore returned to the caller as
  // "kept" AND deleted from disk in the same call, leaving the entity
  // pointing at a file that no longer exists. Silently, since both loops
  // swallow their errors and nothing increments `failedCount`.
  final keptSet = keptPaths.map(p.canonicalize).toSet();
  for (final entry in directory.listSync()) {
    if (entry is! File) continue;
    if (keptSet.contains(p.canonicalize(entry.path))) continue;
    try {
      entry.deleteSync();
    } catch (_) {
      // Best-effort -- a stray file that fails to delete is harmless.
    }
  }

  return LocalPhotoCopyResult(paths: keptPaths, failedCount: failedCount);
}

/// Returns [basename] unchanged if it is not already in [reservedNames],
/// otherwise prefixes it with the lowest unused `<index>_` so two photos
/// sharing an OS-picker basename never collide (does not mutate
/// [reservedNames] -- callers add the chosen name themselves).
String _collisionFreeName(Set<String> reservedNames, String basename) {
  if (!reservedNames.contains(basename)) return basename;
  var index = 1;
  while (reservedNames.contains('${index}_$basename')) {
    index++;
  }
  return '${index}_$basename';
}

/// Deletes [localId]'s unsynced photo directory (and its `waypoints/`
/// subtree) recursively, if present. Used both on a successful drain and on
/// an unsynced-trail delete (D-02/D-14).
///
/// Best-effort: a failure to delete is swallowed, matching
/// `account_data_purge_util.dart`'s discipline for this kind of best-effort
/// on-disk cleanup. [localId] is still validated via [unsyncedTrailPhotoDir]
/// before that try/catch begins, so a malformed id is a caller bug that
/// surfaces as an [ArgumentError], not a silently swallowed no-op.
Future<void> deleteUnsyncedPhotoDir(String localId) async {
  final appDir = await getApplicationDocumentsDirectory();
  final dir = Directory(unsyncedTrailPhotoDir(appDir.path, localId));
  try {
    if (await dir.exists()) {
      await dir.delete(recursive: true);
    }
  } catch (_) {
    // Best-effort -- a failure to delete here must never crash the caller.
  }
}

/// Deletes every immediate child directory of `<app-docs>/unsynced` whose
/// basename is not in [keepLocalIds], and returns how many were deleted.
///
/// The caller supplies [keepLocalIds] from live not-yet-synced rows, so a
/// row still in the `uploading` state after a crash keeps its photos for
/// the resume (D-05), while a directory leaked by a crash between "server
/// accepted" and "local files deleted" is reclaimed (D-02).
///
/// Only the immediate children of the unsynced root are ever enumerated or
/// deleted, so this can never reach `library/`, `regions/`, `map_cache/` or
/// `objectbox/` (T-36-02-03). Best-effort throughout: a failure to list the
/// root, or to delete any one child directory, is swallowed so it can never
/// abort the sweep or throw.
Future<int> sweepOrphanedUnsyncedPhotos({
  required Set<String> keepLocalIds,
}) async {
  var deletedCount = 0;
  try {
    final appDir = await getApplicationDocumentsDirectory();
    final root = Directory(unsyncedPhotoRoot(appDir.path));
    if (!await root.exists()) return 0;

    for (final entry in root.listSync()) {
      if (entry is! Directory) continue;
      final name = p.basename(entry.path);
      if (keepLocalIds.contains(name)) continue;
      try {
        await entry.delete(recursive: true);
        deletedCount++;
      } catch (_) {
        // Best-effort -- one bad directory must never stop the sweep.
      }
    }
  } catch (_) {
    // Best-effort -- a failure to even list the root must never throw.
  }
  return deletedCount;
}
