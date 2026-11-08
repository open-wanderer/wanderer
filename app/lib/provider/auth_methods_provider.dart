import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/oauth_provider.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'auth_methods_provider.g.dart';

/// What `GET /api/v1/auth/oauth` reports about signing in on a server.
class AuthMethods {
  final List<OAuthProvider> oauthProviders;

  /// Whether the instance turned off registration (`PUBLIC_DISABLE_SIGNUP`).
  /// Servers older than this field don't send it, so a missing value keeps
  /// the register option visible as before.
  final bool signupDisabled;

  const AuthMethods({
    required this.oauthProviders,
    required this.signupDisabled,
  });

  factory AuthMethods.fromJson(Map<String, dynamic> json) => AuthMethods(
    oauthProviders: ((json['oauth2']?['providers'] as List<dynamic>?) ?? [])
        .map((p) => OAuthProvider.fromJson(p as Map<String, dynamic>))
        .toList(),
    signupDisabled: json['signupDisabled'] == true,
  );
}

/// Fetches the sign-in options of the currently selected server.
///
/// Keyed by [serverUrl] so switching servers (and re-triggering a login after
/// a failed/cancelled attempt) naturally refetches: `state`/`codeVerifier` are
/// single-use PKCE values minted per request and must never be cached across
/// login attempts.
///
/// Assumes [apiProvider]'s base URL has already been pointed at [serverUrl]
/// (done by [ServerSelector]/server selection before this is watched).
@riverpod
Future<AuthMethods> authMethods(Ref ref, String serverUrl) async {
  final response = await ref.read(apiProvider).get('/auth/oauth');
  return AuthMethods.fromJson(response.data as Map<String, dynamic>);
}
