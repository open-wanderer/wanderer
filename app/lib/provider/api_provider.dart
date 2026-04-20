import 'package:dio_cookie_manager/dio_cookie_manager.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:dio/dio.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';

part 'api_provider.g.dart';

@riverpod
Dio api(Ref ref) {
  final cookieJar = ref.watch(cookieJarProvider).value;

  final dio = Dio(BaseOptions(baseUrl: 'https://demo.wanderer.to/api/v1'));
  if (cookieJar != null) {
    dio.interceptors.add(CookieManager(cookieJar));
  }

  return dio;
}
