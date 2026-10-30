import 'package:cached_network_image/cached_network_image.dart';
import 'package:cookie_jar/cookie_jar.dart';
import 'package:dio_cookie_manager/dio_cookie_manager.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';

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

  static final WandererImageCacheManager _instance =
      WandererImageCacheManager._();

  factory WandererImageCacheManager() => _instance;

  WandererImageCacheManager._()
    : super(Config(DefaultCacheManager.key, fileService: CookieFileService()));
}

/// [HttpFileService] that adds the [WandererImageCacheManager.cookieJar]
/// cookies matching each requested URL.
class CookieFileService extends HttpFileService {
  CookieFileService({super.httpClient});

  @override
  Future<FileServiceResponse> get(
    String url, {
    Map<String, String>? headers,
  }) async {
    final jar = WandererImageCacheManager.cookieJar;
    if (jar == null) return super.get(url, headers: headers);

    // The jar matches cookies by domain and path, so a remote actor's avatar
    // or a DiceBear fallback never receives this instance's session token.
    final cookies = await jar.loadForRequest(Uri.parse(url));
    if (cookies.isEmpty) return super.get(url, headers: headers);

    return super.get(
      url,
      headers: {...?headers, 'cookie': CookieManager.getCookies(cookies)},
    );
  }
}
