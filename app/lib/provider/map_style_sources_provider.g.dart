// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'map_style_sources_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(MapStyleSourcesNotifier)
final mapStyleSourcesProvider = MapStyleSourcesNotifierProvider._();

final class MapStyleSourcesNotifierProvider
    extends $AsyncNotifierProvider<MapStyleSourcesNotifier, MapStyleSources> {
  MapStyleSourcesNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'mapStyleSourcesProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$mapStyleSourcesNotifierHash();

  @$internal
  @override
  MapStyleSourcesNotifier create() => MapStyleSourcesNotifier();
}

String _$mapStyleSourcesNotifierHash() =>
    r'433a213bbb321951d2326a75fb2140e0407c8a56';

abstract class _$MapStyleSourcesNotifier
    extends $AsyncNotifier<MapStyleSources> {
  FutureOr<MapStyleSources> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<MapStyleSources>, MapStyleSources>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<MapStyleSources>, MapStyleSources>,
              AsyncValue<MapStyleSources>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
