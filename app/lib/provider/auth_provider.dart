import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/auth_response.dart';
import 'package:wanderer/models/user.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';

part 'auth_provider.g.dart';

@riverpod
class AuthNotifier extends _$AuthNotifier {
  @override
  FutureOr<User?> build() async {
    final jar = await ref.watch(cookieJarProvider.future);

    final cookies = await jar.loadForRequest(
      Uri.parse('https://demo.wanderer.to'),
    );
    final pbAuthCookie = cookies.where((c) => c.name == 'pb_auth').firstOrNull;
    if (pbAuthCookie != null) {
      try {
        return User.fromCookie(pbAuthCookie.value);
      } catch (e) {
        await jar.deleteAll();
        return null;
      }
    } else {
      return null;
    }
  }

  Future<User?> login(String username, String password) async {
    if (username.isEmpty || password.isEmpty) {
      state = AsyncError(
        Exception("Fields cannot be empty"),
        StackTrace.current,
      );
      return null;
    }

    state = const AsyncLoading();

    state = await AsyncValue.guard(() async {
      final response = await ref
          .read(apiProvider)
          .post(
            '/auth/login',
            data: {'username': username, 'password': password},
          );

      final authData = AuthResponse.fromJson(response.data);

      return authData.record;
    });
    return null;
  }
}
