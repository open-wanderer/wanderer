// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'router_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

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

String _$routerHash() => r'bc89a8936b3386244e6969e26aa76cfcaea93783';

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
