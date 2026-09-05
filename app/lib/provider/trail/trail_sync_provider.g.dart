// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail_sync_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The local ids of trails whose upload is currently draining, shared
/// across every trigger (app foreground, connectivity regained, cold
/// start, manual retry) exactly like `DownloadingTrailIds` shares download
/// state across its own entry points. `keepAlive` so an in-flight drain
/// survives whichever widget happens to rebuild or unmount mid-upload.
///
/// The KEY is the local id, not the server `id` -- a trail mid-drain may
/// still have no server id at all (pre-create) or one assigned moments ago
/// (post-create, pre-waypoints), so only the local id is stable across the
/// whole resume-from-step sequence.

@ProviderFor(TrailSync)
final trailSyncProvider = TrailSyncProvider._();

/// The local ids of trails whose upload is currently draining, shared
/// across every trigger (app foreground, connectivity regained, cold
/// start, manual retry) exactly like `DownloadingTrailIds` shares download
/// state across its own entry points. `keepAlive` so an in-flight drain
/// survives whichever widget happens to rebuild or unmount mid-upload.
///
/// The KEY is the local id, not the server `id` -- a trail mid-drain may
/// still have no server id at all (pre-create) or one assigned moments ago
/// (post-create, pre-waypoints), so only the local id is stable across the
/// whole resume-from-step sequence.
final class TrailSyncProvider
    extends $NotifierProvider<TrailSync, Set<String>> {
  /// The local ids of trails whose upload is currently draining, shared
  /// across every trigger (app foreground, connectivity regained, cold
  /// start, manual retry) exactly like `DownloadingTrailIds` shares download
  /// state across its own entry points. `keepAlive` so an in-flight drain
  /// survives whichever widget happens to rebuild or unmount mid-upload.
  ///
  /// The KEY is the local id, not the server `id` -- a trail mid-drain may
  /// still have no server id at all (pre-create) or one assigned moments ago
  /// (post-create, pre-waypoints), so only the local id is stable across the
  /// whole resume-from-step sequence.
  TrailSyncProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'trailSyncProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$trailSyncHash();

  @$internal
  @override
  TrailSync create() => TrailSync();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Set<String> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Set<String>>(value),
    );
  }
}

String _$trailSyncHash() => r'2e7ac9bd1208e83add1c86d4e405f930644279b7';

/// The local ids of trails whose upload is currently draining, shared
/// across every trigger (app foreground, connectivity regained, cold
/// start, manual retry) exactly like `DownloadingTrailIds` shares download
/// state across its own entry points. `keepAlive` so an in-flight drain
/// survives whichever widget happens to rebuild or unmount mid-upload.
///
/// The KEY is the local id, not the server `id` -- a trail mid-drain may
/// still have no server id at all (pre-create) or one assigned moments ago
/// (post-create, pre-waypoints), so only the local id is stable across the
/// whole resume-from-step sequence.

abstract class _$TrailSync extends $Notifier<Set<String>> {
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
