import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/util/local/id.dart';
import 'package:wanderer/store/local_trail_store.dart';

part 'local_trail_provider.g.dart';

/// The not-yet-uploaded trail identified by [localId], or null when no
/// such row belongs to the signed-in account (or its cached GPX no
/// longer parses).
///
/// Synchronous on purpose. The row is already on this device: there is
/// no request to await, no `AsyncError` to be auto-retried, and no
/// `AsyncLoading` for a `.when()` to render as a spinner -- exactly the
/// failure mode the own-trails list deliberately avoids. `Trail?` is
/// also a truer type than `AsyncValue<Trail>`: "this local id has no
/// row" is an ordinary outcome, not an error.
///
/// NOT a variant of `trailProvider`. That one is keyed on the SERVER id
/// and its ObjectBox fallback filters on
/// `savedByUserIds.containsElement(userId)`, which a local capture never
/// sets and MUST never set -- writing it would give an unsynced
/// trail the downloaded badge and make the two states indistinguishable.
@riverpod
Trail? localTrail(Ref ref, String localId) {
  if (!isLocalId(localId)) return null;

  final store = ref.watch(objectBoxProvider);
  final accountId = currentAccountId(store);
  if (accountId == null) return null;

  return readOwnLocalTrail(store, localId: localId, accountId: accountId);
}

/// Whether [localId] identifies a live, not-yet-uploaded capture owned by
/// the signed-in account.
///
/// The single UI-facing handle on `isOwnLiveCapture`: it exists for
/// exactly two call sites -- `trail_dropdown.dart`'s `_allowDelete` and its
/// download-family guard, and `library_screen.dart`'s Remove tile -- and
/// there must be exactly one predicate serving both, never two.
///
/// [localId] is nullable on purpose so both call sites can watch this
/// unconditionally with `ref.watch(ownLiveCaptureProvider(trail.localId))`
/// rather than writing a conditional `ref.watch`.
@riverpod
bool ownLiveCapture(Ref ref, String? localId) {
  final store = ref.watch(objectBoxProvider);
  final accountId = currentAccountId(store);
  if (accountId == null) return false;

  return isOwnLiveCapture(store, localId: localId, accountId: accountId);
}
