// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'map_camera_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(MapCamera)
final mapCameraProvider = MapCameraProvider._();

final class MapCameraProvider
    extends $NotifierProvider<MapCamera, MapCameraState?> {
  MapCameraProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'mapCameraProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$mapCameraHash();

  @$internal
  @override
  MapCamera create() => MapCamera();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(MapCameraState? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<MapCameraState?>(value),
    );
  }
}

String _$mapCameraHash() => r'313f26026cafc59bc234e6d6627672d1c6cd08a0';

abstract class _$MapCamera extends $Notifier<MapCameraState?> {
  MapCameraState? build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<MapCameraState?, MapCameraState?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<MapCameraState?, MapCameraState?>,
              MapCameraState?,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
