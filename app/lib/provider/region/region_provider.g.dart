// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'region_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Construction-only provider seam -- builds a [RegionRepository]
/// from the existing [apiProvider]/[objectBoxProvider] without performing
/// any fetch on build; the Settings/Regions screen drives
/// [RegionRepository.refreshCatalogAndFetchHierarchy] on open.

@ProviderFor(regionRepository)
final regionRepositoryProvider = RegionRepositoryProvider._();

/// Construction-only provider seam -- builds a [RegionRepository]
/// from the existing [apiProvider]/[objectBoxProvider] without performing
/// any fetch on build; the Settings/Regions screen drives
/// [RegionRepository.refreshCatalogAndFetchHierarchy] on open.

final class RegionRepositoryProvider
    extends
        $FunctionalProvider<
          RegionRepository,
          RegionRepository,
          RegionRepository
        >
    with $Provider<RegionRepository> {
  /// Construction-only provider seam -- builds a [RegionRepository]
  /// from the existing [apiProvider]/[objectBoxProvider] without performing
  /// any fetch on build; the Settings/Regions screen drives
  /// [RegionRepository.refreshCatalogAndFetchHierarchy] on open.
  RegionRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'regionRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$regionRepositoryHash();

  @$internal
  @override
  $ProviderElement<RegionRepository> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  RegionRepository create(Ref ref) {
    return regionRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(RegionRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<RegionRepository>(value),
    );
  }
}

String _$regionRepositoryHash() => r'3eb7cb07afb9eb85eea394d6fab49f0359cccb0e';

/// Synchronous ObjectBox snapshot of every persisted [RegionEntity], sorted
/// alphabetically by [RegionEntity.name] -- mirrors
/// `trail_library_provider.dart`'s `TrailLibraryNotifier` structural
/// precedent verbatim. No mutation methods live here; all region mutations
/// flow through `TileRepositoryStatus` (`tile_repository_provider.dart`),
/// which invalidates this provider itself at every terminal point of a
/// download/cancel/delete (ObjectBox `ToOne.target` caches per-instance
/// after first read). That invalidation deliberately
/// lives in the keepAlive notifier, NOT in the calling widget: a widget-side
/// `mounted` guard skipped it whenever the user left the Settings/Regions
/// screen mid-download.

@ProviderFor(RegionListNotifier)
final regionListNotifierProvider = RegionListNotifierProvider._();

/// Synchronous ObjectBox snapshot of every persisted [RegionEntity], sorted
/// alphabetically by [RegionEntity.name] -- mirrors
/// `trail_library_provider.dart`'s `TrailLibraryNotifier` structural
/// precedent verbatim. No mutation methods live here; all region mutations
/// flow through `TileRepositoryStatus` (`tile_repository_provider.dart`),
/// which invalidates this provider itself at every terminal point of a
/// download/cancel/delete (ObjectBox `ToOne.target` caches per-instance
/// after first read). That invalidation deliberately
/// lives in the keepAlive notifier, NOT in the calling widget: a widget-side
/// `mounted` guard skipped it whenever the user left the Settings/Regions
/// screen mid-download.
final class RegionListNotifierProvider
    extends $NotifierProvider<RegionListNotifier, List<RegionEntity>> {
  /// Synchronous ObjectBox snapshot of every persisted [RegionEntity], sorted
  /// alphabetically by [RegionEntity.name] -- mirrors
  /// `trail_library_provider.dart`'s `TrailLibraryNotifier` structural
  /// precedent verbatim. No mutation methods live here; all region mutations
  /// flow through `TileRepositoryStatus` (`tile_repository_provider.dart`),
  /// which invalidates this provider itself at every terminal point of a
  /// download/cancel/delete (ObjectBox `ToOne.target` caches per-instance
  /// after first read). That invalidation deliberately
  /// lives in the keepAlive notifier, NOT in the calling widget: a widget-side
  /// `mounted` guard skipped it whenever the user left the Settings/Regions
  /// screen mid-download.
  RegionListNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'regionListNotifierProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$regionListNotifierHash();

  @$internal
  @override
  RegionListNotifier create() => RegionListNotifier();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(List<RegionEntity> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<List<RegionEntity>>(value),
    );
  }
}

String _$regionListNotifierHash() =>
    r'6d1dc36084b4f91dda9f630d31e9e5c1650eb6a0';

/// Synchronous ObjectBox snapshot of every persisted [RegionEntity], sorted
/// alphabetically by [RegionEntity.name] -- mirrors
/// `trail_library_provider.dart`'s `TrailLibraryNotifier` structural
/// precedent verbatim. No mutation methods live here; all region mutations
/// flow through `TileRepositoryStatus` (`tile_repository_provider.dart`),
/// which invalidates this provider itself at every terminal point of a
/// download/cancel/delete (ObjectBox `ToOne.target` caches per-instance
/// after first read). That invalidation deliberately
/// lives in the keepAlive notifier, NOT in the calling widget: a widget-side
/// `mounted` guard skipped it whenever the user left the Settings/Regions
/// screen mid-download.

abstract class _$RegionListNotifier extends $Notifier<List<RegionEntity>> {
  List<RegionEntity> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<List<RegionEntity>, List<RegionEntity>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<List<RegionEntity>, List<RegionEntity>>,
              List<RegionEntity>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
