// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'router_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

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

String _$routerListenableHash() => r'643820ed0e59076cb5f0680ee20cba2ae21e3a45';

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

String _$routerHash() => r'5ddac1e50a0fb1137d5a7038d285538c01f29863';

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
