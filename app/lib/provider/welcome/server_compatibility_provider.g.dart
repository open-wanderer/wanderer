// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'server_compatibility_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Whether the server at [serverUrl] is compatible with the app.
///
/// Keyed by URL so it runs as soon as a server is selected, before anyone
/// types credentials, and again for every server switched to. Takes the URL
/// explicitly rather than reading [apiProvider]'s base URL, so it never races
/// the base-URL update that accompanies a selection.

@ProviderFor(serverCompatibility)
final serverCompatibilityProvider = ServerCompatibilityFamily._();

/// Whether the server at [serverUrl] is compatible with the app.
///
/// Keyed by URL so it runs as soon as a server is selected, before anyone
/// types credentials, and again for every server switched to. Takes the URL
/// explicitly rather than reading [apiProvider]'s base URL, so it never races
/// the base-URL update that accompanies a selection.

final class ServerCompatibilityProvider
    extends
        $FunctionalProvider<
          AsyncValue<ServerCompatibility>,
          ServerCompatibility,
          FutureOr<ServerCompatibility>
        >
    with
        $FutureModifier<ServerCompatibility>,
        $FutureProvider<ServerCompatibility> {
  /// Whether the server at [serverUrl] is compatible with the app.
  ///
  /// Keyed by URL so it runs as soon as a server is selected, before anyone
  /// types credentials, and again for every server switched to. Takes the URL
  /// explicitly rather than reading [apiProvider]'s base URL, so it never races
  /// the base-URL update that accompanies a selection.
  ServerCompatibilityProvider._({
    required ServerCompatibilityFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'serverCompatibilityProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$serverCompatibilityHash();

  @override
  String toString() {
    return r'serverCompatibilityProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ServerCompatibility> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ServerCompatibility> create(Ref ref) {
    final argument = this.argument as String;
    return serverCompatibility(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is ServerCompatibilityProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$serverCompatibilityHash() =>
    r'2c92080450d42be1bbc21e1d0737b13e1d083d9d';

/// Whether the server at [serverUrl] is compatible with the app.
///
/// Keyed by URL so it runs as soon as a server is selected, before anyone
/// types credentials, and again for every server switched to. Takes the URL
/// explicitly rather than reading [apiProvider]'s base URL, so it never races
/// the base-URL update that accompanies a selection.

final class ServerCompatibilityFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<ServerCompatibility>, String> {
  ServerCompatibilityFamily._()
    : super(
        retry: null,
        name: r'serverCompatibilityProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Whether the server at [serverUrl] is compatible with the app.
  ///
  /// Keyed by URL so it runs as soon as a server is selected, before anyone
  /// types credentials, and again for every server switched to. Takes the URL
  /// explicitly rather than reading [apiProvider]'s base URL, so it never races
  /// the base-URL update that accompanies a selection.

  ServerCompatibilityProvider call(String serverUrl) =>
      ServerCompatibilityProvider._(argument: serverUrl, from: this);

  @override
  String toString() => r'serverCompatibilityProvider';
}

/// Whether the currently selected server is known to be incompatible — what
/// the welcome, login and register screens block on.
///
/// False while the check is still running or could not tell: blocking on
/// anything less than a definite answer would lock people out of a working
/// server over a moment offline.

@ProviderFor(selectedServerIncompatible)
final selectedServerIncompatibleProvider =
    SelectedServerIncompatibleProvider._();

/// Whether the currently selected server is known to be incompatible — what
/// the welcome, login and register screens block on.
///
/// False while the check is still running or could not tell: blocking on
/// anything less than a definite answer would lock people out of a working
/// server over a moment offline.

final class SelectedServerIncompatibleProvider
    extends $FunctionalProvider<bool, bool, bool>
    with $Provider<bool> {
  /// Whether the currently selected server is known to be incompatible — what
  /// the welcome, login and register screens block on.
  ///
  /// False while the check is still running or could not tell: blocking on
  /// anything less than a definite answer would lock people out of a working
  /// server over a moment offline.
  SelectedServerIncompatibleProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'selectedServerIncompatibleProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$selectedServerIncompatibleHash();

  @$internal
  @override
  $ProviderElement<bool> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  bool create(Ref ref) {
    return selectedServerIncompatible(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(bool value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<bool>(value),
    );
  }
}

String _$selectedServerIncompatibleHash() =>
    r'83f31f91a4d744e1954152fb65653003f0923e18';
