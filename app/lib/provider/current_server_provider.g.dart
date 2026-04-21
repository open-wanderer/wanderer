// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'current_server_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(CurrentServer)
final currentServerProvider = CurrentServerProvider._();

final class CurrentServerProvider
    extends $NotifierProvider<CurrentServer, String?> {
  CurrentServerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'currentServerProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$currentServerHash();

  @$internal
  @override
  CurrentServer create() => CurrentServer();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(String? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<String?>(value),
    );
  }
}

String _$currentServerHash() => r'87c152518baf735a992888b77e1a4fcb03d5bc9f';

abstract class _$CurrentServer extends $Notifier<String?> {
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
