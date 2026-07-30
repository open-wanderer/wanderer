// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'api_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(Api)
final apiProvider = ApiProvider._();

final class ApiProvider extends $NotifierProvider<Api, Dio> {
  ApiProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'apiProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$apiHash();

  @$internal
  @override
  Api create() => Api();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Dio value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Dio>(value),
    );
  }
}

String _$apiHash() => r'0d66870c46bb27323f51f24eeb9570da21f1dcef';

abstract class _$Api extends $Notifier<Dio> {
  Dio build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<Dio, Dio>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<Dio, Dio>,
              Dio,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
