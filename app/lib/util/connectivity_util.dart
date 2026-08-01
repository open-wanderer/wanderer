import 'package:dio/dio.dart';

/// The app's underlying backend reachability probe.
///
/// Probes the unauthenticated `/health` endpoint on the SvelteKit proxy — the
/// app always talks to that proxy, never PocketBase directly — under a short
/// timeout. The endpoint pings PocketBase in turn, so a 200 means both the
/// proxy and the database are up. Returns true only on a 200; a 503 (database
/// down), a connection error, or a timeout all read as offline.
///
/// This is an active reachability check, not mere network-interface state, so
/// it also reports offline when a self-hosted instance is simply down. It
/// fails fast in airplane mode — the api client's `connectTimeout` bounds it,
/// and it returns immediately when there is no route at all.
///
/// This is the underlying probe behind `onlineStatusProvider`
/// (`online_status_provider.dart`) and is no longer called directly by
/// screens.
Future<bool> isBackendReachable(Dio api) async {
  try {
    final res = await api.get('/health').timeout(const Duration(seconds: 5));
    return res.statusCode == 200;
  } catch (_) {
    return false;
  }
}
