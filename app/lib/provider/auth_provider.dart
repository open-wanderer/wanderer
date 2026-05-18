import 'package:dio/dio.dart';
import 'package:objectbox/objectbox.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/auth_response.dart';
import 'package:wanderer/models/user.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';

part 'auth_provider.g.dart';

@riverpod
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

    ref.read(apiProvider.notifier).updateBaseUrl("http://localhost:5173");

    final jar = ref.watch(cookieJarProvider);

    final cookies = await jar.loadForRequest(
      Uri.parse(savedUserEntity.serverUrl),
    );
    final pbAuthCookie = cookies.where((c) => c.name == 'pb_auth').firstOrNull;

    if (pbAuthCookie != null) {
      _updateUserEntity(savedUserEntity.id).catchError((err) {
        if (err is DioException && err.response?.statusCode == 404) {
          logout();
        }
        return null;
      });

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

  Future<void> logout() async {
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
          queryParameters: {"expand": "activitypub_actors_via_user"},
        );
    final userData = User.fromJson(userResponse.data);

    final UserEntity userEntity = userData.toEntity();
    _box.put(userEntity);

    return userEntity;
  }
}
