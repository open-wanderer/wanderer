import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/oauth_provider.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'oauth_providers_provider.g.dart';

/// Fetches the OAuth2 providers configured on the currently selected server.
///
/// Keyed by [serverUrl] so switching servers (and re-triggering a login after
/// a failed/cancelled attempt) naturally refetches: `state`/`codeVerifier` are
/// single-use PKCE values minted per request and must never be cached across
/// login attempts.
///
/// Assumes [apiProvider]'s base URL has already been pointed at [serverUrl]
/// (done by [ServerSelector]/server selection before this is watched).
@riverpod
Future<List<OAuthProvider>> oauthProviders(Ref ref, String serverUrl) async {
  final response = await ref.read(apiProvider).get('/auth/oauth');
  final providers =
      (response.data['oauth2']?['providers'] as List<dynamic>?) ?? [];
  return providers
      .map((p) => OAuthProvider.fromJson(p as Map<String, dynamic>))
      .toList();
}
