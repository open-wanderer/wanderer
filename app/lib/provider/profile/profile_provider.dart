import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/actor_entity.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';

part 'profile_provider.g.dart';

/// Typed exception thrown when the current user is not authenticated.
/// Callers can catch this specifically to show a login prompt rather than
/// a generic error message (distinguishes from network errors).
class NotAuthenticatedException implements Exception {
  const NotAuthenticatedException();

  @override
  String toString() => 'NotAuthenticatedException: user is not authenticated';
}

/// Auto-dispose family provider — fetches any user's profile by handle.
/// Call site: `ref.watch(profileProvider(handle))`
/// Auto-disposes when no longer watched; re-fetches on each navigation.
@riverpod
class ProfileNotifier extends _$ProfileNotifier {
  @override
  FutureOr<Actor> build(String handle) async {
    final api = ref.watch(apiProvider);
    final response = await api.get('/profile/$handle');
    return Actor.fromJson(response.data["actor"] as Map<String, dynamic>);
  }
}

/// keepAlive provider — fetches the current user's own profile Actor.
/// Reads handle from authProvider.preferredUsername.
/// Cache refreshed on pull-to-refresh AND on any auth change: watching
/// `authProvider` instead of reading it once means this provider rebuilds —
/// and re-fetches — the moment auth resolves to a different session, rather
/// than holding the first account's `Actor` forever. Still no app-resume
/// invalidation.
///
/// Writes every successful fetch through to `UserEntity.actor` and falls back
/// to that cached actor when the fetch fails, so the own-profile screen renders
/// its real layout offline instead of an error page. Only the genuinely
/// network-bound sections (counts, lists, feed) degrade.
@Riverpod(keepAlive: true)
class OwnProfile extends _$OwnProfile {
  @override
  FutureOr<Actor> build() async {
    final user = await ref.watch(authProvider.future);
    if (user == null) throw const NotAuthenticatedException();

    try {
      final api = ref.read(apiProvider);
      final response = await api.get('/profile/${user.preferredUsername}');
      final actor = Actor.fromJson(
        response.data["actor"] as Map<String, dynamic>,
      );

      // Refresh the offline cache. ObjectBox cascade-puts the new ToOne
      // target, and ActorEntity's @Unique(onConflict: replace) id keeps this
      // to a single row per actor rather than accumulating copies.
      // `actorEntityForUpsert` reuses the existing row's ObjectBox id so the
      // replace is an in-place update -- a fresh entity would re-mint the id
      // and orphan every `TrailEntity.author` pointing at it.
      final store = ref.read(objectBoxProvider);
      user.actor.target = actorEntityForUpsert(store, actor);
      store.box<UserEntity>().put(user);

      return actor;
    } catch (_) {
      // Any fetch failure — offline, backend down, or a server error — falls
      // back to the last known actor when one is cached. A stale own-profile
      // beats an error page, and pull-to-refresh re-runs this. With no cache
      // (a pre-upgrade install that has not refreshed auth yet) the error
      // propagates so the screen can show its offline/error state.
      final cached = user.actor.target;
      if (cached != null) return cached.toModel();
      rethrow;
    }
  }
}
