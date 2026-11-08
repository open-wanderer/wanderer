import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';

/// Deterministic on-disk avatar cache — no new package, mirrors the
/// dio-download-to-disk shape already used by
/// `trail_download_service.dart`'s `_downloadPhotos`.
///
/// Composes the cached filename from [userId] and the basename of
/// [remoteFilename]. `p.basename` is the sanitisation step and is mandatory
/// (T-i4k-01): [remoteFilename] originates from the server, and without
/// stripping any directory component a crafted value like `../../evil.png`
/// or `/etc/passwd` could traverse out of the avatars directory.
String cachedAvatarFileName(String userId, String remoteFilename) {
  final base = p.basename(Uri.parse(remoteFilename).path);
  return '${userId}_$base';
}

/// Resolves (and creates if absent) `<appDocDir>/avatars`.
Future<Directory> avatarCacheDir() async {
  final appDocDir = await getApplicationDocumentsDirectory();
  final dir = Directory(p.join(appDocDir.path, 'avatars'));
  if (!await dir.exists()) {
    await dir.create(recursive: true);
  }
  return dir;
}

/// Returns the cached avatar `File` for [userId] / [remoteFilename] only
/// when it exists on disk; returns `null` (never throws) when
/// [remoteFilename] is null/empty or the file hasn't been cached yet.
Future<File?> cachedAvatarFile(String userId, String? remoteFilename) async {
  if (remoteFilename == null || remoteFilename.isEmpty) return null;
  try {
    final dir = await avatarCacheDir();
    final file = File(
      p.join(dir.path, cachedAvatarFileName(userId, remoteFilename)),
    );
    return file.existsSync() ? file : null;
  } catch (_) {
    return null;
  }
}

/// Best-effort avatar download to the deterministic cache path. Never
/// throws and never blocks sign-in — a failure here only means the nav
/// avatar / profile screen fall back to a network fetch (while online) or a
/// glyph (while offline), never a hang or a crash.
///
/// Also deletes any other cached file for [userId] first, so a changed
/// avatar filename doesn't leave stale files behind.
Future<void> cacheAvatar(
  Dio api,
  String url,
  String userId,
  String remoteFilename,
) async {
  try {
    final dir = await avatarCacheDir();
    final targetName = cachedAvatarFileName(userId, remoteFilename);

    // Clear any previously cached avatar for this user before writing the
    // new one, so a changed remote filename doesn't accumulate stale files.
    final entries = dir.listSync();
    for (final entry in entries) {
      if (entry is! File) continue;
      final name = p.basename(entry.path);
      if (name.startsWith('${userId}_') && name != targetName) {
        try {
          entry.deleteSync();
        } catch (_) {
          // Best-effort cleanup; a stray stale file is harmless.
        }
      }
    }

    final savePath = p.join(dir.path, targetName);
    await api.download(url, savePath);
  } catch (e) {
    debugPrint('avatar_cache_util: failed to cache avatar — $e');
  }
}
