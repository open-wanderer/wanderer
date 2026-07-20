import 'package:freezed_annotation/freezed_annotation.dart';

part 'oauth_provider.freezed.dart';
part 'oauth_provider.g.dart';

/// A single OAuth2 provider as returned by `GET /api/v1/auth/oauth`.
///
/// `state` and `codeVerifier` are single-use PKCE values minted per request —
/// this must be re-fetched for every login attempt, never cached long-term.
/// `url` is the ready-to-open authorization URL (PocketBase's `authURL` with
/// the web app's `redirect_uri` already appended); `img` is a base64 data URL
/// for the provider's logo.
@Freezed()
abstract class OAuthProvider with _$OAuthProvider {
  const factory OAuthProvider({
    required String name,
    required String displayName,
    required String state,
    required String codeVerifier,
    required String url,
    String? img,
  }) = _OAuthProvider;

  factory OAuthProvider.fromJson(Map<String, dynamic> json) =>
      _$OAuthProviderFromJson(json);
}
