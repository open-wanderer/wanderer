// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail_download_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(TrailDownloadServiceNotifier)
final trailDownloadServiceProvider = TrailDownloadServiceNotifierProvider._();

final class TrailDownloadServiceNotifierProvider
    extends
        $NotifierProvider<TrailDownloadServiceNotifier, TrailDownloadService> {
  TrailDownloadServiceNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'trailDownloadServiceProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$trailDownloadServiceNotifierHash();

  @$internal
  @override
  TrailDownloadServiceNotifier create() => TrailDownloadServiceNotifier();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(TrailDownloadService value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<TrailDownloadService>(value),
    );
  }
}

String _$trailDownloadServiceNotifierHash() =>
    r'28b89ddb1f81feb84a58340012e14ff4dc73f1bf';

abstract class _$TrailDownloadServiceNotifier
    extends $Notifier<TrailDownloadService> {
  TrailDownloadService build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<TrailDownloadService, TrailDownloadService>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<TrailDownloadService, TrailDownloadService>,
              TrailDownloadService,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
