import 'dart:io' show SocketException;

import 'package:dio/dio.dart';

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
///
/// Re-exported by `online_status_provider.dart`, which is where most callers
/// import it from.
bool isConnectionFailure(DioException e) {
  if (e.type == DioExceptionType.connectionError) return true;
  if (e.type == DioExceptionType.connectionTimeout) return true;
  if (e.type == DioExceptionType.unknown && e.error is SocketException) {
    return true;
  }
  return false;
}

/// Ceiling on the whole probe. Must stay >= the api client's `connectTimeout`
/// (8s, see `api_provider.dart`): a shorter ceiling makes the probe give up on
/// a connection the transport itself would still have completed, reporting
/// offline on nothing but a slow link. The extra headroom covers the response
/// phase — an authenticated `/health` costs four sequential PocketBase
/// round-trips server-side (`authRefresh` + settings + actor lookups in
/// web/src/hooks.server.ts, then the route's own health check), which is why
/// this probe is far more expensive after login than before it.
const Duration _probeTimeout = Duration(seconds: 10);

/// The app's underlying backend reachability probe.
///
/// Probes the unauthenticated `/health` endpoint on the SvelteKit proxy — the
/// app always talks to that proxy, never PocketBase directly.
///
/// Reachability, not health: ANY answer from the far end proves the backend is
/// reachable, whatever its status — the same rule [isConnectionFailure] states
/// for ordinary traffic. Only two things read as offline:
/// - a connection-level failure or a timeout (nothing answered), and
/// - a 503, the one status `/health` itself defines, meaning the proxy is up
///   but its database is not.
///
/// A 404 explicitly reads as ONLINE: the route was added late
/// (`web/src/routes/api/v1/health/+server.ts`) and servers running an older
/// build simply do not have it, so requiring a 200 reported every such
/// instance offline while every other call to it worked.
///
/// This is an active check, not mere network-interface state, so it still
/// reports offline when a self-hosted instance is down. It fails fast in
/// airplane mode — the api client's `connectTimeout` bounds it, and it returns
/// immediately when there is no route at all.
///
/// This is the underlying probe behind `onlineStatusProvider`
/// (`online_status_provider.dart`) and is no longer called directly by
/// screens.
Future<bool> isBackendReachable(Dio api) async {
  try {
    final res = await api
        .get(
          '/health',
          // Hand every status back as a Response instead of throwing, so the
          // "a server answered" case is decided here rather than by Dio's
          // default 200-only validator.
          options: Options(validateStatus: (_) => true),
        )
        .timeout(_probeTimeout);
    return res.statusCode != 503;
  } on DioException catch (e) {
    return !isConnectionFailure(e);
  } catch (_) {
    // Includes the TimeoutException from [_probeTimeout]: nothing answered in
    // time, so treat it as unreachable.
    return false;
  }
}
