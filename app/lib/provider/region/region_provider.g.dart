// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'region_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Construction-only provider seam (D-02) -- builds a [RegionRepository]
/// from the existing [apiProvider]/[objectBoxProvider] without performing
/// any fetch on build. Not wired to any screen lifecycle yet; Phase 24
/// decides when [RegionRepository.refreshCatalog] is actually invoked.

@ProviderFor(regionRepository)
final regionRepositoryProvider = RegionRepositoryProvider._();

/// Construction-only provider seam (D-02) -- builds a [RegionRepository]
/// from the existing [apiProvider]/[objectBoxProvider] without performing
/// any fetch on build. Not wired to any screen lifecycle yet; Phase 24
/// decides when [RegionRepository.refreshCatalog] is actually invoked.

final class RegionRepositoryProvider
    extends
        $FunctionalProvider<
          RegionRepository,
          RegionRepository,
          RegionRepository
        >
    with $Provider<RegionRepository> {
  /// Construction-only provider seam (D-02) -- builds a [RegionRepository]
  /// from the existing [apiProvider]/[objectBoxProvider] without performing
  /// any fetch on build. Not wired to any screen lifecycle yet; Phase 24
  /// decides when [RegionRepository.refreshCatalog] is actually invoked.
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
