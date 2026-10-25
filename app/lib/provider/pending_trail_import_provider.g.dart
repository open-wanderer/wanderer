// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'pending_trail_import_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Path of a trail file handed to the app by the OS ("Open with" or the share
/// sheet), waiting for `TrailSourceSelectScreen` to import it.
///
/// The router's post-splash redirect reads this to land on `/trail/create`
/// instead of `/map`, so an inbound file goes through the same screen and
/// loading state as + → Import file. Kept alive because the file must survive
/// a detour through the auth routes when the cached session turns out to be
/// invalid.

@ProviderFor(PendingTrailImport)
final pendingTrailImportProvider = PendingTrailImportProvider._();

/// Path of a trail file handed to the app by the OS ("Open with" or the share
/// sheet), waiting for `TrailSourceSelectScreen` to import it.
///
/// The router's post-splash redirect reads this to land on `/trail/create`
/// instead of `/map`, so an inbound file goes through the same screen and
/// loading state as + → Import file. Kept alive because the file must survive
/// a detour through the auth routes when the cached session turns out to be
/// invalid.
final class PendingTrailImportProvider
    extends $NotifierProvider<PendingTrailImport, String?> {
  /// Path of a trail file handed to the app by the OS ("Open with" or the share
  /// sheet), waiting for `TrailSourceSelectScreen` to import it.
  ///
  /// The router's post-splash redirect reads this to land on `/trail/create`
  /// instead of `/map`, so an inbound file goes through the same screen and
  /// loading state as + → Import file. Kept alive because the file must survive
  /// a detour through the auth routes when the cached session turns out to be
  /// invalid.
  PendingTrailImportProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'pendingTrailImportProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$pendingTrailImportHash();

  @$internal
  @override
  PendingTrailImport create() => PendingTrailImport();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(String? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<String?>(value),
    );
  }
}

String _$pendingTrailImportHash() =>
    r'82a94aa47b9ccb1d91593c1d1828bde90fb0090f';

/// Path of a trail file handed to the app by the OS ("Open with" or the share
/// sheet), waiting for `TrailSourceSelectScreen` to import it.
///
/// The router's post-splash redirect reads this to land on `/trail/create`
/// instead of `/map`, so an inbound file goes through the same screen and
/// loading state as + → Import file. Kept alive because the file must survive
/// a detour through the auth routes when the cached session turns out to be
/// invalid.

abstract class _$PendingTrailImport extends $Notifier<String?> {
  String? build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<String?, String?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<String?, String?>,
              String?,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
