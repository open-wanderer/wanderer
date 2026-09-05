// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'tile_repository_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Construction-only provider seam (mirrors `regionRepositoryProvider`) --
/// builds a [TileRepositoryManager] from the existing
/// [objectBoxProvider]/[apiProvider] without performing any I/O at build
/// time.

@ProviderFor(tileRepositoryManager)
final tileRepositoryManagerProvider = TileRepositoryManagerProvider._();

/// Construction-only provider seam (mirrors `regionRepositoryProvider`) --
/// builds a [TileRepositoryManager] from the existing
/// [objectBoxProvider]/[apiProvider] without performing any I/O at build
/// time.

final class TileRepositoryManagerProvider
    extends
        $FunctionalProvider<
          TileRepositoryManager,
          TileRepositoryManager,
          TileRepositoryManager
        >
    with $Provider<TileRepositoryManager> {
  /// Construction-only provider seam (mirrors `regionRepositoryProvider`) --
  /// builds a [TileRepositoryManager] from the existing
  /// [objectBoxProvider]/[apiProvider] without performing any I/O at build
  /// time.
  TileRepositoryManagerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'tileRepositoryManagerProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$tileRepositoryManagerHash();

  @$internal
  @override
  $ProviderElement<TileRepositoryManager> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  TileRepositoryManager create(Ref ref) {
    return tileRepositoryManager(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(TileRepositoryManager value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<TileRepositoryManager>(value),
    );
  }
}

String _$tileRepositoryManagerHash() =>
    r'64c5e2fb8758d08be2dcc4154282b30efea8b9c6';

/// Per-region download state, keyed by region PATH (never the server record
/// id, which the backend re-mints) -- the Settings/Regions screen subscribes
/// to this. `keepAlive` so in-progress state survives
/// whichever widget happens to rebuild or unmount mid-download (mirrors
/// `DownloadingTrailIds`). Vector and DEM downloads are fully independent:
/// each has its own start/cancel method and its own progress field on
/// [RegionDownloadState].

@ProviderFor(TileRepositoryStatus)
final tileRepositoryStatusProvider = TileRepositoryStatusProvider._();

/// Per-region download state, keyed by region PATH (never the server record
/// id, which the backend re-mints) -- the Settings/Regions screen subscribes
/// to this. `keepAlive` so in-progress state survives
/// whichever widget happens to rebuild or unmount mid-download (mirrors
/// `DownloadingTrailIds`). Vector and DEM downloads are fully independent:
/// each has its own start/cancel method and its own progress field on
/// [RegionDownloadState].
final class TileRepositoryStatusProvider
    extends
        $NotifierProvider<
          TileRepositoryStatus,
          Map<String, RegionDownloadState>
        > {
  /// Per-region download state, keyed by region PATH (never the server record
  /// id, which the backend re-mints) -- the Settings/Regions screen subscribes
  /// to this. `keepAlive` so in-progress state survives
  /// whichever widget happens to rebuild or unmount mid-download (mirrors
  /// `DownloadingTrailIds`). Vector and DEM downloads are fully independent:
  /// each has its own start/cancel method and its own progress field on
  /// [RegionDownloadState].
  TileRepositoryStatusProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'tileRepositoryStatusProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$tileRepositoryStatusHash();

  @$internal
  @override
  TileRepositoryStatus create() => TileRepositoryStatus();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Map<String, RegionDownloadState> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Map<String, RegionDownloadState>>(
        value,
      ),
    );
  }
}

String _$tileRepositoryStatusHash() =>
    r'cd2960ab5024729bb82ed1ef823290e270abc8bc';

/// Per-region download state, keyed by region PATH (never the server record
/// id, which the backend re-mints) -- the Settings/Regions screen subscribes
/// to this. `keepAlive` so in-progress state survives
/// whichever widget happens to rebuild or unmount mid-download (mirrors
/// `DownloadingTrailIds`). Vector and DEM downloads are fully independent:
/// each has its own start/cancel method and its own progress field on
/// [RegionDownloadState].

abstract class _$TileRepositoryStatus
    extends $Notifier<Map<String, RegionDownloadState>> {
  Map<String, RegionDownloadState> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              Map<String, RegionDownloadState>,
              Map<String, RegionDownloadState>
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                Map<String, RegionDownloadState>,
                Map<String, RegionDownloadState>
              >,
              Map<String, RegionDownloadState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
