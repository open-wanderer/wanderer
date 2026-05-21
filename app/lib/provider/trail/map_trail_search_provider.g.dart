// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'map_trail_search_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(MapTrailSearch)
final mapTrailSearchProvider = MapTrailSearchProvider._();

final class MapTrailSearchProvider
    extends $AsyncNotifierProvider<MapTrailSearch, List<TrailSearchResult>> {
  MapTrailSearchProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'mapTrailSearchProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$mapTrailSearchHash();

  @$internal
  @override
  MapTrailSearch create() => MapTrailSearch();
}

String _$mapTrailSearchHash() => r'a2575154dc3d11137e278bedb350e81e902fa50e';

abstract class _$MapTrailSearch
    extends $AsyncNotifier<List<TrailSearchResult>> {
  FutureOr<List<TrailSearchResult>> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<List<TrailSearchResult>>,
              List<TrailSearchResult>
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<List<TrailSearchResult>>,
                List<TrailSearchResult>
              >,
              AsyncValue<List<TrailSearchResult>>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
