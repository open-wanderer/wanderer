import 'package:dio/dio.dart';
import 'package:dio_cookie_manager/dio_cookie_manager.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';

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

    return dio;
  }

  void updateBaseUrl(String baseUrl) {
    state.options.baseUrl = "$baseUrl/api/v1";
  }
}
