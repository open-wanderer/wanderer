// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail_search_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(TrailSearchNotifier)
final trailSearchProvider = TrailSearchNotifierProvider._();

final class TrailSearchNotifierProvider
    extends $AsyncNotifierProvider<TrailSearchNotifier, TrailSearchState> {
  TrailSearchNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'trailSearchProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$trailSearchNotifierHash();

  @$internal
  @override
  TrailSearchNotifier create() => TrailSearchNotifier();
}

String _$trailSearchNotifierHash() =>
    r'240dc4954df3ee0aca822c0cc16d732f92a34328';

abstract class _$TrailSearchNotifier extends $AsyncNotifier<TrailSearchState> {
  FutureOr<TrailSearchState> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<TrailSearchState>, TrailSearchState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<TrailSearchState>, TrailSearchState>,
              AsyncValue<TrailSearchState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
