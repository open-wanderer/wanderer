import 'dart:io' show SocketException;

import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/util/connectivity_util.dart';

part 'online_status_provider.g.dart';

/// Returns true only when [e] means the request never reached a server —
/// the signal this app treats as "we are offline".
///
/// Covers `connectionError` (no route to host / DNS failure), plus
/// `connectionTimeout`, plus `unknown` wrapping a [SocketException] (some
/// platform channels surface a dropped socket as `unknown` rather than a
/// typed Dio exception).
///
/// Deliberately excludes:
/// - `badResponse` — a server answered (even with a 404/422/500/503), so the
///   backend is reachable; this must never flip the app offline.
/// - `badCertificate` — same reasoning: a TLS handshake means something at
///   the far end responded.
/// - `cancel` — app-initiated, not a connectivity signal.
/// - `sendTimeout` / `receiveTimeout` — cannot fire in this app: `Api.build()`
///   only sets `connectTimeout` (see `api_provider.dart`), never a send or
///   receive timeout, so these types never occur in practice, but they are
///   still excluded on principle since they don't mean "unreachable" either.
bool isConnectionFailure(DioException e) {
  if (e.type == DioExceptionType.connectionError) return true;
  if (e.type == DioExceptionType.connectionTimeout) return true;
  if (e.type == DioExceptionType.unknown && e.error is SocketException) {
    return true;
  }
  return false;
}

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
