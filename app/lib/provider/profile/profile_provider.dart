import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';

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
    return Actor.fromJson(response.data as Map<String, dynamic>);
  }
}

/// keepAlive provider — fetches the current user's own profile Actor.
/// Reads handle from authProvider.preferredUsername (per D-02).
/// Cache refreshed only on pull-to-refresh (Phase 2); no app-resume invalidation (D-03).
@Riverpod(keepAlive: true)
class OwnProfile extends _$OwnProfile {
  @override
  FutureOr<Actor> build() async {
    final user = await ref.read(authProvider.future);
    if (user == null) throw const NotAuthenticatedException();
    final api = ref.read(apiProvider);
    final response = await api.get('/profile/${user.preferredUsername}');
    return Actor.fromJson(response.data["actor"] as Map<String, dynamic>);
  }
}
