import 'package:dio/dio.dart';
import 'package:dio_cookie_manager/dio_cookie_manager.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';

part 'api_provider.g.dart';

@Riverpod(keepAlive: true)
class Api extends _$Api {
  @override
  Dio build() {
    final cookieJar = ref.watch(cookieJarProvider);

    final baseUrl = "https://unknown-server.local";

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
    dio.interceptors.add(CookieManager(cookieJar));

    // Feeds `onlineStatusProvider` from every request this shared client
    // makes. The notifier is resolved lazily inside each closure body (never
    // here, at build time) — resolving it during `Api.build()` would
    // re-enter `OnlineStatus` mid-build. Interceptor callbacks run in the
    // async request pipeline, outside the widget build phase, so this cannot
    // trip Riverpod's debug "modified a provider while the widget tree was
    // building" guard. `handler.next(...)` is always called so the response
    // or error still reaches its original caller.
    dio.interceptors.add(
      InterceptorsWrapper(
        onResponse: (response, handler) {
          ref.read(onlineStatusProvider.notifier).markOnline();
          handler.next(response);
        },
        onError: (err, handler) {
          final notifier = ref.read(onlineStatusProvider.notifier);
          if (isConnectionFailure(err)) {
            notifier.markOffline();
          } else {
            notifier.markOnline();
          }
          handler.next(err);
        },
      ),
    );

    return dio;
  }

  void updateBaseUrl(String baseUrl) {
    state.options.baseUrl = "$baseUrl/api/v1";
  }
}
