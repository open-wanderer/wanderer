/// App-owned storage for photos attached to a trail that has not yet
/// uploaded (synced) to the server.
///
/// `image_picker` (and OS share sheets) hand back paths into an OS-purgeable
/// cache -- the OS is free to reclaim that space at any time, and an
/// unsynced trail can sit on-device for days on a multi-day hike. Without a
/// copy into app-owned storage here, a picked photo can be gone by the time
/// the sync drain runs.
///
/// Root: `<app-docs>/unsynced/<accountId>/<localId>/`, deliberately named
/// distinctly from `library/` (downloaded trails, see
/// `trail_download_service.dart`) so [deleteUnsyncedPhotoDir] and
/// [sweepOrphanedUnsyncedPhotos] can target it without ever touching a
/// downloaded trail's files -- `account_data_purge.dart`'s
/// `accountScopedDirNames` doc comment reasons about
/// `library`/`regions`/`map_cache` the same way.
///
/// The `<accountId>` segment exists because a destructive action must be
/// scoped by the identity it actually destroys. An earlier layout had no
/// account component at all, so `<appDocs>/unsynced/<localId>/` was
/// literally the same directory for every account, and one account's Delete
/// could recursively remove another account's un-uploaded photos. There is
/// deliberately NO migration and no first-launch cleanup for that old
/// one-level layout -- any leftover `unsynced/<localId>/` directory is
/// simply never read again.
///
/// Every path below is built with `p.join` -- never string interpolation.
/// `trail_download_service.dart`'s `'${appDir.path}/library/$trailId'`
/// interpolation style is deliberately NOT reused.
import 'dart:io';

import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/util/local/id.dart';

/// Root of all unsynced-trail photo storage: `<appDocsPath>/unsynced`.
String unsyncedPhotoRoot(String appDocsPath) {
  return p.join(appDocsPath, 'unsynced');
}

/// Directory holding [accountId]'s unsynced photo storage.
///
/// [accountId] is a PocketBase-issued [UserEntity.id], routed through
/// [recordIdDirSegment] (not [localIdDirSegment], which is reserved for
/// device-minted ids) before any path is built, so a tampered or malformed
/// account id throws [ArgumentError] before any filesystem call happens --
/// the same discipline extended one level up.
String unsyncedAccountPhotoDir(String appDocsPath, String accountId) {
  return p.join(unsyncedPhotoRoot(appDocsPath), recordIdDirSegment(accountId));
}

/// Directory holding [localId]'s trail-level photo copies, nested under
/// [accountId]'s account directory.
///
/// [localId] is routed through [localIdDirSegment] before any path is
/// built, so a tampered or malformed id throws [ArgumentError] before any
/// filesystem call happens. Parameter order is always
/// `appDocsPath, accountId, localId` across every builder and every caller
/// in this file, so a two-string call site cannot silently transpose them.
String unsyncedTrailPhotoDir(
  String appDocsPath,
  String accountId,
  String localId,
) {
  return p.join(
    unsyncedAccountPhotoDir(appDocsPath, accountId),
    localIdDirSegment(localId),
  );
}

/// Directory holding a single waypoint's photo copies, nested under
/// [accountId]'s and [localId]'s trail directory.
///
/// [accountId], [localId] and [waypointLocalKey] are all routed through
/// their respective validators before any path is built.
String unsyncedWaypointPhotoDir(
  String appDocsPath,
  String accountId,
  String localId,
  String waypointLocalKey,
) {
  return p.join(
    unsyncedTrailPhotoDir(appDocsPath, accountId, localId),
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
/// crafted source basename can never climb out of [dir] -- and
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
/// that motivated this whole util is most likely to have already
/// happened.
///
/// The directory operations obey the same contract: a failure
/// to create [dir] is reported as `failedCount == desiredPaths.length` with
/// no kept paths, and either directory listing degrades to empty. This
/// function raises nothing at all -- the only way its caller learns of a
/// failure is [LocalPhotoCopyResult.failedCount].
Future<LocalPhotoCopyResult> reconcileLocalPhotos({
  required String dir,
  required List<String> desiredPaths,
}) async {
  final directory = Directory(dir);

  // The directory calls are inside the contract, not outside
  // it. `create`/`listSync` used to sit outside any try, so a permission or
  // I/O error on one of them threw straight past `_copyPhotosForLocalSave`
  // into `_onSave`'s generic catch -- the hiker saw `error_saving_trail`
  // (the whole save failed) instead of `photo_copy_failed_toast(n)`, which
  // is precisely the failure mode the `failedCount` machinery exists to
  // report. "Per-photo failures never abort the batch" is only true if the
  // batch's own setup cannot abort it either.
  try {
    if (!await directory.exists()) {
      await directory.create(recursive: true);
    }
  } catch (_) {
    // No destination directory means no photo can be copied. Count every
    // one of them as failed and return no kept paths -- the honest
    // `failedCount` the caller's toast is built on. Nothing to reconcile
    // away either: a directory that does not exist holds no stale files.
    return LocalPhotoCopyResult(
      paths: const [],
      failedCount: desiredPaths.length,
    );
  }

  // Filenames already claimed in `dir`, refreshed as new copies land so a
  // collision-free destination never overwrites a pre-existing file or a
  // photo copied earlier in this same call.
  final reservedNames = <String>{
    for (final entry in _listSyncBestEffort(directory))
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
      // the picker path -- see doc comment above.
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
  for (final entry in _listSyncBestEffort(directory)) {
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

/// [Directory.listSync], degrading to an empty listing instead of throwing
///
/// Both of [reconcileLocalPhotos]'s listings are advisory, and both must
/// obey its "never raise back to the caller" contract:
/// - the first builds `reservedNames`, so an empty listing only costs
///   collision detection against pre-existing files -- and the copy that
///   would collide is itself inside a try that counts the failure;
/// - the second drives the stale-file cleanup, so an empty listing only
///   leaves a stale file on disk, which the next successful save reconciles
///   away.
///
/// Neither can be allowed to abort the save. See [reconcileLocalPhotos]'s
/// doc comment for why this file inverts `trail_download_service.dart`'s
/// abort-everything polarity.
List<FileSystemEntity> _listSyncBestEffort(Directory directory) {
  try {
    return directory.listSync();
  } catch (_) {
    return const [];
  }
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

/// Returns every entry of [pickedPaths] that is NOT already inside
/// [unsyncedDir], preserving input order.
///
/// A picked path that lives under `unsynced/<localId>/` (tested with
/// `p.isWithin`, so a non-canonical spelling like `<unsyncedDir>/./a.jpg`
/// still counts) is an app-owned copy the drain's step 2 has already sent as
/// part of `PUT /trail/form` -- re-sending it doubles the server-side photo
/// set, because `form_data_util.dart` emits new photos under the append-only
/// `photos+` key, not a replace. A path anywhere else is a fresh
/// `image_picker` pick the server has never seen.
///
/// Diffing on LOCATION rather than on filename is required, not a style
/// choice: PocketBase renames every uploaded file, so `trail.photos`'
/// server-side filenames can never equal a local basename. A basename diff
/// against `trail.photos` (the review's first suggested fix) would exclude
/// nothing, and every save in the `alreadyUploaded` window would keep
/// doubling the server-side photo set.
///
/// Pure -- no I/O, no filesystem check that [pickedPaths] actually exist.
List<String> photosNotYetOnServer({
  required String unsyncedDir,
  required List<String> pickedPaths,
}) {
  return pickedPaths.where((path) => !p.isWithin(unsyncedDir, path)).toList();
}

/// Deletes [accountId]'s [localId] unsynced photo directory (and its
/// `waypoints/` subtree) recursively, if present. Used both on a successful
/// drain and on an unsynced-trail delete.
///
/// Best-effort: a failure to delete is swallowed, matching
/// `account_data_purge.dart`'s discipline for this kind of best-effort
/// on-disk cleanup. [accountId] and [localId] are still validated via
/// [unsyncedTrailPhotoDir] before that try/catch begins, so a malformed id
/// is a caller bug that surfaces as an [ArgumentError], not a silently
/// swallowed no-op.
///
/// An earlier layout resolved `<appDocs>/unsynced/<localId>/` with no
/// account component at all -- so when account B's Delete button routed
/// here carrying a `localId` still owned by account A (`savedByUserIds`
/// scoping let a shared row hand B a `localId`/`failed` `syncState` that
/// were actually A's), this function recursively deleted account A's
/// un-uploaded photos and reported success. The account segment makes that
/// directory structurally unreachable across accounts.
Future<void> deleteUnsyncedPhotoDir(String accountId, String localId) async {
  final appDir = await getApplicationDocumentsDirectory();
  final dir = Directory(unsyncedTrailPhotoDir(appDir.path, accountId, localId));
  try {
    if (await dir.exists()) {
      await dir.delete(recursive: true);
    }
  } catch (_) {
    // Best-effort -- a failure to delete here must never crash the caller.
  }
}

/// Deletes every second-level (account/localId) directory under
/// `<app-docs>/unsynced` whose basename is a local id absent from
/// [keepLocalIds], and returns how many were deleted.
///
/// The caller supplies [keepLocalIds] from live not-yet-synced rows across
/// ALL accounts (`local_trail_store.dart`'s `unsyncedLocalIds` is
/// deliberately not account-scoped, so a signed-out account's pending
/// photos are never swept), so a row still in the `uploading` state after a
/// crash keeps its photos for the resume, while a directory leaked
/// by a crash between "server accepted" and "local files deleted" is
/// reclaimed.
///
/// Two levels are enumerated, never more and never fewer: the immediate
/// children of the unsynced root are always treated as ACCOUNT directories
/// and are never deletion CANDIDATES, and only THEIR immediate children --
/// where `isLocalId(basename)` is true -- are candidates for deletion. An
/// account directory left EMPTY once its candidates are gone is reclaimed
/// (non-recursively, so a file that landed in the meantime is never
/// destroyed) and is not counted in the return value -- that is a
/// reclamation of an empty inode, not a widening of what this sweep
/// considers deletable. The `isLocalId` term is load-bearing: it is what
/// keeps this sweep off a `waypoints/` directory sitting at the second
/// level, and off the contents of any pre-existing one-level
/// `unsynced/<localId>/` directory. Such a legacy directory is deliberately
/// left alone forever, not "helpfully" reclaimed by widening this sweep to
/// look at it.
///
/// This can never reach `library/`, `regions/`, `map_cache/` or
/// `objectbox/`. Best-effort throughout: a failure to list the
/// root, an account directory, or to delete any one local-id directory, is
/// swallowed so it can never abort the sweep or throw.
Future<int> sweepOrphanedUnsyncedPhotos({
  required Set<String> keepLocalIds,
}) async {
  var deletedCount = 0;
  try {
    final appDir = await getApplicationDocumentsDirectory();
    final root = Directory(unsyncedPhotoRoot(appDir.path));
    if (!await root.exists()) return 0;

    for (final accountEntry in root.listSync()) {
      if (accountEntry is! Directory) continue;
      try {
        for (final entry in accountEntry.listSync()) {
          if (entry is! Directory) continue;
          final name = p.basename(entry.path);
          if (!isLocalId(name)) continue;
          if (keepLocalIds.contains(name)) continue;
          try {
            await entry.delete(recursive: true);
            deletedCount++;
          } catch (_) {
            // Best-effort -- one bad directory must never stop the sweep.
          }
        }

        // An account directory is never a deletion CANDIDATE
        // (the two-level rule above is what keeps this sweep off the legacy
        // one-level layout), but an EMPTY one has nothing left to protect.
        // Without this, every account that ever saved an unsynced photo
        // leaves a permanent inode behind -- including accounts that have
        // been signed out and whose `UserEntity` row is long gone
        // (`current_account.dart` enforces at most one `UserEntity`, so a
        // departed account has no representation anywhere to sweep by).
        // `deleteSync()` without `recursive`, deliberately: it deletes an
        // empty directory and throws on a non-empty one, so a file that
        // landed between the listing and this call is never destroyed.
        if (accountEntry.listSync().isEmpty) {
          accountEntry.deleteSync();
        }
      } catch (_) {
        // Best-effort -- one unreadable account directory must never abort
        // the sweep of the others.
      }
    }
  } catch (_) {
    // Best-effort -- a failure to even list the root must never throw.
  }
  return deletedCount;
}
