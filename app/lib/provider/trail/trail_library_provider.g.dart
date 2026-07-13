// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail_library_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(TrailLibraryNotifier)
final trailLibraryProvider = TrailLibraryNotifierProvider._();

final class TrailLibraryNotifierProvider
    extends $NotifierProvider<TrailLibraryNotifier, List<Trail>> {
  TrailLibraryNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'trailLibraryProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$trailLibraryNotifierHash();

  @$internal
  @override
  TrailLibraryNotifier create() => TrailLibraryNotifier();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(List<Trail> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<List<Trail>>(value),
    );
  }
}

String _$trailLibraryNotifierHash() =>
    r'b908a461c0e9b2175998c184f84fa5cd324561ec';

abstract class _$TrailLibraryNotifier extends $Notifier<List<Trail>> {
  List<Trail> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<List<Trail>, List<Trail>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<List<Trail>, List<Trail>>,
              List<Trail>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
