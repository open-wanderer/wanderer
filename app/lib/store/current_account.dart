/// Resolves the signed-in account id straight from the `UserEntity` box.
///
/// Deliberately a plain function rather than a provider: the value must never
/// be served from a Riverpod cache. Every account-scoped read (the offline
/// library and the offline single-trail lookups) filters on this id, and a
/// stale cached id would show the previous account's content to the new one --
/// the exact leak this scoping exists to prevent. Reading the box costs a
/// single indexed lookup, so there is nothing to gain by caching it.
///
/// This is the same expression `Auth._updateUserEntity` uses to detect an
/// account switch, and it relies on the same invariant: the store holds at
/// most ONE `UserEntity` row, enforced by `_box.removeAll()` before every
/// `_box.put(userEntity)`.
library;

import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/user_entity.dart';

/// The signed-in account's id, or null when no user is signed in.
///
/// Callers must treat null as "no account scope" — an empty library, not an
/// unfiltered one.
String? currentAccountId(Store store) {
  return store.box<UserEntity>().getAll().firstOrNull?.id;
}
