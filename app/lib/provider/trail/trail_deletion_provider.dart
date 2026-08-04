import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'trail_deletion_provider.g.dart';

/// One announced trail deletion. [seq] exists purely so two deletions are
/// never `==` to each other: Riverpod skips notifying listeners when the new
/// state equals the old one, and a record of a single `id` field would make a
/// re-announced id (a retried delete, or the same id after a re-login) a
/// silent no-op for every listener below.
class TrailDeletion {
  final String id;
  final int seq;
  const TrailDeletion(this.id, this.seq);
}

/// Broadcasts "this trail no longer exists on the server" to anything holding
/// a cached copy of it.
///
/// Deliberately NOT `ref.invalidate` on the map providers: `MapTrailSearch`
/// and `MapClusterSearch` keep their last bounds in instance fields, so
/// invalidating them drops the user back to an empty result list (and an empty
/// map) until they pan far enough to trigger a fresh bounds search. Listeners
/// splice the deleted id out of the state they already hold instead.
///
/// Only server deletes are announced here. Un-downloading a trail
/// (`TrailLibrary.deleteTrail`) leaves it on the server, so it must keep
/// showing up in map results.
@Riverpod(keepAlive: true)
class TrailDeletions extends _$TrailDeletions {
  int _seq = 0;

  @override
  TrailDeletion? build() => null;

  void announce(String id) {
    if (id.isEmpty) return;
    state = TrailDeletion(id, ++_seq);
  }
}
