import 'package:disk_space_2/disk_space_2.dart';
import 'package:meta/meta.dart';

/// Disk free-space query + fail-closed safety-margin decision for the
/// region download engine.
///
/// `disk_space_2` is the only genuinely new third-party dependency this
/// region work introduces, and it was vetted by hand before adoption. The
/// installed package's public API
/// (`~/.pub-cache/hosted/pub.dev/disk_space_2-1.0.12/lib/disk_space_2.dart`)
/// exposes exactly three read-only static free/total-space queries backed by
/// `StatFs` on Android and `NSFileManager`/`URLResourceKey` on iOS — no
/// network, file-write, reflection, or shell-out — confirmed by direct
/// inspection of both native plugin sources
/// (`android/.../DiskSpace_2Plugin.kt`, `ios/.../DiskSpace_2Plugin.swift`)
/// during the Task 1 legitimacy review.
///
/// This file quarantines that plugin behind [freeDiskSpaceBytes], which
/// never throws to callers, and exposes the actual pass/fail decision as the
/// pure, unit-tested [hasEnoughSpace].

/// A path-specific free-space query: given an absolute path, returns free
/// mebibytes on that path's volume (or throws — e.g. the path doesn't exist
/// yet).
typedef PathSpaceQuery = Future<double?> Function(String path);

/// A device-wide free-space query: returns free mebibytes for the device's
/// primary storage volume, with no path argument (or throws).
typedef DeviceSpaceQuery = Future<double?> Function();

/// Injectable orchestrator behind [freeDiskSpaceBytes]: resolves free disk
/// space, in bytes, using [pathQuery] when [forPath] is given, falling back
/// to [deviceQuery] when the path-specific query throws (or when [forPath]
/// is omitted, in which case [deviceQuery] is used directly).
///
/// `regionStorageDir(root, id)` — the only real caller argument for
/// [forPath] — is always a subdirectory of the app documents root, which is
/// always on the same logical volume as whatever [deviceQuery] targets. So
/// when the path doesn't exist yet (a region's first-ever download, before
/// its storage directory is created) the device-wide number is a valid
/// substitute for the not-yet-existing path's free space.
///
/// When BOTH queries throw (or [forPath] is `null` and [deviceQuery]
/// throws), returns `null` — there is no further fallback, and callers must
/// fail closed via [hasEnoughSpace]. This preserves the genuine
/// low-disk protection: a real "disk is full" condition still can't be
/// worked around by this fallback, only the "path doesn't exist yet"
/// condition can.
///
/// Marked `@visibleForTesting` so tests can inject fake `pathQuery`/
/// `deviceQuery` closures and exercise the fallback ordering deterministically,
/// with no platform channel involved.
@visibleForTesting
Future<int?> resolveFreeDiskSpaceBytes({
  required String? forPath,
  required PathSpaceQuery pathQuery,
  required DeviceSpaceQuery deviceQuery,
}) async {
  double? freeMebibytes;
  try {
    freeMebibytes = forPath != null
        ? await pathQuery(forPath)
        : await deviceQuery();
  } catch (e) {
    if (forPath == null) {
      // No path was queried in the first place, so there's no further
      // fallback available.
      return null;
    }
    try {
      freeMebibytes = await deviceQuery();
    } catch (e) {
      // Both the path-specific and device-wide queries failed — fail
      // closed, the low-disk protection is preserved.
      return null;
    }
  }
  if (freeMebibytes == null) return null;
  return (freeMebibytes * 1024 * 1024).round();
}

/// Queries free disk space, in bytes, for [forPath] if given (falls back to
/// the device-wide free space when omitted or when the path-specific query
/// fails).
///
/// The installed `disk_space_2` API reports free space in mebibytes
/// (2^20 bytes) as a `double`; this wrapper converts to bytes.
///
/// Never throws — any plugin exception (including the path-specific query's
/// documented "path does not exist" exception, which is retried against the
/// device-wide query before giving up) or a `null` result is swallowed and
/// reported as `null`, so callers can fail closed via [hasEnoughSpace]
/// rather than crash mid-download.
Future<int?> freeDiskSpaceBytes([String? forPath]) {
  return resolveFreeDiskSpaceBytes(
    forPath: forPath,
    pathQuery: DiskSpace.getFreeDiskSpaceForPath,
    deviceQuery: () => DiskSpace.getFreeDiskSpace,
  );
}

/// Pure decision: is there enough free space to safely write a file of
/// [declaredSizeBytes]?
///
/// Fails closed (`false`) when [freeBytes] is `null` (unknown free space is
/// treated as "not enough" — the fail-closed requirement) — refusing a
/// download is always safer than risking a mid-write `ENOSPC`.
///
/// [safetyMultiplier] is a tunable heuristic, not a
/// fixed guarantee: it accounts for the coexisting `.part` file during a
/// resumed download (twice the declared size, momentarily) plus a margin for
/// the OS's own reserved space. Defaults to `1.75`.
bool hasEnoughSpace({
  required int? freeBytes,
  required int declaredSizeBytes,
  double safetyMultiplier = 1.75,
}) {
  if (freeBytes == null) return false;
  return freeBytes > declaredSizeBytes * safetyMultiplier;
}
