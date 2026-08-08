// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'map_trail_search_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(MapTrailSearch)
final mapTrailSearchProvider = MapTrailSearchFamily._();

final class MapTrailSearchProvider
    extends $AsyncNotifierProvider<MapTrailSearch, List<TrailSearchResult>> {
  MapTrailSearchProvider._({
    required MapTrailSearchFamily super.from,
    required ({String? authorId, String filterId}) super.argument,
  }) : super(
         retry: null,
         name: r'mapTrailSearchProvider',
         isAutoDispose: false,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$mapTrailSearchHash();

  @override
  String toString() {
    return r'mapTrailSearchProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  MapTrailSearch create() => MapTrailSearch();

  @override
  bool operator ==(Object other) {
    return other is MapTrailSearchProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$mapTrailSearchHash() => r'1c980d29f68487b8f87a773c29dfe58c9d69e01c';

final class MapTrailSearchFamily extends $Family
    with
        $ClassFamilyOverride<
          MapTrailSearch,
          AsyncValue<List<TrailSearchResult>>,
          List<TrailSearchResult>,
          FutureOr<List<TrailSearchResult>>,
          ({String? authorId, String filterId})
        > {
  MapTrailSearchFamily._()
    : super(
        retry: null,
        name: r'mapTrailSearchProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: false,
      );

  MapTrailSearchProvider call({String? authorId, required String filterId}) =>
      MapTrailSearchProvider._(
        argument: (authorId: authorId, filterId: filterId),
        from: this,
      );

  @override
  String toString() => r'mapTrailSearchProvider';
}

abstract class _$MapTrailSearch
    extends $AsyncNotifier<List<TrailSearchResult>> {
  late final _$args = ref.$arg as ({String? authorId, String filterId});
  String? get authorId => _$args.authorId;
  String get filterId => _$args.filterId;

  FutureOr<List<TrailSearchResult>> build({
    String? authorId,
    required String filterId,
  });
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
    element.handleCreate(
      ref,
      () => build(authorId: _$args.authorId, filterId: _$args.filterId),
    );
  }
}
