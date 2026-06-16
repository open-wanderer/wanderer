// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'navigation_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(Navigation)
final navigationProvider = NavigationFamily._();

final class NavigationProvider
    extends $NotifierProvider<Navigation, NavigationState> {
  NavigationProvider._({
    required NavigationFamily super.from,
    required NavigateResponse super.argument,
  }) : super(
         retry: null,
         name: r'navigationProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$navigationHash();

  @override
  String toString() {
    return r'navigationProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  Navigation create() => Navigation();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(NavigationState value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<NavigationState>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is NavigationProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$navigationHash() => r'9fff8311bcfa9517373d5dad98908dc29f15fe55';

final class NavigationFamily extends $Family
    with
        $ClassFamilyOverride<
          Navigation,
          NavigationState,
          NavigationState,
          NavigationState,
          NavigateResponse
        > {
  NavigationFamily._()
    : super(
        retry: null,
        name: r'navigationProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  NavigationProvider call(NavigateResponse response) =>
      NavigationProvider._(argument: response, from: this);

  @override
  String toString() => r'navigationProvider';
}

abstract class _$Navigation extends $Notifier<NavigationState> {
  late final _$args = ref.$arg as NavigateResponse;
  NavigateResponse get response => _$args;

  NavigationState build(NavigateResponse response);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<NavigationState, NavigationState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<NavigationState, NavigationState>,
              NavigationState,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
