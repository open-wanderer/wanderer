import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter_web_auth_2/flutter_web_auth_2.dart';
import 'package:objectbox/objectbox.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/actor_entity.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/auth_response.dart';
import 'package:wanderer/models/oauth_provider.dart';
import 'package:wanderer/models/user.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/store/account_data_purge.dart';
import 'package:wanderer/store/avatar_cache.dart';

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

    if (pbAuthCookie == null) {
      return null;
    }

    unawaited(_validateInBackground(savedUserEntity));
    return savedUserEntity;
  }

  /// Re-reads the signed-in user from the server without holding up the
  /// session.
  ///
  /// Everything the router needs is one bit — whether anyone is signed in —
  /// and [build] has already resolved it from disk above: a cached
  /// [UserEntity] plus an unexpired `pb_auth` cookie. `PersistCookieJar` drops
  /// expired cookies as it loads them, and the server stamps the cookie's
  /// `expires` from the token's own `exp` claim (`exportToCookie` in
  /// web/src/hooks.server.ts), so cookie-present *is* token-unexpired. An
  /// expired token therefore resolves to `/welcome` off local disk, with no
  /// request at all. Awaiting the network to learn the same bit again is what
  /// left the splash sitting on a finished animation.
  ///
  /// What the round-trip still buys is a session the server revoked inside a
  /// still-valid token window, plus fresher profile data. Both are fine to
  /// land late.
  Future<void> _validateInBackground(UserEntity cached) async {
    final UserEntity? fresh;
    try {
      fresh = await _updateUserEntity(cached.id);
    } catch (err) {
      if (_isAuthError(err)) {
        await logout();
      }
      // Any other error (a 5xx, or a Dio connectionError/connectionTimeout) is
      // treated as offline: the cached session stands.
      return;
    }

    if (fresh == null || !ref.mounted) return;

    // Only a material change is published. 29 call sites watch this provider,
    // and two of them (trail_search_provider, profile_provider) watch
    // `.future` — so an unconditional emission would refetch trails and
    // profile a beat after every cold start, to redisplay identical data. Same
    // reasoning as the flip-only filter in `routerListenable`
    // (router_provider.dart), one layer up.
    if (_materiallyDiffers(state.value, fresh)) {
      state = AsyncData(fresh);
    }
  }

  /// Whether re-publishing [fresh] would change anything a widget can see.
  ///
  /// Compares the scalar fields read off this provider and nothing else. The
  /// two related rows the fetch also refreshes are deliberately out of scope:
  /// settings propagate through `settingsProvider.updateFromServer`, and the
  /// actor row is upserted in place — no caller reads `user.actor.target` off
  /// the auth state, only the scalar [UserEntity.actorId].
  ///
  /// `updated` is excluded on purpose: any server-side bump unrelated to what
  /// is displayed would defeat the filter and make every launch emit.
  bool _materiallyDiffers(UserEntity? cached, UserEntity fresh) {
    if (cached == null) return true;
    return cached.id != fresh.id ||
        cached.actorId != fresh.actorId ||
        cached.username != fresh.username ||
        cached.preferredUsername != fresh.preferredUsername ||
        cached.email != fresh.email ||
        cached.avatar != fresh.avatar ||
        cached.iri != fresh.iri ||
        cached.serverUrl != fresh.serverUrl;
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

  /// Signs the current user out.
  ///
  /// Note for callers reading the auth state: this drops the provider to a
  /// value-less [AsyncLoading] for the duration, which `settings_screen`
  /// renders as a spinner on the sign-out button. Widgets that read the user
  /// must therefore tolerate a null — see the note on [_validateInBackground],
  /// which can now trigger this with `/map` live rather than during the splash.
  Future<void> logout() async {
    state = const AsyncLoading();
    final jar = ref.read(cookieJarProvider);
    await jar.deleteAll();
    _box.removeAll();
    // Privacy invariant: after logout, no content belonging to the signed-out
    // account is REACHABLE by whoever signs in next. Not
    // the same as erasing it -- the offline library deliberately survives and
    // is scoped per account via `TrailEntity.savedByUserIds`, because deleting
    // it meant signing out of an account destroyed every trail it had
    // downloaded.
    await purgeAccountScopedData(ref.read(objectBoxProvider));
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

    // `toEntity()` builds a fresh ActorEntity (obxId 0); putting that would
    // replace the existing row under a new ObjectBox id and orphan every
    // `TrailEntity.author` ToOne pointing at it -- which is what made
    // already-recorded local trails degrade to "Unknown" with a blank avatar
    // after a re-auth. Reuse the cached row's id so the put updates in place.
    final incomingActor = userData.expand?.actor;
    if (incomingActor != null) {
      userEntity.actor.target = actorEntityForUpsert(
        ref.read(objectBoxProvider),
        incomingActor,
      );
    }

    // Defense in depth for a logout that never ran or was interrupted
    // mid-flight: if the incoming account differs from whatever is cached,
    // purge before this account's data is written. Must run before the
    // `updateFromServer` call below (it purges SettingsEntity, so the
    // incoming account's settings must not be written first), and
    // `_box.removeAll()` must run before the later `_box.put(userEntity)` so
    // the store can never hold two UserEntity rows — every reader resolves
    // the session via `getAll().firstOrNull`, which would otherwise be able
    // to return the previous account.
    final cachedUserId = _box.getAll().firstOrNull?.id;
    if (shouldPurgeForIncomingUser(cachedUserId, userEntity.id)) {
      _box.removeAll();
      await purgeAccountScopedData(ref.read(objectBoxProvider));
    }

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
