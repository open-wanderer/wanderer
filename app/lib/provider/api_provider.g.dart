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

String _$apiHash() => r'40b22755c87959a9d2bce34294c98aa11146b5e5';

abstract class _$Api extends $Notifier<Dio> {
  Dio build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<Dio, Dio>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<Dio, Dio>,
              Dio,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
