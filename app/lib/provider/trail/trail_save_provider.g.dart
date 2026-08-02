// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail_save_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(TrailSave)
final trailSaveProvider = TrailSaveProvider._();

final class TrailSaveProvider extends $AsyncNotifierProvider<TrailSave, void> {
  TrailSaveProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'trailSaveProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$trailSaveHash();

  @$internal
  @override
  TrailSave create() => TrailSave();
}

String _$trailSaveHash() => r'29973b00101ca9ab596bf12850b33d1ef514f88a';

abstract class _$TrailSave extends $AsyncNotifier<void> {
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
