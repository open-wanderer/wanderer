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
    required (
      NavigateResponse, {
      int? resumeManeuverIndex,
      List<Geographic>? resumeBreadcrumb,
    })
    super.argument,
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
        '$argument';
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

String _$navigationHash() => r'd54003a7561c0336c0eb32cbf9576536154772fc';

final class NavigationFamily extends $Family
    with
        $ClassFamilyOverride<
          Navigation,
          NavigationState,
          NavigationState,
          NavigationState,
          (
            NavigateResponse, {
            int? resumeManeuverIndex,
            List<Geographic>? resumeBreadcrumb,
          })
        > {
  NavigationFamily._()
    : super(
        retry: null,
        name: r'navigationProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  NavigationProvider call(
    NavigateResponse response, {
    int? resumeManeuverIndex,
    List<Geographic>? resumeBreadcrumb,
  }) => NavigationProvider._(
    argument: (
      response,
      resumeManeuverIndex: resumeManeuverIndex,
      resumeBreadcrumb: resumeBreadcrumb,
    ),
    from: this,
  );

  @override
  String toString() => r'navigationProvider';
}

abstract class _$Navigation extends $Notifier<NavigationState> {
  late final _$args =
      ref.$arg
          as (
            NavigateResponse, {
            int? resumeManeuverIndex,
            List<Geographic>? resumeBreadcrumb,
          });
  NavigateResponse get response => _$args.$1;
  int? get resumeManeuverIndex => _$args.resumeManeuverIndex;
  List<Geographic>? get resumeBreadcrumb => _$args.resumeBreadcrumb;

  NavigationState build(
    NavigateResponse response, {
    int? resumeManeuverIndex,
    List<Geographic>? resumeBreadcrumb,
  });
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
    element.handleCreate(
      ref,
      () => build(
        _$args.$1,
        resumeManeuverIndex: _$args.resumeManeuverIndex,
        resumeBreadcrumb: _$args.resumeBreadcrumb,
      ),
    );
  }
}
