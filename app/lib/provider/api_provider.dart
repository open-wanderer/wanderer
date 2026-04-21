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

    final dio = Dio(BaseOptions(baseUrl: "$baseUrl/api/v1"));
    dio.interceptors.add(CookieManager(cookieJar));

    return dio;
  }

  void updateBaseUrl(String baseURL) {
    state.options.baseUrl = baseURL;
  }
}
