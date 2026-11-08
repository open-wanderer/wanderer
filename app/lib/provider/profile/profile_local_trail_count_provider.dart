import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/store/local_trail_store.dart';

part 'profile_local_trail_count_provider.g.dart';

/// How many of [handle]'s trails are on THIS device: the signed-in account's
/// not-yet-uploaded captures plus the downloaded trails it authored itself.
///
/// Counts exactly the rows `readOwnLocalTrails` returns, so the number the
/// profile shows offline is the number of trails `/profile/<handle>/trails`
/// can actually render offline — no second, differently-scoped query that
/// could disagree with the list it links to.
///
/// Null — not zero — for anyone else's handle and for a signed-out store:
/// "this device holds none of their trails" is not an answer worth showing,
/// while a genuine zero for the signed-in hiker is.
///
/// The own-handle
/// test and the actor id are both re-derived here from fresh
/// `authProvider`/`currentAccountId` reads rather than passed in, which is
/// The "never from a cached value" — a stale actor id would pair this
/// account with the previous one's trails.
@riverpod
int? profileLocalTrailCount(Ref ref, String handle) {
  final store = ref.watch(objectBoxProvider);
  final accountId = currentAccountId(store);
  final user = ref.watch(authProvider).value;
  if (accountId == null || user == null) return null;

  if (handle != '@${user.preferredUsername}') return null;

  return readOwnLocalTrails(
    store,
    accountId: accountId,
    authorActorId: user.actorId,
  ).length;
}
