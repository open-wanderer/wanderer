import 'package:disk_space_2/disk_space_2.dart';

/// Disk free-space query + fail-closed safety-margin decision for the
/// region download engine (TILE-03).
///
/// `disk_space_2` is the only genuinely new third-party dependency this
/// milestone introduces (v1.6 Phase 23), added behind a blocking
/// `checkpoint:human-verify` legitimacy gate because `slopcheck` cannot scan
/// the pub.dev ecosystem. The installed package's public API
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

/// Queries free disk space, in bytes, for [forPath] if given (falls back to
/// the device-wide free space when omitted or when the path-specific query
/// fails).
///
/// The installed `disk_space_2` API reports free space in mebibytes
/// (2^20 bytes) as a `double`; this wrapper converts to bytes.
///
/// Never throws — any plugin exception (including the path-specific query's
/// documented "path does not exist" exception) or a `null` result is
/// swallowed and reported as `null`, so callers can fail closed via
/// [hasEnoughSpace] rather than crash mid-download.
Future<int?> freeDiskSpaceBytes([String? forPath]) async {
  try {
    final double? freeMebibytes = forPath != null
        ? await DiskSpace.getFreeDiskSpaceForPath(forPath)
        : await DiskSpace.getFreeDiskSpace;
    if (freeMebibytes == null) return null;
    return (freeMebibytes * 1024 * 1024).round();
  } catch (_) {
    // Fail closed at the query level too — a thrown plugin exception must
    // never propagate to a download-loop caller.
    return null;
  }
}

/// Pure decision: is there enough free space to safely write a file of
/// [declaredSizeBytes]?
///
/// Fails closed (`false`) when [freeBytes] is `null` (unknown free space is
/// treated as "not enough" — TILE-03's fail-closed requirement) — refusing a
/// download is always safer than risking a mid-write `ENOSPC`.
///
/// [safetyMultiplier] is a tunable heuristic (RESEARCH Assumption A3), not a
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
