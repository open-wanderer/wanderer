import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/util/connectivity.dart';

// `isConnectionFailure` lives next to the probe it shares its rule with, in
// util/connectivity.dart — a pure helper that must not depend on a provider.
// Re-exported here because this is where every caller already imports it from.
export 'package:wanderer/util/connectivity.dart' show isConnectionFailure;

part 'online_status_provider.g.dart';

/// The app's single keepAlive source of truth for "is the backend reachable
/// right now". Optimistic by default (`true`), kept live by an
/// `InterceptorsWrapper` on the shared api client (see `api_provider.dart`)
/// so ordinary request/response traffic maintains it without any screen
/// having to probe explicitly.
///
/// `build()` reads no other provider — this is what prevents mutual
/// synchronous initialisation with `Api.build()`, since the api client's
/// interceptor reads this notifier lazily inside its callbacks, never during
/// its own `build()`.
@Riverpod(keepAlive: true)
class OnlineStatus extends _$OnlineStatus {
  @override
  bool build() => true;

  void markOnline() {
    if (state != true) state = true;
  }

  void markOffline() {
    if (state != false) state = false;
  }

  /// Re-probes the backend directly (via [isBackendReachable]) and updates
  /// [state] to match, returning the fresh result. Runs post-build, so
  /// writing `state` here is always safe.
  ///
  /// No-ops while the api client still points at [kUnconfiguredApiHost]: the
  /// startup seed in `main.dart` fires before `Auth.build()` applies the real
  /// server URL, so probing here would hit a host that cannot resolve and
  /// report the app offline on every cold start. Leaving the optimistic default
  /// in place is correct — the interceptor settles the true status from auth's
  /// own traffic moments later.
  Future<bool> refresh() async {
    if (!ref.read(apiProvider.notifier).isConfigured) return state;
    final api = ref.read(apiProvider);
    final result = await isBackendReachable(api);
    state = result;
    return result;
  }
}
