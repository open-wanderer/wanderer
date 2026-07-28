// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail_download_state_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Trail ids currently being downloaded, shared across every download entry
/// point (detail screen button, dropdown menu item) so starting a download
/// from one immediately disables the others instead of allowing a duplicate,
/// racing download of the same trail. `keepAlive` so the in-progress state
/// survives whichever widget happens to rebuild or unmount mid-download.

@ProviderFor(DownloadingTrailIds)
final downloadingTrailIdsProvider = DownloadingTrailIdsProvider._();

/// Trail ids currently being downloaded, shared across every download entry
/// point (detail screen button, dropdown menu item) so starting a download
/// from one immediately disables the others instead of allowing a duplicate,
/// racing download of the same trail. `keepAlive` so the in-progress state
/// survives whichever widget happens to rebuild or unmount mid-download.
final class DownloadingTrailIdsProvider
    extends $NotifierProvider<DownloadingTrailIds, Set<String>> {
  /// Trail ids currently being downloaded, shared across every download entry
  /// point (detail screen button, dropdown menu item) so starting a download
  /// from one immediately disables the others instead of allowing a duplicate,
  /// racing download of the same trail. `keepAlive` so the in-progress state
  /// survives whichever widget happens to rebuild or unmount mid-download.
  DownloadingTrailIdsProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'downloadingTrailIdsProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$downloadingTrailIdsHash();

  @$internal
  @override
  DownloadingTrailIds create() => DownloadingTrailIds();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Set<String> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Set<String>>(value),
    );
  }
}

String _$downloadingTrailIdsHash() =>
    r'65116bc3eec324e90704826a636b63b80a323f52';

/// Trail ids currently being downloaded, shared across every download entry
/// point (detail screen button, dropdown menu item) so starting a download
/// from one immediately disables the others instead of allowing a duplicate,
/// racing download of the same trail. `keepAlive` so the in-progress state
/// survives whichever widget happens to rebuild or unmount mid-download.

abstract class _$DownloadingTrailIds extends $Notifier<Set<String>> {
  Set<String> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<Set<String>, Set<String>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<Set<String>, Set<String>>,
              Set<String>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
