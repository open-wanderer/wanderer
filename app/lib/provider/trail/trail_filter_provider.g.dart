// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail_filter_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(TrailFilterNotifier)
final trailFilterProvider = TrailFilterNotifierProvider._();

final class TrailFilterNotifierProvider
    extends $AsyncNotifierProvider<TrailFilterNotifier, TrailFilter> {
  TrailFilterNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'trailFilterProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$trailFilterNotifierHash();

  @$internal
  @override
  TrailFilterNotifier create() => TrailFilterNotifier();
}

String _$trailFilterNotifierHash() =>
    r'601cc8745b20a1514d0f2961a139a7e57c8bfa5f';

abstract class _$TrailFilterNotifier extends $AsyncNotifier<TrailFilter> {
  FutureOr<TrailFilter> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<TrailFilter>, TrailFilter>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<TrailFilter>, TrailFilter>,
              AsyncValue<TrailFilter>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
