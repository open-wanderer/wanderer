import 'dart:async';

import 'package:objectbox/objectbox.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
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
      // Optimistically return the cached user immediately so startup does not
      // block on a network round-trip. Refresh the user in the background; if
      // the refresh reveals an invalid session (e.g. a 404 DioException, or any
      // other error), log the user out.
      unawaited(
        _updateUserEntity(savedUserEntity.id).catchError((Object err) {
          logout();
          return null;
        }),
      );
      return savedUserEntity;
    }
    return null;
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

    return userEntity;
  }
}
