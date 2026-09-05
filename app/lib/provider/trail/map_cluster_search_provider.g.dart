// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'map_cluster_search_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Debounced, bounds+zoom-keyed provider hitting `POST /search/trails/cluster`.
/// Companion to [MapTrailSearch] — that provider still powers the
/// bottom-sheet trail list (full attributes); this one only feeds the
/// native cluster circle/count layers (`cluster_layer.dart`), so it returns
/// the server's already-clustered GeoJSON `FeatureCollection` verbatim.

@ProviderFor(MapClusterSearch)
final mapClusterSearchProvider = MapClusterSearchFamily._();

/// Debounced, bounds+zoom-keyed provider hitting `POST /search/trails/cluster`.
/// Companion to [MapTrailSearch] — that provider still powers the
/// bottom-sheet trail list (full attributes); this one only feeds the
/// native cluster circle/count layers (`cluster_layer.dart`), so it returns
/// the server's already-clustered GeoJSON `FeatureCollection` verbatim.
final class MapClusterSearchProvider
    extends $AsyncNotifierProvider<MapClusterSearch, Map<String, dynamic>> {
  /// Debounced, bounds+zoom-keyed provider hitting `POST /search/trails/cluster`.
  /// Companion to [MapTrailSearch] — that provider still powers the
  /// bottom-sheet trail list (full attributes); this one only feeds the
  /// native cluster circle/count layers (`cluster_layer.dart`), so it returns
  /// the server's already-clustered GeoJSON `FeatureCollection` verbatim.
  MapClusterSearchProvider._({
    required MapClusterSearchFamily super.from,
    required ({String? authorId, String filterId}) super.argument,
  }) : super(
         retry: null,
         name: r'mapClusterSearchProvider',
         isAutoDispose: false,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$mapClusterSearchHash();

  @override
  String toString() {
    return r'mapClusterSearchProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  MapClusterSearch create() => MapClusterSearch();

  @override
  bool operator ==(Object other) {
    return other is MapClusterSearchProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$mapClusterSearchHash() => r'6bc581a57f0b4699562916ccb2854ad26be319a0';

/// Debounced, bounds+zoom-keyed provider hitting `POST /search/trails/cluster`.
/// Companion to [MapTrailSearch] — that provider still powers the
/// bottom-sheet trail list (full attributes); this one only feeds the
/// native cluster circle/count layers (`cluster_layer.dart`), so it returns
/// the server's already-clustered GeoJSON `FeatureCollection` verbatim.

final class MapClusterSearchFamily extends $Family
    with
        $ClassFamilyOverride<
          MapClusterSearch,
          AsyncValue<Map<String, dynamic>>,
          Map<String, dynamic>,
          FutureOr<Map<String, dynamic>>,
          ({String? authorId, String filterId})
        > {
  MapClusterSearchFamily._()
    : super(
        retry: null,
        name: r'mapClusterSearchProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: false,
      );

  /// Debounced, bounds+zoom-keyed provider hitting `POST /search/trails/cluster`.
  /// Companion to [MapTrailSearch] — that provider still powers the
  /// bottom-sheet trail list (full attributes); this one only feeds the
  /// native cluster circle/count layers (`cluster_layer.dart`), so it returns
  /// the server's already-clustered GeoJSON `FeatureCollection` verbatim.

  MapClusterSearchProvider call({String? authorId, required String filterId}) =>
      MapClusterSearchProvider._(
        argument: (authorId: authorId, filterId: filterId),
        from: this,
      );

  @override
  String toString() => r'mapClusterSearchProvider';
}

/// Debounced, bounds+zoom-keyed provider hitting `POST /search/trails/cluster`.
/// Companion to [MapTrailSearch] — that provider still powers the
/// bottom-sheet trail list (full attributes); this one only feeds the
/// native cluster circle/count layers (`cluster_layer.dart`), so it returns
/// the server's already-clustered GeoJSON `FeatureCollection` verbatim.

abstract class _$MapClusterSearch extends $AsyncNotifier<Map<String, dynamic>> {
  late final _$args = ref.$arg as ({String? authorId, String filterId});
  String? get authorId => _$args.authorId;
  String get filterId => _$args.filterId;

  FutureOr<Map<String, dynamic>> build({
    String? authorId,
    required String filterId,
  });
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
    element.handleCreate(
      ref,
      () => build(authorId: _$args.authorId, filterId: _$args.filterId),
    );
  }
}
