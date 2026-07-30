// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(TrailNotifier)
final trailProvider = TrailNotifierFamily._();

final class TrailNotifierProvider
    extends $AsyncNotifierProvider<TrailNotifier, Trail> {
  TrailNotifierProvider._({
    required TrailNotifierFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'trailProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$trailNotifierHash();

  @override
  String toString() {
    return r'trailProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  TrailNotifier create() => TrailNotifier();

  @override
  bool operator ==(Object other) {
    return other is TrailNotifierProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$trailNotifierHash() => r'6fc8c62c595b430cfa612b9dd2c94b7ed15f4720';

final class TrailNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          TrailNotifier,
          AsyncValue<Trail>,
          Trail,
          FutureOr<Trail>,
          String
        > {
  TrailNotifierFamily._()
    : super(
        retry: null,
        name: r'trailProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  TrailNotifierProvider call(String id) =>
      TrailNotifierProvider._(argument: id, from: this);

  @override
  String toString() => r'trailProvider';
}

abstract class _$TrailNotifier extends $AsyncNotifier<Trail> {
  late final _$args = ref.$arg as String;
  String get id => _$args;

  FutureOr<Trail> build(String id);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<Trail>, Trail>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<Trail>, Trail>,
              AsyncValue<Trail>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
