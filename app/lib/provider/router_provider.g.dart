// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'router_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Whether the splash screen's trail reveal has finished, been skipped, or hit
/// its failsafe deadline.
///
/// `HomeScreen` cannot time its own exit: the redirect below leaves `/` the
/// moment auth settles, tearing the splash down mid-animation. This flag is how
/// the splash asks the router to wait for it.
///
/// One-way by construction — it flips false→true exactly once per process and
/// never back. Anything that holds a route on it is therefore guaranteed to be
/// released, provided *someone* calls [SplashReveal.complete]; `HomeScreen`
/// arms a wall-clock failsafe so that holds even if the animation never runs.

@ProviderFor(SplashReveal)
final splashRevealProvider = SplashRevealProvider._();

/// Whether the splash screen's trail reveal has finished, been skipped, or hit
/// its failsafe deadline.
///
/// `HomeScreen` cannot time its own exit: the redirect below leaves `/` the
/// moment auth settles, tearing the splash down mid-animation. This flag is how
/// the splash asks the router to wait for it.
///
/// One-way by construction — it flips false→true exactly once per process and
/// never back. Anything that holds a route on it is therefore guaranteed to be
/// released, provided *someone* calls [SplashReveal.complete]; `HomeScreen`
/// arms a wall-clock failsafe so that holds even if the animation never runs.
final class SplashRevealProvider extends $NotifierProvider<SplashReveal, bool> {
  /// Whether the splash screen's trail reveal has finished, been skipped, or hit
  /// its failsafe deadline.
  ///
  /// `HomeScreen` cannot time its own exit: the redirect below leaves `/` the
  /// moment auth settles, tearing the splash down mid-animation. This flag is how
  /// the splash asks the router to wait for it.
  ///
  /// One-way by construction — it flips false→true exactly once per process and
  /// never back. Anything that holds a route on it is therefore guaranteed to be
  /// released, provided *someone* calls [SplashReveal.complete]; `HomeScreen`
  /// arms a wall-clock failsafe so that holds even if the animation never runs.
  SplashRevealProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'splashRevealProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$splashRevealHash();

  @$internal
  @override
  SplashReveal create() => SplashReveal();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(bool value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<bool>(value),
    );
  }
}

String _$splashRevealHash() => r'c391f59d0f4b9c5549af32dc9359039501e0f126';

/// Whether the splash screen's trail reveal has finished, been skipped, or hit
/// its failsafe deadline.
///
/// `HomeScreen` cannot time its own exit: the redirect below leaves `/` the
/// moment auth settles, tearing the splash down mid-animation. This flag is how
/// the splash asks the router to wait for it.
///
/// One-way by construction — it flips false→true exactly once per process and
/// never back. Anything that holds a route on it is therefore guaranteed to be
/// released, provided *someone* calls [SplashReveal.complete]; `HomeScreen`
/// arms a wall-clock failsafe so that holds even if the animation never runs.

abstract class _$SplashReveal extends $Notifier<bool> {
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

@ProviderFor(routerListenable)
final routerListenableProvider = RouterListenableProvider._();

final class RouterListenableProvider
    extends $FunctionalProvider<Listenable, Listenable, Listenable>
    with $Provider<Listenable> {
  RouterListenableProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'routerListenableProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$routerListenableHash();

  @$internal
  @override
  $ProviderElement<Listenable> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  Listenable create(Ref ref) {
    return routerListenable(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Listenable value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Listenable>(value),
    );
  }
}

String _$routerListenableHash() => r'9e88effedf206074301db22e9ac4a0be86287b05';

@ProviderFor(Router)
final routerProvider = RouterProvider._();

final class RouterProvider extends $NotifierProvider<Router, GoRouter> {
  RouterProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'routerProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$routerHash();

  @$internal
  @override
  Router create() => Router();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(GoRouter value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<GoRouter>(value),
    );
  }
}

String _$routerHash() => r'ae008b445e0f14c2e2688248430a8f34bec6bcb9';

abstract class _$Router extends $Notifier<GoRouter> {
  GoRouter build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<GoRouter, GoRouter>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<GoRouter, GoRouter>,
              GoRouter,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
