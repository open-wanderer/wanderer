// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'pending_launch_location_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Route an app shortcut asked for before the router could show it — on a
/// cold start, or while the user is signed out.
///
/// The router's post-splash redirect takes it to land there instead of `/map`.
/// Kept alive for the same reason as `PendingTrailImport`: the target must
/// survive a detour through the auth routes.

@ProviderFor(PendingLaunchLocation)
final pendingLaunchLocationProvider = PendingLaunchLocationProvider._();

/// Route an app shortcut asked for before the router could show it — on a
/// cold start, or while the user is signed out.
///
/// The router's post-splash redirect takes it to land there instead of `/map`.
/// Kept alive for the same reason as `PendingTrailImport`: the target must
/// survive a detour through the auth routes.
final class PendingLaunchLocationProvider
    extends $NotifierProvider<PendingLaunchLocation, String?> {
  /// Route an app shortcut asked for before the router could show it — on a
  /// cold start, or while the user is signed out.
  ///
  /// The router's post-splash redirect takes it to land there instead of `/map`.
  /// Kept alive for the same reason as `PendingTrailImport`: the target must
  /// survive a detour through the auth routes.
  PendingLaunchLocationProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'pendingLaunchLocationProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$pendingLaunchLocationHash();

  @$internal
  @override
  PendingLaunchLocation create() => PendingLaunchLocation();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(String? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<String?>(value),
    );
  }
}

String _$pendingLaunchLocationHash() =>
    r'060eeb3b4c75c704a7c68e720402bbfded38dd18';

/// Route an app shortcut asked for before the router could show it — on a
/// cold start, or while the user is signed out.
///
/// The router's post-splash redirect takes it to land there instead of `/map`.
/// Kept alive for the same reason as `PendingTrailImport`: the target must
/// survive a detour through the auth routes.

abstract class _$PendingLaunchLocation extends $Notifier<String?> {
  String? build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<String?, String?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<String?, String?>,
              String?,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
