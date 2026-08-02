// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'waypoint_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(WaypointSave)
final waypointSaveProvider = WaypointSaveProvider._();

final class WaypointSaveProvider
    extends $AsyncNotifierProvider<WaypointSave, void> {
  WaypointSaveProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'waypointSaveProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$waypointSaveHash();

  @$internal
  @override
  WaypointSave create() => WaypointSave();
}

String _$waypointSaveHash() => r'54df1ac9b3163eb14823108ce3b08a677aa75f37';

abstract class _$WaypointSave extends $AsyncNotifier<void> {
  FutureOr<void> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<void>, void>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<void>, void>,
              AsyncValue<void>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
