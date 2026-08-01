/// The single sanctioned account-scoped purge, run at both account-change
/// chokepoints (`Auth.logout()` and a detected account switch in
/// `Auth._updateUserEntity`).
///
/// The invariant this enforces is that **no other account's content is
/// reachable** after a switch — NOT that no other account's bytes remain.
/// Those are different goals, and an earlier revision chased the wrong one:
/// it deleted the whole offline library on logout, so signing out of account
/// A and back in destroyed every trail A had downloaded. For an offline-first
/// app that is a worse failure than the on-device forensic exposure it was
/// guarding against, which nobody asked for.
///
/// So the library is now SCOPED rather than deleted: `TrailEntity` carries
/// `savedByUserIds`, and every read path filters on the signed-in account
/// (see `trail_library_provider.dart`, `trail_provider.dart`,
/// `navigation_launch_util.dart`). This purge only clears rows and caches
/// that are cheap to refetch and hold no durable user work.
library;

import 'dart:io';

import 'package:objectbox/objectbox.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/entities/active_navigation_entity.dart';
import 'package:wanderer/entities/settings_entity.dart';

/// The app-docs subdirectory names holding account-scoped content.
///
/// Deliberately does NOT include `library`, `objectbox`, `regions`,
/// `map_cache` or `.cookies`: `library` holds downloaded trails, which now
/// survive an account switch and are scoped per account via
/// `TrailEntity.savedByUserIds`; `objectbox` is the database itself (its rows
/// are purged selectively via [purgeAccountScopedBoxes], never by deleting the
/// directory); `regions` and `map_cache` are server/basemap data — expensive
/// to re-download and not tied to any one account; and the cookie jar is
/// cleared by `Auth.logout()` itself via `PersistCookieJar.deleteAll()`.
///
/// `avatars` stays: it is a cache of public avatar images, cheap to refetch.
const accountScopedDirNames = <String>['avatars'];

/// Pure switch predicate: whether the incoming user id requires a purge of
/// the previous account's local data.
///
/// A `null` [cachedUserId] is NOT a purge trigger — a first login on a
/// clean install has nothing to purge, and purging there would delete
/// nothing while still costing a directory walk on every cold start.
bool shouldPurgeForIncomingUser(String? cachedUserId, String incomingUserId) {
  return cachedUserId != null && cachedUserId != incomingUserId;
}

/// Deletes `<root>/<name>` recursively for every name in
/// [accountScopedDirNames], if present. Every path is built via
/// `package:path`'s [p.join], never string concatenation.
///
/// Best-effort: each deletion is wrapped in its own try/catch so a single
/// unwritable/absent path can never prevent the others from being cleared,
/// and this function never throws.
Future<void> purgeAccountScopedDirectories(String root) async {
  for (final name in accountScopedDirNames) {
    try {
      final dir = Directory(p.join(root, name));
      if (await dir.exists()) {
        await dir.delete(recursive: true);
      }
    } catch (_) {
      // Best-effort — a failure to delete one account-scoped directory must
      // never prevent the others from being cleared.
    }
  }
}

/// Clears the ObjectBox boxes that are cheap to refetch and hold no durable
/// user work: `SettingsEntity` and `ActiveNavigationEntity`.
///
/// Deliberately NOT cleared:
/// - `TrailEntity`/`WaypointEntity` — the offline library. Downloaded trails
///   are durable user work and must survive an account switch; they are hidden
///   from other accounts by `TrailEntity.savedByUserIds` membership, not by
///   deletion.
/// - `ActorEntity` — backs `TrailEntity.author`. Clearing it while keeping
///   trails would strand every kept trail's author (ObjectBox does not cascade
///   a `ToOne`), and it only caches public actor data anyway.
/// - `UserEntity` — the `Auth` notifier owns that box and clears it at both of
///   its own call sites.
///
/// `SettingsEntity` is per-account server state, refetched on the next login
/// via `updateFromServer`. `ActiveNavigationEntity` is a deliberately
/// ephemeral in-progress session.
///
/// Best-effort (mirrors `active_navigation_store.dart`'s discipline): the
/// whole transaction is wrapped in try/catch and this function never
/// throws.
void purgeAccountScopedBoxes(Store store) {
  try {
    store.runInTransaction(TxMode.write, () {
      store.box<SettingsEntity>().removeAll();
      store.box<ActiveNavigationEntity>().removeAll();
    });
  } catch (_) {
    // Best-effort — a failed purge transaction must never crash the caller.
  }
}

/// Entry point production code calls: purges every account-scoped
/// ObjectBox row, then every account-scoped on-disk directory.
Future<void> purgeAccountScopedData(Store store) async {
  purgeAccountScopedBoxes(store);
  final root = (await getApplicationDocumentsDirectory()).path;
  await purgeAccountScopedDirectories(root);
}
