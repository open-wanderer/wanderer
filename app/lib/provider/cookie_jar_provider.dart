import 'package:cookie_jar/cookie_jar.dart';
import 'package:path_provider/path_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'dart:io';

part 'cookie_jar_provider.g.dart';

@Riverpod(keepAlive: true)
Future<PersistCookieJar> cookieJar(Ref ref) async {
  final appDocDir = await getApplicationDocumentsDirectory();
  final String path = "${appDocDir.path}/.cookies/";

  final dir = Directory(path);
  if (!await dir.exists()) {
    await dir.create(recursive: true);
  }

  return PersistCookieJar(storage: FileStorage(path), ignoreExpires: false);
}
