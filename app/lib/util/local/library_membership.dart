/// Pure set arithmetic for `TrailEntity.savedByUserIds` — which accounts hold
/// a trail in their offline library.
///
/// A trail is ONE row and ONE copy of its files, shared by every account that
/// downloaded it. That makes the add/remove rules the load-bearing part: get
/// them wrong and either an account keeps seeing a trail it deleted, or one
/// account's delete takes another account's offline copy with it. Both rules
/// live here as pure functions so they are testable without an ObjectBox
/// `Store` (this repo cannot construct one under `flutter test`).
library;

/// The membership set after [userId] saves the trail.
///
/// Idempotent — re-downloading a trail the account already holds must not
/// duplicate its id, or the delete-time reference count would never reach
/// zero and the files would leak forever. A null [userId] (no signed-in
/// account) adds nothing rather than granting everyone access.
List<String> libraryMembersAfterSave(Iterable<String> current, String? userId) {
  final members = {...current};
  if (userId != null) members.add(userId);
  return members.toList();
}

/// The membership set after [userId] removes the trail from their library.
///
/// An empty result is the signal that the LAST holder gave it up, so the row
/// and the `library/<id>/` directory can finally be deleted. A non-empty
/// result means another account still has a working offline copy and nothing
/// on disk may be touched.
List<String> libraryMembersAfterDelete(
  Iterable<String> current,
  String userId,
) {
  return current.where((candidate) => candidate != userId).toList();
}
