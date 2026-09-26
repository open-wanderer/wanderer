import 'package:dio/dio.dart';
import 'package:dio_cookie_manager/dio_cookie_manager.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/util/server_url.dart';

part 'api_provider.g.dart';

/// Placeholder host the shared client starts on, before a signed-in user's
/// server URL is known (applied later by [Api.updateBaseUrl], from
/// `Auth.build()`).
///
/// Requests issued against it are meaningless — the host does not resolve — so
/// connectivity must never be judged from their failures. Doing so reported the
/// app offline on every cold start: the startup probe in `main.dart` runs before
/// auth has applied the real server URL, and its DNS failure looked exactly
/// like a genuine connection error.
const String kUnconfiguredApiHost = 'unknown-server.local';

/// Whether [uri] addresses the configured Wanderer backend — the only traffic
/// that says anything about that backend's reachability.
///
/// This client is shared by more than the API: it also warms the map
/// glyph/sprite cache, whose URLs come from `/map/style-sources` and default to
/// a THIRD-PARTY host (`protomaps.github.io`) unless the operator sets
/// `MAP_ASSETS_URL`. That warm is ~1000 requests on a fresh install
/// (`glyph_sprite_cache_provider.dart`), and without this gate a single dropped
/// connection anywhere in that burst — throttling, a carrier NAT, a slow link —
/// flipped the whole app offline while the backend was perfectly reachable.
///
/// Compared by host alone: a differing port or path still addresses the same
/// server. Traffic issued while the client is still on [kUnconfiguredApiHost]
/// is excluded too — those requests are meaningless (the host does not
/// resolve), which is the whole point of the placeholder.
bool isBackendRequest(Dio dio, Uri uri) {
  final baseHost = Uri.parse(dio.options.baseUrl).host;
  if (baseHost.isEmpty || baseHost == kUnconfiguredApiHost) return false;
  return uri.host == baseHost;
}

@Riverpod(keepAlive: true)
class Api extends _$Api {
  /// The default [CancelToken] attached to any request that does not bring
  /// its own — which is every ordinary API call (auth, feed, trail, profile).
  /// [cancelPendingRequests] cancels this one and swaps in a fresh instance,
  /// so `Auth.logout()` can abort whatever is still awaiting a response on
  /// the shared client without disturbing callers that already manage their
  /// own token, notably region/tile downloads and trail uploads
  /// (`tile_repository_manager.dart`, `trail_download_service.dart`), which
  /// must keep running across a logout per `account_scope_invalidation.dart`.
  ///
  /// Not `late final`: this field is assigned at construction and reassigned
  /// by [cancelPendingRequests], never inside [build] — `build` can run again
  /// on the same instance and a `late final` reassignment would throw.
  CancelToken _sessionCancelToken = CancelToken();

  @override
  Dio build() {
    final cookieJar = ref.watch(cookieJarProvider);

    final baseUrl = "https://$kUnconfiguredApiHost";

    // Bound the connect phase so an offline call fails fast instead of hanging
    // on the OS socket timeout — this is what lets the trail-load fallback in
    // TrailNotifier reach its local-entity path quickly when offline. Only
    // connectTimeout is set: a receiveTimeout would fire on inter-chunk gaps
    // and could abort legitimate large region/tile downloads.
    final dio = Dio(
      BaseOptions(
        baseUrl: "$baseUrl/api/v1",
        connectTimeout: const Duration(seconds: 8),
      ),
    );

    // Must run before CookieManager below: a request still awaiting its
    // response when `Auth.logout()` calls `cancelPendingRequests()` must be
    // aborted before CookieManager's `onResponse` can persist a Set-Cookie
    // from it into the (just-cleared) jar. `web/src/hooks.server.ts` appends
    // a fresh `pb_auth` Set-Cookie to every response, so any response landing
    // after logout would otherwise resurrect the signed-out session. See
    // `.planning/debug/app-relogin-after-logout.md`.
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          options.cancelToken ??= _sessionCancelToken;
          handler.next(options);
        },
      ),
    );
    dio.interceptors.add(CookieManager(cookieJar));

    // Feeds `onlineStatusProvider` from every request this shared client makes
    // TO THE CONFIGURED BACKEND — see [isBackendRequest] for why third-party
    // traffic on this same client must never move the status. The notifier is
    // resolved lazily inside each closure body (never here, at build time) —
    // resolving it during `Api.build()` would re-enter `OnlineStatus`
    // mid-build. Interceptor callbacks run in the async request pipeline,
    // outside the widget build phase, so this cannot trip Riverpod's debug
    // "modified a provider while the widget tree was building" guard.
    // `handler.next(...)` is always called so the response or error still
    // reaches its original caller.
    dio.interceptors.add(
      InterceptorsWrapper(
        onResponse: (response, handler) {
          if (isBackendRequest(dio, response.requestOptions.uri)) {
            ref.read(onlineStatusProvider.notifier).markOnline();
          }
          handler.next(response);
        },
        onError: (err, handler) {
          if (isBackendRequest(dio, err.requestOptions.uri)) {
            final notifier = ref.read(onlineStatusProvider.notifier);
            if (isConnectionFailure(err)) {
              notifier.markOffline();
            } else {
              notifier.markOnline();
            }
          }
          handler.next(err);
        },
      ),
    );

    return dio;
  }

  /// Aborts every request currently in flight on the shared client that did
  /// not bring its own [CancelToken], then arms a fresh token for requests
  /// issued from now on.
  ///
  /// Called from `Auth.logout()` before any local teardown, so a request
  /// that was already awaiting a response — e.g. `_validateInBackground`'s
  /// GET `/user/:id` — is aborted instead of allowed to land afterwards. A
  /// request genuinely still awaiting network I/O at that point has its
  /// underlying socket aborted (`dio`'s `IOHttpClientAdapter` ties
  /// `CancelToken` to `HttpClientRequest.abort()`), so neither
  /// `CookieManager` nor the caller ever sees a response to act on. See
  /// `.planning/debug/app-relogin-after-logout.md` for the full mechanism.
  ///
  /// Deliberately scoped to the default token only: requests that already
  /// carry their own `CancelToken` (region/tile downloads, trail uploads)
  /// are untouched, since those are designed to survive a logout — see the
  /// exclusion notes in `account_scope_invalidation.dart`.
  void cancelPendingRequests() {
    final expiring = _sessionCancelToken;
    _sessionCancelToken = CancelToken();
    if (!expiring.isCancelled) {
      expiring.cancel('Signed out');
    }
  }

  /// Points the shared client at [baseUrl].
  ///
  /// Normalised first, so a bare host still yields a valid absolute URL —
  /// Dio's `baseUrl` setter THROWS on a hostless value, which used to surface
  /// as the instance picker refusing to close. A value that cannot be
  /// normalised at all is passed through unchanged, so it fails loudly here
  /// rather than being silently swallowed into an unusable client.
  void updateBaseUrl(String baseUrl) {
    final normalized = normalizeServerUrl(baseUrl) ?? baseUrl;
    state.options.baseUrl = "$normalized/api/v1";
  }

  /// Whether a real server URL has been applied yet. Until it has, no request
  /// this client makes can report anything meaningful about connectivity.
  bool get isConfigured =>
      !Uri.parse(state.options.baseUrl).host.contains(kUnconfiguredApiHost);
}
