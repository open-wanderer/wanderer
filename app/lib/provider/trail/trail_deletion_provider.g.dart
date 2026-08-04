// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail_deletion_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Broadcasts "this trail no longer exists on the server" to anything holding
/// a cached copy of it.
///
/// Deliberately NOT `ref.invalidate` on the map providers: `MapTrailSearch`
/// and `MapClusterSearch` keep their last bounds in instance fields, so
/// invalidating them drops the user back to an empty result list (and an empty
/// map) until they pan far enough to trigger a fresh bounds search. Listeners
/// splice the deleted id out of the state they already hold instead.
///
/// Only server deletes are announced here. Un-downloading a trail
/// (`TrailLibrary.deleteTrail`) leaves it on the server, so it must keep
/// showing up in map results.

@ProviderFor(TrailDeletions)
final trailDeletionsProvider = TrailDeletionsProvider._();

/// Broadcasts "this trail no longer exists on the server" to anything holding
/// a cached copy of it.
///
/// Deliberately NOT `ref.invalidate` on the map providers: `MapTrailSearch`
/// and `MapClusterSearch` keep their last bounds in instance fields, so
/// invalidating them drops the user back to an empty result list (and an empty
/// map) until they pan far enough to trigger a fresh bounds search. Listeners
/// splice the deleted id out of the state they already hold instead.
///
/// Only server deletes are announced here. Un-downloading a trail
/// (`TrailLibrary.deleteTrail`) leaves it on the server, so it must keep
/// showing up in map results.
final class TrailDeletionsProvider
    extends $NotifierProvider<TrailDeletions, TrailDeletion?> {
  /// Broadcasts "this trail no longer exists on the server" to anything holding
  /// a cached copy of it.
  ///
  /// Deliberately NOT `ref.invalidate` on the map providers: `MapTrailSearch`
  /// and `MapClusterSearch` keep their last bounds in instance fields, so
  /// invalidating them drops the user back to an empty result list (and an empty
  /// map) until they pan far enough to trigger a fresh bounds search. Listeners
  /// splice the deleted id out of the state they already hold instead.
  ///
  /// Only server deletes are announced here. Un-downloading a trail
  /// (`TrailLibrary.deleteTrail`) leaves it on the server, so it must keep
  /// showing up in map results.
  TrailDeletionsProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'trailDeletionsProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$trailDeletionsHash();

  @$internal
  @override
  TrailDeletions create() => TrailDeletions();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(TrailDeletion? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<TrailDeletion?>(value),
    );
  }
}

String _$trailDeletionsHash() => r'4bb901962ed8e7ec7e9018aa01cc6d6d1ee88dbf';

/// Broadcasts "this trail no longer exists on the server" to anything holding
/// a cached copy of it.
///
/// Deliberately NOT `ref.invalidate` on the map providers: `MapTrailSearch`
/// and `MapClusterSearch` keep their last bounds in instance fields, so
/// invalidating them drops the user back to an empty result list (and an empty
/// map) until they pan far enough to trigger a fresh bounds search. Listeners
/// splice the deleted id out of the state they already hold instead.
///
/// Only server deletes are announced here. Un-downloading a trail
/// (`TrailLibrary.deleteTrail`) leaves it on the server, so it must keep
/// showing up in map results.

abstract class _$TrailDeletions extends $Notifier<TrailDeletion?> {
  TrailDeletion? build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<TrailDeletion?, TrailDeletion?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<TrailDeletion?, TrailDeletion?>,
              TrailDeletion?,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
