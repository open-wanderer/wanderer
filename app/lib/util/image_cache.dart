import 'package:cached_network_image/cached_network_image.dart';
import 'package:cookie_jar/cookie_jar.dart';
import 'package:dio_cookie_manager/dio_cookie_manager.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';
import 'package:http/http.dart' as http;

/// Disk-cached image provider for every network image in the app.
///
/// Use this instead of a bare `CachedNetworkImageProvider`: that one fetches
/// through [DefaultCacheManager], whose HTTP client never sees the session
/// cookie. On a private instance (`PUBLIC_PRIVATE_INSTANCE=true`)
/// `web/src/hooks.server.ts` rejects every `/api/v1/files` request without
/// `pb_auth` with a 401, so all photos and avatars failed to load (#1456).
ImageProvider wandererImageProvider(String url) =>
    CachedNetworkImageProvider(url, cacheManager: WandererImageCacheManager());

/// [CacheManager] whose fetches carry the cookies the API client holds for
/// the requested host.
///
/// Reuses [DefaultCacheManager.key], so images cached before this manager
/// existed are still served from disk. Nothing else may open a
/// [DefaultCacheManager]: two managers on one key share a database without
/// sharing their in-memory state.
class WandererImageCacheManager extends CacheManager with ImageCacheManager {
  /// The API client's cookie jar, set once in `main()` before `runApp`.
  ///
  /// Static rather than read from `cookieJarProvider` because image providers
  /// are built in plain widgets with no [Ref]. Null (in tests) sends no
  /// cookie.
  static CookieJar? cookieJar;

  /// Configured server host. Cookie domains may cover more than this host.
  static String? serverHost;

  static final WandererImageCacheManager _instance =
      WandererImageCacheManager._();

  factory WandererImageCacheManager() => _instance;

  WandererImageCacheManager._()
    : super(Config(DefaultCacheManager.key, fileService: CookieFileService()));
}

/// Image fetches authenticated only on the configured server's host.
class CookieFileService extends FileService {
  CookieFileService({http.Client? httpClient})
    : _httpClient = httpClient ?? http.Client();

  final http.Client _httpClient;

  @override
  Future<FileServiceResponse> get(
    String url, {
    Map<String, String>? headers,
  }) async {
    var uri = Uri.parse(url);
    for (var redirects = 0; ; redirects++) {
      final request = http.Request('GET', uri)..followRedirects = false;
      request.headers.addAll({
        for (final entry in (headers ?? const <String, String>{}).entries)
          if (entry.key.toLowerCase() != 'cookie') entry.key: entry.value,
      });

      final jar = WandererImageCacheManager.cookieJar;
      if (jar != null && uri.host == WandererImageCacheManager.serverHost) {
        final cookies = (await jar.loadForRequest(uri))
            .where((cookie) => !cookie.secure || uri.isScheme('https'))
            .toList();
        if (cookies.isNotEmpty) {
          request.headers['cookie'] = CookieManager.getCookies(cookies);
        }
      }

      final response = await _httpClient.send(request);
      final location = response.headers['location'];
      if (![301, 302, 303, 307, 308].contains(response.statusCode) ||
          location == null) {
        return HttpGetResponse(response);
      }
      await response.stream.drain<void>();
      if (redirects >= 5) {
        throw http.ClientException('Too many image redirects', uri);
      }
      uri = uri.resolve(location);
    }
  }
}
