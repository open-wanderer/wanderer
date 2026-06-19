// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'settings_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(SettingsNotifier)
final settingsProvider = SettingsNotifierProvider._();

final class SettingsNotifierProvider
    extends $NotifierProvider<SettingsNotifier, Settings?> {
  SettingsNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'settingsProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$settingsNotifierHash();

  @$internal
  @override
  SettingsNotifier create() => SettingsNotifier();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Settings? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Settings?>(value),
    );
  }
}

String _$settingsNotifierHash() => r'd518bfd735f9a5935533ee1e0dd6cbf3f2b136cd';

abstract class _$SettingsNotifier extends $Notifier<Settings?> {
  Settings? build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<Settings?, Settings?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<Settings?, Settings?>,
              Settings?,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
