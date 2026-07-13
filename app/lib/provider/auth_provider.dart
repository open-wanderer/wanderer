import 'dart:async';

import 'package:dio/dio.dart';
import 'package:objectbox/objectbox.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/local_settings_entity.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/auth_response.dart';
import 'package:wanderer/models/user.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';

part 'auth_provider.g.dart';

@Riverpod(keepAlive: true)
class Auth extends _$Auth {
  static const _maxConsecutiveValidationFailures = 3;

  Box<UserEntity> get _box => ref.read(objectBoxProvider).box<UserEntity>();

  Box<LocalSettingsEntity> get _localSettingsBox =>
      ref.read(objectBoxProvider).box<LocalSettingsEntity>();

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
      // Gate navigation on a bounded validation round-trip so the router does
      // not treat an unconfirmed cached session as "logged in" and mount the
      // ~20 authenticated /map providers before the session is confirmed
      // (the concurrent-burst trigger). Offline/slow network still falls back
      // to the cached user within the timeout window.
      final validation = _updateUserEntity(savedUserEntity.id);
      // The .timeout below does NOT cancel the underlying Dio request; attach
      // a no-op sink to the source future so a rejection that arrives AFTER
      // the timeout has already fired does not surface as an unhandled async
      // error.
      unawaited(validation.catchError((_) => null));

      try {
        final validated = await validation.timeout(const Duration(seconds: 3));
        return validated ?? savedUserEntity;
      } on TimeoutException {
        // Offline/slow network: preserve the cached session, do not touch
        // the failure counter.
        return savedUserEntity;
      } catch (err) {
        if (_isAuthError(err)) {
          // logout() calls ref.invalidateSelf(), which is illegal while
          // build() is still running — defer it and return null now so the
          // router treats the session as logged-out immediately.
          unawaited(Future.microtask(logout));
          return null;
        }
        if (_isServerError(err)) {
          final failures = _bumpConsecutiveValidationFailures();
          if (failures >= _maxConsecutiveValidationFailures) {
            unawaited(Future.microtask(logout));
            return null;
          }
          return savedUserEntity;
        }
        // Any other error (e.g. Dio connectionError/connectionTimeout) is
        // treated as offline; preserve the cached session.
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

  bool _isServerError(Object err) {
    if (err is DioException) {
      final statusCode = err.response?.statusCode;
      return statusCode != null && statusCode >= 500;
    }
    return false;
  }

  int _bumpConsecutiveValidationFailures() {
    final entity = _localSettingsBox.getAll().firstOrNull ?? LocalSettingsEntity();
    entity.consecutiveAuthValidationFailures += 1;
    _localSettingsBox.put(entity);
    return entity.consecutiveAuthValidationFailures;
  }

  void _resetConsecutiveValidationFailures() {
    final entity = _localSettingsBox.getAll().firstOrNull;
    if (entity == null || entity.consecutiveAuthValidationFailures == 0) {
      return;
    }
    entity.consecutiveAuthValidationFailures = 0;
    _localSettingsBox.put(entity);
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
    _resetConsecutiveValidationFailures();

    return userEntity;
  }
}
