// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'map_cluster_search_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Debounced, bounds+zoom-keyed provider hitting `POST /search/trails/cluster`
/// (CLUS-01/04/05). Companion to [MapTrailSearch] — that provider still powers
/// the bottom-sheet trail list (full attributes); this one only feeds the
/// native cluster circle/count layers (`cluster_layer.dart`), so it returns
/// the server's already-clustered GeoJSON `FeatureCollection` verbatim.

@ProviderFor(MapClusterSearch)
final mapClusterSearchProvider = MapClusterSearchProvider._();

/// Debounced, bounds+zoom-keyed provider hitting `POST /search/trails/cluster`
/// (CLUS-01/04/05). Companion to [MapTrailSearch] — that provider still powers
/// the bottom-sheet trail list (full attributes); this one only feeds the
/// native cluster circle/count layers (`cluster_layer.dart`), so it returns
/// the server's already-clustered GeoJSON `FeatureCollection` verbatim.
final class MapClusterSearchProvider
    extends $AsyncNotifierProvider<MapClusterSearch, Map<String, dynamic>> {
  /// Debounced, bounds+zoom-keyed provider hitting `POST /search/trails/cluster`
  /// (CLUS-01/04/05). Companion to [MapTrailSearch] — that provider still powers
  /// the bottom-sheet trail list (full attributes); this one only feeds the
  /// native cluster circle/count layers (`cluster_layer.dart`), so it returns
  /// the server's already-clustered GeoJSON `FeatureCollection` verbatim.
  MapClusterSearchProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'mapClusterSearchProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$mapClusterSearchHash();

  @$internal
  @override
  MapClusterSearch create() => MapClusterSearch();
}

String _$mapClusterSearchHash() => r'4985253958ced31fa3459836d36559f536e11990';

/// Debounced, bounds+zoom-keyed provider hitting `POST /search/trails/cluster`
/// (CLUS-01/04/05). Companion to [MapTrailSearch] — that provider still powers
/// the bottom-sheet trail list (full attributes); this one only feeds the
/// native cluster circle/count layers (`cluster_layer.dart`), so it returns
/// the server's already-clustered GeoJSON `FeatureCollection` verbatim.

abstract class _$MapClusterSearch extends $AsyncNotifier<Map<String, dynamic>> {
  FutureOr<Map<String, dynamic>> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<AsyncValue<Map<String, dynamic>>, Map<String, dynamic>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<Map<String, dynamic>>,
                Map<String, dynamic>
              >,
              AsyncValue<Map<String, dynamic>>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
