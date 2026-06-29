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

String _$settingsNotifierHash() => r'12a65d598d18cd7fd2480035fc341462a649cb74';

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
