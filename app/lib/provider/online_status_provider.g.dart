// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'online_status_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The app's single keepAlive source of truth for "is the backend reachable
/// right now". Optimistic by default (`true`), kept live by an
/// `InterceptorsWrapper` on the shared api client (see `api_provider.dart`)
/// so ordinary request/response traffic maintains it without any screen
/// having to probe explicitly.
///
/// `build()` reads no other provider — this is what prevents mutual
/// synchronous initialisation with `Api.build()`, since the api client's
/// interceptor reads this notifier lazily inside its callbacks, never during
/// its own `build()`.

@ProviderFor(OnlineStatus)
final onlineStatusProvider = OnlineStatusProvider._();

/// The app's single keepAlive source of truth for "is the backend reachable
/// right now". Optimistic by default (`true`), kept live by an
/// `InterceptorsWrapper` on the shared api client (see `api_provider.dart`)
/// so ordinary request/response traffic maintains it without any screen
/// having to probe explicitly.
///
/// `build()` reads no other provider — this is what prevents mutual
/// synchronous initialisation with `Api.build()`, since the api client's
/// interceptor reads this notifier lazily inside its callbacks, never during
/// its own `build()`.
final class OnlineStatusProvider extends $NotifierProvider<OnlineStatus, bool> {
  /// The app's single keepAlive source of truth for "is the backend reachable
  /// right now". Optimistic by default (`true`), kept live by an
  /// `InterceptorsWrapper` on the shared api client (see `api_provider.dart`)
  /// so ordinary request/response traffic maintains it without any screen
  /// having to probe explicitly.
  ///
  /// `build()` reads no other provider — this is what prevents mutual
  /// synchronous initialisation with `Api.build()`, since the api client's
  /// interceptor reads this notifier lazily inside its callbacks, never during
  /// its own `build()`.
  OnlineStatusProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'onlineStatusProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$onlineStatusHash();

  @$internal
  @override
  OnlineStatus create() => OnlineStatus();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(bool value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<bool>(value),
    );
  }
}

String _$onlineStatusHash() => r'ddc57c807dc7ceb952ac866ee246ffb5d3cb0fae';

/// The app's single keepAlive source of truth for "is the backend reachable
/// right now". Optimistic by default (`true`), kept live by an
/// `InterceptorsWrapper` on the shared api client (see `api_provider.dart`)
/// so ordinary request/response traffic maintains it without any screen
/// having to probe explicitly.
///
/// `build()` reads no other provider — this is what prevents mutual
/// synchronous initialisation with `Api.build()`, since the api client's
/// interceptor reads this notifier lazily inside its callbacks, never during
/// its own `build()`.

abstract class _$OnlineStatus extends $Notifier<bool> {
  bool build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<bool, bool>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<bool, bool>,
              bool,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
