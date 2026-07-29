import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter_web_auth_2/flutter_web_auth_2.dart';
import 'package:objectbox/objectbox.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/auth_response.dart';
import 'package:wanderer/models/oauth_provider.dart';
import 'package:wanderer/models/user.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/util/avatar_cache_util.dart';

part 'auth_provider.g.dart';

/// The custom URL scheme registered natively (Android intent-filter, iOS
/// CFBundleURLSchemes) to receive the OAuth callback relayed by the web app's
/// `/login/redirect` page. Must match [OAuthProvider.url]'s redirect_uri host
/// registered with each OAuth provider — see login/+page.svelte redirect page.
const _oauthCallbackScheme = "wanderer";

/// Appended to the PocketBase-generated `state` value before opening the
/// authorization URL, so `/login/redirect` can tell an app-originated flow
/// apart from a plain web login by looking at the state value itself, rather
/// than inferring it from the absence of web-only signals (localStorage),
/// which browsers can also legitimately lack. PocketBase's OAuth2 code
/// exchange never inspects `state` server-side — it's a purely client-side
/// CSRF check — so it's safe to tag. Keep in sync with the identical
/// constant in login/redirect/+page.svelte.
const _oauthAppStateMarker = ".wanderer-app";

@Riverpod(keepAlive: true)
class Auth extends _$Auth {
  Box<UserEntity> get _box => ref.read(objectBoxProvider).box<UserEntity>();

  @override
  FutureOr<UserEntity?> build() async {
    final store = ref.watch(objectBoxProvider);

    final box = store.box<UserEntity>();

    final savedUserEntity = box.getAll().firstOrNull;
    if (savedUserEntity == null) {
      return null;
    }

    ref.read(apiProvider.notifier).updateBaseUrl(savedUserEntity.serverUrl);

    final jar = ref.watch(cookieJarProvider);

    final cookies = await jar.loadForRequest(
      Uri.parse(savedUserEntity.serverUrl),
    );
    final pbAuthCookie = cookies.where((c) => c.name == 'pb_auth').firstOrNull;

    if (pbAuthCookie != null) {
      final validation = _updateUserEntity(savedUserEntity.id);
      unawaited(validation.catchError((_) => null));

      try {
        final validated = await validation.timeout(const Duration(seconds: 3));
        return validated ?? savedUserEntity;
      } on TimeoutException {
        // Offline/slow network: preserve the cached session.
        return savedUserEntity;
      } catch (err) {
        if (_isAuthError(err)) {
          unawaited(Future.microtask(logout));
          return null;
        }
        // Any other error (e.g. a 5xx or Dio connectionError/
        // connectionTimeout) is treated as offline; preserve the cached
        // session.
        return savedUserEntity;
      }
    }
    return null;
  }

  bool _isAuthError(Object err) {
    if (err is DioException) {
      final statusCode = err.response?.statusCode;
      return statusCode == 401 || statusCode == 403 || statusCode == 404;
    }
    return false;
  }

  Future<UserEntity?> register(
    String username,
    String email,
    String password,
  ) async {
    if (username.isEmpty || email.isEmpty || password.isEmpty) {
      state = AsyncError(
        Exception("Fields cannot be empty"),
        StackTrace.current,
      );
      return null;
    }

    state = const AsyncLoading();

    // register
    state = await AsyncValue.guard(() async {
      await ref
          .read(apiProvider)
          .put(
            '/user',
            data: {
              'username': username,
              'email': email,
              'password': password,
              'passwordConfirm': password,
            },
          );
      return await login(username, password);
    });
    return state.value;
  }

  Future<UserEntity?> login(String username, String password) async {
    if (username.isEmpty || password.isEmpty) {
      state = AsyncError(
        Exception("Fields cannot be empty"),
        StackTrace.current,
      );
      return null;
    }

    state = const AsyncLoading();

    // Login
    state = await AsyncValue.guard(() async {
      final loginResponse = await ref
          .read(apiProvider)
          .post(
            '/auth/login',
            data: {'username': username, 'password': password},
          );
      final authData = AuthResponse.fromJson(loginResponse.data);

      // Fetch user data with expanded actor
      final userEntity = await _updateUserEntity(authData.record.id);
      return userEntity;
    });
    return state.value;
  }

  Future<UserEntity?> loginWithOAuth(OAuthProvider provider) async {
    state = const AsyncLoading();

    state = await AsyncValue.guard(() async {
      final authUrl = Uri.parse(provider.url);
      final launchUrl = authUrl.replace(
        queryParameters: {
          ...authUrl.queryParameters,
          'state': '${provider.state}$_oauthAppStateMarker',
        },
      );

      final callbackUrl = await FlutterWebAuth2.authenticate(
        url: launchUrl.toString(),
        callbackUrlScheme: _oauthCallbackScheme,
        // Also activates flutter_web_auth_2's browser-compatibility check for
        // Chrome's Auth Tab feature (unsupported by most non-Chrome browsers,
        // e.g. Firefox): without this, it unconditionally opts into Auth Tab,
        // which can race the OAuth redirect and report a spurious cancel.
        options: const FlutterWebAuth2Options(),
      );

      final callback = Uri.parse(callbackUrl);
      final oauthError = callback.queryParameters['error'];
      if (oauthError != null) {
        throw Exception(
          callback.queryParameters['error_description'] ?? oauthError,
        );
      }

      final code = callback.queryParameters['code'];
      final rawState = callback.queryParameters['state'];
      final oauthState = rawState?.endsWith(_oauthAppStateMarker) == true
          ? rawState!.substring(
              0,
              rawState.length - _oauthAppStateMarker.length,
            )
          : rawState;
      if (code == null || oauthState == null || oauthState != provider.state) {
        throw Exception(
          "OAuth provider response did not match the requested login.",
        );
      }

      final exchangeResponse = await ref
          .read(apiProvider)
          .post(
            '/auth/oauth',
            data: {
              'name': provider.name,
              'code': code,
              'codeVerifier': provider.codeVerifier,
            },
          );
      final authData = AuthResponse.fromJson(exchangeResponse.data);

      return await _updateUserEntity(authData.record.id);
    });
    return state.value;
  }

  Future<void> refresh() async {
    final id = state.value?.id;
    if (id == null) return;
    state = await AsyncValue.guard(() => _updateUserEntity(id));
  }

  Future<void> logout() async {
    state = const AsyncLoading();
    final jar = ref.read(cookieJarProvider);
    await jar.deleteAll();
    _box.removeAll();
    ref.invalidateSelf();
  }

  Future<UserEntity?> _updateUserEntity(String id) async {
    final userResponse = await ref
        .read(apiProvider)
        .get(
          "/user/$id",
          queryParameters: {
            "expand": "activitypub_actors_via_user, settings_via_user",
          },
        );

    final userData = User.fromJson(userResponse.data);

    final UserEntity userEntity = userData.toEntity();

    if (userData.expand?.settings != null) {
      await ref
          .read(settingsProvider.notifier)
          .updateFromServer(userData.expand!.settings!);
    }
    _box.put(userEntity);

    // Best-effort avatar cache refresh — fire-and-forget so a slow or failed
    // download never delays or breaks sign-in/refresh. cacheAvatar() itself
    // swallows all errors.
    final avatar = userEntity.avatar;
    if (avatar != null && avatar.isNotEmpty) {
      unawaited(
        cacheAvatar(
          ref.read(apiProvider),
          userEntity.getFileUrl(userEntity.serverUrl, avatar)!,
          userEntity.id,
          avatar,
        ),
      );
    }

    return userEntity;
  }
}
