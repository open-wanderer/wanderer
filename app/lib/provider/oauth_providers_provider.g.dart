// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'oauth_providers_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Fetches the OAuth2 providers configured on the currently selected server.
///
/// Keyed by [serverUrl] so switching servers (and re-triggering a login after
/// a failed/cancelled attempt) naturally refetches: `state`/`codeVerifier` are
/// single-use PKCE values minted per request and must never be cached across
/// login attempts.
///
/// Assumes [apiProvider]'s base URL has already been pointed at [serverUrl]
/// (done by [ServerSelector]/server selection before this is watched).

@ProviderFor(oauthProviders)
final oauthProvidersProvider = OauthProvidersFamily._();

/// Fetches the OAuth2 providers configured on the currently selected server.
///
/// Keyed by [serverUrl] so switching servers (and re-triggering a login after
/// a failed/cancelled attempt) naturally refetches: `state`/`codeVerifier` are
/// single-use PKCE values minted per request and must never be cached across
/// login attempts.
///
/// Assumes [apiProvider]'s base URL has already been pointed at [serverUrl]
/// (done by [ServerSelector]/server selection before this is watched).

final class OauthProvidersProvider
    extends
        $FunctionalProvider<
          AsyncValue<List<OAuthProvider>>,
          List<OAuthProvider>,
          FutureOr<List<OAuthProvider>>
        >
    with
        $FutureModifier<List<OAuthProvider>>,
        $FutureProvider<List<OAuthProvider>> {
  /// Fetches the OAuth2 providers configured on the currently selected server.
  ///
  /// Keyed by [serverUrl] so switching servers (and re-triggering a login after
  /// a failed/cancelled attempt) naturally refetches: `state`/`codeVerifier` are
  /// single-use PKCE values minted per request and must never be cached across
  /// login attempts.
  ///
  /// Assumes [apiProvider]'s base URL has already been pointed at [serverUrl]
  /// (done by [ServerSelector]/server selection before this is watched).
  OauthProvidersProvider._({
    required OauthProvidersFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'oauthProvidersProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$oauthProvidersHash();

  @override
  String toString() {
    return r'oauthProvidersProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<List<OAuthProvider>> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<List<OAuthProvider>> create(Ref ref) {
    final argument = this.argument as String;
    return oauthProviders(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is OauthProvidersProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$oauthProvidersHash() => r'860d442d460f1f6c1f98d0a0a5cd47eeb5d1c7d1';

/// Fetches the OAuth2 providers configured on the currently selected server.
///
/// Keyed by [serverUrl] so switching servers (and re-triggering a login after
/// a failed/cancelled attempt) naturally refetches: `state`/`codeVerifier` are
/// single-use PKCE values minted per request and must never be cached across
/// login attempts.
///
/// Assumes [apiProvider]'s base URL has already been pointed at [serverUrl]
/// (done by [ServerSelector]/server selection before this is watched).

final class OauthProvidersFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<List<OAuthProvider>>, String> {
  OauthProvidersFamily._()
    : super(
        retry: null,
        name: r'oauthProvidersProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Fetches the OAuth2 providers configured on the currently selected server.
  ///
  /// Keyed by [serverUrl] so switching servers (and re-triggering a login after
  /// a failed/cancelled attempt) naturally refetches: `state`/`codeVerifier` are
  /// single-use PKCE values minted per request and must never be cached across
  /// login attempts.
  ///
  /// Assumes [apiProvider]'s base URL has already been pointed at [serverUrl]
  /// (done by [ServerSelector]/server selection before this is watched).

  OauthProvidersProvider call(String serverUrl) =>
      OauthProvidersProvider._(argument: serverUrl, from: this);

  @override
  String toString() => r'oauthProvidersProvider';
}
