import 'package:cookie_jar/cookie_jar.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'cookie_jar_provider.g.dart';

@Riverpod(keepAlive: true)
PersistCookieJar cookieJar(Ref ref) {
  // This will be overridden in main.dart
  throw UnimplementedError();
}
