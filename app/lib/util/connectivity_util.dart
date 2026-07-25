import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/provider/api_provider.dart';

/// The app's single source of truth for "is the backend reachable right now".
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
Future<bool> isBackendReachable(WidgetRef ref) async {
  try {
    final api = ref.read(apiProvider);
    final res = await api.get('/health').timeout(const Duration(seconds: 5));
    return res.statusCode == 200;
  } catch (_) {
    return false;
  }
}
