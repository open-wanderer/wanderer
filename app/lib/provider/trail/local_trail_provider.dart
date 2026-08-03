import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/util/current_account.dart';
import 'package:wanderer/util/local_id.dart';
import 'package:wanderer/util/local_trail_store.dart';

part 'local_trail_provider.g.dart';

/// The not-yet-uploaded trail identified by [localId], or null when no
/// such row belongs to the signed-in account (or its cached GPX no
/// longer parses).
///
/// Synchronous on purpose. The row is already on this device: there is
/// no request to await, no `AsyncError` to be auto-retried, and no
/// `AsyncLoading` for a `.when()` to render as a spinner -- exactly the
/// failure mode 36-09 removed from the own-trails list. `Trail?` is
/// also a truer type than `AsyncValue<Trail>`: "this local id has no
/// row" is an ordinary outcome, not an error.
///
/// NOT a variant of `trailProvider`. That one is keyed on the SERVER id
/// and its ObjectBox fallback filters on
/// `savedByUserIds.containsElement(userId)`, which a local capture never
/// sets and MUST never set (D-10) -- writing it would give an unsynced
/// trail the downloaded badge and make the two states indistinguishable.
@riverpod
Trail? localTrail(Ref ref, String localId) {
  if (!isLocalId(localId)) return null;

  final store = ref.watch(objectBoxProvider);
  final accountId = currentAccountId(store);
  if (accountId == null) return null;

  return readOwnLocalTrail(store, localId: localId, accountId: accountId);
}
