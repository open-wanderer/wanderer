// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail_filter_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(TrailFilterNotifier)
final trailFilterProvider = TrailFilterNotifierFamily._();

final class TrailFilterNotifierProvider
    extends $AsyncNotifierProvider<TrailFilterNotifier, TrailFilter> {
  TrailFilterNotifierProvider._({
    required TrailFilterNotifierFamily super.from,
    required String super.argument,
  }) : super(
         retry: trailFilterRetry,
         name: r'trailFilterProvider',
         isAutoDispose: false,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$trailFilterNotifierHash();

  @override
  String toString() {
    return r'trailFilterProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  TrailFilterNotifier create() => TrailFilterNotifier();

  @override
  bool operator ==(Object other) {
    return other is TrailFilterNotifierProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$trailFilterNotifierHash() =>
    r'75b9f393ae833264cfa8a5512058b52938e4dfcb';

final class TrailFilterNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          TrailFilterNotifier,
          AsyncValue<TrailFilter>,
          TrailFilter,
          FutureOr<TrailFilter>,
          String
        > {
  TrailFilterNotifierFamily._()
    : super(
        retry: trailFilterRetry,
        name: r'trailFilterProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: false,
      );

  TrailFilterNotifierProvider call(String filterId) =>
      TrailFilterNotifierProvider._(argument: filterId, from: this);

  @override
  String toString() => r'trailFilterProvider';
}

abstract class _$TrailFilterNotifier extends $AsyncNotifier<TrailFilter> {
  late final _$args = ref.$arg as String;
  String get filterId => _$args;

  FutureOr<TrailFilter> build(String filterId);
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
    element.handleCreate(ref, () => build(_$args));
  }
}
