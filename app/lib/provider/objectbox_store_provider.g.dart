// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'objectbox_store_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ObjectBox)
final objectBoxProvider = ObjectBoxProvider._();

final class ObjectBoxProvider extends $NotifierProvider<ObjectBox, Store> {
  ObjectBoxProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'objectBoxProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$objectBoxHash();

  @$internal
  @override
  ObjectBox create() => ObjectBox();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Store value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Store>(value),
    );
  }
}

String _$objectBoxHash() => r'61f7ca205899848e8ab5170e636ab2c45ec0db82';

abstract class _$ObjectBox extends $Notifier<Store> {
  Store build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<Store, Store>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<Store, Store>,
              Store,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
