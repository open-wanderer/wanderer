// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'auth_methods_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Fetches the sign-in options of the currently selected server.
///
/// Keyed by [serverUrl] so switching servers (and re-triggering a login after
/// a failed/cancelled attempt) naturally refetches: `state`/`codeVerifier` are
/// single-use PKCE values minted per request and must never be cached across
/// login attempts.
///
/// Assumes [apiProvider]'s base URL has already been pointed at [serverUrl]
/// (done by [ServerSelector]/server selection before this is watched).

@ProviderFor(authMethods)
final authMethodsProvider = AuthMethodsFamily._();

/// Fetches the sign-in options of the currently selected server.
///
/// Keyed by [serverUrl] so switching servers (and re-triggering a login after
/// a failed/cancelled attempt) naturally refetches: `state`/`codeVerifier` are
/// single-use PKCE values minted per request and must never be cached across
/// login attempts.
///
/// Assumes [apiProvider]'s base URL has already been pointed at [serverUrl]
/// (done by [ServerSelector]/server selection before this is watched).

final class AuthMethodsProvider
    extends
        $FunctionalProvider<
          AsyncValue<AuthMethods>,
          AuthMethods,
          FutureOr<AuthMethods>
        >
    with $FutureModifier<AuthMethods>, $FutureProvider<AuthMethods> {
  /// Fetches the sign-in options of the currently selected server.
  ///
  /// Keyed by [serverUrl] so switching servers (and re-triggering a login after
  /// a failed/cancelled attempt) naturally refetches: `state`/`codeVerifier` are
  /// single-use PKCE values minted per request and must never be cached across
  /// login attempts.
  ///
  /// Assumes [apiProvider]'s base URL has already been pointed at [serverUrl]
  /// (done by [ServerSelector]/server selection before this is watched).
  AuthMethodsProvider._({
    required AuthMethodsFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'authMethodsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$authMethodsHash();

  @override
  String toString() {
    return r'authMethodsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<AuthMethods> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<AuthMethods> create(Ref ref) {
    final argument = this.argument as String;
    return authMethods(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AuthMethodsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$authMethodsHash() => r'cf3fd3d3a6d681dedb8e7f40da5d9c0ba32448cc';

/// Fetches the sign-in options of the currently selected server.
///
/// Keyed by [serverUrl] so switching servers (and re-triggering a login after
/// a failed/cancelled attempt) naturally refetches: `state`/`codeVerifier` are
/// single-use PKCE values minted per request and must never be cached across
/// login attempts.
///
/// Assumes [apiProvider]'s base URL has already been pointed at [serverUrl]
/// (done by [ServerSelector]/server selection before this is watched).

final class AuthMethodsFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<AuthMethods>, String> {
  AuthMethodsFamily._()
    : super(
        retry: null,
        name: r'authMethodsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Fetches the sign-in options of the currently selected server.
  ///
  /// Keyed by [serverUrl] so switching servers (and re-triggering a login after
  /// a failed/cancelled attempt) naturally refetches: `state`/`codeVerifier` are
  /// single-use PKCE values minted per request and must never be cached across
  /// login attempts.
  ///
  /// Assumes [apiProvider]'s base URL has already been pointed at [serverUrl]
  /// (done by [ServerSelector]/server selection before this is watched).

  AuthMethodsProvider call(String serverUrl) =>
      AuthMethodsProvider._(argument: serverUrl, from: this);

  @override
  String toString() => r'authMethodsProvider';
}
