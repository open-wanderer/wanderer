/// The single sanctioned account-scoped purge: clears every ObjectBox row
/// and on-disk file that belongs to a signed-in account, at both
/// account-change chokepoints (`Auth.logout()` and a detected account
/// switch in `Auth._updateUserEntity`).
///
/// This is the structural fix for the privacy leak where a local library,
/// its downloaded files, and other per-account rows/caches survived a
/// logout and remained readable by whichever account signs in next (T-h2p-01).
library;

import 'dart:io';

import 'package:objectbox/objectbox.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/entities/active_navigation_entity.dart';
import 'package:wanderer/entities/actor_entity.dart';
import 'package:wanderer/entities/settings_entity.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/waypoint_entity.dart';

/// The app-docs subdirectory names holding account-scoped content.
///
/// Deliberately does NOT include `objectbox`, `regions`, `map_cache` or
/// `.cookies`: `objectbox` is the database itself (its rows are purged
/// separately via [purgeAccountScopedBoxes], not by deleting the directory);
/// `regions` and `map_cache` are server/basemap data — expensive to
/// re-download and not tied to any one account — so they must survive an
/// account switch; and the cookie jar is cleared by `Auth.logout()` itself
/// via `PersistCookieJar.deleteAll()`, not by this util.
const accountScopedDirNames = <String>['library', 'avatars'];

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

/// Clears every ObjectBox box holding account-scoped rows: `TrailEntity`,
/// `WaypointEntity`, `ActorEntity`, `SettingsEntity` and
/// `ActiveNavigationEntity`.
///
/// `WaypointEntity` and `ActorEntity` are cleared explicitly because
/// ObjectBox does not cascade a `ToOne`/`ToMany` target's deletion —
/// clearing only `TrailEntity` would strand account A's waypoint and actor
/// rows. `UserEntity` is deliberately NOT cleared here — the `Auth` notifier
/// owns that box and clears it at both of its own call sites.
///
/// Best-effort (mirrors `active_navigation_store.dart`'s discipline): the
/// whole transaction is wrapped in try/catch and this function never
/// throws.
void purgeAccountScopedBoxes(Store store) {
  try {
    store.runInTransaction(TxMode.write, () {
      store.box<TrailEntity>().removeAll();
      store.box<WaypointEntity>().removeAll();
      store.box<ActorEntity>().removeAll();
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
